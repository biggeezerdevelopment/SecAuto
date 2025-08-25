#!/usr/bin/env python3
"""
SecAuto Base Module
Provides easy access to SecAuto base functions for automation scripts
"""

import json
import sys
import os
from pathlib import Path

# Try to find and import SoarBaseAPI from various locations
def _find_and_import_base_api():
    """Find and import SoarBaseAPI functions"""
    
    # Possible locations for SoarBaseAPI
    possible_paths = [
        # Current directory
        ".",
        # Server directory (relative to current script)
        #"../server",
        #"../../server", 
        #"../../../server",
        # Absolute path if we can determine it
        #str(Path(__file__).parent),
    ]
    
    for path in possible_paths:
        try:
            if path not in sys.path:
                sys.path.insert(0, path)
            
            from SoarBaseAPI import load_context as _load_context, return_context as _return_context
            return _load_context, _return_context
        except ImportError:
            continue
    
    # If we can't import, create simple fallback versions
    def _load_context():
        """Load execution context from stdin"""
        try:
            stdin_data = sys.stdin.read().strip()
            if stdin_data:
                return json.loads(stdin_data)
        except Exception:
            pass
        
        # Try environment variable as fallback
        env_context = os.environ.get('SECAUTO_CONTEXT')
        if env_context:
            try:
                return json.loads(env_context)
            except Exception:
                pass
        
        # Try command line argument as fallback
        if len(sys.argv) > 1:
            try:
                return json.loads(sys.argv[1])
            except Exception:
                pass
        
        return {}
    
    def _return_context(result):
        """Return execution result as JSON to stdout"""
        print(json.dumps(result, indent=2))
    
    return _load_context, _return_context


# Import the functions
load_context, return_context = _find_and_import_base_api()

# Also provide some helper functions
def get_client_id():
    """Get client ID from context"""
    context = load_context()
    return context.get('client_id')

def get_config():
    """Get configuration from context"""
    context = load_context()
    return context.get('config', {})

def get_credentials():
    """Get credentials from context"""
    context = load_context()
    return context.get('credentials', {})

def send_result(success=True, data=None, message=None, error=None):
    """Send a formatted result"""
    result = {
        'success': success,
        'timestamp': json.dumps(None)  # Will be set by JSON encoder
    }
    
    if data is not None:
        result['data'] = data
    
    if message:
        result['message'] = message
    
    if error:
        result['error'] = error
    
    return_context(result)

def use_integration(integration_name, function_name, client_id=None, **kwargs):
    """
    Call another integration from within an automation script
    This is a simplified version for automation scripts
    """
    # Import the integration loader
    try:
        sys.path.insert(0, str(Path(__file__).parent))
        from integration_loader import IntegrationLoader
        
        # Calculate the correct base path for the integration system
        # From SoarAuto/server, we need to go up to SoarAuto directory
        server_dir = Path(__file__).parent
        project_root = server_dir.parent  # SoarAuto/server -> SoarAuto
        
        loader = IntegrationLoader(str(project_root))
        result = loader.use_integration(integration_name, function_name, client_id, **kwargs)
        return result
        
    except Exception as e:
        return {
            'success': False,
            'error': f'Failed to call integration {integration_name}.{function_name}: {str(e)}'
        }