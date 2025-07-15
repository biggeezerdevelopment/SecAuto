# SecAuto Rule Development Guide

## Table of Contents
1. [Overview](#overview)
2. [Basic Rule Structure](#basic-rule-structure)
3. [Variable Resolution](#variable-resolution)
4. [Conditional Logic](#conditional-logic)
5. [Automation Execution](#automation-execution)
6. [Common Patterns](#common-patterns)
7. [Troubleshooting](#troubleshooting)
8. [Best Practices](#best-practices)

## Overview

SecAuto uses a JSONLogic-based rules engine that processes playbooks as arrays of operations. Each operation can be an automation execution, conditional logic, or variable manipulation.

### Key Concepts
- **Operations**: Individual actions in a playbook (run, if, var, etc.)
- **Context**: The data available to all operations in a playbook
- **Template Variables**: `{{variable}}` syntax for parameter substitution
- **Variable Lookup**: `{"var": "path"}` syntax for condition evaluation

## Basic Rule Structure

### Playbook Format
```json
[
  {
    "run": "automation_name",
    "parameter": "value"
  },
  {
    "if": {
      "conditions": [...],
      "logic": "and",
      "true": {...},
      "false": {...}
    }
  }
]
```

### Supported Operations
- `run`: Execute Python automation
- `if`: Conditional logic
- `play`: Execute nested playbook
- `plugin`: Execute Go plugin
- `var`: Variable lookup

## Variable Resolution

### Template Variables (`{{...}}`)
**Use for:** Parameter substitution in automation calls

**Syntax:**
```json
{
  "run": "virustotal_url_scanner",
  "urls": "{{threat_intelligence.domains}}",
  "api_key": "{{virustotal_api_key}}"
}
```

**How it works:**
- Processed by `processTemplateVariables()` before automation execution
- Resolves dot notation: `{{threat_intelligence.domains}}` → `["malicious.example.com", "suspicious.net"]`
- Converts values to strings for parameter passing

**Supported patterns:**
```json
// Direct context access
"{{incident.id}}"

// Nested object access
"{{threat_intelligence.domains}}"

// Array access (if supported)
"{{threat_intelligence.ip_addresses.0}}"
```

### Variable Lookup (`{"var": "..."}`)
**Use for:** Condition evaluation and logic operations

**Syntax:**
```json
{"var": "virustotal.summary.malicious_urls"}
```

**How it works:**
- Processed by `evaluateVarOperation()` → `evaluateDotNotation()`
- Returns actual values (not strings) for comparison
- Handles type conversion properly

**Supported patterns:**
```json
// Direct context access
{"var": "incident.id"}

// Nested object access
{"var": "threat_intelligence.domains"}

// Deep nesting
{"var": "virustotal.results.0.verdict.stats.malicious"}
```

## Conditional Logic

### If Statement Structure

**Object-based format (recommended):**
```json
{
  "if": {
    "conditions": [
      ["==", {"var": "virustotal.summary.malicious_urls"}, 0],
      ["gt", {"var": "incident.threat_score"}, 50]
    ],
    "logic": "and",
    "true": {"run": "safe_automation"},
    "false": {"run": "incident_response"}
  }
}
```

**Array-based format (legacy):**
```json
{
  "if": [
    {"eq": ["{{virustotal.summary.malicious_urls}}", 0]},
    {"run": "safe_automation"},
    {"run": "incident_response"}
  ]
}
```

### Comparison Operators

**Supported operators:**
- `"eq"` or `"=="` or `"==="` - Equality
- `"gt"` or `">"` - Greater than
- `"lt"` or `"<"` - Less than
- `"gte"` or `">="` - Greater than or equal
- `"lte"` or `"<="` - Less than or equal
- `"!="` or `"!=="` - Not equal

**Format for conditions:**
```json
["operator", {"var": "path.to.value"}, comparison_value]
```

**Examples:**
```json
// Numeric comparisons
["==", {"var": "virustotal.summary.malicious_urls"}, 0]
["gt", {"var": "incident.threat_score"}, 75]
["lte", {"var": "user_context.login_count"}, 10]

// String comparisons
["==", {"var": "incident.status"}, "open"]
["!=", {"var": "user_context.department"}, "IT"]
```

### Logical Operators

**Supported operators:**
- `"and"` - All conditions must be true
- `"or"` - At least one condition must be true
- `"not"` - Negates a condition

**Examples:**
```json
{
  "if": {
    "conditions": [
      ["==", {"var": "virustotal.summary.malicious_urls"}, 0],
      ["gt", {"var": "incident.threat_score"}, 50]
    ],
    "logic": "and",
    "true": {"run": "safe_automation"},
    "false": {"run": "incident_response"}
  }
}
```

## Automation Execution

### Basic Automation Call
```json
{
  "run": "automation_name",
  "parameter1": "value1",
  "parameter2": "value2"
}
```

### With Template Variables
```json
{
  "run": "virustotal_url_scanner",
  "urls": "{{threat_intelligence.domains}}",
  "api_key": "{{virustotal_api_key}}",
  "timeout": "{{scan_timeout}}"
}
```

### Context Access in Automations
Python automations receive the full context as a flat dictionary:
```python
def main(context=None):
    # Access context data
    urls = context.get('urls', [])
    threat_score = context.get('incident', {}).get('threat_score', 0)
    
    # Process and return results
    return {"result": "success", "processed_urls": len(urls)}
```

## Common Patterns

### 1. Data Enrichment → Analysis → Response
```json
[
  {
    "run": "data_enrichment"
  },
  {
    "run": "virustotal_url_scanner",
    "urls": "{{threat_intelligence.domains}}"
  },
  {
    "if": {
      "conditions": [
        ["==", {"var": "virustotal.summary.malicious_urls"}, 0]
      ],
      "logic": "and",
      "true": {"run": "safe_automation"},
      "false": {"run": "incident_response"}
    }
  }
]
```

### 2. Multi-Condition Decision Tree
```json
{
  "if": {
    "conditions": [
      ["gt", {"var": "incident.threat_score"}, 90],
      ["==", {"var": "virustotal.summary.malicious_urls"}, 0]
    ],
    "logic": "and",
    "true": {"run": "high_priority_response"},
    "false": {
      "if": {
        "conditions": [
          ["gt", {"var": "incident.threat_score"}, 50]
        ],
        "logic": "and",
        "true": {"run": "medium_priority_response"},
        "false": {"run": "low_priority_response"}
      }
    }
  }
}
```

### 3. Plugin Execution
```json
{
  "plugin": {
    "name": "tcp_scanner",
    "params": {
      "target": "{{incident.affected_asset}}",
      "ports": "{{scan_ports}}"
    }
  }
}
```

### 4. Nested Playbook Execution
```json
{
  "play": "incident_response_playbook"
}
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Template Variable Not Resolved
**Problem:**
```json
{"eq": ["{{virustotal.summary.malicious_urls}}", 0]}
```
**Error:** Condition evaluates to false when it should be true

**Solution:** Use variable lookup instead
```json
["==", {"var": "virustotal.summary.malicious_urls"}, 0]
```

#### 2. Unknown Operation Error
**Problem:**
```
unknown operation: map[==:[0 0]]
```

**Solution:** Use the correct operator format
```json
// ❌ Wrong
{"==": ["{{variable}}", 0]}

// ✅ Correct
["==", {"var": "variable"}, 0]
```

#### 3. Type Mismatch in Comparison
**Problem:**
```json
["==", {"var": "virustotal.summary.malicious_urls"}, "0"]
```
**Error:** Comparing integer with string

**Solution:** Use matching types
```json
["==", {"var": "virustotal.summary.malicious_urls"}, 0]
```

#### 4. Variable Not Found
**Problem:**
```
Template variable not found: virustotal.summary.malicious_urls
```

**Solution:** Check context structure and variable path
```json
// Verify the path exists in context
{"var": "virustotal.summary.malicious_urls"}
```

### Debugging Tips

1. **Check Context Structure:**
   ```json
   {
     "run": "debug_context",
     "output": "{{context}}"
   }
   ```

2. **Test Individual Conditions:**
   ```json
   {
     "run": "test_condition",
     "condition": ["==", {"var": "virustotal.summary.malicious_urls"}, 0]
   }
   ```

3. **Verify Template Variables:**
   ```json
   {
     "run": "test_template",
     "test_var": "{{threat_intelligence.domains}}"
   }
   ```

## Best Practices

### 1. Variable Resolution
- **Use `{"var": "..."}` for conditions** - Ensures proper type handling
- **Use `{{...}}` for parameters** - Good for string substitution
- **Never mix the two** in the same context

### 2. Condition Structure
- **Use array format for comparisons:** `["operator", {"var": "path"}, value]`
- **Use object format for if statements:** `{"if": {"conditions": [...], "logic": "and"}}`
- **Match data types:** Compare numbers with numbers, strings with strings

### 3. Automation Design
- **Return structured data:** Use consistent JSON output format
- **Handle errors gracefully:** Include error messages in output
- **Use descriptive names:** Make automation and parameter names clear

### 4. Playbook Organization
- **Logical flow:** Data enrichment → Analysis → Decision → Action
- **Error handling:** Include fallback automations for failures
- **Modularity:** Break complex playbooks into smaller, reusable pieces

### 5. Testing
- **Test individual operations:** Verify each automation works independently
- **Test condition logic:** Ensure conditions evaluate as expected
- **Test with real data:** Use realistic context data for testing

## Example Playbook

```json
[
  {
    "run": "data_enrichment"
  },
  {
    "run": "virustotal_url_scanner",
    "urls": "{{threat_intelligence.domains}}"
  },
  {
    "if": {
      "conditions": [
        ["==", {"var": "virustotal.summary.malicious_urls"}, 0],
        ["gt", {"var": "incident.threat_score"}, 50]
      ],
      "logic": "and",
      "true": {
        "run": "safe_automation",
        "message": "No malicious URLs detected"
      },
      "false": {
        "run": "incident_response",
        "priority": "high"
      }
    }
  },
  {
    "if": {
      "conditions": [
        ["gt", {"var": "virustotal.summary.malicious_urls"}, 5]
      ],
      "logic": "and",
      "true": {
        "run": "emergency_response",
        "escalation": "immediate"
      }
    }
  }
]
```

This guide should help you develop reliable, maintainable SecAuto playbooks that follow the engine's patterns and avoid common pitfalls. 