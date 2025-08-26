#!/usr/bin/env python3
"""
SecAuto SDK Test Suite

Comprehensive tests for the SecAuto Python SDK.
These tests require a running SecAuto server for integration testing.
"""

import unittest
import os
import time
import json
from unittest.mock import Mock, patch

from secauto_sdk import SecAutoClient
from secauto_sdk.exceptions import (
    SecAutoError, SecAutoAPIError, SecAutoConnectionError,
    SecAutoAuthenticationError, SecAutoNotFoundError
)
from secauto_sdk.models import Job, PlaybookResponse, Client


class TestSecAutoClient(unittest.TestCase):
    """Test cases for SecAutoClient."""
    
    @classmethod
    def setUpClass(cls):
        """Set up test class with client instance."""
        cls.url = os.getenv('SECAUTO_TEST_URL', 'http://localhost:9090')
        cls.api_key = os.getenv('SECAUTO_TEST_API_KEY', 'secauto-api-key-2024-07-14')
        cls.client = SecAutoClient(cls.url, cls.api_key)
        
        # Test connection
        try:
            cls.connection_available = cls.client.test_connection()
        except Exception:
            cls.connection_available = False
        
        if not cls.connection_available:
            print(f"Warning: SecAuto server not available at {cls.url}")
            print("Some tests will be skipped.")
    
    def setUp(self):
        """Set up each test case."""
        if not self.connection_available:
            self.skipTest("SecAuto server not available")
    
    def test_client_initialization(self):
        """Test client initialization."""
        client = SecAutoClient('http://test.com', 'test-key')
        self.assertEqual(client._url, 'http://test.com')
        self.assertEqual(client._api_key, 'test-key')
        self.assertEqual(client._session.headers['X-API-Key'], 'test-key')
    
    def test_health_check(self):
        """Test health endpoint."""
        health = self.client.health()
        self.assertIsInstance(health, dict)
        # Health endpoint should be accessible without authentication
    
    def test_connection_test(self):
        """Test connection testing method."""
        result = self.client.test_connection()
        self.assertTrue(result)
    
    def test_playbook_execution(self):
        """Test synchronous playbook execution."""
        # Simple test playbook
        test_playbook = {
            "name": "test_playbook",
            "steps": [
                {
                    "name": "test_step",
                    "action": "log",
                    "params": {"message": "test message"}
                }
            ]
        }
        
        context = {
            'test_run': True,
            'timestamp': time.time()
        }
        
        try:
            response = self.client.execute_playbook(
                playbook=test_playbook,
                context=context
            )
            self.assertIsInstance(response, PlaybookResponse)
            # Note: Success depends on server configuration
        except SecAutoError:
            # Expected if playbook execution is not configured
            pass
    
    def test_async_playbook_execution(self):
        """Test asynchronous playbook execution."""
        test_playbook = {
            "name": "async_test_playbook",
            "steps": [
                {
                    "name": "async_test_step",
                    "action": "log",
                    "params": {"message": "async test message"}
                }
            ]
        }
        
        context = {'test_run': True}
        
        try:
            response = self.client.execute_playbook_async(
                playbook=test_playbook,
                context=context
            )
            self.assertIsInstance(response, PlaybookResponse)
            if response.success and response.job_id:
                # Monitor the job briefly
                time.sleep(1)
                job = self.client.get_job(response.job_id)
                self.assertIsInstance(job, Job)
        except SecAutoError:
            # Expected if async execution is not configured
            pass
    
    def test_job_management(self):
        """Test job listing and management."""
        try:
            # List jobs
            jobs = self.client.list_jobs(limit=5)
            self.assertIsInstance(jobs, list)
            
            # Get job stats
            stats = self.client.get_job_stats()
            self.assertIsNotNone(stats.total_jobs)
            self.assertGreaterEqual(stats.total_jobs, 0)
            
            # If there are jobs, test getting individual job
            if jobs:
                job = jobs[0]
                retrieved_job = self.client.get_job(job.id)
                self.assertEqual(job.id, retrieved_job.id)
                
        except SecAutoError as e:
            self.fail(f"Job management failed: {e}")
    
    def test_cache_operations(self):
        """Test cache operations."""
        test_key = f'sdk_test_{int(time.time())}'
        test_value = {
            'test_data': 'test_value',
            'timestamp': time.time(),
            'nested': {'key': 'value'}
        }
        
        try:
            # Set cache value
            set_response = self.client.set_cache_value(test_key, test_value)
            self.assertIsInstance(set_response, dict)
            
            # Get cache value
            retrieved_value = self.client.get_cache_value(test_key)
            self.assertEqual(retrieved_value['test_data'], test_value['test_data'])
            
            # Get cache info
            cache_info = self.client.get_cache_info()
            self.assertIsInstance(cache_info, dict)
            
            # Get cache stats
            cache_stats = self.client.get_cache_stats()
            self.assertIsNotNone(cache_stats)
            
            # Delete cache value
            delete_response = self.client.delete_cache_value(test_key)
            self.assertIsInstance(delete_response, dict)
            
        except SecAutoError as e:
            self.fail(f"Cache operations failed: {e}")
    
    def test_list_operations(self):
        """Test Redis list operations."""
        test_list = f'sdk_test_list_{int(time.time())}'
        test_items = [
            {'id': 1, 'name': 'item1'},
            {'id': 2, 'name': 'item2'},
            {'id': 3, 'name': 'item3'}
        ]
        
        try:
            # Add items to list
            add_response = self.client.add_to_list(test_list, test_items)
            self.assertIsInstance(add_response, dict)
            
            # Get list items
            retrieved_items = self.client.get_list(test_list)
            self.assertIsInstance(retrieved_items, list)
            self.assertGreater(len(retrieved_items), 0)
            
            # Remove items from list
            remove_response = self.client.remove_from_list(test_list, [test_items[0]])
            self.assertIsInstance(remove_response, dict)
            
            # Delete entire list
            delete_response = self.client.delete_list(test_list)
            self.assertIsInstance(delete_response, dict)
            
        except SecAutoError as e:
            self.fail(f"List operations failed: {e}")
    
    def test_integration_management(self):
        """Test integration listing."""
        try:
            integrations = self.client.list_integrations()
            self.assertIsInstance(integrations, list)
            
            # If there are integrations, test getting individual integration
            if integrations:
                integration = integrations[0]
                retrieved_integration = self.client.get_integration(integration.name)
                self.assertEqual(integration.name, retrieved_integration.name)
                
        except SecAutoError as e:
            self.fail(f"Integration management failed: {e}")
    
    def test_automation_management(self):
        """Test automation listing."""
        try:
            automations = self.client.list_automations()
            self.assertIsInstance(automations, list)
            
            # Test automation metadata
            metadata_list = self.client.list_automation_metadata()
            self.assertIsInstance(metadata_list, list)
            
        except SecAutoError as e:
            self.fail(f"Automation management failed: {e}")
    
    def test_client_management(self):
        """Test client management operations."""
        test_client_name = f'sdk_test_client_{int(time.time())}'
        
        try:
            # List existing clients
            clients = self.client.list_clients()
            self.assertIsInstance(clients, list)
            
            # Create a test client
            created_client = self.client.create_client(
                name=test_client_name,
                description='Test client created by SDK',
                metadata={'test': True}
            )
            self.assertIsInstance(created_client, Client)
            self.assertEqual(created_client.name, test_client_name)
            
            # Get the created client
            retrieved_client = self.client.get_client(created_client.id)
            self.assertEqual(created_client.id, retrieved_client.id)
            
            # Update the client
            updated_client = self.client.update_client(
                created_client.id,
                description='Updated test client'
            )
            self.assertEqual(updated_client.description, 'Updated test client')
            
            # Delete the test client
            delete_response = self.client.delete_client(created_client.id)
            self.assertIsInstance(delete_response, dict)
            self.assertTrue(delete_response.get('success', False))
            
        except SecAutoError as e:
            self.fail(f"Client management failed: {e}")
    
    def test_schedule_management(self):
        """Test schedule management."""
        try:
            # List schedules
            schedules = self.client.list_schedules()
            self.assertIsInstance(schedules, list)
            
            # Get schedule stats
            schedule_stats = self.client.get_schedule_stats()
            self.assertIsInstance(schedule_stats, dict)
            
        except SecAutoError as e:
            self.fail(f"Schedule management failed: {e}")
    
    def test_api_key_management(self):
        """Test API key management."""
        try:
            # List API keys
            api_keys = self.client.list_api_keys()
            self.assertIsInstance(api_keys, list)
            
            # Get API key stats
            api_key_stats = self.client.get_api_key_stats()
            self.assertIsInstance(api_key_stats, dict)
            
        except SecAutoError as e:
            self.fail(f"API key management failed: {e}")
    
    def test_cluster_management(self):
        """Test cluster management."""
        try:
            # Get cluster status
            cluster_status = self.client.get_cluster_status()
            self.assertIsInstance(cluster_status, dict)
            
            # List cluster jobs
            cluster_jobs = self.client.list_cluster_jobs()
            self.assertIsInstance(cluster_jobs, list)
            
        except SecAutoError as e:
            self.fail(f"Cluster management failed: {e}")
    
    def test_error_handling(self):
        """Test error handling for various scenarios."""
        # Test with invalid API key
        invalid_client = SecAutoClient(self.url, 'invalid-api-key')
        
        with self.assertRaises(SecAutoAuthenticationError):
            invalid_client.get_job('test-job-id')
        
        # Test not found error
        with self.assertRaises(SecAutoNotFoundError):
            self.client.get_job('non-existent-job-id')
        
        # Test with invalid URL
        invalid_url_client = SecAutoClient('http://invalid-url:9999', self.api_key)
        with self.assertRaises(SecAutoConnectionError):
            invalid_url_client.health()


