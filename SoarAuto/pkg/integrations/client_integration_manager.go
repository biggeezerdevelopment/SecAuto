package integrations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"SoarAuto/pkg/security"
	"SoarAuto/pkg/types"
)

// ClientIntegrationManager manages client-specific integration configurations
type ClientIntegrationManager struct {
	clientsPath string
	encryptor   *security.Encryptor
	mutex       sync.RWMutex
	logger      types.Logger
}

// NewClientIntegrationManager creates a new client integration manager
func NewClientIntegrationManager(clientsPath string, logger types.Logger) (*ClientIntegrationManager, error) {
	encryptor, err := security.NewEncryptor()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize encryptor: %v", err)
	}

	return &ClientIntegrationManager{
		clientsPath: clientsPath,
		encryptor:   encryptor,
		logger:      logger,
	}, nil
}

// ClientIntegrationConfig represents a client-specific integration configuration
type ClientIntegrationConfig struct {
	Name        string                 `json:"name"`        // References the global integration
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`      // Client-specific settings
	Credentials map[string]interface{} `json:"credentials"` // Encrypted credentials
	ClientID    string                 `json:"client_id"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

// GetClientIntegrationConfig retrieves a client's configuration for a specific integration
func (cim *ClientIntegrationManager) GetClientIntegrationConfig(clientID, integrationName string) (*ClientIntegrationConfig, error) {
	cim.mutex.RLock()
	defer cim.mutex.RUnlock()

	configPath := filepath.Join(cim.clientsPath, clientID, "integrations", "configs", integrationName+".json")
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("integration config not found for client %s: %s", clientID, integrationName)
		}
		return nil, fmt.Errorf("failed to read integration config: %v", err)
	}

	var config ClientIntegrationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal integration config: %v", err)
	}

	// Decrypt credentials
	if err := cim.decryptCredentials(&config, clientID); err != nil {
		cim.logger.Warning("Failed to decrypt credentials", map[string]interface{}{
			"component":   "client_integration",
			"client_id":   clientID,
			"integration": integrationName,
			"error":      err.Error(),
		})
	}

	return &config, nil
}

// SaveClientIntegrationConfig saves a client's configuration for an integration
func (cim *ClientIntegrationManager) SaveClientIntegrationConfig(clientID string, config *ClientIntegrationConfig) error {
	cim.mutex.Lock()
	defer cim.mutex.Unlock()

	// Ensure the client integration directory exists
	configDir := filepath.Join(cim.clientsPath, clientID, "integrations", "configs")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create integration config directory: %v", err)
	}

	// Set metadata
	config.ClientID = clientID
	if config.CreatedAt == "" {
		config.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	config.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Encrypt credentials before saving
	encryptedConfig := *config
	if err := cim.encryptCredentials(&encryptedConfig, clientID); err != nil {
		return fmt.Errorf("failed to encrypt credentials: %v", err)
	}

	// Marshal and save
	data, err := json.MarshalIndent(encryptedConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal integration config: %v", err)
	}

	configPath := filepath.Join(configDir, config.Name+".json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write integration config: %v", err)
	}

	cim.logger.Info("Client integration config saved", map[string]interface{}{
		"component":   "client_integration",
		"client_id":   clientID,
		"integration": config.Name,
	})

	return nil
}

