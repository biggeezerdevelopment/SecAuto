package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"SoarAuto/pkg/config"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel represents logging levels
type LogLevel string

const (
	LogLevelDebug   LogLevel = "DEBUG"
	LogLevelInfo    LogLevel = "INFO"
	LogLevelWarning LogLevel = "WARNING"
	LogLevelError   LogLevel = "ERROR"
)

// StructuredLogger provides structured JSON logging
type StructuredLogger struct {
	level           LogLevel
	dest            string // "stdout", "file", "both"
	fileWriter      io.Writer
	rotation        *config.RotationConfig
	componentLevels map[string]LogLevel
	performance     *config.LoggingPerformanceConfig
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp  string                 `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Component  string                 `json:"component,omitempty"`
	JobID      string                 `json:"job_id,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	RemoteAddr string                 `json:"remote_addr,omitempty"`
	Path       string                 `json:"path,omitempty"`
	Method     string                 `json:"method,omitempty"`
	StatusCode int                    `json:"status_code,omitempty"`
	Duration   float64                `json:"duration_ms,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Script     string                 `json:"script,omitempty"`
	Playbook   string                 `json:"playbook,omitempty"`
	RuleIndex  int                    `json:"rule_index,omitempty"`
	Condition  string                 `json:"condition,omitempty"`
	Variable   string                 `json:"variable,omitempty"`
	Value      interface{}            `json:"value,omitempty"`
	WebhookURL string                 `json:"webhook_url,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
	Attempt    int                    `json:"attempt,omitempty"`
	RetryCount int                    `json:"retry_count,omitempty"`
	Stats      map[string]interface{} `json:"stats,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(level LogLevel, dest string, filePath string, rotation *config.RotationConfig) *StructuredLogger {
	return NewStructuredLoggerWithConfig(level, dest, filePath, rotation, nil, nil)
}

// NewStructuredLoggerWithConfig creates a new structured logger with component-specific configuration
func NewStructuredLoggerWithConfig(level LogLevel, dest string, filePath string, rotation *config.RotationConfig, componentLevels map[string]string, performance *config.LoggingPerformanceConfig) *StructuredLogger {
	var fileWriter io.Writer
	if dest == "file" || dest == "both" {
		// Ensure directory exists
		dir := filepath.Dir(filePath)
		os.MkdirAll(dir, 0755)

		if rotation != nil {
			// Use lumberjack for log rotation
			fileWriter = &lumberjack.Logger{
				Filename:   filePath,
				MaxSize:    rotation.MaxSizeMB, // megabytes
				MaxBackups: rotation.MaxBackups,
				MaxAge:     rotation.MaxAgeDays, // days
				Compress:   rotation.Compress,
			}
		} else {
			// No rotation, just create the file
			file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Printf("Failed to create log file: %v", err)
				fileWriter = nil
			} else {
				fileWriter = file
			}
		}
	}

	// Convert component levels to LogLevel type
	convertedLevels := make(map[string]LogLevel)
	if componentLevels != nil {
		for component, level := range componentLevels {
			convertedLevels[component] = LogLevel(strings.ToUpper(level))
		}
	}

	return &StructuredLogger{
		level:           level,
		dest:            dest,
		fileWriter:      fileWriter,
		rotation:        rotation,
		componentLevels: convertedLevels,
		performance:     performance,
	}
}

// shouldLog determines if a message should be logged based on component-specific levels
func (sl *StructuredLogger) shouldLog(level LogLevel, component string) bool {
	// Get component-specific level if available
	if componentLevel, exists := sl.componentLevels[component]; exists {
		return shouldLogAtLevel(level, componentLevel)
	}

	// Fall back to default level
	return shouldLogAtLevel(level, sl.level)
}

// shouldLogAtLevel compares log levels
func shouldLogAtLevel(messageLevel, configuredLevel LogLevel) bool {
	levelPriority := map[LogLevel]int{
		LogLevelDebug:   0,
		LogLevelInfo:    1,
		LogLevelWarning: 2,
		LogLevelError:   3,
	}

	messagePriority, messageExists := levelPriority[messageLevel]
	configPriority, configExists := levelPriority[configuredLevel]

	// If either level is unknown, allow logging
	if !messageExists || !configExists {
		return true
	}

	return messagePriority >= configPriority
}

// IsLogLevelEnabled checks if a log level is enabled for a component
func (sl *StructuredLogger) IsLogLevelEnabled(level, component string) bool {
	return sl.shouldLog(LogLevel(strings.ToUpper(level)), component)
}

// truncateField truncates a field value if it exceeds the maximum length
func (sl *StructuredLogger) truncateField(value interface{}) interface{} {
	if sl.performance == nil || sl.performance.MaxLogFieldLength <= 0 {
		return value
	}

	switch v := value.(type) {
	case string:
		if len(v) > sl.performance.MaxLogFieldLength {
			return v[:sl.performance.MaxLogFieldLength] + "...[truncated]"
		}
	case []byte:
		if len(v) > sl.performance.MaxLogFieldLength {
			return string(v[:sl.performance.MaxLogFieldLength]) + "...[truncated]"
		}
		return string(v)
	}

	return value
}

// log writes a structured log entry
func (sl *StructuredLogger) log(level LogLevel, message string, fields map[string]interface{}) {
	// Get component from fields or use "system" as default
	component := "system"
	if comp, exists := fields["component"]; exists {
		if compStr, ok := comp.(string); ok {
			component = compStr
		}
	}

	// Check if we should log this message
	if !sl.shouldLog(level, component) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     string(level),
		Message:   message,
		Component: component,
	}

	// Initialize Extra map for additional fields
	entry.Extra = make(map[string]interface{})
	
	// Add additional fields from the fields map
	for key, value := range fields {
		// Skip component as it's already handled
		if key == "component" {
			continue
		}
		
		// Apply field truncation if configured
		truncatedValue := sl.truncateField(value)

		// Skip context logging in production if configured
		if sl.performance != nil && sl.performance.SkipContextLogging && 
			(key == "context" || key == "Context") {
			continue
		}

		switch key {
		case "job_id":
			if str, ok := truncatedValue.(string); ok {
				entry.JobID = str
			}
		case "request_id":
			if str, ok := truncatedValue.(string); ok {
				entry.RequestID = str
			}
		case "user_agent":
			if str, ok := truncatedValue.(string); ok {
				entry.UserAgent = str
			}
		case "remote_addr":
			if str, ok := truncatedValue.(string); ok {
				entry.RemoteAddr = str
			}
		case "path":
			if str, ok := truncatedValue.(string); ok {
				entry.Path = str
			}
		case "method":
			if str, ok := truncatedValue.(string); ok {
				entry.Method = str
			}
		case "status_code":
			if num, ok := truncatedValue.(int); ok {
				entry.StatusCode = num
			}
		case "duration_ms":
			if dur, ok := truncatedValue.(float64); ok {
				entry.Duration = dur
			}
		case "error":
			if str, ok := truncatedValue.(string); ok {
				entry.Error = str
			}
		case "script":
			if str, ok := truncatedValue.(string); ok {
				entry.Script = str
			}
		case "playbook":
			if str, ok := truncatedValue.(string); ok {
				entry.Playbook = str
			}
		case "rule_index":
			if num, ok := truncatedValue.(int); ok {
				entry.RuleIndex = num
			}
		case "condition":
			if str, ok := truncatedValue.(string); ok {
				entry.Condition = str
			}
		case "variable":
			if str, ok := truncatedValue.(string); ok {
				entry.Variable = str
			}
		case "value":
			entry.Value = truncatedValue
		case "webhook_url":
			if str, ok := truncatedValue.(string); ok {
				entry.WebhookURL = str
			}
		case "attempt":
			if num, ok := truncatedValue.(int); ok {
				entry.Attempt = num
			}
		case "retry_count":
			if num, ok := truncatedValue.(int); ok {
				entry.RetryCount = num
			}
		case "stats":
			if stats, ok := truncatedValue.(map[string]interface{}); ok {
				entry.Stats = stats
			}
		case "context":
			if ctx, ok := truncatedValue.(map[string]interface{}); ok {
				entry.Context = ctx
			}
		default:
			// Add any unmatched fields to Extra
			entry.Extra[key] = truncatedValue
		}
	}
	
	// Clean up empty Extra map
	if len(entry.Extra) == 0 {
		entry.Extra = nil
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple logging if JSON marshalling fails
		fmt.Printf("ERROR: Failed to marshal log entry: %v\n", err)
		return
	}

	jsonStr := string(jsonBytes) + "\n"

	// Write to appropriate destinations
	switch sl.dest {
	case "stdout":
		fmt.Print(jsonStr)
	case "file":
		if sl.fileWriter != nil {
			sl.fileWriter.Write([]byte(jsonStr))
		}
	case "both":
		fmt.Print(jsonStr)
		if sl.fileWriter != nil {
			sl.fileWriter.Write([]byte(jsonStr))
		}
	}
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(message string, fields map[string]interface{}) {
	sl.log(LogLevelDebug, message, fields)
}

// Info logs an info message
func (sl *StructuredLogger) Info(message string, fields map[string]interface{}) {
	sl.log(LogLevelInfo, message, fields)
}

// Warning logs a warning message
func (sl *StructuredLogger) Warning(message string, fields map[string]interface{}) {
	sl.log(LogLevelWarning, message, fields)
}

// Error logs an error message
func (sl *StructuredLogger) Error(message string, fields map[string]interface{}) {
	sl.log(LogLevelError, message, fields)
}