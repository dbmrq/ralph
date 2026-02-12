#!/bin/bash
#
# lib/download.sh - File download functions library
#
# This file contains functions for downloading Ralph Loop files from GitHub.
# It is meant to be sourced by other scripts.
#
# Usage:
#   source "$(dirname "$0")/lib/download.sh"
#   or
#   source lib/download.sh
#
# Environment variables:
#   REPO_NAME - GitHub repository (default: "dbmrq/ralph")
#

# Guard to prevent double-sourcing
if [ -n "$__DOWNLOAD_SH_SOURCED__" ]; then
    return 0
fi
__DOWNLOAD_SH_SOURCED__=1

# Source common library
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

# Default repository name
REPO_NAME="${REPO_NAME:-dbmrq/ralph}"

#==============================================================================
# FILE DOWNLOAD FUNCTIONS
#==============================================================================

download_file() {
    local file_path="$1"
    local dest_path="$2"

    # Download file content from GitHub API
    gh api "repos/$REPO_NAME/contents/$file_path" --jq '.content' 2>/dev/null | base64 -d > "$dest_path" 2>/dev/null

    if [ $? -eq 0 ] && [ -s "$dest_path" ]; then
        return 0
    else
        return 1
    fi
}

download_ralph_files() {
    local ralph_dir="$1"

    print_step "Downloading Ralph Loop files..."
    echo ""

    # Create directory structure
    mkdir -p "$ralph_dir"
    mkdir -p "$ralph_dir/logs"
    mkdir -p "$ralph_dir/docs"

    # Download core files (from core/ directory in repo)
    local core_files=(
        "ralph_loop.sh"
        "base_prompt.txt"
    )

    for file in "${core_files[@]}"; do
        if download_file "core/$file" "$ralph_dir/$file"; then
            print_success "Downloaded $file"
        else
            print_error "Failed to download $file"
            return 1
        fi
    done

    # Make scripts executable
    chmod +x "$ralph_dir/ralph_loop.sh" 2>/dev/null

    echo ""
    return 0
}

download_template_files() {
    local ralph_dir="$1"

    # Create templates directory
    mkdir -p "$ralph_dir/templates"

    # Get list of files in templates directory (flat structure now)
    local template_files=$(gh api "repos/$REPO_NAME/contents/templates" --jq '.[] | select(.type == "file") | .name' 2>/dev/null)

    if [ -n "$template_files" ]; then
        while IFS= read -r file; do
            if download_file "templates/$file" "$ralph_dir/templates/$file"; then
                print_success "Downloaded templates/$file"
            fi
        done <<< "$template_files"
    fi
}

