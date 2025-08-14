package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

func generateRandomAPIKey() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "default-insecure-key"
	}
	return hex.EncodeToString(b)
}

// loadAPIKeysFromConfig loads API keys from the loaded configuration
func loadAPIKeysFromConfig(config *Config) {
	// Initialize the allowed API keys map
	allowedAPIKeys = make(map[string]struct{})

	// Add API keys from config
	for _, key := range config.Security.APIKeys {
		if key != "" && key != "your-secauto-api-key-here" {
			allowedAPIKeys[key] = struct{}{}
		}
	}

	// Also check for environment variable (for backward compatibility)
	if envKey := os.Getenv("SECAUTO_API_KEY"); envKey != "" {
		allowedAPIKeys[envKey] = struct{}{}
	}

	// If no API keys found, generate a random one
	if len(allowedAPIKeys) == 0 {
		apiKey := generateRandomAPIKey()
		allowedAPIKeys[apiKey] = struct{}{}
		logger.Warning("No API keys found in config or environment, generated random API key", map[string]interface{}{
			"component": "auth",
			"api_key":   apiKey,
		})
	} else {
		logger.Info("Loaded API keys from configuration", map[string]interface{}{
			"component": "auth",
			"key_count": len(allowedAPIKeys),
		})
	}

	// Set environment variables for Python integrations
	setEnvironmentVariablesForIntegrations(config)
}

// setEnvironmentVariablesForIntegrations sets environment variables that Python integrations can use
func setEnvironmentVariablesForIntegrations(config *Config) {
	// Set SECAUTO_API_KEY to the first valid API key
	for key := range allowedAPIKeys {
		if key != "" && key != "your-secauto-api-key-here" {
			os.Setenv("SECAUTO_API_KEY", key)
			logger.Info("Set SECAUTO_API_KEY environment variable for Python integrations", map[string]interface{}{
				"component": "integrations",
				"api_key":   key[:10] + "..." + key[len(key)-10:], // Show first and last 10 chars
			})
			break
		}
	}

	// Set SECAUTO_URL based on server configuration
	host := config.Server.Host
	if host == "" {
		host = "localhost"
	}
	port := config.Server.Port
	if port == 0 {
		port = 8000 // Use 8000 as default instead of 8080
	}

	secautoURL := fmt.Sprintf("http://%s:%d", host, port)
	os.Setenv("SECAUTO_URL", secautoURL)

	logger.Info("Set SECAUTO_URL environment variable for Python integrations", map[string]interface{}{
		"component": "integrations",
		"url":       secautoURL,
	})

	// Also write a config file that Python integrations can read
	writeIntegrationConfigFile(config, secautoURL)
}

// writeIntegrationConfigFile writes a JSON config file that Python integrations can read
func writeIntegrationConfigFile(config *Config, secautoURL string) {
	// Find the first valid API key
	var apiKey string
	for key := range allowedAPIKeys {
		if key != "" && key != "your-secauto-api-key-here" {
			apiKey = key
			break
		}
	}

	if apiKey == "" {
		logger.Warning("No valid API key found, skipping integration config file", map[string]interface{}{
			"component": "integrations",
		})
		return
	}

	// Create integration config
	integrationConfig := map[string]interface{}{
		"secauto_url":     secautoURL,
		"secauto_api_key": apiKey,
		"server_host":     config.Server.Host,
		"server_port":     config.Server.Port,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}

	// Convert to JSON
	configData, err := json.MarshalIndent(integrationConfig, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal integration config", map[string]interface{}{
			"component": "integrations",
			"error":     err.Error(),
		})
		return
	}

	// Write to file
	configPath := "data/integration_config.json"
	if err := os.MkdirAll("data", 0755); err != nil {
		logger.Error("Failed to create data directory", map[string]interface{}{
			"component": "integrations",
			"error":     err.Error(),
		})
		return
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		logger.Error("Failed to write integration config file", map[string]interface{}{
			"component": "integrations",
			"path":      configPath,
			"error":     err.Error(),
		})
		return
	}

	logger.Info("Wrote integration config file for Python integrations", map[string]interface{}{
		"component": "integrations",
		"path":      configPath,
		"url":       secautoURL,
	})
}

// apiKeyAuthMiddleware enforces API key authentication
func apiKeyAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow /health without auth
		if r.URL.Path == "/health" {
			next(w, r)
			return
		}

		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = r.URL.Query().Get("api_key")
		}
		if _, ok := allowedAPIKeys[key]; !ok {
			logger.Error("Unauthorized API access", map[string]interface{}{
				"component":   "auth",
				"remote_addr": r.RemoteAddr,
				"path":        r.URL.Path,
				"user_agent":  r.UserAgent(),
			})
			http.Error(w, "Unauthorized: missing or invalid API key", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// getClientIP extracts the real client IP
func getClientIP(r *http.Request) string {
	// Check for forwarded headers
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Fallback to remote address
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// loggingMiddleware adds structured logging to handlers
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := uuid.New().String()

		// Log request start
		logger.Info("HTTP request started", map[string]interface{}{
			"component":   "http",
			"request_id":  requestID,
			"remote_addr": r.RemoteAddr,
			"path":        r.URL.Path,
			"method":      r.Method,
			"user_agent":  r.UserAgent(),
		})

		// Create response writer wrapper to capture status code
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: 200}

		// Call next handler
		next(wrappedWriter, r)

		// Calculate duration
		duration := time.Since(start).Milliseconds()

		// Log request completion
		logger.Info("HTTP request completed", map[string]interface{}{
			"component":   "http",
			"request_id":  requestID,
			"remote_addr": r.RemoteAddr,
			"path":        r.URL.Path,
			"method":      r.Method,
			"status_code": wrappedWriter.statusCode,
			"duration_ms": float64(duration),
		})
	}
}
