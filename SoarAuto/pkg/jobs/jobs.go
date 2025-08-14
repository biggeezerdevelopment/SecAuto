package jobs

import (
	"fmt"
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
	redis interface{} // Will be properly typed
}

// NewJobStore creates a new Redis-based job store
func NewJobStore(redis interface{}) *JobStore {
	return &JobStore{
		redis: redis,
	}
}

// StoreJob stores a job in Redis
func (js *JobStore) StoreJob(job *types.Job) error {
	// Implementation would serialize job to JSON and store in Redis
	// For now, return nil to allow compilation
	return nil
}

// LoadJob loads a job from Redis
func (js *JobStore) LoadJob(id string) (*types.Job, bool) {
	// Implementation would load job from Redis and deserialize
	// For now, return nil, false to allow compilation
	return nil, false
}

// UpdateJobStatus updates job status in Redis
func (js *JobStore) UpdateJobStatus(id string, status string) error {
	// Implementation would update job status in Redis
	return nil
}

// UpdateJobResults updates job results in Redis
func (js *JobStore) UpdateJobResults(id string, results interface{}, errorMsg string) error {
	// Implementation would update job results in Redis
	return nil
}

// UpdateJobContext updates job context in Redis
func (js *JobStore) UpdateJobContext(id string, context map[string]interface{}) error {
	// Implementation would update job context in Redis
	return nil
}

// ListJobs lists jobs from Redis with optional filters
func (js *JobStore) ListJobs(status string, limit int) []*types.Job {
	// Implementation would query Redis for jobs
	// For now, return empty slice to allow compilation
	return []*types.Job{}
}

// GetJobStats returns job statistics from Redis
func (js *JobStore) GetJobStats() (map[string]int, error) {
	// Implementation would calculate stats from Redis
	// For now, return basic stats to allow compilation
	return map[string]int{
		"total":     0,
		"completed": 0,
		"failed":    0,
		"running":   0,
		"pending":   0,
	}, nil
}

// DeleteJob deletes a job from Redis
func (js *JobStore) DeleteJob(id string) error {
	// Implementation would delete job from Redis
	return nil
}

// Close closes the job store connections
func (js *JobStore) Close() error {
	// Implementation would close Redis connections
	return nil
}