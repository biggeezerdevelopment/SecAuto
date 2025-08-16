package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// ProfilerConfig holds configuration for the profiler
type ProfilerConfig struct {
	Enabled           bool          `yaml:"enabled" json:"enabled"`
	HTTPPort          int           `yaml:"http_port" json:"http_port"`
	CPUProfileEnabled bool          `yaml:"cpu_profile_enabled" json:"cpu_profile_enabled"`
	MemProfileEnabled bool          `yaml:"mem_profile_enabled" json:"mem_profile_enabled"`
	BlockProfileEnabled bool        `yaml:"block_profile_enabled" json:"block_profile_enabled"`
	MutexProfileEnabled bool        `yaml:"mutex_profile_enabled" json:"mutex_profile_enabled"`
	GoroutineProfileEnabled bool    `yaml:"goroutine_profile_enabled" json:"goroutine_profile_enabled"`
	MetricsInterval   time.Duration `yaml:"metrics_interval" json:"metrics_interval"`
	GCStatsEnabled    bool          `yaml:"gc_stats_enabled" json:"gc_stats_enabled"`
	MemStatsEnabled   bool          `yaml:"mem_stats_enabled" json:"mem_stats_enabled"`
}

// DefaultProfilerConfig returns default profiler configuration
func DefaultProfilerConfig() *ProfilerConfig {
	return &ProfilerConfig{
		Enabled:                 true,
		HTTPPort:                6060,
		CPUProfileEnabled:       true,
		MemProfileEnabled:       true,
		BlockProfileEnabled:     true,
		MutexProfileEnabled:     true,
		GoroutineProfileEnabled: true,
		MetricsInterval:         30 * time.Second,
		GCStatsEnabled:          true,
		MemStatsEnabled:         true,
	}
}

// RuntimeMetrics holds runtime performance metrics
type RuntimeMetrics struct {
	mu sync.RWMutex

	// Memory metrics
	Alloc         uint64 `json:"alloc"`
	TotalAlloc    uint64 `json:"total_alloc"`
	Sys           uint64 `json:"sys"`
	Lookups       uint64 `json:"lookups"`
	Mallocs       uint64 `json:"mallocs"`
	Frees         uint64 `json:"frees"`
	HeapAlloc     uint64 `json:"heap_alloc"`
	HeapSys       uint64 `json:"heap_sys"`
	HeapIdle      uint64 `json:"heap_idle"`
	HeapInuse     uint64 `json:"heap_inuse"`
	HeapReleased  uint64 `json:"heap_released"`
	HeapObjects   uint64 `json:"heap_objects"`
	StackInuse    uint64 `json:"stack_inuse"`
	StackSys      uint64 `json:"stack_sys"`
	MSpanInuse    uint64 `json:"mspan_inuse"`
	MSpanSys      uint64 `json:"mspan_sys"`
	MCacheInuse   uint64 `json:"mcache_inuse"`
	MCacheSys     uint64 `json:"mcache_sys"`
	BuckHashSys   uint64 `json:"buck_hash_sys"`
	GCSys         uint64 `json:"gc_sys"`
	OtherSys      uint64 `json:"other_sys"`

	// GC metrics
	NextGC        uint64        `json:"next_gc"`
	LastGC        uint64        `json:"last_gc"`
	PauseTotalNs  uint64        `json:"pause_total_ns"`
	PauseNs       []uint64      `json:"pause_ns"`
	PauseEnd      []uint64      `json:"pause_end"`
	NumGC         uint32        `json:"num_gc"`
	NumForcedGC   uint32        `json:"num_forced_gc"`
	GCCPUFraction float64       `json:"gc_cpu_fraction"`
	EnableGC      bool          `json:"enable_gc"`
	DebugGC       bool          `json:"debug_gc"`

	// Goroutine metrics
	NumGoroutine int `json:"num_goroutine"`
	NumCgoCall   int64 `json:"num_cgo_call"`

	// CPU metrics
	NumCPU       int `json:"num_cpu"`
	GOMAXPROCS   int `json:"gomaxprocs"`

	// Timing
	Timestamp time.Time `json:"timestamp"`
	Uptime    time.Duration `json:"uptime"`
}

