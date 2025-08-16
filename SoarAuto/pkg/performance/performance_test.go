package performance

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

func TestDefaultPerformanceConfig(t *testing.T) {
	config := DefaultPerformanceConfig()

	if config == nil {
		t.Fatal("Expected config to be created")
	}

	if config.Cache == nil {
		t.Error("Expected cache config to be set")
	}

	if config.Pool == nil {
		t.Error("Expected pool config to be set")
	}

	if config.Async == nil {
		t.Error("Expected async config to be set")
	}

	if config.Profiler == nil {
		t.Error("Expected profiler config to be set")
	}
}

func TestPerformanceManagerCreation(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	
	// Disable components that require external dependencies for testing
	config.Cache.Enabled = false
	config.Profiler.Enabled = false

	pm, err := NewPerformanceManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager: %v", err)
	}

	if pm == nil {
		t.Fatal("Expected performance manager to be created")
	}

	if pm.config != config {
		t.Error("Expected config to be set")
	}

	if pm.logger != logger {
		t.Error("Expected logger to be set")
	}

	if pm.cache == nil {
		t.Error("Expected cache to be initialized")
	}

	if pm.pool == nil {
		t.Error("Expected pool manager to be initialized")
	}

	if pm.async == nil {
		t.Error("Expected async processor to be initialized")
	}

	if pm.profiler == nil {
		t.Error("Expected profiler to be initialized")
	}
}

func TestPerformanceManagerWithNilConfig(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	pm, err := NewPerformanceManager(nil, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager with nil config: %v", err)
	}

	if pm.config == nil {
		t.Error("Expected default config to be used")
	}
}

func TestPerformanceManagerStartStop(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	
	// Disable components that require external dependencies
	config.Cache.Enabled = false
	config.Profiler.Enabled = false
	config.Pool.MonitoringEnabled = false

	pm, err := NewPerformanceManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager: %v", err)
	}

	// Test initial state
	if pm.IsStarted() {
		t.Error("Expected performance manager to not be started initially")
	}

	// Start performance manager
	err = pm.Start()
	if err != nil {
		t.Fatalf("Failed to start performance manager: %v", err)
	}

	if !pm.IsStarted() {
		t.Error("Expected performance manager to be started")
	}

	// Test double start
	err = pm.Start()
	if err == nil {
		t.Error("Expected error when starting already started performance manager")
	}

	// Stop performance manager
	err = pm.Stop()
	if err != nil {
		t.Errorf("Failed to stop performance manager: %v", err)
	}

	if pm.IsStarted() {
		t.Error("Expected performance manager to be stopped")
	}

	// Test double stop
	err = pm.Stop()
	if err != nil {
		t.Errorf("Stop should not error when already stopped: %v", err)
	}
}

func TestPerformanceManagerComponentAccess(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	config.Cache.Enabled = false
	config.Profiler.Enabled = false

	pm, err := NewPerformanceManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager: %v", err)
	}

	// Test component access
	cache := pm.GetCache()
	if cache == nil {
		t.Error("Expected cache to be accessible")
	}

	pool := pm.GetPoolManager()
	if pool == nil {
		t.Error("Expected pool manager to be accessible")
	}

	async := pm.GetAsyncProcessor()
	if async == nil {
		t.Error("Expected async processor to be accessible")
	}

	profiler := pm.GetProfiler()
	if profiler == nil {
		t.Error("Expected profiler to be accessible")
	}
}

func TestPerformanceManagerHealthCheck(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	config.Cache.Enabled = false
	config.Profiler.Enabled = false
	config.Pool.MonitoringEnabled = false

	pm, err := NewPerformanceManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager: %v", err)
	}

	ctx := context.Background()

	// Health check before start should fail
	err = pm.HealthCheck(ctx)
	if err == nil {
		t.Error("Expected health check to fail before start")
	}

	// Start and check health
	err = pm.Start()
	if err != nil {
		t.Fatalf("Failed to start performance manager: %v", err)
	}
	defer pm.Stop()

	err = pm.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check should pass after start: %v", err)
	}
}

func TestPerformanceManagerStatus(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	config.Cache.Enabled = false
	config.Profiler.Enabled = false

	pm, err := NewPerformanceManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager: %v", err)
	}

	status := pm.GetStatus()

	// Check that status contains expected keys
	expectedKeys := []string{"started", "cache", "pool", "async", "profiler"}
	for _, key := range expectedKeys {
		if _, exists := status[key]; !exists {
			t.Errorf("Expected status to contain key: %s", key)
		}
	}

	// Check started status
	if started, ok := status["started"].(bool); ok && started {
		t.Error("Expected started to be false initially")
	}
}

