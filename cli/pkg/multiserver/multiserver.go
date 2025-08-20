package multiserver

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"secauto-cli/pkg/client"
	"secauto-cli/pkg/database"
)

// Manager handles multi-server operations
type Manager struct {
	db       *database.DB
	clients  map[int]*client.Client
	mu       sync.RWMutex
}

// ExecutionResult represents the result of an execution on a server
type ExecutionResult struct {
	ServerID   int         `json:"server_id"`
	ServerName string      `json:"server_name"`
	Success    bool        `json:"success"`
	Result     interface{} `json:"result,omitempty"`
	Error      string      `json:"error,omitempty"`
	Duration   string      `json:"duration"`
}

// NewManager creates a new multi-server manager
func NewManager(dbPath string) (*Manager, error) {
	db, err := database.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	return &Manager{
		db:      db,
		clients: make(map[int]*client.Client),
	}, nil
}

// Close closes the manager and database connection
func (m *Manager) Close() error {
	return m.db.Close()
}

// getClient gets or creates a client for a server
func (m *Manager) getClient(server *database.Server) *client.Client {
	m.mu.RLock()
	if c, exists := m.clients[server.ID]; exists {
		m.mu.RUnlock()
		return c
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if c, exists := m.clients[server.ID]; exists {
		return c
	}

	// Create new client
	c := client.NewClient(server.URL, server.APIKey)
	m.clients[server.ID] = c
	return c
}

// AddServer adds a new server to the database
func (m *Manager) AddServer(name, url, apiKey, description string, isDefault bool) error {
	server := &database.Server{
		Name:        name,
		URL:         url,
		APIKey:      apiKey,
		Description: description,
		IsDefault:   isDefault,
		IsActive:    true,
	}

	// Test connection before adding
	testClient := client.NewClient(url, apiKey)
	if err := testClient.HealthCheck(); err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}

	return m.db.AddServer(server)
}

// RemoveServer removes a server from the database
func (m *Manager) RemoveServer(nameOrID string) error {
	// Try to parse as ID first
	var server *database.Server
	var err error

	// Try by name
	server, err = m.db.GetServerByName(nameOrID)
	if err != nil {
		return fmt.Errorf("server not found: %s", nameOrID)
	}

	// Remove from client cache
	m.mu.Lock()
	delete(m.clients, server.ID)
	m.mu.Unlock()

	return m.db.DeleteServer(server.ID)
}

// ListServers lists all configured servers
func (m *Manager) ListServers() ([]*database.Server, error) {
	return m.db.ListServers(false)
}

// SetDefaultServer sets a server as the default
func (m *Manager) SetDefaultServer(nameOrID string) error {
	server, err := m.db.GetServerByName(nameOrID)
	if err != nil {
		return fmt.Errorf("server not found: %s", nameOrID)
	}

	server.IsDefault = true
	return m.db.UpdateServer(server)
}

// ToggleServer enables or disables a server
func (m *Manager) ToggleServer(nameOrID string, active bool) error {
	server, err := m.db.GetServerByName(nameOrID)
	if err != nil {
		return fmt.Errorf("server not found: %s", nameOrID)
	}

	server.IsActive = active
	return m.db.UpdateServer(server)
}

// ExecuteOnAllServers executes a function on all active servers in parallel
func (m *Manager) ExecuteOnAllServers(
	execType string,
	name string,
	executor func(*client.Client) (interface{}, error),
) ([]ExecutionResult, error) {
	servers, err := m.db.GetActiveServers()
	if err != nil {
		return nil, fmt.Errorf("failed to get active servers: %v", err)
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("no active servers configured")
	}

	// Execute in parallel
	var wg sync.WaitGroup
	results := make([]ExecutionResult, len(servers))

	for i, server := range servers {
		wg.Add(1)
		go func(idx int, srv *database.Server) {
			defer wg.Done()

			start := time.Now()
			result := ExecutionResult{
				ServerID:   srv.ID,
				ServerName: srv.Name,
			}

			// Execute the operation
			c := m.getClient(srv)
			res, err := executor(c)
			
			duration := time.Since(start)
			result.Duration = duration.String()

			if err != nil {
				result.Success = false
				result.Error = err.Error()
			} else {
				result.Success = true
				result.Result = res
			}

			// Record execution
			exec := &database.Execution{
				ServerID:   srv.ID,
				ServerName: srv.Name,
				Type:       execType,
				Name:       name,
				Status:     "success",
			}

			if !result.Success {
				exec.Status = "failed"
				exec.Error = result.Error
			} else {
				// Store result as JSON
				if resBytes, err := json.Marshal(result.Result); err == nil {
					exec.Result = string(resBytes)
				}
			}

			m.db.RecordExecution(exec)
			results[idx] = result
		}(i, server)
	}

	wg.Wait()
	return results, nil
}

