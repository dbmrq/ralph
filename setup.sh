#!/bin/bash
#
# Ralph Loop - Interactive Setup Wizard
#
# This script helps you set up Ralph Loop for your project.
# It will:
#   1. Ask for your project location
#   2. Detect or ask for project type
#   3. Configure build commands
#   4. Set up git branch
#   5. Create all necessary files
#   6. Provide instructions to run
#

set -e

# Script directory
RALPH_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Default values
DEFAULT_BRANCH="feature/ralph-automation"
DEFAULT_AGENT="cursor"
DEFAULT_MAX_ITERATIONS=50

#==============================================================================
# UTILITY FUNCTIONS
#==============================================================================

print_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}   $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
}

print_step() {
    echo -e "${CYAN}▶ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

ask() {
    local prompt="$1"
    local default="$2"
    local result

    # Write prompt to stderr so it's visible even when stdout is captured
    if [ -n "$default" ]; then
        echo -en "${BOLD}$prompt${NC} [${default}]: " >&2
    else
        echo -en "${BOLD}$prompt${NC}: " >&2
    fi

    # Read from /dev/tty to handle piped execution (e.g., bash <(...))
    read result </dev/tty

    if [ -z "$result" ] && [ -n "$default" ]; then
        result="$default"
    fi

    echo "$result"
}

ask_yes_no() {
    local prompt="$1"
    local default="$2"
    local result

    # Write prompt to stderr so it's visible even when stdout is captured
    if [ "$default" = "y" ]; then
        echo -en "${BOLD}$prompt${NC} [Y/n]: " >&2
    else
        echo -en "${BOLD}$prompt${NC} [y/N]: " >&2
    fi

    # Read from /dev/tty to handle piped execution (e.g., bash <(...))
    read result </dev/tty
    result=$(echo "$result" | tr '[:upper:]' '[:lower:]')

    if [ -z "$result" ]; then
        result="$default"
    fi

    [ "$result" = "y" ] || [ "$result" = "yes" ]
}

ask_choice() {
    local prompt="$1"
    shift
    local options=("$@")
    local choice

    echo -e "${BOLD}$prompt${NC}" >&2
    for i in "${!options[@]}"; do
        echo "  $((i+1))) ${options[$i]}" >&2
    done
    echo -en "Enter choice [1]: " >&2
    # Read from /dev/tty to handle piped execution (e.g., bash <(...))
    read choice </dev/tty

    if [ -z "$choice" ]; then
        choice=1
    fi

    echo "${options[$((choice-1))]}"
}

#==============================================================================
# AGENT DETECTION
#==============================================================================

# Check if Cursor CLI (agent command) is available
is_cursor_available() {
    command -v agent &> /dev/null
}

# Check if Augment CLI (auggie command) is available
is_auggie_available() {
    command -v auggie &> /dev/null
}

# Get list of available agents
get_available_agents() {
    local agents=()
    if is_cursor_available; then
        agents+=("cursor")
    fi
    if is_auggie_available; then
        agents+=("auggie")
    fi
    # Custom is always available as an option
    agents+=("custom")
    echo "${agents[@]}"
}

# Detect and select agent automatically or with user input
detect_or_select_agent() {
    local cursor_available=false
    local auggie_available=false

    if is_cursor_available; then
        cursor_available=true
    fi
    if is_auggie_available; then
        auggie_available=true
    fi

    # If both are available, let user choose
    if [ "$cursor_available" = true ] && [ "$auggie_available" = true ]; then
        echo -e "Multiple AI agents detected:" >&2
        echo -e "  ${GREEN}✓${NC} Cursor CLI (agent)" >&2
        echo -e "  ${GREEN}✓${NC} Augment CLI (auggie)" >&2
        echo "" >&2

        echo -e "${BOLD}Select AI agent:${NC}" >&2
        echo "  1) cursor" >&2
        echo "  2) auggie" >&2
        echo "  3) custom" >&2
        echo -en "Enter choice [1]: " >&2
        local choice
        read choice </dev/tty

        case "$choice" in
            2) echo "auggie" ;;
            3) echo "custom" ;;
            *) echo "cursor" ;;
        esac
        return
    fi

    # If only Cursor is available, auto-select it
    if [ "$cursor_available" = true ]; then
        echo -e "Detected: ${GREEN}✓${NC} Cursor CLI (agent command)" >&2
        echo "cursor"
        return
    fi

    # If only Auggie is available, auto-select it
    if [ "$auggie_available" = true ]; then
        echo -e "Detected: ${GREEN}✓${NC} Augment CLI (auggie command)" >&2
        echo "auggie"
        return
    fi

    # Neither is available - warn and ask
    echo -e "${YELLOW}⚠ No AI agent CLI detected!${NC}" >&2
    echo "" >&2
    echo "Ralph Loop requires one of the following:" >&2
    echo "  • Cursor CLI ('agent' command) - https://cursor.sh" >&2
    echo "  • Augment CLI ('auggie' command) - https://augmentcode.com" >&2
    echo "" >&2
    echo "You can still set up Ralph Loop and install an agent later." >&2
    echo "" >&2

    echo -e "${BOLD}Which agent do you plan to use?${NC}" >&2
    echo "  1) cursor (will need to install Cursor CLI)" >&2
    echo "  2) auggie (will need to install Augment CLI)" >&2
    echo "  3) custom (define your own in config.sh)" >&2
    echo -en "Enter choice [1]: " >&2
    local choice
    read choice </dev/tty

    case "$choice" in
        2) echo "auggie" ;;
        3) echo "custom" ;;
        *) echo "cursor" ;;
    esac
}

