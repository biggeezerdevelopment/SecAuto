#!/usr/bin/env python3
"""
Client-Aware VirusTotal Scanner for SecAuto

This automation demonstrates a real-world client-aware integration that:
1. Automatically detects client context
2. Loads client-specific VirusTotal API configuration
3. Falls back to global config if no client-specific config exists
4. Performs URL scanning with client-specific settings

Configuration Structure:
{
    "name": "virustotal",
    "type": "api", 
    "enabled": true,
    "config": {
        "api_url": "https://www.virustotal.com/api/v3",
        "timeout": 30,
        "max_retries": 3
    },
    "credentials": {
        "api_key": "your-virustotal-api-key"
    }
}

Usage in playbooks:
    {"run": "client_virustotal_scanner", "urls": ["https://example.com", "https://suspicious-site.com"]}
"""

import json
import sys
import time
import requests
from typing import Dict, Any, List

def scan_url_with_virustotal(url: str, config: Dict[str, Any]) -> Dict[str, Any]:
    """
    Scan a URL using VirusTotal with client-specific configuration
    
    Args:
        url: URL to scan
        config: Client-specific or global VirusTotal configuration
        
    Returns:
        Scan results dictionary
    """
    api_key = config.get("credentials", {}).get("api_key")
    if not api_key:
        return {
            "url": url,
            "success": False,
            "error": "No VirusTotal API key found in configuration"
        }
    
    api_url = config.get("config", {}).get("api_url", "https://www.virustotal.com/api/v3")
    timeout = config.get("config", {}).get("timeout", 30)
    max_retries = config.get("config", {}).get("max_retries", 3)
    
    headers = {
        'x-apikey': api_key,
        'Content-Type': 'application/json'
    }
    
    try:
        # Submit URL for analysis
        submit_url = f"{api_url}/urls"
        submit_data = {"url": url}
        
        response = requests.post(submit_url, headers=headers, json=submit_data, timeout=timeout)
        
        if response.status_code == 200:
            result = response.json()
            analysis_id = result.get("data", {}).get("id")
            
            if analysis_id:
                # Get analysis results
                analysis_url = f"{api_url}/analyses/{analysis_id}"
                
                # Poll for results with retries
                for attempt in range(max_retries):
                    time.sleep(2)  # Wait between polls
                    
                    analysis_response = requests.get(analysis_url, headers=headers, timeout=timeout)
                    
                    if analysis_response.status_code == 200:
                        analysis_data = analysis_response.json()
                        attributes = analysis_data.get("data", {}).get("attributes", {})
                        
                        if attributes.get("status") == "completed":
                            stats = attributes.get("stats", {})
                            return {
                                "url": url,
                                "success": True,
                                "stats": stats,
                                "scan_date": attributes.get("date"),
                                "config_source": "client-specific" if config.get("client_id") else "global"
                            }
                
                return {
                    "url": url,
                    "success": False,
                    "error": "Analysis timeout - results not ready"
                }
            else:
                return {
                    "url": url,
                    "success": False,
                    "error": "Failed to submit URL for analysis"
                }
        else:
            return {
                "url": url,
                "success": False,
                "error": f"API request failed with status {response.status_code}"
            }
            
    except requests.exceptions.RequestException as e:
        return {
            "url": url,
            "success": False,
            "error": f"Request exception: {str(e)}"
        }
    except Exception as e:
        return {
            "url": url,
            "success": False,
            "error": f"Unexpected error: {str(e)}"
        }

def main():
    """
    Main function for client-aware VirusTotal scanning
    """
    # Load the execution context
    context = load_context()
    if not context:
        context = {}
    
    # Get URLs to scan from context
    urls = context.get('urls', [])
    if not urls:
        result = {
            "client_virustotal_scanner": {
                "success": False,
                "error": "No URLs provided in context",
                "client_id": get_client_context()
            }
        }
        return_context(result)
        return
    
    # Detect client context
    client_id = get_client_context()
    
    # Load client-specific or global VirusTotal configuration
    config = get_client_integration_config("virustotal")
    
    if not config:
        result = {
            "client_virustotal_scanner": {
                "success": False,
                "error": "No VirusTotal configuration found for this client or globally",
                "client_id": client_id,
                "suggestion": "Create a VirusTotal integration configuration"
            }
        }
        return_context(result)
        return
    
    # Check if integration is enabled
    if not config.get("enabled", True):
        result = {
            "client_virustotal_scanner": {
                "success": False,
                "error": "VirusTotal integration is disabled",
                "client_id": client_id,
                "config_source": "client-specific" if config.get("client_id") else "global"
            }
        }
        return_context(result)
        return
    
    # Scan each URL
    scan_results = []
    for url in urls:
        if isinstance(url, str) and url.strip():
            scan_result = scan_url_with_virustotal(url.strip(), config)
            scan_results.append(scan_result)
    
    # Compile summary statistics
    successful_scans = [r for r in scan_results if r.get("success")]
    failed_scans = [r for r in scan_results if not r.get("success")]
    
    malicious_count = 0
    suspicious_count = 0
    clean_count = 0
    
    for scan in successful_scans:
        stats = scan.get("stats", {})
        if stats.get("malicious", 0) > 0:
            malicious_count += 1
        elif stats.get("suspicious", 0) > 0:
            suspicious_count += 1
        else:
            clean_count += 1
    
    # Return comprehensive results
    result = {
        "client_virustotal_scanner": {
            "success": len(failed_scans) == 0,
            "execution_info": {
                "client_id": client_id,
                "config_source": "client-specific" if config.get("client_id") else "global",
                "api_endpoint": config.get("config", {}).get("api_url", "default"),
                "total_urls": len(urls),
                "processed_urls": len(scan_results)
            },
            "summary": {
                "successful_scans": len(successful_scans),
                "failed_scans": len(failed_scans),
                "malicious_urls": malicious_count,
                "suspicious_urls": suspicious_count,
                "clean_urls": clean_count
            },
            "detailed_results": scan_results,
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
        }
    }
    
    return_context(result)

if __name__ == "__main__":
    main()