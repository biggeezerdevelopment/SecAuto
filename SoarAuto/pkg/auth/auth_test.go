package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"SoarAuto/pkg/testutil"
)

func TestNewAPIKeyManager(t *testing.T) {
	tempDir := testutil.TempDir(t)
	apiKeysFile := filepath.Join(tempDir, "api_keys.json")
	
	staticKeys := []string{"static-key-1", "static-key-2"}
	
	manager := NewAPIKeyManager(apiKeysFile, staticKeys)
	
	if manager == nil {
		t.Fatal("Expected APIKeyManager but got nil")
	}
	
	// Test that static keys are loaded
	if !manager.IsValidKey("static-key-1") {
		t.Error("Expected static-key-1 to be valid")
	}
	if !manager.IsValidKey("static-key-2") {
		t.Error("Expected static-key-2 to be valid")
	}
}

func TestAPIKeyManager_CreateKey(t *testing.T) {
	tempDir := testutil.TempDir(t)
	apiKeysFile := filepath.Join(tempDir, "api_keys.json")
	
	manager := NewAPIKeyManager(apiKeysFile, []string{})
	
	// Create a new API key
	keyInfo, err := manager.CreateKey("test-key", "Test API Key")
	testutil.AssertNoError(t, err)
	
	if keyInfo.Name != "test-key" {
		t.Errorf("Expected name 'test-key', got %s", keyInfo.Name)
	}
	if keyInfo.Description != "Test API Key" {
		t.Errorf("Expected description 'Test API Key', got %s", keyInfo.Description)
	}
	if keyInfo.Key == "" {
		t.Error("Expected generated key to be non-empty")
	}
	if keyInfo.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	
	// Test that the key is now valid
	if !manager.IsValidKey(keyInfo.Key) {
		t.Error("Expected generated key to be valid")
	}
}

func TestAPIKeyManager_CreateKeyDuplicate(t *testing.T) {
	tempDir := testutil.TempDir(t)
	apiKeysFile := filepath.Join(tempDir, "api_keys.json")
	
	manager := NewAPIKeyManager(apiKeysFile, []string{})
	
	// Create first key
	_, err := manager.CreateKey("test-key", "Test API Key")
	testutil.AssertNoError(t, err)
	
	// Try to create duplicate
	_, err = manager.CreateKey("test-key", "Duplicate Key")
	testutil.AssertError(t, err, "already exists")
}

func TestAPIKeyManager_IsValidKey(t *testing.T) {
	tempDir := testutil.TempDir(t)
	apiKeysFile := filepath.Join(tempDir, "api_keys.json")
	
	staticKeys := []string{"static-key"}
	manager := NewAPIKeyManager(apiKeysFile, staticKeys)
	
	// Test static key
	if !manager.IsValidKey("static-key") {
		t.Error("Expected static key to be valid")
	}
	
	// Test invalid key
	if manager.IsValidKey("invalid-key") {
		t.Error("Expected invalid key to be invalid")
	}
	
	// Create and test dynamic key
	keyInfo, err := manager.CreateKey("dynamic", "Dynamic Key")
	testutil.AssertNoError(t, err)
	
	if !manager.IsValidKey(keyInfo.Key) {
		t.Error("Expected dynamic key to be valid")
	}
}

func TestAPIKeyManager_UpdateLastUsed(t *testing.T) {
	tempDir := testutil.TempDir(t)
	apiKeysFile := filepath.Join(tempDir, "api_keys.json")
	
	manager := NewAPIKeyManager(apiKeysFile, []string{})
	
	// Create a key
	keyInfo, err := manager.CreateKey("test-key", "Test Key")
	testutil.AssertNoError(t, err)
	
	originalLastUsed := keyInfo.LastUsed
	
	// Wait a bit to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)
	
	// Update last used
	manager.UpdateLastUsed(keyInfo.Key)
	
	// Get updated info
	keys := manager.ListKeys()
	var updatedKey *APIKeyInfo
	for _, key := range keys {
		if key.Key == keyInfo.Key {
			updatedKey = key
			break
		}
	}
	
	if updatedKey == nil {
		t.Fatal("Key not found after update")
	}
	
	if !updatedKey.LastUsed.After(originalLastUsed) {
		t.Error("Expected LastUsed to be updated")
	}
}

func TestAPIKeyManager_ListKeys(t *testing.T) {
	tempDir := testutil.TempDir(t)
	apiKeysFile := filepath.Join(tempDir, "api_keys.json")
	
	staticKeys := []string{"static-key"}
	manager := NewAPIKeyManager(apiKeysFile, staticKeys)
	
	// Initially should have static key
	keys := manager.ListKeys()
	if len(keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(keys))
	}
	
	// Create dynamic keys
	_, err := manager.CreateKey("key1", "Key 1")
	testutil.AssertNoError(t, err)
	
	_, err = manager.CreateKey("key2", "Key 2")
	testutil.AssertNoError(t, err)
	
	// Should now have 3 keys
	keys = manager.ListKeys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}
	
	// Verify key names
	names := make(map[string]bool)
	for _, key := range keys {
		names[key.Name] = true
	}
	
	expectedNames := []string{"static-key", "key1", "key2"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("Expected key name %s not found", name)
		}
	}
}

func TestAPIKeyManager_DeleteKey(t *testing.T) {
	tempDir := testutil.TempDir(t)
	apiKeysFile := filepath.Join(tempDir, "api_keys.json")
	
	manager := NewAPIKeyManager(apiKeysFile, []string{})
	
	// Create a key
	keyInfo, err := manager.CreateKey("test-key", "Test Key")
	testutil.AssertNoError(t, err)
	
	// Verify key exists
	if !manager.IsValidKey(keyInfo.Key) {
		t.Error("Expected key to be valid before deletion")
	}
	
	// Delete key
	err = manager.DeleteKey("test-key")
	testutil.AssertNoError(t, err)
	
	// Verify key is no longer valid
	if manager.IsValidKey(keyInfo.Key) {
		t.Error("Expected key to be invalid after deletion")
	}
	
	// Try to delete non-existent key
	err = manager.DeleteKey("non-existent")
	testutil.AssertError(t, err, "not found")
}

