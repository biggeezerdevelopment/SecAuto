# SecAuto Python SDK

A comprehensive Python SDK for interacting with the SecAuto SOAR (Security Orchestration, Automation, and Response) platform. Built using `restfly` for robust REST API interactions.

## Features

- **Complete API Coverage**: All SecAuto API endpoints supported
- **Type Safety**: Full type hints and data models
- **Error Handling**: Comprehensive exception handling with detailed error information
- **Async Support**: Asynchronous playbook execution
- **File Operations**: Upload playbooks, automations, and integrations
- **Caching**: Redis-based caching operations
- **Client Management**: Multi-tenant client support
- **Real-time Monitoring**: Job tracking and cluster management

## Installation

```bash
pip install secauto-sdk
```

Or from source:

```bash
git clone <repository-url>
cd secauto-sdk
pip install -e .
```

## Quick Start

```python
from secauto_sdk import SecAutoClient

# Initialize the client
client = SecAutoClient('http://localhost:9090', api_key='your-api-key')

# Test connection
if client.test_connection():
    print("✅ Connected to SecAuto server")
else:
    print("❌ Connection failed")

# Check health
health = client.health()
print(f"Server status: {health}")
```

## Usage Examples

### Playbook Execution

```python
# Execute a playbook synchronously
response = client.execute_playbook(
    playbook_name='incident_response',
    context={
        'incident_id': 'INC-001',
        'severity': 'high',
        'source_ip': '192.168.1.100'
    }
)

if response.success:
    print(f"Playbook executed successfully: {response.result}")
else:
    print(f"Playbook failed: {response.message}")

# Execute asynchronously for long-running tasks
async_response = client.execute_playbook_async(
    playbook_name='forensic_analysis',
    context={'target_host': 'server-01'}
)

if async_response.success:
    job_id = async_response.job_id
    print(f"Job started with ID: {job_id}")
    
    # Monitor job progress
    job = client.get_job(job_id)
    print(f"Job status: {job.status}")
```

### Job Management

```python
# List running jobs
running_jobs = client.list_jobs(status='running')
print(f"Running jobs: {len(running_jobs)}")

# Get job details
job = client.get_job('job-12345')
print(f"Job {job.id}: {job.status}")
print(f"Created: {job.created_at}")
print(f"Results: {job.results}")

# Cancel a job
if job.status == 'running':
    cancel_response = client.cancel_job(job.id)
    print(f"Job cancelled: {cancel_response['success']}")

# Get job statistics
stats = client.get_job_stats()
print(f"Total jobs: {stats.total_jobs}")
print(f"Success rate: {stats.completed / stats.total_jobs * 100:.1f}%")
```

### Cache Operations

```python
# Store data in cache
client.set_cache_value('incident-001', {
    'status': 'investigating',
    'assigned_to': 'analyst-1',
    'created': '2024-01-15T10:30:00Z'
})

# Retrieve cached data
incident_data = client.get_cache_value('incident-001')
print(f"Incident status: {incident_data['status']}")

# Cache with TTL
client.set_cache_value('temp-data', {'value': 42}, ttl=300)  # 5 minutes

# Get cache statistics
cache_stats = client.get_cache_stats()
print(f"Cache hit rate: {cache_stats.context_hits / (cache_stats.context_hits + cache_stats.context_misses) * 100:.1f}%")
```

### Integration Management

```python
# List available integrations
integrations = client.list_integrations()
for integration in integrations:
    print(f"Integration: {integration.name} ({integration.type})")
    print(f"Enabled: {integration.enabled}")

# Upload a new integration
upload_response = client.upload_integration('/path/to/integration.py')
print(f"Integration uploaded: {upload_response['success']}")

# Check build status
build_status = client.get_integration_build_status('my_integration')
print(f"Build status: {build_status}")
```

### Client Management

```python
# Create a new client
client_obj = client.create_client(
    name='ACME Corp',
    description='Main corporate client',
    metadata={'department': 'security', 'region': 'us-east'}
)

# Configure integration for client
config_response = client.set_client_integration_config(
    client_id=client_obj.id,
    integration_name='virustotal',
    config={
        'api_key': 'vt-api-key',
        'rate_limit': 100
    },
    enabled=True
)

# Execute integration for client
result = client.execute_client_integration(
    client_id=client_obj.id,
    integration_name='virustotal',
    function='scan_url',
    params={'url': 'https://suspicious-site.com'}
)
```

### Automation Management

```python
# List automations
automations = client.list_automations()
for automation in automations:
    print(f"Automation: {automation.name}")
    print(f"Language: {automation.language}")
    print(f"Valid: {automation.is_valid}")

# Upload automation script
upload_response = client.upload_automation('/path/to/script.py')
print(f"Automation uploaded: {upload_response['success']}")

# Get automation metadata
metadata = client.get_automation_metadata('my_script')
print(f"Version: {metadata.version}")
print(f"Author: {metadata.author}")
print(f"Parameters: {metadata.parameters}")
```

