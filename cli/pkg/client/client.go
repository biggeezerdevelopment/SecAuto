package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

// Client represents the SecAuto API client
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	UserAgent  string
}

// NewClient creates a new SecAuto API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UserAgent: "secauto-cli/1.0.0",
	}
}

// Request represents an API request
type Request struct {
	Method string
	Path   string
	Body   interface{}
	Query  map[string]string
}

// Response represents an API response
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Do performs an HTTP request to the SecAuto API
func (c *Client) Do(req *Request) (*Response, error) {
	// Build URL
	u, err := url.Parse(c.BaseURL + req.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}

	// Add query parameters
	if req.Query != nil {
		q := u.Query()
		for key, value := range req.Query {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
	}

	// Prepare request body
	var body io.Reader
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		body = bytes.NewBuffer(jsonBody)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(req.Method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", c.UserAgent)
	if c.APIKey != "" {
		httpReq.Header.Set("X-API-Key", c.APIKey)
	}

	// Perform request
	httpResp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	return &Response{
		StatusCode: httpResp.StatusCode,
		Body:       respBody,
		Headers:    httpResp.Header,
	}, nil
}

// HealthCheck checks if the SecAuto server is healthy
func (c *Client) HealthCheck() error {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/health",
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	return nil
}

// PlaybookRequest represents a playbook execution request
type PlaybookRequest struct {
	Playbook     interface{}            `json:"playbook,omitempty"`
	PlaybookName string                 `json:"playbook_name,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// PlaybookResponse represents a playbook execution response
type PlaybookResponse struct {
	Success   bool        `json:"success"`
	Results   interface{} `json:"results,omitempty"`
	Error     string      `json:"error,omitempty"`
	JobID     string      `json:"job_id,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// ExecutePlaybook executes a playbook synchronously
func (c *Client) ExecutePlaybook(req *PlaybookRequest) (*PlaybookResponse, error) {
	resp, err := c.Do(&Request{
		Method: "POST",
		Path:   "/playbook",
		Body:   req,
	})
	if err != nil {
		return nil, err
	}

	var result PlaybookResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &result, nil
}

// ExecutePlaybookAsync executes a playbook asynchronously
func (c *Client) ExecutePlaybookAsync(req *PlaybookRequest) (*PlaybookResponse, error) {
	resp, err := c.Do(&Request{
		Method: "POST",
		Path:   "/playbook/async",
		Body:   req,
	})
	if err != nil {
		return nil, err
	}

	var result PlaybookResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &result, nil
}

// UploadPlaybook uploads a new playbook
func (c *Client) UploadPlaybook(name string, playbook interface{}) error {
	req := map[string]interface{}{
		"name":     name,
		"playbook": playbook,
	}

	resp, err := c.Do(&Request{
		Method: "POST",
		Path:   "/playbook/upload",
		Body:   req,
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("upload failed: status %d", resp.StatusCode)
	}

	return nil
}

// ListPlaybooks lists all available playbooks
func (c *Client) ListPlaybooks() ([]string, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/playbooks",
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success   bool     `json:"success"`
		Playbooks []string `json:"playbooks"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result.Playbooks, nil
}

// Job represents a job
type Job struct {
	ID          string                 `json:"id"`
	Status      string                 `json:"status"`
	Playbook    interface{}            `json:"playbook"`
	Context     map[string]interface{} `json:"context"`
	Results     interface{}            `json:"results,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   *time.Time             `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Priority    int                    `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ListJobs lists all jobs
func (c *Client) ListJobs() ([]*Job, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/jobs",
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool   `json:"success"`
		Jobs    []*Job `json:"jobs"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result.Jobs, nil
}

// GetJob gets a specific job by ID
func (c *Client) GetJob(jobID string) (*Job, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/job/" + jobID,
	})
	if err != nil {
		return nil, err
	}

	var job Job
	if err := json.Unmarshal(resp.Body, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &job, nil
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalKeys     int `json:"total_keys"`
	TotalMemory   int `json:"total_memory"`
	HitRate       float64 `json:"hit_rate"`
	MissRate      float64 `json:"miss_rate"`
}

// GetCacheStats gets cache statistics
func (c *Client) GetCacheStats() (*CacheStats, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/cache/stats",
	})
	if err != nil {
		return nil, err
	}

	var stats CacheStats
	if err := json.Unmarshal(resp.Body, &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &stats, nil
}

// GetCacheKey gets a value from cache
func (c *Client) GetCacheKey(key string) (interface{}, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/cache/" + key,
	})
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result["value"], nil
}

// SetCacheKey sets a value in cache
func (c *Client) SetCacheKey(key string, value interface{}) error {
	req := map[string]interface{}{
		"value": value,
	}

	resp, err := c.Do(&Request{
		Method: "POST",
		Path:   "/cache/" + key,
		Body:   req,
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set cache key: status %d", resp.StatusCode)
	}

	return nil
}

// DeleteCacheKey deletes a key from cache
func (c *Client) DeleteCacheKey(key string) error {
	resp, err := c.Do(&Request{
		Method: "DELETE",
		Path:   "/cache/" + key,
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete cache key: status %d", resp.StatusCode)
	}

	return nil
}

// ClearCache clears all cache entries
func (c *Client) ClearCache() error {
	resp, err := c.Do(&Request{
		Method: "POST",
		Path:   "/cache/clear",
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to clear cache: status %d", resp.StatusCode)
	}

	return nil
}

// AutomationInfo represents information about an automation
type AutomationInfo struct {
	Name          string `json:"name"`
	Filename      string `json:"filename"`
	Size          int    `json:"size"`
	FileType      string `json:"file_type"`
	Language      string `json:"language"`
	LineCount     int    `json:"line_count"`
	FunctionCount int    `json:"function_count"`
	ImportCount   int    `json:"import_count"`
	ModifiedAt    string `json:"modified_at"`
	IsValid       bool   `json:"is_valid"`
}

// ListAutomations lists all automations
func (c *Client) ListAutomations() ([]string, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/automations",
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success     bool             `json:"success"`
		Automations []AutomationInfo `json:"automations"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// Extract just the names for backward compatibility
	names := make([]string, len(result.Automations))
	for i, automation := range result.Automations {
		names[i] = automation.Name
	}

	return names, nil
}

// IntegrationInfo represents information about an integration
type IntegrationInfo struct {
	Name          string                 `json:"name"`
	Version       string                 `json:"version"`
	Description   string                 `json:"description"`
	Author        string                 `json:"author"`
	Built         bool                   `json:"built"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
	Configuration map[string]interface{} `json:"configuration"`
	Functions     map[string]interface{} `json:"functions"`
}

// ListIntegrations lists all integrations
func (c *Client) ListIntegrations() ([]string, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/integrations",
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success      bool               `json:"success"`
		Integrations []IntegrationInfo `json:"integrations"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// Extract just the names for backward compatibility
	names := make([]string, len(result.Integrations))
	for i, integration := range result.Integrations {
		names[i] = integration.Name
	}

	return names, nil
}

// GetIntegration gets a specific integration
func (c *Client) GetIntegration(name string) (interface{}, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/integrations/" + name,
	})
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if success, ok := result["success"].(bool); ok && !success {
		if msg, ok := result["message"].(string); ok {
			return nil, fmt.Errorf("server error: %s", msg)
		}
	}

	if integration, ok := result["integration"]; ok {
		return integration, nil
	}

	return result, nil
}

// ExecuteIntegration executes an integration
// Note: Direct integration execution is not supported. Integrations should be executed through playbooks or client-specific endpoints.
func (c *Client) ExecuteIntegration(name string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("direct integration execution is not supported. Integrations should be executed through playbooks or client-specific endpoints. Use 'integration info' for more details")
}

// ClusterStatus represents cluster status
type ClusterStatus struct {
	NodeID      string `json:"node_id"`
	TotalNodes  int    `json:"total_nodes"`
	ActiveNodes int    `json:"active_nodes"`
	Status      string `json:"status"`
}

// GetClusterStatus gets cluster status
func (c *Client) GetClusterStatus() (*ClusterStatus, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/cluster/status",
	})
	if err != nil {
		return nil, err
	}

	var status ClusterStatus
	if err := json.Unmarshal(resp.Body, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &status, nil
}

// GetClusterJobs gets all cluster jobs
func (c *Client) GetClusterJobs() ([]interface{}, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/cluster/jobs",
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool          `json:"success"`
		Jobs    []interface{} `json:"jobs"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result.Jobs, nil
}

// GetClusterJob gets a specific cluster job
func (c *Client) GetClusterJob(jobID string) (interface{}, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/cluster/jobs/" + jobID,
	})
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result, nil
}

// Schedule represents a job schedule
type Schedule struct {
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

// ListSchedules lists all schedules with optional status filter
func (c *Client) ListSchedules(status string) ([]*Schedule, error) {
	query := make(map[string]string)
	if status != "" {
		query["status"] = status
	}

	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/schedules",
		Query:  query,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success   bool        `json:"success"`
		Schedules []*Schedule `json:"schedules"`
		Message   string      `json:"message"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("server error: %s", result.Message)
	}

	return result.Schedules, nil
}

// GetSchedule gets a specific schedule by ID
func (c *Client) GetSchedule(scheduleID string) (*Schedule, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/schedule/" + scheduleID,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success  bool      `json:"success"`
		Schedule *Schedule `json:"schedule"`
		Message  string    `json:"message"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("server error: %s", result.Message)
	}

	return result.Schedule, nil
}

// CreateSchedule creates a new schedule
func (c *Client) CreateSchedule(schedule interface{}) (*Schedule, error) {
	resp, err := c.Do(&Request{
		Method: "POST",
		Path:   "/schedules",
		Body:   schedule,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success  bool      `json:"success"`
		Schedule *Schedule `json:"schedule"`
		Message  string    `json:"message"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("server error: %s", result.Message)
	}

	return result.Schedule, nil
}

