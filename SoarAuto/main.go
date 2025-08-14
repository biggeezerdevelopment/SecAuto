package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"SoarAuto/pkg/automations"
	"SoarAuto/pkg/cluster"
	"SoarAuto/pkg/config"
	"SoarAuto/pkg/integrations"
	"SoarAuto/pkg/jobs"
	"SoarAuto/pkg/logger"
	"SoarAuto/pkg/redis"
	"SoarAuto/pkg/rules"
	"SoarAuto/pkg/schedules"
	"SoarAuto/pkg/swagger"
	"SoarAuto/pkg/types"
	"SoarAuto/pkg/validator"
)

// SecAutoServer represents the modular SOAR automation server
type SecAutoServer struct {
	config                    *config.Config
	logger                    types.Logger
	validator                 *validator.Validator
	engine                    *rules.Engine
	redis                     *redis.RedisClient
	swagger                   *swagger.SwaggerUIHandler
	jobManager               *jobs.JobManager
	integrationManager       *integrations.IntegrationManager
	clientIntegrationManager *integrations.ClientIntegrationManager
	automationManager        *automations.AutomationManager
	clusterManager           *cluster.ClusterManager
	scheduleManager          *schedules.ScheduleManager
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

	// Create rules engine
	ruleEngine := rules.NewEngine(cfg)

	// Create Redis client
	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %v", err)
	}

	// Create Swagger handler
	swaggerHandler, err := swagger.NewSwaggerUIHandler(strconv.Itoa(cfg.Server.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to create Swagger handler: %v", err)
	}

	// Create job store and manager
	jobStore := jobs.NewJobStore(redisClient)
	jobManager := jobs.NewJobManager(jobStore)

	// Create integration manager
	integrationManager := integrations.NewIntegrationManager(
		filepath.Join("data", "integrations", "configs"),
		filepath.Join("data", "integrations", "scripts"),
	)

	// Create client integration manager
	clientIntegrationManager := integrations.NewClientIntegrationManager(
		integrationManager,
		filepath.Join("data", "clients"),
	)

	// Create automation manager
	automationManager := automations.NewAutomationManager(
		filepath.Join("data", "automations", "scripts"),
		filepath.Join("data", "automations", "metadata"),
	)

	// Create cluster manager
	clusterManager := cluster.NewClusterManager("node-modular-1")
	clusterManager.StartClusterServices()

	// Create server instance for schedule manager (implements JobExecutor interface)
	server := &SecAutoServer{
		config:                    cfg,
		logger:                    lgr,
		validator:                 val,
		engine:                    ruleEngine,
		redis:                     redisClient,
		swagger:                   swaggerHandler,
		jobManager:               jobManager,
		integrationManager:       integrationManager,
		clientIntegrationManager: clientIntegrationManager,
		automationManager:        automationManager,
		clusterManager:           clusterManager,
	}

	// Create schedule manager (needs server for job execution)
	scheduleManager := schedules.NewScheduleManager(
		filepath.Join("data", "schedules"),
		server, // server implements JobExecutor interface
	)
	server.scheduleManager = scheduleManager

	return server, nil
}

