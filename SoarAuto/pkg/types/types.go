package types

import (
	"time"
)

// PythonOutput struct to unmarshal JSON output from Python scripts
type PythonOutput struct {
	Context  map[string]interface{} `json:"context"`
	Incident map[string]interface{} `json:"incident"`
}

// JobListResponse represents the response for listing jobs
type JobListResponse struct {
	Success   bool   `json:"success"`
	Jobs      []*Job `json:"jobs"`
	Total     int    `json:"total"`
	Timestamp string `json:"timestamp"`
}

// JobStatsResponse represents job statistics
type JobStatsResponse struct {
	Success     bool    `json:"success"`
	TotalJobs   int     `json:"total_jobs"`
	Completed   int     `json:"completed"`
	Failed      int     `json:"failed"`
	Running     int     `json:"running"`
	Pending     int     `json:"pending"`
	AvgDuration float64 `json:"avg_duration_seconds"`
	RecentJobs  []*Job  `json:"recent_jobs"`
	Timestamp   string  `json:"timestamp"`
}

// CancelJobResponse represents the response for canceling a job
type CancelJobResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	JobID     string `json:"job_id"`
	Timestamp string `json:"timestamp"`
}

// Job represents a job in the system
type Job struct {
	ID          string                 `json:"id" redis:"id"`
	Status      string                 `json:"status" redis:"status"`
	Playbook    interface{}            `json:"playbook" redis:"playbook"`
	Context     map[string]interface{} `json:"context" redis:"context"`
	Results     interface{}            `json:"results,omitempty" redis:"results"`
	Error       string                 `json:"error,omitempty" redis:"error"`
	CreatedAt   *time.Time             `json:"created_at" redis:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty" redis:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty" redis:"completed_at"`
	Priority    int                    `json:"priority" redis:"priority"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" redis:"metadata"`
}

// PlaybookRequest represents a playbook execution request
type PlaybookRequest struct {
	Playbook     interface{}            `json:"playbook,omitempty"`
	PlaybookName string                 `json:"playbook_name,omitempty"`
	Context      map[string]interface{} `json:"context"`
}

