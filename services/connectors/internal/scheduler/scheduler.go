// Package scheduler provides database-driven connector sync scheduling.
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/aws"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/azure"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/connector"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/gcp"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/k8s"
	syncpkg "github.com/quantumlayerhq/ql-rf/services/connectors/internal/sync"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/vsphere"
)

// ConnectorRecord represents a connector from the database.
type ConnectorRecord struct {
	ID                 uuid.UUID
	OrgID              uuid.UUID
	Name               string
	Platform           string
	Enabled            bool
	Config             json.RawMessage
	SyncSchedule       string
	SyncEnabled        bool
	NextSyncAt         *time.Time
	SyncTimeoutSeconds int
}

// SyncHistoryRecord represents a sync history entry.
type SyncHistoryRecord struct {
	ID               uuid.UUID
	ConnectorID      uuid.UUID
	OrgID            uuid.UUID
	StartedAt        time.Time
	CompletedAt      *time.Time
	DurationMs       *int
	Status           string
	AssetsDiscovered int
	AssetsCreated    int
	AssetsUpdated    int
	AssetsRemoved    int
	ImagesDiscovered int
	ErrorMessage     *string
	ErrorCode        *string
	TriggerType      string
	Metadata         json.RawMessage
}

// Scheduler manages database-driven connector sync scheduling.
type Scheduler struct {
	pool         *pgxpool.Pool
	syncSvc      *syncpkg.Service
	log          *logger.Logger
	pollInterval time.Duration

	// Active sync tracking
	activeSyncs sync.Map // connectorID -> true

	// Shutdown management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Config holds scheduler configuration.
type Config struct {
	PollInterval time.Duration
	MaxConcurrent int
}

// DefaultConfig returns default scheduler configuration.
func DefaultConfig() Config {
	return Config{
		PollInterval:  30 * time.Second,
		MaxConcurrent: 5,
	}
}

// New creates a new database-driven scheduler.
func New(pool *pgxpool.Pool, syncSvc *syncpkg.Service, log *logger.Logger, cfg Config) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		pool:         pool,
		syncSvc:      syncSvc,
		log:          log.WithComponent("scheduler"),
		pollInterval: cfg.PollInterval,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins the scheduler loop.
func (s *Scheduler) Start() {
	s.log.Info("starting database-driven scheduler", "poll_interval", s.pollInterval.String())
	s.wg.Add(1)
	go s.pollLoop()
}

// Stop gracefully shuts down the scheduler.
func (s *Scheduler) Stop() {
	s.log.Info("stopping scheduler")
	s.cancel()
	s.wg.Wait()
	s.log.Info("scheduler stopped")
}

// pollLoop continuously checks for connectors that need syncing.
func (s *Scheduler) pollLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	// Run initial check
	s.checkAndSyncDue()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAndSyncDue()
		}
	}
}

// checkAndSyncDue finds connectors due for sync and triggers them.
func (s *Scheduler) checkAndSyncDue() {
	ctx, cancel := context.WithTimeout(s.ctx, 60*time.Second)
	defer cancel()

	connectors, err := s.getDueConnectors(ctx)
	if err != nil {
		s.log.Error("failed to get due connectors", "error", err)
		return
	}

	if len(connectors) == 0 {
		s.log.Debug("no connectors due for sync")
		return
	}

	s.log.Info("found connectors due for sync", "count", len(connectors))

	for _, conn := range connectors {
		// Skip if already syncing
		if _, syncing := s.activeSyncs.Load(conn.ID); syncing {
			s.log.Debug("connector already syncing, skipping", "connector_id", conn.ID)
			continue
		}

		// Mark as active and sync in background
		s.activeSyncs.Store(conn.ID, true)
		s.wg.Add(1)
		go func(c ConnectorRecord) {
			defer s.wg.Done()
			defer s.activeSyncs.Delete(c.ID)
			s.syncConnector(c, "scheduled")
		}(conn)
	}
}

