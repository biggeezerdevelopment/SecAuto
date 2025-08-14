#!/usr/bin/env python3
import json
import sys

def main():
    # Load context from command line argument
    if len(sys.argv) > 1:
        context = json.loads(sys.argv[1])
    else:
        context = {}
    
    # Simple data enrichment simulation
    result = {
        "enrichment": {
            "timestamp": "2024-01-01T12:00:00Z",
            "source": "test_enrichment",
            "data": {
                "processed": True,
                "test_value": context.get("test_value", 0)
            }
        }
    }
    
    # Update context with enriched data
    context.update(result)
    
    # Return updated context
    print(json.dumps(context))

if __name__ == "__main__":
    main()