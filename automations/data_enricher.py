#!/usr/bin/env python3
"""
Data enrichment that adds context for next steps
"""

def main():
    context = load_context()
    
    # Simulate enriching IP addresses
    ips = context.get("ips", ["8.8.8.8"])
    
    enriched_ips = {}
    for ip in ips:
        enriched_ips[ip] = {
            "reputation": "clean" if ip == "8.8.8.8" else "suspicious",
            "country": "US" if ip == "8.8.8.8" else "Unknown",
            "threat_score": 1 if ip == "8.8.8.8" else 7
        }
    
    # Return enriched data that next steps can use
    return_context({
        "ip_enrichment": enriched_ips,
        "enrichment_timestamp": "2025-08-16T02:24:00Z",
        "total_ips_analyzed": len(ips)
    })

if __name__ == "__main__":
    main()