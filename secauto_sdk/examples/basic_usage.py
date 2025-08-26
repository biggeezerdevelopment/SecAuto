#!/usr/bin/env python3
"""
SecAuto SDK - Basic Usage Examples

This script demonstrates basic usage of the SecAuto Python SDK
including connection testing, playbook execution, and job monitoring.
"""

import os
import time
from secauto_sdk import SecAutoClient
from secauto_sdk.exceptions import SecAutoError, SecAutoNotFoundError


def main():
    """Main example function."""
    
    # Initialize client with environment variables or defaults
    url = os.getenv('SECAUTO_URL', 'http://localhost:9090')
    api_key = os.getenv('SECAUTO_API_KEY', 'secauto-api-key-2024-07-14')
    
    print("🚀 SecAuto SDK Basic Usage Examples")
    print(f"📡 Connecting to: {url}")
    
    try:
        # Initialize the client
        client = SecAutoClient(url, api_key)
        
        # Test connection
        print("\n1️⃣ Testing Connection...")
        if client.test_connection():
            print("✅ Connected successfully!")
            
            # Get server health
            health = client.health()
            print(f"   Server status: {health}")
        else:
            print("❌ Connection failed!")
            return
        
        # List available playbooks
        print("\n2️⃣ Listing Playbooks...")
        try:
            playbooks = client.list_playbooks()
            print(f"📋 Found {len(playbooks)} playbooks:")
            for playbook in playbooks[:5]:  # Show first 5
                print(f"   - {playbook.get('name', 'Unknown')}")
        except SecAutoError as e:
            print(f"   Error listing playbooks: {e}")
        
        # Execute a simple playbook
        print("\n3️⃣ Executing Sample Playbook...")
        try:
            # Create a simple test playbook
            test_playbook = {
                "name": "test_playbook",
                "steps": [
                    {
                        "name": "log_message",
                        "action": "log",
                        "params": {
                            "message": "Hello from SDK!"
                        }
                    }
                ]
            }
            
            response = client.execute_playbook(
                playbook=test_playbook,
                context={
                    'test_run': True,
                    'timestamp': time.time()
                }
            )
            
            if response.success:
                print(f"✅ Playbook executed successfully!")
                print(f"   Result: {response.result}")
                if response.job_id:
                    print(f"   Job ID: {response.job_id}")
            else:
                print(f"❌ Playbook execution failed: {response.message}")
                
        except SecAutoError as e:
            print(f"   Error executing playbook: {e}")
        
        # List and monitor jobs
        print("\n4️⃣ Job Management...")
        try:
            # Get job statistics
            job_stats = client.get_job_stats()
            print(f"📊 Job Statistics:")
            print(f"   Total jobs: {job_stats.total_jobs}")
            print(f"   Running: {job_stats.running}")
            print(f"   Completed: {job_stats.completed}")
            print(f"   Failed: {job_stats.failed}")
            print(f"   Average duration: {job_stats.avg_duration_seconds:.2f}s")
            
            # List recent jobs
            recent_jobs = client.list_jobs(limit=5)
            print(f"\n📋 Recent Jobs ({len(recent_jobs)}):")
            for job in recent_jobs:
                print(f"   {job.id}: {job.status} (Created: {job.created_at})")
                
        except SecAutoError as e:
            print(f"   Error managing jobs: {e}")
        
        # Cache operations
        print("\n5️⃣ Cache Operations...")
        try:
            # Store some test data
            test_data = {
                'example_key': 'example_value',
                'timestamp': time.time(),
                'source': 'sdk_example'
            }
            
            client.set_cache_value('sdk_test', test_data)
            print("✅ Data stored in cache")
            
            # Retrieve the data
            retrieved_data = client.get_cache_value('sdk_test')
            print(f"📥 Retrieved data: {retrieved_data}")
            
            # Get cache stats
            cache_stats = client.get_cache_stats()
            print(f"📊 Cache Statistics:")
            print(f"   Context hits: {cache_stats.context_hits}")
            print(f"   Context misses: {cache_stats.context_misses}")
            print(f"   Total size: {cache_stats.total_size}")
            
        except SecAutoError as e:
            print(f"   Error with cache operations: {e}")
        
        # Integration management
        print("\n6️⃣ Integration Management...")
        try:
            integrations = client.list_integrations()
            print(f"🔗 Found {len(integrations)} integrations:")
            for integration in integrations[:3]:  # Show first 3
                print(f"   - {integration.name} ({integration.type}) - Enabled: {integration.enabled}")
                
        except SecAutoError as e:
            print(f"   Error listing integrations: {e}")
        
        # Client management
        print("\n7️⃣ Client Management...")
        try:
            clients = client.list_clients()
            print(f"👥 Found {len(clients)} clients:")
            for client_obj in clients[:3]:  # Show first 3
                print(f"   - {client_obj.name} (ID: {client_obj.id}) - Enabled: {client_obj.enabled}")
                
        except SecAutoError as e:
            print(f"   Error listing clients: {e}")
        
        # API Key statistics
        print("\n8️⃣ API Key Statistics...")
        try:
            api_key_stats = client.get_api_key_stats()
            print(f"🔑 API Key Stats: {api_key_stats}")
            
        except SecAutoError as e:
            print(f"   Error getting API key stats: {e}")
        
        print("\n✅ Basic usage examples completed successfully!")
        
    except SecAutoError as e:
        print(f"\n❌ SDK Error: {e}")
    except Exception as e:
        print(f"\n💥 Unexpected Error: {e}")


if __name__ == '__main__':
    main()
