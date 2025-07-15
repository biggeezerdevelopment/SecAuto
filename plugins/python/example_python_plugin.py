#!/usr/bin/env python3
"""
Example Python Plugin for SecAuto

This plugin demonstrates how to create a Python plugin that can be
loaded by the SecAuto plugin system.
"""

import json
import sys
import time
from datetime import datetime, timezone
from typing import Dict, Any, List, Optional


class ExamplePythonPlugin:
    """Example Python plugin for SecAuto"""
    
    def __init__(self):
        self.name = "example_python_plugin"
        self.version = "1.0.0"
        self.description = "Example Python plugin for SecAuto"
        self.author = "SecAuto Team"
        self.plugin_type = "automation"
        self.status = "loaded"
        self.config = {}
        self.loaded_at = datetime.now(timezone.utc)
    
    def get_info(self) -> Dict[str, Any]:
        """Get plugin information"""
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
        """Initialize the plugin with configuration"""
        self.config = config
        self.status = "loaded"
        self.loaded_at = datetime.now(timezone.utc)
    
    def execute(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute the plugin with given parameters"""
        # Example automation logic
        result = {
            "message": "Example Python plugin executed successfully",
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "parameters": params,
            "plugin_name": self.name,
            "python_version": sys.version,
            "config": self.config
        }
        
        # Simulate some work
        time.sleep(0.1)
        
        return result
    
    def cleanup(self) -> None:
        """Cleanup plugin resources"""
        self.status = "unloaded"
    
    def get_supported_operations(self) -> List[str]:
        """Get list of supported operations"""
        return [
            "python_example_operation",
            "python_test_operation",
            "python_demo_operation"
        ]


def main():
    """Main function for standalone plugin execution"""
    if len(sys.argv) < 2:
        print("Usage: example_python_plugin.py <command> [args...]")
        sys.exit(1)
    
    command = sys.argv[1]
    plugin = ExamplePythonPlugin()
    
    if command == "info":
        # Return plugin info
        info = plugin.get_info()
        print(json.dumps(info, indent=2))
    
    elif command == "execute":
        # Execute plugin with parameters
        if len(sys.argv) < 3:
            print("Usage: example_python_plugin.py execute <json_params>")
            sys.exit(1)
        
        try:
            params = json.loads(sys.argv[2])
            result = plugin.execute(params)
            print(json.dumps(result, indent=2))
        except json.JSONDecodeError as e:
            print(f"Invalid JSON parameters: {e}", file=sys.stderr)
            sys.exit(1)
        except Exception as e:
            print(f"Plugin execution failed: {e}", file=sys.stderr)
            sys.exit(1)
    
    elif command == "cleanup":
        # Cleanup plugin
        try:
            plugin.cleanup()
        except Exception as e:
            print(f"Plugin cleanup failed: {e}", file=sys.stderr)
            sys.exit(1)
    
    else:
        print(f"Unknown command: {command}")
        sys.exit(1)


if __name__ == "__main__":
    main() 