#==============================================================================
# PROJECT DETECTION
#==============================================================================

detect_project_type() {
    local project_dir="$1"

    # iOS/Xcode - check for Xcode projects, Swift packages, or XcodeGen
    if ls "$project_dir"/*.xcodeproj &>/dev/null || ls "$project_dir"/*.xcworkspace &>/dev/null; then
        echo "ios"
        return
    fi

    # XcodeGen project files
    if [ -f "$project_dir/project.yml" ] || [ -f "$project_dir/project.yaml" ]; then
        echo "ios"
        return
    fi

    # Swift Package Manager
    if [ -f "$project_dir/Package.swift" ]; then
        echo "ios"
        return
    fi

    # Check subdirectories for Xcode projects or XcodeGen
    if find "$project_dir" -maxdepth 2 -name "*.xcodeproj" 2>/dev/null | grep -q . || \
       find "$project_dir" -maxdepth 2 -name "*.xcworkspace" 2>/dev/null | grep -q . || \
       find "$project_dir" -maxdepth 2 -name "project.yml" 2>/dev/null | grep -q .; then
        echo "ios"
        return
    fi

    # React/Node
    if [ -f "$project_dir/package.json" ]; then
        if grep -q "react" "$project_dir/package.json" 2>/dev/null; then
            echo "web-react"
            return
        fi
        echo "node"
        return
    fi

    # Python
    if [ -f "$project_dir/requirements.txt" ] || [ -f "$project_dir/pyproject.toml" ] || [ -f "$project_dir/setup.py" ]; then
        echo "python"
        return
    fi

    # Go
    if [ -f "$project_dir/go.mod" ]; then
        echo "go"
        return
    fi

    # Rust
    if [ -f "$project_dir/Cargo.toml" ]; then
        echo "rust"
        return
    fi

    echo "unknown"
}

detect_xcode_schemes() {
    local project_dir="$1"
    local xcodeproj xcworkspace

    # Prefer workspace over project
    xcworkspace=$(find "$project_dir" -maxdepth 2 -name "*.xcworkspace" -type d 2>/dev/null | grep -v ".xcodeproj" | head -1)
    xcodeproj=$(find "$project_dir" -maxdepth 2 -name "*.xcodeproj" -type d 2>/dev/null | head -1)

    if [ -n "$xcworkspace" ]; then
        xcodebuild -workspace "$xcworkspace" -list 2>/dev/null | grep -A 100 "Schemes:" | tail -n +2 | grep -v "^$" | sed 's/^[[:space:]]*//' | grep -v "^$"
    elif [ -n "$xcodeproj" ]; then
        xcodebuild -project "$xcodeproj" -list 2>/dev/null | grep -A 100 "Schemes:" | tail -n +2 | grep -v "^$" | sed 's/^[[:space:]]*//' | grep -v "^$"
    fi
}

