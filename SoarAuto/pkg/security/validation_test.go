package security

import (
	"strings"
	"testing"

	"SoarAuto/pkg/testutil"
)

func TestSecurityValidator_ValidateInput(t *testing.T) {
	validator := NewSecurityValidator()
	
	tests := []struct {
		name        string
		input       string
		fieldName   string
		expectError bool
		errorType   string
	}{
		{
			name:        "valid input",
			input:       "hello world",
			fieldName:   "test_field",
			expectError: false,
		},
		{
			name:        "empty input",
			input:       "",
			fieldName:   "test_field",
			expectError: false,
		},
		{
			name:        "sql injection attempt",
			input:       "'; DROP TABLE users; --",
			fieldName:   "test_field",
			expectError: true,
			errorType:   "sql_injection",
		},
		{
			name:        "xss attempt",
			input:       "<script>alert('xss')</script>",
			fieldName:   "test_field",
			expectError: true,
			errorType:   "xss",
		},
		{
			name:        "path traversal attempt",
			input:       "../../../etc/passwd",
			fieldName:   "test_field",
			expectError: true,
			errorType:   "path_traversal",
		},
		{
			name:        "command injection attempt",
			input:       "test; rm -rf /",
			fieldName:   "test_field",
			expectError: true,
			errorType:   "command_injection",
		},
		{
			name:        "null byte injection",
			input:       "test\x00null",
			fieldName:   "test_field",
			expectError: true,
		},
		{
			name:        "input too long",
			input:       strings.Repeat("a", 20000),
			fieldName:   "test_field",
			expectError: true,
		},
		{
			name:        "blocked pattern",
			input:       "javascript:alert(1)",
			fieldName:   "test_field",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateInput(tt.input, tt.fieldName)
			
			if tt.expectError {
				testutil.AssertError(t, err, "")
			} else {
				testutil.AssertNoError(t, err)
			}
		})
	}
}

func TestSecurityValidator_ValidateFilename(t *testing.T) {
	validator := NewSecurityValidator()
	
	tests := []struct {
		name        string
		filename    string
		expectError bool
	}{
		{
			name:        "valid filename",
			filename:    "document.txt",
			expectError: false,
		},
		{
			name:        "valid filename with extension",
			filename:    "script.py",
			expectError: false,
		},
		{
			name:        "empty filename",
			filename:    "",
			expectError: true,
		},
		{
			name:        "path traversal in filename",
			filename:    "../../../etc/passwd",
			expectError: true,
		},
		{
			name:        "invalid character in filename",
			filename:    "file<name>.txt",
			expectError: true,
		},
		{
			name:        "filename too long",
			filename:    strings.Repeat("a", 300) + ".txt",
			expectError: true,
		},
		{
			name:        "disallowed extension",
			filename:    "malware.exe",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFilename(tt.filename)
			
			if tt.expectError {
				testutil.AssertError(t, err, "")
			} else {
				testutil.AssertNoError(t, err)
			}
		})
	}
}

func TestSecurityValidator_ValidateEmail(t *testing.T) {
	validator := NewSecurityValidator()
	
	tests := []struct {
		name        string
		email       string
		expectError bool
	}{
		{
			name:        "valid email",
			email:       "user@example.com",
			expectError: false,
		},
		{
			name:        "valid email with subdomain",
			email:       "user@mail.example.com",
			expectError: false,
		},
		{
			name:        "empty email",
			email:       "",
			expectError: true,
		},
		{
			name:        "invalid email format",
			email:       "invalid-email",
			expectError: true,
		},
		{
			name:        "email with xss",
			email:       "user@example.com<script>alert(1)</script>",
			expectError: true,
		},
		{
			name:        "email too long",
			email:       strings.Repeat("a", 250) + "@example.com",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEmail(tt.email)
			
			if tt.expectError {
				testutil.AssertError(t, err, "")
			} else {
				testutil.AssertNoError(t, err)
			}
		})
	}
}

