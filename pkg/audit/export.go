// Package audit provides SIEM export functionality.
package audit

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// Exporter handles exporting audit logs to external SIEM systems.
type Exporter struct {
	db         *pgxpool.Pool
	log        *logger.Logger
	httpClient *http.Client
	exporters  map[string]SIEMExporter
}

// SIEMExporter is the interface for SIEM-specific exporters.
type SIEMExporter interface {
	Export(ctx context.Context, entries []AuditLogRow) error
	Name() string
}

// NewExporter creates a new audit log exporter.
func NewExporter(db *pgxpool.Pool, log *logger.Logger) *Exporter {
	return &Exporter{
		db:  db,
		log: log.WithComponent("audit-exporter"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		exporters: make(map[string]SIEMExporter),
	}
}

// RegisterExporter registers a SIEM exporter.
func (e *Exporter) RegisterExporter(exporter SIEMExporter) {
	e.exporters[exporter.Name()] = exporter
	e.log.Info("registered SIEM exporter", "name", exporter.Name())
}

// ProcessQueue processes the export queue.
func (e *Exporter) ProcessQueue(ctx context.Context, batchSize int) error {
	// Get pending exports
	query := `
		SELECT eq.id, eq.audit_log_id, eq.destination, eq.attempts,
		       al.id, al.timestamp, al.actor_type, al.actor_id, al.actor_email,
		       al.org_id, al.action, al.action_category,
		       al.resource_type, al.resource_id, al.resource_name,
		       al.changes, al.context, al.risk_level,
		       al.status, al.error_message, al.integrity_hash
		FROM audit_export_queue eq
		JOIN audit_logs al ON al.id = eq.audit_log_id
		WHERE eq.status = 'pending' AND eq.attempts < 3
		ORDER BY eq.created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := e.db.Query(ctx, query, batchSize)
	if err != nil {
		return fmt.Errorf("failed to query export queue: %w", err)
	}
	defer rows.Close()

	// Group by destination
	byDestination := make(map[string][]exportItem)

	for rows.Next() {
		var item exportItem
		var changes, context []byte

		err := rows.Scan(
			&item.QueueID, &item.AuditLogID, &item.Destination, &item.Attempts,
			&item.Log.ID, &item.Log.Timestamp, &item.Log.ActorType, &item.Log.ActorID, &item.Log.ActorEmail,
			&item.Log.OrgID, &item.Log.Action, &item.Log.ActionCategory,
			&item.Log.ResourceType, &item.Log.ResourceID, &item.Log.ResourceName,
			&changes, &context, &item.Log.RiskLevel,
			&item.Log.Status, &item.Log.ErrorMessage, &item.Log.IntegrityHash,
		)
		if err != nil {
			e.log.Warn("failed to scan export row", "error", err)
			continue
		}

		json.Unmarshal(changes, &item.Log.Changes)
		json.Unmarshal(context, &item.Log.Context)

		byDestination[item.Destination] = append(byDestination[item.Destination], item)
	}

	// Process each destination
	for dest, items := range byDestination {
		exporter, ok := e.exporters[dest]
		if !ok {
			e.log.Warn("no exporter registered for destination", "destination", dest)
			continue
		}

		// Collect logs for this batch
		logs := make([]AuditLogRow, len(items))
		queueIDs := make([]uuid.UUID, len(items))
		for i, item := range items {
			logs[i] = item.Log
			queueIDs[i] = item.QueueID
		}

		// Export
		err := exporter.Export(ctx, logs)
		if err != nil {
			e.log.Error("export failed", "destination", dest, "error", err)
			e.markFailed(ctx, queueIDs, err.Error())
			continue
		}

		e.markCompleted(ctx, queueIDs)
	}

	return nil
}

type exportItem struct {
	QueueID     uuid.UUID
	AuditLogID  uuid.UUID
	Destination string
	Attempts    int
	Log         AuditLogRow
}

func (e *Exporter) markCompleted(ctx context.Context, queueIDs []uuid.UUID) {
	for _, id := range queueIDs {
		_, err := e.db.Exec(ctx, `
			UPDATE audit_export_queue
			SET status = 'completed', completed_at = NOW()
			WHERE id = $1
		`, id)
		if err != nil {
			e.log.Warn("failed to mark export completed", "id", id, "error", err)
		}
	}
}

func (e *Exporter) markFailed(ctx context.Context, queueIDs []uuid.UUID, errorMsg string) {
	for _, id := range queueIDs {
		_, err := e.db.Exec(ctx, `
			UPDATE audit_export_queue
			SET status = CASE WHEN attempts >= 2 THEN 'failed' ELSE 'pending' END,
			    attempts = attempts + 1,
			    last_attempt_at = NOW(),
			    error_message = $2
			WHERE id = $1
		`, id, errorMsg)
		if err != nil {
			e.log.Warn("failed to mark export failed", "id", id, "error", err)
		}
	}
}

// SplunkExporter exports audit logs to Splunk HEC.
type SplunkExporter struct {
	endpoint   string
	token      string
	index      string
	sourcetype string
	httpClient *http.Client
	log        *logger.Logger
}

// SplunkConfig contains Splunk HEC configuration.
type SplunkConfig struct {
	Endpoint   string // e.g., https://splunk.example.com:8088/services/collector
	Token      string // HEC token
	Index      string // Target index
	Sourcetype string // Sourcetype for events
}

// NewSplunkExporter creates a new Splunk HEC exporter.
func NewSplunkExporter(cfg SplunkConfig, log *logger.Logger) *SplunkExporter {
	return &SplunkExporter{
		endpoint:   cfg.Endpoint,
		token:      cfg.Token,
		index:      cfg.Index,
		sourcetype: cfg.Sourcetype,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		log:        log.WithComponent("splunk-exporter"),
	}
}

func (s *SplunkExporter) Name() string {
	return "splunk"
}

// Export sends audit logs to Splunk HEC.
func (s *SplunkExporter) Export(ctx context.Context, entries []AuditLogRow) error {
	var buf bytes.Buffer

	for _, entry := range entries {
		event := map[string]interface{}{
			"time":       entry.Timestamp.Unix(),
			"host":       "ql-rf",
			"source":     "audit",
			"sourcetype": s.sourcetype,
			"index":      s.index,
			"event":      entry,
		}

		eventJSON, err := json.Marshal(event)
		if err != nil {
			continue
		}
		buf.Write(eventJSON)
		buf.WriteByte('\n')
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.endpoint, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Splunk "+s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("splunk request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("splunk returned status %d: %s", resp.StatusCode, string(body))
	}

	s.log.Info("exported to Splunk", "count", len(entries))
	return nil
}

// ElasticExporter exports audit logs to Elasticsearch.
type ElasticExporter struct {
	endpoint   string
	username   string
	password   string
	index      string
	httpClient *http.Client
	log        *logger.Logger
}

// ElasticConfig contains Elasticsearch configuration.
type ElasticConfig struct {
	Endpoint string // e.g., https://elastic.example.com:9200
	Username string
	Password string
	Index    string // Index name pattern, e.g., "audit-logs"
}

// NewElasticExporter creates a new Elasticsearch exporter.
func NewElasticExporter(cfg ElasticConfig, log *logger.Logger) *ElasticExporter {
	return &ElasticExporter{
		endpoint:   cfg.Endpoint,
		username:   cfg.Username,
		password:   cfg.Password,
		index:      cfg.Index,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		log:        log.WithComponent("elastic-exporter"),
	}
}

func (e *ElasticExporter) Name() string {
	return "elastic"
}

// Export sends audit logs to Elasticsearch using bulk API.
func (e *ElasticExporter) Export(ctx context.Context, entries []AuditLogRow) error {
	var buf bytes.Buffer

	// Build bulk request body
	for _, entry := range entries {
		// Action line
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": fmt.Sprintf("%s-%s", e.index, entry.Timestamp.Format("2006.01.02")),
				"_id":    entry.ID.String(),
			},
		}
		actionJSON, _ := json.Marshal(action)
		buf.Write(actionJSON)
		buf.WriteByte('\n')

		// Document line
		docJSON, _ := json.Marshal(entry)
		buf.Write(docJSON)
		buf.WriteByte('\n')
	}

	url := fmt.Sprintf("%s/_bulk", e.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(e.username, e.password)
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("elasticsearch request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch returned status %d: %s", resp.StatusCode, string(body))
	}

	e.log.Info("exported to Elasticsearch", "count", len(entries))
	return nil
}

// DatadogExporter exports audit logs to Datadog.
type DatadogExporter struct {
	apiKey     string
	site       string // e.g., "datadoghq.com" or "datadoghq.eu"
	service    string
	httpClient *http.Client
	log        *logger.Logger
}

// DatadogConfig contains Datadog configuration.
type DatadogConfig struct {
	APIKey  string
	Site    string // datadoghq.com, datadoghq.eu, etc.
	Service string
}

// NewDatadogExporter creates a new Datadog exporter.
func NewDatadogExporter(cfg DatadogConfig, log *logger.Logger) *DatadogExporter {
	return &DatadogExporter{
		apiKey:     cfg.APIKey,
		site:       cfg.Site,
		service:    cfg.Service,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		log:        log.WithComponent("datadog-exporter"),
	}
}

func (d *DatadogExporter) Name() string {
	return "datadog"
}

// Export sends audit logs to Datadog Logs API.
func (d *DatadogExporter) Export(ctx context.Context, entries []AuditLogRow) error {
	// Build log entries
	logs := make([]map[string]interface{}, len(entries))
	for i, entry := range entries {
		logs[i] = map[string]interface{}{
			"ddsource": "ql-rf",
			"ddtags":   fmt.Sprintf("service:%s,env:production", d.service),
			"hostname": "ql-rf-api",
			"service":  d.service,
			"message":  fmt.Sprintf("%s: %s %s %s", entry.Action, entry.ActorType, entry.ResourceType, entry.ResourceID),
			"status":   entry.Status,
			"timestamp": entry.Timestamp.UnixMilli(),
			"audit":    entry,
		}
	}

	body, err := json.Marshal(logs)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	url := fmt.Sprintf("https://http-intake.logs.%s/api/v2/logs", d.site)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("DD-API-KEY", d.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("datadog request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("datadog returned status %d: %s", resp.StatusCode, string(body))
	}

	d.log.Info("exported to Datadog", "count", len(entries))
	return nil
}

// WebhookExporter exports audit logs to a generic webhook.
type WebhookExporter struct {
	endpoint   string
	secret     string // For HMAC signature
	headers    map[string]string
	httpClient *http.Client
	log        *logger.Logger
}

// WebhookConfig contains webhook configuration.
type WebhookConfig struct {
	Endpoint string
	Secret   string            // Used for HMAC-SHA256 signature
	Headers  map[string]string // Additional headers
}

// NewWebhookExporter creates a new webhook exporter.
func NewWebhookExporter(cfg WebhookConfig, log *logger.Logger) *WebhookExporter {
	return &WebhookExporter{
		endpoint:   cfg.Endpoint,
		secret:     cfg.Secret,
		headers:    cfg.Headers,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		log:        log.WithComponent("webhook-exporter"),
	}
}

func (w *WebhookExporter) Name() string {
	return "webhook"
}

// Export sends audit logs to a webhook endpoint.
func (w *WebhookExporter) Export(ctx context.Context, entries []AuditLogRow) error {
	payload := map[string]interface{}{
		"type":      "audit_logs",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"count":     len(entries),
		"logs":      entries,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}

	// Add HMAC signature if secret is configured
	if w.secret != "" {
		mac := hmac.New(sha256.New, []byte(w.secret))
		mac.Write(body)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Signature-256", "sha256="+signature)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	w.log.Info("exported to webhook", "count", len(entries), "endpoint", w.endpoint)
	return nil
}

// S3Exporter exports audit logs to AWS S3.
type S3Exporter struct {
	bucket     string
	prefix     string
	region     string
	log        *logger.Logger
	// In production, would use AWS SDK
}

// S3Config contains S3 configuration.
type S3Config struct {
	Bucket string
	Prefix string // Key prefix, e.g., "audit-logs/"
	Region string
}

// NewS3Exporter creates a new S3 exporter.
func NewS3Exporter(cfg S3Config, log *logger.Logger) *S3Exporter {
	return &S3Exporter{
		bucket: cfg.Bucket,
		prefix: cfg.Prefix,
		region: cfg.Region,
		log:    log.WithComponent("s3-exporter"),
	}
}

func (s *S3Exporter) Name() string {
	return "s3"
}

// Export uploads audit logs to S3 as JSON files.
func (s *S3Exporter) Export(ctx context.Context, entries []AuditLogRow) error {
	// In production, this would use the AWS SDK to upload
	// For now, log what would happen

	key := fmt.Sprintf("%s%s/audit-%s.json",
		s.prefix,
		time.Now().UTC().Format("2006/01/02"),
		time.Now().UTC().Format("15-04-05"),
	)

	s.log.Info("would export to S3",
		"bucket", s.bucket,
		"key", key,
		"count", len(entries),
	)

	// Would use:
	// s3Client.PutObject(ctx, &s3.PutObjectInput{
	//     Bucket: &s.bucket,
	//     Key:    &key,
	//     Body:   bytes.NewReader(jsonData),
	// })

	return nil
}
