package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"SoarAuto/pkg/auth"
	"SoarAuto/pkg/automations"
	"SoarAuto/pkg/clients"
	"SoarAuto/pkg/cluster"
	"SoarAuto/pkg/config"
	"SoarAuto/pkg/errors"
	"SoarAuto/pkg/integrations"
	"SoarAuto/pkg/jobs"
	"SoarAuto/pkg/logger"
	"SoarAuto/pkg/playbooks"
	"SoarAuto/pkg/redis"
	"SoarAuto/pkg/rules"
	"SoarAuto/pkg/schedules"
	"SoarAuto/pkg/security"
	"SoarAuto/pkg/swagger"
	tlsManager "SoarAuto/pkg/tls"
	"SoarAuto/pkg/types"
	"SoarAuto/pkg/validator"
	
	_ "github.com/lib/pq"
)

// Helper function for min of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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
	playbookManager          *playbooks.PlaybookManager
	apiKeyManager            *auth.APIKeyManager
	clientManager            *clients.ClientManager
	tlsManager               *tlsManager.TLSManager
	securityMiddleware       *security.SecurityMiddleware
	auditLogger              *security.AuditLogger
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

	// Create PostgreSQL database connection
	var db *sql.DB
	if cfg.Database.Postgres.Host != "" {
		connStr := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable",
			cfg.Database.Postgres.Host,
			cfg.Database.Postgres.Port,
			cfg.Database.Postgres.Username,
			cfg.Database.Postgres.Database)
		
		if cfg.Database.Postgres.Password != "" {
			connStr += fmt.Sprintf(" password=%s", cfg.Database.Postgres.Password)
		}

		db, err = sql.Open("postgres", connStr)
		if err != nil {
			lgr.Warning("Failed to open database connection", map[string]interface{}{
				"error": err.Error(),
			})
		} else if err = db.Ping(); err != nil {
			lgr.Warning("Failed to ping database", map[string]interface{}{
				"error": err.Error(),
			})
			db.Close()
			db = nil
		} else {
			lgr.Info("PostgreSQL database connected successfully", nil)
		}
	}

	// Create Swagger handler
	swaggerHandler, err := swagger.NewSwaggerUIHandler(strconv.Itoa(cfg.Server.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to create Swagger handler: %v", err)
	}

	// Create job store with TTL configuration and manager
	jobStore := jobs.NewJobStoreWithConfig(
		redisClient,
		cfg.Cluster.RunningJobTTL,   // TTL for running jobs
		cfg.Cluster.CompletedJobTTL, // TTL for completed jobs
		cfg.Cluster.FailedJobTTL,    // TTL for failed jobs
		cfg.Cluster.JobStorageTTL,   // Default TTL
	)
	jobManager := jobs.NewJobManager(jobStore)

	// Get Python path from config
	pythonPath := "python3"
	if cfg.Python.VenvPath != "" {
		pythonPath = filepath.Join(cfg.Python.VenvPath, "bin", "python")
		if _, err := os.Stat(filepath.Join(cfg.Python.VenvPath, "Scripts", "python.exe")); err == nil {
			pythonPath = filepath.Join(cfg.Python.VenvPath, "Scripts", "python.exe")
		}
	}

	// Create integration manager
	integrationManager := integrations.NewIntegrationManager(
		cfg.Integrations.ConfigsPath,
		cfg.Integrations.ScriptsPath,
		cfg.Integrations.BuilderScriptPath,
		pythonPath,
		lgr,
	)

	// Create client integration manager with database support
	var clientIntegrationManager *integrations.ClientIntegrationManager
	if db != nil {
		// Use full hybrid storage with database + Redis
		encryptionKey := cfg.Security.IntegrationEncryptionKey
		if encryptionKey == "" {
			encryptionKey = "default-encryption-key-change-in-production"
		}
		clientIntegrationManager, err = integrations.NewClientIntegrationManager(
			db,
			redisClient.GetClient(),
			encryptionKey,
			lgr,
		)
	} else {
		// Fallback to legacy path-based method
		clientIntegrationManager, err = integrations.NewClientIntegrationManagerWithLegacyPath(
			filepath.Join("data", "clients"),
			lgr,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create client integration manager: %v", err)
	}

	// Create automation manager
	// Get absolute paths to ensure they work regardless of working directory
	scriptsPath, _ := filepath.Abs(filepath.Join("data", "automations", "scripts"))
	metadataPath, _ := filepath.Abs(filepath.Join("data", "automations", "metadata"))
	automationManager := automations.NewAutomationManager(scriptsPath, metadataPath)

	// Create playbook manager
	playbooksPath, _ := filepath.Abs(filepath.Join("data", "playbooks"))
	playbookManager := playbooks.NewPlaybookManager(playbooksPath)

	// Create client manager
	clientManager, err := clients.NewClientManager(
		filepath.Join("data", "clients"),
		lgr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client manager: %v", err)
	}

	// Create cluster manager (will be set after server creation)
	var clusterManager *cluster.ClusterManager

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
		playbookManager:          playbookManager,
		clientManager:            clientManager,
	}

	// Set integration managers on the rules engine for integration context execution
	ruleEngine.SetIntegrationManagers(integrationManager, clientIntegrationManager)
	
	// Create cluster manager now that we have the server
	clusterManager = cluster.NewClusterManager("node-modular-1", jobManager, server, lgr)
	server.clusterManager = clusterManager
	clusterManager.StartClusterServices()

	// Create schedule manager (needs server for job execution)
	scheduleManager := schedules.NewScheduleManager(
		filepath.Join("data", "schedules"),
		server, // server implements JobExecutor interface
	)
	server.scheduleManager = scheduleManager

	// Create API key manager
	apiKeyManager := auth.NewAPIKeyManager(
		cfg.Security.APIKeysFile,
		cfg.Security.APIKeys,
	)
	server.apiKeyManager = apiKeyManager

	// Create TLS manager if TLS is enabled
	if cfg.Security.TLS.Enabled {
		tlsMgr := tlsManager.NewTLSManager(&cfg.Security.TLS, lgr)
		server.tlsManager = tlsMgr
		
		// Validate certificates
		if err := tlsMgr.ValidateCertificates(); err != nil {
			lgr.Error("TLS certificate validation failed", map[string]interface{}{
				"component": "server",
				"error":     err.Error(),
			})
			return nil, fmt.Errorf("TLS certificate validation failed: %v", err)
		}
	}

	// Create security middleware
	securityConfig := security.DefaultSecurityConfig()
	securityConfig.EnableRateLimiting = cfg.Security.RateLimiting.Enabled
	securityConfig.RequireHTTPS = cfg.Security.TLS.Enabled
	securityMiddleware := security.NewSecurityMiddleware(lgr, securityConfig)
	server.securityMiddleware = securityMiddleware

	// Create audit logger
	auditConfig := security.DefaultAuditConfig()
	auditLogger, err := security.NewAuditLogger(lgr, auditConfig)
	if err != nil {
		lgr.Error("Failed to create audit logger", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("failed to create audit logger: %v", err)
	}
	server.auditLogger = auditLogger

	return server, nil
}

// corsMiddleware handles CORS (Cross-Origin Resource Sharing) headers
func (s *SecAutoServer) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only apply CORS if enabled in config
		if !s.config.Security.CORS.Enabled {
			next(w, r)
			return
		}

		// Get origin from request
		origin := r.Header.Get("Origin")
		
		// Check if origin is allowed
		allowedOrigin := ""
		for _, configOrigin := range s.config.Security.CORS.AllowedOrigins {
			if configOrigin == "*" || configOrigin == origin {
				allowedOrigin = origin
				if allowedOrigin == "" {
					allowedOrigin = "*"
				}
				break
			}
		}
		
		// Set CORS headers
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}
		
		// Set allowed methods
		if len(s.config.Security.CORS.AllowedMethods) > 0 {
			methods := strings.Join(s.config.Security.CORS.AllowedMethods, ", ")
			w.Header().Set("Access-Control-Allow-Methods", methods)
		}
		
		// Set allowed headers
		if len(s.config.Security.CORS.AllowedHeaders) > 0 {
			headers := strings.Join(s.config.Security.CORS.AllowedHeaders, ", ")
			w.Header().Set("Access-Control-Allow-Headers", headers)
		}
		
		// Set max age for preflight cache
		if s.config.Security.CORS.MaxAge > 0 {
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(s.config.Security.CORS.MaxAge))
		}
		
		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next(w, r)
	}
}