func TestSecurityValidator_ValidateURL(t *testing.T) {
	validator := NewSecurityValidator()
	
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "valid http url",
			url:         "http://example.com",
			expectError: false,
		},
		{
			name:        "valid https url",
			url:         "https://example.com/path",
			expectError: false,
		},
		{
			name:        "empty url",
			url:         "",
			expectError: true,
		},
		{
			name:        "invalid url format",
			url:         "not-a-url",
			expectError: true,
		},
		{
			name:        "disallowed scheme",
			url:         "ftp://example.com",
			expectError: true,
		},
		{
			name:        "localhost url",
			url:         "http://localhost:8080",
			expectError: true,
		},
		{
			name:        "private ip url",
			url:         "http://192.168.1.1",
			expectError: true,
		},
		{
			name:        "url with xss",
			url:         "http://example.com<script>alert(1)</script>",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateURL(tt.url)
			
			if tt.expectError {
				testutil.AssertError(t, err, "")
			} else {
				testutil.AssertNoError(t, err)
			}
		})
	}
}

func TestSecurityValidator_ValidateIPAddress(t *testing.T) {
	validator := NewSecurityValidator()
	
	tests := []struct {
		name        string
		ip          string
		expectError bool
	}{
		{
			name:        "valid ipv4",
			ip:          "192.168.1.1",
			expectError: false,
		},
		{
			name:        "valid public ipv4",
			ip:          "8.8.8.8",
			expectError: false,
		},
		{
			name:        "empty ip",
			ip:          "",
			expectError: true,
		},
		{
			name:        "invalid ip format",
			ip:          "not-an-ip",
			expectError: true,
		},
		{
			name:        "localhost ip",
			ip:          "127.0.0.1",
			expectError: true,
		},
		{
			name:        "ip with xss",
			ip:          "192.168.1.1<script>alert(1)</script>",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateIPAddress(tt.ip)
			
			if tt.expectError {
				testutil.AssertError(t, err, "")
			} else {
				testutil.AssertNoError(t, err)
			}
		})
	}
}

func TestSecurityValidator_SanitizeInput(t *testing.T) {
	validator := NewSecurityValidator()
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal input",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "input with null bytes",
			input:    "hello\x00world",
			expected: "helloworld",
		},
		{
			name:     "input with control characters",
			input:    "hello\x01\x02world",
			expected: "helloworld",
		},
		{
			name:     "input with allowed whitespace",
			input:    "hello\t\n\rworld",
			expected: "hello\t\n\rworld",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSecurityValidator_ValidateJSONStructure(t *testing.T) {
	validator := NewSecurityValidator()
	
	tests := []struct {
		name        string
		data        interface{}
		maxDepth    int
		expectError bool
	}{
		{
			name: "valid json object",
			data: map[string]interface{}{
				"key": "value",
			},
			maxDepth:    5,
			expectError: false,
		},
		{
			name: "valid json array",
			data: []interface{}{
				"item1", "item2",
			},
			maxDepth:    5,
			expectError: false,
		},
		{
			name: "json with dangerous key",
			data: map[string]interface{}{
				"__proto__": "dangerous",
			},
			maxDepth:    5,
			expectError: true,
		},
		{
			name: "deeply nested json",
			data: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": map[string]interface{}{
							"level4": map[string]interface{}{
								"level5": map[string]interface{}{
									"level6": "too deep",
								},
							},
						},
					},
				},
			},
			maxDepth:    5,
			expectError: true,
		},
		{
			name: "json with xss in value",
			data: map[string]interface{}{
				"content": "<script>alert('xss')</script>",
			},
			maxDepth:    5,
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateJSONStructure(tt.data, tt.maxDepth, 0)
			
			if tt.expectError {
				testutil.AssertError(t, err, "")
			} else {
				testutil.AssertNoError(t, err)
			}
		})
	}
}

func TestSecurityValidator_ValidateAPIKey(t *testing.T) {
	validator := NewSecurityValidator()
	
	tests := []struct {
		name        string
		apiKey      string
		expectError bool
	}{
		{
			name:        "valid api key",
			apiKey:      "sk-1234567890abcdef",
			expectError: false,
		},
		{
			name:        "valid long api key",
			apiKey:      "sk-1234567890abcdef1234567890abcdef",
			expectError: false,
		},
		{
			name:        "empty api key",
			apiKey:      "",
			expectError: true,
		},
		{
			name:        "api key too short",
			apiKey:      "short",
			expectError: true,
		},
		{
			name:        "api key too long",
			apiKey:      strings.Repeat("a", 200),
			expectError: true,
		},
		{
			name:        "api key with invalid characters",
			apiKey:      "sk-1234567890abcdef!@#$",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateAPIKey(tt.apiKey)
			
			if tt.expectError {
				testutil.AssertError(t, err, "")
			} else {
				testutil.AssertNoError(t, err)
			}
		})
	}
}