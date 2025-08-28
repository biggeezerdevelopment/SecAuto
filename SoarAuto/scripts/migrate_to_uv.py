#!/usr/bin/env python3
"""
Migration script to convert existing integrations to UV-based system
Helps transition from .site-packages to UV virtual environments
"""

import os
import sys
import json
import shutil
import subprocess
from pathlib import Path
from typing import Dict, List
import logging

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


class IntegrationMigrator:
    """Migrates integrations from old system to UV-based system"""
    
    def __init__(self, base_path: str = None):
        """Initialize migrator"""
        self.base_path = Path(base_path or os.getcwd())
        self.old_integrations_dir = self.base_path / "SoarAuto" / "data" / "integrations"
        self.new_venvs_dir = self.old_integrations_dir / "venvs"
        self.scripts_dir = self.old_integrations_dir / "scripts"
        self.configs_dir = self.old_integrations_dir / "configs"
        
        # Migration status file
        self.migration_status_file = self.old_integrations_dir / ".migration_status.json"
        
        self._ensure_directories()
    
    def _ensure_directories(self):
        """Ensure necessary directories exist"""
        self.new_venvs_dir.mkdir(parents=True, exist_ok=True)
        self.scripts_dir.mkdir(parents=True, exist_ok=True)
        self.configs_dir.mkdir(parents=True, exist_ok=True)
    
    def discover_old_integrations(self) -> List[str]:
        """Discover integrations using the old .site-packages system"""
        old_integrations = []
        
        if not self.old_integrations_dir.exists():
            logger.warning(f"Integrations directory not found: {self.old_integrations_dir}")
            return old_integrations
        
        # Look for directories with .site-packages subdirectories
        for item in self.old_integrations_dir.iterdir():
            if item.is_dir() and item.name not in ['venvs', 'scripts', 'configs']:
                site_packages_dir = item / ".site-packages"
                if site_packages_dir.exists():
                    old_integrations.append(item.name)
                    logger.info(f"Found old integration: {item.name}")
        
        return old_integrations
    
    def analyze_integration(self, integration_name: str) -> Dict:
        """Analyze an old integration to prepare for migration"""
        integration_dir = self.old_integrations_dir / integration_name
        site_packages_dir = integration_dir / ".site-packages"
        
        analysis = {
            "name": integration_name,
            "old_path": str(integration_dir),
            "site_packages_path": str(site_packages_dir),
            "has_config": False,
            "has_script": False,
            "installed_packages": [],
            "migration_ready": False
        }
        
        # Check for config file
        config_path = self.configs_dir / f"{integration_name}.json"
        if config_path.exists():
            analysis["has_config"] = True
            analysis["config_path"] = str(config_path)
        
        # Look for script files
        script_patterns = [
            f"{integration_name}_integration.py",
            f"{integration_name}.py",
            "integration.py"
        ]
        
        for pattern in script_patterns:
            script_path = self.scripts_dir / pattern
            if script_path.exists():
                analysis["has_script"] = True
                analysis["script_path"] = str(script_path)
                break
        
        # Try to discover installed packages
        if site_packages_dir.exists():
            packages = self._discover_packages_in_site_packages(site_packages_dir)
            analysis["installed_packages"] = packages
        
        # Determine if ready for migration
        analysis["migration_ready"] = analysis["has_config"] and analysis["has_script"]
        
        return analysis
    
    def _discover_packages_in_site_packages(self, site_packages_dir: Path) -> List[str]:
        """Try to discover what packages are installed in .site-packages"""
        packages = []
        
        try:
            # Look for .dist-info directories to identify packages
            for item in site_packages_dir.iterdir():
                if item.is_dir() and item.name.endswith('.dist-info'):
                    package_name = item.name.replace('.dist-info', '')
                    # Try to extract version from METADATA file
                    metadata_file = item / "METADATA"
                    version = "unknown"
                    if metadata_file.exists():
                        try:
                            with open(metadata_file, 'r') as f:
                                for line in f:
                                    if line.startswith('Version: '):
                                        version = line.replace('Version: ', '').strip()
                                        break
                        except:
                            pass
                    
                    packages.append(f"{package_name}=={version}" if version != "unknown" else package_name)
            
            # Also look for egg-info directories
            for item in site_packages_dir.iterdir():
                if item.is_dir() and item.name.endswith('.egg-info'):
                    package_name = item.name.replace('.egg-info', '')
                    if not any(pkg.startswith(package_name) for pkg in packages):
                        packages.append(package_name)
        
        except Exception as e:
            logger.warning(f"Failed to discover packages in {site_packages_dir}: {e}")
        
        return packages
    
    def migrate_integration(self, integration_name: str, force: bool = False) -> Dict:
        """Migrate a single integration to UV system"""
        logger.info(f"Starting migration of {integration_name}")
        
        # Analyze the integration first
        analysis = self.analyze_integration(integration_name)
        
        if not analysis["migration_ready"] and not force:
            return {
                "success": False,
                "error": f"Integration {integration_name} not ready for migration. Missing config or script.",
                "analysis": analysis
            }
        
        try:
            # Load existing config if available
            config = None
            if analysis["has_config"]:
                with open(analysis["config_path"], 'r') as f:
                    config = json.load(f)
            else:
                # Create a basic config
                config = self._create_basic_config(integration_name, analysis)
            
            # Update config for UV if needed
            config = self._update_config_for_uv(config, analysis)
            
            # Save updated config
            config_path = self.configs_dir / f"{integration_name}.json"
            with open(config_path, 'w') as f:
                json.dump(config, f, indent=2)
            
            # Use UV builder to create new environment
            builder_script = self.base_path / "scripts" / "integration_builder.py"
            
            if not builder_script.exists():
                return {
                    "success": False,
                    "error": f"UV builder script not found: {builder_script}"
                }
            
            logger.info(f"Building UV environment for {integration_name}")
            result = subprocess.run([
                sys.executable, str(builder_script), 
                "build", "--config", str(config_path)
            ], capture_output=True, text=True)
            
            if result.returncode != 0:
                return {
                    "success": False,
                    "error": f"UV build failed: {result.stderr}",
                    "stdout": result.stdout
                }
            
            # Parse build result
            try:
                build_result = json.loads(result.stdout)
                if not build_result.get("success"):
                    return {
                        "success": False,
                        "error": f"UV build reported failure: {build_result.get('error')}",
                        "build_result": build_result
                    }
            except json.JSONDecodeError:
                logger.warning("Could not parse UV builder output as JSON")
            
            # Clean up old .site-packages directory
            old_integration_dir = self.old_integrations_dir / integration_name
            if old_integration_dir.exists() and not force:
                # Backup before removal
                backup_dir = old_integration_dir.with_suffix('.backup')
                if backup_dir.exists():
                    shutil.rmtree(backup_dir)
                shutil.move(str(old_integration_dir), str(backup_dir))
                logger.info(f"Backed up old integration to: {backup_dir}")
            elif old_integration_dir.exists():
                shutil.rmtree(old_integration_dir)
                logger.info(f"Removed old integration directory: {old_integration_dir}")
            
            # Update migration status
            self._update_migration_status(integration_name, {
                "migrated": True,
                "migration_time": self._get_current_time(),
                "old_packages": analysis["installed_packages"],
                "config_updated": True
            })
            
            logger.info(f"Successfully migrated {integration_name} to UV")
            return {
                "success": True,
                "integration": integration_name,
                "config_path": str(config_path),
                "packages_migrated": len(analysis["installed_packages"]),
                "analysis": analysis
            }
        
        except Exception as e:
            logger.error(f"Migration failed for {integration_name}: {e}")
            return {
                "success": False,
                "error": str(e),
                "analysis": analysis
            }
    
    def _create_basic_config(self, integration_name: str, analysis: Dict) -> Dict:
        """Create a basic configuration for an integration"""
        return {
            "name": integration_name,
            "version": "1.0.0",
            "description": f"Migrated {integration_name} integration",
            "author": "SecAuto Migration",
            "dependencies": {
                "packages": analysis["installed_packages"],
                "system": [],
                "optional": []
            },
            "backend": {
                "type": "python",
                "entry_point": f"{integration_name}_integration.py",
                "timeout": 30,
                "memory_limit": 512,
                "requires_build": True
            },
            "configuration": {},
            "functions": {},
            "build": {
                "pre_install": [],
                "post_install": [],
                "environment": {}
            },
            "created_at": "",
            "updated_at": ""
        }
    
    def _update_config_for_uv(self, config: Dict, analysis: Dict) -> Dict:
        """Update configuration to work better with UV"""
        # Ensure requires_build is True for UV
        if "backend" not in config:
            config["backend"] = {}
        config["backend"]["requires_build"] = True
        
        # Add discovered packages to dependencies if not already there
        if "dependencies" not in config:
            config["dependencies"] = {"packages": [], "system": [], "optional": []}
        
        existing_packages = config["dependencies"].get("packages", [])
        for package in analysis["installed_packages"]:
            if package not in existing_packages:
                existing_packages.append(package)
        
        config["dependencies"]["packages"] = existing_packages
        
        # Update timestamps
        config["updated_at"] = self._get_current_time()
        if not config.get("created_at"):
            config["created_at"] = config["updated_at"]
        
        return config
    
    def _get_current_time(self) -> str:
        """Get current timestamp"""
        import datetime
        return datetime.datetime.now().isoformat()
    
    def _update_migration_status(self, integration_name: str, status: Dict):
        """Update migration status file"""
        try:
            all_status = {}
            if self.migration_status_file.exists():
                with open(self.migration_status_file, 'r') as f:
                    all_status = json.load(f)
            
            all_status[integration_name] = status
            
            with open(self.migration_status_file, 'w') as f:
                json.dump(all_status, f, indent=2)
        
        except Exception as e:
            logger.error(f"Failed to update migration status: {e}")
    
    def get_migration_status(self, integration_name: str = None) -> Dict:
        """Get migration status"""
        try:
            if not self.migration_status_file.exists():
                return {}
            
            with open(self.migration_status_file, 'r') as f:
                all_status = json.load(f)
            
            if integration_name:
                return all_status.get(integration_name, {})
            return all_status
        
        except Exception as e:
            logger.error(f"Failed to get migration status: {e}")
            return {}
    
    def migrate_all(self, force: bool = False) -> Dict:
        """Migrate all discovered integrations"""
        old_integrations = self.discover_old_integrations()
        
        if not old_integrations:
            return {
                "success": True,
                "message": "No old integrations found to migrate",
                "migrated": []
            }
        
        results = {}
        migrated = []
        failed = []
        
        for integration in old_integrations:
            logger.info(f"Migrating {integration}...")
            result = self.migrate_integration(integration, force)
            results[integration] = result
            
            if result["success"]:
                migrated.append(integration)
            else:
                failed.append(integration)
        
        return {
            "success": len(failed) == 0,
            "total": len(old_integrations),
            "migrated": migrated,
            "failed": failed,
            "results": results
        }


