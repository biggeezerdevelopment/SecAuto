package validator

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
)

// Validator provides input validation and sanitization
type Validator struct {
	scriptNameRegex     *regexp.Regexp
	pathRegex           *regexp.Regexp
	urlRegex            *regexp.Regexp
	uuidRegex           *regexp.Regexp
	underscoreRegex     *regexp.Regexp
	maxContextSize      int
	maxPlaybookSize     int
	maxNestingDepth     int
	maxKeys             int
}

// Validator configuration constants
const (
	DefaultMaxContextSize  = 1024 * 1024 // 1MB
	DefaultMaxPlaybookSize = 1024 * 1024 // 1MB
	DefaultMaxNestingDepth = 10
	DefaultMaxKeys         = 100
)

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		scriptNameRegex: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
		pathRegex:       regexp.MustCompile(`^[a-zA-Z0-9/._-]+$`),
		urlRegex:        regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`),
		uuidRegex:       regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
		underscoreRegex: regexp.MustCompile(`_+`),
		maxContextSize:  DefaultMaxContextSize,
		maxPlaybookSize: DefaultMaxPlaybookSize,
		maxNestingDepth: DefaultMaxNestingDepth,
		maxKeys:         DefaultMaxKeys,
	}
}

// ValidateScriptName validates script names
func (v *Validator) ValidateScriptName(name string) error {
	if name == "" {
		return fmt.Errorf("script name cannot be empty")
	}
	if !v.scriptNameRegex.MatchString(name) {
		return fmt.Errorf("invalid script name format")
	}
	return nil
}

// ValidatePath validates file paths
func (v *Validator) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if !v.pathRegex.MatchString(path) {
		return fmt.Errorf("invalid path format")
	}
	return nil
}

// ValidateURL validates URLs
func (v *Validator) ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	if !v.urlRegex.MatchString(url) {
		return fmt.Errorf("invalid URL format")
	}
	return nil
}

// ValidateUUID validates UUIDs
func (v *Validator) ValidateUUID(uuid string) error {
	if uuid == "" {
		return fmt.Errorf("UUID cannot be empty")
	}
	if !v.uuidRegex.MatchString(uuid) {
		return fmt.Errorf("invalid UUID format")
	}
	return nil
}

// SanitizeInput sanitizes user input
func (v *Validator) SanitizeInput(input string) string {
	// HTML escape
	sanitized := html.EscapeString(input)
	
	// Remove multiple underscores
	sanitized = v.underscoreRegex.ReplaceAllString(sanitized, "_")
	
	return strings.TrimSpace(sanitized)
}

// ValidatePlaybook validates playbook structure
func (v *Validator) ValidatePlaybook(playbook map[string]interface{}) error {
	if playbook == nil {
		return fmt.Errorf("playbook cannot be nil")
	}
	
	name, ok := playbook["name"].(string)
	if !ok || name == "" {
		return fmt.Errorf("playbook name is required")
	}
	
	if err := v.ValidateScriptName(name); err != nil {
		return fmt.Errorf("invalid playbook name: %w", err)
	}
	
	return nil
}

// ValidateContext validates context data
func (v *Validator) ValidateContext(context map[string]interface{}) error {
	if context == nil {
		return nil
	}
	
	// Check context size
	contextJSON, err := json.Marshal(context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}
	
	if len(contextJSON) > v.maxContextSize {
		return fmt.Errorf("context size exceeds maximum allowed size")
	}
	
	// Check nesting depth and key count
	return v.validateStructure(context, 0)
}

// validateStructure validates the structure of nested data
func (v *Validator) validateStructure(data interface{}, depth int) error {
	if depth > v.maxNestingDepth {
		return fmt.Errorf("nesting depth exceeds maximum allowed depth")
	}
	
	switch d := data.(type) {
	case map[string]interface{}:
		if len(d) > v.maxKeys {
			return fmt.Errorf("number of keys exceeds maximum allowed")
		}
		for _, value := range d {
			if err := v.validateStructure(value, depth+1); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, item := range d {
			if err := v.validateStructure(item, depth+1); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// ValidateHTTPRequest validates HTTP request parameters
func (v *Validator) ValidateHTTPRequest(r *http.Request) error {
	if r == nil {
		return fmt.Errorf("request cannot be nil")
	}
	
	// Validate content length
	if r.ContentLength > int64(v.maxPlaybookSize) {
		return fmt.Errorf("request body too large")
	}
	
	return nil
}

// containsDangerousContent checks for dangerous content patterns
func (v *Validator) containsDangerousContent(content string) bool {
	dangerousPatterns := []string{
		"<script",
		"javascript:",
		"vbscript:",
		"onload=",
		"onerror=",
		"eval(",
		"exec(",
	}
	
	contentLower := strings.ToLower(content)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(contentLower, pattern) {
			return true
		}
	}
	
	return false
}

// ValidateContent validates content for dangerous patterns
func (v *Validator) ValidateContent(content string) error {
	if v.containsDangerousContent(content) {
		return fmt.Errorf("content contains dangerous patterns")
	}
	return nil
}