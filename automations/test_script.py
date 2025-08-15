#!/usr/bin/env python3
import json
import sys
import os
from datetime import datetime

def main():
    # Get context from environment or stdin
    context = {}
    try:
        # Try to read context from stdin
        import select
        if select.select([sys.stdin], [], [], 0.1)[0]:
            context_data = sys.stdin.read().strip()
            if context_data:
                context = json.loads(context_data)
    except:
        pass
    
    # Also check for context in environment variables
    if 'SECAUTO_CONTEXT' in os.environ:
        try:
            context.update(json.loads(os.environ['SECAUTO_CONTEXT']))
        except:
            pass
    
    # Create result
    result = {
        "script_name": "test_script.py",
        "executed": True,
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "context_received": context,
        "message": f"Test script executed successfully for user: {context.get('user', {}).get('name', 'unknown')}",
        "output": "Script executed and context processed"
    }
    
    # Output result as JSON
    print(json.dumps(result))
    return 0

if __name__ == "__main__":
    sys.exit(main())