func TestPerformanceManagerMetrics(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	config.Cache.Enabled = false
	config.Profiler.Enabled = false

	pm, err := NewPerformanceManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager: %v", err)
	}

	metrics := pm.GetMetrics()

	// Check that metrics contains expected keys
	expectedKeys := []string{"timestamp", "cache", "pool", "async", "runtime"}
	for _, key := range expectedKeys {
		if _, exists := metrics[key]; !exists {
			t.Errorf("Expected metrics to contain key: %s", key)
		}
	}

	// Check timestamp
	if timestamp, ok := metrics["timestamp"].(time.Time); ok && timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestPerformanceManagerDetailedMetrics(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	config.Cache.Enabled = false
	config.Profiler.Enabled = false

	pm, err := NewPerformanceManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager: %v", err)
	}

	detailedMetrics := pm.GetDetailedMetrics()

	// Check that detailed metrics contains expected keys
	expectedKeys := []string{"timestamp", "cache", "pool", "async", "memory", "gc", "runtime"}
	for _, key := range expectedKeys {
		if _, exists := detailedMetrics[key]; !exists {
			t.Errorf("Expected detailed metrics to contain key: %s", key)
		}
	}
}

func TestPerformanceManagerLoadOptimization(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	config.Cache.Enabled = false
	config.Profiler.Enabled = false
	config.Pool.MonitoringEnabled = false

	pm, err := NewPerformanceManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager: %v", err)
	}

	err = pm.Start()
	if err != nil {
		t.Fatalf("Failed to start performance manager: %v", err)
	}
	defer pm.Stop()

	// Test load optimization
	loadMetrics := map[string]float64{
		"database": 0.8,
		"cpu":      0.6,
		"memory":   0.7,
	}

	// Should not panic or error
	pm.OptimizeForLoad(loadMetrics)
}

func TestPerformanceManagerConfigUpdate(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	config.Cache.Enabled = false
	config.Profiler.Enabled = false

	pm, err := NewPerformanceManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager: %v", err)
	}

	// Test config update when stopped
	newConfig := DefaultPerformanceConfig()
	newConfig.Cache.Enabled = false
	newConfig.Profiler.Enabled = false
	newConfig.Async.WorkerCount = 8

	err = pm.UpdateConfig(newConfig)
	if err != nil {
		t.Errorf("Failed to update config when stopped: %v", err)
	}

	if pm.config.Async.WorkerCount != 8 {
		t.Error("Expected config to be updated")
	}

	// Test config update when started
	err = pm.Start()
	if err != nil {
		t.Fatalf("Failed to start performance manager: %v", err)
	}
	defer pm.Stop()

	err = pm.UpdateConfig(newConfig)
	if err == nil {
		t.Error("Expected error when updating config while started")
	}
}

func TestPerformanceManagerForceGC(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	config.Cache.Enabled = false

	pm, err := NewPerformanceManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager: %v", err)
	}

	// Should not panic
	pm.ForceGarbageCollection()
}

func TestPerformanceManagerGetConfig(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	config.Cache.Enabled = false
	config.Profiler.Enabled = false

	pm, err := NewPerformanceManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager: %v", err)
	}

	retrievedConfig := pm.GetConfig()
	if retrievedConfig != config {
		t.Error("Expected retrieved config to match original")
	}
}

// Integration test
func TestPerformanceManagerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	
	// Configure for testing
	config.Cache.Enabled = false
	config.Profiler.Enabled = false
	config.Pool.MonitoringEnabled = false
	config.Async.WorkerCount = 2
	config.Async.QueueSize = 10

	pm, err := NewPerformanceManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create performance manager: %v", err)
	}

	// Start performance manager
	err = pm.Start()
	if err != nil {
		t.Fatalf("Failed to start performance manager: %v", err)
	}
	defer pm.Stop()

	// Register a test job handler
	handler := NewTestJobHandler("integration-test", 10*time.Millisecond, false)
	pm.async.RegisterHandler(handler)

	// Submit some jobs
	for i := 0; i < 5; i++ {
		job := NewJob(fmt.Sprintf("test-job-%d", i), "integration-test", PriorityNormal, nil)
		err = pm.async.SubmitJob(job)
		if err != nil {
			t.Errorf("Failed to submit job %d: %v", i, err)
		}
	}

	// Wait a bit for jobs to process
	time.Sleep(200 * time.Millisecond)

	// Check metrics
	metrics := pm.GetMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be available")
	}

	// Check detailed metrics
	detailedMetrics := pm.GetDetailedMetrics()
	if detailedMetrics == nil {
		t.Error("Expected detailed metrics to be available")
	}

	// Check status
	status := pm.GetStatus()
	if status == nil {
		t.Error("Expected status to be available")
	}

	// Perform health check
	ctx := context.Background()
	err = pm.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

// Benchmark tests
func BenchmarkPerformanceManagerGetMetrics(b *testing.B) {
	logger := log.New(os.Stdout, "[BENCH] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	config.Cache.Enabled = false
	config.Profiler.Enabled = false

	pm, _ := NewPerformanceManager(config, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pm.GetMetrics()
	}
}

func BenchmarkPerformanceManagerGetStatus(b *testing.B) {
	logger := log.New(os.Stdout, "[BENCH] ", log.LstdFlags)
	config := DefaultPerformanceConfig()
	config.Cache.Enabled = false
	config.Profiler.Enabled = false

	pm, _ := NewPerformanceManager(config, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pm.GetStatus()
	}
}