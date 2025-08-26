#!/usr/bin/env python3
"""
UV-based Integration Backend Builder
Simplified, fast integration environment setup using UV

Configuration:
- Reads integration paths from config.yaml in the SoarAuto directory
- Supports configurable base_path, configs_path, scripts_path, and venvs_path
- Falls back to defaults if config.yaml is not found or missing entries

Usage examples:
  python integration_builder.py list
  python integration_builder.py build --config path/to/integration.json
  python integration_builder.py --config-file /custom/config.yaml status
"""

import os
import sys
import json
import subprocess
import shutil
import yaml
from pathlib import Path
from typing import Dict, List, Optional
import logging

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


class IntegrationBuilder:
    """Integration builder for fast environment setup"""
    
    def __init__(self, base_path: str = None, config_file: str = None):
        """Initialize the integration builder with base paths"""
        self.base_path = Path(base_path or os.getcwd())
        
        # Load configuration from config.yaml
        self.config = self._load_config(config_file)
        
        # Setup paths from configuration
        self._setup_paths_from_config()
        
        self.server_dir = self.base_path / "SoarAuto" / "server"
        self.build_status_file = self.integrations_dir / ".uv_build_status.json"
        
        # Create necessary directories
        self.venvs_dir.mkdir(parents=True, exist_ok=True)
        self.scripts_dir.mkdir(parents=True, exist_ok=True)
        self.configs_dir.mkdir(parents=True, exist_ok=True)
        
        # Check UV availability
        self._check_uv()
    
    def _check_uv(self):
        """Check if UV is available"""
        try:
            result = subprocess.run(['uv', '--version'], 
                                  capture_output=True, text=True, check=True)
            logger.info(f"UV available: {result.stdout.strip()}")
        except (subprocess.CalledProcessError, FileNotFoundError):
            raise RuntimeError("UV is required but not found. Install with: curl -LsSf https://astral.sh/uv/install.sh | sh")
    
    def _load_config(self, config_file: str = None) -> Dict:
        """Load configuration from config.yaml"""
        if config_file is None:
            config_file = self.base_path / "SoarAuto" / "config.yaml"
        else:
            config_file = Path(config_file)
        
        try:
            with open(config_file, 'r') as f:
                config = yaml.safe_load(f)
            logger.info(f"Loaded configuration from {config_file}")
            return config
        except FileNotFoundError:
            logger.warning(f"Config file not found: {config_file}, using defaults")
            return {}
        except Exception as e:
            logger.error(f"Failed to load config file {config_file}: {e}")
            return {}
    
    def _setup_paths_from_config(self):
        """Setup integration paths from configuration"""
        integrations_config = self.config.get('integrations', {})
        
        # Get base integration directory path
        base_integrations_path = integrations_config.get('base_path', 'data/integrations')
        
        # Convert relative paths to absolute paths based on base_path
        if not os.path.isabs(base_integrations_path):
            self.integrations_dir = self.base_path / "SoarAuto" / base_integrations_path
        else:
            self.integrations_dir = Path(base_integrations_path)
        
        # Setup specific directories from config or use defaults
        configs_path = integrations_config.get('configs_path', 'configs')
        scripts_path = integrations_config.get('scripts_path', 'scripts')
        venvs_path = integrations_config.get('venvs_path', 'venvs')
        
        # Convert to absolute paths
        if not os.path.isabs(configs_path):
            self.configs_dir = self.integrations_dir / configs_path.replace('data/integrations/', '')
        else:
            self.configs_dir = Path(configs_path)
            
        if not os.path.isabs(scripts_path):
            self.scripts_dir = self.integrations_dir / scripts_path.replace('data/integrations/', '')
        else:
            self.scripts_dir = Path(scripts_path)
            
        if not os.path.isabs(venvs_path):
            self.venvs_dir = self.integrations_dir / venvs_path
        else:
            self.venvs_dir = Path(venvs_path)
        
        logger.info(f"Integration paths - Base: {self.integrations_dir}, "
                   f"Configs: {self.configs_dir}, Scripts: {self.scripts_dir}, "
                   f"Venvs: {self.venvs_dir}")
    
    def build_integration(self, config_path: str) -> Dict:
        """
        Build integration environment using UV from configuration
        
        Args:
            config_path: Path to integration configuration JSON
            
        Returns:
            Build status dictionary
        """
        try:
            # Load configuration
            with open(config_path, 'r') as f:
                config = json.load(f)
            
            integration_name = config.get('name')
            if not integration_name:
                raise ValueError("Integration configuration missing 'name' field")
            
            logger.info(f"Building UV environment for integration: {integration_name}")
            
            # Check if build is required
            if not config.get('backend', {}).get('requires_build', False):
                logger.info(f"Integration {integration_name} does not require build")
                return {"success": True, "message": "No build required"}
            
            # Get packages to install
            dependencies = config.get('dependencies', {})
            packages = dependencies.get('packages', [])
            
            # Build with UV
            venv_path = self._create_uv_environment(integration_name, packages)
            
            # Run post-install commands if specified
            self._run_post_install_commands(config, integration_name, venv_path)
            
            # Update build status
            status = {
                "integration": integration_name,
                "version": config.get('version', '1.0.0'),
                "status": "completed",
                "venv_path": str(venv_path),
                "dependencies": packages,
                "uv_builder": True,
                "build_time": self._get_current_time()
            }
            
            self._update_build_status(integration_name, status)
            
            logger.info(f"Successfully built UV environment for {integration_name}")
            return {
                "success": True,
                "integration": integration_name,
                "venv_path": str(venv_path),
                "dependencies_installed": len(packages),
                "builder": "UV"
            }
            
        except Exception as e:
            logger.error(f"Failed to build integration environment: {e}")
            return {
                "success": False,
                "error": str(e)
            }
    
    def _create_uv_environment(self, integration_name: str, packages: List[str]) -> Path:
        """Create UV virtual environment and install packages"""
        venv_path = self.venvs_dir / integration_name
        
        # Remove existing environment
        if venv_path.exists():
            logger.info(f"Removing existing environment: {venv_path}")
            shutil.rmtree(venv_path)
        
        logger.info(f"Creating UV virtual environment: {venv_path}")
        
        # Create UV virtual environment
        subprocess.run([
            'uv', 'venv', str(venv_path)
        ], check=True, capture_output=True)
        
        # Setup SoarBaseAPI first
        self._setup_base_api(integration_name, venv_path)
        
        # Install packages if specified
        if packages:
            logger.info(f"Installing {len(packages)} packages with UV")
            self._install_packages_with_uv(venv_path, packages)
        
        return venv_path
    
    def _setup_base_api(self, integration_name: str, venv_path: Path):
        """Setup SoarBaseAPI for the integration"""
        # Create pyproject.toml if it doesn't exist
        pyproject_path = self.server_dir / "pyproject.toml"
        
        if not pyproject_path.exists():
            self._create_base_api_package()
        
        try:
            # Install SoarBaseAPI as editable package
            logger.info(f"Installing SoarBaseAPI for {integration_name}")
            subprocess.run([
                'uv', 'pip', 'install',
                '--python', str(venv_path / 'bin' / 'python'),
                '-e', str(self.server_dir)
            ], check=True, capture_output=True)
            
            logger.info("SoarBaseAPI installed successfully")
            
        except subprocess.CalledProcessError as e:
            logger.warning(f"Failed to install SoarBaseAPI, continuing without it: {e}")
        
        # Install sitecustomize.py for automatic builtin setup
        self._install_sitecustomize(integration_name, venv_path)
    
    def _create_base_api_package(self):
        """Create pyproject.toml for SoarBaseAPI"""
        pyproject_content = '''[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "secauto-base-api"
version = "1.0.0"
description = "SecAuto Base API for automation scripts"
dependencies = []

[project.scripts]
secauto-base = "SoarBaseAPI:main"

[tool.hatch.build.targets.wheel]
packages = ["."]
'''
        
        pyproject_path = self.server_dir / "pyproject.toml"
        with open(pyproject_path, 'w') as f:
            f.write(pyproject_content)
        
        # Also create __init__.py if it doesn't exist
        init_path = self.server_dir / "__init__.py"
        if not init_path.exists():
            init_path.write_text("# SecAuto Server Package\\n")
        
        logger.info("Created SoarBaseAPI package structure")
    
    def _install_sitecustomize(self, integration_name: str, venv_path: Path):
        """Install sitecustomize.py into the virtual environment"""
        try:
            # Find the site-packages directory in the venv
            site_packages_dirs = []
            lib_dir = venv_path / "lib"
            
            if lib_dir.exists():
                # Look for python3.x/site-packages
                for python_dir in lib_dir.iterdir():
                    if python_dir.is_dir() and python_dir.name.startswith("python"):
                        site_packages = python_dir / "site-packages"
                        if site_packages.exists():
                            site_packages_dirs.append(site_packages)
            
            # Also check Lib/site-packages (Windows)
            lib_dir_win = venv_path / "Lib"
            if lib_dir_win.exists():
                site_packages = lib_dir_win / "site-packages"
                if site_packages.exists():
                    site_packages_dirs.append(site_packages)
            
            if not site_packages_dirs:
                logger.warning(f"Could not find site-packages directory in {venv_path}")
                return
            
            # Use the first site-packages directory found
            target_site_packages = site_packages_dirs[0]
            
            # Copy sitecustomize.py from server directory
            sitecustomize_source = self.server_dir / "sitecustomize.py"
            sitecustomize_target = target_site_packages / "sitecustomize.py"
            
            if sitecustomize_source.exists():
                import shutil
                shutil.copy2(sitecustomize_source, sitecustomize_target)
                logger.info(f"Installed sitecustomize.py to {sitecustomize_target}")
            else:
                logger.warning(f"sitecustomize.py not found at {sitecustomize_source}")
            
            # Copy SoarBaseAPI.py so sitecustomize can import it
            soarbase_source = self.server_dir / "SoarBaseAPI.py"
            soarbase_target = target_site_packages / "SoarBaseAPI.py"
            
            if soarbase_source.exists():
                shutil.copy2(soarbase_source, soarbase_target)
                logger.info(f"Installed SoarBaseAPI.py to {soarbase_target}")
            else:
                logger.warning(f"SoarBaseAPI.py not found at {soarbase_source}")
            
            # Also copy secauto_base.py for alternative import method
            secauto_base_source = self.server_dir / "secauto_base.py"
            secauto_base_target = target_site_packages / "secauto_base.py"
            
            if secauto_base_source.exists():
                shutil.copy2(secauto_base_source, secauto_base_target)
                logger.info(f"Installed secauto_base.py to {secauto_base_target}")
            else:
                logger.warning(f"secauto_base.py not found at {secauto_base_source}")
                
        except Exception as e:
            logger.warning(f"Failed to install sitecustomize.py for {integration_name}: {e}")
    
    def _install_packages_with_uv(self, venv_path: Path, packages: List[str]):
        """Install packages using UV"""
        for package in packages:
            try:
                logger.info(f"Installing {package}")
                
                subprocess.run([
                    'uv', 'pip', 'install',
                    '--python', str(venv_path / 'bin' / 'python'),
                    package
                ], check=True, capture_output=True)
                
                logger.info(f"Successfully installed {package}")
                
            except subprocess.CalledProcessError as e:
                # For critical packages like psycopg2, try alternatives
                if 'psycopg2' in package:
                    logger.warning(f"Failed to install {package}, trying psycopg2-binary")
                    try:
                        subprocess.run([
                            'uv', 'pip', 'install',
                            '--python', str(venv_path / 'bin' / 'python'),
                            'psycopg2-binary'
                        ], check=True, capture_output=True)
                        logger.info("Successfully installed psycopg2-binary as alternative")
                        continue
                    except subprocess.CalledProcessError:
                        pass
                
                logger.error(f"Failed to install {package}: {e}")
                raise
    
    def _run_post_install_commands(self, config: Dict, integration_name: str, venv_path: Path):
        """Run post-install commands"""
        post_install = config.get('build', {}).get('post_install', [])
        
        for cmd in post_install:
            logger.info(f"Running post-install command: {cmd}")
            
            # Use UV run to execute in the virtual environment
            try:
                subprocess.run([
                    'uv', 'run',
                    '--python', str(venv_path / 'bin' / 'python'),
                    '-c', cmd
                ], check=True, capture_output=True, text=True, 
                  cwd=str(venv_path))
                
                logger.info(f"Successfully executed: {cmd}")
                
            except subprocess.CalledProcessError as e:
                logger.warning(f"Post-install command failed: {cmd} - {e}")
    
    def _get_current_time(self) -> str:
        """Get current timestamp"""
        import datetime
        return datetime.datetime.now().isoformat()
    
    def _update_build_status(self, integration_name: str, status: Dict):
        """Update build status file"""
        try:
            # Load existing status
            all_status = {}
            if self.build_status_file.exists():
                with open(self.build_status_file, 'r') as f:
                    all_status = json.load(f)
            
            # Update with new status
            all_status[integration_name] = status
            
            # Save updated status
            with open(self.build_status_file, 'w') as f:
                json.dump(all_status, f, indent=2)
                
        except Exception as e:
            logger.error(f"Failed to update build status: {e}")
    
    def get_build_status(self, integration_name: str = None) -> Dict:
        """Get build status for integration(s)"""
        try:
            if not self.build_status_file.exists():
                return {}
            
            with open(self.build_status_file, 'r') as f:
                all_status = json.load(f)
            
            if integration_name:
                return all_status.get(integration_name, {})
            return all_status
            
        except Exception as e:
            logger.error(f"Failed to get build status: {e}")
            return {}
    
    def clean_integration(self, integration_name: str) -> bool:
        """Clean/remove integration environment"""
        try:
            logger.info(f"Cleaning UV environment for integration: {integration_name}")
            
            # Remove venv directory
            venv_path = self.venvs_dir / integration_name
            if venv_path.exists():
                shutil.rmtree(venv_path)
                logger.info(f"Removed venv: {venv_path}")
            
            # Update build status
            all_status = self.get_build_status()
            if integration_name in all_status:
                del all_status[integration_name]
                with open(self.build_status_file, 'w') as f:
                    json.dump(all_status, f, indent=2)
            
            logger.info(f"Successfully cleaned environment for {integration_name}")
            return True
            
        except Exception as e:
            logger.error(f"Failed to clean integration environment: {e}")
            return False
    
    def list_integrations(self) -> List[str]:
        """List all built integrations"""
        integrations = []
        
        if self.venvs_dir.exists():
            for venv_dir in self.venvs_dir.iterdir():
                if venv_dir.is_dir() and (venv_dir / 'bin' / 'python').exists():
                    integrations.append(venv_dir.name)
        
        return sorted(integrations)
    
    def migrate_from_old_system(self, integration_name: str) -> Dict:
        """Migrate an integration from the old .site-packages system to UV"""
        logger.info(f"Migrating {integration_name} from old system to UV")
        
        # Look for old config
        old_config_path = self.configs_dir / f"{integration_name}.json"
        if not old_config_path.exists():
            return {
                "success": False,
                "error": f"Config not found: {old_config_path}"
            }
        
        try:
            # Build with UV
            result = self.build_integration(str(old_config_path))
            
            if result.get("success"):
                # Clean up old .site-packages directory if it exists
                old_env_path = self.integrations_dir / integration_name / ".site-packages"
                if old_env_path.exists():
                    logger.info(f"Removing old .site-packages directory: {old_env_path}")
                    shutil.rmtree(old_env_path.parent)
                
                logger.info(f"Successfully migrated {integration_name} to UV")
                return {
                    "success": True,
                    "message": f"Migrated {integration_name} to UV",
                    "old_system_cleaned": old_env_path.exists()
                }
            else:
                return result
                
        except Exception as e:
            return {
                "success": False,
                "error": f"Migration failed: {e}"
            }


