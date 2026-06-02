// Package config provides configuration management using Viper.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	Env      string `mapstructure:"env"`
	LogLevel string `mapstructure:"log_level"`
	// DevMode (RF_DEV_MODE) forces the API to skip JWT validation even when Clerk
	// is configured. Intended for local/E2E use where the frontend authenticates
	// against Clerk but the API should resolve a deterministic default org.
	// Defaults to false, so production behavior is unchanged.
	DevMode bool `mapstructure:"dev_mode"`

	API           APIConfig          `mapstructure:"api"`
	Database      DatabaseConfig     `mapstructure:"database"`
	Redis         RedisConfig        `mapstructure:"redis"`
	Kafka         KafkaConfig        `mapstructure:"kafka"`
	Clerk         ClerkConfig        `mapstructure:"clerk"`
	Connectors    ConnectorsConfig   `mapstructure:"connectors"`
	Drift         DriftConfig        `mapstructure:"drift"`
	Metrics       MetricsConfig      `mapstructure:"metrics"`
	Orchestrator  OrchestratorConfig `mapstructure:"orchestrator"`
	LLM           LLMConfig          `mapstructure:"llm"`
	Temporal      TemporalConfig     `mapstructure:"temporal"`
	OPA           OPAConfig          `mapstructure:"opa"`
	Notifications NotificationConfig `mapstructure:"notifications"`
	CVE           CVEConfig          `mapstructure:"cve"`
}

// APIConfig holds API server configuration.
type APIConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig holds PostgreSQL configuration.
type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig holds Redis configuration.
type RedisConfig struct {
	URL        string `mapstructure:"url"`
	MaxRetries int    `mapstructure:"max_retries"`
}

// KafkaConfig holds Kafka configuration.
type KafkaConfig struct {
	Brokers       []string `mapstructure:"brokers"`
	ConsumerGroup string   `mapstructure:"consumer_group"`
	Topics        struct {
		AssetDiscovered string `mapstructure:"asset_discovered"`
		DriftDetected   string `mapstructure:"drift_detected"`
		ImagePublished  string `mapstructure:"image_published"`
	} `mapstructure:"topics"`
}

// ClerkConfig holds Clerk authentication configuration.
type ClerkConfig struct {
	SecretKey      string `mapstructure:"secret_key"`
	PublishableKey string `mapstructure:"publishable_key"`
}

// ConnectorsConfig holds connector service configuration.
type ConnectorsConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	SyncInterval time.Duration `mapstructure:"sync_interval"`
	Enabled      []string      `mapstructure:"enabled"`
	OrgID        uuid.UUID     `mapstructure:"org_id"` // Organization ID for asset discovery

	AWS     AWSConfig     `mapstructure:"aws"`
	Azure   AzureConfig   `mapstructure:"azure"`
	GCP     GCPConfig     `mapstructure:"gcp"`
	VSphere VSphereConfig `mapstructure:"vsphere"`
	K8s     K8sConfig     `mapstructure:"k8s"`
}

// AWSConfig holds AWS connector configuration. Shared between the connectors
// service (asset discovery) and the orchestrator (real-tool invocations,
// PR #19 / CONN-001).
type AWSConfig struct {
	Region        string   `mapstructure:"region"`
	AssumeRoleARN string   `mapstructure:"assume_role_arn"`
	Regions       []string `mapstructure:"regions"` // List of regions to scan (optional, discovers all if empty)

	// AssumeRoleExternalID is the optional STS AssumeRole external ID.
	AssumeRoleExternalID string `mapstructure:"assume_role_external_id"`

	// FallbackToMock controls whether the orchestrator's real-tool path
	// (PR #19) falls back to a deterministic mock client when real AWS
	// credentials are absent or invalid. true in dev/CI so the
	// query_aws_instances tool stays registered and demoable without an
	// AWS account; MUST be false in production so a misconfiguration is
	// surfaced loudly at boot instead of silently using fake data.
	FallbackToMock bool `mapstructure:"fallback_to_mock"`

	// AllowLivePatch is the env opt-in for PR #21 / CONN-003 — the first
	// live cloud-mutating tool (ssm_send_patch_command_live). MUST be
	// false in dev/CI; true ONLY in production deployments that want the
	// live SSM SendCommand path. If true and FallbackToMock is also true,
	// the orchestrator REFUSES to start (the combination is incoherent —
	// you cannot mock real cloud mutations).
	AllowLivePatch bool `mapstructure:"allow_live_patch"`

	// LivePatchWhitelistInstanceIDs is the per-instance allowlist for live
	// SSM patches. Parsed from comma-separated env at boot. The tool
	// rejects any instance ID not on this list. Empty + AllowLivePatch=true
	// is also a boot-time refusal: live mode without targets is meaningless.
	LivePatchWhitelistInstanceIDs []string `mapstructure:"live_patch_whitelist_instance_ids"`

	// LivePatchClientMode picks between the real SSM SDK ("real", default)
	// and the deterministic mock ("mock"). "mock" is for local smoke and
	// integration tests where we want to exercise the full live-mode boot
	// path without firing real AWS calls. Production MUST be "real".
	LivePatchClientMode string `mapstructure:"live_patch_client_mode"`
}

