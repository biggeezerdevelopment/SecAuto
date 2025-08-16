package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	
	"SoarAuto/pkg/errors"
)

// Config holds the application configuration
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Logging       LoggingConfig       `yaml:"logging"`
	Database      DatabaseConfig      `yaml:"database"`
	Cluster       ClusterConfig       `yaml:"cluster"`
	Scheduler     SchedulerConfig     `yaml:"scheduler"`
	Plugins       PluginsConfig       `yaml:"plugins"`
	Security      SecurityConfig      `yaml:"security"`
	Webhooks      WebhooksConfig      `yaml:"webhooks"`
	Python        PythonConfig        `yaml:"python"`
	RulesEngine   RulesEngineConfig   `yaml:"rules_engine"`
	Monitoring    MonitoringConfig    `yaml:"monitoring"`
	Performance   PerformanceConfig   `yaml:"performance"`
	Development   DevelopmentConfig   `yaml:"development"`
	Backup        BackupConfig        `yaml:"backup"`
	Notifications NotificationsConfig `yaml:"notifications"`
	Integrations  IntegrationsConfig  `yaml:"integrations"`
	Environments  map[string]Config   `yaml:"environments"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port           int    `yaml:"port"`
	Host           string `yaml:"host"`
	Workers        int    `yaml:"workers"`
	ReadTimeout    string `yaml:"read_timeout"`
	WriteTimeout   string `yaml:"write_timeout"`
	IdleTimeout    string `yaml:"idle_timeout"`
	MaxHeaderBytes int    `yaml:"max_header_bytes"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level            string                    `yaml:"level"`
	Destination      string                    `yaml:"destination"`
	File             string                    `yaml:"file"`
	Rotation         RotationConfig            `yaml:"rotation"`
	Format           string                    `yaml:"format"`
	IncludeTimestamp bool                      `yaml:"include_timestamp"`
	IncludeComponent bool                      `yaml:"include_component"`
	IncludeRequestID bool                      `yaml:"include_request_id"`
	ComponentLevels  map[string]string         `yaml:"component_levels"`
	Performance      LoggingPerformanceConfig  `yaml:"performance"`
}

// LoggingPerformanceConfig holds performance-related logging settings
type LoggingPerformanceConfig struct {
	SkipContextLogging   bool `yaml:"skip_context_logging"`
	MaxLogFieldLength    int  `yaml:"max_log_field_length"`
	BatchLogging         bool `yaml:"batch_logging"`
	AsyncLogging         bool `yaml:"async_logging"`
}