detect_xcode_project_dir() {
    local project_dir="$1"
    local xcodeproj xcworkspace project_yml

    # Check for XcodeGen first
    project_yml=$(find "$project_dir" -maxdepth 2 -name "project.yml" 2>/dev/null | head -1)
    if [ -n "$project_yml" ]; then
        dirname "$project_yml" | sed "s|^$project_dir/||" | sed "s|^$project_dir$|.|"
        return
    fi

    # Check for workspace
    xcworkspace=$(find "$project_dir" -maxdepth 2 -name "*.xcworkspace" -type d 2>/dev/null | grep -v ".xcodeproj" | head -1)
    if [ -n "$xcworkspace" ]; then
        dirname "$xcworkspace" | sed "s|^$project_dir/||" | sed "s|^$project_dir$|.|"
        return
    fi

    # Check for project
    xcodeproj=$(find "$project_dir" -maxdepth 2 -name "*.xcodeproj" -type d 2>/dev/null | head -1)
    if [ -n "$xcodeproj" ]; then
        dirname "$xcodeproj" | sed "s|^$project_dir/||" | sed "s|^$project_dir$|.|"
        return
    fi

    echo "."
}

#==============================================================================
# MAIN WIZARD
#==============================================================================

main() {
    print_header "Ralph Loop Setup Wizard"

    local step_num=1
    local project_path
    local project_type

    #--------------------------------------------------------------------------
    # Step 1: Project Path (skip if RALPH_PROJECT_PATH is set)
    #--------------------------------------------------------------------------
    if [ -n "$RALPH_PROJECT_PATH" ]; then
        # Path was passed from install.sh
        project_path="$RALPH_PROJECT_PATH"
        print_success "Project: $project_path"
        echo ""

        # Use pre-detected type if available
        if [ -n "$RALPH_PROJECT_TYPE" ]; then
            project_type="$RALPH_PROJECT_TYPE"
        fi
    else
        echo "This wizard will help you set up Ralph Loop for your project."
        echo "It will create a .ralph/ directory in your project with all"
        echo "the necessary configuration files."
        echo ""

        print_step "Step $step_num: Project Location"
        ((step_num++))
        echo ""

        while true; do
            project_path=$(ask "Enter path to your project" "")

            if [ -z "$project_path" ]; then
                print_error "Project path is required"
                continue
            fi

            # Expand ~ and make absolute
            project_path="${project_path/#\~/$HOME}"

            if [ ! -d "$project_path" ]; then
                print_error "Directory not found: $project_path"
                continue
            fi

            project_path="$(cd "$project_path" && pwd)"
            break
        done

        print_success "Project: $project_path"
        echo ""
    fi

    # Check if already set up
    if [ -d "$project_path/.ralph" ]; then
        if ! ask_yes_no "A .ralph/ directory already exists. Overwrite?" "n"; then
            echo "Setup cancelled."
            exit 0
        fi
    fi

    #--------------------------------------------------------------------------
    # Step 2: Project Type Detection (skip confirmation if already detected)
    #--------------------------------------------------------------------------
    if [ -z "$project_type" ]; then
        print_step "Step $step_num: Project Type"
        ((step_num++))
        echo ""

        local detected_type=$(detect_project_type "$project_path")

        if [ "$detected_type" != "unknown" ]; then
            echo -e "Detected project type: ${BOLD}$detected_type${NC}"
            if ask_yes_no "Is this correct?" "y"; then
                project_type="$detected_type"
            else
                project_type=$(ask_choice "Select project type:" "ios" "web-react" "python" "node" "go" "rust" "other")
            fi
        else
            project_type=$(ask_choice "Select project type:" "ios" "web-react" "python" "node" "go" "rust" "other")
        fi
    else
        echo -e "Project type: ${BOLD}$project_type${NC}"
    fi

    #--------------------------------------------------------------------------
    # Build Configuration (for supported types)
    #--------------------------------------------------------------------------
    local project_name=$(basename "$project_path")
    local xcode_scheme=""
    local xcode_project_dir="."
    local build_command=""
    local test_command=""
    local commit_scope=""

    if [ "$project_type" = "ios" ]; then
        print_step "Step $step_num: iOS Build Configuration"
        ((step_num++))
        echo ""

        # Auto-detect Xcode project directory
        xcode_project_dir=$(detect_xcode_project_dir "$project_path")
        echo -e "Xcode project directory: ${BOLD}$xcode_project_dir${NC}"
        echo ""

        # Detect available schemes and let user pick
        local schemes_list=$(detect_xcode_schemes "$project_path")

        if [ -n "$schemes_list" ]; then
            local scheme_count=$(echo "$schemes_list" | wc -l | tr -d ' ')

            if [ "$scheme_count" -eq 1 ]; then
                # Only one scheme, use it automatically
                xcode_scheme="$schemes_list"
                echo -e "Xcode scheme: ${BOLD}$xcode_scheme${NC} (auto-detected)"
            else
                # Multiple schemes, let user pick
                echo "Available schemes:"
                local i=1
                while IFS= read -r scheme; do
                    echo "  $i) $scheme" >&2
                    ((i++))
                done <<< "$schemes_list"
                echo "" >&2

                local scheme_choice
                echo -en "${BOLD}Select scheme [1]: ${NC}" >&2
                read scheme_choice </dev/tty

                if [ -z "$scheme_choice" ]; then
                    scheme_choice=1
                fi

                xcode_scheme=$(echo "$schemes_list" | sed -n "${scheme_choice}p")

                if [ -z "$xcode_scheme" ]; then
                    xcode_scheme=$(echo "$schemes_list" | head -1)
                fi
            fi
        else
            # No schemes detected, ask user
            xcode_scheme=$(ask "Xcode scheme" "$project_name")
        fi

        commit_scope="ios"
        print_success "Xcode scheme: $xcode_scheme"
        echo ""

    elif [ "$project_type" = "web-react" ] || [ "$project_type" = "node" ]; then
        print_step "Step $step_num: Node.js Build Configuration"
        ((step_num++))
        echo ""

        build_command=$(ask "Build command" "npm run build")
        test_command=$(ask "Test command" "npm test")
        commit_scope="web"
        echo ""

    elif [ "$project_type" = "python" ]; then
        print_step "Step $step_num: Python Build Configuration"
        ((step_num++))
        echo ""

        build_command=$(ask "Build/lint command" "python -m py_compile *.py")
        test_command=$(ask "Test command" "pytest")
        commit_scope="python"
        echo ""

    else
        print_step "Step $step_num: Build Configuration"
        ((step_num++))
        echo ""

        if ask_yes_no "Do you want to configure build verification?" "y"; then
            build_command=$(ask "Build command" "")
            test_command=$(ask "Test command (optional)" "")
        fi
        commit_scope=$(ask "Commit scope (e.g., 'ios', 'web', 'api')" "")
        echo ""
    fi

    #--------------------------------------------------------------------------
    # Agent Selection (auto-detect if possible)
    #--------------------------------------------------------------------------
    print_step "Step $step_num: AI Agent"
    ((step_num++))
    echo ""

    local agent_type=$(detect_or_select_agent)
    print_success "Agent: $agent_type"
    echo ""

    #--------------------------------------------------------------------------
    # Git Branch
    #--------------------------------------------------------------------------
    print_step "Step $step_num: Git Branch"
    ((step_num++))
    echo ""

    local current_branch=""
    local create_branch=false
    local branch_name=""

    cd "$project_path"

    if git rev-parse --git-dir &>/dev/null; then
        current_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
        echo -e "Current branch: ${BOLD}$current_branch${NC}"

        if [ "$current_branch" = "main" ] || [ "$current_branch" = "master" ]; then
            print_warning "You're on $current_branch. Ralph Loop requires a feature branch."
            if ask_yes_no "Create a new branch?" "y"; then
                create_branch=true
                branch_name=$(ask "Branch name" "$DEFAULT_BRANCH")
            else
                print_warning "You'll need to create a branch before running Ralph Loop."
            fi
        else
            print_success "Branch '$current_branch' looks good for Ralph Loop."
        fi
    else
        print_warning "Not a git repository. Ralph Loop works best with git."
        if ask_yes_no "Initialize git repository?" "y"; then
            git init >/dev/null 2>&1
            git checkout -b main >/dev/null 2>&1
            print_success "Initialized git repository"

            create_branch=true
            branch_name=$(ask "Branch name for Ralph Loop" "$DEFAULT_BRANCH")
        fi
    fi

    cd - > /dev/null
    echo ""

    #--------------------------------------------------------------------------
    # Task File
    #--------------------------------------------------------------------------
    print_step "Step $step_num: Task List"
    ((step_num++))
    echo ""

    local create_sample_tasks=false
    local existing_tasks=""

    echo "Ralph Loop needs a TASKS.md file with your task checklist."
    echo ""

    # Check for existing task files
    if [ -f "$project_path/.ralph/TASKS.md" ]; then
        existing_tasks="$project_path/.ralph/TASKS.md"
    elif [ -f "$project_path/TASKS.md" ]; then
        existing_tasks="$project_path/TASKS.md"
    fi

    if [ -n "$existing_tasks" ]; then
        echo "Found existing task file: $existing_tasks"
        if ! ask_yes_no "Use this file?" "y"; then
            create_sample_tasks=true
        fi
    else
        if ask_yes_no "Create a sample TASKS.md to get started?" "y"; then
            create_sample_tasks=true
        else
            print_warning "You'll need to create .ralph/TASKS.md before running."
        fi
    fi
    echo ""

    #--------------------------------------------------------------------------
    # Step 8: Advanced Options
    #--------------------------------------------------------------------------
    local max_iterations=$DEFAULT_MAX_ITERATIONS
    local build_gate_enabled=true

    if ask_yes_no "Configure advanced options?" "n"; then
        print_step "Advanced Options"
        echo ""

        max_iterations=$(ask "Maximum iterations per run" "$DEFAULT_MAX_ITERATIONS")

        if [ -n "$build_command" ] || [ "$project_type" = "ios" ]; then
            if ask_yes_no "Enable build verification between tasks?" "y"; then
                build_gate_enabled=true
            else
                build_gate_enabled=false
            fi
        fi
        echo ""
    fi

    #--------------------------------------------------------------------------
    # Create Files
    #--------------------------------------------------------------------------
    print_header "Creating Configuration Files"

    # Create .ralph directory
    mkdir -p "$project_path/.ralph"
    print_success "Created .ralph/ directory"

    # Create config.sh
    create_config_file "$project_path" "$project_name" "$project_type" "$agent_type" \
        "$xcode_scheme" "$xcode_project_dir" "$build_command" "$test_command" \
        "$commit_scope" "$max_iterations" "$build_gate_enabled"
    print_success "Created .ralph/config.sh"

    # Create project_prompt.txt (Level 3: Project-specific instructions)
    create_prompt_file "$project_path" "$project_type" "$project_name"
    print_success "Created .ralph/project_prompt.txt"

    # Create TASKS.md if requested
    if [ "$create_sample_tasks" = true ]; then
        create_tasks_file "$project_path" "$project_type"
        print_success "Created .ralph/TASKS.md"
    elif [ -n "$existing_tasks" ] && [ "$existing_tasks" != "$project_path/.ralph/TASKS.md" ]; then
        cp "$existing_tasks" "$project_path/.ralph/TASKS.md"
        print_success "Copied existing tasks to .ralph/TASKS.md"
    fi

    # Create branch if requested
    if [ "$create_branch" = true ] && [ -n "$branch_name" ]; then
        cd "$project_path"
        git checkout -b "$branch_name" 2>/dev/null || git checkout "$branch_name"
        print_success "Switched to branch: $branch_name"
        cd - > /dev/null
    fi

    #--------------------------------------------------------------------------
    # Final Instructions
    #--------------------------------------------------------------------------
    print_header "Setup Complete! 🎉"

    echo "Ralph Loop is now configured for your project."
    echo ""
    echo -e "${BOLD}Files created:${NC}"
    echo "  $project_path/.ralph/config.sh"
    echo "  $project_path/.ralph/project_prompt.txt"
    if [ "$create_sample_tasks" = true ] || [ -n "$existing_tasks" ]; then
        echo "  $project_path/.ralph/TASKS.md"
    fi
    echo ""

    echo -e "${BOLD}Next steps:${NC}"
    echo ""
    echo "  1. Review and customize your configuration:"
    echo -e "     ${CYAN}code $project_path/.ralph/${NC}"
    echo ""

    if [ "$create_sample_tasks" = true ]; then
        echo "  2. Edit TASKS.md with your actual tasks:"
        echo -e "     ${CYAN}code $project_path/.ralph/TASKS.md${NC}"
        echo ""
        echo "  3. Run Ralph Loop:"
    else
        echo "  2. Run Ralph Loop:"
    fi

    # Calculate relative path from project to ralph-loop
    local ralph_relative=$(python3 -c "import os.path; print(os.path.relpath('$RALPH_DIR', '$project_path'))" 2>/dev/null || echo "$RALPH_DIR")

    echo -e "     ${CYAN}cd $project_path${NC}"
    echo -e "     ${CYAN}$ralph_relative/ralph_loop.sh .${NC}"
    echo ""
    echo "  Or from the parent directory:"
    echo -e "     ${CYAN}$RALPH_DIR/ralph_loop.sh $project_path${NC}"
    echo ""

    # Only show warning if the selected agent is not installed
    if [ "$agent_type" = "cursor" ] && ! is_cursor_available; then
        echo -e "${YELLOW}⚠ Action required:${NC} Install Cursor CLI to use the 'agent' command."
        echo "  Visit: https://cursor.sh"
        echo ""
    elif [ "$agent_type" = "auggie" ] && ! is_auggie_available; then
        echo -e "${YELLOW}⚠ Action required:${NC} Install Augment CLI to use the 'auggie' command."
        echo "  Visit: https://augmentcode.com"
        echo ""
    elif [ "$agent_type" = "custom" ]; then
        echo -e "${YELLOW}Note:${NC} Define run_agent_custom() in .ralph/config.sh"
        echo ""
    else
        echo -e "${GREEN}✓ Agent '$agent_type' is ready to use!${NC}"
        echo ""
    fi

    echo "Happy automating! 🤖"
}

