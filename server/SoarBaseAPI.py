import os
import json
import sys
import requests
from typing import Dict, Any

# Try to import urllib3 for SSL warning disable
try:
    import urllib3
    # Disable SSL warnings
    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
    urllib3.disable_warnings(urllib3.exceptions.NotOpenSSLWarning)
    urllib3.disable_warnings(urllib3.exceptions.InsecurePlatformWarning)
except ImportError:
    print("Warning: urllib3 module not found. SSL warnings may appear.", file=sys.stderr)

def base_context():
    context = {}
    return json.dumps(context)

def load_context():
    try:
        # Try environment variable first (more reliable)
        env_context = os.environ.get('SECAUTO_CONTEXT')
        if env_context:
            try:
                return json.loads(env_context)
            except json.JSONDecodeError:
                pass
        
        # Try to read JSON input from stdin
        input_data = None
        try:
            # Check if there's data in stdin
            if not sys.stdin.isatty():
                input_data = json.load(sys.stdin)
        except json.JSONDecodeError:
            # If stdin is not valid JSON, continue without input
            pass
        except Exception:
            # If any other error reading stdin, continue without input
            pass
        return input_data
    except Exception as e:
        print(f"Error loading context: {e}",file=sys.stderr)
        return None

def return_context(data):
    """Return data as JSON string"""
    print(json.dumps(data, indent=2))

def update_context(context, data):
    if isinstance(context, dict):
        context.update(data)
        return context
    else:
        return None

def search_context_iterative(json_obj, key):
    stack = [json_obj]
    while stack:
        current = stack.pop()
        
        if isinstance(current, dict):
            if key in current:
                return current[key]
            for value in current.values():
                if isinstance(value, (dict, list)):
                    stack.append(value)
        
        elif isinstance(current, list):
            for item in current:
                if isinstance(item, (dict, list)):
                    stack.append(item)
    return None    

def search_context(json_obj, key):
    if isinstance(json_obj, dict):
        if key in json_obj:
            return json_obj[key]
        for k, v in json_obj.items():
            result = search_context(v, key)
            if result is not None:
                return result
    elif isinstance(json_obj, list):
        for item in json_obj:
            result = search_context(item, key)
            if result is not None:
                return result
    return None

def search_context_path(json_obj, key_path):
    keys = key_path.split('.')
    current_data = json_obj
    for key in keys:
        if isinstance(current_data, dict) and key in current_data:
            current_data = current_data[key]
        else:
            return None  # Key path does not exist
    return current_data


def get_secauto_config():
        """
        Get SecAuto configuration from config file or environment variables
        
        Returns:
            Tuple of (secauto_url, secauto_api_key)
        """
        
        # Try to read from config file written by server
        config_paths = [
            "data/integration_config.json",
            "SoarAuto/data/integration_config.json",
            "../SoarAuto/data/integration_config.json"
        ]
        
        for config_path in config_paths:
            try:
                if os.path.exists(config_path):
                    with open(config_path, 'r') as f:
                        config = json.load(f)
                    
                    url = config.get("secauto_url")
                    api_key = config.get("secauto_api_key")
                    
                    if url and api_key:
                        #logger.info(f"Using SecAuto config from {config_path}")
                        return url, api_key
            except Exception as e:
                #logger.debug(f"Failed to read config from {config_path}: {e}")
                continue
        
        # Fallback to defaults
        #logger.warning("No SecAuto config found, using defaults")
        return "http://localhost:8080", None

def get_integration_config(integration_name: str, client_id: str = None) -> dict:         
    """
    Get integration configuration, with optional client-specific override
    
    Args:
        integration_name: Name of the integration
        client_id: Optional client ID for client-specific config
        
    Returns:
        Integration configuration dictionary or None
    """
    # Get configuration
    secauto_url, secauto_api_key = get_secauto_config()
    
    if not secauto_api_key:
        return None
        
    headers = {
        "X-API-Key": secauto_api_key,
        "Content-Type": "application/json"
    }
    
    # If client_id is provided, try client-specific config first
    if client_id:
        try:
            response = requests.get(
                f"{secauto_url}/clients/{client_id}/integrations/{integration_name}",
                headers=headers,
                timeout=10
            )               
            if response.status_code == 200:
                data = response.json()
                if data.get("success") and data.get("integration"):
                    return data["integration"]
        except requests.exceptions.RequestException as e:
            # Fall back to global config if client-specific fails
            pass
    
    # Try global integration config
    try:
        response = requests.get(
                    f"{secauto_url}/integrations/{integration_name}",
                    headers=headers,
                    timeout=10
                    )               
        if response.status_code == 200:
            data = response.json()
            if data.get("success") and data.get("integration"):
                return data["integration"]
            else:
                return None
    except requests.exceptions.RequestException as e:
                #logger.debug(f"Request failed for '{integration_name}': {e}")
                return None
                
def set_cache(key: str, value: str) -> Dict[str, Any]:
    headers = {
        "X-API-Key": secauto_api_key,
        "Content-Type": "application/json"
    }
    newvalue = {"value": value}
    resp = requests.post(
        f"{secauto_url}/cache/{key}",
        headers=headers,
        json=newvalue
    )
    if resp.status_code == 200:
        return resp.json()
    else:
        return None

