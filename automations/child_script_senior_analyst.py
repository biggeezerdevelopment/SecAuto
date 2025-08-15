#!/usr/bin/env python3
import json
import sys
from datetime import datetime

def main():
    context = {}
    try:
        context_data = sys.stdin.read().strip()
        if context_data:
            context = json.loads(context_data)
    except:
        pass
    
    result = {
        "script_name": "child_script_senior_analyst.py",
        "executed": True,
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "role_specific_processing": "senior_analyst",
        "context_received": context,
        "message": f"Senior analyst script executed for user: {context.get('user', {}).get('name', 'unknown')}"
    }
    
    print(json.dumps(result))
    return 0

if __name__ == "__main__":
    sys.exit(main())