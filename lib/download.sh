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
REPO_NAME="${REPO_NAME:-dbmrq/ralph-loop}"

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

    # Create docs README
    cat > "$ralph_dir/docs/README.md" << 'DOCS_EOF'
# Ralph Loop Documentation

Place additional documentation files here to provide context for AI agents.

## How It Works

Files in this directory are **not automatically included** in agent prompts,
but agents are instructed to check here when they need more context.

## Suggested Files

- `architecture.md` - High-level system architecture
- `api-reference.md` - API documentation
- `coding-standards.md` - Detailed coding conventions
- `dependencies.md` - Third-party libraries and their usage
- `troubleshooting.md` - Common issues and solutions

## Tips

- Keep files focused and concise
- Use clear headings for easy scanning
- Include code examples where helpful
- Update docs when making significant changes
DOCS_EOF

    # Download core files (from core/ directory in repo)
    local core_files=(
        "ralph_loop.sh"
        "base_prompt.txt"
        "validate.sh"
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
    chmod +x "$ralph_dir/validate.sh" 2>/dev/null

    echo ""
    return 0
}

download_template_files() {
    local ralph_dir="$1"
    local platform_type="$2"

    # Map to template directory
    local template_dir="templates/$platform_type"

    # Create templates directory
    mkdir -p "$ralph_dir/templates/$platform_type"

    # Get list of files in template directory
    local template_files=$(gh api "repos/$REPO_NAME/contents/$template_dir" --jq '.[].name' 2>/dev/null)

    if [ -n "$template_files" ]; then
        while IFS= read -r file; do
            if download_file "$template_dir/$file" "$ralph_dir/templates/$platform_type/$file"; then
                print_success "Downloaded templates/$platform_type/$file"
            fi
        done <<< "$template_files"
    fi

    # Also download generic templates as fallback
    if [ "$platform_type" != "generic" ]; then
        mkdir -p "$ralph_dir/templates/generic"
        local generic_files=$(gh api "repos/$REPO_NAME/contents/templates/generic" --jq '.[].name' 2>/dev/null)

        if [ -n "$generic_files" ]; then
            while IFS= read -r file; do
                download_file "templates/generic/$file" "$ralph_dir/templates/generic/$file" 2>/dev/null
            done <<< "$generic_files"
        fi
    fi
}

