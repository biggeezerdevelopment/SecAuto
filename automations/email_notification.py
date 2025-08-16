#!/usr/bin/env python3
"""
Email Notification Integration for SecAuto

This integration demonstrates client-aware configuration by sending notifications
using client-specific SMTP settings, templates, and recipient lists.

Features:
- Client-specific SMTP configuration
- Custom email templates per client
- Client-specific recipient lists
- Fallback to global notification settings
- Support for HTML and text emails

Usage in playbook:
    {
      "run": "email_notification", 
      "subject": "Security Alert", 
      "message": "Suspicious activity detected",
      "severity": "high"
    }
"""

import json
import sys
import os
import time
from typing import Dict, List, Any

def main():
    """
    Main email notification function
    """
    # Load the execution context
    context = load_context()
    if not context:
        context = {}
    
    # Extract notification parameters
    subject = context.get('subject', 'SecAuto Notification')
    message = context.get('message', 'No message provided')
    severity = context.get('severity', 'info')
    recipients = context.get('recipients', [])
    
    # Step 1: Detect client context
    client_id = get_client_context()
    
    # Step 2: Load client-specific email configuration
    config = get_client_integration_config("email_notification")
    
    # Step 3: Use fallback configuration if no client-specific config
    if not config:
        config = get_fallback_email_config(client_id)
    
    # Extract configuration
    smtp_settings = config.get("config", {})
    credentials = config.get("credentials", {})
    
    # Step 4: Prepare email content using client-specific templates
    email_content = prepare_email_content(
        subject, message, severity, smtp_settings, client_id
    )
    
    # Step 5: Determine recipients
    final_recipients = determine_recipients(recipients, severity, smtp_settings)
    
    # Step 6: Simulate sending email (in real implementation, use smtplib)
    email_result = simulate_email_send(
        email_content, final_recipients, smtp_settings, credentials
    )
    
    # Step 7: Return results
    result = {
        "email_notification": {
            "success": email_result["success"],
            "client_id": client_id,
            "config_source": "client-specific" if client_id and config.get("client_id") else "global",
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ"),
            "email_details": {
                "subject": email_content["subject"],
                "recipients": final_recipients,
                "severity": severity,
                "template_used": email_content["template_used"],
                "smtp_server": smtp_settings.get("smtp_server", "default"),
                "encryption": smtp_settings.get("use_tls", True)
            },
            "delivery_status": email_result.get("status", "unknown")
        }
    }
    
    if not email_result["success"]:
        result["email_notification"]["error"] = email_result.get("error", "Unknown error")
    
    return_context(result)

def get_fallback_email_config(client_id: str) -> Dict[str, Any]:
    """
    Provide fallback email configuration when client-specific config is not available
    """
    return {
        "name": "email_notification",
        "type": "notification",
        "enabled": True,
        "config": {
            "smtp_server": "smtp.company.com",
            "smtp_port": 587,
            "use_tls": True,
            "sender_name": "SecAuto Security Platform",
            "sender_email": "security@company.com",
            "template_style": "corporate",
            "severity_colors": {
                "critical": "#dc3545",
                "high": "#fd7e14", 
                "medium": "#ffc107",
                "low": "#28a745",
                "info": "#17a2b8"
            },
            "default_recipients": {
                "critical": ["security-team@company.com", "ciso@company.com"],
                "high": ["security-team@company.com"],
                "medium": ["security-analysts@company.com"],
                "low": ["security-analysts@company.com"],
                "info": ["security-alerts@company.com"]
            },
            "include_client_prefix": True
        },
        "credentials": {
            "smtp_username": "security-alerts",
            "smtp_password": "fallback-password"
        }
    }

