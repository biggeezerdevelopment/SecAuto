package rules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"SoarAuto/pkg/cache"
	"SoarAuto/pkg/config"
	"SoarAuto/pkg/types"
)

// Engine represents the SOAR rules engine
type Engine struct {
	config        *config.Config
	context       map[string]interface{}
	contextHash   string
	pluginManager types.PlatformPluginManager
	cache         types.ContextCache
	
	// Pre-compiled regular expressions for better performance
	templateVarRegex    *regexp.Regexp
	templateStringRegex *regexp.Regexp
}

// NewEngine creates a new rule engine instance
func NewEngine(cfg *config.Config) *Engine {
	// Create cache configuration from rules engine config
	cacheConfig := &cache.CacheConfig{
		MaxContexts:           cfg.RulesEngine.Caching.MaxContexts,
		MaxExpressions:        cfg.RulesEngine.Caching.MaxExpressions,
		MaxVariables:          cfg.RulesEngine.Caching.MaxVariables,
		EnableLazyEval:        cfg.RulesEngine.Caching.EnableLazyEval,
		EnableExpressionCache: cfg.RulesEngine.Caching.EnableExpressionCache,
		MaxFieldSize:          cfg.RulesEngine.Caching.MaxFieldSize,
	}
	
	// Parse duration strings
	if contextTTL, err := time.ParseDuration(cfg.RulesEngine.Caching.ContextTTL); err == nil {
		cacheConfig.ContextTTL = contextTTL
	} else {
		cacheConfig.ContextTTL = 30 * time.Minute // Default
	}
	
	if exprTTL, err := time.ParseDuration(cfg.RulesEngine.Caching.ExpressionTTL); err == nil {
		cacheConfig.ExpressionTTL = exprTTL
	} else {
		cacheConfig.ExpressionTTL = 15 * time.Minute // Default
	}
	
	if varTTL, err := time.ParseDuration(cfg.RulesEngine.Caching.VariableTTL); err == nil {
		cacheConfig.VariableTTL = varTTL
	} else {
		cacheConfig.VariableTTL = 10 * time.Minute // Default
	}
	
	if cleanupInterval, err := time.ParseDuration(cfg.RulesEngine.Caching.CleanupInterval); err == nil {
		cacheConfig.CleanupInterval = cleanupInterval
	} else {
		cacheConfig.CleanupInterval = 5 * time.Minute // Default
	}

	return &Engine{
		config:        cfg,
		context:       make(map[string]interface{}),
		pluginManager: nil, // Will be set by SetPluginManager
		cache:         cache.NewContextCache(cacheConfig),
		
		// Pre-compile regular expressions for better performance
		templateVarRegex:    regexp.MustCompile(`^\{\{([^}]+)\}\}$`),
		templateStringRegex: regexp.MustCompile(`\{\{([^}]+)\}\}`),
	}
}

// SetContext sets the context for the rule engine with caching support
func (re *Engine) SetContext(context map[string]interface{}) {
	// Check if we already have this context cached
	contextHash := re.cache.StoreContext(context)
	if cachedContext, exists := re.cache.GetContext(contextHash); exists {
		// Use cached context
		re.context = cachedContext.Data
		re.contextHash = contextHash
		return
	}

	// Store new context in cache and use it
	re.contextHash = re.cache.StoreContext(context)
	re.context = context
}

// GetContext returns the current context
func (re *Engine) GetContext() map[string]interface{} {
	return re.context
}

// GetCacheStats returns cache statistics
func (re *Engine) GetCacheStats() types.CacheStats {
	return re.cache.GetStats()
}

// ClearCache clears the cache
func (re *Engine) ClearCache() {
	re.cache.Clear()
}

// Close cleans up resources
func (re *Engine) Close() {
	if re.cache != nil {
		re.cache.Close()
	}
}

// SetPluginManager sets the plugin manager
func (re *Engine) SetPluginManager(pluginManager types.PlatformPluginManager) {
	re.pluginManager = pluginManager
}

// EvaluateRule evaluates a single rule
func (re *Engine) EvaluateRule(rule interface{}) (interface{}, error) {
	return re.evaluate(rule, re.context)
}

