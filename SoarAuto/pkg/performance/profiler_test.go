package performance

import (
	"log"
	"os"
	"testing"
	"time"
)

func TestDefaultProfilerConfig(t *testing.T) {
	config := DefaultProfilerConfig()

	if !config.Enabled {
		t.Error("Expected profiler to be enabled by default")
	}

	if config.HTTPPort != 6060 {
		t.Errorf("Expected HTTP port 6060, got %d", config.HTTPPort)
	}

	if !config.CPUProfileEnabled {
		t.Error("Expected CPU profile to be enabled by default")
	}

	if !config.MemProfileEnabled {
		t.Error("Expected memory profile to be enabled by default")
	}

	if !config.GCStatsEnabled {
		t.Error("Expected GC stats to be enabled by default")
	}

	if config.MetricsInterval != 30*time.Second {
		t.Errorf("Expected metrics interval 30s, got %v", config.MetricsInterval)
	}
}

func TestRuntimeMetrics(t *testing.T) {
	metrics := &RuntimeMetrics{}

	// Test initial values
	if metrics.GetMemoryUsageMB() != 0.0 {
		t.Errorf("Expected initial memory usage to be 0.0, got %f", metrics.GetMemoryUsageMB())
	}

	if metrics.GetHeapUsageMB() != 0.0 {
		t.Errorf("Expected initial heap usage to be 0.0, got %f", metrics.GetHeapUsageMB())
	}

	if metrics.GetGCPauseAvg() != 0.0 {
		t.Errorf("Expected initial GC pause avg to be 0.0, got %f", metrics.GetGCPauseAvg())
	}

	// Set some test values
	metrics.mu.Lock()
	metrics.Alloc = 10 * 1024 * 1024      // 10MB
	metrics.HeapAlloc = 8 * 1024 * 1024   // 8MB
	metrics.PauseTotalNs = 5000000        // 5ms total
	metrics.NumGC = 5                     // 5 GC cycles
	metrics.mu.Unlock()

	expectedMemUsage := 10.0
	if memUsage := metrics.GetMemoryUsageMB(); memUsage != expectedMemUsage {
		t.Errorf("Expected memory usage %f MB, got %f MB", expectedMemUsage, memUsage)
	}

	expectedHeapUsage := 8.0
	if heapUsage := metrics.GetHeapUsageMB(); heapUsage != expectedHeapUsage {
		t.Errorf("Expected heap usage %f MB, got %f MB", expectedHeapUsage, heapUsage)
	}

	expectedGCPause := 1.0 // 5ms / 5 cycles = 1ms average
	if gcPause := metrics.GetGCPauseAvg(); gcPause != expectedGCPause {
		t.Errorf("Expected GC pause avg %f ms, got %f ms", expectedGCPause, gcPause)
	}
}

func TestPerformanceProfilerCreation(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultProfilerConfig()

	profiler := NewPerformanceProfiler(config, logger)

	if profiler == nil {
		t.Fatal("Expected profiler to be created")
	}

	if profiler.config != config {
		t.Error("Expected config to be set")
	}

	if profiler.logger != logger {
		t.Error("Expected logger to be set")
	}

	if profiler.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}

	if profiler.startTime.IsZero() {
		t.Error("Expected start time to be set")
	}
}

func TestPerformanceProfilerDisabled(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultProfilerConfig()
	config.Enabled = false

	profiler := NewPerformanceProfiler(config, logger)

	// Start should succeed but do nothing
	err := profiler.Start()
	if err != nil {
		t.Errorf("Start should succeed for disabled profiler, got: %v", err)
	}

	// Stop should succeed
	err = profiler.Stop()
	if err != nil {
		t.Errorf("Stop should succeed for disabled profiler, got: %v", err)
	}
}

func TestPerformanceProfilerMetricsUpdate(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultProfilerConfig()
	config.MetricsInterval = 10 * time.Millisecond

	profiler := NewPerformanceProfiler(config, logger)

	// Update metrics manually
	profiler.updateMetrics()

	metrics := profiler.GetMetrics()

	// Check that metrics were populated
	if metrics.NumGoroutine <= 0 {
		t.Error("Expected positive number of goroutines")
	}

	if metrics.NumCPU <= 0 {
		t.Error("Expected positive number of CPUs")
	}

	if metrics.GOMAXPROCS <= 0 {
		t.Error("Expected positive GOMAXPROCS")
	}

	if metrics.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	if metrics.Uptime <= 0 {
		t.Error("Expected positive uptime")
	}
}

func TestPerformanceProfilerMemoryStats(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultProfilerConfig()

	profiler := NewPerformanceProfiler(config, logger)

	memStats := profiler.GetMemoryStats()

	// Check that memory stats contain expected keys
	expectedKeys := []string{
		"alloc_mb", "heap_alloc_mb", "heap_sys_mb", "heap_idle_mb",
		"heap_inuse_mb", "heap_released_mb", "heap_objects",
		"stack_inuse_mb", "stack_sys_mb", "sys_mb",
		"mallocs", "frees", "total_alloc_mb",
	}

	for _, key := range expectedKeys {
		if _, exists := memStats[key]; !exists {
			t.Errorf("Expected memory stats to contain key: %s", key)
		}
	}

	// Check that values are reasonable
	if allocMB, ok := memStats["alloc_mb"].(float64); ok && allocMB < 0 {
		t.Error("Expected alloc_mb to be non-negative")
	}
}

