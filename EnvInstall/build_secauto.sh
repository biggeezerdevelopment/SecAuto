#!/bin/bash

# SecAuto Build Script
# Builds SecAuto binary for specified OS and architecture

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
VERSION="1.0.0"
BUILD_DIR="build"
BINARY_NAME="secauto"
OS=""
ARCH=""
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

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
SecAuto Build Script

Usage: $0 --os <OS> --arch <ARCHITECTURE> [OPTIONS]

Required Arguments:
    --os <OS>           Target operating system (linux, windows, darwin)
    --arch <ARCH>       Target architecture (amd64, arm64, 386, arm)

Optional Arguments:
    -h, --help          Show this help message
    -v, --version       Set version (default: $VERSION)
    -o, --output        Set output directory (default: $BUILD_DIR)
    -n, --name          Set binary name (default: $BINARY_NAME)
    --clean             Clean build directory before building
    --all               Build for all OS/arch combinations
    --strip             Strip debug symbols (smaller binary)
    --static            Build static binary (Linux only)

Supported OS/Architecture combinations:
    linux/amd64, linux/arm64, linux/386, linux/arm
    windows/amd64, windows/arm64, windows/386
    darwin/amd64, darwin/arm64

Examples:
    $0 --os linux --arch amd64              # Build for Linux x86_64
    $0 --os windows --arch amd64            # Build for Windows x86_64
    $0 --os darwin --arch arm64             # Build for macOS M1/M2
    $0 --all                                # Build for all platforms
    $0 --os linux --arch arm64 --static     # Static Linux ARM64 build

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
                BUILD_DIR="$2"
                shift 2
                ;;
            -n|--name)
                BINARY_NAME="$2"
                shift 2
                ;;
            --all)
                BUILD_ALL=true
                shift
                ;;
            --clean)
                CLEAN=true
                shift
                ;;
            --strip)
                STRIP=true
                shift
                ;;
            --static)
                STATIC=true
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

# Validate OS and architecture
validate_platform() {
    local os=$1
    local arch=$2
    
    case "$os" in
        linux)
            case "$arch" in
                amd64|arm64|386|arm)
                    return 0
                    ;;
                *)
                    log_error "Unsupported architecture for Linux: $arch"
                    return 1
                    ;;
            esac
            ;;
        windows)
            case "$arch" in
                amd64|arm64|386)
                    return 0
                    ;;
                *)
                    log_error "Unsupported architecture for Windows: $arch"
                    return 1
                    ;;
            esac
            ;;
        darwin)
            case "$arch" in
                amd64|arm64)
                    return 0
                    ;;
                *)
                    log_error "Unsupported architecture for macOS: $arch"
                    return 1
                    ;;
            esac
            ;;
        *)
            log_error "Unsupported OS: $os"
            return 1
            ;;
    esac
}

# Validate Go environment
validate_go() {
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    local go_version=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go version: $go_version"
    
    # Check minimum Go version (1.19)
    local min_version="1.19"
    if [[ "$(printf '%s\n' "$min_version" "$go_version" | sort -V | head -n1)" != "$min_version" ]]; then
        log_error "Go version must be at least $min_version, found $go_version"
        exit 1
    fi
}

# Clean build directory
clean_build() {
    if [[ "$CLEAN" == "true" ]]; then
        log_info "Cleaning build directory..."
        rm -rf "$BUILD_DIR"
    fi
}

# Create build directory
create_build_dir() {
    # Always create build directory relative to script location
    local abs_build_dir="${SCRIPT_DIR}/${BUILD_DIR}"
    if [[ ! -d "$abs_build_dir" ]]; then
        log_info "Creating build directory: $abs_build_dir"
        mkdir -p "$abs_build_dir"
    fi
}

# Check and prepare dependencies
prepare_deps() {
    log_info "Checking project structure..."
    
    # Navigate to project root
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
    
    cd "$PROJECT_ROOT"
    
    # Skip CLI directory - only build main SecAuto
    if [[ -d "cli" ]]; then
        log_info "Skipping CLI directory, building main SecAuto only"
    fi
    
    # Check for go.mod in main project root
    if [[ ! -f "go.mod" ]]; then
        log_warning "go.mod not found in project root, checking SoarAuto directory..."
        
        # Check SoarAuto directory (skip CLI)
        if [[ -f "SoarAuto/go.mod" ]]; then
            log_info "Found go.mod in SoarAuto"
            cd "SoarAuto"
        else
            log_error "No go.mod found in main project. Not a Go project?"
            exit 1
        fi
    fi
    
    log_info "Tidying Go modules..."
    go mod tidy
    if [[ $? -ne 0 ]]; then
        log_warning "Failed to tidy Go modules, continuing..."
    fi
    
    log_info "Downloading dependencies..."
    go mod download
    if [[ $? -ne 0 ]]; then
        log_error "Failed to download dependencies"
        exit 1
    fi
}

