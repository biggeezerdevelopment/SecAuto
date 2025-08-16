package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"SoarAuto/pkg/errors"
	"SoarAuto/pkg/types"
)

// APIKeyManager manages API keys for authentication
type APIKeyManager struct {
	keys        map[string]*types.APIKey
	mutex       sync.RWMutex
	keysFile    string
	configKeys  []string
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(keysFile string, configKeys []string) *APIKeyManager {
	akm := &APIKeyManager{
		keys:       make(map[string]*types.APIKey),
		keysFile:   keysFile,
		configKeys: configKeys,
	}

	// Load config keys first
	akm.loadConfigKeys()

	// Load persisted keys from file
	akm.loadKeysFromFile()

	return akm
}

// loadConfigKeys loads API keys from configuration
func (akm *APIKeyManager) loadConfigKeys() {
	for _, key := range akm.configKeys {
		if key != "" {
			akm.keys[key] = &types.APIKey{
				Key:         key,
				Name:        "Config Key",
				Description: "API key from configuration file",
				CreatedAt:   time.Now().UTC().Format(time.RFC3339),
				CreatedBy:   "system",
				Active:      true,
				Source:      "config",
			}
		}
	}
}

// loadKeysFromFile loads persisted API keys from file
func (akm *APIKeyManager) loadKeysFromFile() {
	if _, err := os.Stat(akm.keysFile); os.IsNotExist(err) {
		return
	}

	data, err := os.ReadFile(akm.keysFile)
	if err != nil {
		return
	}

	var persistedKeys map[string]*types.APIKey
	if err := json.Unmarshal(data, &persistedKeys); err != nil {
		return
	}

	// Merge persisted keys with config keys
	for key, apiKey := range persistedKeys {
		akm.keys[key] = apiKey
	}
}

// saveKeysToFile saves current API keys to file (excluding config keys)
func (akm *APIKeyManager) saveKeysToFile() error {
	// Only save non-config keys
	persistedKeys := make(map[string]*types.APIKey)
	for key, apiKey := range akm.keys {
		if apiKey.Source != "config" {
			persistedKeys[key] = apiKey
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(akm.keysFile), 0755); err != nil {
		return err
	}

	// Serialize keys
	data, err := json.MarshalIndent(persistedKeys, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(akm.keysFile, data, 0644)
}

// IsValidKey checks if the provided API key is valid and active
func (akm *APIKeyManager) IsValidKey(key string) bool {
	akm.mutex.RLock()
	defer akm.mutex.RUnlock()

	if apiKey, exists := akm.keys[key]; exists {
		return apiKey.Active
	}
	return false
}

// CreateAPIKey creates a new API key
func (akm *APIKeyManager) CreateAPIKey(name, description, createdBy string) (*types.APIKey, error) {
	akm.mutex.Lock()
	defer akm.mutex.Unlock()

	// Generate secure random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, errors.SystemError(
			errors.ErrCodeSystemResource,
			"Failed to generate API key",
			err,
		).WithOperation("generate_api_key")
	}

	key := "secauto-" + hex.EncodeToString(keyBytes)

	// Create API key object
	apiKey := &types.APIKey{
		Key:         key,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		CreatedBy:   createdBy,
		Active:      true,
		Source:      "api",
		LastUsed:    "",
	}

	// Store in memory
	akm.keys[key] = apiKey

	// Persist to file
	if err := akm.saveKeysToFile(); err != nil {
		// Remove from memory if save fails
		delete(akm.keys, key)
		return nil, errors.SystemError(
			errors.ErrCodeSystemResource,
			"Failed to persist API key",
			err,
		).WithOperation("save_api_key").
			WithContext("key_name", name)
	}

	return apiKey, nil
}

// ListAPIKeys returns all API keys (excluding the actual key values for security)
func (akm *APIKeyManager) ListAPIKeys() []*types.APIKeySummary {
	akm.mutex.RLock()
	defer akm.mutex.RUnlock()

	var keys []*types.APIKeySummary
	for _, apiKey := range akm.keys {
		keys = append(keys, &types.APIKeySummary{
			KeyPrefix:   apiKey.Key[:12] + "...",
			Name:        apiKey.Name,
			Description: apiKey.Description,
			CreatedAt:   apiKey.CreatedAt,
			CreatedBy:   apiKey.CreatedBy,
			Active:      apiKey.Active,
			Source:      apiKey.Source,
			LastUsed:    apiKey.LastUsed,
		})
	}

	return keys
}

// UpdateLastUsed updates the last used timestamp for an API key
func (akm *APIKeyManager) UpdateLastUsed(key string) {
	akm.mutex.Lock()
	defer akm.mutex.Unlock()

	if apiKey, exists := akm.keys[key]; exists {
		apiKey.LastUsed = time.Now().UTC().Format(time.RFC3339)
		// Note: We don't save to file immediately for performance
		// This could be done periodically or on shutdown
	}
}

// DeactivateAPIKey deactivates an API key
func (akm *APIKeyManager) DeactivateAPIKey(keyPrefix string) error {
	akm.mutex.Lock()
	defer akm.mutex.Unlock()

	for key, apiKey := range akm.keys {
		if key[:12] == keyPrefix[:12] {
			// Don't allow deactivating config keys
			if apiKey.Source == "config" {
				return errors.AuthError(
					errors.ErrCodeAuthPermission,
					"Cannot deactivate configuration API key",
				).WithContext("key_prefix", keyPrefix).
					WithContext("key_source", "config")
			}

			apiKey.Active = false
			return akm.saveKeysToFile()
		}
	}

	return errors.AuthError(
		errors.ErrCodeAuthInvalid,
		"API key not found for deactivation",
	).WithContext("key_prefix", keyPrefix)
}

// DeleteAPIKey permanently deletes an API key
func (akm *APIKeyManager) DeleteAPIKey(keyPrefix string) error {
	akm.mutex.Lock()
	defer akm.mutex.Unlock()

	for key, apiKey := range akm.keys {
		if key[:12] == keyPrefix[:12] {
			// Don't allow deleting config keys
			if apiKey.Source == "config" {
				return errors.AuthError(
					errors.ErrCodeAuthPermission,
					"Cannot delete configuration API key",
				).WithContext("key_prefix", keyPrefix).
					WithContext("key_source", "config")
			}

			delete(akm.keys, key)
			return akm.saveKeysToFile()
		}
	}

	return errors.AuthError(
		errors.ErrCodeAuthInvalid,
		"API key not found for deletion",
	).WithContext("key_prefix", keyPrefix)

	return fmt.Errorf("API key not found")
}

// Shutdown saves all keys to file before server shutdown
func (akm *APIKeyManager) Shutdown() error {
	return akm.saveKeysToFile()
}

// GetStats returns API key statistics
func (akm *APIKeyManager) GetStats() *types.APIKeyStats {
	akm.mutex.RLock()
	defer akm.mutex.RUnlock()

	stats := &types.APIKeyStats{
		Total:        len(akm.keys),
		Active:       0,
		Inactive:     0,
		ConfigKeys:   0,
		GeneratedKeys: 0,
	}

	for _, apiKey := range akm.keys {
		if apiKey.Active {
			stats.Active++
		} else {
			stats.Inactive++
		}

		if apiKey.Source == "config" {
			stats.ConfigKeys++
		} else {
			stats.GeneratedKeys++
		}
	}

	return stats
}