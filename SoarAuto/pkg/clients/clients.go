package clients

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"SoarAuto/pkg/security"
	"SoarAuto/pkg/types"
)

// Client represents a client in the system
type Client struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Active           bool                   `json:"active"`
	APIKeys          []string               `json:"api_keys"`
	EncryptionKeyID  string                 `json:"encryption_key_id"`
	RateLimits       *RateLimitConfig       `json:"rate_limits,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
	LastAccessedAt   string                 `json:"last_accessed_at,omitempty"`
	IntegrationCount int                    `json:"integration_count"`
}

// RateLimitConfig defines rate limiting for a client
type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	BurstLimit        int `json:"burst_limit"`
	Enabled           bool `json:"enabled"`
}

// ClientManager manages client configurations and operations
type ClientManager struct {
	clients     map[string]*Client
	clientsPath string
	logger      types.Logger
	mutex       sync.RWMutex
	encryptor   *security.Encryptor
}

// NewClientManager creates a new client manager
func NewClientManager(clientsPath string, logger types.Logger) (*ClientManager, error) {
	cm := &ClientManager{
		clients:     make(map[string]*Client),
		clientsPath: clientsPath,
		logger:      logger,
	}

	// Initialize encryptor for client-specific encryption
	encryptor, err := security.NewEncryptor()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize encryptor: %v", err)
	}
	cm.encryptor = encryptor

	// Ensure clients directory exists
	if err := os.MkdirAll(clientsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create clients directory: %v", err)
	}

	// Load existing clients
	if err := cm.loadClients(); err != nil {
		logger.Warning("Failed to load existing clients", map[string]interface{}{
			"component": "clients",
			"error":     err.Error(),
		})
	}

	return cm, nil
}

// loadClients loads all client configurations from disk
func (cm *ClientManager) loadClients() error {
	metadataPath := filepath.Join(cm.clientsPath, "clients_metadata.json")
	
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No clients file yet, that's okay
			return nil
		}
		return err
	}

	var clients map[string]*Client
	if err := json.Unmarshal(data, &clients); err != nil {
		return fmt.Errorf("failed to unmarshal clients metadata: %v", err)
	}

	cm.clients = clients
	
	// Update integration counts
	for clientID, client := range cm.clients {
		client.IntegrationCount = cm.countClientIntegrations(clientID)
	}

	return nil
}

// saveClients saves all client configurations to disk
func (cm *ClientManager) saveClients() error {
	metadataPath := filepath.Join(cm.clientsPath, "clients_metadata.json")
	
	data, err := json.MarshalIndent(cm.clients, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal clients metadata: %v", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write clients metadata: %v", err)
	}

	return nil
}

// ListClients returns all clients
func (cm *ClientManager) ListClients() []*Client {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	clients := make([]*Client, 0, len(cm.clients))
	for _, client := range cm.clients {
		clientCopy := *client
		clients = append(clients, &clientCopy)
	}
	return clients
}

// GetClient retrieves a specific client
func (cm *ClientManager) GetClient(clientID string) (*Client, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	client, exists := cm.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	// Return a copy
	clientCopy := *client
	return &clientCopy, nil
}

// CreateClient creates a new client
func (cm *ClientManager) CreateClient(name, description string) (*Client, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Generate unique client ID
	clientID := cm.generateClientID()

	// Check if already exists
	if _, exists := cm.clients[clientID]; exists {
		return nil, fmt.Errorf("client %s already exists", clientID)
	}

	// Generate API key for the client
	apiKey := cm.generateAPIKey()

	// Generate encryption key ID for client-specific encryption
	encryptionKeyID := cm.generateEncryptionKeyID()

	// Create client
	client := &Client{
		ID:              clientID,
		Name:            name,
		Description:     description,
		Active:          true,
		APIKeys:         []string{apiKey},
		EncryptionKeyID: encryptionKeyID,
		RateLimits: &RateLimitConfig{
			RequestsPerMinute: 100,
			BurstLimit:        20,
			Enabled:           true,
		},
		Metadata:         make(map[string]interface{}),
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
		IntegrationCount: 0,
	}

	// Create client directory structure
	if err := cm.createClientDirectories(clientID); err != nil {
		return nil, fmt.Errorf("failed to create client directories: %v", err)
	}

	// Store encryption key
	if err := cm.storeClientEncryptionKey(clientID, encryptionKeyID); err != nil {
		// Cleanup on failure
		cm.deleteClientDirectories(clientID)
		return nil, fmt.Errorf("failed to store encryption key: %v", err)
	}

	// Save client
	cm.clients[clientID] = client
	if err := cm.saveClients(); err != nil {
		// Cleanup on failure
		delete(cm.clients, clientID)
		cm.deleteClientDirectories(clientID)
		return nil, err
	}

	cm.logger.Info("Client created", map[string]interface{}{
		"component":   "clients",
		"client_id":   clientID,
		"client_name": name,
	})

	return client, nil
}

// UpdateClient updates an existing client
func (cm *ClientManager) UpdateClient(clientID string, updates map[string]interface{}) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	client, exists := cm.clients[clientID]
	if !exists {
		return fmt.Errorf("client not found: %s", clientID)
	}

	// Update allowed fields
	if name, ok := updates["name"].(string); ok {
		client.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		client.Description = description
	}
	if active, ok := updates["active"].(bool); ok {
		client.Active = active
	}
	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		client.Metadata = metadata
	}
	if rateLimits, ok := updates["rate_limits"].(map[string]interface{}); ok {
		if rpm, ok := rateLimits["requests_per_minute"].(float64); ok {
			client.RateLimits.RequestsPerMinute = int(rpm)
		}
		if burst, ok := rateLimits["burst_limit"].(float64); ok {
			client.RateLimits.BurstLimit = int(burst)
		}
		if enabled, ok := rateLimits["enabled"].(bool); ok {
			client.RateLimits.Enabled = enabled
		}
	}

	client.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Save changes
	if err := cm.saveClients(); err != nil {
		return err
	}

	cm.logger.Info("Client updated", map[string]interface{}{
		"component": "clients",
		"client_id": clientID,
		"updates":   updates,
	})

	return nil
}

// DeleteClient deletes a client and all associated data
func (cm *ClientManager) DeleteClient(clientID string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if _, exists := cm.clients[clientID]; !exists {
		return fmt.Errorf("client not found: %s", clientID)
	}

	// Delete client directories
	if err := cm.deleteClientDirectories(clientID); err != nil {
		cm.logger.Warning("Failed to delete client directories", map[string]interface{}{
			"component": "clients",
			"client_id": clientID,
			"error":     err.Error(),
		})
	}

	// Remove from memory
	delete(cm.clients, clientID)

	// Save changes
	if err := cm.saveClients(); err != nil {
		return err
	}

	cm.logger.Info("Client deleted", map[string]interface{}{
		"component": "clients",
		"client_id": clientID,
	})

	return nil
}

// ValidateClientAPIKey validates if an API key belongs to a specific client
func (cm *ClientManager) ValidateClientAPIKey(clientID, apiKey string) bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	client, exists := cm.clients[clientID]
	if !exists || !client.Active {
		return false
	}

	for _, key := range client.APIKeys {
		if key == apiKey {
			// Update last accessed time
			client.LastAccessedAt = time.Now().UTC().Format(time.RFC3339)
			return true
		}
	}

	return false
}

// GetClientByAPIKey finds a client by their API key
func (cm *ClientManager) GetClientByAPIKey(apiKey string) (*Client, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	for _, client := range cm.clients {
		if !client.Active {
			continue
		}
		for _, key := range client.APIKeys {
			if key == apiKey {
				// Update last accessed time
				client.LastAccessedAt = time.Now().UTC().Format(time.RFC3339)
				clientCopy := *client
				return &clientCopy, nil
			}
		}
	}

	return nil, fmt.Errorf("no client found for API key")
}

// RegenerateClientAPIKey generates a new API key for a client
func (cm *ClientManager) RegenerateClientAPIKey(clientID string) (string, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	client, exists := cm.clients[clientID]
	if !exists {
		return "", fmt.Errorf("client not found: %s", clientID)
	}

	// Generate new API key
	newAPIKey := cm.generateAPIKey()
	
	// Add to client's API keys (keep old ones for gradual migration)
	client.APIKeys = append(client.APIKeys, newAPIKey)
	client.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Save changes
	if err := cm.saveClients(); err != nil {
		return "", err
	}

	cm.logger.Info("Client API key regenerated", map[string]interface{}{
		"component": "clients",
		"client_id": clientID,
	})

	return newAPIKey, nil
}

// GetClientPath returns the base path for a client's data
func (cm *ClientManager) GetClientPath(clientID string) string {
	return filepath.Join(cm.clientsPath, clientID)
}

// GetClientIntegrationsPath returns the path for a client's integrations
func (cm *ClientManager) GetClientIntegrationsPath(clientID string) string {
	return filepath.Join(cm.GetClientPath(clientID), "integrations")
}

// Helper functions

func (cm *ClientManager) generateClientID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("client_%s", hex.EncodeToString(bytes))
}

func (cm *ClientManager) generateAPIKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return fmt.Sprintf("sk_%s", hex.EncodeToString(bytes))
}

func (cm *ClientManager) generateEncryptionKeyID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (cm *ClientManager) createClientDirectories(clientID string) error {
	clientPath := cm.GetClientPath(clientID)
	
	// Create main client directory
	if err := os.MkdirAll(clientPath, 0755); err != nil {
		return err
	}

	// Create subdirectories
	subdirs := []string{
		"integrations/configs",
		"integrations/scripts",
		"playbooks",
		"logs",
		"cache",
	}

	for _, subdir := range subdirs {
		path := filepath.Join(clientPath, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (cm *ClientManager) deleteClientDirectories(clientID string) error {
	clientPath := cm.GetClientPath(clientID)
	return os.RemoveAll(clientPath)
}

func (cm *ClientManager) storeClientEncryptionKey(clientID, encryptionKeyID string) error {
	// Store encryption key in a secure location
	// This is a placeholder - in production, use a proper key management service
	keyPath := filepath.Join(cm.GetClientPath(clientID), ".encryption_key")
	return os.WriteFile(keyPath, []byte(encryptionKeyID), 0600)
}

func (cm *ClientManager) countClientIntegrations(clientID string) int {
	integrationsPath := filepath.Join(cm.GetClientIntegrationsPath(clientID), "configs")
	files, err := filepath.Glob(filepath.Join(integrationsPath, "*.json"))
	if err != nil {
		return 0
	}
	return len(files)
}

// AuditLog logs client-related actions for audit purposes
func (cm *ClientManager) AuditLog(clientID, action string, details map[string]interface{}) {
	cm.logger.Info("Client action", map[string]interface{}{
		"component": "audit",
		"client_id": clientID,
		"action":    action,
		"details":   details,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}