# Build for specific platform
build_platform() {
    local goos=$1
    local goarch=$2
    local suffix=$3
    
    if ! validate_platform "$goos" "$goarch"; then
        return 1
    fi
    
    local output_name="${BINARY_NAME}"
    if [[ -n "$suffix" ]]; then
        output_name="${BINARY_NAME}-${suffix}"
    else
        output_name="${BINARY_NAME}-${goos}-${goarch}"
    fi
    
    if [[ "$goos" == "windows" ]]; then
        output_name="${output_name}.exe"
    fi
    
    # Always use absolute path for build directory relative to EnvInstall
    local abs_build_dir="${SCRIPT_DIR}/${BUILD_DIR}"
    local output_path="${abs_build_dir}/${output_name}"
    
    log_info "Building for ${goos}/${goarch}..."
    log_info "Output: ${output_path}"
    
    # Build flags
    local ldflags="-X main.Version=$VERSION -X main.BuildDate=$BUILD_DATE"
    
    if [[ "$STRIP" == "true" ]]; then
        ldflags="$ldflags -s -w"
        log_info "Stripping debug symbols..."
    fi
    
    # CGO settings
    local cgo_enabled=0
    if [[ "$STATIC" == "true" && "$goos" == "linux" ]]; then
        ldflags="$ldflags -linkmode external -extldflags '-static'"
        log_info "Building static binary..."
    fi
    
    # Find main.go or cmd/main.go
    local main_file=""
    if [[ -f "main.go" ]]; then
        main_file="."
    else
        log_warning "Could not find main.go, using current directory"
        main_file="."
    fi
    
    # Build command
    CGO_ENABLED=$cgo_enabled GOOS=$goos GOARCH=$goarch go build \
        -ldflags="$ldflags" \
        -o "$output_path" \
        $main_file
    
    if [[ $? -eq 0 ]]; then
        # Get file size
        if [[ -f "$output_path" ]]; then
            local size=$(ls -lah "$output_path" | awk '{print $5}')
            log_success "Built ${output_path} (${size})"
            
            # Make executable on Unix systems
            if [[ "$goos" != "windows" ]]; then
                chmod +x "$output_path"
            fi
        else
            log_error "Build succeeded but output file not found"
            return 1
        fi
    else
        log_error "Failed to build for ${goos}/${goarch}"
        return 1
    fi
}

# Build all platforms
build_all_platforms() {
    log_info "Building for all supported platforms..."
    
    local platforms=(
        "linux:amd64"
        "linux:arm64"
        "linux:386"
        "linux:arm"
        "windows:amd64"
        "windows:arm64"
        "windows:386"
        "darwin:amd64"
        "darwin:arm64"
    )
    
    local success=0
    local failed=0
    
    for platform in "${platforms[@]}"; do
        IFS=':' read -r os arch <<< "$platform"
        if build_platform "$os" "$arch" ""; then
            ((success++))
        else
            ((failed++))
        fi
    done
    
    log_info "Build summary: $success succeeded, $failed failed"
}

# Main function
main() {
    log_info "SecAuto Build Script"
    log_info "Version: $VERSION"
    
    # Validate environment
    validate_go
    
    # Set SCRIPT_DIR globally
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    
    # Prepare build
    clean_build
    create_build_dir
    prepare_deps
    
    if [[ "$BUILD_ALL" == "true" ]]; then
        build_all_platforms
    else
        # Check if OS and ARCH are provided
        if [[ -z "$OS" || -z "$ARCH" ]]; then
            log_error "OS and ARCH are required when not using --all"
            show_help
            exit 1
        fi
        
        if ! build_platform "$OS" "$ARCH" ""; then
            exit 1
        fi
    fi
    
    log_success "Build completed successfully!"
    log_info "Output directory: $BUILD_DIR"
    
    # List built binaries
    if [[ -d "$BUILD_DIR" ]]; then
        log_info "Built binaries:"
        ls -lah "$BUILD_DIR"/ 2>/dev/null | grep -E "$BINARY_NAME" || true
    fi
}

# Parse arguments and run
parse_args "$@"
main