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

// IntegrationManager manages global integration definitions and their environments
type IntegrationManager struct {
	definitions map[string]*IntegrationDefinition
	configsPath string
	scriptsPath string
	envPath     string // Path for integration environments (site-packages)
	pythonPath  string // Path to Python interpreter
	mutex       sync.RWMutex
	logger      types.Logger
}

// NewIntegrationManager creates a new integration manager
func NewIntegrationManager(configsPath, scriptsPath, pythonPath string, logger types.Logger) *IntegrationManager {
	im := &IntegrationManager{
		definitions: make(map[string]*IntegrationDefinition),
		configsPath: configsPath,
		scriptsPath: scriptsPath,
		envPath:     filepath.Join(filepath.Dir(configsPath), ".site-packages"),
		pythonPath:  pythonPath,
		logger:      logger,
	}
	
	// Ensure directories exist
	os.MkdirAll(im.configsPath, 0755)
	os.MkdirAll(im.scriptsPath, 0755)
	os.MkdirAll(im.envPath, 0755)
	
	// Load existing integration definitions
	im.loadDefinitions()
	
	return im
}

// loadDefinitions loads integration definitions from disk
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

// UploadIntegration handles uploading a new integration
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

	// Build the integration environment if required
	if definition.Backend.RequiresBuild {
		if err := im.buildIntegrationEnvironment(definition); err != nil {
			// Rollback
			os.Remove(scriptPath)
			os.Remove(definitionPath)
			return fmt.Errorf("failed to build integration environment: %v", err)
		}
	}

	// Store in memory
	im.definitions[definition.Name] = definition

	im.logger.Info("Integration uploaded successfully", map[string]interface{}{
		"integration": definition.Name,
		"version":     definition.Version,
	})

	return nil
}

// buildIntegrationEnvironment creates the site-packages environment for an integration
func (im *IntegrationManager) buildIntegrationEnvironment(definition *IntegrationDefinition) error {
	// Create integration-specific environment directory
	integrationEnvPath := filepath.Join(im.envPath, definition.Name)
	if err := os.MkdirAll(integrationEnvPath, 0755); err != nil {
		return fmt.Errorf("failed to create environment directory: %v", err)
	}

	// Install packages if specified
	if len(definition.Dependencies.Packages) > 0 {
		im.logger.Info("Installing integration dependencies", map[string]interface{}{
			"integration": definition.Name,
			"packages":    definition.Dependencies.Packages,
		})

		for _, pkg := range definition.Dependencies.Packages {
			cmd := exec.Command(im.pythonPath, "-m", "pip", "install", 
				"--target", integrationEnvPath, 
				"--upgrade", pkg)
			
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				im.logger.Error("Failed to install package", map[string]interface{}{
					"integration": definition.Name,
					"package":     pkg,
					"error":       err.Error(),
					"stderr":      stderr.String(),
				})
				return fmt.Errorf("failed to install %s: %v", pkg, err)
			}
		}
	}

	// Run post-install commands if specified
	for _, cmdStr := range definition.Build.PostInstall {
		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			continue
		}

		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = integrationEnvPath
		
		// Set environment variables
		for k, v := range definition.Build.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		if err := cmd.Run(); err != nil {
			im.logger.Warning("Post-install command failed", map[string]interface{}{
				"integration": definition.Name,
				"command":     cmdStr,
				"error":       err.Error(),
			})
		}
	}

	// Create a marker file to indicate successful build
	markerPath := filepath.Join(integrationEnvPath, ".build_complete")
	os.WriteFile(markerPath, []byte(time.Now().UTC().Format(time.RFC3339)), 0644)

	im.logger.Info("Integration environment built successfully", map[string]interface{}{
		"integration": definition.Name,
		"path":        integrationEnvPath,
	})

	return nil
}

// GetIntegration retrieves an integration definition
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

// ListIntegrations returns all available integrations
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

// DeleteIntegration removes an integration and its environment
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

	// Remove environment directory
	envPath := filepath.Join(im.envPath, name)
	os.RemoveAll(envPath)

	// Remove from memory
	delete(im.definitions, name)

	im.logger.Info("Integration deleted", map[string]interface{}{
		"integration": name,
	})

	return nil
}

// ExecuteIntegration runs an integration with client-specific configuration
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

	// Get script path
	scriptPath := filepath.Join(im.scriptsPath, definition.Backend.EntryPoint)

	// Create command with timeout
	timeout := time.Duration(definition.Backend.Timeout) * time.Second
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	// Prepare Python path with integration's site-packages
	integrationEnvPath := filepath.Join(im.envPath, integrationName)
	
	cmd := exec.Command(im.pythonPath, scriptPath)
	
	// Set PYTHONPATH to include integration's site-packages
	pythonPath := integrationEnvPath
	if existingPath := os.Getenv("PYTHONPATH"); existingPath != "" {
		pythonPath = integrationEnvPath + string(os.PathListSeparator) + existingPath
	}
	
	cmd.Env = append(os.Environ(), 
		fmt.Sprintf("PYTHONPATH=%s", pythonPath),
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
		return nil, fmt.Errorf("failed to start integration: %v", err)
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

// GetIntegrationEnvironmentPath returns the site-packages path for an integration
func (im *IntegrationManager) GetIntegrationEnvironmentPath(integrationName string) string {
	return filepath.Join(im.envPath, integrationName)
}

// IsIntegrationBuilt checks if an integration's environment has been built
func (im *IntegrationManager) IsIntegrationBuilt(integrationName string) bool {
	markerPath := filepath.Join(im.envPath, integrationName, ".build_complete")
	_, err := os.Stat(markerPath)
	return err == nil
}

// RebuildIntegrationEnvironment rebuilds an integration's environment
func (im *IntegrationManager) RebuildIntegrationEnvironment(integrationName string) error {
	im.mutex.RLock()
	definition, exists := im.definitions[integrationName]
	im.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("integration not found: %s", integrationName)
	}

	// Remove existing environment
	envPath := filepath.Join(im.envPath, integrationName)
	os.RemoveAll(envPath)

	// Rebuild
	return im.buildIntegrationEnvironment(definition)
}