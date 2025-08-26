"""
SecAuto Python SDK

A comprehensive Python SDK for interacting with the SecAuto SOAR automation platform.
Built using restfly for robust REST API interactions.

Usage:
    from secauto_sdk import SecAutoClient
    
    client = SecAutoClient('http://localhost:9090', api_key='your-api-key')
    health = client.health()
    print(health)
"""

from .client import SecAutoClient
from .exceptions import SecAutoError, SecAutoAPIError, SecAutoConnectionError
from .models import *

__version__ = '1.0.0'
__author__ = 'SecAuto Team'
__license__ = 'MIT'

__all__ = [
    'SecAutoClient',
    'SecAutoError', 
    'SecAutoAPIError',
    'SecAutoConnectionError'
]