// UploadPlaybookToAll uploads a playbook to all active servers
func (m *Manager) UploadPlaybookToAll(name string, playbook interface{}) ([]ExecutionResult, error) {
	// Calculate hash for sync tracking
	playbookBytes, _ := json.Marshal(playbook)
	hash := calculateHash(playbookBytes)

	return m.ExecuteOnAllServers("playbook_upload", name, func(c *client.Client) (interface{}, error) {
		// Check if already synced with same hash
		servers, _ := m.db.GetActiveServers()
		for _, srv := range servers {
			needsSync, _ := m.db.NeedsSyncing(srv.ID, "playbook", name, hash)
			if !needsSync {
				return map[string]string{"status": "already_synced"}, nil
			}
		}

		// Upload playbook
		err := c.UploadPlaybook(name, playbook)
		if err != nil {
			return nil, err
		}

		// Update sync status
		for _, srv := range servers {
			m.db.UpdateSyncStatus(&database.SyncStatus{
				ServerID: srv.ID,
				Type:     "playbook",
				Name:     name,
				Hash:     hash,
				Status:   "synced",
			})
		}

		return map[string]string{"status": "uploaded"}, nil
	})
}

// UploadAutomationToAll uploads an automation to all active servers
func (m *Manager) UploadAutomationToAll(filename string, content []byte) ([]ExecutionResult, error) {
	hash := calculateHash(content)

	return m.ExecuteOnAllServers("automation_upload", filename, func(c *client.Client) (interface{}, error) {
		// Check if already synced with same hash
		servers, _ := m.db.GetActiveServers()
		for _, srv := range servers {
			needsSync, _ := m.db.NeedsSyncing(srv.ID, "automation", filename, hash)
			if !needsSync {
				return map[string]string{"status": "already_synced"}, nil
			}
		}

		// Upload automation
		err := c.UploadAutomation(filename, content)
		if err != nil {
			return nil, err
		}

		// Update sync status
		for _, srv := range servers {
			m.db.UpdateSyncStatus(&database.SyncStatus{
				ServerID: srv.ID,
				Type:     "automation",
				Name:     filename,
				Hash:     hash,
				Status:   "synced",
			})
		}

		return map[string]string{"status": "uploaded"}, nil
	})
}

// ExecutePlaybookOnAll executes a playbook on all active servers
func (m *Manager) ExecutePlaybookOnAll(req *client.PlaybookRequest) ([]ExecutionResult, error) {
	name := "inline"
	if req.PlaybookName != "" {
		name = req.PlaybookName
	}

	return m.ExecuteOnAllServers("playbook_execute", name, func(c *client.Client) (interface{}, error) {
		return c.ExecutePlaybook(req)
	})
}

// ListPlaybooksFromAll lists playbooks from all active servers
func (m *Manager) ListPlaybooksFromAll() (map[string][]string, error) {
	servers, err := m.db.GetActiveServers()
	if err != nil {
		return nil, fmt.Errorf("failed to get active servers: %v", err)
	}

	result := make(map[string][]string)
	for _, server := range servers {
		c := m.getClient(server)
		playbooks, err := c.ListPlaybooks()
		if err != nil {
			result[server.Name] = []string{fmt.Sprintf("Error: %v", err)}
		} else {
			result[server.Name] = playbooks
		}
	}

	return result, nil
}

