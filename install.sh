#!/usr/bin/env bash
set -e

REPO="alitonbul/httpee"
BIN_NAME="httpee"

OS=$(uname -s)
ARCH=$(uname -m)

DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BIN_NAME}-${OS}-${ARCH}"

# Determine install dir (default to /usr/local/bin or ~/.local/bin)
INSTALL_DIR="/usr/local/bin"

if [ ! -w "$INSTALL_DIR" ]; then
    echo "Warning: No write permissions for $INSTALL_DIR"
    echo "Falling back to $HOME/.local/bin..."
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

echo "=> Downloading ${BIN_NAME} for ${OS} ${ARCH}..."
if ! curl -fsSL -o "${INSTALL_DIR}/${BIN_NAME}" "$DOWNLOAD_URL"; then
    echo "Error: Download failed. Please ensure the Github Release exists for your OS and Architecture: ${OS} ${ARCH}."
    exit 1
fi

echo "=> Applying execute permissions..."
chmod +x "${INSTALL_DIR}/${BIN_NAME}"

echo "=> Successfully installed ${BIN_NAME} to ${INSTALL_DIR}!"

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "Note: ${INSTALL_DIR} is not in your PATH."
    echo "Please add the following to your shell profile (.zshrc, .bashrc, etc.):"
    echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
    echo ""
fi

echo "You can now run '${BIN_NAME}'."
