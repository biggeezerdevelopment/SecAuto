import json
import sys
from datetime import datetime

def run_data_enrichment():
    """Enrich incident data with additional context and threat intelligence"""
    
    def enrich_incident_data(value):
        # Simulate threat intelligence lookup
        threat_indicators = {
            "ip_addresses": ["192.168.1.100", "10.0.0.50"],
            "domains": ["malicious.example.com", "suspicious.net"],
            "file_hashes": ["abc123def456", "xyz789uvw012"],
            "threat_score": 85,
            "reputation": "malicious",
            "first_seen": "2024-01-01T00:00:00Z",
            "last_seen": datetime.now().isoformat()
        }
        
        # Simulate asset information
        asset_info = {
            "hostname": "workstation-001",
            "ip_address": "192.168.1.100",
            "os": "Windows 10",
            "department": "Engineering",
            "owner": "john.doe@company.com",
            "location": "Building A, Floor 2"
        }
        
        # Simulate user context
        user_context = {
            "username": "john.doe",
            "role": "Software Engineer",
            "department": "Engineering",
            "access_level": "standard",
            "last_login": "2024-01-01T08:00:00Z",
            "login_count": 15
        }
        
        return {
            "threat_intelligence": threat_indicators,
            "asset_information": asset_info,
            "user_context": user_context,
            "processed_by": "data_enrichment",
            "enrichment_timestamp": datetime.now().isoformat(),
            "incident_updates": {
                "threat_score": threat_indicators["threat_score"],
                "reputation": threat_indicators["reputation"],
                "affected_asset": asset_info["hostname"],
                "affected_user": user_context["username"],
                "indicators_count": len(threat_indicators["ip_addresses"]) + len(threat_indicators["domains"]),
                "enriched_at": datetime.now().isoformat()
            }
        }
    
    # The Go code passes a flat context, so we need to work with that structure
    # Instead of using get_process_update, we'll return the result directly
    # and let the Go code handle the merging
    result = enrich_incident_data(None)
    #print(json.dumps(result, indent=2))
    return_context(result)

if __name__ == "__main__":
    run_data_enrichment() 