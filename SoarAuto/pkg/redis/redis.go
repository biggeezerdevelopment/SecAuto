package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	
	"SoarAuto/pkg/config"
	"SoarAuto/pkg/types"
)

// RedisClient represents a Redis client wrapper
type RedisClient struct {
	client *redis.Client
	config *config.Config
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	// Parse Redis URL
	opt, err := redis.ParseURL(cfg.Database.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}
	
	client := redis.NewClient(opt)
	
	// Test connection
	ctx := context.Background()
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}
	
	return &RedisClient{
		client: client,
		config: cfg,
	}, nil
}

// GetCache retrieves a value from Redis cache
func (r *RedisClient) GetCache(key string) types.CacheResponse {
	ctx := context.Background()
	
	value, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return types.CacheResponse{
				Success:      false,
				Key:          key,
				ErrorMessage: "Key not found in cache",
				Timestamp:    time.Now().UTC().Format(time.RFC3339),
			}
		}
		return types.CacheResponse{
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
	
	return types.CacheResponse{
		Success:   true,
		Key:       key,
		Value:     parsedValue,
		Message:   "Value retrieved successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// SetCache stores a value in Redis cache
func (r *RedisClient) SetCache(key string, value interface{}) types.CacheResponse {
	ctx := context.Background()
	
	// Convert value to string
	var valueStr string
	switch v := value.(type) {
	case string:
		valueStr = v
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return types.CacheResponse{
				Success:      false,
				Key:          key,
				ErrorMessage: fmt.Sprintf("Failed to serialize value: %v", err),
				Timestamp:    time.Now().UTC().Format(time.RFC3339),
			}
		}
		valueStr = string(jsonBytes)
	}
	
	err := r.client.Set(ctx, key, valueStr, 0).Err()
	if err != nil {
		return types.CacheResponse{
			Success:      false,
			Key:          key,
			ErrorMessage: fmt.Sprintf("Failed to set cache value: %v", err),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
	}
	
	return types.CacheResponse{
		Success:   true,
		Key:       key,
		Value:     value,
		Message:   "Value stored successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// DeleteCache removes a key from Redis cache
func (r *RedisClient) DeleteCache(key string) types.CacheResponse {
	ctx := context.Background()
	
	deleted, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return types.CacheResponse{
			Success:      false,
			Key:          key,
			ErrorMessage: fmt.Sprintf("Failed to delete cache value: %v", err),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
	}
	
	if deleted == 0 {
		return types.CacheResponse{
			Success:   false,
			Key:       key,
			Message:   "Key not found",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}
	
	return types.CacheResponse{
		Success:   true,
		Key:       key,
		Message:   "Key deleted successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// ListCacheKeys lists all cache keys (with optional pattern)
func (r *RedisClient) ListCacheKeys(pattern string) types.CacheListResponse {
	ctx := context.Background()
	
	if pattern == "" {
		pattern = "*"
	}
	
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return types.CacheListResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to list keys: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}
	
	return types.CacheListResponse{
		Success:   true,
		Keys:      keys,
		Message:   fmt.Sprintf("Found %d keys", len(keys)),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// GetCacheStats returns Redis cache statistics
func (r *RedisClient) GetCacheStats() types.CacheStatsResponse {
	ctx := context.Background()
	
	info, err := r.client.Info(ctx).Result()
	if err != nil {
		return types.CacheStatsResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to get Redis info: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}
	
	// Parse Redis info into stats
	stats := make(map[string]interface{})
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				stats[key] = value
			}
		}
	}
	
	return types.CacheStatsResponse{
		Success:   true,
		Stats:     stats,
		Message:   "Cache statistics retrieved successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// ClearCache clears all cache keys
func (r *RedisClient) ClearCache() types.CacheResponse {
	ctx := context.Background()
	
	err := r.client.FlushDB(ctx).Err()
	if err != nil {
		return types.CacheResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to clear cache: %v", err),
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}
	}
	
	return types.CacheResponse{
		Success:   true,
		Message:   "Cache cleared successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// GetList retrieves all items from a Redis list
func (r *RedisClient) GetList(listName string) types.ListResponse {
	ctx := context.Background()
	
	items, err := r.client.LRange(ctx, listName, 0, -1).Result()
	if err != nil {
		return types.ListResponse{
			Success:   false,
			ListName:  listName,
			Error:     fmt.Sprintf("Failed to get list: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}
	
	// Convert strings to interfaces and try to parse JSON
	var parsedItems []interface{}
	for _, item := range items {
		var parsed interface{}
		if err := json.Unmarshal([]byte(item), &parsed); err != nil {
			// Not JSON, keep as string
			parsed = item
		}
		parsedItems = append(parsedItems, parsed)
	}
	
	return types.ListResponse{
		Success:   true,
		ListName:  listName,
		Items:     parsedItems,
		Count:     len(parsedItems),
		Message:   fmt.Sprintf("Retrieved %d items", len(parsedItems)),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// AddToList adds items to a Redis list
func (r *RedisClient) AddToList(listName string, items []interface{}, position string) types.ListResponse {
	ctx := context.Background()
	
	if len(items) == 0 {
		return types.ListResponse{
			Success:   false,
			ListName:  listName,
			Error:     "No items provided",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}
	
	// Convert items to strings
	var stringItems []interface{}
	for _, item := range items {
		var itemStr string
		switch v := item.(type) {
		case string:
			itemStr = v
		default:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return types.ListResponse{
					Success:   false,
					ListName:  listName,
					Error:     fmt.Sprintf("Failed to serialize item: %v", err),
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				}
			}
			itemStr = string(jsonBytes)
		}
		stringItems = append(stringItems, itemStr)
	}
	
	// Add to list based on position
	var addedCount int64
	var err error
	
	if position == "left" {
		addedCount, err = r.client.LPush(ctx, listName, stringItems...).Result()
	} else {
		// Default to right
		addedCount, err = r.client.RPush(ctx, listName, stringItems...).Result()
	}
	
	if err != nil {
		return types.ListResponse{
			Success:   false,
			ListName:  listName,
			Error:     fmt.Sprintf("Failed to add items to list: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}
	
	return types.ListResponse{
		Success:   true,
		ListName:  listName,
		Count:     int(addedCount),
		Message:   fmt.Sprintf("Added %d items to %s of list", len(items), position),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// RemoveFromList removes items from a Redis list
func (r *RedisClient) RemoveFromList(listName string, items []interface{}, count int) types.ListResponse {
	ctx := context.Background()
	
	if len(items) == 0 {
		return types.ListResponse{
			Success:   false,
			ListName:  listName,
			Error:     "No items provided",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}
	
	if count <= 0 {
		count = 1 // Default to removing 1 occurrence
	}
	
	var totalRemoved int64
	for _, item := range items {
		var itemStr string
		switch v := item.(type) {
		case string:
			itemStr = v
		default:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return types.ListResponse{
					Success:   false,
					ListName:  listName,
					Error:     fmt.Sprintf("Failed to serialize item: %v", err),
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				}
			}
			itemStr = string(jsonBytes)
		}
		
		removed, err := r.client.LRem(ctx, listName, int64(count), itemStr).Result()
		if err != nil {
			return types.ListResponse{
				Success:   false,
				ListName:  listName,
				Error:     fmt.Sprintf("Failed to remove item: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
		}
		totalRemoved += removed
	}
	
	return types.ListResponse{
		Success:   true,
		ListName:  listName,
		Count:     int(totalRemoved),
		Message:   fmt.Sprintf("Removed %d items from list", totalRemoved),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// DeleteList deletes an entire Redis list
func (r *RedisClient) DeleteList(listName string) types.ListResponse {
	ctx := context.Background()
	
	deleted, err := r.client.Del(ctx, listName).Result()
	if err != nil {
		return types.ListResponse{
			Success:   false,
			ListName:  listName,
			Error:     fmt.Sprintf("Failed to delete list: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}
	
	if deleted == 0 {
		return types.ListResponse{
			Success:   false,
			ListName:  listName,
			Error:     "List not found",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}
	
	return types.ListResponse{
		Success:   true,
		ListName:  listName,
		Message:   "List deleted successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}