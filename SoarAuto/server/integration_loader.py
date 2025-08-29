#!/usr/bin/env python3
"""
SecAuto UV-based Integration Loader
Provides integration execution using UV for fast, isolated environments
"""

import json
import subprocess
import os
import tempfile
import shutil
from pathlib import Path
from typing import Dict, List, Optional, Any, Tuple
import logging

logger = logging.getLogger(__name__)


class IntegrationLoader:
    """Integration loader for fast, isolated execution"""
    
    def __init__(self, base_path: str = None):
        """Initialize the integration loader"""
        self.base_path = Path(base_path or os.getcwd())
        self.integrations_dir = self.base_path / "data" / "integrations"
        self.scripts_dir = self.integrations_dir / "scripts"
        self.venvs_dir = self.integrations_dir / "venvs"  # Virtual environments
        self.server_dir = self.base_path / "server"
        
        # Create necessary directories
        self.integrations_dir.mkdir(parents=True, exist_ok=True)
        self.scripts_dir.mkdir(parents=True, exist_ok=True)
        self.venvs_dir.mkdir(parents=True, exist_ok=True)
        
        # Check for UV availability
        self._check_uv()
    
    def _check_uv(self):
        """Check if UV is available"""
        try:
            result = subprocess.run(['uv', '--version'], 
                                  capture_output=True, text=True, check=True)
            logger.info(f"UV available: {result.stdout.strip()}")
        except (subprocess.CalledProcessError, FileNotFoundError):
            raise RuntimeError("UV is required but not found. Install with: curl -LsSf https://astral.sh/uv/install.sh | sh")
    
    def create_integration_environment(self, integration_name: str, requirements: List[str]) -> str:
        """
        Create an isolated UV virtual environment for an integration
        
        Args:
            integration_name: Name of the integration
            requirements: List of Python packages to install
            
        Returns:
            Path to the created virtual environment
        """
        venv_path = self.venvs_dir / integration_name
        
        # Remove existing environment if it exists
        if venv_path.exists():
            logger.info(f"Removing existing environment: {venv_path}")
            shutil.rmtree(venv_path)
        
        logger.info(f"Creating UV virtual environment for {integration_name}")
        
        # Create UV virtual environment
        subprocess.run([
            'uv', 'venv', str(venv_path)
        ], check=True, capture_output=True)
        
        # Install SoarBaseAPI as editable package first
        if self._setup_base_api_package():
            logger.info("Installing SoarBaseAPI to integration environment")
            subprocess.run([
                'uv', 'pip', 'install',
                '--python', str(venv_path / 'bin' / 'python'),
                '-e', str(self.server_dir)
            ], check=True, capture_output=True)
        
        # Install requirements if provided
        if requirements:
            logger.info(f"Installing {len(requirements)} packages with UV")
            
            # Create temporary requirements file
            with tempfile.NamedTemporaryFile(mode='w', suffix='.txt', delete=False) as f:
                f.write('\n'.join(requirements))
                temp_requirements = f.name
            
            try:
                # Install with UV (much faster than pip)
                subprocess.run([
                    'uv', 'pip', 'install',
                    '--python', str(venv_path / 'bin' / 'python'),
                    '-r', temp_requirements
                ], check=True, capture_output=True)
                
                logger.info(f"Successfully installed packages for {integration_name}")
            finally:
                os.unlink(temp_requirements)
        
        # Create activation helper script
        self._create_activation_script(integration_name, venv_path)
        
        logger.info(f"Integration environment created: {venv_path}")
        return str(venv_path)
    
    def _setup_base_api_package(self) -> bool:
        """Setup SoarBaseAPI as an installable package"""
        pyproject_path = self.server_dir / "pyproject.toml"
        
        if pyproject_path.exists():
            return True
        
        # Create pyproject.toml for SoarBaseAPI
        pyproject_content = '''[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "secauto-base-api"
version = "1.0.0"
description = "SecAuto Base API for automation scripts"
dependencies = []

[project.optional-dependencies]
dev = ["pytest", "black", "ruff"]

[tool.hatch.build.targets.wheel]
packages = ["SoarBaseAPI.py"]
'''
        
        try:
            with open(pyproject_path, 'w') as f:
                f.write(pyproject_content)
            logger.info("Created pyproject.toml for SoarBaseAPI")
            return True
        except Exception as e:
            logger.warning(f"Could not create pyproject.toml: {e}")
            return False
    
    def _create_activation_script(self, integration_name: str, venv_path: Path):
        """Create a convenient activation script for the integration"""
        activation_script = venv_path / "activate_integration.sh"
        
        script_content = f'''#!/bin/bash
# Activation script for {integration_name} integration

echo "Activating {integration_name} integration environment..."
source "{venv_path}/bin/activate"

# Set integration-specific environment variables
export INTEGRATION_NAME="{integration_name}"
export SECAUTO_ROOT="{self.base_path}"

echo "Integration environment activated."
echo "Python: $(which python)"
echo "Integration: {integration_name}"
'''
        
        with open(activation_script, 'w') as f:
            f.write(script_content)
        
        os.chmod(activation_script, 0o755)
        logger.info(f"Created activation script: {activation_script}")
    
    def use_integration(self, integration_name: str, function: str, 
                       client_id: Optional[str] = None, config: Dict = None,**kwargs) -> Dict[str, Any]:
        """
        Execute an integration function using UV
        
        Args:
            integration_name: Name of the integration to use
            function: Function name to execute
            client_id: Client ID for multi-tenant support
            **kwargs: Function parameters
            
        Returns:
            Dictionary with execution result
        """
        # Prepare the integration execution context
        context = {
            'function': function,
            'params': kwargs,
            'client_id': client_id,
            'config': config
        }
         
        # Get integration script path
        script_path = self.scripts_dir / f"{integration_name}_integration.py"
        
        # Check if script exists
        if not script_path.exists():
            return {
                'success': False,
                'error': f'Integration script not found: {script_path}'
            }
        
        # Get virtual environment path
        venv_path = self.venvs_dir / integration_name
        python_path = venv_path / 'bin' / 'python'
        
        # Check if virtual environment exists
        if not python_path.exists():
            return {
                'success': False,
                'error': f'Integration environment not found for: {integration_name}. Run setup first.'
            }
        
        try:
            # Execute with UV run for optimal performance and isolation
            result = subprocess.run([
                'uv', 'run',
                '--python', str(python_path),
                str(script_path)
            ], 
            input=json.dumps(context),
            text=True,
            capture_output=True,
            timeout=30,
            env={
                **os.environ,
                'INTEGRATION_NAME': integration_name,
                'SECAUTO_ROOT': str(self.base_path)
            })
            
            if result.returncode != 0:
                return {
                    'success': False,
                    'error': f'Integration execution failed: {result.stderr}'
                }
            
            # Parse the result
            if result.stdout.strip():
                return json.loads(result.stdout)
            else:
                return {
                    'success': True,
                    'result': None
                }
                
        except subprocess.TimeoutExpired:
            return {
                'success': False,
                'error': 'Integration execution timed out'
            }
        except json.JSONDecodeError:
            return {
                'success': False,
                'error': 'Invalid JSON response from integration'
            }
        except Exception as e:
            return {
                'success': False,
                'error': f'Integration execution error: {str(e)}'
            }
    
    def list_integrations(self) -> List[str]:
        """List all available integrations"""
        integrations = []
        
        # Look for integration scripts
        for script_file in self.scripts_dir.glob("*_integration.py"):
            integration_name = script_file.stem.replace("_integration", "")
            integrations.append(integration_name)
        
        return sorted(integrations)
    
    def get_integration_info(self, integration_name: str) -> Dict[str, Any]:
        """Get information about an integration"""
        venv_path = self.venvs_dir / integration_name
        script_path = self.scripts_dir / f"{integration_name}_integration.py"
        
        info = {
            'name': integration_name,
            'script_exists': script_path.exists(),
            'environment_exists': venv_path.exists(),
            'script_path': str(script_path),
            'venv_path': str(venv_path)
        }
        
        # Get installed packages if environment exists
        if venv_path.exists():
            try:
                result = subprocess.run([
                    'uv', 'pip', 'list',
                    '--python', str(venv_path / 'bin' / 'python'),
                    '--format', 'json'
                ], capture_output=True, text=True, check=True)
                
                packages = json.loads(result.stdout)
                info['installed_packages'] = [
                    f"{pkg['name']}=={pkg['version']}" 
                    for pkg in packages
                ]
            except Exception as e:
                info['installed_packages'] = f"Error getting packages: {e}"
        
        return info
    
    def clean_integration(self, integration_name: str) -> bool:
        """Clean/remove integration environment"""
        venv_path = self.venvs_dir / integration_name
        
        try:
            if venv_path.exists():
                shutil.rmtree(venv_path)
                logger.info(f"Removed integration environment: {integration_name}")
            
            return True
            
        except Exception as e:
            logger.error(f"Failed to clean integration {integration_name}: {e}")
            return False
    
    def rebuild_integration(self, integration_name: str, requirements: List[str]) -> str:
        """Rebuild an integration environment"""
        logger.info(f"Rebuilding integration environment: {integration_name}")
        
        # Clean existing environment
        self.clean_integration(integration_name)
        
        # Create new environment
        return self.create_integration_environment(integration_name, requirements)
    
    def _get_client_integration_config(self, integration_name: str, client_id: str) -> Tuple[Dict, Dict]:
        """
        Get client integration configuration from the Go server API
        
        Returns:
            Tuple of (config, credentials)
        """
        import requests
        
        # Determine server URL - try common ports
        server_urls = [
            "http://localhost:8080",
            "http://localhost:9090", 
            "http://localhost:3000"
        ]
        
        for base_url in server_urls:
            try:
                config_url = f"{base_url}/clients/{client_id}/integrations/{integration_name}/config"
                
                # Make request with a short timeout
                response = requests.get(config_url, timeout=2)
                
                if response.status_code == 200:
                    data = response.json()
                    
                    # Extract config from response structure
                    if 'config' in data and 'config' in data['config']:
                        config_data = data['config']['config']
                        credentials_data = data['config'].get('credentials', {}) or {}
                        
                        logger.info(f"Retrieved config for {integration_name} from {base_url}")
                        return config_data, credentials_data
                        
            except requests.RequestException:
                # Try next URL
                continue
                
        # If we get here, all servers failed
        raise Exception(f"Could not retrieve config from any server for {integration_name}/{client_id}")
    


