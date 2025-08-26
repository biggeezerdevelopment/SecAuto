#!/bin/bash

# SecAuto Distribution Package Creator
# Creates a compressed tarball with all necessary files to run the service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
VERSION="1.0.0"
DIST_NAME="secauto"
DIST_DIR="dist"
BUILD_TIMESTAMP=$(date -u +%Y%m%d_%H%M%S)
OS=""
ARCH=""

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
SecAuto Distribution Package Creator

Usage: $0 --os <OS> --arch <ARCHITECTURE> [OPTIONS]

Required Arguments:
    --os <OS>           Target operating system (linux, windows, darwin)
    --arch <ARCH>       Target architecture (amd64, arm64, 386, arm)

Optional Arguments:
    -h, --help          Show this help message
    -v, --version       Set version (default: $VERSION)
    -o, --output        Set output directory (default: $DIST_DIR)
    --with-source       Include source code in distribution
    --minimal           Create minimal distribution (binary + configs only)

Examples:
    $0 --os linux --arch amd64              # Create Linux x86_64 distribution
    $0 --os darwin --arch arm64             # Create macOS M1/M2 distribution
    $0 --os linux --arch amd64 --with-source # Include source code

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            --os)
                OS="$2"
                shift 2
                ;;
            --arch)
                ARCH="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -o|--output)
                DIST_DIR="$2"
                shift 2
                ;;
            --with-source)
                WITH_SOURCE=true
                shift
                ;;
            --minimal)
                MINIMAL=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Main distribution creation
