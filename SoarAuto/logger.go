package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

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
	level      LogLevel
	dest       string // "stdout", "file", "both"
	fileWriter io.Writer
	rotation   *RotationConfig
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
	Attempt    int                    `json:"attempt,omitempty"`
	RetryCount int                    `json:"retry_count,omitempty"`
	Stats      map[string]interface{} `json:"stats,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(level LogLevel, dest string, filePath string, rotation *RotationConfig) *StructuredLogger {
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
			// Fallback to simple file writer
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				fileWriter = f
			} else {
				log.Printf("Failed to open log file %s: %v", filePath, err)
			}
		}
	}
	return &StructuredLogger{level: level, dest: dest, fileWriter: fileWriter, rotation: rotation}
}

// shouldLog checks if the message should be logged based on level
func (l *StructuredLogger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug:   0,
		LogLevelInfo:    1,
		LogLevelWarning: 2,
		LogLevelError:   3,
	}
	return levels[level] >= levels[l.level]
}

// log writes a structured log entry
func (l *StructuredLogger) log(entry LogEntry) {
	if !l.shouldLog(LogLevel(entry.Level)) {
		return
	}

	globalLogMutex.Lock()
	defer globalLogMutex.Unlock()

	entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("ERROR: Failed to marshal log entry: %v", err)
		return
	}
	if l.dest == "stdout" {
		fmt.Println(string(jsonData))
	} else if l.dest == "file" && l.fileWriter != nil {
		l.fileWriter.Write(append(jsonData, '\n'))
	} else if l.dest == "both" {
		fmt.Println(string(jsonData))
		if l.fileWriter != nil {
			l.fileWriter.Write(append(jsonData, '\n'))
		}
	}
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(message string, fields map[string]interface{}) {
	entry := LogEntry{
		Level:   string(LogLevelDebug),
		Message: message,
	}
	for k, v := range fields {
		switch k {
		case "component":
			entry.Component = v.(string)
		case "job_id":
			entry.JobID = v.(string)
		case "request_id":
			entry.RequestID = v.(string)
		case "script":
			entry.Script = v.(string)
		case "playbook":
			entry.Playbook = v.(string)
		case "rule_index":
			entry.RuleIndex = v.(int)
		case "condition":
			entry.Condition = v.(string)
		case "variable":
			entry.Variable = v.(string)
		case "value":
			entry.Value = v
		}
	}
	l.log(entry)
}

// Info logs an info message
func (l *StructuredLogger) Info(message string, fields map[string]interface{}) {
	entry := LogEntry{
		Level:   string(LogLevelInfo),
		Message: message,
	}
	for k, v := range fields {
		switch k {
		case "component":
			entry.Component = v.(string)
		case "job_id":
			entry.JobID = v.(string)
		case "request_id":
			entry.RequestID = v.(string)
		case "remote_addr":
			entry.RemoteAddr = v.(string)
		case "path":
			entry.Path = v.(string)
		case "method":
			entry.Method = v.(string)
		case "status_code":
			entry.StatusCode = v.(int)
		case "duration_ms":
			entry.Duration = v.(float64)
		case "script":
			entry.Script = v.(string)
		case "playbook":
			entry.Playbook = v.(string)
		case "webhook_url":
			entry.WebhookURL = v.(string)
		case "attempt":
			entry.Attempt = v.(int)
		case "stats":
			entry.Stats = v.(map[string]interface{})
		}
	}
	l.log(entry)
}

// Warning logs a warning message
func (l *StructuredLogger) Warning(message string, fields map[string]interface{}) {
	entry := LogEntry{
		Level:   string(LogLevelWarning),
		Message: message,
	}
	for k, v := range fields {
		switch k {
		case "component":
			entry.Component = v.(string)
		case "job_id":
			entry.JobID = v.(string)
		case "webhook_url":
			entry.WebhookURL = v.(string)
		case "attempt":
			entry.Attempt = v.(int)
		case "retry_count":
			entry.RetryCount = v.(int)
		}
	}
	l.log(entry)
}

// Error logs an error message
func (l *StructuredLogger) Error(message string, fields map[string]interface{}) {
	entry := LogEntry{
		Level:   string(LogLevelError),
		Message: message,
	}
	for k, v := range fields {
		switch k {
		case "component":
			entry.Component = v.(string)
		case "job_id":
			entry.JobID = v.(string)
		case "request_id":
			entry.RequestID = v.(string)
		case "remote_addr":
			entry.RemoteAddr = v.(string)
		case "path":
			entry.Path = v.(string)
		case "error":
			entry.Error = v.(string)
		case "script":
			entry.Script = v.(string)
		case "playbook":
			entry.Playbook = v.(string)
		case "webhook_url":
			entry.WebhookURL = v.(string)
		case "attempt":
			entry.Attempt = v.(int)
		}
	}
	l.log(entry)
}
