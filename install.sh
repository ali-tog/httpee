#!/usr/bin/env bash
set -e

REPO="ali-tog/httpee"
BIN_NAME="httpee"

OS=$(uname -s)
ARCH=$(uname -m)

DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BIN_NAME}-${OS}-${ARCH}"

# Determine install dir
INSTALL_DIR=""

# 1. Prefer standard bin directories that are ALREADY in the user's PATH and writable
for dir in "/usr/local/bin" "$HOME/.local/bin" "$HOME/bin"; do
    if [[ ":$PATH:" == *":$dir:"* ]] && [ -w "$dir" ]; then
        INSTALL_DIR="$dir"
        break
    fi
done

# 2. If no standard directory in PATH is writable, default to /usr/local/bin if writable
if [ -z "$INSTALL_DIR" ] && [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
fi

# 3. Fallback to ~/.local/bin if all else fails
if [ -z "$INSTALL_DIR" ]; then
    echo "Warning: No standard writable directories found in PATH."
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
    
    # Attempt to automatically add it to the shell profile
    SHELL_PROFILE=""
    if [[ "$SHELL" == *"zsh"* ]]; then
        SHELL_PROFILE="$HOME/.zshrc"
    elif [[ "$SHELL" == *"bash"* ]]; then
        SHELL_PROFILE="$HOME/.bashrc"
    fi

    if [ -n "$SHELL_PROFILE" ] && [ -w "$SHELL_PROFILE" ]; then
        echo "=> Automatically adding $INSTALL_DIR to your PATH in $SHELL_PROFILE"
        echo "" >> "$SHELL_PROFILE"
        echo "# httpee installer" >> "$SHELL_PROFILE"
        echo "export PATH=\"\$PATH:${INSTALL_DIR}\"" >> "$SHELL_PROFILE"
        echo "=> Please run 'source $SHELL_PROFILE' or restart your terminal to apply the changes."
    else
        echo "Please add the following to your shell profile (.zshrc, .bashrc, etc.):"
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
    fi
    echo ""
fi

echo "You can now run '${BIN_NAME}'."
