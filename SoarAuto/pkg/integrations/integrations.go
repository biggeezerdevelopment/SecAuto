package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"SoarAuto/pkg/types"
)

// IntegrationDefinition represents a global integration with its capabilities
type IntegrationDefinition struct {
	Name         string                          `json:"name"`
	Version      string                          `json:"version"`
	Description  string                          `json:"description"`
	Author       string                          `json:"author"`
	Dependencies IntegrationDependencies         `json:"dependencies"`
	Backend      IntegrationBackend              `json:"backend"`
	Configuration map[string]ConfigurationField  `json:"configuration"`
	Functions    map[string]FunctionDefinition   `json:"functions"`
	Build        BuildConfiguration              `json:"build"`
	CreatedAt    string                          `json:"created_at"`
	UpdatedAt    string                          `json:"updated_at"`
}

// IntegrationDependencies defines required packages and dependencies
type IntegrationDependencies struct {
	Packages []string `json:"packages"`
	System   []string `json:"system"`
	Optional []string `json:"optional"`
}

// IntegrationBackend defines how the integration runs
type IntegrationBackend struct {
	Type          string `json:"type"`           // "python", "go", etc.
	EntryPoint    string `json:"entry_point"`    // Script filename
	Timeout       int    `json:"timeout"`        // Execution timeout in seconds
	MemoryLimit   int    `json:"memory_limit"`   // Memory limit in MB
	RequiresBuild bool   `json:"requires_build"` // Whether to build environment
}