// RotationConfig holds log rotation configuration
type RotationConfig struct {
	MaxSizeMB  int  `yaml:"max_size_mb"`
	MaxBackups int  `yaml:"max_backups"`
	MaxAgeDays int  `yaml:"max_age_days"`
	Compress   bool `yaml:"compress"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	RedisURL string `yaml:"redis_url"` // Redis connection URL
	
	// TTL settings for different types of data (in seconds)
	CacheTTL      int `yaml:"cache_ttl"`       // Default TTL for cache entries (0 = no expiration)
	JobTTL        int `yaml:"job_ttl"`         // TTL for completed/failed jobs (0 = no expiration)
	TempDataTTL   int `yaml:"temp_data_ttl"`   // TTL for temporary data (0 = no expiration)
}

// ClusterConfig holds distributed cluster configuration
type ClusterConfig struct {
	Enabled             bool   `yaml:"enabled"`
	RedisURL            string `yaml:"redis_url"`
	RedisDB             int    `yaml:"redis_db"`
	RedisPassword       string `yaml:"redis_password"`
	RedisPoolSize       int    `yaml:"redis_pool_size"`
	RedisPoolTimeout    string `yaml:"redis_pool_timeout"`
	RedisIdleTimeout    string `yaml:"redis_idle_timeout"`
	NodeID              string `yaml:"node_id"`
	ClusterName         string `yaml:"cluster_name"`
	HeartbeatInterval   int    `yaml:"heartbeat_interval"`
	ElectionTimeout     int    `yaml:"election_timeout"`
	JobTimeout          int    `yaml:"job_timeout"`
	JobStorageTTL       int    `yaml:"job_storage_ttl"`      // TTL for job storage in Redis (seconds, 0 = no expiration)
	RunningJobTTL       int    `yaml:"running_job_ttl"`      // TTL for running jobs (seconds, 0 = no expiration)
	CompletedJobTTL     int    `yaml:"completed_job_ttl"`    // TTL for completed jobs (seconds, 0 = no expiration)
	FailedJobTTL        int    `yaml:"failed_job_ttl"`       // TTL for failed jobs (seconds, 0 = no expiration)
	MaxRetries          int    `yaml:"max_retries"`
	RetryDelay          int    `yaml:"retry_delay"`
	LoadBalancing       string `yaml:"load_balancing"`
	HealthCheckInterval int    `yaml:"health_check_interval"`
	FailoverEnabled     bool   `yaml:"failover_enabled"`
}

// SchedulerConfig holds job scheduler configuration
type SchedulerConfig struct {
	Enabled                 bool   `yaml:"enabled"`
	MaxConcurrentJobs       int    `yaml:"max_concurrent_jobs"`
	CleanupInterval         int    `yaml:"cleanup_interval"`
	RetentionDays           int    `yaml:"retention_days"`
	Timezone                string `yaml:"timezone"`
	DefaultTimeout          int    `yaml:"default_timeout"`
	MaxSchedules            int    `yaml:"max_schedules"`
	ScheduleCleanupInterval int    `yaml:"schedule_cleanup_interval"`
	FailedJobRetention      int    `yaml:"failed_job_retention"`
	SuccessfulJobRetention  int    `yaml:"successful_job_retention"`
}

// PlatformConfig holds platform-specific plugin configuration
type PlatformConfig struct {
	Enabled             bool     `yaml:"enabled"`
	Directory           string   `yaml:"directory"`
	SupportedExtensions []string `yaml:"supported_extensions"`
	Timeout             int      `yaml:"timeout"`
	SandboxMode         bool     `yaml:"sandbox_mode"`
	Interpreter         string   `yaml:"interpreter,omitempty"`
	BuildMode           string   `yaml:"build_mode,omitempty"`
	MaxMemory           int      `yaml:"max_memory,omitempty"`
	AllowNetworkAccess  bool     `yaml:"allow_network_access,omitempty"`
	AllowFileAccess     bool     `yaml:"allow_file_access,omitempty"`
}

// PlatformInfo holds platform-specific metadata
type PlatformInfo struct {
	OS           string            `json:"os"`           // "windows", "linux", "darwin"
	Architecture string            `json:"architecture"` // "amd64", "arm64", "386"
	Dependencies []string          `json:"dependencies,omitempty"`
	Requirements map[string]string `json:"requirements,omitempty"`
}

type PluginsConfig struct {
	Enabled          bool     `yaml:"enabled"`
	Directory        string   `yaml:"directory"`
	HotReload        bool     `yaml:"hot_reload"`
	ReloadInterval   int      `yaml:"reload_interval"`
	SupportedTypes   []string `yaml:"supported_types"`
	MaxPlugins       int      `yaml:"max_plugins"`
	PluginTimeout    int      `yaml:"plugin_timeout"`
	SandboxMode      bool     `yaml:"sandbox_mode"`
	AllowExecutables bool     `yaml:"allow_executables"`
	AllowPython      bool     `yaml:"allow_python"`
	AllowGoPlugins   bool     `yaml:"allow_go_plugins"`
	PluginValidation bool     `yaml:"plugin_validation"`
	PluginLogging    bool     `yaml:"plugin_logging"`

	// Platform-specific configurations
	Platforms map[string]PlatformConfig `yaml:"platforms"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	APIKeys                  []string              `yaml:"api_keys"`
	APIKeysFile              string                `yaml:"api_keys_file"`
	IntegrationEncryptionKey string                `yaml:"integration_encryption_key"`
	RateLimiting             RateLimitingConfig    `yaml:"rate_limiting"`
	InputValidation          InputValidationConfig `yaml:"input_validation"`
	CORS                     CORSConfig            `yaml:"cors"`
	TLS                      TLSConfig             `yaml:"tls"`
}

// RateLimitingConfig holds rate limiting settings
type RateLimitingConfig struct {
	Enabled           bool   `yaml:"enabled"`
	RequestsPerMinute int    `yaml:"requests_per_minute"`
	BurstLimit        int    `yaml:"burst_limit"`
	WindowSize        string `yaml:"window_size"`

	// Endpoint-specific rate limits
	Endpoints EndpointRateLimits `yaml:"endpoints"`
}

