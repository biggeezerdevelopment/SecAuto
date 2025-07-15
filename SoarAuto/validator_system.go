package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// ValidationResult represents validation results
type ValidationResult struct {
	Valid   bool              `json:"valid"`
	Errors  []ValidationError `json:"errors,omitempty"`
	Message string            `json:"message,omitempty"`
}

// ValidationResponse represents a validation response
type ValidationResponse struct {
	Success   bool              `json:"success"`
	Valid     bool              `json:"valid"`
	Errors    []ValidationError `json:"errors,omitempty"`
	Message   string            `json:"message,omitempty"`
	Timestamp string            `json:"timestamp"`
}

// Validator provides input validation and sanitization
type Validator struct {
	scriptNameRegex *regexp.Regexp
	pathRegex       *regexp.Regexp
	urlRegex        *regexp.Regexp
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		scriptNameRegex: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
		pathRegex:       regexp.MustCompile(`^[a-zA-Z0-9/._-]+$`),
		urlRegex:        regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`),
	}
}

// ValidatePlaybookRequest validates a playbook execution request
func (v *Validator) ValidatePlaybookRequest(req *PlaybookRequest) ValidationResult {
	var errors []ValidationError

	// Validate playbook name if provided
	if req.PlaybookName != "" {
		if !v.pathRegex.MatchString(req.PlaybookName) {
			errors = append(errors, ValidationError{
				Field:   "playbook_name",
				Message: "Invalid playbook name format",
				Value:   req.PlaybookName,
			})
		}
	}

	// Validate playbook structure if provided
	if req.Playbook != nil {
		if err := v.validatePlaybookStructure(req.Playbook); err != nil {
			errors = append(errors, ValidationError{
				Field:   "playbook",
				Message: err.Error(),
			})
		}
	}

	// Validate context if provided
	if req.Context != nil {
		if err := v.validateContext(req.Context); err != nil {
			errors = append(errors, ValidationError{
				Field:   "context",
				Message: err.Error(),
			})
		}
	}

	// Ensure either playbook or playbook_name is provided
	if req.Playbook == nil && req.PlaybookName == "" {
		errors = append(errors, ValidationError{
			Field:   "request",
			Message: "Either playbook or playbook_name must be provided",
		})
	}

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ValidateWebhookConfig validates webhook configuration
func (v *Validator) ValidateWebhookConfig(config *WebhookConfig) ValidationResult {
	var errors []ValidationError

	// Validate URL
	if config.URL == "" {
		errors = append(errors, ValidationError{
			Field:   "url",
			Message: "Webhook URL is required",
		})
	} else if !v.urlRegex.MatchString(config.URL) {
		errors = append(errors, ValidationError{
			Field:   "url",
			Message: "Invalid webhook URL format",
			Value:   config.URL,
		})
	}

	// Validate events
	if len(config.Events) == 0 {
		errors = append(errors, ValidationError{
			Field:   "events",
			Message: "At least one event type must be specified",
		})
	} else {
		validEvents := map[string]bool{
			"job_started":   true,
			"job_completed": true,
			"job_failed":    true,
			"job_cancelled": true,
		}
		for _, event := range config.Events {
			if !validEvents[event] {
				errors = append(errors, ValidationError{
					Field:   "events",
					Message: "Invalid event type",
					Value:   event,
				})
			}
		}
	}

	// Validate timeout
	if config.Timeout < 0 || config.Timeout > 300 {
		errors = append(errors, ValidationError{
			Field:   "timeout_seconds",
			Message: "Timeout must be between 0 and 300 seconds",
		})
	}

	// Validate retry count
	if config.RetryCount < 0 || config.RetryCount > 10 {
		errors = append(errors, ValidationError{
			Field:   "retry_count",
			Message: "Retry count must be between 0 and 10",
		})
	}

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ValidateJobID validates a job ID
func (v *Validator) ValidateJobID(jobID string) ValidationResult {
	if jobID == "" {
		return ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "job_id",
				Message: "Job ID is required",
			}},
		}
	}

	// UUID format validation
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(jobID) {
		return ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "job_id",
				Message: "Invalid job ID format",
				Value:   jobID,
			}},
		}
	}

	return ValidationResult{Valid: true}
}

// validatePlaybookStructure validates playbook structure
func (v *Validator) validatePlaybookStructure(playbook []interface{}) error {
	if len(playbook) == 0 {
		return fmt.Errorf("playbook cannot be empty")
	}

	for i, rule := range playbook {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			return fmt.Errorf("rule %d must be an object", i+1)
		}

		// Validate script names
		if script, exists := ruleMap["run"]; exists {
			scriptStr, ok := script.(string)
			if !ok {
				return fmt.Errorf("script name in rule %d must be a string", i+1)
			}
			if !v.scriptNameRegex.MatchString(scriptStr) {
				return fmt.Errorf("invalid script name in rule %d: %s", i+1, scriptStr)
			}
		}

		// Validate conditions
		if condition, exists := ruleMap["if"]; exists {
			if err := v.validateCondition(condition); err != nil {
				return fmt.Errorf("invalid condition in rule %d: %v", i+1, err)
			}
		}
	}

	return nil
}

// validateCondition validates a condition structure
func (v *Validator) validateCondition(condition interface{}) error {
	condMap, ok := condition.(map[string]interface{})
	if !ok {
		return fmt.Errorf("condition must be an object")
	}

	// Validate comparison operators
	if op, exists := condMap["op"]; exists {
		opStr, ok := op.(string)
		if !ok {
			return fmt.Errorf("operator must be a string")
		}
		validOps := map[string]bool{"eq": true, "ne": true, "gt": true, "lt": true, "gte": true, "lte": true}
		if !validOps[opStr] {
			return fmt.Errorf("invalid operator: %s", opStr)
		}
	}

	return nil
}

// validateContext validates context data
func (v *Validator) validateContext(context map[string]interface{}) error {
	// Limit context size to prevent memory issues
	if len(context) > 100 {
		return fmt.Errorf("context too large (max 100 keys)")
	}

	// Validate context values (basic type checking)
	for key, value := range context {
		if len(key) > 50 {
			return fmt.Errorf("context key too long: %s", key)
		}

		// Check for potentially dangerous types
		switch v := value.(type) {
		case string:
			if len(v) > 10000 {
				return fmt.Errorf("context value too large for key: %s", key)
			}
		case map[string]interface{}:
			if len(v) > 50 {
				return fmt.Errorf("nested context too large for key: %s", key)
			}
		}
	}

	return nil
}

// SanitizePath sanitizes file paths
func (v *Validator) SanitizePath(path string) string {
	// Remove any directory traversal attempts
	cleanPath := filepath.Clean(path)
	cleanPath = strings.ReplaceAll(cleanPath, "..", "")
	cleanPath = strings.ReplaceAll(cleanPath, "\\", "/")

	// For playbook paths, just return the clean name without adding prefix
	// The engine will handle the full path construction
	return cleanPath
}

// SanitizeScriptName sanitizes script names
func (v *Validator) SanitizeScriptName(name string) string {
	// Remove any potentially dangerous characters
	clean := v.scriptNameRegex.FindString(name)
	if clean == "" {
		return "default"
	}
	return clean
}

// IsValidFilename validates a filename for security
func (v *Validator) IsValidFilename(filename string) bool {
	// Check for path traversal attempts
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return false
	}

	// Check for dangerous characters
	dangerousChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			return false
		}
	}

	// Check length
	if len(filename) > 255 {
		return false
	}

	return true
}

// SanitizeFilename sanitizes a filename for safe storage
func (v *Validator) SanitizeFilename(filename string) string {
	// Remove path traversal attempts
	sanitized := strings.ReplaceAll(filename, "..", "")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")

	// Replace dangerous characters
	dangerousChars := map[string]string{
		"<":  "_",
		">":  "_",
		":":  "_",
		"\"": "_",
		"|":  "_",
		"?":  "_",
		"*":  "_",
		" ":  "_",
	}

	for old, new := range dangerousChars {
		sanitized = strings.ReplaceAll(sanitized, old, new)
	}

	// Remove multiple underscores
	regex := regexp.MustCompile(`_+`)
	sanitized = regex.ReplaceAllString(sanitized, "_")

	// Remove leading/trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "automation"
	}

	return sanitized
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// validationMiddleware adds validation to handlers
func validationMiddleware(validator *Validator) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Add validation headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			next(w, r)
		}
	}
}
