package integrations

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
	"SoarAuto/pkg/types"
)

// ClientIntegrationManager manages client-specific integration configurations
// Now uses hybrid database + Redis storage instead of files
type ClientIntegrationManager struct {
	configManager *ConfigManager
	mutex         sync.RWMutex
	logger        types.Logger
}

// NewClientIntegrationManager creates a new client integration manager
func NewClientIntegrationManager(db *sql.DB, redis *redis.Client, encryptionKey string, logger types.Logger) (*ClientIntegrationManager, error) {
	configManager := NewConfigManager(db, redis, encryptionKey, logger)

	return &ClientIntegrationManager{
		configManager: configManager,
		logger:        logger,
	}, nil
}

// NewClientIntegrationManagerWithLegacyPath creates a new client integration manager with legacy file-based storage
// This is a temporary function to maintain compatibility until database migration is complete
func NewClientIntegrationManagerWithLegacyPath(clientsPath string, logger types.Logger) (*ClientIntegrationManager, error) {
	// For now, use Redis-only config manager with nil database
	configManager := NewConfigManager(nil, nil, "default-encryption-key", logger)

	return &ClientIntegrationManager{
		configManager: configManager,
		logger:        logger,
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

	// Use the hybrid ConfigManager instead of file system
	dbConfig, err := cim.configManager.GetClientIntegrationConfig(clientID, integrationName)
	if err != nil {
		return nil, fmt.Errorf("failed to get client integration config: %v", err)
	}

	// Convert from database model to the existing ClientIntegrationConfig format
	config := &ClientIntegrationConfig{
		Name:        dbConfig.IntegrationName,
		Enabled:     dbConfig.Enabled,
		Config:      dbConfig.Config,
		Credentials: dbConfig.Credentials,
		ClientID:    dbConfig.ClientID,
		CreatedAt:   dbConfig.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   dbConfig.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	return config, nil
}

// SaveClientIntegrationConfig saves a client's configuration for an integration
func (cim *ClientIntegrationManager) SaveClientIntegrationConfig(clientID string, config *ClientIntegrationConfig) error {
	cim.mutex.Lock()
	defer cim.mutex.Unlock()

	// Convert to database model
	dbConfig := &DBClientIntegrationConfig{
		ClientID:        clientID,
		IntegrationName: config.Name,
		Enabled:         config.Enabled,
		Config:          config.Config,
		Credentials:     config.Credentials,
	}

	// Use the hybrid ConfigManager to save
	return cim.configManager.SaveClientIntegrationConfig(dbConfig)
}

// ListClientIntegrations lists all configured integrations for a client
func (cim *ClientIntegrationManager) ListClientIntegrations(clientID string) ([]*ClientIntegrationConfig, error) {
	cim.mutex.RLock()
	defer cim.mutex.RUnlock()

	// Use the hybrid ConfigManager to list integrations
	dbConfigs, err := cim.configManager.ListClientIntegrations(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list client integrations: %v", err)
	}

	// Convert from database model to the existing ClientIntegrationConfig format
	var configs []*ClientIntegrationConfig
	for _, dbConfig := range dbConfigs {
		config := &ClientIntegrationConfig{
			Name:        dbConfig.IntegrationName,
			Enabled:     dbConfig.Enabled,
			Config:      dbConfig.Config,
			Credentials: dbConfig.Credentials,
			ClientID:    dbConfig.ClientID,
			CreatedAt:   dbConfig.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   dbConfig.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// DeleteClientIntegrationConfig removes a client's integration configuration
func (cim *ClientIntegrationManager) DeleteClientIntegrationConfig(clientID, integrationName string) error {
	cim.mutex.Lock()
	defer cim.mutex.Unlock()

	// Use the hybrid ConfigManager to delete
	return cim.configManager.DeleteClientIntegrationConfig(clientID, integrationName)
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

