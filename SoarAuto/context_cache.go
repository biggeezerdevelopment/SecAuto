package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ContextCache provides efficient caching for rule engine contexts and expressions
type ContextCache struct {
	contextCache    map[string]*CachedContext
	expressionCache map[string]*CachedExpression
	variables       map[string]*LazyVariable
	config          *CacheConfig
	stats           *CacheStats
	mutex           sync.RWMutex
	cleanupTicker   *time.Ticker
	stopCleanup     chan bool
}

// CacheConfig holds configuration for the context cache
type CacheConfig struct {
	MaxContexts         int           `yaml:"max_contexts"`
	MaxExpressions      int           `yaml:"max_expressions"`
	MaxVariables        int           `yaml:"max_variables"`
	ContextTTL          time.Duration `yaml:"context_ttl"`
	ExpressionTTL       time.Duration `yaml:"expression_ttl"`
	VariableTTL         time.Duration `yaml:"variable_ttl"`
	CleanupInterval     time.Duration `yaml:"cleanup_interval"`
	EnableLazyEval      bool          `yaml:"enable_lazy_eval"`
	EnableExpressionCache bool        `yaml:"enable_expression_cache"`
	MaxFieldSize        int           `yaml:"max_field_size"`
}

// CachedContext represents a cached context with metadata
type CachedContext struct {
	Data      map[string]interface{}
	Hash      string
	CreatedAt time.Time
	LastUsed  time.Time
	UseCount  int64
	Size      int64
}

// CachedExpression represents a cached expression evaluation result
type CachedExpression struct {
	Result    interface{}
	Hash      string
	CreatedAt time.Time
	LastUsed  time.Time
	UseCount  int64
	Error     string
}

// LazyVariable represents a variable that is evaluated only when needed
type LazyVariable struct {
	Name      string
	Path      string
	Evaluator func() (interface{}, error)
	Result    interface{}
	Error     error
	Evaluated bool
	CreatedAt time.Time
	LastUsed  time.Time
	UseCount  int64
	mutex     sync.RWMutex
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	ContextHits        int64
	ContextMisses      int64
	ExpressionHits     int64
	ExpressionMisses   int64
	VariableHits       int64
	VariableMisses     int64
	EvictedContexts    int64
	EvictedExpressions int64
	EvictedVariables   int64
	TotalSize          int64
	CleanupRuns        int64
	mutex              sync.RWMutex
}

// NewContextCache creates a new context cache with the given configuration
func NewContextCache(config *CacheConfig) *ContextCache {
	if config == nil {
		config = &CacheConfig{
			MaxContexts:           1000,
			MaxExpressions:        5000,
			MaxVariables:          2000,
			ContextTTL:            30 * time.Minute,
			ExpressionTTL:         15 * time.Minute,
			VariableTTL:           10 * time.Minute,
			CleanupInterval:       5 * time.Minute,
			EnableLazyEval:        true,
			EnableExpressionCache: true,
			MaxFieldSize:          10000,
		}
	}

	cache := &ContextCache{
		contextCache:    make(map[string]*CachedContext),
		expressionCache: make(map[string]*CachedExpression),
		variables:       make(map[string]*LazyVariable),
		config:          config,
		stats:           &CacheStats{},
		stopCleanup:     make(chan bool),
	}

	// Start cleanup routine
	if config.CleanupInterval > 0 {
		cache.cleanupTicker = time.NewTicker(config.CleanupInterval)
		go cache.cleanupRoutine()
	}

	return cache
}

// GetContext retrieves a cached context by hash
func (cc *ContextCache) GetContext(hash string) (*CachedContext, bool) {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	context, exists := cc.contextCache[hash]
	if exists {
		// Update access statistics
		context.LastUsed = time.Now()
		context.UseCount++
		cc.stats.mutex.Lock()
		cc.stats.ContextHits++
		cc.stats.mutex.Unlock()
		return context, true
	}

	cc.stats.mutex.Lock()
	cc.stats.ContextMisses++
	cc.stats.mutex.Unlock()
	return nil, false
}

// StoreContext stores a context in the cache
func (cc *ContextCache) StoreContext(context map[string]interface{}) string {
	hash := cc.hashContext(context)
	size := cc.calculateContextSize(context)

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	// Check if we need to evict old entries
	if len(cc.contextCache) >= cc.config.MaxContexts {
		cc.evictOldestContext()
	}

	cachedContext := &CachedContext{
		Data:      cc.cloneContext(context),
		Hash:      hash,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		UseCount:  1,
		Size:      size,
	}

	cc.contextCache[hash] = cachedContext
	cc.stats.mutex.Lock()
	cc.stats.TotalSize += size
	cc.stats.mutex.Unlock()

	return hash
}

