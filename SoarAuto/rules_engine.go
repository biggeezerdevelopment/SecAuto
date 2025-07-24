package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

// RuleEngine represents the SOAR rules engine
type RuleEngine struct {
	config        *Config
	context       map[string]interface{}
	pluginManager *PlatformPluginManager
}

// NewRuleEngine creates a new rule engine instance
func NewRuleEngine(config *Config) *RuleEngine {
	return &RuleEngine{
		config:        config,
		context:       make(map[string]interface{}),
		pluginManager: nil, // Will be set by SetPluginManager
	}
}

// SetContext sets the context for the rule engine
func (re *RuleEngine) SetContext(context map[string]interface{}) {
	logger.Info("Setting context", map[string]interface{}{
		"component": "rules_engine",
		"context":   context,
	})

	// Create a single, flat context object
	re.context = make(map[string]interface{})

	// If context already has a nested structure, extract and merge it
	if nestedContext, exists := context["context"]; exists {
		logger.Info("Found nested context", map[string]interface{}{
			"component": "rules_engine",
			"nested":    nestedContext,
		})
		if contextMap, ok := nestedContext.(map[string]interface{}); ok {
			// Merge nested context into flat structure
			for k, v := range contextMap {
				re.context[k] = v
			}
		}
	} else {
		logger.Info("No nested context found, using direct context", map[string]interface{}{
			"component": "rules_engine",
		})
		// No nested context, merge all data directly into the flat context
		for k, v := range context {
			re.context[k] = v
		}
	}

	logger.Info("Final context", map[string]interface{}{
		"component":    "rules_engine",
		"context":      re.context,
		"context_keys": len(re.context),
		"has_incident": re.context["incident"] != nil,
	})

	// Log the incident object specifically
	if incident, exists := re.context["incident"]; exists {
		logger.Info("Incident object found", map[string]interface{}{
			"component":     "rules_engine",
			"incident":      incident,
			"incident_type": fmt.Sprintf("%T", incident),
		})
		if incidentMap, ok := incident.(map[string]interface{}); ok {
			logger.Info("Incident map details", map[string]interface{}{
				"component":         "rules_engine",
				"incident_keys":     len(incidentMap),
				"has_threat_score":  incidentMap["threat_score"] != nil,
				"threat_score":      incidentMap["threat_score"],
				"threat_score_type": fmt.Sprintf("%T", incidentMap["threat_score"]),
			})
		}
	} else {
		logger.Info("No incident object found in context", map[string]interface{}{
			"component": "rules_engine",
		})
	}
}

// GetContext returns the current context
func (re *RuleEngine) GetContext() map[string]interface{} {
	return re.context
}

// SetPluginManager sets the plugin manager for the rule engine
func (re *RuleEngine) SetPluginManager(pluginManager *PlatformPluginManager) {
	re.pluginManager = pluginManager
}

// EvaluateRule evaluates a single rule
func (re *RuleEngine) EvaluateRule(rule interface{}) (interface{}, error) {
	return re.evaluate(rule, re.context)
}

// EvaluatePlaybook evaluates a playbook (array of rules)
func (re *RuleEngine) EvaluatePlaybook(playbook []interface{}) ([]interface{}, error) {
	var results []interface{}

	logger.Info("Evaluating playbook", map[string]interface{}{
		"component":  "rules_engine",
		"rule_count": len(playbook),
	})

	for i, rule := range playbook {
		logger.Info("Evaluating rule", map[string]interface{}{
			"component":  "rules_engine",
			"rule_index": i + 1,
			"rule":       rule,
		})
		result, err := re.evaluate(rule, re.context)
		if err != nil {
			logger.Error("Rule evaluation failed", map[string]interface{}{
				"component":  "rules_engine",
				"rule_index": i + 1,
				"error":      err.Error(),
			})
			return nil, fmt.Errorf("error evaluating rule %d: %v", i+1, err)
		}

		// Handle nested results from play operations
		logger.Info("Processing rule result", map[string]interface{}{
			"component":   "rules_engine",
			"result":      result,
			"result_type": fmt.Sprintf("%T", result),
		})

		if resultArray, ok := result.([]interface{}); ok {
			logger.Info("Flattening nested results", map[string]interface{}{
				"component": "rules_engine",
				"array_len": len(resultArray),
			})
			// Flatten nested results into the main results array
			results = append(results, resultArray...)
		} else {
			logger.Info("Adding single result", map[string]interface{}{
				"component": "rules_engine",
			})
			results = append(results, result)
		}
	}

	logger.Info("Completed playbook evaluation", map[string]interface{}{
		"component":  "rules_engine",
		"rule_count": len(playbook),
	})
	return results, nil
}