// EndpointRateLimits holds rate limits for specific endpoints
type EndpointRateLimits struct {
	Health        int `yaml:"health"`         // /health endpoint
	Playbook      int `yaml:"playbook"`       // /playbook endpoint
	PlaybookAsync int `yaml:"playbook_async"` // /playbook/async endpoint
	Jobs          int `yaml:"jobs"`           // /jobs endpoint
	JobStatus     int `yaml:"job_status"`     // /job/{id} endpoint
	JobStats      int `yaml:"job_stats"`      // /jobs/stats endpoint
	JobMetrics    int `yaml:"job_metrics"`    // /jobs/metrics endpoint
	Context       int `yaml:"context"`        // /context endpoint
	Webhooks      int `yaml:"webhooks"`       // /webhooks endpoint
	Plugins       int `yaml:"plugins"`        // /plugins endpoints
	Cluster       int `yaml:"cluster"`        // /cluster endpoints
	Schedules     int `yaml:"schedules"`      // /schedules endpoints
	Default       int `yaml:"default"`        // Default for all other endpoints
}

// InputValidationConfig holds input validation settings
type InputValidationConfig struct {
	Enabled                 bool     `yaml:"enabled"`
	MaxContextSize          int      `yaml:"max_context_size"`
	MaxPlaybookSize         int      `yaml:"max_playbook_size"`
	MaxScriptSize           int      `yaml:"max_script_size"`
	AllowedScriptExtensions []string `yaml:"allowed_script_extensions"`
	SanitizeInputs          bool     `yaml:"sanitize_inputs"`
}

// CORSConfig holds CORS settings
type CORSConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
	MaxAge         int      `yaml:"max_age"`
}

// TLSConfig holds TLS settings
type TLSConfig struct {
	Enabled      bool         `yaml:"enabled"`
	Port         int          `yaml:"port"`
	CertFile     string       `yaml:"cert_file"`
	KeyFile      string       `yaml:"key_file"`
	AutoRedirect bool         `yaml:"auto_redirect"`
	MinVersion   string       `yaml:"min_version"`
	MaxVersion   string       `yaml:"max_version"`
	CipherSuites []string     `yaml:"cipher_suites"`
	AutoCert     AutoCertConfig `yaml:"auto_cert"`
	ClientAuth   ClientAuthConfig `yaml:"client_auth"`
}

// AutoCertConfig holds automatic certificate management settings
type AutoCertConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Domains  []string `yaml:"domains"`
	CacheDir string   `yaml:"cache_dir"`
	Email    string   `yaml:"email"`
}

// ClientAuthConfig holds client certificate authentication settings
type ClientAuthConfig struct {
	Enabled     bool   `yaml:"enabled"`
	CAFile      string `yaml:"ca_file"`
	RequireCert bool   `yaml:"require_cert"`
}

// WebhooksConfig holds webhook configuration
type WebhooksConfig struct {
	Enabled        bool              `yaml:"enabled"`
	Timeout        int               `yaml:"timeout"`
	RetryCount     int               `yaml:"retry_count"`
	RetryDelay     int               `yaml:"retry_delay"`
	MaxWebhooks    int               `yaml:"max_webhooks"`
	Events         []string          `yaml:"events"`
	DefaultHeaders map[string]string `yaml:"default_headers"`
}

// PythonConfig holds Python integration configuration
type PythonConfig struct {
	VenvPath           string `yaml:"venv_path"`
	ScriptsPath        string `yaml:"scripts_path"`
	PlaybooksPath      string `yaml:"playbooks_path"`
	DefaultScript      string `yaml:"default_script"`
	ScriptTimeout      int    `yaml:"script_timeout"`
	MaxScriptMemory    int    `yaml:"max_script_memory"`
	AllowNetworkAccess bool   `yaml:"allow_network_access"`
	AllowFileAccess    bool   `yaml:"allow_file_access"`
	SandboxMode        bool   `yaml:"sandbox_mode"`
	HotReload          bool   `yaml:"hot_reload"`
	ScriptValidation   bool   `yaml:"script_validation"`
}

// RulesEngineConfig holds rules engine configuration
type RulesEngineConfig struct {
	MaxNestingDepth        int                     `yaml:"max_nesting_depth"`
	MaxConditionsPerRule   int                     `yaml:"max_conditions_per_rule"`
	MaxVariablesPerContext int                     `yaml:"max_variables_per_context"`
	EnableDebugMode        bool                    `yaml:"enable_debug_mode"`
	StrictMode             bool                    `yaml:"strict_mode"`
	AllowCustomFunctions   bool                    `yaml:"allow_custom_functions"`
	MaxExecutionTime       int                     `yaml:"max_execution_time"`
	MemoryLimit            int                     `yaml:"memory_limit"`
	Caching                RulesEngineCacheConfig  `yaml:"caching"`
}

