package validator

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"

	"SoarAuto/pkg/errors"
	"SoarAuto/pkg/types"
)

// Validator provides input validation and sanitization
type Validator struct {
	scriptNameRegex     *regexp.Regexp
	pathRegex           *regexp.Regexp
	urlRegex            *regexp.Regexp
	uuidRegex           *regexp.Regexp
	underscoreRegex     *regexp.Regexp
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		scriptNameRegex: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
		pathRegex:       regexp.MustCompile(`^[a-zA-Z0-9/._-]+$`),
		urlRegex:        regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`),
		uuidRegex:       regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
		underscoreRegex: regexp.MustCompile(`_+`),
	}
}

// ValidatePlaybookRequest validates a playbook execution request
func (v *Validator) ValidatePlaybookRequest(req *types.PlaybookRequest) types.ValidationResult {
	var errors []types.ValidationError

	// Validate playbook name if provided
	if req.PlaybookName != "" {
		if !v.pathRegex.MatchString(req.PlaybookName) {
			errors = append(errors, types.ValidationError{
				Field:   "playbook_name",
				Message: "Invalid playbook name format",
				Value:   req.PlaybookName,
			})
		}
	}

	// Validate playbook structure if provided
	if req.Playbook != nil {
		if err := v.validatePlaybookStructure(req.Playbook); err != nil {
			errors = append(errors, types.ValidationError{
				Field:   "playbook",
				Message: err.Error(),
			})
		}
	}

	// Validate context if provided
	if req.Context != nil {
		if err := v.validateContext(req.Context); err != nil {
			errors = append(errors, types.ValidationError{
				Field:   "context",
				Message: err.Error(),
			})
		}
	}

	// Ensure either playbook or playbook_name is provided
	if req.Playbook == nil && req.PlaybookName == "" {
		errors = append(errors, types.ValidationError{
			Field:   "request",
			Message: "Either playbook or playbook_name must be provided",
		})
	}

	return types.ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ValidateScriptName validates a script name
func (v *Validator) ValidateScriptName(scriptName string) types.ValidationResult {
	if scriptName == "" {
		return types.ValidationResult{
			Valid: false,
			Errors: []types.ValidationError{{
				Field:   "script_name",
				Message: "Script name is required",
			}},
		}
	}

	if !v.scriptNameRegex.MatchString(scriptName) {
		return types.ValidationResult{
			Valid: false,
			Errors: []types.ValidationError{{
				Field:   "script_name",
				Message: "Invalid script name format",
				Value:   scriptName,
			}},
		}
	}

	return types.ValidationResult{Valid: true}
}

// ValidateURL validates a URL format
func (v *Validator) ValidateURL(url string) types.ValidationResult {
	if url == "" {
		return types.ValidationResult{
			Valid: false,
			Errors: []types.ValidationError{{
				Field:   "url",
				Message: "URL is required",
			}},
		}
	}

	if !v.urlRegex.MatchString(url) {
		return types.ValidationResult{
			Valid: false,
			Errors: []types.ValidationError{{
				Field:   "url",
				Message: "Invalid URL format",
				Value:   url,
			}},
		}
	}

	return types.ValidationResult{Valid: true}
}

// ValidateJobID validates a job ID format
func (v *Validator) ValidateJobID(jobID string) types.ValidationResult {
	if jobID == "" {
		return types.ValidationResult{
			Valid: false,
			Errors: []types.ValidationError{{
				Field:   "job_id",
				Message: "Job ID is required",
			}},
		}
	}

	// UUID format validation using pre-compiled regex
	if !v.uuidRegex.MatchString(jobID) {
		return types.ValidationResult{
			Valid: false,
			Errors: []types.ValidationError{{
				Field:   "job_id",
				Message: "Invalid job ID format",
				Value:   jobID,
			}},
		}
	}

	return types.ValidationResult{Valid: true}
}

// validatePlaybookStructure validates playbook structure
func (v *Validator) validatePlaybookStructure(playbook interface{}) error {
	playbookSlice, ok := playbook.([]interface{})
	if !ok {
		return fmt.Errorf("playbook must be an array")
	}

	if len(playbookSlice) == 0 {
		return fmt.Errorf("playbook cannot be empty")
	}

	for i, rule := range playbookSlice {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			return fmt.Errorf("rule %d must be an object", i)
		}

		// Basic validation - ensure at least one operation exists
		hasOperation := false
		validOperations := []string{"run", "if", "var", "play", "plugin"}
		for _, op := range validOperations {
			if _, exists := ruleMap[op]; exists {
				hasOperation = true
				break
			}
		}

		if !hasOperation {
			return fmt.Errorf("rule %d must contain at least one operation", i)
		}
	}

	return nil
}

// validateContext validates context structure
func (v *Validator) validateContext(context map[string]interface{}) error {
	// Basic validation - ensure it's not too large
	if len(context) > 1000 { // Configurable limit
		return fmt.Errorf("context too large")
	}

	// Additional context-specific validation could go here
	return nil
}

// ValidateWebhookConfig validates webhook configuration
func (v *Validator) ValidateWebhookConfig(config *types.WebhookConfig) types.ValidationResult {
	var errors []types.ValidationError

	// Validate URL
	if config.URL == "" {
		errors = append(errors, types.ValidationError{
			Field:   "url",
			Message: "Webhook URL is required",
		})
	} else if !v.urlRegex.MatchString(config.URL) {
		errors = append(errors, types.ValidationError{
			Field:   "url",
			Message: "Invalid webhook URL format",
			Value:   config.URL,
		})
	}

	// Validate events
	if len(config.Events) == 0 {
		errors = append(errors, types.ValidationError{
			Field:   "events",
			Message: "At least one event must be specified",
		})
	}

	validEvents := []string{
		"job_started", "job_completed", "job_failed", "job_cancelled",
		"schedule_created", "schedule_updated", "schedule_deleted",
		"playbook_executed", "automation_uploaded", "plugin_executed",
	}

	for _, event := range config.Events {
		isValid := false
		for _, validEvent := range validEvents {
			if event == validEvent {
				isValid = true
				break
			}
		}
		if !isValid {
			errors = append(errors, types.ValidationError{
				Field:   "events",
				Message: fmt.Sprintf("Invalid event: %s", event),
				Value:   event,
			})
		}
	}

	return types.ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// IsValidFilename checks if a filename is valid and safe
func (v *Validator) IsValidFilename(filename string) bool {
	// Reject empty filenames
	if filename == "" {
		return false
	}

	// Reject filenames with dangerous characters
	dangerousChars := []string{"../", "..\\", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			return false
		}
	}

	// Reject system files
	systemFiles := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	upperFilename := strings.ToUpper(filename)
	for _, sysFile := range systemFiles {
		if upperFilename == sysFile || strings.HasPrefix(upperFilename, sysFile+".") {
			return false
		}
	}

	// Check length
	if len(filename) > 255 {
		return false
	}

	return true
}

// SanitizeFilename sanitizes a filename for safe use
func (v *Validator) SanitizeFilename(filename string) string {
	// Replace dangerous characters with underscores
	dangerousChars := map[string]string{
		"/":  "_",
		"\\": "_",
		":":  "_",
		"*":  "_",
		"?":  "_",
		"\"": "_",
		"<":  "_",
		">":  "_",
		"|":  "_",
		" ":  "_",
	}

	sanitized := filename
	for old, new := range dangerousChars {
		sanitized = strings.ReplaceAll(sanitized, old, new)
	}

	// Remove multiple underscores using pre-compiled regex
	sanitized = v.underscoreRegex.ReplaceAllString(sanitized, "_")

	// Remove leading/trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "automation"
	}

	// Ensure it's not too long
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}

	return sanitized
}

// ValidateHTTPRequest performs basic HTTP request validation
func (v *Validator) ValidateHTTPRequest(r *http.Request, maxBodySize int64) types.ValidationResult {
	var errors []types.ValidationError

	// Check content length
	if r.ContentLength > maxBodySize {
		errors = append(errors, types.ValidationError{
			Field:   "content_length",
			Message: fmt.Sprintf("Request body too large (max: %d bytes)", maxBodySize),
			Value:   fmt.Sprintf("%d", r.ContentLength),
		})
	}

	// Check content type for POST/PUT requests
	if r.Method == "POST" || r.Method == "PUT" {
		contentType := r.Header.Get("Content-Type")
		if contentType != "" && !strings.Contains(contentType, "application/json") && !strings.Contains(contentType, "multipart/form-data") {
			errors = append(errors, types.ValidationError{
				Field:   "content_type",
				Message: "Invalid content type",
				Value:   contentType,
			})
		}
	}

	return types.ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// containsDangerousContent checks for potentially dangerous content in uploaded files
func (v *Validator) containsDangerousContent(content []byte) bool {
	contentStr := strings.ToLower(string(content))
	
	// Check for dangerous patterns
	dangerousPatterns := []string{
		"import os",
		"import subprocess",
		"import sys",
		"os.system",
		"subprocess.call",
		"subprocess.run",
		"subprocess.popen",
		"eval(",
		"exec(",
		"__import__",
		"open(",
		"file(",
		"input(",
		"raw_input(",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(contentStr, pattern) {
			return true
		}
	}

	return false
}

// ContainsDangerousContent exposes the dangerous content check
func (v *Validator) ContainsDangerousContent(content []byte) bool {
	return v.containsDangerousContent(content)
}
/
/ Validator configuration constants
const (
	DefaultMaxContextSize  = 1024 * 1024 // 1MB
	DefaultMaxPlaybookSize = 1024 * 1024 // 1MB
	DefaultMaxNestingDepth = 10
	DefaultMaxKeys         = 100
)

// Enhanced Validator with configuration
type ValidatorConfig struct {
	MaxContextSize  int
	MaxPlaybookSize int
	MaxNestingDepth int
	MaxKeys         int
}

// NewValidatorWithConfig creates a validator with custom configuration
func NewValidatorWithConfig(config *ValidatorConfig) *Validator {
	v := NewValidator()
	v.maxContextSize = config.MaxContextSize
	v.maxPlaybookSize = config.MaxPlaybookSize
	v.maxNestingDepth = config.MaxNestingDepth
	v.maxKeys = config.MaxKeys
	return v
}

// Add configuration fields to Validator
type EnhancedValidator struct {
	*Validator
	maxContextSize  int
	maxPlaybookSize int
	maxNestingDepth int
	maxKeys         int
}

// validatePlaybookName validates a playbook name for security
func (v *Validator) validatePlaybookName(name string) bool {
	if name == "" || len(name) > 255 {
		return false
	}
	
	// Check for path traversal
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	
	// Check for null bytes and control characters
	for _, char := range name {
		if char < 32 || char == 127 {
			return false
		}
	}
	
	return true
}

// validateContext validates context data
func (v *Validator) validateContext(context map[string]interface{}) []types.ValidationError {
	var errors []types.ValidationError
	
	if context == nil {
		return errors
	}
	
	// Check for dangerous keys
	dangerousKeys := []string{"__proto__", "constructor", "prototype"}
	for _, key := range dangerousKeys {
		if _, exists := context[key]; exists {
			errors = append(errors, types.ValidationError{
				Field:   "context",
				Message: fmt.Sprintf("Dangerous key not allowed: %s", key),
				Value:   key,
			})
		}
	}
	
	// Check nesting depth
	if v.checkNestingDepth(context, 0) > 15 {
		errors = append(errors, types.ValidationError{
			Field:   "context",
			Message: "Context nesting too deep",
		})
	}
	
	// Check number of keys
	if v.countKeys(context) > 500 {
		errors = append(errors, types.ValidationError{
			Field:   "context",
			Message: "Context has too many keys",
		})
	}
	
	return errors
}

// validatePlaybook validates playbook structure
func (v *Validator) validatePlaybook(playbook interface{}) []types.ValidationError {
	var errors []types.ValidationError
	
	if playbook == nil {
		errors = append(errors, types.ValidationError{
			Field:   "playbook",
			Message: "Playbook cannot be nil",
		})
		return errors
	}
	
	// Check if it's an array
	if playbookArray, ok := playbook.([]interface{}); ok {
		if len(playbookArray) == 0 {
			errors = append(errors, types.ValidationError{
				Field:   "playbook",
				Message: "Playbook cannot be empty",
			})
			return errors
		}
		
		// Validate each rule
		for i, rule := range playbookArray {
			if ruleErrors := v.validateRule(rule, fmt.Sprintf("playbook[%d]", i)); len(ruleErrors) > 0 {
				errors = append(errors, ruleErrors...)
			}
		}
	} else {
		// Single rule
		if ruleErrors := v.validateRule(playbook, "playbook"); len(ruleErrors) > 0 {
			errors = append(errors, ruleErrors...)
		}
	}
	
	return errors
}

// validateRule validates a single playbook rule
func (v *Validator) validateRule(rule interface{}, fieldPath string) []types.ValidationError {
	var errors []types.ValidationError
	
	ruleMap, ok := rule.(map[string]interface{})
	if !ok {
		errors = append(errors, types.ValidationError{
			Field:   fieldPath,
			Message: "Rule must be an object",
			Value:   rule,
		})
		return errors
	}
	
	// Check if rule has required fields
	hasRun := false
	hasIf := false
	
	for key := range ruleMap {
		if key == "run" {
			hasRun = true
		}
		if key == "if" {
			hasIf = true
		}
	}
	
	if !hasRun && !hasIf {
		errors = append(errors, types.ValidationError{
			Field:   fieldPath,
			Message: "Rule must have 'run' or 'if' field",
		})
	}
	
	return errors
}

// sanitizeInput sanitizes input strings
func (v *Validator) sanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Replace control characters with spaces
	input = regexp.MustCompile(`[\x00-\x1f\x7f]`).ReplaceAllString(input, " ")
	
	// HTML escape
	input = html.EscapeString(input)
	
	return input
}

// calculateJSONSize calculates the JSON size of data
func (v *Validator) calculateJSONSize(data interface{}) int {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0
	}
	return len(jsonData)
}

// checkNestingDepth checks the nesting depth of a map
func (v *Validator) checkNestingDepth(data interface{}, currentDepth int) int {
	if currentDepth > 20 { // Prevent infinite recursion
		return currentDepth
	}
	
	maxDepth := currentDepth
	
	switch v := data.(type) {
	case map[string]interface{}:
		for _, value := range v {
			depth := v.checkNestingDepth(value, currentDepth+1)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	case []interface{}:
		for _, value := range v {
			depth := v.checkNestingDepth(value, currentDepth+1)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}
	
	return maxDepth
}

// countKeys counts the total number of keys in nested maps
func (v *Validator) countKeys(data interface{}) int {
	count := 0
	
	switch v := data.(type) {
	case map[string]interface{}:
		count += len(v)
		for _, value := range v {
			count += v.countKeys(value)
		}
	case []interface{}:
		for _, value := range v {
			count += v.countKeys(value)
		}
	}
	
	return count
}