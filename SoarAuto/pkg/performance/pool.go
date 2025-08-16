package performance

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// PoolConfig holds configuration for connection pools
type PoolConfig struct {
	// Database pool configuration
	DBMaxOpenConns    int           `yaml:"db_max_open_conns" json:"db_max_open_conns"`
	DBMaxIdleConns    int           `yaml:"db_max_idle_conns" json:"db_max_idle_conns"`
	DBConnMaxLifetime time.Duration `yaml:"db_conn_max_lifetime" json:"db_conn_max_lifetime"`
	DBConnMaxIdleTime time.Duration `yaml:"db_conn_max_idle_time" json:"db_conn_max_idle_time"`

	// HTTP client pool configuration
	HTTPMaxIdleConns        int           `yaml:"http_max_idle_conns" json:"http_max_idle_conns"`
	HTTPMaxIdleConnsPerHost int           `yaml:"http_max_idle_conns_per_host" json:"http_max_idle_conns_per_host"`
	HTTPMaxConnsPerHost     int           `yaml:"http_max_conns_per_host" json:"http_max_conns_per_host"`
	HTTPIdleConnTimeout     time.Duration `yaml:"http_idle_conn_timeout" json:"http_idle_conn_timeout"`
	HTTPTimeout             time.Duration `yaml:"http_timeout" json:"http_timeout"`
	HTTPKeepAlive           time.Duration `yaml:"http_keep_alive" json:"http_keep_alive"`
	HTTPTLSHandshakeTimeout time.Duration `yaml:"http_tls_handshake_timeout" json:"http_tls_handshake_timeout"`

	// Monitoring configuration
	MonitoringEnabled  bool          `yaml:"monitoring_enabled" json:"monitoring_enabled"`
	MonitoringInterval time.Duration `yaml:"monitoring_interval" json:"monitoring_interval"`
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		// Database defaults
		DBMaxOpenConns:    25,
		DBMaxIdleConns:    5,
		DBConnMaxLifetime: 30 * time.Minute,
		DBConnMaxIdleTime: 5 * time.Minute,

		// HTTP defaults
		HTTPMaxIdleConns:        100,
		HTTPMaxIdleConnsPerHost: 10,
		HTTPMaxConnsPerHost:     50,
		HTTPIdleConnTimeout:     90 * time.Second,
		HTTPTimeout:             30 * time.Second,
		HTTPKeepAlive:           30 * time.Second,
		HTTPTLSHandshakeTimeout: 10 * time.Second,

		// Monitoring defaults
		MonitoringEnabled:  true,
		MonitoringInterval: 30 * time.Second,
	}
}

// PoolMetrics holds connection pool metrics
type PoolMetrics struct {
	mu sync.RWMutex

	// Database metrics
	DBOpenConnections int `json:"db_open_connections"`
	DBInUseConns      int `json:"db_in_use_connections"`
	DBIdleConns       int `json:"db_idle_connections"`
	DBWaitCount       int `json:"db_wait_count"`
	DBWaitDuration    int `json:"db_wait_duration_ms"`
	DBMaxIdleClosed   int `json:"db_max_idle_closed"`
	DBMaxLifetimeClosed int `json:"db_max_lifetime_closed"`

	// HTTP metrics
	HTTPActiveConns   int `json:"http_active_connections"`
	HTTPIdleConns     int `json:"http_idle_connections"`
	HTTPRequestsTotal int `json:"http_requests_total"`
	HTTPErrorsTotal   int `json:"http_errors_total"`

	// General metrics
	LastUpdated time.Time `json:"last_updated"`
}

// GetDBUtilization returns database connection utilization percentage
func (m *PoolMetrics) GetDBUtilization() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.DBOpenConnections
	if total == 0 {
		return 0.0
	}
	return float64(m.DBInUseConns) / float64(total) * 100.0
}

// GetHTTPUtilization returns HTTP connection utilization percentage
func (m *PoolMetrics) GetHTTPUtilization() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.HTTPActiveConns + m.HTTPIdleConns
	if total == 0 {
		return 0.0
	}
	return float64(m.HTTPActiveConns) / float64(total) * 100.0
}

