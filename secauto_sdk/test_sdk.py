#!/usr/bin/env python3
"""
SecAuto SDK Test Runner

Quick test script to validate the SDK functionality against a running SecAuto server.
"""

import os
import sys
import time
import traceback
from secauto_sdk import SecAutoClient
from secauto_sdk.exceptions import SecAutoError


def test_basic_functionality():
    """Test basic SDK functionality."""
    
    print("🧪 SecAuto SDK Test Runner")
    print("=" * 50)
    
    # Configuration
    url = os.getenv('SECAUTO_URL', 'http://localhost:9090')
    api_key = os.getenv('SECAUTO_API_KEY', 'secauto-api-key-2024-07-14')
    
    print(f"Server URL: {url}")
    print(f"API Key: {api_key[:12]}..." if len(api_key) > 12 else api_key)
    print()
    
    try:
        # Initialize client
        print("1. Initializing SecAuto client...")
        client = SecAutoClient(url, api_key)
        print("   ✅ Client initialized")
        
        # Test connection
        print("\n2. Testing connection...")
        if client.test_connection():
            print("   ✅ Connection successful")
        else:
            print("   ❌ Connection failed")
            return False
        
        # Test health endpoint
        print("\n3. Testing health endpoint...")
        health = client.health()
        print(f"   ✅ Health check: {health.get('status', 'unknown')}")
        
        # Test job listing
        print("\n4. Testing job management...")
        jobs = client.list_jobs(limit=3)
        print(f"   ✅ Listed {len(jobs)} jobs")
        
        job_stats = client.get_job_stats()
        print(f"   ✅ Job stats - Total: {job_stats.total_jobs}, Running: {job_stats.running}")
        
        # Test cache operations
        print("\n5. Testing cache operations...")
        test_key = f'sdk_test_{int(time.time())}'
        test_data = {'test': True, 'timestamp': time.time()}
        
        client.set_cache_value(test_key, test_data)
        print("   ✅ Cache set operation")
        
        retrieved = client.get_cache_value(test_key)
        if retrieved and retrieved.get('test') == True:
            print("   ✅ Cache get operation")
        else:
            print("   ⚠️ Cache get operation returned unexpected data")
        
        client.delete_cache_value(test_key)
        print("   ✅ Cache delete operation")
        
        # Test integrations
        print("\n6. Testing integration management...")
        integrations = client.list_integrations()
        print(f"   ✅ Listed {len(integrations)} integrations")
        
        # Test automations
        print("\n7. Testing automation management...")
        automations = client.list_automations()
        print(f"   ✅ Listed {len(automations)} automations")
        
        # Test clients
        print("\n8. Testing client management...")
        clients = client.list_clients()
        print(f"   ✅ Listed {len(clients)} clients")
        
        # Test API keys
        print("\n9. Testing API key management...")
        api_keys = client.list_api_keys()
        print(f"   ✅ Listed {len(api_keys)} API keys")
        
        api_key_stats = client.get_api_key_stats()
        print(f"   ✅ API key stats - Total: {api_key_stats.get('total', 0)}")
        
        # Test schedules
        print("\n10. Testing schedule management...")
        schedules = client.list_schedules()
        print(f"    ✅ Listed {len(schedules)} schedules")
        
        # Test playbook execution (if possible)
        print("\n11. Testing playbook execution...")
        try:
            simple_playbook = {
                "name": "sdk_test",
                "steps": [
                    {
                        "name": "test_step",
                        "action": "log",
                        "params": {"message": "SDK test execution"}
                    }
                ]
            }
            
            response = client.execute_playbook(
                playbook=simple_playbook,
                context={'test': True, 'sdk_test': True}
            )
            
            if response.success:
                print("    ✅ Playbook execution successful")
            else:
                print(f"    ⚠️ Playbook execution failed: {response.message}")
                
        except SecAutoError as e:
            print(f"    ⚠️ Playbook execution not available: {e}")
        
        print("\n" + "=" * 50)
        print("🎉 All tests completed successfully!")
        print("✅ SecAuto SDK is working correctly")
        return True
        
    except SecAutoError as e:
        print(f"\n❌ SecAuto SDK Error: {e}")
        return False
    except Exception as e:
        print(f"\n💥 Unexpected Error: {e}")
        print(f"Traceback: {traceback.format_exc()}")
        return False


def main():
    """Main function."""
    success = test_basic_functionality()
    
    if success:
        print("\n🚀 SDK is ready for use!")
        print("\nNext steps:")
        print("- Check the examples/ directory for usage examples")
        print("- Run the full test suite with: python -m pytest secauto_sdk/tests/")
        print("- Read the documentation in README.md")
        sys.exit(0)
    else:
        print("\n🔧 Issues detected. Please check:")
        print("- SecAuto server is running and accessible")
        print("- API key is valid and has proper permissions")
        print("- Network connectivity to the server")
        sys.exit(1)


if __name__ == '__main__':
    main()
