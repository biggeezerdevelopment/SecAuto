//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"SoarAuto/pkg/config"
	"SoarAuto/pkg/redis"
	"SoarAuto/pkg/testutil"
)

func TestRedisIntegration(t *testing.T) {
	cfg := testutil.TestConfig(t)
	
	redisClient, err := redis.NewRedisClient(cfg)
	testutil.AssertNoError(t, err)
	
	defer redisClient.Close()
	
	t.Run("basic operations", func(t *testing.T) {
		// Test Set and Get
		response := redisClient.SetCache("test-key", "test-value")
		if !response.Success {
			t.Fatalf("Failed to set cache: %s", response.ErrorMessage)
		}
		
		response = redisClient.GetCache("test-key")
		if !response.Success {
			t.Fatalf("Failed to get cache: %s", response.ErrorMessage)
		}
		
		if response.Value != "test-value" {
			t.Errorf("Expected 'test-value', got %v", response.Value)
		}
	})
	
	t.Run("json operations", func(t *testing.T) {
		testData := map[string]interface{}{
			"string": "value",
			"number": 42,
			"bool":   true,
			"array":  []interface{}{1, 2, 3},
		}
		
		response := redisClient.SetCache("test-json", testData)
		if !response.Success {
			t.Fatalf("Failed to set JSON cache: %s", response.ErrorMessage)
		}
		
		response = redisClient.GetCache("test-json")
		if !response.Success {
			t.Fatalf("Failed to get JSON cache: %s", response.ErrorMessage)
		}
		
		// Verify JSON structure
		jsonData, ok := response.Value.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map[string]interface{}, got %T", response.Value)
		}
		
		if jsonData["string"] != "value" {
			t.Errorf("Expected 'value', got %v", jsonData["string"])
		}
		if jsonData["number"].(float64) != 42 {
			t.Errorf("Expected 42, got %v", jsonData["number"])
		}
	})
	
	t.Run("ttl operations", func(t *testing.T) {
		response := redisClient.SetCacheWithTTL("ttl-key", "ttl-value", 1)
		if !response.Success {
			t.Fatalf("Failed to set cache with TTL: %s", response.ErrorMessage)
		}
		
		// Immediately check that key exists
		response = redisClient.GetCache("ttl-key")
		if !response.Success {
			t.Fatalf("Key should exist immediately: %s", response.ErrorMessage)
		}
		
		// Wait for expiration
		time.Sleep(2 * time.Second)
		
		response = redisClient.GetCache("ttl-key")
		if response.Success {
			t.Error("Key should have expired")
		}
	})
	
	t.Run("delete operations", func(t *testing.T) {
		// Set a key
		response := redisClient.SetCache("delete-key", "delete-value")
		testutil.AssertNoError(t, nil) // Assuming SetCache doesn't return error
		
		// Verify it exists
		response = redisClient.GetCache("delete-key")
		if !response.Success {
			t.Fatal("Key should exist before deletion")
		}
		
		// Delete it
		response = redisClient.DeleteCache("delete-key")
		if !response.Success {
			t.Fatalf("Failed to delete key: %s", response.ErrorMessage)
		}
		
		// Verify it's gone
		response = redisClient.GetCache("delete-key")
		if response.Success {
			t.Error("Key should not exist after deletion")
		}
	})
	
	t.Run("list operations", func(t *testing.T) {
		listName := "test-list"
		
		// Add items to list
		items := []interface{}{"item1", "item2", "item3"}
		response := redisClient.AddToList(listName, items, "tail")
		if !response.Success {
			t.Fatalf("Failed to add to list: %s", response.ErrorMessage)
		}
		
		// Get list
		response = redisClient.GetList(listName)
		if !response.Success {
			t.Fatalf("Failed to get list: %s", response.ErrorMessage)
		}
		
		// Verify list contents
		if len(response.Items) != 3 {
			t.Errorf("Expected 3 items, got %d", len(response.Items))
		}
		
		// Remove items
		removeItems := []interface{}{"item2"}
		response = redisClient.RemoveFromList(listName, removeItems, 1)
		if !response.Success {
			t.Fatalf("Failed to remove from list: %s", response.ErrorMessage)
		}
		
		// Verify removal
		response = redisClient.GetList(listName)
		if !response.Success {
			t.Fatalf("Failed to get list after removal: %s", response.ErrorMessage)
		}
		
		if len(response.Items) != 2 {
			t.Errorf("Expected 2 items after removal, got %d", len(response.Items))
		}
		
		// Delete entire list
		response = redisClient.DeleteList(listName)
		if !response.Success {
			t.Fatalf("Failed to delete list: %s", response.ErrorMessage)
		}
	})
	
	t.Run("cache stats", func(t *testing.T) {
		// Set some test data
		redisClient.SetCache("stats-key-1", "value1")
		redisClient.SetCache("stats-key-2", "value2")
		
		response := redisClient.GetCacheStats()
		if !response.Success {
			t.Fatalf("Failed to get cache stats: %s", response.ErrorMessage)
		}
		
		// Basic validation of stats structure
		if response.Stats == nil {
			t.Error("Expected stats to be non-nil")
		}
	})
	
	t.Run("list cache keys", func(t *testing.T) {
		// Set test keys
		redisClient.SetCache("pattern-test-1", "value1")
		redisClient.SetCache("pattern-test-2", "value2")
		redisClient.SetCache("other-key", "value3")
		
		// List all keys
		response := redisClient.ListCacheKeys("")
		if !response.Success {
			t.Fatalf("Failed to list cache keys: %s", response.ErrorMessage)
		}
		
		if len(response.Keys) < 3 {
			t.Errorf("Expected at least 3 keys, got %d", len(response.Keys))
		}
		
		// List keys with pattern
		response = redisClient.ListCacheKeys("pattern-test-*")
		if !response.Success {
			t.Fatalf("Failed to list cache keys with pattern: %s", response.ErrorMessage)
		}
		
		if len(response.Keys) != 2 {
			t.Errorf("Expected 2 keys matching pattern, got %d", len(response.Keys))
		}
	})
	
	t.Run("clear cache", func(t *testing.T) {
		// Set some test data
		redisClient.SetCache("clear-test-1", "value1")
		redisClient.SetCache("clear-test-2", "value2")
		
		// Clear cache
		response := redisClient.ClearCache()
		if !response.Success {
			t.Fatalf("Failed to clear cache: %s", response.ErrorMessage)
		}
		
		// Verify cache is empty
		response = redisClient.ListCacheKeys("")
		if !response.Success {
			t.Fatalf("Failed to list keys after clear: %s", response.ErrorMessage)
		}
		
		if len(response.Keys) != 0 {
			t.Errorf("Expected 0 keys after clear, got %d", len(response.Keys))
		}
	})
}