// PoolManager manages connection pools
type PoolManager struct {
	config  *PoolConfig
	logger  *log.Logger
	metrics *PoolMetrics

	// Database pool
	db *sql.DB

	// HTTP client pool
	httpClient *http.Client

	// Monitoring
	stopMonitoring chan struct{}
	monitoringWG   sync.WaitGroup
}

// NewPoolManager creates a new pool manager
func NewPoolManager(config *PoolConfig, logger *log.Logger) *PoolManager {
	return &PoolManager{
		config:         config,
		logger:         logger,
		metrics:        &PoolMetrics{LastUpdated: time.Now()},
		stopMonitoring: make(chan struct{}),
	}
}

// ConfigureDatabase configures the database connection pool
func (pm *PoolManager) ConfigureDatabase(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Configure connection pool settings
	db.SetMaxOpenConns(pm.config.DBMaxOpenConns)
	db.SetMaxIdleConns(pm.config.DBMaxIdleConns)
	db.SetConnMaxLifetime(pm.config.DBConnMaxLifetime)
	db.SetConnMaxIdleTime(pm.config.DBConnMaxIdleTime)

	pm.db = db

	pm.logger.Printf("Database pool configured: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v, MaxIdleTime=%v",
		pm.config.DBMaxOpenConns,
		pm.config.DBMaxIdleConns,
		pm.config.DBConnMaxLifetime,
		pm.config.DBConnMaxIdleTime)

	return nil
}

// GetHTTPClient returns a configured HTTP client with connection pooling
func (pm *PoolManager) GetHTTPClient() *http.Client {
	if pm.httpClient != nil {
		return pm.httpClient
	}

	transport := &http.Transport{
		MaxIdleConns:        pm.config.HTTPMaxIdleConns,
		MaxIdleConnsPerHost: pm.config.HTTPMaxIdleConnsPerHost,
		MaxConnsPerHost:     pm.config.HTTPMaxConnsPerHost,
		IdleConnTimeout:     pm.config.HTTPIdleConnTimeout,
		TLSHandshakeTimeout: pm.config.HTTPTLSHandshakeTimeout,
		DisableKeepAlives:   false,
	}

	pm.httpClient = &http.Client{
		Transport: transport,
		Timeout:   pm.config.HTTPTimeout,
	}

	pm.logger.Printf("HTTP client pool configured: MaxIdle=%d, MaxIdlePerHost=%d, MaxPerHost=%d, Timeout=%v",
		pm.config.HTTPMaxIdleConns,
		pm.config.HTTPMaxIdleConnsPerHost,
		pm.config.HTTPMaxConnsPerHost,
		pm.config.HTTPTimeout)

	return pm.httpClient
}

// StartMonitoring starts connection pool monitoring
func (pm *PoolManager) StartMonitoring() {
	if !pm.config.MonitoringEnabled {
		return
	}

	pm.monitoringWG.Add(1)
	go pm.monitoringLoop()

	pm.logger.Printf("Pool monitoring started (interval: %v)", pm.config.MonitoringInterval)
}

// StopMonitoring stops connection pool monitoring
func (pm *PoolManager) StopMonitoring() {
	if !pm.config.MonitoringEnabled {
		return
	}

	select {
	case <-pm.stopMonitoring:
		// Already closed
		return
	default:
		close(pm.stopMonitoring)
	}
	
	pm.monitoringWG.Wait()

	pm.logger.Println("Pool monitoring stopped")
}

// monitoringLoop runs the monitoring loop
func (pm *PoolManager) monitoringLoop() {
	defer pm.monitoringWG.Done()

	ticker := time.NewTicker(pm.config.MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.updateMetrics()
		case <-pm.stopMonitoring:
			return
		}
	}
}

// updateMetrics updates connection pool metrics
func (pm *PoolManager) updateMetrics() {
	pm.metrics.mu.Lock()
	defer pm.metrics.mu.Unlock()

	// Update database metrics
	if pm.db != nil {
		stats := pm.db.Stats()
		pm.metrics.DBOpenConnections = stats.OpenConnections
		pm.metrics.DBInUseConns = stats.InUse
		pm.metrics.DBIdleConns = stats.Idle
		pm.metrics.DBWaitCount = int(stats.WaitCount)
		pm.metrics.DBWaitDuration = int(stats.WaitDuration.Milliseconds())
		pm.metrics.DBMaxIdleClosed = int(stats.MaxIdleClosed)
		pm.metrics.DBMaxLifetimeClosed = int(stats.MaxLifetimeClosed)
	}

	// Update HTTP metrics (basic tracking)
	// Note: Go's http.Transport doesn't expose detailed connection metrics
	// This would need custom instrumentation for more detailed metrics

	pm.metrics.LastUpdated = time.Now()
}

