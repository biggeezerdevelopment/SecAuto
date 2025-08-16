#!/usr/bin/env python3
"""
Test script to see what context is available after previous step
"""

def main():
    # Load context to see what's available
    context = load_context()
    
    result = {
        "context_reader_result": {
            "message": "This shows what context was available",
            "available_context": context,
            "context_keys": list(context.keys()) if context else [],
            "has_previous_result": "context_test_result" in (context or {}),
            "client_id": get_client_context()
        }
    }
    
    return_context(result)

if __name__ == "__main__":
    main()