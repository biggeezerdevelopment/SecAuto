package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"SoarAuto/pkg/config"
	"SoarAuto/pkg/logger"
	"SoarAuto/pkg/types"
	"SoarAuto/pkg/validator"
)

// SecAutoServer represents the modular SOAR automation server
type SecAutoServer struct {
	config    *config.Config
	logger    types.Logger
	validator *validator.Validator
}

// NewSecAutoServer creates a new server instance
func NewSecAutoServer() (*SecAutoServer, error) {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %v", err)
	}

	// Create logger
	lgr := logger.NewStructuredLoggerWithConfig(
		logger.LogLevel(cfg.Logging.Level),
		cfg.Logging.Destination,
		cfg.Logging.File,
		&cfg.Logging.Rotation,
		cfg.Logging.ComponentLevels,
		&cfg.Logging.Performance,
	)

	// Create validator
	val := validator.NewValidator()

	return &SecAutoServer{
		config:    cfg,
		logger:    lgr,
		validator: val,
	}, nil
}

// healthHandler handles health check requests
func (s *SecAutoServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "modular-1.0.0",
		"modules": map[string]string{
			"config":    "loaded",
			"logger":    "active",
			"validator": "initialized",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Health check accessed", map[string]interface{}{
		"component":   "server",
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.UserAgent(),
	})
}

// playbookHandler handles playbook execution requests
func (s *SecAutoServer) playbookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.PlaybookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate the request
	validation := s.validator.ValidatePlaybookRequest(&req)
	if !validation.Valid {
		response := types.ValidationResponse{
			Success:   false,
			Valid:     false,
			Errors:    validation.Errors,
			Message:   "Validation failed",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// For this modular demo, just return a success response
	response := types.PlaybookResponse{
		Success:   true,
		Result:    "Playbook validation successful - execution would happen here",
		Message:   "Request validated successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Playbook request processed", map[string]interface{}{
		"component":      "server",
		"playbook_name":  req.PlaybookName,
		"has_context":    req.Context != nil,
		"validation":     "passed",
	})
}

// Run starts the modular server
func (s *SecAutoServer) Run() error {
	// Setup routes
	http.HandleFunc("/health", s.healthHandler)
	http.HandleFunc("/playbook", s.playbookHandler)

	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
		Handler: nil,
	}

	s.logger.Info("Starting modular SecAuto server", map[string]interface{}{
		"component": "server",
		"host":      s.config.Server.Host,
		"port":      s.config.Server.Port,
		"version":   "modular-1.0.0",
	})

	return server.ListenAndServe()
}

func main() {
	server, err := NewSecAutoServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	fmt.Println("🚀 SecAuto Modular Server Starting...")
	fmt.Printf("📁 Modules loaded: config, logger, validator, types\n")
	fmt.Printf("🌐 Server will start on %s:%d\n", server.config.Server.Host, server.config.Server.Port)
	fmt.Printf("🏥 Health endpoint: http://%s:%d/health\n", server.config.Server.Host, server.config.Server.Port)
	fmt.Printf("📋 Playbook endpoint: http://%s:%d/playbook\n", server.config.Server.Host, server.config.Server.Port)

	if err := server.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}