// RulesEngineCacheConfig holds caching configuration for the rules engine
type RulesEngineCacheConfig struct {
	MaxContexts           int    `yaml:"max_contexts"`
	MaxExpressions        int    `yaml:"max_expressions"`
	MaxVariables          int    `yaml:"max_variables"`
	ContextTTL            string `yaml:"context_ttl"`
	ExpressionTTL         string `yaml:"expression_ttl"`
	VariableTTL           string `yaml:"variable_ttl"`
	CleanupInterval       string `yaml:"cleanup_interval"`
	EnableLazyEval        bool   `yaml:"enable_lazy_eval"`
	EnableExpressionCache bool   `yaml:"enable_expression_cache"`
	MaxFieldSize          int    `yaml:"max_field_size"`
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	Enabled             bool `yaml:"enabled"`
	MetricsInterval     int  `yaml:"metrics_interval"`
	HealthCheckInterval int  `yaml:"health_check_interval"`
	PerformanceTracking bool `yaml:"performance_tracking"`
	SlowQueryThreshold  int  `yaml:"slow_query_threshold"`
	MemoryUsageTracking bool `yaml:"memory_usage_tracking"`
	CPUUsageTracking    bool `yaml:"cpu_usage_tracking"`
	DiskUsageTracking   bool `yaml:"disk_usage_tracking"`
	CustomMetrics       bool `yaml:"custom_metrics"`
}

// PerformanceConfig holds performance configuration
type PerformanceConfig struct {
	WorkerPoolSize        int  `yaml:"worker_pool_size"`
	JobQueueSize          int  `yaml:"job_queue_size"`
	ContextCacheSize      int  `yaml:"context_cache_size"`
	PluginCacheSize       int  `yaml:"plugin_cache_size"`
	ScheduleCacheSize     int  `yaml:"schedule_cache_size"`
	WebhookQueueSize      int  `yaml:"webhook_queue_size"`
	MaxConcurrentRequests int  `yaml:"max_concurrent_requests"`
	RequestTimeout        int  `yaml:"request_timeout"`
	EnableCompression     bool `yaml:"enable_compression"`
	EnableCaching         bool `yaml:"enable_caching"`
	CacheTTL              int  `yaml:"cache_ttl"`
}

// DevelopmentConfig holds development configuration
type DevelopmentConfig struct {
	DebugMode            bool `yaml:"debug_mode"`
	VerboseLogging       bool `yaml:"verbose_logging"`
	HotReloadEnabled     bool `yaml:"hot_reload_enabled"`
	AutoRestart          bool `yaml:"auto_restart"`
	ProfileEnabled       bool `yaml:"profile_enabled"`
	TraceEnabled         bool `yaml:"trace_enabled"`
	MockExternalServices bool `yaml:"mock_external_services"`
	TestMode             bool `yaml:"test_mode"`
}

// BackupConfig holds backup configuration
type BackupConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Schedule       string `yaml:"schedule"`
	RetentionDays  int    `yaml:"retention_days"`
	Compression    bool   `yaml:"compression"`
	Encryption     bool   `yaml:"encryption"`
	EncryptionKey  string `yaml:"encryption_key"`
	BackupLocation string `yaml:"backup_location"`
	IncludeLogs    bool   `yaml:"include_logs"`
	IncludeConfig  bool   `yaml:"include_config"`
	IncludePlugins bool   `yaml:"include_plugins"`
	MaxBackupSize  int    `yaml:"max_backup_size"`
}

// NotificationsConfig holds notification configuration
type NotificationsConfig struct {
	Email   EmailConfig   `yaml:"email"`
	Slack   SlackConfig   `yaml:"slack"`
	Teams   TeamsConfig   `yaml:"teams"`
	Discord DiscordConfig `yaml:"discord"`
}

