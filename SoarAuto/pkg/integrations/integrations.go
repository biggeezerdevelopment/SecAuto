package integrations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"SoarAuto/pkg/types"
)

// IntegrationManager manages integration configurations and files
type IntegrationManager struct {
	configs     map[string]*types.IntegrationConfig
	configsPath string
	scriptsPath string
	mutex       sync.RWMutex
}

// NewIntegrationManager creates a new integration manager
func NewIntegrationManager(configsPath, scriptsPath string) *IntegrationManager {
	im := &IntegrationManager{
		configs:     make(map[string]*types.IntegrationConfig),
		configsPath: configsPath,
		scriptsPath: scriptsPath,
	}
	
	// Load existing configurations
	im.loadConfigurations()
	
	return im
}

// loadConfigurations loads integration configurations from disk
func (im *IntegrationManager) loadConfigurations() {
	// Ensure config directory exists
	if err := os.MkdirAll(im.configsPath, 0755); err != nil {
		return
	}

	// Load config files
	files, err := filepath.Glob(filepath.Join(im.configsPath, "*.json"))
	if err != nil {
		return
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var config types.IntegrationConfig
		if err := json.Unmarshal(data, &config); err != nil {
			continue
		}

		// Extract name from filename
		name := filepath.Base(file)
		name = name[:len(name)-5] // Remove .json extension
		config.Name = name

		im.configs[name] = &config
	}
}

// ListConfigs returns all integration configurations
func (im *IntegrationManager) ListConfigs() map[string]*types.IntegrationConfig {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*types.IntegrationConfig)
	for name, config := range im.configs {
		configCopy := *config
		result[name] = &configCopy
	}
	return result
}

// GetConfig retrieves a specific integration configuration
func (im *IntegrationManager) GetConfig(name string) (*types.IntegrationConfig, bool) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	config, exists := im.configs[name]
	if !exists {
		return nil, false
	}

	// Return a copy
	configCopy := *config
	return &configCopy, true
}

// CreateConfig creates a new integration configuration
func (im *IntegrationManager) CreateConfig(name string, config *types.IntegrationConfig) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	// Check if already exists
	if _, exists := im.configs[name]; exists {
		return fmt.Errorf("integration %s already exists", name)
	}

	// Set metadata
	config.Name = name
	config.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	config.UpdatedAt = config.CreatedAt

	// Save to disk
	if err := im.saveConfig(name, config); err != nil {
		return err
	}

	// Store in memory
	im.configs[name] = config

	return nil
}

// UpdateConfig updates an existing integration configuration
func (im *IntegrationManager) UpdateConfig(name string, config *types.IntegrationConfig) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	// Check if exists
	if _, exists := im.configs[name]; !exists {
		return fmt.Errorf("integration %s not found", name)
	}

	// Preserve creation time, update modification time
	if existing := im.configs[name]; existing != nil {
		config.CreatedAt = existing.CreatedAt
	}
	config.Name = name
	config.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Save to disk
	if err := im.saveConfig(name, config); err != nil {
		return err
	}

	// Store in memory
	im.configs[name] = config

	return nil
}

// DeleteConfig deletes an integration configuration
func (im *IntegrationManager) DeleteConfig(name string) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	// Check if exists
	if _, exists := im.configs[name]; !exists {
		return fmt.Errorf("integration %s not found", name)
	}

	// Remove from disk
	configFile := filepath.Join(im.configsPath, name+".json")
	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove from memory
	delete(im.configs, name)

	return nil
}

// saveConfig saves a configuration to disk
func (im *IntegrationManager) saveConfig(name string, config *types.IntegrationConfig) error {
	// Ensure directory exists
	if err := os.MkdirAll(im.configsPath, 0755); err != nil {
		return err
	}

	// Serialize configuration
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	configFile := filepath.Join(im.configsPath, name+".json")
	return os.WriteFile(configFile, data, 0644)
}

// SaveIntegrationFile saves an uploaded integration Python file
func (im *IntegrationManager) SaveIntegrationFile(name string, content []byte) error {
	// Ensure directory exists
	if err := os.MkdirAll(im.scriptsPath, 0755); err != nil {
		return err
	}

	// Save file
	filename := filepath.Join(im.scriptsPath, name+".py")
	return os.WriteFile(filename, content, 0644)
}

