# SecAuto Plugin System

The SecAuto Plugin System provides a modular architecture for extending SecAuto's capabilities with custom automation scripts, playbook types, integrations, and validators. The system supports both Go and Python plugins with hot-reload capabilities.

## 🏗️ Architecture Overview

### Plugin Types

- **Automation Plugins**: Custom automation scripts and workflows
- **Playbook Plugins**: Custom playbook types and execution engines
- **Integration Plugins**: Third-party system integrations
- **Validator Plugins**: Custom validation rules and data validation

### Key Features

- ✅ **Hot-Reload**: Plugins can be updated without restarting the server
- ✅ **Multi-Language**: Support for both Go and Python plugins
- ✅ **Type Safety**: Strong typing and validation for Go plugins
- ✅ **Configuration**: Per-plugin configuration management
- ✅ **Monitoring**: Plugin health and performance monitoring
- ✅ **API Integration**: RESTful API for plugin management

## 📁 Plugin Directory Structure

```
plugins/
├── go_plugins/
│   ├── example_automation.go
│   └── custom_validator.go
├── python_plugins/
│   ├── example_python_plugin.py
│   └── integration_plugin.py
└── compiled/
    ├── example_automation.so
    └── custom_validator.so
```

## 🔧 Plugin Development

### Go Plugin Development

#### Basic Structure

```go
package main

import (
    "time"
)

// YourPlugin implements the PluginInterface
type YourPlugin struct {
    info PluginInfo
}

// Plugin is the main plugin instance
var Plugin = &YourPlugin{}

func (p *YourPlugin) GetInfo() PluginInfo {
    return p.info
}

func (p *YourPlugin) Initialize(config map[string]interface{}) error {
    p.info = PluginInfo{
        Name:        "your_plugin",
        Type:        "automation",
        Version:     "1.0.0",
        Description: "Your plugin description",
        Author:      "Your Name",
        Status:      "loaded",
        LoadedAt:    time.Now(),
        Config:      config,
    }
    return nil
}

func (p *YourPlugin) Execute(params map[string]interface{}) (interface{}, error) {
    // Your plugin logic here
    result := map[string]interface{}{
        "message": "Plugin executed successfully",
        "params":  params,
    }
    return result, nil
}

func (p *YourPlugin) Cleanup() error {
    // Cleanup logic here
    return nil
}
```

#### Compilation

```bash
# Compile Go plugin to shared library
go build -buildmode=plugin -o plugins/compiled/your_plugin.so plugins/go_plugins/your_plugin.go
```

### Python Plugin Development

#### Basic Structure

```python
#!/usr/bin/env python3

import json
import sys
from datetime import datetime
from typing import Dict, Any

class YourPythonPlugin:
    def __init__(self):
        self.name = "your_python_plugin"
        self.version = "1.0.0"
        self.description = "Your Python plugin description"
        self.author = "Your Name"
        self.plugin_type = "automation"
        self.status = "loaded"
        self.config = {}
        self.loaded_at = datetime.now()
    
    def get_info(self) -> Dict[str, Any]:
        return {
            "name": self.name,
            "type": self.plugin_type,
            "version": self.version,
            "description": self.description,
            "author": self.author,
            "status": self.status,
            "loaded_at": self.loaded_at.isoformat(),
            "config": self.config
        }
    
    def initialize(self, config: Dict[str, Any]) -> None:
        self.config = config
        self.status = "loaded"
        self.loaded_at = datetime.now()
    
    def execute(self, params: Dict[str, Any]) -> Dict[str, Any]:
        # Your plugin logic here
        return {
            "message": "Python plugin executed successfully",
            "params": params,
            "timestamp": datetime.now().isoformat()
        }
    
    def cleanup(self) -> None:
        self.status = "unloaded"

def main():
    if len(sys.argv) < 2:
        print("Usage: your_plugin.py <command> [args...]")
        sys.exit(1)
    
    command = sys.argv[1]
    plugin = YourPythonPlugin()
    
    if command == "info":
        print(json.dumps(plugin.get_info(), indent=2))
    elif command == "execute":
        if len(sys.argv) < 3:
            print("Usage: your_plugin.py execute <json_params>")
            sys.exit(1)
        params = json.loads(sys.argv[2])
        result = plugin.execute(params)
        print(json.dumps(result, indent=2))
    elif command == "cleanup":
        plugin.cleanup()
    else:
        print(f"Unknown command: {command}")
        sys.exit(1)

if __name__ == "__main__":
    main()
```

## 🚀 Plugin Management API

### List All Plugins

```bash
curl -X GET http://localhost:8080/plugins \
  -H "Authorization: Bearer YOUR_API_KEY"
```

Response:
```json
{
  "success": true,
  "plugins": {
    "example_automation": {
      "name": "example_automation",
      "type": "automation",
      "version": "1.0.0",
      "description": "Example automation plugin",
      "author": "SecAuto Team",
      "status": "loaded",
      "loaded_at": "2024-01-15T10:30:00Z"
    }
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Get Plugin Information

```bash
curl -X GET http://localhost:8080/plugins/example_automation \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Execute Plugin

