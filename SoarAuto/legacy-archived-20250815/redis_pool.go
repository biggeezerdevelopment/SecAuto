package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisPool manages a pool of Redis connections
type RedisPool struct {
	client      *redis.Client
	clusterClient *redis.Client
	config      *Config
	mu          sync.RWMutex
	stats       PoolStats
}

// PoolStats tracks connection pool statistics
type PoolStats struct {
	TotalConnections   int64
	ActiveConnections  int64
	IdleConnections    int64
	StaleConnections   int64
	Hits               int64
	Misses             int64
	Timeouts           int64
	LastHealthCheck    time.Time
}

// RedisPoolConfig holds pool configuration
type RedisPoolConfig struct {
	URL            string
	MaxRetries     int
	PoolSize       int
	MinIdleConns   int
	MaxIdleTime    time.Duration
	PoolTimeout    time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	DialTimeout    time.Duration
	MaxConnAge     time.Duration
}

// Global Redis pool instance
var (
	redisPool  *RedisPool
	poolOnce   sync.Once
	poolMutex  sync.RWMutex
)

// InitializeRedisPool initializes the global Redis connection pool
func InitializeRedisPool(config *Config) (*RedisPool, error) {
	var poolErr error
	
	poolOnce.Do(func() {
		pool := &RedisPool{
			config: config,
		}
		
		// Initialize main Redis client with connection pooling
		mainPoolConfig := createPoolConfig(config, false)
		pool.client, poolErr = createPooledClient(mainPoolConfig)
		if poolErr != nil {
			return
		}
		
		// Initialize cluster Redis client if clustering is enabled
		if config.Cluster.Enabled {
			clusterPoolConfig := createPoolConfig(config, true)
			pool.clusterClient, poolErr = createPooledClient(clusterPoolConfig)
			if poolErr != nil {
				// Close main client if cluster client fails
				pool.client.Close()
				return
			}
		}
		
		// Start health check routine
		go pool.healthCheckRoutine()
		
		redisPool = pool
		
		logger.Info("Redis connection pool initialized", map[string]interface{}{
			"component": "redis_pool",
			"pool_size": config.Cluster.RedisPoolSize,
			"main_url": config.Database.RedisURL,
			"cluster_enabled": config.Cluster.Enabled,
		})
	})
	
	if poolErr != nil {
		return nil, poolErr
	}
	
	return redisPool, nil
}

// GetRedisPool returns the global Redis pool instance
func GetRedisPool() *RedisPool {
	poolMutex.RLock()
	defer poolMutex.RUnlock()
	return redisPool
}

// createPoolConfig creates a pool configuration from the app config
func createPoolConfig(config *Config, isCluster bool) *RedisPoolConfig {
	poolConfig := &RedisPoolConfig{
		URL:          config.Database.RedisURL,
		MaxRetries:   3,
		PoolSize:     10,
		MinIdleConns: 2,
		MaxIdleTime:  5 * time.Minute,
		PoolTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		DialTimeout:  5 * time.Second,
		MaxConnAge:   30 * time.Minute,
	}
	
	if isCluster {
		poolConfig.URL = config.Cluster.RedisURL
		if config.Cluster.RedisPoolSize > 0 {
			poolConfig.PoolSize = config.Cluster.RedisPoolSize
		}
		
		// Parse timeout from config string
		if timeout, err := time.ParseDuration(config.Cluster.RedisPoolTimeout); err == nil {
			poolConfig.PoolTimeout = timeout
		}
		
		if idleTimeout, err := time.ParseDuration(config.Cluster.RedisIdleTimeout); err == nil {
			poolConfig.MaxIdleTime = idleTimeout
		}
	}
	
	return poolConfig
}

// createPooledClient creates a Redis client with connection pooling
func createPooledClient(config *RedisPoolConfig) (*redis.Client, error) {
	opts, err := redis.ParseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}
	
	// Configure connection pooling
	opts.PoolSize = config.PoolSize
	opts.MinIdleConns = config.MinIdleConns
	opts.MaxRetries = config.MaxRetries
	opts.PoolTimeout = config.PoolTimeout
	opts.ReadTimeout = config.ReadTimeout
	opts.WriteTimeout = config.WriteTimeout
	opts.DialTimeout = config.DialTimeout
	opts.ConnMaxIdleTime = config.MaxIdleTime
	opts.ConnMaxLifetime = config.MaxConnAge
	
	// Connection pool hooks for statistics
	opts.OnConnect = func(ctx context.Context, cn *redis.Conn) error {
		poolMutex.Lock()
		if redisPool != nil {
			redisPool.stats.TotalConnections++
			redisPool.stats.ActiveConnections++
		}
		poolMutex.Unlock()
		return nil
	}
	
	client := redis.NewClient(opts)
	
	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}
	
	return client, nil
}

