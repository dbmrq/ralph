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

create_prompt_file() {
    local ralph_dir="$1"
    local project_type="$2"
    local project_name="$3"

    local prompt_file="$ralph_dir/project_prompt.txt"

    # Check if template exists for this project type
    local template_file="$LIB_DIR/../templates/$project_type/project_prompt.txt"

    if [ -f "$template_file" ]; then
        # Use template and replace project name placeholders
        sed "s/\[Your App Name\]/$project_name/g; s/My iOS App/$project_name/g; s/MyApp/$project_name/g" "$template_file" > "$prompt_file"
    else
        # Create generic project prompt
        cat > "$prompt_file" << EOF
# $project_name - Project-Specific Instructions

<!--
This file contains instructions specific to YOUR project.
The platform-level guidelines are loaded automatically based on PLATFORM_TYPE in config.sh.
Edit this file to describe your project's unique requirements.
-->

## Project Overview

Project Name: $project_name
Description: [Brief description of the project]

## Project Structure

<!-- Describe your specific folder structure -->

## Key Files & Patterns

<!-- Point the agent to important files to reference -->

## Coding Conventions

<!-- Any project-specific conventions -->

## Things to Avoid

<!-- Warn the agent about pitfalls -->

## Reference Materials

<!-- Links to docs, designs, etc. -->

---

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

