package security

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"SoarAuto/pkg/errors"
	"SoarAuto/pkg/types"
)

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	AuditEventLogin          AuditEventType = "login"
	AuditEventLogout         AuditEventType = "logout"
	AuditEventAPIKeyCreated  AuditEventType = "api_key_created"
	AuditEventAPIKeyDeleted  AuditEventType = "api_key_deleted"
	AuditEventPlaybookExec   AuditEventType = "playbook_execution"
	AuditEventConfigChange   AuditEventType = "config_change"
	AuditEventSecurityAlert  AuditEventType = "security_alert"
	AuditEventRateLimitHit   AuditEventType = "rate_limit_exceeded"
	AuditEventAuthFailure    AuditEventType = "authentication_failure"
	AuditEventAccessDenied   AuditEventType = "access_denied"
	AuditEventDataAccess     AuditEventType = "data_access"
	AuditEventDataModify     AuditEventType = "data_modification"
	AuditEventSystemError    AuditEventType = "system_error"
)

// AuditSeverity represents the severity of an audit event
type AuditSeverity string

const (
	AuditSeverityInfo     AuditSeverity = "info"
	AuditSeverityWarning  AuditSeverity = "warning"
	AuditSeverityError    AuditSeverity = "error"
	AuditSeverityCritical AuditSeverity = "critical"
)

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	EventType   AuditEventType         `json:"event_type"`
	Severity    AuditSeverity          `json:"severity"`
	UserID      string                 `json:"user_id,omitempty"`
	ClientID    string                 `json:"client_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Resource    string                 `json:"resource,omitempty"`
	Action      string                 `json:"action,omitempty"`
	Result      string                 `json:"result"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Duration    int64                  `json:"duration_ms,omitempty"`
	Component   string                 `json:"component,omitempty"`
	Environment string                 `json:"environment,omitempty"`
}

// AuditLogger handles security audit logging
type AuditLogger struct {
	logger     types.Logger
	auditFile  *os.File
	mutex      sync.Mutex
	config     *AuditConfig
	eventQueue chan *AuditEvent
	stopChan   chan struct{}
}

// AuditConfig holds audit logging configuration
type AuditConfig struct {
	Enabled           bool   `json:"enabled"`
	LogFile           string `json:"log_file"`
	MaxFileSize       int64  `json:"max_file_size"`
	MaxBackups        int    `json:"max_backups"`
	BufferSize        int    `json:"buffer_size"`
	FlushInterval     int    `json:"flush_interval_seconds"`
	IncludeStackTrace bool   `json:"include_stack_trace"`
	Environment       string `json:"environment"`
}

// DefaultAuditConfig returns default audit configuration
func DefaultAuditConfig() *AuditConfig {
	return &AuditConfig{
		Enabled:           true,
		LogFile:           "logs/audit.log",
		MaxFileSize:       100 * 1024 * 1024, // 100MB
		MaxBackups:        10,
		BufferSize:        1000,
		FlushInterval:     5,
		IncludeStackTrace: false,
		Environment:       "development",
	}
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger types.Logger, config *AuditConfig) (*AuditLogger, error) {
	if config == nil {
		config = DefaultAuditConfig()
	}
	
	if !config.Enabled {
		return &AuditLogger{
			logger: logger,
			config: config,
		}, nil
	}
	
	// Create audit log directory if it doesn't exist
	logDir := filepath.Dir(config.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, errors.WrapErrorBuilder(
			errors.ErrCodeSystemResource,
			"Failed to create audit log directory",
			err,
		).WithContext("log_dir", logDir).Build()
	}
	
	// Open audit log file
	auditFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, errors.WrapErrorBuilder(
			errors.ErrCodeSystemResource,
			"Failed to open audit log file",
			err,
		).WithContext("log_file", config.LogFile).Build()
	}
	
	al := &AuditLogger{
		logger:     logger,
		auditFile:  auditFile,
		config:     config,
		eventQueue: make(chan *AuditEvent, config.BufferSize),
		stopChan:   make(chan struct{}),
	}
	
	// Start background writer
	go al.backgroundWriter()
	
	return al, nil
}

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(event *AuditEvent) {
	if !al.config.Enabled {
		return
	}
	
	// Set default values
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Environment == "" {
		event.Environment = al.config.Environment
	}
	
	// Try to queue the event (non-blocking)
	select {
	case al.eventQueue <- event:
		// Event queued successfully
	default:
		// Queue is full, log to main logger
		al.logger.Error("Audit event queue full, dropping event", map[string]interface{}{
			"component":  "audit_logger",
			"event_type": event.EventType,
			"event_id":   event.ID,
		})
	}
}

// LogSecurityEvent logs a security-related event
func (al *AuditLogger) LogSecurityEvent(eventType AuditEventType, severity AuditSeverity, message string, r *http.Request, details map[string]interface{}) {
	event := &AuditEvent{
		EventType: eventType,
		Severity:  severity,
		Message:   message,
		Details:   details,
		Result:    "unknown",
	}
	
	// Extract request information if available
	if r != nil {
		event.IPAddress = extractClientIP(r)
		event.UserAgent = r.UserAgent()
		event.RequestID = r.Header.Get("X-Request-ID")
		event.Resource = r.URL.Path
		event.Action = r.Method
	}
	
	al.LogEvent(event)
}

