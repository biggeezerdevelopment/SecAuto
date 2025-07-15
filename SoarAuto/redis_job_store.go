package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisJobStore provides persistent storage for jobs using Redis
type RedisJobStore struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisJobStore creates a new Redis job store
func NewRedisJobStore(redisURL string) (*RedisJobStore, error) {
	// Parse Redis URL (format: redis://host:port/db)
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opt)
	ctx := context.Background()

	// Test the connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	store := &RedisJobStore{
		client: client,
		ctx:    ctx,
	}

	logger.Info("Initialized Redis job store", map[string]interface{}{
		"component": "job_store",
		"redis_url": redisURL,
	})

	return store, nil
}

// SaveJob persists a job to Redis
func (rjs *RedisJobStore) SaveJob(job *Job) error {
	// Serialize job to JSON
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %v", err)
	}

	// Store job with 24-hour TTL
	key := fmt.Sprintf("job:%s", job.ID)
	err = rjs.client.Set(rjs.ctx, key, data, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to save job: %v", err)
	}

	// Also store in job list for easy querying
	listKey := "jobs:list"
	err = rjs.client.ZAdd(rjs.ctx, listKey, &redis.Z{
		Score:  float64(job.CreatedAt.Unix()),
		Member: job.ID,
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to add job to list: %v", err)
	}

	return nil
}

// LoadJob retrieves a job by ID from Redis
func (rjs *RedisJobStore) LoadJob(jobID string) (*Job, bool) {
	key := fmt.Sprintf("job:%s", jobID)
	data, err := rjs.client.Get(rjs.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false
		}
		logger.Error("Failed to load job", map[string]interface{}{
			"component": "job_store",
			"job_id":    jobID,
			"error":     err.Error(),
		})
		return nil, false
	}

	var job Job
	if err := json.Unmarshal([]byte(data), &job); err != nil {
		logger.Error("Failed to unmarshal job", map[string]interface{}{
			"component": "job_store",
			"job_id":    jobID,
			"error":     err.Error(),
		})
		return nil, false
	}

	return &job, true
}

// ListJobs retrieves jobs based on status and limit from Redis
func (rjs *RedisJobStore) ListJobs(status string, limit int) []*Job {
	var jobs []*Job

	// Get job IDs from sorted set (ordered by creation time)
	listKey := "jobs:list"
	jobIDs, err := rjs.client.ZRevRange(rjs.ctx, listKey, 0, int64(limit-1)).Result()
	if err != nil {
		logger.Error("Failed to get job IDs", map[string]interface{}{
			"component": "job_store",
			"error":     err.Error(),
		})
		return jobs
	}

	// Load each job
	for _, jobID := range jobIDs {
		job, exists := rjs.LoadJob(jobID)
		if !exists {
			continue
		}

		// Filter by status if specified
		if status == "" || job.Status == status {
			jobs = append(jobs, job)
		}
	}

	return jobs
}

// UpdateJobStatus updates a job's status in Redis
func (rjs *RedisJobStore) UpdateJobStatus(jobID, status string) error {
	// Load current job
	job, exists := rjs.LoadJob(jobID)
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Update status and timestamps
	job.Status = status
	now := time.Now()

	switch status {
	case "running":
		job.StartedAt = &now
	case "completed", "failed", "cancelled":
		job.CompletedAt = &now
	}

	// Save updated job
	return rjs.SaveJob(job)
}

// UpdateJobResults updates a job's results and error in Redis
func (rjs *RedisJobStore) UpdateJobResults(jobID string, results []interface{}, errorMsg string) error {
	// Load current job
	job, exists := rjs.LoadJob(jobID)
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Update results and error
	job.Results = results
	job.Error = errorMsg
	now := time.Now()
	job.CompletedAt = &now

	// Save updated job
	return rjs.SaveJob(job)
}

// UpdateJobContext updates a job's context in Redis
func (rjs *RedisJobStore) UpdateJobContext(jobID string, context map[string]interface{}) error {
	// Load current job
	job, exists := rjs.LoadJob(jobID)
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Update context
	job.Context = context

	// Save updated job
	return rjs.SaveJob(job)
}

// DeleteJob removes a job from Redis
func (rjs *RedisJobStore) DeleteJob(jobID string) error {
	key := fmt.Sprintf("job:%s", jobID)

	// Remove from job storage
	err := rjs.client.Del(rjs.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete job: %v", err)
	}

	// Remove from job list
	listKey := "jobs:list"
	err = rjs.client.ZRem(rjs.ctx, listKey, jobID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove job from list: %v", err)
	}

	return nil
}

// CleanupOldJobs removes jobs older than specified duration from Redis
func (rjs *RedisJobStore) CleanupOldJobs(maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)

	// Get all job IDs
	listKey := "jobs:list"
	jobIDs, err := rjs.client.ZRange(rjs.ctx, listKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to get job IDs: %v", err)
	}

	deleted := 0
	for _, jobID := range jobIDs {
		job, exists := rjs.LoadJob(jobID)
		if !exists {
			continue
		}

		// Check if job is older than cutoff
		if job.CreatedAt.Before(cutoff) {
			if err := rjs.DeleteJob(jobID); err != nil {
				logger.Error("Failed to delete old job", map[string]interface{}{
					"component": "job_store",
					"job_id":    jobID,
					"error":     err.Error(),
				})
			} else {
				deleted++
			}
		}
	}

	if deleted > 0 {
		logger.Info("Cleaned up old jobs", map[string]interface{}{
			"component": "job_store",
			"deleted":   deleted,
			"max_age":   maxAge.String(),
		})
	}

	return nil
}

