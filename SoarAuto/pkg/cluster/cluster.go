package cluster

import (
	"fmt"
	"sync"
	"time"

	"SoarAuto/pkg/types"
)

// JobManagerInterface defines the interface for job management
type JobManagerInterface interface {
	CreateJob(playbook interface{}, context map[string]interface{}) (*types.Job, error)
	GetJob(jobID string) (*types.Job, error)
	UpdateJobStatus(jobID string, status string) error
	UpdateJobResults(jobID string, results interface{}, errorMsg string) error
}

// JobExecutorInterface defines the interface for job execution
type JobExecutorInterface interface {
	ExecutePlaybook(playbook interface{}, context map[string]interface{}) (interface{}, error)
}

// LoggerInterface defines the interface for logging
type LoggerInterface interface {
	Info(message string, fields map[string]interface{})
	Error(message string, fields map[string]interface{})
	Debug(message string, fields map[string]interface{})
}

// ClusterManager implements cluster functionality for distributed job processing
type ClusterManager struct {
	nodeID      string
	nodes       map[string]*ClusterNode
	jobQueue    chan *types.Job
	jobManager  JobManagerInterface
	jobExecutor JobExecutorInterface
	logger      LoggerInterface
	mutex       sync.RWMutex
	running     bool
	workers     int
}