// getDueConnectors retrieves connectors that are due for sync.
func (s *Scheduler) getDueConnectors(ctx context.Context) ([]ConnectorRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, org_id, name, platform, enabled, config,
		       COALESCE(sync_schedule, '1h') as sync_schedule,
		       COALESCE(sync_enabled, true) as sync_enabled,
		       next_sync_at,
		       COALESCE(sync_timeout_seconds, 300) as sync_timeout_seconds
		FROM connectors
		WHERE enabled = true
		  AND COALESCE(sync_enabled, true) = true
		  AND (next_sync_at IS NULL OR next_sync_at <= NOW())
		ORDER BY next_sync_at ASC NULLS FIRST
		LIMIT 10
	`)
	if err != nil {
		return nil, fmt.Errorf("query due connectors: %w", err)
	}
	defer rows.Close()

	var connectors []ConnectorRecord
	for rows.Next() {
		var c ConnectorRecord
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.Name, &c.Platform, &c.Enabled, &c.Config,
			&c.SyncSchedule, &c.SyncEnabled, &c.NextSyncAt, &c.SyncTimeoutSeconds,
		); err != nil {
			return nil, fmt.Errorf("scan connector: %w", err)
		}
		connectors = append(connectors, c)
	}

	return connectors, rows.Err()
}

// syncConnector performs a sync for a single connector.
func (s *Scheduler) syncConnector(conn ConnectorRecord, triggerType string) {
	startTime := time.Now()
	s.log.Info("starting connector sync",
		"connector_id", conn.ID,
		"connector_name", conn.Name,
		"platform", conn.Platform,
		"trigger", triggerType,
	)

	// Create sync history record
	historyID, err := s.createSyncHistory(conn.ID, conn.OrgID, triggerType)
	if err != nil {
		s.log.Error("failed to create sync history", "error", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(s.ctx, time.Duration(conn.SyncTimeoutSeconds)*time.Second)
	defer cancel()

	// Create connector instance
	cloudConn, err := s.createConnector(conn)
	if err != nil {
		s.completeSyncHistory(historyID, "failed", nil, err)
		s.updateNextSyncAt(conn.ID, conn.SyncSchedule)
		s.log.Error("failed to create connector",
			"connector_id", conn.ID,
			"error", err,
		)
		return
	}
	defer cloudConn.Close()

	// Connect to the platform
	if err := cloudConn.Connect(ctx); err != nil {
		s.completeSyncHistory(historyID, "failed", nil, err)
		s.updateNextSyncAt(conn.ID, conn.SyncSchedule)
		s.log.Error("failed to connect to platform",
			"connector_id", conn.ID,
			"platform", conn.Platform,
			"error", err,
		)
		return
	}

	// Discover assets
	assets, err := cloudConn.DiscoverAssets(ctx, conn.OrgID)
	if err != nil {
		s.completeSyncHistory(historyID, "failed", nil, err)
		s.updateNextSyncAt(conn.ID, conn.SyncSchedule)
		s.log.Error("failed to discover assets",
			"connector_id", conn.ID,
			"error", err,
		)
		return
	}

	s.log.Info("discovered assets",
		"connector_id", conn.ID,
		"count", len(assets),
	)

	// Sync to database
	result, err := s.syncSvc.SyncAssets(ctx, conn.OrgID, conn.Platform, assets)
	if err != nil {
		s.completeSyncHistory(historyID, "failed", nil, err)
		s.updateNextSyncAt(conn.ID, conn.SyncSchedule)
		s.log.Error("failed to sync assets",
			"connector_id", conn.ID,
			"error", err,
		)
		return
	}

	// Update connector sync status
	duration := time.Since(startTime)
	s.updateConnectorSyncStatus(conn.ID, conn.OrgID, "completed", "")

	// Complete sync history
	s.completeSyncHistory(historyID, "completed", result, nil)

	// Update next sync time
	s.updateNextSyncAt(conn.ID, conn.SyncSchedule)

	s.log.Info("connector sync completed",
		"connector_id", conn.ID,
		"connector_name", conn.Name,
		"duration", duration.String(),
		"assets_found", result.AssetsFound,
		"assets_new", result.AssetsNew,
		"assets_updated", result.AssetsUpdated,
		"assets_removed", result.AssetsRemoved,
	)
}

// createConnector creates a cloud connector instance from a database record.
func (s *Scheduler) createConnector(conn ConnectorRecord) (connector.Connector, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(conn.Config, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	switch conn.Platform {
	case "aws":
		return s.createAWSConnector(config)
	case "azure":
		return s.createAzureConnector(config)
	case "gcp":
		return s.createGCPConnector(config)
	case "vsphere":
		return s.createVSphereConnector(config)
	case "kubernetes", "k8s":
		return s.createK8sConnector(config)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", conn.Platform)
	}
}

func (s *Scheduler) createAWSConnector(config map[string]interface{}) (connector.Connector, error) {
	region := getString(config, "region")
	assumeRoleARN := getString(config, "assume_role_arn")
	externalID := getString(config, "external_id")

	cfg := aws.Config{
		Region:        region,
		AssumeRoleARN: assumeRoleARN,
		ExternalID:    externalID,
	}

	// Parse additional regions
	if regions, ok := config["regions"].([]interface{}); ok {
		for _, r := range regions {
			if rs, ok := r.(string); ok {
				cfg.Regions = append(cfg.Regions, rs)
			}
		}
	}
	if len(cfg.Regions) == 0 && region != "" {
		cfg.Regions = []string{region}
	}

	return aws.New(cfg, s.log), nil
}

func (s *Scheduler) createAzureConnector(config map[string]interface{}) (connector.Connector, error) {
	cfg := azure.Config{
		SubscriptionID: getString(config, "subscription_id"),
		TenantID:       getString(config, "tenant_id"),
		ClientID:       getString(config, "client_id"),
		ClientSecret:   getString(config, "client_secret"),
	}
	return azure.New(cfg, s.log), nil
}

func (s *Scheduler) createGCPConnector(config map[string]interface{}) (connector.Connector, error) {
	cfg := gcp.Config{
		ProjectID:       getString(config, "project_id"),
		CredentialsFile: getString(config, "credentials_file"),
	}

	// Parse zones if provided
	if zones, ok := config["zones"].([]interface{}); ok {
		for _, z := range zones {
			if zs, ok := z.(string); ok {
				cfg.Zones = append(cfg.Zones, zs)
			}
		}
	}

	return gcp.New(cfg, s.log), nil
}

func (s *Scheduler) createVSphereConnector(config map[string]interface{}) (connector.Connector, error) {
	host := getString(config, "host")
	url := host
	if url != "" && !hasScheme(url) {
		url = "https://" + url
	}
	if url != "" && !hasPath(url, "/sdk") {
		url = url + "/sdk"
	}

	cfg := vsphere.Config{
		URL:      url,
		User:     getString(config, "username"),
		Password: getString(config, "password"),
		Insecure: true,
	}

	// Parse datacenters if provided
	if dc := getString(config, "datacenter"); dc != "" {
		cfg.Datacenters = []string{dc}
	}
	if dcs, ok := config["datacenters"].([]interface{}); ok {
		for _, d := range dcs {
			if ds, ok := d.(string); ok {
				cfg.Datacenters = append(cfg.Datacenters, ds)
			}
		}
	}

	// Parse clusters if provided
	if clusters, ok := config["clusters"].([]interface{}); ok {
		for _, c := range clusters {
			if cs, ok := c.(string); ok {
				cfg.Clusters = append(cfg.Clusters, cs)
			}
		}
	}

	return vsphere.New(cfg, s.log), nil
}

func (s *Scheduler) createK8sConnector(config map[string]interface{}) (connector.Connector, error) {
	cfg := k8s.Config{
		Kubeconfig:          getString(config, "kubeconfig"),
		Context:             getString(config, "context"),
		ClusterName:         getString(config, "cluster_name"),
		DiscoverNodes:       true,
		DiscoverDeployments: true,
	}

	// Parse namespaces
	if ns, ok := config["namespaces"].([]interface{}); ok {
		for _, n := range ns {
			if s, ok := n.(string); ok {
				cfg.Namespaces = append(cfg.Namespaces, s)
			}
		}
	}

	return k8s.New(cfg, s.log), nil
}

// createSyncHistory creates a new sync history record.
func (s *Scheduler) createSyncHistory(connectorID, orgID uuid.UUID, triggerType string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(s.ctx, `
		INSERT INTO connector_sync_history (connector_id, org_id, trigger_type, status)
		VALUES ($1, $2, $3, 'running')
		RETURNING id
	`, connectorID, orgID, triggerType).Scan(&id)
	return id, err
}

// completeSyncHistory updates a sync history record with results.
func (s *Scheduler) completeSyncHistory(historyID uuid.UUID, status string, result *syncpkg.SyncResult, syncErr error) {
	if historyID == uuid.Nil {
		return
	}

	var errorMsg *string
	if syncErr != nil {
		msg := syncErr.Error()
		errorMsg = &msg
	}

	var assetsDiscovered, assetsCreated, assetsUpdated, assetsRemoved int
	var durationMs int
	if result != nil {
		assetsDiscovered = result.AssetsFound
		assetsCreated = result.AssetsNew
		assetsUpdated = result.AssetsUpdated
		assetsRemoved = result.AssetsRemoved
		durationMs = int(result.Duration.Milliseconds())
	}

	_, err := s.pool.Exec(s.ctx, `
		UPDATE connector_sync_history
		SET completed_at = NOW(),
		    duration_ms = $2,
		    status = $3,
		    assets_discovered = $4,
		    assets_created = $5,
		    assets_updated = $6,
		    assets_removed = $7,
		    error_message = $8
		WHERE id = $1
	`, historyID, durationMs, status, assetsDiscovered, assetsCreated, assetsUpdated, assetsRemoved, errorMsg)
	if err != nil {
		s.log.Error("failed to complete sync history", "history_id", historyID, "error", err)
	}
}

// updateConnectorSyncStatus updates the connector's last sync info.
func (s *Scheduler) updateConnectorSyncStatus(connectorID, orgID uuid.UUID, status, errMsg string) {
	var errorPtr *string
	if errMsg != "" {
		errorPtr = &errMsg
	}

	_, err := s.pool.Exec(s.ctx, `
		UPDATE connectors
		SET last_sync_at = NOW(),
		    last_sync_status = $3,
		    last_sync_error = $4,
		    updated_at = NOW()
		WHERE id = $1 AND org_id = $2
	`, connectorID, orgID, status, errorPtr)
	if err != nil {
		s.log.Error("failed to update connector sync status",
			"connector_id", connectorID,
			"error", err,
		)
	}
}

// updateNextSyncAt calculates and updates the next sync time.
func (s *Scheduler) updateNextSyncAt(connectorID uuid.UUID, schedule string) {
	interval := parseScheduleInterval(schedule)
	nextSync := time.Now().Add(interval)

	_, err := s.pool.Exec(s.ctx, `
		UPDATE connectors
		SET next_sync_at = $2, updated_at = NOW()
		WHERE id = $1
	`, connectorID, nextSync)
	if err != nil {
		s.log.Error("failed to update next sync time",
			"connector_id", connectorID,
			"error", err,
		)
	}
}

// TriggerSync manually triggers a sync for a connector.
func (s *Scheduler) TriggerSync(connectorID, orgID uuid.UUID) error {
	// Get connector from database
	var conn ConnectorRecord
	err := s.pool.QueryRow(s.ctx, `
		SELECT id, org_id, name, platform, enabled, config,
		       COALESCE(sync_schedule, '1h') as sync_schedule,
		       COALESCE(sync_enabled, true) as sync_enabled,
		       next_sync_at,
		       COALESCE(sync_timeout_seconds, 300) as sync_timeout_seconds
		FROM connectors
		WHERE id = $1 AND org_id = $2
	`, connectorID, orgID).Scan(
		&conn.ID, &conn.OrgID, &conn.Name, &conn.Platform, &conn.Enabled, &conn.Config,
		&conn.SyncSchedule, &conn.SyncEnabled, &conn.NextSyncAt, &conn.SyncTimeoutSeconds,
	)
	if err != nil {
		return fmt.Errorf("get connector: %w", err)
	}

	// Check if already syncing
	if _, syncing := s.activeSyncs.Load(conn.ID); syncing {
		return fmt.Errorf("connector is already syncing")
	}

	// Run sync in background
	s.activeSyncs.Store(conn.ID, true)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer s.activeSyncs.Delete(conn.ID)
		s.syncConnector(conn, "manual")
	}()

	return nil
}

// parseScheduleInterval parses a schedule string into a duration.
func parseScheduleInterval(schedule string) time.Duration {
	// Try parsing as Go duration
	if d, err := time.ParseDuration(schedule); err == nil {
		return d
	}

	// Default to 1 hour
	return time.Hour
}

// getString safely extracts a string from a map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func hasScheme(url string) bool {
	return len(url) > 8 && (url[:7] == "http://" || url[:8] == "https://")
}

func hasPath(url, path string) bool {
	return len(url) >= len(path) && url[len(url)-len(path):] == path
}

// DiscoverAssetsForConnector is a helper for manual discovery via API.
func (s *Scheduler) DiscoverAssetsForConnector(ctx context.Context, conn ConnectorRecord) ([]models.NormalizedAsset, error) {
	cloudConn, err := s.createConnector(conn)
	if err != nil {
		return nil, fmt.Errorf("create connector: %w", err)
	}
	defer cloudConn.Close()

	if err := cloudConn.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	return cloudConn.DiscoverAssets(ctx, conn.OrgID)
}
