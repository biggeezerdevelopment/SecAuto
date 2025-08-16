package performance

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver for testing
)

func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()

	// Test database defaults
	if config.DBMaxOpenConns != 25 {
		t.Errorf("Expected DBMaxOpenConns 25, got %d", config.DBMaxOpenConns)
	}

	if config.DBMaxIdleConns != 5 {
		t.Errorf("Expected DBMaxIdleConns 5, got %d", config.DBMaxIdleConns)
	}

	if config.DBConnMaxLifetime != 30*time.Minute {
		t.Errorf("Expected DBConnMaxLifetime 30m, got %v", config.DBConnMaxLifetime)
	}

	// Test HTTP defaults
	if config.HTTPMaxIdleConns != 100 {
		t.Errorf("Expected HTTPMaxIdleConns 100, got %d", config.HTTPMaxIdleConns)
	}

	if config.HTTPTimeout != 30*time.Second {
		t.Errorf("Expected HTTPTimeout 30s, got %v", config.HTTPTimeout)
	}

	// Test monitoring defaults
	if !config.MonitoringEnabled {
		t.Error("Expected monitoring to be enabled by default")
	}

	if config.MonitoringInterval != 30*time.Second {
		t.Errorf("Expected MonitoringInterval 30s, got %v", config.MonitoringInterval)
	}
}

func TestPoolMetrics(t *testing.T) {
	metrics := &PoolMetrics{LastUpdated: time.Now()}

	// Test initial state
	if metrics.GetDBUtilization() != 0.0 {
		t.Errorf("Expected initial DB utilization to be 0.0, got %f", metrics.GetDBUtilization())
	}

	if metrics.GetHTTPUtilization() != 0.0 {
		t.Errorf("Expected initial HTTP utilization to be 0.0, got %f", metrics.GetHTTPUtilization())
	}

	// Test DB utilization calculation
	metrics.mu.Lock()
	metrics.DBOpenConnections = 10
	metrics.DBInUseConns = 7
	metrics.mu.Unlock()

	expectedDBUtil := 70.0
	if dbUtil := metrics.GetDBUtilization(); dbUtil != expectedDBUtil {
		t.Errorf("Expected DB utilization %f, got %f", expectedDBUtil, dbUtil)
	}

	// Test HTTP utilization calculation
	metrics.mu.Lock()
	metrics.HTTPActiveConns = 3
	metrics.HTTPIdleConns = 7
	metrics.mu.Unlock()

	expectedHTTPUtil := 30.0
	if httpUtil := metrics.GetHTTPUtilization(); httpUtil != expectedHTTPUtil {
		t.Errorf("Expected HTTP utilization %f, got %f", expectedHTTPUtil, httpUtil)
	}
}

func TestPoolManagerCreation(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPoolConfig()

	pm := NewPoolManager(config, logger)

	if pm == nil {
		t.Fatal("Expected pool manager to be created")
	}

	if pm.config != config {
		t.Error("Expected config to be set")
	}

	if pm.logger != logger {
		t.Error("Expected logger to be set")
	}

	if pm.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}
}

func TestDatabasePoolConfiguration(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPoolConfig()
	pm := NewPoolManager(config, logger)

	// Test with nil database
	err := pm.ConfigureDatabase(nil)
	if err == nil {
		t.Error("Expected error when configuring with nil database")
	}

	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Configure database pool
	err = pm.ConfigureDatabase(db)
	if err != nil {
		t.Fatalf("Failed to configure database pool: %v", err)
	}

	// Verify configuration was applied
	stats := pm.GetDatabaseStats()
	if stats.MaxOpenConnections != config.DBMaxOpenConns {
		t.Errorf("Expected MaxOpenConnections %d, got %d", config.DBMaxOpenConns, stats.MaxOpenConnections)
	}
}

func TestHTTPClientConfiguration(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPoolConfig()
	pm := NewPoolManager(config, logger)

	// Get HTTP client
	client := pm.GetHTTPClient()
	if client == nil {
		t.Fatal("Expected HTTP client to be created")
	}

	// Verify timeout configuration
	if client.Timeout != config.HTTPTimeout {
		t.Errorf("Expected timeout %v, got %v", config.HTTPTimeout, client.Timeout)
	}

	// Get client again - should return same instance
	client2 := pm.GetHTTPClient()
	if client != client2 {
		t.Error("Expected same HTTP client instance")
	}
}

