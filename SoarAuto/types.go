package main

// PythonOutput struct to unmarshal JSON output from Python scripts
type PythonOutput struct {
	Context  map[string]interface{} `json:"context"`
	Incident map[string]interface{} `json:"incident"`
}

// SecAutoServer represents the SOAR automation server
type SecAutoServer struct {
	engine                   *RuleEngine
	port                     string
	jobManager               *JobManager
	webhookManager           *WebhookManager
	validator                *Validator
	pluginManager            *PlatformPluginManager
	clusterManager           *ClusterManager
	jobScheduler             *JobScheduler
	integrationConfigManager *IntegrationConfigManager
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
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// JobResponse represents the response for job submission
type JobResponse struct {
	Success   bool   `json:"success"`
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// JobStats represents job statistics
type JobStats struct {
	TotalJobs   int     `json:"total_jobs"`
	Completed   int     `json:"completed"`
	Failed      int     `json:"failed"`
	Running     int     `json:"running"`
	Pending     int     `json:"pending"`
	AvgDuration float64 `json:"avg_duration_seconds"`
	RecentJobs  []*Job  `json:"recent_jobs"`
}

// PlaybookRequest represents a request to execute a playbook
type PlaybookRequest struct {
	Playbook     []interface{}          `json:"playbook,omitempty"`
	PlaybookName string                 `json:"playbook_name,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// PlaybookResponse represents the response from a playbook execution
type PlaybookResponse struct {
	Success   bool                   `json:"success"`
	Results   []interface{}          `json:"results,omitempty"`
	Context   map[string]interface{} `json:"context"`
	Error     string                 `json:"error,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// AutomationUploadResponse represents the response for automation upload
type AutomationUploadResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	AutomationName string `json:"automation_name"`
	Filename       string `json:"filename"`
	Size           int64  `json:"size"`
	Timestamp      string `json:"timestamp"`
}

// PlaybookUploadResponse represents the response for playbook upload
type PlaybookUploadResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	PlaybookName string `json:"playbook_name"`
	Filename     string `json:"filename"`
	Size         int64  `json:"size"`
	Timestamp    string `json:"timestamp"`
}

// PlaybookInfo represents information about a playbook
type PlaybookInfo struct {
	Name       string         `json:"name"`
	Filename   string         `json:"filename"`
	Size       int64          `json:"size"`
	RuleCount  int            `json:"rule_count"`
	Operations map[string]int `json:"operations"`
	ModifiedAt string         `json:"modified_at"`
	IsValid    bool           `json:"is_valid"`
}

// PlaybookListResponse represents the response for playbook list
type PlaybookListResponse struct {
	Success   bool           `json:"success"`
	Message   string         `json:"message"`
	Playbooks []PlaybookInfo `json:"playbooks"`
	Count     int            `json:"count"`
	Timestamp string         `json:"timestamp"`
}

// PlaybookDeleteResponse represents the response for playbook deletion
type PlaybookDeleteResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	PlaybookName string `json:"playbook_name"`
	Timestamp    string `json:"timestamp"`
}

// PluginUploadResponse represents the response for plugin upload
type PluginUploadResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	PluginName string `json:"plugin_name"`
	PluginType string `json:"plugin_type"`
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
	Timestamp  string `json:"timestamp"`
}

// PluginDeleteResponse represents the response for plugin deletion
type PluginDeleteResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	PluginName string `json:"plugin_name"`
	PluginType string `json:"plugin_type"`
	Timestamp  string `json:"timestamp"`
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

// AutomationListResponse represents the response for automation list
type AutomationListResponse struct {
	Success     bool             `json:"success"`
	Message     string           `json:"message"`
	Automations []AutomationInfo `json:"automations"`
	Count       int              `json:"count"`
	Timestamp   string           `json:"timestamp"`
}

// AutomationDeleteResponse represents the response for automation deletion
type AutomationDeleteResponse struct {
	Success        bool     `json:"success"`
	Message        string   `json:"message"`
	AutomationName string   `json:"automation_name"`
	Dependencies   []string `json:"dependencies"`
	Timestamp      string   `json:"timestamp"`
}

// IntegrationResponse represents the response for integration operations
type IntegrationResponse struct {
	Success      bool                 `json:"success"`
	Message      string               `json:"message"`
	Integration  *IntegrationConfig   `json:"integration,omitempty"`
	Integrations []*IntegrationConfig `json:"integrations,omitempty"`
	Timestamp    string               `json:"timestamp"`
}

// IntegrationUploadResponse represents the response for integration upload
type IntegrationUploadResponse struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	IntegrationName string `json:"integration_name"`
	Filename        string `json:"filename"`
	Size            int64  `json:"size"`
	Timestamp       string `json:"timestamp"`
}

// IntegrationDeleteResponse represents the response for integration file deletion
type IntegrationDeleteResponse struct {
	Success         bool     `json:"success"`
	Message         string   `json:"message"`
	IntegrationName string   `json:"integration_name"`
	Dependencies    []string `json:"dependencies"`
	Timestamp       string   `json:"timestamp"`
}
