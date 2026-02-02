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
    
    if [ -n "$default" ]; then
        echo -en "${BOLD}$prompt${NC} [${default}]: "
    else
        echo -en "${BOLD}$prompt${NC}: "
    fi
    
    read result
    
    if [ -z "$result" ] && [ -n "$default" ]; then
        result="$default"
    fi
    
    echo "$result"
}

ask_yes_no() {
    local prompt="$1"
    local default="$2"
    local result
    
    if [ "$default" = "y" ]; then
        echo -en "${BOLD}$prompt${NC} [Y/n]: "
    else
        echo -en "${BOLD}$prompt${NC} [y/N]: "
    fi
    
    read result
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
    
    echo -e "${BOLD}$prompt${NC}"
    for i in "${!options[@]}"; do
        echo "  $((i+1))) ${options[$i]}"
    done
    echo -en "Enter choice [1]: "
    read choice
    
    if [ -z "$choice" ]; then
        choice=1
    fi
    
    echo "${options[$((choice-1))]}"
}

#==============================================================================
# PROJECT DETECTION
#==============================================================================

detect_project_type() {
    local project_dir="$1"
    
    # iOS/Xcode
    if ls "$project_dir"/*.xcodeproj &>/dev/null || ls "$project_dir"/*.xcworkspace &>/dev/null; then
        echo "ios"
        return
    fi
    
    # Check subdirectories for Xcode projects
    if find "$project_dir" -maxdepth 2 -name "*.xcodeproj" -o -name "*.xcworkspace" 2>/dev/null | head -1 | grep -q .; then
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

detect_xcode_scheme() {
    local project_dir="$1"
    local xcodeproj

    # Find xcodeproj
    xcodeproj=$(find "$project_dir" -maxdepth 2 -name "*.xcodeproj" -type d 2>/dev/null | head -1)

    if [ -n "$xcodeproj" ]; then
        # Try to get schemes
        local schemes=$(xcodebuild -project "$xcodeproj" -list 2>/dev/null | grep -A 100 "Schemes:" | tail -n +2 | grep -v "^$" | head -5 | sed 's/^[[:space:]]*//')
        echo "$schemes" | head -1
    fi
}

#==============================================================================
# MAIN WIZARD
#==============================================================================