// LogAuthenticationEvent logs authentication-related events
func (al *AuditLogger) LogAuthenticationEvent(success bool, userID, clientID string, r *http.Request, err error) {
	var eventType AuditEventType
	var severity AuditSeverity
	var result string
	var message string
	
	if success {
		eventType = AuditEventLogin
		severity = AuditSeverityInfo
		result = "success"
		message = "Authentication successful"
	} else {
		eventType = AuditEventAuthFailure
		severity = AuditSeverityWarning
		result = "failure"
		message = "Authentication failed"
	}
	
	event := &AuditEvent{
		EventType: eventType,
		Severity:  severity,
		UserID:    userID,
		ClientID:  clientID,
		Result:    result,
		Message:   message,
		Action:    "authenticate",
	}
	
	if r != nil {
		event.IPAddress = extractClientIP(r)
		event.UserAgent = r.UserAgent()
		event.RequestID = r.Header.Get("X-Request-ID")
	}
	
	if err != nil {
		event.Error = err.Error()
		if secAutoErr, ok := err.(*errors.SecAutoError); ok {
			event.Details = map[string]interface{}{
				"error_code": secAutoErr.Code,
				"context":    secAutoErr.Context,
			}
		}
	}
	
	al.LogEvent(event)
}

// LogDataAccessEvent logs data access events
func (al *AuditLogger) LogDataAccessEvent(resource, action, userID, clientID string, r *http.Request, success bool) {
	var severity AuditSeverity
	var result string
	
	if success {
		severity = AuditSeverityInfo
		result = "success"
	} else {
		severity = AuditSeverityWarning
		result = "failure"
	}
	
	event := &AuditEvent{
		EventType: AuditEventDataAccess,
		Severity:  severity,
		UserID:    userID,
		ClientID:  clientID,
		Resource:  resource,
		Action:    action,
		Result:    result,
		Message:   fmt.Sprintf("Data access: %s %s", action, resource),
	}
	
	if r != nil {
		event.IPAddress = extractClientIP(r)
		event.UserAgent = r.UserAgent()
		event.RequestID = r.Header.Get("X-Request-ID")
	}
	
	al.LogEvent(event)
}

// LogPlaybookExecution logs playbook execution events
func (al *AuditLogger) LogPlaybookExecution(playbookName, userID, clientID string, duration time.Duration, success bool, err error) {
	var severity AuditSeverity
	var result string
	var message string
	
	if success {
		severity = AuditSeverityInfo
		result = "success"
		message = fmt.Sprintf("Playbook executed successfully: %s", playbookName)
	} else {
		severity = AuditSeverityError
		result = "failure"
		message = fmt.Sprintf("Playbook execution failed: %s", playbookName)
	}
	
	event := &AuditEvent{
		EventType: AuditEventPlaybookExec,
		Severity:  severity,
		UserID:    userID,
		ClientID:  clientID,
		Resource:  playbookName,
		Action:    "execute",
		Result:    result,
		Message:   message,
		Duration:  duration.Milliseconds(),
	}
	
	if err != nil {
		event.Error = err.Error()
		if secAutoErr, ok := err.(*errors.SecAutoError); ok {
			event.Details = map[string]interface{}{
				"error_code": secAutoErr.Code,
				"context":    secAutoErr.Context,
			}
		}
	}
	
	al.LogEvent(event)
}

// LogConfigurationChange logs configuration change events
func (al *AuditLogger) LogConfigurationChange(configType, userID string, oldValue, newValue interface{}) {
	event := &AuditEvent{
		EventType: AuditEventConfigChange,
		Severity:  AuditSeverityWarning,
		UserID:    userID,
		Resource:  configType,
		Action:    "modify",
		Result:    "success",
		Message:   fmt.Sprintf("Configuration changed: %s", configType),
		Details: map[string]interface{}{
			"old_value": oldValue,
			"new_value": newValue,
		},
	}
	
	al.LogEvent(event)
}

// backgroundWriter handles writing audit events to file
func (al *AuditLogger) backgroundWriter() {
	ticker := time.NewTicker(time.Duration(al.config.FlushInterval) * time.Second)
	defer ticker.Stop()
	
	var events []*AuditEvent
	
	for {
		select {
		case event := <-al.eventQueue:
			events = append(events, event)
			
			// Write immediately if buffer is full
			if len(events) >= al.config.BufferSize/2 {
				al.writeEvents(events)
				events = events[:0]
			}
			
		case <-ticker.C:
			// Periodic flush
			if len(events) > 0 {
				al.writeEvents(events)
				events = events[:0]
			}
			
		case <-al.stopChan:
			// Final flush before shutdown
			if len(events) > 0 {
				al.writeEvents(events)
			}
			return
		}
	}
}

// writeEvents writes events to the audit log file
func (al *AuditLogger) writeEvents(events []*AuditEvent) {
	al.mutex.Lock()
	defer al.mutex.Unlock()
	
	if al.auditFile == nil {
		return
	}
	
	for _, event := range events {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			al.logger.Error("Failed to marshal audit event", map[string]interface{}{
				"component": "audit_logger",
				"event_id":  event.ID,
				"error":     err.Error(),
			})
			continue
		}
		
		// Write event to file
		if _, err := al.auditFile.Write(append(eventJSON, '\n')); err != nil {
			al.logger.Error("Failed to write audit event", map[string]interface{}{
				"component": "audit_logger",
				"event_id":  event.ID,
				"error":     err.Error(),
			})
		}
	}
	
	// Sync to disk
	al.auditFile.Sync()
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	if !al.config.Enabled {
		return nil
	}
	
	// Signal background writer to stop
	close(al.stopChan)
	
	// Close audit file
	al.mutex.Lock()
	defer al.mutex.Unlock()
	
	if al.auditFile != nil {
		return al.auditFile.Close()
	}
	
	return nil
}

// Helper functions

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), os.Getpid())
}

// extractClientIP extracts the client IP from the request
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	if ip := strings.Split(r.RemoteAddr, ":"); len(ip) > 0 {
		return ip[0]
	}
	
	return r.RemoteAddr
}