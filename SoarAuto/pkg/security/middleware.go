package security

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"SoarAuto/pkg/errors"
	"SoarAuto/pkg/types"
)

// SecurityMiddleware provides comprehensive security middleware
type SecurityMiddleware struct {
	validator    *SecurityValidator
	rateLimiter  *RateLimiter
	logger       types.Logger
	config       *SecurityConfig
}

// SecurityConfig holds security middleware configuration
type SecurityConfig struct {
	EnableCSRFProtection    bool     `json:"enable_csrf_protection"`
	EnableXSSProtection     bool     `json:"enable_xss_protection"`
	EnableContentTypeSniff  bool     `json:"enable_content_type_sniff"`
	EnableFrameOptions      bool     `json:"enable_frame_options"`
	EnableHSTS              bool     `json:"enable_hsts"`
	HSTSMaxAge              int      `json:"hsts_max_age"`
	TrustedProxies          []string `json:"trusted_proxies"`
	MaxRequestSize          int64    `json:"max_request_size"`
	EnableRateLimiting      bool     `json:"enable_rate_limiting"`
	EnableInputValidation   bool     `json:"enable_input_validation"`
	BlockedUserAgents       []string `json:"blocked_user_agents"`
	BlockedIPs              []string `json:"blocked_ips"`
	RequireHTTPS            bool     `json:"require_https"`
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		EnableCSRFProtection:   true,
		EnableXSSProtection:    true,
		EnableContentTypeSniff: false,
		EnableFrameOptions:     true,
		EnableHSTS:             true,
		HSTSMaxAge:             31536000, // 1 year
		TrustedProxies:         []string{"127.0.0.1", "::1"},
		MaxRequestSize:         10 * 1024 * 1024, // 10MB
		EnableRateLimiting:     true,
		EnableInputValidation:  true,
		BlockedUserAgents:      []string{"bot", "crawler", "spider"},
		BlockedIPs:             []string{},
		RequireHTTPS:           false, // Set to true in production
	}
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(logger types.Logger, config *SecurityConfig) *SecurityMiddleware {
	if config == nil {
		config = DefaultSecurityConfig()
	}
	
	return &SecurityMiddleware{
		validator:   NewSecurityValidator(),
		rateLimiter: NewRateLimiter(logger),
		logger:      logger,
		config:      config,
	}
}

// SecurityHeaders adds security headers to the response
func (sm *SecurityMiddleware) SecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// X-Content-Type-Options
		if !sm.config.EnableContentTypeSniff {
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}
		
		// X-Frame-Options
		if sm.config.EnableFrameOptions {
			w.Header().Set("X-Frame-Options", "DENY")
		}
		
		// X-XSS-Protection
		if sm.config.EnableXSSProtection {
			w.Header().Set("X-XSS-Protection", "1; mode=block")
		}
		
		// Strict-Transport-Security (HSTS)
		if sm.config.EnableHSTS && r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", 
				fmt.Sprintf("max-age=%d; includeSubDomains", sm.config.HSTSMaxAge))
		}
		
		// Content-Security-Policy
		csp := "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';"
		w.Header().Set("Content-Security-Policy", csp)
		
		// Referrer-Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Permissions-Policy
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		next(w, r)
	}
}

// HTTPSRedirect redirects HTTP requests to HTTPS
func (sm *SecurityMiddleware) HTTPSRedirect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if sm.config.RequireHTTPS && r.TLS == nil {
			// Check if request came through a trusted proxy with HTTPS
			if r.Header.Get("X-Forwarded-Proto") != "https" {
				httpsURL := "https://" + r.Host + r.RequestURI
				http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
				return
			}
		}
		
		next(w, r)
	}
}

// RequestSizeLimit limits the size of incoming requests
func (sm *SecurityMiddleware) RequestSizeLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > sm.config.MaxRequestSize {
			sm.logger.Warning("Request size limit exceeded", map[string]interface{}{
				"component":      "security_middleware",
				"content_length": r.ContentLength,
				"max_size":       sm.config.MaxRequestSize,
				"remote_addr":    r.RemoteAddr,
				"user_agent":     r.UserAgent(),
			})
			
			err := errors.ValidationError("Request too large", map[string]interface{}{
				"content_length": r.ContentLength,
				"max_size":       sm.config.MaxRequestSize,
			})
			
			sm.writeErrorResponse(w, err)
			return
		}
		
		// Limit request body size
		r.Body = http.MaxBytesReader(w, r.Body, sm.config.MaxRequestSize)
		
		next(w, r)
	}
}

// IPBlocking blocks requests from blocked IP addresses
func (sm *SecurityMiddleware) IPBlocking(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := sm.getClientIP(r)
		
		// Check if IP is blocked
		for _, blockedIP := range sm.config.BlockedIPs {
			if clientIP == blockedIP {
				sm.logger.Warning("Blocked IP attempted access", map[string]interface{}{
					"component":   "security_middleware",
					"blocked_ip":  clientIP,
					"user_agent":  r.UserAgent(),
					"request_uri": r.RequestURI,
				})
				
				err := errors.AuthError(errors.ErrCodeAuthPermission, "Access denied")
				sm.writeErrorResponse(w, err)
				return
			}
		}
		
		next(w, r)
	}
}