// EvaluatePlaybook evaluates a playbook (sequence of rules)
func (re *Engine) EvaluatePlaybook(playbook []interface{}) ([]interface{}, error) {
	var results []interface{}
	
	// Create a mutable copy of the context for this playbook execution
	playbookContext := make(map[string]interface{})
	for k, v := range re.context {
		playbookContext[k] = v
	}
	
	for i, rule := range playbook {
		result, err := re.evaluate(rule, playbookContext)
		if err != nil {
			return results, fmt.Errorf("error in rule %d: %v", i, err)
		}
		
		results = append(results, result)
		
		// Handle special rule results that modify context
		if ruleMap, ok := rule.(map[string]interface{}); ok {
			// If this is a 'var' operation, update the context
			if varName, exists := ruleMap["var"]; exists {
				if varNameStr, ok := varName.(string); ok {
					playbookContext[varNameStr] = result
				}
			}
			
			// If this is a 'run' operation, merge script results into context
			if _, isRunOperation := ruleMap["run"]; isRunOperation {
				if resultMap, ok := result.(map[string]interface{}); ok {
					// Merge all keys from script result into playbook context
					for k, v := range resultMap {
						playbookContext[k] = v
					}
				}
			}
		}
	}
	
	return results, nil
}

// evaluate processes a rule or expression
func (re *Engine) evaluate(expr interface{}, data map[string]interface{}) (interface{}, error) {
	// Check expression cache
	exprHash := re.cache.HashExpression(expr, re.contextHash)
	if cachedExpr, exists := re.cache.GetExpression(exprHash); exists {
		if cachedExpr.Error != "" {
			return cachedExpr.Result, fmt.Errorf(cachedExpr.Error)
		}
		return cachedExpr.Result, nil
	}

	// Process template variables first
	processedExpr := re.processTemplateVariables(expr, data)
	
	result, err := re.evaluateExpression(processedExpr, data)
	
	// Store result in cache
	re.cache.StoreExpression(exprHash, result, err)
	
	return result, err
}

// evaluateExpression evaluates the processed expression
func (re *Engine) evaluateExpression(processedExpr interface{}, data map[string]interface{}) (interface{}, error) {
	switch expr := processedExpr.(type) {
	case string:
		// Simple string value - process any remaining template variables
		return re.processStringTemplate(expr, data), nil
	case map[string]interface{}:
		// Check for different operation types
		for key, value := range expr {
			switch key {
			case "run":
				return re.evaluateRunOperation(value, expr, data)
			case "play":
				return re.evaluatePlayOperation(value, data)
			case "plugin":
				return re.evaluatePluginOperation(value, data)
			case "if":
				return re.evaluateIfOperation(value, data)
			case "var":
				return re.evaluateVarOperation(value, data)
			default:
				// Check for comparison or logical operations
				if re.isComparisonOp(key) {
					return re.evaluateComparison(expr, key, data)
				}
				if re.isLogicalOp(key) {
					return re.evaluateLogical(expr, key, data)
				}
			}
		}
		// If no specific operation, return the map as-is
		return expr, nil
	case []interface{}:
		// Array - evaluate each element
		var results []interface{}
		for _, item := range expr {
			result, err := re.evaluate(item, data)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return results, nil
	default:
		// Primitive value, return as-is
		return processedExpr, nil
	}
}

// isComparisonOp checks if a key is a comparison operator
func (re *Engine) isComparisonOp(key string) bool {
	compOps := []string{"==", "!=", "<", ">", "<=", ">=", "eq", "ne", "lt", "gt", "lte", "gte", "in", "contains", "matches"}
	for _, op := range compOps {
		if key == op {
			return true
		}
	}
	return false
}

// isLogicalOp checks if a key is a logical operator
func (re *Engine) isLogicalOp(key string) bool {
	logOps := []string{"and", "or", "not"}
	for _, op := range logOps {
		if key == op {
			return true
		}
	}
	return false
}

// evaluateRunOperation handles 'run' operations
func (re *Engine) evaluateRunOperation(scriptName interface{}, operation map[string]interface{}, data map[string]interface{}) (interface{}, error) {
	// Convert scriptName to string
	scriptNameStr, ok := scriptName.(string)
	if !ok {
		return nil, fmt.Errorf("script name must be a string")
	}

	// Process template variables in script name
	processedScriptName := re.processStringTemplate(scriptNameStr, data)
	
	// Get full path to script
	scriptPath := re.getScriptPath(processedScriptName)
	
	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("script not found: %s", scriptPath)
	}
	
	// Process additional parameters from the operation and merge with context
	enhancedData := make(map[string]interface{})
	
	// Copy original context
	for k, v := range data {
		enhancedData[k] = v
	}
	
	// Process additional parameters (everything except "run")
	for key, value := range operation {
		if key != "run" {
			// Process template variables in the parameter value
			processedValue := re.processTemplateVariables(value, data)
			enhancedData[key] = processedValue
		}
	}
	
	// Execute the script with enhanced context
	return re.executeScript(scriptPath, enhancedData)
}

