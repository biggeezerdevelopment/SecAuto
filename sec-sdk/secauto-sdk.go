package secsdk

//package secauto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Client represents the SecAuto SOAR Rules Engine API client
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new SecAuto API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithHTTPClient allows setting a custom HTTP client
func (c *Client) WithHTTPClient(client *http.Client) *Client {
	c.HTTPClient = client
	return c
}

// Error represents an API error response
type Error struct {
	Message   string    `json:"message"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
	Valid     bool      `json:"valid,omitempty"`
	Errors    []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
		Value   string `json:"value"`
	} `json:"errors,omitempty"`
}

func (e *Error) Error() string {
	if len(e.Errors) > 0 {
		var msgs []string
		for _, err := range e.Errors {
			msgs = append(msgs, fmt.Sprintf("%s: %s", err.Field, err.Message))
		}
		return fmt.Sprintf("%s: %s", e.Message, strings.Join(msgs, ", "))
	}
	return e.Message
}

// Common request/response types
type BaseResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type ValidationErrors struct {
	BaseResponse
	Valid  bool `json:"valid"`
	Errors []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
		Value   string `json:"value"`
	} `json:"errors"`
}

// Health Check
type HealthResponse struct {
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

func (c *Client) GetHealth(ctx context.Context) (*HealthResponse, error) {
	var response HealthResponse
	err := c.makeRequest(ctx, "GET", "/health", nil, &response, false)
	return &response, err
}

// Automations
type Automation struct {
	Name          string    `json:"name"`
	Filename      string    `json:"filename"`
	Size          int       `json:"size"`
	FileType      string    `json:"file_type"`
	Language      string    `json:"language"`
	LineCount     int       `json:"line_count"`
	FunctionCount int       `json:"function_count"`
	ImportCount   int       `json:"import_count"`
	IsValid       bool      `json:"is_valid"`
	ModifiedAt    time.Time `json:"modified_at"`
}

type AutomationsResponse struct {
	BaseResponse
	Automations []Automation `json:"automations"`
	Count       int          `json:"count"`
}

type AutomationUploadResponse struct {
	BaseResponse
	AutomationName string    `json:"automation_name"`
	Filename       string    `json:"filename"`
	Size           int       `json:"size"`
	Timestamp      time.Time `json:"timestamp"`
}

type AutomationDeleteResponse struct {
	BaseResponse
	AutomationName string   `json:"automation_name"`
	Dependencies   []string `json:"dependencies"`
}

func (c *Client) ListAutomations(ctx context.Context) (*AutomationsResponse, error) {
	var response AutomationsResponse
	err := c.makeRequest(ctx, "GET", "/automations", nil, &response, false)
	return &response, err
}

func (c *Client) UploadAutomation(ctx context.Context, filename string, content []byte) (*AutomationUploadResponse, error) {
	var response AutomationUploadResponse
	err := c.uploadFile(ctx, "/automation", "automation", filename, content, &response)
	return &response, err
}

func (c *Client) DeleteAutomation(ctx context.Context, name string) (*AutomationDeleteResponse, error) {
	var response AutomationDeleteResponse
	err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/automation/%s", url.PathEscape(name)), nil, &response, false)
	return &response, err
}

// Integrations
type Integration struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
	Version     string                 `json:"version"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

type IntegrationsResponse struct {
	BaseResponse
	Integrations []Integration `json:"integrations"`
}

type IntegrationResponse struct {
	BaseResponse
	Integration Integration `json:"integration"`
}

type CreateIntegrationRequest struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled,omitempty"`
	Version     string                 `json:"version,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Username    string                 `json:"username,omitempty"`
	Password    string                 `json:"password,omitempty"`
	APIKey      string                 `json:"apikey,omitempty"`
	Token       string                 `json:"token,omitempty"`
	Secret      string                 `json:"secret,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

type IntegrationUploadResponse struct {
	BaseResponse
	IntegrationName string `json:"integration_name"`
	Filename        string `json:"filename"`
	Size            int    `json:"size"`
}

type IntegrationDeleteResponse struct {
	BaseResponse
	IntegrationName string   `json:"integration_name"`
	Dependencies    []string `json:"dependencies"`
}

func (c *Client) ListIntegrations(ctx context.Context) (*IntegrationsResponse, error) {
	var response IntegrationsResponse
	err := c.makeRequest(ctx, "GET", "/integrations", nil, &response, true)
	return &response, err
}

func (c *Client) GetIntegration(ctx context.Context, name string) (*IntegrationResponse, error) {
	var response IntegrationResponse
	err := c.makeRequest(ctx, "GET", fmt.Sprintf("/integrations/%s", url.PathEscape(name)), nil, &response, true)
	return &response, err
}

func (c *Client) CreateIntegration(ctx context.Context, req *CreateIntegrationRequest) (*IntegrationResponse, error) {
	var response IntegrationResponse
	err := c.makeRequest(ctx, "POST", "/integrations", req, &response, true)
	return &response, err
}

func (c *Client) UpdateIntegration(ctx context.Context, name string, req *CreateIntegrationRequest) (*IntegrationResponse, error) {
	var response IntegrationResponse
	err := c.makeRequest(ctx, "PUT", fmt.Sprintf("/integrations/%s", url.PathEscape(name)), req, &response, true)
	return &response, err
}

func (c *Client) DeleteIntegration(ctx context.Context, name string) (*BaseResponse, error) {
	var response BaseResponse
	err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/integrations/%s", url.PathEscape(name)), nil, &response, true)
	return &response, err
}

func (c *Client) UploadIntegrationFile(ctx context.Context, filename string, content []byte) (*IntegrationUploadResponse, error) {
	var response IntegrationUploadResponse
	err := c.uploadFile(ctx, "/integrations/upload", "integration", filename, content, &response)
	return &response, err
}

func (c *Client) DeleteIntegrationFile(ctx context.Context, name string) (*IntegrationDeleteResponse, error) {
	var response IntegrationDeleteResponse
	err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/integrations/delete/%s", url.PathEscape(name)), nil, &response, true)
	return &response, err
}

// Playbooks
type Playbook struct {
	Name       string         `json:"name"`
	Filename   string         `json:"filename"`
	Size       int            `json:"size"`
	RuleCount  int            `json:"rule_count"`
	Operations map[string]int `json:"operations"`
	IsValid    bool           `json:"is_valid"`
	ModifiedAt time.Time      `json:"modified_at"`
}

type PlaybooksResponse struct {
	BaseResponse
	Playbooks []Playbook `json:"playbooks"`
	Count     int        `json:"count"`
}

type PlaybookUploadResponse struct {
	BaseResponse
	PlaybookName string `json:"playbook_name"`
	Filename     string `json:"filename"`
	Size         int    `json:"size"`
}

type PlaybookDeleteResponse struct {
	BaseResponse
	PlaybookName string `json:"playbook_name"`
}

type ExecutePlaybookRequest struct {
	Playbook []map[string]interface{} `json:"playbook"`
	Context  map[string]interface{}   `json:"context,omitempty"`
	Options  *ExecutionOptions        `json:"options,omitempty"`
}

type ExecutionOptions struct {
	Priority string `json:"priority,omitempty"` // low, normal, high, critical
	Timeout  int    `json:"timeout,omitempty"`  // seconds
}

type ExecutePlaybookResponse struct {
	Success   bool                     `json:"success"`
	Timestamp string                   `json:"timestamp"`
	Context   map[string]interface{}   `json:"context"`
	Results   []map[string]interface{} `json:"results"`
}

type AsyncExecuteResponse struct {
	JobID string `json:"job_id"`
}

func (c *Client) ListPlaybooks(ctx context.Context) (*PlaybooksResponse, error) {
	var response PlaybooksResponse
	err := c.makeRequest(ctx, "GET", "/playbooks", nil, &response, false)
	return &response, err
}

func (c *Client) UploadPlaybook(ctx context.Context, filename string, content []byte) (*PlaybookUploadResponse, error) {
	var response PlaybookUploadResponse
	err := c.uploadFile(ctx, "/playbook/upload", "playbook", filename, content, &response)
	return &response, err
}

func (c *Client) DeletePlaybook(ctx context.Context, name string) (*PlaybookDeleteResponse, error) {
	var response PlaybookDeleteResponse
	err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/playbook/%s", url.PathEscape(name)), nil, &response, false)
	return &response, err
}

func (c *Client) ExecutePlaybook(ctx context.Context, req *ExecutePlaybookRequest) (*ExecutePlaybookResponse, error) {
	var response ExecutePlaybookResponse
	err := c.makeRequest(ctx, "POST", "/playbook", req, &response, false)
	return &response, err
}

func (c *Client) ExecutePlaybookAsync(ctx context.Context, req *ExecutePlaybookRequest) (*AsyncExecuteResponse, error) {
	var response AsyncExecuteResponse
	err := c.makeRequest(ctx, "POST", "/playbook/async", req, &response, false)
	return &response, err
}

// Jobs
type Job struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	// Add more fields as needed based on actual response
}

