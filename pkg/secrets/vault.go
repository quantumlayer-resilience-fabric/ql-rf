// Package secrets provides secret management with HashiCorp Vault integration.
// It supports multiple secret backends and provides a unified interface for secret operations.
package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	vault "github.com/hashicorp/vault/api"
)

// ErrSecretNotFound is returned when a requested secret does not exist. Callers
// can distinguish a genuine miss from a backend failure with errors.Is.
var ErrSecretNotFound = errors.New("secret not found")

// Backend represents a secret storage backend.
type Backend string

const (
	BackendVault  Backend = "vault"
	BackendEnv    Backend = "env"
	BackendMemory Backend = "memory"
)

// Config holds configuration for the secrets manager.
type Config struct {
	Backend Backend

	// Vault configuration
	VaultAddr      string
	VaultToken     string
	VaultRoleID    string // For AppRole auth
	VaultSecretID  string
	VaultMountPath string // e.g., "secret" for KV v2
	VaultNamespace string // Vault Enterprise namespace

	// General settings
	CacheTTL          time.Duration
	AutoRenewToken    bool
	MaxRetries        int
	ConnectionTimeout time.Duration
}

// DefaultConfig returns default secrets configuration.
func DefaultConfig() *Config {
	return &Config{
		Backend:           BackendEnv,
		VaultAddr:         os.Getenv("VAULT_ADDR"),
		VaultToken:        os.Getenv("VAULT_TOKEN"),
		VaultMountPath:    "secret",
		CacheTTL:          5 * time.Minute,
		AutoRenewToken:    true,
		MaxRetries:        3,
		ConnectionTimeout: 30 * time.Second,
	}
}

