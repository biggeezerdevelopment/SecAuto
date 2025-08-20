#!/usr/bin/env python3
"""
Integration Loader for Automations
Provides seamless access to integration functions with automatic dependency loading
"""

import os
import sys
import json
import importlib
import importlib.util
from pathlib import Path
from typing import Any, Dict, Optional, Callable
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class IntegrationLoader:
    """Loads and manages integration functions with their dependencies"""
    
    def __init__(self):
        self.base_path = Path(__file__).parent.parent
        self.integrations_dir = self.base_path / "integrations"
        self.site_packages_base = self.integrations_dir / ".site-packages"
        self.loaded_integrations = {}
        self.build_status = self._load_build_status()
        
    def _load_build_status(self) -> Dict:
        """Load integration build status"""
        try:
            status_file = self.integrations_dir / ".build_status.json"
            if status_file.exists():
                with open(status_file, 'r') as f:
                    return json.load(f)
        except Exception as e:
            logger.warning(f"Could not load build status: {e}")
        return {}
    
    def load_integration(self, integration_name: str) -> Optional[object]:
        """
        Load an integration with its dependencies
        
        Args:
            integration_name: Name of the integration to load
            
        Returns:
            Integration module object or None
        """
        try:
            # Check if already loaded
            if integration_name in self.loaded_integrations:
                return self.loaded_integrations[integration_name]
            
            # Check if integration is built
            if integration_name not in self.build_status:
                logger.error(f"Integration {integration_name} not built")
                return None
            
            # Get integration site-packages path
            integration_status = self.build_status[integration_name]
            site_packages = integration_status.get('site_packages')
            
            if not site_packages or not os.path.exists(site_packages):
                logger.error(f"Site-packages not found for {integration_name}")
                return None
            
            # Add integration site-packages to path (at beginning for priority)
            if site_packages not in sys.path:
                sys.path.insert(0, site_packages)
            
            # Load integration configuration
            config_path = self.integrations_dir / f"{integration_name}_config.json"
            if not config_path.exists():
                config_path = self.integrations_dir / integration_name / "config.json"
            
            if config_path.exists():
                with open(config_path, 'r') as f:
                    config = json.load(f)
                    entry_point = config.get('backend', {}).get('entry_point')
            else:
                # Default entry point
                entry_point = f"{integration_name}.py"
            
            # Load the integration module
            integration_file = self.integrations_dir / integration_name / entry_point
            if not integration_file.exists():
                integration_file = self.integrations_dir / entry_point
            
            if not integration_file.exists():
                logger.error(f"Integration file not found: {integration_file}")
                return None
            
            # Load module dynamically
            spec = importlib.util.spec_from_file_location(
                integration_name,
                integration_file
            )
            module = importlib.util.module_from_spec(spec)
            
            # Store in loaded integrations
            self.loaded_integrations[integration_name] = module
            
            # Execute the module
            spec.loader.exec_module(module)
            
            logger.info(f"Successfully loaded integration: {integration_name}")
            return module
            
        except Exception as e:
            logger.error(f"Failed to load integration {integration_name}: {e}")
            return None
    
    def call_function(self, integration_name: str, function_name: str, **kwargs) -> Any:
        """
        Call a function from an integration
        
        Args:
            integration_name: Name of the integration
            function_name: Name of the function to call
            **kwargs: Arguments to pass to the function
            
        Returns:
            Function result or error dictionary
        """
        try:
            # Set environment variable for integration context
            os.environ['SECAUTO_INTEGRATION'] = integration_name
            
            # Load the integration
            module = self.load_integration(integration_name)
            if not module:
                return {
                    "success": False,
                    "error": f"Could not load integration: {integration_name}"
                }
            
            # Get the function
            if not hasattr(module, function_name):
                return {
                    "success": False,
                    "error": f"Function {function_name} not found in {integration_name}"
                }
            
            func = getattr(module, function_name)
            
            # Call the function
            result = func(**kwargs)
            
            return {
                "success": True,
                "result": result
            }
            
        except Exception as e:
            logger.error(f"Error calling {integration_name}.{function_name}: {e}")
            return {
                "success": False,
                "error": str(e)
            }
        finally:
            # Clean up environment
            if 'SECAUTO_INTEGRATION' in os.environ:
                del os.environ['SECAUTO_INTEGRATION']
    
    def list_functions(self, integration_name: str) -> Dict[str, list]:
        """
        List available functions in an integration
        
        Args:
            integration_name: Name of the integration
            
        Returns:
            Dictionary with function names and their signatures
        """
        try:
            module = self.load_integration(integration_name)
            if not module:
                return {"success": False, "error": "Could not load integration"}
            
            functions = []
            for name in dir(module):
                if not name.startswith('_'):  # Skip private functions
                    attr = getattr(module, name)
                    if callable(attr):
                        # Get function signature if possible
                        try:
                            import inspect
                            sig = inspect.signature(attr)
                            functions.append({
                                "name": name,
                                "parameters": str(sig),
                                "doc": attr.__doc__ or "No documentation"
                            })
                        except:
                            functions.append({
                                "name": name,
                                "parameters": "Unknown",
                                "doc": attr.__doc__ or "No documentation"
                            })
            
            return {
                "success": True,
                "integration": integration_name,
                "functions": functions
            }
            
        except Exception as e:
            return {
                "success": False,
                "error": str(e)
            }
    
    def get_integration_status(self, integration_name: str = None) -> Dict:
        """Get status of integration(s)"""
        if integration_name:
            return self.build_status.get(integration_name, {
                "status": "not_found",
                "message": f"Integration {integration_name} not found"
            })
        return self.build_status