// GetMetrics returns current pool metrics
func (pm *PoolManager) GetMetrics() *PoolMetrics {
	pm.updateMetrics()
	return pm.metrics
}

// GetDatabaseStats returns database connection statistics
func (pm *PoolManager) GetDatabaseStats() sql.DBStats {
	if pm.db == nil {
		return sql.DBStats{}
	}
	return pm.db.Stats()
}

// PingDatabase tests the database connection
func (pm *PoolManager) PingDatabase(ctx context.Context) error {
	if pm.db == nil {
		return fmt.Errorf("database not configured")
	}
	return pm.db.PingContext(ctx)
}

// Close closes all connections and stops monitoring
func (pm *PoolManager) Close() error {
	pm.StopMonitoring()

	var err error
	if pm.db != nil {
		if closeErr := pm.db.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close database: %w", closeErr)
		}
	}

	// HTTP client connections are managed by the transport
	// and will be closed automatically

	pm.logger.Println("Pool manager closed")
	return err
}

// HealthCheck performs a health check on all pools
func (pm *PoolManager) HealthCheck(ctx context.Context) error {
	// Check database connection
	if pm.db != nil {
		if err := pm.PingDatabase(ctx); err != nil {
			return fmt.Errorf("database health check failed: %w", err)
		}
	}

	// Check HTTP client (basic test)
	if pm.httpClient != nil {
		// HTTP client health is typically checked by making actual requests
		// This is a basic validation that the client is configured
		if pm.httpClient.Transport == nil {
			return fmt.Errorf("HTTP client transport not configured")
		}
	}

	return nil
}

// GetPoolStatus returns a summary of pool status
func (pm *PoolManager) GetPoolStatus() map[string]interface{} {
	metrics := pm.GetMetrics()

	status := map[string]interface{}{
		"database": map[string]interface{}{
			"configured":     pm.db != nil,
			"open_conns":     metrics.DBOpenConnections,
			"in_use_conns":   metrics.DBInUseConns,
			"idle_conns":     metrics.DBIdleConns,
			"utilization":    metrics.GetDBUtilization(),
			"wait_count":     metrics.DBWaitCount,
			"wait_duration":  metrics.DBWaitDuration,
		},
		"http_client": map[string]interface{}{
			"configured":    pm.httpClient != nil,
			"active_conns":  metrics.HTTPActiveConns,
			"idle_conns":    metrics.HTTPIdleConns,
			"utilization":   metrics.GetHTTPUtilization(),
			"total_requests": metrics.HTTPRequestsTotal,
			"total_errors":  metrics.HTTPErrorsTotal,
		},
		"monitoring": map[string]interface{}{
			"enabled":      pm.config.MonitoringEnabled,
			"last_updated": metrics.LastUpdated,
		},
	}

	return status
}

// OptimizeForLoad adjusts pool settings based on current load
func (pm *PoolManager) OptimizeForLoad(currentLoad float64) {
	if pm.db == nil {
		return
	}

	// Simple optimization logic based on load
	// In production, this could be more sophisticated
	if currentLoad > 0.8 {
		// High load - increase connections
		newMaxOpen := int(float64(pm.config.DBMaxOpenConns) * 1.5)
		if newMaxOpen > 100 { // Cap at reasonable limit
			newMaxOpen = 100
		}
		pm.db.SetMaxOpenConns(newMaxOpen)
		pm.logger.Printf("Optimized DB pool for high load: MaxOpen=%d", newMaxOpen)
	} else if currentLoad < 0.3 {
		// Low load - reduce connections to save resources
		newMaxOpen := int(float64(pm.config.DBMaxOpenConns) * 0.8)
		if newMaxOpen < 5 { // Maintain minimum
			newMaxOpen = 5
		}
		pm.db.SetMaxOpenConns(newMaxOpen)
		pm.logger.Printf("Optimized DB pool for low load: MaxOpen=%d", newMaxOpen)
	}
}