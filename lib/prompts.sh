#!/bin/bash
#
# lib/prompts.sh - Prompt file generation library
#
# This file contains functions for generating prompt files for AI agents.
# It is meant to be sourced by other scripts.
#
# Usage:
#   source "$(dirname "$0")/lib/prompts.sh"
#   or
#   source lib/prompts.sh
#

# Guard to prevent double-sourcing
if [ -n "$__PROMPTS_SH_SOURCED__" ]; then
    return 0
fi
__PROMPTS_SH_SOURCED__=1

# Source common library
LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$LIB_DIR/common.sh"

#==============================================================================
# PROMPT FILE GENERATION
#==============================================================================

create_prompt_files() {
    local ralph_dir="$1"
    local project_name="$2"
    local platform_template="$ralph_dir/templates/platform_prompt.txt"
    local project_template="$ralph_dir/templates/project_prompt.txt"

    # Copy placeholder templates from downloaded templates directory
    if [ -f "$platform_template" ]; then
        cp "$platform_template" "$ralph_dir/platform_prompt.txt"
    else
        # Fallback: create minimal placeholder
        cat > "$ralph_dir/platform_prompt.txt" << 'EOF'
# Platform Guidelines

<!-- PLACEHOLDER: Configure platform-specific guidelines -->
EOF
    fi

    if [ -f "$project_template" ]; then
        cp "$project_template" "$ralph_dir/project_prompt.txt"
        # Replace project name placeholder
        sed -i '' "s/\[Your project name\]/$project_name/g" "$ralph_dir/project_prompt.txt" 2>/dev/null || \
            sed -i "s/\[Your project name\]/$project_name/g" "$ralph_dir/project_prompt.txt"
    else
        # Fallback: create minimal placeholder
        cat > "$ralph_dir/project_prompt.txt" << EOF
# Project-Specific Instructions

<!-- PLACEHOLDER: Configure project-specific instructions -->

**Project Name:** $project_name

Begin now. Find the next unchecked task and complete it.
EOF
    fi
}

create_docs_readme() {
    local ralph_dir="$1"
    local docs_dir="$ralph_dir/docs"

    # Create docs directory if it doesn't exist
    mkdir -p "$docs_dir"

    local readme_file="$docs_dir/README.md"

    cat > "$readme_file" << 'EOF'
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
EOF
}

