package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// NodeStatus represents the status of a cluster node
type NodeStatus string

const (
	NodeStatusActive   NodeStatus = "active"
	NodeStatusInactive NodeStatus = "inactive"
	NodeStatusStarting NodeStatus = "starting"
	NodeStatusStopping NodeStatus = "stopping"
)

// NodeInfo contains information about a cluster node
type NodeInfo struct {
	ID            string     `json:"id"`
	Host          string     `json:"host"`
	Port          int        `json:"port"`
	Status        NodeStatus `json:"status"`
	StartedAt     time.Time  `json:"started_at"`
	LastSeen      time.Time  `json:"last_seen"`
	Capabilities  []string   `json:"capabilities"`
	Load          float64    `json:"load"`
	JobsRunning   int        `json:"jobs_running"`
	JobsCompleted int        `json:"jobs_completed"`
	JobsFailed    int        `json:"jobs_failed"`
}

// ClusterConfig is defined in config.go

// DistributedJob represents a job in the distributed queue
type DistributedJob struct {
	ID          string                 `json:"id"`
	Playbook    []interface{}          `json:"playbook"`
	Context     map[string]interface{} `json:"context"`
	Status      string                 `json:"status"`
	SubmittedAt time.Time              `json:"submitted_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	AssignedTo  string                 `json:"assigned_to,omitempty"`
	Results     []interface{}          `json:"results,omitempty"`
	Error       string                 `json:"error,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	Priority    int                    `json:"priority"`
	Tags        []string               `json:"tags"`
}

// ClusterManager manages the distributed cluster
type ClusterManager struct {
	config        *ClusterConfig
	redisClient   *redis.Client
	nodeInfo      *NodeInfo
	nodes         map[string]*NodeInfo
	mutex         sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	jobQueue      *DistributedJobQueue
	healthChecker *HealthChecker
	logger        *StructuredLogger
	server        *SecAutoServer
}

// DistributedJobQueue manages the Redis-based job queue
type DistributedJobQueue struct {
	redisClient *redis.Client
	queueName   string
	ctx         context.Context
	logger      *StructuredLogger
}

// HealthChecker monitors cluster node health
type HealthChecker struct {
	clusterManager *ClusterManager
	interval       time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewClusterManager creates a new cluster manager
func NewClusterManager(config *ClusterConfig, server *SecAutoServer) (*ClusterManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		cancel() // Cancel context on error
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	// Generate node ID if not provided
	if config.NodeID == "" {
		config.NodeID = uuid.New().String()
	}

	// Parse port from string to int
	port, err := strconv.Atoi(server.port)
	if err != nil {
		port = 8000 // Default port if parsing fails
	}

	// Create node info
	nodeInfo := &NodeInfo{
		ID:            config.NodeID,
		Host:          getLocalIP(),
		Port:          port,
		Status:        NodeStatusStarting,
		StartedAt:     time.Now(),
		LastSeen:      time.Now(),
		Capabilities:  []string{"playbook_execution", "job_processing"},
		Load:          0.0,
		JobsRunning:   0,
		JobsCompleted: 0,
		JobsFailed:    0,
	}

	clusterManager := &ClusterManager{
		config:      config,
		redisClient: redisClient,
		nodeInfo:    nodeInfo,
		nodes:       make(map[string]*NodeInfo),
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger,
		server:      server,
	}

	// Initialize job queue
	jobQueue := &DistributedJobQueue{
		redisClient: redisClient,
		queueName:   fmt.Sprintf("secauto:jobs:%s", config.ClusterName),
		ctx:         ctx,
		logger:      logger,
	}
	clusterManager.jobQueue = jobQueue

	// Initialize health checker with its own context
	healthCtx, healthCancel := context.WithCancel(ctx)
	healthChecker := &HealthChecker{
		clusterManager: clusterManager,
		interval:       time.Duration(config.HeartbeatInterval) * time.Second,
		ctx:            healthCtx,
		cancel:         healthCancel,
	}
	clusterManager.healthChecker = healthChecker

	// Start cluster services
	if err := clusterManager.start(); err != nil {
		return nil, fmt.Errorf("failed to start cluster: %v", err)
	}

	return clusterManager, nil
}

// start initializes and starts the cluster
func (cm *ClusterManager) start() error {
	// Register this node
	if err := cm.registerNode(); err != nil {
		return fmt.Errorf("failed to register node: %v", err)
	}

	// Start health checker
	go cm.healthChecker.start()

	// Start job processor
	go cm.startJobProcessor()

	// Start node discovery
	go cm.startNodeDiscovery()

	cm.logger.Info("Cluster manager started", map[string]interface{}{
		"component": "cluster_manager",
		"node_id":   cm.nodeInfo.ID,
		"cluster":   cm.config.ClusterName,
		"redis_url": cm.config.RedisURL,
	})

	return nil
}

// registerNode registers this node with the cluster
func (cm *ClusterManager) registerNode() error {
	nodeData, err := json.Marshal(cm.nodeInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal node info: %v", err)
	}

	key := fmt.Sprintf("secauto:nodes:%s:%s", cm.config.ClusterName, cm.nodeInfo.ID)
	err = cm.redisClient.Set(cm.ctx, key, nodeData, time.Duration(cm.config.HeartbeatInterval*3)*time.Second).Err()
	if err != nil {
		return fmt.Errorf("failed to register node: %v", err)
	}

	cm.nodeInfo.Status = NodeStatusActive
	return nil
}

// startJobProcessor starts the distributed job processor
func (cm *ClusterManager) startJobProcessor() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			// Try to get a job from the queue
			job, err := cm.jobQueue.dequeueJob()
			if err != nil {
				cm.logger.Error("Failed to dequeue job", map[string]interface{}{
					"component": "cluster_manager",
					"error":     err.Error(),
				})
				continue
			}

			if job != nil {
				// Process the job
				go cm.processJob(job)
			}
		}
	}
}

