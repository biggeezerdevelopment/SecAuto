package jobs

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"SoarAuto/pkg/types"
)

// JobManager manages job lifecycle and storage
type JobManager struct {
	store  types.RedisJobStore
	jobs   map[string]*types.Job
	mutex  sync.RWMutex
	config interface{} // Will be typed properly when config is used
}

// NewJobManager creates a new job manager
func NewJobManager(store types.RedisJobStore) *JobManager {
	return &JobManager{
		store: store,
		jobs:  make(map[string]*types.Job),
	}
}

// CreateJob creates a new job with the given playbook and context
func (jm *JobManager) CreateJob(playbook interface{}, context map[string]interface{}) (*types.Job, error) {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	// Generate unique job ID
	jobID := uuid.New().String()
	now := time.Now()

	job := &types.Job{
		ID:        jobID,
		Status:    "pending",
		Playbook:  playbook,
		Context:   context,
		CreatedAt: &now,
		Priority:  1, // Default priority
		Metadata:  make(map[string]interface{}),
	}

	// Store in memory cache
	jm.jobs[jobID] = job

	// Store in Redis
	if err := jm.store.StoreJob(job); err != nil {
		delete(jm.jobs, jobID)
		return nil, fmt.Errorf("failed to store job: %v", err)
	}

	return job, nil
}

// GetJob retrieves a job by ID
func (jm *JobManager) GetJob(jobID string) (*types.Job, error) {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	// Check memory cache first
	if job, exists := jm.jobs[jobID]; exists {
		return job, nil
	}

	// Check Redis store
	if job, exists := jm.store.LoadJob(jobID); exists {
		// Cache in memory
		jm.jobs[jobID] = job
		return job, nil
	}

	return nil, fmt.Errorf("job not found: %s", jobID)
}

// UpdateJobStatus updates a job's status
func (jm *JobManager) UpdateJobStatus(jobID string, status string) error {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	// Update memory cache
	if job, exists := jm.jobs[jobID]; exists {
		job.Status = status
		if status == "running" && job.StartedAt == nil {
			now := time.Now()
			job.StartedAt = &now
		} else if (status == "completed" || status == "failed") && job.CompletedAt == nil {
			now := time.Now()
			job.CompletedAt = &now
		}
	}

	// Update Redis store
	return jm.store.UpdateJobStatus(jobID, status)
}

// UpdateJobResults updates a job's results
func (jm *JobManager) UpdateJobResults(jobID string, results interface{}, errorMsg string) error {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	// Update memory cache
	if job, exists := jm.jobs[jobID]; exists {
		job.Results = results
		job.Error = errorMsg
		if errorMsg != "" {
			job.Status = "failed"
		} else {
			job.Status = "completed"
		}
		now := time.Now()
		job.CompletedAt = &now
	}

	// Update Redis store
	return jm.store.UpdateJobResults(jobID, results, errorMsg)
}

// ListJobs returns a list of jobs with optional status filter and limit
func (jm *JobManager) ListJobs(status string, limit int) []*types.Job {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	// Get from Redis store (more comprehensive)
	jobs := jm.store.ListJobs(status, limit)
	
	// Update memory cache with fetched jobs
	for _, job := range jobs {
		jm.jobs[job.ID] = job
	}

	return jobs
}

// GetStats returns job statistics
func (jm *JobManager) GetStats() (*JobStats, error) {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	stats, err := jm.store.GetJobStats()
	if err != nil {
		return nil, err
	}

	// Calculate additional stats
	var totalDuration float64
	var completedCount int
	recentJobs := jm.store.ListJobs("", 10) // Get 10 most recent jobs

	for _, job := range recentJobs {
		if job.CompletedAt != nil && job.StartedAt != nil {
			duration := job.CompletedAt.Sub(*job.StartedAt).Seconds()
			totalDuration += duration
			completedCount++
		}
	}

	var avgDuration float64
	if completedCount > 0 {
		avgDuration = totalDuration / float64(completedCount)
	}

	return &JobStats{
		TotalJobs:   stats["total"],
		Completed:   stats["completed"],
		Failed:      stats["failed"],
		Running:     stats["running"],
		Pending:     stats["pending"],
		AvgDuration: avgDuration,
		RecentJobs:  recentJobs,
	}, nil
}

// DeleteJob removes a job
func (jm *JobManager) DeleteJob(jobID string) error {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	// Remove from memory cache
	delete(jm.jobs, jobID)

	// Remove from Redis store
	return jm.store.DeleteJob(jobID)
}

// JobStats represents job statistics
type JobStats struct {
	TotalJobs   int           `json:"total_jobs"`
	Completed   int           `json:"completed"`
	Failed      int           `json:"failed"`
	Running     int           `json:"running"`
	Pending     int           `json:"pending"`
	AvgDuration float64       `json:"avg_duration_seconds"`
	RecentJobs  []*types.Job  `json:"recent_jobs"`
}