// evaluate recursively evaluates JSONLogic expressions
func (re *RuleEngine) evaluate(expr interface{}, data map[string]interface{}) (interface{}, error) {
	logger.Info("Evaluating expression", map[string]interface{}{
		"component": "rules_engine",
		"expr":      expr,
		"expr_type": fmt.Sprintf("%T", expr),
	})

	if expr == nil {
		return nil, nil
	}

	// Process template variables in the expression
	processedExpr := re.processTemplateVariables(expr, data)

	logger.Info("Template variable processing in evaluate", map[string]interface{}{
		"component":       "rules_engine",
		"original_expr":   expr,
		"processed_expr":  processedExpr,
		"data_keys":       len(data),
		"has_virustotal":  data["virustotal"] != nil,
		"virustotal_type": fmt.Sprintf("%T", data["virustotal"]),
	})

	switch v := processedExpr.(type) {
	case map[string]interface{}:
		logger.Info("Evaluating map operation", map[string]interface{}{
			"component": "rules_engine",
			"operation": v,
		})
		return re.evaluateOperation(v, data)
	case []interface{}:
		logger.Info("Evaluating array", map[string]interface{}{
			"component": "rules_engine",
			"array_len": len(v),
		})

		// Check if this is a comparison operation array (e.g., [">=", {"var": "..."}, 10])
		if len(v) == 3 {
			operator, ok1 := v[0].(string)
			if ok1 {
				// Check if it's a comparison operator
				switch operator {
				case ">", "gt", "<", "lt", ">=", "gte", "<=", "lte", "==", "eq", "!=", "!===":
					logger.Info("Found comparison operation in array", map[string]interface{}{
						"component": "rules_engine",
						"operator":  operator,
						"operands":  []interface{}{v[1], v[2]},
					})

					// Convert array format to map format for evaluation
					operation := map[string]interface{}{
						operator: []interface{}{v[1], v[2]},
					}
					return re.evaluateComparison(operation, operator, data)
				}
			}
		}

		// Handle as regular array
		var results []interface{}
		for _, item := range v {
			result, err := re.evaluate(item, data)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
		return results, nil
	default:
		logger.Info("Evaluating primitive value", map[string]interface{}{
			"component": "rules_engine",
			"value":     v,
			"type":      fmt.Sprintf("%T", v),
		})
		return v, nil
	}
}

// evaluateOperation handles different operation types
func (re *RuleEngine) evaluateOperation(operation map[string]interface{}, data map[string]interface{}) (interface{}, error) {
	logger.Info("Evaluating operation", map[string]interface{}{
		"component": "rules_engine",
		"operation": operation,
	})

	// Check for custom operations first
	if _, exists := operation["run"]; exists {
		logger.Info("Found run operation", map[string]interface{}{
			"component": "rules_engine",
		})
		return re.evaluateRunOperation(operation["run"], operation, data)
	}

	if _, exists := operation["play"]; exists {
		logger.Info("Found play operation", map[string]interface{}{
			"component": "rules_engine",
		})
		return re.evaluatePlayOperation(operation["play"], data)
	}

	if _, exists := operation["if"]; exists {
		logger.Info("Found if operation", map[string]interface{}{
			"component": "rules_engine",
		})
		return re.evaluateIfOperation(operation["if"], data)
	}

	if _, exists := operation["plugin"]; exists {
		logger.Info("Found plugin operation", map[string]interface{}{
			"component": "rules_engine",
		})
		return re.evaluatePluginOperation(operation["plugin"], data)
	}

	// Check for variable operations
	if _, exists := operation["var"]; exists {
		logger.Info("Found var operation", map[string]interface{}{
			"component": "rules_engine",
		})
		return re.evaluateVarOperation(operation["var"], data)
	}

	// Check for comparison operations
	for op := range operation {
		switch op {
		case "eq", "gt", "lt", "gte", "lte":
			logger.Info("Found comparison operation", map[string]interface{}{
				"component": "rules_engine",
				"operator":  op,
			})
			return re.evaluateComparison(operation, op, data)
		}
	}

	// Check for logical operations
	for op := range operation {
		switch op {
		case "and", "or", "not":
			logger.Info("Found logical operation", map[string]interface{}{
				"component": "rules_engine",
				"operator":  op,
			})
			return re.evaluateLogical(operation, op, data)
		}
	}

	logger.Error("Unknown operation", map[string]interface{}{
		"component": "rules_engine",
		"operation": operation,
	})
	return nil, fmt.Errorf("unknown operation: %v", operation)
}

// evaluateRunOperation handles the "run" operation
func (re *RuleEngine) evaluateRunOperation(scriptName interface{}, operation map[string]interface{}, data map[string]interface{}) (interface{}, error) {
	scriptNameStr, ok := scriptName.(string)
	if !ok {
		return nil, fmt.Errorf("script name must be a string")
	}

	scriptPath := re.getScriptPath(scriptNameStr)
	logger.Info("Running Python script", map[string]interface{}{
		"component": "rules_engine",
		"script":    scriptNameStr,
	})

	logger.Debug("Context before Python script", map[string]interface{}{
		"component": "rules_engine",
		"context":   re.context,
	})

	// Process template variables in the operation parameters before passing to Python script
	// This ensures that parameters like "urls": "{{threat_intelligence.domains}}" are resolved
	processedOperation := re.processTemplateVariables(operation, re.context)

	// Merge the processed operation parameters with the context data
	processedData := make(map[string]interface{})
	for k, v := range data {
		processedData[k] = v
	}

	// Type assert processedOperation to map for merging
	if processedOperationMap, ok := processedOperation.(map[string]interface{}); ok {
		for k, v := range processedOperationMap {
			if k != "run" { // Don't override the script name
				processedData[k] = v
			}
		}
	}

	logger.Info("Template variable processing", map[string]interface{}{
		"component":      "rules_engine",
		"original_data":  data,
		"processed_data": processedData,
		"has_urls":       processedData["urls"] != nil,
		"urls_type":      fmt.Sprintf("%T", processedData["urls"]),
		"urls_value":     processedData["urls"],
	})

	// Pass the processed context to Python scripts
	outputBytes, err := RunPythonFromVenvWithJSONSeparateOutput(re.config.GetVenvPath(), scriptPath, processedData)
	if err != nil {
		logger.Error("Python script execution failed", map[string]interface{}{
			"component": "rules_engine",
			"script":    scriptNameStr,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("failed to run Python script %s: %v", scriptNameStr, err)
	}

	// Parse the raw JSON output from the Python script
	var resultData map[string]interface{}
	if err := json.Unmarshal(outputBytes, &resultData); err != nil {
		// Try to clean the output by removing any non-JSON content
		outputStr := string(outputBytes)
		cleanedOutput := cleanPythonOutput(outputStr)

		if err := json.Unmarshal([]byte(cleanedOutput), &resultData); err != nil {
			logger.Error("Failed to parse Python script output", map[string]interface{}{
				"component": "rules_engine",
				"script":    scriptNameStr,
				"error":     err.Error(),
				"output":    string(outputBytes),
				"cleaned":   cleanedOutput,
			})
			return nil, fmt.Errorf("failed to parse Python script output: %v", err)
		}
	}

	logger.Debug("Python script output structure", map[string]interface{}{
		"component":   "rules_engine",
		"result_data": resultData,
	})

	// Merge the result into the context
	if resultData != nil {
		logger.Debug("Merging Python script result", map[string]interface{}{
			"component": "rules_engine",
			"result":    resultData,
		})

		// Handle incident_updates if present (from get_process_update)
		if incidentUpdates, exists := resultData["incident_updates"]; exists {
			logger.Debug("Found incident_updates", map[string]interface{}{
				"component": "rules_engine",
				"updates":   incidentUpdates,
			})

			if re.context["incident"] == nil {
				re.context["incident"] = make(map[string]interface{})
			}
			if incidentMap, ok := re.context["incident"].(map[string]interface{}); ok {
				if updatesMap, ok := incidentUpdates.(map[string]interface{}); ok {
					for k, v := range updatesMap {
						incidentMap[k] = v
					}
					logger.Debug("Merged incident_updates", map[string]interface{}{
						"component": "rules_engine",
						"incident":  incidentMap,
					})
				}
			}

			// Remove incident_updates from the result since it's now merged
			delete(resultData, "incident_updates")
		}

		// Merge remaining context data directly into the flat context structure
		for k, v := range resultData {
			re.context[k] = v
		}

		logger.Debug("Context after Python script merge", map[string]interface{}{
			"component": "rules_engine",
			"context":   re.context,
		})
	}

	logger.Info("Completed Python script", map[string]interface{}{
		"component": "rules_engine",
		"script":    scriptNameStr,
	})

	// Python scripts update context but don't return results to be added to the results array
	// Return a simple success indicator instead of the full context
	return map[string]interface{}{
		"script": scriptNameStr,
		"status": "completed",
	}, nil
}

// evaluatePlayOperation handles the "play" operation
func (re *RuleEngine) evaluatePlayOperation(playbookName interface{}, data map[string]interface{}) (interface{}, error) {
	playbookNameStr, ok := playbookName.(string)
	if !ok {
		return nil, fmt.Errorf("playbook name must be a string")
	}

	playbookPath := re.getPlaybookPath(playbookNameStr)
	logger.Info("Starting playbook", map[string]interface{}{
		"component": "rules_engine",
		"playbook":  playbookNameStr,
	})

	// Load and evaluate the nested playbook
	playbookData, err := re.LoadPlaybookFromFile(playbookPath)
	if err != nil {
		logger.Error("Failed to load playbook", map[string]interface{}{
			"component": "rules_engine",
			"playbook":  playbookNameStr,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("failed to load playbook %s: %v", playbookNameStr, err)
	}

	results, err := re.EvaluatePlaybook(playbookData)
	if err != nil {
		logger.Error("Failed to evaluate playbook", map[string]interface{}{
			"component": "rules_engine",
			"playbook":  playbookNameStr,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("failed to evaluate playbook %s: %v", playbookNameStr, err)
	}

	logger.Info("Completed playbook", map[string]interface{}{
		"component": "rules_engine",
		"playbook":  playbookNameStr,
	})
	return results, nil
}

// evaluatePluginOperation handles the "plugin" operation
func (re *RuleEngine) evaluatePluginOperation(pluginExpr interface{}, data map[string]interface{}) (interface{}, error) {
	if re.pluginManager == nil {
		return nil, fmt.Errorf("plugin manager not available")
	}

	// Parse plugin expression
	var pluginName string
	var params map[string]interface{}

	switch v := pluginExpr.(type) {
	case string:
		// Simple case: just plugin name
		pluginName = v
		params = make(map[string]interface{})
	case map[string]interface{}:
		// Complex case: plugin name and parameters
		if name, ok := v["name"].(string); ok {
			pluginName = name
		} else {
			return nil, fmt.Errorf("plugin name is required")
		}

		// Extract parameters
		if pluginParams, ok := v["params"].(map[string]interface{}); ok {
			params = pluginParams
		} else {
			params = make(map[string]interface{})
		}

		// Merge context into parameters
		for k, v := range data {
			params[k] = v
		}
	default:
		return nil, fmt.Errorf("invalid plugin expression: expected string or object")
	}

	logger.Info("Executing plugin", map[string]interface{}{
		"component": "rules_engine",
		"plugin":    pluginName,
		"params":    params,
	})

	// Execute the plugin
	result, err := re.pluginManager.ExecutePlugin(pluginName, params)
	if err != nil {
		logger.Error("Plugin execution failed", map[string]interface{}{
			"component": "rules_engine",
			"plugin":    pluginName,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("failed to execute plugin %s: %v", pluginName, err)
	}

	// Merge plugin result into context if it's a map
	if resultMap, ok := result.(map[string]interface{}); ok {
		logger.Debug("Merging plugin result", map[string]interface{}{
			"component": "rules_engine",
			"result":    resultMap,
		})

		// Handle incident updates if present
		if incidentUpdates, exists := resultMap["incident"]; exists {
			logger.Debug("Found incident updates in plugin result", map[string]interface{}{
				"component": "rules_engine",
				"updates":   incidentUpdates,
			})

			if re.context["incident"] == nil {
				re.context["incident"] = make(map[string]interface{})
			}
			if incidentMap, ok := re.context["incident"].(map[string]interface{}); ok {
				if updatesMap, ok := incidentUpdates.(map[string]interface{}); ok {
					for k, v := range updatesMap {
						incidentMap[k] = v
					}
					logger.Debug("Merged incident updates from plugin", map[string]interface{}{
						"component": "rules_engine",
						"incident":  incidentMap,
					})
				}
			}

			// Remove incident from the result since it's now merged
			delete(resultMap, "incident")
		}

		// Merge remaining context data directly into the flat context structure
		for k, v := range resultMap {
			re.context[k] = v
		}
	}

	logger.Info("Completed plugin execution", map[string]interface{}{
		"component": "rules_engine",
		"plugin":    pluginName,
	})

	return map[string]interface{}{
		"plugin": pluginName,
		"status": "completed",
		"result": result,
	}, nil
}

// LoadPlaybookFromFile loads a playbook from a JSON file
func (re *RuleEngine) LoadPlaybookFromFile(filename string) ([]interface{}, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read playbook file: %v", err)
	}

	// Parse JSON
	var playbook []interface{}
	if err := json.Unmarshal(data, &playbook); err != nil {
		return nil, fmt.Errorf("failed to parse playbook JSON: %v", err)
	}

	return playbook, nil
}

// getScriptPath returns the full path to a Python script
func (re *RuleEngine) getScriptPath(scriptName string) string {
	return re.config.GetScriptPath(scriptName)
}

// getPlaybookPath returns the full path to a playbook file
func (re *RuleEngine) getPlaybookPath(playbookName string) string {
	return re.config.GetPlaybookPath(playbookName)
}

// evaluateIfOperation handles the "if" operation
func (re *RuleEngine) evaluateIfOperation(ifExpr interface{}, data map[string]interface{}) (interface{}, error) {
	// Check if it's the new object-based structure
	if ifMap, ok := ifExpr.(map[string]interface{}); ok {
		return re.evaluateObjectBasedIf(ifMap, data)
	}

	// Handle existing array-based structure for backward compatibility
	ifArr, ok := ifExpr.([]interface{})
	if !ok {
		return nil, fmt.Errorf("if operation requires an array or object")
	}

	if len(ifArr) < 2 {
		return nil, fmt.Errorf("if operation requires at least condition and then action")
	}

	// Evaluate the condition
	condition, err := re.evaluate(ifArr[0], data)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate if condition: %v", err)
	}

	logger.Debug("If condition evaluated", map[string]interface{}{
		"component": "rules_engine",
		"condition": condition,
		"truthy":    re.isTruthy(condition),
	})

	// If condition is truthy, execute the "then" branch
	if re.isTruthy(condition) {
		logger.Debug("Executing 'then' branch", map[string]interface{}{
			"component": "rules_engine",
		})
		return re.evaluate(ifArr[1], data)
	}

	// If condition is falsy and there's an "else" branch, execute it
	if len(ifArr) > 2 {
		logger.Debug("Executing 'else' branch", map[string]interface{}{
			"component": "rules_engine",
		})
		return re.evaluate(ifArr[2], data)
	}

	return nil, nil
}

// evaluateObjectBasedIf handles the new object-based if structure
func (re *RuleEngine) evaluateObjectBasedIf(ifMap map[string]interface{}, data map[string]interface{}) (interface{}, error) {
	// Extract conditions
	conditions, ok := ifMap["conditions"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("conditions must be an array")
	}

	// Extract logic (default to "and")
	logic, ok := ifMap["logic"].(string)
	if !ok {
		logic = "and"
	}

	// Extract true and false actions
	trueAction := ifMap["true"]
	falseAction := ifMap["false"]

	logger.Debug("Evaluating object-based if", map[string]interface{}{
		"component":  "rules_engine",
		"conditions": len(conditions),
		"logic":      logic,
		"has_true":   trueAction != nil,
		"has_false":  falseAction != nil,
	})

	// Evaluate conditions based on logic
	var conditionResult bool
	var err error

	switch logic {
	case "and":
		conditionResult, err = re.evaluateAnd(conditions, data)
	case "or":
		conditionResult, err = re.evaluateOr(conditions, data)
	default:
		return nil, fmt.Errorf("unsupported logic: %s", logic)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to evaluate conditions: %v", err)
	}

	logger.Debug("Object-based if condition result", map[string]interface{}{
		"component": "rules_engine",
		"result":    conditionResult,
	})

	// Execute appropriate action (which could be another if)
	if conditionResult {
		if trueAction != nil {
			logger.Debug("Executing 'true' branch", map[string]interface{}{
				"component": "rules_engine",
			})
			return re.evaluate(trueAction, data)
		}
	} else if falseAction != nil {
		logger.Debug("Executing 'false' branch", map[string]interface{}{
			"component": "rules_engine",
		})
		return re.evaluate(falseAction, data)
	}

	return nil, nil
}

// evaluateVarOperation handles the "var" operation
func (re *RuleEngine) evaluateVarOperation(varName interface{}, data map[string]interface{}) (interface{}, error) {
	varNameStr, ok := varName.(string)
	if !ok {
		return nil, fmt.Errorf("variable name must be a string")
	}

	logger.Info("Looking for variable", map[string]interface{}{
		"component": "rules_engine",
		"variable":  varNameStr,
		"data":      data,
	})

	// Check if it's a direct context access
	if varNameStr == "context" {
		logger.Info("Found context variable", map[string]interface{}{
			"component": "rules_engine",
			"variable":  varNameStr,
			"value":     data,
		})
		return data, nil
	}

	// Check if it's a direct access to a key in context
	if value, exists := data[varNameStr]; exists {
		logger.Debug("Found variable in context", map[string]interface{}{
			"component": "rules_engine",
			"variable":  varNameStr,
			"value":     value,
		})
		return value, nil
	}

	// Check if it's a dot notation access (e.g., "context.incident.id")
	if value, err := re.evaluateDotNotation(varNameStr, data); err == nil && value != nil {
		logger.Debug("Found variable via dot notation", map[string]interface{}{
			"component":  "rules_engine",
			"variable":   varNameStr,
			"value":      value,
			"value_type": fmt.Sprintf("%T", value),
		})
		return value, nil
	}

	logger.Debug("Variable not found", map[string]interface{}{
		"component": "rules_engine",
		"variable":  varNameStr,
	})
	return nil, nil
}

// evaluateDotNotation handles dot notation for nested access
func (re *RuleEngine) evaluateDotNotation(path string, data map[string]interface{}) (interface{}, error) {
	logger.Debug("evaluateDotNotation: resolving", map[string]interface{}{
		"path":      path,
		"data_keys": len(data),
	})

	keys := []string{}
	var current interface{} = data

	// Split the path by dots using strings.Split
	parts := strings.Split(path, ".")

	// Filter out empty parts (handles consecutive dots)
	for _, part := range parts {
		if part != "" {
			keys = append(keys, part)
		}
	}

	logger.Debug("evaluateDotNotation: parts", map[string]interface{}{
		"path":  path,
		"parts": keys,
	})

	// Navigate through the nested structure
	for i, key := range keys {
		logger.Debug("evaluateDotNotation: navigating", map[string]interface{}{
			"step":         i + 1,
			"key":          key,
			"current_type": fmt.Sprintf("%T", current),
		})

		if currentMap, ok := current.(map[string]interface{}); ok {
			logger.Debug("evaluateDotNotation: current is map", map[string]interface{}{
				"map_keys": len(currentMap),
				"has_key":  currentMap[key] != nil,
			})

			if value, exists := currentMap[key]; exists {
				current = value
				logger.Debug("evaluateDotNotation: found value", map[string]interface{}{
					"value":      value,
					"value_type": fmt.Sprintf("%T", value),
				})
			} else {
				logger.Debug("evaluateDotNotation: key not found", map[string]interface{}{
					"key":            key,
					"available_keys": getMapKeys(currentMap),
				})
				return nil, fmt.Errorf("key %s not found", key)
			}
		} else {
			logger.Debug("evaluateDotNotation: current is not map", map[string]interface{}{
				"current":      current,
				"current_type": fmt.Sprintf("%T", current),
			})
			return nil, fmt.Errorf("cannot access key %s in non-map value", key)
		}
	}

	logger.Debug("evaluateDotNotation: resolved value", map[string]interface{}{
		"path":     path,
		"resolved": current,
		"type":     fmt.Sprintf("%T", current),
	})

	return current, nil
}

// getMapKeys returns the keys of a map as a slice of strings
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// evaluateComparison handles comparison operations
func (re *RuleEngine) evaluateComparison(operation map[string]interface{}, op string, data map[string]interface{}) (bool, error) {
	operands, exists := operation[op]
	if !exists {
		return false, fmt.Errorf("comparison operator %s not found", op)
	}

	operandsArr, ok := operands.([]interface{})
	if !ok {
		return false, fmt.Errorf("comparison operator %s requires an array of operands", op)
	}

	if len(operandsArr) != 2 {
		return false, fmt.Errorf("comparison operator %s requires exactly 2 operands, got %d", op, len(operandsArr))
	}

	// Evaluate left operand
	left, err := re.evaluate(operandsArr[0], data)
	if err != nil {
		return false, err
	}

	// Evaluate right operand
	right, err := re.evaluate(operandsArr[1], data)
	if err != nil {
		return false, err
	}

	logger.Debug("Comparing values", map[string]interface{}{
		"component": "rules_engine",
		"left":      left,
		"right":     right,
		"operator":  op,
	})
	result, err := re.compareValues(left, right, op)
	logger.Debug("Comparison result", map[string]interface{}{
		"component": "rules_engine",
		"result":    result,
	})
	return result, err
}

// evaluateLogical handles logical operations
func (re *RuleEngine) evaluateLogical(operation map[string]interface{}, op string, data map[string]interface{}) (interface{}, error) {
	operands, exists := operation[op]
	if !exists {
		return nil, fmt.Errorf("logical operation %s not found", op)
	}

	switch op {
	case "and":
		return re.evaluateAnd(operands, data)
	case "or":
		return re.evaluateOr(operands, data)
	case "not":
		return re.evaluateNot(operands, data)
	default:
		return nil, fmt.Errorf("unknown logical operation: %s", op)
	}
}

// evaluateAnd evaluates AND operation
func (re *RuleEngine) evaluateAnd(operands interface{}, data map[string]interface{}) (bool, error) {
	operandsArr, ok := operands.([]interface{})
	if !ok {
		return false, fmt.Errorf("and operation requires an array")
	}

	for _, operand := range operandsArr {
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

// evaluateOr evaluates OR operation
func (re *RuleEngine) evaluateOr(operands interface{}, data map[string]interface{}) (bool, error) {
	operandsArr, ok := operands.([]interface{})
	if !ok {
		return false, fmt.Errorf("or operation requires an array")
	}

	for _, operand := range operandsArr {
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

// evaluateNot evaluates NOT operation
func (re *RuleEngine) evaluateNot(operand interface{}, data map[string]interface{}) (bool, error) {
	result, err := re.evaluate(operand, data)
	if err != nil {
		return false, err
	}
	return !re.isTruthy(result), nil
}

// isTruthy checks if a value is truthy
func (re *RuleEngine) isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case float64:
		return v != 0
	case int:
		return v != 0
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true
	}
}

// compareValues compares two values using the specified operator
func (re *RuleEngine) compareValues(left, right interface{}, op string) (bool, error) {
	logger.Debug("Comparing values", map[string]interface{}{
		"component":  "rules_engine",
		"left":       left,
		"left_type":  fmt.Sprintf("%T", left),
		"right":      right,
		"right_type": fmt.Sprintf("%T", right),
		"operator":   op,
	})

	// Normalize values for comparison
	leftNorm := re.normalizeValue(left)
	rightNorm := re.normalizeValue(right)

	// Handle string comparison
	if leftStr, leftOk := leftNorm.(string); leftOk {
		if rightStr, rightOk := rightNorm.(string); rightOk {
			switch op {
			case "==", "===", "eq":
				result := leftStr == rightStr
				logger.Debug("String comparison result", map[string]interface{}{
					"component": "rules_engine",
					"result":    result,
				})
				return result, nil
			case "!=", "!==":
				return leftStr != rightStr, nil
			}
		}
	}

	// Handle numeric comparison
	if leftNum, leftOk := leftNorm.(float64); leftOk {
		if rightNum, rightOk := rightNorm.(float64); rightOk {
			result, err := re.compareNumeric(leftNum, rightNum, op)
			logger.Debug("Numeric comparison result", map[string]interface{}{
				"component": "rules_engine",
				"result":    result,
			})
			return result, err
		}
	}

	// Handle integer comparison (convert to float64)
	if leftInt, leftOk := leftNorm.(int); leftOk {
		if rightInt, rightOk := rightNorm.(int); rightOk {
			result, err := re.compareNumeric(float64(leftInt), float64(rightInt), op)
			logger.Debug("Integer comparison result", map[string]interface{}{
				"component": "rules_engine",
				"result":    result,
			})
			return result, err
		}
	}

	// Handle mixed numeric comparison (int vs float64)
	if leftInt, leftOk := leftNorm.(int); leftOk {
		if rightNum, rightOk := rightNorm.(float64); rightOk {
			result, err := re.compareNumeric(float64(leftInt), rightNum, op)
			logger.Debug("Mixed numeric comparison result", map[string]interface{}{
				"component": "rules_engine",
				"result":    result,
			})
			return result, err
		}
	}

	if leftNum, leftOk := leftNorm.(float64); leftOk {
		if rightInt, rightOk := rightNorm.(int); rightOk {
			result, err := re.compareNumeric(leftNum, float64(rightInt), op)
			logger.Debug("Mixed numeric comparison result", map[string]interface{}{
				"component": "rules_engine",
				"result":    result,
			})
			return result, err
		}
	}

	// Handle boolean comparison
	if leftBool, leftOk := leftNorm.(bool); leftOk {
		if rightBool, rightOk := rightNorm.(bool); rightOk {
			switch op {
			case "==", "===", "eq":
				return leftBool == rightBool, nil
			case "!=", "!==":
				return leftBool != rightBool, nil
			}
		}
	}

	// Default comparison
	switch op {
	case "==", "===", "eq":
		result := reflect.DeepEqual(leftNorm, rightNorm)
		logger.Debug("DeepEqual comparison result", map[string]interface{}{
			"component": "rules_engine",
			"result":    result,
		})
		return result, nil
	case "!=", "!==":
		return !reflect.DeepEqual(leftNorm, rightNorm), nil
	case ">", "gt", "<", "lt", ">=", "gte", "<=", "lte":
		// Try numeric comparison for these operators
		if leftNum, leftOk := leftNorm.(float64); leftOk {
			if rightNum, rightOk := rightNorm.(float64); rightOk {
				return re.compareNumeric(leftNum, rightNum, op)
			}
		}
		// If not numeric, try converting to numeric
		if leftNum, leftOk := leftNorm.(int); leftOk {
			if rightNum, rightOk := rightNorm.(int); rightOk {
				return re.compareNumeric(float64(leftNum), float64(rightNum), op)
			}
		}
		// If still not numeric, return error
		return false, fmt.Errorf("numeric comparison operator %s requires numeric operands", op)
	default:
		return false, fmt.Errorf("unsupported comparison operator: %s", op)
	}
}

// normalizeValue converts values to comparable types
func (re *RuleEngine) normalizeValue(value interface{}) interface{} {
	logger.Debug("Normalizing value", map[string]interface{}{
		"component":  "rules_engine",
		"value":      value,
		"value_type": fmt.Sprintf("%T", value),
	})

	switch v := value.(type) {
	case string:
		// Don't convert strings to numbers - keep them as strings for proper comparison
		logger.Debug("Normalizing string value", map[string]interface{}{
			"component": "rules_engine",
			"value":     v,
		})
		return v
	case int:
		logger.Debug("Normalizing int value to float64", map[string]interface{}{
			"component": "rules_engine",
			"value":     v,
		})
		return float64(v)
	case float64:
		logger.Debug("Normalizing float64 value", map[string]interface{}{
			"component": "rules_engine",
			"value":     v,
		})
		return v
	case json.Number:
		// Convert json.Number to float64 for numeric operations
		logger.Debug("Normalizing json.Number value", map[string]interface{}{
			"component": "rules_engine",
			"value":     v.String(),
		})
		if floatVal, err := v.Float64(); err == nil {
			logger.Debug("Converted json.Number to float64", map[string]interface{}{
				"component": "rules_engine",
				"float_val": floatVal,
			})
			return floatVal
		}
		// If conversion fails, return as string
		logger.Debug("Failed to convert json.Number to float64, returning as string", map[string]interface{}{
			"component": "rules_engine",
			"value":     v.String(),
		})
		return v.String()
	default:
		logger.Debug("Normalizing default value", map[string]interface{}{
			"component":  "rules_engine",
			"value":      v,
			"value_type": fmt.Sprintf("%T", v),
		})
		return v
	}
}

// compareNumeric compares two float64 values using the specified operator
func (re *RuleEngine) compareNumeric(left, right interface{}, op string) (bool, error) {
	leftNum, ok1 := left.(float64)
	rightNum, ok2 := right.(float64)
	if !ok1 || !ok2 {
		return false, fmt.Errorf("numeric comparison requires float64 operands")
	}
	switch op {
	case ">", "gt":
		return leftNum > rightNum, nil
	case "<", "lt":
		return leftNum < rightNum, nil
	case ">=", "gte":
		return leftNum >= rightNum, nil
	case "<=", "lte":
		return leftNum <= rightNum, nil
	default:
		return false, fmt.Errorf("unknown numeric comparison operator: %s", op)
	}
}

// processTemplateVariables processes {{variable}} syntax in strings
func (re *RuleEngine) processTemplateVariables(value interface{}, data map[string]interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// Check if the string is exactly a template variable (e.g. "{{threat_intelligence.domains}}")
		regex := regexp.MustCompile(`^\{\{([^}]+)\}\}$`)
		if matches := regex.FindStringSubmatch(v); len(matches) == 2 {
			variableName := strings.TrimSpace(matches[1])
			logger.Debug("processTemplateVariables: Found exact template variable", map[string]interface{}{
				"variable": variableName,
				"value":    v,
			})
			// Try direct lookup
			if resolved, exists := data[variableName]; exists {
				logger.Debug("processTemplateVariables: Direct lookup resolved", map[string]interface{}{
					"variable": variableName,
					"resolved": resolved,
					"type":     fmt.Sprintf("%T", resolved),
				})
				return resolved
			}
			// Try dot notation
			if resolved, err := re.evaluateDotNotation(variableName, data); err == nil && resolved != nil {
				logger.Info("processTemplateVariables: Dot notation resolved", map[string]interface{}{
					"variable": variableName,
					"resolved": resolved,
					"type":     fmt.Sprintf("%T", resolved),
				})
				return resolved
			} else if err != nil {
				logger.Info("processTemplateVariables: Dot notation error", map[string]interface{}{
					"variable": variableName,
					"error":    err.Error(),
				})
			}
			// If not found, fall through to string template
		}
		return re.processStringTemplate(v, data)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = re.processTemplateVariables(val, data)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = re.processTemplateVariables(val, data)
		}
		return result
	default:
		return value
	}
}

// processStringTemplate processes {{variable}} syntax in a string
func (re *RuleEngine) processStringTemplate(template string, data map[string]interface{}) string {
	// Regular expression to match {{variable}} patterns
	regex := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	return regex.ReplaceAllStringFunc(template, func(match string) string {
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

		// If not found directly, try dot notation (e.g., "threat_intelligence.domains")
		if value, err := re.evaluateDotNotation(variableName, data); err == nil && value != nil {
			if strValue, ok := value.(string); ok {
				return strValue
			}
			// Convert non-string values to string
			return fmt.Sprintf("%v", value)
		}

		// If variable not found, return the original template
		logger.Warning("Template variable not found", map[string]interface{}{
			"component": "rules_engine",
			"variable":  variableName,
			"template":  template,
		})
		return match
	})
}

// cleanPythonOutput attempts to extract valid JSON from Python script output
func cleanPythonOutput(output string) string {
	// Remove leading/trailing whitespace
	output = strings.TrimSpace(output)

	// Find the first '{' and last '}' to extract JSON
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")

	if start != -1 && end != -1 && end > start {
		jsonPart := output[start : end+1]

		// Remove any non-printable characters that might interfere
		var cleaned strings.Builder
		for _, r := range jsonPart {
			if unicode.IsPrint(r) || unicode.IsSpace(r) {
				cleaned.WriteRune(r)
			}
		}

		return cleaned.String()
	}

	// If no JSON braces found, return original output
	return output
}
