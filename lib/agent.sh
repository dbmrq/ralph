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

Analyze this project and help configure Ralph Loop by:

### 1. Fill in .ralph/project_prompt.txt
Look at the codebase and fill in each section:
- **Project Overview**: What does this project do?
- **Architecture**: What patterns are used? (MVVM, MVC, Clean Architecture, etc.)
- **Key Directories**: Map out the important folders and their purposes
- **Coding Standards**: What conventions does this project follow?
- **Testing Requirements**: How are tests structured and run?
- **Things to Avoid**: Any files or patterns to stay away from?

Be specific and accurate based on what you find in the code.

### 2. Configure build and test commands in .ralph/config.sh

This is CRITICAL. The script uses these functions to verify code quality.

Look at the existing config.sh and make sure these functions are properly defined:

#### project_build()
This function should build/compile the project. Examples:
- iOS/Xcode: \`xcodebuild -scheme MyApp -destination 'platform=iOS Simulator,...' build\`
- Swift Package: \`swift build\`
- Node.js: \`npm run build\`
- Python: \`python -m py_compile *.py\` or a linter

#### project_test()
This function should run the project's tests. Examples:
- iOS/Xcode: \`xcodebuild -scheme MyApp -destination 'platform=iOS Simulator,...' test\`
- Swift Package: \`swift test\`
- Node.js: \`npm test\`
- Python: \`pytest\`

Look at the project structure to determine the correct commands. Check for:
- package.json scripts
- Makefile targets
- project.yml or .xcodeproj for iOS
- setup.py or pyproject.toml for Python

If the project uses XcodeGen (has project.yml), you may need to run \`xcodegen\` first.

### 3. Do NOT modify .ralph/TASKS.md
Leave the task list for the user to fill in themselves.

### 4. Final Instructions
After completing the setup, tell the user:
- What you configured in project_prompt.txt
- What build command you set up
- What test command you set up
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