#==============================================================================
# FILE CREATION FUNCTIONS
#==============================================================================

create_config_file() {
    local project_path="$1"
    local project_name="$2"
    local project_type="$3"
    local agent_type="$4"
    local xcode_scheme="$5"
    local xcode_project_dir="$6"
    local build_command="$7"
    local test_command="$8"
    local commit_scope="$9"
    local max_iterations="${10}"
    local build_gate_enabled="${11}"

    local config_file="$project_path/.ralph/config.sh"

    cat > "$config_file" << 'HEADER'
#!/bin/bash
#
# Ralph Loop - Project Configuration
# Generated by setup wizard
#

HEADER

    # Map project type to platform type
    local platform_type="generic"
    case "$project_type" in
        ios) platform_type="ios" ;;
        python) platform_type="python" ;;
        web-react|node) platform_type="generic" ;;  # TODO: add web platform
        *) platform_type="generic" ;;
    esac

    cat >> "$config_file" << EOF
#==============================================================================
# PROJECT SETTINGS
#==============================================================================

PROJECT_NAME="$project_name"

# Platform type - determines which platform_prompt.txt to use
# Options: ios, python, generic (more can be added in templates/)
PLATFORM_TYPE="$platform_type"

#==============================================================================
# AGENT SETTINGS
#==============================================================================

