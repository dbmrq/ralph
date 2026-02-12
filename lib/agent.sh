#!/bin/bash
#
# Ralph Loop - AI Agent Library
#
# This library provides functions for AI agent detection and setup.
# It should be sourced by other scripts, not executed directly.
#
# Usage:
#   source "$(dirname "${BASH_SOURCE[0]}")/agent.sh"
#

# Guard against double-sourcing
if [ -n "$__RALPH_AGENT_SOURCED__" ]; then
    return 0
fi
__RALPH_AGENT_SOURCED__=1

# Source common utilities
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

#==============================================================================
# AGENT DETECTION
#==============================================================================

is_cursor_available() {
    command -v agent &> /dev/null
}

is_auggie_available() {
    command -v auggie &> /dev/null
}

detect_or_select_agent() {
    local cursor_installed=false
    local auggie_installed=false

    if is_cursor_available; then
        cursor_installed=true
    fi

    if is_auggie_available; then
        auggie_installed=true
    fi

    # If only one is available, auto-select it
    if [ "$cursor_installed" = true ] && [ "$auggie_installed" = false ]; then
        echo -e "Auto-detected: ${BOLD}Cursor${NC} (agent CLI found)" >&2
        echo "cursor"
        return
    fi

    if [ "$auggie_installed" = true ] && [ "$cursor_installed" = false ]; then
        echo -e "Auto-detected: ${BOLD}Augment${NC} (auggie CLI found)" >&2
        echo "auggie"
        return
    fi

    # If both are available, let user choose between them
    if [ "$cursor_installed" = true ] && [ "$auggie_installed" = true ]; then
        echo "Both Cursor and Augment are installed." >&2
        local choice=$(ask_choice "Which AI agent do you want to use?" "cursor" "auggie")
        echo "$choice"
        return
    fi

    # Neither is installed - let user choose anyway (they can install later)
    print_warning "No AI agent CLI detected." >&2
    echo "You can still set up Ralph Loop - just install the agent before running." >&2
    echo "" >&2

    local choice=$(ask_choice "Which AI agent will you use?" "cursor" "auggie" "custom")
    echo "$choice"
}

#==============================================================================
# AI SETUP ASSISTANT
#==============================================================================

run_ai_setup_assistant() {
    local project_path="$1"
    local agent_type="$2"

    if [ -z "$project_path" ] || [ -z "$agent_type" ]; then
        print_error "Usage: run_ai_setup_assistant <project_path> <agent_type>"
        return 1
    fi

    if ask_yes_no "Run AI setup assistant now?" "y"; then
        echo ""
        print_step "Starting AI setup assistant..."
        echo ""

        cd "$project_path" || return 1

        # Create the setup prompt
        local setup_prompt="You are helping set up Ralph Loop, an automated AI agent task runner.

## Your Task

Analyze this project and configure all placeholder files in .ralph/:

### 1. Configure .ralph/build.sh (CRITICAL)

Edit .ralph/build.sh to build/compile the project.
The script must exit 0 on success and non-zero on failure.
Remove the placeholder block and add the actual build command.

Look at the project and configure the appropriate build command. Examples:
- iOS/Xcode: \`xcodebuild -scheme MyApp -destination 'platform=iOS Simulator,...' build\`
- Swift Package: \`swift build\`
- Node.js: \`npm run build\`
- Python: \`ruff check .\` or \`python -m py_compile *.py\`

Check for project.yml (XcodeGen), Package.swift (SPM), package.json, Makefile, etc.

**Run the build script to verify it works**: \`.ralph/build.sh\`

### 2. Configure .ralph/test.sh (CRITICAL)

Edit .ralph/test.sh to run the project's tests.
The script must exit 0 on success and non-zero on failure.
Remove the placeholder block and add the actual test command.

Examples:
- iOS/Xcode: \`xcodebuild -scheme MyApp -destination 'platform=iOS Simulator,...' test\`
- Swift Package: \`swift test\`
- Node.js: \`npm test\`
- Python: \`pytest\`

**Run the test script to verify it works**: \`.ralph/test.sh\`

### 3. Configure .ralph/platform_prompt.txt

Replace the placeholder content with platform-specific guidelines for this project.
Include best practices for the language/framework (Swift, Python, React, etc.).
Remove the \`<!-- PLACEHOLDER:\` comment block when done.

### 4. Configure .ralph/project_prompt.txt

Replace the placeholder content with project-specific instructions:
- **Project Overview**: What does this project do?
- **Architecture**: What patterns are used? (MVVM, MVC, Clean Architecture, etc.)
- **Key Directories**: Map out the important folders and their purposes
- **Coding Standards**: What conventions does this project follow?
- **Things to Avoid**: Any files or patterns to stay away from?

Remove the \`<!-- PLACEHOLDER:\` comment block when done.

### 5. Do NOT modify .ralph/TASKS.md
Leave the task list for the user to fill in themselves.

### 6. Final Summary
After completing the setup, tell the user:
- What build command you configured (and whether it passed)
- What test command you configured (and whether tests passed)
- What platform guidelines you added
- What project-specific instructions you added
- How to add tasks to .ralph/TASKS.md
- How to run Ralph Loop: .ralph/ralph_loop.sh

Output DONE when finished."

        # Run the agent
        if [ "$agent_type" = "cursor" ]; then
            agent "$setup_prompt"
        elif [ "$agent_type" = "auggie" ]; then
            auggie "$setup_prompt"
        fi

        echo ""
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}Setup complete! 🎉${NC}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    else
        echo ""
        echo "You can run the setup assistant later with:"
        echo -e "  ${CYAN}cd $project_path${NC}"
        if [ "$agent_type" = "cursor" ]; then
            echo -e "  ${CYAN}agent \"Help me set up Ralph Loop by filling in .ralph/project_prompt.txt\"${NC}"
        else
            echo -e "  ${CYAN}auggie \"Help me set up Ralph Loop by filling in .ralph/project_prompt.txt\"${NC}"
        fi
        echo ""
        echo "Or start Ralph Loop directly:"
        echo -e "  ${CYAN}.ralph/ralph_loop.sh${NC}"
    fi
}

