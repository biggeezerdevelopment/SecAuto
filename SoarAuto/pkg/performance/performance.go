// Package performance provides comprehensive performance optimization tools
// including caching, connection pooling, async processing, and profiling.
package performance

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// PerformanceConfig holds configuration for all performance components
type PerformanceConfig struct {
	Cache    *CacheConfig    `yaml:"cache" json:"cache"`
	Pool     *PoolConfig     `yaml:"pool" json:"pool"`
	Async    *AsyncConfig    `yaml:"async" json:"async"`
	Profiler *ProfilerConfig `yaml:"profiler" json:"profiler"`
}

// DefaultPerformanceConfig returns default performance configuration
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		Cache:    DefaultCacheConfig(),
		Pool:     DefaultPoolConfig(),
		Async:    DefaultAsyncConfig(),
		Profiler: DefaultProfilerConfig(),
	}
}

// PerformanceManager manages all performance components
type PerformanceManager struct {
	config   *PerformanceConfig
	logger   *log.Logger
	
	// Components
	cache     CacheInterface
	pool      *PoolManager
	async     *AsyncProcessor
	profiler  *PerformanceProfiler
	
	// State
	started bool
	mu      sync.RWMutex
}

// NewPerformanceManager creates a new performance manager
func NewPerformanceManager(config *PerformanceConfig, logger *log.Logger) (*PerformanceManager, error) {
	if config == nil {
		config = DefaultPerformanceConfig()
	}
	
	pm := &PerformanceManager{
		config: config,
		logger: logger,
	}
	
	// Initialize components
	if err := pm.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize performance components: %w", err)
	}
	
	return pm, nil
}

// initializeComponents initializes all performance components
func (pm *PerformanceManager) initializeComponents() error {
	var err error
	
	// Initialize cache
	pm.cache, err = NewCache(pm.config.Cache, pm.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}
	
	// Initialize connection pool manager
	pm.pool = NewPoolManager(pm.config.Pool, pm.logger)
	
	// Initialize async processor
	pm.async = NewAsyncProcessor(pm.config.Async, pm.logger)
	
	// Initialize profiler
	pm.profiler = NewPerformanceProfiler(pm.config.Profiler, pm.logger)
	
	pm.logger.Println("Performance components initialized successfully")
	return nil
}

// Start starts all performance components
func (pm *PerformanceManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.started {
		return fmt.Errorf("performance manager already started")
	}
	
	// Start connection pool monitoring
	pm.pool.StartMonitoring()
	
	// Start async processor
	if err := pm.async.Start(); err != nil {
		return fmt.Errorf("failed to start async processor: %w", err)
	}
	
	// Start profiler
	if err := pm.profiler.Start(); err != nil {
		return fmt.Errorf("failed to start profiler: %w", err)
	}
	
	pm.started = true
	pm.logger.Println("Performance manager started successfully")
	return nil
}

// Stop stops all performance components
func (pm *PerformanceManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if !pm.started {
		return nil
	}
	
	var errors []error
	
	// Stop profiler
	if err := pm.profiler.Stop(); err != nil {
		errors = append(errors, fmt.Errorf("failed to stop profiler: %w", err))
	}
	
	// Stop async processor
	if err := pm.async.Stop(); err != nil {
		errors = append(errors, fmt.Errorf("failed to stop async processor: %w", err))
	}
	
	// Stop connection pool monitoring
	pm.pool.StopMonitoring()
	
	// Close pool manager
	if err := pm.pool.Close(); err != nil {
		errors = append(errors, fmt.Errorf("failed to close pool manager: %w", err))
	}
	
	// Close cache
	if err := pm.cache.Close(); err != nil {
		errors = append(errors, fmt.Errorf("failed to close cache: %w", err))
	}
	
	pm.started = false
	
	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}
	
	pm.logger.Println("Performance manager stopped successfully")
	return nil
}

// GetCache returns the cache interface
func (pm *PerformanceManager) GetCache() CacheInterface {
	return pm.cache
}

// GetPoolManager returns the pool manager
func (pm *PerformanceManager) GetPoolManager() *PoolManager {
	return pm.pool
}

// GetAsyncProcessor returns the async processor
func (pm *PerformanceManager) GetAsyncProcessor() *AsyncProcessor {
	return pm.async
}

// GetProfiler returns the profiler
func (pm *PerformanceManager) GetProfiler() *PerformanceProfiler {
	return pm.profiler
}

// HealthCheck performs a health check on all components
func (pm *PerformanceManager) HealthCheck(ctx context.Context) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	if !pm.started {
		return fmt.Errorf("performance manager not started")
	}
	
	// Check cache
	if err := pm.cache.Ping(ctx); err != nil && pm.config.Cache.Enabled {
		return fmt.Errorf("cache health check failed: %w", err)
	}
	
	// Check pool manager
	if err := pm.pool.HealthCheck(ctx); err != nil {
		return fmt.Errorf("pool manager health check failed: %w", err)
	}
	
	// Async processor and profiler don't have explicit health checks
	// but we can verify they're running by checking their status
	
	return nil
}

// GetStatus returns the status of all performance components
func (pm *PerformanceManager) GetStatus() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	status := map[string]interface{}{
		"started": pm.started,
		"cache": map[string]interface{}{
			"enabled": pm.config.Cache.Enabled,
			"metrics": pm.cache.GetMetrics(),
		},
		"pool": pm.pool.GetPoolStatus(),
		"async": pm.async.GetStatus(),
		"profiler": pm.profiler.GetProfilerStatus(),
	}
	
	return status
}

