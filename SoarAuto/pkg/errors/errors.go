package errors

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	// Configuration errors
	ErrCodeConfigLoad     ErrorCode = "CONFIG_LOAD_ERROR"
	ErrCodeConfigValidate ErrorCode = "CONFIG_VALIDATE_ERROR"
	ErrCodeConfigParse    ErrorCode = "CONFIG_PARSE_ERROR"

	// Authentication errors
	ErrCodeAuthInvalid    ErrorCode = "AUTH_INVALID_KEY"
	ErrCodeAuthMissing    ErrorCode = "AUTH_MISSING_KEY"
	ErrCodeAuthExpired    ErrorCode = "AUTH_EXPIRED_KEY"
	ErrCodeAuthPermission ErrorCode = "AUTH_PERMISSION_DENIED"

	// Validation errors
	ErrCodeValidationFailed ErrorCode = "VALIDATION_FAILED"
	ErrCodeValidationSize   ErrorCode = "VALIDATION_SIZE_EXCEEDED"
	ErrCodeValidationFormat ErrorCode = "VALIDATION_FORMAT_INVALID"

	// Database errors
	ErrCodeDatabaseConnect    ErrorCode = "DATABASE_CONNECTION_ERROR"
	ErrCodeDatabaseOperation  ErrorCode = "DATABASE_OPERATION_ERROR"
	ErrCodeDatabaseTimeout    ErrorCode = "DATABASE_TIMEOUT"
	ErrCodeDatabaseNotFound   ErrorCode = "DATABASE_NOT_FOUND"

	// Playbook execution errors
	ErrCodePlaybookExecution ErrorCode = "PLAYBOOK_EXECUTION_ERROR"
	ErrCodePlaybookNotFound  ErrorCode = "PLAYBOOK_NOT_FOUND"
	ErrCodePlaybookTimeout   ErrorCode = "PLAYBOOK_TIMEOUT"
	ErrCodePlaybookSyntax    ErrorCode = "PLAYBOOK_SYNTAX_ERROR"

	// Integration errors
	ErrCodeIntegrationConfig ErrorCode = "INTEGRATION_CONFIG_ERROR"
	ErrCodeIntegrationAPI    ErrorCode = "INTEGRATION_API_ERROR"
	ErrCodeIntegrationAuth   ErrorCode = "INTEGRATION_AUTH_ERROR"

	// System errors
	ErrCodeSystemResource ErrorCode = "SYSTEM_RESOURCE_ERROR"
	ErrCodeSystemTimeout  ErrorCode = "SYSTEM_TIMEOUT"
	ErrCodeSystemInternal ErrorCode = "SYSTEM_INTERNAL_ERROR"

	// Network errors
	ErrCodeNetworkConnection ErrorCode = "NETWORK_CONNECTION_ERROR"
	ErrCodeNetworkTimeout    ErrorCode = "NETWORK_TIMEOUT"
	ErrCodeNetworkDNS        ErrorCode = "NETWORK_DNS_ERROR"

	// File system errors
	ErrCodeFileNotFound   ErrorCode = "FILE_NOT_FOUND"
	ErrCodeFilePermission ErrorCode = "FILE_PERMISSION_ERROR"
	ErrCodeFileCorrupted  ErrorCode = "FILE_CORRUPTED"

	// Job management errors
	ErrCodeJobNotFound ErrorCode = "JOB_NOT_FOUND"
	ErrCodeJobTimeout  ErrorCode = "JOB_TIMEOUT"
	ErrCodeJobFailed   ErrorCode = "JOB_EXECUTION_FAILED"

	// TLS/Security errors
	ErrCodeTLSCertificate ErrorCode = "TLS_CERTIFICATE_ERROR"
	ErrCodeTLSHandshake   ErrorCode = "TLS_HANDSHAKE_ERROR"
	ErrCodeTLSValidation  ErrorCode = "TLS_VALIDATION_ERROR"
)

// Severity levels for errors
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// SecAutoError represents a standardized error in the SecAuto system
type SecAutoError struct {
	Code        ErrorCode              `json:"code"`
	Message     string                 `json:"message"`
	Cause       error                  `json:"-"` // Original error (not serialized)
	CauseString string                 `json:"cause,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Severity    Severity               `json:"severity"`
	Component   string                 `json:"component"`
	Operation   string                 `json:"operation,omitempty"`
	StackTrace  []string               `json:"stack_trace,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	ClientID    string                 `json:"client_id,omitempty"`
	Retryable   bool                   `json:"retryable"`
}

