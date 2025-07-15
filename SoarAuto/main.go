package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
	"time"

	"io"
	"mime/multipart"
	"path/filepath"
	"sort"
	"strings"
)

// Global logger instance
var logger *StructuredLogger
var globalLogMutex sync.Mutex // Global mutex for all logging operations

// APIKeyAuth holds allowed API keys
var allowedAPIKeys map[string]struct{}

func main() {
	// Define command line flags
	standalone := flag.Bool("s", false, "Run in standalone mode")
	contextFile := flag.String("c", "", "Context JSON file path")
	playbookFile := flag.String("p", "", "Playbook JSON file path")
	port := flag.String("port", "8080", "Server port (for server mode)")
	workers := flag.Int("workers", 5, "Number of worker threads for async jobs")
	logLevel := flag.String("log-level", "INFO", "Log level (DEBUG, INFO, WARNING, ERROR)")
	// logDest and logFile are no longer needed as variables

	// Parse flags
	flag.Parse()

	// Load configuration
	config, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Standalone mode: always log to logs/secauto_standalone.log
	if *standalone {
		// Use default rotation config for standalone mode
		standaloneRotation := &RotationConfig{
			MaxSizeMB:  10, // 10 MB
			MaxBackups: 3,  // Keep 3 old files
			MaxAgeDays: 7,  // Delete after 7 days
			Compress:   true,
		}
		logger = NewStructuredLogger(LogLevel(*logLevel), "file", "logs/secauto_standalone.log", standaloneRotation)
		if *playbookFile == "" {
			fmt.Println("Error: Playbook file (-p) is required for standalone mode")
			fmt.Println("Usage: ./secauto.exe -s -p <playbook.json> [-c <context.json>]")
			os.Exit(1)
		}
		runStandaloneWithFlags(*playbookFile, *contextFile)
		return
	}

	// Server mode: use config.yaml logging config
	level := LogLevel(config.Logging.Level)
	dest := config.Logging.Destination
	file := config.Logging.File
	rotation := &RotationConfig{
		MaxSizeMB:  config.Logging.Rotation.MaxSizeMB,
		MaxBackups: config.Logging.Rotation.MaxBackups,
		MaxAgeDays: config.Logging.Rotation.MaxAgeDays,
		Compress:   config.Logging.Rotation.Compress,
	}
	logger = NewStructuredLogger(level, dest, file, rotation)

	runServer(*port, *workers)
}

func runServer(port string, workerCount int) {
	// Load configuration
	config, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Load API keys from config
	loadAPIKeysFromConfig(config)

	// Configuration
	serverPort := getEnv("SECAUTO_PORT", port)

	// Create rule engine
	engine := NewRuleEngine(config)

	// Create webhook manager
	webhookManager := NewWebhookManager()

	// Create job manager
	jobManager, err := NewJobManager(workerCount, webhookManager, config)
	if err != nil {
		log.Fatalf("Failed to create job manager: %v", err)
	}

	// Create rate limiter
	rateLimiter := NewRateLimiter(config)

	// Create validator
	validator := NewValidator()

	// Create platform-aware plugin manager
	pluginManager, err := NewPlatformPluginManager(config)
	if err != nil {
		log.Fatalf("Failed to create platform plugin manager: %v", err)
	}

	// Set plugin manager on rule engine
	engine.SetPluginManager(pluginManager)

	// Create cluster manager if enabled
	var clusterManager *ClusterManager
	if config.Cluster.Enabled {
		clusterManager, err = NewClusterManager(&config.Cluster, &SecAutoServer{
			engine:         engine,
			port:           serverPort,
			jobManager:     jobManager,
			webhookManager: webhookManager,
			validator:      validator,
			pluginManager:  pluginManager,
		})
		if err != nil {
			log.Fatalf("Failed to create cluster manager: %v", err)
		}
	}

	// Create job scheduler if enabled
	var jobScheduler *JobScheduler
	if config.Scheduler.Enabled {
		jobScheduler, err = NewJobScheduler(&config.Scheduler, &SecAutoServer{
			engine:         engine,
			port:           serverPort,
			jobManager:     jobManager,
			webhookManager: webhookManager,
			validator:      validator,
			pluginManager:  pluginManager,
			clusterManager: clusterManager,
		})
		if err != nil {
			log.Fatalf("Failed to create job scheduler: %v", err)
		}
	}

	// Initialize Swagger UI handler
	swaggerHandler, err := NewSwaggerUIHandler()
	if err != nil {
		log.Fatalf("Failed to initialize Swagger UI handler: %v", err)
	}

	// Recover jobs that were running during crash
	jobManager.store.RecoverJobs(engine, webhookManager)

	// Create integration config manager
	integrationConfigManager, err := NewIntegrationConfigManager("data/integration_configs.enc", config.Security.IntegrationEncryptionKey)
	if err != nil {
		log.Fatalf("Failed to create integration config manager: %v", err)
	}

	// Create default configs if none exist
	if len(integrationConfigManager.ListConfigs()) == 0 {
		if err := integrationConfigManager.CreateDefaultConfigs(); err != nil {
			log.Fatalf("Failed to create default integration configs: %v", err)
		}
	}

	// Create server
	server := &SecAutoServer{
		engine:                   engine,
		port:                     serverPort,
		jobManager:               jobManager,
		webhookManager:           webhookManager,
		validator:                validator,
		pluginManager:            pluginManager,
		clusterManager:           clusterManager,
		jobScheduler:             jobScheduler,
		integrationConfigManager: integrationConfigManager,
	}

	// Set up routes with logging, validation, rate limiting, and auth middleware
	http.HandleFunc("/health", loggingMiddleware(server.healthHandler))
	http.HandleFunc("/playbook", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.playbookHandler)))))
	http.HandleFunc("/playbook/async", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.playbookAsyncHandler)))))
	http.HandleFunc("/jobs", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.jobsHandler)))))
	http.HandleFunc("/jobs/stats", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.jobStatsHandler)))))
	http.HandleFunc("/jobs/metrics", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.jobMetricsHandler)))))
	http.HandleFunc("/plugins", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.pluginsHandler)))))
	http.HandleFunc("/plugins/", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.pluginHandler)))))
	http.HandleFunc("/cluster", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.clusterHandler)))))
	http.HandleFunc("/cluster/jobs", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.clusterJobsHandler)))))
	http.HandleFunc("/cluster/jobs/", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.clusterJobHandler)))))
	http.HandleFunc("/schedules", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.schedulesHandler)))))
	http.HandleFunc("/schedules/", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.scheduleHandler)))))
	http.HandleFunc("/job/", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.jobHandler)))))
	http.HandleFunc("/context", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.contextHandler)))))
	http.HandleFunc("/webhooks", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.webhooksHandler)))))
	http.HandleFunc("/validate", loggingMiddleware(validationMiddleware(validator)(server.validateHandler)))
	http.HandleFunc("/automation", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.automationUploadHandler)))))
	http.HandleFunc("/playbook/upload", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.playbookUploadHandler)))))
	http.HandleFunc("/playbooks", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.playbookListHandler)))))
	http.HandleFunc("/automations", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.automationListHandler)))))
	http.HandleFunc("/automation/", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.automationDeleteHandler)))))
	http.HandleFunc("/playbook/", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.playbookDeleteHandler)))))
	http.HandleFunc("/plugin/", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.pluginUploadHandler)))))
	http.HandleFunc("/plugin/delete/", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.pluginDeleteHandler)))))

	// Integration configuration endpoints
	http.HandleFunc("/integrations", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.integrationsHandler)))))
	http.HandleFunc("/integrations/", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.integrationHandler)))))
	http.HandleFunc("/integrations/upload", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.integrationUploadHandler)))))
	http.HandleFunc("/integrations/delete/", loggingMiddleware(validationMiddleware(validator)(rateLimitMiddleware(rateLimiter)(apiKeyAuthMiddleware(server.integrationDeleteHandler)))))

	// Swagger UI documentation routes (no auth required)
	http.HandleFunc("/docs", swaggerHandler.ServeHTTP)
	http.HandleFunc("/docs/", swaggerHandler.ServeHTTP)
	http.HandleFunc("/api-docs", swaggerHandler.ServeHTTP)

	// Start server
	logger.Info("SecAuto Server starting", map[string]interface{}{
		"component": "server",
		"port":      serverPort,
		"workers":   workerCount,
		"features":  []string{"rate_limiting", "input_validation", "structured_logging", "webhook_notifications", "job_management", "redis_persistence"},
	})

	logger.Info("Available endpoints", map[string]interface{}{
		"component": "server",
		"endpoints": []map[string]string{
			{"method": "GET", "path": "/health", "description": "Health check"},
			{"method": "POST", "path": "/playbook", "description": "Execute playbook (synchronous)"},
			{"method": "POST", "path": "/playbook/async", "description": "Execute playbook (asynchronous)"},
			{"method": "GET", "path": "/jobs", "description": "List all jobs"},
			{"method": "GET", "path": "/jobs/stats", "description": "Job statistics"},
			{"method": "GET", "path": "/jobs/metrics", "description": "Database performance metrics"},
			{"method": "GET", "path": "/plugins", "description": "List all plugins"},
			{"method": "GET", "path": "/plugins/{name}", "description": "Get plugin information"},
			{"method": "POST", "path": "/plugins/{name}", "description": "Execute plugin"},
			{"method": "POST", "path": "/automation", "description": "Upload automation script"},
			{"method": "POST", "path": "/playbook/upload", "description": "Upload playbook file"},
			{"method": "GET", "path": "/playbooks", "description": "List all playbooks"},
			{"method": "GET", "path": "/automations", "description": "List all automations"},
			{"method": "DELETE", "path": "/automation/{name}", "description": "Delete an automation"},
			{"method": "GET", "path": "/cluster", "description": "Get cluster information"},
			{"method": "POST", "path": "/cluster/jobs", "description": "Submit job to distributed queue"},
			{"method": "GET", "path": "/cluster/jobs/{id}", "description": "Get distributed job status"},
			{"method": "GET", "path": "/job/{id}", "description": "Get job status and results"},
			{"method": "DELETE", "path": "/job/{id}", "description": "Cancel/delete job"},
			{"method": "GET", "path": "/context", "description": "Get current context"},
			{"method": "POST", "path": "/webhooks", "description": "Configure webhooks"},
			{"method": "POST", "path": "/validate", "description": "Validate playbook/context"},
			{"method": "GET", "path": "/docs", "description": "Interactive API documentation (Swagger UI)"},
			{"method": "GET", "path": "/api-docs", "description": "OpenAPI specification"},
			{"method": "DELETE", "path": "/automation/{name}", "description": "Delete an automation"},
			{"method": "DELETE", "path": "/playbook/{name}", "description": "Delete a playbook"},
			{"method": "POST", "path": "/plugin/{type}", "description": "Upload plugin file"},
			{"method": "DELETE", "path": "/plugin/{type}/{name}", "description": "Delete a plugin"},
			{"method": "GET", "path": "/integrations", "description": "List all integrations"},
			{"method": "GET", "path": "/integrations/{name}", "description": "Get integration information by name"},
			{"method": "POST", "path": "/integrations", "description": "Create a new integration"},
			{"method": "PUT", "path": "/integrations/{name}", "description": "Update an existing integration by name"},
			{"method": "DELETE", "path": "/integrations/{name}", "description": "Delete an integration by name"},
			{"method": "POST", "path": "/integrations/upload", "description": "Upload integration Python file"},
			{"method": "DELETE", "path": "/integrations/delete/{name}", "description": "Delete integration Python file"},
		},
	})

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := http.ListenAndServe(":"+serverPort, nil); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-stop
	logger.Info("Shutting down server gracefully...", map[string]interface{}{
		"component": "server",
	})

	// Close plugin manager
	if err := server.pluginManager.Close(); err != nil {
		logger.Error("Failed to close plugin manager", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
	}

	// Close cluster manager
	if server.clusterManager != nil {
		if err := server.clusterManager.Close(); err != nil {
			logger.Error("Failed to close cluster manager", map[string]interface{}{
				"component": "server",
				"error":     err.Error(),
			})
		}
	}

	jobManager.Cleanup()
	logger.Info("Job manager cleanup completed", map[string]interface{}{
		"component": "server",
	})
}

