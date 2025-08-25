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
