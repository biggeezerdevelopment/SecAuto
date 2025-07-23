#!/usr/bin/env python3
"""
VirusTotal URL Scanner Automation for SecAuto

This automation demonstrates how to use the VirusTotal integration
from SecAuto playbooks to scan URLs for malicious content.

The integration now uses SecAuto's encrypted configuration system
instead of requiring API keys to be passed as parameters.

Usage in playbooks:
    {"run": "virustotal_url_scanner", "urls": ["https://example.com", "https://suspicious-site.com"]}
"""

import sys
import os
import json
import logging
from typing import List, Dict, Any
from datetime import datetime

# Add the integrations directory to the Python path
sys.path.append(os.path.join(os.path.dirname(__file__), '..', 'integrations'))

try:
    from virustotal_integration import VirusTotalIntegration
except ImportError as e:
    # Print error to stderr, not stdout
    import sys
    print(f"Error importing VirusTotal integration: {e}", file=sys.stderr)
    print("Make sure the integrations directory is accessible", file=sys.stderr)
    sys.exit(1)

# Configure logging
logging.basicConfig(level=logging.INFO, stream=sys.stderr)
#logger = logging.getLogger(__name__)

def main():
    """
    Main function for VirusTotal URL scanner automation.
    
    Args:
        context: Dictionary containing the automation context
        
    Returns:
        Dictionary with scan results and summary
    """
    local_context = context
    urls = local_context.get('urls', [])
    if not urls:
        result = {"virustotal": {
            'success': False,
            'error': 'No URLs provided in context. Expected "urls" key or "threat_intelligence.domains"',
            'results': [],
            'summary': {
                'total_urls': 0,
                'scanned_urls': 0,
                'malicious_urls': 0,
                'clean_urls': 0,
                'failed_scans': 0
            }
        }}
        
        # Print the result to stdout for the Go server to read
        #print(json.dumps(result))
        return_context(result)
        #return result
    
    # Filter out template variables that weren't resolved
    valid_urls = []
    invalid_urls = []
    
    for url in urls:
        if isinstance(url, str) and url.startswith('{{') and url.endswith('}}'):
            # This is an unresolved template variable
            invalid_urls.append(url)
        elif isinstance(url, str) and url.strip():
            # Check if this is a string representation of an array (from template resolution)
            if url.startswith('[') and url.endswith(']'):
                # This might be a string representation of an array
                try:
                    # Try to parse it as JSON
                    parsed_array = json.loads(url)
                    if isinstance(parsed_array, list):
                        # It's an array, add each item as a valid URL
                        for item in parsed_array:
                            if isinstance(item, str) and item.strip():
                                valid_urls.append(item)
                        continue
                except (json.JSONDecodeError, ValueError):
                    # Not a valid JSON array, treat as regular string
                    pass
            
            valid_urls.append(url)
        else:
            invalid_urls.append(str(url))
    
    if not valid_urls:
        result = {"virustotal": {
            'success': False,
            'error': f'No valid URLs provided. Invalid URLs: {invalid_urls}',
            'results': [],
            'summary': {
                'total_urls': len(urls),
                'scanned_urls': 0,
                'malicious_urls': 0,
                'clean_urls': 0,
                'failed_scans': len(urls)
            }
        }}
        
        # Print the result to stdout for the Go server to read
        #print(json.dumps(result))
        return_context(result)
        #return result
    
    # Initialize VirusTotal integration
    vt = VirusTotalIntegration()
    
    results = []
    malicious_count = 0
    clean_count = 0
    failed_count = 0
    
    # Scan each valid URL
    for url in valid_urls:
        try:
            # Scan the URL
            result = vt.scan_url(url)
            
            # Extract verdict information
            virustotal_result = result.get('virustotal', {})
            stats = virustotal_result.get('stats', {})
            
            # Count malicious detections
            malicious_count_for_url = 0
            if stats:
                malicious_count_for_url = stats.get('malicious', 0)
                malicious_count += malicious_count_for_url
                if malicious_count_for_url == 0:
                    clean_count += 1
            else:
                failed_count += 1
            
            # Add to results
            results.append({
                'url': url,
                'verdicts': stats,
                'success': virustotal_result.get('success', False),
                'error_message': virustotal_result.get('error_message', '')
            })
                
        except Exception as e:
            results.append({
                'url': url,
                'verdict': {},
                'success': False,
                'error_message': str(e)
            })
            failed_count += 1
    
    # Add invalid URLs to results
    for url in invalid_urls:
        results.append({
            'url': url,
            'verdicts': {},
            'success': False,
            'error_message': f'Template variable not resolved: {url}'
        })
        failed_count += 1
    
    # Create summary
    summary = {
        'total_urls': len(urls),
        'scanned_urls': len(valid_urls) - failed_count + len(invalid_urls),
        'malicious_urls': malicious_count,
        'clean_urls': clean_count,
        'failed_scans': failed_count,
        'invalid_urls': len(invalid_urls),
        'success_rate': ((len(valid_urls) - failed_count) / len(valid_urls) * 100) if valid_urls else 0
    }
    
    # Determine overall success
    overall_success = failed_count == 0 and len(valid_urls) > 0
    
    v_t = {"virustotal": {
        'success': overall_success,
        'results': results,
        'summary': summary,
        'timestamp': datetime.now().isoformat(),
    }}
    return_context(v_t)
    
    #return v_t

if __name__ == "__main__":
    main()