// ExecutePlaybook implements the JobExecutor interface for scheduled jobs
func (s *SecAutoServer) ExecutePlaybook(playbook interface{}, context map[string]interface{}) (interface{}, error) {
	// Set context in rules engine
	if context != nil {
		s.engine.SetContext(context)
	} else {
		s.engine.SetContext(map[string]interface{}{})
	}

	// Execute based on playbook type
	if playbookArray, ok := playbook.([]interface{}); ok {
		return s.engine.EvaluatePlaybook(playbookArray)
	} else {
		// Single rule execution
		return s.engine.EvaluateRule(playbook)
	}
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
			"config":      "loaded",
			"logger":      "active", 
			"validator":   "initialized",
			"rules_engine": "active",
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

	// Execute the playbook
	startTime := time.Now()
	
	// Set context in rules engine
	if req.Context != nil {
		s.engine.SetContext(req.Context)
	} else {
		s.engine.SetContext(map[string]interface{}{})
	}

	var result interface{}
	var err error

	// Execute based on what was provided
	if req.Playbook != nil {
		// Direct playbook execution
		if playbookArray, ok := req.Playbook.([]interface{}); ok {
			result, err = s.engine.EvaluatePlaybook(playbookArray)
		} else {
			// Single rule execution
			result, err = s.engine.EvaluateRule(req.Playbook)
		}
	} else if req.PlaybookName != "" {
		// Load playbook from file
		result, err = s.executePlaybookFromFile(req.PlaybookName)
	} else {
		err = fmt.Errorf("no playbook or playbook_name provided")
	}

	executionTime := time.Since(startTime)

	if err != nil {
		response := types.PlaybookResponse{
			Success:   false,
			Result:    nil,
			Message:   fmt.Sprintf("Playbook execution failed: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		
		s.logger.Error("Playbook execution failed", map[string]interface{}{
			"component":      "server",
			"playbook_name":  req.PlaybookName,
			"error":          err.Error(),
			"execution_time": executionTime.Milliseconds(),
		})
		return
	}

	response := types.PlaybookResponse{
		Success:   true,
		Result:    result,
		Message:   "Playbook executed successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Playbook executed successfully", map[string]interface{}{
		"component":      "server",
		"playbook_name":  req.PlaybookName,
		"has_context":    req.Context != nil,
		"execution_time": executionTime.Milliseconds(),
		"result_type":    fmt.Sprintf("%T", result),
	})
}

// executePlaybookFromFile loads and executes a playbook from a file
func (s *SecAutoServer) executePlaybookFromFile(playbookName string) (interface{}, error) {
	// Get the full path to the playbook
	playbookPath := s.config.GetPlaybookPath(playbookName)
	
	// Check if file exists
	if _, err := os.Stat(playbookPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("playbook file not found: %s", playbookPath)
	}
	
	// Read the playbook file
	data, err := os.ReadFile(playbookPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read playbook file: %v", err)
	}
	
	// Parse JSON
	var playbook []interface{}
	if err := json.Unmarshal(data, &playbook); err != nil {
		return nil, fmt.Errorf("failed to parse playbook JSON: %v", err)
	}
	
	s.logger.Info("Loaded playbook from file", map[string]interface{}{
		"component":     "server",
		"playbook_path": playbookPath,
		"rule_count":    len(playbook),
	})
	
	// Execute the playbook
	return s.engine.EvaluatePlaybook(playbook)
}

// cacheHandler handles cache operations
func (s *SecAutoServer) cacheHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	switch r.Method {
	case http.MethodGet:
		// List cache keys
		pattern := r.URL.Query().Get("pattern")
		response := s.redis.ListCacheKeys(pattern)
		json.NewEncoder(w).Encode(response)
		
		s.logger.Info("Cache keys listed", map[string]interface{}{
			"component": "server",
			"pattern":   pattern,
			"success":   response.Success,
		})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// cacheKeyHandler handles operations on specific cache keys
func (s *SecAutoServer) cacheKeyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Extract key from URL path
	path := strings.TrimPrefix(r.URL.Path, "/cache/")
	if path == "" || path == "stats" || path == "clear" {
		http.Error(w, "Invalid cache key", http.StatusBadRequest)
		return
	}
	
	switch r.Method {
	case http.MethodGet:
		response := s.redis.GetCache(path)
		if !response.Success {
			w.WriteHeader(http.StatusNotFound)
		}
		json.NewEncoder(w).Encode(response)
		
		s.logger.Info("Cache value retrieved", map[string]interface{}{
			"component": "server",
			"key":       path,
			"success":   response.Success,
		})
		
	case http.MethodPost:
		var requestBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		value, exists := requestBody["value"]
		if !exists {
			http.Error(w, "Value required", http.StatusBadRequest)
			return
		}
		
		response := s.redis.SetCache(path, value)
		json.NewEncoder(w).Encode(response)
		
		s.logger.Info("Cache value set", map[string]interface{}{
			"component": "server",
			"key":       path,
			"success":   response.Success,
		})
		
	case http.MethodDelete:
		response := s.redis.DeleteCache(path)
		if !response.Success {
			w.WriteHeader(http.StatusNotFound)
		}
		json.NewEncoder(w).Encode(response)
		
		s.logger.Info("Cache value deleted", map[string]interface{}{
			"component": "server",
			"key":       path,
			"success":   response.Success,
		})
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// cacheStatsHandler handles cache statistics
func (s *SecAutoServer) cacheStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	response := s.redis.GetCacheStats()
	json.NewEncoder(w).Encode(response)
	
	s.logger.Info("Cache stats retrieved", map[string]interface{}{
		"component": "server",
		"success":   response.Success,
	})
}

// cacheClearHandler handles clearing the entire cache
func (s *SecAutoServer) cacheClearHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	response := s.redis.ClearCache()
	json.NewEncoder(w).Encode(response)
	
	s.logger.Info("Cache cleared", map[string]interface{}{
		"component": "server",
		"success":   response.Success,
	})
}

// listHandler handles Redis list operations
func (s *SecAutoServer) listHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Extract list name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/lists/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "List name required", http.StatusBadRequest)
		return
	}
	
	listName := parts[0]
	
	// Check if this is an items operation
	isItemsOperation := len(parts) > 1 && parts[1] == "items"
	
	switch r.Method {
	case http.MethodGet:
		if isItemsOperation {
			http.Error(w, "Method not allowed for items endpoint", http.StatusMethodNotAllowed)
			return
		}
		
		response := s.redis.GetList(listName)
		json.NewEncoder(w).Encode(response)
		
		s.logger.Info("List retrieved", map[string]interface{}{
			"component": "server",
			"list_name": listName,
			"success":   response.Success,
		})
		
	case http.MethodPost:
		if isItemsOperation {
			// Add items to list
			var req types.ListAddRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			
			response := s.redis.AddToList(listName, req.Items, req.Position)
			json.NewEncoder(w).Encode(response)
			
			s.logger.Info("Items added to list", map[string]interface{}{
				"component": "server",
				"list_name": listName,
				"count":     len(req.Items),
				"position":  req.Position,
				"success":   response.Success,
			})
		} else {
			http.Error(w, "POST only allowed on /lists/{name}/items", http.StatusMethodNotAllowed)
		}
		
	case http.MethodDelete:
		if isItemsOperation {
			// Remove specific items from list
			var req types.ListRemoveRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			
			response := s.redis.RemoveFromList(listName, req.Items, req.Count)
			json.NewEncoder(w).Encode(response)
			
			s.logger.Info("Items removed from list", map[string]interface{}{
				"component": "server",
				"list_name": listName,
				"count":     len(req.Items),
				"success":   response.Success,
			})
		} else {
			// Delete entire list
			response := s.redis.DeleteList(listName)
			if !response.Success {
				w.WriteHeader(http.StatusNotFound)
			}
			json.NewEncoder(w).Encode(response)
			
			s.logger.Info("List deleted", map[string]interface{}{
				"component": "server",
				"list_name": listName,
				"success":   response.Success,
			})
		}
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// integrationsHandler handles integration listing and creation
func (s *SecAutoServer) integrationsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// List all integrations
		configs := s.integrationManager.ListConfigs()
		response := types.IntegrationResponse{
			Success:      true,
			Message:      "Integrations retrieved successfully",
			Integrations: make([]*types.IntegrationConfig, 0, len(configs)),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}

		for _, config := range configs {
			response.Integrations = append(response.Integrations, config)
		}

		json.NewEncoder(w).Encode(response)

	case http.MethodPost:
		// Create new integration
		var config types.IntegrationConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Extract name from URL or body
		name := config.Name
		if name == "" {
			http.Error(w, "Integration name is required", http.StatusBadRequest)
			return
		}

		// Validate configuration
		if errors := s.integrationManager.ValidateIntegrationConfig(&config); len(errors) > 0 {
			response := types.ValidationResponse{
				Success:   false,
				Valid:     false,
				Errors:    errors,
				Message:   "Validation failed",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Create integration
		if err := s.integrationManager.CreateConfig(name, &config); err != nil {
			response := types.IntegrationResponse{
				Success:   false,
				Message:   fmt.Sprintf("Failed to create integration: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := types.IntegrationResponse{
			Success:     true,
			Message:     "Integration created successfully",
			Integration: &config,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// integrationHandler handles specific integration operations
func (s *SecAutoServer) integrationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract integration name from URL
	path := strings.TrimPrefix(r.URL.Path, "/integrations/")
	if path == "" {
		http.Error(w, "Integration name required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get specific integration
		config, exists := s.integrationManager.GetConfig(path)
		if !exists {
			response := types.IntegrationResponse{
				Success:   false,
				Message:   "Integration not found",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := types.IntegrationResponse{
			Success:     true,
			Message:     "Integration retrieved successfully",
			Integration: config,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

	case http.MethodPut:
		// Update integration
		var config types.IntegrationConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate configuration
		if errors := s.integrationManager.ValidateIntegrationConfig(&config); len(errors) > 0 {
			response := types.ValidationResponse{
				Success:   false,
				Valid:     false,
				Errors:    errors,
				Message:   "Validation failed",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Update integration
		if err := s.integrationManager.UpdateConfig(path, &config); err != nil {
			response := types.IntegrationResponse{
				Success:   false,
				Message:   fmt.Sprintf("Failed to update integration: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := types.IntegrationResponse{
			Success:     true,
			Message:     "Integration updated successfully",
			Integration: &config,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

	case http.MethodDelete:
		// Delete integration
		if err := s.integrationManager.DeleteConfig(path); err != nil {
			response := types.IntegrationResponse{
				Success:   false,
				Message:   fmt.Sprintf("Failed to delete integration: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := types.IntegrationResponse{
			Success:   true,
			Message:   "Integration deleted successfully",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
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

	w.Header().Set("Content-Type", "application/json")

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get integration name
	name := r.FormValue("name")
	if name == "" {
		// Extract from filename
		name = strings.TrimSuffix(handler.Filename, filepath.Ext(handler.Filename))
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Save integration file
	if err := s.integrationManager.SaveIntegrationFile(name, content); err != nil {
		response := types.IntegrationUploadResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to save integration file: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := types.IntegrationUploadResponse{
		Success:         true,
		Message:         "Integration file uploaded successfully",
		IntegrationName: name,
		Filename:        handler.Filename,
		Size:            handler.Size,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)
}

// Run starts the modular server
func (s *SecAutoServer) Run() error {
	// Setup routes
	http.HandleFunc("/health", s.healthHandler)
	http.HandleFunc("/playbook", s.playbookHandler)
	
	// Cache endpoints
	http.HandleFunc("/cache", s.cacheHandler)
	http.HandleFunc("/cache/stats", s.cacheStatsHandler)
	http.HandleFunc("/cache/clear", s.cacheClearHandler)
	http.HandleFunc("/cache/", s.cacheKeyHandler)
	
	// Redis list endpoints
	http.HandleFunc("/lists/", s.listHandler)
	
	// Integration management endpoints
	http.HandleFunc("/integrations", s.integrationsHandler)
	http.HandleFunc("/integrations/", s.integrationHandler)
	http.HandleFunc("/integrations/upload", s.integrationUploadHandler)
	
	// Swagger/API documentation endpoints
	http.HandleFunc("/docs", s.swagger.ServeHTTP)
	http.HandleFunc("/docs/", s.swagger.ServeHTTP)
	http.HandleFunc("/api-docs", s.swagger.ServeHTTP)

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
	fmt.Printf("📁 Modules loaded: config, logger, validator, rules, cache, types, swagger\n")
	fmt.Printf("🌐 Server will start on %s:%d\n", server.config.Server.Host, server.config.Server.Port)
	fmt.Printf("🏥 Health endpoint: http://%s:%d/health\n", server.config.Server.Host, server.config.Server.Port)
	fmt.Printf("📋 Playbook execution endpoint: http://%s:%d/playbook\n", server.config.Server.Host, server.config.Server.Port)
	fmt.Printf("📚 API Documentation: http://%s:%d/docs\n", server.config.Server.Host, server.config.Server.Port)
	fmt.Printf("🔧 Cache & List endpoints available\n")
	fmt.Printf("⚡ Rules engine active with caching enabled\n")

	if err := server.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}