// GetExpression retrieves a cached expression result
func (cc *ContextCache) GetExpression(hash string) (*CachedExpression, bool) {
	if !cc.config.EnableExpressionCache {
		return nil, false
	}

	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	expr, exists := cc.expressionCache[hash]
	if exists && time.Since(expr.CreatedAt) < cc.config.ExpressionTTL {
		expr.LastUsed = time.Now()
		expr.UseCount++
		cc.stats.mutex.Lock()
		cc.stats.ExpressionHits++
		cc.stats.mutex.Unlock()
		return expr, true
	}

	cc.stats.mutex.Lock()
	cc.stats.ExpressionMisses++
	cc.stats.mutex.Unlock()
	return nil, false
}

// StoreExpression stores an expression evaluation result
func (cc *ContextCache) StoreExpression(exprHash string, result interface{}, err error) {
	if !cc.config.EnableExpressionCache {
		return
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	// Check if we need to evict old entries
	if len(cc.expressionCache) >= cc.config.MaxExpressions {
		cc.evictOldestExpression()
	}

	errorStr := ""
	if err != nil {
		errorStr = err.Error()
	}

	cachedExpr := &CachedExpression{
		Result:    result,
		Hash:      exprHash,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		UseCount:  1,
		Error:     errorStr,
	}

	cc.expressionCache[exprHash] = cachedExpr
}

// GetOrCreateLazyVariable gets or creates a lazy variable
func (cc *ContextCache) GetOrCreateLazyVariable(name, path string, evaluator func() (interface{}, error)) *LazyVariable {
	if !cc.config.EnableLazyEval {
		// If lazy evaluation is disabled, evaluate immediately
		result, err := evaluator()
		return &LazyVariable{
			Name:      name,
			Path:      path,
			Result:    result,
			Error:     err,
			Evaluated: true,
			CreatedAt: time.Now(),
			LastUsed:  time.Now(),
			UseCount:  1,
		}
	}

	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", name, path)
	if variable, exists := cc.variables[key]; exists {
		variable.LastUsed = time.Now()
		variable.UseCount++
		cc.stats.mutex.Lock()
		cc.stats.VariableHits++
		cc.stats.mutex.Unlock()
		return variable
	}

	// Check if we need to evict old entries
	if len(cc.variables) >= cc.config.MaxVariables {
		cc.evictOldestVariable()
	}

	variable := &LazyVariable{
		Name:      name,
		Path:      path,
		Evaluator: evaluator,
		Evaluated: false,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		UseCount:  1,
	}

	cc.variables[key] = variable
	cc.stats.mutex.Lock()
	cc.stats.VariableMisses++
	cc.stats.mutex.Unlock()

	return variable
}

// Evaluate evaluates a lazy variable if not already evaluated
func (lv *LazyVariable) Evaluate() (interface{}, error) {
	lv.mutex.Lock()
	defer lv.mutex.Unlock()

	if lv.Evaluated {
		lv.LastUsed = time.Now()
		lv.UseCount++
		return lv.Result, lv.Error
	}

	if lv.Evaluator != nil {
		lv.Result, lv.Error = lv.Evaluator()
		lv.Evaluated = true
		lv.LastUsed = time.Now()
		lv.UseCount++
	}

	return lv.Result, lv.Error
}

// hashContext creates a hash for a context map
func (cc *ContextCache) hashContext(context map[string]interface{}) string {
	data, _ := json.Marshal(context)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// hashExpression creates a hash for an expression and context combination
func (cc *ContextCache) HashExpression(expr interface{}, contextHash string) string {
	exprData, _ := json.Marshal(expr)
	combined := fmt.Sprintf("%s:%s", string(exprData), contextHash)
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%x", hash)
}

// calculateContextSize estimates the memory size of a context
func (cc *ContextCache) calculateContextSize(context map[string]interface{}) int64 {
	data, _ := json.Marshal(context)
	return int64(len(data))
}

// cloneContext creates a deep copy of a context
func (cc *ContextCache) cloneContext(context map[string]interface{}) map[string]interface{} {
	clone := make(map[string]interface{})
	for k, v := range context {
		// Simple clone - for more complex types, implement deeper cloning
		clone[k] = v
	}
	return clone
}

// evictOldestContext removes the least recently used context
func (cc *ContextCache) evictOldestContext() {
	var oldestKey string
	var oldestTime time.Time
	var oldestSize int64

	for key, context := range cc.contextCache {
		if oldestKey == "" || context.LastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = context.LastUsed
			oldestSize = context.Size
		}
	}

	if oldestKey != "" {
		delete(cc.contextCache, oldestKey)
		cc.stats.mutex.Lock()
		cc.stats.EvictedContexts++
		cc.stats.TotalSize -= oldestSize
		cc.stats.mutex.Unlock()
	}
}

// evictOldestExpression removes the least recently used expression
func (cc *ContextCache) evictOldestExpression() {
	var oldestKey string
	var oldestTime time.Time

	for key, expr := range cc.expressionCache {
		if oldestKey == "" || expr.LastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = expr.LastUsed
		}
	}

	if oldestKey != "" {
		delete(cc.expressionCache, oldestKey)
		cc.stats.mutex.Lock()
		cc.stats.EvictedExpressions++
		cc.stats.mutex.Unlock()
	}
}

