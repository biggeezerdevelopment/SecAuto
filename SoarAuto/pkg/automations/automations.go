package automations

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"SoarAuto/pkg/security"
	"SoarAuto/pkg/types"
)

// AutomationManager manages automation scripts and metadata
type AutomationManager struct {
	scriptsPath  string
	metadataPath string
	metadata     map[string]*types.AutomationMetadata
	mutex        sync.RWMutex
}

// NewAutomationManager creates a new automation manager
func NewAutomationManager(scriptsPath, metadataPath string) *AutomationManager {
	am := &AutomationManager{
		scriptsPath:  scriptsPath,
		metadataPath: metadataPath,
		metadata:     make(map[string]*types.AutomationMetadata),
	}

	// Load existing metadata
	am.loadMetadata()

	return am
}

// loadMetadata loads automation metadata from disk
func (am *AutomationManager) loadMetadata() {
	// Ensure metadata directory exists
	if err := os.MkdirAll(am.metadataPath, 0755); err != nil {
		return
	}

	// Load metadata files
	files, err := filepath.Glob(filepath.Join(am.metadataPath, "*.json"))
	if err != nil {
		return
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var metadata types.AutomationMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			continue
		}

		// Extract name from filename
		name := filepath.Base(file)
		name = name[:len(name)-5] // Remove .json extension
		metadata.Name = name

		am.metadata[name] = &metadata
	}
}

// ListAutomations returns information about all automation scripts
func (am *AutomationManager) ListAutomations() ([]types.AutomationInfo, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	var automations []types.AutomationInfo

	// Ensure scripts directory exists
	if err := os.MkdirAll(am.scriptsPath, 0755); err != nil {
		return nil, err
	}

	// Read all Python files in the scripts directory
	files, err := filepath.Glob(filepath.Join(am.scriptsPath, "*.py"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		info, err := am.getAutomationInfo(file)
		if err != nil {
			continue // Skip files that can't be processed
		}
		automations = append(automations, *info)
	}

	return automations, nil
}

// getAutomationInfo analyzes a Python file and returns automation information
func (am *AutomationManager) getAutomationInfo(filePath string) (*types.AutomationInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	name := filepath.Base(filePath)
	name = name[:len(name)-3] // Remove .py extension

	// Analyze the Python file
	lineCount, functionCount, importCount, isValid := am.analyzePythonFile(filePath)

	// Determine language and file type
	language := "python"
	fileType := "script"
	if strings.Contains(name, "integration") {
		fileType = "integration"
	} else if strings.Contains(name, "automation") {
		fileType = "automation"
	}

	return &types.AutomationInfo{
		Name:          name,
		Filename:      filepath.Base(filePath),
		Size:          stat.Size(),
		FileType:      fileType,
		Language:      language,
		LineCount:     lineCount,
		FunctionCount: functionCount,
		ImportCount:   importCount,
		ModifiedAt:    stat.ModTime().Format(time.RFC3339),
		IsValid:       isValid,
	}, nil
}

// analyzePythonFile analyzes a Python file for metrics
func (am *AutomationManager) analyzePythonFile(filePath string) (lineCount, functionCount, importCount int, isValid bool) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, 0, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	isValid = true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineCount++

		// Count function definitions
		if strings.HasPrefix(line, "def ") {
			functionCount++
		}

		// Count imports
		if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "from ") {
			importCount++
		}

		// Basic syntax validation
		if strings.Contains(line, "SyntaxError") {
			isValid = false
		}
	}

	if err := scanner.Err(); err != nil {
		isValid = false
	}

	return lineCount, functionCount, importCount, isValid
}

// SaveAutomationScript saves an automation script to disk
func (am *AutomationManager) SaveAutomationScript(name string, content []byte) error {
	// Validate the automation name for security
	if err := security.ValidateUploadName(name); err != nil {
		return fmt.Errorf("invalid automation name: %v", err)
	}

	// Validate file extension
	if err := security.ValidateFileExtension(name+".py", []string{".py"}); err != nil {
		return err
	}

	// Validate content size (10MB limit)
	if err := security.ValidateContentSize(int64(len(content)), 10*1024*1024); err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(am.scriptsPath, 0755); err != nil {
		return err
	}

	// Securely join the path
	filename, err := security.SecureJoinPath(am.scriptsPath, name+".py")
	if err != nil {
		return fmt.Errorf("failed to create secure file path: %v", err)
	}

	return os.WriteFile(filename, content, 0644)
}