// ConfigurationField defines a configuration parameter
type ConfigurationField struct {
	Type        string      `json:"type"`        // "string", "number", "boolean", "array", "object"
	Required    bool        `json:"required"`
	Description string      `json:"description"`
	Example     interface{} `json:"example,omitempty"`
	Encrypted   bool        `json:"encrypted,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// FunctionDefinition defines an integration function
type FunctionDefinition struct {
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Returns     string                 `json:"returns,omitempty"`
}

// BuildConfiguration defines build steps for the integration
type BuildConfiguration struct {
	PreInstall   []string          `json:"pre_install"`
	PostInstall  []string          `json:"post_install"`
	Environment  map[string]string `json:"environment"`
}

// IntegrationManager manages integrations with fast, isolated execution
type IntegrationManager struct {
	definitions map[string]*IntegrationDefinition
	configsPath string
	scriptsPath string
	venvPath    string // Path for UV virtual environments
	builderPath string // Path to UV builder script
	pythonPath  string // Path to Python interpreter
	mutex       sync.RWMutex
	logger      types.Logger
}

// NewIntegrationManager creates a new integration manager
func NewIntegrationManager(configsPath, scriptsPath, pythonPath string, logger types.Logger) *IntegrationManager {
	// Make paths absolute if they're relative
	if !filepath.IsAbs(configsPath) {
		workDir, _ := os.Getwd()
		configsPath = filepath.Join(workDir, configsPath)
	}
	if !filepath.IsAbs(scriptsPath) {
		workDir, _ := os.Getwd()
		scriptsPath = filepath.Join(workDir, scriptsPath)
	}
	
	baseDir := filepath.Dir(configsPath)
	
	// Calculate project root path from current working directory (should be in SoarAuto/)
	workDir, _ := os.Getwd()
	projectRoot := filepath.Dir(workDir)
	
	im := &IntegrationManager{
		definitions: make(map[string]*IntegrationDefinition),
		configsPath: configsPath,
		scriptsPath: scriptsPath,
		venvPath:    filepath.Join(baseDir, "venvs"), // Virtual environments directory
		builderPath: filepath.Join(projectRoot, "scripts", "integration_builder.py"),
		pythonPath:  pythonPath,
		logger:      logger,
	}
	
	// Ensure directories exist
	os.MkdirAll(im.configsPath, 0755)
	os.MkdirAll(im.scriptsPath, 0755)
	os.MkdirAll(im.venvPath, 0755)
	
	// Load existing integration definitions
	im.loadDefinitions()
	
	return im
}

// checkUVAvailable checks if UV is available on the system
func (im *IntegrationManager) checkUVAvailable() error {
	cmd := exec.Command("uv", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("UV is required but not found. Install with: curl -LsSf https://astral.sh/uv/install.sh | sh")
	}
	return nil
}

// loadDefinitions loads integration definitions from disk (same as original)
func (im *IntegrationManager) loadDefinitions() {
	files, err := filepath.Glob(filepath.Join(im.configsPath, "*.json"))
	if err != nil {
		im.logger.Warning("Failed to list integration configs", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var definition IntegrationDefinition
		if err := json.Unmarshal(data, &definition); err != nil {
			continue
		}

		// Extract name from filename if not set
		if definition.Name == "" {
			name := filepath.Base(file)
			name = strings.TrimSuffix(name, ".json")
			definition.Name = name
		}

		im.definitions[definition.Name] = &definition
	}

	im.logger.Info("Loaded integration definitions", map[string]interface{}{
		"count": len(im.definitions),
	})
}

// UploadIntegration handles uploading a new integration with UV building
func (im *IntegrationManager) UploadIntegration(definition *IntegrationDefinition, scriptContent []byte) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	// Validate the definition
	if definition.Name == "" {
		return fmt.Errorf("integration name is required")
	}
	if definition.Backend.EntryPoint == "" {
		return fmt.Errorf("integration entry point is required")
	}

	// Check UV availability
	if err := im.checkUVAvailable(); err != nil {
		return fmt.Errorf("UV check failed: %v", err)
	}

	// Set timestamps
	if definition.CreatedAt == "" {
		definition.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	definition.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Save the script file
	scriptPath := filepath.Join(im.scriptsPath, definition.Backend.EntryPoint)
	if err := os.WriteFile(scriptPath, scriptContent, 0644); err != nil {
		return fmt.Errorf("failed to save integration script: %v", err)
	}

	// Save the definition
	definitionData, err := json.MarshalIndent(definition, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal integration definition: %v", err)
	}

	definitionPath := filepath.Join(im.configsPath, definition.Name+".json")
	if err := os.WriteFile(definitionPath, definitionData, 0644); err != nil {
		// Rollback script file
		os.Remove(scriptPath)
		return fmt.Errorf("failed to save integration definition: %v", err)
	}

	// Build the integration environment with UV if required
	if definition.Backend.RequiresBuild {
		if err := im.buildIntegrationEnvironment(definition); err != nil {
			// Rollback
			os.Remove(scriptPath)
			os.Remove(definitionPath)
			return fmt.Errorf("failed to build integration environment with UV: %v", err)
		}
	}

	// Store in memory
	im.definitions[definition.Name] = definition

	im.logger.Info("Integration uploaded successfully with UV", map[string]interface{}{
		"integration": definition.Name,
		"version":     definition.Version,
		"builder":     "UV",
	})

	return nil
}

// buildIntegrationEnvironment uses the UV builder to create the environment
func (im *IntegrationManager) buildIntegrationEnvironment(definition *IntegrationDefinition) error {
	im.logger.Info("Building integration environment with UV", map[string]interface{}{
		"integration": definition.Name,
	})

	// Use the UV builder script
	configPath := filepath.Join(im.configsPath, definition.Name+".json")
	
	cmd := exec.Command(im.pythonPath, im.builderPath, "build", "--config", configPath)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("UV builder failed: %v\nStderr: %s\nStdout: %s", 
			err, stderr.String(), stdout.String())
	}

	// Parse the result
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return fmt.Errorf("failed to parse UV builder result: %v", err)
	}

	if !result["success"].(bool) {
		return fmt.Errorf("UV builder reported failure: %v", result["error"])
	}

	im.logger.Info("Integration environment built successfully with UV", map[string]interface{}{
		"integration": definition.Name,
		"builder":     "UV",
		"venv_path":   result["venv_path"],
	})

	return nil
}

// ExecuteIntegration runs an integration using UV for optimal performance
func (im *IntegrationManager) ExecuteIntegration(integrationName string, clientConfig *ClientIntegrationConfig, function string, params map[string]interface{}) (interface{}, error) {
	im.mutex.RLock()
	definition, exists := im.definitions[integrationName]
	im.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("integration not found: %s", integrationName)
	}

	// Validate the function exists
	if _, exists := definition.Functions[function]; !exists {
		return nil, fmt.Errorf("function '%s' not found in integration '%s'", function, integrationName)
	}

	// Check UV availability
	if err := im.checkUVAvailable(); err != nil {
		return nil, fmt.Errorf("UV check failed: %v", err)
	}

	// Check if integration environment needs to be built
	if definition.Backend.RequiresBuild && !im.IsIntegrationBuilt(integrationName) {
		im.logger.Info("Building integration environment with UV", map[string]interface{}{
			"integration": integrationName,
		})
		if err := im.buildIntegrationEnvironment(definition); err != nil {
			return nil, fmt.Errorf("failed to build integration environment: %v", err)
		}
	}

	// Prepare the execution context
	context := map[string]interface{}{
		"function": function,
		"params":   params,
		"config":   clientConfig.Config,
		"credentials": clientConfig.Credentials,
	}

	contextJSON, err := json.Marshal(context)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context: %v", err)
	}

	// Get script path and venv path
	scriptPath := filepath.Join(im.scriptsPath, definition.Backend.EntryPoint)
	venvPath := filepath.Join(im.venvPath, integrationName)
	pythonPath := filepath.Join(venvPath, "bin", "python")

	// Check if venv exists
	if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("UV virtual environment not found for integration: %s", integrationName)
	}

	// Create command with timeout using UV run for optimal execution
	timeout := time.Duration(definition.Backend.Timeout) * time.Second
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	// Use UV run for the best performance and isolation
	cmd := exec.Command("uv", "run", "--python", pythonPath, scriptPath)
	
	// Set environment variables
	secautoRoot, _ := filepath.Abs(filepath.Join(filepath.Dir(im.configsPath), ".."))
	cmd.Env = append(os.Environ(), 
		fmt.Sprintf("SECAUTO_ROOT=%s", secautoRoot),
		fmt.Sprintf("INTEGRATION_NAME=%s", integrationName),
		fmt.Sprintf("INTEGRATION_FUNCTION=%s", function),
	)

	// Add build environment variables
	for k, v := range definition.Build.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Pass context via stdin
	cmd.Stdin = bytes.NewReader(contextJSON)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start integration with UV: %v", err)
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("integration execution failed: %v\nStderr: %s", err, stderr.String())
		}
	case <-time.After(timeout):
		cmd.Process.Kill()
		return nil, fmt.Errorf("integration execution timed out after %v", timeout)
	}

	// Parse output
	outputStr := strings.TrimSpace(stdout.String())
	if outputStr == "" {
		return map[string]interface{}{
			"success": true,
			"message": "Integration executed successfully with no output",
		}, nil
	}

	// Try to parse as JSON
	var result interface{}
	if err := json.Unmarshal([]byte(outputStr), &result); err != nil {
		// Return as plain text if not JSON
		return map[string]interface{}{
			"success": true,
			"output":  outputStr,
		}, nil
	}

	return result, nil
}

// IsIntegrationBuilt checks if an integration's UV environment has been built
func (im *IntegrationManager) IsIntegrationBuilt(integrationName string) bool {
	venvPath := filepath.Join(im.venvPath, integrationName)
	pythonPath := filepath.Join(venvPath, "bin", "python")
	_, err := os.Stat(pythonPath)
	if err != nil && im.logger != nil {
		im.logger.Debug("Integration built check", map[string]interface{}{
			"integration": integrationName,
			"venvPath": venvPath,
			"pythonPath": pythonPath,
			"exists": err == nil,
		})
	}
	return err == nil
}

// GetIntegrationEnvironmentPath returns the venv path for an integration
func (im *IntegrationManager) GetIntegrationEnvironmentPath(integrationName string) string {
	return filepath.Join(im.venvPath, integrationName)
}

// CleanIntegration removes an integration's UV environment
func (im *IntegrationManager) CleanIntegration(integrationName string) error {
	im.logger.Info("Cleaning UV integration environment", map[string]interface{}{
		"integration": integrationName,
	})

	// Use the UV builder to clean
	cmd := exec.Command(im.pythonPath, im.builderPath, "clean", "--name", integrationName)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		im.logger.Error("Failed to clean UV integration environment", map[string]interface{}{
			"integration": integrationName,
			"error":       err.Error(),
			"stderr":      stderr.String(),
		})
		return fmt.Errorf("failed to clean UV environment: %v", err)
	}

	im.logger.Info("Successfully cleaned UV integration environment", map[string]interface{}{
		"integration": integrationName,
	})

	return nil
}

// RebuildIntegrationEnvironment rebuilds an integration's UV environment
func (im *IntegrationManager) RebuildIntegrationEnvironment(integrationName string) error {
	im.mutex.RLock()
	definition, exists := im.definitions[integrationName]
	im.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("integration not found: %s", integrationName)
	}

	// Clean existing environment
	im.CleanIntegration(integrationName)

	// Rebuild with UV
	return im.buildIntegrationEnvironment(definition)
}

// ListIntegrations returns all available integrations (same as original)
func (im *IntegrationManager) ListIntegrations() []*IntegrationDefinition {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	integrations := make([]*IntegrationDefinition, 0, len(im.definitions))
	for _, def := range im.definitions {
		defCopy := *def
		integrations = append(integrations, &defCopy)
	}

	return integrations
}

// GetIntegration retrieves an integration definition (same as original)
func (im *IntegrationManager) GetIntegration(name string) (*IntegrationDefinition, error) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	definition, exists := im.definitions[name]
	if !exists {
		return nil, fmt.Errorf("integration not found: %s", name)
	}

	// Return a copy
	defCopy := *definition
	return &defCopy, nil
}

// GetDefinition returns the definition for a specific integration (same as original)
func (im *IntegrationManager) GetDefinition(name string) (*IntegrationDefinition, error) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()
	
	definition, exists := im.definitions[name]
	if !exists {
		return nil, fmt.Errorf("integration not found: %s", name)
	}
	
	return definition, nil
}

// DeleteIntegration removes an integration and its UV environment
func (im *IntegrationManager) DeleteIntegration(name string) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	definition, exists := im.definitions[name]
	if !exists {
		return fmt.Errorf("integration not found: %s", name)
	}

	// Remove script file
	scriptPath := filepath.Join(im.scriptsPath, definition.Backend.EntryPoint)
	os.Remove(scriptPath)

	// Remove definition file
	definitionPath := filepath.Join(im.configsPath, name+".json")
	os.Remove(definitionPath)

	// Clean UV environment
	im.CleanIntegration(name)

	// Remove from memory
	delete(im.definitions, name)

	im.logger.Info("Integration deleted with UV cleanup", map[string]interface{}{
		"integration": name,
	})

	return nil
}

// MigrateFromOldSystem migrates an integration from the old .site-packages system to UV
func (im *IntegrationManager) MigrateFromOldSystem(integrationName string) error {
	im.logger.Info("Migrating integration to UV system", map[string]interface{}{
		"integration": integrationName,
	})

	// Use the UV builder to migrate
	cmd := exec.Command(im.pythonPath, im.builderPath, "migrate", "--name", integrationName)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("migration failed: %v\nStderr: %s", err, stderr.String())
	}

	// Parse the result
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return fmt.Errorf("failed to parse migration result: %v", err)
	}

	if !result["success"].(bool) {
		return fmt.Errorf("migration failed: %v", result["error"])
	}

	im.logger.Info("Successfully migrated integration to UV", map[string]interface{}{
		"integration": integrationName,
		"message":     result["message"],
	})

	return nil
}