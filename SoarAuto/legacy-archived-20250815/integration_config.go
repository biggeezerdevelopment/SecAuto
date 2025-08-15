package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// IntegrationConfig represents a single integration configuration
type IntegrationConfig struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Client      string                 `json:"client,omitempty"`
	URL         string                 `json:"url,omitempty"`
	APIKey      string                 `json:"apikey,omitempty"`
	Username    string                 `json:"username,omitempty"`
	Password    string                 `json:"password,omitempty"`
	Token       string                 `json:"token,omitempty"`
	Secret      string                 `json:"secret,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Description string                 `json:"description,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// IntegrationConfigManager handles integration configurations with encryption
type IntegrationConfigManager struct {
	configPath    string
	encryptionKey []byte
	mutex         sync.RWMutex
	configs       map[string]*IntegrationConfig
}

// NewIntegrationConfigManager creates a new integration config manager
func NewIntegrationConfigManager(configPath string, encryptionKey string) (*IntegrationConfigManager, error) {
	// Generate encryption key from password using SHA256
	hash := sha256.Sum256([]byte(encryptionKey))
	key := hash[:]

	manager := &IntegrationConfigManager{
		configPath:    configPath,
		encryptionKey: key,
		configs:       make(map[string]*IntegrationConfig),
	}

	// Load existing configurations
	if err := manager.loadConfigs(); err != nil {
		return nil, fmt.Errorf("failed to load integration configs: %v", err)
	}

	return manager, nil
}

// generateConfigKey generates a unique key for storing integration configs
// Format: "client:integration_name" or just "integration_name" for global configs
func (icm *IntegrationConfigManager) generateConfigKey(client, integrationName string) string {
	if client != "" {
		return fmt.Sprintf("%s:%s", client, integrationName)
	}
	return integrationName
}

// parseConfigKey parses a config key to extract client and integration name
func (icm *IntegrationConfigManager) parseConfigKey(key string) (client, integrationName string) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}