// Secret represents a secret with metadata.
type Secret struct {
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Version   int               `json:"version,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
}

// Manager provides secret management functionality.
type Manager struct {
	cfg         *Config
	vaultClient *vault.Client
	cache       *secretCache
}

// secretCache provides in-memory caching for secrets.
type secretCache struct {
	secrets map[string]*cachedSecret
	mu      sync.RWMutex
	ttl     time.Duration
}

type cachedSecret struct {
	secret    *Secret
	expiresAt time.Time
}

func newSecretCache(ttl time.Duration) *secretCache {
	return &secretCache{
		secrets: make(map[string]*cachedSecret),
		ttl:     ttl,
	}
}

func (c *secretCache) Get(key string) (*Secret, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.secrets[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(cached.expiresAt) {
		return nil, false
	}

	return cached.secret, true
}

func (c *secretCache) Set(key string, secret *Secret) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.secrets[key] = &cachedSecret{
		secret:    secret,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *secretCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.secrets, key)
}

func (c *secretCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.secrets = make(map[string]*cachedSecret)
}

// NewManager creates a new secret manager.
func NewManager(cfg *Config) (*Manager, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	m := &Manager{
		cfg:   cfg,
		cache: newSecretCache(cfg.CacheTTL),
	}

	if cfg.Backend == BackendVault {
		if err := m.initVault(); err != nil {
			return nil, fmt.Errorf("failed to initialize Vault: %w", err)
		}
	}

	return m, nil
}

func (m *Manager) initVault() error {
	vaultCfg := vault.DefaultConfig()
	vaultCfg.Address = m.cfg.VaultAddr
	vaultCfg.Timeout = m.cfg.ConnectionTimeout
	vaultCfg.MaxRetries = m.cfg.MaxRetries

	client, err := vault.NewClient(vaultCfg)
	if err != nil {
		return fmt.Errorf("failed to create Vault client: %w", err)
	}

	// Set namespace if configured (Vault Enterprise)
	if m.cfg.VaultNamespace != "" {
		client.SetNamespace(m.cfg.VaultNamespace)
	}

	// Authenticate
	if m.cfg.VaultToken != "" {
		client.SetToken(m.cfg.VaultToken)
	} else if m.cfg.VaultRoleID != "" && m.cfg.VaultSecretID != "" {
		// AppRole authentication
		if err := m.appRoleAuth(client); err != nil {
			return fmt.Errorf("AppRole auth failed: %w", err)
		}
	} else {
		return fmt.Errorf("no Vault authentication method configured")
	}

	m.vaultClient = client

	// Start token renewal if configured
	if m.cfg.AutoRenewToken {
		go m.renewTokenPeriodically()
	}

	return nil
}

func (m *Manager) appRoleAuth(client *vault.Client) error {
	data := map[string]interface{}{
		"role_id":   m.cfg.VaultRoleID,
		"secret_id": m.cfg.VaultSecretID,
	}

	resp, err := client.Logical().Write("auth/approle/login", data)
	if err != nil {
		return err
	}

	if resp.Auth == nil {
		return fmt.Errorf("no auth info in response")
	}

	client.SetToken(resp.Auth.ClientToken)
	return nil
}

func (m *Manager) renewTokenPeriodically() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if m.vaultClient != nil {
			_, err := m.vaultClient.Auth().Token().RenewSelf(3600)
			if err != nil {
				// Log error but don't crash
				continue
			}
		}
	}
}

// Get retrieves a secret by key.
func (m *Manager) Get(ctx context.Context, key string) (*Secret, error) {
	// Check cache first
	if secret, ok := m.cache.Get(key); ok {
		return secret, nil
	}

	var secret *Secret
	var err error

	switch m.cfg.Backend {
	case BackendVault:
		secret, err = m.getFromVault(ctx, key)
	case BackendEnv:
		secret, err = m.getFromEnv(key)
	case BackendMemory:
		// Memory backend only uses cache
		return nil, fmt.Errorf("%w: %s", ErrSecretNotFound, key)
	default:
		return nil, fmt.Errorf("unknown backend: %s", m.cfg.Backend)
	}

	if err != nil {
		return nil, err
	}

	// Cache the result
	m.cache.Set(key, secret)

	return secret, nil
}

func (m *Manager) getFromVault(ctx context.Context, key string) (*Secret, error) {
	if m.vaultClient == nil {
		return nil, fmt.Errorf("Vault client not initialized")
	}

	path := fmt.Sprintf("%s/data/%s", m.cfg.VaultMountPath, key)
	secret, err := m.vaultClient.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret from Vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("%w: %s", ErrSecretNotFound, key)
	}

	// KV v2 returns data in a nested structure
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid secret format")
	}

	// Get the value - support both "value" key and first key
	var value string
	if v, ok := data["value"].(string); ok {
		value = v
	} else {
		// Try to get the first string value
		for _, v := range data {
			if s, ok := v.(string); ok {
				value = s
				break
			}
		}
	}

	result := &Secret{
		Key:   key,
		Value: value,
	}

	// Extract metadata
	if metadata, ok := secret.Data["metadata"].(map[string]interface{}); ok {
		if version, ok := metadata["version"].(json.Number); ok {
			v, err := version.Int64()
			if err == nil {
				result.Version = int(v)
			}
		}
		if created, ok := metadata["created_time"].(string); ok {
			if t, err := time.Parse(time.RFC3339, created); err == nil {
				result.CreatedAt = t
			}
		}
	}

	return result, nil
}

func (m *Manager) getFromEnv(key string) (*Secret, error) {
	value := os.Getenv(key)
	if value == "" {
		// Try with RF_ prefix
		value = os.Getenv("RF_" + key)
	}

	if value == "" {
		return nil, fmt.Errorf("%w: environment variable %s", ErrSecretNotFound, key)
	}

	return &Secret{
		Key:   key,
		Value: value,
	}, nil
}

// Set stores a secret.
func (m *Manager) Set(ctx context.Context, key, value string, metadata map[string]string) error {
	m.cache.Delete(key)

	switch m.cfg.Backend {
	case BackendVault:
		return m.setInVault(ctx, key, value, metadata)
	case BackendEnv:
		return fmt.Errorf("cannot set environment variables at runtime")
	case BackendMemory:
		secret := &Secret{
			Key:      key,
			Value:    value,
			Metadata: metadata,
		}
		m.cache.Set(key, secret)
		return nil
	default:
		return fmt.Errorf("unknown backend: %s", m.cfg.Backend)
	}
}

func (m *Manager) setInVault(ctx context.Context, key, value string, metadata map[string]string) error {
	if m.vaultClient == nil {
		return fmt.Errorf("Vault client not initialized")
	}

	path := fmt.Sprintf("%s/data/%s", m.cfg.VaultMountPath, key)
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"value": value,
		},
	}

	// Add custom metadata
	if len(metadata) > 0 {
		if inner, ok := data["data"].(map[string]interface{}); ok {
			for k, v := range metadata {
				inner[k] = v
			}
		}
	}

	_, err := m.vaultClient.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return fmt.Errorf("failed to write secret to Vault: %w", err)
	}

	return nil
}

// Delete removes a secret.
func (m *Manager) Delete(ctx context.Context, key string) error {
	m.cache.Delete(key)

	switch m.cfg.Backend {
	case BackendVault:
		return m.deleteFromVault(ctx, key)
	case BackendEnv:
		return fmt.Errorf("cannot delete environment variables at runtime")
	case BackendMemory:
		return nil // Already deleted from cache
	default:
		return fmt.Errorf("unknown backend: %s", m.cfg.Backend)
	}
}

func (m *Manager) deleteFromVault(ctx context.Context, key string) error {
	if m.vaultClient == nil {
		return fmt.Errorf("Vault client not initialized")
	}

	// Soft delete (destroys all versions)
	path := fmt.Sprintf("%s/metadata/%s", m.cfg.VaultMountPath, key)
	_, err := m.vaultClient.Logical().DeleteWithContext(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to delete secret from Vault: %w", err)
	}

	return nil
}

// List returns all secret keys at a path.
func (m *Manager) List(ctx context.Context, path string) ([]string, error) {
	switch m.cfg.Backend {
	case BackendVault:
		return m.listFromVault(ctx, path)
	case BackendEnv:
		return nil, fmt.Errorf("cannot list environment variables")
	case BackendMemory:
		return m.listFromCache(), nil
	default:
		return nil, fmt.Errorf("unknown backend: %s", m.cfg.Backend)
	}
}

func (m *Manager) listFromVault(ctx context.Context, path string) ([]string, error) {
	if m.vaultClient == nil {
		return nil, fmt.Errorf("Vault client not initialized")
	}

	fullPath := fmt.Sprintf("%s/metadata/%s", m.cfg.VaultMountPath, path)
	secret, err := m.vaultClient.Logical().ListWithContext(ctx, fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets from Vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	result := make([]string, 0, len(keys))
	for _, k := range keys {
		if s, ok := k.(string); ok {
			result = append(result, s)
		}
	}

	return result, nil
}

func (m *Manager) listFromCache() []string {
	m.cache.mu.RLock()
	defer m.cache.mu.RUnlock()

	keys := make([]string, 0, len(m.cache.secrets))
	for k := range m.cache.secrets {
		keys = append(keys, k)
	}
	return keys
}

// GetString is a convenience method that returns the secret value as a string.
func (m *Manager) GetString(ctx context.Context, key string) (string, error) {
	secret, err := m.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return secret.Value, nil
}

// GetOrDefault returns the secret value or a default if not found.
func (m *Manager) GetOrDefault(ctx context.Context, key, defaultValue string) string {
	value, err := m.GetString(ctx, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// ClearCache clears the secret cache.
func (m *Manager) ClearCache() {
	m.cache.Clear()
}

// Close closes the secret manager.
func (m *Manager) Close() error {
	m.cache.Clear()
	return nil
}

// HealthCheck checks if the secret backend is healthy.
func (m *Manager) HealthCheck(_ context.Context) error {
	switch m.cfg.Backend {
	case BackendVault:
		if m.vaultClient == nil {
			return fmt.Errorf("Vault client not initialized")
		}
		health, err := m.vaultClient.Sys().Health()
		if err != nil {
			return fmt.Errorf("Vault health check failed: %w", err)
		}
		if !health.Initialized {
			return fmt.Errorf("Vault is not initialized")
		}
		if health.Sealed {
			return fmt.Errorf("Vault is sealed")
		}
		return nil
	case BackendEnv:
		return nil // Environment is always available
	case BackendMemory:
		return nil // Memory is always available
	default:
		return fmt.Errorf("unknown backend: %s", m.cfg.Backend)
	}
}

// RotateSecret rotates a secret by generating a new value.
func (m *Manager) RotateSecret(ctx context.Context, key string, generator func() (string, error)) error {
	newValue, err := generator()
	if err != nil {
		return fmt.Errorf("failed to generate new secret: %w", err)
	}

	// Preserve existing metadata across rotation. A genuine miss (first rotation)
	// is fine; any other error means the store is unhealthy, so don't rotate blindly.
	existing, err := m.Get(ctx, key)
	if err != nil && !errors.Is(err, ErrSecretNotFound) {
		return fmt.Errorf("failed to read existing secret %q before rotation: %w", key, err)
	}
	metadata := map[string]string{
		"rotated_at": time.Now().Format(time.RFC3339),
	}
	if existing != nil && existing.Metadata != nil {
		for k, v := range existing.Metadata {
			if k != "rotated_at" {
				metadata[k] = v
			}
		}
	}

	return m.Set(ctx, key, newValue, metadata)
}

// DynamicCredentials represents credentials from a dynamic secret engine.
type DynamicCredentials struct {
	Username  string        `json:"username"`
	Password  string        `json:"password"`
	LeaseTTL  time.Duration `json:"lease_ttl"`
	LeaseID   string        `json:"lease_id"`
	ExpiresAt time.Time     `json:"expires_at"`
}

// GetDatabaseCredentials retrieves dynamic database credentials from Vault.
func (m *Manager) GetDatabaseCredentials(ctx context.Context, role string) (*DynamicCredentials, error) {
	if m.cfg.Backend != BackendVault {
		return nil, fmt.Errorf("dynamic credentials only supported with Vault backend")
	}

	if m.vaultClient == nil {
		return nil, fmt.Errorf("Vault client not initialized")
	}

	path := fmt.Sprintf("database/creds/%s", role)
	secret, err := m.vaultClient.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get database credentials: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no credentials returned for role: %s", role)
	}

	username, ok1 := secret.Data["username"].(string)
	password, ok2 := secret.Data["password"].(string)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("invalid credential format for role: %s", role)
	}

	return &DynamicCredentials{
		Username:  username,
		Password:  password,
		LeaseTTL:  time.Duration(secret.LeaseDuration) * time.Second,
		LeaseID:   secret.LeaseID,
		ExpiresAt: time.Now().Add(time.Duration(secret.LeaseDuration) * time.Second),
	}, nil
}

// GetAWSCredentials retrieves dynamic AWS credentials from Vault.
func (m *Manager) GetAWSCredentials(ctx context.Context, role string) (map[string]string, error) {
	if m.cfg.Backend != BackendVault {
		return nil, fmt.Errorf("dynamic credentials only supported with Vault backend")
	}

	if m.vaultClient == nil {
		return nil, fmt.Errorf("Vault client not initialized")
	}

	path := fmt.Sprintf("aws/creds/%s", role)
	secret, err := m.vaultClient.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no credentials returned for role: %s", role)
	}

	accessKey, ok1 := secret.Data["access_key"].(string)
	secretKey, ok2 := secret.Data["secret_key"].(string)
	securityToken, ok3 := secret.Data["security_token"].(string)
	if !ok1 || !ok2 || !ok3 {
		return nil, fmt.Errorf("invalid AWS credential format for role: %s", role)
	}

	return map[string]string{
		"access_key":     accessKey,
		"secret_key":     secretKey,
		"security_token": securityToken,
	}, nil
}

// RenewLease renews a Vault lease.
func (m *Manager) RenewLease(ctx context.Context, leaseID string, increment int) error {
	if m.cfg.Backend != BackendVault {
		return fmt.Errorf("lease renewal only supported with Vault backend")
	}

	if m.vaultClient == nil {
		return fmt.Errorf("Vault client not initialized")
	}

	_, err := m.vaultClient.Sys().RenewWithContext(ctx, leaseID, increment)
	if err != nil {
		return fmt.Errorf("failed to renew lease: %w", err)
	}

	return nil
}

// RevokeLease revokes a Vault lease.
func (m *Manager) RevokeLease(ctx context.Context, leaseID string) error {
	if m.cfg.Backend != BackendVault {
		return fmt.Errorf("lease revocation only supported with Vault backend")
	}

	if m.vaultClient == nil {
		return fmt.Errorf("Vault client not initialized")
	}

	err := m.vaultClient.Sys().RevokeWithContext(ctx, leaseID)
	if err != nil {
		return fmt.Errorf("failed to revoke lease: %w", err)
	}

	return nil
}