// authMiddleware provides API key authentication for all endpoints with enhanced security
func (s *SecAutoServer) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for health endpoint
		if r.URL.Path == "/health" || r.URL.Path == "/docs" || strings.HasPrefix(r.URL.Path, "/docs/") {
			next(w, r)
			return
		}

		// Get API key from header or query parameter
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		if apiKey == "" {
			err := errors.AuthError(errors.ErrCodeAuthMissing, "API key required")
			
			// Log authentication failure
			s.auditLogger.LogAuthenticationEvent(false, "", "", r, err)
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(err.HTTPStatusCode())
			json.NewEncoder(w).Encode(err.ToAPIResponse())
			return
		}

		// Validate API key format
		if err := s.securityMiddleware.ValidateAPIKey(apiKey); err != nil {
			// Log authentication failure
			s.auditLogger.LogAuthenticationEvent(false, "", "", r, err)
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(err.(*errors.SecAutoError).HTTPStatusCode())
			json.NewEncoder(w).Encode(err.(*errors.SecAutoError).ToAPIResponse())
			return
		}

		if !s.apiKeyManager.IsValidKey(apiKey) {
			err := errors.NewErrorBuilder(errors.ErrCodeAuthInvalid, "Invalid API key").
				WithContext("api_key_prefix", apiKey[:8]).
				Build()
			
			// Log authentication failure
			s.auditLogger.LogAuthenticationEvent(false, "", "", r, err)
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(err.HTTPStatusCode())
			json.NewEncoder(w).Encode(err.ToAPIResponse())
			return
		}

		// Update last used timestamp
		s.apiKeyManager.UpdateLastUsed(apiKey)
		
		// Log successful authentication
		s.auditLogger.LogAuthenticationEvent(true, apiKey[:8], "", r, nil)

		// Call the next handler
		next(w, r)
	}
}

// middleware chains CORS and authentication middleware
func (s *SecAutoServer) middleware(next http.HandlerFunc) http.HandlerFunc {
	return s.corsMiddleware(s.authMiddleware(next))
}

