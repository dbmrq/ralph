#!/bin/bash
#
# Ralph - Installation Script
#
# This script downloads and installs the ralph binary from GitHub releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/dbmrq/ralph/main/scripts/install.sh | bash
#
# Options:
#   RALPH_VERSION=v1.0.0  Install a specific version
#   RALPH_INSTALL_DIR=/usr/local/bin  Custom install directory
#

set -e

# Configuration
REPO="dbmrq/ralph"
BINARY_NAME="ralph"
INSTALL_DIR="${RALPH_INSTALL_DIR:-/usr/local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

print_step() { echo -e "${CYAN}▶ $1${NC}"; }
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }

# Detect OS and architecture
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux*)  os="Linux" ;;
        Darwin*) os="Darwin" ;;
        MINGW*|MSYS*|CYGWIN*) os="Windows" ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64) arch="x86_64" ;;
        arm64|aarch64) arch="arm64" ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

# Get the latest version from GitHub
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install_ralph() {
    local platform version archive_name archive_ext download_url tmp_dir

    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}   🤖 Ralph Installer${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo ""

    # Detect platform
    print_step "Detecting platform..."
    platform=$(detect_platform)
    print_success "Platform: $platform"

    # Determine version
    if [ -n "${RALPH_VERSION:-}" ]; then
        version="$RALPH_VERSION"
        print_success "Using specified version: $version"
    else
        print_step "Fetching latest version..."
        version=$(get_latest_version)
        if [ -z "$version" ]; then
            print_error "Failed to fetch latest version"
            exit 1
        fi
        print_success "Latest version: $version"
    fi

    # Determine archive extension
    if [[ "$platform" == *"Windows"* ]]; then
        archive_ext="zip"
    else
        archive_ext="tar.gz"
    fi

    # Build download URL
    archive_name="${BINARY_NAME}_${version#v}_${platform}.${archive_ext}"
    download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"

    # Create temp directory
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download archive
    print_step "Downloading ${archive_name}..."
    if ! curl -fsSL "$download_url" -o "${tmp_dir}/${archive_name}"; then
        print_error "Failed to download from: $download_url"
        echo ""
        echo "This could mean:"
        echo "  - The version doesn't exist"
        echo "  - Your platform isn't supported in this release"
        echo "  - Network issues"
        echo ""
        echo "Try installing with Go instead:"
        echo -e "  ${CYAN}go install github.com/${REPO}/cmd/ralph@latest${NC}"
        exit 1
    fi
    print_success "Downloaded successfully"

    # Extract archive
    print_step "Extracting archive..."
    cd "$tmp_dir"
    if [[ "$archive_ext" == "zip" ]]; then
        unzip -q "$archive_name"
    else
        tar -xzf "$archive_name"
    fi
    print_success "Extracted successfully"

    # Install binary
    print_step "Installing to ${INSTALL_DIR}..."
    if [ ! -d "$INSTALL_DIR" ]; then
        print_warning "Install directory doesn't exist, creating..."
        sudo mkdir -p "$INSTALL_DIR"
    fi

    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_NAME" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        sudo mv "$BINARY_NAME" "${INSTALL_DIR}/${BINARY_NAME}"
    fi
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    print_success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"

    # Verify installation
    echo ""
    if command -v ralph &> /dev/null; then
        print_success "Installation complete!"
        echo ""
        ralph --version
    else
        print_warning "Installation complete, but ralph is not in your PATH"
        echo ""
        echo "Add this to your shell profile:"
        echo -e "  ${CYAN}export PATH=\"\$PATH:${INSTALL_DIR}\"${NC}"
    fi

    echo ""
    echo -e "${BOLD}Quick Start:${NC}"
    echo -e "  ${CYAN}cd your-project${NC}"
    echo -e "  ${CYAN}ralph init${NC}      # Set up Ralph for your project"
    echo -e "  ${CYAN}ralph run${NC}       # Start the automation loop"
    echo ""
}

# Run installer
install_ralph

