#!/usr/bin/env python3
"""
Threat Intelligence Lookup Integration for SecAuto

This integration demonstrates client-aware configuration by performing threat
intelligence lookups using client-specific API keys and settings.

Features:
- Automatic client context detection
- Client-specific API credentials
- Configurable threat intel sources
- Fallback to global configuration
- Rate limiting per client

Usage in playbook:
    {"run": "threat_intel_lookup", "indicators": ["8.8.8.8", "malware.exe"]}
"""

import json
import sys
import os
import time
from typing import Dict, List, Any

# Client-aware functions are available as builtins via sitecustomize.py
def main():
    """
    Main threat intelligence lookup function
    """
    # Load the execution context
    context = load_context()
    if not context:
        context = {}
    
    # Get indicators to lookup from context
    indicators = context.get('indicators', [])
    if not indicators:
        return_context({
            "threat_intel_lookup": {
                "success": False,
                "error": "No indicators provided",
                "usage": "Include 'indicators' array in playbook context"
            }
        })
        return
    
    # Step 1: Detect client context
    client_id = get_client_context()
    
    # Step 2: Load client-specific threat intel configuration
    config = get_client_integration_config("threat_intel")
    
    if not config:
        return_context({
            "threat_intel_lookup": {
                "success": False,
                "error": "Threat intelligence integration not configured",
                "client_id": client_id,
                "note": "Contact admin to configure threat intel sources"
            }
        })
        return
    
    # Extract configuration settings
    api_settings = config.get("config", {})
    credentials = config.get("credentials", {})
    
    # Step 3: Perform lookups based on client configuration
    results = {
        "threat_intel_lookup": {
            "success": True,
            "client_id": client_id,
            "config_source": "client-specific" if client_id else "global",
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ"),
            "indicators_processed": len(indicators),
            "sources_used": api_settings.get("enabled_sources", []),
            "results": []
        }
    }
    
    # Process each indicator
    for indicator in indicators:
        indicator_result = process_indicator(indicator, api_settings, credentials)
        results["threat_intel_lookup"]["results"].append(indicator_result)
    
    # Step 4: Apply client-specific post-processing
    if api_settings.get("auto_block_malicious", False):
        malicious_indicators = [
            r["indicator"] for r in results["threat_intel_lookup"]["results"] 
            if r.get("threat_score", 0) >= api_settings.get("block_threshold", 8)
        ]
        if malicious_indicators:
            results["threat_intel_lookup"]["auto_blocked"] = malicious_indicators
    
    # Return enriched results
    return_context(results)

def process_indicator(indicator: str, api_settings: Dict, credentials: Dict) -> Dict[str, Any]:
    """
    Process a single indicator using client-specific sources
    """
    indicator_result = {
        "indicator": indicator,
        "type": detect_indicator_type(indicator),
        "sources_checked": [],
        "threat_score": 0,
        "classifications": [],
        "details": {}
    }
    
    # Simulate different threat intel sources based on client config
    enabled_sources = api_settings.get("enabled_sources", ["virustotal", "alienvault"])
    
    for source in enabled_sources:
        if source == "virustotal" and credentials.get("virustotal_api_key"):
            vt_result = lookup_virustotal(indicator, credentials["virustotal_api_key"])
            indicator_result["sources_checked"].append("virustotal")
            indicator_result["details"]["virustotal"] = vt_result
            indicator_result["threat_score"] = max(indicator_result["threat_score"], vt_result.get("score", 0))
            
        elif source == "alienvault" and credentials.get("alienvault_api_key"):
            av_result = lookup_alienvault(indicator, credentials["alienvault_api_key"])
            indicator_result["sources_checked"].append("alienvault")
            indicator_result["details"]["alienvault"] = av_result
            indicator_result["threat_score"] = max(indicator_result["threat_score"], av_result.get("score", 0))
            
        elif source == "custom_feed" and api_settings.get("custom_feed_url"):
            cf_result = lookup_custom_feed(indicator, api_settings["custom_feed_url"], credentials)
            indicator_result["sources_checked"].append("custom_feed")
            indicator_result["details"]["custom_feed"] = cf_result
            indicator_result["threat_score"] = max(indicator_result["threat_score"], cf_result.get("score", 0))
    
    # Apply client-specific scoring weights
    scoring_weights = api_settings.get("scoring_weights", {})
    if scoring_weights:
        weighted_score = 0
        total_weight = 0
        for source in indicator_result["sources_checked"]:
            if source in scoring_weights:
                source_score = indicator_result["details"][source].get("score", 0)
                weight = scoring_weights[source]
                weighted_score += source_score * weight
                total_weight += weight
        
        if total_weight > 0:
            indicator_result["weighted_threat_score"] = weighted_score / total_weight
    
    # Determine classification based on threat score
    if indicator_result["threat_score"] >= api_settings.get("malicious_threshold", 8):
        indicator_result["classification"] = "malicious"
    elif indicator_result["threat_score"] >= api_settings.get("suspicious_threshold", 5):
        indicator_result["classification"] = "suspicious"
    else:
        indicator_result["classification"] = "clean"
    
    return indicator_result