def get_cache(key: str) -> Dict[str, Any]:
    """Get cache value"""
    headers = {
        "X-API-Key": secauto_api_key,
        "Content-Type": "application/json"
    }
    resp = requests.get(
        f"{secauto_url}/cache/{key}",
        headers=headers,
    )
    if resp.status_code == 200:
        return resp.json()
    else:
        return None

def delete_cache(key: str) -> Dict[str, Any]:
    """Delete cache value"""
    headers = {
        "X-API-Key": secauto_api_key,
        "Content-Type": "application/json"
    }
    resp = requests.delete(
        f"{secauto_url}/cache/{key}",
        headers=headers,
    )
    if resp.status_code == 200:
        return resp.json()
    else:
        return None
    
def get_list(list_name: str) -> Dict[str, Any]:
    """Get list value"""
    headers = {
        "X-API-Key": secauto_api_key,
        "Content-Type": "application/json"
    }
    resp = requests.get(
        f"{secauto_url}/list/{list_name}",
        headers=headers,
    )
    if resp.status_code == 200:
        return resp.json()
    else:
        return None
    
def set_list_array(list_name: str, value) -> Dict[str, Any]:
    """Set list value"""
    headers = {
        "X-API-Key": secauto_api_key,
        "Content-Type": "application/json"
    }
    if isinstance(value, list): 
        newvalue = {"items": value}
    else:
        newvalue = {"items": [value]}
    resp = requests.post(
        f"{secauto_url}/list/{list_name}",
        headers=headers,
        json=newvalue
    )
    if resp.status_code == 200:
        return resp.json()
    else:
        return None
    
def set_list_json(list_name: str, value: dict) -> Dict[str, Any]:
    """Set list value"""
    headers = {
        "X-API-Key": secauto_api_key,
        "Content-Type": "application/json"
    }
    newvalue = {"items": [value]}
    resp = requests.post(
        f"{secauto_url}/list/{list_name}",
        headers=headers,
        json=newvalue
    )
    if resp.status_code == 200:
        return resp.json()
    else:
        return None
    
def delete_list(list_name: str) -> Dict[str, Any]:
    """Delete list value"""
    headers = {
        "X-API-Key": secauto_api_key,
        "Content-Type": "application/json"
    }
    resp = requests.delete(
        f"{secauto_url}/list/{list_name}",
        headers=headers,
    )
    if resp.status_code == 200:
        return resp.json()
    else:
        return None
    
def get_list_item(list_name: str, item_name: str) -> Dict[str, Any]:
    """Get list item value"""
    headers = {
        "X-API-Key": secauto_api_key,
        "Content-Type": "application/json"
    }
    resp = requests.get(
        f"{secauto_url}/list/{list_name}/{item_name}",
        headers=headers,
    )   
    if resp.status_code == 200:
        return resp.json()
    else:
        return None

def get_client_context() -> str:
    """
    Get the current client ID from the execution context
    
    Returns:
        Client ID string or None if not in client context
    """
    # Check environment variable first (set by playbook execution)
    import os
    client_id = os.environ.get('SECAUTO_CLIENT_ID')
    if client_id:
        return client_id
    
    # Check global context for client information
    try:
        context = load_context()
        if context and isinstance(context, dict):
            # Look for client_id in various context locations
            client_id = context.get('client_id')
            if client_id:
                return client_id
                
            # Check nested contexts
            if 'metadata' in context:
                client_id = context['metadata'].get('client_id')
                if client_id:
                    return client_id
                    
            if 'execution_context' in context:
                client_id = context['execution_context'].get('client_id')
                if client_id:
                    return client_id
    except:
        pass
    
    return None

def get_client_integration_config(integration_name: str) -> dict:
    """
    Get integration config with automatic client context detection
    
    Args:
        integration_name: Name of the integration
        
    Returns:
        Integration configuration dictionary or None
    """
    client_id = get_client_context()
    return get_integration_config(integration_name, client_id)

# Integration function support
def use_integration(integration_name: str, function_name: str, **kwargs):
    """
    Call an integration function with automatic dependency loading
    
    Args:
        integration_name: Name of the integration
        function_name: Name of the function to call
        **kwargs: Arguments to pass to the function
        
    Returns:
        Function result or error dictionary
        
    Example:
        from server.SoarBaseAPI import use_integration
        result = use_integration('qualys', 'scan_hosts', hosts=['10.0.0.1'])
    """
    try:
        from server.integration_loader import use_integration as _use_integration
        return _use_integration(integration_name, function_name, **kwargs)
    except ImportError:
        # Fallback if integration_loader not available
        return {
            "success": False,
            "error": "Integration loader not available"
        }

def list_integration_functions(integration_name: str):
    """
    List available functions in an integration
    
    Args:
        integration_name: Name of the integration
        
    Returns:
        Dictionary with function names and signatures
    """
    try:
        from server.integration_loader import list_integration_functions as _list_funcs
        return _list_funcs(integration_name)
    except ImportError:
        return {
            "success": False,
            "error": "Integration loader not available"
        }

def check_integration_available(integration_name: str) -> bool:
    """
    Check if an integration is available and built
    
    Args:
        integration_name: Name of the integration
        
    Returns:
        True if integration is available, False otherwise
    """
    try:
        from server.integration_loader import check_integration_available as _check_avail
        return _check_avail(integration_name)
    except ImportError:
        return False
    