// AzureConfig holds Azure connector configuration.
type AzureConfig struct {
	TenantID       string `mapstructure:"tenant_id"`
	ClientID       string `mapstructure:"client_id"`
	ClientSecret   string `mapstructure:"client_secret"`
	SubscriptionID string `mapstructure:"subscription_id"`

	// FallbackToMock controls whether the orchestrator's real-tool path
	// (PR #26 / CONN-006) falls back to a deterministic mock client when
	// real Azure credentials are absent or invalid. true in dev/CI so the
	// query_azure_vms tool stays registered and demoable without an Azure
	// subscription; MUST be false in production so a misconfiguration is
	// surfaced loudly at boot instead of silently using fake data.
	FallbackToMock bool `mapstructure:"fallback_to_mock"`

	// AllowLiveRunCommand is the PR #28 / CONN-008 env opt-in — the first
	// live Azure cloud-mutating tool (azure_run_command_live). MUST be
	// false in dev/CI; true ONLY in production deployments that want the
	// live Azure Run Command path. If true and FallbackToMock is also
	// true, the orchestrator REFUSES to start (incoherent combination,
	// same gate the SSM live path uses).
	AllowLiveRunCommand bool `mapstructure:"allow_live_run_command"`

	// LiveRunCommandWhitelistVMs is the per-VM allowlist for live Azure
	// Run Command invocations. Parsed from comma-separated env at boot.
	// Format: "rg/vm-name" pairs (rg is the resource group). The tool
	// rejects any pair not on this list. Empty + AllowLiveRunCommand=true
	// also fails boot.
	LiveRunCommandWhitelistVMs []string `mapstructure:"live_run_command_whitelist_vms"`

	// LiveRunCommandClientMode picks between the real Azure SDK ("real",
	// default) and the deterministic mock ("mock"). "mock" is for local
	// smoke and integration tests where we want to exercise the full
	// live-mode boot path without firing real Azure calls. Production
	// MUST be "real".
	LiveRunCommandClientMode string `mapstructure:"live_run_command_client_mode"`
}

// GCPConfig holds GCP connector configuration.
type GCPConfig struct {
	ProjectID       string `mapstructure:"project_id"`
	CredentialsFile string `mapstructure:"credentials_file"`

	// FallbackToMock controls whether the orchestrator's real-tool path
	// (PR #29 / CONN-009) falls back to a deterministic mock client when
	// real GCP credentials are absent or invalid. true in dev/CI so the
	// query_gcp_instances tool stays registered and demoable without a
	// GCP project; MUST be false in production so a misconfiguration is
	// surfaced loudly at boot instead of silently using fake data.
	FallbackToMock bool `mapstructure:"fallback_to_mock"`
}

