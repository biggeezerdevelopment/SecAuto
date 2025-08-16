package performance

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
)

func TestCacheKeyBuilder(t *testing.T) {
	builder := NewCacheKeyBuilder("secauto")

	tests := []struct {
		name     string
		method   func() string
		expected string
	}{
		{
			name:     "Build with components",
			method:   func() string { return builder.Build("test", "key", "123") },
			expected: "secauto:test:key:123",
		},
		{
			name:     "PlaybookKey",
			method:   func() string { return builder.PlaybookKey("test-playbook") },
			expected: "secauto:playbook:test-playbook",
		},
		{
			name:     "JobKey",
			method:   func() string { return builder.JobKey("job-123") },
			expected: "secauto:job:job-123",
		},
		{
			name:     "UserSessionKey",
			method:   func() string { return builder.UserSessionKey("user-456") },
			expected: "secauto:session:user-456",
		},
		{
			name:     "APIResponseKey",
			method:   func() string { return builder.APIResponseKey("playbook", "name=test") },
			expected: "secauto:api:playbook:name=test",
		},
		{
			name:     "ConfigKey",
			method:   func() string { return builder.ConfigKey("database") },
			expected: "secauto:config:database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.method()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCacheMetrics(t *testing.T) {
	metrics := &CacheMetrics{LastReset: time.Now()}

	// Test initial state
	if metrics.GetHitRate() != 0.0 {
		t.Errorf("Expected initial hit rate to be 0.0, got %f", metrics.GetHitRate())
	}

	if metrics.GetAverageLatency() != 0.0 {
		t.Errorf("Expected initial average latency to be 0.0, got %f", metrics.GetAverageLatency())
	}

	// Simulate some operations
	metrics.mu.Lock()
	metrics.Hits = 80
	metrics.Misses = 20
	metrics.Operations = 100
	metrics.TotalLatency = 500 // 500ms total
	metrics.mu.Unlock()

	expectedHitRate := 80.0
	if hitRate := metrics.GetHitRate(); hitRate != expectedHitRate {
		t.Errorf("Expected hit rate %f, got %f", expectedHitRate, hitRate)
	}

	expectedAvgLatency := 5.0 // 500ms / 100 operations
	if avgLatency := metrics.GetAverageLatency(); avgLatency != expectedAvgLatency {
		t.Errorf("Expected average latency %f, got %f", expectedAvgLatency, avgLatency)
	}

	// Test reset
	metrics.Reset()
	if metrics.Hits != 0 || metrics.Misses != 0 || metrics.Operations != 0 {
		t.Error("Metrics not properly reset")
	}
}

func TestCacheDisabled(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultCacheConfig()
	config.Enabled = false

	cache, err := NewCache(config, logger)
	if err != nil {
		t.Fatalf("Failed to create disabled cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test operations on disabled cache
	err = cache.Set(ctx, "test", "value", time.Minute)
	if err != nil {
		t.Errorf("Set on disabled cache should not return error, got: %v", err)
	}

	_, err = cache.Get(ctx, "test")
	if err == nil {
		t.Error("Get on disabled cache should return error")
	}

	exists, err := cache.Exists(ctx, "test")
	if err != nil || exists {
		t.Error("Exists on disabled cache should return false without error")
	}

	err = cache.Delete(ctx, "test")
	if err != nil {
		t.Errorf("Delete on disabled cache should not return error, got: %v", err)
	}
}

func TestCacheConfiguration(t *testing.T) {
	config := DefaultCacheConfig()

	// Test default values
	if config.RedisAddr != "localhost:6379" {
		t.Errorf("Expected default Redis address localhost:6379, got %s", config.RedisAddr)
	}

	if config.DefaultTTL != 15*time.Minute {
		t.Errorf("Expected default TTL 15m, got %v", config.DefaultTTL)
	}

	if config.PoolSize != 10 {
		t.Errorf("Expected default pool size 10, got %d", config.PoolSize)
	}

	if !config.Enabled {
		t.Error("Expected cache to be enabled by default")
	}
}

// TestCacheOperationsWithMockRedis tests cache operations without requiring Redis
func TestCacheOperationsWithMockRedis(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultCacheConfig()
	config.Enabled = false // Disable to avoid Redis dependency

	cache, err := NewCache(config, logger)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	// Test metrics
	metrics := cache.GetMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be available")
	}

	// Test ping on disabled cache
	ctx := context.Background()
	err = cache.Ping(ctx)
	if err == nil {
		t.Error("Ping on disabled cache should return error")
	}
}

func TestCacheJSONOperations(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultCacheConfig()
	config.Enabled = false // Disable to test JSON marshaling without Redis

	cache, err := NewCache(config, logger)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test JSON operations on disabled cache
	testData := map[string]interface{}{
		"name":  "test",
		"value": 123,
		"items": []string{"a", "b", "c"},
	}

	// SetJSON should not error on disabled cache
	err = cache.SetJSON(ctx, "test-json", testData, time.Minute)
	if err != nil {
		t.Errorf("SetJSON on disabled cache should not error, got: %v", err)
	}

	// GetJSON should error on disabled cache
	var result map[string]interface{}
	err = cache.GetJSON(ctx, "test-json", &result)
	if err == nil {
		t.Error("GetJSON on disabled cache should return error")
	}
}

func TestCacheIncrementAndExpire(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultCacheConfig()
	config.Enabled = false

	cache, err := NewCache(config, logger)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test increment on disabled cache
	_, err = cache.Increment(ctx, "counter")
	if err == nil {
		t.Error("Increment on disabled cache should return error")
	}

	// Test expire on disabled cache
	err = cache.Expire(ctx, "test", time.Minute)
	if err != nil {
		t.Errorf("Expire on disabled cache should not error, got: %v", err)
	}
}

// Benchmark tests for cache operations
func BenchmarkCacheKeyBuilder(b *testing.B) {
	builder := NewCacheKeyBuilder("secauto")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.Build("test", "key", "123")
	}
}

func BenchmarkCacheMetricsHitRate(b *testing.B) {
	metrics := &CacheMetrics{
		Hits:   800,
		Misses: 200,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetHitRate()
	}
}

func BenchmarkCacheMetricsAverageLatency(b *testing.B) {
	metrics := &CacheMetrics{
		Operations:   1000,
		TotalLatency: 5000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetAverageLatency()
	}
}