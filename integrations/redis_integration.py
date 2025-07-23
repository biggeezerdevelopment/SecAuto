"""
Redis Integration for SecAuto

This module provides Redis caching functionality that can be used
from Python automations in SecAuto playbooks.

Usage in automations:
    from integrations.redis_integration import RedisIntegration
    
    redis = RedisIntegration()
    result = redis.get_cache("key")
"""

import json
import sys
from typing import Dict, Any, Optional, List
from dataclasses import dataclass, asdict

# Try to import urllib3 for SSL warning disable

try:
    import redis
except ImportError:
    print("Warning: redis module not found. Install with: pip install redis", file=sys.stderr)
    redis = None


class RedisIntegration:
    """Redis integration for SecAuto"""
    
    def __init__(self, config_manager=None):
        """
        Initialize VirusTotal integration
        
        Args:
            config_manager: Optional configuration manager for testing
        """
        self.config_manager = config_manager
        self.redis_host = None
        self.redis_port = None
        self.redis_password = None
        
        # Load configuration
        self._load_config()
    
    def _load_config(self):
        """Load configuration from SecAuto's integration config system"""
        try:
           # Try to fetch config from SecAuto server
           config = get_integration_config("redis")
           if config:
                self.redis_host = config.get("host", "localhost")
                self.redis_port = config.get("port", 6379)
                self.redis_password = config.get("password", "")
                return
        except NameError:
            # get_integration_config function not available (not running in SecAuto context)
            # Use default configuration
            self.redis_host = "localhost"
            self.redis_port = 6379
            self.redis_password = ""
        except Exception as e:
            self.redis_host = "localhost"
            self.redis_port = 6379
            self.redis_password = ""
            #logger.error(f"Failed to load Redis configuration: {e}")
    
    def get_cache(self, key: str) -> Dict[str, Any]:
        """
        Get a value from Redis
        
        Args:
            key: Key to get value for
            
        Returns:
            Dictionary with scan results
        """
        if not self.redis_host:
            return {
                "redis": {
                    "success": False,
                    "error_message": "Redis host not configured"
                }
            }
        
        if redis is None:
            return {
                "redis": {
                    "success": False,
                    "error_message": "Redis module not available. Install with: pip install redis"
                }
            }
            
        try:
            # Get value from Redis
            redis_client = redis.Redis(host=self.redis_host, 
                                       port=self.redis_port, 
                                       password=self.redis_password)
            value = redis_client.get(key)
            
            if value:
                return {
                    "redis": {
                        "success": True,
                        "value": value.decode('utf-8')
                    }
                }
            else:
                return {
                    "redis": {
                        "success": False,
                        "error_message": "Key not found in Redis"
                    }
                }
            
        except Exception as e:
            return {
                "redis": {
                    "success": False,
                    "error_message": f"Unexpected error: {str(e)}"
                }
            }
            
    def set_cache(self, key: str, value: str) -> Dict[str, Any]:
        """Set a value in Redis"""
        

            
        try:
            # Set value in Redis
            redis_client = redis.Redis(host="localhost", 
                                       port=6379, 
                                       password="")
            redis_client.set(key, value)
            
            return {
                "redis": {
                    "success": True,
                    "message": "Value set in Redis"
                }
            }
                
        except Exception as e:
            return {
                "redis": {
                    "success": False,
                    "error_message": f"Unexpected error: {str(e)}"
                }
            }
    
    def delete_cache(self, key: str) -> Dict[str, Any]:
        """
        Delete a value from Redis
        
        Args:
            key: Key to delete value for
            
        Returns:
            Dictionary with report data
        """
        if not self.redis_host:
            return {
                "redis": {
                    "success": False,
                    "error_message": "Redis host not configured"
                }
            }
        
        if redis is None:
            return {
                "redis": {
                    "success": False,
                    "error_message": "Redis module not available. Install with: pip install redis"
                }
            }
            
        try:
            # Delete value from Redis
            redis_client = redis.Redis(host=self.redis_host, 
                                       port=self.redis_port, 
                                       password=self.redis_password)
            redis_client.delete(key)
            
            return {
                "redis": {
                    "success": True,
                    "message": "Value deleted from Redis"
                }
            }
            
        except Exception as e:
            return {
                "redis": {
                    "success": False,
                    "error_message": f"Unexpected error: {str(e)}"
                }
            }


if __name__ == "__main__":
    print("Testing Redis integration...")
    redis_integration = RedisIntegration("redis")
    tenable_result = redis_integration.get_cache("tenable_user_count_guidepoint")
    qualys_result = redis_integration.get_cache("qualys_user_count_guidepoint")

    print(f"Tenable Result: {json.dumps(tenable_result, indent=2)}") 
    print(f"Qualys Result: {json.dumps(qualys_result, indent=2)}") 
