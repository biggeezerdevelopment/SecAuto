# 🎯 SecAuto Playbook Development Guide

## 📋 Table of Contents
1. [Rules Engine Overview](#rules-engine-overview)
2. [Playbook Structure](#playbook-structure)
3. [Control Operations](#control-operations)
4. [Variable Management](#variable-management)
5. [Conditional Logic](#conditional-logic)
6. [Automation Execution](#automation-execution)
7. [Integration Context](#integration-context)
8. [Best Practices](#best-practices)
9. [Example Playbooks](#example-playbooks)

## 🏗️ Rules Engine Overview

The SecAuto rules engine is a high-performance JSON-based automation processor that executes **playbooks** as sequences of operations. It features:

### 🎯 **Core Features**
- **Template Variables**: `{{variable}}` syntax for parameter substitution
- **Variable Lookup**: `{"var": "path"}` for condition evaluation  
- **Conditional Logic**: Multi-condition if/then/else statements
- **Automation Execution**: Python script execution with context
- **Integration Context**: Client-specific integration execution
- **Caching System**: Performance optimization for repeated expressions
- **Context Management**: Mutable variable state throughout execution

### ⚙️ **Architecture Components**

| Component | Purpose | File |
|-----------|---------|------|
| **Rules Engine** | Core processing logic | `pkg/rules/engine.go` |
| **Playbook Manager** | File management and validation | `pkg/playbooks/playbooks.go` |
| **Context Cache** | Expression and variable caching | `pkg/cache/` |
| **Integration Manager** | Client-specific execution | `pkg/integrations/` |

## 📝 Playbook Structure

### 🗂️ **Basic Format**
```json
[
  {
    "metadata": {
      "name": "Playbook Name",
      "description": "What this playbook does",
      "version": "1.0",
      "author": "Your Name"
    }
  },
  {
    "operation": "value",
    "parameter": "{{variable}}"
  }
]
```

### 📊 **Metadata Fields**
```json
{
  "metadata": {
    "name": "Human-readable name",
    "description": "Detailed description",
    "version": "Semantic version (1.0.0)",
    "author": "Creator name",
    "category": "incident_response|threat_hunting|compliance",
    "tags": ["tag1", "tag2"],
    "created": "2024-12-29",
    "dependencies": ["integration1", "integration2"]
  }
}
```

## 🎛️ Control Operations

### ✅ **Supported Operations**

| Operation | Syntax | Purpose |
|-----------|--------|---------|
| `run` | `{"run": "script_name"}` | Execute Python automation |
| `run_i` | `{"run": "script", "run_i": "integration"}` | Execute in integration context |
| `if` | `{"if": {"condition": ..., "then": ..., "else": ...}}` | Conditional logic |
| `var` | `{"var": "variable_name"}` | Variable lookup |
| `play` | `{"play": "playbook_name"}` | Execute nested playbook |
| `plugin` | `{"plugin": {"name": "plugin_name"}}` | Execute Go plugin |
| `log` | `{"log": "message"}` | Logging output |
| `comment` | `{"comment": "description"}` | Documentation |

### 🔍 **Comparison Operators**

| Operator | Aliases | Description | Example |
|----------|---------|-------------|---------|
| `==` | `===`, `eq` | Equal to | `["==", {"var": "score"}, 100]` |
| `!=` | `!==`, `ne` | Not equal to | `["!=", {"var": "status"}, "closed"]` |
| `>` | `gt` | Greater than | `[">", {"var": "threat_score"}, 75]` |
| `<` | `lt` | Less than | `["<", {"var": "age"}, 30]` |
| `>=` | `gte` | Greater than or equal | `[">=", {"var": "confidence"}, 80]` |
| `<=` | `lte` | Less than or equal | `["<=", {"var": "attempts"}, 3]` |
| `contains` | | Container includes value | `["contains", {"var": "tags"}, "malware"]` |
| `in` | | Value exists in container | `["in", "admin", {"var": "user_roles"}]` |
| `matches` | | Regular expression match | `["matches", {"var": "email"}, ".*@company\\.com"]` |

### ⚡ **Logical Operators**

| Operator | Description | Example |
|----------|-------------|---------|
| `and` | All conditions true | `{"logic": "and", "conditions": [...]}` |
| `or` | Any condition true | `{"logic": "or", "conditions": [...]}` |
| `not` | Negates condition | `{"not": condition}` |

## 🔧 Variable Management

### 1️⃣ **Template Variables** `{{variable}}`

**Purpose**: Parameter substitution in automation calls
**Processing**: String conversion for parameter passing
**Use Cases**: Passing values to scripts and integrations

```json
{
  "run": "virustotal_scanner",
  "api_key": "{{virustotal_api_key}}",
  "urls": "{{incident.suspicious_urls}}",
  "timeout": "{{scan_timeout}}"
}
```

**Supported Patterns**:
- Direct access: `{{incident_id}}`
- Nested objects: `{{threat_intelligence.domains}}`
- Array access: `{{results.detections.0}}`

### 2️⃣ **Variable Lookup** `{"var": "path"}`

**Purpose**: Condition evaluation and logic operations
**Processing**: Type-preserved values for comparison
**Use Cases**: Conditional logic, mathematical operations

```json
{
  "if": {
    "condition": ["gt", {"var": "threat_score"}, 75],
    "then": {"run": "high_priority_response"}
  }
}
```

**Dot Notation Support**:
- Simple: `{"var": "user_id"}`
- Nested: `{"var": "analysis.virustotal.malicious_count"}`
- Deep: `{"var": "network.scan.results.0.vulnerabilities"}`

### 3️⃣ **Variable Assignment**

```json
{
  "var": "threat_level",
  "value": "HIGH"
}
```

### 4️⃣ **Context Updates**

The engine automatically updates context from:
- **Variable operations**: `var` assignments add to context
- **Script results**: `run` operations merge JSON results
- **Integration responses**: `run_i` operations add integration data

## 🎯 Conditional Logic

### 📋 **Simple Conditions**

```json
{
  "if": {
    "condition": ["==", {"var": "incident.status"}, "open"],
    "then": {"log": "Processing open incident"},
    "else": {"log": "Incident already closed"}
  }
}
```

### 🔄 **Multi-Condition Logic**

```json
{
  "if": {
    "conditions": [
      ["gt", {"var": "threat_score"}, 80],
      ["==", {"var": "department"}, "Finance"],
      ["contains", {"var": "incident.tags"}, "urgent"]
    ],
    "logic": "and",
    "then": [
      {"log": "Critical finance security incident"},
      {"run": "emergency_response"},
      {"var": "escalated", "value": true}
    ],
    "else": {"run": "standard_processing"}
  }
}
```

### 🌳 **Nested Conditions**

```json
{
  "if": {
    "condition": ["gt", {"var": "threat_score"}, 90],
    "then": {"run": "critical_response"},
    "else": {
      "if": {
        "condition": ["gt", {"var": "threat_score"}, 60],
        "then": {"run": "elevated_response"},
        "else": {"run": "standard_response"}
      }
    }
  }
}
```

### 🔗 **Complex Logical Expressions**

```json
{
  "if": {
    "conditions": [
      {
        "or": [
          ["contains", {"var": "indicators"}, "malware"],
          ["gt", {"var": "virustotal.detections"}, 5]
        ]
      },
      {
        "and": [
          ["==", {"var": "source.trusted"}, false],
          ["lt", {"var": "reputation_score"}, 50]
        ]
      }
    ],
    "logic": "or",
    "then": {"run": "threat_mitigation"}
  }
}
```

## 🚀 Automation Execution

### 1️⃣ **Basic Script Execution**

```json
{
  "run": "ip_reputation_check",
  "ip_addresses": "{{incident.suspicious_ips}}",
  "confidence_threshold": 75
}
```

**How it works**:
1. Engine calls `evaluateRunOperation()`
2. Script path resolved: `config.GetScriptPath("ip_reputation_check")`
3. Context merged with parameters
4. Python process executed with JSON context
5. Results parsed and merged into playbook context

### 2️⃣ **Script with Custom Parameters**

```json
{
  "run": "vulnerability_scanner",
  "targets": "{{network.endpoints}}",
  "scan_type": "comprehensive",
  "exclude_ports": [22, 443],
  "timeout": 300,
  "report_format": "json"
}
```

### 3️⃣ **Error Handling in Scripts**

Scripts should return JSON with standard format:
```json
{
  "success": true,
  "results": { ... },
  "error": null,
  "timestamp": "2024-12-29T10:30:00Z"
}
```

## 🔗 Integration Context

### 🏢 **Client-Specific Execution**

```json
{
  "run": "domain_analysis",
  "run_i": "virustotal",
  "domains": "{{incident.suspicious_domains}}",
  "api_version": "v3"
}
```

**Process Flow**:
1. Engine calls `executeInIntegrationContext()`
2. Retrieves client integration configuration
3. Merges client credentials with automation parameters
4. Executes in integration's Python environment
5. Returns results with integration context

### 🔧 **Integration Requirements**

```json
{
  "context": {
    "client_id": "acme-corp-001",
    "integration_name": "virustotal",
    "integration_config": {
      "api_key": "client_specific_key",
      "rate_limit": 1000,
      "enable_premium": true
    }
  }
}
```

### 📊 **Integration Response Format**

```json
{
  "integration": "virustotal",
  "client_id": "acme-corp-001",
  "results": {
    "scanned_domains": 5,
    "malicious_count": 2,
    "clean_count": 3,
    "detections": [...]
  },
  "metadata": {
    "scan_time": "2024-12-29T10:30:00Z",
    "api_version": "v3",
    "credits_used": 5
  }
}
```

## 💡 Best Practices

### 1️⃣ **Playbook Design**

- **Use Metadata**: Always include comprehensive metadata
- **Modular Design**: Break complex logic into reusable playbooks
- **Clear Naming**: Use descriptive operation names and comments
- **Error Handling**: Include fallback logic for failed operations

### 2️⃣ **Variable Management**

- **Consistent Naming**: Use clear, consistent variable names
- **Type Awareness**: Understand template vs. variable lookup differences
- **Context Planning**: Plan variable flow through playbook stages

### 3️⃣ **Performance Optimization**

- **Cache Awareness**: Leverage built-in expression caching
- **Conditional Execution**: Use conditions to avoid unnecessary operations
- **Batch Operations**: Group related operations together

### 4️⃣ **Security Considerations**

- **Validate Inputs**: Always validate external data
- **Secure Templates**: Be careful with template variable expansion
- **Integration Security**: Use client-specific configurations

## 📚 Example Playbooks

### 🔰 **Basic Example** 
See `example_basic_playbook.json` - Demonstrates fundamental controls:
- Variable operations
- Simple conditionals
- Automation execution
- Integration context

### 🎯 **Advanced Example**
See `example_comprehensive_playbook.json` - Demonstrates:
- Multi-stage investigation workflow
- Complex conditional logic
- Integration orchestration
- Threat scoring and classification
- Automated response decisions

### 🧪 **Testing Template**

```json
[
  {
    "metadata": {
      "name": "Test Playbook Template",
      "description": "Template for testing playbook operations"
    }
  },
  {
    "var": "test_data",
    "value": {
      "input": "{{context.test_input}}",
      "expected": "{{context.expected_result}}"
    }
  },
  {
    "run": "test_automation",
    "test_input": "{{test_data.input}}"
  },
  {
    "if": {
      "condition": ["==", {"var": "context.test_result"}, {"var": "test_data.expected"}],
      "then": {"log": "✅ Test passed"},
      "else": {"log": "❌ Test failed"}
    }
  }
]
```

## 🔬 Troubleshooting

### ❌ **Common Issues**

1. **Template Not Resolved**
   ```json
   // ❌ Wrong
   {"eq": ["{{variable}}", 0]}
   
   // ✅ Correct  
   ["==", {"var": "variable"}, 0]
   ```

2. **Type Mismatch**
   ```json
   // ❌ Wrong - comparing number to string
   ["==", {"var": "count"}, "5"]
   
   // ✅ Correct
   ["==", {"var": "count"}, 5]
   ```

3. **Variable Path Error**
   ```json
   // ❌ Wrong - undefined path
   {"var": "results.analysis.nonexistent"}
   
   // ✅ Correct - check path exists
   {"var": "results.analysis.summary"}
   ```

### 🐛 **Debugging Tips**

1. **Add Logging**: Use `{"log": "Debug: {{variable}}"}` to inspect values
2. **Check Context**: Verify variable paths exist in context
3. **Validate JSON**: Ensure playbook JSON is valid
4. **Test Incrementally**: Build complex logic step by step

---

## 🎉 Conclusion

The SecAuto rules engine provides a powerful, flexible platform for security automation. By understanding these controls and patterns, you can build sophisticated incident response and threat hunting playbooks that adapt to your organization's specific needs.

**Key Takeaways**:
- Use template variables `{{}}` for parameter passing
- Use variable lookup `{"var": ""}` for conditions
- Structure complex logic with nested conditionals
- Leverage integration context for client-specific operations
- Follow best practices for maintainable playbooks

Happy automating! 🚀
