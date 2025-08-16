package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"SoarAuto/pkg/config"
	"SoarAuto/pkg/logger"
	"SoarAuto/pkg/types"
)

// TestConfig loads a test configuration file
func TestConfig(t *testing.T) *config.Config {
	t.Helper()
	
	configPath := filepath.Join("../../testdata/configs/test-config.yaml")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	
	return cfg
}

// TestLogger creates a test logger that writes to testing.T
func TestLogger(t *testing.T) types.Logger {
	t.Helper()
	
	return logger.NewStructuredLoggerWithConfig(
		logger.LogLevel("DEBUG"),
		"stdout",
		"",
		nil,
		map[string]string{"default": "DEBUG"},
		nil,
	)
}

// TestRedisClient creates a Redis client for testing
func TestRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use test database
	})
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}
	
	// Clean test database
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("Failed to clean test database: %v", err)
	}
	
	return client
}

// HTTPTestRequest creates an HTTP test request with proper headers
func HTTPTestRequest(t *testing.T, method, url string, body io.Reader) *http.Request {
	t.Helper()
	
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-api-key-12345")
	
	return req
}

// AssertJSONResponse validates a JSON HTTP response
func AssertJSONResponse(t *testing.T, resp *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) {
	t.Helper()
	
	if resp.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, resp.Code)
	}
	
	contentType := resp.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	
	for field, expectedValue := range expectedFields {
		if actualValue, exists := response[field]; !exists {
			t.Errorf("Expected field %s not found in response", field)
		} else if actualValue != expectedValue {
			t.Errorf("Field %s: expected %v, got %v", field, expectedValue, actualValue)
		}
	}
}

// LoadTestData loads test data from a file
func LoadTestData(t *testing.T, filename string) []byte {
	t.Helper()
	
	path := filepath.Join("../../testdata", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to load test data from %s: %v", path, err)
	}
	
	return data
}

// LoadTestJSON loads and parses JSON test data
func LoadTestJSON(t *testing.T, filename string, v interface{}) {
	t.Helper()
	
	data := LoadTestData(t, filename)
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("Failed to parse JSON test data from %s: %v", filename, err)
	}
}

// TempDir creates a temporary directory for tests
func TempDir(t *testing.T) string {
	t.Helper()
	
	dir, err := os.MkdirTemp("", "secauto-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	
	return dir
}

// MockHTTPServer creates a mock HTTP server for testing external integrations
func MockHTTPServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	
	server := httptest.NewServer(handler)
	
	t.Cleanup(func() {
		server.Close()
	})
	
	return server
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()
	
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	
	t.Fatalf("Condition not met within %v: %s", timeout, message)
}

// AssertNoError is a helper to assert no error occurred
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// AssertError is a helper to assert an error occurred
func AssertError(t *testing.T, err error, expectedMessage string) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if expectedMessage != "" && !strings.Contains(err.Error(), expectedMessage) {
		t.Fatalf("Expected error containing %q, got %q", expectedMessage, err.Error())
	}
}

// CreateTestFile creates a temporary test file with content
func CreateTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", path, err)
	}
	
	return path
}

// RandomString generates a random string for testing
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}