// UpdateSchedule updates an existing schedule
func (c *Client) UpdateSchedule(scheduleID string, updates interface{}) error {
	resp, err := c.Do(&Request{
		Method: "PUT",
		Path:   "/schedule/" + scheduleID,
		Body:   updates,
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var result struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(resp.Body, &result); err == nil && result.Message != "" {
			return fmt.Errorf("update failed: %s", result.Message)
		}
		return fmt.Errorf("update failed: status %d", resp.StatusCode)
	}

	return nil
}

// DeleteSchedule deletes a schedule
func (c *Client) DeleteSchedule(scheduleID string) error {
	resp, err := c.Do(&Request{
		Method: "DELETE",
		Path:   "/schedule/" + scheduleID,
	})
	if err != nil {
		return err
	}

	var result struct {
		Success string `json:"success"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if result.Success != "true" {
		return fmt.Errorf("server error: %s", result.Message)
	}

	return nil
}

// ScheduleExecuteResult represents the result of executing a schedule
type ScheduleExecuteResult struct {
	Success bool   `json:"success"`
	JobID   string `json:"job_id"`
	Message string `json:"message"`
}

// ExecuteSchedule executes a schedule immediately
func (c *Client) ExecuteSchedule(scheduleID string) (*ScheduleExecuteResult, error) {
	resp, err := c.Do(&Request{
		Method: "POST",
		Path:   "/schedule/execute/" + scheduleID,
	})
	if err != nil {
		return nil, err
	}

	var result ScheduleExecuteResult
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !result.Success && result.Message != "" {
		return nil, fmt.Errorf("execution failed: %s", result.Message)
	}

	return &result, nil
}

// GetScheduleStats gets schedule statistics
func (c *Client) GetScheduleStats() (map[string]interface{}, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/schedules/stats",
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool                   `json:"success"`
		Stats   map[string]interface{} `json:"stats"`
		Message string                 `json:"message"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("server error: %s", result.Message)
	}

	return result.Stats, nil
}

// UploadAutomation uploads a new automation script
func (c *Client) UploadAutomation(filename string, content []byte) error {
	// Create a buffer to hold the multipart form data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Create the file field
	part, err := writer.CreateFormFile("automation", filename)
	if err != nil {
		return fmt.Errorf("failed to create form file: %v", err)
	}

	// Write the content
	if _, err := part.Write(content); err != nil {
		return fmt.Errorf("failed to write file content: %v", err)
	}

	// Close the writer to finalize the form
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}

	// Create the HTTP request
	u, err := url.Parse(c.BaseURL + "/automation")
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	httpReq, err := http.NewRequest("POST", u.String(), &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("User-Agent", c.UserAgent)
	if c.APIKey != "" {
		httpReq.Header.Set("X-API-Key", c.APIKey)
	}

	// Perform request
	httpResp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	// Check status code
	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		var result map[string]interface{}
		if err := json.Unmarshal(respBody, &result); err == nil {
			if msg, ok := result["message"].(string); ok {
				return fmt.Errorf("upload failed: %s", msg)
			}
		}
		return fmt.Errorf("upload failed: status %d", httpResp.StatusCode)
	}

	return nil
}

