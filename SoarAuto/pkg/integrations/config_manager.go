package integrations

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/redis/go-redis/v9"
	"context"
	"SoarAuto/pkg/types"
)

// ConfigManager handles hybrid storage of client integration configurations
type ConfigManager struct {
	db            *sql.DB
	redis         *redis.Client
	encryptionKey []byte
	logger        types.Logger
	cacheTTL      time.Duration
	dataPath      string // Fallback file storage path when database is not available
}

// DBClientIntegrationConfig represents a client's integration configuration in the database
type DBClientIntegrationConfig struct {
	ID              string                 `json:"id" db:"id"`
	ClientID        string                 `json:"client_id" db:"client_id"`
	IntegrationName string                 `json:"integration_name" db:"integration_name"`
	Enabled         bool                   `json:"enabled" db:"enabled"`
	Config          map[string]interface{} `json:"config"`
	Credentials     map[string]interface{} `json:"credentials,omitempty"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// NewConfigManager creates a new hybrid configuration manager
func NewConfigManager(db *sql.DB, redis *redis.Client, encryptionKey string, logger types.Logger) *ConfigManager {
	// Derive 32-byte key from provided string
	hash := sha256.Sum256([]byte(encryptionKey))
	
	return &ConfigManager{
		db:            db,
		redis:         redis,
		encryptionKey: hash[:],
		logger:        logger,
		cacheTTL:      5 * time.Minute,
		dataPath:      "", // No fallback path when using database
	}
}

// NewConfigManagerWithFallback creates a new configuration manager with file-based fallback
func NewConfigManagerWithFallback(db *sql.DB, redis *redis.Client, encryptionKey string, dataPath string, logger types.Logger) *ConfigManager {
	// Derive 32-byte key from provided string
	hash := sha256.Sum256([]byte(encryptionKey))
	
	return &ConfigManager{
		db:            db,
		redis:         redis,
		encryptionKey: hash[:],
		logger:        logger,
		cacheTTL:      5 * time.Minute,
		dataPath:      dataPath,
	}
}

// GetClientIntegrationConfig retrieves a client's integration configuration
func (cm *ConfigManager) GetClientIntegrationConfig(clientID, integrationName string) (*DBClientIntegrationConfig, error) {
	// Try Redis cache first
	cacheKey := cm.getCacheKey(clientID, integrationName)
	
	if cm.redis != nil {
		ctx := context.Background()
		cached, err := cm.redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var config DBClientIntegrationConfig
			if err := json.Unmarshal([]byte(cached), &config); err == nil {
				cm.logger.Debug("Retrieved config from cache", map[string]interface{}{
					"client_id": clientID,
					"integration": integrationName,
					"cache_key": cacheKey,
				})
				return &config, nil
			} else {
				cm.logger.Warning("Failed to unmarshal cached config", map[string]interface{}{
					"client_id": clientID,
					"integration": integrationName,
					"cache_key": cacheKey,
					"error": err.Error(),
				})
			}
		} else if err != nil && err != redis.Nil {
			cm.logger.Warning("Redis error when retrieving config", map[string]interface{}{
				"client_id": clientID,
				"integration": integrationName,
				"cache_key": cacheKey,
				"error": err.Error(),
			})
		} else {
			cm.logger.Debug("Cache miss for config", map[string]interface{}{
				"client_id": clientID,
				"integration": integrationName,
				"cache_key": cacheKey,
			})
		}
	}
	
	// Fallback to database
	cm.logger.Debug("Fetching config from database", map[string]interface{}{
		"client_id": clientID,
		"integration": integrationName,
	})
	
	config, err := cm.getFromDatabase(clientID, integrationName)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	if cm.redis != nil && config != nil {
		cm.cacheConfig(config)
	}
	
	return config, nil
}

// SaveClientIntegrationConfig saves or updates a client's integration configuration
func (cm *ConfigManager) SaveClientIntegrationConfig(config *DBClientIntegrationConfig) error {
	if cm.db == nil {
		return fmt.Errorf("database not available - cannot save integration config")
	}
	
	// Encrypt the configuration
	encryptedConfig, err := cm.encryptConfig(config.Config, config.Credentials)
	if err != nil {
		return fmt.Errorf("failed to encrypt config: %v", err)
	}
	
	// Save to database
	query := `
		INSERT INTO client_integration_configs (client_id, integration_name, enabled, config_encrypted)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (client_id, integration_name) 
		DO UPDATE SET 
			enabled = EXCLUDED.enabled,
			config_encrypted = EXCLUDED.config_encrypted,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`
	
	err = cm.db.QueryRow(query, 
		config.ClientID, 
		config.IntegrationName, 
		config.Enabled, 
		encryptedConfig,
	).Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to save config to database: %v", err)
	}
	
	// Invalidate cache
	if cm.redis != nil {
		cacheKey := cm.getCacheKey(config.ClientID, config.IntegrationName)
		ctx := context.Background()
		cm.redis.Del(ctx, cacheKey)
		
		// Cache the new config
		cm.cacheConfig(config)
	}
	
	cm.logger.Info("Saved client integration config", map[string]interface{}{
		"client_id": config.ClientID,
		"integration": config.IntegrationName,
		"enabled": config.Enabled,
	})
	
	return nil
}

// ListClientIntegrations returns all integrations for a client
func (cm *ConfigManager) ListClientIntegrations(clientID string) ([]*DBClientIntegrationConfig, error) {
	if cm.db == nil {
		return []*DBClientIntegrationConfig{}, nil
	}
	
	query := `
		SELECT id, client_id, integration_name, enabled, config_encrypted, created_at, updated_at
		FROM client_integration_configs 
		WHERE client_id = $1 
		ORDER BY integration_name`
	
	rows, err := cm.db.Query(query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to list integrations: %v", err)
	}
	defer rows.Close()
	
	var configs []*DBClientIntegrationConfig
	for rows.Next() {
		var config DBClientIntegrationConfig
		var encryptedConfig string
		
		err := rows.Scan(
			&config.ID,
			&config.ClientID,
			&config.IntegrationName,
			&config.Enabled,
			&encryptedConfig,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			continue
		}
		
		// Decrypt configuration
		if err := cm.decryptConfig(encryptedConfig, &config); err != nil {
			cm.logger.Warning("Failed to decrypt config", map[string]interface{}{
				"client_id": clientID,
				"integration": config.IntegrationName,
				"error": err.Error(),
			})
			continue
		}
		
		configs = append(configs, &config)
	}
	
	return configs, nil
}

// DeleteClientIntegrationConfig removes a client's integration configuration
func (cm *ConfigManager) DeleteClientIntegrationConfig(clientID, integrationName string) error {
	if cm.db == nil {
		return fmt.Errorf("database not available - cannot delete integration config")
	}
	
	query := `DELETE FROM client_integration_configs WHERE client_id = $1 AND integration_name = $2`
	
	result, err := cm.db.Exec(query, clientID, integrationName)
	if err != nil {
		return fmt.Errorf("failed to delete config: %v", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("integration config not found")
	}
	
	// Invalidate cache
	if cm.redis != nil {
		cacheKey := cm.getCacheKey(clientID, integrationName)
		ctx := context.Background()
		cm.redis.Del(ctx, cacheKey)
	}
	
	cm.logger.Info("Deleted client integration config", map[string]interface{}{
		"client_id": clientID,
		"integration": integrationName,
	})
	
	return nil
}

// Private helper methods

func (cm *ConfigManager) getCacheKey(clientID, integrationName string) string {
	return fmt.Sprintf("client:%s:integration:%s", clientID, integrationName)
}


func (cm *ConfigManager) getFromDatabase(clientID, integrationName string) (*DBClientIntegrationConfig, error) {
	if cm.db == nil {
		// If database is not available but we have a data path, try file-based storage
		if cm.dataPath != "" {
			cm.logger.Debug("Database not available, trying file-based fallback", map[string]interface{}{
				"client_id": clientID,
				"integration": integrationName,
				"data_path": cm.dataPath,
			})
			return cm.getFromFile(clientID, integrationName)
		}
		
		cm.logger.Debug("Database not available and no fallback path configured", map[string]interface{}{
			"client_id": clientID,
			"integration": integrationName,
		})
		return nil, fmt.Errorf("database not available - integration config not found")
	}
	
	query := `
		SELECT id, client_id, integration_name, enabled, config_encrypted, created_at, updated_at
		FROM client_integration_configs 
		WHERE client_id = $1 AND integration_name = $2`
	
	var config DBClientIntegrationConfig
	var encryptedConfig string
	
	err := cm.db.QueryRow(query, clientID, integrationName).Scan(
		&config.ID,
		&config.ClientID,
		&config.IntegrationName,
		&config.Enabled,
		&encryptedConfig,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("integration config not found")
		}
		return nil, fmt.Errorf("failed to query config: %v", err)
	}
	
	// Decrypt configuration
	if err := cm.decryptConfig(encryptedConfig, &config); err != nil {
		return nil, fmt.Errorf("failed to decrypt config: %v", err)
	}
	
	// Log what we retrieved from database
	cm.logger.Debug("Retrieved config from database", map[string]interface{}{
		"client_id": config.ClientID,
		"integration": config.IntegrationName,
		"enabled": config.Enabled,
		"has_config": config.Config != nil && len(config.Config) > 0,
		"has_credentials": config.Credentials != nil && len(config.Credentials) > 0,
		"config_keys": getMapKeys(config.Config),
		"credential_keys": getMapKeys(config.Credentials),
	})
	
	return &config, nil
}

func (cm *ConfigManager) cacheConfig(config *DBClientIntegrationConfig) {
	ctx := context.Background()
	cacheKey := cm.getCacheKey(config.ClientID, config.IntegrationName)
	
	configJSON, err := json.Marshal(config)
	if err != nil {
		cm.logger.Warning("Failed to marshal config for cache", map[string]interface{}{
			"client_id": config.ClientID,
			"integration": config.IntegrationName,
			"cache_key": cacheKey,
			"error": err.Error(),
		})
		return
	}
	
	err = cm.redis.SetEx(ctx, cacheKey, configJSON, cm.cacheTTL).Err()
	if err != nil {
		cm.logger.Warning("Failed to cache config", map[string]interface{}{
			"client_id": config.ClientID,
			"integration": config.IntegrationName,
			"cache_key": cacheKey,
			"error": err.Error(),
		})
	} else {
		cm.logger.Debug("Successfully cached config", map[string]interface{}{
			"client_id": config.ClientID,
			"integration": config.IntegrationName,
			"cache_key": cacheKey,
			"ttl_seconds": cm.cacheTTL.Seconds(),
		})
	}
}

func (cm *ConfigManager) encryptConfig(config, credentials map[string]interface{}) (string, error) {
	// Combine config and credentials
	combined := map[string]interface{}{
		"config": config,
		"credentials": credentials,
	}
	
	// Marshal to JSON
	data, err := json.Marshal(combined)
	if err != nil {
		return "", err
	}
	
	// Create cipher
	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", err
	}
	
	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	
	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	
	// Encode to base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (cm *ConfigManager) decryptConfig(encryptedData string, config *DBClientIntegrationConfig) error {
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return err
	}
	
	// Create cipher
	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return err
	}
	
	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	
	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return fmt.Errorf("ciphertext too short")
	}
	
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	
	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}
	
	// Unmarshal JSON
	var combined map[string]interface{}
	if err := json.Unmarshal(plaintext, &combined); err != nil {
		return err
	}
	
	// Extract config and credentials
	if configData, ok := combined["config"]; ok {
		if configMap, ok := configData.(map[string]interface{}); ok {
			config.Config = configMap
		}
	}
	
	if credentialsData, ok := combined["credentials"]; ok {
		if credentialsMap, ok := credentialsData.(map[string]interface{}); ok {
			config.Credentials = credentialsMap
		}
	}
	
	return nil
}

// MigrateFromFileSystem migrates existing file-based configs to database
func (cm *ConfigManager) MigrateFromFileSystem(dataPath string) error {
	if cm.db == nil {
		return fmt.Errorf("database not available for migration")
	}
	
	cm.logger.Info("Starting migration from file system to database", map[string]interface{}{
		"data_path": dataPath,
	})
	
	clientsPath := filepath.Join(dataPath, "clients")
	
	// Check if clients directory exists
	if _, err := os.Stat(clientsPath); os.IsNotExist(err) {
		cm.logger.Info("No clients directory found, migration complete", nil)
		return nil
	}
	
	// Walk through all client directories
	clientDirs, err := filepath.Glob(filepath.Join(clientsPath, "*"))
	if err != nil {
		return fmt.Errorf("failed to list client directories: %v", err)
	}
	
	migratedCount := 0
	for _, clientDir := range clientDirs {
		if !isDirectory(clientDir) {
			continue
		}
		
		clientID := filepath.Base(clientDir)
		integrationConfigsPath := filepath.Join(clientDir, "integrations", "configs")
		
		// Check if integration configs directory exists
		if _, err := os.Stat(integrationConfigsPath); os.IsNotExist(err) {
			continue
		}
		
		// Read all JSON config files
		configFiles, err := filepath.Glob(filepath.Join(integrationConfigsPath, "*.json"))
		if err != nil {
			cm.logger.Warning("Failed to list config files", map[string]interface{}{
				"client_id": clientID,
				"error": err.Error(),
			})
			continue
		}
		
		for _, configFile := range configFiles {
			if err := cm.migrateConfigFile(clientID, configFile); err != nil {
				cm.logger.Warning("Failed to migrate config file", map[string]interface{}{
					"client_id": clientID,
					"file": configFile,
					"error": err.Error(),
				})
			} else {
				migratedCount++
			}
		}
	}
	
	cm.logger.Info("Migration completed", map[string]interface{}{
		"migrated_configs": migratedCount,
	})
	
	return nil
}

func (cm *ConfigManager) migrateConfigFile(clientID, configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	
	var legacyConfig struct {
		Name        string                 `json:"name"`
		Enabled     bool                   `json:"enabled"`
		Config      map[string]interface{} `json:"config"`
		Credentials map[string]interface{} `json:"credentials"`
		ClientID    string                 `json:"client_id"`
		CreatedAt   string                 `json:"created_at"`
		UpdatedAt   string                 `json:"updated_at"`
	}
	
	if err := json.Unmarshal(data, &legacyConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %v", err)
	}
	
	// Parse timestamps
	createdAt, _ := time.Parse("2006-01-02T15:04:05Z", legacyConfig.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt, _ := time.Parse("2006-01-02T15:04:05Z", legacyConfig.UpdatedAt)
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	
	// Convert to database model
	dbConfig := &DBClientIntegrationConfig{
		ClientID:        clientID,
		IntegrationName: legacyConfig.Name,
		Enabled:         legacyConfig.Enabled,
		Config:          legacyConfig.Config,
		Credentials:     legacyConfig.Credentials,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
	
	// Save to database
	return cm.SaveClientIntegrationConfig(dbConfig)
}

func (cm *ConfigManager) getFromFile(clientID, integrationName string) (*DBClientIntegrationConfig, error) {
	// Build the file path
	configFile := filepath.Join(cm.dataPath, clientID, "integrations", "configs", integrationName+".json")
	
	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		cm.logger.Debug("Config file not found", map[string]interface{}{
			"client_id": clientID,
			"integration": integrationName,
			"file_path": configFile,
		})
		return nil, fmt.Errorf("integration config not found")
	}
	
	// Read the file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}
	
	// Parse the legacy format
	var legacyConfig struct {
		Name        string                 `json:"name"`
		Enabled     bool                   `json:"enabled"`
		Config      map[string]interface{} `json:"config"`
		Credentials map[string]interface{} `json:"credentials"`
		ClientID    string                 `json:"client_id"`
		CreatedAt   string                 `json:"created_at"`
		UpdatedAt   string                 `json:"updated_at"`
	}
	
	if err := json.Unmarshal(data, &legacyConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	
	// Parse timestamps
	createdAt, _ := time.Parse("2006-01-02T15:04:05Z", legacyConfig.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt, _ := time.Parse("2006-01-02T15:04:05Z", legacyConfig.UpdatedAt)
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	
	// Convert to database model
	result := &DBClientIntegrationConfig{
		ID:              fmt.Sprintf("%s_%s", clientID, integrationName), // Generate an ID
		ClientID:        clientID,
		IntegrationName: integrationName,
		Enabled:         legacyConfig.Enabled,
		Config:          legacyConfig.Config,
		Credentials:     legacyConfig.Credentials,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
	
	// Log what we read from file
	cm.logger.Debug("Retrieved config from file", map[string]interface{}{
		"client_id": clientID,
		"integration": integrationName,
		"file_path": configFile,
		"enabled": result.Enabled,
		"has_config": result.Config != nil && len(result.Config) > 0,
		"has_credentials": result.Credentials != nil && len(result.Credentials) > 0,
		"config_keys": getMapKeys(result.Config),
		"credential_keys": getMapKeys(result.Credentials),
	})
	
	return result, nil
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}