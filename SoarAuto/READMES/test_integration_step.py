#!/usr/bin/env python3
"""
Test script to see how integration steps are handled in playbooks
"""

import requests
import json

def test_integration_step():
    """Test playbook with integration step"""
    
    base_url = "http://localhost:9090"  # Your test server
    
    # Create a simple playbook with an integration step
    test_playbook = {
        "name": "test_integration_step",
        "steps": [
            {
                "name": "test_connection",
                "type": "integration",
                "integration": "postgresql",
                "function": "test_connection",
                "params": {}
            }
        ]
    }
    
    print("Testing Integration Step Handling")
    print("=" * 50)
    print(f"Playbook: {json.dumps(test_playbook, indent=2)}")
    print()
    
    # Execute the playbook
    execute_url = f"{base_url}/api/v1/playbooks/execute"
    headers = {"Content-Type": "application/json"}
    
    payload = {
        "playbook": test_playbook,
        "client_id": "client_a4fba03c29631889"
    }
    
    print("Sending request...")
    print(f"URL: {execute_url}")
    print(f"Payload: {json.dumps(payload, indent=2)}")
    print()
    
    try:
        response = requests.post(execute_url, json=payload, headers=headers)
        
        print(f"Response Status: {response.status_code}")
        print(f"Response Headers: {dict(response.headers)}")
        print()
        
        if response.text:
            try:
                result = response.json()
                print("Response Body (JSON):")
                print(json.dumps(result, indent=2))
            except json.JSONDecodeError:
                print("Response Body (Raw):")
                print(response.text)
        else:
            print("Empty response body")
            
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    test_integration_step()