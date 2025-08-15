#!/bin/bash

# SecAuto CLI Installation Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="secauto-cli"
VERSION="latest"
GITHUB_REPO="your-org/secauto"  # Update with actual repo

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
SecAuto CLI Installation Script

Usage: $0 [OPTIONS]

Options:
    -h, --help          Show this help message
    -d, --dir           Installation directory (default: $INSTALL_DIR)
    -v, --version       Version to install (default: $VERSION)
    --local             Install from local build directory
    --uninstall         Uninstall SecAuto CLI

Examples:
    $0                  # Install latest version
    $0 -v v1.0.0        # Install specific version
    $0 --local          # Install from local build
    $0 --uninstall      # Uninstall

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
            -d|--dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            --local)
                LOCAL_INSTALL=true
                shift
                ;;
            --uninstall)
                UNINSTALL=true
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Check if running as root for system installation
check_permissions() {
    if [[ "$INSTALL_DIR" == "/usr/local/bin" ]] || [[ "$INSTALL_DIR" == "/usr/bin" ]]; then
        if [[ $EUID -ne 0 ]]; then
            print_error "Installation to $INSTALL_DIR requires sudo privileges"
            print_status "Please run: sudo $0 $*"
            exit 1
        fi
    fi
}

# Detect platform
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case $os in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        mingw*|cygwin*|msys*)
            OS="windows"
            ;;
        *)
            print_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac
    
    case $arch in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
    
    PLATFORM="${OS}-${ARCH}"
    print_status "Detected platform: $PLATFORM"
}

# Install from local build
install_local() {
    local build_dir="build"
    local binary_path=""
    
    # Look for platform-specific binary first
    if [[ -f "${build_dir}/${BINARY_NAME}-${PLATFORM}" ]]; then
        binary_path="${build_dir}/${BINARY_NAME}-${PLATFORM}"
    elif [[ -f "${build_dir}/${BINARY_NAME}" ]]; then
        binary_path="${build_dir}/${BINARY_NAME}"
    else
        print_error "No binary found in build directory"
        print_status "Run './scripts/build.sh' to build the binary first"
        exit 1
    fi
    
    print_status "Installing from local build: $binary_path"
    
    # Create installation directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"
    
    # Copy binary
    cp "$binary_path" "$INSTALL_DIR/secauto"
    chmod +x "$INSTALL_DIR/secauto"
    
    print_success "SecAuto CLI installed to $INSTALL_DIR/secauto"
}

# Download and install from GitHub releases
install_remote() {
    print_error "Remote installation not yet implemented"
    print_status "Please use --local installation for now"
    print_status "Run: ./scripts/build.sh && ./scripts/install.sh --local"
    exit 1
    
    # TODO: Implement GitHub releases download
    # local download_url="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${BINARY_NAME}-${PLATFORM}"
    # if [[ "$OS" == "windows" ]]; then
    #     download_url="${download_url}.exe"
    # fi
    
    # print_status "Downloading from: $download_url"
    # curl -L "$download_url" -o "/tmp/${BINARY_NAME}"
    # chmod +x "/tmp/${BINARY_NAME}"
    # mv "/tmp/${BINARY_NAME}" "$INSTALL_DIR/secauto"
}

# Uninstall SecAuto CLI
uninstall() {
    local binary_path="$INSTALL_DIR/secauto"
    
    if [[ -f "$binary_path" ]]; then
        print_status "Removing $binary_path"
        rm -f "$binary_path"
        print_success "SecAuto CLI uninstalled"
    else
        print_warning "SecAuto CLI not found at $binary_path"
    fi
    
    # Remove config directory (optional)
    local config_dir="$HOME/.config/secauto-cli"
    if [[ -d "$config_dir" ]]; then
        echo -n "Remove configuration directory $config_dir? (y/N): "
        read -r response
        if [[ "$response" == "y" ]] || [[ "$response" == "Y" ]]; then
            rm -rf "$config_dir"
            print_success "Configuration directory removed"
        fi
    fi
}

# Verify installation
verify_installation() {
    local binary_path="$INSTALL_DIR/secauto"
    
    if [[ -f "$binary_path" ]] && [[ -x "$binary_path" ]]; then
        print_success "Installation verified"
        print_status "SecAuto CLI version:"
        "$binary_path" --version || true
        
        # Check if in PATH
        if command -v secauto &> /dev/null; then
            print_success "SecAuto CLI is available in PATH"
        else
            print_warning "SecAuto CLI is not in PATH"
            print_status "Add $INSTALL_DIR to your PATH or create a symlink:"
            print_status "  export PATH=\"$INSTALL_DIR:\$PATH\""
            print_status "  # or"
            print_status "  ln -s $INSTALL_DIR/secauto /usr/local/bin/secauto"
        fi
        
        print_status ""
        print_status "Quick start:"
        print_status "  secauto config set server http://localhost:9090"
        print_status "  secauto config set api-key your-api-key"
        print_status "  secauto health"
    else
        print_error "Installation verification failed"
        exit 1
    fi
}

# Main function
main() {
    print_status "SecAuto CLI Installation Script"
    
    if [[ "$UNINSTALL" == "true" ]]; then
        check_permissions
        uninstall
        return
    fi
    
    detect_platform
    check_permissions
    
    # Create installation directory
    mkdir -p "$INSTALL_DIR"
    
    if [[ "$LOCAL_INSTALL" == "true" ]]; then
        install_local
    else
        install_remote
    fi
    
    verify_installation
}

# Parse arguments and run
parse_args "$@"
main