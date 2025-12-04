package llmcache

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestKey_String(t *testing.T) {
	key := &Key{
		AgentName:   "drift-agent",
		OrgID:       uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Environment: "production",
		IntentHash:  "abc123def456",
	}

	s := key.String()
	if s == "" {
		t.Error("expected non-empty string")
	}

	// Should contain all components
	expected := "llmcache:drift-agent:11111111-1111-1111-1111-111111111111:production:abc123def456"
	if s != expected {
		t.Errorf("expected %q, got %q", expected, s)
	}
}

func TestNewKey(t *testing.T) {
	orgID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	key := NewKey("patch-agent", orgID, "staging", "Apply security patches")

	if key.AgentName != "patch-agent" {
		t.Errorf("expected agent_name 'patch-agent', got %q", key.AgentName)
	}

	if key.OrgID != orgID {
		t.Errorf("expected org_id %s, got %s", orgID, key.OrgID)
	}

	if key.Environment != "staging" {
		t.Errorf("expected environment 'staging', got %q", key.Environment)
	}

	if key.IntentHash == "" {
		t.Error("expected non-empty intent_hash")
	}

	if len(key.IntentHash) != 16 {
		t.Errorf("expected intent_hash length 16, got %d", len(key.IntentHash))
	}
}

func TestHashIntent_Normalization(t *testing.T) {
	// These should all produce the same hash
	intents := []string{
		"Apply security patches",
		"apply security patches",
		"  Apply   Security   Patches  ",
		"APPLY SECURITY PATCHES",
	}

	var firstHash string
	for i, intent := range intents {
		hash := hashIntent(intent)
		if i == 0 {
			firstHash = hash
		} else if hash != firstHash {
			t.Errorf("expected same hash for %q, got different", intent)
		}
	}
}

func TestHashIntent_DifferentIntents(t *testing.T) {
	hash1 := hashIntent("Apply security patches")
	hash2 := hashIntent("Check drift status")

	if hash1 == hash2 {
		t.Error("different intents should produce different hashes")
	}
}

func TestResult_Age(t *testing.T) {
	result := &Result{
		Response:  "test",
		CreatedAt: time.Now().Add(-5 * time.Minute),
	}

	age := result.Age()
	if age < 4*time.Minute || age > 6*time.Minute {
		t.Errorf("unexpected age: %v", age)
	}
}

func TestResult_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		createdAt time.Time
		ttl       time.Duration
		expired   bool
	}{
		{
			name:      "not expired",
			createdAt: time.Now().Add(-5 * time.Minute),
			ttl:       10 * time.Minute,
			expired:   false,
		},
		{
			name:      "expired",
			createdAt: time.Now().Add(-15 * time.Minute),
			ttl:       10 * time.Minute,
			expired:   true,
		},
		{
			name:      "just expired",
			createdAt: time.Now().Add(-10*time.Minute - time.Second),
			ttl:       10 * time.Minute,
			expired:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{CreatedAt: tt.createdAt}
			if result.IsExpired(tt.ttl) != tt.expired {
				t.Errorf("expected expired=%v, got %v", tt.expired, !tt.expired)
			}
		})
	}
}