### Schedule Management

```python
# Create a scheduled job
schedule = client.create_schedule(
    name='Daily Threat Intel Update',
    description='Updates threat intelligence feeds daily',
    cron_expression='0 2 * * *',  # Daily at 2 AM
    playbook={'name': 'update_threat_intel'},
    context={'sources': ['all']},
    enabled=True
)

# List schedules
schedules = client.list_schedules(status='enabled')
for sched in schedules:
    print(f"Schedule: {sched.name}")
    print(f"Next run: {sched.next_run}")

# Execute schedule manually
execution_response = client.execute_schedule(schedule.id)
print(f"Manual execution: {execution_response['success']}")
```

### List Operations (Redis)

```python
# Add items to a list
client.add_to_list('incident_queue', [
    {'id': 'INC-001', 'priority': 'high'},
    {'id': 'INC-002', 'priority': 'medium'}
])

# Get list items
incidents = client.get_list('incident_queue')
print(f"Incidents in queue: {len(incidents)}")

# Remove processed incidents
client.remove_from_list('incident_queue', [{'id': 'INC-001'}])
```

### Error Handling

```python
from secauto_sdk.exceptions import (
    SecAutoAPIError, SecAutoAuthenticationError, 
    SecAutoNotFoundError, SecAutoValidationError
)

try:
    job = client.get_job('non-existent-job')
except SecAutoNotFoundError as e:
    print(f"Job not found: {e.message}")
except SecAutoAuthenticationError as e:
    print(f"Authentication failed: {e.message}")
except SecAutoValidationError as e:
    print(f"Validation failed: {e.validation_errors}")
except SecAutoAPIError as e:
    print(f"API error: {e.message} (Status: {e.status_code})")
```

## Configuration

### Environment Variables

You can set default configuration using environment variables:

```bash
export SECAUTO_URL="http://localhost:9090"
export SECAUTO_API_KEY="your-api-key"
export SECAUTO_VERIFY_SSL="true"
export SECAUTO_TIMEOUT="30"
```

### Advanced Configuration

```python
# Custom configuration
client = SecAutoClient(
    url='https://secauto.company.com',
    api_key='your-api-key',
    verify_ssl=True,
    timeout=60,
    retries=5,
    backoff=2.0
)

# Custom headers
client._session.headers.update({
    'X-Custom-Header': 'value'
})
```

## API Coverage

The SDK covers all SecAuto API endpoints:

### ✅ Implemented Endpoints

- **Health & System**: `/health`, `/docs`
- **Playbooks**: `/playbook`, `/playbook/async`, `/playbook/upload`, `/playbooks`
- **Jobs**: `/jobs`, `/jobs/stats`, `/job/{id}`
- **Schedules**: `/schedules`, `/schedule/{id}`, `/schedule/execute/{id}`
- **Cache**: `/cache`, `/cache/stats`, `/cache/{key}`
- **Lists**: `/lists/{name}`, `/lists/{name}/items`
- **Integrations**: `/integrations`, `/integrations/upload`
- **Automations**: `/automations`, `/automation`, `/automation/metadata`
- **Clients**: `/clients`, `/clients/{id}`
- **Client Integrations**: `/clients/{id}/integrations/{name}/config`, `/clients/{id}/integrations/{name}/execute`
- **API Keys**: `/api-keys`, `/api-keys/stats`
- **Cluster**: `/cluster`, `/cluster/jobs`

## Data Models

The SDK includes comprehensive data models for type safety:

- `Job` - Job execution details
- `PlaybookRequest`/`PlaybookResponse` - Playbook execution
- `Client` - Client information
- `Integration` - Integration configuration
- `JobSchedule` - Scheduled job details
- `AutomationInfo`/`AutomationMetadata` - Automation details
- `APIKey`/`APIKeySummary` - API key management
- `CacheStats`/`JobStats` - Statistics

## Error Handling

Comprehensive exception hierarchy:

- `SecAutoError` - Base exception
- `SecAutoAPIError` - API-related errors
- `SecAutoConnectionError` - Connection issues
- `SecAutoAuthenticationError` - Authentication failures
- `SecAutoValidationError` - Validation errors
- `SecAutoNotFoundError` - Resource not found
- `SecAutoTimeoutError` - Request timeouts

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

- Documentation: [SecAuto Docs](https://secauto.readthedocs.io/)
- Issues: [GitHub Issues](https://github.com/secauto/secauto-sdk/issues)
- Discord: [SecAuto Community](https://discord.gg/secauto)
