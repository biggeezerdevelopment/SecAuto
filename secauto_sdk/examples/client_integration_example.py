#!/usr/bin/env python3
"""
SecAuto SDK - Client Integration Management Example

This script demonstrates comprehensive client and integration management
using the SecAuto Python SDK, including client-specific integration configurations.
"""

import os
import json
import time
from secauto_sdk import SecAutoClient
from secauto_sdk.exceptions import SecAutoError, SecAutoNotFoundError


def main():
    """Main example function demonstrating client integration management."""
    
    # Initialize client
    url = os.getenv('SECAUTO_URL', 'http://localhost:9090')
    api_key = os.getenv('SECAUTO_API_KEY', 'secauto-api-key-2024-07-14')
    
    print("🚀 SecAuto SDK - Client Integration Management Example")
    print(f"📡 Connecting to: {url}")
    print("=" * 60)
    
    try:
        client = SecAutoClient(url, api_key)
        
        # Test connection
        if not client.test_connection():
            print("❌ Failed to connect to SecAuto server")
            return
        
        print("✅ Connected to SecAuto server\n")
        
        # ================================================================
        # 1. Client Management
        # ================================================================
        print("1️⃣ CLIENT MANAGEMENT")
        print("-" * 30)
        
        # List existing clients
        print("📋 Listing existing clients...")
        existing_clients = client.list_clients()
        print(f"   Found {len(existing_clients)} existing clients:")
        for client_obj in existing_clients[:3]:  # Show first 3
            print(f"   - {client_obj.name} (ID: {client_obj.id}) - Enabled: {client_obj.enabled}")
        
        # Create a new test client
        test_client_name = f'sdk_test_client_{int(time.time())}'
        print(f"\n🆕 Creating new client: {test_client_name}")
        
        new_client = client.create_client(
            name=test_client_name,
            description='Test client created by SDK for integration demo',
            enabled=True,
            metadata={
                'created_by': 'sdk_example',
                'purpose': 'integration_testing',
                'department': 'security',
                'region': 'us-east'
            }
        )
        
        print(f"   ✅ Client created successfully!")
        print(f"   - ID: {new_client.id}")
        print(f"   - Name: {new_client.name}")
        print(f"   - Enabled: {new_client.enabled}")
        print(f"   - Metadata: {json.dumps(new_client.metadata, indent=6)}")
        
        # ================================================================
        # 2. Integration Management
        # ================================================================
        print(f"\n2️⃣ INTEGRATION MANAGEMENT")
        print("-" * 30)
        
        # List available integrations
        print("🔗 Listing available integrations...")
        integrations = client.list_integrations()
        print(f"   Found {len(integrations)} integrations:")
        
        available_integrations = []
        for integration in integrations[:5]:  # Show first 5
            print(f"   - {integration.name} ({integration.type}) - Enabled: {integration.enabled}")
            if integration.enabled:
                available_integrations.append(integration)
        
        if not available_integrations:
            print("   ⚠️ No enabled integrations found. Creating mock integration data for demo.")
            # For demo purposes, we'll assume some common integrations exist
            mock_integrations = ['virustotal', 'abuseipdb', 'shodan', 'urlvoid']
        else:
            mock_integrations = [integ.name for integ in available_integrations[:4]]
        
        # ================================================================
        # 3. Client-Specific Integration Configuration
        # ================================================================
        print(f"\n3️⃣ CLIENT INTEGRATION CONFIGURATION")
        print("-" * 40)
        
        client_id = new_client.id
        configured_integrations = []
        
        for integration_name in mock_integrations:
            print(f"\n🔧 Configuring integration: {integration_name}")
            
            # Create client-specific configuration
            if integration_name == 'virustotal':
                config = {
                    'api_key': 'vt_api_key_for_client',
                    'rate_limit': 100,
                    'timeout': 30,
                    'enable_file_scan': True,
                    'enable_url_scan': True
                }
            elif integration_name == 'abuseipdb':
                config = {
                    'api_key': 'abuse_api_key_for_client',
                    'confidence_threshold': 75,
                    'max_age_days': 30,
                    'verbose': True
                }
            elif integration_name == 'shodan':
                config = {
                    'api_key': 'shodan_api_key_for_client',
                    'search_filters': ['port:22', 'port:80', 'port:443'],
                    'max_results': 100
                }
            else:
                config = {
                    'api_key': f'{integration_name}_api_key',
                    'enabled_features': ['scan', 'lookup'],
                    'timeout': 30
                }
            
            try:
                # Set integration configuration for the client
                config_response = client.set_client_integration_config(
                    client_id=client_id,
                    integration_name=integration_name,
                    config=config,
                    enabled=True
                )
                
                if config_response.get('success'):
                    print(f"   ✅ {integration_name} configured successfully")
                    configured_integrations.append(integration_name)
                    print(f"      Config keys: {list(config.keys())}")
                else:
                    print(f"   ❌ Failed to configure {integration_name}: {config_response.get('message')}")
                    
            except SecAutoError as e:
                print(f"   ⚠️ Error configuring {integration_name}: {e}")
        
        # ================================================================
        # 4. Retrieve and Verify Configurations
        # ================================================================
        print(f"\n4️⃣ CONFIGURATION VERIFICATION")
        print("-" * 35)
        
        # List client integrations
        print(f"📋 Listing integrations for client {client_id}...")
        try:
            client_integrations = client.list_client_integrations(client_id)
            print(f"   Found {len(client_integrations)} configured integrations")
            
            for integration in client_integrations:
                integration_name = integration.get('name', 'unknown')
                enabled = integration.get('enabled', False)
                print(f"   - {integration_name}: {'Enabled' if enabled else 'Disabled'}")
        except SecAutoError as e:
            print(f"   ⚠️ Error listing client integrations: {e}")
        
        # Verify individual configurations
        for integration_name in configured_integrations:
            try:
                print(f"\n🔍 Verifying {integration_name} configuration...")
                config = client.get_client_integration_config(client_id, integration_name)
                
                if config:
                    print(f"   ✅ Configuration retrieved successfully")
                    print(f"   Keys: {list(config.keys()) if isinstance(config, dict) else 'N/A'}")
                else:
                    print(f"   ⚠️ No configuration found")
                    
            except SecAutoNotFoundError:
                print(f"   ⚠️ Configuration not found for {integration_name}")
            except SecAutoError as e:
                print(f"   ❌ Error retrieving configuration: {e}")
        
        # ================================================================
        # 5. Execute Integration Functions
        # ================================================================
        print(f"\n5️⃣ INTEGRATION EXECUTION")
        print("-" * 30)
        
        # Try executing integration functions
        for integration_name in configured_integrations[:2]:  # Test first 2
            print(f"\n⚡ Executing {integration_name} integration...")
            
            # Define function and parameters based on integration type
            if integration_name == 'virustotal':
                function = 'scan_url'
                params = {'url': 'https://example.com'}
            elif integration_name == 'abuseipdb':
                function = 'check_ip'
                params = {'ip': '192.168.1.1'}
            elif integration_name == 'shodan':
                function = 'host_info'
                params = {'ip': '8.8.8.8'}
            else:
                function = 'test_function'
                params = {'test_param': 'test_value'}
            
            try:
                result = client.execute_client_integration(
                    client_id=client_id,
                    integration_name=integration_name,
                    function=function,
                    params=params
                )
                
                if result.get('success'):
                    print(f"   ✅ Execution successful")
                    print(f"   Function: {function}")
                    print(f"   Parameters: {json.dumps(params, indent=6)}")
                    if 'result' in result:
                        print(f"   Result type: {type(result['result'])}")
                else:
                    print(f"   ❌ Execution failed: {result.get('message', 'Unknown error')}")
                    
            except SecAutoError as e:
                print(f"   ⚠️ Execution error: {e}")
        
        # ================================================================
        # 6. Update Configuration
        # ================================================================
        print(f"\n6️⃣ CONFIGURATION UPDATES")
        print("-" * 30)
        
        if configured_integrations:
            integration_to_update = configured_integrations[0]
            print(f"🔄 Updating configuration for {integration_to_update}...")
            
            try:
                # Update configuration
                updated_config = {'timeout': 60, 'verbose': True}
                update_response = client.update_client_integration_config(
                    client_id=client_id,
                    integration_name=integration_to_update,
                    config=updated_config
                )
                
                if update_response.get('success'):
                    print(f"   ✅ Configuration updated successfully")
                    print(f"   Updated fields: {list(updated_config.keys())}")
                else:
                    print(f"   ❌ Update failed: {update_response.get('message')}")
                    
            except SecAutoError as e:
                print(f"   ⚠️ Update error: {e}")
        
        # ================================================================
        # 7. Performance and Statistics
        # ================================================================
        print(f"\n7️⃣ PERFORMANCE & STATISTICS")
        print("-" * 35)
        
        # Get various statistics
        print("📊 Gathering system statistics...")
        
        try:
            # Job statistics
            job_stats = client.get_job_stats()
            print(f"   Jobs - Total: {job_stats.total_jobs}, Running: {job_stats.running}")
            
            # Cache statistics
            cache_stats = client.get_cache_stats()
            hit_rate = 0
            if (cache_stats.context_hits + cache_stats.context_misses) > 0:
                hit_rate = cache_stats.context_hits / (cache_stats.context_hits + cache_stats.context_misses) * 100
            print(f"   Cache - Hit rate: {hit_rate:.1f}%, Size: {cache_stats.total_size}")
            
            # API key statistics
            api_key_stats = client.get_api_key_stats()
            print(f"   API Keys - Total: {api_key_stats.get('total', 0)}, Active: {api_key_stats.get('active', 0)}")
            
        except SecAutoError as e:
            print(f"   ⚠️ Error getting statistics: {e}")
        
        # ================================================================
        # 8. Cleanup (Optional)
        # ================================================================
        print(f"\n8️⃣ CLEANUP")
        print("-" * 15)
        
        cleanup = input("🗑️ Do you want to clean up the test client? (y/N): ").lower().strip()
        
        if cleanup == 'y':
            print(f"🧹 Cleaning up test client {client_id}...")
            
            # Delete integration configurations
            for integration_name in configured_integrations:
                try:
                    delete_response = client.delete_client_integration_config(client_id, integration_name)
                    if delete_response.get('success'):
                        print(f"   ✅ Deleted {integration_name} configuration")
                    else:
                        print(f"   ⚠️ Failed to delete {integration_name} configuration")
                except SecAutoError as e:
                    print(f"   ⚠️ Error deleting {integration_name} config: {e}")
            
            # Delete the client
            try:
                delete_response = client.delete_client(client_id)
                if delete_response.get('success'):
                    print(f"   ✅ Test client deleted successfully")
                else:
                    print(f"   ❌ Failed to delete test client: {delete_response.get('message')}")
            except SecAutoError as e:
                print(f"   ❌ Error deleting client: {e}")
        else:
            print(f"📝 Test client preserved: {client_id}")
            print(f"   You can manage it through the SecAuto interface or delete it later")
        
        print("\n" + "=" * 60)
        print("🎉 Client Integration Management Example Completed!")
        print("✅ All operations demonstrated successfully")
        
        # Summary
        print(f"\n📋 SUMMARY:")
        print(f"   - Created client: {new_client.name} ({new_client.id})")
        print(f"   - Configured {len(configured_integrations)} integrations")
        print(f"   - Demonstrated configuration retrieval and updates")
        print(f"   - Showed integration execution capabilities")
        
    except SecAutoError as e:
        print(f"\n❌ SecAuto SDK Error: {e}")
    except Exception as e:
        print(f"\n💥 Unexpected Error: {e}")
        import traceback
        print(f"Traceback: {traceback.format_exc()}")


if __name__ == '__main__':
    main()