// evaluatePlayOperation handles 'play' operations
func (re *Engine) evaluatePlayOperation(playbookName interface{}, data map[string]interface{}) (interface{}, error) {
	playbookNameStr, ok := playbookName.(string)
	if !ok {
		return nil, fmt.Errorf("playbook name must be a string")
	}

	// Process template variables
	processedPlaybookName := re.processStringTemplate(playbookNameStr, data)
	
	// Auto-append .json extension if not present
	if !strings.HasSuffix(processedPlaybookName, ".json") {
		processedPlaybookName += ".json"
	}
	
	// Load and execute playbook
	playbook, err := re.LoadPlaybookFromFile(re.getPlaybookPath(processedPlaybookName))
	if err != nil {
		return nil, fmt.Errorf("failed to load playbook %s: %v", processedPlaybookName, err)
	}
	
	return re.EvaluatePlaybook(playbook)
}

// evaluatePluginOperation handles 'plugin' operations
func (re *Engine) evaluatePluginOperation(pluginExpr interface{}, data map[string]interface{}) (interface{}, error) {
	pluginMap, ok := pluginExpr.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("plugin expression must be an object")
	}

	pluginName, exists := pluginMap["name"]
	if !exists {
		return nil, fmt.Errorf("plugin name is required")
	}

	pluginNameStr, ok := pluginName.(string)
	if !ok {
		return nil, fmt.Errorf("plugin name must be a string")
	}

	// Process template variables
	processedPluginName := re.processStringTemplate(pluginNameStr, data)

	// Get plugin parameters
	pluginParams := data
	if params, exists := pluginMap["params"]; exists {
		if paramsMap, ok := params.(map[string]interface{}); ok {
			// Merge plugin params with context data
			mergedParams := make(map[string]interface{})
			for k, v := range data {
				mergedParams[k] = v
			}
			for k, v := range paramsMap {
				mergedParams[k] = re.processTemplateVariables(v, data)
			}
			pluginParams = mergedParams
		}
	}

	// Execute plugin if plugin manager is available
	if re.pluginManager != nil {
		return re.pluginManager.ExecutePlugin(processedPluginName, pluginParams)
	}

	// Fallback if no plugin manager
	return map[string]interface{}{
		"plugin":    processedPluginName,
		"params":    pluginParams,
		"executed":  false,
		"message":   "Plugin manager not available",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// LoadPlaybookFromFile loads a playbook from a file
func (re *Engine) LoadPlaybookFromFile(filename string) ([]interface{}, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var playbook []interface{}
	if err := json.Unmarshal(data, &playbook); err != nil {
		return nil, err
	}

	return playbook, nil
}

// getScriptPath returns the full path to a script
func (re *Engine) getScriptPath(scriptName string) string {
	return re.config.GetScriptPath(scriptName)
}

// getPlaybookPath returns the full path to a playbook
func (re *Engine) getPlaybookPath(playbookName string) string {
	return re.config.GetPlaybookPath(playbookName)
}

// Additional helper methods would be implemented here...
// Due to space constraints, I'm including the most critical parts
// The full implementation would include all methods from the original rules_engine.go

// processTemplateVariables processes {{variable}} syntax in strings
func (re *Engine) processTemplateVariables(value interface{}, data map[string]interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// Check if the string is exactly a template variable (e.g. "{{threat_intelligence.domains}}")
		if matches := re.templateVarRegex.FindStringSubmatch(v); len(matches) == 2 {
			variableName := strings.TrimSpace(matches[1])
			
			// Try direct lookup
			if resolved, exists := data[variableName]; exists {
				return resolved
			}
			// Try dot notation
			if resolved, err := re.evaluateDotNotation(variableName, data); err == nil && resolved != nil {
				return resolved
			}
			
			// Return the original string if can't resolve
			return v
		}
		
		// Process template strings like "Hello {{name}}"
		return re.processStringTemplate(v, data)
	case map[string]interface{}:
		// Recursively process map values
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = re.processTemplateVariables(val, data)
		}
		return result
	case []interface{}:
		// Recursively process array elements
		var result []interface{}
		for _, item := range v {
			result = append(result, re.processTemplateVariables(item, data))
		}
		return result
	default:
		// Return primitive values as-is
		return value
	}
}