AGENT_TYPE="$agent_type"

#==============================================================================
# LOOP SETTINGS
#==============================================================================

MAX_ITERATIONS=$max_iterations
PAUSE_SECONDS=5
MAX_CONSECUTIVE_FAILURES=3

#==============================================================================
# TEST RUN SETTINGS
#==============================================================================

# Pause after first N tasks for user verification
TEST_RUN_ENABLED=true
TEST_RUN_TASKS=2

#==============================================================================
# GIT SETTINGS
#==============================================================================

REQUIRE_BRANCH=true
ALLOWED_BRANCHES=""
AUTO_COMMIT=true
COMMIT_PREFIX="feat"
COMMIT_SCOPE="$commit_scope"

#==============================================================================
# BUILD SETTINGS
#==============================================================================

BUILD_GATE_ENABLED=$build_gate_enabled
BUILD_FIX_ATTEMPTS=1

EOF

    # Add build commands based on project type
    if [ "$project_type" = "ios" ]; then
        cat >> "$config_file" << EOF
#==============================================================================
# iOS BUILD CONFIGURATION
#==============================================================================

XCODE_SCHEME="$xcode_scheme"
XCODE_PROJECT_DIR="$xcode_project_dir"
SIMULATOR_DESTINATION="platform=iOS Simulator,name=iPhone 15"

