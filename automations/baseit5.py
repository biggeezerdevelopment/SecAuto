import json
import sys

  
def run_base5():
    # Example 1: Add a simple element to incident
    r = {"baseit5": {  
            "status": "Condition_met",
            "incident": {
                "severity": "high",
                "assigned_to": "security_team",
                "last_updated": "2024-01-01T12:00:00Z"
            }
        }}
    return_context(r)

# Run the function and output JSON
run_base5() 