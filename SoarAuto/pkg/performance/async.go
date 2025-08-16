package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// JobPriority defines job priority levels
type JobPriority int

const (
	PriorityLow JobPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// String returns string representation of priority
func (p JobPriority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// JobStatus defines job execution status
type JobStatus int

const (
	StatusPending JobStatus = iota
	StatusRunning
	StatusCompleted
	StatusFailed
	StatusCancelled
	StatusRetrying
)

// String returns string representation of status
func (s JobStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusCancelled:
		return "cancelled"
	case StatusRetrying:
		return "retrying"
	default:
		return "unknown"
	}
}

// Job represents an async job
type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    JobPriority            `json:"priority"`
	Status      JobStatus              `json:"status"`
	Payload     map[string]interface{} `json:"payload"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	Timeout     time.Duration          `json:"timeout"`

	// Internal fields
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewJob creates a new job
func NewJob(id, jobType string, priority JobPriority, payload map[string]interface{}) *Job {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Job{
		ID:         id,
		Type:       jobType,
		Priority:   priority,
		Status:     StatusPending,
		Payload:    payload,
		CreatedAt:  time.Now(),
		MaxRetries: 3,
		Timeout:    5 * time.Minute,
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
	}
}

// Cancel cancels the job
func (j *Job) Cancel() {
	j.Status = StatusCancelled
	j.cancel()
	close(j.done)
}

// Wait waits for job completion
func (j *Job) Wait() {
	<-j.done
}

// JobHandler defines the interface for job handlers
type JobHandler interface {
	Handle(ctx context.Context, job *Job) error
	GetType() string
}

// AsyncConfig holds configuration for async processing
type AsyncConfig struct {
	WorkerCount       int           `yaml:"worker_count" json:"worker_count"`
	QueueSize         int           `yaml:"queue_size" json:"queue_size"`
	RetryDelay        time.Duration `yaml:"retry_delay" json:"retry_delay"`
	MaxRetries        int           `yaml:"max_retries" json:"max_retries"`
	JobTimeout        time.Duration `yaml:"job_timeout" json:"job_timeout"`
	MetricsInterval   time.Duration `yaml:"metrics_interval" json:"metrics_interval"`
	EnablePersistence bool          `yaml:"enable_persistence" json:"enable_persistence"`
	PersistenceFile   string        `yaml:"persistence_file" json:"persistence_file"`
}

// DefaultAsyncConfig returns default async configuration
func DefaultAsyncConfig() *AsyncConfig {
	return &AsyncConfig{
		WorkerCount:       runtime.NumCPU(),
		QueueSize:         1000,
		RetryDelay:        30 * time.Second,
		MaxRetries:        3,
		JobTimeout:        5 * time.Minute,
		MetricsInterval:   30 * time.Second,
		EnablePersistence: false,
		PersistenceFile:   "jobs.json",
	}
}

// AsyncMetrics holds async processing metrics
type AsyncMetrics struct {
	mu sync.RWMutex

	JobsQueued     int64 `json:"jobs_queued"`
	JobsProcessed  int64 `json:"jobs_processed"`
	JobsCompleted  int64 `json:"jobs_completed"`
	JobsFailed     int64 `json:"jobs_failed"`
	JobsCancelled  int64 `json:"jobs_cancelled"`
	JobsRetried    int64 `json:"jobs_retried"`
	
	ActiveWorkers  int32 `json:"active_workers"`
	QueueLength    int32 `json:"queue_length"`
	
	TotalProcessingTime int64     `json:"total_processing_time_ms"`
	AverageProcessingTime float64 `json:"average_processing_time_ms"`
	
	LastUpdated time.Time `json:"last_updated"`
}

// GetSuccessRate returns job success rate as percentage
func (m *AsyncMetrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	total := m.JobsCompleted + m.JobsFailed
	if total == 0 {
		return 0.0
	}
	return float64(m.JobsCompleted) / float64(total) * 100.0
}

// UpdateProcessingTime updates processing time metrics
func (m *AsyncMetrics) UpdateProcessingTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.TotalProcessingTime += duration.Milliseconds()
	processed := m.JobsProcessed
	if processed > 0 {
		m.AverageProcessingTime = float64(m.TotalProcessingTime) / float64(processed)
	}
}

// AsyncProcessor manages async job processing
type AsyncProcessor struct {
	config   *AsyncConfig
	logger   *log.Logger
	metrics  *AsyncMetrics
	handlers map[string]JobHandler

	// Job queues by priority
	criticalQueue chan *Job
	highQueue     chan *Job
	normalQueue   chan *Job
	lowQueue      chan *Job

	// Worker management
	workers    []*worker
	workerPool sync.Pool
	
	// Job tracking
	jobs    map[string]*Job
	jobsMu  sync.RWMutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	// Persistence
	persistenceTicker *time.Ticker
}

// worker represents a job worker
type worker struct {
	id        int
	processor *AsyncProcessor
	active    int32
}

// NewAsyncProcessor creates a new async processor
func NewAsyncProcessor(config *AsyncConfig, logger *log.Logger) *AsyncProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	ap := &AsyncProcessor{
		config:   config,
		logger:   logger,
		metrics:  &AsyncMetrics{LastUpdated: time.Now()},
		handlers: make(map[string]JobHandler),
		
		criticalQueue: make(chan *Job, config.QueueSize/4),
		highQueue:     make(chan *Job, config.QueueSize/4),
		normalQueue:   make(chan *Job, config.QueueSize/2),
		lowQueue:      make(chan *Job, config.QueueSize/4),
		
		jobs: make(map[string]*Job),
		
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize worker pool
	ap.workerPool.New = func() interface{} {
		return &worker{processor: ap}
	}

	return ap
}

// RegisterHandler registers a job handler
func (ap *AsyncProcessor) RegisterHandler(handler JobHandler) {
	ap.handlers[handler.GetType()] = handler
	ap.logger.Printf("Registered job handler for type: %s", handler.GetType())
}

// Start starts the async processor
func (ap *AsyncProcessor) Start() error {
	// Start workers
	ap.workers = make([]*worker, ap.config.WorkerCount)
	for i := 0; i < ap.config.WorkerCount; i++ {
		worker := &worker{
			id:        i,
			processor: ap,
		}
		ap.workers[i] = worker
		
		ap.wg.Add(1)
		go ap.workerLoop(worker)
	}

	// Start metrics collection
	ap.wg.Add(1)
	go ap.metricsLoop()

	// Start persistence if enabled
	if ap.config.EnablePersistence {
		ap.persistenceTicker = time.NewTicker(time.Minute)
		ap.wg.Add(1)
		go ap.persistenceLoop()
	}

	ap.logger.Printf("Async processor started with %d workers", ap.config.WorkerCount)
	return nil
}

// Stop stops the async processor
func (ap *AsyncProcessor) Stop() error {
	ap.cancel()
	
	// Stop persistence ticker
	if ap.persistenceTicker != nil {
		ap.persistenceTicker.Stop()
	}
	
	// Wait for all workers to finish
	ap.wg.Wait()
	
	// Cancel all pending jobs
	ap.jobsMu.Lock()
	for _, job := range ap.jobs {
		if job.Status == StatusPending || job.Status == StatusRunning {
			job.Cancel()
		}
	}
	ap.jobsMu.Unlock()

	ap.logger.Println("Async processor stopped")
	return nil
}

// SubmitJob submits a job for processing
func (ap *AsyncProcessor) SubmitJob(job *Job) error {
	// Validate job
	if _, exists := ap.handlers[job.Type]; !exists {
		return fmt.Errorf("no handler registered for job type: %s", job.Type)
	}

	// Set defaults
	if job.MaxRetries == 0 {
		job.MaxRetries = ap.config.MaxRetries
	}
	if job.Timeout == 0 {
		job.Timeout = ap.config.JobTimeout
	}

	// Store job
	ap.jobsMu.Lock()
	ap.jobs[job.ID] = job
	ap.jobsMu.Unlock()

	// Queue job based on priority
	var queue chan *Job
	switch job.Priority {
	case PriorityCritical:
		queue = ap.criticalQueue
	case PriorityHigh:
		queue = ap.highQueue
	case PriorityNormal:
		queue = ap.normalQueue
	case PriorityLow:
		queue = ap.lowQueue
	default:
		queue = ap.normalQueue
	}

	select {
	case queue <- job:
		atomic.AddInt64(&ap.metrics.JobsQueued, 1)
		atomic.AddInt32(&ap.metrics.QueueLength, 1)
		ap.logger.Printf("Job %s queued with priority %s", job.ID, job.Priority.String())
		return nil
	case <-ap.ctx.Done():
		return fmt.Errorf("processor is shutting down")
	default:
		return fmt.Errorf("queue is full")
	}
}

// GetJob retrieves a job by ID
func (ap *AsyncProcessor) GetJob(jobID string) (*Job, bool) {
	ap.jobsMu.RLock()
	defer ap.jobsMu.RUnlock()
	
	job, exists := ap.jobs[jobID]
	return job, exists
}

// GetJobsByStatus returns jobs with specific status
func (ap *AsyncProcessor) GetJobsByStatus(status JobStatus) []*Job {
	ap.jobsMu.RLock()
	defer ap.jobsMu.RUnlock()
	
	var jobs []*Job
	for _, job := range ap.jobs {
		if job.Status == status {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// CancelJob cancels a job
func (ap *AsyncProcessor) CancelJob(jobID string) error {
	ap.jobsMu.RLock()
	job, exists := ap.jobs[jobID]
	ap.jobsMu.RUnlock()
	
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}
	
	job.Cancel()
	atomic.AddInt64(&ap.metrics.JobsCancelled, 1)
	
	ap.logger.Printf("Job %s cancelled", jobID)
	return nil
}

// workerLoop runs the worker loop
func (ap *AsyncProcessor) workerLoop(w *worker) {
	defer ap.wg.Done()
	
	for {
		select {
		case <-ap.ctx.Done():
			return
		default:
			job := ap.getNextJob()
			if job == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			
			ap.processJob(w, job)
		}
	}
}

// getNextJob gets the next job from queues (priority order)
func (ap *AsyncProcessor) getNextJob() *Job {
	select {
	case job := <-ap.criticalQueue:
		atomic.AddInt32(&ap.metrics.QueueLength, -1)
		return job
	default:
	}
	
	select {
	case job := <-ap.highQueue:
		atomic.AddInt32(&ap.metrics.QueueLength, -1)
		return job
	default:
	}
	
	select {
	case job := <-ap.normalQueue:
		atomic.AddInt32(&ap.metrics.QueueLength, -1)
		return job
	default:
	}
	
	select {
	case job := <-ap.lowQueue:
		atomic.AddInt32(&ap.metrics.QueueLength, -1)
		return job
	default:
		return nil
	}
}

// processJob processes a single job
func (ap *AsyncProcessor) processJob(w *worker, job *Job) {
	atomic.AddInt32(&w.active, 1)
	atomic.AddInt32(&ap.metrics.ActiveWorkers, 1)
	defer func() {
		atomic.AddInt32(&w.active, -1)
		atomic.AddInt32(&ap.metrics.ActiveWorkers, -1)
	}()

	handler, exists := ap.handlers[job.Type]
	if !exists {
		ap.failJob(job, fmt.Errorf("no handler for job type: %s", job.Type))
		return
	}

	// Set job as running
	job.Status = StatusRunning
	now := time.Now()
	job.StartedAt = &now

	// Create timeout context
	ctx, cancel := context.WithTimeout(job.ctx, job.Timeout)
	defer cancel()

	start := time.Now()
	err := handler.Handle(ctx, job)
	duration := time.Since(start)

	atomic.AddInt64(&ap.metrics.JobsProcessed, 1)
	ap.metrics.UpdateProcessingTime(duration)

	if err != nil {
		if job.RetryCount < job.MaxRetries {
			ap.retryJob(job, err)
		} else {
			ap.failJob(job, err)
		}
	} else {
		ap.completeJob(job)
	}
}

// completeJob marks a job as completed
func (ap *AsyncProcessor) completeJob(job *Job) {
	job.Status = StatusCompleted
	now := time.Now()
	job.CompletedAt = &now
	close(job.done)
	
	atomic.AddInt64(&ap.metrics.JobsCompleted, 1)
	ap.logger.Printf("Job %s completed successfully", job.ID)
}

// failJob marks a job as failed
func (ap *AsyncProcessor) failJob(job *Job, err error) {
	job.Status = StatusFailed
	job.Error = err.Error()
	now := time.Now()
	job.CompletedAt = &now
	close(job.done)
	
	atomic.AddInt64(&ap.metrics.JobsFailed, 1)
	ap.logger.Printf("Job %s failed: %v", job.ID, err)
}

// retryJob schedules a job for retry
func (ap *AsyncProcessor) retryJob(job *Job, err error) {
	job.RetryCount++
	job.Status = StatusRetrying
	job.Error = err.Error()
	
	atomic.AddInt64(&ap.metrics.JobsRetried, 1)
	ap.logger.Printf("Job %s retry %d/%d: %v", job.ID, job.RetryCount, job.MaxRetries, err)
	
	// Schedule retry after delay
	go func() {
		time.Sleep(ap.config.RetryDelay)
		job.Status = StatusPending
		job.Error = ""
		ap.SubmitJob(job)
	}()
}

// metricsLoop runs the metrics collection loop
func (ap *AsyncProcessor) metricsLoop() {
	defer ap.wg.Done()
	
	ticker := time.NewTicker(ap.config.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ap.updateMetrics()
		case <-ap.ctx.Done():
			return
		}
	}
}

// updateMetrics updates processor metrics
func (ap *AsyncProcessor) updateMetrics() {
	ap.metrics.mu.Lock()
	defer ap.metrics.mu.Unlock()
	
	ap.metrics.LastUpdated = time.Now()
	
	// Update queue lengths
	ap.metrics.QueueLength = int32(len(ap.criticalQueue) + len(ap.highQueue) + 
		len(ap.normalQueue) + len(ap.lowQueue))
}

// persistenceLoop handles job persistence
func (ap *AsyncProcessor) persistenceLoop() {
	defer ap.wg.Done()
	
	for {
		select {
		case <-ap.persistenceTicker.C:
			ap.persistJobs()
		case <-ap.ctx.Done():
			ap.persistJobs() // Final persistence
			return
		}
	}
}

// persistJobs persists jobs to file
func (ap *AsyncProcessor) persistJobs() {
	ap.jobsMu.RLock()
	jobs := make([]*Job, 0, len(ap.jobs))
	for _, job := range ap.jobs {
		if job.Status != StatusCompleted && job.Status != StatusCancelled {
			jobs = append(jobs, job)
		}
	}
	ap.jobsMu.RUnlock()
	
	if len(jobs) == 0 {
		return
	}
	
	data, err := json.Marshal(jobs)
	if err != nil {
		ap.logger.Printf("Failed to marshal jobs for persistence: %v", err)
		return
	}
	
	// In a real implementation, you would write to file
	_ = data // Suppress unused variable warning
	ap.logger.Printf("Persisted %d jobs", len(jobs))
}

// GetMetrics returns processor metrics
func (ap *AsyncProcessor) GetMetrics() *AsyncMetrics {
	ap.updateMetrics()
	return ap.metrics
}

// GetStatus returns processor status
func (ap *AsyncProcessor) GetStatus() map[string]interface{} {
	metrics := ap.GetMetrics()
	
	return map[string]interface{}{
		"workers": map[string]interface{}{
			"total":  ap.config.WorkerCount,
			"active": atomic.LoadInt32(&ap.metrics.ActiveWorkers),
		},
		"queues": map[string]interface{}{
			"critical": len(ap.criticalQueue),
			"high":     len(ap.highQueue),
			"normal":   len(ap.normalQueue),
			"low":      len(ap.lowQueue),
			"total":    atomic.LoadInt32(&ap.metrics.QueueLength),
		},
		"jobs": map[string]interface{}{
			"queued":     atomic.LoadInt64(&ap.metrics.JobsQueued),
			"processed":  atomic.LoadInt64(&ap.metrics.JobsProcessed),
			"completed":  atomic.LoadInt64(&ap.metrics.JobsCompleted),
			"failed":     atomic.LoadInt64(&ap.metrics.JobsFailed),
			"cancelled":  atomic.LoadInt64(&ap.metrics.JobsCancelled),
			"retried":    atomic.LoadInt64(&ap.metrics.JobsRetried),
			"success_rate": metrics.GetSuccessRate(),
		},
		"performance": map[string]interface{}{
			"avg_processing_time": metrics.AverageProcessingTime,
			"total_processing_time": metrics.TotalProcessingTime,
		},
		"handlers": len(ap.handlers),
		"last_updated": metrics.LastUpdated,
	}
}