func TestPerformanceProfilerGCStats(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultProfilerConfig()

	profiler := NewPerformanceProfiler(config, logger)

	gcStats := profiler.GetGCStats()

	// Check that GC stats contain expected keys
	expectedKeys := []string{
		"num_gc", "num_forced_gc", "pause_total_ms", "pause_avg_ms",
		"gc_cpu_fraction", "next_gc_mb", "last_gc", "enable_gc", "debug_gc",
	}

	for _, key := range expectedKeys {
		if _, exists := gcStats[key]; !exists {
			t.Errorf("Expected GC stats to contain key: %s", key)
		}
	}

	// Check that values are reasonable
	if numGC, ok := gcStats["num_gc"].(uint32); ok && numGC < 0 {
		t.Error("Expected num_gc to be non-negative")
	}
}

func TestPerformanceProfilerRuntimeStats(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultProfilerConfig()

	profiler := NewPerformanceProfiler(config, logger)

	runtimeStats := profiler.GetRuntimeStats()

	// Check that runtime stats contain expected keys
	expectedKeys := []string{
		"goroutines", "cgo_calls", "num_cpu", "gomaxprocs",
		"go_version", "go_os", "go_arch", "uptime", "uptime_seconds",
	}

	for _, key := range expectedKeys {
		if _, exists := runtimeStats[key]; !exists {
			t.Errorf("Expected runtime stats to contain key: %s", key)
		}
	}

	// Check specific values
	if goroutines, ok := runtimeStats["goroutines"].(int); ok && goroutines <= 0 {
		t.Error("Expected positive number of goroutines")
	}

	if numCPU, ok := runtimeStats["num_cpu"].(int); ok && numCPU <= 0 {
		t.Error("Expected positive number of CPUs")
	}

	if goVersion, ok := runtimeStats["go_version"].(string); ok && goVersion == "" {
		t.Error("Expected Go version to be set")
	}
}

func TestPerformanceProfilerStatus(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultProfilerConfig()

	profiler := NewPerformanceProfiler(config, logger)

	status := profiler.GetProfilerStatus()

	// Check that status contains expected keys
	expectedKeys := []string{
		"enabled", "http_port", "profiles", "metrics",
		"metrics_interval", "uptime",
	}

	for _, key := range expectedKeys {
		if _, exists := status[key]; !exists {
			t.Errorf("Expected status to contain key: %s", key)
		}
	}

	// Check enabled status
	if enabled, ok := status["enabled"].(bool); ok && !enabled {
		t.Error("Expected profiler to be enabled")
	}

	// Check profiles
	if profiles, ok := status["profiles"].(map[string]bool); ok {
		expectedProfiles := []string{"cpu", "memory", "block", "mutex", "goroutine"}
		for _, profile := range expectedProfiles {
			if _, exists := profiles[profile]; !exists {
				t.Errorf("Expected profiles to contain: %s", profile)
			}
		}
	} else {
		t.Error("Expected profiles to be a map[string]bool")
	}
}

func TestPerformanceProfilerForceGC(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultProfilerConfig()

	profiler := NewPerformanceProfiler(config, logger)

	// Get GC count before
	beforeStats := profiler.GetGCStats()
	beforeGC := beforeStats["num_gc"].(uint32)

	// Force GC
	profiler.ForceGC()

	// Get GC count after
	afterStats := profiler.GetGCStats()
	afterGC := afterStats["num_gc"].(uint32)

	// GC count should have increased
	if afterGC <= beforeGC {
		t.Error("Expected GC count to increase after ForceGC")
	}
}

// Integration test for HTTP server (requires available port)
func TestPerformanceProfilerHTTPServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HTTP server test in short mode")
	}

	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultProfilerConfig()
	config.HTTPPort = 0 // Use random available port
	config.MetricsInterval = 100 * time.Millisecond

	profiler := NewPerformanceProfiler(config, logger)

	// Start profiler
	err := profiler.Start()
	if err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}
	defer profiler.Stop()

	// Wait a bit for server to start
	time.Sleep(200 * time.Millisecond)

	// Note: Since we're using port 0, we can't easily test HTTP endpoints
	// In a real test environment, you would use a fixed test port
	// and make HTTP requests to verify the endpoints work
}

func TestPerformanceProfilerConcurrentAccess(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultProfilerConfig()
	config.MetricsInterval = 10 * time.Millisecond

	profiler := NewPerformanceProfiler(config, logger)

	// Start profiler
	err := profiler.Start()
	if err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}
	defer profiler.Stop()

	// Concurrent access to metrics
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				_ = profiler.GetMetrics()
				_ = profiler.GetMemoryStats()
				_ = profiler.GetGCStats()
				_ = profiler.GetRuntimeStats()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Benchmark tests
func BenchmarkRuntimeMetricsUpdate(b *testing.B) {
	logger := log.New(os.Stdout, "[BENCH] ", log.LstdFlags)
	config := DefaultProfilerConfig()
	profiler := NewPerformanceProfiler(config, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profiler.updateMetrics()
	}
}

func BenchmarkRuntimeMetricsGetMemoryUsage(b *testing.B) {
	metrics := &RuntimeMetrics{
		Alloc: 10 * 1024 * 1024, // 10MB
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetMemoryUsageMB()
	}
}

func BenchmarkRuntimeMetricsGetGCPauseAvg(b *testing.B) {
	metrics := &RuntimeMetrics{
		PauseTotalNs: 5000000, // 5ms
		NumGC:        5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetGCPauseAvg()
	}
}

func BenchmarkPerformanceProfilerGetStats(b *testing.B) {
	logger := log.New(os.Stdout, "[BENCH] ", log.LstdFlags)
	config := DefaultProfilerConfig()
	profiler := NewPerformanceProfiler(config, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = profiler.GetMemoryStats()
		_ = profiler.GetGCStats()
		_ = profiler.GetRuntimeStats()
	}
}