// JobStore implements the RedisJobStore interface using Redis
type JobStore struct {
	redis redisClient
	config jobTTLConfig // TTL configuration for different job statuses
}

// jobTTLConfig holds TTL configuration for jobs
type jobTTLConfig struct {
	RunningJobTTL   int // TTL for running jobs (seconds)
	CompletedJobTTL int // TTL for completed jobs (seconds)
	FailedJobTTL    int // TTL for failed jobs (seconds)
	DefaultJobTTL   int // Default TTL for other statuses (seconds)
}

// redisClient interface for Redis operations
type redisClient interface {
	GetCache(key string) types.CacheResponse
	SetCache(key string, value interface{}) types.CacheResponse
	SetCacheWithTTL(key string, value interface{}, ttl int) types.CacheResponse
	DeleteCache(key string) types.CacheResponse
	ListCacheKeys(pattern string) types.CacheListResponse
	GetCacheStats() types.CacheStatsResponse
	Close() error
}

// NewJobStore creates a new Redis-based job store
func NewJobStore(redis redisClient) *JobStore {
	return &JobStore{
		redis: redis,
		config: jobTTLConfig{
			RunningJobTTL:   7200,  // 2 hours
			CompletedJobTTL: 86400, // 24 hours  
			FailedJobTTL:    604800, // 7 days
			DefaultJobTTL:   0,     // No expiration for other statuses
		},
	}
}

// NewJobStoreWithConfig creates a new Redis-based job store with custom TTL config
func NewJobStoreWithConfig(redis redisClient, runningTTL, completedTTL, failedTTL, defaultTTL int) *JobStore {
	return &JobStore{
		redis: redis,
		config: jobTTLConfig{
			RunningJobTTL:   runningTTL,
			CompletedJobTTL: completedTTL,
			FailedJobTTL:    failedTTL,
			DefaultJobTTL:   defaultTTL,
		},
	}
}

// getTTLForJobStatus returns the appropriate TTL for a job based on its status
func (js *JobStore) getTTLForJobStatus(status string) int {
	switch status {
	case "running":
		return js.config.RunningJobTTL
	case "completed":
		return js.config.CompletedJobTTL
	case "failed":
		return js.config.FailedJobTTL
	default:
		return js.config.DefaultJobTTL
	}
}

// StoreJob stores a job in Redis with appropriate TTL based on status
func (js *JobStore) StoreJob(job *types.Job) error {
	// Serialize job to JSON
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to serialize job: %v", err)
	}
	
	// Get TTL for this job status
	ttl := js.getTTLForJobStatus(job.Status)
	
	// Store job data in Redis with key prefix and TTL
	jobKey := fmt.Sprintf("job:%s", job.ID)
	var response types.CacheResponse
	if ttl > 0 {
		response = js.redis.SetCacheWithTTL(jobKey, string(jobData), ttl)
	} else {
		response = js.redis.SetCache(jobKey, string(jobData))
	}
	
	// Check if Redis operation was successful
	if !response.Success {
		return fmt.Errorf("failed to store job in Redis: %s", response.ErrorMessage)
	}
	
	// Add job ID to jobs list for indexing
	listKey := "jobs:list"
	js.addJobToList(listKey, job.ID)
	
	// Add job ID to status-specific list
	statusKey := fmt.Sprintf("jobs:status:%s", job.Status)
	js.addJobToList(statusKey, job.ID)
	
	return nil
}

// LoadJob loads a job from Redis
func (js *JobStore) LoadJob(id string) (*types.Job, bool) {
	jobKey := fmt.Sprintf("job:%s", id)
	response := js.redis.GetCache(jobKey)
	
	// Check if Redis operation was successful
	if !response.Success {
		return nil, false
	}
	
	// Extract value from response
	if response.Value == nil {
		return nil, false
	}
	
	var job types.Job
	
	// Handle both string and map responses (Redis client may auto-parse JSON)
	switch v := response.Value.(type) {
	case string:
		// Value is a string, unmarshal it
		if err := json.Unmarshal([]byte(v), &job); err != nil {
			return nil, false
		}
	case map[string]interface{}:
		// Value was already parsed by Redis client, re-encode and decode to Job struct
		jobBytes, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		if err := json.Unmarshal(jobBytes, &job); err != nil {
			return nil, false
		}
	default:
		// Unexpected type
		return nil, false
	}
	
	return &job, true
}

