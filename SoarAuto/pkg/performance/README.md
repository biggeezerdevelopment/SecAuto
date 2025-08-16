# Performance Package

The performance package provides comprehensive performance optimization tools for the SecAuto platform, including caching, connection pooling, async processing, and profiling capabilities.

## Features

### 🚀 **Caching System**
- **Redis-based distributed caching** with fallback support
- **Cache-aside and write-through patterns**
- **Automatic cache invalidation** and TTL management
- **Performance metrics** with hit/miss rates and latency tracking
- **JSON serialization support** for complex data structures

### 🔗 **Connection Pooling**
- **Database connection pooling** with configurable limits
- **HTTP client connection reuse** and optimization
- **Connection health monitoring** and automatic cleanup
- **Load-based optimization** with dynamic pool sizing
- **Comprehensive connection metrics** and utilization tracking

### ⚡ **Async Processing**
- **Priority-based job queues** (Critical, High, Normal, Low)
- **Worker pool management** with configurable concurrency
- **Automatic retry logic** with exponential backoff
- **Job persistence** and recovery capabilities
- **Real-time job monitoring** and metrics collection

### 📊 **Performance Profiling**
- **Built-in pprof endpoints** for CPU, memory, and goroutine profiling
- **Runtime metrics collection** with memory and GC statistics
- **HTTP endpoints** for metrics and health checks
- **Automatic garbage collection** monitoring and optimization
- **Performance benchmarking** and regression detection

## Quick Start

### Basic Usage

```go
import "SoarAuto/pkg/performance"

// Create performance manager with default configuration
config := performance.DefaultPerformanceConfig()
logger := log.New(os.Stdout, "[PERF] ", log.LstdFlags)

pm, err := performance.NewPerformanceManager(config, logger)
if err != nil {
    log.Fatal(err)
}

// Start all performance components
err = pm.Start()
if err != nil {
    log.Fatal(err)
}
defer pm.Stop()

// Use individual components
cache := pm.GetCache()
asyncProcessor := pm.GetAsyncProcessor()
profiler := pm.GetProfiler()
```

### Caching

```go
// Basic cache operations
ctx := context.Background()

// Set a value with TTL
err := cache.Set(ctx, "user:123", userData, 15*time.Minute)

// Get a value
data, err := cache.Get(ctx, "user:123")

// JSON operations
err = cache.SetJSON(ctx, "config", configData, time.Hour)
err = cache.GetJSON(ctx, "config", &configData)

// Cache metrics
metrics := cache.GetMetrics()
fmt.Printf("Hit rate: %.2f%%\n", metrics.GetHitRate())
```

### Async Processing

```go
// Register a job handler
type PlaybookHandler struct{}

func (h *PlaybookHandler) Handle(ctx context.Context, job *performance.Job) error {
    // Process the job
    playbookName := job.Payload["playbook"].(string)
    // ... execute playbook logic
    job.Result = map[string]interface{}{
        "status": "completed",
        "output": "Playbook executed successfully",
    }
    return nil
}

func (h *PlaybookHandler) GetType() string {
    return "playbook_execution"
}

// Register handler
handler := &PlaybookHandler{}
asyncProcessor.RegisterHandler(handler)

// Submit a job
job := performance.NewJob(
    "job-123",
    "playbook_execution",
    performance.PriorityHigh,
    map[string]interface{}{
        "playbook": "security_scan",
        "target":   "192.168.1.100",
    },
)

err := asyncProcessor.SubmitJob(job)
if err != nil {
    log.Printf("Failed to submit job: %v", err)
}

// Wait for completion
job.Wait()
fmt.Printf("Job status: %s\n", job.Status.String())
```

### Connection Pooling

```go
// Configure database connection pool
db, err := sql.Open("postgres", connectionString)
if err != nil {
    log.Fatal(err)
}

poolManager := pm.GetPoolManager()
err = poolManager.ConfigureDatabase(db)
if err != nil {
    log.Fatal(err)
}

// Get optimized HTTP client
httpClient := poolManager.GetHTTPClient()

// Monitor pool metrics
metrics := poolManager.GetMetrics()
fmt.Printf("DB utilization: %.2f%%\n", metrics.GetDBUtilization())
```

### Performance Profiling

```go
// Access profiling endpoints
// http://localhost:6060/debug/pprof/
// http://localhost:6060/debug/metrics
// http://localhost:6060/debug/health

// Get runtime metrics
profiler := pm.GetProfiler()
memStats := profiler.GetMemoryStats()
gcStats := profiler.GetGCStats()
runtimeStats := profiler.GetRuntimeStats()

// Force garbage collection
profiler.ForceGC()
```

## Configuration

### Complete Configuration Example