// Error implements the error interface
func (e *SecAutoError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for error unwrapping
func (e *SecAutoError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for errors.Is
func (e *SecAutoError) Is(target error) bool {
	if secAutoErr, ok := target.(*SecAutoError); ok {
		return e.Code == secAutoErr.Code
	}
	return false
}

// MarshalJSON implements custom JSON marshaling
func (e *SecAutoError) MarshalJSON() ([]byte, error) {
	type Alias SecAutoError
	aux := &struct {
		*Alias
		CauseString string `json:"cause,omitempty"`
	}{
		Alias: (*Alias)(e),
	}
	
	if e.Cause != nil {
		aux.CauseString = e.Cause.Error()
	}
	
	return json.Marshal(aux)
}

// NewError creates a new SecAutoError
func NewError(code ErrorCode, message string) *SecAutoError {
	return &SecAutoError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC(),
		Severity:  SeverityMedium,
		Retryable: false,
	}
}

// WrapError wraps an existing error with SecAuto error context
func WrapError(code ErrorCode, message string, cause error) *SecAutoError {
	err := NewError(code, message)
	err.Cause = cause
	if cause != nil {
		err.CauseString = cause.Error()
	}
	return err
}

// ErrorBuilder provides a fluent interface for building errors
type ErrorBuilder struct {
	error *SecAutoError
}

// NewErrorBuilder creates a new error builder
func NewErrorBuilder(code ErrorCode, message string) *ErrorBuilder {
	return &ErrorBuilder{
		error: NewError(code, message),
	}
}

// WrapErrorBuilder creates a new error builder wrapping an existing error
func WrapErrorBuilder(code ErrorCode, message string, cause error) *ErrorBuilder {
	return &ErrorBuilder{
		error: WrapError(code, message, cause),
	}
}

// WithSeverity sets the error severity
func (eb *ErrorBuilder) WithSeverity(severity Severity) *ErrorBuilder {
	eb.error.Severity = severity
	return eb
}

// WithComponent sets the component that generated the error
func (eb *ErrorBuilder) WithComponent(component string) *ErrorBuilder {
	eb.error.Component = component
	return eb
}

// WithOperation sets the operation that failed
func (eb *ErrorBuilder) WithOperation(operation string) *ErrorBuilder {
	eb.error.Operation = operation
	return eb
}

// WithContext adds context information to the error
func (eb *ErrorBuilder) WithContext(key string, value interface{}) *ErrorBuilder {
	if eb.error.Context == nil {
		eb.error.Context = make(map[string]interface{})
	}
	eb.error.Context[key] = value
	return eb
}

// WithContextMap adds multiple context values
func (eb *ErrorBuilder) WithContextMap(context map[string]interface{}) *ErrorBuilder {
	if eb.error.Context == nil {
		eb.error.Context = make(map[string]interface{})
	}
	for k, v := range context {
		eb.error.Context[k] = v
	}
	return eb
}

// WithRequestID sets the request ID
func (eb *ErrorBuilder) WithRequestID(requestID string) *ErrorBuilder {
	eb.error.RequestID = requestID
	return eb
}

// WithUserID sets the user ID
func (eb *ErrorBuilder) WithUserID(userID string) *ErrorBuilder {
	eb.error.UserID = userID
	return eb
}

// WithClientID sets the client ID
func (eb *ErrorBuilder) WithClientID(clientID string) *ErrorBuilder {
	eb.error.ClientID = clientID
	return eb
}

// WithRetryable marks the error as retryable
func (eb *ErrorBuilder) WithRetryable(retryable bool) *ErrorBuilder {
	eb.error.Retryable = retryable
	return eb
}

// WithStackTrace captures the current stack trace
func (eb *ErrorBuilder) WithStackTrace() *ErrorBuilder {
	eb.error.StackTrace = captureStackTrace()
	return eb
}

// Build returns the constructed error
func (eb *ErrorBuilder) Build() *SecAutoError {
	return eb.error
}

// captureStackTrace captures the current stack trace
func captureStackTrace() []string {
	var stackTrace []string
	
	// Skip the first few frames (this function, the caller, etc.)
	for i := 2; i < 10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		
		stackTrace = append(stackTrace, fmt.Sprintf("%s:%d %s", file, line, fn.Name()))
	}
	
	return stackTrace
}

// HTTPStatusCode returns the appropriate HTTP status code for the error
func (e *SecAutoError) HTTPStatusCode() int {
	switch e.Code {
	case ErrCodeAuthInvalid, ErrCodeAuthMissing, ErrCodeAuthExpired:
		return 401
	case ErrCodeAuthPermission:
		return 403
	case ErrCodePlaybookNotFound, ErrCodeJobNotFound, ErrCodeFileNotFound, ErrCodeDatabaseNotFound:
		return 404
	case ErrCodeValidationFailed, ErrCodeValidationSize, ErrCodeValidationFormat, ErrCodeConfigValidate:
		return 400
	case ErrCodePlaybookTimeout, ErrCodeJobTimeout, ErrCodeDatabaseTimeout, ErrCodeNetworkTimeout, ErrCodeSystemTimeout:
		return 408
	case ErrCodeSystemInternal, ErrCodeSystemResource:
		return 500
	case ErrCodeDatabaseConnect, ErrCodeNetworkConnection:
		return 503
	default:
		return 500
	}
}

