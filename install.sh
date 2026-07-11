#!/usr/bin/env bash
# install.sh — OpenConveneCLI installer for Linux and macOS
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.sh | bash
#
# This script downloads the latest release binary from GitHub and installs it
# to /usr/local/bin (or ~/.local/bin if /usr/local/bin is not writable).

set -euo pipefail

REPO="masteryee-labs/open-convene-cli"
BINARY_NAME="openconvene"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }

# Detect OS
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux*)  OS="linux";;
    Darwin*) OS="darwin";;
    *)       error "Unsupported OS: $OS (only Linux and macOS are supported)";;
esac

case "$ARCH" in
    x86_64|amd64) ARCH="amd64";;
    arm64|aarch64) ARCH="arm64";;
    *)             error "Unsupported architecture: $ARCH (only amd64 and arm64 are supported)";;
esac

info "Detected: ${OS}/${ARCH}"

# Get latest release tag from GitHub API
info "Fetching latest release version..."
LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    error "Failed to determine latest release version. Please check your internet connection."
fi

info "Latest version: ${LATEST_TAG}"

# Construct download URL
ASSET_NAME="${BINARY_NAME}-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${ASSET_NAME}"

# Create temp directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

info "Downloading ${ASSET_NAME}..."
if ! curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${BINARY_NAME}"; then
    error "Failed to download ${DOWNLOAD_URL}\nPlease check that the release asset exists."
fi

# Make executable
chmod +x "${TMP_DIR}/${BINARY_NAME}"

# Determine install directory
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    INSTALL_DIR="${HOME}/.local/bin"
    info "Install dir /usr/local/bin not writable, using ${INSTALL_DIR}"
    mkdir -p "$INSTALL_DIR"
fi

# Install
info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"

success "OpenConveneCLI ${LATEST_TAG} installed successfully!"

# Check PATH
case ":${PATH}:" in
    *":${INSTALL_DIR}:"*)
        ;;
    *)
        warn "${INSTALL_DIR} is not in your PATH."
        warn "Add this line to your ~/.bashrc or ~/.zshrc:"
        echo ""
        echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
        echo ""
        ;;
esac

echo ""
echo "  Get started:"
echo "    openconvene detect    # detect installed AI CLIs"
echo "    openconvene init      # generate config"
echo "    openconvene           # enter interactive REPL"
echo ""
success "Done!"
