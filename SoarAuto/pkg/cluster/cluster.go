package cluster

import (
	"fmt"
	"sync"
	"time"

	"SoarAuto/pkg/types"
)

// ClusterManager implements cluster functionality for distributed job processing
type ClusterManager struct {
	nodeID    string
	nodes     map[string]*ClusterNode
	jobQueue  chan *types.Job
	jobManager types.ClusterManager
	mutex     sync.RWMutex
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
func NewClusterManager(nodeID string) *ClusterManager {
	return &ClusterManager{
		nodeID:   nodeID,
		nodes:    make(map[string]*ClusterNode),
		jobQueue: make(chan *types.Job, 1000), // Buffer for 1000 jobs
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
	// Create a job (this would normally use the job manager)
	job := &types.Job{
		ID:       generateJobID(),
		Status:   "queued",
		Playbook: playbook,
		Context:  context,
	}

	// Add to queue (non-blocking)
	select {
	case cm.jobQueue <- job:
		return job.ID, nil
	default:
		return "", fmt.Errorf("job queue is full")
	}
}

// GetJob retrieves a job by ID (cluster implementation)
func (cm *ClusterManager) GetJob(jobID string) (*types.Job, error) {
	// In a real implementation, this would check all nodes in the cluster
	// For now, return a basic response
	return &types.Job{
		ID:     jobID,
		Status: "not_found",
	}, fmt.Errorf("job not found in cluster: %s", jobID)
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

// ProcessJobQueue processes jobs from the queue
func (cm *ClusterManager) ProcessJobQueue() {
	for job := range cm.jobQueue {
		// Find available node
		node := cm.GetAvailableNode()
		if node == nil {
			// No available nodes, put job back in queue
			// In a real implementation, this would handle this more gracefully
			continue
		}

		// Execute job on node (simplified)
		go cm.executeJobOnNode(job, node)
	}
}

// executeJobOnNode executes a job on a specific node
func (cm *ClusterManager) executeJobOnNode(job *types.Job, node *ClusterNode) {
	// Update node status
	cm.mutex.Lock()
	node.JobsRunning++
	cm.mutex.Unlock()

	// Simulate job execution
	time.Sleep(1 * time.Second)

	// Update job status
	job.Status = "completed"

	// Update node status
	cm.mutex.Lock()
	node.JobsRunning--
	node.JobsComplete++
	cm.mutex.Unlock()
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