```yaml
performance:
  cache:
    enabled: true
    redis_addr: "localhost:6379"
    redis_password: ""
    redis_db: 0
    default_ttl: "15m"
    max_retries: 3
    dial_timeout: "5s"
    read_timeout: "3s"
    write_timeout: "3s"
    pool_size: 10
    min_idle_conns: 2
    max_conn_age: "30m"
    pool_timeout: "4s"
    idle_timeout: "5m"

  pool:
    db_max_open_conns: 25
    db_max_idle_conns: 5
    db_conn_max_lifetime: "30m"
    db_conn_max_idle_time: "5m"
    http_max_idle_conns: 100
    http_max_idle_conns_per_host: 10
    http_max_conns_per_host: 50
    http_idle_conn_timeout: "90s"
    http_timeout: "30s"
    http_keep_alive: "30s"
    http_tls_handshake_timeout: "10s"
    monitoring_enabled: true
    monitoring_interval: "30s"

  async:
    worker_count: 4
    queue_size: 1000
    retry_delay: "30s"
    max_retries: 3
    job_timeout: "5m"
    metrics_interval: "30s"
    enable_persistence: false
    persistence_file: "jobs.json"

  profiler:
    enabled: true
    http_port: 6060
    cpu_profile_enabled: true
    mem_profile_enabled: true
    block_profile_enabled: true
    mutex_profile_enabled: true
    goroutine_profile_enabled: true
    metrics_interval: "30s"
    gc_stats_enabled: true
    mem_stats_enabled: true
```

### Environment-Specific Configurations

```go
// Development configuration
func DevelopmentConfig() *PerformanceConfig {
    config := DefaultPerformanceConfig()
    config.Cache.Enabled = false  // Use in-memory for dev
    config.Profiler.HTTPPort = 6061
    config.Async.WorkerCount = 2
    return config
}

// Production configuration
func ProductionConfig() *PerformanceConfig {
    config := DefaultPerformanceConfig()
    config.Cache.RedisAddr = "redis-cluster:6379"
    config.Pool.DBMaxOpenConns = 50
    config.Async.WorkerCount = runtime.NumCPU() * 2
    config.Profiler.HTTPPort = 6060
    return config
}
```

## Monitoring & Metrics

### Health Checks

```go
// Comprehensive health check
ctx := context.Background()
err := pm.HealthCheck(ctx)
if err != nil {
    log.Printf("Health check failed: %v", err)
}

// Component-specific health checks
err = poolManager.HealthCheck(ctx)
err = cache.Ping(ctx)
```

### Metrics Collection

```go
// Get comprehensive metrics
metrics := pm.GetMetrics()
fmt.Printf("Cache hit rate: %.2f%%\n", metrics["cache"].(map[string]interface{})["hit_rate"])
fmt.Printf("Active workers: %d\n", metrics["async"].(map[string]interface{})["active_workers"])
fmt.Printf("Memory usage: %.2f MB\n", metrics["runtime"].(map[string]interface{})["memory_mb"])

// Get detailed metrics for monitoring systems
detailedMetrics := pm.GetDetailedMetrics()
// Export to Prometheus, InfluxDB, etc.
```

### Performance Optimization

```go
// Automatic load-based optimization
loadMetrics := map[string]float64{
    "database": 0.85,  // 85% utilization
    "cpu":      0.70,  // 70% utilization
    "memory":   0.60,  // 60% utilization
}

pm.OptimizeForLoad(loadMetrics)
```

## HTTP Endpoints

When the profiler is enabled, the following HTTP endpoints are available:

### Profiling Endpoints
- `GET /debug/pprof/` - Profile index
- `GET /debug/pprof/profile` - CPU profile
- `GET /debug/pprof/heap` - Memory heap profile
- `GET /debug/pprof/goroutine` - Goroutine profile
- `GET /debug/pprof/block` - Block profile
- `GET /debug/pprof/mutex` - Mutex profile

### Custom Endpoints
- `GET /debug/metrics` - Runtime metrics (JSON)
- `GET /debug/health` - Health check (JSON)
- `POST /debug/gc` - Trigger garbage collection
- `GET /debug/version` - Version information

### Example Responses

```bash
# Get health status
curl http://localhost:6060/debug/health
{
  "status": "healthy",
  "uptime": "2h30m15s",
  "goroutines": 25,
  "memory_mb": 45.2,
  "heap_mb": 32.1,
  "gc_pause_avg_ms": 0.8,
  "timestamp": "2024-01-15T10:30:00Z"
}

# Get runtime metrics
curl http://localhost:6060/debug/metrics
{
  "alloc": 47456256,
  "total_alloc": 156789123,
  "sys": 89123456,
  "num_goroutine": 25,
  "num_gc": 15,
  "gc_cpu_fraction": 0.002,
  "uptime": "2h30m15s"
}

# Trigger garbage collection
curl -X POST http://localhost:6060/debug/gc
{
  "triggered": true,
  "before": {
    "heap_alloc_mb": 32.1,
    "num_gc": 15
  },
  "after": {
    "heap_alloc_mb": 28.5,
    "num_gc": 16
  },
  "freed_mb": 3.6,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Best Practices

### Caching Strategy

```go
// Use cache keys with consistent naming
keyBuilder := performance.NewCacheKeyBuilder("secauto")
playbookKey := keyBuilder.PlaybookKey("security_scan")
jobKey := keyBuilder.JobKey("job-123")

