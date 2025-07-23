import os
import json
import sys
import requests

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


def get_secauto_config() -> tuple[str, str]:
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

def get_integration_config(integration_name: str) -> dict:         
    # Get configuration
    secauto_url, secauto_api_key = get_secauto_config()
    
    if not secauto_api_key:
        return None
        
    headers = {
        "X-API-Key": secauto_api_key,
        "Content-Type": "application/json"
    }
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
                
        