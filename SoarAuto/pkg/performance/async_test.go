package performance

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

// TestJobHandler is a test implementation of JobHandler
type TestJobHandler struct {
	jobType    string
	delay      time.Duration
	shouldFail bool
	callCount  int32
	mu         sync.Mutex
}

func NewTestJobHandler(jobType string, delay time.Duration, shouldFail bool) *TestJobHandler {
	return &TestJobHandler{
		jobType:    jobType,
		delay:      delay,
		shouldFail: shouldFail,
	}
}

func (h *TestJobHandler) Handle(ctx context.Context, job *Job) error {
	h.mu.Lock()
	h.callCount++
	h.mu.Unlock()

	if h.delay > 0 {
		select {
		case <-time.After(h.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if h.shouldFail {
		return fmt.Errorf("test handler failure")
	}

	job.Result = map[string]interface{}{
		"processed": true,
		"timestamp": time.Now(),
	}

	return nil
}

func (h *TestJobHandler) GetType() string {
	return h.jobType
}

func (h *TestJobHandler) GetCallCount() int32 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.callCount
}

func TestJobPriorityString(t *testing.T) {
	tests := []struct {
		priority JobPriority
		expected string
	}{
		{PriorityLow, "low"},
		{PriorityNormal, "normal"},
		{PriorityHigh, "high"},
		{PriorityCritical, "critical"},
		{JobPriority(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.priority.String(); got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestJobStatusString(t *testing.T) {
	tests := []struct {
		status   JobStatus
		expected string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusCancelled, "cancelled"},
		{StatusRetrying, "retrying"},
		{JobStatus(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestNewJob(t *testing.T) {
	payload := map[string]interface{}{
		"test": "data",
		"num":  123,
	}

	job := NewJob("test-job-1", "test", PriorityHigh, payload)

	if job.ID != "test-job-1" {
		t.Errorf("Expected ID 'test-job-1', got %s", job.ID)
	}

	if job.Type != "test" {
		t.Errorf("Expected Type 'test', got %s", job.Type)
	}

	if job.Priority != PriorityHigh {
		t.Errorf("Expected Priority PriorityHigh, got %v", job.Priority)
	}

	if job.Status != StatusPending {
		t.Errorf("Expected Status StatusPending, got %v", job.Status)
	}

	if job.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", job.MaxRetries)
	}

	if job.Timeout != 5*time.Minute {
		t.Errorf("Expected Timeout 5m, got %v", job.Timeout)
	}

	if job.ctx == nil {
		t.Error("Expected context to be set")
	}

	if job.cancel == nil {
		t.Error("Expected cancel function to be set")
	}

	if job.done == nil {
		t.Error("Expected done channel to be set")
	}
}

func TestJobCancel(t *testing.T) {
	job := NewJob("test-job", "test", PriorityNormal, nil)

	// Cancel the job
	job.Cancel()

	if job.Status != StatusCancelled {
		t.Errorf("Expected Status StatusCancelled, got %v", job.Status)
	}

	// Check if context is cancelled
	select {
	case <-job.ctx.Done():
		// Expected
	default:
		t.Error("Expected context to be cancelled")
	}

	// Check if done channel is closed
	select {
	case <-job.done:
		// Expected
	default:
		t.Error("Expected done channel to be closed")
	}
}

func TestDefaultAsyncConfig(t *testing.T) {
	config := DefaultAsyncConfig()

	if config.WorkerCount <= 0 {
		t.Error("Expected WorkerCount to be positive")
	}

	if config.QueueSize != 1000 {
		t.Errorf("Expected QueueSize 1000, got %d", config.QueueSize)
	}

	if config.RetryDelay != 30*time.Second {
		t.Errorf("Expected RetryDelay 30s, got %v", config.RetryDelay)
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", config.MaxRetries)
	}

	if config.JobTimeout != 5*time.Minute {
		t.Errorf("Expected JobTimeout 5m, got %v", config.JobTimeout)
	}
}

func TestAsyncMetrics(t *testing.T) {
	metrics := &AsyncMetrics{}

	// Test initial success rate
	if rate := metrics.GetSuccessRate(); rate != 0.0 {
		t.Errorf("Expected initial success rate 0.0, got %f", rate)
	}

	// Set some values
	metrics.mu.Lock()
	metrics.JobsCompleted = 80
	metrics.JobsFailed = 20
	metrics.mu.Unlock()

	expectedRate := 80.0
	if rate := metrics.GetSuccessRate(); rate != expectedRate {
		t.Errorf("Expected success rate %f, got %f", expectedRate, rate)
	}

	// Test processing time update
	metrics.mu.Lock()
	metrics.JobsProcessed = 2
	metrics.mu.Unlock()
	
	metrics.UpdateProcessingTime(100 * time.Millisecond)
	metrics.UpdateProcessingTime(200 * time.Millisecond)

	expectedAvg := 150.0 // (100 + 200) / 2
	if avg := metrics.AverageProcessingTime; avg != expectedAvg {
		t.Errorf("Expected average processing time %f, got %f", expectedAvg, avg)
	}
}

func TestAsyncProcessorCreation(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultAsyncConfig()
	config.WorkerCount = 2
	config.QueueSize = 100

	processor := NewAsyncProcessor(config, logger)

	if processor == nil {
		t.Fatal("Expected processor to be created")
	}

	if processor.config != config {
		t.Error("Expected config to be set")
	}

	if processor.logger != logger {
		t.Error("Expected logger to be set")
	}

	if processor.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}

	if len(processor.handlers) != 0 {
		t.Error("Expected handlers map to be empty initially")
	}
}

func TestAsyncProcessorHandlerRegistration(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultAsyncConfig()
	processor := NewAsyncProcessor(config, logger)

	handler := NewTestJobHandler("test-type", 0, false)
	processor.RegisterHandler(handler)

	if len(processor.handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(processor.handlers))
	}

	if processor.handlers["test-type"] != handler {
		t.Error("Expected handler to be registered correctly")
	}
}

func TestAsyncProcessorJobSubmission(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultAsyncConfig()
	config.WorkerCount = 1
	config.QueueSize = 10
	processor := NewAsyncProcessor(config, logger)

	// Register handler
	handler := NewTestJobHandler("test-type", 0, false)
	processor.RegisterHandler(handler)

	// Start processor
	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	defer processor.Stop()

	// Submit job
	job := NewJob("test-job", "test-type", PriorityNormal, map[string]interface{}{
		"data": "test",
	})

	err = processor.SubmitJob(job)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Wait for job completion
	job.Wait()

	if job.Status != StatusCompleted {
		t.Errorf("Expected job status Completed, got %v", job.Status)
	}

	if handler.GetCallCount() != 1 {
		t.Errorf("Expected handler to be called once, got %d", handler.GetCallCount())
	}
}

func TestAsyncProcessorJobFailure(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultAsyncConfig()
	config.WorkerCount = 1
	config.MaxRetries = 1
	config.RetryDelay = 10 * time.Millisecond
	processor := NewAsyncProcessor(config, logger)

	// Register failing handler
	handler := NewTestJobHandler("test-type", 0, true)
	processor.RegisterHandler(handler)

	// Start processor
	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	defer processor.Stop()

	// Submit job
	job := NewJob("test-job", "test-type", PriorityNormal, nil)
	job.MaxRetries = 1

	err = processor.SubmitJob(job)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Wait for job completion
	job.Wait()

	if job.Status != StatusFailed {
		t.Errorf("Expected job status Failed, got %v", job.Status)
	}

	if job.Error == "" {
		t.Error("Expected job to have error message")
	}

	// Handler should be called MaxRetries + 1 times (initial + retries)
	expectedCalls := int32(2) // 1 initial + 1 retry
	if handler.GetCallCount() != expectedCalls {
		t.Errorf("Expected handler to be called %d times, got %d", expectedCalls, handler.GetCallCount())
	}
}

func TestAsyncProcessorJobPriority(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultAsyncConfig()
	config.WorkerCount = 1
	config.QueueSize = 100
	processor := NewAsyncProcessor(config, logger)

	// Register handler with delay to control execution order
	handler := NewTestJobHandler("test-type", 50*time.Millisecond, false)
	processor.RegisterHandler(handler)

	// Start processor
	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	defer processor.Stop()

	// Submit jobs with different priorities
	lowJob := NewJob("low-job", "test-type", PriorityLow, nil)
	normalJob := NewJob("normal-job", "test-type", PriorityNormal, nil)
	highJob := NewJob("high-job", "test-type", PriorityHigh, nil)
	criticalJob := NewJob("critical-job", "test-type", PriorityCritical, nil)

	// Submit in reverse priority order
	processor.SubmitJob(lowJob)
	processor.SubmitJob(normalJob)
	processor.SubmitJob(highJob)
	processor.SubmitJob(criticalJob)

	// Wait for all jobs to complete
	criticalJob.Wait()
	highJob.Wait()
	normalJob.Wait()
	lowJob.Wait()

	// All jobs should complete successfully
	jobs := []*Job{criticalJob, highJob, normalJob, lowJob}
	for _, job := range jobs {
		if job.Status != StatusCompleted {
			t.Errorf("Job %s expected status Completed, got %v", job.ID, job.Status)
		}
	}
}

func TestAsyncProcessorJobCancellation(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultAsyncConfig()
	config.WorkerCount = 1
	processor := NewAsyncProcessor(config, logger)

	// Register handler with delay
	handler := NewTestJobHandler("test-type", 100*time.Millisecond, false)
	processor.RegisterHandler(handler)

	// Start processor
	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	defer processor.Stop()

	// Submit job
	job := NewJob("test-job", "test-type", PriorityNormal, nil)
	err = processor.SubmitJob(job)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Cancel job immediately
	err = processor.CancelJob(job.ID)
	if err != nil {
		t.Fatalf("Failed to cancel job: %v", err)
	}

	// Wait for job to be cancelled
	job.Wait()

	if job.Status != StatusCancelled {
		t.Errorf("Expected job status Cancelled, got %v", job.Status)
	}
}

func TestAsyncProcessorJobRetrieval(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultAsyncConfig()
	processor := NewAsyncProcessor(config, logger)

	// Submit job without starting processor
	job := NewJob("test-job", "test-type", PriorityNormal, nil)
	processor.jobs[job.ID] = job

	// Test GetJob
	retrievedJob, exists := processor.GetJob(job.ID)
	if !exists {
		t.Error("Expected job to exist")
	}
	if retrievedJob != job {
		t.Error("Expected retrieved job to match original")
	}

	// Test GetJob with non-existent ID
	_, exists = processor.GetJob("non-existent")
	if exists {
		t.Error("Expected job to not exist")
	}

	// Test GetJobsByStatus
	jobs := processor.GetJobsByStatus(StatusPending)
	if len(jobs) != 1 {
		t.Errorf("Expected 1 pending job, got %d", len(jobs))
	}
	if jobs[0] != job {
		t.Error("Expected retrieved job to match original")
	}
}

func TestAsyncProcessorMetrics(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultAsyncConfig()
	config.MetricsInterval = 10 * time.Millisecond
	processor := NewAsyncProcessor(config, logger)

	// Start processor
	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	defer processor.Stop()

	// Wait for metrics to be updated
	time.Sleep(50 * time.Millisecond)

	metrics := processor.GetMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be available")
	}

	if metrics.LastUpdated.IsZero() {
		t.Error("Expected metrics to be updated")
	}
}

func TestAsyncProcessorStatus(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultAsyncConfig()
	config.WorkerCount = 2
	processor := NewAsyncProcessor(config, logger)

	// Register handler
	handler := NewTestJobHandler("test-type", 0, false)
	processor.RegisterHandler(handler)

	status := processor.GetStatus()

	// Check workers
	workers, ok := status["workers"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected workers status to be present")
	}
	if workers["total"].(int) != 2 {
		t.Errorf("Expected 2 total workers, got %v", workers["total"])
	}

	// Check handlers
	if status["handlers"].(int) != 1 {
		t.Errorf("Expected 1 handler, got %v", status["handlers"])
	}
}

func TestAsyncProcessorUnknownJobType(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultAsyncConfig()
	processor := NewAsyncProcessor(config, logger)

	// Submit job without registering handler
	job := NewJob("test-job", "unknown-type", PriorityNormal, nil)
	err := processor.SubmitJob(job)

	if err == nil {
		t.Error("Expected error when submitting job with unknown type")
	}

	expectedError := "no handler registered for job type: unknown-type"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// Benchmark tests
func BenchmarkJobCreation(b *testing.B) {
	payload := map[string]interface{}{
		"data": "test",
		"num":  123,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job := NewJob("test-job", "test", PriorityNormal, payload)
		_ = job
	}
}

func BenchmarkAsyncProcessorJobSubmission(b *testing.B) {
	logger := log.New(os.Stdout, "[BENCH] ", log.LstdFlags)
	config := DefaultAsyncConfig()
	config.WorkerCount = 4
	processor := NewAsyncProcessor(config, logger)

	handler := NewTestJobHandler("test-type", 0, false)
	processor.RegisterHandler(handler)

	processor.Start()
	defer processor.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job := NewJob(fmt.Sprintf("job-%d", i), "test-type", PriorityNormal, nil)
		processor.SubmitJob(job)
	}
}