# SecAuto Security Implementation

This directory contains a comprehensive security implementation for SecAuto, providing jail-based isolation, systemd hardening, and secure management tools.

## 📋 Overview

The security implementation provides:

- **🔒 Chroot Jail Environment** - Complete filesystem isolation
- **🛡️ SystemD Security Hardening** - Maximum process-level security
- **🔥 Firewall Configuration** - Network access control
- **📊 Health Monitoring** - Automated health checks and alerting
- **🛠️ Management Tools** - Secure administration interface
- **🔐 Network Isolation** - Optional network namespace isolation

## 🚀 Quick Installation

### Prerequisites

- Linux system with systemd
- Root access (sudo)
- SecAuto application built (`go build -o secauto .`)
- Python virtual environment set up in `Venv/`

### Basic Installation

```bash
# Make installation script executable
chmod +x security/secure-install.sh

# Run basic secure installation
sudo ./security/secure-install.sh

# Check status
sudo secauto-admin status
```

### Advanced Installation with Network Isolation

```bash
# Install with maximum security features
sudo ./security/secure-install.sh \
    --jail-path /opt/secauto-jail \
    --user secauto \
    --port 9090 \
    --network-isolation \
    --readonly-venv
```

## 📁 File Structure

```
security/
├── secure-install.sh              # Main installation script
├── secauto-security-config.yaml   # Hardened configuration
├── systemd-services/              # SystemD service files
│   ├── secauto.service            # Main service (with jail)
│   ├── secauto-firewall.service   # Firewall rules
│   ├── secauto-healthcheck.service # Health monitoring
│   ├── secauto-healthcheck.timer   # Health check timer
│   └── secauto-network-isolated.service # Network isolated service
├── admin-scripts/
│   └── secauto-admin              # Administration script
└── README.md                      # This file
```

## 🛠️ Installation Options

### Command Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `--jail-path <path>` | Custom jail location | `/opt/secauto-jail` |
| `--user <username>` | Service user name | `secauto` |
| `--port <port>` | API port | `9090` |
| `--network-isolation` | Enable network namespace | Disabled |
| `--readonly-venv` | Make Python venv read-only | Disabled |

### Example Installations

```bash
# Minimal secure installation
sudo ./security/secure-install.sh

# Production installation with custom settings
sudo ./security/secure-install.sh \
    --jail-path /srv/secauto \
    --user secautosvc \
    --port 8443

# Maximum security installation  
sudo ./security/secure-install.sh \
    --network-isolation \
    --readonly-venv \
    --port 9090
```

## 🔒 Security Features

### 1. Chroot Jail Environment

**What it provides:**
- Complete filesystem isolation
- Python automation scripts cannot access host system files
- Limited to essential system libraries and binaries

**Directory structure:**
```
/opt/secauto-jail/
├── bin/          # Essential binaries (sh, bash, ls, etc.)
├── lib/          # Required libraries
├── lib64/        # 64-bit libraries  
├── dev/          # Minimal device files (null, urandom, etc.)
├── tmp/          # Temporary files
├── var/          # Variable data (logs, run files)
└── opt/secauto/  # SecAuto application
    ├── secauto          # Main binary
    ├── Venv/            # Python virtual environment
    ├── config.yaml      # Configuration
    ├── data/            # Application data
    ├── automations/     # Automation scripts
    ├── integrations/    # Integration files
    ├── playbooks/       # Playbook files
    └── logs/            # Log files
```

### 2. SystemD Security Hardening

**Process-level security:**
- `NoNewPrivileges=yes` - Prevents privilege escalation
- `ProtectSystem=strict` - Read-only system directories
- `ProtectHome=yes` - No access to user home directories
- `PrivateTmp=yes` - Isolated temporary directory
- `RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX` - Network restrictions

**Resource limits:**
- Memory: 2GB maximum
- CPU: 80% maximum
- File descriptors: 8192 maximum
- Processes: 256 maximum

**Capabilities:**
- All capabilities removed (`CapabilityBoundingSet=`)
- No ambient capabilities

### 3. Network Security

**Firewall rules:**
- API port restricted to authorized networks (RFC 1918 private networks)
- Rate limiting (60 requests/minute, 10 burst)
- Attack detection and logging
- Automatic blocking of suspicious activity

**Network isolation (optional):**
- Dedicated network namespace
- Controlled outbound access
- Complete network isolation from host

### 4. Application Security

**Configuration hardening:**
- Sandbox mode enabled for Python execution
- No network access for automation scripts
- No file system access outside designated directories
- Resource limits for script execution
- Input validation and sanitization

## 🛡️ SystemD Services

### Primary Service: `secauto.service`

Main SecAuto service running in chroot jail with maximum security hardening.

```bash
# Service management
sudo systemctl start secauto
sudo systemctl stop secauto
sudo systemctl restart secauto
sudo systemctl status secauto
```

### Firewall Service: `secauto-firewall.service`

Manages iptables rules for API access control.

```bash
# Firewall management
sudo systemctl start secauto-firewall
sudo systemctl stop secauto-firewall
sudo systemctl status secauto-firewall
```

### Health Monitoring: `secauto-healthcheck.service` + Timer

Automated health checks every 2 minutes with detailed diagnostics.

```bash
# Health monitoring
sudo systemctl start secauto-healthcheck.timer
sudo systemctl status secauto-healthcheck
```

### Network Isolated Service: `secauto-network-isolated.service`