// evictOldestVariable removes the least recently used variable
func (cc *ContextCache) evictOldestVariable() {
	var oldestKey string
	var oldestTime time.Time

	for key, variable := range cc.variables {
		if oldestKey == "" || variable.LastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = variable.LastUsed
		}
	}

	if oldestKey != "" {
		delete(cc.variables, oldestKey)
		cc.stats.mutex.Lock()
		cc.stats.EvictedVariables++
		cc.stats.mutex.Unlock()
	}
}

// cleanupRoutine periodically cleans up expired cache entries
func (cc *ContextCache) cleanupRoutine() {
	for {
		select {
		case <-cc.cleanupTicker.C:
			cc.cleanup()
		case <-cc.stopCleanup:
			return
		}
	}
}

// cleanup removes expired cache entries
func (cc *ContextCache) cleanup() {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	now := time.Now()
	
	// Cleanup contexts
	for key, context := range cc.contextCache {
		if now.Sub(context.LastUsed) > cc.config.ContextTTL {
			cc.stats.mutex.Lock()
			cc.stats.TotalSize -= context.Size
			cc.stats.mutex.Unlock()
			delete(cc.contextCache, key)
		}
	}

	// Cleanup expressions
	for key, expr := range cc.expressionCache {
		if now.Sub(expr.LastUsed) > cc.config.ExpressionTTL {
			delete(cc.expressionCache, key)
		}
	}

	// Cleanup variables
	for key, variable := range cc.variables {
		if now.Sub(variable.LastUsed) > cc.config.VariableTTL {
			delete(cc.variables, key)
		}
	}

	cc.stats.mutex.Lock()
	cc.stats.CleanupRuns++
	cc.stats.mutex.Unlock()
}

// GetStats returns current cache statistics
func (cc *ContextCache) GetStats() CacheStats {
	cc.stats.mutex.RLock()
	defer cc.stats.mutex.RUnlock()
	
	// Create a copy to avoid race conditions
	return CacheStats{
		ContextHits:        cc.stats.ContextHits,
		ContextMisses:      cc.stats.ContextMisses,
		ExpressionHits:     cc.stats.ExpressionHits,
		ExpressionMisses:   cc.stats.ExpressionMisses,
		VariableHits:       cc.stats.VariableHits,
		VariableMisses:     cc.stats.VariableMisses,
		EvictedContexts:    cc.stats.EvictedContexts,
		EvictedExpressions: cc.stats.EvictedExpressions,
		EvictedVariables:   cc.stats.EvictedVariables,
		TotalSize:          cc.stats.TotalSize,
		CleanupRuns:        cc.stats.CleanupRuns,
	}
}

// Clear clears all cache entries
func (cc *ContextCache) Clear() {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	cc.contextCache = make(map[string]*CachedContext)
	cc.expressionCache = make(map[string]*CachedExpression)
	cc.variables = make(map[string]*LazyVariable)
	
	cc.stats.mutex.Lock()
	cc.stats.TotalSize = 0
	cc.stats.mutex.Unlock()
}

// Close stops the cleanup routine and clears the cache
func (cc *ContextCache) Close() {
	if cc.cleanupTicker != nil {
		cc.cleanupTicker.Stop()
		close(cc.stopCleanup)
	}
	cc.Clear()
}