// DeleteIntegrationFile deletes an integration Python file
func (im *IntegrationManager) DeleteIntegrationFile(name string) error {
	filename := filepath.Join(im.scriptsPath, name+".py")
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// GetIntegrationFileInfo returns information about an integration file
func (im *IntegrationManager) GetIntegrationFileInfo(name string) (*IntegrationFileInfo, error) {
	filename := filepath.Join(im.scriptsPath, name+".py")
	
	stat, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("integration file not found: %s", name)
		}
		return nil, err
	}

	return &IntegrationFileInfo{
		Name:       name,
		Filename:   filename,
		Size:       stat.Size(),
		ModifiedAt: stat.ModTime().Format(time.RFC3339),
		Exists:     true,
	}, nil
}

// IntegrationFileInfo represents information about an integration file
type IntegrationFileInfo struct {
	Name       string `json:"name"`
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
	Exists     bool   `json:"exists"`
}

// ValidateIntegrationConfig validates an integration configuration
func (im *IntegrationManager) ValidateIntegrationConfig(config *types.IntegrationConfig) []types.ValidationError {
	var errors []types.ValidationError

	if config.Name == "" {
		errors = append(errors, types.ValidationError{
			Field:   "name",
			Message: "Integration name is required",
		})
	}

	if config.Type == "" {
		errors = append(errors, types.ValidationError{
			Field:   "type",
			Message: "Integration type is required",
		})
	}

	// Validate type is supported
	supportedTypes := []string{"api", "database", "file", "webhook", "custom"}
	validType := false
	for _, t := range supportedTypes {
		if config.Type == t {
			validType = true
			break
		}
	}
	if !validType {
		errors = append(errors, types.ValidationError{
			Field:   "type",
			Message: "Invalid integration type. Supported types: api, database, file, webhook, custom",
			Value:   config.Type,
		})
	}

	return errors
}

// ClientIntegrationManager manages client-specific integration configurations
type ClientIntegrationManager struct {
	baseManager *IntegrationManager
	clientsPath string
	mutex       sync.RWMutex
}

// NewClientIntegrationManager creates a new client integration manager
func NewClientIntegrationManager(baseManager *IntegrationManager, clientsPath string) *ClientIntegrationManager {
	return &ClientIntegrationManager{
		baseManager: baseManager,
		clientsPath: clientsPath,
	}
}

// GetClientIntegrationPath returns the path for client-specific integrations
func (cim *ClientIntegrationManager) GetClientIntegrationPath(client string) string {
	return filepath.Join(cim.clientsPath, client, "integrations")
}

// ListClientConfigs returns all integration configurations for a specific client
func (cim *ClientIntegrationManager) ListClientConfigs(client string) (map[string]*types.IntegrationConfig, error) {
	clientPath := cim.GetClientIntegrationPath(client)
	
	// Create temporary manager for client-specific configs
	clientManager := NewIntegrationManager(clientPath, filepath.Join(clientPath, "scripts"))
	
	return clientManager.ListConfigs(), nil
}

// GetClientConfig retrieves a client-specific integration configuration
func (cim *ClientIntegrationManager) GetClientConfig(client, name string) (*types.IntegrationConfig, bool) {
	clientPath := cim.GetClientIntegrationPath(client)
	
	// Create temporary manager for client-specific configs
	clientManager := NewIntegrationManager(clientPath, filepath.Join(clientPath, "scripts"))
	
	return clientManager.GetConfig(name)
}

// CreateClientConfig creates a client-specific integration configuration
func (cim *ClientIntegrationManager) CreateClientConfig(client, name string, config *types.IntegrationConfig) error {
	clientPath := cim.GetClientIntegrationPath(client)
	
	// Create temporary manager for client-specific configs
	clientManager := NewIntegrationManager(clientPath, filepath.Join(clientPath, "scripts"))
	
	return clientManager.CreateConfig(name, config)
}

// UpdateClientConfig updates a client-specific integration configuration
func (cim *ClientIntegrationManager) UpdateClientConfig(client, name string, config *types.IntegrationConfig) error {
	clientPath := cim.GetClientIntegrationPath(client)
	
	// Create temporary manager for client-specific configs
	clientManager := NewIntegrationManager(clientPath, filepath.Join(clientPath, "scripts"))
	
	return clientManager.UpdateConfig(name, config)
}

// DeleteClientConfig deletes a client-specific integration configuration
func (cim *ClientIntegrationManager) DeleteClientConfig(client, name string) error {
	clientPath := cim.GetClientIntegrationPath(client)
	
	// Create temporary manager for client-specific configs
	clientManager := NewIntegrationManager(clientPath, filepath.Join(clientPath, "scripts"))
	
	return clientManager.DeleteConfig(name)
}