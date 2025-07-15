package main

import (
	"fmt"
	"time"
)

// JobStoreInterface defines the interface for job storage implementations
type JobStoreInterface interface {
	// Core job operations
	SaveJob(job *Job) error
	LoadJob(jobID string) (*Job, bool)
	ListJobs(status string, limit int) []*Job
	UpdateJobStatus(jobID, status string) error
	UpdateJobResults(jobID string, results []interface{}, errorMsg string) error
	UpdateJobContext(jobID string, context map[string]interface{}) error
	DeleteJob(jobID string) error

	// Maintenance operations
	CleanupOldJobs(maxAge time.Duration) error
	GetStats() JobStats
	BackupJobs() error
	RecoverJobs(engine *RuleEngine, webhookManager *WebhookManager)
	Close() error

	// Database metrics
	GetDatabaseMetrics() map[string]interface{}

	// Schedule operations (optional - may return errors if not implemented)
	SaveSchedule(schedule *JobSchedule) error
	LoadSchedule(scheduleID string) (*JobSchedule, bool)
	ListSchedules(status string, limit int) []*JobSchedule
	UpdateSchedule(schedule *JobSchedule) error
	DeleteSchedule(scheduleID string) error
	GetSchedulesDueForExecution() []*JobSchedule
}

// NewJobStore creates a job store based on configuration
func NewJobStore(config *Config) (JobStoreInterface, error) {
	// Only support Redis job store
	if config.Database.RedisURL == "" {
		return nil, fmt.Errorf("redis URL is required for job store")
	}
	return NewRedisJobStore(config.Database.RedisURL)
}