// GetMetrics returns comprehensive performance metrics
func (pm *PerformanceManager) GetMetrics() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	metrics := map[string]interface{}{
		"timestamp": time.Now(),
		"cache": map[string]interface{}{
			"enabled": pm.config.Cache.Enabled,
		},
		"pool": map[string]interface{}{
			"database": map[string]interface{}{},
			"http": map[string]interface{}{},
		},
		"async": map[string]interface{}{},
		"runtime": map[string]interface{}{},
	}
	
	// Add cache metrics if enabled
	if pm.config.Cache.Enabled {
		cacheMetrics := pm.cache.GetMetrics()
		metrics["cache"] = map[string]interface{}{
			"enabled":     true,
			"hit_rate":    cacheMetrics.GetHitRate(),
			"hits":        cacheMetrics.Hits,
			"misses":      cacheMetrics.Misses,
			"sets":        cacheMetrics.Sets,
			"deletes":     cacheMetrics.Deletes,
			"errors":      cacheMetrics.Errors,
			"avg_latency": cacheMetrics.GetAverageLatency(),
		}
	}
	
	// Add pool metrics
	poolMetrics := pm.pool.GetMetrics()
	metrics["pool"] = map[string]interface{}{
		"database": map[string]interface{}{
			"open_connections": poolMetrics.DBOpenConnections,
			"in_use":          poolMetrics.DBInUseConns,
			"idle":            poolMetrics.DBIdleConns,
			"utilization":     poolMetrics.GetDBUtilization(),
			"wait_count":      poolMetrics.DBWaitCount,
			"wait_duration":   poolMetrics.DBWaitDuration,
		},
		"http": map[string]interface{}{
			"active_connections": poolMetrics.HTTPActiveConns,
			"idle_connections":   poolMetrics.HTTPIdleConns,
			"utilization":        poolMetrics.GetHTTPUtilization(),
			"total_requests":     poolMetrics.HTTPRequestsTotal,
			"total_errors":       poolMetrics.HTTPErrorsTotal,
		},
	}
	
	// Add async metrics
	asyncMetrics := pm.async.GetMetrics()
	metrics["async"] = map[string]interface{}{
		"jobs_queued":     asyncMetrics.JobsQueued,
		"jobs_processed":  asyncMetrics.JobsProcessed,
		"jobs_completed":  asyncMetrics.JobsCompleted,
		"jobs_failed":     asyncMetrics.JobsFailed,
		"jobs_cancelled":  asyncMetrics.JobsCancelled,
		"jobs_retried":    asyncMetrics.JobsRetried,
		"active_workers":  asyncMetrics.ActiveWorkers,
		"queue_length":    asyncMetrics.QueueLength,
		"success_rate":    asyncMetrics.GetSuccessRate(),
		"avg_processing_time": asyncMetrics.AverageProcessingTime,
	}
	
	// Add runtime metrics
	runtimeMetrics := pm.profiler.GetMetrics()
	metrics["runtime"] = map[string]interface{}{
		"memory_mb":       runtimeMetrics.GetMemoryUsageMB(),
		"heap_mb":         runtimeMetrics.GetHeapUsageMB(),
		"goroutines":      runtimeMetrics.NumGoroutine,
		"gc_pause_avg_ms": runtimeMetrics.GetGCPauseAvg(),
		"num_gc":          runtimeMetrics.NumGC,
		"gc_cpu_fraction": runtimeMetrics.GCCPUFraction,
		"uptime":          runtimeMetrics.Uptime.String(),
	}
	
	return metrics
}

// OptimizeForLoad optimizes performance settings based on current load
func (pm *PerformanceManager) OptimizeForLoad(loadMetrics map[string]float64) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	if !pm.started {
		return
	}
	
	// Optimize database connections based on load
	if dbLoad, exists := loadMetrics["database"]; exists {
		pm.pool.OptimizeForLoad(dbLoad)
	}
	
	// Additional optimizations could be added here based on other load metrics
	pm.logger.Printf("Performance optimized for load: %v", loadMetrics)
}

// IsStarted returns whether the performance manager is started
func (pm *PerformanceManager) IsStarted() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.started
}

// GetConfig returns the performance configuration
func (pm *PerformanceManager) GetConfig() *PerformanceConfig {
	return pm.config
}

// UpdateConfig updates the performance configuration
// Note: This requires restart to take effect for most settings
func (pm *PerformanceManager) UpdateConfig(config *PerformanceConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.started {
		return fmt.Errorf("cannot update config while performance manager is running")
	}
	
	pm.config = config
	
	// Re-initialize components with new config
	if err := pm.initializeComponents(); err != nil {
		return fmt.Errorf("failed to re-initialize components with new config: %w", err)
	}
	
	pm.logger.Println("Performance configuration updated")
	return nil
}

// ForceGarbageCollection triggers garbage collection
func (pm *PerformanceManager) ForceGarbageCollection() {
	pm.profiler.ForceGC()
}

// GetDetailedMetrics returns detailed metrics for monitoring and alerting
func (pm *PerformanceManager) GetDetailedMetrics() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	return map[string]interface{}{
		"timestamp": time.Now(),
		"cache":     pm.cache.GetMetrics(),
		"pool":      pm.pool.GetMetrics(),
		"async":     pm.async.GetMetrics(),
		"memory":    pm.profiler.GetMemoryStats(),
		"gc":        pm.profiler.GetGCStats(),
		"runtime":   pm.profiler.GetRuntimeStats(),
	}
}