// VSphereConfig holds vSphere connector configuration.
type VSphereConfig struct {
	URL      string `mapstructure:"url"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Insecure bool   `mapstructure:"insecure"`
}

// K8sConfig holds Kubernetes connector configuration.
type K8sConfig struct {
	// Kubeconfig file path. If empty, uses in-cluster config or default kubeconfig location.
	Kubeconfig string `mapstructure:"kubeconfig"`

	// Context to use from kubeconfig. If empty, uses current-context.
	Context string `mapstructure:"context"`

	// Namespaces to scan. If empty, scans all namespaces.
	Namespaces []string `mapstructure:"namespaces"`

	// ExcludeNamespaces to skip during discovery (e.g., kube-system, kube-public).
	ExcludeNamespaces []string `mapstructure:"exclude_namespaces"`

	// DiscoverNodes enables node discovery in addition to pods.
	DiscoverNodes bool `mapstructure:"discover_nodes"`

	// DiscoverDeployments includes deployment metadata in asset discovery.
	DiscoverDeployments bool `mapstructure:"discover_deployments"`

	// LabelSelector to filter pods (e.g., "app=myapp,env=prod").
	LabelSelector string `mapstructure:"label_selector"`

	// ClusterName is an optional friendly name for the cluster.
	ClusterName string `mapstructure:"cluster_name"`
}

// DriftConfig holds drift service configuration.
type DriftConfig struct {
	Host                string        `mapstructure:"host"`
	Port                int           `mapstructure:"port"`
	CalculationInterval time.Duration `mapstructure:"calculation_interval"`
	ThresholdWarning    float64       `mapstructure:"threshold_warning"`
	ThresholdCritical   float64       `mapstructure:"threshold_critical"`
}

// MetricsConfig holds metrics configuration.
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// OrchestratorConfig holds AI orchestrator service configuration.
type OrchestratorConfig struct {
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	Enabled bool   `mapstructure:"enabled"`
	DevMode bool   `mapstructure:"dev_mode"` // Skip auth in development
}

// LLMConfig holds LLM provider configuration.
type LLMConfig struct {
	Provider    string  `mapstructure:"provider"`    // anthropic, azure_openai, openai
	APIKey      string  `mapstructure:"api_key"`     // API key for the provider
	Model       string  `mapstructure:"model"`       // Model name (e.g., claude-3-5-sonnet-20241022)
	MaxTokens   int     `mapstructure:"max_tokens"`  // Maximum tokens for completion
	Temperature float64 `mapstructure:"temperature"` // Temperature for sampling (0.0-2.0)

	// Azure OpenAI specific
	AzureEndpoint   string `mapstructure:"azure_endpoint"`    // Azure OpenAI endpoint
	AzureAPIVersion string `mapstructure:"azure_api_version"` // Azure API version
	AzureDeployment string `mapstructure:"azure_deployment"`  // Azure OpenAI deployment name

	// Azure Anthropic (Microsoft Foundry) specific
	AzureAnthropicEndpoint string `mapstructure:"azure_anthropic_endpoint"` // Azure Anthropic endpoint (e.g., https://<resource>.services.ai.azure.com)

	// Rate limiting
	MaxRequestsPerMinute int `mapstructure:"max_requests_per_minute"`

	// Fallback configuration
	FallbackProvider string `mapstructure:"fallback_provider"` // Fallback provider if primary fails
	FallbackModel    string `mapstructure:"fallback_model"`    // Fallback model
}

// TemporalConfig holds Temporal workflow configuration.
type TemporalConfig struct {
	Host      string `mapstructure:"host"`       // Temporal server host
	Port      int    `mapstructure:"port"`       // Temporal server port
	Namespace string `mapstructure:"namespace"`  // Temporal namespace
	TaskQueue string `mapstructure:"task_queue"` // Task queue name for workers

	// Worker configuration
	WorkerCount             int `mapstructure:"worker_count"`              // Number of workflow workers
	MaxConcurrentWorkflows  int `mapstructure:"max_concurrent_workflows"`  // Max concurrent workflow executions
	MaxConcurrentActivities int `mapstructure:"max_concurrent_activities"` // Max concurrent activity executions

	// TLS (for Temporal Cloud)
	TLSEnabled  bool   `mapstructure:"tls_enabled"`
	TLSCertPath string `mapstructure:"tls_cert_path"`
	TLSKeyPath  string `mapstructure:"tls_key_path"`
}

// OPAConfig holds OPA policy engine configuration.
type OPAConfig struct {
	Enabled     bool   `mapstructure:"enabled"`      // Enable OPA validation
	Mode        string `mapstructure:"mode"`         // embedded or remote
	URL         string `mapstructure:"url"`          // OPA server URL (for remote mode)
	PoliciesDir string `mapstructure:"policies_dir"` // Directory containing .rego files (for embedded mode)

	// Timeout for policy evaluation
	EvalTimeout time.Duration `mapstructure:"eval_timeout"`
}

// NotificationConfig holds notification service configuration.
type NotificationConfig struct {
	// Application base URL for links in notifications
	AppBaseURL string `mapstructure:"app_base_url"`

	// Slack
	SlackEnabled    bool   `mapstructure:"slack_enabled"`
	SlackWebhookURL string `mapstructure:"slack_webhook_url"`
	SlackChannel    string `mapstructure:"slack_channel"`

	// Email
	EmailEnabled    bool     `mapstructure:"email_enabled"`
	SMTPHost        string   `mapstructure:"smtp_host"`
	SMTPPort        int      `mapstructure:"smtp_port"`
	SMTPUser        string   `mapstructure:"smtp_user"`
	SMTPPassword    string   `mapstructure:"smtp_password"`
	EmailFrom       string   `mapstructure:"email_from"`
	EmailRecipients []string `mapstructure:"email_recipients"`

	// Webhook
	WebhookEnabled bool   `mapstructure:"webhook_enabled"`
	WebhookURL     string `mapstructure:"webhook_url"`
	WebhookSecret  string `mapstructure:"webhook_secret"`

	// Microsoft Teams
	TeamsEnabled    bool   `mapstructure:"teams_enabled"`
	TeamsWebhookURL string `mapstructure:"teams_webhook_url"`
}

// CVEConfig holds CVE aggregator service configuration.
type CVEConfig struct {
	// Service configuration
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`

	// Poll interval for checking feeds
	PollInterval time.Duration `mapstructure:"poll_interval"`

	// NVD (National Vulnerability Database) configuration
	NVDEnabled bool   `mapstructure:"nvd_enabled"`
	NVDAPIKey  string `mapstructure:"nvd_api_key"` // Optional, increases rate limit

	// OSV (Open Source Vulnerabilities) configuration
	OSVEnabled bool `mapstructure:"osv_enabled"`

	// GitHub Security Advisories configuration
	GitHubEnabled bool   `mapstructure:"github_enabled"`
	GitHubToken   string `mapstructure:"github_token"` // Optional, increases rate limit

	// CISA KEV (Known Exploited Vulnerabilities) configuration
	CISAKEVEnabled bool `mapstructure:"cisa_kev_enabled"`

	// Alert thresholds
	CriticalThreshold float64 `mapstructure:"critical_threshold"` // Urgency score for critical alerts
	HighThreshold     float64 `mapstructure:"high_threshold"`     // Urgency score for high alerts

	// Kafka topics for CVE events
	Topics struct {
		CVEDiscovered   string `mapstructure:"cve_discovered"`
		CVEMatchFound   string `mapstructure:"cve_match_found"`
		CVEAlertCreated string `mapstructure:"cve_alert_created"`
	} `mapstructure:"topics"`
}

