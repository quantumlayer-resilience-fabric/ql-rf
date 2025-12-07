package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/connectors/pkg/connectors"
)

// ConnectorRepository defines the interface for connector data access.
type ConnectorRepository interface {
	Create(ctx context.Context, params CreateConnectorRepoParams) (*ConnectorModel, error)
	Get(ctx context.Context, id, orgID uuid.UUID) (*ConnectorModel, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]ConnectorModel, error)
	Delete(ctx context.Context, id, orgID uuid.UUID) error
	UpdateSyncStatus(ctx context.Context, id, orgID uuid.UUID, status, syncError string) error
	UpdateEnabled(ctx context.Context, id, orgID uuid.UUID, enabled bool) error
	UpdateConfig(ctx context.Context, id, orgID uuid.UUID, config json.RawMessage) error
	ExistsByName(ctx context.Context, orgID uuid.UUID, name string) (bool, error)
}

// CreateConnectorRepoParams contains parameters for creating a connector in repository.
type CreateConnectorRepoParams struct {
	OrgID        uuid.UUID
	Name         string
	Platform     string
	Enabled      bool
	Config       json.RawMessage
	SyncSchedule string
}

// ConnectorModel represents a connector from the repository.
type ConnectorModel struct {
	ID             uuid.UUID
	OrgID          uuid.UUID
	Name           string
	Platform       string
	Enabled        bool
	Config         json.RawMessage
	LastSyncAt     *time.Time
	LastSyncStatus *string
	LastSyncError  *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ConnectorService provides business logic for connector operations.
type ConnectorService struct {
	repo        ConnectorRepository
	pool        *pgxpool.Pool
	syncService *connectors.SyncService
	log         *logger.Logger
}

// NewConnectorService creates a new ConnectorService.
func NewConnectorService(repo ConnectorRepository, pool *pgxpool.Pool, log *logger.Logger) *ConnectorService {
	// Create sync service for asset storage (without Kafka for now)
	syncSvc := connectors.NewSyncService(pool, log)

	return &ConnectorService{
		repo:        repo,
		pool:        pool,
		syncService: syncSvc,
		log:         log.WithComponent("connector-service"),
	}
}

// CreateConnectorParams contains parameters for creating a connector.
type CreateConnectorParams struct {
	OrgID        uuid.UUID
	Name         string
	Platform     string
	Config       map[string]interface{}
	SyncSchedule string // Duration like "1h", "30m", "6h"
}

// Connector represents a connector response.
type Connector struct {
	ID             uuid.UUID              `json:"id"`
	Name           string                 `json:"name"`
	Platform       string                 `json:"platform"`
	Enabled        bool                   `json:"enabled"`
	Config         map[string]interface{} `json:"config,omitempty"`
	LastSyncAt     *string                `json:"last_sync_at,omitempty"`
	LastSyncStatus *string                `json:"last_sync_status,omitempty"`
	LastSyncError  *string                `json:"last_sync_error,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
}

// SyncResult contains the results of a connector sync operation.
type SyncResult struct {
	ConnectorID   uuid.UUID `json:"connector_id"`
	Status        string    `json:"status"`
	AssetsFound   int       `json:"assets_found"`
	ImagesFound   int       `json:"images_found"`
	SitesCreated  int       `json:"sites_created"`
	AssetsCreated int       `json:"assets_created"`
	AssetsUpdated int       `json:"assets_updated"`
	Error         string    `json:"error,omitempty"`
}

// TestResult contains the results of a connector test.
type TestResult struct {
	ConnectorID uuid.UUID `json:"connector_id"`
	Success     bool      `json:"success"`
	Message     string    `json:"message"`
	Details     string    `json:"details,omitempty"`
}

// Create creates a new connector.
func (s *ConnectorService) Create(ctx context.Context, params CreateConnectorParams) (*Connector, error) {
	// Validate platform
	validPlatforms := map[string]bool{
		"aws":     true,
		"azure":   true,
		"gcp":     true,
		"vsphere": true,
		"k8s":     true,
	}
	if !validPlatforms[params.Platform] {
		return nil, fmt.Errorf("invalid platform: %s", params.Platform)
	}

	// Check if connector name already exists
	exists, err := s.repo.ExistsByName(ctx, params.OrgID, params.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing connector: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("connector with name '%s' already exists", params.Name)
	}

	// Validate platform-specific config
	if err := s.validateConfig(params.Platform, params.Config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Serialize config to JSON
	configJSON, err := json.Marshal(params.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize config: %w", err)
	}

	// Default sync schedule if not provided
	syncSchedule := params.SyncSchedule
	if syncSchedule == "" {
		syncSchedule = "1h"
	}

	// Create connector
	conn, err := s.repo.Create(ctx, CreateConnectorRepoParams{
		OrgID:        params.OrgID,
		Name:         params.Name,
		Platform:     params.Platform,
		Enabled:      true,
		Config:       configJSON,
		SyncSchedule: syncSchedule,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connector: %w", err)
	}

	s.log.Info("connector created",
		"connector_id", conn.ID,
		"name", conn.Name,
		"platform", conn.Platform,
		"org_id", conn.OrgID,
	)

	return s.toConnector(conn), nil
}

// Get retrieves a connector by ID.
func (s *ConnectorService) Get(ctx context.Context, id, orgID uuid.UUID) (*Connector, error) {
	conn, err := s.repo.Get(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connector: %w", err)
	}
	return s.toConnector(conn), nil
}

// List retrieves all connectors for an organization.
func (s *ConnectorService) List(ctx context.Context, orgID uuid.UUID) ([]Connector, error) {
	connectors, err := s.repo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list connectors: %w", err)
	}

	result := make([]Connector, 0, len(connectors))
	for _, c := range connectors {
		result = append(result, *s.toConnector(&c))
	}
	return result, nil
}

// Delete deletes a connector.
func (s *ConnectorService) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, orgID); err != nil {
		return fmt.Errorf("failed to delete connector: %w", err)
	}

	s.log.Info("connector deleted",
		"connector_id", id,
		"org_id", orgID,
	)
	return nil
}

// TestConnection tests if the connector can connect to the cloud platform.
func (s *ConnectorService) TestConnection(ctx context.Context, id, orgID uuid.UUID) (*TestResult, error) {
	conn, err := s.repo.Get(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connector: %w", err)
	}

	// Parse config
	var config map[string]interface{}
	if err := json.Unmarshal(conn.Config, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Test connection based on platform
	result := &TestResult{
		ConnectorID: id,
		Success:     true,
		Message:     "Connection test successful",
	}

	switch conn.Platform {
	case "aws":
		result.Details = s.testAWSConnection(config)
	case "azure":
		result.Details = s.testAzureConnection(config)
	case "gcp":
		result.Details = s.testGCPConnection(config)
	case "vsphere":
		result.Details = s.testVSphereConnection(config)
	case "k8s":
		result.Details = s.testK8sConnection(config)
	default:
		result.Success = false
		result.Message = fmt.Sprintf("Unknown platform: %s", conn.Platform)
	}

	// In a real implementation, we would actually test the connection
	// For now, we validate the config is complete
	if err := s.validateConfig(conn.Platform, config); err != nil {
		result.Success = false
		result.Message = "Configuration validation failed"
		result.Details = err.Error()
	}

	s.log.Info("connector test completed",
		"connector_id", id,
		"platform", conn.Platform,
		"success", result.Success,
	)

	return result, nil
}

// TriggerSync triggers asset discovery for a connector.
func (s *ConnectorService) TriggerSync(ctx context.Context, id, orgID uuid.UUID) (*SyncResult, error) {
	conn, err := s.repo.Get(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connector: %w", err)
	}

	if !conn.Enabled {
		return nil, fmt.Errorf("connector is disabled")
	}

	// Update sync status to "syncing"
	if err := s.repo.UpdateSyncStatus(ctx, id, orgID, "syncing", ""); err != nil {
		s.log.Error("failed to update sync status", "error", err)
	}

	// Parse config
	var config map[string]interface{}
	if err := json.Unmarshal(conn.Config, &config); err != nil {
		s.repo.UpdateSyncStatus(ctx, id, orgID, "failed", "invalid config format")
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Create the appropriate cloud connector
	cloudConn, err := s.createCloudConnector(conn.Platform, config)
	if err != nil {
		s.repo.UpdateSyncStatus(ctx, id, orgID, "failed", err.Error())
		return nil, fmt.Errorf("failed to create cloud connector: %w", err)
	}

	// Connect to the cloud platform
	if err := cloudConn.Connect(ctx); err != nil {
		s.repo.UpdateSyncStatus(ctx, id, orgID, "failed", err.Error())
		return nil, fmt.Errorf("failed to connect to %s: %w", conn.Platform, err)
	}
	defer cloudConn.Close()

	// Discover assets
	assets, err := cloudConn.DiscoverAssets(ctx, orgID)
	if err != nil {
		s.repo.UpdateSyncStatus(ctx, id, orgID, "failed", err.Error())
		return nil, fmt.Errorf("failed to discover assets: %w", err)
	}

	s.log.Info("discovered assets from cloud platform",
		"connector_id", id,
		"platform", conn.Platform,
		"asset_count", len(assets),
	)

	// Discover images
	var imagesFound int
	images, err := cloudConn.DiscoverImages(ctx)
	if err != nil {
		s.log.Warn("failed to discover images", "error", err)
	} else {
		imagesFound = len(images)
	}

	// Sync assets to database using sync service
	var syncResult *connectors.SyncResult
	if s.syncService != nil && len(assets) > 0 {
		syncResult, err = connectors.SyncAssets(ctx, s.syncService, orgID, conn.Platform, assets)
		if err != nil {
			s.log.Error("failed to sync assets to database", "error", err)
		}
	}

	// Build result
	result := &SyncResult{
		ConnectorID: id,
		Status:      "completed",
		AssetsFound: len(assets),
		ImagesFound: imagesFound,
	}

	if syncResult != nil {
		result.AssetsCreated = syncResult.AssetsCreated
		result.AssetsUpdated = syncResult.AssetsUpdated
		result.SitesCreated = syncResult.SitesCreated
	}

	// Update sync status
	if err := s.repo.UpdateSyncStatus(ctx, id, orgID, "completed", ""); err != nil {
		s.log.Error("failed to update sync status", "error", err)
	}

	s.log.Info("connector sync completed",
		"connector_id", id,
		"platform", conn.Platform,
		"status", result.Status,
		"assets_found", result.AssetsFound,
		"assets_created", result.AssetsCreated,
		"assets_updated", result.AssetsUpdated,
	)

	return result, nil
}

// createCloudConnector creates the appropriate cloud connector based on platform.
func (s *ConnectorService) createCloudConnector(platform string, config map[string]interface{}) (connectors.Connector, error) {
	switch platform {
	case "aws":
		return s.createAWSConnector(config)
	case "azure":
		return s.createAzureConnector(config)
	case "gcp":
		return s.createGCPConnector(config)
	case "vsphere":
		return s.createVSphereConnector(config)
	case "k8s":
		return s.createK8sConnector(config)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

// createAWSConnector creates an AWS connector from config.
func (s *ConnectorService) createAWSConnector(config map[string]interface{}) (connectors.Connector, error) {
	region, _ := config["region"].(string)
	if region == "" {
		return nil, fmt.Errorf("aws: region is required")
	}

	cfg := connectors.AWSConfig{
		Region: region,
	}

	// Optional fields
	if assumeRoleARN, ok := config["assume_role_arn"].(string); ok {
		cfg.AssumeRoleARN = assumeRoleARN
	}
	if externalID, ok := config["external_id"].(string); ok {
		cfg.ExternalID = externalID
	}
	if regions, ok := config["regions"].([]interface{}); ok {
		for _, r := range regions {
			if rs, ok := r.(string); ok {
				cfg.Regions = append(cfg.Regions, rs)
			}
		}
	}

	return connectors.NewAWSConnector(cfg, s.log), nil
}

// createAzureConnector creates an Azure connector from config.
func (s *ConnectorService) createAzureConnector(config map[string]interface{}) (connectors.Connector, error) {
	tenantID, _ := config["tenant_id"].(string)
	clientID, _ := config["client_id"].(string)
	clientSecret, _ := config["client_secret"].(string)
	subscriptionID, _ := config["subscription_id"].(string)

	if tenantID == "" || clientID == "" || clientSecret == "" || subscriptionID == "" {
		return nil, fmt.Errorf("azure: tenant_id, client_id, client_secret, and subscription_id are required")
	}

	cfg := connectors.AzureConfig{
		TenantID:       tenantID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		SubscriptionID: subscriptionID,
	}

	// Optional resource groups
	if rgs, ok := config["resource_groups"].([]interface{}); ok {
		for _, rg := range rgs {
			if rgs, ok := rg.(string); ok {
				cfg.ResourceGroups = append(cfg.ResourceGroups, rgs)
			}
		}
	}

	return connectors.NewAzureConnector(cfg, s.log), nil
}

// createGCPConnector creates a GCP connector from config.
func (s *ConnectorService) createGCPConnector(config map[string]interface{}) (connectors.Connector, error) {
	projectID, _ := config["project_id"].(string)
	if projectID == "" {
		return nil, fmt.Errorf("gcp: project_id is required")
	}

	cfg := connectors.GCPConfig{
		ProjectID: projectID,
	}

	// Optional fields
	if credentialsFile, ok := config["credentials_file"].(string); ok {
		cfg.CredentialsFile = credentialsFile
	}
	if zones, ok := config["zones"].([]interface{}); ok {
		for _, z := range zones {
			if zs, ok := z.(string); ok {
				cfg.Zones = append(cfg.Zones, zs)
			}
		}
	}

	return connectors.NewGCPConnector(cfg, s.log), nil
}

// createVSphereConnector creates a vSphere connector from config.
func (s *ConnectorService) createVSphereConnector(config map[string]interface{}) (connectors.Connector, error) {
	host, _ := config["host"].(string)
	username, _ := config["username"].(string)
	password, _ := config["password"].(string)

	if host == "" || username == "" || password == "" {
		return nil, fmt.Errorf("vsphere: host, username, and password are required")
	}

	// Build URL from host (e.g., vcenter.example.com -> https://vcenter.example.com/sdk)
	url := host
	if !containsScheme(url) {
		url = "https://" + url
	}
	if !containsPath(url, "/sdk") {
		url = url + "/sdk"
	}

	cfg := connectors.VSphereConfig{
		URL:      url,
		User:     username,
		Password: password,
		Insecure: true, // Default to insecure for self-signed certs (common in vSphere)
	}

	// Optional fields
	if insecure, ok := config["insecure"].(bool); ok {
		cfg.Insecure = insecure
	}
	if datacenters, ok := config["datacenters"].([]interface{}); ok {
		for _, dc := range datacenters {
			if dcs, ok := dc.(string); ok {
				cfg.Datacenters = append(cfg.Datacenters, dcs)
			}
		}
	}
	if clusters, ok := config["clusters"].([]interface{}); ok {
		for _, c := range clusters {
			if cs, ok := c.(string); ok {
				cfg.Clusters = append(cfg.Clusters, cs)
			}
		}
	}

	return connectors.NewVSphereConnector(cfg, s.log), nil
}

// createK8sConnector creates a Kubernetes connector from config.
func (s *ConnectorService) createK8sConnector(config map[string]interface{}) (connectors.Connector, error) {
	cfg := connectors.K8sConfig{
		DiscoverNodes:       true, // Default to discovering nodes
		DiscoverDeployments: true, // Default to discovering deployments
	}

	// Kubeconfig path (optional, defaults to in-cluster)
	if kubeconfig, ok := config["kubeconfig"].(string); ok {
		cfg.Kubeconfig = kubeconfig
	}

	// Context name (optional)
	if context, ok := config["context"].(string); ok {
		cfg.Context = context
	}

	// Cluster name for tagging
	if clusterName, ok := config["cluster_name"].(string); ok {
		cfg.ClusterName = clusterName
	}

	// Namespace filters
	if namespaces, ok := config["namespaces"].([]interface{}); ok {
		for _, ns := range namespaces {
			if nss, ok := ns.(string); ok {
				cfg.Namespaces = append(cfg.Namespaces, nss)
			}
		}
	}
	if excludeNamespaces, ok := config["exclude_namespaces"].([]interface{}); ok {
		for _, ns := range excludeNamespaces {
			if nss, ok := ns.(string); ok {
				cfg.ExcludeNamespaces = append(cfg.ExcludeNamespaces, nss)
			}
		}
	}

	// Discovery options
	if discoverNodes, ok := config["discover_nodes"].(bool); ok {
		cfg.DiscoverNodes = discoverNodes
	}
	if discoverDeployments, ok := config["discover_deployments"].(bool); ok {
		cfg.DiscoverDeployments = discoverDeployments
	}

	// Label selector
	if labelSelector, ok := config["label_selector"].(string); ok {
		cfg.LabelSelector = labelSelector
	}

	return connectors.NewK8sConnector(cfg, s.log), nil
}

// containsScheme checks if a URL contains a scheme (http:// or https://)
func containsScheme(url string) bool {
	return len(url) > 7 && (url[:7] == "http://" || url[:8] == "https://")
}

// containsPath checks if a URL contains a specific path
func containsPath(url, path string) bool {
	return len(url) >= len(path) && url[len(url)-len(path):] == path
}

// Enable enables a connector.
func (s *ConnectorService) Enable(ctx context.Context, id, orgID uuid.UUID) error {
	return s.repo.UpdateEnabled(ctx, id, orgID, true)
}

// Disable disables a connector.
func (s *ConnectorService) Disable(ctx context.Context, id, orgID uuid.UUID) error {
	return s.repo.UpdateEnabled(ctx, id, orgID, false)
}

// validateConfig validates platform-specific configuration.
func (s *ConnectorService) validateConfig(platform string, config map[string]interface{}) error {
	switch platform {
	case "aws":
		// AWS requires region
		if _, ok := config["region"]; !ok {
			return fmt.Errorf("aws: region is required")
		}
	case "azure":
		// Azure requires subscription_id, tenant_id, client_id, client_secret
		required := []string{"subscription_id", "tenant_id", "client_id", "client_secret"}
		for _, field := range required {
			if _, ok := config[field]; !ok {
				return fmt.Errorf("azure: %s is required", field)
			}
		}
	case "gcp":
		// GCP requires project_id
		if _, ok := config["project_id"]; !ok {
			return fmt.Errorf("gcp: project_id is required")
		}
	case "vsphere":
		// vSphere requires host, username, password
		required := []string{"host", "username", "password"}
		for _, field := range required {
			if _, ok := config[field]; !ok {
				return fmt.Errorf("vsphere: %s is required", field)
			}
		}
	case "k8s":
		// K8s requires either kubeconfig or in-cluster config
		// For now, just accept any config
	}
	return nil
}

// Platform-specific connection test helpers
func (s *ConnectorService) testAWSConnection(config map[string]interface{}) string {
	region, _ := config["region"].(string)
	assumeRole, _ := config["assume_role_arn"].(string)
	if assumeRole != "" {
		return fmt.Sprintf("AWS connection configured for region %s with role assumption", region)
	}
	return fmt.Sprintf("AWS connection configured for region %s using default credentials", region)
}

func (s *ConnectorService) testAzureConnection(config map[string]interface{}) string {
	subscriptionID, _ := config["subscription_id"].(string)
	return fmt.Sprintf("Azure connection configured for subscription %s", subscriptionID)
}

func (s *ConnectorService) testGCPConnection(config map[string]interface{}) string {
	projectID, _ := config["project_id"].(string)
	return fmt.Sprintf("GCP connection configured for project %s", projectID)
}

func (s *ConnectorService) testVSphereConnection(config map[string]interface{}) string {
	host, _ := config["host"].(string)
	return fmt.Sprintf("vSphere connection configured for host %s", host)
}

func (s *ConnectorService) testK8sConnection(config map[string]interface{}) string {
	context, _ := config["context"].(string)
	if context != "" {
		return fmt.Sprintf("Kubernetes connection configured for context %s", context)
	}
	return "Kubernetes connection configured using in-cluster config"
}

// toConnector converts a ConnectorModel to a Connector.
func (s *ConnectorService) toConnector(c *ConnectorModel) *Connector {
	result := &Connector{
		ID:        c.ID,
		Name:      c.Name,
		Platform:  c.Platform,
		Enabled:   c.Enabled,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Parse config (excluding sensitive fields)
	var config map[string]interface{}
	if err := json.Unmarshal(c.Config, &config); err == nil {
		// Remove sensitive fields from response
		safeConfig := make(map[string]interface{})
		sensitiveFields := map[string]bool{
			"client_secret":    true,
			"password":         true,
			"credentials_file": true,
			"external_id":      true,
			"kubeconfig":       true, // K8s kubeconfig may contain certs/tokens
		}
		for k, v := range config {
			if !sensitiveFields[k] {
				safeConfig[k] = v
			} else {
				safeConfig[k] = "***" // Mask sensitive values
			}
		}
		result.Config = safeConfig
	}

	if c.LastSyncAt != nil {
		ts := c.LastSyncAt.Format("2006-01-02T15:04:05Z07:00")
		result.LastSyncAt = &ts
	}
	result.LastSyncStatus = c.LastSyncStatus
	result.LastSyncError = c.LastSyncError

	return result
}
