#!/usr/bin/env python3
import json
import sys
import select
from datetime import datetime

def main():
    # Get context from stdin
    context = {}
    raw_data = ""
    error_info = ""
    
    try:
        # Try to read stdin data directly
        raw_data = sys.stdin.read()
        if raw_data.strip():
            context = json.loads(raw_data.strip())
        else:
            error_info = "Empty stdin data"
    except Exception as e:
        error_info = f"Failed to parse context: {e}"
        context = {"parsing_error": str(e)}
    
    result = {
        "script_name": "debug_context.py", 
        "executed": True,
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "received_context": context,
        "context_keys": list(context.keys()) if isinstance(context, dict) else "not a dict",
        "raw_stdin_data": raw_data[:200] if raw_data else "empty",
        "error_info": error_info,
        "data_length": len(raw_data) if raw_data else 0
    }
    
    print(json.dumps(result, indent=2))
    return 0

if __name__ == "__main__":
    sys.exit(main())