#!/usr/bin/env python3
"""
SecAuto SDK - Async Playbook Execution Example

This script demonstrates asynchronous playbook execution and job monitoring
using the SecAuto Python SDK.
"""

import os
import time
import json
from secauto_sdk import SecAutoClient
from secauto_sdk.exceptions import SecAutoError


def create_sample_playbook():
    """Create a sample playbook for testing."""
    return {
        "name": "sample_investigation_playbook",
        "description": "Sample investigation workflow",
        "version": "1.0",
        "steps": [
            {
                "name": "initial_analysis",
                "action": "analyze",
                "params": {
                    "target": "{{context.target_ip}}",
                    "analysis_type": "initial"
                },
                "timeout": 30
            },
            {
                "name": "reputation_check",
                "action": "reputation_lookup",
                "params": {
                    "ip": "{{context.target_ip}}",
                    "sources": ["virustotal", "abuseipdb"]
                },
                "timeout": 45
            },
            {
                "name": "generate_report",
                "action": "create_report",
                "params": {
                    "template": "investigation_summary",
                    "data": "{{steps.reputation_check.result}}"
                },
                "timeout": 15
            }
        ]
    }


def monitor_job_progress(client, job_id, max_wait_time=300):
    """
    Monitor job progress until completion or timeout.
    
    Args:
        client: SecAuto client instance
        job_id: ID of the job to monitor
        max_wait_time: Maximum time to wait in seconds
        
    Returns:
        Final job object
    """
    start_time = time.time()
    print(f"📊 Monitoring job {job_id}...")
    
    while time.time() - start_time < max_wait_time:
        try:
            job = client.get_job(job_id)
            elapsed = time.time() - start_time
            
            print(f"   [{elapsed:.1f}s] Status: {job.status}")
            
            if job.status in ['completed', 'failed', 'cancelled']:
                print(f"🏁 Job finished with status: {job.status}")
                if job.error:
                    print(f"   Error: {job.error}")
                if job.results:
                    print(f"   Results: {json.dumps(job.results, indent=2)}")
                return job
            
            time.sleep(5)  # Wait 5 seconds before checking again
            
        except SecAutoError as e:
            print(f"   Error checking job status: {e}")
            break
    
    print(f"⏰ Job monitoring timed out after {max_wait_time} seconds")
    return None


def main():
    """Main example function."""
    
    # Initialize client
    url = os.getenv('SECAUTO_URL', 'http://localhost:9090')
    api_key = os.getenv('SECAUTO_API_KEY', 'secauto-api-key-2024-07-14')
    
    print("🚀 SecAuto SDK - Async Playbook Execution Example")
    print(f"📡 Connecting to: {url}")
    
    try:
        client = SecAutoClient(url, api_key)
        
        # Test connection
        if not client.test_connection():
            print("❌ Failed to connect to SecAuto server")
            return
        
        print("✅ Connected to SecAuto server")
        
        # Create sample playbook
        playbook = create_sample_playbook()
        print(f"\n📋 Created sample playbook: {playbook['name']}")
        
        # Define execution context
        context = {
            'target_ip': '192.168.1.100',
            'incident_id': 'INC-2024-001',
            'analyst': 'security-team',
            'priority': 'high',
            'investigation_type': 'malware_analysis'
        }
        
        print(f"🔧 Execution context: {json.dumps(context, indent=2)}")
        
        # Execute playbook asynchronously
        print("\n⚡ Starting async playbook execution...")
        
        response = client.execute_playbook_async(
            playbook=playbook,
            context=context
        )
        
        if not response.success:
            print(f"❌ Failed to start playbook: {response.message}")
            return
        
        job_id = response.job_id
        print(f"✅ Playbook started successfully!")
        print(f"   Job ID: {job_id}")
        print(f"   Message: {response.message}")
        
        # Monitor job progress
        final_job = monitor_job_progress(client, job_id)
        
        if final_job:
            print(f"\n📋 Final Job Details:")
            print(f"   ID: {final_job.id}")
            print(f"   Status: {final_job.status}")
            print(f"   Created: {final_job.created_at}")
            print(f"   Started: {final_job.started_at}")
            print(f"   Completed: {final_job.completed_at}")
            
            if final_job.completed_at and final_job.started_at:
                # Calculate duration (this would need proper datetime parsing in real implementation)
                print(f"   Duration: Job completed")
        
        # Demonstrate multiple async executions
        print("\n🔄 Demonstrating multiple async executions...")
        
        job_ids = []
        for i in range(3):
            context_copy = context.copy()
            context_copy['batch_id'] = f'batch-{i+1}'
            context_copy['target_ip'] = f'192.168.1.{100 + i}'
            
            response = client.execute_playbook_async(
                playbook=playbook,
                context=context_copy
            )
            
            if response.success:
                job_ids.append(response.job_id)
                print(f"   Batch {i+1}: Job {response.job_id} started")
            else:
                print(f"   Batch {i+1}: Failed - {response.message}")
        
        # Monitor all jobs
        print(f"\n📊 Monitoring {len(job_ids)} concurrent jobs...")
        
        completed_jobs = []
        max_iterations = 60  # Max 5 minutes
        iteration = 0
        
        while job_ids and iteration < max_iterations:
            iteration += 1
            time.sleep(5)
            
            for job_id in job_ids[:]:  # Copy list for safe iteration
                try:
                    job = client.get_job(job_id)
                    
                    if job.status in ['completed', 'failed', 'cancelled']:
                        completed_jobs.append(job)
                        job_ids.remove(job_id)
                        print(f"   ✅ Job {job_id}: {job.status}")
                    else:
                        print(f"   ⏳ Job {job_id}: {job.status}")
                        
                except SecAutoError as e:
                    print(f"   ❌ Error checking job {job_id}: {e}")
                    job_ids.remove(job_id)
        
        # Summary
        print(f"\n📈 Execution Summary:")
        print(f"   Completed jobs: {len(completed_jobs)}")
        print(f"   Still running: {len(job_ids)}")
        
        successful_jobs = [j for j in completed_jobs if j.status == 'completed']
        failed_jobs = [j for j in completed_jobs if j.status == 'failed']
        
        print(f"   Successful: {len(successful_jobs)}")
        print(f"   Failed: {len(failed_jobs)}")
        
        if failed_jobs:
            print(f"\n❌ Failed Jobs:")
            for job in failed_jobs:
                print(f"   - {job.id}: {job.error}")
        
        # Get overall job statistics
        print(f"\n📊 Overall Job Statistics:")
        try:
            stats = client.get_job_stats()
            print(f"   Total jobs: {stats.total_jobs}")
            print(f"   Running: {stats.running}")
            print(f"   Completed: {stats.completed}")
            print(f"   Failed: {stats.failed}")
            print(f"   Average duration: {stats.avg_duration_seconds:.2f}s")
        except SecAutoError as e:
            print(f"   Error getting stats: {e}")
        
        print("\n✅ Async playbook execution example completed!")
        
    except SecAutoError as e:
        print(f"\n❌ SDK Error: {e}")
    except Exception as e:
        print(f"\n💥 Unexpected Error: {e}")


if __name__ == '__main__':
    main()