// healthHandler handles health check requests
func (s *SecAutoServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// playbookHandler handles synchronous playbook execution requests
func (s *SecAutoServer) playbookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req PlaybookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate request
	validationResult := s.validator.ValidatePlaybookRequest(&req)
	if !validationResult.Valid {
		response := ValidationResponse{
			Success:   false,
			Valid:     false,
			Errors:    validationResult.Errors,
			Message:   "Validation failed",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Sanitize inputs
	if req.PlaybookName != "" {
		req.PlaybookName = s.validator.SanitizePath(req.PlaybookName)
	}

	// Set context if provided
	if req.Context != nil {
		s.engine.SetContext(req.Context)
	}

	// Execute playbook
	var results []interface{}
	var err error

	if req.Playbook != nil {
		// Execute inline playbook
		results, err = s.engine.EvaluatePlaybook(req.Playbook)
	} else if req.PlaybookName != "" {
		// Load and execute playbook from file
		playbookPath := s.engine.getPlaybookPath(req.PlaybookName)
		playbook, err := s.engine.LoadPlaybookFromFile(playbookPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load playbook: %v", err), http.StatusBadRequest)
			return
		}
		results, _ = s.engine.EvaluatePlaybook(playbook)
	} else {
		http.Error(w, "Either playbook or playbook_name must be provided", http.StatusBadRequest)
		return
	}

	response := PlaybookResponse{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if err != nil {
		response.Success = false
		response.Error = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response.Success = true
		response.Results = results
		response.Context = s.engine.GetContext()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// playbookAsyncHandler handles asynchronous playbook execution requests
func (s *SecAutoServer) playbookAsyncHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req PlaybookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate request
	validationResult := s.validator.ValidatePlaybookRequest(&req)
	if !validationResult.Valid {
		response := ValidationResponse{
			Success:   false,
			Valid:     false,
			Errors:    validationResult.Errors,
			Message:   "Validation failed",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Sanitize inputs
	if req.PlaybookName != "" {
		req.PlaybookName = s.validator.SanitizePath(req.PlaybookName)
	}

	// Submit job for asynchronous execution
	var jobID string

	if req.Playbook != nil {
		// Submit inline playbook
		jobID = s.jobManager.SubmitJob(req.Playbook, req.Context)
	} else if req.PlaybookName != "" {
		// Load playbook from file and submit
		playbookPath := s.engine.getPlaybookPath(req.PlaybookName)
		playbook, err := s.engine.LoadPlaybookFromFile(playbookPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load playbook: %v", err), http.StatusBadRequest)
			return
		}
		jobID = s.jobManager.SubmitJob(playbook, req.Context)
	} else {
		http.Error(w, "Either playbook or playbook_name must be provided", http.StatusBadRequest)
		return
	}

	response := JobResponse{
		Success:   true,
		JobID:     jobID,
		Status:    "pending",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// jobsHandler handles job listing requests
func (s *SecAutoServer) jobsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	limit := 50 // default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Get jobs from job manager
	jobs := s.jobManager.ListJobs(status, limit)

	response := JobListResponse{
		Success:   true,
		Jobs:      jobs,
		Total:     len(jobs),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// jobStatsHandler handles job statistics requests
func (s *SecAutoServer) jobStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.jobManager.GetStats()

	response := JobStatsResponse{
		Success:     true,
		TotalJobs:   stats.TotalJobs,
		Completed:   stats.Completed,
		Failed:      stats.Failed,
		Running:     stats.Running,
		Pending:     stats.Pending,
		AvgDuration: stats.AvgDuration,
		RecentJobs:  stats.RecentJobs,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// jobMetricsHandler handles database metrics requests
func (s *SecAutoServer) jobMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.jobManager.store.GetDatabaseMetrics()
	response := map[string]interface{}{
		"success":   true,
		"metrics":   metrics,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// pluginsHandler handles plugin listing and management
func (s *SecAutoServer) pluginsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pluginInfo := s.pluginManager.GetPluginInfo()
	response := map[string]interface{}{
		"success":   true,
		"plugins":   pluginInfo,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// pluginHandler handles individual plugin operations
func (s *SecAutoServer) pluginHandler(w http.ResponseWriter, r *http.Request) {
	// Extract plugin name from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid plugin path", http.StatusBadRequest)
		return
	}
	pluginName := pathParts[1]

	switch r.Method {
	case http.MethodGet:
		// Get plugin info
		pluginInfo := s.pluginManager.GetPluginInfo()
		if info, exists := pluginInfo[pluginName]; exists {
			response := map[string]interface{}{
				"success":   true,
				"plugin":    info,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			http.Error(w, "Plugin not found", http.StatusNotFound)
		}

	case http.MethodPost:
		// Execute plugin
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		result, err := s.pluginManager.ExecutePlugin(pluginName, request)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"error":     err.Error(),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"result":    result,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// jobHandler handles job status and cancellation requests
func (s *SecAutoServer) jobHandler(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from URL path
	jobID := r.URL.Path[len("/job/"):]
	if jobID == "" {
		http.Error(w, "Job ID required", http.StatusBadRequest)
		return
	}

	// Validate job ID
	validationResult := s.validator.ValidateJobID(jobID)
	if !validationResult.Valid {
		response := ValidationResponse{
			Success:   false,
			Valid:     false,
			Errors:    validationResult.Errors,
			Message:   "Invalid job ID",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get job status
		job, exists := s.jobManager.GetJob(jobID)
		if !exists {
			http.Error(w, "Job not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)

	case http.MethodDelete:
		// Cancel job
		success, message := s.jobManager.CancelJob(jobID)

		response := CancelJobResponse{
			Success:   success,
			JobID:     jobID,
			Status:    "cancelled",
			Message:   message,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		if !success {
			w.WriteHeader(http.StatusBadRequest)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// contextHandler handles context retrieval requests
func (s *SecAutoServer) contextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	context := s.engine.GetContext()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"context":   context,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// webhooksHandler handles webhook configuration requests
func (s *SecAutoServer) webhooksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse webhook configuration
	var webhookConfig WebhookConfig
	if err := json.NewDecoder(r.Body).Decode(&webhookConfig); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate webhook configuration
	validationResult := s.validator.ValidateWebhookConfig(&webhookConfig)
	if !validationResult.Valid {
		response := ValidationResponse{
			Success:   false,
			Valid:     false,
			Errors:    validationResult.Errors,
			Message:   "Webhook validation failed",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Add webhook to manager
	s.webhookManager.AddWebhook(webhookConfig)

	response := struct {
		Success   bool   `json:"success"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
	}{
		Success:   true,
		Message:   "Webhook configuration added",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// validateHandler handles validation requests
func (s *SecAutoServer) validateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse validation request
	var req PlaybookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate playbook request
	validationResult := s.validator.ValidatePlaybookRequest(&req)

	response := ValidationResponse{
		Success:   validationResult.Valid,
		Valid:     validationResult.Valid,
		Errors:    validationResult.Errors,
		Message:   validationResult.Message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// automationUploadHandler handles automation script uploads
func (s *SecAutoServer) automationUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		logger.Error("Failed to parse multipart form", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("automation")
	if err != nil {
		logger.Error("Failed to get uploaded file", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
		http.Error(w, "No automation file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file
	validationResult := s.validateAutomationFile(header, file)
	if !validationResult.Valid {
		response := ValidationResponse{
			Success:   false,
			Valid:     false,
			Errors:    validationResult.Errors,
			Message:   "Automation file validation failed",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Save the automation file
	automationName, err := s.saveAutomationFile(file, header)
	if err != nil {
		logger.Error("Failed to save automation file", map[string]interface{}{
			"component": "server",
			"filename":  header.Filename,
			"error":     err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to save automation: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := AutomationUploadResponse{
		Success:        true,
		Message:        "Automation uploaded successfully",
		AutomationName: automationName,
		Filename:       header.Filename,
		Size:           header.Size,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.Info("Automation uploaded successfully", map[string]interface{}{
		"component": "server",
		"filename":  header.Filename,
		"size":      header.Size,
		"name":      automationName,
	})
}

// playbookUploadHandler handles playbook file uploads
func (s *SecAutoServer) playbookUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 5MB for playbooks)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		logger.Error("Failed to parse multipart form", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("playbook")
	if err != nil {
		logger.Error("Failed to get uploaded file", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
		http.Error(w, "No playbook file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file
	validationResult := s.validatePlaybookFile(header, file)
	if !validationResult.Valid {
		response := ValidationResponse{
			Success:   false,
			Valid:     false,
			Errors:    validationResult.Errors,
			Message:   "Playbook file validation failed",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Save the playbook file
	playbookName, err := s.savePlaybookFile(file, header)
	if err != nil {
		logger.Error("Failed to save playbook file", map[string]interface{}{
			"component": "server",
			"filename":  header.Filename,
			"error":     err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to save playbook: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := PlaybookUploadResponse{
		Success:      true,
		Message:      "Playbook uploaded successfully",
		PlaybookName: playbookName,
		Filename:     header.Filename,
		Size:         header.Size,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.Info("Playbook uploaded successfully", map[string]interface{}{
		"component": "server",
		"filename":  header.Filename,
		"size":      header.Size,
		"name":      playbookName,
	})
}

// playbookListHandler handles listing all available playbooks
func (s *SecAutoServer) playbookListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get list of playbooks
	playbooks, err := s.getPlaybookList()
	if err != nil {
		logger.Error("Failed to get playbook list", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to get playbook list: %v", err), http.StatusInternalServerError)
		return
	}

	// Return playbook list
	response := PlaybookListResponse{
		Success:   true,
		Message:   "Playbooks retrieved successfully",
		Playbooks: playbooks,
		Count:     len(playbooks),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.Info("Playbook list retrieved", map[string]interface{}{
		"component": "server",
		"count":     len(playbooks),
	})
}

// getPlaybookList scans the playbooks directory and returns playbook information
func (s *SecAutoServer) getPlaybookList() ([]PlaybookInfo, error) {
	playbooksDir := "../playbooks"

	// Check if directory exists
	if _, err := os.Stat(playbooksDir); os.IsNotExist(err) {
		// Return empty list if directory doesn't exist
		return []PlaybookInfo{}, nil
	}

	// Read directory contents
	files, err := os.ReadDir(playbooksDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read playbooks directory: %v", err)
	}

	var playbooks []PlaybookInfo

	for _, file := range files {
		// Skip directories and non-JSON files
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".json") {
			continue
		}

		// Get file info
		filePath := filepath.Join(playbooksDir, file.Name())
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			logger.Warning("Failed to get file info", map[string]interface{}{
				"component": "server",
				"filename":  file.Name(),
				"error":     err.Error(),
			})
			continue
		}

		// Read and validate playbook content
		content, err := os.ReadFile(filePath)
		if err != nil {
			logger.Warning("Failed to read playbook file", map[string]interface{}{
				"component": "server",
				"filename":  file.Name(),
				"error":     err.Error(),
			})
			continue
		}

		// Validate JSON structure
		var playbookData []interface{}
		if err := json.Unmarshal(content, &playbookData); err != nil {
			logger.Warning("Invalid JSON in playbook file", map[string]interface{}{
				"component": "server",
				"filename":  file.Name(),
				"error":     err.Error(),
			})
			continue
		}

		// Extract playbook metadata
		playbookName := strings.TrimSuffix(file.Name(), ".json")
		ruleCount := len(playbookData)

		// Count different operation types
		operationCounts := s.countPlaybookOperations(playbookData)

		playbook := PlaybookInfo{
			Name:       playbookName,
			Filename:   file.Name(),
			Size:       fileInfo.Size(),
			RuleCount:  ruleCount,
			Operations: operationCounts,
			ModifiedAt: fileInfo.ModTime().UTC().Format(time.RFC3339),
			IsValid:    true,
		}

		playbooks = append(playbooks, playbook)
	}

	// Sort playbooks by name
	sort.Slice(playbooks, func(i, j int) bool {
		return playbooks[i].Name < playbooks[j].Name
	})

	return playbooks, nil
}

// countPlaybookOperations counts the different types of operations in a playbook
func (s *SecAutoServer) countPlaybookOperations(playbookData []interface{}) map[string]int {
	operations := make(map[string]int)

	for _, rule := range playbookData {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}

		for op := range ruleMap {
			switch op {
			case "run":
				operations["run"]++
			case "if":
				operations["if"]++
			case "play":
				operations["play"]++
			case "plugin":
				operations["plugin"]++
			}
		}
	}

	return operations
}

// validateAutomationFile validates the uploaded automation file
func (s *SecAutoServer) validateAutomationFile(header *multipart.FileHeader, file multipart.File) ValidationResult {
	var errors []ValidationError

	// Check file size (max 1MB)
	if header.Size > 1<<20 {
		errors = append(errors, ValidationError{
			Field:   "file_size",
			Message: "File size exceeds 1MB limit",
		})
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".py" {
		errors = append(errors, ValidationError{
			Field:   "file_extension",
			Message: "Only Python (.py) files are supported",
		})
	}

	// Check filename for security
	if !s.validator.IsValidFilename(header.Filename) {
		errors = append(errors, ValidationError{
			Field:   "filename",
			Message: "Invalid filename",
			Value:   header.Filename,
		})
	}

	// Read and validate file content
	content, err := io.ReadAll(file)
	if err != nil {
		errors = append(errors, ValidationError{
			Field:   "file_content",
			Message: "Failed to read file content",
		})
		return ValidationResult{Valid: false, Errors: errors}
	}

	// Check for dangerous content
	if s.containsDangerousContent(content) {
		errors = append(errors, ValidationError{
			Field:   "content",
			Message: "File contains potentially dangerous content",
		})
	}

	// Check for required Python structure
	if !s.isValidPythonScript(content) {
		errors = append(errors, ValidationError{
			Field:   "content",
			Message: "File does not appear to be a valid Python script",
		})
	}

	// Reset file pointer for later use
	file.Seek(0, 0)

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// containsDangerousContent checks for potentially dangerous content
func (s *SecAutoServer) containsDangerousContent(content []byte) bool {
	dangerousPatterns := []string{
		"import os",
		"import subprocess",
		"import sys",
		"exec(",
		"eval(",
		"__import__",
		"open(",
		"file(",
		"input(",
		"raw_input(",
		"compile(",
		"reload(",
		"delattr(",
		"setattr(",
		"__builtins__",
		"__globals__",
		"__dict__",
		"__code__",
		"__class__",
		"__bases__",
		"__subclasses__",
		"__mro__",
		"__metaclass__",
		"__new__",
		"__init__",
		"__call__",
		"__getattr__",
		"__setattr__",
		"__delattr__",
		"__getattribute__",
		"__getitem__",
		"__setitem__",
		"__delitem__",
		"__iter__",
		"__next__",
		"__len__",
		"__contains__",
		"__add__",
		"__sub__",
		"__mul__",
		"__div__",
		"__mod__",
		"__pow__",
		"__lshift__",
		"__rshift__",
		"__and__",
		"__or__",
		"__xor__",
		"__invert__",
		"__neg__",
		"__pos__",
		"__abs__",
		"__round__",
		"__floor__",
		"__ceil__",
		"__trunc__",
		"__index__",
		"__int__",
		"__float__",
		"__complex__",
		"__str__",
		"__repr__",
		"__bytes__",
		"__format__",
		"__hash__",
		"__bool__",
		"__lt__",
		"__le__",
		"__eq__",
		"__ne__",
		"__gt__",
		"__ge__",
		"__cmp__",
		"__rcmp__",
		"__radd__",
		"__rsub__",
		"__rmul__",
		"__rdiv__",
		"__rmod__",
		"__rpow__",
		"__rlshift__",
		"__rrshift__",
		"__rand__",
		"__ror__",
		"__rxor__",
		"__iadd__",
		"__isub__",
		"__imul__",
		"__idiv__",
		"__imod__",
		"__ipow__",
		"__ilshift__",
		"__irshift__",
		"__iand__",
		"__ior__",
		"__ixor__",
		"__enter__",
		"__exit__",
		"__await__",
		"__aiter__",
		"__anext__",
		"__aenter__",
		"__aexit__",
	}

	contentStr := strings.ToLower(string(content))
	for _, pattern := range dangerousPatterns {
		if strings.Contains(contentStr, pattern) {
			return true
		}
	}

	return false
}

// isValidPythonScript checks if the content is a valid Python script
func (s *SecAutoServer) isValidPythonScript(content []byte) bool {
	contentStr := string(content)

	// Check for basic Python syntax indicators
	hasPythonIndicators := strings.Contains(contentStr, "def ") ||
		strings.Contains(contentStr, "import ") ||
		strings.Contains(contentStr, "from ") ||
		strings.Contains(contentStr, "class ") ||
		strings.Contains(contentStr, "if __name__") ||
		strings.Contains(contentStr, "print(") ||
		strings.Contains(contentStr, "return ")

	// Check for proper indentation (basic check)
	lines := strings.Split(contentStr, "\n")
	hasProperIndentation := false
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && (strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")) {
			hasProperIndentation = true
			break
		}
	}

	return hasPythonIndicators || hasProperIndentation
}

// saveAutomationFile saves the uploaded automation file
func (s *SecAutoServer) saveAutomationFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	// Create automations directory if it doesn't exist
	automationsDir := "../automations"
	if err := os.MkdirAll(automationsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create automations directory: %v", err)
	}

	// Generate safe filename
	filename := s.validator.SanitizeFilename(header.Filename)
	if filename == "" {
		filename = "automation_" + time.Now().Format("20060102_150405") + ".py"
	}

	// Ensure .py extension
	if !strings.HasSuffix(filename, ".py") {
		filename += ".py"
	}

	// Create full path
	filepath := filepath.Join(automationsDir, filename)

	// Create the file
	dst, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	// Copy the uploaded file to the destination
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	// Return the automation name (without extension)
	automationName := strings.TrimSuffix(filename, ".py")
	return automationName, nil
}

// validatePlaybookFile validates the uploaded playbook file
func (s *SecAutoServer) validatePlaybookFile(header *multipart.FileHeader, file multipart.File) ValidationResult {
	var errors []ValidationError

	// Check file size (max 1MB for playbooks)
	if header.Size > 1<<20 {
		errors = append(errors, ValidationError{
			Field:   "file_size",
			Message: "File size exceeds 1MB limit",
		})
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".json" {
		errors = append(errors, ValidationError{
			Field:   "file_extension",
			Message: "Only JSON (.json) files are supported",
		})
	}

	// Check filename for security
	if !s.validator.IsValidFilename(header.Filename) {
		errors = append(errors, ValidationError{
			Field:   "filename",
			Message: "Invalid filename",
			Value:   header.Filename,
		})
	}

	// Read and validate file content
	content, err := io.ReadAll(file)
	if err != nil {
		errors = append(errors, ValidationError{
			Field:   "file_content",
			Message: "Failed to read file content",
		})
		return ValidationResult{Valid: false, Errors: errors}
	}

	// Validate JSON structure
	if !s.isValidPlaybookJSON(content) {
		errors = append(errors, ValidationError{
			Field:   "content",
			Message: "File does not appear to be a valid playbook JSON",
		})
	}

	// Validate playbook structure
	if err := s.validatePlaybookStructure(content); err != nil {
		errors = append(errors, ValidationError{
			Field:   "content",
			Message: fmt.Sprintf("Invalid playbook structure: %v", err),
		})
	}

	// Reset file pointer for later use
	file.Seek(0, 0)

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// isValidPlaybookJSON checks if the content is valid JSON
func (s *SecAutoServer) isValidPlaybookJSON(content []byte) bool {
	var playbook []interface{}
	return json.Unmarshal(content, &playbook) == nil
}

// validatePlaybookStructure validates the playbook structure
func (s *SecAutoServer) validatePlaybookStructure(content []byte) error {
	var playbook []interface{}
	if err := json.Unmarshal(content, &playbook); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}

	// Validate it's an array
	if len(playbook) == 0 {
		return fmt.Errorf("playbook cannot be empty")
	}

	// Validate each rule
	for i, rule := range playbook {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			return fmt.Errorf("rule %d must be an object", i+1)
		}

		// Check for valid operations
		hasValidOp := false
		for op := range ruleMap {
			switch op {
			case "run", "if", "play", "plugin":
				hasValidOp = true
			}
		}

		if !hasValidOp {
			return fmt.Errorf("rule %d must contain a valid operation (run, if, play, plugin)", i+1)
		}
	}

	return nil
}

// savePlaybookFile saves the uploaded playbook file
func (s *SecAutoServer) savePlaybookFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	// Create playbooks directory if it doesn't exist
	playbooksDir := "../playbooks"
	if err := os.MkdirAll(playbooksDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create playbooks directory: %v", err)
	}

	// Generate safe filename
	filename := s.validator.SanitizeFilename(header.Filename)
	if filename == "" {
		filename = "playbook_" + time.Now().Format("20060102_150405") + ".json"
	}

	// Ensure .json extension
	if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}

	// Create full path
	filepath := filepath.Join(playbooksDir, filename)

	// Create the file
	dst, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	// Copy the uploaded file to the destination
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	// Return the playbook name (without extension)
	playbookName := strings.TrimSuffix(filename, ".json")
	return playbookName, nil
}

// executeJob executes a job in the worker pool
func (jm *JobManager) executeJob(jobID string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Job execution panicked", map[string]interface{}{
				"component": "job_manager",
				"job_id":    jobID,
				"panic":     fmt.Sprintf("%v", r),
				"stack":     string(debug.Stack()),
			})
			jm.updateJobStatus(jobID, "failed", nil, fmt.Sprintf("Job execution panicked: %v", r))
		}
	}()

	logger.Info("Starting job execution", map[string]interface{}{
		"component": "job_manager",
		"job_id":    jobID,
	})

	// Load the job from the store
	job, exists := jm.store.LoadJob(jobID)
	if !exists {
		logger.Error("Job not found for execution", map[string]interface{}{
			"component": "job_manager",
			"job_id":    jobID,
		})
		return
	}

	// Log before loading config
	logger.Info("Before LoadConfig", map[string]interface{}{"job_id": jobID})
	config, err := LoadConfig("config.yaml")
	if err != nil {
		jm.updateJobStatus(jobID, "failed", nil, err.Error())
		return
	}
	logger.Info("After LoadConfig", map[string]interface{}{"job_id": jobID})

	engine := NewRuleEngine(config)

	// Create platform-aware plugin manager for job execution
	jobPluginManager, err := NewPlatformPluginManager(config)
	if err != nil {
		logger.Error("Failed to create plugin manager for job", map[string]interface{}{
			"component": "job_manager",
			"job_id":    jobID,
			"error":     err.Error(),
		})
		jm.updateJobStatus(jobID, "failed", nil, fmt.Sprintf("Failed to create plugin manager: %v", err))
		return
	}

	// Set plugin manager on rule engine
	engine.SetPluginManager(jobPluginManager)

	// Log before setting context
	logger.Info("Before SetContext", map[string]interface{}{
		"job_id":       jobID,
		"context":      job.Context,
		"context_type": fmt.Sprintf("%T", job.Context),
		"context_keys": len(job.Context),
	})
	engine.SetContext(job.Context)
	logger.Info("After SetContext", map[string]interface{}{"job_id": jobID})

	// Add a simple test log to see if we reach this point
	logger.Info("After SetContext - test log", map[string]interface{}{"job_id": jobID})

	// Log before playbook evaluation - be more careful with the playbook logging
	logger.Info("Before EvaluatePlaybook", map[string]interface{}{
		"job_id":        jobID,
		"playbook_len":  len(job.Playbook),
		"playbook_type": fmt.Sprintf("%T", job.Playbook),
	})

	// Validate playbook before evaluation
	if job.Playbook == nil {
		logger.Error("Playbook is nil", map[string]interface{}{
			"component": "job_manager",
			"job_id":    jobID,
		})
		jm.updateJobStatus(jobID, "failed", nil, "Playbook is nil")
		return
	}

	logger.Info("Playbook validation passed", map[string]interface{}{
		"component":    "job_manager",
		"job_id":       jobID,
		"playbook_len": len(job.Playbook),
	})

	// Add panic recovery around EvaluatePlaybook call
	defer func() {
		if r := recover(); r != nil {
			logger.Error("EvaluatePlaybook panicked", map[string]interface{}{
				"component": "job_manager",
				"job_id":    jobID,
				"panic":     fmt.Sprintf("%v", r),
			})
			panic(r) // Re-panic to be caught by the outer defer
		}
	}()

	results, err := engine.EvaluatePlaybook(job.Playbook)
	logger.Info("After EvaluatePlaybook", map[string]interface{}{"job_id": jobID, "results": results, "err": err})

	if err != nil {
		logger.Info("Playbook evaluation failed, updating job status to failed", map[string]interface{}{
			"component": "job_manager",
			"job_id":    jobID,
			"error":     err.Error(),
		})
		jm.updateJobStatus(jobID, "failed", nil, err.Error())
	} else {
		// Get the final updated context from the engine
		finalContext := engine.GetContext()
		logger.Info("Playbook evaluation succeeded, updating job status to completed", map[string]interface{}{
			"component":    "job_manager",
			"job_id":       jobID,
			"results_len":  len(results),
			"context_keys": len(finalContext),
		})

		// Update job with results and final context
		jm.updateJobStatusWithContext(jobID, "completed", results, "", finalContext)
	}
}

// updateJobStatus updates a job's status and results
func (jm *JobManager) updateJobStatus(jobID, status string, results []interface{}, errorMsg string) {
	jm.updateJobStatusWithContext(jobID, status, results, errorMsg, nil)
}

// updateJobStatusWithContext updates a job's status, results, and context
func (jm *JobManager) updateJobStatusWithContext(jobID, status string, results []interface{}, errorMsg string, finalContext map[string]interface{}) {
	logger.Info("Updating job status", map[string]interface{}{
		"component":    "job_manager",
		"job_id":       jobID,
		"status":       status,
		"results_len":  len(results),
		"error_msg":    errorMsg,
		"context_keys": len(finalContext),
	})

	// Add panic recovery to prevent cascading failures
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Job status update panicked", map[string]interface{}{
				"component": "job_manager",
				"job_id":    jobID,
				"status":    status,
				"panic":     fmt.Sprintf("%v", r),
			})
		}
	}()

	// Update job status
	if err := jm.store.UpdateJobStatus(jobID, status); err != nil {
		logger.Error("Failed to update job status", map[string]interface{}{
			"component": "job_manager",
			"job_id":    jobID,
			"status":    status,
			"error":     err.Error(),
		})
		return
	}

	logger.Info("Job status updated successfully", map[string]interface{}{
		"component": "job_manager",
		"job_id":    jobID,
		"status":    status,
	})

	// Update job results and context
	if err := jm.store.UpdateJobResults(jobID, results, errorMsg); err != nil {
		logger.Error("Failed to update job results", map[string]interface{}{
			"component": "job_manager",
			"job_id":    jobID,
			"error":     err.Error(),
		})
		return
	}

	// Update job context if provided
	if finalContext != nil {
		if err := jm.store.UpdateJobContext(jobID, finalContext); err != nil {
			logger.Error("Failed to update job context", map[string]interface{}{
				"component": "job_manager",
				"job_id":    jobID,
				"error":     err.Error(),
			})
			return
		}
	}

	// Get updated job for webhook
	job, exists := jm.store.LoadJob(jobID)
	if !exists {
		logger.Error("Job not found for webhook notification", map[string]interface{}{
			"component": "job_manager",
			"job_id":    jobID,
		})
		return
	}

	// Calculate duration
	var duration float64
	if job.StartedAt != nil && job.CompletedAt != nil {
		duration = job.CompletedAt.Sub(*job.StartedAt).Seconds()
	}

	// Send webhook notification for job completion/failure
	if jm.webhookManager != nil {
		eventType := "job_completed"
		if status == "failed" {
			eventType = "job_failed"
		}

		jm.webhookManager.SendWebhook(WebhookEvent{
			Event:     eventType,
			JobID:     jobID,
			Status:    status,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Playbook:  job.Playbook,
			Context:   job.Context,
			Results:   results,
			Error:     errorMsg,
			Duration:  duration,
		})
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// clusterHandler handles cluster information requests
func (s *SecAutoServer) clusterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.clusterManager == nil {
		response := map[string]interface{}{
			"success":   false,
			"error":     "Cluster mode not enabled",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	clusterInfo := s.clusterManager.GetClusterInfo()
	response := map[string]interface{}{
		"success":   true,
		"cluster":   clusterInfo,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// clusterJobsHandler handles distributed job submission
func (s *SecAutoServer) clusterJobsHandler(w http.ResponseWriter, r *http.Request) {
	if s.clusterManager == nil {
		http.Error(w, "Cluster mode not enabled", http.StatusServiceUnavailable)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req PlaybookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate request
	validationResult := s.validator.ValidatePlaybookRequest(&req)
	if !validationResult.Valid {
		response := ValidationResponse{
			Success:   false,
			Valid:     false,
			Errors:    validationResult.Errors,
			Message:   "Validation failed",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Submit job to distributed queue
	var jobID string
	var err error

	if req.Playbook != nil {
		jobID, err = s.clusterManager.SubmitJob(req.Playbook, req.Context)
	} else if req.PlaybookName != "" {
		// Load playbook from file and submit
		playbookPath := s.engine.getPlaybookPath(req.PlaybookName)
		playbook, err := s.engine.LoadPlaybookFromFile(playbookPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load playbook: %v", err), http.StatusBadRequest)
			return
		}
		jobID, err = s.clusterManager.SubmitJob(playbook, req.Context)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"error":     err.Error(),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	} else {
		http.Error(w, "Either playbook or playbook_name must be provided", http.StatusBadRequest)
		return
	}

	if err != nil {
		response := map[string]interface{}{
			"success":   false,
			"error":     err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"job_id":    jobID,
		"status":    "submitted",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// clusterJobHandler handles individual distributed job operations
func (s *SecAutoServer) clusterJobHandler(w http.ResponseWriter, r *http.Request) {
	if s.clusterManager == nil {
		http.Error(w, "Cluster mode not enabled", http.StatusServiceUnavailable)
		return
	}

	// Extract job ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid job path", http.StatusBadRequest)
		return
	}
	jobID := pathParts[2]

	switch r.Method {
	case http.MethodGet:
		// Get job status
		job, err := s.clusterManager.GetJob(jobID)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"error":     err.Error(),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"job":       job,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// schedulesHandler handles schedule listing and creation
func (s *SecAutoServer) schedulesHandler(w http.ResponseWriter, r *http.Request) {
	if s.jobScheduler == nil {
		http.Error(w, "Job scheduler not enabled", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// List schedules
		status := r.URL.Query().Get("status")
		limit := 50
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}

		schedules := s.jobScheduler.ListSchedules(ScheduleStatus(status), limit)
		response := map[string]interface{}{
			"success":   true,
			"schedules": schedules,
			"total":     len(schedules),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case http.MethodPost:
		// Create new schedule
		var schedule JobSchedule
		if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := s.jobScheduler.CreateSchedule(&schedule); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"error":     err.Error(),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"schedule":  schedule,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// scheduleHandler handles individual schedule operations
func (s *SecAutoServer) scheduleHandler(w http.ResponseWriter, r *http.Request) {
	if s.jobScheduler == nil {
		http.Error(w, "Job scheduler not enabled", http.StatusServiceUnavailable)
		return
	}

	// Extract schedule ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid schedule path", http.StatusBadRequest)
		return
	}
	scheduleID := pathParts[1]

	switch r.Method {
	case http.MethodGet:
		// Get schedule
		schedule, exists := s.jobScheduler.GetSchedule(scheduleID)
		if !exists {
			http.Error(w, "Schedule not found", http.StatusNotFound)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"schedule":  schedule,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case http.MethodPut:
		// Update schedule
		var schedule JobSchedule
		if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		schedule.ID = scheduleID

		if err := s.jobScheduler.UpdateSchedule(&schedule); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"error":     err.Error(),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"schedule":  schedule,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case http.MethodDelete:
		// Delete schedule
		if err := s.jobScheduler.DeleteSchedule(scheduleID); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"error":     err.Error(),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Schedule deleted",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// automationListHandler handles listing all available automations
func (s *SecAutoServer) automationListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get list of automations
	automations, err := s.getAutomationList()
	if err != nil {
		logger.Error("Failed to get automation list", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to get automation list: %v", err), http.StatusInternalServerError)
		return
	}

	// Return automation list
	response := AutomationListResponse{
		Success:     true,
		Message:     "Automations retrieved successfully",
		Automations: automations,
		Count:       len(automations),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.Info("Automation list retrieved", map[string]interface{}{
		"component": "server",
		"count":     len(automations),
	})
}

// getAutomationList scans the automations directory and returns automation information
func (s *SecAutoServer) getAutomationList() ([]AutomationInfo, error) {
	automationsDir := "../automations"

	// Check if directory exists
	if _, err := os.Stat(automationsDir); os.IsNotExist(err) {
		// Return empty list if directory doesn't exist
		return []AutomationInfo{}, nil
	}

	// Read directory contents
	files, err := os.ReadDir(automationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read automations directory: %v", err)
	}

	var automations []AutomationInfo

	for _, file := range files {
		// Skip directories
		if file.IsDir() {
			continue
		}

		// Get file info
		filePath := filepath.Join(automationsDir, file.Name())
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			logger.Warning("Failed to get file info", map[string]interface{}{
				"component": "server",
				"filename":  file.Name(),
				"error":     err.Error(),
			})
			continue
		}

		// Determine file type and language
		fileType := s.getAutomationFileType(file.Name())
		language := s.getAutomationLanguage(file.Name())

		// Read file content for analysis
		content, err := os.ReadFile(filePath)
		if err != nil {
			logger.Warning("Failed to read automation file", map[string]interface{}{
				"component": "server",
				"filename":  file.Name(),
				"error":     err.Error(),
			})
			continue
		}

		// Analyze automation content
		analysis := s.analyzeAutomationContent(content, language)

		automation := AutomationInfo{
			Name:          strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())),
			Filename:      file.Name(),
			Size:          fileInfo.Size(),
			FileType:      fileType,
			Language:      language,
			LineCount:     analysis.LineCount,
			FunctionCount: analysis.FunctionCount,
			ImportCount:   analysis.ImportCount,
			ModifiedAt:    fileInfo.ModTime().UTC().Format(time.RFC3339),
			IsValid:       analysis.IsValid,
		}

		automations = append(automations, automation)
	}

	// Sort automations by name
	sort.Slice(automations, func(i, j int) bool {
		return automations[i].Name < automations[j].Name
	})

	return automations, nil
}

// getAutomationFileType determines the file type based on extension
func (s *SecAutoServer) getAutomationFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".sh":
		return "shell"
	case ".ps1":
		return "powershell"
	case ".bat":
		return "batch"
	case ".exe":
		return "executable"
	default:
		return "unknown"
	}
}

// getAutomationLanguage determines the programming language
func (s *SecAutoServer) getAutomationLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".py":
		return "Python"
	case ".js":
		return "JavaScript"
	case ".sh":
		return "Bash"
	case ".ps1":
		return "PowerShell"
	case ".bat":
		return "Batch"
	case ".exe":
		return "Executable"
	default:
		return "Unknown"
	}
}

// AutomationAnalysis represents analysis results of automation content
type AutomationAnalysis struct {
	LineCount     int  `json:"line_count"`
	FunctionCount int  `json:"function_count"`
	ImportCount   int  `json:"import_count"`
	IsValid       bool `json:"is_valid"`
}

// analyzeAutomationContent analyzes the content of an automation file
func (s *SecAutoServer) analyzeAutomationContent(content []byte, language string) AutomationAnalysis {
	lines := strings.Split(string(content), "\n")
	lineCount := len(lines)

	functionCount := 0
	importCount := 0
	isValid := true

	// Basic content validation
	if len(content) == 0 {
		isValid = false
	}

	// Language-specific analysis
	switch language {
	case "Python":
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)

			// Count imports
			if strings.HasPrefix(trimmedLine, "import ") || strings.HasPrefix(trimmedLine, "from ") {
				importCount++
			}

			// Count function definitions
			if strings.HasPrefix(trimmedLine, "def ") {
				functionCount++
			}
		}
	case "JavaScript":
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)

			// Count imports
			if strings.HasPrefix(trimmedLine, "import ") || strings.HasPrefix(trimmedLine, "require(") {
				importCount++
			}

			// Count function definitions
			if strings.Contains(trimmedLine, "function ") || strings.Contains(trimmedLine, "=>") {
				functionCount++
			}
		}
	case "PowerShell":
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)

			// Count function definitions
			if strings.HasPrefix(trimmedLine, "function ") {
				functionCount++
			}
		}
	}

	return AutomationAnalysis{
		LineCount:     lineCount,
		FunctionCount: functionCount,
		ImportCount:   importCount,
		IsValid:       isValid,
	}
}

// automationDeleteHandler handles deleting an automation
func (s *SecAutoServer) automationDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract automation name from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid automation name", http.StatusBadRequest)
		return
	}
	automationName := pathParts[2]

	// Check for dependencies in playbooks
	dependencies, err := s.checkAutomationDependencies(automationName)
	if err != nil {
		logger.Error("Failed to check automation dependencies", map[string]interface{}{
			"component":  "server",
			"automation": automationName,
			"error":      err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to check dependencies: %v", err), http.StatusInternalServerError)
		return
	}

	// If dependencies exist, return error with dependency information
	if len(dependencies) > 0 {
		response := AutomationDeleteResponse{
			Success:        false,
			Message:        "Cannot delete automation - it is used by playbooks",
			AutomationName: automationName,
			Dependencies:   dependencies,
			Timestamp:      time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusConflict)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Delete the automation file
	err = s.deleteAutomationFile(automationName)
	if err != nil {
		logger.Error("Failed to delete automation file", map[string]interface{}{
			"component":  "server",
			"automation": automationName,
			"error":      err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to delete automation: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := AutomationDeleteResponse{
		Success:        true,
		Message:        "Automation deleted successfully",
		AutomationName: automationName,
		Dependencies:   []string{},
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.Info("Automation deleted successfully", map[string]interface{}{
		"component":  "server",
		"automation": automationName,
	})
}

// checkAutomationDependencies checks if an automation is used by any playbooks
func (s *SecAutoServer) checkAutomationDependencies(automationName string) ([]string, error) {
	playbooksDir := "../playbooks"
	var dependencies []string

	// Check if playbooks directory exists
	if _, err := os.Stat(playbooksDir); os.IsNotExist(err) {
		return dependencies, nil
	}

	// Read all playbook files
	files, err := os.ReadDir(playbooksDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read playbooks directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".json") {
			continue
		}

		// Read playbook content
		filePath := filepath.Join(playbooksDir, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			logger.Warning("Failed to read playbook file", map[string]interface{}{
				"component": "server",
				"filename":  file.Name(),
				"error":     err.Error(),
			})
			continue
		}

		// Parse playbook JSON
		var playbookData []interface{}
		if err := json.Unmarshal(content, &playbookData); err != nil {
			logger.Warning("Invalid JSON in playbook file", map[string]interface{}{
				"component": "server",
				"filename":  file.Name(),
				"error":     err.Error(),
			})
			continue
		}

		// Check if automation is used in this playbook
		if s.isAutomationUsedInPlaybook(playbookData, automationName) {
			playbookName := strings.TrimSuffix(file.Name(), ".json")
			dependencies = append(dependencies, playbookName)
		}
	}

	return dependencies, nil
}

// isAutomationUsedInPlaybook checks if an automation is used in a playbook
func (s *SecAutoServer) isAutomationUsedInPlaybook(playbookData []interface{}, automationName string) bool {
	for _, rule := range playbookData {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}

		// Check "run" operations
		if runOp, exists := ruleMap["run"]; exists {
			if runStr, ok := runOp.(string); ok && runStr == automationName {
				return true
			}
		}

		// Check "if" operations recursively
		if ifOp, exists := ruleMap["if"]; exists {
			if ifMap, ok := ifOp.(map[string]interface{}); ok {
				if s.isAutomationUsedInIfBlock(ifMap, automationName) {
					return true
				}
			}
		}
	}

	return false
}

// isAutomationUsedInIfBlock recursively checks if an automation is used in an if block
func (s *SecAutoServer) isAutomationUsedInIfBlock(ifBlock map[string]interface{}, automationName string) bool {
	// Check "true" branch
	if trueOp, exists := ifBlock["true"]; exists {
		if trueMap, ok := trueOp.(map[string]interface{}); ok {
			if runOp, exists := trueMap["run"]; exists {
				if runStr, ok := runOp.(string); ok && runStr == automationName {
					return true
				}
			}
		}
	}

	// Check "false" branch
	if falseOp, exists := ifBlock["false"]; exists {
		if falseMap, ok := falseOp.(map[string]interface{}); ok {
			if runOp, exists := falseMap["run"]; exists {
				if runStr, ok := runOp.(string); ok && runStr == automationName {
					return true
				}
			}
		}
	}

	return false
}

// deleteAutomationFile deletes an automation file
func (s *SecAutoServer) deleteAutomationFile(automationName string) error {
	automationsDir := "../automations"

	// Try different file extensions
	extensions := []string{".py", ".js", ".sh", ".ps1", ".bat", ".exe"}

	for _, ext := range extensions {
		filename := automationName + ext
		filePath := filepath.Join(automationsDir, filename)

		// Check if file exists
		if _, err := os.Stat(filePath); err == nil {
			// File exists, delete it
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to delete file %s: %v", filename, err)
			}
			return nil
		}
	}

	return fmt.Errorf("automation '%s' not found", automationName)
}

// playbookDeleteHandler handles deleting a playbook
func (s *SecAutoServer) playbookDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract playbook name from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid playbook name", http.StatusBadRequest)
		return
	}
	playbookName := pathParts[2]

	// Delete the playbook file
	err := s.deletePlaybookFile(playbookName)
	if err != nil {
		logger.Error("Failed to delete playbook file", map[string]interface{}{
			"component": "server",
			"playbook":  playbookName,
			"error":     err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to delete playbook: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := PlaybookDeleteResponse{
		Success:      true,
		Message:      "Playbook deleted successfully",
		PlaybookName: playbookName,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.Info("Playbook deleted successfully", map[string]interface{}{
		"component": "server",
		"playbook":  playbookName,
	})
}

// deletePlaybookFile deletes a playbook file
func (s *SecAutoServer) deletePlaybookFile(playbookName string) error {
	playbooksDir := "../playbooks"

	// Try different file extensions
	extensions := []string{".json"}

	for _, ext := range extensions {
		filename := playbookName + ext
		filePath := filepath.Join(playbooksDir, filename)

		// Check if file exists
		if _, err := os.Stat(filePath); err == nil {
			// File exists, delete it
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to delete file %s: %v", filename, err)
			}
			return nil
		}
	}

	return fmt.Errorf("playbook '%s' not found", playbookName)
}

// pluginUploadHandler handles plugin file uploads by type
func (s *SecAutoServer) pluginUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract plugin type from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid plugin type", http.StatusBadRequest)
		return
	}
	pluginType := pathParts[3]

	// Validate plugin type
	if !s.isValidPluginType(pluginType) {
		http.Error(w, "Invalid plugin type. Supported types: linux, windows, python, go", http.StatusBadRequest)
		return
	}

	// Parse multipart form (max 10MB for plugins)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		logger.Error("Failed to parse multipart form", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("plugin")
	if err != nil {
		logger.Error("Failed to get uploaded file", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
		http.Error(w, "No plugin file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file
	validationResult := s.validatePluginFile(header, file, pluginType)
	if !validationResult.Valid {
		response := ValidationResponse{
			Success:   false,
			Valid:     false,
			Errors:    validationResult.Errors,
			Message:   "Plugin file validation failed",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Save the plugin file
	pluginName, err := s.savePluginFile(file, header, pluginType)
	if err != nil {
		logger.Error("Failed to save plugin file", map[string]interface{}{
			"component": "server",
			"filename":  header.Filename,
			"type":      pluginType,
			"error":     err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to save plugin: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := PluginUploadResponse{
		Success:    true,
		Message:    "Plugin uploaded successfully",
		PluginName: pluginName,
		PluginType: pluginType,
		Filename:   header.Filename,
		Size:       header.Size,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.Info("Plugin uploaded successfully", map[string]interface{}{
		"component": "server",
		"filename":  header.Filename,
		"type":      pluginType,
		"size":      header.Size,
		"name":      pluginName,
	})
}

// isValidPluginType checks if the plugin type is valid
func (s *SecAutoServer) isValidPluginType(pluginType string) bool {
	validTypes := []string{"linux", "windows", "python", "go"}
	for _, validType := range validTypes {
		if pluginType == validType {
			return true
		}
	}
	return false
}

// validatePluginFile validates the uploaded plugin file
func (s *SecAutoServer) validatePluginFile(header *multipart.FileHeader, file multipart.File, pluginType string) ValidationResult {
	var errors []ValidationError

	// Check file size (max 10MB for plugins)
	if header.Size > 10<<20 {
		errors = append(errors, ValidationError{
			Field:   "file_size",
			Message: "File size exceeds 10MB limit",
		})
	}

	// Check file extension based on plugin type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !s.isValidPluginExtension(ext, pluginType) {
		errors = append(errors, ValidationError{
			Field:   "file_extension",
			Message: fmt.Sprintf("Invalid file extension for %s plugin type", pluginType),
			Value:   ext,
		})
	}

	// Check filename for security
	if !s.validator.IsValidFilename(header.Filename) {
		errors = append(errors, ValidationError{
			Field:   "filename",
			Message: "Invalid filename",
			Value:   header.Filename,
		})
	}

	// Read and validate file content
	content, err := io.ReadAll(file)
	if err != nil {
		errors = append(errors, ValidationError{
			Field:   "file_content",
			Message: "Failed to read file content",
		})
		return ValidationResult{Valid: false, Errors: errors}
	}

	// Validate plugin content based on type
	if !s.isValidPluginContent(content, pluginType) {
		errors = append(errors, ValidationError{
			Field:   "content",
			Message: fmt.Sprintf("File does not appear to be a valid %s plugin", pluginType),
		})
	}

	// Reset file pointer for later use
	file.Seek(0, 0)

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// isValidPluginExtension checks if the file extension is valid for the plugin type
func (s *SecAutoServer) isValidPluginExtension(ext, pluginType string) bool {
	switch pluginType {
	case "linux":
		return ext == "" || ext == ".sh" || ext == ".py" || ext == ".go"
	case "windows":
		return ext == ".exe" || ext == ".ps1" || ext == ".bat" || ext == ".py" || ext == ".go"
	case "python":
		return ext == ".py"
	case "go":
		return ext == ".go"
	default:
		return false
	}
}

// isValidPluginContent checks if the content is valid for the plugin type
func (s *SecAutoServer) isValidPluginContent(content []byte, pluginType string) bool {
	// Basic content validation
	if len(content) == 0 {
		return false
	}

	switch pluginType {
	case "python":
		// Check for Python syntax indicators
		contentStr := string(content)
		return strings.Contains(contentStr, "def ") || strings.Contains(contentStr, "import ") || strings.Contains(contentStr, "class ")
	case "go":
		// Check for Go syntax indicators
		contentStr := string(content)
		return strings.Contains(contentStr, "package ") || strings.Contains(contentStr, "func ") || strings.Contains(contentStr, "import ")
	case "linux", "windows":
		// For binary files or scripts, just check they're not empty
		return len(content) > 0
	default:
		return false
	}
}

// savePluginFile saves the uploaded plugin file
func (s *SecAutoServer) savePluginFile(file multipart.File, header *multipart.FileHeader, pluginType string) (string, error) {
	// Create plugins directory structure
	pluginsDir := "../plugins"
	platformDir := filepath.Join(pluginsDir, pluginType+"_plugins")

	if err := os.MkdirAll(platformDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create plugin directory: %v", err)
	}

	// Generate safe filename
	filename := s.validator.SanitizeFilename(header.Filename)
	if filename == "" {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		filename = "plugin_" + time.Now().Format("20060102_150405") + ext
	}

	// Create full path
	fullPath := filepath.Join(platformDir, filename)

	// Create the file
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	// Copy the uploaded file to the destination
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	// Return the plugin name (without extension)
	pluginName := strings.TrimSuffix(filename, filepath.Ext(filename))
	return pluginName, nil
}

// pluginDeleteHandler handles deleting a plugin
func (s *SecAutoServer) pluginDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract plugin type and name from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid plugin type or name", http.StatusBadRequest)
		return
	}
	pluginType := pathParts[2]
	pluginName := pathParts[3]

	// Validate plugin type
	if !s.isValidPluginType(pluginType) {
		http.Error(w, "Invalid plugin type. Supported types: linux, windows, python, go", http.StatusBadRequest)
		return
	}

	// Delete the plugin file
	err := s.deletePluginFile(pluginName, pluginType)
	if err != nil {
		logger.Error("Failed to delete plugin file", map[string]interface{}{
			"component": "server",
			"plugin":    pluginName,
			"type":      pluginType,
			"error":     err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to delete plugin: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := PluginDeleteResponse{
		Success:    true,
		Message:    "Plugin deleted successfully",
		PluginName: pluginName,
		PluginType: pluginType,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.Info("Plugin deleted successfully", map[string]interface{}{
		"component": "server",
		"plugin":    pluginName,
		"type":      pluginType,
	})
}

// deletePluginFile deletes a plugin file
func (s *SecAutoServer) deletePluginFile(pluginName, pluginType string) error {
	pluginsDir := "../plugins"
	platformDir := filepath.Join(pluginsDir, pluginType+"_plugins")

	// Try different file extensions based on plugin type
	var extensions []string
	switch pluginType {
	case "linux":
		extensions = []string{".sh", ".py", ".go", ""}
	case "windows":
		extensions = []string{".exe", ".ps1", ".bat", ".py", ".go"}
	case "python":
		extensions = []string{".py"}
	case "go":
		extensions = []string{".go"}
	default:
		return fmt.Errorf("unsupported plugin type: %s", pluginType)
	}

	for _, ext := range extensions {
		filename := pluginName + ext
		filePath := filepath.Join(platformDir, filename)

		// Check if file exists
		if _, err := os.Stat(filePath); err == nil {
			// File exists, delete it
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to delete file %s: %v", filename, err)
			}
			return nil
		}
	}

	return fmt.Errorf("plugin '%s' of type '%s' not found", pluginName, pluginType)
}

// integrationsHandler handles integration configuration management
func (s *SecAutoServer) integrationsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List all integrations
		configs := s.integrationConfigManager.ListConfigs()
		response := IntegrationResponse{
			Success:      true,
			Message:      "Integrations retrieved successfully",
			Integrations: make([]*IntegrationConfig, 0, len(configs)),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}

		// Include integration names in the response
		for name, config := range configs {
			// Create a copy with the name included
			configCopy := *config
			configCopy.Name = name // Ensure the name is set correctly
			response.Integrations = append(response.Integrations, &configCopy)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case http.MethodPost:
		// Create new integration
		var config IntegrationConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate configuration
		if err := s.integrationConfigManager.ValidateConfig(&config); err != nil {
			response := IntegrationResponse{
				Success:   false,
				Message:   fmt.Sprintf("Validation failed: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Use the provided name or generate from type
		integrationName := config.Name
		if integrationName == "" {
			integrationName = config.Type
		}
		if integrationName == "" {
			http.Error(w, "Integration name or type is required", http.StatusBadRequest)
			return
		}

		// Set configuration
		if err := s.integrationConfigManager.SetConfig(integrationName, &config); err != nil {
			response := IntegrationResponse{
				Success:   false,
				Message:   fmt.Sprintf("Failed to create integration: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response := IntegrationResponse{
			Success:     true,
			Message:     "Integration created successfully",
			Integration: &config,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// integrationHandler handles individual integration operations
func (s *SecAutoServer) integrationHandler(w http.ResponseWriter, r *http.Request) {
	// Extract integration name from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid integration path", http.StatusBadRequest)
		return
	}
	integrationName := pathParts[1]

	switch r.Method {
	case http.MethodGet:
		// Get integration configuration
		config, exists := s.integrationConfigManager.GetConfig(integrationName)
		if !exists {
			http.Error(w, "Integration not found", http.StatusNotFound)
			return
		}

		// Ensure the name is set correctly in the response
		configCopy := *config
		configCopy.Name = integrationName

		response := IntegrationResponse{
			Success:     true,
			Message:     "Integration retrieved successfully",
			Integration: &configCopy,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case http.MethodPut:
		// Update integration configuration
		var config IntegrationConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate configuration
		if err := s.integrationConfigManager.ValidateConfig(&config); err != nil {
			response := IntegrationResponse{
				Success:   false,
				Message:   fmt.Sprintf("Validation failed: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Update configuration
		if err := s.integrationConfigManager.SetConfig(integrationName, &config); err != nil {
			response := IntegrationResponse{
				Success:   false,
				Message:   fmt.Sprintf("Failed to update integration: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response := IntegrationResponse{
			Success:     true,
			Message:     "Integration updated successfully",
			Integration: &config,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	case http.MethodDelete:
		// Delete integration configuration
		if err := s.integrationConfigManager.DeleteConfig(integrationName); err != nil {
			response := IntegrationResponse{
				Success:   false,
				Message:   fmt.Sprintf("Failed to delete integration: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response := IntegrationResponse{
			Success:   true,
			Message:   "Integration deleted successfully",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// integrationUploadHandler handles integration file uploads
func (s *SecAutoServer) integrationUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 5MB for integration files)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		logger.Error("Failed to parse multipart form", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("integration")
	if err != nil {
		logger.Error("Failed to get uploaded file", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
		http.Error(w, "No integration file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file
	validationResult := s.validateIntegrationFile(header, file)
	if !validationResult.Valid {
		response := ValidationResponse{
			Success:   false,
			Valid:     false,
			Errors:    validationResult.Errors,
			Message:   "Integration file validation failed",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Save the integration file
	integrationName, err := s.saveIntegrationFile(file, header)
	if err != nil {
		logger.Error("Failed to save integration file", map[string]interface{}{
			"component": "server",
			"filename":  header.Filename,
			"error":     err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to save integration: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := IntegrationUploadResponse{
		Success:         true,
		Message:         "Integration uploaded successfully",
		IntegrationName: integrationName,
		Filename:        header.Filename,
		Size:            header.Size,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.Info("Integration uploaded successfully", map[string]interface{}{
		"component": "server",
		"filename":  header.Filename,
		"size":      header.Size,
		"name":      integrationName,
	})
}

// validateIntegrationFile validates the uploaded integration file
func (s *SecAutoServer) validateIntegrationFile(header *multipart.FileHeader, file multipart.File) ValidationResult {
	var errors []ValidationError

	// Check file size (max 1MB for integration files)
	if header.Size > 1<<20 {
		errors = append(errors, ValidationError{
			Field:   "file_size",
			Message: "File size exceeds 1MB limit",
		})
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".py" {
		errors = append(errors, ValidationError{
			Field:   "file_extension",
			Message: "Only Python (.py) files are supported for integrations",
		})
	}

	// Check filename for security
	if !s.validator.IsValidFilename(header.Filename) {
		errors = append(errors, ValidationError{
			Field:   "filename",
			Message: "Invalid filename",
			Value:   header.Filename,
		})
	}

	// Read and validate file content
	content, err := io.ReadAll(file)
	if err != nil {
		errors = append(errors, ValidationError{
			Field:   "file_content",
			Message: "Failed to read file content",
		})
		return ValidationResult{Valid: false, Errors: errors}
	}

	// Check for dangerous content
	if s.containsDangerousContent(content) {
		errors = append(errors, ValidationError{
			Field:   "content",
			Message: "File contains potentially dangerous content",
		})
	}

	// Check for required integration structure
	if !s.isValidIntegrationModule(content) {
		errors = append(errors, ValidationError{
			Field:   "content",
			Message: "File does not appear to be a valid integration module",
		})
	}

	// Reset file pointer for later use
	file.Seek(0, 0)

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// isValidIntegrationModule checks if the content is a valid integration module
func (s *SecAutoServer) isValidIntegrationModule(content []byte) bool {
	contentStr := string(content)

	// Check for basic Python module indicators
	hasPythonIndicators := strings.Contains(contentStr, "def ") ||
		strings.Contains(contentStr, "import ") ||
		strings.Contains(contentStr, "from ") ||
		strings.Contains(contentStr, "class ") ||
		strings.Contains(contentStr, "print(") ||
		strings.Contains(contentStr, "return ")

	// Check for integration-specific indicators
	hasIntegrationIndicators := strings.Contains(contentStr, "def ") ||
		strings.Contains(contentStr, "class ") ||
		strings.Contains(contentStr, "import requests") ||
		strings.Contains(contentStr, "import json") ||
		strings.Contains(contentStr, "def ") ||
		strings.Contains(contentStr, "return ")

	// Check for proper indentation (basic check)
	lines := strings.Split(contentStr, "\n")
	hasProperIndentation := false
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && (strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")) {
			hasProperIndentation = true
			break
		}
	}

	return (hasPythonIndicators || hasProperIndentation) && hasIntegrationIndicators
}

// saveIntegrationFile saves the uploaded integration file
func (s *SecAutoServer) saveIntegrationFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	// Create integrations directory if it doesn't exist
	integrationsDir := "../integrations"
	if err := os.MkdirAll(integrationsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create integrations directory: %v", err)
	}

	// Generate safe filename
	filename := s.validator.SanitizeFilename(header.Filename)
	if filename == "" {
		filename = "integration_" + time.Now().Format("20060102_150405") + ".py"
	}

	// Ensure .py extension
	if !strings.HasSuffix(filename, ".py") {
		filename += ".py"
	}

	// Create full path
	filepath := filepath.Join(integrationsDir, filename)

	// Create the file
	dst, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	// Copy the uploaded file to the destination
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	// Return the integration name (without extension)
	integrationName := strings.TrimSuffix(filename, ".py")
	return integrationName, nil
}

// integrationDeleteHandler handles deleting integration Python files
func (s *SecAutoServer) integrationDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract integration name from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid integration name", http.StatusBadRequest)
		return
	}
	integrationName := pathParts[3]

	// Check for dependencies in automations
	dependencies, err := s.checkIntegrationDependencies(integrationName)
	if err != nil {
		logger.Error("Failed to check integration dependencies", map[string]interface{}{
			"component":   "server",
			"integration": integrationName,
			"error":       err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to check dependencies: %v", err), http.StatusInternalServerError)
		return
	}

	// If dependencies exist, return error with dependency information
	if len(dependencies) > 0 {
		response := IntegrationDeleteResponse{
			Success:         false,
			Message:         "Cannot delete integration - it is used by automations",
			IntegrationName: integrationName,
			Dependencies:    dependencies,
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusConflict)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Delete the integration file
	err = s.deleteIntegrationFile(integrationName)
	if err != nil {
		logger.Error("Failed to delete integration file", map[string]interface{}{
			"component":   "server",
			"integration": integrationName,
			"error":       err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to delete integration: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := IntegrationDeleteResponse{
		Success:         true,
		Message:         "Integration deleted successfully",
		IntegrationName: integrationName,
		Dependencies:    []string{},
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.Info("Integration deleted successfully", map[string]interface{}{
		"component":   "server",
		"integration": integrationName,
	})
}

// checkIntegrationDependencies checks if an integration is used by any automations
func (s *SecAutoServer) checkIntegrationDependencies(integrationName string) ([]string, error) {
	automationsDir := "../automations"
	var dependencies []string

	// Check if automations directory exists
	if _, err := os.Stat(automationsDir); os.IsNotExist(err) {
		return dependencies, nil
	}

	// Read all automation files
	files, err := os.ReadDir(automationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read automations directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".py") {
			continue
		}

		// Read automation content
		filePath := filepath.Join(automationsDir, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			logger.Warning("Failed to read automation file", map[string]interface{}{
				"component": "server",
				"filename":  file.Name(),
				"error":     err.Error(),
			})
			continue
		}

		// Check if integration is imported in this automation
		if s.isIntegrationUsedInAutomation(content, integrationName) {
			automationName := strings.TrimSuffix(file.Name(), ".py")
			dependencies = append(dependencies, automationName)
		}
	}

	return dependencies, nil
}

// isIntegrationUsedInAutomation checks if an integration is used in an automation
func (s *SecAutoServer) isIntegrationUsedInAutomation(content []byte, integrationName string) bool {
	contentStr := string(content)

	// Check for import statements
	importPatterns := []string{
		fmt.Sprintf("import %s", integrationName),
		fmt.Sprintf("from %s", integrationName),
		fmt.Sprintf("from integrations.%s", integrationName),
		fmt.Sprintf("import integrations.%s", integrationName),
	}

	for _, pattern := range importPatterns {
		if strings.Contains(contentStr, pattern) {
			return true
		}
	}

	// Check for dynamic imports or usage
	if strings.Contains(contentStr, integrationName) {
		// Additional check to avoid false positives
		lines := strings.Split(contentStr, "\n")
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if strings.Contains(trimmedLine, integrationName) &&
				(strings.HasPrefix(trimmedLine, "import ") ||
					strings.HasPrefix(trimmedLine, "from ") ||
					strings.Contains(trimmedLine, "importlib.import_module")) {
				return true
			}
		}
	}

	return false
}

// deleteIntegrationFile deletes an integration file
func (s *SecAutoServer) deleteIntegrationFile(integrationName string) error {
	integrationsDir := "../integrations"

	// Try different file extensions
	extensions := []string{".py"}

	for _, ext := range extensions {
		filename := integrationName + ext
		filePath := filepath.Join(integrationsDir, filename)

		// Check if file exists
		if _, err := os.Stat(filePath); err == nil {
			// File exists, delete it
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to delete file %s: %v", filename, err)
			}
			return nil
		}
	}

	return fmt.Errorf("integration '%s' not found", integrationName)
}