# Backward compatibility function
def use_integration(integration_name: str, function: str, 
                   client_id: Optional[str] = None, **kwargs) -> Dict[str, Any]:
    """
    Backward compatibility function for existing code
    """
    loader = IntegrationLoader()
    return loader.use_integration(integration_name, function, client_id, **kwargs)


if __name__ == "__main__":
    # CLI interface for testing
    import argparse
    
    parser = argparse.ArgumentParser(description="UV Integration Loader")
    parser.add_argument("action", choices=["list", "info", "clean", "test"],
                       help="Action to perform")
    parser.add_argument("--integration", help="Integration name")
    parser.add_argument("--function", help="Function to test")
    
    args = parser.parse_args()
    
    loader = IntegrationLoader()
    
    if args.action == "list":
        integrations = loader.list_integrations()
        print("Available integrations:")
        for integration in integrations:
            print(f"  - {integration}")
    
    elif args.action == "info":
        if not args.integration:
            print("Error: --integration required for info action")
            exit(1)
        info = loader.get_integration_info(args.integration)
        print(json.dumps(info, indent=2))
    
    elif args.action == "clean":
        if not args.integration:
            print("Error: --integration required for clean action")
            exit(1)
        success = loader.clean_integration(args.integration)
        print(f"Clean {'successful' if success else 'failed'}")
    
    elif args.action == "test":
        if not args.integration or not args.function:
            print("Error: --integration and --function required for test action")
            exit(1)
        result = loader.use_integration(args.integration, args.function)
        print(json.dumps(result, indent=2))