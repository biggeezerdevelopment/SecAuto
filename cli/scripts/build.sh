#!/bin/bash

# SecAuto CLI Build Script

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
BINARY_NAME="secauto-cli"

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
SecAuto CLI Build Script

Usage: $0 [OPTIONS]

Options:
    -h, --help          Show this help message
    -v, --version       Set version (default: $VERSION)
    -o, --output        Set output directory (default: $BUILD_DIR)
    -n, --name          Set binary name (default: $BINARY_NAME)
    --all               Build for all platforms
    --linux             Build for Linux
    --windows           Build for Windows
    --darwin            Build for macOS
    --arm64             Build for ARM64 architecture
    --clean             Clean build directory before building

Examples:
    $0                  # Build for current platform
    $0 --all            # Build for all platforms
    $0 --linux --arm64  # Build for Linux ARM64
    $0 --clean --all    # Clean and build for all platforms

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
            --linux)
                BUILD_LINUX=true
                shift
                ;;
            --windows)
                BUILD_WINDOWS=true
                shift
                ;;
            --darwin)
                BUILD_DARWIN=true
                shift
                ;;
            --arm64)
                BUILD_ARM64=true
                shift
                ;;
            --clean)
                CLEAN=true
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

# Clean build directory
clean_build() {
    if [[ "$CLEAN" == "true" ]]; then
        print_status "Cleaning build directory..."
        rm -rf "$BUILD_DIR"
    fi
}

# Create build directory
create_build_dir() {
    if [[ ! -d "$BUILD_DIR" ]]; then
        print_status "Creating build directory: $BUILD_DIR"
        mkdir -p "$BUILD_DIR"
    fi
}

# Build for specific platform
build_platform() {
    local goos=$1
    local goarch=$2
    local suffix=$3
    
    local output_name="${BINARY_NAME}"
    if [[ -n "$suffix" ]]; then
        output_name="${BINARY_NAME}-${suffix}"
    fi
    
    if [[ "$goos" == "windows" ]]; then
        output_name="${output_name}.exe"
    fi
    
    local output_path="${BUILD_DIR}/${output_name}"
    
    print_status "Building for ${goos}/${goarch}..."
    
    CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
        -ldflags="-s -w -X main.version=$VERSION" \
        -o "$output_path" \
        .
    
    if [[ $? -eq 0 ]]; then
        local size=$(ls -lah "$output_path" | awk '{print $5}')
        print_success "Built ${output_path} (${size})"
    else
        print_error "Failed to build for ${goos}/${goarch}"
        exit 1
    fi
}

# Build current platform
build_current() {
    local goos=$(go env GOOS)
    local goarch=$(go env GOARCH)
    build_platform "$goos" "$goarch" ""
}

# Build all platforms
build_all_platforms() {
    print_status "Building for all platforms..."
    
    # Linux
    build_platform "linux" "amd64" "linux-amd64"
    build_platform "linux" "arm64" "linux-arm64"
    
    # Windows
    build_platform "windows" "amd64" "windows-amd64"
    build_platform "windows" "arm64" "windows-arm64"
    
    # macOS
    build_platform "darwin" "amd64" "darwin-amd64"
    build_platform "darwin" "arm64" "darwin-arm64"
}

# Validate Go environment
validate_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    print_status "Go version: $(go version)"
}

# Check dependencies
check_deps() {
    print_status "Checking dependencies..."
    go mod tidy
    if [[ $? -ne 0 ]]; then
        print_error "Failed to tidy Go modules"
        exit 1
    fi
}

# Main function
main() {
    print_status "SecAuto CLI Build Script"
    print_status "Version: $VERSION"
    
    validate_go
    check_deps
    clean_build
    create_build_dir
    
    if [[ "$BUILD_ALL" == "true" ]]; then
        build_all_platforms
    elif [[ "$BUILD_LINUX" == "true" ]]; then
        if [[ "$BUILD_ARM64" == "true" ]]; then
            build_platform "linux" "arm64" "linux-arm64"
        else
            build_platform "linux" "amd64" "linux-amd64"
        fi
    elif [[ "$BUILD_WINDOWS" == "true" ]]; then
        if [[ "$BUILD_ARM64" == "true" ]]; then
            build_platform "windows" "arm64" "windows-arm64"
        else
            build_platform "windows" "amd64" "windows-amd64"
        fi
    elif [[ "$BUILD_DARWIN" == "true" ]]; then
        if [[ "$BUILD_ARM64" == "true" ]]; then
            build_platform "darwin" "arm64" "darwin-arm64"
        else
            build_platform "darwin" "amd64" "darwin-amd64"
        fi
    else
        build_current
    fi
    
    print_success "Build completed successfully!"
    print_status "Output directory: $BUILD_DIR"
    
    # List built binaries
    if [[ -d "$BUILD_DIR" ]]; then
        print_status "Built binaries:"
        ls -lah "$BUILD_DIR"/ | grep -E "secauto-cli|\.exe$"
    fi
}

# Parse arguments and run
parse_args "$@"
main