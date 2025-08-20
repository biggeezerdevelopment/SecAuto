#!/usr/bin/env python3
"""
Integration Upload Hook
Automatically builds integration backend when configuration is uploaded
This script should be called by the Go server after integration upload
"""

import os
import sys
import json
import subprocess
from pathlib import Path
import logging

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from scripts.build_integration_backend import IntegrationBackendBuilder

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def process_integration_upload(config_path: str, integration_name: str = None) -> Dict:
    """
    Process newly uploaded integration configuration
    
    Args:
        config_path: Path to uploaded integration config
        integration_name: Optional integration name override
        
    Returns:
        Processing result dictionary
    """
    try:
        logger.info(f"Processing integration upload: {config_path}")
        
        # Initialize builder
        base_path = Path(__file__).parent.parent
        builder = IntegrationBackendBuilder(base_path)
        
        # Build the integration backend
        result = builder.build_integration(config_path)
        
        if result.get('success'):
            logger.info(f"Successfully built backend for integration")
            
            # Optional: Notify the Go server about build completion
            notify_server_build_complete(result)
            
        else:
            logger.error(f"Failed to build integration backend: {result.get('error')}")
        
        return result
        
    except Exception as e:
        logger.error(f"Error processing integration upload: {e}")
        return {
            "success": False,
            "error": str(e)
        }

def notify_server_build_complete(build_result: Dict):
    """Notify the Go server that integration build is complete"""
    try:
        # This could make an API call to the Go server
        # For now, we'll just log it
        logger.info(f"Integration build complete: {build_result.get('integration')}")
        
        # Optional: Update a status file that Go server monitors
        status_file = Path(__file__).parent.parent / "integrations" / ".last_build.json"
        with open(status_file, 'w') as f:
            json.dump({
                "timestamp": __import__('datetime').datetime.now().isoformat(),
                "result": build_result
            }, f, indent=2)
            
    except Exception as e:
        logger.warning(f"Could not notify server: {e}")

def main():
    """Main entry point when called from command line or Go server"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Process integration upload")
    parser.add_argument("config_path", help="Path to integration configuration")
    parser.add_argument("--name", help="Integration name (optional)")
    parser.add_argument("--async", action="store_true", 
                       help="Run build asynchronously")
    
    args = parser.parse_args()
    
    if args.async:
        # Run in background
        subprocess.Popen([
            sys.executable, __file__, 
            args.config_path,
            "--name", args.name or ""
        ])
        print(json.dumps({"status": "building", "async": True}))
    else:
        # Run synchronously
        result = process_integration_upload(args.config_path, args.name)
        print(json.dumps(result))

if __name__ == "__main__":
    main()