#!/usr/bin/env python3
"""
Client-Aware Integration Demo for SecAuto

This automation demonstrates how Python scripts can automatically access
client-specific integration configurations, with fallback to global configs.

This script will:
1. Detect the current client context
2. Load client-specific integration config (or global fallback)
3. Use the configuration to perform operations
4. Return results with client context information

Usage in playbooks:
    {"run": "client_aware_demo", "integration_name": "test_client_integration"}
"""

import json
import sys
import os
from typing import Dict, Any

# Add server directory to path to import SoarBaseAPI
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'server'))

try:
    from SoarBaseAPI import load_context, return_context, get_client_context, get_client_integration_config
except ImportError:
    # Fallback for when sitecustomize.py loads the functions globally
    def load_context():
        try:
            # Try environment variable first (more reliable)
            env_context = os.environ.get('SECAUTO_CONTEXT')
            if env_context:
                return json.loads(env_context)
            
            # Try to read JSON input from stdin
            input_data = None
            try:
                # Check if there's data in stdin
                if not sys.stdin.isatty():
                    input_data = json.load(sys.stdin)
            except json.JSONDecodeError:
                # If stdin is not valid JSON, continue without input
                pass
            except Exception:
                # If any other error reading stdin, continue without input
                pass
            return input_data if input_data else {}
        except Exception as e:
            return {}
    
    def return_context(data):
        print(json.dumps(data, indent=2))
    
    def get_client_context():
        # Check environment variable for client-specific variable
        client_id = os.environ.get('SECAUTO_CLIENT_ID')
        if client_id:
            return client_id
        
        # Check context from environment variable
        try:
            env_context = os.environ.get('SECAUTO_CONTEXT')
            if env_context:
                context = json.loads(env_context)
                if context and isinstance(context, dict):
                    client_id = context.get('client_id')
                    if client_id:
                        return client_id
        except:
            pass
        
        # Check global context for client information  
        try:
            context = load_context()
            if context and isinstance(context, dict):
                return context.get('client_id')
        except:
            pass
        
        return None
    
    def get_client_integration_config(integration_name):
        # This is a fallback - in real usage, the SoarBaseAPI functions should be available
        return None

def main():
    """
    Main function demonstrating client-aware configuration access
    """
    # Load the execution context
    context = load_context()
    if not context:
        context = {}
    
    # Get the integration name from context
    integration_name = context.get('integration_name', 'test_client_integration')
    
    # Step 1: Detect client context
    client_id = get_client_context()
    
    # Debug information
    debug_info = {
        "env_secauto_context": os.environ.get('SECAUTO_CONTEXT'),
        "env_secauto_client_id": os.environ.get('SECAUTO_CLIENT_ID'),
        "loaded_context": context,
        "detected_client_id": client_id
    }
    
    # Step 2: Load client-specific configuration
    config = get_client_integration_config(integration_name)
    
    # Step 3: Demonstrate usage
    result = {
        "client_aware_demo": {
            "success": True,
            "debug_info": debug_info,
            "execution_info": {
                "client_id": client_id,
                "integration_name": integration_name,
                "config_source": "client-specific" if client_id else "global",
                "config_loaded": config is not None
            },
            "configuration": config if config else {},
            "example_usage": {
                "description": "This shows how integrations can access client-specific configs",
                "benefits": [
                    "Multi-tenant isolation",
                    "Client-specific credentials",
                    "Per-client settings",
                    "Automatic fallback to global config"
                ]
            }
        }
    }
    
    # If we have configuration, demonstrate using it
    if config:
        result["client_aware_demo"]["config_details"] = {
            "type": config.get("type"),
            "enabled": config.get("enabled"),
            "has_credentials": bool(config.get("credentials")),
            "config_keys": list(config.get("config", {}).keys()) if config.get("config") else [],
            "client_specific": config.get("client_id") is not None
        }
        
        # Example: Use API URL from config
        if config.get("config", {}).get("api_url"):
            api_url = config["config"]["api_url"]
            result["client_aware_demo"]["example_api_call"] = {
                "would_call": api_url,
                "with_timeout": config.get("config", {}).get("timeout", 30),
                "note": "This is where you'd make the actual API call using client-specific settings"
            }
    else:
        result["client_aware_demo"]["warning"] = "No configuration found - integration may not be set up for this client"
    
    # Return the results
    return_context(result)

if __name__ == "__main__":
    main()