```bash
curl -X POST http://localhost:8080/plugins/example_automation \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "param1": "value1",
    "param2": "value2"
  }'
```

Response:
```json
{
  "success": true,
  "result": {
    "message": "Example automation executed successfully",
    "timestamp": "2024-01-15T10:30:00Z",
    "parameters": {
      "param1": "value1",
      "param2": "value2"
    },
    "plugin_name": "example_automation"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## ⚙️ Configuration

### Plugin Configuration in config.yaml

```yaml
plugins:
  example_automation:
    enabled: true
    config:
      api_key: "your_api_key"
      endpoint: "https://api.example.com"
      timeout: 30
  
  example_python_plugin:
    enabled: true
    config:
      python_path: "/usr/bin/python3"
      working_dir: "/tmp/plugins"
      log_level: "info"
```

## 🔄 Hot-Reload System

The plugin system includes a file watcher that automatically detects changes to plugin files and reloads them without restarting the server.

### How It Works

1. **File Monitoring**: The system watches the `plugins/` directory for file changes
2. **Change Detection**: When a plugin file is modified, the system detects the change
3. **Graceful Reload**: The old plugin is unloaded and the new version is loaded
4. **Status Updates**: Plugin status is updated to reflect the reload

### Supported File Types

- **Go Source Files**: `.go` files (compiled on-the-fly)
- **Compiled Go Plugins**: `.so` files
- **Python Scripts**: `.py` files

## 📊 Plugin Monitoring

### Plugin Health Checks

The system automatically monitors plugin health and provides metrics:

- Plugin load/unload events
- Execution success/failure rates
- Performance metrics
- Error tracking

### Logging

Plugin events are logged with structured logging:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "component": "plugin_manager",
  "message": "Plugin loaded successfully",
  "plugin_name": "example_automation",
  "plugin_path": "plugins/example_automation.so"
}
```

## 🛡️ Security Considerations

### Plugin Isolation

- Plugins run in isolated environments
- Resource limits are enforced
- File system access is restricted
- Network access is controlled

### Validation

- Plugin metadata is validated
- Configuration is sanitized
- Input parameters are validated
- Output is sanitized

### Best Practices

1. **Validate Inputs**: Always validate and sanitize plugin inputs
2. **Handle Errors**: Implement proper error handling
3. **Resource Management**: Clean up resources in the cleanup method
4. **Logging**: Use structured logging for debugging
5. **Testing**: Test plugins thoroughly before deployment

## 🔧 Troubleshooting

### Common Issues

1. **Plugin Not Loading**
   - Check file permissions
   - Verify plugin implements required interface
   - Check logs for error messages

2. **Hot-Reload Not Working**
   - Ensure file watcher has proper permissions
   - Check if file system supports inotify
   - Verify plugin file is in correct directory

3. **Plugin Execution Fails**
   - Check plugin logs
   - Verify plugin configuration
   - Test plugin standalone

### Debugging

Enable debug logging:

```yaml
logging:
  level: "debug"
  destination: "stdout"
```

### Plugin Testing

Test plugins standalone before loading:

```bash
# Test Go plugin
go run plugins/go_plugins/example_automation.go info

# Test Python plugin
python3 plugins/python_plugins/example_python_plugin.py info
```

## 📈 Performance Considerations

### Optimization Tips

1. **Lazy Loading**: Plugins are loaded on-demand
2. **Connection Pooling**: Reuse connections where possible
3. **Caching**: Cache frequently accessed data
4. **Async Operations**: Use goroutines for long-running operations

### Resource Limits

- Memory usage per plugin
- CPU time limits
- File descriptor limits
- Network connection limits

## 🔮 Future Enhancements

### Planned Features

- **Plugin Marketplace**: Centralized plugin repository
- **Version Management**: Plugin versioning and updates
- **Dependency Management**: Plugin dependencies
- **Advanced Monitoring**: Real-time plugin metrics
- **Plugin Templates**: Code generation for new plugins
- **Multi-Language Support**: Additional programming languages
- **Plugin Chaining**: Plugin composition and pipelines

### Plugin Ecosystem

The plugin system is designed to support a rich ecosystem of:

- **Security Tools**: Integration with security tools and APIs
- **Automation Scripts**: Custom automation workflows
- **Data Processors**: Data transformation and analysis
- **Validators**: Custom validation rules
- **Integrations**: Third-party system integrations

## 📚 Examples

See the `plugins/` directory for complete examples:

- `example_automation.go`: Go automation plugin
- `example_python_plugin.py`: Python automation plugin

## 🤝 Contributing

When contributing plugins:

1. Follow the plugin interface specifications
2. Include comprehensive documentation
3. Add proper error handling
4. Include unit tests
5. Follow security best practices
6. Add configuration examples

## 📄 License

This plugin system is part of SecAuto and follows the same license terms. 