def prepare_email_content(subject: str, message: str, severity: str, config: Dict, client_id: str) -> Dict[str, Any]:
    """
    Prepare email content using client-specific templates and styling
    """
    template_style = config.get("template_style", "corporate")
    severity_colors = config.get("severity_colors", {})
    include_client_prefix = config.get("include_client_prefix", True)
    
    # Add client prefix to subject if configured
    if include_client_prefix and client_id:
        final_subject = f"[{client_id.upper()}] {subject}"
    else:
        final_subject = subject
    
    # Get severity color
    severity_color = severity_colors.get(severity, "#17a2b8")
    
    # Generate HTML content based on template style
    if template_style == "modern":
        html_body = generate_modern_template(final_subject, message, severity, severity_color, client_id)
    elif template_style == "minimal":
        html_body = generate_minimal_template(final_subject, message, severity, severity_color, client_id)
    else:  # corporate (default)
        html_body = generate_corporate_template(final_subject, message, severity, severity_color, client_id)
    
    # Generate plain text version
    text_body = f"""
{final_subject}

Severity: {severity.upper()}
Client: {client_id or 'Global'}
Timestamp: {time.strftime('%Y-%m-%d %H:%M:%S UTC')}

Message:
{message}

---
This alert was generated by SecAuto Security Platform
"""
    
    return {
        "subject": final_subject,
        "html_body": html_body,
        "text_body": text_body.strip(),
        "template_used": template_style
    }

def generate_corporate_template(subject: str, message: str, severity: str, color: str, client_id: str) -> str:
    """Generate corporate-style HTML email template"""
    return f"""
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>{subject}</title>
</head>
<body style="font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f5f5f5;">
    <div style="max-width: 600px; margin: 0 auto; background-color: white; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 10px rgba(0,0,0,0.1);">
        <div style="background-color: {color}; color: white; padding: 20px;">
            <h1 style="margin: 0; font-size: 24px;">SecAuto Security Alert</h1>
            <p style="margin: 5px 0 0 0; opacity: 0.9;">Severity: {severity.upper()}</p>
        </div>
        <div style="padding: 30px;">
            <h2 style="color: #333; margin-top: 0;">{subject}</h2>
            <div style="background-color: #f8f9fa; padding: 20px; border-radius: 5px; margin: 20px 0;">
                <p style="margin: 0; line-height: 1.6; color: #555;">{message}</p>
            </div>
            <table style="width: 100%; border-collapse: collapse; margin: 20px 0;">
                <tr>
                    <td style="padding: 8px 0; color: #666; font-weight: bold;">Client:</td>
                    <td style="padding: 8px 0; color: #333;">{client_id or 'Global'}</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0; color: #666; font-weight: bold;">Timestamp:</td>
                    <td style="padding: 8px 0; color: #333;">{time.strftime('%Y-%m-%d %H:%M:%S UTC')}</td>
                </tr>
            </table>
        </div>
        <div style="background-color: #f8f9fa; padding: 20px; text-align: center; color: #666; font-size: 12px;">
            This alert was generated by SecAuto Security Platform
        </div>
    </div>
</body>
</html>
"""

def generate_modern_template(subject: str, message: str, severity: str, color: str, client_id: str) -> str:
    """Generate modern-style HTML email template"""
    return f"""
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>{subject}</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);">
    <div style="max-width: 600px; margin: 0 auto; background-color: white; border-radius: 12px; overflow: hidden; box-shadow: 0 8px 32px rgba(0,0,0,0.15);">
        <div style="padding: 40px 30px; text-align: center;">
            <div style="width: 60px; height: 60px; background-color: {color}; border-radius: 50%; margin: 0 auto 20px; display: flex; align-items: center; justify-content: center;">
                <span style="color: white; font-size: 24px; font-weight: bold;">!</span>
            </div>
            <h1 style="margin: 0 0 10px 0; font-size: 28px; color: #333;">{subject}</h1>
            <p style="margin: 0; color: {color}; font-size: 16px; font-weight: 600;">{severity.upper()} SEVERITY</p>
        </div>
        <div style="padding: 0 30px 30px;">
            <div style="background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%); padding: 25px; border-radius: 8px; margin: 20px 0;">
                <p style="margin: 0; line-height: 1.7; color: #444; font-size: 16px;">{message}</p>
            </div>
            <div style="display: flex; justify-content: space-between; margin: 25px 0; padding: 20px; background-color: #f8f9fa; border-radius: 8px;">
                <div>
                    <p style="margin: 0; color: #666; font-size: 12px; text-transform: uppercase; letter-spacing: 1px;">Client</p>
                    <p style="margin: 5px 0 0 0; color: #333; font-size: 16px; font-weight: 600;">{client_id or 'Global'}</p>
                </div>
                <div style="text-align: right;">
                    <p style="margin: 0; color: #666; font-size: 12px; text-transform: uppercase; letter-spacing: 1px;">Time</p>
                    <p style="margin: 5px 0 0 0; color: #333; font-size: 16px; font-weight: 600;">{time.strftime('%H:%M UTC')}</p>
                </div>
            </div>
        </div>
    </div>
</body>
</html>
"""