// UserAgentBlocking blocks requests from blocked user agents
func (sm *SecurityMiddleware) UserAgentBlocking(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userAgent := strings.ToLower(r.UserAgent())
		
		// Check if user agent contains blocked patterns
		for _, blockedUA := range sm.config.BlockedUserAgents {
			if strings.Contains(userAgent, strings.ToLower(blockedUA)) {
				sm.logger.Warning("Blocked user agent attempted access", map[string]interface{}{
					"component":   "security_middleware",
					"user_agent":  r.UserAgent(),
					"remote_addr": r.RemoteAddr,
					"request_uri": r.RequestURI,
				})
				
				err := errors.AuthError(errors.ErrCodeAuthPermission, "Access denied")
				sm.writeErrorResponse(w, err)
				return
			}
		}
		
		next(w, r)
	}
}

// RateLimitingMiddleware applies rate limiting
func (sm *SecurityMiddleware) RateLimitingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sm.config.EnableRateLimiting {
			next(w, r)
			return
		}
		
		clientID := sm.getClientID(r)
		endpoint := r.URL.Path
		
		// Check rate limit
		if err := sm.rateLimiter.Allow(clientID, endpoint); err != nil {
			// Add rate limit headers
			stats := sm.rateLimiter.GetStats(clientID, endpoint)
			if remaining, ok := stats["remaining_requests"].(int); ok {
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			}
			if resetTime, ok := stats["reset_time"].(int64); ok {
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
			}
			if limit, ok := stats["requests_per_window"].(int); ok {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			}
			
			sm.writeErrorResponse(w, err)
			return
		}
		
		next(w, r)
	}
}

// InputValidationMiddleware validates request inputs
func (sm *SecurityMiddleware) InputValidationMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sm.config.EnableInputValidation {
			next(w, r)
			return
		}
		
		// Validate query parameters
		for key, values := range r.URL.Query() {
			for _, value := range values {
				if err := sm.validator.ValidateInput(value, fmt.Sprintf("query_%s", key)); err != nil {
					sm.logger.Warning("Invalid query parameter detected", map[string]interface{}{
						"component":   "security_middleware",
						"parameter":   key,
						"remote_addr": r.RemoteAddr,
						"error":       err.Error(),
					})
					
					sm.writeErrorResponse(w, err)
					return
				}
			}
		}
		
		// Validate headers (common attack vectors)
		suspiciousHeaders := []string{"User-Agent", "Referer", "X-Forwarded-For"}
		for _, header := range suspiciousHeaders {
			if value := r.Header.Get(header); value != "" {
				if err := sm.validator.ValidateInput(value, fmt.Sprintf("header_%s", header)); err != nil {
					sm.logger.Warning("Invalid header detected", map[string]interface{}{
						"component":   "security_middleware",
						"header":      header,
						"remote_addr": r.RemoteAddr,
						"error":       err.Error(),
					})
					
					sm.writeErrorResponse(w, err)
					return
				}
			}
		}
		
		next(w, r)
	}
}

// getClientIP extracts the real client IP address
func (sm *SecurityMiddleware) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (from trusted proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			// Validate that the request came from a trusted proxy
			if sm.isTrustedProxy(r.RemoteAddr) {
				return clientIP
			}
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" && sm.isTrustedProxy(r.RemoteAddr) {
		return xri
	}
	
	// Fall back to RemoteAddr
	if ip := strings.Split(r.RemoteAddr, ":"); len(ip) > 0 {
		return ip[0]
	}
	
	return r.RemoteAddr
}

// getClientID generates a client ID for rate limiting
func (sm *SecurityMiddleware) getClientID(r *http.Request) string {
	// Try to get API key first
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return "api:" + apiKey[:8] // Use first 8 chars for privacy
	}
	
	// Fall back to IP address
	return "ip:" + sm.getClientIP(r)
}

// isTrustedProxy checks if the request comes from a trusted proxy
func (sm *SecurityMiddleware) isTrustedProxy(remoteAddr string) bool {
	ip := strings.Split(remoteAddr, ":")[0]
	for _, trustedIP := range sm.config.TrustedProxies {
		if ip == trustedIP {
			return true
		}
	}
	return false
}

// writeErrorResponse writes a standardized error response
func (sm *SecurityMiddleware) writeErrorResponse(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	
	if secAutoErr, ok := err.(*errors.SecAutoError); ok {
		w.WriteHeader(secAutoErr.HTTPStatusCode())
		json.NewEncoder(w).Encode(secAutoErr.ToAPIResponse())
	} else {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   false,
			"message":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// ValidateAPIKey validates an API key format
func (sm *SecurityMiddleware) ValidateAPIKey(apiKey string) error {
	return sm.validator.ValidateAPIKey(apiKey)
}

// CombinedSecurityMiddleware combines all security middleware
func (sm *SecurityMiddleware) CombinedSecurityMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return sm.SecurityHeaders(
		sm.HTTPSRedirect(
			sm.RequestSizeLimit(
				sm.IPBlocking(
					sm.UserAgentBlocking(
						sm.RateLimitingMiddleware(
							sm.InputValidationMiddleware(next),
						),
					),
				),
			),
		),
	)
}