func TestMemoryCache_Basic(t *testing.T) {
	cache := NewMemoryCache(nil)
	ctx := context.Background()

	key := NewKey("test-agent", uuid.New(), "test", "test intent")
	result := &Result{
		Response:     "test response",
		CreatedAt:    time.Now(),
		ApproxTokens: 100,
	}

	// Initially empty
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil result for missing key")
	}

	// Put
	err = cache.Put(ctx, key, result, 5*time.Minute)
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	// Get
	got, err = cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected result")
	}
	if got.Response != "test response" {
		t.Errorf("expected 'test response', got %q", got.Response)
	}

	// Delete
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Should be gone
	got, err = cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("get after delete failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	cache := NewMemoryCache(&MemoryCacheConfig{
		MaxEntries: 100,
		DefaultTTL: 50 * time.Millisecond,
	})
	ctx := context.Background()

	key := NewKey("test-agent", uuid.New(), "test", "test intent")
	result := &Result{
		Response:  "test response",
		CreatedAt: time.Now(),
	}

	// Put with short TTL
	err := cache.Put(ctx, key, result, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	// Should exist initially
	got, _ := cache.Get(ctx, key)
	if got == nil {
		t.Fatal("expected result before expiration")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	got, _ = cache.Get(ctx, key)
	if got != nil {
		t.Error("expected nil after expiration")
	}
}

func TestMemoryCache_Eviction(t *testing.T) {
	cache := NewMemoryCache(&MemoryCacheConfig{
		MaxEntries: 3,
		DefaultTTL: 5 * time.Minute,
	})
	ctx := context.Background()

	// Add entries with small delays to ensure ordering
	for i := 0; i < 5; i++ {
		key := NewKey("agent", uuid.New(), "test", string(rune('a'+i)))
		result := &Result{
			Response:  string(rune('a' + i)),
			CreatedAt: time.Now(),
		}
		_ = cache.Put(ctx, key, result, 5*time.Minute)
		time.Sleep(10 * time.Millisecond)
	}

	// Should only have 3 entries
	if cache.Size() != 3 {
		t.Errorf("expected size 3, got %d", cache.Size())
	}
}

func TestMemoryCache_Stats(t *testing.T) {
	cache := NewMemoryCache(nil)
	ctx := context.Background()

	key := NewKey("test-agent", uuid.New(), "test", "test intent")
	result := &Result{
		Response:     "test",
		CreatedAt:    time.Now(),
		ApproxTokens: 50,
	}

	// Miss
	_, _ = cache.Get(ctx, key)

	// Put
	_ = cache.Put(ctx, key, result, 5*time.Minute)

	// Hit
	_, _ = cache.Get(ctx, key)
	_, _ = cache.Get(ctx, key)

	stats := cache.Stats()

	if stats.Misses != 1 {
		t.Errorf("expected misses=1, got %d", stats.Misses)
	}

	if stats.Hits != 2 {
		t.Errorf("expected hits=2, got %d", stats.Hits)
	}

	if stats.Puts != 1 {
		t.Errorf("expected puts=1, got %d", stats.Puts)
	}

	// Hit rate should be 2/3 = 0.666...
	if stats.HitRate < 0.6 || stats.HitRate > 0.7 {
		t.Errorf("unexpected hit rate: %f", stats.HitRate)
	}

	// Tokens saved = 2 hits * 50 tokens
	if stats.TotalSaved != 100 {
		t.Errorf("expected total_saved=100, got %d", stats.TotalSaved)
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache(nil)
	ctx := context.Background()

	// Add some entries
	for i := 0; i < 5; i++ {
		key := NewKey("agent", uuid.New(), "test", string(rune('a'+i)))
		_ = cache.Put(ctx, key, &Result{Response: "test", CreatedAt: time.Now()}, 5*time.Minute)
	}

	if cache.Size() != 5 {
		t.Fatalf("expected size 5, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", cache.Size())
	}
}

func TestMemoryCache_HitCountIncrement(t *testing.T) {
	cache := NewMemoryCache(nil)
	ctx := context.Background()

	key := NewKey("test-agent", uuid.New(), "test", "test intent")
	result := &Result{
		Response:  "test",
		CreatedAt: time.Now(),
		HitCount:  0,
	}

	_ = cache.Put(ctx, key, result, 5*time.Minute)

	// Get multiple times
	for i := 0; i < 5; i++ {
		got, _ := cache.Get(ctx, key)
		if got.HitCount != i+1 {
			t.Errorf("expected hit_count=%d, got %d", i+1, got.HitCount)
		}
	}
}

func TestNoOpCache(t *testing.T) {
	cache := NewNoOpCache()
	ctx := context.Background()

	key := NewKey("test-agent", uuid.New(), "test", "test intent")
	result := &Result{Response: "test", CreatedAt: time.Now()}

	// Put should succeed but not store
	err := cache.Put(ctx, key, result, 5*time.Minute)
	if err != nil {
		t.Errorf("put should not error: %v", err)
	}

	// Get should always return nil
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Errorf("get should not error: %v", err)
	}
	if got != nil {
		t.Error("get should always return nil")
	}

	// Stats should be empty
	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("stats should be empty")
	}
}

func TestCacheWrapper_GetOrCompute(t *testing.T) {
	cache := NewMemoryCache(nil)
	wrapper := Wrap(cache)
	ctx := context.Background()

	key := NewKey("test-agent", uuid.New(), "test", "test intent")
	computeCount := 0

	compute := func() (string, int, error) {
		computeCount++
		return "computed response", 75, nil
	}

	// First call should compute
	resp1, err := wrapper.GetOrCompute(ctx, key, 5*time.Minute, compute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp1.CacheHit {
		t.Error("first call should be cache miss")
	}
	if computeCount != 1 {
		t.Errorf("expected compute_count=1, got %d", computeCount)
	}

	// Second call should hit cache
	resp2, err := wrapper.GetOrCompute(ctx, key, 5*time.Minute, compute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp2.CacheHit {
		t.Error("second call should be cache hit")
	}
	if computeCount != 1 {
		t.Errorf("expected compute_count=1 (no new compute), got %d", computeCount)
	}

	// Response should match
	if resp2.Result.Response != "computed response" {
		t.Errorf("unexpected response: %q", resp2.Result.Response)
	}
}

func TestMarshalUnmarshalKey(t *testing.T) {
	key := &Key{
		AgentName:   "test-agent",
		OrgID:       uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		Environment: "staging",
		IntentHash:  "abcdef123456",
	}

	data, err := MarshalKey(key)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	unmarshaled, err := UnmarshalKey(data)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if unmarshaled.AgentName != key.AgentName {
		t.Errorf("agent_name mismatch: %q vs %q", unmarshaled.AgentName, key.AgentName)
	}

	if unmarshaled.OrgID != key.OrgID {
		t.Errorf("org_id mismatch: %s vs %s", unmarshaled.OrgID, key.OrgID)
	}
}

func TestMarshalUnmarshalResult(t *testing.T) {
	result := &Result{
		Response:     "test response",
		CreatedAt:    time.Now().Truncate(time.Second), // Truncate for comparison
		HitCount:     5,
		ApproxTokens: 100,
		Model:        "claude-3",
		Metadata: map[string]any{
			"key": "value",
		},
	}

	data, err := MarshalResult(result)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	unmarshaled, err := UnmarshalResult(data)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if unmarshaled.Response != result.Response {
		t.Errorf("response mismatch")
	}

	if unmarshaled.HitCount != result.HitCount {
		t.Errorf("hit_count mismatch")
	}

	if unmarshaled.ApproxTokens != result.ApproxTokens {
		t.Errorf("approx_tokens mismatch")
	}

	if unmarshaled.Model != result.Model {
		t.Errorf("model mismatch")
	}
}
