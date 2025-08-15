package playbooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"SoarAuto/pkg/types"
)

// PlaybookManager manages playbook files and metadata
type PlaybookManager struct {
	playbooksPath string
	mutex         sync.RWMutex
}

// PlaybookInfo represents information about a playbook file
type PlaybookInfo struct {
	Name        string    `json:"name"`
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	RuleCount   int       `json:"rule_count"`
	ModifiedAt  string    `json:"modified_at"`
	IsValid     bool      `json:"is_valid"`
	Description string    `json:"description,omitempty"`
	Author      string    `json:"author,omitempty"`
	Version     string    `json:"version,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
}

// NewPlaybookManager creates a new playbook manager
func NewPlaybookManager(playbooksPath string) *PlaybookManager {
	return &PlaybookManager{
		playbooksPath: playbooksPath,
	}
}

// ListPlaybooks returns information about all playbook files
func (pm *PlaybookManager) ListPlaybooks() ([]PlaybookInfo, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var playbooks []PlaybookInfo

	// Ensure playbooks directory exists
	if err := os.MkdirAll(pm.playbooksPath, 0755); err != nil {
		return nil, err
	}

	// Read all JSON files in the playbooks directory
	files, err := filepath.Glob(filepath.Join(pm.playbooksPath, "*.json"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		info, err := pm.getPlaybookInfo(file)
		if err != nil {
			continue // Skip files that can't be processed
		}
		playbooks = append(playbooks, *info)
	}

	return playbooks, nil
}

// getPlaybookInfo analyzes a playbook file and returns information
func (pm *PlaybookManager) getPlaybookInfo(filePath string) (*PlaybookInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	name := filepath.Base(filePath)
	name = name[:len(name)-5] // Remove .json extension

	// Read and parse the playbook file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return &PlaybookInfo{
			Name:       name,
			Filename:   filepath.Base(filePath),
			Size:       stat.Size(),
			ModifiedAt: stat.ModTime().Format(time.RFC3339),
			IsValid:    false,
		}, nil
	}

	// Try to parse as JSON array (rules)
	var rules []interface{}
	ruleCount := 0
	isValid := true

	if err := json.Unmarshal(data, &rules); err != nil {
		// Try to parse as single rule object
		var singleRule interface{}
		if err := json.Unmarshal(data, &singleRule); err != nil {
			isValid = false
		} else {
			ruleCount = 1
		}
	} else {
		ruleCount = len(rules)
	}

	// Extract metadata if present
	var metadata map[string]interface{}
	description := ""
	author := ""
	version := ""
	var tags []string

	// Try to find metadata in the playbook
	if len(rules) > 0 {
		if firstRule, ok := rules[0].(map[string]interface{}); ok {
			if meta, exists := firstRule["metadata"]; exists {
				if metaMap, ok := meta.(map[string]interface{}); ok {
					metadata = metaMap
					if desc, ok := metadata["description"].(string); ok {
						description = desc
					}
					if auth, ok := metadata["author"].(string); ok {
						author = auth
					}
					if ver, ok := metadata["version"].(string); ok {
						version = ver
					}
					if tagList, ok := metadata["tags"].([]interface{}); ok {
						for _, tag := range tagList {
							if tagStr, ok := tag.(string); ok {
								tags = append(tags, tagStr)
							}
						}
					}
				}
			}
		}
	}

	return &PlaybookInfo{
		Name:        name,
		Filename:    filepath.Base(filePath),
		Size:        stat.Size(),
		RuleCount:   ruleCount,
		ModifiedAt:  stat.ModTime().Format(time.RFC3339),
		IsValid:     isValid,
		Description: description,
		Author:      author,
		Version:     version,
		Tags:        tags,
	}, nil
}

// SavePlaybook saves a playbook to disk
func (pm *PlaybookManager) SavePlaybook(name string, content []byte) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(pm.playbooksPath, 0755); err != nil {
		return err
	}

	// Validate JSON format
	var playbook interface{}
	if err := json.Unmarshal(content, &playbook); err != nil {
		return fmt.Errorf("invalid JSON format: %v", err)
	}

	// Ensure .json extension
	if !strings.HasSuffix(name, ".json") {
		name += ".json"
	}

	// Save file
	filename := filepath.Join(pm.playbooksPath, name)
	return os.WriteFile(filename, content, 0644)
}

// DeletePlaybook deletes a playbook file
func (pm *PlaybookManager) DeletePlaybook(name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Ensure .json extension
	if !strings.HasSuffix(name, ".json") {
		name += ".json"
	}

	filename := filepath.Join(pm.playbooksPath, name)
	
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("playbook not found: %s", name)
	}

	return os.Remove(filename)
}

// GetPlaybook retrieves a playbook by name
func (pm *PlaybookManager) GetPlaybook(name string) ([]byte, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Ensure .json extension
	if !strings.HasSuffix(name, ".json") {
		name += ".json"
	}

	filename := filepath.Join(pm.playbooksPath, name)
	return os.ReadFile(filename)
}

// PlaybookExists checks if a playbook exists
func (pm *PlaybookManager) PlaybookExists(name string) bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Ensure .json extension
	if !strings.HasSuffix(name, ".json") {
		name += ".json"
	}

	filename := filepath.Join(pm.playbooksPath, name)
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// ValidatePlaybook validates a playbook's structure
func (pm *PlaybookManager) ValidatePlaybook(content []byte) []types.ValidationError {
	var errors []types.ValidationError

	// Parse JSON
	var playbook interface{}
	if err := json.Unmarshal(content, &playbook); err != nil {
		errors = append(errors, types.ValidationError{
			Field:   "playbook",
			Message: "Invalid JSON format",
			Value:   string(content),
		})
		return errors
	}

	// Check if it's an array of rules or a single rule
	switch v := playbook.(type) {
	case []interface{}:
		// Array of rules
		if len(v) == 0 {
			errors = append(errors, types.ValidationError{
				Field:   "playbook",
				Message: "Playbook must contain at least one rule",
			})
		}
		// Validate each rule
		for i, rule := range v {
			if ruleMap, ok := rule.(map[string]interface{}); ok {
				if err := pm.validateRule(ruleMap); err != nil {
					errors = append(errors, types.ValidationError{
						Field:   fmt.Sprintf("playbook[%d]", i),
						Message: err.Error(),
						Value:   rule,
					})
				}
			} else {
				errors = append(errors, types.ValidationError{
					Field:   fmt.Sprintf("playbook[%d]", i),
					Message: "Rule must be an object",
					Value:   rule,
				})
			}
		}
	case map[string]interface{}:
		// Single rule
		if err := pm.validateRule(v); err != nil {
			errors = append(errors, types.ValidationError{
				Field:   "playbook",
				Message: err.Error(),
				Value:   playbook,
			})
		}
	default:
		errors = append(errors, types.ValidationError{
			Field:   "playbook",
			Message: "Playbook must be an object or array of objects",
			Value:   playbook,
		})
	}

	return errors
}

// validateRule validates a single rule's structure
func (pm *PlaybookManager) validateRule(rule map[string]interface{}) error {
	// Check for at least one operation
	operations := []string{"if", "set", "run", "call", "log", "sleep", "stop", "metadata"}
	hasOperation := false

	for _, op := range operations {
		if _, exists := rule[op]; exists {
			hasOperation = true
			break
		}
	}

	if !hasOperation {
		return fmt.Errorf("rule must contain at least one operation")
	}

	return nil
}

// GetPlaybookDependencies analyzes a playbook for dependencies
func (pm *PlaybookManager) GetPlaybookDependencies(content []byte) ([]string, error) {
	var dependencies []string
	var playbook interface{}

	if err := json.Unmarshal(content, &playbook); err != nil {
		return nil, err
	}

	// Extract dependencies from call operations
	dependencies = pm.extractDependencies(playbook)

	// Remove duplicates
	seen := make(map[string]bool)
	var uniqueDeps []string
	for _, dep := range dependencies {
		if !seen[dep] {
			seen[dep] = true
			uniqueDeps = append(uniqueDeps, dep)
		}
	}

	return uniqueDeps, nil
}

// extractDependencies recursively extracts dependencies from playbook content
func (pm *PlaybookManager) extractDependencies(data interface{}) []string {
	var dependencies []string

	switch v := data.(type) {
	case map[string]interface{}:
		// Check for call operation
		if call, exists := v["call"]; exists {
			if callStr, ok := call.(string); ok {
				dependencies = append(dependencies, callStr)
			}
		}
		// Recursively check all values
		for _, value := range v {
			dependencies = append(dependencies, pm.extractDependencies(value)...)
		}
	case []interface{}:
		// Recursively check all elements
		for _, item := range v {
			dependencies = append(dependencies, pm.extractDependencies(item)...)
		}
	}

	return dependencies
}