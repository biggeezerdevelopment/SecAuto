package validator

import (
	"fmt"
	"strings"
	"testing"

	"SoarAuto/pkg/testutil"
	"SoarAuto/pkg/types"
)

func TestNewValidator(t *testing.T) {
	validator := NewValidator()
	if validator == nil {
		t.Fatal("Expected validator but got nil")
	}
}

func TestValidatePlaybookRequest(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		request     *types.PlaybookRequest
		expectValid bool
		expectError string
	}{
		{
			name: "valid playbook request",
			request: &types.PlaybookRequest{
				Playbook: []interface{}{
					map[string]interface{}{
						"run": "test_script",
					},
				},
				Context: map[string]interface{}{
					"test": "value",
				},
			},
			expectValid: true,
		},
		{
			name: "valid playbook name request",
			request: &types.PlaybookRequest{
				PlaybookName: "test-playbook",
				Context: map[string]interface{}{
					"test": "value",
				},
			},
			expectValid: true,
		},
		{
			name: "empty request",
			request: &types.PlaybookRequest{},
			expectValid: false,
			expectError: "playbook or playbook_name required",
		},
		{
			name: "both playbook and playbook_name",
			request: &types.PlaybookRequest{
				Playbook: []interface{}{
					map[string]interface{}{"run": "test"},
				},
				PlaybookName: "test-playbook",
			},
			expectValid: false,
			expectError: "cannot specify both",
		},
		{
			name: "invalid playbook name",
			request: &types.PlaybookRequest{
				PlaybookName: "../../../etc/passwd",
			},
			expectValid: false,
			expectError: "invalid playbook name",
		},
		{
			name: "context too large",
			request: &types.PlaybookRequest{
				PlaybookName: "test",
				Context: map[string]interface{}{
					"large_data": strings.Repeat("x", 2*1024*1024), // 2MB
				},
			},
			expectValid: false,
			expectError: "context too large",
		},
		{
			name: "playbook too large",
			request: &types.PlaybookRequest{
				Playbook: createLargePlaybook(2 * 1024 * 1024), // 2MB
			},
			expectValid: false,
			expectError: "playbook too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidatePlaybookRequest(tt.request)

			if tt.expectValid {
				if !result.Valid {
					t.Errorf("Expected valid request, got errors: %v", result.Errors)
				}
			} else {
				if result.Valid {
					t.Error("Expected invalid request but got valid")
				}
				if tt.expectError != "" {
					found := false
					for _, err := range result.Errors {
						if strings.Contains(err.Message, tt.expectError) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing %q, got errors: %v", tt.expectError, result.Errors)
					}
				}
			}
		})
	}
}

func TestValidatePlaybookName(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		playbookName string
		expectValid bool
	}{
		{"valid name", "test-playbook", true},
		{"valid with underscore", "test_playbook", true},
		{"valid with numbers", "playbook123", true},
		{"valid with extension", "playbook.json", true},
		{"empty name", "", false},
		{"path traversal", "../test", false},
		{"absolute path", "/etc/passwd", false},
		{"windows path", "C:\\test", false},
		{"null bytes", "test\x00", false},
		{"control characters", "test\n", false},
		{"too long", strings.Repeat("a", 256), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validator.validatePlaybookName(tt.playbookName)
			if valid != tt.expectValid {
				t.Errorf("Expected %v for %q, got %v", tt.expectValid, tt.playbookName, valid)
			}
		})
	}
}

func TestValidateContext(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		context     map[string]interface{}
		expectValid bool
		expectError string
	}{
		{
			name:        "nil context",
			context:     nil,
			expectValid: true,
		},
		{
			name: "valid context",
			context: map[string]interface{}{
				"string": "value",
				"number": 42,
				"bool":   true,
				"array":  []interface{}{1, 2, 3},
				"object": map[string]interface{}{"nested": "value"},
			},
			expectValid: true,
		},
		{
			name: "context with dangerous keys",
			context: map[string]interface{}{
				"__proto__": "dangerous",
				"constructor": "dangerous",
			},
			expectValid: false,
			expectError: "dangerous key",
		},
		{
			name: "deeply nested context",
			context: createDeeplyNestedContext(20),
			expectValid: false,
			expectError: "nesting too deep",
		},
		{
			name: "context with too many keys",
			context: createContextWithManyKeys(1000),
			expectValid: false,
			expectError: "too many keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.validateContext(tt.context)
			valid := len(errors) == 0

			if valid != tt.expectValid {
				t.Errorf("Expected %v, got %v (errors: %v)", tt.expectValid, valid, errors)
			}

			if !tt.expectValid && tt.expectError != "" {
				found := false
				for _, err := range errors {
					if strings.Contains(err.Message, tt.expectError) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing %q, got errors: %v", tt.expectError, errors)
				}
			}
		})
	}
}