type JobsResponse struct {
	Jobs []Job `json:"jobs"`
	// Add pagination fields as needed
}

type ListJobsParams struct {
	Status string `url:"status,omitempty"`
	Limit  int    `url:"limit,omitempty"`
	Offset int    `url:"offset,omitempty"`
}

func (c *Client) ListJobs(ctx context.Context, params *ListJobsParams) (*JobsResponse, error) {
	endpoint := "/jobs"
	if params != nil {
		values := url.Values{}
		if params.Status != "" {
			values.Add("status", params.Status)
		}
		if params.Limit > 0 {
			values.Add("limit", strconv.Itoa(params.Limit))
		}
		if params.Offset > 0 {
			values.Add("offset", strconv.Itoa(params.Offset))
		}
		if len(values) > 0 {
			endpoint += "?" + values.Encode()
		}
	}

	var response JobsResponse
	err := c.makeRequest(ctx, "GET", endpoint, nil, &response, false)
	return &response, err
}

func (c *Client) GetJobStats(ctx context.Context) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.makeRequest(ctx, "GET", "/jobs/stats", nil, &response, false)
	return response, err
}

func (c *Client) GetJobMetrics(ctx context.Context) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.makeRequest(ctx, "GET", "/jobs/metrics", nil, &response, false)
	return response, err
}

