#!/usr/bin/env python3
import json
import sys
import os
from datetime import datetime

def main():
    # Get Python interpreter info
    python_info = {
        "script_name": "python_info.py",
        "executed": True,
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "python_executable": sys.executable,
        "python_version": sys.version,
        "python_path": sys.path[:3],  # Show first 3 paths
        "virtual_env": os.environ.get('VIRTUAL_ENV', 'Not set'),
        "working_directory": os.getcwd(),
        "environment_check": {
            "venv_detected": hasattr(sys, 'real_prefix') or (hasattr(sys, 'base_prefix') and sys.base_prefix != sys.prefix),
            "prefix": sys.prefix,
            "base_prefix": getattr(sys, 'base_prefix', 'Not available')
        }
    }
    
    print(json.dumps(python_info, indent=2))
    return 0

if __name__ == "__main__":
    sys.exit(main())