// DeleteAutomation deletes an automation script
func (c *Client) DeleteAutomation(name string) error {
	resp, err := c.Do(&Request{
		Method: "DELETE",
		Path:   "/automation/" + name,
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete failed: status %d", resp.StatusCode)
	}

	return nil
}

// ClientInfo represents a SecAuto client
type ClientInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Status       string                 `json:"status"`
	LastSeen     string                 `json:"last_seen"`
	Integrations int                    `json:"integrations"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ListClients lists all clients
func (c *Client) ListClients() ([]*ClientInfo, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/clients",
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool          `json:"success"`
		Clients []*ClientInfo `json:"clients"`
		Message string        `json:"message"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("server error: %s", result.Message)
	}

	return result.Clients, nil
}

// GetClient gets a specific client by ID
func (c *Client) GetClient(clientID string) (*ClientInfo, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/clients/" + clientID,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool        `json:"success"`
		Client  *ClientInfo `json:"client"`
		Message string      `json:"message"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("server error: %s", result.Message)
	}

	return result.Client, nil
}

// DeleteClient deletes a client
func (c *Client) DeleteClient(clientID string) error {
	resp, err := c.Do(&Request{
		Method: "DELETE",
		Path:   "/clients/" + clientID,
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete failed: status %d", resp.StatusCode)
	}

	return nil
}

// APIKeyInfo represents API key information
type APIKeyInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Key         string   `json:"key,omitempty"`
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"created_at"`
	LastUsed    string   `json:"last_used,omitempty"`
	UsageCount  int      `json:"usage_count"`
}

