# sitecustomize.py
import sys
import os
import time
from pathlib import Path
from importlib import reload

# Get the workspace root directory (two levels up from site-packages)
workspace_root = Path(__file__).parent.parent.parent.parent
server_path = workspace_root / "server"
soar_api_path = server_path / "SoarBaseAPI.py"


# Add server directory to Python path
if str(server_path) not in sys.path:
    sys.path.insert(0, str(server_path))

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
        #_log_message(f"SoarBaseAPI reloaded at {time.strftime('%H:%M:%S')}")
        return True
        
    except Exception as e:
        #_log_message(f"Failed to reload SoarBaseAPI: {e}")
        return False

def check_and_reload():
    """Check if SoarBaseAPI.py has been modified and reload if needed"""
    global last_modified_time
    
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