// EmailConfig holds email notification settings
type EmailConfig struct {
	Enabled     bool     `yaml:"enabled"`
	SMTPServer  string   `yaml:"smtp_server"`
	SMTPPort    int      `yaml:"smtp_port"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
	FromAddress string   `yaml:"from_address"`
	ToAddresses []string `yaml:"to_addresses"`
}

// SlackConfig holds Slack notification settings
type SlackConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
	Channel    string `yaml:"channel"`
	Username   string `yaml:"username"`
}

// TeamsConfig holds Teams notification settings
type TeamsConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
}

// DiscordConfig holds Discord notification settings
type DiscordConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
}

// IntegrationsConfig holds integration configuration
type IntegrationsConfig struct {
	ConfigsPath  string             `yaml:"configs_path"`
	ScriptsPath  string             `yaml:"scripts_path"`
	ExternalAPIs ExternalAPIsConfig `yaml:"external_apis"`
	FileSystems  FileSystemsConfig  `yaml:"file_systems"`
	Network      NetworkConfig      `yaml:"network"`
}

// ExternalAPIsConfig holds external API settings
type ExternalAPIsConfig struct {
	MaxConnections int `yaml:"max_connections"`
	Timeout        int `yaml:"timeout"`
	RetryCount     int `yaml:"retry_count"`
	RetryDelay     int `yaml:"retry_delay"`
}

// FileSystemsConfig holds file system settings
type FileSystemsConfig struct {
	AllowedPaths      []string `yaml:"allowed_paths"`
	MaxFileSize       int      `yaml:"max_file_size"`
	AllowedExtensions []string `yaml:"allowed_extensions"`
}

// NetworkConfig holds network settings
type NetworkConfig struct {
	AllowedHosts      []string `yaml:"allowed_hosts"`
	BlockedHosts      []string `yaml:"blocked_hosts"`
	MaxConnections    int      `yaml:"max_connections"`
	ConnectionTimeout int      `yaml:"connection_timeout"`
}

// LoadConfig loads configuration from config.yaml, falling back to defaults if missing or invalid
func LoadConfig(configPath string) (*Config, error) {
	defaults := &Config{
		Server: ServerConfig{
			Port:           8000,
			Host:           "localhost",
			Workers:        5,
			ReadTimeout:    "30s",
			WriteTimeout:   "30s",
			IdleTimeout:    "60s",
			MaxHeaderBytes: 1048576,
		},
		Logging: LoggingConfig{
			Level:            "INFO",
			Destination:      "both",
			File:             "logs/secauto.log",
			Format:           "json",
			IncludeTimestamp: true,
			IncludeComponent: true,
			IncludeRequestID: true,
			Rotation: RotationConfig{
				MaxSizeMB:  100,
				MaxBackups: 5,
				MaxAgeDays: 30,
				Compress:   true,
			},
		},
		Database: DatabaseConfig{
			RedisURL: "redis://localhost:6379/0",
		},
		Integrations: IntegrationsConfig{
			ConfigsPath: "data/integrations/configs",
			ScriptsPath: "data/integrations/scripts",
		},
		Security: SecurityConfig{
			APIKeys:     []string{"secauto-default-key-2024"},
			APIKeysFile: "data/security/api_keys.json",
		},
		// ... (rest of defaults would be here, abbreviated for brevity)
	}

	// Try to read config.yaml
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("[WARN] Could not read %s: %v. Using defaults.", configPath, err)
		return defaults, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.ConfigError(
			"Failed to parse configuration file",
			err,
		).WithContext("config_path", configPath)
	}

	// Merge with defaults for missing fields
	mergeConfig(&cfg, defaults)

	// Override with environment variables if present
	overrideFromEnv(&cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// mergeConfig merges the loaded config with defaults for missing fields
func mergeConfig(cfg *Config, defaults *Config) {
	// This is a simplified merge - in a real implementation, you'd want to do deep merging
	if cfg.Server.Port == 0 {
		cfg.Server = defaults.Server
	}
	if cfg.Logging.Level == "" {
		cfg.Logging = defaults.Logging
	}

	// Handle database configuration - only Redis is supported
	if cfg.Database.RedisURL == "" {
		cfg.Database.RedisURL = defaults.Database.RedisURL
	}

	if len(cfg.Security.APIKeys) == 0 {
		cfg.Security = defaults.Security
	}
	// Add more merges as needed
}

// overrideFromEnv overrides config values with environment variables
func overrideFromEnv(cfg *Config) {
	if venvPath := os.Getenv("SECAUTO_VENV_PATH"); venvPath != "" {
		cfg.Python.VenvPath = venvPath
	}
	if scriptsPath := os.Getenv("SECAUTO_SCRIPTS_PATH"); scriptsPath != "" {
		cfg.Python.ScriptsPath = scriptsPath
	}
	if playbooksPath := os.Getenv("SECAUTO_PLAYBOOKS_PATH"); playbooksPath != "" {
		cfg.Python.PlaybooksPath = playbooksPath
	}
	if apiKey := os.Getenv("SECAUTO_API_KEY"); apiKey != "" {
		cfg.Security.APIKeys = append(cfg.Security.APIKeys, apiKey)
	}
	if port := os.Getenv("SECAUTO_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}
}

// GetVenvPath returns the full path to the virtual environment
func (c *Config) GetVenvPath() string {
	return c.Python.VenvPath
}

// GetScriptPath returns the full path to a Python script
func (c *Config) GetScriptPath(scriptName string) string {
	if scriptName == "" {
		scriptName = c.Python.DefaultScript
	}

	// If scriptName doesn't end with .py, add it
	if len(scriptName) < 3 || scriptName[len(scriptName)-3:] != ".py" {
		scriptName += ".py"
	}

	return filepath.Join(c.Python.ScriptsPath, scriptName)
}

// GetPlaybookPath returns the full path to a playbook file
func (c *Config) GetPlaybookPath(playbookName string) string {
	if len(playbookName) < 5 || playbookName[len(playbookName)-5:] != ".json" {
		playbookName += ".json"
	}
	return filepath.Join(c.Python.PlaybooksPath, playbookName)
}

// GetDataDirectory returns the default data directory (for compatibility)
func (c *Config) GetDataDirectory() string {
	return "data"
}

// GetAPIKeys returns the list of valid API keys
func (c *Config) GetAPIKeys() []string {
	return c.Security.APIKeys
}

// IsAPIKeyValid checks if the provided API key is valid
func (c *Config) IsAPIKeyValid(key string) bool {
	for _, validKey := range c.Security.APIKeys {
		if validKey == key {
			return true
		}
	}
	return false
}
// 
Validate validates the configuration and returns standardized errors
func (c *Config) Validate() error {
	// Validate server configuration
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.NewErrorBuilder(errors.ErrCodeConfigValidate, "Invalid server port").
			WithComponent("config").
			WithSeverity(errors.SeverityHigh).
			WithContext("port", c.Server.Port).
			WithContext("valid_range", "1-65535").
			Build()
	}

	// Validate Redis URL if provided
	if c.Database.RedisURL != "" {
		if _, err := parseRedisURL(c.Database.RedisURL); err != nil {
			return errors.NewErrorBuilder(errors.ErrCodeConfigValidate, "Invalid Redis URL").
				WithComponent("config").
				WithSeverity(errors.SeverityHigh).
				WithContext("redis_url", c.Database.RedisURL).
				WithContext("parse_error", err.Error()).
				Build()
		}
	}

	// Validate TLS configuration
	if c.Security.TLS.Enabled {
		if !c.Security.TLS.AutoCert.Enabled {
			if c.Security.TLS.CertFile == "" || c.Security.TLS.KeyFile == "" {
				return errors.NewErrorBuilder(errors.ErrCodeConfigValidate, "TLS enabled but no certificates configured").
					WithComponent("config").
					WithSeverity(errors.SeverityHigh).
					WithContext("tls_enabled", true).
					WithContext("autocert_enabled", false).
					WithContext("cert_file", c.Security.TLS.CertFile).
					WithContext("key_file", c.Security.TLS.KeyFile).
					Build()
			}
		}
	}

	// Validate API keys
	if len(c.Security.APIKeys) == 0 {
		return errors.NewErrorBuilder(errors.ErrCodeConfigValidate, "No API keys configured").
			WithComponent("config").
			WithSeverity(errors.SeverityHigh).
			WithContext("api_keys_count", 0).
			Build()
	}

	// Validate logging configuration
	if c.Logging.Level == "" {
		return errors.NewErrorBuilder(errors.ErrCodeConfigValidate, "Logging level not specified").
			WithComponent("config").
			WithSeverity(errors.SeverityMedium).
			Build()
	}

	return nil
}

// Helper function to parse Redis URL (simplified version)
func parseRedisURL(url string) (map[string]string, error) {
	if !strings.HasPrefix(url, "redis://") {
		return nil, fmt.Errorf("invalid Redis URL format")
	}
	return map[string]string{"url": url}, nil
}