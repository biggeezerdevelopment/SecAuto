# sitecustomize.py
import sys
import os
import time
from pathlib import Path
from importlib import reload
import urllib3

# Disable SSL warnings
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
urllib3.disable_warnings(urllib3.exceptions.NotOpenSSLWarning)
urllib3.disable_warnings(urllib3.exceptions.InsecurePlatformWarning)

# Get the workspace root directory (two levels up from site-packages)
workspace_root = Path(__file__).parent.parent.parent
workspace_root = Path(__file__).parent.parent.parent
server_path = f"{workspace_root}/server"
user_path = f"{workspace_root}/user"
if str(server_path) not in sys.path:    
    sys.path.append(server_path)
if str(user_path) not in sys.path:
    sys.path.append(user_path)

# Add server directory to Python path
if str(server_path) not in sys.path:
    sys.path.insert(0, str(server_path))
    print(f"Added {server_path} to sys.path")

def _log_message(message):
    """Log message to stderr to avoid interfering with stdout JSON output"""
    print(message, file=sys.stderr)

def _is_go_execution():
    """Check if this is being executed from Go"""
    # Check for common Go execution patterns
    if any(arg.startswith('-c') for arg in sys.argv):
        return True
    
    # Check if stdin has data (Go might pass JSON via stdin)
    try:
        if not sys.stdin.isatty():
            return True
    except:
        pass
    
    # Check for specific script names that Go might call
    if len(sys.argv) > 0:
        script_name = sys.argv[0].lower()
        if 'baseit.py' in script_name or 'automation' in script_name:
            return True
    
    return False

def reload_soar_api():
    """Reload SoarBaseAPI and update builtins"""
    try:
        # Remove from sys.modules to force reimport
        if 'SoarBaseAPI' in sys.modules:
            del sys.modules['SoarBaseAPI']
        
        # Reimport the module
        import SoarBaseAPI
        import builtins
        
        # Update builtins
        builtins.SoarBaseAPI = SoarBaseAPI
        builtins.search_context = SoarBaseAPI.search_context
        builtins.search_context_iterative = SoarBaseAPI.search_context_iterative
        builtins.search_context_path = SoarBaseAPI.search_context_path
        builtins.load_context = SoarBaseAPI.load_context
        builtins.base_context = SoarBaseAPI.base_context
        builtins.return_context = SoarBaseAPI.return_context
        builtins.get_secauto_config = SoarBaseAPI.get_secauto_config
        builtins.get_integration_config = SoarBaseAPI.get_integration_config
        builtins.get_cache = SoarBaseAPI.get_cache
        builtins.set_cache = SoarBaseAPI.set_cache
        builtins.delete_cache = SoarBaseAPI.delete_cache
        builtins.get_list = SoarBaseAPI.get_list
        builtins.set_list_array = SoarBaseAPI.set_list_array
        builtins.set_list_json = SoarBaseAPI.set_list_json
        builtins.delete_list = SoarBaseAPI.delete_list
        builtins.get_list_item = SoarBaseAPI.get_list_item
        builtins.get_client_context = SoarBaseAPI.get_client_context
        builtins.get_client_integration_config = SoarBaseAPI.get_client_integration_config

        #_log_message(f"SoarBaseAPI reloaded at {time.strftime('%H:%M:%S')}")
        return True
        
    except Exception as e:
        #_log_message(f"Failed to reload SoarBaseAPI: {e}")
        return False

def check_and_reload():
    """Check if SoarBaseAPI.py has been modified and reload if needed"""
    global last_modified_time
    
    # Define the path to SoarBaseAPI.py
    soar_api_path = Path(server_path) / "SoarBaseAPI.py"
    
    if not soar_api_path.exists():
        return
    
    current_mtime = soar_api_path.stat().st_mtime
    
    if 'last_modified_time' not in globals():
        globals()['last_modified_time'] = current_mtime
        return
    
    if current_mtime > last_modified_time:
        globals()['last_modified_time'] = current_mtime
        reload_soar_api()