// PlaybookResponse represents a playbook execution response
type PlaybookResponse struct {
	Success   bool        `json:"success"`
	Result    interface{} `json:"result,omitempty"`
	JobID     string      `json:"job_id,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationResponse represents validation response
type ValidationResponse struct {
	Success   bool              `json:"success"`
	Valid     bool              `json:"valid"`
	Errors    []ValidationError `json:"errors,omitempty"`
	Message   string            `json:"message,omitempty"`
	Timestamp string            `json:"timestamp"`
}

// WebhookConfig represents webhook configuration
type WebhookConfig struct {
	URL     string            `json:"url"`
	Events  []string          `json:"events"`
	Headers map[string]string `json:"headers,omitempty"`
	Enabled bool              `json:"enabled"`
}

// WebhookEvent represents a webhook event
type WebhookEvent struct {
	Event     string      `json:"event"`
	JobID     string      `json:"job_id,omitempty"`
	Status    string      `json:"status,omitempty"`
	Timestamp string      `json:"timestamp"`
	Playbook  interface{} `json:"playbook,omitempty"`
	Context   interface{} `json:"context,omitempty"`
	Results   interface{} `json:"results,omitempty"`
	Error     string      `json:"error,omitempty"`
	Duration  float64     `json:"duration,omitempty"`
}

// RedisJobStore interface
type RedisJobStore interface {
	StoreJob(job *Job) error
	LoadJob(id string) (*Job, bool)
	UpdateJobStatus(id string, status string) error
	UpdateJobResults(id string, results interface{}, errorMsg string) error
	UpdateJobContext(id string, context map[string]interface{}) error
	ListJobs(status string, limit int) []*Job
	GetJobStats() (map[string]int, error)
	DeleteJob(id string) error
	Close() error
}

// ClusterManager interface
type ClusterManager interface {
	GetClusterInfo() interface{}
	SubmitJob(playbook interface{}, context map[string]interface{}) (string, error)
	GetJob(jobID string) (*Job, error)
}

// PlatformPluginManager interface  
type PlatformPluginManager interface {
	ExecutePlugin(pluginName string, data map[string]interface{}) (interface{}, error)
}

// IntegrationConfig represents integration configuration
type IntegrationConfig struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
	Credentials map[string]interface{} `json:"credentials,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

// AutomationInfo represents information about an automation
type AutomationInfo struct {
	Name          string `json:"name"`
	Filename      string `json:"filename"`
	Size          int64  `json:"size"`
	FileType      string `json:"file_type"`
	Language      string `json:"language"`
	LineCount     int    `json:"line_count"`
	FunctionCount int    `json:"function_count"`
	ImportCount   int    `json:"import_count"`
	ModifiedAt    string `json:"modified_at"`
	IsValid       bool   `json:"is_valid"`
}

// AutomationListResponse represents the response for listing automations
type AutomationListResponse struct {
	Success     bool              `json:"success"`
	Message     string            `json:"message"`
	Automations []AutomationInfo  `json:"automations"`
	Count       int               `json:"count"`
	Timestamp   string            `json:"timestamp"`
}

// JobMetricsResponse represents job metrics
type JobMetricsResponse struct {
	Success    bool                   `json:"success"`
	Metrics    map[string]interface{} `json:"metrics"`
	Timestamp  string                 `json:"timestamp"`
}

// RedisIntegration represents Redis integration interface
type RedisIntegration interface {
	GetCache(key string) map[string]interface{}
	SetCache(key string, value interface{}) map[string]interface{}
	DeleteCache(key string) map[string]interface{}
	GetList(listName string) map[string]interface{}
	AddToList(listName string, items []interface{}, position string) map[string]interface{}
	RemoveFromList(listName string, items []interface{}, count int) map[string]interface{}
	DeleteList(listName string) map[string]interface{}
	Close() error
}

// RateLimiter interface
type RateLimiter interface {
	Allow(endpoint string) bool
}

// Logger interface
type Logger interface {
	Debug(message string, fields map[string]interface{})
	Info(message string, fields map[string]interface{})
	Warning(message string, fields map[string]interface{})
	Error(message string, fields map[string]interface{})
	IsLogLevelEnabled(level, component string) bool
}

// ContextCache interface
type ContextCache interface {
	GetContext(hash string) (*CachedContext, bool)
	StoreContext(context map[string]interface{}) string
	GetExpression(hash string) (*CachedExpression, bool)
	StoreExpression(exprHash string, result interface{}, err error)
	GetOrCreateLazyVariable(name, path string, evaluator func() (interface{}, error)) *LazyVariable
	HashContext(context map[string]interface{}) string
	HashExpression(expr interface{}, contextHash string) string
	GetStats() CacheStats
	Clear()
	Close()
}

// CachedContext represents a cached context
type CachedContext struct {
	Data      map[string]interface{}
	Hash      string
	CreatedAt time.Time
	LastUsed  time.Time
	UseCount  int64
	Size      int64
}

// CachedExpression represents a cached expression
type CachedExpression struct {
	Result    interface{}
	Hash      string
	CreatedAt time.Time
	LastUsed  time.Time
	UseCount  int64
	Error     string
}

// LazyVariable represents a lazy-evaluated variable
type LazyVariable struct {
	Name      string
	Path      string
	Evaluator func() (interface{}, error)
	Result    interface{}
	Error     error
	Evaluated bool
	CreatedAt time.Time
	LastUsed  time.Time
	UseCount  int64
}

// Evaluate executes the lazy evaluation if not already done
func (lv *LazyVariable) Evaluate() (interface{}, error) {
	if lv.Evaluated {
		return lv.Result, lv.Error
	}
	
	if lv.Evaluator != nil {
		lv.Result, lv.Error = lv.Evaluator()
		lv.Evaluated = true
		lv.LastUsed = time.Now()
		lv.UseCount++
	}
	
	return lv.Result, lv.Error
}

// CacheStats represents cache statistics
type CacheStats struct {
	ContextHits        int64
	ContextMisses      int64
	ExpressionHits     int64
	ExpressionMisses   int64
	VariableHits       int64
	VariableMisses     int64
	EvictedContexts    int64
	EvictedExpressions int64
	EvictedVariables   int64
	TotalSize          int64
	CleanupRuns        int64
}

// Additional response types for API endpoints
type AutomationDeleteResponse struct {
	Success        bool     `json:"success"`
	Message        string   `json:"message"`
	AutomationName string   `json:"automation_name"`
	Dependencies   []string `json:"dependencies"`
	Timestamp      string   `json:"timestamp"`
}

type PlaybookDeleteResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	PlaybookName string `json:"playbook_name"`
	Timestamp    string `json:"timestamp"`
}

type PluginUploadResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	PluginName string `json:"plugin_name"`
	PluginType string `json:"plugin_type"`
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
	Timestamp  string `json:"timestamp"`
}

type PluginDeleteResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	PluginName string `json:"plugin_name"`
	PluginType string `json:"plugin_type"`
	Timestamp  string `json:"timestamp"`
}

type IntegrationResponse struct {
	Success      bool                 `json:"success"`
	Message      string               `json:"message"`
	Integration  *IntegrationConfig   `json:"integration,omitempty"`
	Integrations []*IntegrationConfig `json:"integrations,omitempty"`
	Timestamp    string               `json:"timestamp"`
}

type IntegrationUploadResponse struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	IntegrationName string `json:"integration_name"`
	Filename        string `json:"filename"`
	Size            int64  `json:"size"`
	Timestamp       string `json:"timestamp"`
}

type IntegrationDeleteResponse struct {
	Success         bool     `json:"success"`
	Message         string   `json:"message"`
	IntegrationName string   `json:"integration_name"`
	Dependencies    []string `json:"dependencies"`
	Timestamp       string   `json:"timestamp"`
}

type ListAddRequest struct {
	Items    []interface{} `json:"items"`
	Position string        `json:"position,omitempty"` // "left" or "right", defaults to "right"
}

type ListRemoveRequest struct {
	Items []interface{} `json:"items"`
	Count int           `json:"count,omitempty"` // Number of occurrences to remove, defaults to 1
}

type AutomationMetadata struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	Author       string                 `json:"author"`
	Category     string                 `json:"category"`
	Tags         []string               `json:"tags"`
	Parameters   []AutomationParameter  `json:"parameters"`
	Dependencies []string               `json:"dependencies"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
	Config       map[string]interface{} `json:"config,omitempty"`
}

type AutomationParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

type AutomationMetadataResponse struct {
	Success   bool                  `json:"success"`
	Message   string                `json:"message"`
	Metadata  []AutomationMetadata  `json:"metadata,omitempty"`
	Timestamp string                `json:"timestamp"`
}

type JobSchedule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	CronExpr    string                 `json:"cron_expression"`
	Playbook    interface{}            `json:"playbook"`
	Context     map[string]interface{} `json:"context"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	NextRun     string                 `json:"next_run,omitempty"`
	LastRun     string                 `json:"last_run,omitempty"`
	Status      string                 `json:"status"`
}

type ScheduleStatus string

const (
	ScheduleStatusEnabled  ScheduleStatus = "enabled"
	ScheduleStatusDisabled ScheduleStatus = "disabled"
	ScheduleStatusAll      ScheduleStatus = ""
)

// Cache operation response types
type CacheResponse struct {
	Success      bool        `json:"success"`
	Key          string      `json:"key,omitempty"`
	Value        interface{} `json:"value,omitempty"`
	Message      string      `json:"message,omitempty"`
	ErrorMessage string      `json:"error,omitempty"`
	Timestamp    string      `json:"timestamp"`
}

type CacheListResponse struct {
	Success   bool     `json:"success"`
	Keys      []string `json:"keys,omitempty"`
	Message   string   `json:"message,omitempty"`
	Timestamp string   `json:"timestamp"`
}

type CacheStatsResponse struct {
	Success   bool                   `json:"success"`
	Stats     map[string]interface{} `json:"stats,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// Redis list operation types
type ListResponse struct {
	Success   bool          `json:"success"`
	ListName  string        `json:"list_name,omitempty"`
	Items     []interface{} `json:"items,omitempty"`
	Count     int           `json:"count,omitempty"`
	Message   string        `json:"message,omitempty"`
	Error     string        `json:"error,omitempty"`
	Timestamp string        `json:"timestamp"`
}