func TestValidatePlaybook(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		playbook    interface{}
		expectValid bool
		expectError string
	}{
		{
			name: "valid array playbook",
			playbook: []interface{}{
				map[string]interface{}{"run": "test_script"},
				map[string]interface{}{"run": "another_script"},
			},
			expectValid: true,
		},
		{
			name: "valid single rule",
			playbook: map[string]interface{}{
				"run": "test_script",
			},
			expectValid: true,
		},
		{
			name: "valid conditional rule",
			playbook: map[string]interface{}{
				"if": map[string]interface{}{
					"conditions": []interface{}{
						[]interface{}{"==", map[string]interface{}{"var": "test"}, "value"},
					},
					"true":  map[string]interface{}{"run": "script1"},
					"false": map[string]interface{}{"run": "script2"},
				},
			},
			expectValid: true,
		},
		{
			name:        "nil playbook",
			playbook:    nil,
			expectValid: false,
			expectError: "playbook cannot be nil",
		},
		{
			name:        "empty array",
			playbook:    []interface{}{},
			expectValid: false,
			expectError: "playbook cannot be empty",
		},
		{
			name: "invalid rule structure",
			playbook: []interface{}{
				"invalid rule",
			},
			expectValid: false,
			expectError: "invalid rule",
		},
		{
			name: "rule without action",
			playbook: []interface{}{
				map[string]interface{}{
					"invalid": "no run or if",
				},
			},
			expectValid: false,
			expectError: "must have 'run' or 'if'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.validatePlaybook(tt.playbook)
			valid := len(errors) == 0

			if valid != tt.expectValid {
				t.Errorf("Expected %v, got %v (errors: %v)", tt.expectValid, valid, errors)
			}

			if !tt.expectValid && tt.expectError != "" {
				found := false
				for _, err := range errors {
					if strings.Contains(err.Message, tt.expectError) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing %q, got errors: %v", tt.expectError, errors)
				}
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal string", "hello world", "hello world"},
		{"with html", "<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
		{"with sql", "'; DROP TABLE users; --", "&#39;; DROP TABLE users; --"},
		{"with null bytes", "test\x00null", "testnull"},
		{"with control chars", "test\n\r\t", "test   "},
		{"unicode", "café", "café"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.sanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCalculateJSONSize(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name     string
		data     interface{}
		expected int
	}{
		{
			name:     "simple string",
			data:     "hello",
			expected: 7, // "hello" with quotes
		},
		{
			name:     "simple object",
			data:     map[string]interface{}{"key": "value"},
			expected: 15, // {"key":"value"}
		},
		{
			name:     "array",
			data:     []interface{}{1, 2, 3},
			expected: 7, // [1,2,3]
		},
		{
			name:     "nil",
			data:     nil,
			expected: 4, // null
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := validator.calculateJSONSize(tt.data)
			if size != tt.expected {
				t.Errorf("Expected size %d, got %d", tt.expected, size)
			}
		})
	}
}

// Helper functions for tests

func createLargePlaybook(size int) []interface{} {
	largeString := strings.Repeat("x", size/2)
	return []interface{}{
		map[string]interface{}{
			"run":  "test",
			"data": largeString,
		},
	}
}

func createDeeplyNestedContext(depth int) map[string]interface{} {
	result := map[string]interface{}{}
	current := result
	
	for i := 0; i < depth; i++ {
		next := map[string]interface{}{}
		current["nested"] = next
		current = next
	}
	
	current["value"] = "deep"
	return result
}

func createContextWithManyKeys(count int) map[string]interface{} {
	result := map[string]interface{}{}
	for i := 0; i < count; i++ {
		result[fmt.Sprintf("key%d", i)] = "value"
	}
	return result
}

// Test the validator with configuration
func TestValidatorWithConfig(t *testing.T) {
	// Test that validator respects configuration limits
	config := &ValidatorConfig{
		MaxContextSize:  1024,
		MaxPlaybookSize: 1024,
		MaxNestingDepth: 5,
		MaxKeys:         10,
	}
	
	validator := NewValidatorWithConfig(config)
	
	// Test context size limit
	largeContext := map[string]interface{}{
		"data": strings.Repeat("x", 2048),
	}
	
	result := validator.ValidatePlaybookRequest(&types.PlaybookRequest{
		PlaybookName: "test",
		Context:      largeContext,
	})
	
	if result.Valid {
		t.Error("Expected validation to fail for large context")
	}
}

// Mock validator config for testing
type ValidatorConfig struct {
	MaxContextSize  int
	MaxPlaybookSize int
	MaxNestingDepth int
	MaxKeys         int
}

func NewValidatorWithConfig(config *ValidatorConfig) *Validator {
	return &Validator{
		maxContextSize:  config.MaxContextSize,
		maxPlaybookSize: config.MaxPlaybookSize,
		maxNestingDepth: config.MaxNestingDepth,
		maxKeys:         config.MaxKeys,
	}
}