// processJob processes a distributed job
func (cm *ClusterManager) processJob(job *DistributedJob) {
	cm.logger.Info("Processing distributed job", map[string]interface{}{
		"component": "cluster_manager",
		"job_id":    job.ID,
		"node_id":   cm.nodeInfo.ID,
	})

	// Update job status
	now := time.Now()
	job.AssignedTo = cm.nodeInfo.ID
	job.StartedAt = &now
	job.Status = "running"

	// Update node stats
	cm.mutex.Lock()
	cm.nodeInfo.JobsRunning++
	cm.mutex.Unlock()

	// Execute the job using the existing engine
	results, err := cm.server.engine.EvaluatePlaybook(job.Playbook)

	// Update job with results
	job.CompletedAt = &now
	cm.nodeInfo.JobsRunning--

	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		cm.nodeInfo.JobsFailed++
		cm.logger.Error("Job execution failed", map[string]interface{}{
			"component": "cluster_manager",
			"job_id":    job.ID,
			"error":     err.Error(),
		})
	} else {
		job.Status = "completed"
		job.Results = results
		cm.nodeInfo.JobsCompleted++
		cm.logger.Info("Job completed successfully", map[string]interface{}{
			"component": "cluster_manager",
			"job_id":    job.ID,
		})
	}

	// Update job in Redis
	if err := cm.jobQueue.updateJob(job); err != nil {
		cm.logger.Error("Failed to update job", map[string]interface{}{
			"component": "cluster_manager",
			"job_id":    job.ID,
			"error":     err.Error(),
		})
	}
}

// startNodeDiscovery starts the node discovery process
func (cm *ClusterManager) startNodeDiscovery() {
	ticker := time.NewTicker(time.Duration(cm.config.HeartbeatInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.discoverNodes()
		}
	}
}

// discoverNodes discovers other nodes in the cluster
func (cm *ClusterManager) discoverNodes() {
	pattern := fmt.Sprintf("secauto:nodes:%s:*", cm.config.ClusterName)
	keys, err := cm.redisClient.Keys(cm.ctx, pattern).Result()
	if err != nil {
		cm.logger.Error("Failed to discover nodes", map[string]interface{}{
			"component": "cluster_manager",
			"error":     err.Error(),
		})
		return
	}

	newNodes := make(map[string]*NodeInfo)
	for _, key := range keys {
		nodeData, err := cm.redisClient.Get(cm.ctx, key).Result()
		if err != nil {
			continue
		}

		var nodeInfo NodeInfo
		if err := json.Unmarshal([]byte(nodeData), &nodeInfo); err != nil {
			continue
		}

		// Skip our own node
		if nodeInfo.ID == cm.nodeInfo.ID {
			continue
		}

		newNodes[nodeInfo.ID] = &nodeInfo
	}

	// Update nodes list
	cm.mutex.Lock()
	cm.nodes = newNodes
	cm.mutex.Unlock()

	cm.logger.Debug("Node discovery completed", map[string]interface{}{
		"component": "cluster_manager",
		"nodes":     len(newNodes),
	})
}

// SubmitJob submits a job to the distributed queue
func (cm *ClusterManager) SubmitJob(playbook []interface{}, context map[string]interface{}) (string, error) {
	job := &DistributedJob{
		ID:          uuid.New().String(),
		Playbook:    playbook,
		Context:     context,
		Status:      "pending",
		SubmittedAt: time.Now(),
		RetryCount:  0,
		Priority:    0,
		Tags:        []string{},
	}

	if err := cm.jobQueue.enqueueJob(job); err != nil {
		return "", fmt.Errorf("failed to submit job: %v", err)
	}

	cm.logger.Info("Job submitted to distributed queue", map[string]interface{}{
		"component": "cluster_manager",
		"job_id":    job.ID,
	})

	return job.ID, nil
}