// DeleteAutomationScript deletes an automation script
func (am *AutomationManager) DeleteAutomationScript(name string) error {
	filename := filepath.Join(am.scriptsPath, name+".py")
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Also delete metadata if it exists
	am.DeleteMetadata(name)

	return nil
}

// ListMetadata returns all automation metadata
func (am *AutomationManager) ListMetadata() []types.AutomationMetadata {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	var result []types.AutomationMetadata
	for _, metadata := range am.metadata {
		metadataCopy := *metadata
		result = append(result, metadataCopy)
	}
	return result
}

// GetMetadata retrieves metadata for a specific automation
func (am *AutomationManager) GetMetadata(name string) (*types.AutomationMetadata, bool) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	metadata, exists := am.metadata[name]
	if !exists {
		return nil, false
	}

	// Return a copy
	metadataCopy := *metadata
	return &metadataCopy, true
}

// CreateMetadata creates metadata for an automation
func (am *AutomationManager) CreateMetadata(name string, metadata *types.AutomationMetadata) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check if already exists
	if _, exists := am.metadata[name]; exists {
		return fmt.Errorf("metadata for automation %s already exists", name)
	}

	// Set metadata
	metadata.Name = name
	metadata.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	metadata.UpdatedAt = metadata.CreatedAt

	// Save to disk
	if err := am.saveMetadata(name, metadata); err != nil {
		return err
	}

	// Store in memory
	am.metadata[name] = metadata

	return nil
}

// UpdateMetadata updates metadata for an automation
func (am *AutomationManager) UpdateMetadata(name string, metadata *types.AutomationMetadata) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Preserve creation time, update modification time
	if existing := am.metadata[name]; existing != nil {
		metadata.CreatedAt = existing.CreatedAt
	}
	metadata.Name = name
	metadata.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Save to disk
	if err := am.saveMetadata(name, metadata); err != nil {
		return err
	}

	// Store in memory
	am.metadata[name] = metadata

	return nil
}

// DeleteMetadata deletes metadata for an automation
func (am *AutomationManager) DeleteMetadata(name string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Remove from disk
	metadataFile := filepath.Join(am.metadataPath, name+".json")
	if err := os.Remove(metadataFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove from memory
	delete(am.metadata, name)

	return nil
}

// saveMetadata saves metadata to disk
func (am *AutomationManager) saveMetadata(name string, metadata *types.AutomationMetadata) error {
	// Ensure directory exists
	if err := os.MkdirAll(am.metadataPath, 0755); err != nil {
		return err
	}

	// Serialize metadata
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	metadataFile := filepath.Join(am.metadataPath, name+".json")
	return os.WriteFile(metadataFile, data, 0644)
}

// GetAutomationDependencies returns dependencies for an automation
func (am *AutomationManager) GetAutomationDependencies(name string) ([]string, error) {
	// For now, return empty dependencies
	// In a real implementation, this would analyze the Python file for imports
	// and determine which other automations or integrations it depends on
	return []string{}, nil
}

// ValidateAutomationMetadata validates automation metadata
func (am *AutomationManager) ValidateAutomationMetadata(metadata *types.AutomationMetadata) []types.ValidationError {
	var errors []types.ValidationError

	if metadata.Name == "" {
		errors = append(errors, types.ValidationError{
			Field:   "name",
			Message: "Automation name is required",
		})
	}

	if metadata.Description == "" {
		errors = append(errors, types.ValidationError{
			Field:   "description",
			Message: "Automation description is required",
		})
	}

	if metadata.Version == "" {
		errors = append(errors, types.ValidationError{
			Field:   "version",
			Message: "Automation version is required",
		})
	}

	// Validate parameters
	for i, param := range metadata.Parameters {
		if param.Name == "" {
			errors = append(errors, types.ValidationError{
				Field:   fmt.Sprintf("parameters[%d].name", i),
				Message: "Parameter name is required",
			})
		}
		if param.Type == "" {
			errors = append(errors, types.ValidationError{
				Field:   fmt.Sprintf("parameters[%d].type", i),
				Message: "Parameter type is required",
			})
		}
	}

	return errors
}