# Global loader instance
_loader = None

def get_loader() -> IntegrationLoader:
    """Get or create the global integration loader"""
    global _loader
    if _loader is None:
        _loader = IntegrationLoader()
    return _loader

# Convenience functions for automations
def use_integration(integration_name: str, function_name: str, **kwargs) -> Any:
    """
    Easy-to-use function for automations to call integration functions
    
    Example:
        from server.integration_loader import use_integration
        result = use_integration('qualys_integration', 'scan_hosts', 
                                hosts=['10.0.0.1'], scan_type='vulnerability')
    """
    loader = get_loader()
    return loader.call_function(integration_name, function_name, **kwargs)

def list_integration_functions(integration_name: str) -> Dict:
    """List available functions in an integration"""
    loader = get_loader()
    return loader.list_functions(integration_name)

def check_integration_available(integration_name: str) -> bool:
    """Check if an integration is available and built"""
    loader = get_loader()
    status = loader.get_integration_status(integration_name)
    return status.get('status') == 'completed'

# Import all available integrations functions (optional auto-import)
def import_all_integrations():
    """
    Import all available integration functions into the global namespace
    This allows using integration functions directly without specifying the integration
    
    Example:
        from server.integration_loader import import_all_integrations
        import_all_integrations()
        # Now you can use: scan_hosts() instead of use_integration('qualys', 'scan_hosts')
    """
    loader = get_loader()
    
    for integration_name in loader.build_status:
        if loader.build_status[integration_name].get('status') == 'completed':
            module = loader.load_integration(integration_name)
            if module:
                # Add all public functions to globals
                for name in dir(module):
                    if not name.startswith('_'):
                        attr = getattr(module, name)
                        if callable(attr):
                            # Prefix with integration name to avoid conflicts
                            globals()[f"{integration_name}_{name}"] = attr
                            # Also add without prefix if no conflict
                            if name not in globals():
                                globals()[name] = attr

if __name__ == "__main__":
    # Test the loader
    import argparse
    
    parser = argparse.ArgumentParser(description="Test integration loader")
    parser.add_argument("action", choices=["list", "call", "status"])
    parser.add_argument("--integration", help="Integration name")
    parser.add_argument("--function", help="Function name")
    parser.add_argument("--args", help="Function arguments as JSON")
    
    args = parser.parse_args()
    
    loader = get_loader()
    
    if args.action == "list":
        if args.integration:
            result = loader.list_functions(args.integration)
        else:
            result = {"integrations": list(loader.build_status.keys())}
        print(json.dumps(result, indent=2))
        
    elif args.action == "call":
        if not args.integration or not args.function:
            print("Error: --integration and --function required")
            sys.exit(1)
        
        kwargs = {}
        if args.args:
            kwargs = json.loads(args.args)
        
        result = loader.call_function(args.integration, args.function, **kwargs)
        print(json.dumps(result, indent=2))
        
    elif args.action == "status":
        result = loader.get_integration_status(args.integration)
        print(json.dumps(result, indent=2))