def main():
    """Main CLI interface"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Migrate integrations to UV system")
    parser.add_argument("action", choices=["discover", "analyze", "migrate", "migrate-all", "status"],
                       help="Action to perform")
    parser.add_argument("--integration", help="Integration name (for analyze/migrate)")
    parser.add_argument("--force", action="store_true", help="Force migration even if not ready")
    parser.add_argument("--base-path", help="Base path for SecAuto", 
                       default="/Volumes/My Shared Files/Home/Downloads/SecAuto")
    
    args = parser.parse_args()
    
    migrator = IntegrationMigrator(args.base_path)
    
    if args.action == "discover":
        integrations = migrator.discover_old_integrations()
        print(f"Found {len(integrations)} old integration(s):")
        for integration in integrations:
            print(f"  - {integration}")
    
    elif args.action == "analyze":
        if not args.integration:
            print("Error: --integration required for analyze action")
            sys.exit(1)
        analysis = migrator.analyze_integration(args.integration)
        print(json.dumps(analysis, indent=2))
    
    elif args.action == "migrate":
        if not args.integration:
            print("Error: --integration required for migrate action")
            sys.exit(1)
        result = migrator.migrate_integration(args.integration, args.force)
        print(json.dumps(result, indent=2))
    
    elif args.action == "migrate-all":
        result = migrator.migrate_all(args.force)
        print(json.dumps(result, indent=2))
    
    elif args.action == "status":
        status = migrator.get_migration_status(args.integration)
        print(json.dumps(status, indent=2))


if __name__ == "__main__":
    main()