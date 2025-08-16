#!/usr/bin/env python3
"""
File Analysis Integration for SecAuto

This integration demonstrates client-aware configuration for file analysis,
including client-specific sandboxes, analysis engines, and quarantine policies.

Features:
- Client-specific sandbox environments
- Configurable analysis engines (static, dynamic, ML)
- Client-specific file type filtering
- Custom quarantine and reporting policies
- Integration with multiple analysis services

Usage in playbook:
    {
      "run": "file_analysis", 
      "file_path": "/tmp/suspicious.exe",
      "file_hash": "d41d8cd98f00b204e9800998ecf8427e",
      "analysis_type": "full"
    }
"""

import json
import sys
import os
import time
import hashlib
from typing import Dict, List, Any

def main():
    """
    Main file analysis function
    """
    # Load the execution context
    context = load_context()
    if not context:
        context = {}
    
    # Extract file analysis parameters
    file_path = context.get('file_path')
    file_hash = context.get('file_hash')
    analysis_type = context.get('analysis_type', 'basic')
    file_size = context.get('file_size', 0)
    
    if not file_path and not file_hash:
        return_context({
            "file_analysis": {
                "success": False,
                "error": "Either file_path or file_hash must be provided",
                "usage": "Include 'file_path' or 'file_hash' in playbook context"
            }
        })
        return
    
    # Step 1: Detect client context
    client_id = get_client_context()
    
    # Step 2: Load client-specific file analysis configuration
    config = get_client_integration_config("file_analysis")
    
    # Step 3: Use fallback configuration if needed
    if not config:
        config = get_fallback_analysis_config(client_id)
    
    # Extract configuration
    analysis_settings = config.get("config", {})
    credentials = config.get("credentials", {})
    
    # Step 4: Validate file against client policies
    validation_result = validate_file_policies(file_path, file_hash, file_size, analysis_settings)
    if not validation_result["allowed"]:
        return_context({
            "file_analysis": {
                "success": False,
                "client_id": client_id,
                "error": "File analysis blocked by client policy",
                "policy_violation": validation_result["reason"],
                "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ")
            }
        })
        return
    
    # Step 5: Perform analysis based on client configuration
    analysis_results = perform_file_analysis(
        file_path, file_hash, analysis_type, analysis_settings, credentials
    )
    
    # Step 6: Apply client-specific post-processing
    processed_results = apply_client_policies(analysis_results, analysis_settings, client_id)
    
    # Step 7: Generate final response
    result = {
        "file_analysis": {
            "success": True,
            "client_id": client_id,
            "config_source": "client-specific" if client_id and config.get("client_id") else "global",
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ"),
            "file_info": {
                "file_path": file_path,
                "file_hash": file_hash or generate_mock_hash(file_path),
                "file_size": file_size,
                "analysis_type": analysis_type
            },
            "analysis_summary": {
                "threat_level": processed_results["threat_level"],
                "confidence": processed_results["confidence"],
                "engines_used": processed_results["engines_used"],
                "analysis_time": processed_results["analysis_time"]
            },
            "detailed_results": processed_results["details"],
            "client_actions": processed_results["client_actions"]
        }
    }
    
    return_context(result)

def get_fallback_analysis_config(client_id: str) -> Dict[str, Any]:
    """
    Provide fallback file analysis configuration
    """
    return {
        "name": "file_analysis",
        "type": "security_analysis",
        "enabled": True,
        "config": {
            "enabled_engines": ["static_analysis", "signature_scan", "ml_classification"],
            "sandbox_environment": "isolated_vm",
            "analysis_timeout": 300,
            "max_file_size": 100 * 1024 * 1024,  # 100MB
            "allowed_file_types": [
                "application/x-executable",
                "application/x-dosexec", 
                "application/pdf",
                "application/msword",
                "application/zip"
            ],
            "blocked_file_types": [
                "application/x-virus"
            ],
            "quarantine_policy": {
                "auto_quarantine_threshold": 7,
                "quarantine_duration_days": 30,
                "notify_on_quarantine": True
            },
            "reporting": {
                "generate_detailed_report": True,
                "include_screenshots": False,
                "include_network_analysis": True,
                "export_formats": ["json", "pdf"]
            },
            "threat_thresholds": {
                "critical": 9,
                "high": 7,
                "medium": 5,
                "low": 3
            }
        },
        "credentials": {
            "sandbox_api_key": "fallback-sandbox-key",
            "ml_engine_token": "fallback-ml-token"
        }
    }