// processStringTemplate processes {{variable}} syntax in a string
func (re *Engine) processStringTemplate(template string, data map[string]interface{}) string {
	// Use pre-compiled regex to match {{variable}} patterns
	return re.templateStringRegex.ReplaceAllStringFunc(template, func(match string) string {
		// Extract variable name from {{variable}}
		variableName := strings.TrimSpace(match[2 : len(match)-2])

		// First try direct lookup in context
		if value, exists := data[variableName]; exists {
			if strValue, ok := value.(string); ok {
				return strValue
			}
			// Convert non-string values to string
			return fmt.Sprintf("%v", value)
		}

		// Try dot notation
		if resolved, err := re.evaluateDotNotation(variableName, data); err == nil {
			if strValue, ok := resolved.(string); ok {
				return strValue
			}
			return fmt.Sprintf("%v", resolved)
		}

		// Return original if not found
		return match
	})
}

// evaluateDotNotation evaluates dot notation like "user.name"
func (re *Engine) evaluateDotNotation(path string, data map[string]interface{}) (interface{}, error) {
	parts := strings.Split(path, ".")
	var current interface{} = data
	
	for i, part := range parts {
		if current == nil {
			return nil, fmt.Errorf("cannot access property '%s' of null at path '%s'", part, strings.Join(parts[:i], "."))
		}
		
		switch v := current.(type) {
		case map[string]interface{}:
			if val, exists := v[part]; exists {
				if i == len(parts)-1 {
					// Last part, return the value
					return val, nil
				} else {
					// Continue traversal
					if nextMap, ok := val.(map[string]interface{}); ok {
						current = nextMap
					} else {
						return nil, fmt.Errorf("cannot access property '%s' of non-object at path '%s'", parts[i+1], strings.Join(parts[:i+1], "."))
					}
				}
			} else {
				return nil, fmt.Errorf("property '%s' not found at path '%s'", part, strings.Join(parts[:i+1], "."))
			}
		default:
			return nil, fmt.Errorf("cannot access property '%s' of non-object", part)
		}
	}
	
	return nil, fmt.Errorf("empty path")
}

// Additional methods for completeness (abbreviated for space)
// These would include all the comparison, logical operations, etc.

// evaluateIfOperation handles 'if' operations
func (re *Engine) evaluateIfOperation(ifExpr interface{}, data map[string]interface{}) (interface{}, error) {
	ifMap, ok := ifExpr.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("if expression must be an object")
	}

	// Get condition
	condition, exists := ifMap["condition"]
	if !exists {
		return nil, fmt.Errorf("if expression must have a condition")
	}

	// Evaluate condition
	conditionResult, err := re.evaluate(condition, data)
	if err != nil {
		return nil, fmt.Errorf("error evaluating condition: %v", err)
	}

	// Convert to boolean
	conditionBool := re.isTruthy(conditionResult)

	if conditionBool {
		// Execute 'then' branch
		if thenExpr, exists := ifMap["then"]; exists {
			return re.evaluate(thenExpr, data)
		}
		return true, nil
	} else {
		// Execute 'else' branch if it exists
		if elseExpr, exists := ifMap["else"]; exists {
			return re.evaluate(elseExpr, data)
		}
		return false, nil
	}
}

// evaluateVarOperation handles 'var' operations  
func (re *Engine) evaluateVarOperation(varName interface{}, data map[string]interface{}) (interface{}, error) {
	varNameStr, ok := varName.(string)
	if !ok {
		return nil, fmt.Errorf("variable name must be a string")
	}

	// Use lazy evaluation if enabled
	if re.cache != nil {
		lazyVar := re.cache.GetOrCreateLazyVariable(varNameStr, varNameStr, func() (interface{}, error) {
			return re.evaluateDotNotation(varNameStr, data)
		})
		
		return lazyVar.Evaluate()
	}

	// Direct evaluation
	return re.evaluateDotNotation(varNameStr, data)
}