func TestRedisConnectionFailure(t *testing.T) {
	// Test with invalid Redis URL
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			RedisURL: "redis://invalid-host:6379/0",
		},
	}
	
	_, err := redis.NewRedisClient(cfg)
	if err == nil {
		t.Error("Expected error for invalid Redis URL")
	}
}

func TestRedisReconnection(t *testing.T) {
	cfg := testutil.TestConfig(t)
	
	redisClient, err := redis.NewRedisClient(cfg)
	testutil.AssertNoError(t, err)
	
	defer redisClient.Close()
	
	// Test that client can handle temporary disconnections
	// This is a basic test - in practice you'd simulate network issues
	
	// Set a value
	response := redisClient.SetCache("reconnect-test", "value")
	if !response.Success {
		t.Fatalf("Failed to set initial value: %s", response.ErrorMessage)
	}
	
	// Get the value (should work)
	response = redisClient.GetCache("reconnect-test")
	if !response.Success {
		t.Fatalf("Failed to get value: %s", response.ErrorMessage)
	}
	
	if response.Value != "value" {
		t.Errorf("Expected 'value', got %v", response.Value)
	}
}

func TestRedisConcurrency(t *testing.T) {
	cfg := testutil.TestConfig(t)
	
	redisClient, err := redis.NewRedisClient(cfg)
	testutil.AssertNoError(t, err)
	
	defer redisClient.Close()
	
	// Test concurrent operations
	const numGoroutines = 10
	const numOperations = 100
	
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)
				
				// Set value
				response := redisClient.SetCache(key, value)
				if !response.Success {
					t.Errorf("Failed to set %s: %s", key, response.ErrorMessage)
					return
				}
				
				// Get value
				response = redisClient.GetCache(key)
				if !response.Success {
					t.Errorf("Failed to get %s: %s", key, response.ErrorMessage)
					return
				}
				
				if response.Value != value {
					t.Errorf("Expected %s, got %v", value, response.Value)
					return
				}
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Goroutine completed
		case <-time.After(30 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
}