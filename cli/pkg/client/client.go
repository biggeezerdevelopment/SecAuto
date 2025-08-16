package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
		Success     bool     `json:"success"`
		Automations []string `json:"automations"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result.Automations, nil
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
		Success      bool     `json:"success"`
		Integrations []string `json:"integrations"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result.Integrations, nil
}

// ExecuteIntegration executes an integration
func (c *Client) ExecuteIntegration(name string, params map[string]interface{}) (interface{}, error) {
	req := map[string]interface{}{
		"name":   name,
		"params": params,
	}

	resp, err := c.Do(&Request{
		Method: "POST",
		Path:   "/integration",
		Body:   req,
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
func (c *Client) CreateSchedule(schedule *Schedule) (*Schedule, error) {
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
func (c *Client) UpdateSchedule(scheduleID string, updates *Schedule) (*Schedule, error) {
	resp, err := c.Do(&Request{
		Method: "PUT",
		Path:   "/schedule/" + scheduleID,
		Body:   updates,
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

// ExecuteSchedule manually executes a schedule
func (c *Client) ExecuteSchedule(scheduleID string) (interface{}, error) {
	resp, err := c.Do(&Request{
		Method: "POST",
		Path:   "/schedule/execute/" + scheduleID,
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

	return result["result"], nil
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