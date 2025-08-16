#!/usr/bin/env python3
"""
Test script to see how return_context data is handled in playbook flow
"""

def main():
    # Load input context
    context = load_context()
    if not context:
        context = {}
    
    # Return some data that should be available to next steps
    result = {
        "context_test_result": {
            "message": "This data should be available to next playbook steps",
            "new_variable": "test_value_123",
            "client_id": get_client_context(),
            "enriched_data": {
                "ip_reputation": "clean",
                "threat_score": 2,
                "analysis_complete": True
            }
        }
    }
    
    return_context(result)

if __name__ == "__main__":
    main()