// GetMemoryUsageMB returns memory usage in megabytes
func (m *RuntimeMetrics) GetMemoryUsageMB() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return float64(m.Alloc) / 1024 / 1024
}

// GetHeapUsageMB returns heap usage in megabytes
func (m *RuntimeMetrics) GetHeapUsageMB() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return float64(m.HeapAlloc) / 1024 / 1024
}

// GetGCPauseAvg returns average GC pause time in milliseconds
func (m *RuntimeMetrics) GetGCPauseAvg() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.NumGC == 0 {
		return 0.0
	}
	return float64(m.PauseTotalNs) / float64(m.NumGC) / 1e6
}

// PerformanceProfiler manages performance profiling
type PerformanceProfiler struct {
	config    *ProfilerConfig
	logger    *log.Logger
	metrics   *RuntimeMetrics
	server    *http.Server
	startTime time.Time

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewPerformanceProfiler creates a new performance profiler
func NewPerformanceProfiler(config *ProfilerConfig, logger *log.Logger) *PerformanceProfiler {
	ctx, cancel := context.WithCancel(context.Background())

	return &PerformanceProfiler{
		config:    config,
		logger:    logger,
		metrics:   &RuntimeMetrics{},
		startTime: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the performance profiler
func (p *PerformanceProfiler) Start() error {
	if !p.config.Enabled {
		p.logger.Println("Performance profiler disabled")
		return nil
	}

	// Configure runtime profiling
	p.configureRuntimeProfiling()

	// Start HTTP server for pprof endpoints
	if err := p.startHTTPServer(); err != nil {
		return fmt.Errorf("failed to start profiler HTTP server: %w", err)
	}

	// Start metrics collection
	p.wg.Add(1)
	go p.metricsLoop()

	p.logger.Printf("Performance profiler started on port %d", p.config.HTTPPort)
	return nil
}

// Stop stops the performance profiler
func (p *PerformanceProfiler) Stop() error {
	if !p.config.Enabled {
		return nil
	}

	p.cancel()

	// Stop HTTP server
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := p.server.Shutdown(ctx); err != nil {
			p.logger.Printf("Error shutting down profiler server: %v", err)
		}
	}

	// Wait for metrics collection to stop
	p.wg.Wait()

	p.logger.Println("Performance profiler stopped")
	return nil
}

// configureRuntimeProfiling configures Go runtime profiling
func (p *PerformanceProfiler) configureRuntimeProfiling() {
	if p.config.BlockProfileEnabled {
		runtime.SetBlockProfileRate(1)
	}

	if p.config.MutexProfileEnabled {
		runtime.SetMutexProfileFraction(1)
	}

	// Set GC target percentage (optional optimization)
	debug.SetGCPercent(100)
}

// startHTTPServer starts the HTTP server for pprof endpoints
func (p *PerformanceProfiler) startHTTPServer() error {
	mux := http.NewServeMux()

	// Register pprof handlers
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Custom handlers
	mux.HandleFunc("/debug/metrics", p.metricsHandler)
	mux.HandleFunc("/debug/health", p.healthHandler)
	mux.HandleFunc("/debug/gc", p.gcHandler)
	mux.HandleFunc("/debug/version", p.versionHandler)

	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.config.HTTPPort),
		Handler: mux,
	}

	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			p.logger.Printf("Profiler HTTP server error: %v", err)
		}
	}()

	return nil
}

// metricsLoop runs the metrics collection loop
func (p *PerformanceProfiler) metricsLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.updateMetrics()
		case <-p.ctx.Done():
			return
		}
	}
}

