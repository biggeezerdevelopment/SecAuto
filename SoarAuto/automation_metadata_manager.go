package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/redis/go-redis/v9"
)

// AutomationMetadataManager handles automation metadata operations
type AutomationMetadataManager struct {
	metadataFile string
	redisClient  *redis.Client
	metadata     []AutomationMetadata
	mutex        sync.RWMutex
}

// NewAutomationMetadataManager creates a new metadata manager
func NewAutomationMetadataManager(metadataFile string, redisClient *redis.Client) *AutomationMetadataManager {
	return &AutomationMetadataManager{
		metadataFile: metadataFile,
		redisClient:  redisClient,
		metadata:     make([]AutomationMetadata, 0),
	}
}

// LoadMetadataFromDisk loads automation metadata from the specified file
func (amm *AutomationMetadataManager) LoadMetadataFromDisk() error {
	amm.mutex.Lock()
	defer amm.mutex.Unlock()

	// Check if metadata file exists
	if _, err := os.Stat(amm.metadataFile); os.IsNotExist(err) {
		logger.Info("No automation metadata file found, starting with empty metadata", map[string]interface{}{
			"component": "automation_metadata_manager",
			"file":      amm.metadataFile,
		})
		return nil
	}

	// Read metadata file
	data, err := ioutil.ReadFile(amm.metadataFile)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %v", err)
	}

	// Parse JSON metadata
	if err := json.Unmarshal(data, &amm.metadata); err != nil {
		return fmt.Errorf("failed to parse metadata JSON: %v", err)
	}

	logger.Info("Loaded automation metadata from disk", map[string]interface{}{
		"component": "automation_metadata_manager",
		"file":      amm.metadataFile,
		"count":     len(amm.metadata),
	})

	return nil
}

// SaveMetadataToDisk saves automation metadata to the specified file
func (amm *AutomationMetadataManager) SaveMetadataToDisk() error {
	amm.mutex.RLock()
	defer amm.mutex.RUnlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(amm.metadataFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %v", err)
	}

	// Marshal metadata to JSON
	data, err := json.MarshalIndent(amm.metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata to JSON: %v", err)
	}

	// Write to file
	if err := ioutil.WriteFile(amm.metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %v", err)
	}

	logger.Info("Saved automation metadata to disk", map[string]interface{}{
		"component": "automation_metadata_manager",
		"file":      amm.metadataFile,
		"count":     len(amm.metadata),
	})

	return nil
}

// LoadMetadataToRedis loads all automation metadata into Redis
func (amm *AutomationMetadataManager) LoadMetadataToRedis() error {
	amm.mutex.RLock()
	defer amm.mutex.RUnlock()

	ctx := context.Background()

	// Clear existing metadata in Redis
	if err := amm.redisClient.Del(ctx, "automation_metadata").Err(); err != nil {
		return fmt.Errorf("failed to clear existing metadata in Redis: %v", err)
	}

	// Store metadata in Redis
	for _, meta := range amm.metadata {
		key := fmt.Sprintf("automation_metadata:%s", meta.Name)
		data, err := json.Marshal(meta)
		if err != nil {
			logger.Error("Failed to marshal automation metadata", map[string]interface{}{
				"component": "automation_metadata_manager",
				"name":      meta.Name,
				"error":     err.Error(),
			})
			continue
		}

		if err := amm.redisClient.Set(ctx, key, data, 0).Err(); err != nil {
			logger.Error("Failed to store automation metadata in Redis", map[string]interface{}{
				"component": "automation_metadata_manager",
				"name":      meta.Name,
				"error":     err.Error(),
			})
			continue
		}
	}

	// Store the complete list as well
	if len(amm.metadata) > 0 {
		listData, err := json.Marshal(amm.metadata)
		if err == nil {
			amm.redisClient.Set(ctx, "automation_metadata", listData, 0)
		}
	}

	logger.Info("Loaded automation metadata to Redis", map[string]interface{}{
		"component": "automation_metadata_manager",
		"count":     len(amm.metadata),
	})

	return nil
}

// AddMetadata adds or updates automation metadata
func (amm *AutomationMetadataManager) AddMetadata(metadata AutomationMetadata) error {
	amm.mutex.Lock()
	defer amm.mutex.Unlock()

	// Check if metadata already exists for this automation
	for i, existing := range amm.metadata {
		if existing.Name == metadata.Name {
			// Update existing metadata
			amm.metadata[i] = metadata
			logger.Info("Updated automation metadata", map[string]interface{}{
				"component": "automation_metadata_manager",
				"name":      metadata.Name,
			})
			return nil
		}
	}

	// Add new metadata
	amm.metadata = append(amm.metadata, metadata)
	logger.Info("Added new automation metadata", map[string]interface{}{
		"component": "automation_metadata_manager",
		"name":      metadata.Name,
	})

	return nil
}

// GetMetadata retrieves metadata for a specific automation
func (amm *AutomationMetadataManager) GetMetadata(name string) (*AutomationMetadata, error) {
	amm.mutex.RLock()
	defer amm.mutex.RUnlock()

	for _, meta := range amm.metadata {
		if meta.Name == name {
			return &meta, nil
		}
	}

	return nil, fmt.Errorf("metadata not found for automation: %s", name)
}

// GetAllMetadata retrieves all automation metadata
func (amm *AutomationMetadataManager) GetAllMetadata() []AutomationMetadata {
	amm.mutex.RLock()
	defer amm.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]AutomationMetadata, len(amm.metadata))
	copy(result, amm.metadata)
	return result
}

// RemoveMetadata removes metadata for a specific automation
func (amm *AutomationMetadataManager) RemoveMetadata(name string) error {
	amm.mutex.Lock()
	defer amm.mutex.Unlock()

	for i, meta := range amm.metadata {
		if meta.Name == name {
			// Remove from slice
			amm.metadata = append(amm.metadata[:i], amm.metadata[i+1:]...)

			// Also remove from Redis
			ctx := context.Background()
			key := fmt.Sprintf("automation_metadata:%s", name)
			amm.redisClient.Del(ctx, key)

			logger.Info("Removed automation metadata", map[string]interface{}{
				"component": "automation_metadata_manager",
				"name":      name,
			})
			return nil
		}
	}

	return fmt.Errorf("metadata not found for automation: %s", name)
}

// GetMetadataFromRedis retrieves metadata for a specific automation from Redis
func (amm *AutomationMetadataManager) GetMetadataFromRedis(name string) (*AutomationMetadata, error) {
	ctx := context.Background()
	key := fmt.Sprintf("automation_metadata:%s", name)

	data, err := amm.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("metadata not found in Redis for automation: %s", name)
		}
		return nil, fmt.Errorf("failed to get metadata from Redis: %v", err)
	}

	var metadata AutomationMetadata
	if err := json.Unmarshal([]byte(data), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata from Redis: %v", err)
	}

	return &metadata, nil
}

// GetAllMetadataFromRedis retrieves all automation metadata from Redis
func (amm *AutomationMetadataManager) GetAllMetadataFromRedis() ([]AutomationMetadata, error) {
	ctx := context.Background()
	data, err := amm.redisClient.Get(ctx, "automation_metadata").Result()
	if err != nil {
		if err == redis.Nil {
			return []AutomationMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to get metadata from Redis: %v", err)
	}

	var metadata []AutomationMetadata
	if err := json.Unmarshal([]byte(data), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata from Redis: %v", err)
	}

	return metadata, nil
}

// Close performs cleanup operations
func (amm *AutomationMetadataManager) Close() error {
	// Save metadata to disk before closing
	return amm.SaveMetadataToDisk()
}
