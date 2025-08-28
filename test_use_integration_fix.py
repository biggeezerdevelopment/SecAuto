#!/usr/bin/env python3
"""
Test script to verify use_integration() fix works
"""

import sys
import os
from pathlib import Path

# Add the server directory to Python path
server_dir = Path(__file__).parent / "SoarAuto" / "server"
sys.path.insert(0, str(server_dir))

def test_use_integration():
    """Test the fixed use_integration function"""
    
    try:
        from integration_loader import IntegrationLoader
        
        print("Testing Fixed use_integration Function")
        print("=" * 50)
        
        # Create loader with the SoarAuto directory
        project_root = Path(__file__).parent / "SoarAuto"
        loader = IntegrationLoader(str(project_root))
        
        print("Testing config retrieval...")
        
        # Test the config retrieval method directly
        try:
            config, credentials = loader._get_client_integration_config(
                "postgresql", 
                "client_a4fba03c29631889"
            )
            
            print("✓ Config retrieval successful:")
            print(f"  Config keys: {list(config.keys())}")
            print(f"  Credential keys: {list(credentials.keys())}")
            print(f"  Has password in config: {'password' in config}")
            
            if 'password' in config:
                pwd = config['password']
                print(f"  Password: {pwd[:2]}*** (length: {len(pwd)})")
            
        except Exception as e:
            print(f"✗ Config retrieval failed: {e}")
            return False
            
        print("\nTesting full integration execution...")
        
        # Test the full use_integration call
        try:
            result = loader.use_integration(
                "postgresql",
                "test_connection", 
                client_id="client_a4fba03c29631889"
            )
            
            print("✓ Integration execution completed:")
            print(f"  Success: {result.get('success', 'unknown')}")
            
            if result.get('success'):
                print("  ✓ Integration executed successfully!")
            else:
                print(f"  ✗ Integration failed: {result.get('error', 'unknown error')}")
                
                # Check if it's still the password issue
                if 'no password supplied' in str(result.get('error', '')):
                    print("  → Still getting password issue - config not being passed properly")
                else:
                    print("  → Different error - progress made!")
                    
        except Exception as e:
            print(f"✗ Integration execution failed: {e}")
            return False
            
    except ImportError as e:
        print(f"Import error: {e}")
        print("Make sure you're running this from the SecAuto parent directory")
        return False
        
    return True

if __name__ == "__main__":
    if test_use_integration():
        print("\n✓ Test completed successfully")
    else:
        print("\n✗ Test failed")
        sys.exit(1)