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
                **incident,
                "enriched_by_plugin": True,
                "plugin_confidence": 0.85,
                "additional_indicators": [
                    "suspicious_activity_detected",
                    "threat_level_elevated"
                ]
            },
            "user": {
                **user,
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