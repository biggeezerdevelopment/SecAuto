#!/bin/bash

# SecAuto UV-based Virtual Environment Installation Script
# Creates a default Python virtual environment with required packages using UV

set -e  # Exit on any error

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
VENV_PATH="$PROJECT_ROOT/Venv"
REQUIREMENTS_FILE="$SCRIPT_DIR/requirements.txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Check if UV is available
check_uv() {
    if command -v uv &> /dev/null; then
        log_info "Found UV: $(uv --version)"
    else
        log_error "UV is required but not found."
        log_info "Install UV with: curl -LsSf https://astral.sh/uv/install.sh | sh"
        log_info "Or visit: https://docs.astral.sh/uv/getting-started/installation/"
        exit 1
    fi
}

# Create virtual environment with UV
create_venv() {
    log_info "Creating virtual environment at: $VENV_PATH"
    
    if [ -d "$VENV_PATH" ]; then
        log_warning "Virtual environment already exists. Removing..."
        rm -rf "$VENV_PATH"
    fi
    
    uv venv "$VENV_PATH"
    log_success "Virtual environment created successfully with UV"
}

# Install requirements with UV
install_requirements() {
    if [ ! -f "$REQUIREMENTS_FILE" ]; then
        log_error "Requirements file not found: $REQUIREMENTS_FILE"
        exit 1
    fi
    
    log_info "Installing packages from requirements.txt using UV..."
    log_info "UV is much faster than pip - this should only take seconds..."
    
    # UV automatically handles virtual environment activation
    uv pip install --python "$VENV_PATH/bin/python" -r "$REQUIREMENTS_FILE"
    
    log_success "All packages installed successfully with UV"
}

# Create activation script
create_activation_script() {
    ACTIVATE_SCRIPT="$PROJECT_ROOT/activate_venv.sh"
    
    cat > "$ACTIVATE_SCRIPT" << 'EOF'
#!/bin/bash
# SecAuto Virtual Environment Activation Script

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_PATH="$SCRIPT_DIR/Venv"

if [ ! -d "$VENV_PATH" ]; then
    echo "Virtual environment not found at: $VENV_PATH"
    echo "Please run EnvInstall/install_venv.sh first"
    exit 1
fi

echo "Activating SecAuto virtual environment..."
source "$VENV_PATH/bin/activate"
echo "Virtual environment activated. Type 'deactivate' to exit."
EOF
    
    chmod +x "$ACTIVATE_SCRIPT"
    log_success "Created activation script: $ACTIVATE_SCRIPT"
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."
    
    # Check if venv was created
    if [ ! -d "$VENV_PATH" ]; then
        log_error "Virtual environment directory not found"
        exit 1
    fi
    
    # Check if python works in venv
    if [ ! -f "$VENV_PATH/bin/python" ]; then
        log_error "Python interpreter not found in virtual environment"
        exit 1
    fi
    
    # Test a few key packages
    log_info "Testing package imports..."
    "$VENV_PATH/bin/python" -c "import requests; print(f'requests: {requests.__version__}')" || {
        log_error "Failed to import requests package"
        exit 1
    }
    
    "$VENV_PATH/bin/python" -c "import pandas; print(f'pandas: {pandas.__version__}')" || {
        log_warning "pandas import failed - this may be normal for minimal installs"
    }
    
    log_success "Installation verification completed"
}

# Print usage information
print_usage() {
    echo ""
    echo "SecAuto Virtual Environment Setup Complete!"
    echo "======================================="
    echo ""
    echo "To activate the virtual environment:"
    echo "  source $PROJECT_ROOT/activate_venv.sh"
    echo "  # or manually:"
    echo "  source $VENV_PATH/bin/activate"
    echo ""
    echo "To deactivate:"
    echo "  deactivate"
    echo ""
    echo "Virtual environment location: $VENV_PATH"
    echo "Python interpreter: $VENV_PATH/bin/python"
    echo "Pip: $VENV_PATH/bin/pip"
    echo ""
}

# Main execution
main() {
    log_info "Starting SecAuto UV-based virtual environment installation..."
    log_info "Project root: $PROJECT_ROOT"
    
    check_uv
    create_venv
    install_requirements
    create_activation_script
    verify_installation
    print_usage
    
    log_success "SecAuto UV-based virtual environment installation completed successfully!"
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi