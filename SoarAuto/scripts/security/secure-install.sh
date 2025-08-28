#!/bin/bash
#
# SecAuto Secure Installation Script
# This script sets up SecAuto in a secure jail environment with systemd service
# 
# Usage: sudo ./secure-install.sh [OPTIONS]
# Options:
#   --jail-path <path>    Custom jail path (default: /opt/secauto-jail)
#   --user <username>     Custom service user (default: secauto)
#   --port <port>         API port (default: 9090)
#   --network-isolation   Enable network namespace isolation
#   --readonly-venv       Make Python venv read-only after install
#   --help               Show this help message
#

set -e

# Configuration defaults
JAIL_PATH="/opt/secauto-jail"
SECAUTO_USER="secauto"
SECAUTO_GROUP="secauto"
API_PORT="9090"
SECAUTO_SRC="$(dirname "$(readlink -f "$0")")/.."
NETWORK_ISOLATION=false
READONLY_VENV=false
INSTALL_DIR="/opt/secauto"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_debug() { echo -e "${BLUE}[DEBUG]${NC} $1"; }

# Help function
show_help() {
    echo "SecAuto Secure Installation Script"
    echo ""
    echo "Usage: sudo $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --jail-path <path>      Custom jail path (default: $JAIL_PATH)"
    echo "  --user <username>       Custom service user (default: $SECAUTO_USER)"
    echo "  --port <port>           API port (default: $API_PORT)"
    echo "  --network-isolation     Enable network namespace isolation"
    echo "  --readonly-venv         Make Python venv read-only after install"
    echo "  --help                  Show this help message"
    echo ""
    echo "This script will:"
    echo "  1. Create a secure jail environment for SecAuto"
    echo "  2. Set up dedicated user/group with minimal privileges"
    echo "  3. Configure systemd service with security hardening"
    echo "  4. Set up firewall rules for API access"
    echo "  5. Create monitoring and logging infrastructure"
    exit 0
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --jail-path)
            JAIL_PATH="$2"
            shift 2
            ;;
        --user)
            SECAUTO_USER="$2"
            SECAUTO_GROUP="$2"
            shift 2
            ;;
        --port)
            API_PORT="$2"
            shift 2
            ;;
        --network-isolation)
            NETWORK_ISOLATION=true
            shift
            ;;
        --readonly-venv)
            READONLY_VENV=true
            shift
            ;;
        --help)
            show_help
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            ;;
    esac
done

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root (use sudo)"
    exit 1
fi

# Check if SecAuto source exists
if [[ ! -f "$SECAUTO_SRC/secauto" ]] && [[ ! -f "$SECAUTO_SRC/main.go" ]]; then
    log_error "SecAuto binary or source not found at $SECAUTO_SRC"
    log_error "Please run this script from the SecAuto directory"
    exit 1
fi

log_info "Starting SecAuto secure installation..."
log_info "Jail Path: $JAIL_PATH"
log_info "User: $SECAUTO_USER"
log_info "API Port: $API_PORT"
log_info "Network Isolation: $NETWORK_ISOLATION"
log_info "Read-only Venv: $READONLY_VENV"

# Step 1: Create secure user and group
log_info "Creating secure user and group..."
if ! id "$SECAUTO_USER" &>/dev/null; then
    groupadd --system "$SECAUTO_GROUP"
    useradd --system --gid "$SECAUTO_GROUP" --no-create-home \
            --shell /sbin/nologin --comment "SecAuto Service User" \
            --home-dir "$JAIL_PATH/opt/secauto" "$SECAUTO_USER"
    log_info "Created user: $SECAUTO_USER"
else
    log_info "User $SECAUTO_USER already exists"
fi

# Step 2: Create jail directory structure
log_info "Creating jail environment at $JAIL_PATH..."
mkdir -p "$JAIL_PATH"/{bin,lib,lib64,usr/{bin,lib,lib64},dev,proc,sys,tmp,var/{log,run},opt/secauto}

# Step 3: Copy essential system binaries and libraries
log_info "Setting up minimal system environment in jail..."

# Essential binaries
ESSENTIAL_BINS="/bin/sh /bin/bash /bin/cat /bin/ls /bin/echo /bin/mkdir /bin/rm /bin/cp /bin/mv"
for bin in $ESSENTIAL_BINS; do
    if [[ -f $bin ]]; then
        cp "$bin" "$JAIL_PATH$bin"
        # Copy required libraries
        ldd "$bin" 2>/dev/null | grep "=> /" | awk '{print $3}' | while read lib; do
            if [[ -f "$lib" ]] && [[ ! -f "$JAIL_PATH$lib" ]]; then
                mkdir -p "$JAIL_PATH$(dirname "$lib")"
                cp "$lib" "$JAIL_PATH$lib"
            fi
        done
    fi
