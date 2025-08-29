# SecAuto Playbook Execution Flow Map

## 1. HTTP Entry Point
**Handler:** `playbookHandler(w http.ResponseWriter, r *http.Request)`  
**Location:** `main.go:470`

### Arguments:
- `w http.ResponseWriter` - HTTP response writer
- `r *http.Request` - HTTP request with JSON payload

### Request Structure:
```go
type PlaybookRequest struct {
    Playbook     interface{}            `json:"playbook"`      // Array of rules or single rule
    PlaybookName string                 `json:"playbook_name"` // File-based playbook name
    Context      map[string]interface{} `json:"context"`       // Execution context
    Options      map[string]interface{} `json:"options"`       // Execution options
}
```

---

## 2. Context Setup
**Function:** `s.engine.SetContext(context)`  
**Location:** `main.go:505`

### Arguments:
- `context map[string]interface{}` - Playbook execution context

---

## 3. Execution Router
**Location:** `main.go:514-527`

### Path A: Direct Playbook Array
```go
// Line 517
result, err = s.engine.EvaluatePlaybook(playbookArray)
```
**Arguments:**
- `playbookArray []interface{}` - Array of rule objects

### Path B: Single Rule
```go
// Line 520  
result, err = s.engine.EvaluateRule(req.Playbook)
```
**Arguments:**
- `req.Playbook interface{}` - Single rule object

### Path C: File-based Playbook
```go
// Line 524
result, err = s.executePlaybookFromFile(req.PlaybookName)
```
**Arguments:**
- `req.PlaybookName string` - Filename of playbook

---

## 4. Rules Engine Core Functions

### 4.1 EvaluatePlaybook
**Function:** `(re *Engine) EvaluatePlaybook(playbook []interface{}) ([]interface{}, error)`  
**Location:** `pkg/rules/engine.go:149`

#### Arguments:
- `playbook []interface{}` - Array of rule objects

#### Process:
1. Creates mutable context copy
2. Iterates through rules calling `re.evaluate(rule, playbookContext)`
3. Handles `PlaybookStopError` for early termination
4. Merges `run` operation results into context

#### Calls:
```go
// Line 160
result, err := re.evaluate(rule, playbookContext)
```

### 4.2 EvaluateRule  
**Function:** `(re *Engine) EvaluateRule(rule interface{}) (interface{}, error)`  
**Location:** `pkg/rules/engine.go:145`

#### Arguments:
- `rule interface{}` - Single rule object

#### Calls:
```go
// Line 146
return re.evaluate(rule, re.context)
```

---

## 5. Core Evaluation Engine

### 5.1 evaluate
**Function:** `(re *Engine) evaluate(expr interface{}, data map[string]interface{}) (interface{}, error)`  
**Location:** `pkg/rules/engine.go:209`

#### Arguments:
- `expr interface{}` - Rule/expression to evaluate
- `data map[string]interface{}` - Current context data

#### Process:
1. Checks expression cache
2. Processes template variables
3. Calls `re.evaluateExpression(processedExpr, data)`

### 5.2 evaluateExpression
**Function:** `(re *Engine) evaluateExpression(processedExpr interface{}, data map[string]interface{}) (interface{}, error)`  
**Location:** `pkg/rules/engine.go:231`

#### Arguments:
- `processedExpr interface{}` - Processed expression
- `data map[string]interface{}` - Context data

#### Route Map:
```go
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
case "error":
    return re.evaluateErrorOperation(value, data)
case "math":
    return re.evaluateMathOperation(value, data)
case "integration":
    return re.evaluateIntegrationStep(expr, data)
default:
    // Comparison/logical operations
}
```

---

## 6. Operation Handlers

### 6.1 evaluateRunOperation
**Function:** `(re *Engine) evaluateRunOperation(scriptName interface{}, operation map[string]interface{}, data map[string]interface{}) (interface{}, error)`  
**Location:** `pkg/rules/engine.go:306`

#### Arguments:
- `scriptName interface{}` - Script name to execute
- `operation map[string]interface{}` - Full operation object
- `data map[string]interface{}` - Context data

#### Decision Tree:
```go
// Line 327
if integrationName, hasIntegration := operation["run_i"]; hasIntegration {
    return re.executeInIntegrationContext(processedScriptName, integrationName, operation, data)
} else {
    return re.executeScript(scriptPath, enhancedData)
}
```

### 6.2 executeInIntegrationContext
**Function:** `(re *Engine) executeInIntegrationContext(scriptName string, integrationName interface{}, operation map[string]interface{}, data map[string]interface{}) (interface{}, error)`  
**Location:** `pkg/rules/engine.go:1275`

#### Arguments:
- `scriptName string` - Automation script name
- `integrationName interface{}` - Integration name (converted to string)
- `operation map[string]interface{}` - Operation parameters
- `data map[string]interface{}` - Context data

#### Process Flow:
1. **Validation:**
   ```go
   // Line 1278-1282
   if re.integrationManager == nil || re.clientIntegrationManager == nil {
       return nil, fmt.Errorf("integration managers not available")
   }
   ```

2. **Client ID Extraction:**
   ```go
   // Line 1291-1299  
   clientID, exists := data["client_id"]
   clientIDStr, ok := clientID.(string)
   ```

3. **Config Retrieval:**
   ```go
   // Line 1302
   clientConfig, err := re.clientIntegrationManager.GetClientIntegrationConfig(clientIDStr, integrationNameStr)
   ```