// GetClient returns the main Redis client
func (p *RedisPool) GetClient() *redis.Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	p.stats.Hits++
	return p.client
}

// GetClusterClient returns the cluster Redis client
func (p *RedisPool) GetClusterClient() *redis.Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.clusterClient == nil {
		p.stats.Misses++
		return p.client // Fallback to main client
	}
	
	p.stats.Hits++
	return p.clusterClient
}

// healthCheckRoutine performs periodic health checks on the connection pool
func (p *RedisPool) healthCheckRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		p.performHealthCheck()
	}
}

// performHealthCheck checks the health of Redis connections
func (p *RedisPool) performHealthCheck() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Check main client
	if p.client != nil {
		if err := p.client.Ping(ctx).Err(); err != nil {
			logger.Error("Main Redis client health check failed", map[string]interface{}{
				"component": "redis_pool",
				"error": err.Error(),
			})
			
			// Try to reconnect
			if newClient, err := createPooledClient(createPoolConfig(p.config, false)); err == nil {
				p.client.Close()
				p.client = newClient
				logger.Info("Main Redis client reconnected successfully", map[string]interface{}{
					"component": "redis_pool",
				})
			}
		}
		
		// Update pool statistics
		poolStats := p.client.PoolStats()
		p.stats.ActiveConnections = int64(poolStats.TotalConns - poolStats.IdleConns)
		p.stats.IdleConnections = int64(poolStats.IdleConns)
		p.stats.StaleConnections = int64(poolStats.StaleConns)
	}
	
	// Check cluster client
	if p.clusterClient != nil {
		if err := p.clusterClient.Ping(ctx).Err(); err != nil {
			logger.Error("Cluster Redis client health check failed", map[string]interface{}{
				"component": "redis_pool",
				"error": err.Error(),
			})
			
			// Try to reconnect
			if newClient, err := createPooledClient(createPoolConfig(p.config, true)); err == nil {
				p.clusterClient.Close()
				p.clusterClient = newClient
				logger.Info("Cluster Redis client reconnected successfully", map[string]interface{}{
					"component": "redis_pool",
				})
			}
		}
	}
	
	p.stats.LastHealthCheck = time.Now()
}

// GetStats returns current pool statistics
func (p *RedisPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	// Update current stats from client
	if p.client != nil {
		poolStats := p.client.PoolStats()
		p.stats.ActiveConnections = int64(poolStats.TotalConns - poolStats.IdleConns)
		p.stats.IdleConnections = int64(poolStats.IdleConns)
		p.stats.StaleConnections = int64(poolStats.StaleConns)
		p.stats.Timeouts = int64(poolStats.Timeouts)
	}
	
	return p.stats
}

// Close closes all Redis connections in the pool
func (p *RedisPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	var errors []error
	
	if p.client != nil {
		if err := p.client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close main client: %v", err))
		}
	}
	
	if p.clusterClient != nil {
		if err := p.clusterClient.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close cluster client: %v", err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors closing Redis connections: %v", errors)
	}
	
	logger.Info("Redis connection pool closed", map[string]interface{}{
		"component": "redis_pool",
		"total_connections": p.stats.TotalConnections,
		"total_hits": p.stats.Hits,
		"total_misses": p.stats.Misses,
	})
	
	return nil
}

// WithRetry executes a Redis operation with retry logic
func (p *RedisPool) WithRetry(fn func(*redis.Client) error, useCluster bool) error {
	var client *redis.Client
	if useCluster && p.clusterClient != nil {
		client = p.GetClusterClient()
	} else {
		client = p.GetClient()
	}
	
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := fn(client)
		if err == nil {
			return nil
		}
		
		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}
		
		// Exponential backoff
		backoff := time.Duration(i+1) * 100 * time.Millisecond
		time.Sleep(backoff)
		
		logger.Warning("Redis operation failed, retrying", map[string]interface{}{
			"component": "redis_pool",
			"attempt": i + 1,
			"max_retries": maxRetries,
			"error": err.Error(),
		})
	}
	
	return fmt.Errorf("operation failed after %d retries", maxRetries)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for specific Redis errors that are retryable
	if err == redis.ErrClosed {
		return true
	}
	
	// Check error string for network-related issues and pool timeouts
	errStr := err.Error()
	return contains(errStr, "connection refused") ||
		contains(errStr, "connection reset") ||
		contains(errStr, "i/o timeout") ||
		contains(errStr, "network is unreachable") ||
		contains(errStr, "pool timeout") ||
		contains(errStr, "context deadline exceeded")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && s[0:len(s)] != "" && substr != "" && findSubstring(s, substr) != -1)
}

// findSubstring finds a substring in a string
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}