class TestMockedClient(unittest.TestCase):
    """Test cases using mocked responses."""
    
    def setUp(self):
        """Set up mocked client."""
        self.client = SecAutoClient('http://test.com', 'test-key')
    
    @patch('requests.Session.request')
    def test_successful_health_check(self, mock_request):
        """Test successful health check with mocked response."""
        mock_response = Mock()
        mock_response.ok = True
        mock_response.json.return_value = {'status': 'healthy', 'success': True}
        mock_request.return_value = mock_response
        
        health = self.client.health()
        self.assertEqual(health['status'], 'healthy')
    
    @patch('requests.Session.request')
    def test_api_error_handling(self, mock_request):
        """Test API error handling with mocked responses."""
        # Test 401 Unauthorized
        mock_response = Mock()
        mock_response.ok = False
        mock_response.status_code = 401
        mock_response.json.return_value = {'message': 'Unauthorized'}
        mock_request.return_value = mock_response
        
        with self.assertRaises(SecAutoAuthenticationError):
            self.client.health()
        
        # Test 404 Not Found
        mock_response.status_code = 404
        mock_response.json.return_value = {'message': 'Not Found'}
        
        with self.assertRaises(SecAutoNotFoundError):
            self.client.get_job('test-id')
        
        # Test 500 Server Error
        mock_response.status_code = 500
        mock_response.json.return_value = {'message': 'Internal Server Error'}
        
        with self.assertRaises(SecAutoAPIError):
            self.client.health()


if __name__ == '__main__':
    # Configure logging for tests
    import logging
    logging.basicConfig(level=logging.INFO)
    
    # Run tests
    unittest.main(verbosity=2)