// ListClientIntegrations lists all configured integrations for a client
func (cim *ClientIntegrationManager) ListClientIntegrations(clientID string) ([]*ClientIntegrationConfig, error) {
	cim.mutex.RLock()
	defer cim.mutex.RUnlock()

	configDir := filepath.Join(cim.clientsPath, clientID, "integrations", "configs")
	
	// Check if directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return []*ClientIntegrationConfig{}, nil
	}

	files, err := filepath.Glob(filepath.Join(configDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list integration configs: %v", err)
	}

	var configs []*ClientIntegrationConfig
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			cim.logger.Warning("Failed to read integration config", map[string]interface{}{
				"component": "client_integration",
				"file":      file,
				"error":     err.Error(),
			})
			continue
		}

		var config ClientIntegrationConfig
		if err := json.Unmarshal(data, &config); err != nil {
			cim.logger.Warning("Failed to unmarshal integration config", map[string]interface{}{
				"component": "client_integration",
				"file":      file,
				"error":     err.Error(),
			})
			continue
		}

		// Decrypt credentials
		if err := cim.decryptCredentials(&config, clientID); err != nil {
			cim.logger.Warning("Failed to decrypt credentials", map[string]interface{}{
				"component":   "client_integration",
				"client_id":   clientID,
				"integration": config.Name,
				"error":      err.Error(),
			})
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

// DeleteClientIntegrationConfig removes a client's integration configuration
func (cim *ClientIntegrationManager) DeleteClientIntegrationConfig(clientID, integrationName string) error {
	cim.mutex.Lock()
	defer cim.mutex.Unlock()

	configPath := filepath.Join(cim.clientsPath, clientID, "integrations", "configs", integrationName+".json")
	
	if err := os.Remove(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("integration config not found for client %s: %s", clientID, integrationName)
		}
		return fmt.Errorf("failed to delete integration config: %v", err)
	}

	cim.logger.Info("Client integration config deleted", map[string]interface{}{
		"component":   "client_integration",
		"client_id":   clientID,
		"integration": integrationName,
	})

	return nil
}

// ValidateClientConfig validates a client's integration configuration against the global integration
func (cim *ClientIntegrationManager) ValidateClientConfig(config *ClientIntegrationConfig, globalIntegration *IntegrationDefinition) error {
	// Check if the integration name matches
	if config.Name != globalIntegration.Name {
		return fmt.Errorf("integration name mismatch: config references '%s' but validating against '%s'", 
			config.Name, globalIntegration.Name)
	}

	// Validate required configuration fields
	for key, fieldDef := range globalIntegration.Configuration {
		if fieldDef.Required {
			// Check in both config and credentials
			_, inConfig := config.Config[key]
			_, inCredentials := config.Credentials[key]
			
			if !inConfig && !inCredentials {
				return fmt.Errorf("required field '%s' not found in client configuration", key)
			}
		}
	}

	return nil
}

// Helper functions for encryption/decryption

func (cim *ClientIntegrationManager) encryptCredentials(config *ClientIntegrationConfig, clientID string) error {
	if config.Credentials == nil || len(config.Credentials) == 0 {
		return nil
	}

	// Get client's encryption key
	keyPath := filepath.Join(cim.clientsPath, clientID, ".encryption_key")
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read encryption key: %v", err)
	}

	// Encrypt each credential field
	encryptedCreds := make(map[string]interface{})
	for key, value := range config.Credentials {
		valueStr, ok := value.(string)
		if !ok {
			encryptedCreds[key] = value
			continue
		}

		encrypted, err := cim.encryptor.Encrypt([]byte(valueStr), string(keyData))
		if err != nil {
			return fmt.Errorf("failed to encrypt credential '%s': %v", key, err)
		}
		encryptedCreds[key] = encrypted
	}

	config.Credentials = encryptedCreds
	return nil
}

func (cim *ClientIntegrationManager) decryptCredentials(config *ClientIntegrationConfig, clientID string) error {
	if config.Credentials == nil || len(config.Credentials) == 0 {
		return nil
	}

	// Get client's encryption key
	keyPath := filepath.Join(cim.clientsPath, clientID, ".encryption_key")
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read encryption key: %v", err)
	}

	// Decrypt each credential field
	decryptedCreds := make(map[string]interface{})
	for key, value := range config.Credentials {
		valueStr, ok := value.(string)
		if !ok {
			decryptedCreds[key] = value
			continue
		}

		// Check if it looks like an encrypted value
		if len(valueStr) > 0 && (len(valueStr)%4 == 0 || strings.Contains(valueStr, "=")) {
			decrypted, err := cim.encryptor.Decrypt([]byte(valueStr), string(keyData))
			if err != nil {
				// If decryption fails, keep original value
				decryptedCreds[key] = value
			} else {
				decryptedCreds[key] = string(decrypted)
			}
		} else {
			decryptedCreds[key] = value
		}
	}

	config.Credentials = decryptedCreds
	return nil
}