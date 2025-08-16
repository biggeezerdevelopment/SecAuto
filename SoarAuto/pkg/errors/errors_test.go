package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"SoarAuto/pkg/testutil"
)

func TestNewError(t *testing.T) {
	err := NewError(ErrCodeConfigLoad, "Failed to load configuration")
	
	if err.Code != ErrCodeConfigLoad {
		t.Errorf("Expected code %s, got %s", ErrCodeConfigLoad, err.Code)
	}
	
	if err.Message != "Failed to load configuration" {
		t.Errorf("Expected message 'Failed to load configuration', got %s", err.Message)
	}
	
	if err.Severity != SeverityMedium {
		t.Errorf("Expected default severity %s, got %s", SeverityMedium, err.Severity)
	}
	
	if err.Retryable {
		t.Error("Expected default retryable to be false")
	}
	
	if err.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestWrapError(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	err := WrapError(ErrCodeDatabaseConnect, "Database connection failed", originalErr)
	
	if err.Code != ErrCodeDatabaseConnect {
		t.Errorf("Expected code %s, got %s", ErrCodeDatabaseConnect, err.Code)
	}
	
	if err.Cause != originalErr {
		t.Errorf("Expected cause to be original error")
	}
	
	if err.CauseString != "original error" {
		t.Errorf("Expected cause string 'original error', got %s", err.CauseString)
	}
	
	// Test error unwrapping
	if !errors.Is(err, originalErr) {
		t.Error("Expected wrapped error to match original error")
	}
}

func TestErrorBuilder(t *testing.T) {
	originalErr := fmt.Errorf("database timeout")
	
	err := NewErrorBuilder(ErrCodeDatabaseTimeout, "Query timed out").
		WithSeverity(SeverityHigh).
		WithComponent("database").
		WithOperation("SELECT").
		WithContext("query", "SELECT * FROM users").
		WithContext("timeout", 30).
		WithRequestID("req-123").
		WithUserID("user-456").
		WithClientID("client-789").
		WithRetryable(true).
		Build()
	
	if err.Code != ErrCodeDatabaseTimeout {
		t.Errorf("Expected code %s, got %s", ErrCodeDatabaseTimeout, err.Code)
	}
	
	if err.Severity != SeverityHigh {
		t.Errorf("Expected severity %s, got %s", SeverityHigh, err.Severity)
	}
	
	if err.Component != "database" {
		t.Errorf("Expected component 'database', got %s", err.Component)
	}
	
	if err.Operation != "SELECT" {
		t.Errorf("Expected operation 'SELECT', got %s", err.Operation)
	}
	
	if err.RequestID != "req-123" {
		t.Errorf("Expected request ID 'req-123', got %s", err.RequestID)
	}
	
	if err.UserID != "user-456" {
		t.Errorf("Expected user ID 'user-456', got %s", err.UserID)
	}
	
	if err.ClientID != "client-789" {
		t.Errorf("Expected client ID 'client-789', got %s", err.ClientID)
	}
	
	if !err.Retryable {
		t.Error("Expected retryable to be true")
	}
	
	if err.Context["query"] != "SELECT * FROM users" {
		t.Errorf("Expected query context, got %v", err.Context["query"])
	}
	
	if err.Context["timeout"] != 30 {
		t.Errorf("Expected timeout context 30, got %v", err.Context["timeout"])
	}
}

func TestErrorBuilderWithContextMap(t *testing.T) {
	context := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	
	err := NewErrorBuilder(ErrCodeValidationFailed, "Validation failed").
		WithContextMap(context).
		Build()
	
	for key, expectedValue := range context {
		if actualValue := err.Context[key]; actualValue != expectedValue {
			t.Errorf("Expected context[%s] = %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestErrorBuilderWithStackTrace(t *testing.T) {
	err := NewErrorBuilder(ErrCodeSystemInternal, "Internal error").
		WithStackTrace().
		Build()
	
	if len(err.StackTrace) == 0 {
		t.Error("Expected stack trace to be captured")
	}
	
	// Verify stack trace contains this test function
	found := false
	for _, frame := range err.StackTrace {
		if strings.Contains(frame, "TestErrorBuilderWithStackTrace") {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected stack trace to contain test function name")
	}
}

func TestErrorInterface(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	err := WrapError(ErrCodeDatabaseConnect, "Connection failed", originalErr)
	
	// Test Error() method
	errorString := err.Error()
	expectedString := "[DATABASE_CONNECTION_ERROR] Connection failed: original error"
	if errorString != expectedString {
		t.Errorf("Expected error string %s, got %s", expectedString, errorString)
	}
	
	// Test error without cause
	err2 := NewError(ErrCodeAuthInvalid, "Invalid API key")
	errorString2 := err2.Error()
	expectedString2 := "[AUTH_INVALID_KEY] Invalid API key"
	if errorString2 != expectedString2 {
		t.Errorf("Expected error string %s, got %s", expectedString2, errorString2)
	}
}

func TestErrorUnwrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	err := WrapError(ErrCodeDatabaseConnect, "Connection failed", originalErr)
	
	unwrapped := errors.Unwrap(err)
	if unwrapped != originalErr {
		t.Error("Expected unwrapped error to match original error")
	}
	
	// Test error without cause
	err2 := NewError(ErrCodeAuthInvalid, "Invalid API key")
	unwrapped2 := errors.Unwrap(err2)
	if unwrapped2 != nil {
		t.Error("Expected unwrapped error to be nil for error without cause")
	}
}

func TestErrorIs(t *testing.T) {
	err1 := NewError(ErrCodeDatabaseConnect, "Connection failed")
	err2 := NewError(ErrCodeDatabaseConnect, "Another connection error")
	err3 := NewError(ErrCodeAuthInvalid, "Invalid key")
	
	// Same error code should match
	if !errors.Is(err1, err2) {
		t.Error("Expected errors with same code to match")
	}
	
	// Different error codes should not match
	if errors.Is(err1, err3) {
		t.Error("Expected errors with different codes to not match")
	}
	
	// Test with non-SecAutoError
	stdErr := fmt.Errorf("standard error")
	if errors.Is(err1, stdErr) {
		t.Error("Expected SecAutoError to not match standard error")
	}
}

func TestHTTPStatusCode(t *testing.T) {
	tests := []struct {
		code           ErrorCode
		expectedStatus int
	}{
		{ErrCodeAuthInvalid, 401},
		{ErrCodeAuthMissing, 401},
		{ErrCodeAuthExpired, 401},
		{ErrCodeAuthPermission, 403},
		{ErrCodePlaybookNotFound, 404},
		{ErrCodeJobNotFound, 404},
		{ErrCodeValidationFailed, 400},
		{ErrCodePlaybookTimeout, 408},
		{ErrCodeSystemInternal, 500},
		{ErrCodeDatabaseConnect, 503},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := NewError(tt.code, "Test error")
			status := err.HTTPStatusCode()
			if status != tt.expectedStatus {
				t.Errorf("Expected status %d for code %s, got %d", tt.expectedStatus, tt.code, status)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		code      ErrorCode
		retryable bool
		expected  bool
	}{
		{ErrCodeDatabaseTimeout, false, true},  // Inherently retryable
		{ErrCodeNetworkTimeout, false, true},   // Inherently retryable
		{ErrCodeDatabaseConnect, false, true},  // Inherently retryable
		{ErrCodeAuthInvalid, false, false},     // Not retryable
		{ErrCodeValidationFailed, true, true},  // Explicitly retryable
		{ErrCodeSystemInternal, false, false},  // Not retryable
	}
	
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := NewError(tt.code, "Test error")
			err.Retryable = tt.retryable
			
			if err.IsRetryable() != tt.expected {
				t.Errorf("Expected IsRetryable() = %v for code %s, got %v", tt.expected, tt.code, err.IsRetryable())
			}
		})
	}
}

func TestToAPIResponse(t *testing.T) {
	err := NewErrorBuilder(ErrCodeValidationFailed, "Validation failed").
		WithRequestID("req-123").
		WithContext("field", "email").
		WithContext("value", "invalid-email").
		WithRetryable(false).
		Build()
	
	response := err.ToAPIResponse()
	
	if response["success"] != false {
		t.Error("Expected success to be false")
	}
	
	if response["error"] != ErrCodeValidationFailed {
		t.Errorf("Expected error code %s, got %v", ErrCodeValidationFailed, response["error"])
	}
	
	if response["message"] != "Validation failed" {
		t.Errorf("Expected message 'Validation failed', got %v", response["message"])
	}
	
	if response["request_id"] != "req-123" {
		t.Errorf("Expected request_id 'req-123', got %v", response["request_id"])
	}
	
	context, ok := response["context"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected context to be map[string]interface{}")
	}
	
	if context["field"] != "email" {
		t.Errorf("Expected context field 'email', got %v", context["field"])
	}
	
	if context["value"] != "invalid-email" {
		t.Errorf("Expected context value 'invalid-email', got %v", context["value"])
	}
	
	// Test retryable error
	retryableErr := NewError(ErrCodeDatabaseTimeout, "Timeout")
	retryableResponse := retryableErr.ToAPIResponse()
	
	if retryableResponse["retryable"] != true {
		t.Error("Expected retryable to be true for timeout error")
	}
}

func TestJSONMarshaling(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	err := WrapErrorBuilder(ErrCodeDatabaseConnect, "Connection failed", originalErr).
		WithComponent("database").
		WithSeverity(SeverityHigh).
		WithContext("host", "localhost").
		WithRequestID("req-123").
		Build()
	
	jsonData, jsonErr := json.Marshal(err)
	testutil.AssertNoError(t, jsonErr)
	
	var unmarshaled map[string]interface{}
	jsonErr = json.Unmarshal(jsonData, &unmarshaled)
	testutil.AssertNoError(t, jsonErr)
	
	if unmarshaled["code"] != string(ErrCodeDatabaseConnect) {
		t.Errorf("Expected code %s, got %v", ErrCodeDatabaseConnect, unmarshaled["code"])
	}
	
	if unmarshaled["message"] != "Connection failed" {
		t.Errorf("Expected message 'Connection failed', got %v", unmarshaled["message"])
	}
	
	if unmarshaled["cause"] != "original error" {
		t.Errorf("Expected cause 'original error', got %v", unmarshaled["cause"])
	}
	
	if unmarshaled["component"] != "database" {
		t.Errorf("Expected component 'database', got %v", unmarshaled["component"])
	}
	
	if unmarshaled["severity"] != string(SeverityHigh) {
		t.Errorf("Expected severity %s, got %v", SeverityHigh, unmarshaled["severity"])
	}
}

func TestCommonErrorFunctions(t *testing.T) {
	t.Run("ConfigError", func(t *testing.T) {
		originalErr := fmt.Errorf("file not found")
		err := ConfigError("Failed to load config", originalErr)
		
		if err.Code != ErrCodeConfigLoad {
			t.Errorf("Expected code %s, got %s", ErrCodeConfigLoad, err.Code)
		}
		
		if err.Component != "config" {
			t.Errorf("Expected component 'config', got %s", err.Component)
		}
		
		if err.Severity != SeverityHigh {
			t.Errorf("Expected severity %s, got %s", SeverityHigh, err.Severity)
		}
	})
	
	t.Run("ValidationError", func(t *testing.T) {
		context := map[string]interface{}{
			"field": "email",
			"value": "invalid",
		}
		err := ValidationError("Invalid email format", context)
		
		if err.Code != ErrCodeValidationFailed {
			t.Errorf("Expected code %s, got %s", ErrCodeValidationFailed, err.Code)
		}
		
		if err.Component != "validator" {
			t.Errorf("Expected component 'validator', got %s", err.Component)
		}
		
		if err.Context["field"] != "email" {
			t.Errorf("Expected context field 'email', got %v", err.Context["field"])
		}
	})
	
	t.Run("AuthError", func(t *testing.T) {
		err := AuthError(ErrCodeAuthInvalid, "Invalid API key")
		
		if err.Code != ErrCodeAuthInvalid {
			t.Errorf("Expected code %s, got %s", ErrCodeAuthInvalid, err.Code)
		}
		
		if err.Component != "auth" {
			t.Errorf("Expected component 'auth', got %s", err.Component)
		}
		
		if err.Severity != SeverityHigh {
			t.Errorf("Expected severity %s, got %s", SeverityHigh, err.Severity)
		}
	})
	
	t.Run("DatabaseError", func(t *testing.T) {
		originalErr := fmt.Errorf("connection refused")
		err := DatabaseError(ErrCodeDatabaseConnect, "Cannot connect to Redis", originalErr)
		
		if err.Code != ErrCodeDatabaseConnect {
			t.Errorf("Expected code %s, got %s", ErrCodeDatabaseConnect, err.Code)
		}
		
		if err.Component != "database" {
			t.Errorf("Expected component 'database', got %s", err.Component)
		}
		
		if !err.Retryable {
			t.Error("Expected database error to be retryable")
		}
	})
	
	t.Run("JobError", func(t *testing.T) {
		originalErr := fmt.Errorf("execution failed")
		err := JobError(ErrCodeJobFailed, "Job execution failed", originalErr, "job-123")
		
		if err.Code != ErrCodeJobFailed {
			t.Errorf("Expected code %s, got %s", ErrCodeJobFailed, err.Code)
		}
		
		if err.Component != "job_manager" {
			t.Errorf("Expected component 'job_manager', got %s", err.Component)
		}
		
		if err.Context["job_id"] != "job-123" {
			t.Errorf("Expected job_id 'job-123', got %v", err.Context["job_id"])
		}
	})
	
	t.Run("TLSError", func(t *testing.T) {
		originalErr := fmt.Errorf("certificate expired")
		err := TLSError(ErrCodeTLSCertificate, "Certificate validation failed", originalErr)
		
		if err.Code != ErrCodeTLSCertificate {
			t.Errorf("Expected code %s, got %s", ErrCodeTLSCertificate, err.Code)
		}
		
		if err.Component != "tls" {
			t.Errorf("Expected component 'tls', got %s", err.Component)
		}
		
		if err.Severity != SeverityHigh {
			t.Errorf("Expected severity %s, got %s", SeverityHigh, err.Severity)
		}
	})
}

func TestErrorCodes(t *testing.T) {
	// Test that all error codes are defined and unique
	codes := []ErrorCode{
		ErrCodeConfigLoad, ErrCodeConfigValidate, ErrCodeConfigParse,
		ErrCodeAuthInvalid, ErrCodeAuthMissing, ErrCodeAuthExpired, ErrCodeAuthPermission,
		ErrCodeValidationFailed, ErrCodeValidationSize, ErrCodeValidationFormat,
		ErrCodeDatabaseConnect, ErrCodeDatabaseOperation, ErrCodeDatabaseTimeout, ErrCodeDatabaseNotFound,
		ErrCodePlaybookExecution, ErrCodePlaybookNotFound, ErrCodePlaybookTimeout, ErrCodePlaybookSyntax,
		ErrCodeIntegrationConfig, ErrCodeIntegrationAPI, ErrCodeIntegrationAuth,
		ErrCodeSystemResource, ErrCodeSystemTimeout, ErrCodeSystemInternal,
		ErrCodeNetworkConnection, ErrCodeNetworkTimeout, ErrCodeNetworkDNS,
		ErrCodeFileNotFound, ErrCodeFilePermission, ErrCodeFileCorrupted,
		ErrCodeJobNotFound, ErrCodeJobTimeout, ErrCodeJobFailed,
		ErrCodeTLSCertificate, ErrCodeTLSHandshake, ErrCodeTLSValidation,
	}
	
	// Check that all codes are non-empty
	for _, code := range codes {
		if string(code) == "" {
			t.Errorf("Error code is empty")
		}
	}
	
	// Check for uniqueness
	codeMap := make(map[ErrorCode]bool)
	for _, code := range codes {
		if codeMap[code] {
			t.Errorf("Duplicate error code: %s", code)
		}
		codeMap[code] = true
	}
}

func TestSeverityLevels(t *testing.T) {
	severities := []Severity{SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical}
	
	for _, severity := range severities {
		if string(severity) == "" {
			t.Errorf("Severity level is empty")
		}
	}
}

func TestErrorWithoutCause(t *testing.T) {
	err := NewError(ErrCodeAuthInvalid, "Invalid API key")
	
	// Test JSON marshaling without cause
	jsonData, jsonErr := json.Marshal(err)
	testutil.AssertNoError(t, jsonErr)
	
	var unmarshaled map[string]interface{}
	jsonErr = json.Unmarshal(jsonData, &unmarshaled)
	testutil.AssertNoError(t, jsonErr)
	
	// Cause should not be present in JSON
	if _, exists := unmarshaled["cause"]; exists {
		t.Error("Expected cause to not be present in JSON when nil")
	}
}