def main():
    """Main entry point for CLI usage"""
    import argparse
    
    parser = argparse.ArgumentParser(description="UV Integration Builder")
    parser.add_argument("action", choices=["build", "clean", "status", "list", "migrate"],
                       help="Action to perform")
    parser.add_argument("--config", help="Path to integration config (for build)")
    parser.add_argument("--name", help="Integration name (for clean/status/migrate)")
    parser.add_argument("--base-path", help="Base path for SecAuto", 
                       default="/Volumes/My Shared Files/Home/Downloads/SecAuto")
    parser.add_argument("--config-file", help="Path to config.yaml file")
    
    args = parser.parse_args()
    
    builder = IntegrationBuilder(args.base_path, args.config_file)
    
    if args.action == "build":
        if not args.config:
            print("Error: --config required for build action")
            sys.exit(1)
        result = builder.build_integration(args.config)
        print(json.dumps(result, indent=2))
        
    elif args.action == "clean":
        if not args.name:
            print("Error: --name required for clean action")
            sys.exit(1)
        success = builder.clean_integration(args.name)
        print(f"Clean {'successful' if success else 'failed'}")
        
    elif args.action == "status":
        status = builder.get_build_status(args.name)
        print(json.dumps(status, indent=2))
    
    elif args.action == "list":
        integrations = builder.list_integrations()
        print("Built integrations:")
        for integration in integrations:
            print(f"  - {integration}")
    
    elif args.action == "migrate":
        if not args.name:
            print("Error: --name required for migrate action")
            sys.exit(1)
        result = builder.migrate_from_old_system(args.name)
        print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()