// evaluateComparison handles comparison operations
func (re *Engine) evaluateComparison(operation map[string]interface{}, op string, data map[string]interface{}) (bool, error) {
	left, leftExists := operation["left"]
	right, rightExists := operation["right"]

	if !leftExists || !rightExists {
		// Try to get values directly from the operation
		for key, value := range operation {
			if key == op {
				continue // Skip the operator key
			}
			if left == nil {
				left = key
			} else if right == nil {
				right = value
				break
			}
		}
	}

	if left == nil || right == nil {
		return false, fmt.Errorf("comparison requires both left and right operands")
	}

	// Evaluate operands
	leftResult, err := re.evaluate(left, data)
	if err != nil {
		return false, err
	}

	rightResult, err := re.evaluate(right, data)
	if err != nil {
		return false, err
	}

	// Perform comparison
	return re.compareValues(leftResult, rightResult, op)
}

// evaluateLogical handles logical operations
func (re *Engine) evaluateLogical(operation map[string]interface{}, op string, data map[string]interface{}) (interface{}, error) {
	switch op {
	case "and":
		operands, exists := operation["and"]
		if !exists {
			return false, fmt.Errorf("and operation requires operands")
		}
		return re.evaluateAnd(operands, data)
		
	case "or":
		operands, exists := operation["or"]
		if !exists {
			return false, fmt.Errorf("or operation requires operands")
		}
		return re.evaluateOr(operands, data)
		
	case "not":
		operand, exists := operation["not"]
		if !exists {
			return false, fmt.Errorf("not operation requires operand")
		}
		return re.evaluateNot(operand, data)
		
	default:
		return false, fmt.Errorf("unknown logical operator: %s", op)
	}
}

// compareValues compares two values using the specified operator
func (re *Engine) compareValues(left, right interface{}, op string) (bool, error) {
	switch op {
	case "==", "===", "eq":
		return re.deepEqual(left, right), nil
	case "!=", "!==", "ne":
		return !re.deepEqual(left, right), nil
	case ">", "gt":
		return re.compareNumeric(left, right, ">")
	case "<", "lt":
		return re.compareNumeric(left, right, "<")
	case ">=", "gte":
		return re.compareNumeric(left, right, ">=")
	case "<=", "lte":
		return re.compareNumeric(left, right, "<=")
	case "contains":
		return re.containsValue(left, right), nil
	case "in":
		return re.containsValue(right, left), nil
	default:
		return false, fmt.Errorf("unknown comparison operator: %s", op)
	}
}

// deepEqual performs deep equality comparison using type switches instead of reflection
func (re *Engine) deepEqual(left, right interface{}) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	
	switch leftVal := left.(type) {
	case string:
		if rightVal, ok := right.(string); ok {
			return leftVal == rightVal
		}
	case int:
		if rightVal, ok := right.(int); ok {
			return leftVal == rightVal
		}
		if rightVal, ok := right.(float64); ok {
			return float64(leftVal) == rightVal
		}
	case float64:
		if rightVal, ok := right.(float64); ok {
			return leftVal == rightVal
		}
		if rightVal, ok := right.(int); ok {
			return leftVal == float64(rightVal)
		}
	case bool:
		if rightVal, ok := right.(bool); ok {
			return leftVal == rightVal
		}
	case []interface{}:
		if rightVal, ok := right.([]interface{}); ok {
			if len(leftVal) != len(rightVal) {
				return false
			}
			for i, leftItem := range leftVal {
				if !re.deepEqual(leftItem, rightVal[i]) {
					return false
				}
			}
			return true
		}
	case map[string]interface{}:
		if rightVal, ok := right.(map[string]interface{}); ok {
			if len(leftVal) != len(rightVal) {
				return false
			}
			for key, leftItem := range leftVal {
				rightItem, exists := rightVal[key]
				if !exists || !re.deepEqual(leftItem, rightItem) {
					return false
				}
			}
			return true
		}
	}
	
	return left == right
}

// compareNumeric compares numeric values
func (re *Engine) compareNumeric(left, right interface{}, op string) (bool, error) {
	leftFloat, leftOk := re.toFloat64(left)
	rightFloat, rightOk := re.toFloat64(right)
	
	if !leftOk || !rightOk {
		return false, fmt.Errorf("cannot compare non-numeric values")
	}
	
	switch op {
	case ">":
		return leftFloat > rightFloat, nil
	case "<":
		return leftFloat < rightFloat, nil
	case ">=":
		return leftFloat >= rightFloat, nil
	case "<=":
		return leftFloat <= rightFloat, nil
	default:
		return false, fmt.Errorf("unknown numeric operator: %s", op)
	}
}