func TestPoolManagerHealthCheck(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPoolConfig()
	pm := NewPoolManager(config, logger)

	ctx := context.Background()

	// Health check without database should pass
	err := pm.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check without database should pass, got: %v", err)
	}

	// Create and configure database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = pm.ConfigureDatabase(db)
	if err != nil {
		t.Fatalf("Failed to configure database: %v", err)
	}

	// Health check with database should pass
	err = pm.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check with database should pass, got: %v", err)
	}

	// Configure HTTP client and check again
	_ = pm.GetHTTPClient()
	err = pm.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check with HTTP client should pass, got: %v", err)
	}
}

func TestPoolManagerMonitoring(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPoolConfig()
	config.MonitoringInterval = 100 * time.Millisecond // Fast interval for testing
	pm := NewPoolManager(config, logger)

	// Start monitoring
	pm.StartMonitoring()

	// Wait a bit for monitoring to run
	time.Sleep(200 * time.Millisecond)

	// Stop monitoring
	pm.StopMonitoring()

	// Verify metrics were updated
	metrics := pm.GetMetrics()
	if metrics.LastUpdated.IsZero() {
		t.Error("Expected metrics to be updated")
	}
}

func TestPoolManagerWithDisabledMonitoring(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPoolConfig()
	config.MonitoringEnabled = false
	pm := NewPoolManager(config, logger)

	// Start monitoring (should be no-op)
	pm.StartMonitoring()

	// Stop monitoring (should be no-op)
	pm.StopMonitoring()

	// Should not cause any issues
}

func TestPoolStatus(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPoolConfig()
	pm := NewPoolManager(config, logger)

	// Get status without any configuration
	status := pm.GetPoolStatus()

	dbStatus, ok := status["database"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected database status to be present")
	}

	if dbStatus["configured"].(bool) {
		t.Error("Expected database to not be configured initially")
	}

	httpStatus, ok := status["http_client"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected HTTP client status to be present")
	}

	if httpStatus["configured"].(bool) {
		t.Error("Expected HTTP client to not be configured initially")
	}

	// Configure database and HTTP client
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = pm.ConfigureDatabase(db)
	if err != nil {
		t.Fatalf("Failed to configure database: %v", err)
	}

	_ = pm.GetHTTPClient()

	// Get status again
	status = pm.GetPoolStatus()

	dbStatus = status["database"].(map[string]interface{})
	if !dbStatus["configured"].(bool) {
		t.Error("Expected database to be configured")
	}

	httpStatus = status["http_client"].(map[string]interface{})
	if !httpStatus["configured"].(bool) {
		t.Error("Expected HTTP client to be configured")
	}
}

func TestLoadOptimization(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPoolConfig()
	pm := NewPoolManager(config, logger)

	// Create and configure database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	err = pm.ConfigureDatabase(db)
	if err != nil {
		t.Fatalf("Failed to configure database: %v", err)
	}

	originalMaxOpen := config.DBMaxOpenConns

	// Test high load optimization
	pm.OptimizeForLoad(0.9)
	stats := pm.GetDatabaseStats()
	if stats.MaxOpenConnections <= originalMaxOpen {
		t.Error("Expected max open connections to increase for high load")
	}

	// Reset to original
	db.SetMaxOpenConns(originalMaxOpen)

	// Test low load optimization
	pm.OptimizeForLoad(0.2)
	stats = pm.GetDatabaseStats()
	if stats.MaxOpenConnections >= originalMaxOpen {
		t.Error("Expected max open connections to decrease for low load")
	}

	// Test normal load (should not change much)
	db.SetMaxOpenConns(originalMaxOpen)
	pm.OptimizeForLoad(0.5)
	stats = pm.GetDatabaseStats()
	if stats.MaxOpenConnections != originalMaxOpen {
		t.Error("Expected max open connections to remain unchanged for normal load")
	}
}

func TestPoolManagerClose(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := DefaultPoolConfig()
	pm := NewPoolManager(config, logger)

	// Create and configure database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	err = pm.ConfigureDatabase(db)
	if err != nil {
		t.Fatalf("Failed to configure database: %v", err)
	}

	// Start monitoring
	pm.StartMonitoring()

	// Close pool manager
	err = pm.Close()
	if err != nil {
		t.Errorf("Failed to close pool manager: %v", err)
	}

	// Verify database is closed
	err = pm.PingDatabase(context.Background())
	if err == nil {
		t.Error("Expected database ping to fail after close")
	}
}

// Benchmark tests
func BenchmarkPoolMetricsDBUtilization(b *testing.B) {
	metrics := &PoolMetrics{
		DBOpenConnections: 25,
		DBInUseConns:      15,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetDBUtilization()
	}
}

func BenchmarkPoolMetricsHTTPUtilization(b *testing.B) {
	metrics := &PoolMetrics{
		HTTPActiveConns: 10,
		HTTPIdleConns:   20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetHTTPUtilization()
	}
}