// loadConfigs loads all integration configurations from disk
func (icm *IntegrationConfigManager) loadConfigs() error {
	icm.mutex.Lock()
	defer icm.mutex.Unlock()

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(icm.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Check if config file exists
	if _, err := os.Stat(icm.configPath); os.IsNotExist(err) {
		// Create empty config file
		emptyConfig := map[string]*IntegrationConfig{}
		if err := icm.saveConfigsToFile(emptyConfig); err != nil {
			return fmt.Errorf("failed to create empty config file: %v", err)
		}
		return nil
	}

	// Read and decrypt config file
	data, err := os.ReadFile(icm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	// Decrypt the data
	decryptedData, err := icm.decrypt(data)
	if err != nil {
		return fmt.Errorf("failed to decrypt config file: %v", err)
	}

	// Parse JSON
	var configs map[string]*IntegrationConfig
	if err := json.Unmarshal(decryptedData, &configs); err != nil {
		return fmt.Errorf("failed to parse config JSON: %v", err)
	}

	icm.configs = configs
	return nil
}

// saveConfigsToFile saves configurations to disk with encryption
func (icm *IntegrationConfigManager) saveConfigsToFile(configs map[string]*IntegrationConfig) error {
	// Convert to JSON
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configs: %v", err)
	}

	// Encrypt the data
	encryptedData, err := icm.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt configs: %v", err)
	}

	// Write to file
	if err := os.WriteFile(icm.configPath, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// SaveConfigs saves all configurations to disk
func (icm *IntegrationConfigManager) SaveConfigs() error {
	icm.mutex.Lock()
	defer icm.mutex.Unlock()

	return icm.saveConfigsToFile(icm.configs)
}

// GetConfig retrieves a specific integration configuration (global or client-specific)
func (icm *IntegrationConfigManager) GetConfig(integrationName string) (*IntegrationConfig, bool) {
	icm.mutex.RLock()
	defer icm.mutex.RUnlock()

	config, exists := icm.configs[integrationName]
	return config, exists
}

// GetConfigByClient retrieves a client-specific integration configuration
func (icm *IntegrationConfigManager) GetConfigByClient(client, integrationName string) (*IntegrationConfig, bool) {
	icm.mutex.RLock()
	defer icm.mutex.RUnlock()

	key := icm.generateConfigKey(client, integrationName)
	config, exists := icm.configs[key]
	if !exists {
		return nil, false
	}

	// Check if the integration is enabled
	if !config.Enabled {
		return nil, false
	}

	return config, true
}

// GetConfigIntegrationByClient retrieves a client-specific integration configuration (legacy method)
func (icm *IntegrationConfigManager) GetConfigIntegrationByClient(integrationName string, clientName string) (*IntegrationConfig, bool) {
	return icm.GetConfigByClient(clientName, integrationName)
}

// SetConfig sets or updates an integration configuration
func (icm *IntegrationConfigManager) SetConfig(integrationName string, config *IntegrationConfig) error {
	icm.mutex.Lock()
	defer icm.mutex.Unlock()

	// Set timestamps
	now := time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	// Generate the appropriate key based on whether it's client-specific
	key := icm.generateConfigKey(config.Client, integrationName)
	icm.configs[key] = config

	// Save to disk
	return icm.saveConfigsToFile(icm.configs)
}

// SetConfigByClient sets or updates a client-specific integration configuration
func (icm *IntegrationConfigManager) SetConfigByClient(client, integrationName string, config *IntegrationConfig) error {
	// Set the client field
	config.Client = client
	return icm.SetConfig(integrationName, config)
}

// DeleteConfig removes an integration configuration
func (icm *IntegrationConfigManager) DeleteConfig(integrationName string) error {
	icm.mutex.Lock()
	defer icm.mutex.Unlock()

	delete(icm.configs, integrationName)

	// Save to disk
	return icm.saveConfigsToFile(icm.configs)
}

// DeleteConfigByClient removes a client-specific integration configuration
func (icm *IntegrationConfigManager) DeleteConfigByClient(client, integrationName string) error {
	icm.mutex.Lock()
	defer icm.mutex.Unlock()

	key := icm.generateConfigKey(client, integrationName)
	delete(icm.configs, key)

	// Save to disk
	return icm.saveConfigsToFile(icm.configs)
}

// ListConfigs returns all integration configurations
func (icm *IntegrationConfigManager) ListConfigs() map[string]*IntegrationConfig {
	icm.mutex.RLock()
	defer icm.mutex.RUnlock()

	// Create a copy to avoid race conditions
	configs := make(map[string]*IntegrationConfig)
	for name, config := range icm.configs {
		configs[name] = config
	}

	return configs
}

// ListConfigsByClient returns all integration configurations for a specific client
func (icm *IntegrationConfigManager) ListConfigsByClient(client string) map[string]*IntegrationConfig {
	icm.mutex.RLock()
	defer icm.mutex.RUnlock()

	configs := make(map[string]*IntegrationConfig)
	for key, config := range icm.configs {
		configClient, integrationName := icm.parseConfigKey(key)
		if configClient == client {
			configs[integrationName] = config
		}
	}

	return configs
}

// ListGlobalConfigs returns all global integration configurations (no client specified)
func (icm *IntegrationConfigManager) ListGlobalConfigs() map[string]*IntegrationConfig {
	icm.mutex.RLock()
	defer icm.mutex.RUnlock()

	configs := make(map[string]*IntegrationConfig)
	for key, config := range icm.configs {
		configClient, _ := icm.parseConfigKey(key)
		if configClient == "" {
			configs[key] = config
		}
	}

	return configs
}

// GetConfigValue retrieves a specific value from an integration configuration
func (icm *IntegrationConfigManager) GetConfigValue(integrationName, key string) (string, bool) {
	config, exists := icm.GetConfig(integrationName)
	if !exists {
		return "", false
	}

	switch key {
	case "apikey":
		return config.APIKey, true
	case "url":
		return config.URL, true
	case "username":
		return config.Username, true
	case "password":
		return config.Password, true
	case "token":
		return config.Token, true
	case "secret":
		return config.Secret, true
	default:
		// Check in settings map
		if config.Settings != nil {
			if value, exists := config.Settings[key]; exists {
				if strValue, ok := value.(string); ok {
					return strValue, true
				}
			}
		}
		return "", false
	}
}

// GetConfigValueByClient retrieves a specific value from a client-specific integration configuration
func (icm *IntegrationConfigManager) GetConfigValueByClient(client, integrationName, key string) (string, bool) {
	config, exists := icm.GetConfigByClient(client, integrationName)
	if !exists {
		return "", false
	}

	switch key {
	case "apikey":
		return config.APIKey, true
	case "url":
		return config.URL, true
	case "username":
		return config.Username, true
	case "password":
		return config.Password, true
	case "token":
		return config.Token, true
	case "secret":
		return config.Secret, true
	default:
		// Check in settings map
		if config.Settings != nil {
			if value, exists := config.Settings[key]; exists {
				if strValue, ok := value.(string); ok {
					return strValue, true
				}
			}
		}
		return "", false
	}
}

// encrypt encrypts data using AES-256-GCM
func (icm *IntegrationConfigManager) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(icm.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-256-GCM
func (icm *IntegrationConfigManager) decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(icm.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// CreateDefaultConfigs creates default integration configurations
func (icm *IntegrationConfigManager) CreateDefaultConfigs() error {
	defaultConfigs := map[string]*IntegrationConfig{
		"virustotal": {
			Name:        "virustotal",
			Type:        "virustotal",
			URL:         "https://www.virustotal.com/vtapi/v2",
			APIKey:      "",
			Enabled:     false,
			Description: "VirusTotal URL and file scanning integration",
			Version:     "1.0.0",
			Settings: map[string]interface{}{
				"timeout": 60,
				"retries": 3,
			},
		},
		"slack": {
			Name:        "slack",
			Type:        "slack",
			URL:         "https://hooks.slack.com/services/",
			Token:       "",
			Enabled:     false,
			Description: "Slack webhook notifications",
			Version:     "1.0.0",
			Settings: map[string]interface{}{
				"channel":  "#security",
				"username": "SecAuto Bot",
			},
		},
		"email": {
			Name:        "email",
			Type:        "email",
			URL:         "smtp.gmail.com:587",
			Username:    "",
			Password:    "",
			Enabled:     false,
			Description: "Email notification integration",
			Version:     "1.0.0",
			Settings: map[string]interface{}{
				"from": "secauto@company.com",
				"to":   "security@company.com",
			},
		},
	}

	for name, config := range defaultConfigs {
		if err := icm.SetConfig(name, config); err != nil {
			return fmt.Errorf("failed to create default config for %s: %v", name, err)
		}
	}

	return nil
}

// ValidateConfig validates an integration configuration
func (icm *IntegrationConfigManager) ValidateConfig(config *IntegrationConfig) error {
	if config.Type == "" {
		return fmt.Errorf("integration type is required")
	}

	// Basic validation - allow empty credentials for initial setup
	// Users can configure credentials later through the API

	return nil
}
