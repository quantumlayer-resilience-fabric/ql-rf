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

	API        APIConfig        `mapstructure:"api"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Kafka      KafkaConfig      `mapstructure:"kafka"`
	Clerk      ClerkConfig      `mapstructure:"clerk"`
	Connectors ConnectorsConfig `mapstructure:"connectors"`
	Drift      DriftConfig      `mapstructure:"drift"`
	Metrics    MetricsConfig    `mapstructure:"metrics"`
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
}

// AWSConfig holds AWS connector configuration.
type AWSConfig struct {
	Region        string   `mapstructure:"region"`
	AssumeRoleARN string   `mapstructure:"assume_role_arn"`
	Regions       []string `mapstructure:"regions"` // List of regions to scan (optional, discovers all if empty)
}

// AzureConfig holds Azure connector configuration.
type AzureConfig struct {
	TenantID       string `mapstructure:"tenant_id"`
	ClientID       string `mapstructure:"client_id"`
	ClientSecret   string `mapstructure:"client_secret"`
	SubscriptionID string `mapstructure:"subscription_id"`
}

// GCPConfig holds GCP connector configuration.
type GCPConfig struct {
	ProjectID       string `mapstructure:"project_id"`
	CredentialsFile string `mapstructure:"credentials_file"`
}

// VSphereConfig holds vSphere connector configuration.
type VSphereConfig struct {
	URL      string `mapstructure:"url"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Insecure bool   `mapstructure:"insecure"`
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

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Application
	v.SetDefault("env", "development")
	v.SetDefault("log_level", "info")

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

	// Drift
	v.SetDefault("drift.host", "0.0.0.0")
	v.SetDefault("drift.port", 8082)
	v.SetDefault("drift.calculation_interval", "1h")
	v.SetDefault("drift.threshold_warning", 90.0)
	v.SetDefault("drift.threshold_critical", 70.0)

	// Metrics
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
}

func bindEnvVars(v *viper.Viper) error {
	envVars := []string{
		"env",
		"log_level",
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
		"drift.host",
		"drift.port",
		"drift.calculation_interval",
		"drift.threshold_warning",
		"drift.threshold_critical",
		"metrics.enabled",
		"metrics.path",
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
