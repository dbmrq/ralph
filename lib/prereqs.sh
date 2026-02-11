#!/bin/bash
#
# lib/prereqs.sh - Prerequisite Checking and Installation Library
#
# This library provides functions for checking and installing prerequisites
# needed for Ralph Loop, including macOS detection, Homebrew, GitHub CLI,
# and GitHub authentication.
#
# Usage:
#   source lib/prereqs.sh
#   check_and_install_prerequisites
#

# Guard against double-sourcing
if [[ -n "${RALPH_PREREQS_SOURCED}" ]]; then
    return 0
fi
RALPH_PREREQS_SOURCED=1

# Source common utilities
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

#==============================================================================
# PREREQUISITE FUNCTIONS
#==============================================================================

check_macos() {
    if [[ "$(uname)" != "Darwin" ]]; then
        print_warning "This script is optimized for macOS."
        print_warning "On other systems, please install git and gh manually."
        return 1
    fi
    return 0
}

install_homebrew() {
    print_step "Installing Homebrew..."
    echo ""
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

    # Add Homebrew to PATH for this session
    if [[ -f "/opt/homebrew/bin/brew" ]]; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
    elif [[ -f "/usr/local/bin/brew" ]]; then
        eval "$(/usr/local/bin/brew shellenv)"
    fi

    print_success "Homebrew installed!"
}

install_gh() {
    print_step "Installing GitHub CLI..."
    brew install gh
    print_success "GitHub CLI installed!"
}

authenticate_gh() {
    print_step "Authenticating with GitHub..."
    echo ""
    echo "This will open a browser to authenticate with GitHub."
    echo "Please follow the prompts to complete authentication."
    echo ""
    gh auth login
    print_success "GitHub CLI authenticated!"
}

check_and_install_prerequisites() {
    print_subheader "Checking Prerequisites"

    local is_macos=true
    check_macos || is_macos=false

    # Check for git
    if ! command -v git &> /dev/null; then
        print_error "Git is not installed."
        if $is_macos; then
            echo "Git comes with Xcode Command Line Tools."
            if ask_yes_no "Install Xcode Command Line Tools?" "y"; then
                xcode-select --install
                echo ""
                print_warning "Please complete the installation and run this script again."
                exit 0
            else
                print_error "Git is required. Please install it and try again."
                exit 1
            fi
        else
            print_error "Please install git and try again."
            exit 1
        fi
    else
        print_success "Git is installed"
    fi

    # Check for Homebrew (macOS only)
    if $is_macos; then
        if ! command -v brew &> /dev/null; then
            print_warning "Homebrew is not installed."
            echo ""
            echo "Homebrew is a package manager for macOS that makes it easy to"
            echo "install developer tools like the GitHub CLI."
            echo ""
            if ask_yes_no "Install Homebrew?" "y"; then
                install_homebrew
            else
                print_warning "Skipping Homebrew. You may need to install gh manually."
            fi
        else
            print_success "Homebrew is installed"
        fi
    fi

    # Check for GitHub CLI
    if ! command -v gh &> /dev/null; then
        print_warning "GitHub CLI (gh) is not installed."
        echo ""
        echo "The GitHub CLI is needed to access the private ralph-loop repository"
        echo "and for git operations."
        echo ""
        if command -v brew &> /dev/null; then
            if ask_yes_no "Install GitHub CLI via Homebrew?" "y"; then
                install_gh
            else
                print_error "GitHub CLI is required for private repo access."
                exit 1
            fi
        else
            print_error "Please install the GitHub CLI manually: https://cli.github.com/"
            exit 1
        fi
    else
        print_success "GitHub CLI is installed"
    fi

    # Check if gh is authenticated
    if ! gh auth status &> /dev/null; then
        print_warning "GitHub CLI is not authenticated."
        echo ""
        echo "You need to authenticate with GitHub to access the repository."
        echo ""
        if ask_yes_no "Authenticate now?" "y"; then
            authenticate_gh
        else
            print_error "GitHub authentication is required."
            exit 1
        fi
    else
        print_success "GitHub CLI is authenticated"
    fi

    echo ""
    print_success "All prerequisites are ready!"
}

