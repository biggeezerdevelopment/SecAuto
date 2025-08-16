package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"SoarAuto/pkg/errors"
	"SoarAuto/pkg/types"
)

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	RequestsPerWindow int           `json:"requests_per_window"`
	WindowSize        time.Duration `json:"window_size"`
	BurstLimit        int           `json:"burst_limit"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
}

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerWindow: 100,
		WindowSize:        time.Minute,
		BurstLimit:        20,
		CleanupInterval:   5 * time.Minute,
	}
}

// SlidingWindowEntry represents a single request in the sliding window
type SlidingWindowEntry struct {
	Timestamp time.Time
	ClientID  string
	Endpoint  string
}

// SlidingWindow implements a sliding window rate limiter
type SlidingWindow struct {
	config    *RateLimitConfig
	entries   []SlidingWindowEntry
	mutex     sync.RWMutex
	lastClean time.Time
}

// NewSlidingWindow creates a new sliding window rate limiter
func NewSlidingWindow(config *RateLimitConfig) *SlidingWindow {
	if config == nil {
		config = DefaultRateLimitConfig()
	}
	
	return &SlidingWindow{
		config:    config,
		entries:   make([]SlidingWindowEntry, 0),
		lastClean: time.Now(),
	}
}

// Allow checks if a request should be allowed
func (sw *SlidingWindow) Allow(clientID, endpoint string) bool {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	
	now := time.Now()
	
	// Clean old entries periodically
	if now.Sub(sw.lastClean) > sw.config.CleanupInterval {
		sw.cleanOldEntries(now)
		sw.lastClean = now
	}
	
	// Count requests in the current window
	windowStart := now.Add(-sw.config.WindowSize)
	count := 0
	
	for _, entry := range sw.entries {
		if entry.Timestamp.After(windowStart) && 
		   entry.ClientID == clientID && 
		   entry.Endpoint == endpoint {
			count++
		}
	}
	
	// Check if request should be allowed
	if count >= sw.config.RequestsPerWindow {
		return false
	}
	
	// Add the current request
	sw.entries = append(sw.entries, SlidingWindowEntry{
		Timestamp: now,
		ClientID:  clientID,
		Endpoint:  endpoint,
	})
	
	return true
}

// cleanOldEntries removes entries older than the window size
func (sw *SlidingWindow) cleanOldEntries(now time.Time) {
	windowStart := now.Add(-sw.config.WindowSize)
	
	// Filter out old entries
	validEntries := make([]SlidingWindowEntry, 0, len(sw.entries))
	for _, entry := range sw.entries {
		if entry.Timestamp.After(windowStart) {
			validEntries = append(validEntries, entry)
		}
	}
	
	sw.entries = validEntries
}

// GetStats returns current rate limiting statistics
func (sw *SlidingWindow) GetStats(clientID, endpoint string) map[string]interface{} {
	sw.mutex.RLock()
	defer sw.mutex.RUnlock()
	
	now := time.Now()
	windowStart := now.Add(-sw.config.WindowSize)
	
	count := 0
	for _, entry := range sw.entries {
		if entry.Timestamp.After(windowStart) && 
		   entry.ClientID == clientID && 
		   entry.Endpoint == endpoint {
			count++
		}
	}
	
	return map[string]interface{}{
		"requests_in_window": count,
		"requests_per_window": sw.config.RequestsPerWindow,
		"window_size_seconds": sw.config.WindowSize.Seconds(),
		"remaining_requests": sw.config.RequestsPerWindow - count,
		"reset_time": windowStart.Add(sw.config.WindowSize).Unix(),
	}
}

// RateLimiter manages rate limiting for multiple clients and endpoints
type RateLimiter struct {
	windows map[string]*SlidingWindow
	configs map[string]*RateLimitConfig
	mutex   sync.RWMutex
	logger  types.Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(logger types.Logger) *RateLimiter {
	return &RateLimiter{
		windows: make(map[string]*SlidingWindow),
		configs: make(map[string]*RateLimitConfig),
		logger:  logger,
	}
}

// SetEndpointConfig sets rate limiting configuration for a specific endpoint
func (rl *RateLimiter) SetEndpointConfig(endpoint string, config *RateLimitConfig) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	rl.configs[endpoint] = config
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(clientID, endpoint string) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	// Get or create sliding window for this client-endpoint combination
	key := fmt.Sprintf("%s:%s", clientID, endpoint)
	window, exists := rl.windows[key]
	
	if !exists {
		// Get configuration for this endpoint
		config, configExists := rl.configs[endpoint]
		if !configExists {
			config = DefaultRateLimitConfig()
		}
		
		window = NewSlidingWindow(config)
		rl.windows[key] = window
	}
	
	// Check if request is allowed
	if !window.Allow(clientID, endpoint) {
		stats := window.GetStats(clientID, endpoint)
		
		rl.logger.Warning("Rate limit exceeded", map[string]interface{}{
			"component": "rate_limiter",
			"client_id": clientID,
			"endpoint":  endpoint,
			"stats":     stats,
		})
		
		return errors.NewErrorBuilder(errors.ErrCodeSystemResource, "Rate limit exceeded").
			WithComponent("rate_limiter").
			WithSeverity(errors.SeverityMedium).
			WithContext("client_id", clientID).
			WithContext("endpoint", endpoint).
			WithContext("stats", stats).
			WithRetryable(true).
			Build()
	}
	
	return nil
}

// GetStats returns rate limiting statistics for a client-endpoint combination
func (rl *RateLimiter) GetStats(clientID, endpoint string) map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	
	key := fmt.Sprintf("%s:%s", clientID, endpoint)
	window, exists := rl.windows[key]
	
	if !exists {
		config, configExists := rl.configs[endpoint]
		if !configExists {
			config = DefaultRateLimitConfig()
		}
		
		return map[string]interface{}{
			"requests_in_window": 0,
			"requests_per_window": config.RequestsPerWindow,
			"window_size_seconds": config.WindowSize.Seconds(),
			"remaining_requests": config.RequestsPerWindow,
			"reset_time": time.Now().Add(config.WindowSize).Unix(),
		}
	}
	
	return window.GetStats(clientID, endpoint)
}

// Cleanup removes old windows and entries
func (rl *RateLimiter) Cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	
	// Remove windows that haven't been used recently
	for key, window := range rl.windows {
		window.mutex.Lock()
		if len(window.entries) == 0 || 
		   now.Sub(window.entries[len(window.entries)-1].Timestamp) > window.config.WindowSize*2 {
			delete(rl.windows, key)
		}
		window.mutex.Unlock()
	}
}

// StartCleanupRoutine starts a background cleanup routine
func (rl *RateLimiter) StartCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.Cleanup()
		}
	}
}

// TokenBucket implements a token bucket rate limiter for burst handling
type TokenBucket struct {
	capacity     int
	tokens       int
	refillRate   int           // tokens per second
	lastRefill   time.Time
	mutex        sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity, refillRate int) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request should be allowed (consumes one token)
func (tb *TokenBucket) Allow() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	now := time.Now()
	
	// Refill tokens based on elapsed time
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int(elapsed.Seconds()) * tb.refillRate
	
	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}
	
	// Check if we have tokens available
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	
	return false
}

// GetTokens returns the current number of available tokens
func (tb *TokenBucket) GetTokens() int {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	return tb.tokens
}

// HybridRateLimiter combines sliding window and token bucket approaches
type HybridRateLimiter struct {
	slidingWindow *SlidingWindow
	tokenBucket   *TokenBucket
	config        *RateLimitConfig
	logger        types.Logger
}

// NewHybridRateLimiter creates a new hybrid rate limiter
func NewHybridRateLimiter(config *RateLimitConfig, logger types.Logger) *HybridRateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}
	
	return &HybridRateLimiter{
		slidingWindow: NewSlidingWindow(config),
		tokenBucket:   NewTokenBucket(config.BurstLimit, config.RequestsPerWindow/int(config.WindowSize.Seconds())),
		config:        config,
		logger:        logger,
	}
}

// Allow checks if a request should be allowed using both algorithms
func (hrl *HybridRateLimiter) Allow(clientID, endpoint string) error {
	// First check token bucket for burst protection
	if !hrl.tokenBucket.Allow() {
		hrl.logger.Warning("Token bucket limit exceeded", map[string]interface{}{
			"component": "hybrid_rate_limiter",
			"client_id": clientID,
			"endpoint":  endpoint,
			"tokens":    hrl.tokenBucket.GetTokens(),
		})
		
		return errors.NewErrorBuilder(errors.ErrCodeSystemResource, "Burst limit exceeded").
			WithComponent("hybrid_rate_limiter").
			WithSeverity(errors.SeverityHigh).
			WithContext("client_id", clientID).
			WithContext("endpoint", endpoint).
			WithContext("limit_type", "burst").
			WithRetryable(true).
			Build()
	}
	
	// Then check sliding window for sustained rate
	if !hrl.slidingWindow.Allow(clientID, endpoint) {
		stats := hrl.slidingWindow.GetStats(clientID, endpoint)
		
		hrl.logger.Warning("Sliding window limit exceeded", map[string]interface{}{
			"component": "hybrid_rate_limiter",
			"client_id": clientID,
			"endpoint":  endpoint,
			"stats":     stats,
		})
		
		return errors.NewErrorBuilder(errors.ErrCodeSystemResource, "Rate limit exceeded").
			WithComponent("hybrid_rate_limiter").
			WithSeverity(errors.SeverityMedium).
			WithContext("client_id", clientID).
			WithContext("endpoint", endpoint).
			WithContext("limit_type", "sustained").
			WithContext("stats", stats).
			WithRetryable(true).
			Build()
	}
	
	return nil
}