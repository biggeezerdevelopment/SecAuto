#!/usr/bin/env python3
"""
Threat analysis that uses enriched data from previous step
"""

def main():
    context = load_context()
    
    # Use enriched IP data from previous step
    ip_enrichment = context.get("ip_enrichment", {})
    
    high_threat_ips = []
    total_threat_score = 0
    
    for ip, data in ip_enrichment.items():
        threat_score = data.get("threat_score", 0)
        total_threat_score += threat_score
        
        if threat_score >= 5:
            high_threat_ips.append({
                "ip": ip,
                "threat_score": threat_score,
                "reputation": data.get("reputation"),
                "country": data.get("country")
            })
    
    # Return analysis results
    return_context({
        "threat_analysis": {
            "high_threat_count": len(high_threat_ips),
            "high_threat_ips": high_threat_ips,
            "average_threat_score": total_threat_score / len(ip_enrichment) if ip_enrichment else 0,
            "recommendation": "quarantine" if high_threat_ips else "monitor"
        },
        "analysis_complete": True
    })

if __name__ == "__main__":
    main()