def generate_minimal_template(subject: str, message: str, severity: str, color: str, client_id: str) -> str:
    """Generate minimal-style HTML email template"""
    return f"""
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>{subject}</title>
</head>
<body style="font-family: 'Courier New', monospace; margin: 0; padding: 20px; background-color: #fafafa;">
    <div style="max-width: 600px; margin: 0 auto; background-color: white; border: 2px solid {color}; padding: 30px;">
        <h1 style="margin: 0 0 20px 0; font-size: 20px; color: {color}; border-bottom: 1px solid {color}; padding-bottom: 10px;">
            [{severity.upper()}] {subject}
        </h1>
        <div style="margin: 20px 0; padding: 20px; background-color: #f9f9f9; border-left: 4px solid {color};">
            <pre style="margin: 0; font-family: inherit; white-space: pre-wrap; line-height: 1.5;">{message}</pre>
        </div>
        <div style="margin: 20px 0; font-size: 14px; color: #666;">
            <p style="margin: 5px 0;">Client: {client_id or 'Global'}</p>
            <p style="margin: 5px 0;">Time: {time.strftime('%Y-%m-%d %H:%M:%S UTC')}</p>
        </div>
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="margin: 0; font-size: 12px; color: #999;">SecAuto Security Platform</p>
    </div>
</body>
</html>
"""

def determine_recipients(custom_recipients: List[str], severity: str, config: Dict) -> List[str]:
    """
    Determine final recipient list based on severity and configuration
    """
    recipients = []
    
    # Add custom recipients if provided
    if custom_recipients:
        recipients.extend(custom_recipients)
    
    # Add default recipients based on severity
    default_recipients = config.get("default_recipients", {})
    severity_recipients = default_recipients.get(severity, [])
    
    # For high-severity alerts, also include critical recipients
    if severity in ["critical", "high"]:
        critical_recipients = default_recipients.get("critical", [])
        severity_recipients.extend(critical_recipients)
    
    recipients.extend(severity_recipients)
    
    # Remove duplicates while preserving order
    seen = set()
    final_recipients = []
    for email in recipients:
        if email not in seen:
            seen.add(email)
            final_recipients.append(email)
    
    return final_recipients

def simulate_email_send(content: Dict, recipients: List[str], smtp_config: Dict, credentials: Dict) -> Dict[str, Any]:
    """
    Simulate email sending (in real implementation, use smtplib)
    """
    if not recipients:
        return {
            "success": False,
            "error": "No recipients specified",
            "status": "failed"
        }
    
    # Simulate different outcomes based on configuration
    smtp_server = smtp_config.get("smtp_server", "localhost")
    
    if smtp_server == "smtp.company.com":
        return {
            "success": True,
            "status": "delivered",
            "message_id": f"<{int(time.time())}.{hash(content['subject']) % 10000}@secauto>",
            "delivery_time": "< 1 second"
        }
    elif "test" in smtp_server:
        return {
            "success": True,
            "status": "test_mode",
            "message_id": f"<test-{int(time.time())}@secauto>",
            "note": "Email sent to test server"
        }
    else:
        return {
            "success": False,
            "error": "SMTP server configuration invalid",
            "status": "failed"
        }

if __name__ == "__main__":
    main()