// toFloat64 converts a value to float64 if possible
func (re *Engine) toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	default:
		return 0, false
	}
}

// containsValue checks if a container contains a value
func (re *Engine) containsValue(container, value interface{}) bool {
	switch cont := container.(type) {
	case string:
		if strVal, ok := value.(string); ok {
			return strings.Contains(cont, strVal)
		}
	case []interface{}:
		for _, item := range cont {
			if re.deepEqual(item, value) {
				return true
			}
		}
	case map[string]interface{}:
		if keyStr, ok := value.(string); ok {
			_, exists := cont[keyStr]
			return exists
		}
	}
	return false
}

// isTruthy determines if a value is truthy
func (re *Engine) isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}
	
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case int:
		return v != 0
	case float64:
		return v != 0.0
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true
	}
}

// evaluateAnd evaluates logical AND
func (re *Engine) evaluateAnd(operands interface{}, data map[string]interface{}) (bool, error) {
	operandArray, ok := operands.([]interface{})
	if !ok {
		return false, fmt.Errorf("and operands must be an array")
	}
	
	for _, operand := range operandArray {
		result, err := re.evaluate(operand, data)
		if err != nil {
			return false, err
		}
		if !re.isTruthy(result) {
			return false, nil
		}
	}
	return true, nil
}

// evaluateOr evaluates logical OR
func (re *Engine) evaluateOr(operands interface{}, data map[string]interface{}) (bool, error) {
	operandArray, ok := operands.([]interface{})
	if !ok {
		return false, fmt.Errorf("or operands must be an array")
	}
	
	for _, operand := range operandArray {
		result, err := re.evaluate(operand, data)
		if err != nil {
			return false, err
		}
		if re.isTruthy(result) {
			return true, nil
		}
	}
	return false, nil
}

// evaluateNot evaluates logical NOT
func (re *Engine) evaluateNot(operand interface{}, data map[string]interface{}) (bool, error) {
	result, err := re.evaluate(operand, data)
	if err != nil {
		return false, err
	}
	return !re.isTruthy(result), nil
}

// executeScript executes a Python script with the given context
func (re *Engine) executeScript(scriptPath string, data map[string]interface{}) (interface{}, error) {
	// Get Python interpreter path from config
	pythonPath := "python3" // Default
	if re.config.Python.VenvPath != "" {
		// Use virtual environment if configured
		pythonPath = re.config.Python.VenvPath + "/bin/python"
		// Check if it's Windows
		if _, err := os.Stat(re.config.Python.VenvPath + "/Scripts/python.exe"); err == nil {
			pythonPath = re.config.Python.VenvPath + "/Scripts/python.exe"
		}
	}
	
	// Prepare context as JSON for the script
	contextJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context: %v", err)
	}
	
	// Create command with timeout
	timeout := time.Duration(re.config.Python.ScriptTimeout) * time.Second
	if timeout == 0 {
		timeout = 300 * time.Second // 5 minute default
	}
	
	// Create the command
	cmd := exec.Command(pythonPath, scriptPath)
	
	// Set environment variables
	cmd.Env = append(os.Environ(), fmt.Sprintf("SECAUTO_CONTEXT=%s", string(contextJSON)))
	
	// Prepare stdin with context data
	cmd.Stdin = bytes.NewReader(contextJSON)
	
	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start script %s: %v", scriptPath, err)
	}
	
	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	
	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("script %s failed: %v\nStderr: %s", scriptPath, err, stderr.String())
		}
	case <-time.After(timeout):
		cmd.Process.Kill()
		return nil, fmt.Errorf("script %s timed out after %v", scriptPath, timeout)
	}
	
	// Parse the output
	outputStr := strings.TrimSpace(stdout.String())
	if outputStr == "" {
		// If no JSON output, return basic execution info
		return map[string]interface{}{
			"script":    scriptPath,
			"executed":  true,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"output":    "",
			"stderr":    stderr.String(),
		}, nil
	}
	
	// Try to parse as JSON
	var result interface{}
	if err := json.Unmarshal([]byte(outputStr), &result); err != nil {
		// If not JSON, return as plain text
		return map[string]interface{}{
			"script":    scriptPath,
			"executed":  true,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"output":    outputStr,
			"stderr":    stderr.String(),
		}, nil
	}
	
	return result, nil
}