// GetJob retrieves a job from the distributed queue
func (cm *ClusterManager) GetJob(jobID string) (*DistributedJob, error) {
	return cm.jobQueue.getJob(jobID)
}

// GetClusterInfo returns information about the cluster
func (cm *ClusterManager) GetClusterInfo() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	nodes := make([]*NodeInfo, 0, len(cm.nodes)+1)
	nodes = append(nodes, cm.nodeInfo)

	for _, node := range cm.nodes {
		nodes = append(nodes, node)
	}

	return map[string]interface{}{
		"cluster_name": cm.config.ClusterName,
		"node_id":      cm.nodeInfo.ID,
		"total_nodes":  len(nodes),
		"nodes":        nodes,
		"queue_size":   cm.jobQueue.getQueueSize(),
	}
}

// Close shuts down the cluster manager
func (cm *ClusterManager) Close() error {
	cm.cancel()

	// Stop health checker
	if cm.healthChecker != nil {
		cm.healthChecker.stop()
	}

	// Deregister node
	key := fmt.Sprintf("secauto:nodes:%s:%s", cm.config.ClusterName, cm.nodeInfo.ID)
	cm.redisClient.Del(cm.ctx, key)

	// Close Redis connection
	if err := cm.redisClient.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %v", err)
	}

	cm.logger.Info("Cluster manager stopped", map[string]interface{}{
		"component": "cluster_manager",
		"node_id":   cm.nodeInfo.ID,
	})

	return nil
}

// DistributedJobQueue methods

func (dq *DistributedJobQueue) enqueueJob(job *DistributedJob) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %v", err)
	}

	// Add to queue with priority
	score := float64(time.Now().Unix()) + float64(job.Priority)
	err = dq.redisClient.ZAdd(dq.ctx, dq.queueName, &redis.Z{
		Score:  score,
		Member: job.ID,
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to add job to queue: %v", err)
	}

	// Store job data
	jobKey := fmt.Sprintf("secauto:job:%s", job.ID)
	err = dq.redisClient.Set(dq.ctx, jobKey, jobData, time.Duration(24*time.Hour)).Err()
	if err != nil {
		return fmt.Errorf("failed to store job data: %v", err)
	}

	return nil
}

func (dq *DistributedJobQueue) dequeueJob() (*DistributedJob, error) {
	// Get job with highest priority (lowest score)
	result, err := dq.redisClient.ZPopMin(dq.ctx, dq.queueName).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue job: %v", err)
	}

	if len(result) == 0 {
		return nil, nil // No jobs in queue
	}

	jobID := result[0].Member.(string)
	return dq.getJob(jobID)
}

func (dq *DistributedJobQueue) getJob(jobID string) (*DistributedJob, error) {
	jobKey := fmt.Sprintf("secauto:job:%s", jobID)
	jobData, err := dq.redisClient.Get(dq.ctx, jobKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get job data: %v", err)
	}

	var job DistributedJob
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %v", err)
	}

	return &job, nil
}

func (dq *DistributedJobQueue) updateJob(job *DistributedJob) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %v", err)
	}

	jobKey := fmt.Sprintf("secauto:job:%s", job.ID)
	err = dq.redisClient.Set(dq.ctx, jobKey, jobData, time.Duration(24*time.Hour)).Err()
	if err != nil {
		return fmt.Errorf("failed to update job: %v", err)
	}

	return nil
}

func (dq *DistributedJobQueue) getQueueSize() int64 {
	size, err := dq.redisClient.ZCard(dq.ctx, dq.queueName).Result()
	if err != nil {
		return 0
	}
	return size
}

// HealthChecker methods

func (hc *HealthChecker) start() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-ticker.C:
			hc.updateHeartbeat()
		}
	}
}

func (hc *HealthChecker) stop() {
	if hc.cancel != nil {
		hc.cancel()
	}
}

func (hc *HealthChecker) updateHeartbeat() {
	cm := hc.clusterManager

	// Update node info
	cm.mutex.Lock()
	cm.nodeInfo.LastSeen = time.Now()
	cm.nodeInfo.Load = cm.calculateLoad()
	cm.mutex.Unlock()

	// Update node registration
	if err := cm.registerNode(); err != nil {
		cm.logger.Error("Failed to update heartbeat", map[string]interface{}{
			"component": "cluster_manager",
			"error":     err.Error(),
		})
	}
}

func (cm *ClusterManager) calculateLoad() float64 {
	// Simple load calculation based on running jobs
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if cm.nodeInfo.JobsRunning == 0 {
		return 0.0
	}

	// Calculate load as percentage of capacity (assuming max 10 concurrent jobs)
	return float64(cm.nodeInfo.JobsRunning) / 10.0
}

// Helper function to get local IP
func getLocalIP() string {
	// Simple implementation - in production, you'd want more robust IP detection
	return "localhost"
}