main() {
    print_header "Ralph Loop Setup Wizard"

    echo "This wizard will help you set up Ralph Loop for your project."
    echo "It will create a .ralph/ directory in your project with all"
    echo "the necessary configuration files."
    echo ""

    #--------------------------------------------------------------------------
    # Step 1: Project Path
    #--------------------------------------------------------------------------
    print_step "Step 1: Project Location"
    echo ""

    local project_path
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

    # Check if already set up
    if [ -d "$project_path/.ralph" ]; then
        if ! ask_yes_no "A .ralph/ directory already exists. Overwrite?" "n"; then
            echo "Setup cancelled."
            exit 0
        fi
    fi

    #--------------------------------------------------------------------------
    # Step 2: Project Type Detection
    #--------------------------------------------------------------------------
    print_step "Step 2: Project Type"
    echo ""

    local detected_type=$(detect_project_type "$project_path")
    local project_type

    if [ "$detected_type" != "unknown" ]; then
        echo "Detected project type: ${BOLD}$detected_type${NC}"
        if ask_yes_no "Is this correct?" "y"; then
            project_type="$detected_type"
        else
            project_type=$(ask_choice "Select project type:" "ios" "web-react" "python" "node" "go" "rust" "other")
        fi
    else
        project_type=$(ask_choice "Select project type:" "ios" "web-react" "python" "node" "go" "rust" "other")
    fi

    print_success "Project type: $project_type"
    echo ""

    #--------------------------------------------------------------------------
    # Step 3: Project Name
    #--------------------------------------------------------------------------
    print_step "Step 3: Project Details"
    echo ""

    local default_name=$(basename "$project_path")
    local project_name=$(ask "Project name" "$default_name")

    print_success "Project name: $project_name"
    echo ""

    #--------------------------------------------------------------------------
    # Step 4: Build Configuration (for supported types)
    #--------------------------------------------------------------------------
    local xcode_scheme=""
    local xcode_project_dir="."
    local build_command=""
    local test_command=""
    local commit_scope=""

    if [ "$project_type" = "ios" ]; then
        print_step "Step 4: iOS Build Configuration"
        echo ""

        # Detect Xcode scheme
        local detected_scheme=$(detect_xcode_scheme "$project_path")
        if [ -n "$detected_scheme" ]; then
            xcode_scheme=$(ask "Xcode scheme" "$detected_scheme")
        else
            xcode_scheme=$(ask "Xcode scheme" "$default_name")
        fi

        # Find Xcode project directory
        local xcodeproj_path=$(find "$project_path" -maxdepth 2 -name "*.xcodeproj" -type d 2>/dev/null | head -1)
        if [ -n "$xcodeproj_path" ]; then
            xcode_project_dir=$(dirname "$xcodeproj_path")
            xcode_project_dir="${xcode_project_dir#$project_path/}"
            if [ "$xcode_project_dir" = "$(dirname "$xcodeproj_path")" ]; then
                xcode_project_dir="."
            fi
        fi
        xcode_project_dir=$(ask "Xcode project directory (relative to project root)" "$xcode_project_dir")

        commit_scope="ios"
        print_success "Xcode scheme: $xcode_scheme"
        echo ""

    elif [ "$project_type" = "web-react" ] || [ "$project_type" = "node" ]; then
        print_step "Step 4: Node.js Build Configuration"
        echo ""

        build_command=$(ask "Build command" "npm run build")
        test_command=$(ask "Test command" "npm test")
        commit_scope="web"
        echo ""

    elif [ "$project_type" = "python" ]; then
        print_step "Step 4: Python Build Configuration"
        echo ""

        build_command=$(ask "Build/lint command" "python -m py_compile *.py")
        test_command=$(ask "Test command" "pytest")
        commit_scope="python"
        echo ""

    else
        print_step "Step 4: Build Configuration"
        echo ""

        if ask_yes_no "Do you want to configure build verification?" "y"; then
            build_command=$(ask "Build command" "")
            test_command=$(ask "Test command (optional)" "")
        fi
        commit_scope=$(ask "Commit scope (e.g., 'ios', 'web', 'api')" "")
        echo ""
    fi

    #--------------------------------------------------------------------------
    # Step 5: Agent Selection
    #--------------------------------------------------------------------------
    print_step "Step 5: AI Agent"
    echo ""

    local agent_type=$(ask_choice "Select AI agent:" "cursor" "auggie" "custom")
    print_success "Agent: $agent_type"
    echo ""

    #--------------------------------------------------------------------------
    # Step 6: Git Branch
    #--------------------------------------------------------------------------
    print_step "Step 6: Git Branch"
    echo ""

    local current_branch=""
    local create_branch=false
    local branch_name=""

    cd "$project_path"

    if git rev-parse --git-dir &>/dev/null; then
        current_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
        echo "Current branch: ${BOLD}$current_branch${NC}"

        if [ "$current_branch" = "main" ] || [ "$current_branch" = "master" ]; then
            print_warning "You're on $current_branch. Ralph Loop requires a feature branch."
            if ask_yes_no "Create a new branch?" "y"; then
                create_branch=true
                branch_name=$(ask "Branch name" "$DEFAULT_BRANCH")
            else
                print_warning "You'll need to create a branch before running Ralph Loop."
            fi
        else
            echo "Branch looks good for Ralph Loop."
            if ask_yes_no "Create a different branch instead?" "n"; then
                create_branch=true
                branch_name=$(ask "Branch name" "$DEFAULT_BRANCH")
            fi
        fi
    else
        print_warning "Not a git repository. Ralph Loop works best with git."
        if ask_yes_no "Initialize git repository?" "y"; then
            git init
            git checkout -b main
            print_success "Initialized git repository"

            create_branch=true
            branch_name=$(ask "Branch name for Ralph Loop" "$DEFAULT_BRANCH")
        fi
    fi

    cd - > /dev/null
    echo ""

    #--------------------------------------------------------------------------
    # Step 7: Task File
    #--------------------------------------------------------------------------
    print_step "Step 7: Task List"
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

    if [ "$agent_type" = "cursor" ]; then
        echo -e "${YELLOW}Note:${NC} Make sure Cursor CLI is installed and 'agent' command is available."
    elif [ "$agent_type" = "auggie" ]; then
        echo -e "${YELLOW}Note:${NC} Make sure Augment CLI is installed and 'auggie' command is available."
    fi
    echo ""
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
