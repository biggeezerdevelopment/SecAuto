#!/usr/bin/env python3
"""
Debug script to help diagnose integration configuration issues
"""

import requests
import json
import sys
import os

def test_integration_config(base_url="http://localhost:8080", client_id="client_a4fba03c29631889", integration="postgresql"):
    """Test and debug integration configuration retrieval"""
    
    print("Integration Configuration Debugger")
    print("=" * 60)
    
    # First, get the client's integration config
    print(f"\n1. Getting integration config for:")
    print(f"   Client ID: {client_id}")
    print(f"   Integration: {integration}")
    
    config_url = f"{base_url}/api/v1/clients/{client_id}/integrations/{integration}/config"
    
    try:
        response = requests.get(config_url)
        print(f"\n   Status: {response.status_code}")
        
        if response.status_code == 200:
            config_data = response.json()
            print("\n   Retrieved Configuration:")
            print("   " + "-" * 40)
            
            # Pretty print the config (with credentials masked)
            if isinstance(config_data, dict):
                # Check if it's wrapped in a response
                if 'config' in config_data:
                    actual_config = config_data['config']
                else:
                    actual_config = config_data
                    
                # Display config keys
                if 'config' in actual_config:
                    print(f"   Config keys: {list(actual_config['config'].keys()) if actual_config['config'] else 'None'}")
                    print("\n   Config values:")
                    for k, v in (actual_config.get('config') or {}).items():
                        print(f"     {k}: {v}")
                
                # Display credential keys (masked)
                if 'credentials' in actual_config:
                    print(f"\n   Credential keys: {list(actual_config['credentials'].keys()) if actual_config['credentials'] else 'None'}")
                    print("\n   Credential values (masked):")
                    for k, v in (actual_config.get('credentials') or {}).items():
                        if v:
                            masked = str(v)[:2] + "***" if len(str(v)) > 2 else "***"
                            print(f"     {k}: {masked}")
                        else:
                            print(f"     {k}: (empty)")
                
                # Display other metadata
                print(f"\n   Enabled: {actual_config.get('enabled', 'N/A')}")
                print(f"   Client ID: {actual_config.get('client_id', 'N/A')}")
                
            print("\n   Full response (for debugging):")
            print(json.dumps(config_data, indent=2))
        else:
            print(f"\n   ERROR: Failed to get config")
            print(f"   Response: {response.text}")
    except Exception as e:
        print(f"\n   ERROR: {e}")
        return False
    
    # Now test the integration
    print(f"\n2. Testing integration execution:")
    print("   " + "-" * 40)
    
    test_playbook = {
        "name": "debug_integration_test",
        "steps": [
            {
                "name": "test_connection",
                "type": "integration",
                "integration": integration,
                "function": "test_connection",
                "params": {}
            }
        ]
    }
    
    execute_url = f"{base_url}/api/v1/playbooks/execute"
    headers = {"Content-Type": "application/json"}
    
    try:
        print(f"   Executing test_connection for {integration}...")
        response = requests.post(
            execute_url,
            json={
                "playbook": test_playbook,
                "client_id": client_id
            },
            headers=headers
        )
        
        print(f"   Status: {response.status_code}")
        result = response.json()
        
        if response.status_code == 200:
            if result.get('success'):
                print("   SUCCESS: Integration executed successfully")
                print(f"   Result: {json.dumps(result, indent=2)}")
            else:
                print("   FAILED: Integration execution failed")
                print(f"   Error details: {json.dumps(result, indent=2)}")
                
                # Parse error for specific issues
                if 'result' in result and isinstance(result['result'], list):
                    for step_result in result['result']:
                        if 'details' in step_result and 'error' in step_result['details']:
                            error_msg = step_result['details']['error']
                            
                            # Check for common issues
                            if 'no password supplied' in error_msg:
                                print("\n   ⚠️  ISSUE DETECTED: Password not being passed to integration")
                                print("   Check that:")
                                print("   1. The password is saved in the client's integration config")
                                print("   2. The credentials are being properly decrypted")
                                print("   3. The integration script is reading the password from context['credentials']")
                            elif 'connection refused' in error_msg:
                                print("\n   ⚠️  ISSUE DETECTED: Cannot connect to database")
                                print("   Check that:")
                                print("   1. The database server is running")
                                print("   2. The host/port in config are correct")
                            elif 'FATAL:  password authentication failed' in error_msg:
                                print("\n   ⚠️  ISSUE DETECTED: Invalid password")
                                print("   Check that:")
                                print("   1. The password in the config is correct")
                                print("   2. The username has access to the database")
        else:
            print(f"   ERROR: Request failed")
            print(f"   Response: {response.text}")
            
    except Exception as e:
        print(f"\n   ERROR: {e}")
        return False
    
    # Check logs
    print(f"\n3. Check server logs for debug output:")
    print("   " + "-" * 40)
    print("   Look for these log entries in your server output:")
    print("   - 'Retrieved client integration config' - shows what config was loaded")
    print("   - 'Executing integration with context' - shows what's passed to integration")
    print("   - 'Cache miss for config' or 'Retrieved config from cache' - shows caching behavior")
    print("\n   To see debug logs, make sure your server is running with debug log level")
    
    return True

if __name__ == "__main__":
    # Parse command line arguments
    base_url = os.getenv("SECAUTO_URL", "http://localhost:8080")
    client_id = os.getenv("SECAUTO_CLIENT_ID", "client_a4fba03c29631889")
    integration = "postgresql"
    
    if len(sys.argv) > 1:
        integration = sys.argv[1]
    if len(sys.argv) > 2:
        client_id = sys.argv[2]
    if len(sys.argv) > 3:
        base_url = sys.argv[3]
    
    print(f"Usage: {sys.argv[0]} [integration] [client_id] [base_url]")
    print(f"Using: integration={integration}, client_id={client_id}, url={base_url}\n")
    
    test_integration_config(base_url, client_id, integration)