// Cache Management
type CacheResponse struct {
	BaseResponse
	Key          string      `json:"key,omitempty"`
	Value        interface{} `json:"value,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

type CacheInfoResponse struct {
	BaseResponse
	Operations []struct {
		Method      string `json:"method"`
		Path        string `json:"path"`
		Description string `json:"description"`
	} `json:"operations"`
}

type CacheSetRequest struct {
	Value interface{} `json:"value"`
}

func (c *Client) GetCacheInfo(ctx context.Context) (*CacheInfoResponse, error) {
	var response CacheInfoResponse
	err := c.makeRequest(ctx, "GET", "/cache", nil, &response, true)
	return &response, err
}

func (c *Client) GetCacheValue(ctx context.Context, key string) (*CacheResponse, error) {
	var response CacheResponse
	err := c.makeRequest(ctx, "GET", fmt.Sprintf("/cache/%s", url.PathEscape(key)), nil, &response, true)
	return &response, err
}

func (c *Client) SetCacheValue(ctx context.Context, key string, value interface{}) (*CacheResponse, error) {
	req := CacheSetRequest{Value: value}
	var response CacheResponse
	err := c.makeRequest(ctx, "POST", fmt.Sprintf("/cache/%s", url.PathEscape(key)), req, &response, true)
	return &response, err
}

func (c *Client) DeleteCacheValue(ctx context.Context, key string) (*CacheResponse, error) {
	var response CacheResponse
	err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/cache/%s", url.PathEscape(key)), nil, &response, true)
	return &response, err
}

// List Management
type ListResponse struct {
	BaseResponse
	ListName     string        `json:"list_name,omitempty"`
	Items        []interface{} `json:"items,omitempty"`
	Count        int           `json:"count,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

type ListAddRequest struct {
	Items    []interface{} `json:"items"`
	Position string        `json:"position,omitempty"` // "left" or "right"
}

type ListRemoveRequest struct {
	Items []interface{} `json:"items"`
	Count int           `json:"count,omitempty"` // Number of occurrences to remove
}

func (c *Client) GetListItems(ctx context.Context, listName string) (*ListResponse, error) {
	var response ListResponse
	err := c.makeRequest(ctx, "GET", fmt.Sprintf("/lists/%s", url.PathEscape(listName)), nil, &response, true)
	return &response, err
}

func (c *Client) AddListItems(ctx context.Context, listName string, req *ListAddRequest) (*ListResponse, error) {
	var response ListResponse
	err := c.makeRequest(ctx, "POST", fmt.Sprintf("/lists/%s/items", url.PathEscape(listName)), req, &response, true)
	return &response, err
}

func (c *Client) RemoveListItems(ctx context.Context, listName string, req *ListRemoveRequest) (*ListResponse, error) {
	var response ListResponse
	err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/lists/%s/items", url.PathEscape(listName)), req, &response, true)
	return &response, err
}

func (c *Client) DeleteList(ctx context.Context, listName string) (*ListResponse, error) {
	var response ListResponse
	err := c.makeRequest(ctx, "DELETE", fmt.Sprintf("/lists/%s", url.PathEscape(listName)), nil, &response, true)
	return &response, err
}

// Plugins
type Plugin struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Filename   string    `json:"filename"`
	Size       int       `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
}

type PluginsResponse struct {
	Plugins []Plugin `json:"plugins"`
	// Add more fields as needed
}

type PluginUploadResponse struct {
	BaseResponse
	PluginName string `json:"plugin_name"`
	PluginType string `json:"plugin_type"`
	Filename   string `json:"filename"`
	Size       int    `json:"size"`
}

type PluginDeleteResponse struct {
	BaseResponse
	PluginName string `json:"plugin_name"`
	PluginType string `json:"plugin_type"`
}

func (c *Client) ListPlugins(ctx context.Context) (*PluginsResponse, error) {
	var response PluginsResponse
	err := c.makeRequest(ctx, "GET", "/plugins", nil, &response, false)
	return &response, err
}

func (c *Client) UploadPlugin(ctx context.Context, pluginType, filename string, content []byte) (*PluginUploadResponse, error) {
	var response PluginUploadResponse
	endpoint := fmt.Sprintf("/plugin/%s", url.PathEscape(pluginType))
	err := c.uploadFile(ctx, endpoint, "plugin", filename, content, &response)
	return &response, err
}

func (c *Client) DeletePlugin(ctx context.Context, pluginType, name string) (*PluginDeleteResponse, error) {
	var response PluginDeleteResponse
	endpoint := fmt.Sprintf("/plugin/%s/%s", url.PathEscape(pluginType), url.PathEscape(name))
	err := c.makeRequest(ctx, "DELETE", endpoint, nil, &response, false)
	return &response, err
}

// Schedules
type Schedule struct {
	Name            string                   `json:"name"`
	ScheduleType    string                   `json:"schedule_type"`
	Description     string                   `json:"description"`
	Status          string                   `json:"status"`
	CronExpression  string                   `json:"cron_expression,omitempty"`
	IntervalSeconds int                      `json:"interval_seconds,omitempty"`
	Playbook        []map[string]interface{} `json:"playbook"`
	Context         map[string]interface{}   `json:"context,omitempty"`
}

type SchedulesResponse struct {
	Schedules []Schedule `json:"schedules"`
	// Add more fields as needed
}

func (c *Client) ListSchedules(ctx context.Context) (*SchedulesResponse, error) {
	var response SchedulesResponse
	err := c.makeRequest(ctx, "GET", "/schedules", nil, &response, false)
	return &response, err
}

func (c *Client) CreateSchedule(ctx context.Context, schedule *Schedule) (*BaseResponse, error) {
	var response BaseResponse
	err := c.makeRequest(ctx, "POST", "/schedules", schedule, &response, false)
	return &response, err
}

// Validation
type ValidateRequest struct {
	Playbook []map[string]interface{} `json:"playbook"`
	Context  map[string]interface{}   `json:"context,omitempty"`
}

func (c *Client) ValidatePlaybook(ctx context.Context, req *ValidateRequest) (*BaseResponse, error) {
	var response BaseResponse
	err := c.makeRequest(ctx, "POST", "/validate", req, &response, false)
	return &response, err
}

// Webhooks
type WebhookConfig struct {
	URL        string            `json:"url"`
	Events     []string          `json:"events"`
	Headers    map[string]string `json:"headers,omitempty"`
	RetryCount int               `json:"retry_count,omitempty"`
	Timeout    int               `json:"timeout,omitempty"`
}

func (c *Client) ConfigureWebhook(ctx context.Context, config *WebhookConfig) (*BaseResponse, error) {
	var response BaseResponse
	err := c.makeRequest(ctx, "POST", "/webhooks", config, &response, false)
	return &response, err
}

// Cluster
func (c *Client) GetClusterInfo(ctx context.Context) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.makeRequest(ctx, "GET", "/cluster", nil, &response, false)
	return response, err
}

// Helper methods
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, body interface{}, response interface{}, requiresAuth bool) error {
	var reqBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+endpoint, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if requiresAuth && c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr Error
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			return &apiErr
		}
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if response != nil {
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

func (c *Client) uploadFile(ctx context.Context, endpoint, fieldName, filename string, content []byte, response interface{}) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile(fieldName, filepath.Base(filename))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(content); err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+endpoint, &buf)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute upload request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read upload response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr Error
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			return &apiErr
		}
		return fmt.Errorf("upload API error %d: %s", resp.StatusCode, string(respBody))
	}

	if response != nil {
		if err := json.Unmarshal(respBody, response); err != nil {
			return fmt.Errorf("failed to unmarshal upload response: %w", err)
		}
	}

	return nil
}