// ListAPIKeys lists all API keys
func (c *Client) ListAPIKeys() ([]*APIKeyInfo, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/api-keys",
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool          `json:"success"`
		Keys    []*APIKeyInfo `json:"keys"`
		Message string        `json:"message"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("server error: %s", result.Message)
	}

	return result.Keys, nil
}

// CreateAPIKey creates a new API key
func (c *Client) CreateAPIKey(name, description string, permissions []string) (*APIKeyInfo, error) {
	req := map[string]interface{}{
		"name":        name,
		"description": description,
		"permissions": permissions,
	}

	resp, err := c.Do(&Request{
		Method: "POST",
		Path:   "/api-keys",
		Body:   req,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool        `json:"success"`
		Key     *APIKeyInfo `json:"key"`
		Message string      `json:"message"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("server error: %s", result.Message)
	}

	return result.Key, nil
}

// DeleteAPIKey deletes an API key
func (c *Client) DeleteAPIKey(keyID string) error {
	resp, err := c.Do(&Request{
		Method: "DELETE",
		Path:   "/api-keys",
		Body:   map[string]interface{}{"key_id": keyID},
	})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete failed: status %d", resp.StatusCode)
	}

	return nil
}

// GetAPIKeyStats gets API key usage statistics
func (c *Client) GetAPIKeyStats() (map[string]interface{}, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/api-keys/stats",
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool                   `json:"success"`
		Stats   map[string]interface{} `json:"stats"`
		Message string                 `json:"message"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("server error: %s", result.Message)
	}

	return result.Stats, nil
}

// GetJobStats gets job statistics
func (c *Client) GetJobStats() (map[string]interface{}, error) {
	resp, err := c.Do(&Request{
		Method: "GET",
		Path:   "/jobs/stats",
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool                   `json:"success"`
		Stats   map[string]interface{} `json:"stats"`
		Message string                 `json:"message"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("server error: %s", result.Message)
	}

	return result.Stats, nil
}