// GetStats returns job statistics from Redis
func (rjs *RedisJobStore) GetStats() JobStats {
	var stats JobStats

	// Get all jobs
	jobs := rjs.ListJobs("", 1000) // Get up to 1000 jobs for stats

	stats.TotalJobs = len(jobs)

	// Calculate stats
	var totalDuration float64
	completedCount := 0

	for _, job := range jobs {
		switch job.Status {
		case "completed":
			stats.Completed++
			completedCount++
			if job.StartedAt != nil && job.CompletedAt != nil {
				duration := job.CompletedAt.Sub(*job.StartedAt).Seconds()
				totalDuration += duration
			}
		case "failed":
			stats.Failed++
		case "running":
			stats.Running++
		case "pending":
			stats.Pending++
		}
	}

	// Calculate average duration
	if completedCount > 0 {
		stats.AvgDuration = totalDuration / float64(completedCount)
	}

	// Get recent jobs (last 10)
	if len(jobs) > 10 {
		stats.RecentJobs = jobs[:10]
	} else {
		stats.RecentJobs = jobs
	}

	return stats
}

// BackupJobs creates a backup of jobs from Redis
func (rjs *RedisJobStore) BackupJobs() error {
	// Get all jobs
	jobs := rjs.ListJobs("", 10000) // Get up to 10k jobs for backup

	if len(jobs) == 0 {
		return nil
	}

	// Create backup data
	backup := struct {
		Timestamp time.Time `json:"timestamp"`
		JobCount  int       `json:"job_count"`
		Jobs      []*Job    `json:"jobs"`
	}{
		Timestamp: time.Now(),
		JobCount:  len(jobs),
		Jobs:      jobs,
	}

	// Serialize backup
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup: %v", err)
	}

	// Store backup in Redis with 7-day TTL
	backupKey := fmt.Sprintf("backup:%s", time.Now().Format("2006-01-02-15-04-05"))
	err = rjs.client.Set(rjs.ctx, backupKey, data, 7*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to store backup: %v", err)
	}

	logger.Info("Created job backup", map[string]interface{}{
		"component":  "job_store",
		"backup_key": backupKey,
		"job_count":  len(jobs),
	})

	return nil
}

// RecoverJobs recovers jobs that were running during a crash
func (rjs *RedisJobStore) RecoverJobs(engine *RuleEngine, webhookManager *WebhookManager) {
	// Get all running jobs
	runningJobs := rjs.ListJobs("running", 1000)

	if len(runningJobs) > 0 {
		logger.Info("Recovering jobs that were running during crash", map[string]interface{}{
			"component": "job_store",
			"count":     len(runningJobs),
		})

		for _, job := range runningJobs {
			// Mark as failed since we can't guarantee the state
			rjs.UpdateJobStatus(job.ID, "failed")
			rjs.UpdateJobResults(job.ID, nil, "Job failed due to server restart")

			// Send webhook notification
			if webhookManager != nil {
				webhookManager.SendWebhook(WebhookEvent{
					Event:     "job_failed",
					JobID:     job.ID,
					Status:    "failed",
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Playbook:  job.Playbook,
					Context:   job.Context,
					Error:     "Job failed due to server restart",
				})
			}
		}
	}
}

// Close closes the Redis connection
func (rjs *RedisJobStore) Close() error {
	if rjs.client != nil {
		logger.Info("Closing Redis job store", map[string]interface{}{
			"component": "job_store",
		})
		return rjs.client.Close()
	}
	return nil
}

// GetDatabaseMetrics returns Redis metrics (simplified)
func (rjs *RedisJobStore) GetDatabaseMetrics() map[string]interface{} {
	info, err := rjs.client.Info(rjs.ctx).Result()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	return map[string]interface{}{
		"type": "redis",
		"info": info,
	}
}

// Schedule-related methods (placeholder implementations)
func (rjs *RedisJobStore) SaveSchedule(schedule *JobSchedule) error {
	// TODO: Implement schedule storage in Redis
	return fmt.Errorf("schedule storage not implemented in Redis job store")
}

func (rjs *RedisJobStore) LoadSchedule(scheduleID string) (*JobSchedule, bool) {
	// TODO: Implement schedule loading from Redis
	return nil, false
}

func (rjs *RedisJobStore) ListSchedules(status string, limit int) []*JobSchedule {
	// TODO: Implement schedule listing from Redis
	return []*JobSchedule{}
}

func (rjs *RedisJobStore) UpdateSchedule(schedule *JobSchedule) error {
	// TODO: Implement schedule updating in Redis
	return fmt.Errorf("schedule updating not implemented in Redis job store")
}

func (rjs *RedisJobStore) DeleteSchedule(scheduleID string) error {
	// TODO: Implement schedule deletion from Redis
	return fmt.Errorf("schedule deletion not implemented in Redis job store")
}

func (rjs *RedisJobStore) GetSchedulesDueForExecution() []*JobSchedule {
	// TODO: Implement schedule execution querying from Redis
	return []*JobSchedule{}
}
