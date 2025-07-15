# SecAuto Plugin Development Guide

This guide covers how to create plugins for SecAuto in both Python and Go, including common pitfalls and best practices learned from real development experience.

## Table of Contents

1. [Overview](#overview)
2. [Plugin Types](#plugin-types)
3. [Python Plugin Development](#python-plugin-development)
4. [Go Plugin Development](#go-plugin-development)
5. [Common Pitfalls and Solutions](#common-pitfalls-and-solutions)
6. [Best Practices](#best-practices)
7. [Testing Plugins](#testing-plugins)
8. [Debugging Plugins](#debugging-plugins)
9. [Plugin Integration with Playbooks](#plugin-integration-with-playbooks)

## Overview

SecAuto supports two types of plugins:
- **Python Plugins**: Scripts that can be executed from playbooks
- **Go Plugins**: Compiled binaries that integrate directly with the system

Plugins provide extensibility for:
- Automation tasks
- Data enrichment
- External integrations
- Custom validators
- Complex business logic

## Plugin Types

### Python Plugins
- **Location**: `plugins/` directory
- **Extension**: `.py` files
- **Execution**: Via Python interpreter from virtual environment
- **Use Case**: Quick prototyping, data processing, external API calls

### Go Plugins
- **Location**: `plugins/` directory
- **Extension**: `.go` files (compiled to binaries)
- **Execution**: Direct binary execution
- **Use Case**: High-performance operations, system integration

## Python Plugin Development

### Basic Structure

```python
#!/usr/bin/env python3
"""
Example Plugin for SecAuto

This plugin demonstrates how to create a plugin that can be executed
from playbooks using the "plugin" operation.
"""

import json
import sys
import time
from typing import Dict, Any, Optional


class ExamplePlugin:
    """Example plugin that can be executed from playbooks."""
    
    def __init__(self):
        self.name = "example_plugin"
        self.version = "1.0.0"
        self.description = "Example plugin for SecAuto playbooks"
        self.author = "SecAuto Team"
        self.type = "automation"
    
    def get_info(self) -> Dict[str, Any]:
        """Return plugin information."""
        return {
            "name": self.name,
            "version": self.version,
            "description": self.description,
            "author": self.author,
            "type": self.type,
            "supported_operations": ["process_incident", "enrich_data"]
        }
    
    def execute(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute the plugin with given parameters."""
        print(f"Example plugin executing with params: {params}", file=sys.stderr)
        
        # Extract context from parameters
        incident = params.get("incident", {})
        user = params.get("user", {})
        
        # Simulate some processing
        time.sleep(0.1)  # Simulate work
        
        # Add enriched data to the context
        enriched_data = {
            "plugin_processed": True,
            "processing_timestamp": time.time(),
            "incident": {
                **incident,  # Preserve existing incident data
                "enriched_by_plugin": True,
                "plugin_confidence": 0.85,
                "additional_indicators": [
                    "suspicious_activity_detected",
                    "threat_level_elevated"
                ]
            },
            "user": {
                **user,  # Preserve existing user data
                "last_processed_by": "example_plugin"
            }
        }
        
        return enriched_data
    
    def cleanup(self):
        """Perform cleanup operations."""
        print("Example plugin cleanup completed", file=sys.stderr)


def main():
    """Main entry point for the plugin."""
    if len(sys.argv) < 2:
        print("Usage: python example_plugin.py <command> [params_json]", file=sys.stderr)
        sys.exit(1)
    
    command = sys.argv[1]
    plugin = ExamplePlugin()
    
    if command == "info":
        # Return plugin information
        info = plugin.get_info()
        print(json.dumps(info, indent=2))
        
    elif command == "execute":
        # Execute plugin with parameters
        if len(sys.argv) < 3:
            print("Error: No parameters provided for execute command", file=sys.stderr)
            sys.exit(1)
        
        try:
            # Log the raw input for debugging
            print(f"DEBUG_RAW_INPUT: '{sys.argv[2]}'", file=sys.stderr)
            
            params = json.loads(sys.argv[2])
            result = plugin.execute(params)
            print(json.dumps(result, indent=2))
        except json.JSONDecodeError as e:
            print(f"Error: Invalid JSON parameters: {e}", file=sys.stderr)
            print(f"Error: Raw input was: '{sys.argv[2]}'", file=sys.stderr)
            sys.exit(1)
        except Exception as e:
            print(f"Error: Plugin execution failed: {e}", file=sys.stderr)
            import traceback
            print(f"Error: Traceback: {traceback.format_exc()}", file=sys.stderr)
            sys.exit(1)
            
    elif command == "cleanup":
        # Perform cleanup
        plugin.cleanup()
        print("Cleanup completed", file=sys.stderr)
        
    else:
        print(f"Error: Unknown command '{command}'", file=sys.stderr)
        print("Available commands: info, execute, cleanup", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
```

### Critical Requirements

1. **JSON Output Only**: All output to stdout must be valid JSON
2. **Error Output to stderr**: All debug, error, and informational messages must go to stderr
3. **Command Interface**: Must support `info`, `execute`, and `cleanup` commands
4. **Parameter Handling**: Must accept JSON parameters via command line
5. **Return Structure**: Must return valid JSON that can be parsed by Go

## Go Plugin Development

### Basic Structure

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"
)

// PluginInfo represents plugin metadata
type PluginInfo struct {
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Description string   `json:"description"`
    Author      string   `json:"author"`
    Type        string   `json:"type"`
    Operations  []string `json:"supported_operations"`
}

// PluginResult represents the result of plugin execution
type PluginResult struct {
    Status    string                 `json:"status"`
    Message   string                 `json:"message"`
    Data      map[string]interface{} `json:"data"`
    Timestamp time.Time              `json:"timestamp"`
}

// ExamplePlugin represents an example Go plugin
type ExamplePlugin struct {
    info PluginInfo
}

// NewExamplePlugin creates a new example plugin
func NewExamplePlugin() *ExamplePlugin {
    return &ExamplePlugin{
        info: PluginInfo{
            Name:        "example_go_plugin",
            Version:     "1.0.0",
            Description: "Example Go plugin for SecAuto",
            Author:      "SecAuto Team",
            Type:        "automation",
            Operations:  []string{"process_incident", "enrich_data"},
        },
    }
}

// GetInfo returns plugin information
func (p *ExamplePlugin) GetInfo() PluginInfo {
    return p.info
}

// Execute runs the plugin with given parameters
func (p *ExamplePlugin) Execute(params map[string]interface{}) PluginResult {
    log.Printf("Go plugin executing with params: %+v", params)
    
    // Extract context from parameters
    incident, _ := params["incident"].(map[string]interface{})
    user, _ := params["user"].(map[string]interface{})
    
    // Simulate processing
    time.Sleep(100 * time.Millisecond)
    
    // Create enriched data
    enrichedData := map[string]interface{}{
        "plugin_processed": true,
        "processing_timestamp": time.Now().Unix(),
        "incident": map[string]interface{}{
            "enriched_by_plugin": true,
            "plugin_confidence":   0.95,
            "additional_indicators": []string{
                "go_plugin_processed",
                "high_confidence_result",
            },
        },
        "user": map[string]interface{}{
            "last_processed_by": "example_go_plugin",
        },
    }
    
    // Merge with existing incident data if present
    if incident != nil {
        for k, v := range incident {
            enrichedData["incident"].(map[string]interface{})[k] = v
        }
    }
    
    return PluginResult{
        Status:    "success",
        Message:   "Plugin executed successfully",
        Data:      enrichedData,
        Timestamp: time.Now(),
    }
}

func main() {
    if len(os.Args) < 2 {
        log.Fatal("Usage: ./example_go_plugin <command> [params_json]")
    }
    
    command := os.Args[1]
    plugin := NewExamplePlugin()
    
    switch command {
    case "info":
        info := plugin.GetInfo()
        json.NewEncoder(os.Stdout).Encode(info)
        
    case "execute":
        if len(os.Args) < 3 {
            log.Fatal("Error: No parameters provided for execute command")
        }
        
        var params map[string]interface{}
        if err := json.Unmarshal([]byte(os.Args[2]), &params); err != nil {
            log.Fatalf("Error: Invalid JSON parameters: %v", err)
        }
        
        result := plugin.Execute(params)
        json.NewEncoder(os.Stdout).Encode(result)
        
    case "cleanup":
        log.Println("Go plugin cleanup completed")
        
    default:
        log.Fatalf("Error: Unknown command '%s'", command)
    }
}
```

## Common Pitfalls and Solutions

### 1. JSON Output Pollution

**Problem**: Debug messages printed to stdout break JSON parsing
```python
# WRONG - This breaks JSON parsing
print("Debug: Processing data...")  # Goes to stdout
print(json.dumps(result))           # Invalid JSON due to previous output
```

**Solution**: Use stderr for all non-JSON output
```python
# CORRECT - Debug messages to stderr, only JSON to stdout
print("Debug: Processing data...", file=sys.stderr)
print(json.dumps(result))  # Clean JSON to stdout
```

### 2. Plugin Name Mismatch

**Problem**: Plugin manager stores plugins by filename but rules engine looks up by plugin name
```python
# WRONG - Plugin name doesn't match filename
class MyPlugin:
    def __init__(self):
        self.name = "my_plugin"  # Different from filename
```

**Solution**: Ensure plugin name matches filename
```python
# CORRECT - Plugin name matches filename
class ExamplePlugin:
    def __init__(self):
        self.name = "example_plugin"  # Matches example_plugin.py
```

### 3. Context Data Overwriting

**Problem**: Plugins overwrite entire objects instead of merging
```python
# WRONG - Overwrites entire incident object
return {
    "incident": {
        "new_field": "value"  # Loses existing incident data
    }
}
```

**Solution**: Preserve existing data using spread operator
```python
# CORRECT - Merges with existing incident data
return {
    "incident": {
        **incident,  # Preserve existing data
        "new_field": "value"  # Add new data
    }
}
```

### 4. Timezone Issues

**Problem**: Timestamps without timezone info cause parsing errors
```python
# WRONG - No timezone info
timestamp = datetime.now().isoformat()
```

**Solution**: Use UTC timestamps with timezone info
```python
# CORRECT - UTC timestamp with timezone
from datetime import datetime, timezone
timestamp = datetime.now(timezone.utc).isoformat()
```

### 5. Command Line Argument Escaping

**Problem**: Special characters in JSON arguments cause parsing issues
```bash
# WRONG - Unescaped special characters
python plugin.py execute '{"key": "value with spaces"}'
```

**Solution**: Proper JSON escaping
```bash
# CORRECT - Properly escaped JSON
python plugin.py execute '{"key": "value with spaces"}'
```

### 6. Plugin Loading Errors

**Problem**: Exit status 9009 (command not found) on Windows
```
Error: plugin not found: example_plugin
```

**Solution**: Ensure proper file extensions and paths
```python
# Add proper shebang and ensure .py extension
#!/usr/bin/env python3
# File must be named: example_plugin.py
```

## Best Practices

### 1. Error Handling

```python
def execute(self, params):
    try:
        # Plugin logic here
        result = process_data(params)
        return result
    except Exception as e:
        # Log error to stderr
        print(f"Plugin error: {e}", file=sys.stderr)
        # Return error structure
        return {
            "error": str(e),
            "status": "failed"
        }
```

### 2. Input Validation

```python
def execute(self, params):
    # Validate required parameters
    if "incident" not in params:
        return {"error": "Missing incident parameter", "status": "failed"}
    
    incident = params["incident"]
    if not isinstance(incident, dict):
        return {"error": "Invalid incident format", "status": "failed"}
    
    # Process validated data
    return process_incident(incident)
```

### 3. Logging and Debugging

```python
def execute(self, params):
    # Debug logging to stderr
    print(f"DEBUG: Received params: {params}", file=sys.stderr)
    
    # Process data
    result = process_data(params)
    
    # Debug logging to stderr
    print(f"DEBUG: Returning result: {result}", file=sys.stderr)
    
    # Clean JSON to stdout
    return result
```

### 4. Data Preservation

```python
def execute(self, params):
    incident = params.get("incident", {})
    
    # Always preserve existing data
    enriched_incident = {
        **incident,  # Preserve existing fields
        "enriched_by_plugin": True,
        "plugin_timestamp": datetime.now(timezone.utc).isoformat(),
        "new_field": "new_value"
    }
    
    return {
        "incident": enriched_incident,
        "plugin_status": "completed"
    }
```

## Testing Plugins

### Manual Testing

```bash
# Test plugin info
python example_plugin.py info

# Test plugin execution
python example_plugin.py execute '{"incident": {"id": "test"}}'

# Test plugin cleanup
python example_plugin.py cleanup
```

### Integration Testing

```python
# test_plugin.py
import json
import subprocess
import sys

def test_plugin():
    # Test info command
    result = subprocess.run([
        "python", "example_plugin.py", "info"
    ], capture_output=True, text=True)
    
    if result.returncode != 0:
        print("INFO test failed")
        return False
    
    # Test execute command
    test_params = {
        "incident": {"id": "test-123"},
        "user": {"name": "test_user"}
    }
    
    result = subprocess.run([
        "python", "example_plugin.py", "execute",
        json.dumps(test_params)
    ], capture_output=True, text=True)
    
    if result.returncode != 0:
        print("EXECUTE test failed")
        return False
    
    print("All tests passed")
    return True

if __name__ == "__main__":
    test_plugin()
```

## Debugging Plugins

### 1. Enable Debug Logging

```python
# Add debug logging to your plugin
import logging

logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    stream=sys.stderr  # Important: Use stderr
)
```

### 2. Check Server Logs

```bash
# Monitor server logs for plugin execution
tail -f logs/secauto.log | grep plugin
```

### 3. Test Plugin Isolation

```bash
# Test plugin directly
python plugins/example_plugin.py execute '{"test": "data"}'
```

## Plugin Integration with Playbooks

### Basic Plugin Usage

```json
[
  {
    "plugin": "example_plugin"
  }
]
```

### Conditional Plugin Execution

```json
[
  {
    "if": {
      "conditions": [
        {"gt": [{"var": "incident.threat_score"}, 50]}
      ],
      "logic": "and",
      "true": {
        "plugin": "high_threat_plugin"
      },
      "false": {
        "plugin": "low_threat_plugin"
      }
    }
  }
]
```

### Plugin with Parameters

```json
[
  {
    "plugin": {
      "name": "example_plugin",
      "params": {
        "custom_param": "value",
        "threshold": 75
      }
    }
  }
]
```

### Plugin in Workflow

```json
[
  {
    "run": "data_enrichment"
  },
  {
    "plugin": "example_plugin"
  },
  {
    "if": {
      "conditions": [
        {"gt": [{"var": "incident.threat_score"}, 50]}
      ],
      "logic": "and",
      "true": {
        "run": "notification_system"
      }
    }
  }
]
```

## Configuration

### Plugin Configuration in config.yaml

```yaml
plugins:
  enabled: true
  directory: "../plugins"
  hot_reload: true
  reload_interval: 30
  supported_types: ["automation", "playbook", "integration", "validator"]
  max_plugins: 100
  plugin_timeout: 300
  sandbox_mode: false
  allow_executables: true
  allow_python: true
  allow_go_plugins: true
  plugin_validation: true
  plugin_logging: true
```

### Environment Variables

```bash
# Set plugin directory
export SECAUTO_PLUGIN_DIR=/path/to/plugins

# Set plugin timeout
export SECAUTO_PLUGIN_TIMEOUT=300

# Enable debug mode
export SECAUTO_DEBUG=true
```

## Troubleshooting

### Common Error Messages

1. **"plugin not found"**
   - Check plugin filename matches plugin name
   - Ensure plugin is in correct directory
   - Verify plugin has proper permissions

2. **"invalid character" in JSON**
   - Check for debug output in stdout
   - Ensure all non-JSON output goes to stderr
   - Validate JSON structure

3. **"exit status 9009" (Windows)**
   - Ensure Python is in PATH
   - Check file extensions (.py)
   - Verify virtual environment setup

4. **"time parsing error"**
   - Use UTC timestamps with timezone info
   - Format: `2025-07-11T12:00:00Z`

5. **"variable not found"**
   - Check context structure
   - Verify data merging in plugins
   - Ensure dot notation paths are correct

### Debug Commands

```bash
# Check plugin loading
curl -X GET "http://localhost:8080/plugins" \
  -H "X-API-Key: your-api-key"

# Test plugin execution
curl -X POST "http://localhost:8080/playbook/async" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "playbook": "plugin_example.json",
    "context": {"incident": {"id": "test"}}
  }'
```

## Conclusion

Plugin development for SecAuto requires careful attention to:
- JSON output formatting
- Error handling and logging
- Data preservation and merging
- Cross-platform compatibility
- Proper integration with the rules engine

Following these guidelines will help you create robust, maintainable plugins that integrate seamlessly with the SecAuto platform. 