def detect_indicator_type(indicator: str) -> str:
    """Detect the type of indicator (IP, domain, hash, etc.)"""
    import re
    
    # IP address pattern
    ip_pattern = r'^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$'
    
    # Hash patterns
    md5_pattern = r'^[a-fA-F0-9]{32}$'
    sha1_pattern = r'^[a-fA-F0-9]{40}$'
    sha256_pattern = r'^[a-fA-F0-9]{64}$'
    
    # Domain pattern (basic)
    domain_pattern = r'^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$'
    
    if re.match(ip_pattern, indicator):
        return "ip"
    elif re.match(sha256_pattern, indicator):
        return "sha256"
    elif re.match(sha1_pattern, indicator):
        return "sha1"
    elif re.match(md5_pattern, indicator):
        return "md5"
    elif re.match(domain_pattern, indicator):
        return "domain"
    else:
        return "unknown"

def lookup_virustotal(indicator: str, api_key: str) -> Dict[str, Any]:
    """
    Simulate VirusTotal API lookup
    In real implementation, this would make actual API calls
    """
    # Simulate API response based on indicator
    if "malware" in indicator.lower() or indicator == "evil.com":
        return {
            "source": "virustotal",
            "score": 9,
            "detections": 45,
            "total_engines": 50,
            "first_seen": "2024-01-15",
            "last_analysis": time.strftime("%Y-%m-%dT%H:%M:%SZ")
        }
    elif indicator in ["8.8.8.8", "google.com"]:
        return {
            "source": "virustotal",
            "score": 0,
            "detections": 0,
            "total_engines": 50,
            "first_seen": "2020-01-01",
            "last_analysis": time.strftime("%Y-%m-%dT%H:%M:%SZ")
        }
    else:
        return {
            "source": "virustotal",
            "score": 3,
            "detections": 2,
            "total_engines": 50,
            "first_seen": "2024-08-01",
            "last_analysis": time.strftime("%Y-%m-%dT%H:%M:%SZ")
        }

def lookup_alienvault(indicator: str, api_key: str) -> Dict[str, Any]:
    """
    Simulate AlienVault OTX lookup
    """
    if "malware" in indicator.lower():
        return {
            "source": "alienvault",
            "score": 8,
            "pulse_count": 12,
            "malware_families": ["trojan", "backdoor"],
            "countries": ["CN", "RU"]
        }
    else:
        return {
            "source": "alienvault",
            "score": 1,
            "pulse_count": 0,
            "malware_families": [],
            "countries": []
        }

def lookup_custom_feed(indicator: str, feed_url: str, credentials: Dict) -> Dict[str, Any]:
    """
    Simulate custom threat feed lookup
    """
    return {
        "source": "custom_feed",
        "score": 2,
        "feed_url": feed_url,
        "categories": ["suspicious"],
        "confidence": "medium"
    }

if __name__ == "__main__":
    main()