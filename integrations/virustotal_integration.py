#!/usr/bin/env python3
"""
VirusTotal Integration for SecAuto

This module provides VirusTotal URL scanning functionality that can be used
from Python automations in SecAuto playbooks.

The integration now uses SecAuto's encrypted configuration system instead
of environment variables for better security.

Usage in automations:
    from integrations.virustotal_integration import VirusTotalIntegration
    
    vt = VirusTotalIntegration()
    result = vt.scan_url("https://example.com")
"""

import json
import requests
from typing import Dict, Any, Optional, List
from dataclasses import dataclass, asdict


try:
    import requests
except ImportError:
    print("Warning: requests module not found. Install with: pip install requests")

try:
    import yaml
except ImportError:
    print("Warning: yaml module not found. Install with: pip install pyyaml")

class VirusTotalIntegration:
    """VirusTotal integration for SecAuto"""
    
    def __init__(self, config_manager=None):
        """
        Initialize VirusTotal integration
        
        Args:
            config_manager: Optional configuration manager for testing
        """
        self.config_manager = config_manager
        self.api_key = None
        self.base_url = "https://www.virustotal.com/api/v3"
        self.timeout = 60
        self.retries = 3
        
        # Load configuration
        self._load_config()
    
    def _load_config(self):
        """Load configuration from SecAuto's integration config system"""
        try:
           # Try to fetch config from SecAuto server
           config = get_integration_config("virustotal")
           if config:
                self.api_key = config.get("apikey")
                self.base_url = config.get("url", self.base_url)
                self.timeout = config.get("settings", {}).get("timeout", self.timeout)
                self.retries = config.get("settings", {}).get("retries", self.retries)
                return
        except Exception as e:
            pass
            #logger.error(f"Failed to load VirusTotal configuration: {e}")
    
    def scan_url(self, url: str) -> Dict[str, Any]:
        """
        Scan a URL with VirusTotal
        
        Args:
            url: URL to scan
            
        Returns:
            Dictionary with scan results
        """
        if not self.api_key:
            return {
                "virustotal": {
                    "success": False,
                    "error_message": "VirusTotal API key not configured"
                }
            }
        
        try:
            # Submit URL for scanning
            headers = {
                'x-apikey': self.api_key,
                'accept': 'application/json',
                'content-type': 'application/x-www-form-urlencoded'
            }
            
            data = {
                'url': url
            }
            
            response = requests.post(
                f"{self.base_url}/urls",
                headers=headers,
                data=data,
                timeout=self.timeout
            )
            
            if response.status_code == 200:
                data = response.json()
                analysis_id = data.get('data', {}).get('id', '')
                
                if analysis_id:
                    # Get the analysis results
                    report = self.get_report(analysis_id)
                    d = {"virustotal": {
                            "success": True,
                            "url": url,
                            "analysis_id": analysis_id,
                        }
                      }
                    d["virustotal"].update(report)
                    return d
                else:
                    return {
                        "virustotal": {
                            "success": False,
                            "error_message": "No analysis ID returned from VirusTotal"
                        }
                    }
            else:
                return {
                    "virustotal": {
                        "success": False,
                        "error_message": f"HTTP {response.status_code}: {response.text}"
                    }
                }
                
        except requests.exceptions.RequestException as e:
            return {
                "virustotal": {
                    "success": False,
                    "error_message": f"Request failed: {str(e)}"
                }
            }
        except Exception as e:
            return {
                "virustotal": {
                    "success": False,
                    "error_message": f"Unexpected error: {str(e)}"
                }
            }
    
    def get_report(self, analysis_id: str) -> Dict[str, Any]:
        """
        Get an existing VirusTotal analysis report
        
        Args:
            analysis_id: Analysis ID from scan submission
            
        Returns:
            Dictionary with report data
        """
        if not self.api_key:
            return {
                "success": False,
                "error_message": "VirusTotal API key not configured"
            }
        
        try:
            headers = {
                'x-apikey': self.api_key,
                'accept': 'application/json'
            }
            
            response = requests.get(
                f"{self.base_url}/analyses/{analysis_id}",
                headers=headers,
                timeout=self.timeout
            )
            
            if response.status_code == 200:
                data = response.json()
                attributes = data.get('data', {}).get('attributes', {})
                
                return {
                    "success": True,
                    "status": attributes.get('status', 'unknown'),
                    "stats": attributes.get('stats', {}),
                }
            else:
                return {
                    "success": False,
                    "error_message": f"HTTP {response.status_code}: {response.text}"
                }
                
        except requests.exceptions.RequestException as e:
            return {
                "success": False,
                "error_message": f"Request failed: {str(e)}"
            }
        except Exception as e:
            return {
                "success": False,
                "error_message": f"Unexpected error: {str(e)}"
            }

def scan_url_with_virustotal(url: str, config_manager=None) -> Dict[str, Any]:
    """
    Scan a URL with VirusTotal integration
    
    Args:
        url: URL to scan
        config_manager: Optional configuration manager
        
    Returns:
        Dictionary with scan results
    """
    vt = VirusTotalIntegration(config_manager)
    return vt.scan_url(url)

def get_virustotal_report(analysis_id: str, config_manager=None) -> Dict[str, Any]:
    """
    Get a VirusTotal analysis report
    
    Args:
        analysis_id: Analysis ID to get report for
        config_manager: Optional configuration manager
        
    Returns:
        Dictionary with report data
    """
    vt = VirusTotalIntegration(config_manager)
    return vt.get_report(analysis_id)

# Example usage
if __name__ == "__main__":
    import json
    
    # Test URL
    test_url = "https://example.com"
    
    print("Testing VirusTotal integration...")
    print(f"Scanning URL: {test_url}")
    result = scan_url_with_virustotal(test_url)
    print(f"Result: {json.dumps(result, indent=2)}") 