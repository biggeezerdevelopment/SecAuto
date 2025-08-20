#!/usr/bin/env python3
"""
Integration Backend Builder
Builds integration-specific environments when configurations are uploaded
"""

import os
import sys
import json
import subprocess
import shutil
import hashlib
from pathlib import Path
from typing import Dict, List, Optional
import logging

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class IntegrationBackendBuilder:
    def __init__(self, base_path: str = None):
        """Initialize the builder with base paths"""
        self.base_path = Path(base_path or os.getcwd())
        self.integrations_dir = self.base_path / "integrations"
        self.venv_path = self.base_path / "Venv"
        self.site_packages_base = self.integrations_dir / ".site-packages"
        self.build_status_file = self.integrations_dir / ".build_status.json"
        
        # Create necessary directories
        self.site_packages_base.mkdir(parents=True, exist_ok=True)
        
    def build_integration(self, config_path: str) -> Dict:
        """
        Build integration backend from configuration
        
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
            
            logger.info(f"Building backend for integration: {integration_name}")
            
            # Check if build is required
            if not config.get('backend', {}).get('requires_build', False):
                logger.info(f"Integration {integration_name} does not require build")
                return {"success": True, "message": "No build required"}
            
            # Create integration-specific site-packages directory
            integration_site_packages = self.site_packages_base / integration_name
            integration_site_packages.mkdir(parents=True, exist_ok=True)
            
            # Build status tracking
            status = {
                "integration": integration_name,
                "version": config.get('version', '1.0.0'),
                "status": "building",
                "site_packages": str(integration_site_packages),
                "dependencies": []
            }
            
            # Install dependencies
            dependencies = config.get('dependencies', {})
            packages = dependencies.get('packages', [])
            
            if packages:
                logger.info(f"Installing {len(packages)} packages for {integration_name}")
                status['dependencies'] = self._install_packages(
                    packages, 
                    integration_site_packages,
                    integration_name
                )
            
            # Run post-install commands
            post_install = config.get('build', {}).get('post_install', [])
            for cmd in post_install:
                logger.info(f"Running post-install command: {cmd}")
                # Create integration directory if it doesn't exist
                integration_dir = self.integrations_dir / integration_name
                integration_dir.mkdir(parents=True, exist_ok=True)
                self._run_command(cmd, cwd=integration_dir)
            
            # Register with Python path
            self._register_site_packages(integration_name, integration_site_packages)
            
            # Update build status
            status['status'] = 'completed'
            self._update_build_status(integration_name, status)
            
            logger.info(f"Successfully built backend for {integration_name}")
            return {
                "success": True,
                "integration": integration_name,
                "site_packages": str(integration_site_packages),
                "dependencies_installed": len(status['dependencies'])
            }
            
        except Exception as e:
            logger.error(f"Failed to build integration backend: {e}")
            return {
                "success": False,
                "error": str(e)
            }
    
    def _install_packages(self, packages: List[str], target_dir: Path, integration_name: str) -> List[Dict]:
        """Install packages to integration-specific directory"""
        installed = []
        
        for package in packages:
            try:
                logger.info(f"Installing {package} for {integration_name}")
                
                # Use pip to install to specific directory
                cmd = [
                    sys.executable, "-m", "pip", "install",
                    package,
                    "--target", str(target_dir),
                    "--no-deps" if "==" in package else "--upgrade",
                    "--quiet"
                ]
                
                result = subprocess.run(
                    cmd,
                    capture_output=True,
                    text=True,
                    timeout=300
                )
                
                if result.returncode == 0:
                    installed.append({
                        "package": package,
                        "status": "installed",
                        "location": str(target_dir)
                    })
                    logger.info(f"Successfully installed {package}")
                else:
                    logger.warning(f"Failed to install {package}: {result.stderr}")
                    installed.append({
                        "package": package,
                        "status": "failed",
                        "error": result.stderr
                    })
                    
            except subprocess.TimeoutExpired:
                logger.error(f"Timeout installing {package}")
                installed.append({
                    "package": package,
                    "status": "timeout"
                })
            except Exception as e:
                logger.error(f"Error installing {package}: {e}")
                installed.append({
                    "package": package,
                    "status": "error",
                    "error": str(e)
                })
        
        # Install dependencies recursively
        self._install_dependencies(target_dir)
        
        return installed
    
    def _install_dependencies(self, target_dir: Path):
        """Install missing dependencies for packages in target directory"""
        try:
            # Get list of installed packages and their requirements
            cmd = [
                sys.executable, "-m", "pip", "check",
                "--target", str(target_dir)
            ]
            
            result = subprocess.run(cmd, capture_output=True, text=True)
            
            if result.returncode != 0 and "requires" in result.stdout:
                # Parse missing dependencies
                lines = result.stdout.split('\n')
                missing = set()
                
                for line in lines:
                    if "requires" in line and "which is not installed" in line:
                        # Extract package name
                        parts = line.split("requires")[1].split(",")[0].strip()
                        pkg_name = parts.split()[0]
                        missing.add(pkg_name)
                
                # Install missing dependencies
                for pkg in missing:
                    logger.info(f"Installing missing dependency: {pkg}")
                    subprocess.run([
                        sys.executable, "-m", "pip", "install",
                        pkg,
                        "--target", str(target_dir),
                        "--quiet"
                    ])
                    
        except Exception as e:
            logger.warning(f"Could not check/install dependencies: {e}")
    
    def _register_site_packages(self, integration_name: str, site_packages_path: Path):
        """Register integration site-packages with Python path"""
        try:
            # Create .pth file in main venv site-packages
            main_site_packages = self._get_main_site_packages()
            if not main_site_packages:
                logger.warning("Could not find main site-packages directory")
                return
            
            pth_file = main_site_packages / f"integration_{integration_name}.pth"
            
            # Write absolute path to .pth file
            with open(pth_file, 'w') as f:
                f.write(str(site_packages_path.absolute()) + '\n')
            
            logger.info(f"Registered {integration_name} site-packages at {pth_file}")
            
            # Also update sitecustomize.py for dynamic loading
            self._update_sitecustomize(integration_name, site_packages_path)
            
        except Exception as e:
            logger.error(f"Failed to register site-packages: {e}")
    
    def _get_main_site_packages(self) -> Optional[Path]:
        """Get the main venv site-packages directory"""
        try:
            # Try to find site-packages in venv
            venv_site_packages = None
            
            # Check common locations
            possible_paths = [
                self.venv_path / "lib" / f"python{sys.version_info.major}.{sys.version_info.minor}" / "site-packages",
                self.venv_path / "Lib" / "site-packages",  # Windows
                self.venv_path / "lib" / "python3" / "site-packages",
            ]
            
            for path in possible_paths:
                if path.exists():
                    venv_site_packages = path
                    break
            
            if not venv_site_packages:
                # Use current Python's site-packages as fallback
                import site
                site_packages = site.getsitepackages()
                if site_packages:
                    venv_site_packages = Path(site_packages[0])
            
            return venv_site_packages
            
        except Exception as e:
            logger.error(f"Could not find site-packages: {e}")
            return None
    
    def _update_sitecustomize(self, integration_name: str, site_packages_path: Path):
        """Update sitecustomize.py for dynamic integration loading"""
        try:
            main_site_packages = self._get_main_site_packages()
            if not main_site_packages:
                return
            
            sitecustomize_path = main_site_packages / "sitecustomize.py"
            
            # Read existing content or create new
            existing_content = ""
            if sitecustomize_path.exists():
                with open(sitecustomize_path, 'r') as f:
                    existing_content = f.read()
            
            # Check if our integration loader is already present
            if "SECAUTO_INTEGRATION_LOADER" not in existing_content:
                # Add our integration loader code
                loader_code = '''
# SECAUTO_INTEGRATION_LOADER - Dynamic integration package loading
import os
import sys
import json
from pathlib import Path

def load_integration_packages():
    """Dynamically load integration-specific packages based on context"""
    
    # Check for integration context
    integration = os.environ.get('SECAUTO_INTEGRATION')
    if not integration:
        # Check for PID-specific context file
        pid_file = f'/tmp/secauto_{os.getpid()}.integration'
        if os.path.exists(pid_file):
            try:
                with open(pid_file, 'r') as f:
                    integration = f.read().strip()
            except:
                pass
    
    if integration:
        # Load integration-specific site-packages
        integrations_base = Path(__file__).parent.parent.parent / "integrations" / ".site-packages"
        integration_path = integrations_base / integration
        
        if integration_path.exists() and str(integration_path) not in sys.path:
            sys.path.insert(0, str(integration_path))
            
            # Also check for build status to get all registered paths
            build_status_file = integrations_base.parent / ".build_status.json"
            if build_status_file.exists():
                try:
                    with open(build_status_file, 'r') as f:
                        status = json.load(f)
                        if integration in status:
                            site_packages = status[integration].get('site_packages')
                            if site_packages and os.path.exists(site_packages):
                                if site_packages not in sys.path:
                                    sys.path.insert(0, site_packages)
                except:
                    pass

# Auto-load on import
try:
    load_integration_packages()
except Exception:
    pass  # Fail silently to not break Python startup
'''
                
                # Append to existing content
                with open(sitecustomize_path, 'a') as f:
                    f.write('\n' + loader_code)
                
                logger.info(f"Updated sitecustomize.py with integration loader")
            
        except Exception as e:
            logger.warning(f"Could not update sitecustomize.py: {e}")
    
    def _run_command(self, command: str, cwd: Path = None) -> subprocess.CompletedProcess:
        """Run a shell command"""
        return subprocess.run(
            command,
            shell=True,
            cwd=cwd or self.base_path,
            capture_output=True,
            text=True,
            timeout=60
        )
    
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
        """Clean/remove integration backend"""
        try:
            logger.info(f"Cleaning backend for integration: {integration_name}")
            
            # Remove site-packages directory
            integration_site_packages = self.site_packages_base / integration_name
            if integration_site_packages.exists():
                shutil.rmtree(integration_site_packages)
            
            # Remove .pth file
            main_site_packages = self._get_main_site_packages()
            if main_site_packages:
                pth_file = main_site_packages / f"integration_{integration_name}.pth"
                if pth_file.exists():
                    pth_file.unlink()
            
            # Update build status
            all_status = self.get_build_status()
            if integration_name in all_status:
                del all_status[integration_name]
                with open(self.build_status_file, 'w') as f:
                    json.dump(all_status, f, indent=2)
            
            logger.info(f"Successfully cleaned backend for {integration_name}")
            return True
            
        except Exception as e:
            logger.error(f"Failed to clean integration backend: {e}")
            return False


def main():
    """Main entry point for CLI usage"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Build integration backends")
    parser.add_argument("action", choices=["build", "clean", "status"],
                       help="Action to perform")
    parser.add_argument("--config", help="Path to integration config (for build)")
    parser.add_argument("--name", help="Integration name (for clean/status)")
    parser.add_argument("--base-path", help="Base path for SecAuto", 
                       default="/Volumes/My Shared Files/Home/Downloads/SecAuto")
    
    args = parser.parse_args()
    
    builder = IntegrationBackendBuilder(args.base_path)
    
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


if __name__ == "__main__":
    main()