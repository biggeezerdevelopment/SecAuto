#!/usr/bin/env python3
import json
import sys
import os
from datetime import datetime

def main():
    # Get context from stdin
    context_stdin = {}
    raw_data = ""
    error_info = ""
    
    try:
        # Try to read stdin data directly
        raw_data = sys.stdin.read()
        if raw_data.strip():
            context_stdin = json.loads(raw_data.strip())
        else:
            error_info = "Empty stdin data"
    except Exception as e:
        error_info = f"Failed to parse context: {e}"
        context_stdin = {"parsing_error": str(e)}
    
    # Get context from environment variable
    context_env = {}
    env_context = os.environ.get('SECAUTO_CONTEXT')
    if env_context:
        try:
            context_env = json.loads(env_context)
        except json.JSONDecodeError:
            context_env = {"env_parsing_error": "Failed to parse SECAUTO_CONTEXT"}
    
    result = {
        "script_name": "debug_context.py", 
        "executed": True,
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "stdin_context": context_stdin,
        "env_context": context_env,
        "context_keys_stdin": list(context_stdin.keys()) if isinstance(context_stdin, dict) else "not a dict",
        "context_keys_env": list(context_env.keys()) if isinstance(context_env, dict) else "not a dict",
        "raw_stdin_data": raw_data[:200] if raw_data else "empty",
        "error_info": error_info,
        "data_length": len(raw_data) if raw_data else 0,
        "has_env_context": bool(env_context)
    }
    
    print(json.dumps(result, indent=2))
    return 0

if __name__ == "__main__":
    sys.exit(main())