create_distribution() {
    # Validate arguments
    if [[ -z "$OS" || -z "$ARCH" ]]; then
        log_error "OS and ARCH are required"
        show_help
        exit 1
    fi

    # Set up paths
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
    
    # Distribution package name
    PACKAGE_NAME="${DIST_NAME}-${VERSION}-${OS}-${ARCH}"
    PACKAGE_DIR="${DIST_DIR}/${PACKAGE_NAME}"
    TARBALL_NAME="${PACKAGE_NAME}.tar.gz"
    
    # Clean and create distribution directory
    log_info "Creating distribution directory..."
    rm -rf "${SCRIPT_DIR}/${PACKAGE_DIR}"
    mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}"
    
    cd "$PROJECT_ROOT"
    
    # Step 1: Build the binary
    log_info "Building SecAuto binary for ${OS}/${ARCH}..."
    bash "${SCRIPT_DIR}/build_secauto.sh" --os "$OS" --arch "$ARCH" --version "$VERSION" --strip
    
    if [[ $? -ne 0 ]]; then
        log_error "Failed to build binary"
        exit 1
    fi
    
    # Step 2: Copy binary
    log_info "Copying binary..."
    BINARY_NAME="secauto-${OS}-${ARCH}"
    if [[ "$OS" == "windows" ]]; then
        BINARY_NAME="${BINARY_NAME}.exe"
    fi
    
    if [[ -f "${SCRIPT_DIR}/build/${BINARY_NAME}" ]]; then
        cp "${SCRIPT_DIR}/build/${BINARY_NAME}" "${SCRIPT_DIR}/${PACKAGE_DIR}/secauto"
        if [[ "$OS" == "windows" ]]; then
            mv "${SCRIPT_DIR}/${PACKAGE_DIR}/secauto" "${SCRIPT_DIR}/${PACKAGE_DIR}/secauto.exe"
        else
            chmod +x "${SCRIPT_DIR}/${PACKAGE_DIR}/secauto"
        fi
    else
        log_error "Binary not found: ${SCRIPT_DIR}/build/${BINARY_NAME}"
        exit 1
    fi
    
    # Step 3: Copy configuration files
    log_info "Copying configuration files..."
    mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/config"
    
    # Copy main config files
    if [[ -f "SoarAuto/config.yaml" ]]; then
        cp "SoarAuto/config.yaml" "${SCRIPT_DIR}/${PACKAGE_DIR}/config/config.yaml.example"
    fi
    if [[ -f "SoarAuto/config-production.yaml" ]]; then
        cp "SoarAuto/config-production.yaml" "${SCRIPT_DIR}/${PACKAGE_DIR}/config/config-production.yaml.example"
    fi
    
    # Copy database migrations
    if [[ -d "SoarAuto/migrations" ]]; then
        mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/migrations"
        cp -r SoarAuto/migrations/* "${SCRIPT_DIR}/${PACKAGE_DIR}/migrations/" 2>/dev/null || true
    fi
    
    # Step 4: Create necessary directories
    log_info "Creating directory structure..."
    mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/data/automations/metadata"
    mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/data/automations/scripts"
    mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/data/clients"
    mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/data/integrations/configs"
    mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/data/integrations/scripts"
    mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/data/playbooks"
    mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/data/schedules"
    mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/data/security"
    mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/logs"
    
    # Step 5: Copy example files and scripts (unless minimal)
    if [[ "$MINIMAL" != "true" ]]; then
        log_info "Copying example files and scripts..."
        
        # Copy example playbooks
        cp example_*.json "${SCRIPT_DIR}/${PACKAGE_DIR}/" 2>/dev/null || true
        
        # Copy example integrations
        if [[ -d "examples" ]]; then
            mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/examples"
            cp -r examples/* "${SCRIPT_DIR}/${PACKAGE_DIR}/examples/" 2>/dev/null || true
        fi
        
        # Copy Python server files
        if [[ -d "SoarAuto/server" ]]; then
            mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/server"
            cp -r SoarAuto/server/*.py "${SCRIPT_DIR}/${PACKAGE_DIR}/server/" 2>/dev/null || true
            cp SoarAuto/server/pyproject.toml "${SCRIPT_DIR}/${PACKAGE_DIR}/server/" 2>/dev/null || true
        fi
        
        # Copy SDK
        if [[ -d "secauto_sdk" ]]; then
            mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/sdk"
            cp -r secauto_sdk/* "${SCRIPT_DIR}/${PACKAGE_DIR}/sdk/" 2>/dev/null || true
        fi
        
        # Copy security scripts
        if [[ -d "scripts/security" ]]; then
            mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/scripts/security"
            cp -r scripts/security/* "${SCRIPT_DIR}/${PACKAGE_DIR}/scripts/security/" 2>/dev/null || true
        fi
        
        # Copy TLS certificate generation scripts
        cp scripts/generate-certs.sh "${SCRIPT_DIR}/${PACKAGE_DIR}/scripts/" 2>/dev/null || true
        cp scripts/generate-certs.ps1 "${SCRIPT_DIR}/${PACKAGE_DIR}/scripts/" 2>/dev/null || true
        
        # Copy integration builder script
        cp scripts/integration_builder.py "${SCRIPT_DIR}/${PACKAGE_DIR}/scripts/" 2>/dev/null || true
        
        # Copy other utility scripts
        cp scripts/migrate_to_uv.py "${SCRIPT_DIR}/${PACKAGE_DIR}/scripts/" 2>/dev/null || true
        cp scripts/generate_architecture_graph.py "${SCRIPT_DIR}/${PACKAGE_DIR}/scripts/" 2>/dev/null || true
        
        # Copy database installation scripts
        cp scripts/install_postgresql.sh "${SCRIPT_DIR}/${PACKAGE_DIR}/scripts/" 2>/dev/null || true
        cp scripts/init_database.sql "${SCRIPT_DIR}/${PACKAGE_DIR}/scripts/" 2>/dev/null || true
    fi
    
    # Step 6: Copy installation scripts
    log_info "Copying installation scripts..."
    mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/install"
    cp "${SCRIPT_DIR}/install_secauto.sh" "${SCRIPT_DIR}/${PACKAGE_DIR}/install/" 2>/dev/null || true
    cp "${SCRIPT_DIR}/install_venv.sh" "${SCRIPT_DIR}/${PACKAGE_DIR}/install/" 2>/dev/null || true
    cp "${SCRIPT_DIR}/install_uv.sh" "${SCRIPT_DIR}/${PACKAGE_DIR}/install/" 2>/dev/null || true
    cp "${SCRIPT_DIR}/requirements.txt" "${SCRIPT_DIR}/${PACKAGE_DIR}/install/" 2>/dev/null || true
    
    # Step 7: Copy documentation
    log_info "Copying documentation..."
    cp README.md "${SCRIPT_DIR}/${PACKAGE_DIR}/" 2>/dev/null || true
    cp PLAYBOOK_DEVELOPMENT_GUIDE.md "${SCRIPT_DIR}/${PACKAGE_DIR}/" 2>/dev/null || true
    cp SDK_INSTALLATION_GUIDE.md "${SCRIPT_DIR}/${PACKAGE_DIR}/" 2>/dev/null || true
    
    # Step 8: Include source code if requested
    if [[ "$WITH_SOURCE" == "true" ]]; then
        log_info "Including source code..."
        mkdir -p "${SCRIPT_DIR}/${PACKAGE_DIR}/src"
        
        # Copy Go source
        if [[ -d "SoarAuto/pkg" ]]; then
            cp -r SoarAuto/pkg "${SCRIPT_DIR}/${PACKAGE_DIR}/src/"
        fi
        cp SoarAuto/main.go "${SCRIPT_DIR}/${PACKAGE_DIR}/src/" 2>/dev/null || true
        cp SoarAuto/go.mod "${SCRIPT_DIR}/${PACKAGE_DIR}/src/" 2>/dev/null || true
        cp SoarAuto/go.sum "${SCRIPT_DIR}/${PACKAGE_DIR}/src/" 2>/dev/null || true
    fi
    
    # Step 9: Create startup script
    log_info "Creating startup script..."
    if [[ "$OS" == "windows" ]]; then
        cat > "${SCRIPT_DIR}/${PACKAGE_DIR}/start.bat" << 'EOF'
@echo off
echo Starting SecAuto...

REM Check if config exists, if not copy from example
if not exist "config\config.yaml" (
    if exist "config\config.yaml.example" (
        echo Creating config.yaml from example...
        copy "config\config.yaml.example" "config\config.yaml"
    )
)

REM Start SecAuto
secauto.exe
EOF
    else
        cat > "${SCRIPT_DIR}/${PACKAGE_DIR}/start.sh" << 'EOF'
#!/bin/bash

echo "Starting SecAuto..."

# Check if config exists, if not copy from example
if [ ! -f "config/config.yaml" ]; then
    if [ -f "config/config.yaml.example" ]; then
        echo "Creating config.yaml from example..."
        cp "config/config.yaml.example" "config/config.yaml"
    fi
fi

# Start SecAuto
./secauto
EOF
        chmod +x "${SCRIPT_DIR}/${PACKAGE_DIR}/start.sh"
    fi
    
    # Step 10: Create README for distribution
    log_info "Creating distribution README..."
    
    # Set platform-specific instructions
    if [[ "$OS" == "windows" ]]; then
        START_CMD="start.bat or secauto.exe"
        START_BASIC="secauto.exe"
        START_CONFIG="secauto.exe --config config/config.yaml"
        SYS_REQ="Windows 10/Server 2016+"
    elif [[ "$OS" == "darwin" ]]; then
        START_CMD="./start.sh or ./secauto"
        START_BASIC="./secauto"
        START_CONFIG="./secauto --config config/config.yaml"
        SYS_REQ="macOS 10.15+"
    else
        START_CMD="./start.sh or ./secauto"
        START_BASIC="./secauto"
        START_CONFIG="./secauto --config config/config.yaml"
        SYS_REQ="Linux kernel 3.10+"
    fi
    
    cat > "${SCRIPT_DIR}/${PACKAGE_DIR}/DISTRIBUTION_README.md" << EOF
# SecAuto Distribution Package
Version: ${VERSION}
Platform: ${OS}/${ARCH}
Build Date: $(date -u +%Y-%m-%d)

## Quick Start

1. Extract this archive to your desired location
2. Configure SecAuto by editing config/config.yaml
3. Run SecAuto: ${START_CMD}

## Directory Structure

- secauto (or secauto.exe) - Main binary
- config/ - Configuration files
- data/ - Runtime data directories
  - automations/ - Automation scripts and metadata
  - clients/ - Client-specific data
  - integrations/ - Integration configurations and scripts
  - playbooks/ - Playbook definitions
  - schedules/ - Scheduled job definitions
  - security/ - Security configurations
- logs/ - Application logs
- examples/ - Example playbooks and integrations
- scripts/ - Utility scripts
- sdk/ - Python SDK for SecAuto
- install/ - Installation helper scripts

## Database Setup

For PostgreSQL database setup (recommended for production):
1. Run the database installation script:
   \`\`\`
   scripts/install_postgresql.sh --db-name soar_auto --db-user ddfelts
   \`\`\`
2. Or manually run the SQL script:
   \`\`\`
   psql -U postgres -f scripts/init_database.sql
   \`\`\`

## Configuration

1. Copy config/config.yaml.example to config/config.yaml
2. Edit the configuration to match your environment:
   - Set server host and port
   - Configure database connections (use PostgreSQL settings from database setup)
   - Set logging preferences
   - Configure security settings

## Running SecAuto

### Basic Start
${START_BASIC}

### With Custom Config
${START_CONFIG}

## System Requirements

- ${SYS_REQ}
- 512MB RAM minimum, 2GB recommended
- 100MB disk space for application
- Network connectivity for integrations

## Support

For documentation and support, please refer to:
- README.md - Main documentation
- PLAYBOOK_DEVELOPMENT_GUIDE.md - Creating playbooks
- SDK_INSTALLATION_GUIDE.md - Using the Python SDK

EOF
    
    # Step 11: Create version file
    echo "${VERSION}" > "${SCRIPT_DIR}/${PACKAGE_DIR}/VERSION"
    echo "Build: ${BUILD_TIMESTAMP}" >> "${SCRIPT_DIR}/${PACKAGE_DIR}/VERSION"
    echo "Platform: ${OS}/${ARCH}" >> "${SCRIPT_DIR}/${PACKAGE_DIR}/VERSION"
    
    # Step 12: Create the tarball
    log_info "Creating tarball: ${TARBALL_NAME}..."
    cd "${SCRIPT_DIR}/${DIST_DIR}"
    tar -czf "${TARBALL_NAME}" "${PACKAGE_NAME}/"
    
    if [[ $? -eq 0 ]]; then
        # Get file size
        SIZE=$(ls -lah "${TARBALL_NAME}" | awk '{print $5}')
        log_success "Distribution package created: ${SCRIPT_DIR}/${DIST_DIR}/${TARBALL_NAME} (${SIZE})"
        
        # Create checksum
        if command -v sha256sum &> /dev/null; then
            sha256sum "${TARBALL_NAME}" > "${TARBALL_NAME}.sha256"
            log_info "SHA256 checksum: $(cat ${TARBALL_NAME}.sha256)"
        elif command -v shasum &> /dev/null; then
            shasum -a 256 "${TARBALL_NAME}" > "${TARBALL_NAME}.sha256"
            log_info "SHA256 checksum: $(cat ${TARBALL_NAME}.sha256)"
        fi
        
        # Clean up temporary directory
        rm -rf "${PACKAGE_NAME}/"
        
        log_success "Distribution package ready!"
        log_info "Location: ${SCRIPT_DIR}/${DIST_DIR}/${TARBALL_NAME}"
    else
        log_error "Failed to create tarball"
        exit 1
    fi
}

# Parse arguments and run
parse_args "$@"
create_distribution