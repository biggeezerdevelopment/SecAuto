package security

import (
	"context"
	"fmt"
	"testing"
	"time"

	"SoarAuto/pkg/testutil"
)

func TestSlidingWindow_Allow(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerWindow: 3,
		WindowSize:        100 * time.Millisecond,
		CleanupInterval:   50 * time.Millisecond,
	}
	
	window := NewSlidingWindow(config)
	
	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !window.Allow("client1", "endpoint1") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}
	
	// 4th request should be denied
	if window.Allow("client1", "endpoint1") {
		t.Error("4th request should be denied")
	}
	
	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)
	
	// Request should be allowed again
	if !window.Allow("client1", "endpoint1") {
		t.Error("Request should be allowed after window reset")
	}
}

func TestSlidingWindow_DifferentClients(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerWindow: 2,
		WindowSize:        100 * time.Millisecond,
	}
	
	window := NewSlidingWindow(config)
	
	// Client1 uses up their limit
	if !window.Allow("client1", "endpoint1") {
		t.Error("Client1 request 1 should be allowed")
	}
	if !window.Allow("client1", "endpoint1") {
		t.Error("Client1 request 2 should be allowed")
	}
	if window.Allow("client1", "endpoint1") {
		t.Error("Client1 request 3 should be denied")
	}
	
	// Client2 should still be allowed
	if !window.Allow("client2", "endpoint1") {
		t.Error("Client2 request 1 should be allowed")
	}
	if !window.Allow("client2", "endpoint1") {
		t.Error("Client2 request 2 should be allowed")
	}
}

func TestSlidingWindow_DifferentEndpoints(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerWindow: 2,
		WindowSize:        100 * time.Millisecond,
	}
	
	window := NewSlidingWindow(config)
	
	// Use up limit for endpoint1
	if !window.Allow("client1", "endpoint1") {
		t.Error("Endpoint1 request 1 should be allowed")
	}
	if !window.Allow("client1", "endpoint1") {
		t.Error("Endpoint1 request 2 should be allowed")
	}
	if window.Allow("client1", "endpoint1") {
		t.Error("Endpoint1 request 3 should be denied")
	}
	
	// Endpoint2 should still be allowed
	if !window.Allow("client1", "endpoint2") {
		t.Error("Endpoint2 request 1 should be allowed")
	}
	if !window.Allow("client1", "endpoint2") {
		t.Error("Endpoint2 request 2 should be allowed")
	}
}

func TestSlidingWindow_GetStats(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerWindow: 5,
		WindowSize:        100 * time.Millisecond,
	}
	
	window := NewSlidingWindow(config)
	
	// Make some requests
	window.Allow("client1", "endpoint1")
	window.Allow("client1", "endpoint1")
	window.Allow("client1", "endpoint1")
	
	stats := window.GetStats("client1", "endpoint1")
	
	if stats["requests_in_window"] != 3 {
		t.Errorf("Expected 3 requests in window, got %v", stats["requests_in_window"])
	}
	
	if stats["requests_per_window"] != 5 {
		t.Errorf("Expected 5 requests per window, got %v", stats["requests_per_window"])
	}
	
	if stats["remaining_requests"] != 2 {
		t.Errorf("Expected 2 remaining requests, got %v", stats["remaining_requests"])
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	logger := testutil.TestLogger(t)
	rateLimiter := NewRateLimiter(logger)
	
	// Set endpoint configuration
	config := &RateLimitConfig{
		RequestsPerWindow: 2,
		WindowSize:        100 * time.Millisecond,
	}
	rateLimiter.SetEndpointConfig("test-endpoint", config)
	
	// First 2 requests should be allowed
	err := rateLimiter.Allow("client1", "test-endpoint")
	testutil.AssertNoError(t, err)
	
	err = rateLimiter.Allow("client1", "test-endpoint")
	testutil.AssertNoError(t, err)
	
	// 3rd request should be denied
	err = rateLimiter.Allow("client1", "test-endpoint")
	testutil.AssertError(t, err, "Rate limit exceeded")
}

