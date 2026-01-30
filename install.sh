#!/bin/bash
#
# Ralph Loop - One-liner Installer
#
# Usage: curl -fsSL https://raw.githubusercontent.com/W508153_wexinc/ralph-loop/main/install.sh | bash
#
# This script:
#   1. Asks where to install ralph-loop
#   2. Clones the repository
#   3. Runs the setup wizard
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

REPO_URL="https://github.com/W508153_wexinc/ralph-loop.git"

print_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}   $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_step() {
    echo -e "${CYAN}▶ $1${NC}"
}

# Check for required tools
check_requirements() {
    local missing=()
    
    if ! command -v git &> /dev/null; then
        missing+=("git")
    fi
    
    if [ ${#missing[@]} -gt 0 ]; then
        print_error "Missing required tools: ${missing[*]}"
        echo "Please install them and try again."
        exit 1
    fi
}

# Main installation
main() {
    print_header "Ralph Loop Installer"
    
    echo "This installer will set up Ralph Loop on your system."
    echo "Ralph Loop is an automated AI agent task runner that helps"
    echo "you automate repetitive development tasks."
    echo ""
    
    check_requirements
    
    # Ask where to install ralph-loop
    print_step "Step 1: Choose installation location"
    echo ""
    echo "Where should ralph-loop be installed?"
    echo "This should be a directory alongside your projects."
    echo ""
    echo "Examples:"
    echo "  ~/Code/ralph-loop"
    echo "  ~/Projects/ralph-loop"
    echo "  ../ralph-loop (relative to current directory)"
    echo ""
    
    # Suggest a default based on current directory
    local current_dir=$(pwd)
    local parent_dir=$(dirname "$current_dir")
    local default_install="$parent_dir/ralph-loop"
    
    echo -en "${BOLD}Installation path${NC} [$default_install]: "
    read install_path
    
    if [ -z "$install_path" ]; then
        install_path="$default_install"
    fi
    
    # Expand ~ if present
    install_path="${install_path/#\~/$HOME}"
    
    # Convert to absolute path
    if [[ ! "$install_path" = /* ]]; then
        install_path="$(cd "$(dirname "$install_path")" 2>/dev/null && pwd)/$(basename "$install_path")"
    fi
    
    # Check if directory already exists
    if [ -d "$install_path" ]; then
        if [ -d "$install_path/.git" ]; then
            print_success "ralph-loop already installed at $install_path"
            echo "Updating to latest version..."
            cd "$install_path"
            git pull origin main
        else
            print_error "Directory exists but is not a ralph-loop installation: $install_path"
            echo "Please choose a different location or remove the existing directory."
            exit 1
        fi
    else
        # Clone the repository
        print_step "Cloning ralph-loop repository..."
        echo ""
        
        # Create parent directory if needed
        mkdir -p "$(dirname "$install_path")"
        
        git clone "$REPO_URL" "$install_path"
        print_success "Cloned to $install_path"
    fi
    
    # Make scripts executable
    chmod +x "$install_path/ralph_loop.sh"
    chmod +x "$install_path/setup.sh"
    
    echo ""
    print_step "Step 2: Set up your project"
    echo ""
    echo "Now let's configure Ralph Loop for your project."
    echo ""
    
    # Run the setup wizard
    exec "$install_path/setup.sh"
}

# Run main
main

