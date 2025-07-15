package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
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
	Level            string         `yaml:"level"`
	Destination      string         `yaml:"destination"`
	File             string         `yaml:"file"`
	Rotation         RotationConfig `yaml:"rotation"`
	Format           string         `yaml:"format"`
	IncludeTimestamp bool           `yaml:"include_timestamp"`
	IncludeComponent bool           `yaml:"include_component"`
	IncludeRequestID bool           `yaml:"include_request_id"`
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
}

// Note: Removed unused database configuration structs after implementing Redis job store
// The following settings are now handled internally by the Redis job store implementation:
// - Connection pooling (handled by Redis client)
// - Optimization settings (handled by Redis implementation)
// - Backup settings (handled by job store implementation)
// - Retention settings (handled by job store implementation)

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

// PluginsConfig holds plugin system configuration
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
	Enabled    bool   `yaml:"enabled"`
	CertFile   string `yaml:"cert_file"`
	KeyFile    string `yaml:"key_file"`
	MinVersion string `yaml:"min_version"`
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
	MaxNestingDepth        int  `yaml:"max_nesting_depth"`
	MaxConditionsPerRule   int  `yaml:"max_conditions_per_rule"`
	MaxVariablesPerContext int  `yaml:"max_variables_per_context"`
	EnableDebugMode        bool `yaml:"enable_debug_mode"`
	StrictMode             bool `yaml:"strict_mode"`
	AllowCustomFunctions   bool `yaml:"allow_custom_functions"`
	MaxExecutionTime       int  `yaml:"max_execution_time"`
	MemoryLimit            int  `yaml:"memory_limit"`
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
			Port:           8080,
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
		Cluster: ClusterConfig{
			Enabled:             false,
			RedisURL:            "localhost:6379",
			RedisDB:             0,
			RedisPassword:       "",
			RedisPoolSize:       10,
			RedisPoolTimeout:    "30s",
			RedisIdleTimeout:    "5m",
			NodeID:              "",
			ClusterName:         "secauto_cluster",
			HeartbeatInterval:   30,
			ElectionTimeout:     5000,
			JobTimeout:          3600,
			MaxRetries:          3,
			RetryDelay:          5,
			LoadBalancing:       "round_robin",
			HealthCheckInterval: 60,
			FailoverEnabled:     true,
		},
		Scheduler: SchedulerConfig{
			Enabled:                 true,
			MaxConcurrentJobs:       10,
			CleanupInterval:         3600,
			RetentionDays:           30,
			Timezone:                "UTC",
			DefaultTimeout:          3600,
			MaxSchedules:            1000,
			ScheduleCleanupInterval: 86400,
			FailedJobRetention:      7,
			SuccessfulJobRetention:  30,
		},
		Plugins: PluginsConfig{
			Enabled:          true,
			Directory:        "../plugins",
			HotReload:        true,
			ReloadInterval:   30,
			SupportedTypes:   []string{"automation", "playbook", "integration", "validator"},
			MaxPlugins:       100,
			PluginTimeout:    300,
			SandboxMode:      false,
			AllowExecutables: true,
			AllowPython:      true,
			AllowGoPlugins:   true,
			PluginValidation: true,
			PluginLogging:    true,
			Platforms: map[string]PlatformConfig{
				"windows": {
					Enabled:             true,
					Directory:           "../plugins/windows",
					SupportedExtensions: []string{".exe", ".py"},
					Timeout:             300,
					SandboxMode:         true,
					MaxMemory:           512,
					AllowNetworkAccess:  true,
					AllowFileAccess:     true,
				},
				"linux": {
					Enabled:             true,
					Directory:           "../plugins/linux",
					SupportedExtensions: []string{".so", ".py", ".go"},
					Timeout:             300,
					SandboxMode:         false,
					MaxMemory:           1024,
					AllowNetworkAccess:  true,
					AllowFileAccess:     true,
				},
				"python": {
					Enabled:             true,
					Directory:           "../plugins/python",
					SupportedExtensions: []string{".py"},
					Timeout:             300,
					SandboxMode:         false,
					Interpreter:         "python3",
					MaxMemory:           512,
					AllowNetworkAccess:  true,
					AllowFileAccess:     true,
				},
				"go": {
					Enabled:             true,
					Directory:           "../plugins/go",
					SupportedExtensions: []string{".so", ".exe", ".go"},
					Timeout:             300,
					SandboxMode:         false,
					BuildMode:           "plugin",
					MaxMemory:           1024,
					AllowNetworkAccess:  true,
					AllowFileAccess:     true,
				},
			},
		},
		Security: SecurityConfig{
			APIKeys: []string{"your-secure-api-key-here", "backup-api-key"},
			RateLimiting: RateLimitingConfig{
				Enabled:           true,
				RequestsPerMinute: 100,
				BurstLimit:        200,
				WindowSize:        "1m",
				Endpoints: EndpointRateLimits{
					Health:        1000, // Unlimited for health checks
					Playbook:      16,   // 1/6 of total
					PlaybookAsync: 66,   // 2/3 of total (generous for async)
					Jobs:          25,   // 1/4 of total
					JobStatus:     25,   // 1/4 of total
					JobStats:      25,   // 1/4 of total
					JobMetrics:    25,   // 1/4 of total
					Context:       25,   // 1/4 of total
					Webhooks:      20,   // 1/5 of total
					Plugins:       20,   // 1/5 of total
					Cluster:       20,   // 1/5 of total
					Schedules:     20,   // 1/5 of total
					Default:       100,  // Default for all other endpoints
				},
			},
			InputValidation: InputValidationConfig{
				Enabled:                 true,
				MaxContextSize:          100,
				MaxPlaybookSize:         1000,
				MaxScriptSize:           500,
				AllowedScriptExtensions: []string{".py", ".ps1", ".bat"},
				SanitizeInputs:          true,
			},
			CORS: CORSConfig{
				Enabled:        false,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
				AllowedHeaders: []string{"Content-Type", "X-API-Key"},
				MaxAge:         86400,
			},
			TLS: TLSConfig{
				Enabled:    false,
				CertFile:   "certs/server.crt",
				KeyFile:    "certs/server.key",
				MinVersion: "1.2",
			},
		},
		Webhooks: WebhooksConfig{
			Enabled:     true,
			Timeout:     30,
			RetryCount:  3,
			RetryDelay:  5,
			MaxWebhooks: 50,
			Events:      []string{"job_started", "job_completed", "job_failed", "job_cancelled", "schedule_created", "schedule_updated", "schedule_deleted"},
			DefaultHeaders: map[string]string{
				"Content-Type": "application/json",
				"User-Agent":   "SecAuto-Webhook/1.0",
			},
		},
		Python: PythonConfig{
			VenvPath:           "../Venv",
			ScriptsPath:        "../automations",
			PlaybooksPath:      "../playbooks",
			DefaultScript:      "baseit.py",
			ScriptTimeout:      300,
			MaxScriptMemory:    512,
			AllowNetworkAccess: true,
			AllowFileAccess:    true,
			SandboxMode:        false,
			HotReload:          true,
			ScriptValidation:   true,
		},
		RulesEngine: RulesEngineConfig{
			MaxNestingDepth:        10,
			MaxConditionsPerRule:   50,
			MaxVariablesPerContext: 1000,
			EnableDebugMode:        false,
			StrictMode:             true,
			AllowCustomFunctions:   false,
			MaxExecutionTime:       3600,
			MemoryLimit:            1024,
		},
		Monitoring: MonitoringConfig{
			Enabled:             true,
			MetricsInterval:     60,
			HealthCheckInterval: 30,
			PerformanceTracking: true,
			SlowQueryThreshold:  100,
			MemoryUsageTracking: true,
			CPUUsageTracking:    true,
			DiskUsageTracking:   true,
			CustomMetrics:       true,
		},
		Performance: PerformanceConfig{
			WorkerPoolSize:        5,
			JobQueueSize:          1000,
			ContextCacheSize:      1000,
			PluginCacheSize:       100,
			ScheduleCacheSize:     500,
			WebhookQueueSize:      100,
			MaxConcurrentRequests: 100,
			RequestTimeout:        300,
			EnableCompression:     true,
			EnableCaching:         true,
			CacheTTL:              3600,
		},
		Development: DevelopmentConfig{
			DebugMode:            false,
			VerboseLogging:       false,
			HotReloadEnabled:     true,
			AutoRestart:          false,
			ProfileEnabled:       false,
			TraceEnabled:         false,
			MockExternalServices: false,
			TestMode:             false,
		},
		Backup: BackupConfig{
			Enabled:        true,
			Schedule:       "0 2 * * *",
			RetentionDays:  30,
			Compression:    true,
			Encryption:     false,
			EncryptionKey:  "",
			BackupLocation: "backups/",
			IncludeLogs:    true,
			IncludeConfig:  true,
			IncludePlugins: true,
			MaxBackupSize:  1024,
		},
		Notifications: NotificationsConfig{
			Email: EmailConfig{
				Enabled:     false,
				SMTPServer:  "smtp.gmail.com",
				SMTPPort:    587,
				Username:    "",
				Password:    "",
				FromAddress: "",
				ToAddresses: []string{},
			},
			Slack: SlackConfig{
				Enabled:    false,
				WebhookURL: "",
				Channel:    "",
				Username:   "SecAuto",
			},
			Teams: TeamsConfig{
				Enabled:    false,
				WebhookURL: "",
			},
			Discord: DiscordConfig{
				Enabled:    false,
				WebhookURL: "",
			},
		},
		Integrations: IntegrationsConfig{
			ExternalAPIs: ExternalAPIsConfig{
				MaxConnections: 10,
				Timeout:        30,
				RetryCount:     3,
				RetryDelay:     5,
			},
			FileSystems: FileSystemsConfig{
				AllowedPaths:      []string{"../automations", "../playbooks", "data"},
				MaxFileSize:       10485760,
				AllowedExtensions: []string{".py", ".json", ".yaml", ".yml"},
			},
			Network: NetworkConfig{
				AllowedHosts:      []string{"*"},
				BlockedHosts:      []string{},
				MaxConnections:    100,
				ConnectionTimeout: 30,
			},
		},
	}

	// Try to read config.yaml
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("[WARN] Could not read %s: %v. Using defaults.", configPath, err)
		return defaults, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Printf("[WARN] Could not parse %s: %v. Using defaults.", configPath, err)
		return defaults, nil
	}

	// Merge with defaults for missing fields
	mergeConfig(&cfg, defaults)

	// Override with environment variables if present
	overrideFromEnv(&cfg)

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
