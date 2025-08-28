#!/usr/bin/env python3
"""
Script to check what password is actually stored in the integration config
"""

import requests
import json
import sys

def check_password_storage(base_url="http://localhost:8080", client_id="client_a4fba03c29631889", integration="postgresql"):
    """Check what password is stored in the config"""
    
    print("PostgreSQL Password Storage Checker")
    print("=" * 60)
    
    config_url = f"{base_url}/api/v1/clients/{client_id}/integrations/{integration}/config"
    
    try:
        response = requests.get(config_url)
        print(f"Status: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            
            # Navigate through the response structure
            if 'config' in data:
                config = data['config']
            else:
                config = data
                
            print("\nConfiguration Analysis:")
            print("-" * 40)
            
            # Check config section
            if 'config' in config:
                cfg = config['config']
                print(f"Config keys: {list(cfg.keys())}")
                
                # Check for password in config
                if 'password' in cfg:
                    pwd = cfg['password']
                    if pwd is None or pwd == "":
                        print("❌ PASSWORD IS EMPTY OR NULL IN CONFIG!")
                    else:
                        print(f"✓ Password exists in config: {pwd[:2]}*** (length: {len(pwd)})")
                else:
                    print("❌ No 'password' key in config section")
                    
                # Show all config values
                print("\nAll config values:")
                for k, v in cfg.items():
                    if k == 'password':
                        if v:
                            print(f"  {k}: {v[:2]}*** (length: {len(v)})")
                        else:
                            print(f"  {k}: (empty/null)")
                    else:
                        print(f"  {k}: {v}")
            
            # Check credentials section
            if 'credentials' in config:
                creds = config['credentials']
                print(f"\nCredential keys: {list(creds.keys())}")
                
                if 'password' in creds:
                    pwd = creds['password']
                    if pwd is None or pwd == "":
                        print("❌ PASSWORD IS EMPTY OR NULL IN CREDENTIALS!")
                    else:
                        print(f"✓ Password exists in credentials: {pwd[:2]}*** (length: {len(pwd)})")
                else:
                    print("  No 'password' key in credentials section")
            else:
                print("\nNo credentials section")
                
            # Check enabled status
            if 'enabled' in config:
                print(f"\nIntegration enabled: {config['enabled']}")
                
            print("\n" + "=" * 60)
            print("DIAGNOSIS:")
            
            # Determine the issue
            has_pwd_in_config = 'config' in config and 'password' in config['config'] and config['config']['password']
            has_pwd_in_creds = 'credentials' in config and 'password' in config['credentials'] and config['credentials']['password']
            
            if not has_pwd_in_config and not has_pwd_in_creds:
                print("❌ PROBLEM: No password stored in either config or credentials!")
                print("   SOLUTION: Save the PostgreSQL password in the integration config")
            elif has_pwd_in_creds and not has_pwd_in_config:
                print("⚠️  Password is in credentials but integration expects it in config")
                print("   SOLUTION: Either fix the integration script or move password to config")
            elif has_pwd_in_config:
                print("✓ Password is stored in config section where integration expects it")
                print("   If still failing, check that the password value is correct")
            
            # Raw output for debugging
            print("\n" + "=" * 60)
            print("RAW RESPONSE (for debugging):")
            print(json.dumps(data, indent=2))
            
        else:
            print(f"Failed to retrieve config: {response.text}")
            
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    base_url = "http://localhost:8080"
    client_id = "client_a4fba03c29631889"
    integration = "postgresql"
    
    if len(sys.argv) > 1:
        client_id = sys.argv[1]
    if len(sys.argv) > 2:
        base_url = sys.argv[2]
        
    print(f"Checking client: {client_id}")
    print(f"Server: {base_url}")
    print()
    
    check_password_storage(base_url, client_id, integration)