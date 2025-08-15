package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Job represents an asynchronous playbook execution job
type Job struct {
	ID          string                 `json:"id"`
	Status      string                 `json:"status"` // "pending", "running", "completed", "failed"
	Playbook    []interface{}          `json:"playbook"`
	Context     map[string]interface{} `json:"context"`
	Results     []interface{}          `json:"results,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// JobManager manages asynchronous job execution
type JobManager struct {
	store          JobStoreInterface
	workerPool     chan struct{}
	webhookManager *WebhookManager
	cleanupTicker  *time.Ticker
	backupTicker   *time.Ticker
}

// NewJobManager creates a new job manager with specified worker pool size
func NewJobManager(workerCount int, webhookManager *WebhookManager, config *Config) (*JobManager, error) {
	store, err := NewJobStore(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create job store: %v", err)
	}

	jm := &JobManager{
		store:          store,
		workerPool:     make(chan struct{}, workerCount),
		webhookManager: webhookManager,
	}

	// Start background tasks
	jm.startBackgroundTasks()

	return jm, nil
}

// startBackgroundTasks starts cleanup and backup tasks
func (jm *JobManager) startBackgroundTasks() {
	// Start cleanup ticker (every 24 hours)
	jm.cleanupTicker = time.NewTicker(24 * time.Hour)
	go func() {
		for range jm.cleanupTicker.C {
			if err := jm.store.CleanupOldJobs(30 * 24 * time.Hour); err != nil {
				logger.Error("Failed to cleanup old jobs", map[string]interface{}{
					"component": "job_manager",
					"error":     err.Error(),
				})
			}
		}
	}()

	// Start backup ticker (every 7 days)
	jm.backupTicker = time.NewTicker(7 * 24 * time.Hour)
	go func() {
		for range jm.backupTicker.C {
			if err := jm.store.BackupJobs(); err != nil {
				logger.Error("Failed to backup jobs", map[string]interface{}{
					"component": "job_manager",
					"error":     err.Error(),
				})
			}
		}
	}()
}

// SubmitJob submits a new job for execution
func (jm *JobManager) SubmitJob(playbook []interface{}, context map[string]interface{}) string {
	jobID := uuid.New().String()

	logger.Info("Submitting job", map[string]interface{}{
		"component":    "job_manager",
		"job_id":       jobID,
		"context":      context,
		"context_type": fmt.Sprintf("%T", context),
		"context_keys": len(context),
	})

	job := &Job{
		ID:        jobID,
		Status:    "pending",
		Playbook:  playbook,
		Context:   context,
		CreatedAt: time.Now(),
	}

	// Save to persistent storage
	if err := jm.store.SaveJob(job); err != nil {
		logger.Error("Failed to save job", map[string]interface{}{
			"component": "job_manager",
			"job_id":    jobID,
			"error":     err.Error(),
		})
	}

	logger.Info("Job submitted successfully", map[string]interface{}{
		"component": "job_manager",
		"job_id":    jobID,
		"status":    "pending",
		"playbook":  fmt.Sprintf("%d", len(playbook)),
	})

	// Submit to worker pool
	go jm.executeJob(jobID)

	return jobID
}

// GetJob retrieves a job by ID
func (jm *JobManager) GetJob(jobID string) (*Job, bool) {
	return jm.store.LoadJob(jobID)
}

// ListJobs retrieves jobs based on status and limit
func (jm *JobManager) ListJobs(status string, limit int) []*Job {
	return jm.store.ListJobs(status, limit)
}

// GetStats returns job statistics
func (jm *JobManager) GetStats() JobStats {
	return jm.store.GetStats()
}

// CancelJob attempts to cancel a job by ID
func (jm *JobManager) CancelJob(jobID string) (bool, string) {
	job, exists := jm.store.LoadJob(jobID)
	if !exists {
		return false, "Job not found"
	}

	// If the job is running, we can't cancel it immediately.
	// This is a simplified cancellation.
	if job.Status == "running" {
		return false, "Job is currently running and cannot be cancelled immediately."
	}

	// Mark job as cancelled
	if err := jm.store.UpdateJobStatus(jobID, "cancelled"); err != nil {
		return false, fmt.Sprintf("Failed to cancel job: %v", err)
	}

	if err := jm.store.UpdateJobResults(jobID, nil, "Job cancelled by user"); err != nil {
		return false, fmt.Sprintf("Failed to update job results: %v", err)
	}

	return true, "Job cancelled"
}

// Cleanup stops background tasks and closes database connection
func (jm *JobManager) Cleanup() {
	// Stop background tasks
	if jm.cleanupTicker != nil {
		jm.cleanupTicker.Stop()
	}
	if jm.backupTicker != nil {
		jm.backupTicker.Stop()
	}

	// Close database connection
	if jm.store != nil {
		if err := jm.store.Close(); err != nil {
			logger.Error("Failed to close job store", map[string]interface{}{
				"component": "job_manager",
				"error":     err.Error(),
			})
		}
	}

	logger.Info("Job manager cleanup completed", map[string]interface{}{
		"component": "job_manager",
	})
}
