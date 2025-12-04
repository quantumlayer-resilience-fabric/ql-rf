// Package llmcache provides semantic caching for LLM responses.
package llmcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Cache Key
// =============================================================================

// Key uniquely identifies a cached LLM response.
type Key struct {
	// AgentName is the name of the agent making the request.
	AgentName string `json:"agent_name"`

	// OrgID is the organization ID.
	OrgID uuid.UUID `json:"org_id"`

	// Environment is the target environment.
	Environment string `json:"environment"`

	// IntentHash is a stable hash of the normalized intent.
	IntentHash string `json:"intent_hash"`
}

// String returns a string representation of the key.
func (k *Key) String() string {
	return fmt.Sprintf("llmcache:%s:%s:%s:%s",
		k.AgentName, k.OrgID.String(), k.Environment, k.IntentHash)
}

// NewKey creates a cache key from components.
func NewKey(agentName string, orgID uuid.UUID, env, intent string) *Key {
	return &Key{
		AgentName:   agentName,
		OrgID:       orgID,
		Environment: env,
		IntentHash:  hashIntent(intent),
	}
}

// hashIntent creates a stable hash of the normalized intent.
func hashIntent(intent string) string {
	// Normalize: lowercase, trim, collapse whitespace
	normalized := strings.TrimSpace(strings.ToLower(intent))
	normalized = strings.Join(strings.Fields(normalized), " ")

	h := sha256.New()
	h.Write([]byte(normalized))
	return hex.EncodeToString(h.Sum(nil))[:16] // First 16 chars
}

// =============================================================================
// Cached Result
// =============================================================================

// Result represents a cached LLM response.
type Result struct {
	// Response is the LLM response content.
	Response string `json:"response"`

	// CreatedAt is when the result was cached.
	CreatedAt time.Time `json:"created_at"`

	// HitCount is how many times this entry has been retrieved.
	HitCount int `json:"hit_count"`

	// ApproxTokens is the approximate token count.
	ApproxTokens int `json:"approx_tokens"`

	// Model is the model that generated this response.
	Model string `json:"model,omitempty"`

	// Metadata contains additional context.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Age returns how old the cached result is.
func (r *Result) Age() time.Duration {
	return time.Since(r.CreatedAt)
}

// IsExpired returns true if the result has exceeded the TTL.
func (r *Result) IsExpired(ttl time.Duration) bool {
	return r.Age() > ttl
}

// =============================================================================
// Cache Interface
// =============================================================================

// Cache defines the interface for LLM response caching.
type Cache interface {
	// Get retrieves a cached result.
	Get(ctx context.Context, key *Key) (*Result, error)

	// Put stores a result in the cache.
	Put(ctx context.Context, key *Key, result *Result, ttl time.Duration) error

	// Delete removes a cached result.
	Delete(ctx context.Context, key *Key) error

	// Stats returns cache statistics.
	Stats() *Stats
}

// Stats contains cache statistics.
type Stats struct {
	Hits       int64   `json:"hits"`
	Misses     int64   `json:"misses"`
	Puts       int64   `json:"puts"`
	Deletes    int64   `json:"deletes"`
	HitRate    float64 `json:"hit_rate"`
	TotalSaved int64   `json:"total_saved"` // Approximate tokens saved
}

// =============================================================================
// In-Memory Cache Implementation
// =============================================================================

// entry is an internal cache entry.
type entry struct {
	result    *Result
	expiresAt time.Time
}

// MemoryCache is an in-memory cache implementation.
// Suitable for development/testing or single-instance deployments.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*entry

	// Stats
	hits    int64
	misses  int64
	puts    int64
	deletes int64
	saved   int64

	// Config
	maxEntries int
	defaultTTL time.Duration
}

// MemoryCacheConfig configures the memory cache.
type MemoryCacheConfig struct {
	// MaxEntries is the maximum number of cached entries.
	MaxEntries int

	// DefaultTTL is the default TTL for cached entries.
	DefaultTTL time.Duration
}

// NewMemoryCache creates a new in-memory cache.
func NewMemoryCache(config *MemoryCacheConfig) *MemoryCache {
	if config == nil {
		config = &MemoryCacheConfig{
			MaxEntries: 1000,
			DefaultTTL: 15 * time.Minute,
		}
	}

	return &MemoryCache{
		entries:    make(map[string]*entry),
		maxEntries: config.MaxEntries,
		defaultTTL: config.DefaultTTL,
	}
}

