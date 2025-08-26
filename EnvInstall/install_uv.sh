#!/bin/bash

# Install uv - Python package and project manager
# https://github.com/astral-sh/uv

set -e

echo "Installing uv..."

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture names
case "$ARCH" in
    x86_64)
        ARCH="x86_64"
        ;;
    aarch64|arm64)
        ARCH="aarch64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Map OS names for uv
case "$OS" in
    linux)
        PLATFORM="unknown-linux-gnu"
        ;;
    darwin)
        PLATFORM="apple-darwin"
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

# Download and install using the official installer
curl -LsSf https://astral.sh/uv/install.sh | sh

echo "uv installed successfully!"
echo "Please restart your shell or run: source \$HOME/.cargo/env"
echo "To verify installation, run: uv --version"