def validate_file_policies(file_path: str, file_hash: str, file_size: int, config: Dict) -> Dict[str, Any]:
    """
    Validate file against client-specific policies
    """
    max_file_size = config.get("max_file_size", 100 * 1024 * 1024)
    allowed_types = config.get("allowed_file_types", [])
    blocked_types = config.get("blocked_file_types", [])
    
    # Check file size
    if file_size > max_file_size:
        return {
            "allowed": False,
            "reason": f"File size ({file_size} bytes) exceeds maximum allowed ({max_file_size} bytes)"
        }
    
    # Determine file type (mock implementation)
    if file_path:
        file_ext = os.path.splitext(file_path)[1].lower()
        if file_ext == ".exe":
            file_type = "application/x-executable"
        elif file_ext == ".pdf":
            file_type = "application/pdf"
        elif file_ext == ".doc":
            file_type = "application/msword"
        elif file_ext == ".zip":
            file_type = "application/zip"
        else:
            file_type = "application/octet-stream"
        
        # Check blocked types
        if file_type in blocked_types:
            return {
                "allowed": False,
                "reason": f"File type '{file_type}' is blocked by client policy"
            }
        
        # Check allowed types (if specified)
        if allowed_types and file_type not in allowed_types:
            return {
                "allowed": False,
                "reason": f"File type '{file_type}' is not in allowed types list"
            }
    
    return {"allowed": True, "reason": "File passed policy validation"}

def perform_file_analysis(file_path: str, file_hash: str, analysis_type: str, config: Dict, credentials: Dict) -> Dict[str, Any]:
    """
    Perform file analysis using configured engines
    """
    enabled_engines = config.get("enabled_engines", ["static_analysis"])
    analysis_timeout = config.get("analysis_timeout", 300)
    
    results = {
        "engines_used": [],
        "analysis_time": 0,
        "threat_level": "unknown",
        "confidence": 0,
        "details": {},
        "client_actions": []
    }
    
    start_time = time.time()
    max_threat_score = 0
    total_confidence = 0
    
    # Static Analysis Engine
    if "static_analysis" in enabled_engines:
        static_result = run_static_analysis(file_path, file_hash)
        results["engines_used"].append("static_analysis")
        results["details"]["static_analysis"] = static_result
        max_threat_score = max(max_threat_score, static_result["threat_score"])
        total_confidence += static_result["confidence"]
    
    # Signature Scanning
    if "signature_scan" in enabled_engines:
        signature_result = run_signature_scan(file_path, file_hash)
        results["engines_used"].append("signature_scan")
        results["details"]["signature_scan"] = signature_result
        max_threat_score = max(max_threat_score, signature_result["threat_score"])
        total_confidence += signature_result["confidence"]
    
    # ML Classification
    if "ml_classification" in enabled_engines:
        ml_result = run_ml_classification(file_path, file_hash, credentials)
        results["engines_used"].append("ml_classification")
        results["details"]["ml_classification"] = ml_result
        max_threat_score = max(max_threat_score, ml_result["threat_score"])
        total_confidence += ml_result["confidence"]
    
    # Dynamic Analysis (for full analysis)
    if analysis_type == "full" and "dynamic_analysis" in enabled_engines:
        dynamic_result = run_dynamic_analysis(file_path, file_hash, config, credentials)
        results["engines_used"].append("dynamic_analysis")
        results["details"]["dynamic_analysis"] = dynamic_result
        max_threat_score = max(max_threat_score, dynamic_result["threat_score"])
        total_confidence += dynamic_result["confidence"]
    
    # Calculate final scores
    results["analysis_time"] = round(time.time() - start_time, 2)
    results["confidence"] = round(total_confidence / len(results["engines_used"]), 2) if results["engines_used"] else 0
    
    # Determine threat level
    threat_thresholds = config.get("threat_thresholds", {})
    if max_threat_score >= threat_thresholds.get("critical", 9):
        results["threat_level"] = "critical"
    elif max_threat_score >= threat_thresholds.get("high", 7):
        results["threat_level"] = "high"
    elif max_threat_score >= threat_thresholds.get("medium", 5):
        results["threat_level"] = "medium"
    elif max_threat_score >= threat_thresholds.get("low", 3):
        results["threat_level"] = "low"
    else:
        results["threat_level"] = "clean"
    
    return results

def run_static_analysis(file_path: str, file_hash: str) -> Dict[str, Any]:
    """Simulate static analysis engine"""
    # Mock static analysis based on file characteristics
    if file_path and "malware" in file_path.lower():
        return {
            "threat_score": 8,
            "confidence": 85,
            "findings": [
                "Suspicious API calls detected",
                "Obfuscated code sections found", 
                "Known malicious strings identified"
            ],
            "entropy": 7.8,
            "packed": True
        }
    elif file_path and file_path.endswith(".exe"):
        return {
            "threat_score": 3,
            "confidence": 70,
            "findings": [
                "Standard executable structure",
                "No obvious obfuscation"
            ],
            "entropy": 4.2,
            "packed": False
        }
    else:
        return {
            "threat_score": 1,
            "confidence": 90,
            "findings": ["Clean file structure"],
            "entropy": 3.1,
            "packed": False
        }