// updateMetrics updates runtime metrics
func (p *PerformanceProfiler) updateMetrics() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	// Get memory stats
	if p.config.MemStatsEnabled {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		p.metrics.Alloc = memStats.Alloc
		p.metrics.TotalAlloc = memStats.TotalAlloc
		p.metrics.Sys = memStats.Sys
		p.metrics.Lookups = memStats.Lookups
		p.metrics.Mallocs = memStats.Mallocs
		p.metrics.Frees = memStats.Frees
		p.metrics.HeapAlloc = memStats.HeapAlloc
		p.metrics.HeapSys = memStats.HeapSys
		p.metrics.HeapIdle = memStats.HeapIdle
		p.metrics.HeapInuse = memStats.HeapInuse
		p.metrics.HeapReleased = memStats.HeapReleased
		p.metrics.HeapObjects = memStats.HeapObjects
		p.metrics.StackInuse = memStats.StackInuse
		p.metrics.StackSys = memStats.StackSys
		p.metrics.MSpanInuse = memStats.MSpanInuse
		p.metrics.MSpanSys = memStats.MSpanSys
		p.metrics.MCacheInuse = memStats.MCacheInuse
		p.metrics.MCacheSys = memStats.MCacheSys
		p.metrics.BuckHashSys = memStats.BuckHashSys
		p.metrics.GCSys = memStats.GCSys
		p.metrics.OtherSys = memStats.OtherSys
		p.metrics.NextGC = memStats.NextGC
		p.metrics.LastGC = memStats.LastGC
		p.metrics.PauseTotalNs = memStats.PauseTotalNs
		p.metrics.PauseNs = memStats.PauseNs[:]
		p.metrics.PauseEnd = memStats.PauseEnd[:]
		p.metrics.NumGC = memStats.NumGC
		p.metrics.NumForcedGC = memStats.NumForcedGC
		p.metrics.GCCPUFraction = memStats.GCCPUFraction
		p.metrics.EnableGC = memStats.EnableGC
		p.metrics.DebugGC = memStats.DebugGC
	}

	// Get goroutine stats
	p.metrics.NumGoroutine = runtime.NumGoroutine()
	p.metrics.NumCgoCall = runtime.NumCgoCall()

	// Get CPU stats
	p.metrics.NumCPU = runtime.NumCPU()
	p.metrics.GOMAXPROCS = runtime.GOMAXPROCS(0)

	// Update timing
	p.metrics.Timestamp = time.Now()
	p.metrics.Uptime = time.Since(p.startTime)
}

// GetMetrics returns current runtime metrics
func (p *PerformanceProfiler) GetMetrics() *RuntimeMetrics {
	p.updateMetrics()
	return p.metrics
}

// metricsHandler handles HTTP requests for metrics
func (p *PerformanceProfiler) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := p.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
		return
	}
}

// healthHandler handles HTTP requests for health check
func (p *PerformanceProfiler) healthHandler(w http.ResponseWriter, r *http.Request) {
	metrics := p.GetMetrics()

	health := map[string]interface{}{
		"status":     "healthy",
		"uptime":     metrics.Uptime.String(),
		"goroutines": metrics.NumGoroutine,
		"memory_mb":  metrics.GetMemoryUsageMB(),
		"heap_mb":    metrics.GetHeapUsageMB(),
		"gc_pause_avg_ms": metrics.GetGCPauseAvg(),
		"timestamp":  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, "Failed to encode health", http.StatusInternalServerError)
		return
	}
}

// gcHandler handles HTTP requests to trigger garbage collection
func (p *PerformanceProfiler) gcHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	beforeGC := p.GetMetrics()
	runtime.GC()
	afterGC := p.GetMetrics()

	result := map[string]interface{}{
		"triggered": true,
		"before": map[string]interface{}{
			"heap_alloc_mb": float64(beforeGC.HeapAlloc) / 1024 / 1024,
			"num_gc":        beforeGC.NumGC,
		},
		"after": map[string]interface{}{
			"heap_alloc_mb": float64(afterGC.HeapAlloc) / 1024 / 1024,
			"num_gc":        afterGC.NumGC,
		},
		"freed_mb": float64(beforeGC.HeapAlloc-afterGC.HeapAlloc) / 1024 / 1024,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode GC result", http.StatusInternalServerError)
		return
	}
}

