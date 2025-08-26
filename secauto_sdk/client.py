"""
SecAuto API Client

Main client class for interacting with the SecAuto API using restfly.
Provides methods for all SecAuto API endpoints with proper error handling
and data serialization.
"""

import json
import logging
from typing import Dict, List, Optional, Any, Union
from urllib.parse import urljoin

from restfly.session import APISession
from restfly.utils import dict_merge

from .exceptions import (
    SecAutoError, SecAutoAPIError, SecAutoConnectionError,
    SecAutoAuthenticationError, SecAutoValidationError,
    SecAutoNotFoundError, SecAutoTimeoutError
)
from .models import *


class SecAutoClient(APISession):
    """
    SecAuto API Client
    
    A comprehensive client for interacting with the SecAuto SOAR automation platform.
    Built on top of restfly for robust REST API interactions.
    
    Args:
        url: Base URL of the SecAuto server (e.g., 'http://localhost:9090')
        api_key: API key for authentication
        session: Optional custom requests session
        verify_ssl: Whether to verify SSL certificates (default: True)
        timeout: Request timeout in seconds (default: 30)
        retries: Number of retries for failed requests (default: 3)
        backoff: Backoff factor for retries (default: 1.0)
        
    Example:
        >>> client = SecAutoClient('http://localhost:9090', api_key='your-key')
        >>> health = client.health()
        >>> print(health)
    """
    
    _url = None
    _api_key = None
    _logger = None

    def __init__(self, url, api_key, session=None, verify_ssl=True, timeout=30, 
                 retries=3, backoff=1.0, **kwargs):
        self._url = url.rstrip('/')
        self._api_key = api_key
        self._logger = logging.getLogger('secauto_sdk')
        
        # Configure session defaults
        session_kwargs = {
            'url': self._url,
            'retries': retries,
            'backoff': backoff,
            'timeout': timeout,
            'verify': verify_ssl,
        }
        session_kwargs.update(kwargs)
        
        super().__init__(session=session, **session_kwargs)
        
        # Set default headers
        self._session.headers.update({
            'X-API-Key': self._api_key,
            'Content-Type': 'application/json',
            'User-Agent': 'SecAuto-Python-SDK/1.0.0'
        })

    def _error_handler(self, response):
        """Handle API errors and convert to appropriate exceptions."""
        try:
            error_data = response.json()
        except (ValueError, json.JSONDecodeError):
            error_data = {'message': response.text or 'Unknown error'}
        
        message = error_data.get('message', f'HTTP {response.status_code} error')
        
        if response.status_code == 401:
            raise SecAutoAuthenticationError(message, response.status_code, error_data)
        elif response.status_code == 404:
            raise SecAutoNotFoundError(message, response.status_code, error_data)
        elif response.status_code == 400:
            validation_errors = error_data.get('errors', [])
            if validation_errors:
                raise SecAutoValidationError(message, validation_errors)
            else:
                raise SecAutoAPIError(message, response.status_code, error_data)
        elif 400 <= response.status_code < 500:
            raise SecAutoAPIError(message, response.status_code, error_data)
        elif response.status_code >= 500:
            raise SecAutoAPIError(f"Server error: {message}", response.status_code, error_data)
        else:
            raise SecAutoAPIError(message, response.status_code, error_data)

    def _request(self, method, endpoint, **kwargs):
        """Make a request with proper error handling."""
        try:
            response = super()._request(method, endpoint, **kwargs)
            
            if not response.ok:
                self._error_handler(response)
            
            return response
            
        except Exception as e:
            if isinstance(e, (SecAutoError,)):
                raise
            else:
                raise SecAutoConnectionError(f"Connection error: {str(e)}", e)

    def _get_json(self, endpoint, **kwargs):
        """Get JSON response from endpoint."""
        response = self._request('GET', endpoint, **kwargs)
        return response.json()

    def _post_json(self, endpoint, json_data=None, **kwargs):
        """Post JSON data to endpoint."""
        if json_data is not None:
            kwargs['json'] = json_data
        response = self._request('POST', endpoint, **kwargs)
        return response.json()

    def _put_json(self, endpoint, json_data=None, **kwargs):
        """Put JSON data to endpoint."""
        if json_data is not None:
            kwargs['json'] = json_data
        response = self._request('PUT', endpoint, **kwargs)
        return response.json()

    def _delete_json(self, endpoint, **kwargs):
        """Delete resource and return JSON response."""
        response = self._request('DELETE', endpoint, **kwargs)
        return response.json()

    # ========================================================================
    # Health & System Endpoints
    # ========================================================================

    def health(self) -> Dict[str, Any]:
        """
        Get system health status.
        
        Returns:
            Dict containing health information
        """
        return self._get_json('/health')

    # ========================================================================
    # Playbook Management
    # ========================================================================

    def execute_playbook(self, playbook=None, playbook_name=None, context=None) -> PlaybookResponse:
        """
        Execute a playbook synchronously.
        
        Args:
            playbook: Playbook definition (optional if playbook_name provided)
            playbook_name: Name of existing playbook to execute
            context: Execution context variables
            
        Returns:
            PlaybookResponse object with execution results
        """
        request_data = {
            'context': context or {}
        }
        if playbook:
            request_data['playbook'] = playbook
        if playbook_name:
            request_data['playbook_name'] = playbook_name
            
        response = self._post_json('/playbook', request_data)
        return PlaybookResponse(**response)

    def execute_playbook_async(self, playbook=None, playbook_name=None, context=None) -> PlaybookResponse:
        """
        Execute a playbook asynchronously.
        
        Args:
            playbook: Playbook definition (optional if playbook_name provided)
            playbook_name: Name of existing playbook to execute
            context: Execution context variables
            
        Returns:
            PlaybookResponse object with job ID for tracking
        """
        request_data = {
            'context': context or {}
        }
        if playbook:
            request_data['playbook'] = playbook
        if playbook_name:
            request_data['playbook_name'] = playbook_name
            
        response = self._post_json('/playbook/async', request_data)
        return PlaybookResponse(**response)

    def upload_playbook(self, file_path: str, filename: str = None) -> Dict[str, Any]:
        """
        Upload a playbook file.
        
        Args:
            file_path: Path to the playbook file
            filename: Optional filename override
            
        Returns:
            Upload response data
        """
        with open(file_path, 'rb') as f:
            files = {'file': (filename or file_path, f)}
            response = self._request('POST', '/playbook/upload', files=files)
            return response.json()

    def delete_playbook(self, name: str) -> Dict[str, Any]:
        """
        Delete a playbook.
        
        Args:
            name: Name of the playbook to delete
            
        Returns:
            Deletion response data
        """
        return self._delete_json(f'/playbook/{name}')

    def list_playbooks(self) -> List[Dict[str, Any]]:
        """
        List all available playbooks.
        
        Returns:
            List of playbook information
        """
        response = self._get_json('/playbooks')
        return response.get('playbooks', [])

    # ========================================================================
    # Job Management
    # ========================================================================

    def list_jobs(self, status: str = None, limit: int = None, offset: int = None) -> List[Job]:
        """
        List jobs with optional filtering.
        
        Args:
            status: Filter by job status (running, completed, failed, pending)
            limit: Maximum number of jobs to return
            offset: Number of jobs to skip
            
        Returns:
            List of Job objects
        """
        params = {}
        if status:
            params['status'] = status
        if limit:
            params['limit'] = limit
        if offset:
            params['offset'] = offset
            
        response = self._get_json('/jobs', params=params)
        jobs_data = response.get('jobs', [])
        return [Job(**job_data) for job_data in jobs_data]

    def get_job(self, job_id: str) -> Job:
        """
        Get details of a specific job.
        
        Args:
            job_id: ID of the job to retrieve
            
        Returns:
            Job object with full details
        """
        response = self._get_json(f'/job/{job_id}')
        return Job(**response.get('job', {}))

    def cancel_job(self, job_id: str) -> Dict[str, Any]:
        """
        Cancel a running job.
        
        Args:
            job_id: ID of the job to cancel
            
        Returns:
            Cancellation response data
        """
        return self._delete_json(f'/job/{job_id}')

    def get_job_stats(self) -> JobStats:
        """
        Get job statistics.
        
        Returns:
            JobStats object with statistics
        """
        response = self._get_json('/jobs/stats')
        stats_data = response.copy()
        
        # Convert recent_jobs to Job objects
        if 'recent_jobs' in stats_data:
            stats_data['recent_jobs'] = [Job(**job) for job in stats_data['recent_jobs']]
        
        return JobStats(**stats_data)

    # ========================================================================
    # Schedule Management
    # ========================================================================

    def list_schedules(self, status: str = None) -> List[JobSchedule]:
        """
        List job schedules.
        
        Args:
            status: Filter by schedule status (enabled, disabled)
            
        Returns:
            List of JobSchedule objects
        """
        params = {}
        if status:
            params['status'] = status
            
        response = self._get_json('/schedules', params=params)
        schedules_data = response.get('schedules', [])
        return [JobSchedule(**schedule) for schedule in schedules_data]

    def create_schedule(self, name: str, description: str, cron_expression: str,
                       playbook: Any, context: Dict[str, Any] = None, enabled: bool = True) -> JobSchedule:
        """
        Create a new job schedule.
        
        Args:
            name: Schedule name
            description: Schedule description
            cron_expression: Cron expression for scheduling
            playbook: Playbook to execute
            context: Execution context
            enabled: Whether schedule is enabled
            
        Returns:
            Created JobSchedule object
        """
        request_data = {
            'name': name,
            'description': description,
            'cron_expression': cron_expression,
            'playbook': playbook,
            'context': context or {},
            'enabled': enabled
        }
        
        response = self._post_json('/schedules', request_data)
        return JobSchedule(**response.get('schedule', {}))

    def get_schedule(self, schedule_id: str) -> JobSchedule:
        """
        Get details of a specific schedule.
        
        Args:
            schedule_id: ID of the schedule to retrieve
            
        Returns:
            JobSchedule object
        """
        response = self._get_json(f'/schedule/{schedule_id}')
        return JobSchedule(**response.get('schedule', {}))

    def update_schedule(self, schedule_id: str, **kwargs) -> JobSchedule:
        """
        Update a job schedule.
        
        Args:
            schedule_id: ID of the schedule to update
            **kwargs: Fields to update
            
        Returns:
            Updated JobSchedule object
        """
        response = self._put_json(f'/schedule/{schedule_id}', kwargs)
        return JobSchedule(**response.get('schedule', {}))

    def delete_schedule(self, schedule_id: str) -> Dict[str, Any]:
        """
        Delete a job schedule.
        
        Args:
            schedule_id: ID of the schedule to delete
            
        Returns:
            Deletion response data
        """
        return self._delete_json(f'/schedule/{schedule_id}')

    def execute_schedule(self, schedule_id: str) -> Dict[str, Any]:
        """
        Execute a schedule manually.
        
        Args:
            schedule_id: ID of the schedule to execute
            
        Returns:
            Execution response data
        """
        return self._post_json(f'/schedule/execute/{schedule_id}')

    def get_schedule_stats(self) -> Dict[str, Any]:
        """
        Get schedule statistics.
        
        Returns:
            Schedule statistics data
        """
        return self._get_json('/schedules/stats')

    # ========================================================================
    # Cache Operations
    # ========================================================================

    def get_cache_info(self) -> Dict[str, Any]:
        """
        Get cache information.
        
        Returns:
            Cache information data
        """
        return self._get_json('/cache')

    def get_cache_stats(self) -> CacheStats:
        """
        Get cache statistics.
        
        Returns:
            CacheStats object
        """
        response = self._get_json('/cache/stats')
        return CacheStats(**response.get('stats', {}))

    def clear_cache(self) -> Dict[str, Any]:
        """
        Clear all cache entries.
        
        Returns:
            Clear operation response
        """
        return self._post_json('/cache/clear')

    def get_cache_value(self, key: str) -> Any:
        """
        Retrieve a value from cache.
        
        Args:
            key: Cache key
            
        Returns:
            Cached value or None if not found
        """
        response = self._get_json(f'/cache/{key}')
        return response.get('value')

    def set_cache_value(self, key: str, value: Any, ttl: int = None) -> Dict[str, Any]:
        """
        Store a value in cache.
        
        Args:
            key: Cache key
            value: Value to store
            ttl: Time to live in seconds (optional)
            
        Returns:
            Set operation response
        """
        request_data = {'value': value}
        if ttl:
            request_data['ttl'] = ttl
            
        return self._post_json(f'/cache/{key}', request_data)

    def delete_cache_value(self, key: str) -> Dict[str, Any]:
        """
        Delete a value from cache.
        
        Args:
            key: Cache key to delete
            
        Returns:
            Delete operation response
        """
        return self._delete_json(f'/cache/{key}')

    # ========================================================================
    # List Operations (Redis)
    # ========================================================================

    def get_list(self, list_name: str) -> List[Any]:
        """
        Get items from a Redis list.
        
        Args:
            list_name: Name of the list
            
        Returns:
            List of items
        """
        response = self._get_json(f'/lists/{list_name}')
        return response.get('items', [])

    def add_to_list(self, list_name: str, items: List[Any], position: str = 'right') -> Dict[str, Any]:
        """
        Add items to a Redis list.
        
        Args:
            list_name: Name of the list
            items: Items to add
            position: Position to add ('left' or 'right')
            
        Returns:
            Add operation response
        """
        request_data = {
            'items': items,
            'position': position
        }
        return self._post_json(f'/lists/{list_name}/items', request_data)

    def remove_from_list(self, list_name: str, items: List[Any], count: int = 1) -> Dict[str, Any]:
        """
        Remove items from a Redis list.
        
        Args:
            list_name: Name of the list
            items: Items to remove
            count: Number of occurrences to remove
            
        Returns:
            Remove operation response
        """
        request_data = {
            'items': items,
            'count': count
        }
        return self._request('DELETE', f'/lists/{list_name}/items', json=request_data).json()

    def delete_list(self, list_name: str) -> Dict[str, Any]:
        """
        Delete an entire Redis list.
        
        Args:
            list_name: Name of the list to delete
            
        Returns:
            Delete operation response
        """
        return self._delete_json(f'/lists/{list_name}')

    # ========================================================================
    # Integration Management
    # ========================================================================

    def list_integrations(self) -> List[Integration]:
        """
        List all available integrations.
        
        Returns:
            List of Integration objects
        """
        response = self._get_json('/integrations')
        integrations_data = response.get('integrations', [])
        return [Integration(**integration) for integration in integrations_data]

    def get_integration(self, name: str) -> Integration:
        """
        Get details of a specific integration.
        
        Args:
            name: Name of the integration
            
        Returns:
            Integration object
        """
        response = self._get_json(f'/integrations/{name}')
        return Integration(**response.get('integration', {}))

    def upload_integration(self, file_path: str, filename: str = None) -> Dict[str, Any]:
        """
        Upload an integration file.
        
        Args:
            file_path: Path to the integration file
            filename: Optional filename override
            
        Returns:
            Upload response data
        """
        with open(file_path, 'rb') as f:
            files = {'file': (filename or file_path, f)}
            response = self._request('POST', '/integrations/upload', files=files)
            return response.json()

    def get_integration_build_status(self, name: str) -> Dict[str, Any]:
        """
        Get build status of an integration.
        
        Args:
            name: Name of the integration
            
        Returns:
            Build status data
        """
        return self._get_json(f'/integrations/build-status/{name}')

    # ========================================================================
    # Automation Management
    # ========================================================================

    def list_automations(self) -> List[AutomationInfo]:
        """
        List all available automations.
        
        Returns:
            List of AutomationInfo objects
        """
        response = self._get_json('/automations')
        automations_data = response.get('automations', [])
        return [AutomationInfo(**automation) for automation in automations_data]

    def upload_automation(self, file_path: str, filename: str = None) -> Dict[str, Any]:
        """
        Upload an automation script.
        
        Args:
            file_path: Path to the automation file
            filename: Optional filename override
            
        Returns:
            Upload response data
        """
        with open(file_path, 'rb') as f:
            files = {'file': (filename or file_path, f)}
            response = self._request('POST', '/automation', files=files)
            return response.json()

    def delete_automation(self, name: str) -> Dict[str, Any]:
        """
        Delete an automation script.
        
        Args:
            name: Name of the automation to delete
            
        Returns:
            Deletion response data
        """
        return self._delete_json(f'/automation/{name}')

    def list_automation_metadata(self) -> List[AutomationMetadata]:
        """
        List automation metadata.
        
        Returns:
            List of AutomationMetadata objects
        """
        response = self._get_json('/automation/metadata')
        metadata_list = response.get('metadata', [])
        return [AutomationMetadata(**metadata) for metadata in metadata_list]

    def get_automation_metadata(self, name: str) -> AutomationMetadata:
        """
        Get metadata for a specific automation.
        
        Args:
            name: Name of the automation
            
        Returns:
            AutomationMetadata object
        """
        response = self._get_json(f'/automation/metadata/{name}')
        return AutomationMetadata(**response.get('metadata', {}))

    # ========================================================================
    # Client Management
    # ========================================================================

    def list_clients(self) -> List[Client]:
        """
        List all clients.
        
        Returns:
            List of Client objects
        """
        response = self._get_json('/clients')
        clients_data = response.get('clients', [])
        return [Client(**client) for client in clients_data]

    def create_client(self, name: str, description: str = None, enabled: bool = True,
                     metadata: Dict[str, Any] = None) -> Client:
        """
        Create a new client.
        
        Args:
            name: Client name
            description: Client description
            enabled: Whether client is enabled
            metadata: Additional client metadata
            
        Returns:
            Created Client object
        """
        request_data = {
            'name': name,
            'description': description,
            'enabled': enabled,
            'metadata': metadata or {}
        }
        
        response = self._post_json('/clients', request_data)
        return Client(**response.get('client', {}))

    def get_client(self, client_id: str) -> Client:
        """
        Get details of a specific client.
        
        Args:
            client_id: ID of the client
            
        Returns:
            Client object
        """
        response = self._get_json(f'/clients/{client_id}')
        return Client(**response.get('client', {}))

    def update_client(self, client_id: str, **kwargs) -> Client:
        """
        Update a client.
        
        Args:
            client_id: ID of the client to update
            **kwargs: Fields to update
            
        Returns:
            Updated Client object
        """
        response = self._put_json(f'/clients/{client_id}', kwargs)
        return Client(**response.get('client', {}))

    def delete_client(self, client_id: str) -> Dict[str, Any]:
        """
        Delete a client.
        
        Args:
            client_id: ID of the client to delete
            
        Returns:
            Deletion response data
        """
        return self._delete_json(f'/clients/{client_id}')

    # ========================================================================
    # Client Integration Management
    # ========================================================================

    def list_client_integrations(self, client_id: str) -> List[Dict[str, Any]]:
        """
        List integrations for a specific client.
        
        Args:
            client_id: ID of the client
            
        Returns:
            List of client integration data
        """
        response = self._get_json(f'/clients/{client_id}/integrations')
        return response.get('integrations', [])

    def get_client_integration_config(self, client_id: str, integration_name: str) -> Dict[str, Any]:
        """
        Get integration configuration for a client.
        
        Args:
            client_id: ID of the client
            integration_name: Name of the integration
            
        Returns:
            Integration configuration data
        """
        response = self._get_json(f'/clients/{client_id}/integrations/{integration_name}/config')
        return response.get('config', {})

    def set_client_integration_config(self, client_id: str, integration_name: str,
                                    config: Dict[str, Any], enabled: bool = True) -> Dict[str, Any]:
        """
        Set integration configuration for a client.
        
        Args:
            client_id: ID of the client
            integration_name: Name of the integration
            config: Integration configuration
            enabled: Whether integration is enabled
            
        Returns:
            Configuration response data
        """
        request_data = {
            'config': config,
            'enabled': enabled
        }
        
        return self._post_json(f'/clients/{client_id}/integrations/{integration_name}/config', request_data)

    def update_client_integration_config(self, client_id: str, integration_name: str,
                                       config: Dict[str, Any] = None, enabled: bool = None) -> Dict[str, Any]:
        """
        Update integration configuration for a client.
        
        Args:
            client_id: ID of the client
            integration_name: Name of the integration
            config: Integration configuration to update
            enabled: Whether integration is enabled
            
        Returns:
            Update response data
        """
        request_data = {}
        if config is not None:
            request_data['config'] = config
        if enabled is not None:
            request_data['enabled'] = enabled
            
        return self._put_json(f'/clients/{client_id}/integrations/{integration_name}/config', request_data)

    def delete_client_integration_config(self, client_id: str, integration_name: str) -> Dict[str, Any]:
        """
        Delete integration configuration for a client.
        
        Args:
            client_id: ID of the client
            integration_name: Name of the integration
            
        Returns:
            Deletion response data
        """
        return self._delete_json(f'/clients/{client_id}/integrations/{integration_name}/config')

    def execute_client_integration(self, client_id: str, integration_name: str,
                                 function: str, params: Dict[str, Any] = None) -> Dict[str, Any]:
        """
        Execute an integration function for a client.
        
        Args:
            client_id: ID of the client
            integration_name: Name of the integration
            function: Function to execute
            params: Function parameters
            
        Returns:
            Execution response data
        """
        request_data = {
            'function': function,
            'params': params or {}
        }
        
        return self._post_json(f'/clients/{client_id}/integrations/{integration_name}/execute', request_data)

    # ========================================================================
    # API Key Management
    # ========================================================================

    def list_api_keys(self) -> List[APIKeySummary]:
        """
        List all API keys (returns summaries, not full keys).
        
        Returns:
            List of APIKeySummary objects
        """
        response = self._get_json('/api-keys')
        api_keys_data = response.get('api_keys', [])
        return [APIKeySummary(**api_key) for api_key in api_keys_data]

    def create_api_key(self, name: str, description: str = None) -> APIKey:
        """
        Create a new API key.
        
        Args:
            name: Name for the API key
            description: Optional description
            
        Returns:
            Created APIKey object (includes full key)
        """
        request_data = {
            'name': name,
            'description': description
        }
        
        response = self._post_json('/api-keys', request_data)
        return APIKey(**response.get('api_key', {}))

    def get_api_key_stats(self) -> Dict[str, Any]:
        """
        Get API key usage statistics.
        
        Returns:
            API key statistics data
        """
        return self._get_json('/api-keys/stats')

    # ========================================================================
    # Cluster Management
    # ========================================================================

    def get_cluster_status(self) -> Dict[str, Any]:
        """
        Get cluster status information.
        
        Returns:
            Cluster status data
        """
        return self._get_json('/cluster')

    def list_cluster_jobs(self) -> List[Dict[str, Any]]:
        """
        List jobs in the cluster.
        
        Returns:
            List of cluster job data
        """
        response = self._get_json('/cluster/jobs')
        return response.get('jobs', [])

    def get_cluster_job(self, job_id: str) -> Dict[str, Any]:
        """
        Get details of a cluster job.
        
        Args:
            job_id: ID of the cluster job
            
        Returns:
            Cluster job data
        """
        return self._get_json(f'/cluster/jobs/{job_id}')

    # ========================================================================
    # Utility Methods
    # ========================================================================

    def test_connection(self) -> bool:
        """
        Test the connection to the SecAuto server.
        
        Returns:
            True if connection is successful, False otherwise
        """
        try:
            health_data = self.health()
            return health_data.get('success', False)
        except Exception:
            return False

    def get_server_info(self) -> Dict[str, Any]:
        """
        Get server information from health endpoint.
        
        Returns:
            Server information dictionary
        """
        return self.health()
