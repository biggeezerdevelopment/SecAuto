package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimitConfig defines rate limits for different endpoints
type RateLimitConfig struct {
	Requests int           `json:"requests"`
	Window   time.Duration `json:"window"`
}

// RateLimiter manages rate limiting per IP
type RateLimiter struct {
	limits   map[string]RateLimitConfig
	requests map[string][]time.Time // IP -> slice of request times
	mutex    sync.RWMutex
	cleanup  chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *Config) *RateLimiter {
	// Parse window size from config
	windowSize := time.Minute // default
	if config.Security.RateLimiting.WindowSize != "" {
		if duration, err := time.ParseDuration(config.Security.RateLimiting.WindowSize); err == nil {
			windowSize = duration
		}
	}

	// Use config values if rate limiting is enabled
	requestsPerMinute := 100 // default
	if config.Security.RateLimiting.Enabled {
		requestsPerMinute = config.Security.RateLimiting.RequestsPerMinute
	}

	// Get endpoint-specific limits from config, with fallbacks
	endpoints := config.Security.RateLimiting.Endpoints

	// Use endpoint-specific limits if configured, otherwise calculate from total
	healthLimit := endpoints.Health
	if healthLimit == 0 {
		healthLimit = 1000 // Unlimited for health checks
	}

	playbookLimit := endpoints.Playbook
	if playbookLimit == 0 {
		playbookLimit = requestsPerMinute / 6 // 1/6 of total
	}

	asyncLimit := endpoints.PlaybookAsync
	if asyncLimit == 0 {
		asyncLimit = requestsPerMinute * 2 / 3 // 2/3 of total (very generous for async)
	}

	jobsLimit := endpoints.Jobs
	if jobsLimit == 0 {
		jobsLimit = requestsPerMinute / 4 // 1/4 of total
	}

	jobStatusLimit := endpoints.JobStatus
	if jobStatusLimit == 0 {
		jobStatusLimit = requestsPerMinute / 4 // 1/4 of total
	}

	jobStatsLimit := endpoints.JobStats
	if jobStatsLimit == 0 {
		jobStatsLimit = requestsPerMinute / 4 // 1/4 of total
	}

	jobMetricsLimit := endpoints.JobMetrics
	if jobMetricsLimit == 0 {
		jobMetricsLimit = requestsPerMinute / 4 // 1/4 of total
	}

	contextLimit := endpoints.Context
	if contextLimit == 0 {
		contextLimit = requestsPerMinute / 4 // 1/4 of total
	}

	webhooksLimit := endpoints.Webhooks
	if webhooksLimit == 0 {
		webhooksLimit = requestsPerMinute / 5 // 1/5 of total
	}

	pluginsLimit := endpoints.Plugins
	if pluginsLimit == 0 {
		pluginsLimit = requestsPerMinute / 5 // 1/5 of total
	}

	clusterLimit := endpoints.Cluster
	if clusterLimit == 0 {
		clusterLimit = requestsPerMinute / 5 // 1/5 of total
	}

	schedulesLimit := endpoints.Schedules
	if schedulesLimit == 0 {
		schedulesLimit = requestsPerMinute / 5 // 1/5 of total
	}

	defaultLimit := endpoints.Default
	if defaultLimit == 0 {
		defaultLimit = requestsPerMinute // Default for all other endpoints
	}

	rl := &RateLimiter{
		limits: map[string]RateLimitConfig{
			"/health":         {Requests: healthLimit, Window: time.Minute},
			"/playbook":       {Requests: playbookLimit, Window: windowSize},
			"/playbook/async": {Requests: asyncLimit, Window: windowSize},
			"/jobs":           {Requests: jobsLimit, Window: windowSize},
			"/jobs/stats":     {Requests: jobStatsLimit, Window: windowSize},
			"/jobs/metrics":   {Requests: jobMetricsLimit, Window: windowSize},
			"/job/":           {Requests: jobStatusLimit, Window: windowSize},
			"/context":        {Requests: contextLimit, Window: windowSize},
			"/webhooks":       {Requests: webhooksLimit, Window: windowSize},
			"/plugins":        {Requests: pluginsLimit, Window: windowSize},
			"/plugins/":       {Requests: pluginsLimit, Window: windowSize},
			"/cluster":        {Requests: clusterLimit, Window: windowSize},
			"/cluster/":       {Requests: clusterLimit, Window: windowSize},
			"/schedules":      {Requests: schedulesLimit, Window: windowSize},
			"/schedules/":     {Requests: schedulesLimit, Window: windowSize},
			"default":         {Requests: defaultLimit, Window: windowSize},
		},
		requests: make(map[string][]time.Time),
		cleanup:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupOldRequests()

	// Log the rate limits being applied
	logger.Info("Rate limiting configured", map[string]interface{}{
		"component": "rate_limit",
		"enabled":   config.Security.RateLimiting.Enabled,
		"limits": map[string]interface{}{
			"health":         healthLimit,
			"playbook":       playbookLimit,
			"playbook_async": asyncLimit,
			"jobs":           jobsLimit,
			"job_status":     jobStatusLimit,
			"job_stats":      jobStatsLimit,
			"job_metrics":    jobMetricsLimit,
			"context":        contextLimit,
			"webhooks":       webhooksLimit,
			"plugins":        pluginsLimit,
			"cluster":        clusterLimit,
			"schedules":      schedulesLimit,
			"default":        defaultLimit,
		},
		"window_size": windowSize.String(),
	})

	return rl
}

// isAllowed checks if the request is within rate limits
func (rl *RateLimiter) isAllowed(ip, path string) (bool, int, int, time.Time) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Get limit for this path
	limit, exists := rl.limits[path]
	if !exists {
		limit = rl.limits["default"]
	}

	// Clean old requests for this IP
	now := time.Now()
	windowStart := now.Add(-limit.Window)

	var validRequests []time.Time
	for _, reqTime := range rl.requests[ip] {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}
	rl.requests[ip] = validRequests

	// Check if within limit
	remaining := limit.Requests - len(validRequests)
	allowed := remaining > 0

	if allowed {
		rl.requests[ip] = append(rl.requests[ip], now)
	}

	// Calculate reset time
	resetTime := now.Add(limit.Window)
	if len(validRequests) > 0 {
		resetTime = validRequests[0].Add(limit.Window)
	}

	return allowed, remaining, limit.Requests, resetTime
}

// cleanupOldRequests periodically cleans up old request records
func (rl *RateLimiter) cleanupOldRequests() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mutex.Lock()
			now := time.Now()
			for ip, requests := range rl.requests {
				var validRequests []time.Time
				for _, reqTime := range requests {
					if now.Sub(reqTime) < 2*time.Hour { // Keep last 2 hours
						validRequests = append(validRequests, reqTime)
					}
				}
				if len(validRequests) == 0 {
					delete(rl.requests, ip)
				} else {
					rl.requests[ip] = validRequests
				}
			}
			rl.mutex.Unlock()
		case <-rl.cleanup:
			return
		}
	}
}

// rateLimitMiddleware enforces rate limiting
func rateLimitMiddleware(rl *RateLimiter) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			path := r.URL.Path

			allowed, remaining, limit, resetTime := rl.isAllowed(ip, path)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			if !allowed {
				logger.Warning("Rate limit exceeded", map[string]interface{}{
					"component": "rate_limit",
					"ip":        ip,
					"path":      path,
					"limit":     limit,
					"remaining": remaining,
				})
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", time.Until(resetTime).Seconds()))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next(w, r)
		}
	}
}
