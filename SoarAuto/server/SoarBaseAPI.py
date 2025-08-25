#!/usr/bin/env python3
"""
SecAuto Base API for automation scripts
Provides basic context loading and response functions
"""

import json
import sys
import os


def load_context():
    """
    Load execution context from various sources in priority order:
    1. stdin (for integration context execution)
    2. SECAUTO_CONTEXT environment variable
    3. Command line argument
    4. Empty context
    """
    # Try to load from stdin first (playbook execution)
    try:
        stdin_data = sys.stdin.read().strip()
        if stdin_data:
            return json.loads(stdin_data)
    except:
        pass
    
    # Try to load from environment variable
    env_context = os.environ.get('SECAUTO_CONTEXT')
    if env_context:
        try:
            return json.loads(env_context)
        except:
            pass
    
    # Fallback to command line argument
    if len(sys.argv) > 1:
        try:
            return json.loads(sys.argv[1])
        except:
            pass
    
    return {}


def return_context(result):
    """
    Return execution result as JSON to stdout
    """
    print(json.dumps(result, indent=2))