// UpdateJobStatus updates job status in Redis
func (js *JobStore) UpdateJobStatus(id string, status string) error {
	// Load existing job
	job, exists := js.LoadJob(id)
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}
	
	// Remove from old status list
	if job.Status != "" {
		oldStatusKey := fmt.Sprintf("jobs:status:%s", job.Status)
		js.removeJobFromList(oldStatusKey, id)
	}
	
	// Update job status
	job.Status = status
	
	// Update timestamps based on status
	now := time.Now()
	if status == "running" && job.StartedAt == nil {
		job.StartedAt = &now
	} else if (status == "completed" || status == "failed") && job.CompletedAt == nil {
		job.CompletedAt = &now
	}
	
	// Store updated job
	if err := js.StoreJob(job); err != nil {
		return fmt.Errorf("failed to update job status: %v", err)
	}
	
	// Add to new status list
	newStatusKey := fmt.Sprintf("jobs:status:%s", status)
	js.addJobToList(newStatusKey, id)
	
	return nil
}

// UpdateJobResults updates job results in Redis
func (js *JobStore) UpdateJobResults(id string, results interface{}, errorMsg string) error {
	// Load existing job
	job, exists := js.LoadJob(id)
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}
	
	// Update job results and error
	job.Results = results
	job.Error = errorMsg
	
	// Update status based on error
	if errorMsg != "" {
		job.Status = "failed"
	} else {
		job.Status = "completed"
	}
	
	// Set completion timestamp
	now := time.Now()
	job.CompletedAt = &now
	
	// Store updated job
	return js.StoreJob(job)
}

// UpdateJobContext updates job context in Redis
func (js *JobStore) UpdateJobContext(id string, context map[string]interface{}) error {
	// Load existing job
	job, exists := js.LoadJob(id)
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}
	
	// Update job context
	job.Context = context
	
	// Store updated job
	return js.StoreJob(job)
}

// ListJobs lists jobs from Redis with optional filters
func (js *JobStore) ListJobs(status string, limit int) []*types.Job {
	var jobs []*types.Job
	
	// Determine which jobs to retrieve
	var pattern string
	if status != "" {
		// Get jobs with specific status
		pattern = fmt.Sprintf("jobs:status:%s:*", status)
	} else {
		// Get all jobs
		pattern = "job:*"
	}
	
	// Get job keys from Redis
	response := js.redis.ListCacheKeys(pattern)
	var jobKeys []string
	
	if response.Success {
		jobKeys = response.Keys
	}
	
	// Load each job
	for i, jobKey := range jobKeys {
		if limit > 0 && i >= limit {
			break
		}
		
		// Extract job ID from key
		var jobID string
		if status != "" {
			// Extract from status key format: jobs:status:pending:job-id
			parts := strings.Split(jobKey, ":")
			if len(parts) >= 4 {
				jobID = parts[3]
			}
		} else {
			// Extract from job key format: job:job-id
			parts := strings.Split(jobKey, ":")
			if len(parts) >= 2 {
				jobID = parts[1]
			}
		}
		
		if jobID != "" {
			if job, exists := js.LoadJob(jobID); exists {
				jobs = append(jobs, job)
			}
		}
	}
	
	return jobs
}

// GetJobStats returns job statistics from Redis
func (js *JobStore) GetJobStats() (map[string]int, error) {
	stats := map[string]int{
		"total":     0,
		"completed": 0,
		"failed":    0,
		"running":   0,
		"pending":   0,
	}
	
	// Count jobs by status
	statuses := []string{"pending", "running", "completed", "failed"}
	
	for _, status := range statuses {
		jobs := js.ListJobs(status, 0) // No limit to get all jobs
		stats[status] = len(jobs)
		stats["total"] += len(jobs)
	}
	
	return stats, nil
}

// DeleteJob deletes a job from Redis
func (js *JobStore) DeleteJob(id string) error {
	// Load job to get status for cleanup
	job, exists := js.LoadJob(id)
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}
	
	// Delete main job record
	jobKey := fmt.Sprintf("job:%s", id)
	response := js.redis.DeleteCache(jobKey)
	
	// Check if deletion was successful
	if !response.Success {
		return fmt.Errorf("failed to delete job from Redis: %s", response.ErrorMessage)
	}
	
	// Remove from job lists
	js.removeJobFromList("jobs:list", id)
	if job.Status != "" {
		statusKey := fmt.Sprintf("jobs:status:%s", job.Status)
		js.removeJobFromList(statusKey, id)
	}
	
	return nil
}

// Close closes the job store connections
func (js *JobStore) Close() error {
	return js.redis.Close()
}

// addJobToList adds a job ID to a Redis list (helper method)
func (js *JobStore) addJobToList(listKey, jobID string) {
	// This uses the Redis list operations from the Redis client
	// We'll implement this as a simple cache operation for now
	js.redis.SetCache(fmt.Sprintf("%s:%s", listKey, jobID), jobID)
}

// removeJobFromList removes a job ID from a Redis list (helper method)
func (js *JobStore) removeJobFromList(listKey, jobID string) {
	js.redis.DeleteCache(fmt.Sprintf("%s:%s", listKey, jobID))
}