// publicMiddleware applies only CORS middleware (for health/docs endpoints)
func (s *SecAutoServer) publicMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return s.corsMiddleware(next)
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
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// List all global integrations
	definitions := s.integrationManager.ListIntegrations()
	
	integrations := make([]map[string]interface{}, 0, len(definitions))
	for _, def := range definitions {
		integrations = append(integrations, map[string]interface{}{
			"name":         def.Name,
			"version":      def.Version,
			"description":  def.Description,
			"author":       def.Author,
			"functions":    def.Functions,
			"configuration": def.Configuration,
			"created_at":   def.CreatedAt,
			"updated_at":   def.UpdatedAt,
			"built":        s.integrationManager.IsIntegrationBuilt(def.Name),
		})
	}

	response := map[string]interface{}{
		"success":      true,
		"message":      "Integrations retrieved successfully",
		"integrations": integrations,
		"count":        len(integrations),
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
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
		definition, err := s.integrationManager.GetIntegration(path)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Integration not found",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":     true,
			"message":     "Integration retrieved successfully",
			"integration": definition,
			"built":       s.integrationManager.IsIntegrationBuilt(definition.Name),
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

	case http.MethodDelete:
		// Delete integration
		if err := s.integrationManager.DeleteIntegration(path); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to delete integration: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Integration deleted successfully",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// integrationUploadHandler handles integration uploads (definition + script)
func (s *SecAutoServer) integrationUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse multipart form (50MB max for integration uploads)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get integration definition file
	definitionFile, definitionHandler, err := r.FormFile("definition")
	if err != nil {
		http.Error(w, "Integration definition file is required", http.StatusBadRequest)
		return
	}
	defer definitionFile.Close()

	// Read definition file content
	definitionContent, err := io.ReadAll(definitionFile)
	if err != nil {
		http.Error(w, "Failed to read definition file", http.StatusBadRequest)
		return
	}


	// Validate definition file extension
	if !strings.HasSuffix(strings.ToLower(definitionHandler.Filename), ".json") {
		response := map[string]interface{}{
			"success":   false,
			"message":   "Definition file must have .json extension",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Parse integration definition
	var definition integrations.IntegrationDefinition
	if err := json.Unmarshal(definitionContent, &definition); err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Invalid integration definition: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get script file
	scriptFile, handler, err := r.FormFile("script")
	if err != nil {
		http.Error(w, "Integration script file is required", http.StatusBadRequest)
		return
	}
	defer scriptFile.Close()

	// Validate script filename matches entry point
	if handler.Filename != definition.Backend.EntryPoint {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Script filename '%s' does not match entry point '%s'", handler.Filename, definition.Backend.EntryPoint),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Read script content
	scriptContent, err := io.ReadAll(scriptFile)
	if err != nil {
		http.Error(w, "Failed to read script file", http.StatusInternalServerError)
		return
	}

	// Upload the integration using the IntegrationManager
	if err := s.integrationManager.UploadIntegration(&definition, scriptContent); err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Failed to upload integration: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Success response
	response := map[string]interface{}{
		"success":          true,
		"message":          "Integration uploaded and built successfully",
		"integration_name": definition.Name,
		"version":          definition.Version,
		"filename":         handler.Filename,
		"size":             handler.Size,
		"built":            s.integrationManager.IsIntegrationBuilt(definition.Name),
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)
}

// triggerIntegrationBuild triggers the Python script to build integration backend
func (s *SecAutoServer) triggerIntegrationBuild(configPath string, integrationName string) map[string]interface{} {
	// Get Python path from config
	pythonPath := filepath.Join(s.config.Python.VenvPath, "bin", "python")
	if runtime.GOOS == "windows" {
		pythonPath = filepath.Join(s.config.Python.VenvPath, "Scripts", "python.exe")
	}
	
	// Build path to integration builder script from config
	scriptPath := s.config.Integrations.BuilderScriptPath
	if !filepath.IsAbs(scriptPath) {
		workDir, _ := os.Getwd()
		scriptPath = filepath.Join(workDir, scriptPath)
	}
	// Pass the current working directory as base path
	workDir, _ := os.Getwd()
	cmd := exec.Command(pythonPath, scriptPath, "build", "--config", configPath, "--base-path", workDir)
	
	// Set working directory to current working directory
	cmd.Dir = workDir
	
	// Run the command and capture output
	output, err := cmd.CombinedOutput()
	
	// Parse the output as JSON
	var result map[string]interface{}
	if err != nil {
		s.logger.Error("Failed to trigger integration build", map[string]interface{}{
			"integration": integrationName,
			"error":       err.Error(),
			"output":      string(output),
		})
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"output":  string(output),
		}
	}
	
	// Try to parse JSON output
	if err := json.Unmarshal(output, &result); err != nil {
		// If not JSON, return raw output
		return map[string]interface{}{
			"success": true,
			"message": string(output),
		}
	}
	
	s.logger.Info("Integration build triggered", map[string]interface{}{
		"integration": integrationName,
		"result":      result,
	})
	
	return result
}

// integrationBuildStatusHandler handles requests for integration build status
func (s *SecAutoServer) integrationBuildStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	// Get integration name from path
	path := strings.TrimPrefix(r.URL.Path, "/integrations/build-status/")
	integrationName := strings.TrimSpace(path)
	
	// Get Python path from config
	pythonPath := filepath.Join(s.config.Python.VenvPath, "bin", "python")
	if runtime.GOOS == "windows" {
		pythonPath = filepath.Join(s.config.Python.VenvPath, "Scripts", "python.exe")
	}
	
	// Build path to integration builder script from config
	scriptPath := s.config.Integrations.BuilderScriptPath
	if !filepath.IsAbs(scriptPath) {
		workDir, _ := os.Getwd()
		scriptPath = filepath.Join(workDir, scriptPath)
	}
	var cmd *exec.Cmd
	
	// Pass the current working directory as base path
	workDir, _ := os.Getwd()
	if integrationName != "" {
		cmd = exec.Command(pythonPath, scriptPath, "status", "--name", integrationName, "--base-path", workDir)
	} else {
		cmd = exec.Command(pythonPath, scriptPath, "status", "--base-path", workDir)
	}
	
	// Set working directory to current working directory
	cmd.Dir = workDir
	
	// Run the command
	output, err := cmd.CombinedOutput()
	
	// Parse the output as JSON
	var status map[string]interface{}
	if err != nil {
		response := map[string]interface{}{
			"success":   false,
			"error":     err.Error(),
			"output":    string(output),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Parse JSON output
	if err := json.Unmarshal(output, &status); err != nil {
		response := map[string]interface{}{
			"success":   false,
			"error":     "Failed to parse status",
			"output":    string(output),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	response := map[string]interface{}{
		"success":   true,
		"status":    status,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	if integrationName != "" {
		response["integration"] = integrationName
	}
	
	json.NewEncoder(w).Encode(response)
}

// clientIntegrationsHandler handles listing client integrations
func (s *SecAutoServer) clientIntegrationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Extract client ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/clients/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "integrations" {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}
	clientID := parts[0]

	// Verify client exists
	if _, err := s.clientManager.GetClient(clientID); err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Client not found: %s", clientID),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get client integrations
	configs, err := s.clientIntegrationManager.ListClientIntegrations(clientID)
	if err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Failed to list client integrations: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success":      true,
		"message":      "Client integrations retrieved successfully",
		"client_id":    clientID,
		"integrations": configs,
		"count":        len(configs),
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// clientIntegrationConfigHandler handles client-specific integration configuration
func (s *SecAutoServer) clientIntegrationConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract client ID and integration name from URL
	path := strings.TrimPrefix(r.URL.Path, "/clients/")
	parts := strings.Split(path, "/")
	if len(parts) < 4 || parts[1] != "integrations" || parts[3] != "config" {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}
	clientID := parts[0]
	integrationName := parts[2]

	// Verify client exists
	if _, err := s.clientManager.GetClient(clientID); err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Client not found: %s", clientID),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Verify integration exists
	if _, err := s.integrationManager.GetIntegration(integrationName); err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Integration not found: %s", integrationName),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get client integration config
		config, err := s.clientIntegrationManager.GetClientIntegrationConfig(clientID, integrationName)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to get integration config: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":     true,
			"message":     "Integration config retrieved successfully",
			"client_id":   clientID,
			"integration": integrationName,
			"config":      config,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

	case http.MethodPost, http.MethodPut:
		// Create or update client integration config
		var config integrations.ClientIntegrationConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Set required fields
		config.Name = integrationName
		config.ClientID = clientID

		// Validate against global integration
		globalIntegration, err := s.integrationManager.GetIntegration(integrationName)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to get global integration: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := s.clientIntegrationManager.ValidateClientConfig(&config, globalIntegration); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Configuration validation failed: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Save the configuration
		if err := s.clientIntegrationManager.SaveClientIntegrationConfig(clientID, &config); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to save integration config: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		statusCode := http.StatusCreated
		if r.Method == http.MethodPut {
			statusCode = http.StatusOK
		}

		response := map[string]interface{}{
			"success":     true,
			"message":     "Integration config saved successfully",
			"client_id":   clientID,
			"integration": integrationName,
			"config":      config,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)

	case http.MethodDelete:
		// Delete client integration config
		if err := s.clientIntegrationManager.DeleteClientIntegrationConfig(clientID, integrationName); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to delete integration config: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":     true,
			"message":     "Integration config deleted successfully",
			"client_id":   clientID,
			"integration": integrationName,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// clientIntegrationExecuteHandler handles executing integrations for clients
func (s *SecAutoServer) clientIntegrationExecuteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Extract client ID and integration name from URL
	path := strings.TrimPrefix(r.URL.Path, "/clients/")
	parts := strings.Split(path, "/")
	if len(parts) < 4 || parts[1] != "integrations" || parts[3] != "execute" {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}
	clientID := parts[0]
	integrationName := parts[2]

	// Parse request body
	var request struct {
		Function string                 `json:"function"`
		Params   map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Verify client exists
	if _, err := s.clientManager.GetClient(clientID); err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Client not found: %s", clientID),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get client integration config
	clientConfig, err := s.clientIntegrationManager.GetClientIntegrationConfig(clientID, integrationName)
	if err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Client integration config not found: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if integration is enabled for this client
	if !clientConfig.Enabled {
		response := map[string]interface{}{
			"success":   false,
			"message":   "Integration is disabled for this client",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Execute the integration
	result, err := s.integrationManager.ExecuteIntegration(integrationName, clientConfig, request.Function, request.Params)
	if err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Integration execution failed: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success":     true,
		"message":     "Integration executed successfully",
		"client_id":   clientID,
		"integration": integrationName,
		"function":    request.Function,
		"result":      result,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)
}

// clusterHandler handles cluster information requests
func (s *SecAutoServer) clusterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get cluster information
	clusterInfo := s.clusterManager.GetClusterInfo()

	// Create response
	response := map[string]interface{}{
		"success":   true,
		"cluster":   clusterInfo,
		"message":   "Cluster information retrieved successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)

	s.logger.Info("Cluster information retrieved", map[string]interface{}{
		"component":   "server",
		"remote_addr": r.RemoteAddr,
	})
}

// clusterJobsHandler handles distributed job submissions and listings
func (s *SecAutoServer) clusterJobsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// List cluster jobs (for now, return basic info)
		response := map[string]interface{}{
			"success": true,
			"jobs":    []interface{}{}, // Would be populated with actual cluster jobs
			"total":   0,
			"message": "Cluster jobs retrieved successfully", 
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)

	case http.MethodPost:
		// Submit job to distributed queue
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
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Determine playbook to submit
		var playbook interface{}
		if req.Playbook != nil {
			playbook = req.Playbook
		} else if req.PlaybookName != "" {
			// For cluster jobs, we'd need to load the playbook file
			// For now, create a simple reference
			playbook = map[string]interface{}{
				"type": "playbook_reference",
				"name": req.PlaybookName,
			}
		} else {
			http.Error(w, "No playbook or playbook_name provided", http.StatusBadRequest)
			return
		}

		// Submit to cluster
		jobID, err := s.clusterManager.SubmitJob(playbook, req.Context)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to submit job to cluster: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"job_id":    jobID,
			"message":   "Job submitted to cluster successfully",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Job submitted to cluster", map[string]interface{}{
			"component": "server",
			"job_id":    jobID,
			"has_context": req.Context != nil,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// clusterJobHandler handles individual distributed job operations
func (s *SecAutoServer) clusterJobHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract job ID from URL path
	jobID := strings.TrimPrefix(r.URL.Path, "/cluster/jobs/")
	if jobID == "" {
		http.Error(w, "Job ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get distributed job status
		job, err := s.clusterManager.GetJob(jobID)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Job not found: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"job":       job,
			"message":   "Distributed job retrieved successfully",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Distributed job retrieved", map[string]interface{}{
			"component": "server",
			"job_id":    jobID,
			"status":    job.Status,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// playbookAsyncHandler handles asynchronous playbook execution
func (s *SecAutoServer) playbookAsyncHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

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
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Submit to cluster for asynchronous execution
	var playbook interface{}
	if req.Playbook != nil {
		playbook = req.Playbook
	} else if req.PlaybookName != "" {
		// Load playbook from file
		playbookContent, err := s.playbookManager.GetPlaybook(req.PlaybookName)
		if err != nil {
			response := types.PlaybookResponse{
				Success:   false,
				Message:   fmt.Sprintf("Playbook not found: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Parse playbook content
		if err := json.Unmarshal(playbookContent, &playbook); err != nil {
			response := types.PlaybookResponse{
				Success:   false,
				Message:   fmt.Sprintf("Invalid playbook format: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
	} else {
		http.Error(w, "No playbook or playbook_name provided", http.StatusBadRequest)
		return
	}

	// Submit to cluster
	jobID, err := s.clusterManager.SubmitJob(playbook, req.Context)
	if err != nil {
		response := types.PlaybookResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to submit async playbook: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := types.PlaybookResponse{
		Success:   true,
		JobID:     jobID,
		Message:   "Playbook submitted for asynchronous execution",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Async playbook submitted", map[string]interface{}{
		"component":     "server",
		"job_id":        jobID,
		"playbook_name": req.PlaybookName,
		"has_context":   req.Context != nil,
	})
}

// playbookUploadHandler handles playbook file uploads
func (s *SecAutoServer) playbookUploadHandler(w http.ResponseWriter, r *http.Request) {
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

	// Get playbook name
	name := r.FormValue("name")
	if name == "" {
		// Extract from filename
		name = strings.TrimSuffix(handler.Filename, filepath.Ext(handler.Filename))
	}

	// Sanitize the name to prevent path traversal
	name = security.SanitizeFilename(name)
	
	// Additional validation (without .json extension)
	nameWithoutExt := strings.TrimSuffix(name, ".json")
	if err := security.ValidateUploadName(nameWithoutExt); err != nil {
		response := types.ValidationResponse{
			Success:   false,
			Valid:     false,
			Errors:    []types.ValidationError{{Field: "name", Message: err.Error()}},
			Message:   fmt.Sprintf("Invalid playbook name: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Validate playbook content
	if errors := s.playbookManager.ValidatePlaybook(content); len(errors) > 0 {
		response := types.ValidationResponse{
			Success:   false,
			Valid:     false,
			Errors:    errors,
			Message:   "Playbook validation failed",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Save playbook file
	if err := s.playbookManager.SavePlaybook(name, content); err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Failed to save playbook: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success":       true,
		"message":       "Playbook uploaded successfully",
		"playbook_name": name,
		"filename":      handler.Filename,
		"size":          handler.Size,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Playbook uploaded", map[string]interface{}{
		"component":     "server",
		"playbook_name": name,
		"filename":      handler.Filename,
		"size":          handler.Size,
	})
}

// playbooksHandler handles listing all playbooks
func (s *SecAutoServer) playbooksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get all playbooks
	playbooks, err := s.playbookManager.ListPlaybooks()
	if err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Failed to list playbooks: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success":    true,
		"playbooks":  playbooks,
		"count":      len(playbooks),
		"message":    "Playbooks retrieved successfully",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Playbooks listed", map[string]interface{}{
		"component": "server",
		"count":     len(playbooks),
	})
}

// playbookDeleteHandler handles deleting a specific playbook
func (s *SecAutoServer) playbookDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Extract playbook name from URL
	path := strings.TrimPrefix(r.URL.Path, "/playbook/")
	if path == "" {
		http.Error(w, "Playbook name required", http.StatusBadRequest)
		return
	}

	// Check if playbook exists
	if !s.playbookManager.PlaybookExists(path) {
		response := map[string]interface{}{
			"success":   false,
			"message":   "Playbook not found",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get dependencies before deletion
	dependencies := []string{}
	if content, err := s.playbookManager.GetPlaybook(path); err == nil {
		if deps, err := s.playbookManager.GetPlaybookDependencies(content); err == nil {
			dependencies = deps
		}
	}

	// Delete playbook
	if err := s.playbookManager.DeletePlaybook(path); err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Failed to delete playbook: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := types.PlaybookDeleteResponse{
		Success:      true,
		Message:      "Playbook deleted successfully",
		PlaybookName: path,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Playbook deleted", map[string]interface{}{
		"component":     "server",
		"playbook_name": path,
		"dependencies":  dependencies,
	})
}

// automationUploadHandler handles uploading automation scripts
func (s *SecAutoServer) automationUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB limit
		response := map[string]interface{}{
			"success":   false,
			"message":   "Failed to parse form data",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get the file
	file, handler, err := r.FormFile("automation")
	if err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   "No file provided",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer file.Close()

	// Validate file type
	if !strings.HasSuffix(handler.Filename, ".py") {
		response := map[string]interface{}{
			"success":   false,
			"message":   "Only Python (.py) files are allowed",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   "Failed to read file content",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get automation name (remove .py extension)
	name := strings.TrimSuffix(handler.Filename, ".py")
	
	// Override name if provided in form
	if formName := r.FormValue("name"); formName != "" {
		name = formName
	}

	// Sanitize the name to prevent path traversal
	name = security.SanitizeFilename(name)
	
	// Additional validation
	if err := security.ValidateUploadName(name); err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Invalid automation name: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Save automation script
	if err := s.automationManager.SaveAutomationScript(name, content); err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Failed to save automation: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success":         true,
		"message":         "Automation uploaded successfully",
		"filename":        handler.Filename,
		"automation_name": name,
		"size":            handler.Size,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Automation uploaded", map[string]interface{}{
		"component":       "server",
		"automation_name": name,
		"filename":        handler.Filename,
		"size":            handler.Size,
	})
}

// automationsListHandler handles listing all automation scripts
func (s *SecAutoServer) automationsListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get automations list
	automations, err := s.automationManager.ListAutomations()
	if err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Failed to list automations: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := types.AutomationListResponse{
		Success:     true,
		Message:     "Automations retrieved successfully",
		Automations: automations,
		Count:       len(automations),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Automations listed", map[string]interface{}{
		"component": "server",
		"count":     len(automations),
	})
}

// automationDeleteHandler handles deleting a specific automation script
func (s *SecAutoServer) automationDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Extract automation name from URL
	path := strings.TrimPrefix(r.URL.Path, "/automation/")
	if path == "" {
		response := map[string]interface{}{
			"success":   false,
			"message":   "Automation name required",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get dependencies before deletion
	dependencies, _ := s.automationManager.GetAutomationDependencies(path)

	// Delete automation script
	if err := s.automationManager.DeleteAutomationScript(path); err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Failed to delete automation: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success":         true,
		"message":         "Automation deleted successfully",
		"automation_name": path,
		"dependencies":    dependencies,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Automation deleted", map[string]interface{}{
		"component":       "server",
		"automation_name": path,
		"dependencies":    dependencies,
	})
}

// automationMetadataHandler handles metadata CRUD operations
func (s *SecAutoServer) automationMetadataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// List all metadata
		metadata := s.automationManager.ListMetadata()
		response := types.AutomationMetadataResponse{
			Success:   true,
			Message:   "Automation metadata retrieved successfully",
			Metadata:  metadata,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Automation metadata listed", map[string]interface{}{
			"component": "server",
			"count":     len(metadata),
		})

	case http.MethodPost:
		// Create new metadata
		var metadata types.AutomationMetadata
		if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Invalid JSON format",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Validate metadata
		if errors := s.automationManager.ValidateAutomationMetadata(&metadata); len(errors) > 0 {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Validation failed",
				"errors":    errors,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Create metadata
		if err := s.automationManager.CreateMetadata(metadata.Name, &metadata); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to create metadata: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Automation metadata created successfully",
			"metadata":  metadata,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Automation metadata created", map[string]interface{}{
			"component": "server",
			"name":      metadata.Name,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// automationMetadataItemHandler handles specific metadata operations
func (s *SecAutoServer) automationMetadataItemHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract metadata name from URL
	name := strings.TrimPrefix(r.URL.Path, "/automation/metadata/")
	if name == "" {
		response := map[string]interface{}{
			"success":   false,
			"message":   "Automation name required",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get specific metadata
		metadata, exists := s.automationManager.GetMetadata(name)
		if !exists {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Automation metadata not found",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Automation metadata retrieved successfully",
			"metadata":  metadata,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

	case http.MethodPut:
		// Update metadata
		var metadata types.AutomationMetadata
		if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Invalid JSON format",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Validate metadata
		if errors := s.automationManager.ValidateAutomationMetadata(&metadata); len(errors) > 0 {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Validation failed",
				"errors":    errors,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Update metadata
		if err := s.automationManager.UpdateMetadata(name, &metadata); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to update metadata: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Automation metadata updated successfully",
			"metadata":  metadata,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Automation metadata updated", map[string]interface{}{
			"component": "server",
			"name":      name,
		})

	case http.MethodDelete:
		// Delete metadata
		if err := s.automationManager.DeleteMetadata(name); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to delete metadata: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Automation metadata deleted successfully",
			"name":      name,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Automation metadata deleted", map[string]interface{}{
			"component": "server",
			"name":      name,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// jobsHandler handles job listing and creation
func (s *SecAutoServer) jobsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// List jobs with optional filters
		status := r.URL.Query().Get("status")
		limitStr := r.URL.Query().Get("limit")
		
		limit := 50 // Default limit
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		// Get jobs from job manager
		jobs := s.jobManager.ListJobs(status, limit)

		response := map[string]interface{}{
			"success":   true,
			"message":   "Jobs retrieved successfully",
			"jobs":      jobs,
			"count":     len(jobs),
			"filters": map[string]interface{}{
				"status": status,
				"limit":  limit,
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Jobs listed", map[string]interface{}{
			"component": "server",
			"count":     len(jobs),
			"status":    status,
			"limit":     limit,
		})

	case http.MethodPost:
		// Create new job (submit playbook for execution)
		var request struct {
			Playbook interface{}            `json:"playbook"`
			Context  map[string]interface{} `json:"context"`
			Priority int                    `json:"priority,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Invalid JSON format",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Create job through job manager
		job, err := s.jobManager.CreateJob(request.Playbook, request.Context)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to create job: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Set priority if provided
		if request.Priority > 0 {
			job.Priority = request.Priority
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Job created successfully",
			"job":       job,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Job created", map[string]interface{}{
			"component": "server",
			"job_id":    job.ID,
			"status":    job.Status,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// jobsStatsHandler handles job statistics
func (s *SecAutoServer) jobsStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get job statistics
	stats, err := s.jobManager.GetStats()
	if err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Failed to get job statistics: %v", err),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"message":   "Job statistics retrieved successfully",
		"stats":     stats,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Job statistics retrieved", map[string]interface{}{
		"component":  "server",
		"total_jobs": stats.TotalJobs,
		"completed":  stats.Completed,
		"failed":     stats.Failed,
		"running":    stats.Running,
		"pending":    stats.Pending,
	})
}

// jobHandler handles individual job operations
func (s *SecAutoServer) jobHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract job ID from URL
	jobID := strings.TrimPrefix(r.URL.Path, "/job/")
	if jobID == "" {
		response := map[string]interface{}{
			"success":   false,
			"message":   "Job ID required",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get specific job
		job, err := s.jobManager.GetJob(jobID)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Job not found",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Job retrieved successfully",
			"job":       job,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Job retrieved", map[string]interface{}{
			"component": "server",
			"job_id":    jobID,
			"status":    job.Status,
		})

	case http.MethodPut:
		// Update job (mainly for status updates)
		var request struct {
			Status string `json:"status"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Invalid JSON format",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Update job status
		if err := s.jobManager.UpdateJobStatus(jobID, request.Status); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to update job: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Get updated job
		job, err := s.jobManager.GetJob(jobID)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Job updated but failed to retrieve updated version",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Job updated successfully",
			"job":       job,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Job updated", map[string]interface{}{
			"component": "server",
			"job_id":    jobID,
			"status":    request.Status,
		})

	case http.MethodDelete:
		// Delete job
		if err := s.jobManager.DeleteJob(jobID); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to delete job: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Job deleted successfully",
			"job_id":    jobID,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Job deleted", map[string]interface{}{
			"component": "server",
			"job_id":    jobID,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// schedulesHandler handles schedule listing and creation
func (s *SecAutoServer) schedulesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// List schedules with optional status filter
		status := r.URL.Query().Get("status")
		var scheduleStatus types.ScheduleStatus = types.ScheduleStatusAll
		
		switch status {
		case "enabled":
			scheduleStatus = types.ScheduleStatusEnabled
		case "disabled":
			scheduleStatus = types.ScheduleStatusDisabled
		}

		schedules := s.scheduleManager.ListSchedules(scheduleStatus)
		
		response := map[string]interface{}{
			"success":   true,
			"message":   "Schedules retrieved successfully",
			"schedules": schedules,
			"count":     len(schedules),
			"filters": map[string]interface{}{
				"status": status,
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Schedules listed", map[string]interface{}{
			"component": "server",
			"count":     len(schedules),
			"filter":    status,
		})

	case http.MethodPost:
		// Create new schedule
		var schedule types.JobSchedule
		if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Invalid JSON format",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Validate required fields
		if schedule.Name == "" {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Schedule name is required",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		if schedule.CronExpr == "" {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Cron expression is required",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		if schedule.Playbook == nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Playbook is required",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Create schedule
		if err := s.scheduleManager.CreateSchedule(&schedule); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to create schedule: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":    true,
			"message":    "Schedule created successfully",
			"schedule":   &schedule,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Schedule created", map[string]interface{}{
			"component":    "server",
			"schedule_id":  schedule.ID,
			"schedule_name": schedule.Name,
			"enabled":      schedule.Enabled,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// scheduleStatsHandler handles schedule statistics
func (s *SecAutoServer) scheduleStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.scheduleManager.GetScheduleStats()
	
	response := map[string]interface{}{
		"success":   true,
		"message":   "Schedule statistics retrieved successfully", 
		"stats":     stats,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Schedule statistics retrieved", map[string]interface{}{
		"component": "server",
		"stats":     stats,
	})
}

// scheduleHandler handles individual schedule operations
func (s *SecAutoServer) scheduleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract schedule ID from URL
	scheduleID := strings.TrimPrefix(r.URL.Path, "/schedule/")
	if scheduleID == "" {
		response := map[string]interface{}{
			"success":   false,
			"message":   "Schedule ID required",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get specific schedule
		schedule, exists := s.scheduleManager.GetSchedule(scheduleID)
		if !exists {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Schedule not found",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Schedule retrieved successfully",
			"schedule":  schedule,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Schedule retrieved", map[string]interface{}{
			"component":    "server",
			"schedule_id":  scheduleID,
			"schedule_name": schedule.Name,
		})

	case http.MethodPut:
		// Update schedule
		var updates types.JobSchedule
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Invalid JSON format",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := s.scheduleManager.UpdateSchedule(scheduleID, &updates); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to update schedule: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Get updated schedule
		schedule, _ := s.scheduleManager.GetSchedule(scheduleID)
		
		response := map[string]interface{}{
			"success":   true,
			"message":   "Schedule updated successfully",
			"schedule":  schedule,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Schedule updated", map[string]interface{}{
			"component":    "server",
			"schedule_id":  scheduleID,
			"schedule_name": schedule.Name,
		})

	case http.MethodDelete:
		// Delete schedule
		if err := s.scheduleManager.DeleteSchedule(scheduleID); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to delete schedule: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":    true,
			"message":    "Schedule deleted successfully",
			"schedule_id": scheduleID,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Schedule deleted", map[string]interface{}{
			"component":   "server", 
			"schedule_id": scheduleID,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// scheduleExecuteHandler handles manual schedule execution
func (s *SecAutoServer) scheduleExecuteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract schedule ID from URL
	scheduleID := strings.TrimPrefix(r.URL.Path, "/schedule/execute/")
	if scheduleID == "" {
		response := map[string]interface{}{
			"success":   false,
			"message":   "Schedule ID required",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get schedule
	schedule, exists := s.scheduleManager.GetSchedule(scheduleID)
	if !exists {
		response := map[string]interface{}{
			"success":   false,
			"message":   "Schedule not found",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Execute playbook manually
	result, err := s.ExecutePlaybook(schedule.Playbook, schedule.Context)
	if err != nil {
		response := map[string]interface{}{
			"success":   false,
			"message":   fmt.Sprintf("Schedule execution failed: %v", err),
			"schedule_id": scheduleID,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success":     true,
		"message":     "Schedule executed successfully",
		"schedule_id": scheduleID,
		"result":      result,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Schedule executed manually", map[string]interface{}{
		"component":    "server",
		"schedule_id":  scheduleID,
		"schedule_name": schedule.Name,
	})
}

// apiKeysHandler handles API key listing and creation
func (s *SecAutoServer) apiKeysHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// List API keys
		keys := s.apiKeyManager.ListAPIKeys()
		
		response := types.APIKeyListResponse{
			Success:   true,
			Message:   "API keys retrieved successfully",
			APIKeys:   keys,
			Count:     len(keys),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("API keys listed", map[string]interface{}{
			"component": "server",
			"count":     len(keys),
		})

	case http.MethodPost:
		// Create new API key
		var request types.APIKeyCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response := types.APIKeyCreateResponse{
				Success:   false,
				Message:   "Invalid JSON format",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Validate request
		if request.Name == "" {
			response := types.APIKeyCreateResponse{
				Success:   false,
				Message:   "API key name is required",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Create API key (use "api" as created_by since we don't have user context)
		apiKey, err := s.apiKeyManager.CreateAPIKey(request.Name, request.Description, "api")
		if err != nil {
			response := types.APIKeyCreateResponse{
				Success:   false,
				Message:   fmt.Sprintf("Failed to create API key: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := types.APIKeyCreateResponse{
			Success:   true,
			Message:   "API key created successfully",
			APIKey:    apiKey,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("API key created", map[string]interface{}{
			"component": "server",
			"key_name":  apiKey.Name,
			"key_prefix": apiKey.Key[:12] + "...",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// apiKeyStatsHandler handles API key statistics
func (s *SecAutoServer) apiKeyStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.apiKeyManager.GetStats()
	response := types.APIKeyStatsResponse{
		Success:   true,
		Message:   "API key statistics retrieved successfully",
		Stats:     stats,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)

	s.logger.Info("API key stats retrieved", map[string]interface{}{
		"component": "server",
		"total":     stats.Total,
		"active":    stats.Active,
	})
}

// clientsHandler handles client listing and creation
func (s *SecAutoServer) clientsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// List all clients
		clients := s.clientManager.ListClients()
		
		response := map[string]interface{}{
			"success":   true,
			"message":   "Clients retrieved successfully",
			"clients":   clients,
			"count":     len(clients),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.logger.Info("Clients listed", map[string]interface{}{
			"component": "clients",
			"count":     len(clients),
		})

	case http.MethodPost:
		// Create new client
		var request struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Invalid JSON format",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Validate request
		if request.Name == "" {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Client name is required",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Create client
		client, err := s.clientManager.CreateClient(request.Name, request.Description)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to create client: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Client created successfully",
			"client":    client,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.clientManager.AuditLog(client.ID, "client_created", map[string]interface{}{
			"client_name": client.Name,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// clientHandler handles individual client operations
func (s *SecAutoServer) clientHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract client ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/clients/")
	if path == "" {
		http.Error(w, "Client ID required", http.StatusBadRequest)
		return
	}

	// Handle sub-paths like /clients/{id}/integrations
	parts := strings.Split(path, "/")
	clientID := parts[0]

	// Route to client integration handlers
	if len(parts) > 1 && parts[1] == "integrations" {
		if len(parts) == 2 {
			// /clients/{id}/integrations - list integrations
			s.clientIntegrationsHandler(w, r)
			return
		} else if len(parts) == 4 && parts[3] == "config" {
			// /clients/{id}/integrations/{name}/config
			s.clientIntegrationConfigHandler(w, r)
			return
		} else if len(parts) == 4 && parts[3] == "execute" {
			// /clients/{id}/integrations/{name}/execute
			s.clientIntegrationExecuteHandler(w, r)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		// Get specific client
		client, err := s.clientManager.GetClient(clientID)
		if err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   err.Error(),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Client retrieved successfully",
			"client":    client,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

	case http.MethodPost:
		
		// No other POST operations for individual clients
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

	case http.MethodPut:
		// Update client
		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   "Invalid JSON format",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := s.clientManager.UpdateClient(clientID, updates); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to update client: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Client updated successfully",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.clientManager.AuditLog(clientID, "client_updated", updates)

	case http.MethodDelete:
		// Delete client
		if err := s.clientManager.DeleteClient(clientID); err != nil {
			response := map[string]interface{}{
				"success":   false,
				"message":   fmt.Sprintf("Failed to delete client: %v", err),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"success":   true,
			"message":   "Client deleted successfully",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)

		s.clientManager.AuditLog(clientID, "client_deleted", nil)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}


// Run starts the modular server
func (s *SecAutoServer) Run() error {
	// Setup routes
	http.HandleFunc("/health", s.publicMiddleware(s.healthHandler))  // CORS only for health check
	http.HandleFunc("/playbook", s.middleware(s.playbookHandler))
	
	// Enhanced playbook endpoints
	http.HandleFunc("/playbook/async", s.middleware(s.playbookAsyncHandler))
	http.HandleFunc("/playbook/upload", s.middleware(s.playbookUploadHandler))
	http.HandleFunc("/playbook/", s.middleware(s.playbookDeleteHandler))
	http.HandleFunc("/playbooks", s.middleware(s.playbooksHandler))
	
	// Cache endpoints
	http.HandleFunc("/cache", s.middleware(s.cacheHandler))
	http.HandleFunc("/cache/stats", s.middleware(s.cacheStatsHandler))
	http.HandleFunc("/cache/clear", s.middleware(s.cacheClearHandler))
	http.HandleFunc("/cache/", s.middleware(s.cacheKeyHandler))
	
	// Redis list endpoints
	http.HandleFunc("/lists/", s.middleware(s.listHandler))
	
	// Integration management endpoints
	http.HandleFunc("/integrations", s.middleware(s.integrationsHandler))
	http.HandleFunc("/integrations/", s.middleware(s.integrationHandler))
	http.HandleFunc("/integrations/upload", s.middleware(s.integrationUploadHandler))
	http.HandleFunc("/integrations/build-status/", s.middleware(s.integrationBuildStatusHandler))
	
	// Cluster management endpoints
	http.HandleFunc("/cluster", s.middleware(s.clusterHandler))
	http.HandleFunc("/cluster/jobs", s.middleware(s.clusterJobsHandler))
	http.HandleFunc("/cluster/jobs/", s.middleware(s.clusterJobHandler))
	
	// Automation management endpoints
	http.HandleFunc("/automation", s.middleware(s.automationUploadHandler))
	http.HandleFunc("/automations", s.middleware(s.automationsListHandler))
	http.HandleFunc("/automation/", s.middleware(s.automationDeleteHandler))
	http.HandleFunc("/automation/metadata", s.middleware(s.automationMetadataHandler))
	http.HandleFunc("/automation/metadata/", s.middleware(s.automationMetadataItemHandler))
	
	// Job management endpoints
	http.HandleFunc("/jobs", s.middleware(s.jobsHandler))
	http.HandleFunc("/jobs/stats", s.middleware(s.jobsStatsHandler))
	http.HandleFunc("/job/", s.middleware(s.jobHandler))
	
	// Schedule management endpoints
	http.HandleFunc("/schedules", s.middleware(s.schedulesHandler))
	http.HandleFunc("/schedules/stats", s.middleware(s.scheduleStatsHandler))
	http.HandleFunc("/schedule/", s.middleware(s.scheduleHandler))
	http.HandleFunc("/schedule/execute/", s.middleware(s.scheduleExecuteHandler))
	
	// API key management endpoints
	http.HandleFunc("/api-keys", s.middleware(s.apiKeysHandler))
	http.HandleFunc("/api-keys/stats", s.middleware(s.apiKeyStatsHandler))
	
	// Client management endpoints
	http.HandleFunc("/clients", s.middleware(s.clientsHandler))
	http.HandleFunc("/clients/", s.middleware(s.clientHandler))
	
	// Note: Client integration endpoints are handled within the clientHandler using path routing
	
	// Swagger/API documentation endpoints
	http.HandleFunc("/docs", s.publicMiddleware(s.swagger.ServeHTTP))
	http.HandleFunc("/docs/", s.publicMiddleware(s.swagger.ServeHTTP))
	http.HandleFunc("/api-docs", s.publicMiddleware(s.swagger.ServeHTTP))

	// Start servers based on TLS configuration
	if s.config.Security.TLS.Enabled {
		return s.startHTTPSServer()
	} else {
		return s.startHTTPServer()
	}
}

// startHTTPServer starts the HTTP server
func (s *SecAutoServer) startHTTPServer() error {
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
		Handler:      nil,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting HTTP server", map[string]interface{}{
		"component": "server",
		"host":      s.config.Server.Host,
		"port":      s.config.Server.Port,
		"version":   "modular-1.0.0",
		"tls":       false,
	})

	return server.ListenAndServe()
}

// startHTTPSServer starts the HTTPS server and optionally HTTP redirect server
func (s *SecAutoServer) startHTTPSServer() error {
	// Get TLS configuration
	tlsConfig, err := s.tlsManager.GetTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to get TLS config: %v", err)
	}

	// Create HTTPS server
	httpsServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Security.TLS.Port),
		Handler:      nil,
		TLSConfig:    tlsConfig,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP redirect server if auto-redirect is enabled
	if s.config.Security.TLS.AutoRedirect {
		go s.startHTTPRedirectServer()
	}

	// Start autocert HTTP handler if using autocert
	if s.config.Security.TLS.AutoCert.Enabled {
		go s.startAutocertHTTPServer()
	}

	s.logger.Info("Starting HTTPS server", map[string]interface{}{
		"component":     "server",
		"host":          s.config.Server.Host,
		"https_port":    s.config.Security.TLS.Port,
		"version":       "modular-1.0.0",
		"tls":           true,
		"auto_redirect": s.config.Security.TLS.AutoRedirect,
		"autocert":      s.config.Security.TLS.AutoCert.Enabled,
	})

	// Start HTTPS server
	if s.config.Security.TLS.AutoCert.Enabled {
		return httpsServer.ListenAndServeTLS("", "")
	} else {
		return httpsServer.ListenAndServeTLS(s.config.Security.TLS.CertFile, s.config.Security.TLS.KeyFile)
	}
}

// startHTTPRedirectServer starts a server that redirects HTTP to HTTPS
func (s *SecAutoServer) startHTTPRedirectServer() {
	redirectServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
		Handler: http.HandlerFunc(s.tlsManager.GetHTTPSRedirectHandler()),
	}

	s.logger.Info("Starting HTTP redirect server", map[string]interface{}{
		"component":  "server",
		"http_port":  s.config.Server.Port,
		"https_port": s.config.Security.TLS.Port,
	})

	if err := redirectServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("HTTP redirect server failed", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
	}
}

// startAutocertHTTPServer starts the HTTP server for ACME challenges
func (s *SecAutoServer) startAutocertHTTPServer() {
	autocertHandler := s.tlsManager.GetAutocertHandler()
	if autocertHandler == nil {
		return
	}

	autocertServer := &http.Server{
		Addr:    fmt.Sprintf("%s:80", s.config.Server.Host),
		Handler: autocertHandler,
	}

	s.logger.Info("Starting autocert HTTP server", map[string]interface{}{
		"component": "server",
		"port":      80,
	})

	if err := autocertServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("Autocert HTTP server failed", map[string]interface{}{
			"component": "server",
			"error":     err.Error(),
		})
	}
}

func main() {
	server, err := NewSecAutoServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	fmt.Println("🚀 SecAuto Modular Server Starting...")
	fmt.Printf("📁 Modules loaded: config, logger, validator, rules, cache, types, swagger, integrations, cluster\n")
	
	// Display server information based on TLS configuration
	if server.config.Security.TLS.Enabled {
		fmt.Printf("🔒 HTTPS Server will start on %s:%d\n", server.config.Server.Host, server.config.Security.TLS.Port)
		fmt.Printf("🏥 Health endpoint: https://%s:%d/health\n", server.config.Server.Host, server.config.Security.TLS.Port)
		fmt.Printf("📋 Playbook execution endpoint: https://%s:%d/playbook\n", server.config.Server.Host, server.config.Security.TLS.Port)
		fmt.Printf("📚 API Documentation: https://%s:%d/docs\n", server.config.Server.Host, server.config.Security.TLS.Port)
		
		if server.config.Security.TLS.AutoRedirect {
			fmt.Printf("🔄 HTTP redirect server on %s:%d (redirects to HTTPS)\n", server.config.Server.Host, server.config.Server.Port)
		}
		
		if server.config.Security.TLS.AutoCert.Enabled {
			fmt.Printf("🤖 Automatic certificates enabled for domains: %v\n", server.config.Security.TLS.AutoCert.Domains)
		} else {
			fmt.Printf("📜 Using certificate: %s\n", server.config.Security.TLS.CertFile)
		}
	} else {
		fmt.Printf("🌐 HTTP Server will start on %s:%d\n", server.config.Server.Host, server.config.Server.Port)
		fmt.Printf("🏥 Health endpoint: http://%s:%d/health\n", server.config.Server.Host, server.config.Server.Port)
		fmt.Printf("📋 Playbook execution endpoint: http://%s:%d/playbook\n", server.config.Server.Host, server.config.Server.Port)
		fmt.Printf("📚 API Documentation: http://%s:%d/docs\n", server.config.Server.Host, server.config.Server.Port)
	}
	
	fmt.Printf("🔧 Cache & List endpoints available\n")
	fmt.Printf("🔗 Integration management endpoints available\n")
	fmt.Printf("🌐 Cluster management endpoints available\n")
	fmt.Printf("⏰ Schedule management endpoints available\n")
	fmt.Printf("🔐 API key authentication enabled\n")
	fmt.Printf("⚡ Rules engine active with caching enabled\n")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Run(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n🔄 Shutting down server gracefully...")
	
	// Save API keys before shutdown
	if err := server.apiKeyManager.Shutdown(); err != nil {
		log.Printf("Failed to save API keys: %v", err)
	} else {
		fmt.Println("💾 API keys saved successfully")
	}

	fmt.Println("✅ Server shut down complete")
}