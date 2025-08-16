package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheConfig holds configuration for the cache system
type CacheConfig struct {
	RedisAddr     string        `yaml:"redis_addr" json:"redis_addr"`
	RedisPassword string        `yaml:"redis_password" json:"redis_password"`
	RedisDB       int           `yaml:"redis_db" json:"redis_db"`
	DefaultTTL    time.Duration `yaml:"default_ttl" json:"default_ttl"`
	MaxRetries    int           `yaml:"max_retries" json:"max_retries"`
	DialTimeout   time.Duration `yaml:"dial_timeout" json:"dial_timeout"`
	ReadTimeout   time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout  time.Duration `yaml:"write_timeout" json:"write_timeout"`
	PoolSize      int           `yaml:"pool_size" json:"pool_size"`
	MinIdleConns  int           `yaml:"min_idle_conns" json:"min_idle_conns"`
	MaxConnAge    time.Duration `yaml:"max_conn_age" json:"max_conn_age"`
	PoolTimeout   time.Duration `yaml:"pool_timeout" json:"pool_timeout"`
	IdleTimeout   time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	Enabled       bool          `yaml:"enabled" json:"enabled"`
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       0,
		DefaultTTL:    15 * time.Minute,
		MaxRetries:    3,
		DialTimeout:   5 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		PoolSize:      10,
		MinIdleConns:  2,
		MaxConnAge:    30 * time.Minute,
		PoolTimeout:   4 * time.Second,
		IdleTimeout:   5 * time.Minute,
		Enabled:       true,
	}
}

// CacheMetrics holds cache performance metrics
type CacheMetrics struct {
	mu           sync.RWMutex
	Hits         int64     `json:"hits"`
	Misses       int64     `json:"misses"`
	Sets         int64     `json:"sets"`
	Deletes      int64     `json:"deletes"`
	Errors       int64     `json:"errors"`
	LastReset    time.Time `json:"last_reset"`
	TotalLatency int64     `json:"total_latency_ms"`
	Operations   int64     `json:"operations"`
}

// GetHitRate returns the cache hit rate as a percentage
func (m *CacheMetrics) GetHitRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	total := m.Hits + m.Misses
	if total == 0 {
		return 0.0
	}
	return float64(m.Hits) / float64(total) * 100.0
}

// GetAverageLatency returns average operation latency in milliseconds
func (m *CacheMetrics) GetAverageLatency() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.Operations == 0 {
		return 0.0
	}
	return float64(m.TotalLatency) / float64(m.Operations)
}

// Reset resets all metrics
func (m *CacheMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.Hits = 0
	m.Misses = 0
	m.Sets = 0
	m.Deletes = 0
	m.Errors = 0
	m.TotalLatency = 0
	m.Operations = 0
	m.LastReset = time.Now()
}

// CacheInterface defines the cache operations
type CacheInterface interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetJSON(ctx context.Context, key string, dest interface{}) error
	SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Increment(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	GetMetrics() *CacheMetrics
	Close() error
	Ping(ctx context.Context) error
}

// Cache implements the caching functionality
type Cache struct {
	client  *redis.Client
	config  *CacheConfig
	logger  *log.Logger
	metrics *CacheMetrics
}

// NewCache creates a new cache instance
func NewCache(config *CacheConfig, logger *log.Logger) (*Cache, error) {
	if !config.Enabled {
		return &Cache{
			config:  config,
			logger:  logger,
			metrics: &CacheMetrics{LastReset: time.Now()},
		}, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.RedisAddr,
		Password:     config.RedisPassword,
		DB:           config.RedisDB,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		ConnMaxLifetime: config.MaxConnAge,
		PoolTimeout:     config.PoolTimeout,
		ConnMaxIdleTime: config.IdleTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	cache := &Cache{
		client:  client,
		config:  config,
		logger:  logger,
		metrics: &CacheMetrics{LastReset: time.Now()},
	}

	logger.Printf("Cache initialized successfully (Redis: %s)", config.RedisAddr)
	return cache, nil
}

// recordMetrics records operation metrics
func (c *Cache) recordMetrics(operation string, latency time.Duration, err error) {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	c.metrics.Operations++
	c.metrics.TotalLatency += latency.Milliseconds()

	if err != nil {
		c.metrics.Errors++
		return
	}

	switch operation {
	case "get_hit":
		c.metrics.Hits++
	case "get_miss":
		c.metrics.Misses++
	case "set":
		c.metrics.Sets++
	case "delete":
		c.metrics.Deletes++
	}
}

// Get retrieves a value from cache
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	if !c.config.Enabled || c.client == nil {
		return "", fmt.Errorf("cache not enabled")
	}

	start := time.Now()
	result, err := c.client.Get(ctx, key).Result()
	latency := time.Since(start)

	if err == redis.Nil {
		c.recordMetrics("get_miss", latency, nil)
		return "", fmt.Errorf("key not found")
	}

	if err != nil {
		c.recordMetrics("get_miss", latency, err)
		c.logger.Printf("Cache GET error for key %s: %v", key, err)
		return "", err
	}

	c.recordMetrics("get_hit", latency, nil)
	return result, nil
}

