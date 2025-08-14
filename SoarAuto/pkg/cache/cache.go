package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"SoarAuto/pkg/types"
)

// ContextCache provides efficient caching for rule engine contexts and expressions
type ContextCache struct {
	contextCache    map[string]*types.CachedContext
	expressionCache map[string]*types.CachedExpression
	variables       map[string]*types.LazyVariable
	config          *CacheConfig
	stats           *types.CacheStats
	mutex           sync.RWMutex
	cleanupTicker   *time.Ticker
	stopCleanup     chan bool
}

// CacheConfig holds configuration for the context cache
type CacheConfig struct {
	MaxContexts           int           `yaml:"max_contexts"`
	MaxExpressions        int           `yaml:"max_expressions"`
	MaxVariables          int           `yaml:"max_variables"`
	ContextTTL            time.Duration `yaml:"context_ttl"`
	ExpressionTTL         time.Duration `yaml:"expression_ttl"`
	VariableTTL           time.Duration `yaml:"variable_ttl"`
	CleanupInterval       time.Duration `yaml:"cleanup_interval"`
	EnableLazyEval        bool          `yaml:"enable_lazy_eval"`
	EnableExpressionCache bool          `yaml:"enable_expression_cache"`
	MaxFieldSize          int           `yaml:"max_field_size"`
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
		contextCache:    make(map[string]*types.CachedContext),
		expressionCache: make(map[string]*types.CachedExpression),
		variables:       make(map[string]*types.LazyVariable),
		config:          config,
		stats:           &types.CacheStats{},
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
func (cc *ContextCache) GetContext(hash string) (*types.CachedContext, bool) {
	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	context, exists := cc.contextCache[hash]
	if exists {
		// Update access statistics
		context.LastUsed = time.Now()
		context.UseCount++
		cc.stats.ContextHits++
		return context, true
	}

	cc.stats.ContextMisses++
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

	cachedContext := &types.CachedContext{
		Data:      cc.cloneContext(context),
		Hash:      hash,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		UseCount:  1,
		Size:      size,
	}

	cc.contextCache[hash] = cachedContext
	cc.stats.TotalSize += size

	return hash
}

// GetExpression retrieves a cached expression result
func (cc *ContextCache) GetExpression(hash string) (*types.CachedExpression, bool) {
	if !cc.config.EnableExpressionCache {
		return nil, false
	}

	cc.mutex.RLock()
	defer cc.mutex.RUnlock()

	expr, exists := cc.expressionCache[hash]
	if exists && time.Since(expr.CreatedAt) < cc.config.ExpressionTTL {
		expr.LastUsed = time.Now()
		expr.UseCount++
		cc.stats.ExpressionHits++
		return expr, true
	}

	cc.stats.ExpressionMisses++
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

	cachedExpr := &types.CachedExpression{
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
func (cc *ContextCache) GetOrCreateLazyVariable(name, path string, evaluator func() (interface{}, error)) *types.LazyVariable {
	if !cc.config.EnableLazyEval {
		// If lazy evaluation is disabled, evaluate immediately
		result, err := evaluator()
		return &types.LazyVariable{
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
		cc.stats.VariableHits++
		return variable
	}

	// Check if we need to evict old entries
	if len(cc.variables) >= cc.config.MaxVariables {
		cc.evictOldestVariable()
	}

	variable := &types.LazyVariable{
		Name:      name,
		Path:      path,
		Evaluator: evaluator,
		Evaluated: false,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		UseCount:  1,
	}

	cc.variables[key] = variable
	cc.stats.VariableMisses++

	return variable
}

// hashContext creates a hash for a context map
func (cc *ContextCache) hashContext(context map[string]interface{}) string {
	data, _ := json.Marshal(context)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// HashExpression creates a hash for an expression and context combination
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
		cc.stats.EvictedContexts++
		cc.stats.TotalSize -= oldestSize
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
		cc.stats.EvictedExpressions++
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
		cc.stats.EvictedVariables++
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
			cc.stats.TotalSize -= context.Size
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

	cc.stats.CleanupRuns++
}

// GetStats returns current cache statistics
func (cc *ContextCache) GetStats() types.CacheStats {
	// Create a copy to avoid race conditions
	return types.CacheStats{
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

	cc.contextCache = make(map[string]*types.CachedContext)
	cc.expressionCache = make(map[string]*types.CachedExpression)
	cc.variables = make(map[string]*types.LazyVariable)
	
	cc.stats.TotalSize = 0
}

// Close stops the cleanup routine and clears the cache
func (cc *ContextCache) Close() {
	if cc.cleanupTicker != nil {
		cc.cleanupTicker.Stop()
		close(cc.stopCleanup)
	}
	cc.Clear()
}