project_build() {
    cd "\$XCODE_PROJECT_DIR"
    xcodebuild \\
        -scheme "\$XCODE_SCHEME" \\
        -destination "\$SIMULATOR_DESTINATION" \\
        build \\
        2>&1
}

project_test() {
    cd "\$XCODE_PROJECT_DIR"
    xcodebuild \\
        -scheme "\$XCODE_SCHEME" \\
        -destination "\$SIMULATOR_DESTINATION" \\
        test \\
        2>&1
}
EOF
    elif [ -n "$build_command" ]; then
        cat >> "$config_file" << EOF
#==============================================================================
# BUILD COMMANDS
#==============================================================================

project_build() {
    $build_command
}

EOF
        if [ -n "$test_command" ]; then
            cat >> "$config_file" << EOF
project_test() {
    $test_command
}
EOF
        fi
    fi
}

create_prompt_file() {
    local project_path="$1"
    local project_type="$2"
    local project_name="$3"

    # New 3-level system uses project_prompt.txt (not prompt.txt)
    local prompt_file="$project_path/.ralph/project_prompt.txt"

    # Check if template exists
    local template_file="$RALPH_DIR/templates/$project_type/project_prompt.txt"

    if [ -f "$template_file" ]; then
        # Use template and replace project name
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

create_tasks_file() {
    local project_path="$1"
    local project_type="$2"

    local tasks_file="$project_path/.ralph/TASKS.md"

    # Check if template exists
    local template_file="$RALPH_DIR/templates/$project_type/TASKS.md"

    if [ -f "$template_file" ]; then
        cp "$template_file" "$tasks_file"
    else
        # Create generic tasks file
        cat > "$tasks_file" << 'EOF'
# Task List

**Purpose:** Atomic tasks for automated agent completion
**Format:** `- [ ] TASK-ID: Description` (unchecked) or `- [x] TASK-ID: Description` (done)

---

## 🧪 Validation Task

- [ ] TEST-001: Verify the project builds successfully
  > Goal: Run the build command and ensure it passes
  > This validates the Ralph Loop setup is working correctly

---

## 📋 Your Tasks

<!-- Add your tasks here -->

- [ ] TASK-001: Your first task
  > Goal: Describe what this task should accomplish
  > Reference: Link to any relevant documentation

- [ ] TASK-002: Your second task
  > Goal: Describe what this task should accomplish

---

## Task Writing Tips

1. **One atomic change per task** - Completable in one agent run
2. **Clear success criteria** - Agent knows when it's done
3. **Include references** - Links to designs, specs, examples
4. **Order matters** - Dependencies come first
EOF
    fi
}

#==============================================================================
# RUN WIZARD
#==============================================================================

main