def run_signature_scan(file_path: str, file_hash: str) -> Dict[str, Any]:
    """Simulate signature scanning engine"""
    # Mock signature detection
    known_malware_hashes = [
        "d41d8cd98f00b204e9800998ecf8427e",  # Example hash
        "098f6bcd4621d373cade4e832627b4f6"   # Another example
    ]
    
    if file_hash in known_malware_hashes:
        return {
            "threat_score": 10,
            "confidence": 100,
            "detections": [
                {"engine": "ClamAV", "result": "Trojan.Generic.123456"},
                {"engine": "Windows Defender", "result": "Virus:Win32/Malware"}
            ],
            "signature_count": 2
        }
    else:
        return {
            "threat_score": 0,
            "confidence": 95,
            "detections": [],
            "signature_count": 0
        }

def run_ml_classification(file_path: str, file_hash: str, credentials: Dict) -> Dict[str, Any]:
    """Simulate ML classification engine"""
    # Mock ML analysis based on file characteristics
    if file_path and "suspicious" in file_path.lower():
        return {
            "threat_score": 6,
            "confidence": 78,
            "classification": "potentially_malicious",
            "features": {
                "api_usage_score": 6.5,
                "behavioral_score": 5.8,
                "structural_score": 7.2
            },
            "model_version": "v2.3.1"
        }
    else:
        return {
            "threat_score": 2,
            "confidence": 82,
            "classification": "benign",
            "features": {
                "api_usage_score": 2.1,
                "behavioral_score": 1.8,
                "structural_score": 2.5
            },
            "model_version": "v2.3.1"
        }

def run_dynamic_analysis(file_path: str, file_hash: str, config: Dict, credentials: Dict) -> Dict[str, Any]:
    """Simulate dynamic analysis in sandbox"""
    sandbox_env = config.get("sandbox_environment", "isolated_vm")
    
    return {
        "threat_score": 4,
        "confidence": 88,
        "sandbox_environment": sandbox_env,
        "execution_time": 120,
        "behaviors_observed": [
            "File system modifications",
            "Network connections attempted",
            "Registry modifications"
        ],
        "network_activity": {
            "connections": ["192.168.1.100:443", "10.0.0.1:80"],
            "dns_queries": ["malicious-domain.com", "legitimate-site.com"]
        },
        "file_operations": [
            "Created: C:\\temp\\payload.exe",
            "Modified: C:\\Windows\\System32\\drivers\\etc\\hosts"
        ]
    }

def apply_client_policies(results: Dict, config: Dict, client_id: str) -> Dict[str, Any]:
    """Apply client-specific post-processing and actions"""
    quarantine_policy = config.get("quarantine_policy", {})
    reporting_config = config.get("reporting", {})
    
    # Determine if file should be quarantined
    auto_quarantine_threshold = quarantine_policy.get("auto_quarantine_threshold", 7)
    threat_score = max([
        details.get("threat_score", 0) 
        for details in results["details"].values()
    ])
    
    client_actions = []
    
    if threat_score >= auto_quarantine_threshold:
        client_actions.append({
            "action": "quarantine",
            "reason": f"Threat score ({threat_score}) exceeds threshold ({auto_quarantine_threshold})",
            "duration_days": quarantine_policy.get("quarantine_duration_days", 30)
        })
        
        if quarantine_policy.get("notify_on_quarantine", True):
            client_actions.append({
                "action": "notify_security_team",
                "notification_type": "quarantine_alert",
                "severity": "high"
            })
    
    # Generate reports based on client configuration
    if reporting_config.get("generate_detailed_report", True):
        export_formats = reporting_config.get("export_formats", ["json"])
        client_actions.append({
            "action": "generate_report",
            "formats": export_formats,
            "include_screenshots": reporting_config.get("include_screenshots", False),
            "include_network_analysis": reporting_config.get("include_network_analysis", True)
        })
    
    # Update results with client actions
    results["client_actions"] = client_actions
    
    return results

def generate_mock_hash(file_path: str) -> str:
    """Generate a mock hash for demonstration"""
    if file_path:
        return hashlib.md5(file_path.encode()).hexdigest()
    return "d41d8cd98f00b204e9800998ecf8427e"

if __name__ == "__main__":
    main()