// ListAutomationsFromAll lists automations from all active servers
func (m *Manager) ListAutomationsFromAll() (map[string][]string, error) {
	servers, err := m.db.GetActiveServers()
	if err != nil {
		return nil, fmt.Errorf("failed to get active servers: %v", err)
	}

	result := make(map[string][]string)
	for _, server := range servers {
		c := m.getClient(server)
		automations, err := c.ListAutomations()
		if err != nil {
			result[server.Name] = []string{fmt.Sprintf("Error: %v", err)}
		} else {
			result[server.Name] = automations
		}
	}

	return result, nil
}

// SyncDirectory syncs a directory of files to all servers
func (m *Manager) SyncDirectory(dirPath string, fileType string) error {
	servers, err := m.db.GetActiveServers()
	if err != nil {
		return fmt.Errorf("failed to get active servers: %v", err)
	}

	if len(servers) == 0 {
		return fmt.Errorf("no active servers configured")
	}

	// Read directory
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}

	var syncErrors []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		filepath := filepath.Join(dirPath, filename)

		// Determine file type
		switch fileType {
		case "playbook":
			if !hasExtension(filename, ".json") {
				continue
			}

			// Read and parse playbook
			content, err := os.ReadFile(filepath)
			if err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("%s: %v", filename, err))
				continue
			}

			var playbook interface{}
			if err := json.Unmarshal(content, &playbook); err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("%s: invalid JSON: %v", filename, err))
				continue
			}

			// Upload to all servers
			name := removeExtension(filename)
			results, err := m.UploadPlaybookToAll(name, playbook)
			if err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("%s: %v", filename, err))
				continue
			}

			// Check for failures
			for _, result := range results {
				if !result.Success {
					syncErrors = append(syncErrors, 
						fmt.Sprintf("%s on %s: %s", filename, result.ServerName, result.Error))
				}
			}

		case "automation":
			if !hasExtension(filename, ".py") {
				continue
			}

			// Read automation file
			content, err := os.ReadFile(filepath)
			if err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("%s: %v", filename, err))
				continue
			}

			// Upload to all servers
			results, err := m.UploadAutomationToAll(filename, content)
			if err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("%s: %v", filename, err))
				continue
			}

			// Check for failures
			for _, result := range results {
				if !result.Success {
					syncErrors = append(syncErrors, 
						fmt.Sprintf("%s on %s: %s", filename, result.ServerName, result.Error))
				}
			}

		default:
			return fmt.Errorf("unsupported file type: %s", fileType)
		}
	}

	if len(syncErrors) > 0 {
		return fmt.Errorf("sync completed with errors:\n%v", syncErrors)
	}

	return nil
}

// GetExecutionHistory gets execution history
func (m *Manager) GetExecutionHistory(serverName string, limit int) ([]*database.Execution, error) {
	serverID := 0
	if serverName != "" {
		server, err := m.db.GetServerByName(serverName)
		if err != nil {
			return nil, fmt.Errorf("server not found: %s", serverName)
		}
		serverID = server.ID
	}

	return m.db.GetExecutionHistory(serverID, limit)
}

// GetExecutionStats gets execution statistics
func (m *Manager) GetExecutionStats() (map[string]interface{}, error) {
	return m.db.GetExecutionStats()
}

// TestAllServers tests connectivity to all servers
func (m *Manager) TestAllServers() ([]ExecutionResult, error) {
	return m.ExecuteOnAllServers("health_check", "health", func(c *client.Client) (interface{}, error) {
		err := c.HealthCheck()
		if err != nil {
			return nil, err
		}
		return map[string]string{"status": "healthy"}, nil
	})
}

// Helper functions

func calculateHash(data []byte) string {
	h := md5.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func hasExtension(filename, ext string) bool {
	return filepath.Ext(filename) == ext
}

func removeExtension(filename string) string {
	return filename[:len(filename)-len(filepath.Ext(filename))]
}