// Set stores a value in cache
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !c.config.Enabled || c.client == nil {
		return nil // Silently skip if cache is disabled
	}

	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}

	start := time.Now()
	err := c.client.Set(ctx, key, value, ttl).Err()
	latency := time.Since(start)

	c.recordMetrics("set", latency, err)

	if err != nil {
		c.logger.Printf("Cache SET error for key %s: %v", key, err)
		return err
	}

	return nil
}

// Delete removes a value from cache
func (c *Cache) Delete(ctx context.Context, key string) error {
	if !c.config.Enabled || c.client == nil {
		return nil
	}

	start := time.Now()
	err := c.client.Del(ctx, key).Err()
	latency := time.Since(start)

	c.recordMetrics("delete", latency, err)

	if err != nil {
		c.logger.Printf("Cache DELETE error for key %s: %v", key, err)
		return err
	}

	return nil
}

// Exists checks if a key exists in cache
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	if !c.config.Enabled || c.client == nil {
		return false, nil
	}

	start := time.Now()
	result, err := c.client.Exists(ctx, key).Result()
	latency := time.Since(start)

	c.recordMetrics("exists", latency, err)

	if err != nil {
		c.logger.Printf("Cache EXISTS error for key %s: %v", key, err)
		return false, err
	}

	return result > 0, nil
}

// GetJSON retrieves and unmarshals JSON from cache
func (c *Cache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// SetJSON marshals and stores JSON in cache
func (c *Cache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return c.Set(ctx, key, string(data), ttl)
}

// Increment increments a counter in cache
func (c *Cache) Increment(ctx context.Context, key string) (int64, error) {
	if !c.config.Enabled || c.client == nil {
		return 0, fmt.Errorf("cache not enabled")
	}

	start := time.Now()
	result, err := c.client.Incr(ctx, key).Result()
	latency := time.Since(start)

	c.recordMetrics("increment", latency, err)

	if err != nil {
		c.logger.Printf("Cache INCREMENT error for key %s: %v", key, err)
		return 0, err
	}

	return result, nil
}

// Expire sets TTL for a key
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if !c.config.Enabled || c.client == nil {
		return nil
	}

	start := time.Now()
	err := c.client.Expire(ctx, key, ttl).Err()
	latency := time.Since(start)

	c.recordMetrics("expire", latency, err)

	if err != nil {
		c.logger.Printf("Cache EXPIRE error for key %s: %v", key, err)
		return err
	}

	return nil
}

// GetMetrics returns cache metrics
func (c *Cache) GetMetrics() *CacheMetrics {
	return c.metrics
}

// Close closes the cache connection
func (c *Cache) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Ping tests the cache connection
func (c *Cache) Ping(ctx context.Context) error {
	if !c.config.Enabled || c.client == nil {
		return fmt.Errorf("cache not enabled")
	}

	return c.client.Ping(ctx).Err()
}

// CacheKeyBuilder helps build consistent cache keys
type CacheKeyBuilder struct {
	prefix string
}

// NewCacheKeyBuilder creates a new cache key builder
func NewCacheKeyBuilder(prefix string) *CacheKeyBuilder {
	return &CacheKeyBuilder{prefix: prefix}
}

// Build builds a cache key with the given components
func (b *CacheKeyBuilder) Build(components ...string) string {
	key := b.prefix
	for _, component := range components {
		key += ":" + component
	}
	return key
}

// PlaybookKey builds a cache key for playbook data
func (b *CacheKeyBuilder) PlaybookKey(playbookName string) string {
	return b.Build("playbook", playbookName)
}

// JobKey builds a cache key for job data
func (b *CacheKeyBuilder) JobKey(jobID string) string {
	return b.Build("job", jobID)
}

// UserSessionKey builds a cache key for user session data
func (b *CacheKeyBuilder) UserSessionKey(userID string) string {
	return b.Build("session", userID)
}

// APIResponseKey builds a cache key for API response data
func (b *CacheKeyBuilder) APIResponseKey(endpoint, params string) string {
	return b.Build("api", endpoint, params)
}

// ConfigKey builds a cache key for configuration data
func (b *CacheKeyBuilder) ConfigKey(configType string) string {
	return b.Build("config", configType)
}