Alternative service with complete network isolation using network namespaces.

```bash
# Enable network isolated mode
sudo systemctl disable secauto
sudo systemctl enable secauto-network-isolated
sudo systemctl start secauto-network-isolated
```

## 🔧 Management Tools

### SecAuto Admin Script

The `secauto-admin` script provides secure management interface:

```bash
# System management
sudo secauto-admin status          # Service status and health
sudo secauto-admin start           # Start service
sudo secauto-admin restart         # Restart service
sudo secauto-admin logs 100        # View last 100 log lines

# Jail environment access
sudo secauto-admin shell           # Enter jail shell
sudo secauto-admin python          # Python interactive shell
sudo secauto-admin install-package requests  # Install Python package

# Security and monitoring
sudo secauto-admin security-check  # Validate security settings
sudo secauto-admin health-check    # Manual health check
sudo secauto-admin resource-usage  # Resource usage stats

# Data management
sudo secauto-admin backup          # Create data backup
sudo secauto-admin list-automations # List automation scripts
sudo secauto-admin show-config     # Display configuration

# Maintenance
sudo secauto-admin clean-logs      # Clean old logs
sudo secauto-admin generate-api-key # Generate new API key
```

## 🔐 Configuration Security

### Production Configuration

The security implementation includes a hardened configuration (`secauto-security-config.yaml`) with:

- **Strict rate limiting** (60 requests/minute)
- **Reduced resource limits** (128MB per script)
- **Disabled debug features**
- **Network access restrictions**
- **File access restrictions**
- **Comprehensive logging**

### Key Security Settings

```yaml
# Python execution security
python:
  sandbox_mode: true              # Enable sandbox
  allow_network_access: false     # Block network access
  allow_file_access: false        # Block file access
  max_script_memory: 128          # 128MB limit
  script_timeout: 120             # 2-minute timeout

# Security hardening  
security:
  rate_limiting:
    enabled: true
    requests_per_minute: 60       # Strict rate limiting
  input_validation:
    enabled: true
    max_context_size: 102400      # 100KB limit
    sanitize_inputs: true
```

## 🚨 Security Monitoring

### Health Checks

Automated monitoring includes:

- **API responsiveness** - HTTP health endpoint checks
- **Resource usage** - Memory, CPU monitoring
- **Service status** - SystemD service state
- **Jail integrity** - File system permissions
- **Security validation** - Configuration compliance

### Logging and Alerting

- **Structured logging** - JSON format for SIEM integration
- **Security events** - Authentication failures, rate limiting
- **Performance metrics** - Response times, resource usage
- **Audit trails** - Administrative actions, configuration changes

### Log Files

```bash
# Service logs
journalctl -u secauto -f

# Application logs
tail -f /opt/secauto-jail/opt/secauto/logs/secauto.log

# Firewall logs
grep "SecAuto" /var/log/syslog

# Health check logs
journalctl -u secauto-healthcheck -f
```

## 🔄 Backup and Recovery

### Automated Backups

```bash
# Create backup
sudo secauto-admin backup /tmp/secauto-backup.tar.gz

# Restore from backup
sudo secauto-admin restore /tmp/secauto-backup.tar.gz
```

### Backup Contents

- Configuration files
- Automation scripts
- Integration configurations
- Playbook files
- Application data
- Logs (optional)

## 🛠️ Troubleshooting

### Common Issues

1. **Service fails to start**
   ```bash
   sudo secauto-admin status
   sudo journalctl -u secauto -f
   ```

2. **API not accessible**
   ```bash
   sudo secauto-admin api-test
   sudo secauto-admin security-check
   ```

3. **Python scripts fail**
   ```bash
   sudo secauto-admin venv-status
   sudo secauto-admin python
   ```

4. **Permission issues**
   ```bash
   sudo secauto-admin check-permissions
   ```

### Diagnostic Commands

```bash
# Comprehensive system check
sudo secauto-admin status
sudo secauto-admin security-check
sudo secauto-admin resource-usage

# View detailed logs
sudo secauto-admin logs 200
sudo secauto-admin follow-logs

# Test API connectivity
sudo secauto-admin api-test <api-key>
```

## 🔒 Security Best Practices

### 1. API Key Management

```bash
# Generate secure API keys
sudo secauto-admin generate-api-key

# Update configuration
sudo nano /opt/secauto-jail/opt/secauto/config.yaml

# Restart service
sudo secauto-admin restart
```

### 2. Network Access Control

- **Restrict API access** to authorized networks only
- **Use HTTPS** in production (configure TLS in config.yaml)
- **Implement VPN access** for remote administration
- **Monitor access logs** for suspicious activity

### 3. Regular Maintenance

```bash
# Weekly security check
sudo secauto-admin security-check

# Monthly cleanup
sudo secauto-admin clean-logs
sudo secauto-admin clean-temp

# Quarterly backup
sudo secauto-admin backup /secure/location/backup.tar.gz
```

### 4. Monitoring and Alerting

- **Enable health check timer**
- **Configure log monitoring** (ELK, Splunk, etc.)
- **Set up alerting** for service failures
- **Monitor resource usage** trends

## 📞 Support

For issues or questions:

1. Check the logs: `sudo secauto-admin logs`
2. Run security check: `sudo secauto-admin security-check`
3. Validate configuration: `sudo secauto-admin validate-config`
4. Review this documentation
5. Contact system administrator

## 📝 License

This security implementation is part of the SecAuto project and follows the same licensing terms.