// Get retrieves a cached result.
func (c *MemoryCache) Get(ctx context.Context, key *Key) (*Result, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[key.String()]
	if !ok {
		c.misses++
		return nil, nil
	}

	// Check expiration
	if time.Now().After(e.expiresAt) {
		delete(c.entries, key.String())
		c.misses++
		return nil, nil
	}

	c.hits++
	c.saved += int64(e.result.ApproxTokens)
	e.result.HitCount++
	return e.result, nil
}

// Put stores a result in the cache.
func (c *MemoryCache) Put(ctx context.Context, key *Key, result *Result, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	// Evict if at capacity
	if len(c.entries) >= c.maxEntries {
		c.evictOldest()
	}

	c.entries[key.String()] = &entry{
		result:    result,
		expiresAt: time.Now().Add(ttl),
	}
	c.puts++

	return nil
}

// Delete removes a cached result.
func (c *MemoryCache) Delete(ctx context.Context, key *Key) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key.String())
	c.deletes++
	return nil
}

// Stats returns cache statistics.
func (c *MemoryCache) Stats() *Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	var hitRate float64
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return &Stats{
		Hits:       c.hits,
		Misses:     c.misses,
		Puts:       c.puts,
		Deletes:    c.deletes,
		HitRate:    hitRate,
		TotalSaved: c.saved,
	}
}

// evictOldest removes the oldest entry.
func (c *MemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for k, e := range c.entries {
		if oldestKey == "" || e.result.CreatedAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = e.result.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// Size returns the number of cached entries.
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Clear removes all entries from the cache.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*entry)
}

// =============================================================================
// No-Op Cache (for disabled caching)
// =============================================================================

// NoOpCache is a cache that does nothing (for when caching is disabled).
type NoOpCache struct{}

// NewNoOpCache creates a new no-op cache.
func NewNoOpCache() *NoOpCache {
	return &NoOpCache{}
}

// Get always returns nil (cache miss).
func (c *NoOpCache) Get(ctx context.Context, key *Key) (*Result, error) {
	return nil, nil
}

// Put does nothing.
func (c *NoOpCache) Put(ctx context.Context, key *Key, result *Result, ttl time.Duration) error {
	return nil
}

// Delete does nothing.
func (c *NoOpCache) Delete(ctx context.Context, key *Key) error {
	return nil
}

// Stats returns empty stats.
func (c *NoOpCache) Stats() *Stats {
	return &Stats{}
}

// =============================================================================
// Cache-Aware LLM Wrapper
// =============================================================================

// CachedResponse wraps a result with cache metadata.
type CachedResponse struct {
	Result      *Result
	CacheHit    bool
	CacheKey    string
	RetrievedAt time.Time
}

// Wrap creates a helper for caching LLM responses.
func Wrap(cache Cache) *CacheWrapper {
	return &CacheWrapper{cache: cache}
}

// CacheWrapper provides helper methods for caching.
type CacheWrapper struct {
	cache Cache
}

// GetOrCompute retrieves from cache or computes and caches the result.
func (w *CacheWrapper) GetOrCompute(
	ctx context.Context,
	key *Key,
	ttl time.Duration,
	compute func() (string, int, error),
) (*CachedResponse, error) {
	// Try cache first
	result, err := w.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("cache get failed: %w", err)
	}

	if result != nil {
		return &CachedResponse{
			Result:      result,
			CacheHit:    true,
			CacheKey:    key.String(),
			RetrievedAt: time.Now(),
		}, nil
	}

	// Cache miss - compute
	response, tokens, err := compute()
	if err != nil {
		return nil, err
	}

	// Store in cache
	result = &Result{
		Response:     response,
		CreatedAt:    time.Now(),
		HitCount:     0,
		ApproxTokens: tokens,
	}

	if err := w.cache.Put(ctx, key, result, ttl); err != nil {
		// Log but don't fail - caching is best-effort
		// TODO: Add logging
	}

	return &CachedResponse{
		Result:      result,
		CacheHit:    false,
		CacheKey:    key.String(),
		RetrievedAt: time.Now(),
	}, nil
}

// =============================================================================
// Serialization Helpers
// =============================================================================

// MarshalKey serializes a key to JSON.
func MarshalKey(key *Key) ([]byte, error) {
	return json.Marshal(key)
}

// UnmarshalKey deserializes a key from JSON.
func UnmarshalKey(data []byte) (*Key, error) {
	var key Key
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// MarshalResult serializes a result to JSON.
func MarshalResult(result *Result) ([]byte, error) {
	return json.Marshal(result)
}

// UnmarshalResult deserializes a result from JSON.
func UnmarshalResult(data []byte) (*Result, error) {
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