done

# Copy dynamic linker
if [[ -f /lib64/ld-linux-x86-64.so.2 ]]; then
    cp /lib64/ld-linux-x86-64.so.2 "$JAIL_PATH/lib64/"
fi

# Step 4: Create essential device files
log_info "Creating essential device files..."
mknod "$JAIL_PATH/dev/null" c 1 3
mknod "$JAIL_PATH/dev/zero" c 1 5
mknod "$JAIL_PATH/dev/random" c 1 8
mknod "$JAIL_PATH/dev/urandom" c 1 9
chmod 666 "$JAIL_PATH/dev/null" "$JAIL_PATH/dev/zero" "$JAIL_PATH/dev/random" "$JAIL_PATH/dev/urandom"

# Step 5: Copy SecAuto application
log_info "Installing SecAuto application in jail..."

# Build SecAuto if binary doesn't exist
if [[ ! -f "$SECAUTO_SRC/secauto" ]]; then
    log_info "Building SecAuto binary..."
    cd "$SECAUTO_SRC"
    go build -o secauto .
    cd - >/dev/null
fi

# Copy application files
cp -r "$SECAUTO_SRC"/* "$JAIL_PATH/opt/secauto/"

# Ensure Python venv exists and copy its libraries
if [[ -d "$SECAUTO_SRC/Venv" ]]; then
    log_info "Setting up Python virtual environment..."
    
    # Get Python binary path in venv
    PYTHON_BIN="$JAIL_PATH/opt/secauto/Venv/bin/python"
    
    if [[ -f "$PYTHON_BIN" ]]; then
        # Copy Python libraries
        ldd "$PYTHON_BIN" 2>/dev/null | grep "=> /" | awk '{print $3}' | while read lib; do
            if [[ -f "$lib" ]] && [[ ! -f "$JAIL_PATH$lib" ]]; then
                mkdir -p "$JAIL_PATH$(dirname "$lib")"
                cp "$lib" "$JAIL_PATH$lib"
            fi
        done
        
        # Make venv executable
        chmod +x "$PYTHON_BIN"
        
        # Optionally make venv read-only
        if [[ $READONLY_VENV == true ]]; then
            log_info "Making Python venv read-only..."
            chmod -R 555 "$JAIL_PATH/opt/secauto/Venv/lib"
        fi
    fi
else
    log_warn "Python virtual environment not found at $SECAUTO_SRC/Venv"
    log_warn "You may need to create the venv manually after installation"
fi

# Step 6: Set up proper permissions
log_info "Setting up permissions..."
chown -R "$SECAUTO_USER:$SECAUTO_GROUP" "$JAIL_PATH/opt/secauto"
chown -R "$SECAUTO_USER:$SECAUTO_GROUP" "$JAIL_PATH/var/log"
chown -R "$SECAUTO_USER:$SECAUTO_GROUP" "$JAIL_PATH/var/run"
chown -R "$SECAUTO_USER:$SECAUTO_GROUP" "$JAIL_PATH/tmp"

# Make jail root owned by root to prevent escape
chown root:root "$JAIL_PATH"
chmod 755 "$JAIL_PATH"

# Step 7: Create systemd service with security hardening
log_info "Creating systemd service configuration..."

cat > /etc/systemd/system/secauto.service << EOF
[Unit]
Description=SecAuto SOAR Automation Platform (Secure Jail)
Documentation=https://github.com/your-org/secauto
After=network-online.target
Wants=network-online.target
RequiresMountsFor=$JAIL_PATH

[Service]
Type=simple
User=root
Group=root

# Security: Run in chroot jail
ExecStartPre=/bin/bash -c 'mkdir -p $JAIL_PATH/var/run && chown $SECAUTO_USER:$SECAUTO_GROUP $JAIL_PATH/var/run'
ExecStart=/usr/bin/chroot $JAIL_PATH /bin/su -s /opt/secauto/secauto $SECAUTO_USER
ExecReload=/bin/kill -HUP \$MAINPID
KillMode=process
Restart=always
RestartSec=10

# Working directory inside jail
WorkingDirectory=/opt/secauto

# Environment variables
Environment=SECAUTO_PORT=$API_PORT
Environment=SECAUTO_CONFIG=/opt/secauto/config.yaml
Environment=SECAUTO_JAIL=true

# Security Hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectControlGroups=yes
RestrictRealtime=yes
RestrictSUIDSGID=yes
RemoveIPC=yes
PrivateTmp=yes
PrivateDevices=yes
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX
SystemCallFilter=@system-service
SystemCallFilter=~@debug @mount @cpu-emulation @obsolete @privileged @reboot @swap
SystemCallErrorNumber=EPERM

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096
MemoryMax=2G
CPUQuota=80%

# Capabilities
CapabilityBoundingSet=
AmbientCapabilities=

# Network restrictions (if not using network isolation)
# IPAddressDeny=127.0.0.0/8 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16

[Install]
WantedBy=multi-user.target
EOF

# Create network-isolated service variant if requested
if [[ $NETWORK_ISOLATION == true ]]; then
    log_info "Creating network-isolated systemd service..."
    
    cat > /etc/systemd/system/secauto-isolated.service << EOF
[Unit]
Description=SecAuto SOAR Automation Platform (Network Isolated)
Documentation=https://github.com/your-org/secauto
After=network-online.target
Wants=network-online.target
RequiresMountsFor=$JAIL_PATH

[Service]
Type=simple
User=root
Group=root

# Create network namespace and run in jail
ExecStartPre=/bin/bash -c 'ip netns add secauto-ns || true'
ExecStartPre=/bin/bash -c 'ip netns exec secauto-ns ip link set lo up'
ExecStart=/bin/bash -c 'ip netns exec secauto-ns chroot $JAIL_PATH /bin/su -s /opt/secauto/secauto $SECAUTO_USER'
ExecStopPost=/bin/bash -c 'ip netns del secauto-ns || true'

# Same security settings as main service
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
RestrictRealtime=yes
PrivateTmp=yes
LimitNOFILE=65536
MemoryMax=2G
CPUQuota=80%

[Install]
WantedBy=multi-user.target
EOF
fi

# Step 8: Create firewall rules
log_info "Configuring firewall rules..."

# UFW rules
if command -v ufw >/dev/null 2>&1; then
    ufw allow "$API_PORT/tcp" comment "SecAuto API"
    log_info "Added UFW rule for port $API_PORT"
fi

# iptables rules (backup)
cat > /etc/systemd/system/secauto-firewall.service << EOF
[Unit]
Description=SecAuto Firewall Rules
Before=secauto.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/bin/bash -c 'iptables -A INPUT -p tcp --dport $API_PORT -j ACCEPT'
ExecStop=/bin/bash -c 'iptables -D INPUT -p tcp --dport $API_PORT -j ACCEPT || true'

[Install]
WantedBy=multi-user.target
EOF

# Step 9: Create monitoring and health check scripts
log_info "Setting up monitoring and health checks..."

cat > "$JAIL_PATH/opt/secauto/health-check.sh" << 'EOF'
#!/bin/bash
# SecAuto Health Check Script
# Runs inside the jail environment

API_PORT=${SECAUTO_PORT:-9090}
MAX_ATTEMPTS=5
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -s -f "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
        echo "SecAuto API is healthy"
        exit 0
    fi
    
    ATTEMPT=$((ATTEMPT + 1))
    echo "Health check attempt $ATTEMPT failed, retrying..."
    sleep 2
done

echo "SecAuto API health check failed after $MAX_ATTEMPTS attempts"
exit 1
EOF

chmod +x "$JAIL_PATH/opt/secauto/health-check.sh"

# External health check service
cat > /etc/systemd/system/secauto-healthcheck.service << EOF
[Unit]
Description=SecAuto Health Check
After=secauto.service

[Service]
Type=oneshot
ExecStart=/usr/bin/chroot $JAIL_PATH /opt/secauto/health-check.sh

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/secauto-healthcheck.timer << EOF
[Unit]
Description=SecAuto Health Check Timer
Requires=secauto-healthcheck.service

[Timer]
OnBootSec=5min
OnUnitActiveSec=2min

[Install]
WantedBy=timers.target
EOF

# Step 10: Create log rotation
log_info "Setting up log rotation..."

cat > /etc/logrotate.d/secauto << EOF
$JAIL_PATH/opt/secauto/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    notifempty
    create 644 $SECAUTO_USER $SECAUTO_GROUP
    postrotate
        /bin/systemctl reload secauto.service > /dev/null 2>&1 || true
    endscript
}
EOF

# Step 11: Create management scripts
log_info "Creating management scripts..."

cat > /usr/local/bin/secauto-admin << EOF
#!/bin/bash
# SecAuto Administration Script

JAIL_PATH="$JAIL_PATH"
SECAUTO_USER="$SECAUTO_USER"

case "\$1" in
    shell)
        echo "Entering SecAuto jail environment..."
        chroot "\$JAIL_PATH" /bin/bash
        ;;
    python)
        echo "Entering SecAuto Python environment..."
        chroot "\$JAIL_PATH" /opt/secauto/Venv/bin/python "\${@:2}"
        ;;
    logs)
        echo "Viewing SecAuto logs..."
        tail -f "\$JAIL_PATH/opt/secauto/logs/secauto.log"
        ;;
    status)
        systemctl status secauto
        ;;
    restart)
        systemctl restart secauto
        ;;
    backup)
        echo "Creating backup of SecAuto data..."
        tar -czf "/tmp/secauto-backup-\$(date +%Y%m%d-%H%M%S).tar.gz" -C "\$JAIL_PATH/opt/secauto" data automations integrations playbooks
        echo "Backup created in /tmp/"
        ;;
    install-package)
        if [[ -n "\$2" ]]; then
            echo "Installing Python package: \$2"
            chroot "\$JAIL_PATH" /opt/secauto/Venv/bin/pip install "\$2"
        else
            echo "Usage: secauto-admin install-package <package-name>"
        fi
        ;;
    *)
        echo "SecAuto Administration Script"
        echo "Usage: \$0 {shell|python|logs|status|restart|backup|install-package}"
        echo ""
        echo "Commands:"
        echo "  shell              Enter jail shell environment"
        echo "  python [args]      Run Python in jail environment"
        echo "  logs               View SecAuto logs"
        echo "  status             Show service status"
        echo "  restart            Restart SecAuto service"
        echo "  backup             Create data backup"
        echo "  install-package    Install Python package in venv"
        exit 1
        ;;
esac
EOF

chmod +x /usr/local/bin/secauto-admin

# Step 12: Final configuration
log_info "Performing final configuration..."

# Create default config if it doesn't exist
if [[ ! -f "$JAIL_PATH/opt/secauto/config.yaml" ]]; then
    log_warn "config.yaml not found, creating default configuration"
    cat > "$JAIL_PATH/opt/secauto/config.yaml" << EOF
# SecAuto Secure Configuration
server:
  port: $API_PORT
  host: "0.0.0.0"  # Listen on all interfaces for API access
  workers: 5

logging:
  level: "INFO"
  destination: "both"
  file: "logs/secauto.log"
  
security:
  api_keys:
    - "change-this-secure-api-key-$(openssl rand -hex 16)"
  rate_limiting:
    enabled: true
    requests_per_minute: 100

python:
  venv_path: "Venv"
  scripts_path: "automations"
  sandbox_mode: true
  allow_network_access: false
  allow_file_access: false
  max_script_memory: 256
  script_timeout: 120
EOF
    chown "$SECAUTO_USER:$SECAUTO_GROUP" "$JAIL_PATH/opt/secauto/config.yaml"
fi

# Enable and start services
log_info "Enabling and starting services..."
systemctl daemon-reload
systemctl enable secauto.service
systemctl enable secauto-firewall.service
systemctl enable secauto-healthcheck.timer

# Start firewall rules first
systemctl start secauto-firewall.service

# Start main service
if systemctl start secauto.service; then
    log_info "SecAuto service started successfully!"
else
    log_error "Failed to start SecAuto service"
    log_error "Check logs with: journalctl -u secauto.service -f"
fi

# Start health check timer
systemctl start secauto-healthcheck.timer

log_info "SecAuto secure installation completed!"
log_info ""
log_info "=============================================="
log_info "SecAuto Installation Summary"
log_info "=============================================="
log_info "Jail Path:       $JAIL_PATH"
log_info "Service User:    $SECAUTO_USER"
log_info "API Port:        $API_PORT"
log_info "API URL:         http://localhost:$API_PORT"
log_info "Health Check:    http://localhost:$API_PORT/health"
log_info ""
log_info "Management Commands:"
log_info "  sudo secauto-admin status    - Check service status"
log_info "  sudo secauto-admin logs      - View logs"
log_info "  sudo secauto-admin shell     - Enter jail environment"
log_info "  sudo secauto-admin python    - Access Python venv"
log_info ""
log_info "Service Control:"
log_info "  sudo systemctl start secauto"
log_info "  sudo systemctl stop secauto"
log_info "  sudo systemctl restart secauto"
log_info "  sudo systemctl status secauto"
log_info ""
log_info "Security Features Enabled:"
log_info "  ✓ Chroot jail isolation"
log_info "  ✓ Dedicated service user"
log_info "  ✓ systemd security hardening"
log_info "  ✓ Resource limits (2GB RAM, 80% CPU)"
log_info "  ✓ Firewall rules for API port"
log_info "  ✓ Health monitoring"
log_info "  ✓ Log rotation"
if [[ $NETWORK_ISOLATION == true ]]; then
    log_info "  ✓ Network namespace isolation"
fi
if [[ $READONLY_VENV == true ]]; then
    log_info "  ✓ Read-only Python venv"
fi
log_info ""
log_warn "IMPORTANT: Change the API key in $JAIL_PATH/opt/secauto/config.yaml"
log_warn "IMPORTANT: Configure firewall to allow access from authorized networks only"