// IsRetryable returns whether the error is retryable
func (e *SecAutoError) IsRetryable() bool {
	if e.Retryable {
		return true
	}
	
	// Some errors are inherently retryable
	switch e.Code {
	case ErrCodeDatabaseTimeout, ErrCodeNetworkTimeout, ErrCodeSystemTimeout:
		return true
	case ErrCodeDatabaseConnect, ErrCodeNetworkConnection:
		return true
	case ErrCodeSystemResource:
		return true
	default:
		return false
	}
}

// ToAPIResponse converts the error to an API response format
func (e *SecAutoError) ToAPIResponse() map[string]interface{} {
	response := map[string]interface{}{
		"success":   false,
		"error":     e.Code,
		"message":   e.Message,
		"timestamp": e.Timestamp.Format(time.RFC3339),
	}
	
	if e.RequestID != "" {
		response["request_id"] = e.RequestID
	}
	
	if e.Context != nil && len(e.Context) > 0 {
		response["context"] = e.Context
	}
	
	if e.Retryable || e.IsRetryable() {
		response["retryable"] = true
	}
	
	return response
}

// Common error creation functions for frequently used errors

// ConfigError creates a configuration-related error
func ConfigError(message string, cause error) *SecAutoError {
	return WrapErrorBuilder(ErrCodeConfigLoad, message, cause).
		WithComponent("config").
		WithSeverity(SeverityHigh).
		Build()
}

// ValidationError creates a validation error
func ValidationError(message string, context map[string]interface{}) *SecAutoError {
	return NewErrorBuilder(ErrCodeValidationFailed, message).
		WithComponent("validator").
		WithSeverity(SeverityMedium).
		WithContextMap(context).
		Build()
}

// AuthError creates an authentication error
func AuthError(code ErrorCode, message string) *SecAutoError {
	return NewErrorBuilder(code, message).
		WithComponent("auth").
		WithSeverity(SeverityHigh).
		Build()
}

// DatabaseError creates a database-related error
func DatabaseError(code ErrorCode, message string, cause error) *SecAutoError {
	return WrapErrorBuilder(code, message, cause).
		WithComponent("database").
		WithSeverity(SeverityHigh).
		WithRetryable(true).
		Build()
}

// PlaybookError creates a playbook execution error
func PlaybookError(code ErrorCode, message string, cause error, context map[string]interface{}) *SecAutoError {
	return WrapErrorBuilder(code, message, cause).
		WithComponent("playbook").
		WithSeverity(SeverityMedium).
		WithContextMap(context).
		Build()
}

// SystemError creates a system-level error
func SystemError(code ErrorCode, message string, cause error) *SecAutoError {
	return WrapErrorBuilder(code, message, cause).
		WithComponent("system").
		WithSeverity(SeverityCritical).
		WithStackTrace().
		Build()
}

// NetworkError creates a network-related error
func NetworkError(code ErrorCode, message string, cause error) *SecAutoError {
	return WrapErrorBuilder(code, message, cause).
		WithComponent("network").
		WithSeverity(SeverityMedium).
		WithRetryable(true).
		Build()
}

// TLSError creates a TLS-related error
func TLSError(code ErrorCode, message string, cause error) *SecAutoError {
	return WrapErrorBuilder(code, message, cause).
		WithComponent("tls").
		WithSeverity(SeverityHigh).
		Build()
}

// JobError creates a job management error
func JobError(code ErrorCode, message string, cause error, jobID string) *SecAutoError {
	context := map[string]interface{}{}
	if jobID != "" {
		context["job_id"] = jobID
	}
	
	return WrapErrorBuilder(code, message, cause).
		WithComponent("job_manager").
		WithSeverity(SeverityMedium).
		WithContextMap(context).
		Build()
}

// IntegrationError creates an integration-related error
func IntegrationError(code ErrorCode, message string, cause error, integrationName string) *SecAutoError {
	context := map[string]interface{}{}
	if integrationName != "" {
		context["integration"] = integrationName
	}
	
	return WrapErrorBuilder(code, message, cause).
		WithComponent("integration").
		WithSeverity(SeverityMedium).
		WithContextMap(context).
		Build()
}