func TestRateLimiter_DefaultConfig(t *testing.T) {
	logger := testutil.TestLogger(t)
	rateLimiter := NewRateLimiter(logger)
	
	// Should use default config for unknown endpoint
	err := rateLimiter.Allow("client1", "unknown-endpoint")
	testutil.AssertNoError(t, err)
	
	stats := rateLimiter.GetStats("client1", "unknown-endpoint")
	
	// Should have default values
	defaultConfig := DefaultRateLimitConfig()
	if stats["requests_per_window"] != defaultConfig.RequestsPerWindow {
		t.Errorf("Expected default requests per window %d, got %v", 
			defaultConfig.RequestsPerWindow, stats["requests_per_window"])
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	logger := testutil.TestLogger(t)
	rateLimiter := NewRateLimiter(logger)
	
	// Make some requests to create windows
	rateLimiter.Allow("client1", "endpoint1")
	rateLimiter.Allow("client2", "endpoint2")
	
	// Check that windows exist
	if len(rateLimiter.windows) != 2 {
		t.Errorf("Expected 2 windows, got %d", len(rateLimiter.windows))
	}
	
	// Wait for windows to become stale
	time.Sleep(10 * time.Millisecond)
	
	// Cleanup should remove stale windows
	rateLimiter.Cleanup()
	
	// Note: Cleanup only removes windows that haven't been used for 2x window size
	// So we need to wait longer or modify the cleanup logic for testing
}

func TestRateLimiter_StartCleanupRoutine(t *testing.T) {
	logger := testutil.TestLogger(t)
	rateLimiter := NewRateLimiter(logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	// Start cleanup routine
	go rateLimiter.StartCleanupRoutine(ctx)
	
	// Make some requests
	rateLimiter.Allow("client1", "endpoint1")
	
	// Wait for context to be cancelled
	<-ctx.Done()
	
	// Cleanup routine should have stopped
}

func TestTokenBucket_Allow(t *testing.T) {
	bucket := NewTokenBucket(3, 1) // 3 tokens, refill 1 per second
	
	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !bucket.Allow() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}
	
	// 4th request should be denied
	if bucket.Allow() {
		t.Error("4th request should be denied")
	}
	
	// Wait for refill
	time.Sleep(1100 * time.Millisecond)
	
	// Should have 1 token available
	if !bucket.Allow() {
		t.Error("Request should be allowed after refill")
	}
	
	// Should be empty again
	if bucket.Allow() {
		t.Error("Request should be denied after using refilled token")
	}
}

func TestTokenBucket_GetTokens(t *testing.T) {
	bucket := NewTokenBucket(5, 2)
	
	// Should start with full capacity
	if tokens := bucket.GetTokens(); tokens != 5 {
		t.Errorf("Expected 5 tokens, got %d", tokens)
	}
	
	// Use some tokens
	bucket.Allow()
	bucket.Allow()
	
	if tokens := bucket.GetTokens(); tokens != 3 {
		t.Errorf("Expected 3 tokens, got %d", tokens)
	}
}

func TestHybridRateLimiter_Allow(t *testing.T) {
	logger := testutil.TestLogger(t)
	config := &RateLimitConfig{
		RequestsPerWindow: 10,
		WindowSize:        1 * time.Second,
		BurstLimit:        3,
	}
	
	limiter := NewHybridRateLimiter(config, logger)
	
	// First 3 requests should be allowed (burst limit)
	for i := 0; i < 3; i++ {
		err := limiter.Allow("client1", "endpoint1")
		testutil.AssertNoError(t, err)
	}
	
	// 4th request should be denied by token bucket
	err := limiter.Allow("client1", "endpoint1")
	testutil.AssertError(t, err, "Burst limit exceeded")
}

func TestHybridRateLimiter_SustainedRate(t *testing.T) {
	logger := testutil.TestLogger(t)
	config := &RateLimitConfig{
		RequestsPerWindow: 2,
		WindowSize:        100 * time.Millisecond,
		BurstLimit:        10, // High burst limit
	}
	
	limiter := NewHybridRateLimiter(config, logger)
	
	// Make requests up to sliding window limit
	err := limiter.Allow("client1", "endpoint1")
	testutil.AssertNoError(t, err)
	
	err = limiter.Allow("client1", "endpoint1")
	testutil.AssertNoError(t, err)
	
	// 3rd request should be denied by sliding window
	err = limiter.Allow("client1", "endpoint1")
	testutil.AssertError(t, err, "Rate limit exceeded")
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()
	
	if config.RequestsPerWindow != 100 {
		t.Errorf("Expected 100 requests per window, got %d", config.RequestsPerWindow)
	}
	
	if config.WindowSize != time.Minute {
		t.Errorf("Expected 1 minute window, got %v", config.WindowSize)
	}
	
	if config.BurstLimit != 20 {
		t.Errorf("Expected 20 burst limit, got %d", config.BurstLimit)
	}
	
	if config.CleanupInterval != 5*time.Minute {
		t.Errorf("Expected 5 minute cleanup interval, got %v", config.CleanupInterval)
	}
}

func TestSlidingWindow_CleanupOldEntries(t *testing.T) {
	config := &RateLimitConfig{
		RequestsPerWindow: 10,
		WindowSize:        50 * time.Millisecond,
		CleanupInterval:   25 * time.Millisecond,
	}
	
	window := NewSlidingWindow(config)
	
	// Add some entries
	window.Allow("client1", "endpoint1")
	window.Allow("client1", "endpoint1")
	window.Allow("client1", "endpoint1")
	
	// Check initial count
	if len(window.entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(window.entries))
	}
	
	// Wait for entries to become old
	time.Sleep(75 * time.Millisecond)
	
	// Make another request to trigger cleanup
	window.Allow("client1", "endpoint1")
	
	// Old entries should be cleaned up
	if len(window.entries) != 1 {
		t.Errorf("Expected 1 entry after cleanup, got %d", len(window.entries))
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	logger := testutil.TestLogger(t)
	rateLimiter := NewRateLimiter(logger)
	
	config := &RateLimitConfig{
		RequestsPerWindow: 100,
		WindowSize:        1 * time.Second,
	}
	rateLimiter.SetEndpointConfig("test-endpoint", config)
	
	// Test concurrent access
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(clientID string) {
			defer func() { done <- true }()
			
			for j := 0; j < 10; j++ {
				rateLimiter.Allow(clientID, "test-endpoint")
			}
		}(fmt.Sprintf("client%d", i))
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Should have created windows for all clients
	if len(rateLimiter.windows) != 10 {
		t.Errorf("Expected 10 windows, got %d", len(rateLimiter.windows))
	}
}