# Initial import
try:
    import SoarBaseAPI
    
    import builtins
    import json
    
    # Make it available globally
    builtins.SoarBaseAPI = SoarBaseAPI
    builtins.search_context = SoarBaseAPI.search_context
    builtins.search_context_iterative = SoarBaseAPI.search_context_iterative
    builtins.search_context_path = SoarBaseAPI.search_context_path
    builtins.load_context = SoarBaseAPI.load_context
    builtins.return_context = SoarBaseAPI.return_context
    builtins.get_secauto_config = SoarBaseAPI.get_secauto_config
    builtins.get_integration_config = SoarBaseAPI.get_integration_config
    builtins.reload_soar_api = reload_soar_api
    builtins.check_and_reload = check_and_reload
    builtins.get_cache = SoarBaseAPI.get_cache
    builtins.set_cache = SoarBaseAPI.set_cache
    builtins.delete_cache = SoarBaseAPI.delete_cache
    builtins.get_list = SoarBaseAPI.get_list
    builtins.set_list_array = SoarBaseAPI.set_list_array
    builtins.set_list_json = SoarBaseAPI.set_list_json
    builtins.delete_list = SoarBaseAPI.delete_list
    builtins.get_list_item = SoarBaseAPI.get_list_item
    builtins.get_client_context = SoarBaseAPI.get_client_context
    builtins.get_client_integration_config = SoarBaseAPI.get_client_integration_config
    
    # Force context loading in every automation script
    def _ensure_context_loaded():
        """Ensure context is loaded in every automation script"""
        try:
            context = SoarBaseAPI.load_context()
            if context is None:
                context = json.loads(SoarBaseAPI.base_context())
        except Exception as e:
            context = json.loads(SoarBaseAPI.base_context())
        return context
    secauto_url, secauto_api_key = get_secauto_config()
    global_context = _ensure_context_loaded()
    builtins.context = global_context
    builtins.secauto_url = secauto_url
    builtins.secauto_api_key = secauto_api_key
except ImportError as e:
    _log_message(f"Warning: Could not import SoarBaseAPI from {server_path}: {e}")

# Integration Backend Loader - Dynamic integration package loading
def load_integration_packages():
    """Dynamically load integration-specific packages based on context"""
    
    # Check for integration context via environment variable
    integration = os.environ.get('SECAUTO_INTEGRATION')
    
    if not integration:
        # Check for PID-specific context file (fallback)
        pid_file = f'/tmp/secauto_{os.getpid()}.integration'
        if os.path.exists(pid_file):
            try:
                with open(pid_file, 'r') as f:
                    integration = f.read().strip()
                # Clean up the file after reading
                os.remove(pid_file)
            except:
                pass
    
    if integration:
        # Get the base path for integrations
        # This script is in Venv/lib/python3.9/site-packages/
        venv_path = Path(__file__).parent.parent.parent.parent  # Up to Venv/
        base_path = venv_path.parent  # SecAuto directory
        integrations_base = base_path / "integrations" / ".site-packages"
        integration_path = integrations_base / integration
        
        # Add integration-specific site-packages to path if it exists
        if integration_path.exists() and str(integration_path) not in sys.path:
            sys.path.insert(0, str(integration_path))
            
            # Also check for build status to get all registered paths
            build_status_file = integrations_base.parent / ".build_status.json"
            if build_status_file.exists():
                try:
                    import json
                    with open(build_status_file, 'r') as f:
                        status = json.load(f)
                        if integration in status:
                            site_packages = status[integration].get('site_packages')
                            if site_packages and os.path.exists(site_packages):
                                if site_packages not in sys.path:
                                    sys.path.insert(0, site_packages)
                except:
                    pass
    
    # Also check for general integration paths from .pth files
    # This allows integrations to work even without SECAUTO_INTEGRATION set
    try:
        site_packages_dir = Path(__file__).parent
        for pth_file in site_packages_dir.glob("integration_*.pth"):
            try:
                with open(pth_file, 'r') as f:
                    path = f.read().strip()
                    if path and os.path.exists(path) and path not in sys.path:
                        # Add at the end for lower priority than specific integration
                        sys.path.append(path)
            except:
                pass
    except:
        pass

# Load integration packages on startup
try:
    load_integration_packages()
    
    # Add integration functions to builtins if available
    try:
        import builtins
        if hasattr(SoarBaseAPI, 'use_integration'):
            builtins.use_integration = SoarBaseAPI.use_integration
        if hasattr(SoarBaseAPI, 'list_integration_functions'):
            builtins.list_integration_functions = SoarBaseAPI.list_integration_functions
        if hasattr(SoarBaseAPI, 'check_integration_available'):
            builtins.check_integration_available = SoarBaseAPI.check_integration_available
    except:
        pass
        
except Exception:
    # Fail silently to not break Python startup
    pass

# Set a flag to indicate integration loader has been added
os.environ['SECAUTO_INTEGRATION_LOADER'] = '1'