// versionHandler handles HTTP requests for version information
func (p *PerformanceProfiler) versionHandler(w http.ResponseWriter, r *http.Request) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		http.Error(w, "Build info not available", http.StatusInternalServerError)
		return
	}

	version := map[string]interface{}{
		"go_version": runtime.Version(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
		"build_info": buildInfo,
		"timestamp":  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(version); err != nil {
		http.Error(w, "Failed to encode version", http.StatusInternalServerError)
		return
	}
}

// ForceGC triggers garbage collection
func (p *PerformanceProfiler) ForceGC() {
	runtime.GC()
	p.logger.Println("Forced garbage collection")
}

// GetMemoryStats returns current memory statistics
func (p *PerformanceProfiler) GetMemoryStats() map[string]interface{} {
	metrics := p.GetMetrics()

	return map[string]interface{}{
		"alloc_mb":        metrics.GetMemoryUsageMB(),
		"heap_alloc_mb":   metrics.GetHeapUsageMB(),
		"heap_sys_mb":     float64(metrics.HeapSys) / 1024 / 1024,
		"heap_idle_mb":    float64(metrics.HeapIdle) / 1024 / 1024,
		"heap_inuse_mb":   float64(metrics.HeapInuse) / 1024 / 1024,
		"heap_released_mb": float64(metrics.HeapReleased) / 1024 / 1024,
		"heap_objects":    metrics.HeapObjects,
		"stack_inuse_mb":  float64(metrics.StackInuse) / 1024 / 1024,
		"stack_sys_mb":    float64(metrics.StackSys) / 1024 / 1024,
		"sys_mb":          float64(metrics.Sys) / 1024 / 1024,
		"mallocs":         metrics.Mallocs,
		"frees":           metrics.Frees,
		"total_alloc_mb":  float64(metrics.TotalAlloc) / 1024 / 1024,
	}
}

// GetGCStats returns garbage collection statistics
func (p *PerformanceProfiler) GetGCStats() map[string]interface{} {
	metrics := p.GetMetrics()

	return map[string]interface{}{
		"num_gc":           metrics.NumGC,
		"num_forced_gc":    metrics.NumForcedGC,
		"pause_total_ms":   float64(metrics.PauseTotalNs) / 1e6,
		"pause_avg_ms":     metrics.GetGCPauseAvg(),
		"gc_cpu_fraction":  metrics.GCCPUFraction,
		"next_gc_mb":       float64(metrics.NextGC) / 1024 / 1024,
		"last_gc":          time.Unix(0, int64(metrics.LastGC)),
		"enable_gc":        metrics.EnableGC,
		"debug_gc":         metrics.DebugGC,
	}
}

// GetRuntimeStats returns runtime statistics
func (p *PerformanceProfiler) GetRuntimeStats() map[string]interface{} {
	metrics := p.GetMetrics()

	return map[string]interface{}{
		"goroutines":    metrics.NumGoroutine,
		"cgo_calls":     metrics.NumCgoCall,
		"num_cpu":       metrics.NumCPU,
		"gomaxprocs":    metrics.GOMAXPROCS,
		"go_version":    runtime.Version(),
		"go_os":         runtime.GOOS,
		"go_arch":       runtime.GOARCH,
		"uptime":        metrics.Uptime.String(),
		"uptime_seconds": metrics.Uptime.Seconds(),
	}
}

// GetProfilerStatus returns profiler status
func (p *PerformanceProfiler) GetProfilerStatus() map[string]interface{} {
	return map[string]interface{}{
		"enabled":     p.config.Enabled,
		"http_port":   p.config.HTTPPort,
		"profiles": map[string]bool{
			"cpu":       p.config.CPUProfileEnabled,
			"memory":    p.config.MemProfileEnabled,
			"block":     p.config.BlockProfileEnabled,
			"mutex":     p.config.MutexProfileEnabled,
			"goroutine": p.config.GoroutineProfileEnabled,
		},
		"metrics": map[string]bool{
			"gc_stats":  p.config.GCStatsEnabled,
			"mem_stats": p.config.MemStatsEnabled,
		},
		"metrics_interval": p.config.MetricsInterval.String(),
		"uptime":          time.Since(p.startTime).String(),
	}
}