// ClusterNode represents a node in the cluster
type ClusterNode struct {
	ID           string    `json:"id"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	Status       string    `json:"status"`
	LastSeen     time.Time `json:"last_seen"`
	JobsRunning  int       `json:"jobs_running"`
	JobsComplete int       `json:"jobs_complete"`
	LoadAverage  float64   `json:"load_average"`
}

// ClusterInfo represents cluster information
type ClusterInfo struct {
	NodeID       string                 `json:"node_id"`
	TotalNodes   int                    `json:"total_nodes"`
	ActiveNodes  int                    `json:"active_nodes"`
	TotalJobs    int                    `json:"total_jobs"`
	RunningJobs  int                    `json:"running_jobs"`
	QueuedJobs   int                    `json:"queued_jobs"`
	Nodes        []*ClusterNode         `json:"nodes"`
	Timestamp    string                 `json:"timestamp"`
	Status       string                 `json:"status"`
	Config       map[string]interface{} `json:"config"`
}

// NewClusterManager creates a new cluster manager
func NewClusterManager(nodeID string, jobManager JobManagerInterface, jobExecutor JobExecutorInterface, logger LoggerInterface) *ClusterManager {
	return &ClusterManager{
		nodeID:      nodeID,
		nodes:       make(map[string]*ClusterNode),
		jobQueue:    make(chan *types.Job, 1000), // Buffer for 1000 jobs
		jobManager:  jobManager,
		jobExecutor: jobExecutor,
		logger:      logger,
		workers:     4, // Default to 4 worker goroutines
		running:     false,
	}
}

// GetClusterInfo returns information about the cluster
func (cm *ClusterManager) GetClusterInfo() interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var nodes []*ClusterNode
	activeNodes := 0
	runningJobs := 0

	for _, node := range cm.nodes {
		nodes = append(nodes, node)
		if node.Status == "active" {
			activeNodes++
		}
		runningJobs += node.JobsRunning
	}

	// Add current node if not in list
	if _, exists := cm.nodes[cm.nodeID]; !exists {
		currentNode := &ClusterNode{
			ID:          cm.nodeID,
			Host:        "localhost",
			Port:        9090,
			Status:      "active",
			LastSeen:    time.Now(),
			JobsRunning: 0,
		}
		nodes = append(nodes, currentNode)
		activeNodes++
	}

	return &ClusterInfo{
		NodeID:      cm.nodeID,
		TotalNodes:  len(nodes),
		ActiveNodes: activeNodes,
		RunningJobs: runningJobs,
		QueuedJobs:  len(cm.jobQueue),
		Nodes:       nodes,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Status:      "active",
		Config: map[string]interface{}{
			"max_queue_size": 1000,
			"node_timeout":   300, // 5 minutes
			"load_balancer":  "round_robin",
		},
	}
}

// SubmitJob submits a job to the distributed queue
func (cm *ClusterManager) SubmitJob(playbook interface{}, context map[string]interface{}) (string, error) {
	// Create a job using the job manager
	job, err := cm.jobManager.CreateJob(playbook, context)
	if err != nil {
		return "", fmt.Errorf("failed to create job: %v", err)
	}

	// Set status to queued for cluster processing
	if err := cm.jobManager.UpdateJobStatus(job.ID, "queued"); err != nil {
		cm.logger.Error("Failed to update job status to queued", map[string]interface{}{
			"component": "cluster",
			"job_id":    job.ID,
			"error":     err.Error(),
		})
	}

	// Add to queue (non-blocking)
	select {
	case cm.jobQueue <- job:
		cm.logger.Info("Job submitted to cluster queue", map[string]interface{}{
			"component": "cluster",
			"job_id":    job.ID,
			"node_id":   cm.nodeID,
		})
		return job.ID, nil
	default:
		// Queue is full, update job status to failed
		cm.jobManager.UpdateJobResults(job.ID, nil, "Cluster queue is full")
		return "", fmt.Errorf("job queue is full")
	}
}

// GetJob retrieves a job by ID (cluster implementation)
func (cm *ClusterManager) GetJob(jobID string) (*types.Job, error) {
	// Get job from job manager
	job, err := cm.jobManager.GetJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("job not found in cluster: %s", jobID)
	}

	return job, nil
}

// RegisterNode registers a new node in the cluster
func (cm *ClusterManager) RegisterNode(node *ClusterNode) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	node.LastSeen = time.Now()
	cm.nodes[node.ID] = node
}

// UpdateNodeStatus updates the status of a cluster node
func (cm *ClusterManager) UpdateNodeStatus(nodeID string, status string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if node, exists := cm.nodes[nodeID]; exists {
		node.Status = status
		node.LastSeen = time.Now()
	}
}

// RemoveInactiveNodes removes nodes that haven't been seen recently
func (cm *ClusterManager) RemoveInactiveNodes(timeout time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cutoff := time.Now().Add(-timeout)
	for nodeID, node := range cm.nodes {
		if node.LastSeen.Before(cutoff) {
			delete(cm.nodes, nodeID)
		}
	}
}

// GetAvailableNode returns the best available node for job execution
func (cm *ClusterManager) GetAvailableNode() *ClusterNode {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var bestNode *ClusterNode
	minLoad := float64(1000) // Start with high value

	for _, node := range cm.nodes {
		if node.Status == "active" && node.LoadAverage < minLoad {
			bestNode = node
			minLoad = node.LoadAverage
		}
	}

	return bestNode
}

// ProcessJobQueue processes jobs from the queue with multiple workers
func (cm *ClusterManager) ProcessJobQueue() {
	if cm.running {
		return
	}
	
	cm.running = true
	
	// Start multiple worker goroutines
	for i := 0; i < cm.workers; i++ {
		go cm.worker(i)
	}
	
	cm.logger.Info("Cluster job queue processing started", map[string]interface{}{
		"component": "cluster",
		"node_id":   cm.nodeID,
		"workers":   cm.workers,
	})
}

// worker is a worker goroutine that processes jobs from the queue
func (cm *ClusterManager) worker(workerID int) {
	cm.logger.Info("Cluster worker started", map[string]interface{}{
		"component": "cluster",
		"worker_id": workerID,
		"node_id":   cm.nodeID,
	})
	
	for job := range cm.jobQueue {
		cm.executeJob(job, workerID)
	}
}

// executeJob executes a job using the job executor
func (cm *ClusterManager) executeJob(job *types.Job, workerID int) {
	cm.logger.Info("Starting job execution", map[string]interface{}{
		"component": "cluster",
		"job_id":    job.ID,
		"worker_id": workerID,
		"node_id":   cm.nodeID,
	})

	// Update job status to running
	if err := cm.jobManager.UpdateJobStatus(job.ID, "running"); err != nil {
		cm.logger.Error("Failed to update job status to running", map[string]interface{}{
			"component": "cluster",
			"job_id":    job.ID,
			"error":     err.Error(),
		})
	}

	// Update node statistics
	cm.updateNodeStats(1, 0) // +1 running job

	startTime := time.Now()

	// Execute the playbook using the job executor
	result, err := cm.jobExecutor.ExecutePlaybook(job.Playbook, job.Context)
	
	executionTime := time.Since(startTime)

	// Update node statistics
	cm.updateNodeStats(-1, 1) // -1 running, +1 completed

	if err != nil {
		// Job failed
		cm.logger.Error("Job execution failed", map[string]interface{}{
			"component":      "cluster",
			"job_id":         job.ID,
			"worker_id":      workerID,
			"error":          err.Error(),
			"execution_time": executionTime.Milliseconds(),
		})

		// Update job with error
		if updateErr := cm.jobManager.UpdateJobResults(job.ID, nil, err.Error()); updateErr != nil {
			cm.logger.Error("Failed to update job results", map[string]interface{}{
				"component": "cluster",
				"job_id":    job.ID,
				"error":     updateErr.Error(),
			})
		}
	} else {
		// Job succeeded
		cm.logger.Info("Job execution completed successfully", map[string]interface{}{
			"component":      "cluster",
			"job_id":         job.ID,
			"worker_id":      workerID,
			"execution_time": executionTime.Milliseconds(),
		})

		// Update job with results
		if updateErr := cm.jobManager.UpdateJobResults(job.ID, result, ""); updateErr != nil {
			cm.logger.Error("Failed to update job results", map[string]interface{}{
				"component": "cluster",
				"job_id":    job.ID,
				"error":     updateErr.Error(),
			})
		}
	}
}

// updateNodeStats updates the current node's job statistics
func (cm *ClusterManager) updateNodeStats(runningDelta, completedDelta int) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Update current node stats (create if doesn't exist)
	if node, exists := cm.nodes[cm.nodeID]; exists {
		node.JobsRunning += runningDelta
		node.JobsComplete += completedDelta
		node.LastSeen = time.Now()
	} else {
		// Create current node entry
		cm.nodes[cm.nodeID] = &ClusterNode{
			ID:           cm.nodeID,
			Host:         "localhost",
			Port:         9090,
			Status:       "active",
			LastSeen:     time.Now(),
			JobsRunning:  max(0, runningDelta),
			JobsComplete: max(0, completedDelta),
			LoadAverage:  0.0,
		}
	}
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// generateJobID generates a unique job ID
func generateJobID() string {
	// Simple timestamp-based ID for now
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}

// StartClusterServices starts background cluster services
func (cm *ClusterManager) StartClusterServices() {
	// Start job queue processor
	go cm.ProcessJobQueue()

	// Start node cleanup service
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			cm.RemoveInactiveNodes(10 * time.Minute)
		}
	}()
}

// Stop stops the cluster manager
func (cm *ClusterManager) Stop() {
	close(cm.jobQueue)
}