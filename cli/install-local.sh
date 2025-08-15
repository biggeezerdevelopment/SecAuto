#!/bin/bash

# Quick install script for SecAuto CLI

set -e

echo "🚀 Installing SecAuto CLI..."

# Build the CLI
echo "📦 Building CLI..."
./scripts/build.sh

# Install to local bin directory
INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

echo "📂 Installing to $INSTALL_DIR"
cp build/secauto-cli "$INSTALL_DIR/secauto"
chmod +x "$INSTALL_DIR/secauto"

# Check if in PATH
if command -v secauto &> /dev/null; then
    echo "✅ SecAuto CLI installed successfully!"
    echo "🔧 Version: $(secauto --version)"
else
    echo "⚠️  SecAuto CLI installed but not in PATH"
    echo "💡 Add the following to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "   export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo ""
    echo "🔄 Then reload your shell or run: source ~/.bashrc"
fi

echo ""
echo "🎯 Quick start:"
echo "   secauto config set server http://localhost:9090"
echo "   secauto config set api-key your-api-key"
echo "   secauto health"