// Implement cache-aside pattern
func GetPlaybook(ctx context.Context, name string) (*Playbook, error) {
    key := keyBuilder.PlaybookKey(name)
    
    // Try cache first
    var playbook Playbook
    err := cache.GetJSON(ctx, key, &playbook)
    if err == nil {
        return &playbook, nil
    }
    
    // Cache miss - load from database
    playbook, err = loadPlaybookFromDB(name)
    if err != nil {
        return nil, err
    }
    
    // Store in cache
    cache.SetJSON(ctx, key, playbook, 15*time.Minute)
    return &playbook, nil
}
```

### Async Job Design

```go
// Design jobs to be idempotent and resumable
type PlaybookJob struct {
    ID           string
    PlaybookName string
    Target       string
    State        map[string]interface{} // For resumability
}

func (h *PlaybookHandler) Handle(ctx context.Context, job *performance.Job) error {
    // Check if job can be resumed
    if state, exists := job.Payload["state"]; exists {
        // Resume from previous state
        return h.resumeExecution(ctx, job, state)
    }
    
    // Start fresh execution
    return h.executePlaybook(ctx, job)
}
```

### Connection Pool Optimization

```go
// Monitor and adjust pool settings based on load
func optimizeConnections(poolManager *performance.PoolManager) {
    metrics := poolManager.GetMetrics()
    
    if metrics.GetDBUtilization() > 80 {
        // High utilization - consider increasing pool size
        log.Println("High DB utilization detected")
    }
    
    if metrics.DBWaitCount > 100 {
        // Too many waits - increase pool size
        log.Println("High DB wait count detected")
    }
}
```

### Memory Management

```go
// Regular memory monitoring
func monitorMemory(profiler *performance.PerformanceProfiler) {
    memStats := profiler.GetMemoryStats()
    
    if memUsage := memStats["alloc_mb"].(float64); memUsage > 500 {
        log.Printf("High memory usage: %.2f MB", memUsage)
        profiler.ForceGC()
    }
}
```

## Testing

### Unit Tests

```bash
# Run all performance package tests
go test -v ./pkg/performance/

# Run specific test files
go test -v ./pkg/performance/cache_test.go
go test -v ./pkg/performance/async_test.go
go test -v ./pkg/performance/pool_test.go
go test -v ./pkg/performance/profiler_test.go

# Run benchmarks
go test -bench=. ./pkg/performance/
```

### Integration Tests

```bash
# Run integration tests (requires Redis)
go test -v -tags=integration ./pkg/performance/

# Run with race detection
go test -race -v ./pkg/performance/
```

### Load Testing

```go
// Example load test
func TestAsyncProcessorLoad(t *testing.T) {
    // Submit 1000 jobs concurrently
    var wg sync.WaitGroup
    for i := 0; i < 1000; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            job := performance.NewJob(
                fmt.Sprintf("load-test-%d", id),
                "test-handler",
                performance.PriorityNormal,
                nil,
            )
            asyncProcessor.SubmitJob(job)
        }(i)
    }
    wg.Wait()
}
```

## Troubleshooting

### Common Issues

1. **Cache Connection Failures**
   ```go
   // Check Redis connectivity
   err := cache.Ping(ctx)
   if err != nil {
       log.Printf("Cache connection failed: %v", err)
   }
   ```

2. **High Memory Usage**
   ```go
   // Monitor GC frequency
   gcStats := profiler.GetGCStats()
   if gcStats["gc_cpu_fraction"].(float64) > 0.05 {
       log.Println("High GC overhead detected")
   }
   ```

3. **Job Queue Backlog**
   ```go
   // Monitor queue length
   status := asyncProcessor.GetStatus()
   queueInfo := status["queues"].(map[string]interface{})
   if queueInfo["total"].(int32) > 500 {
       log.Println("Job queue backlog detected")
   }
   ```

### Debug Mode

```go
// Enable debug logging
config := performance.DefaultPerformanceConfig()
config.Profiler.Enabled = true
config.Profiler.HTTPPort = 6060

// Access debug endpoints
// http://localhost:6060/debug/pprof/
// Use go tool pprof for analysis
```

## Performance Benchmarks

Typical performance characteristics:

- **Cache Operations**: <5ms latency, 50,000+ ops/sec
- **Job Processing**: 100+ jobs/minute per worker
- **Memory Overhead**: <50MB baseline
- **Connection Pooling**: 90%+ connection reuse
- **GC Overhead**: <2% CPU time

## Contributing

When contributing to the performance package:

1. **Add comprehensive tests** for new features
2. **Include benchmarks** for performance-critical code
3. **Update documentation** and examples
4. **Follow Go best practices** for concurrent code
5. **Test with race detection** enabled

## License

This package is part of the SecAuto project and follows the same licensing terms.