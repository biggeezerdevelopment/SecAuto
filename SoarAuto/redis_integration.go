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

// Close closes the Redis connection
func (r *RedisIntegration) Close() error {
	return r.client.Close()
}
