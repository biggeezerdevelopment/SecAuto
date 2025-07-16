package main

import (
	"net/http"
	"strings"
)

// corsMiddleware creates a CORS middleware based on configuration
func corsMiddleware(config *Config) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Only apply CORS if enabled in config
			if !config.Security.CORS.Enabled {
				next(w, r)
				return
			}

			// Set allowed origins
			origin := r.Header.Get("Origin")
			if origin != "" {
				allowed := false
				for _, allowedOrigin := range config.Security.CORS.AllowedOrigins {
					if allowedOrigin == "*" || allowedOrigin == origin {
						allowed = true
						break
					}
				}
				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}

			// Set allowed methods
			if len(config.Security.CORS.AllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.Security.CORS.AllowedMethods, ", "))
			}

			// Set allowed headers
			if len(config.Security.CORS.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.Security.CORS.AllowedHeaders, ", "))
			}

			// Set max age
			if config.Security.CORS.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", string(rune(config.Security.CORS.MaxAge)))
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.WriteHeader(http.StatusOK)
				return
			}

			// Call next handler
			next(w, r)
		}
	}
}

// corsPreflightHandler handles OPTIONS requests for CORS preflight
func corsPreflightHandler(config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !config.Security.CORS.Enabled {
			http.Error(w, "CORS not enabled", http.StatusForbidden)
			return
		}

		// Set CORS headers for preflight
		origin := r.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, allowedOrigin := range config.Security.CORS.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}

		if len(config.Security.CORS.AllowedMethods) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.Security.CORS.AllowedMethods, ", "))
		}

		if len(config.Security.CORS.AllowedHeaders) > 0 {
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.Security.CORS.AllowedHeaders, ", "))
		}

		if config.Security.CORS.MaxAge > 0 {
			w.Header().Set("Access-Control-Max-Age", string(rune(config.Security.CORS.MaxAge)))
		}

		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}
}
