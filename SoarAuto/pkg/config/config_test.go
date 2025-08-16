package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"SoarAuto/pkg/testutil"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		validate    func(*testing.T, *Config)
	}{
		{
			name: "valid config",
			configYAML: `
server:
  port: 8080
  host: "localhost"
logging:
  level: "INFO"
database:
  redis_url: "redis://localhost:6379/0"
security:
  api_keys:
    - "test-key"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Server.Port != 8080 {
					t.Errorf("Expected port 8080, got %d", cfg.Server.Port)
				}
				if cfg.Server.Host != "localhost" {
					t.Errorf("Expected host localhost, got %s", cfg.Server.Host)
				}
				if cfg.Logging.Level != "INFO" {
					t.Errorf("Expected log level INFO, got %s", cfg.Logging.Level)
				}
			},
		},
		{
			name: "invalid yaml",
			configYAML: `
server:
  port: invalid
  host: "localhost"
`,
			expectError: true,
		},
		{
			name: "missing file",
			configYAML: "",
			expectError: false, // Should use defaults
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Server.Port != 8000 { // Default port
					t.Errorf("Expected default port 8000, got %d", cfg.Server.Port)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string
			
			if tt.configYAML != "" {
				// Create temporary config file
				tempDir := testutil.TempDir(t)
				configPath = testutil.CreateTestFile(t, tempDir, "config.yaml", tt.configYAML)
			} else {
				// Use non-existent file to test defaults
				configPath = "non-existent-config.yaml"
			}

			cfg, err := LoadConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			testutil.AssertNoError(t, err)

			if cfg == nil {
				t.Fatal("Expected config but got nil")
			}

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "localhost",
				},
				Database: DatabaseConfig{
					RedisURL: "redis://localhost:6379/0",
				},
			},
			expectError: false,
		},
		{
			name: "invalid port - too low",
			config: &Config{
				Server: ServerConfig{
					Port: 0,
					Host: "localhost",
				},
			},
			expectError: true,
			errorMsg:    "invalid server port",
		},
		{
			name: "invalid port - too high",
			config: &Config{
				Server: ServerConfig{
					Port: 70000,
					Host: "localhost",
				},
			},
			expectError: true,
			errorMsg:    "invalid server port",
		},
		{
			name: "invalid redis url",
			config: &Config{
				Server: ServerConfig{
					Port: 8080,
					Host: "localhost",
				},
				Database: DatabaseConfig{
					RedisURL: "invalid-url",
				},
			},
			expectError: true,
			errorMsg:    "invalid Redis URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				testutil.AssertError(t, err, tt.errorMsg)
			} else {
				testutil.AssertNoError(t, err)
			}
		})
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	// Save original environment
	originalPort := os.Getenv("SECAUTO_PORT")
	originalAPIKey := os.Getenv("SECAUTO_API_KEY")
	
	// Cleanup
	defer func() {
		os.Setenv("SECAUTO_PORT", originalPort)
		os.Setenv("SECAUTO_API_KEY", originalAPIKey)
	}()

	// Set test environment variables
	os.Setenv("SECAUTO_PORT", "9999")
	os.Setenv("SECAUTO_API_KEY", "env-test-key")

	tempDir := testutil.TempDir(t)
	configYAML := `
server:
  port: 8080
security:
  api_keys:
    - "config-key"
`
	configPath := testutil.CreateTestFile(t, tempDir, "config.yaml", configYAML)

	cfg, err := LoadConfig(configPath)
	testutil.AssertNoError(t, err)

	// Check that environment variables override config
	if cfg.Server.Port != 9999 {
		t.Errorf("Expected port 9999 from env, got %d", cfg.Server.Port)
	}

	// Check that API key was appended
	found := false
	for _, key := range cfg.Security.APIKeys {
		if key == "env-test-key" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Environment API key not found in config")
	}
}

func TestConfigHelperMethods(t *testing.T) {
	cfg := &Config{
		Python: PythonConfig{
			VenvPath:      "/path/to/venv",
			ScriptsPath:   "/path/to/scripts",
			PlaybooksPath: "/path/to/playbooks",
			DefaultScript: "default.py",
		},
		Security: SecurityConfig{
			APIKeys: []string{"key1", "key2"},
		},
	}

	t.Run("GetVenvPath", func(t *testing.T) {
		path := cfg.GetVenvPath()
		if path != "/path/to/venv" {
			t.Errorf("Expected /path/to/venv, got %s", path)
		}
	})

	t.Run("GetScriptPath", func(t *testing.T) {
		path := cfg.GetScriptPath("test_script")
		expected := filepath.Join("/path/to/scripts", "test_script.py")
		if path != expected {
			t.Errorf("Expected %s, got %s", expected, path)
		}

		// Test with .py extension
		path = cfg.GetScriptPath("test_script.py")
		if path != expected {
			t.Errorf("Expected %s, got %s", expected, path)
		}

		// Test with empty name (should use default)
		path = cfg.GetScriptPath("")
		expected = filepath.Join("/path/to/scripts", "default.py")
		if path != expected {
			t.Errorf("Expected %s, got %s", expected, path)
		}
	})

	t.Run("GetPlaybookPath", func(t *testing.T) {
		path := cfg.GetPlaybookPath("test_playbook")
		expected := filepath.Join("/path/to/playbooks", "test_playbook.json")
		if path != expected {
			t.Errorf("Expected %s, got %s", expected, path)
		}

		// Test with .json extension
		path = cfg.GetPlaybookPath("test_playbook.json")
		if path != expected {
			t.Errorf("Expected %s, got %s", expected, path)
		}
	})

	t.Run("IsAPIKeyValid", func(t *testing.T) {
		if !cfg.IsAPIKeyValid("key1") {
			t.Error("Expected key1 to be valid")
		}
		if !cfg.IsAPIKeyValid("key2") {
			t.Error("Expected key2 to be valid")
		}
		if cfg.IsAPIKeyValid("invalid-key") {
			t.Error("Expected invalid-key to be invalid")
		}
	})
}

func TestMergeConfig(t *testing.T) {
	defaults := &Config{
		Server: ServerConfig{
			Port: 8000,
			Host: "localhost",
		},
		Logging: LoggingConfig{
			Level: "INFO",
		},
		Database: DatabaseConfig{
			RedisURL: "redis://localhost:6379/0",
		},
		Security: SecurityConfig{
			APIKeys: []string{"default-key"},
		},
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: 9000, // Override
			// Host missing - should use default
		},
		// Logging missing - should use default
		Database: DatabaseConfig{
			RedisURL: "redis://custom:6379/1", // Override
		},
		// Security missing - should use default
	}

	mergeConfig(cfg, defaults)

	// Check overrides
	if cfg.Server.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", cfg.Server.Port)
	}
	if cfg.Database.RedisURL != "redis://custom:6379/1" {
		t.Errorf("Expected custom Redis URL, got %s", cfg.Database.RedisURL)
	}

	// Check defaults were applied
	if cfg.Server.Host != "localhost" {
		t.Errorf("Expected default host localhost, got %s", cfg.Server.Host)
	}
	if cfg.Logging.Level != "INFO" {
		t.Errorf("Expected default log level INFO, got %s", cfg.Logging.Level)
	}
	if len(cfg.Security.APIKeys) == 0 || cfg.Security.APIKeys[0] != "default-key" {
		t.Errorf("Expected default API keys, got %v", cfg.Security.APIKeys)
	}
}

// Add the Validate method to Config for testing
func (c *Config) Validate() error {
	// Validate server configuration
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate Redis URL if provided
	if c.Database.RedisURL != "" {
		if _, err := parseRedisURL(c.Database.RedisURL); err != nil {
			return fmt.Errorf("invalid Redis URL: %v", err)
		}
	}

	// Validate TLS configuration
	if c.Security.TLS.Enabled {
		if !c.Security.TLS.AutoCert.Enabled {
			if c.Security.TLS.CertFile == "" || c.Security.TLS.KeyFile == "" {
				return fmt.Errorf("TLS enabled but no certificates configured")
			}
		}
	}

	return nil
}

// Helper function to parse Redis URL (simplified version)
func parseRedisURL(url string) (map[string]string, error) {
	if !strings.HasPrefix(url, "redis://") {
		return nil, fmt.Errorf("invalid Redis URL format")
	}
	return map[string]string{"url": url}, nil
}

func TestConfigDefaults(t *testing.T) {
	// Test that defaults are properly set when config file doesn't exist
	cfg, err := LoadConfig("non-existent-file.yaml")
	testutil.AssertNoError(t, err)

	// Verify default values
	if cfg.Server.Port != 8000 {
		t.Errorf("Expected default port 8000, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("Expected default host localhost, got %s", cfg.Server.Host)
	}
	if cfg.Logging.Level != "INFO" {
		t.Errorf("Expected default log level INFO, got %s", cfg.Logging.Level)
	}
	if cfg.Database.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("Expected default Redis URL, got %s", cfg.Database.RedisURL)
	}
}