func TestAPIKeyManager_DeleteStaticKey(t *testing.T) {
	tempDir := testutil.TempDir(t)
	apiKeysFile := filepath.Join(tempDir, "api_keys.json")
	
	staticKeys := []string{"static-key"}
	manager := NewAPIKeyManager(apiKeysFile, staticKeys)
	
	// Try to delete static key (should fail)
	err := manager.DeleteKey("static-key")
	testutil.AssertError(t, err, "cannot delete static")
}

func TestAPIKeyManager_Persistence(t *testing.T) {
	tempDir := testutil.TempDir(t)
	apiKeysFile := filepath.Join(tempDir, "api_keys.json")
	
	// Create first manager and add keys
	manager1 := NewAPIKeyManager(apiKeysFile, []string{"static-key"})
	
	keyInfo1, err := manager1.CreateKey("key1", "Key 1")
	testutil.AssertNoError(t, err)
	
	keyInfo2, err := manager1.CreateKey("key2", "Key 2")
	testutil.AssertNoError(t, err)
	
	// Save keys
	err = manager1.Shutdown()
	testutil.AssertNoError(t, err)
	
	// Create second manager (should load saved keys)
	manager2 := NewAPIKeyManager(apiKeysFile, []string{"static-key"})
	
	// Verify keys were loaded
	if !manager2.IsValidKey(keyInfo1.Key) {
		t.Error("Expected key1 to be loaded")
	}
	if !manager2.IsValidKey(keyInfo2.Key) {
		t.Error("Expected key2 to be loaded")
	}
	
	keys := manager2.ListKeys()
	if len(keys) != 3 { // 2 dynamic + 1 static
		t.Errorf("Expected 3 keys after reload, got %d", len(keys))
	}
}

func TestAPIKeyManager_GetKeyStats(t *testing.T) {
	tempDir := testutil.TempDir(t)
	apiKeysFile := filepath.Join(tempDir, "api_keys.json")
	
	staticKeys := []string{"static-key"}
	manager := NewAPIKeyManager(apiKeysFile, staticKeys)
	
	// Create some keys
	keyInfo1, err := manager.CreateKey("key1", "Key 1")
	testutil.AssertNoError(t, err)
	
	_, err = manager.CreateKey("key2", "Key 2")
	testutil.AssertNoError(t, err)
	
	// Use one key
	manager.UpdateLastUsed(keyInfo1.Key)
	
	stats := manager.GetKeyStats()
	
	if stats.TotalKeys != 3 {
		t.Errorf("Expected 3 total keys, got %d", stats.TotalKeys)
	}
	if stats.StaticKeys != 1 {
		t.Errorf("Expected 1 static key, got %d", stats.StaticKeys)
	}
	if stats.DynamicKeys != 2 {
		t.Errorf("Expected 2 dynamic keys, got %d", stats.DynamicKeys)
	}
	if stats.UsedKeys != 1 {
		t.Errorf("Expected 1 used key, got %d", stats.UsedKeys)
	}
}

func TestAPIKeyManager_LoadInvalidFile(t *testing.T) {
	tempDir := testutil.TempDir(t)
	apiKeysFile := filepath.Join(tempDir, "api_keys.json")
	
	// Create invalid JSON file
	invalidJSON := `{"invalid": json}`
	err := os.WriteFile(apiKeysFile, []byte(invalidJSON), 0644)
	testutil.AssertNoError(t, err)
	
	// Should handle invalid file gracefully
	manager := NewAPIKeyManager(apiKeysFile, []string{"static-key"})
	
	// Should still work with static keys
	if !manager.IsValidKey("static-key") {
		t.Error("Expected static key to work even with invalid file")
	}
}

func TestAPIKeyManager_SaveLoadCycle(t *testing.T) {
	tempDir := testutil.TempDir(t)
	apiKeysFile := filepath.Join(tempDir, "api_keys.json")
	
	manager := NewAPIKeyManager(apiKeysFile, []string{})
	
	// Create keys with different properties
	key1, err := manager.CreateKey("key1", "Description 1")
	testutil.AssertNoError(t, err)
	
	key2, err := manager.CreateKey("key2", "Description 2")
	testutil.AssertNoError(t, err)
	
	// Update usage for one key
	manager.UpdateLastUsed(key1.Key)
	time.Sleep(10 * time.Millisecond)
	manager.UpdateLastUsed(key2.Key)
	
	// Save
	err = manager.Shutdown()
	testutil.AssertNoError(t, err)
	
	// Load and verify
	newManager := NewAPIKeyManager(apiKeysFile, []string{})
	
	keys := newManager.ListKeys()
	if len(keys) != 2 {
		t.Fatalf("Expected 2 keys, got %d", len(keys))
	}
	
	// Verify all properties were preserved
	keyMap := make(map[string]*APIKeyInfo)
	for _, key := range keys {
		keyMap[key.Name] = key
	}
	
	if keyMap["key1"].Description != "Description 1" {
		t.Error("Key1 description not preserved")
	}
	if keyMap["key2"].Description != "Description 2" {
		t.Error("Key2 description not preserved")
	}
	
	// Verify last used times were preserved
	if keyMap["key1"].LastUsed.IsZero() {
		t.Error("Key1 last used time not preserved")
	}
	if keyMap["key2"].LastUsed.IsZero() {
		t.Error("Key2 last used time not preserved")
	}
}