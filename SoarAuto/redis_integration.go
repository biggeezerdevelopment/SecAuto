package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisIntegration creates a new Redis integration instance
func NewRedisIntegration(config *Config) (*RedisIntegration, error) {
	// Parse Redis URL from config
	redisURL := config.Database.RedisURL
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	// Parse Redis URL
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}

	// Create Redis client
	client := redis.NewClient(opts)

	// Test connection
	ctx := context.Background()
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &RedisIntegration{
		client: client,
		config: config,
	}, nil
}

// GetCache retrieves a value from Redis cache
func (r *RedisIntegration) GetCache(key string) CacheResponse {
	ctx := context.Background()

	logger.Info("Getting cache value", map[string]interface{}{
		"component": "redis_integration",
		"key":       key,
	})

	value, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return CacheResponse{
				Success:      false,
				Key:          key,
				ErrorMessage: "Key not found in cache",
				Timestamp:    time.Now().UTC().Format(time.RFC3339),
			}
		}
		return CacheResponse{
			Success:      false,
			Key:          key,
			ErrorMessage: fmt.Sprintf("Failed to get cache value: %v", err),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
	}

	// Try to parse as JSON, if it fails, return as string
	var parsedValue interface{}
	if err := json.Unmarshal([]byte(value), &parsedValue); err != nil {
		// Not JSON, return as string
		parsedValue = value
	}

	return CacheResponse{
		Success:   true,
		Key:       key,
		Value:     parsedValue,
		Message:   "Value retrieved successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// SetCache stores a value in Redis cache
func (r *RedisIntegration) SetCache(key string, value interface{}) CacheResponse {
	ctx := context.Background()

	logger.Info("Setting cache value", map[string]interface{}{
		"component": "redis_integration",
		"key":       key,
	})

	// Convert value to string
	var valueStr string
	switch v := value.(type) {
	case string:
		valueStr = v
	default:
		// Convert to JSON
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return CacheResponse{
				Success:      false,
				Key:          key,
				ErrorMessage: fmt.Sprintf("Failed to serialize value: %v", err),
				Timestamp:    time.Now().UTC().Format(time.RFC3339),
			}
		}
		valueStr = string(jsonBytes)
	}

	// Set value in Redis
	err := r.client.Set(ctx, key, valueStr, 0).Err()
	if err != nil {
		return CacheResponse{
			Success:      false,
			Key:          key,
			ErrorMessage: fmt.Sprintf("Failed to set cache value: %v", err),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
	}

	return CacheResponse{
		Success:   true,
		Key:       key,
		Value:     value,
		Message:   "Value set successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// DeleteCache removes a value from Redis cache
func (r *RedisIntegration) DeleteCache(key string) CacheResponse {
	ctx := context.Background()

	logger.Info("Deleting cache value", map[string]interface{}{
		"component": "redis_integration",
		"key":       key,
	})

	deletedCount, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return CacheResponse{
			Success:      false,
			Key:          key,
			ErrorMessage: fmt.Sprintf("Failed to delete cache value: %v", err),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
	}

	if deletedCount == 0 {
		return CacheResponse{
			Success:      false,
			Key:          key,
			ErrorMessage: "Key not found in cache",
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
	}

	return CacheResponse{
		Success:   true,
		Key:       key,
		Message:   "Value deleted successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// AddToList adds items to a Redis list, checking for duplicates
func (r *RedisIntegration) AddToList(listName string, items []interface{}, position string) ListResponse {
	ctx := context.Background()

	logger.Info("Adding items to list", map[string]interface{}{
		"component": "redis_integration",
		"list_name": listName,
		"count":     len(items),
		"position":  position,
	})

	// Get existing items from the list to check for duplicates
	existingItems, err := r.client.LRange(ctx, listName, 0, -1).Result()
	if err != nil {
		return ListResponse{
			Success:      false,
			ListName:     listName,
			ErrorMessage: fmt.Sprintf("Failed to get existing list items: %v", err),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
	}

	// Create a map for faster duplicate checking
	existingSet := make(map[string]bool)
	for _, item := range existingItems {
		existingSet[item] = true
	}

	// Convert items to strings and filter out duplicates
	var uniqueItems []interface{}
	var duplicateItems []string

	for i, item := range items {
		var itemStr string
		switch v := item.(type) {
		case string:
			itemStr = v
		default:
			// Convert to JSON
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return ListResponse{
					Success:      false,
					ListName:     listName,
					ErrorMessage: fmt.Sprintf("Failed to serialize item %d: %v", i, err),
					Timestamp:    time.Now().UTC().Format(time.RFC3339),
				}
			}
			itemStr = string(jsonBytes)
		}

		// Check if item already exists
		if existingSet[itemStr] {
			duplicateItems = append(duplicateItems, itemStr)
			logger.Info("Skipping duplicate item", map[string]interface{}{
				"component": "redis_integration",
				"list_name": listName,
				"item":      itemStr,
			})
		} else {
			uniqueItems = append(uniqueItems, itemStr)
			// Add to existing set to prevent duplicates within this batch
			existingSet[itemStr] = true
		}
	}

	// If no unique items to add
	if len(uniqueItems) == 0 {
		message := "No items added - all items already exist in the list"
		if len(duplicateItems) > 0 {
			message = fmt.Sprintf("No items added - %d duplicate items skipped", len(duplicateItems))
		}
		return ListResponse{
			Success:   true,
			ListName:  listName,
			Count:     len(existingItems), // Return current list count
			Message:   message,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}

	// Add unique items to list based on position
	var addedCount int64

	if position == "left" {
		addedCount, err = r.client.LPush(ctx, listName, uniqueItems...).Result()
	} else {
		// Default to right
		addedCount, err = r.client.RPush(ctx, listName, uniqueItems...).Result()
	}

	if err != nil {
		return ListResponse{
			Success:      false,
			ListName:     listName,
			ErrorMessage: fmt.Sprintf("Failed to add items to list: %v", err),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
	}

	// Prepare success message
	message := fmt.Sprintf("Successfully added %d unique items to list", len(uniqueItems))
	if len(duplicateItems) > 0 {
		message += fmt.Sprintf(" (%d duplicates skipped)", len(duplicateItems))
	}

	return ListResponse{
		Success:   true,
		ListName:  listName,
		Count:     int(addedCount),
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// GetList retrieves all items from a Redis list
func (r *RedisIntegration) GetList(listName string) ListResponse {
	ctx := context.Background()

	logger.Info("Getting list items", map[string]interface{}{
		"component": "redis_integration",
		"list_name": listName,
	})

	// Get all items from the list (0 to -1 means all)
	items, err := r.client.LRange(ctx, listName, 0, -1).Result()
	if err != nil {
		return ListResponse{
			Success:      false,
			ListName:     listName,
			ErrorMessage: fmt.Sprintf("Failed to get list items: %v", err),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
	}

	// Convert items back to appropriate types
	var parsedItems []interface{}
	for _, item := range items {
		// Try to parse as JSON, if it fails, keep as string
		var parsedItem interface{}
		if err := json.Unmarshal([]byte(item), &parsedItem); err != nil {
			// Not JSON, keep as string
			parsedItem = item
		}
		parsedItems = append(parsedItems, parsedItem)
	}

	return ListResponse{
		Success:   true,
		ListName:  listName,
		Items:     parsedItems,
		Count:     len(parsedItems),
		Message:   fmt.Sprintf("Successfully retrieved %d items from list", len(parsedItems)),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// DeleteList deletes an entire Redis list
func (r *RedisIntegration) DeleteList(listName string) ListResponse {
	ctx := context.Background()

	logger.Info("Deleting list", map[string]interface{}{
		"component": "redis_integration",
		"list_name": listName,
	})

	deletedCount, err := r.client.Del(ctx, listName).Result()
	if err != nil {
		return ListResponse{
			Success:      false,
			ListName:     listName,
			ErrorMessage: fmt.Sprintf("Failed to delete list: %v", err),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
	}

	if deletedCount == 0 {
		return ListResponse{
			Success:      false,
			ListName:     listName,
			ErrorMessage: "List not found",
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
	}

	return ListResponse{
		Success:   true,
		ListName:  listName,
		Message:   "List deleted successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// RemoveFromList removes specific items from a Redis list
func (r *RedisIntegration) RemoveFromList(listName string, items []interface{}, count int) ListResponse {
	ctx := context.Background()

	logger.Info("Removing items from list", map[string]interface{}{
		"component": "redis_integration",
		"list_name": listName,
		"count":     count,
		"items":     len(items),
	})

	if count <= 0 {
		count = 1 // Default to removing 1 occurrence
	}

	totalRemoved := 0

	for _, item := range items {
		// Convert item to string for Redis comparison
		var itemStr string
		switch v := item.(type) {
		case string:
			itemStr = v
		default:
			// Convert to JSON
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return ListResponse{
					Success:      false,
					ListName:     listName,
					ErrorMessage: fmt.Sprintf("Failed to serialize item for removal: %v", err),
					Timestamp:    time.Now().UTC().Format(time.RFC3339),
				}
			}
			itemStr = string(jsonBytes)
		}

		// Remove the item from the list
		removedCount, err := r.client.LRem(ctx, listName, int64(count), itemStr).Result()
		if err != nil {
			return ListResponse{
				Success:      false,
				ListName:     listName,
				ErrorMessage: fmt.Sprintf("Failed to remove item from list: %v", err),
				Timestamp:    time.Now().UTC().Format(time.RFC3339),
			}
		}

		totalRemoved += int(removedCount)
	}

	return ListResponse{
		Success:   true,
		ListName:  listName,
		Count:     totalRemoved,
		Message:   fmt.Sprintf("Successfully removed %d items from list", totalRemoved),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// Close closes the Redis connection
func (r *RedisIntegration) Close() error {
	return r.client.Close()
}