// TemporalAddress returns the Temporal server address.
func (c *TemporalConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// OrchestratorAddress returns the orchestrator service address.
func (c *OrchestratorConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	v := viper.New()

	// Set prefix for environment variables
	v.SetEnvPrefix("RF")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set defaults
	setDefaults(v)

	// Bind environment variables
	if err := bindEnvVars(v); err != nil {
		return nil, fmt.Errorf("failed to bind env vars: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate production configuration
	if err := cfg.validateProduction(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// validateProduction ensures critical configuration is set for non-development environments.
func (c *Config) validateProduction() error {
	// Skip validation in development mode
	if c.Env == "development" || c.Env == "dev" || c.Env == "test" {
		return nil
	}

	var missingConfig []string

	// Database URL must not use default credentials in production
	if strings.Contains(c.Database.URL, "postgres:postgres@localhost") {
		missingConfig = append(missingConfig, "RF_DATABASE_URL (must not use default localhost credentials)")
	}

	// Clerk authentication is required in production
	if c.Clerk.SecretKey == "" {
		missingConfig = append(missingConfig, "RF_CLERK_SECRET_KEY")
	}
	if c.Clerk.PublishableKey == "" {
		missingConfig = append(missingConfig, "RF_CLERK_PUBLISHABLE_KEY")
	}

	// LLM configuration is required if orchestrator is enabled.
	// The "stub" provider is deterministic and external-call-free — it
	// requires no API key. Its constructor logs a loud WARN at startup, so a
	// production deployment that accidentally selects RF_LLM_PROVIDER=stub
	// still screams in every orchestrator log line. See
	// services/orchestrator/internal/llm/stub.go and AI-002.
	if c.Orchestrator.Enabled && c.LLM.Provider != "stub" {
		if c.LLM.Provider == "" || (c.LLM.APIKey == "" && c.LLM.AzureEndpoint == "") {
			missingConfig = append(missingConfig, "RF_LLM_PROVIDER and RF_LLM_API_KEY (or Azure OpenAI config)")
		}
	}

	if len(missingConfig) > 0 {
		return fmt.Errorf("missing required configuration for %s environment: %s",
			c.Env, strings.Join(missingConfig, ", "))
	}

	return nil
}

func setDefaults(v *viper.Viper) {
	// Application
	v.SetDefault("env", "development")
	v.SetDefault("log_level", "info")
	v.SetDefault("dev_mode", false)

	// API
	v.SetDefault("api.host", "0.0.0.0")
	v.SetDefault("api.port", 8080)
	v.SetDefault("api.read_timeout", "30s")
	v.SetDefault("api.write_timeout", "30s")
	v.SetDefault("api.shutdown_timeout", "10s")

	// Database
	v.SetDefault("database.url", "postgres://postgres:postgres@localhost:5432/qlrf?sslmode=disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")

	// Redis
	v.SetDefault("redis.url", "redis://localhost:6379/0")
	v.SetDefault("redis.max_retries", 3)

	// Kafka
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.consumer_group", "ql-rf")
	v.SetDefault("kafka.topics.asset_discovered", "asset.discovered")
	v.SetDefault("kafka.topics.drift_detected", "drift.detected")
	v.SetDefault("kafka.topics.image_published", "image.published")

	// Connectors
	v.SetDefault("connectors.host", "0.0.0.0")
	v.SetDefault("connectors.port", 8081)
	v.SetDefault("connectors.sync_interval", "5m")
	v.SetDefault("connectors.enabled", []string{"aws"})

	// AWS connector defaults. fallback_to_mock defaults to false (so a
	// misconfigured production deployment loudly fails to register the tool
	// rather than silently serving fake data). Dev/CI override to true.
	v.SetDefault("connectors.aws.region", "us-east-1")
	v.SetDefault("connectors.aws.fallback_to_mock", false)
	// PR #21 / CONN-003 live-mode defaults. Off everywhere by default; opt
	// in per-deployment via env var. "real" client mode is the prod path;
	// "mock" is for local smoke and CI tests of the boot path.
	v.SetDefault("connectors.aws.allow_live_patch", false)
	v.SetDefault("connectors.aws.live_patch_whitelist_instance_ids", []string{})
	v.SetDefault("connectors.aws.live_patch_client_mode", "real")

	// PR #26 / CONN-006 Azure connector defaults. Same fallback_to_mock
	// pattern as AWS: false in production so missing credentials surface
	// loudly; dev/CI sets true via env to keep the query_azure_vms tool
	// demoable without a real subscription.
	v.SetDefault("connectors.azure.fallback_to_mock", false)
	// PR #28 / CONN-008 Azure live-mode defaults. Off everywhere by
	// default; opt in per-deployment via env. "real" is the prod path;
	// "mock" exercises the boot path without firing the SDK.
	v.SetDefault("connectors.azure.allow_live_run_command", false)
	v.SetDefault("connectors.azure.live_run_command_whitelist_vms", []string{})
	v.SetDefault("connectors.azure.live_run_command_client_mode", "real")

	// PR #29 / CONN-009 GCP connector defaults. Same fallback_to_mock
	// pattern: false in production, dev/CI sets true via env.
	v.SetDefault("connectors.gcp.fallback_to_mock", false)

	// K8s connector defaults
	v.SetDefault("connectors.k8s.kubeconfig", "")
	v.SetDefault("connectors.k8s.context", "")
	v.SetDefault("connectors.k8s.namespaces", []string{})
	v.SetDefault("connectors.k8s.exclude_namespaces", []string{"kube-system", "kube-public", "kube-node-lease"})
	v.SetDefault("connectors.k8s.discover_nodes", true)
	v.SetDefault("connectors.k8s.discover_deployments", true)
	v.SetDefault("connectors.k8s.label_selector", "")
	v.SetDefault("connectors.k8s.cluster_name", "")

	// Drift
	v.SetDefault("drift.host", "0.0.0.0")
	v.SetDefault("drift.port", 8082)
	v.SetDefault("drift.calculation_interval", "1h")
	v.SetDefault("drift.threshold_warning", 90.0)
	v.SetDefault("drift.threshold_critical", 70.0)

	// Metrics
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")

	// Orchestrator
	v.SetDefault("orchestrator.host", "0.0.0.0")
	v.SetDefault("orchestrator.port", 8083)
	v.SetDefault("orchestrator.enabled", true)

	// LLM
	v.SetDefault("llm.provider", "anthropic")
	v.SetDefault("llm.model", "claude-3-5-sonnet-20241022")
	v.SetDefault("llm.max_tokens", 4096)
	v.SetDefault("llm.temperature", 0.3)
	v.SetDefault("llm.max_requests_per_minute", 60)
	v.SetDefault("llm.azure_api_version", "2024-02-15-preview")

	// Temporal
	v.SetDefault("temporal.host", "localhost")
	v.SetDefault("temporal.port", 7233)
	v.SetDefault("temporal.namespace", "default")
	v.SetDefault("temporal.task_queue", "ql-rf-orchestrator")
	v.SetDefault("temporal.worker_count", 4)
	v.SetDefault("temporal.max_concurrent_workflows", 100)
	v.SetDefault("temporal.max_concurrent_activities", 50)
	v.SetDefault("temporal.tls_enabled", false)

	// OPA
	v.SetDefault("opa.enabled", true)
	v.SetDefault("opa.mode", "embedded")
	v.SetDefault("opa.policies_dir", "./policy")
	v.SetDefault("opa.url", "http://localhost:8181")
	v.SetDefault("opa.eval_timeout", "5s")

	// Notifications
	v.SetDefault("notifications.app_base_url", "http://localhost:3000")
	v.SetDefault("notifications.slack_enabled", false)
	v.SetDefault("notifications.slack_channel", "#alerts")
	v.SetDefault("notifications.email_enabled", false)
	v.SetDefault("notifications.smtp_port", 587)
	v.SetDefault("notifications.webhook_enabled", false)
	v.SetDefault("notifications.teams_enabled", false)

	// CVE Aggregator
	v.SetDefault("cve.host", "0.0.0.0")
	v.SetDefault("cve.port", 8084)
	v.SetDefault("cve.poll_interval", "15m")
	v.SetDefault("cve.nvd_enabled", true)
	v.SetDefault("cve.osv_enabled", true)
	v.SetDefault("cve.github_enabled", true)
	v.SetDefault("cve.cisa_kev_enabled", true)
	v.SetDefault("cve.critical_threshold", 80.0)
	v.SetDefault("cve.high_threshold", 60.0)
	v.SetDefault("cve.topics.cve_discovered", "cve.discovered")
	v.SetDefault("cve.topics.cve_match_found", "cve.match.found")
	v.SetDefault("cve.topics.cve_alert_created", "cve.alert.created")
}

func bindEnvVars(v *viper.Viper) error {
	envVars := []string{
		"env",
		"log_level",
		"dev_mode",
		"api.host",
		"api.port",
		"api.read_timeout",
		"api.write_timeout",
		"api.shutdown_timeout",
		"database.url",
		"database.max_open_conns",
		"database.max_idle_conns",
		"database.conn_max_lifetime",
		"redis.url",
		"redis.max_retries",
		"kafka.brokers",
		"kafka.consumer_group",
		"clerk.secret_key",
		"clerk.publishable_key",
		"connectors.host",
		"connectors.port",
		"connectors.sync_interval",
		"connectors.enabled",
		// AWS connector config (shared between the connectors service for
		// asset discovery and the orchestrator for real-tool invocations,
		// PR #19 / CONN-001).
		"connectors.aws.region",
		"connectors.aws.assume_role_arn",
		"connectors.aws.assume_role_external_id",
		"connectors.aws.fallback_to_mock",
		// PR #21 / CONN-003 — live patch mode env bindings.
		"connectors.aws.allow_live_patch",
		"connectors.aws.live_patch_whitelist_instance_ids",
		"connectors.aws.live_patch_client_mode",
		// PR #26 / CONN-006 — Azure connector for the query_azure_vms
		// orchestrator tool. Credentials (tenant/client/secret/subscription)
		// piggyback on existing connectors.azure.* bindings used by the
		// connectors service.
		"connectors.azure.tenant_id",
		"connectors.azure.client_id",
		"connectors.azure.client_secret",
		"connectors.azure.subscription_id",
		"connectors.azure.fallback_to_mock",
		// PR #28 / CONN-008 — live Azure Run Command env bindings.
		"connectors.azure.allow_live_run_command",
		"connectors.azure.live_run_command_whitelist_vms",
		"connectors.azure.live_run_command_client_mode",
		// PR #29 / CONN-009 — GCP connector env bindings.
		"connectors.gcp.project_id",
		"connectors.gcp.credentials_file",
		"connectors.gcp.fallback_to_mock",
		"drift.host",
		"drift.port",
		"drift.calculation_interval",
		"drift.threshold_warning",
		"drift.threshold_critical",
		"metrics.enabled",
		"metrics.path",
		// Orchestrator
		"orchestrator.host",
		"orchestrator.port",
		"orchestrator.enabled",
		// LLM
		"llm.provider",
		"llm.api_key",
		"llm.model",
		"llm.max_tokens",
		"llm.temperature",
		"llm.azure_endpoint",
		"llm.azure_api_version",
		"llm.azure_anthropic_endpoint",
		"llm.azure_deployment",
		"llm.max_requests_per_minute",
		"llm.fallback_provider",
		"llm.fallback_model",
		// Temporal
		"temporal.host",
		"temporal.port",
		"temporal.namespace",
		"temporal.task_queue",
		"temporal.worker_count",
		"temporal.max_concurrent_workflows",
		"temporal.max_concurrent_activities",
		"temporal.tls_enabled",
		"temporal.tls_cert_path",
		"temporal.tls_key_path",
		// OPA
		"opa.enabled",
		"opa.mode",
		"opa.url",
		"opa.policies_dir",
		"opa.eval_timeout",
		// Notifications
		"notifications.app_base_url",
		"notifications.slack_enabled",
		"notifications.slack_webhook_url",
		"notifications.slack_channel",
		"notifications.email_enabled",
		"notifications.smtp_host",
		"notifications.smtp_port",
		"notifications.smtp_user",
		"notifications.smtp_password",
		"notifications.email_from",
		"notifications.email_recipients",
		"notifications.webhook_enabled",
		"notifications.webhook_url",
		"notifications.webhook_secret",
		"notifications.teams_enabled",
		"notifications.teams_webhook_url",
		// CVE Aggregator
		"cve.host",
		"cve.port",
		"cve.poll_interval",
		"cve.nvd_enabled",
		"cve.nvd_api_key",
		"cve.osv_enabled",
		"cve.github_enabled",
		"cve.github_token",
		"cve.cisa_kev_enabled",
		"cve.critical_threshold",
		"cve.high_threshold",
	}

	for _, key := range envVars {
		if err := v.BindEnv(key); err != nil {
			return fmt.Errorf("failed to bind %s: %w", key, err)
		}
	}

	return nil
}

// Address returns the API server address.
func (c *APIConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