4. **Enabled Check:**
   ```go
   // Line 1312-1317
   if !clientConfig.Enabled {
       return map[string]interface{}{
           "success": false,
           "error":   "Integration is disabled for this client",
       }, nil
   }
   ```

5. **Delegation:**
   ```go
   // Line 1330
   return re.executeAutomationInIntegrationContext(scriptName, integrationNameStr, clientConfig, automationData)
   ```

### 6.3 executeAutomationInIntegrationContext
**Function:** `(re *Engine) executeAutomationInIntegrationContext(scriptName, integrationName string, clientConfig *integrations.ClientIntegrationConfig, automationData map[string]interface{}) (interface{}, error)`  
**Location:** `pkg/rules/engine.go:1333`

#### Arguments:
- `scriptName string` - Automation script filename
- `integrationName string` - Integration name  
- `clientConfig *integrations.ClientIntegrationConfig` - Client's integration configuration
- `automationData map[string]interface{}` - Enhanced context data

#### Process:
1. **Integration Definition Lookup:**
   ```go
   // Line 1336
   definition, err := re.integrationManager.GetDefinition(integrationName)
   ```

2. **Environment Check:**
   ```go
   // Line 1342
   isBuilt := re.integrationManager.IsIntegrationBuilt(integrationName)
   ```

3. **Script Execution:**
   - Builds context with integration configuration
   - Executes Python script in integration's virtual environment
   - Uses UV for isolated execution

### 6.4 evaluateIntegrationStep (NEW)
**Function:** `(re *Engine) evaluateIntegrationStep(step map[string]interface{}, data map[string]interface{}) (interface{}, error)`  
**Location:** `pkg/rules/engine.go:355`

#### Arguments:
- `step map[string]interface{}` - Integration step definition
- `data map[string]interface{}` - Context data

#### Expected Step Structure:
```json
{
  "type": "integration",
  "integration": "postgresql", 
  "function": "test_connection",
  "params": {}
}
```

#### Process Flow:
1. **Field Extraction:**
   ```go
   integrationName := step["integration"]
   function := step["function"] 
   params := step["params"]
   clientID := data["client_id"]
   ```

2. **Config Retrieval:**
   ```go
   clientConfig, err := re.clientIntegrationManager.GetClientIntegrationConfig(clientIDStr, integrationNameStr)
   ```

3. **Enabled Check:**
   ```go
   if !clientConfig.Enabled {
       return map[string]interface{}{"success": false, "error": "Integration is disabled"}, nil
   }
   ```

4. **Direct Integration Execution:**
   ```go
   result, err := re.integrationManager.ExecuteIntegration(integrationNameStr, clientConfig, functionStr, params)
   ```

---

## 7. Integration Manager Interface

### 7.1 ExecuteIntegration
**Function:** `(im *IntegrationManager) ExecuteIntegration(integrationName string, clientConfig *ClientIntegrationConfig, function string, params map[string]interface{}) (interface{}, error)`  
**Location:** `pkg/integrations/integrations.go:274`

#### Arguments:
- `integrationName string` - Integration name
- `clientConfig *ClientIntegrationConfig` - Client configuration
- `function string` - Function to execute
- `params map[string]interface{}` - Function parameters

#### Context Building:
```go
// Line 304-310
context := map[string]interface{}{
    "function":    function,
    "params":      params,
    "config":      clientConfig.Config,
    "credentials": clientConfig.Credentials,
    "client_id":   clientConfig.ClientID,
}
```

#### Execution:
- Uses UV run for Python integration execution
- Passes context via stdin as JSON
- Captures stdout/stderr with timeout

---

## 8. Execution Paths Summary

### Path 1: Script in Integration Context (`run_i`)
```
HTTP → playbookHandler → EvaluatePlaybook → evaluate → evaluateRunOperation 
→ executeInIntegrationContext → executeAutomationInIntegrationContext
```

**Data Flow:**
1. `{"run": "script.py", "run_i": "postgresql"}`
2. Client config retrieved and enabled check performed
3. Script executed in PostgreSQL Python environment
4. Integration config available to script

### Path 2: Direct Integration Call (`integration`)
```
HTTP → playbookHandler → EvaluatePlaybook → evaluate → evaluateIntegrationStep 
→ IntegrationManager.ExecuteIntegration
```

**Data Flow:**
1. `{"type": "integration", "integration": "postgresql", "function": "test_connection"}`
2. Client config retrieved and enabled check performed  
3. Integration function called directly with config/credentials

### Path 3: Simple Script (`run`)
```
HTTP → playbookHandler → EvaluatePlaybook → evaluate → evaluateRunOperation → executeScript
```

**Data Flow:**
1. `{"run": "script.py"}`
2. Script executed in standard Python environment
3. No integration context or config

---

## 9. Error Handling Patterns

### Consistent Error Returns:
```go
// Integration disabled
return map[string]interface{}{
    "success": false,
    "error":   "Integration is disabled for this client",
}, nil

// Integration execution failure
return map[string]interface{}{
    "success": false, 
    "error":   fmt.Sprintf("Integration execution failed: %v", err),
}, nil
```

### Critical Validation Points:
1. **Integration Managers Available** - `executeInIntegrationContext:1278`
2. **Client ID Present** - `executeInIntegrationContext:1291`
3. **Integration Configured** - `executeInIntegrationContext:1302`
4. **Integration Enabled** - `executeInIntegrationContext:1312` & `evaluateIntegrationStep:403`
5. **Integration Built** - `executeAutomationInIntegrationContext:1342`

This architecture provides a flexible, extensible playbook execution system with consistent validation and error handling across all integration execution paths.