#!/bin/bash
#
# Ralph Loop - Universal Installer & Setup Wizard
#
# This is THE single entry point for Ralph Loop. It handles everything:
#   - Installing prerequisites (Homebrew, GitHub CLI)
#   - Downloading Ralph Loop files into your project
#   - Configuring the project (type, agent, build commands)
#   - Running the AI setup assistant
#
# One-liner install (run from your project directory):
#   bash <(curl -fsSL https://raw.githubusercontent.com/dbmrq/ralph-loop/main/install.sh)
#
# Or with GitHub CLI:
#   bash <(gh api repos/dbmrq/ralph/contents/install.sh --jq '.content' | base64 -d)
#

set -e

#==============================================================================
# BOOTSTRAP: Determine script location and source libraries
#==============================================================================

# When run via curl/pipe, we need to download the libraries first
# When run locally, we can source them directly

SCRIPT_SOURCE="${BASH_SOURCE[0]:-}"
if [ -n "$SCRIPT_SOURCE" ] && [ -f "$SCRIPT_SOURCE" ]; then
    # Running from a file - use local libraries
    RALPH_REPO_DIR="$(cd "$(dirname "$SCRIPT_SOURCE")" && pwd)"
    LOCAL_MODE=true
else
    # Running from pipe - need to download
    LOCAL_MODE=false
fi

# GitHub repo for downloading files
REPO_NAME="${RALPH_REPO_NAME:-dbmrq/ralph}"
RALPH_VERSION="2.1.0"

#==============================================================================
# INLINE MINIMAL UTILITIES (for bootstrap before libraries are available)
#==============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

_print_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}   $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
}

_print_step() { echo -e "${CYAN}▶ $1${NC}"; }
_print_success() { echo -e "${GREEN}✓ $1${NC}"; }
_print_error() { echo -e "${RED}✗ $1${NC}"; }
_print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }

#==============================================================================
# BOOTSTRAP: Download or source libraries
#==============================================================================

bootstrap_libraries() {
    if [ "$LOCAL_MODE" = true ]; then
        # Source libraries from local repo
        source "$RALPH_REPO_DIR/lib/common.sh"
        source "$RALPH_REPO_DIR/lib/prereqs.sh"
        source "$RALPH_REPO_DIR/lib/download.sh"
        source "$RALPH_REPO_DIR/lib/detect.sh"
        source "$RALPH_REPO_DIR/lib/config.sh"
        source "$RALPH_REPO_DIR/lib/prompts.sh"
        source "$RALPH_REPO_DIR/lib/tasks.sh"
        source "$RALPH_REPO_DIR/lib/git.sh"
        source "$RALPH_REPO_DIR/lib/agent.sh"
    else
        # Download and source libraries from GitHub
        _print_step "Downloading Ralph Loop libraries..."

        local temp_dir
        temp_dir=$(mktemp -d)
        trap 'rm -rf "$temp_dir"' EXIT

        local libs="common prereqs download detect config prompts tasks git agent"
        for lib in $libs; do
            curl -fsSL "https://raw.githubusercontent.com/$REPO_NAME/main/lib/${lib}.sh" > "$temp_dir/${lib}.sh"
            source "$temp_dir/${lib}.sh"
        done

        _print_success "Libraries loaded"
        echo ""
    fi
}

#==============================================================================
# FILE VALIDATION HELPERS
#==============================================================================

# Check if a config.sh file is valid (has correct syntax and required vars)
is_valid_config() {
    local config_file="$1"

    # Must exist
    [ -f "$config_file" ] || return 1

    # Must have valid bash syntax
    bash -n "$config_file" 2>/dev/null || return 1

    # Must define PROJECT_NAME (basic sanity check)
    grep -q 'PROJECT_NAME=' "$config_file" 2>/dev/null || return 1

    # Must not have any source/. commands that reference non-existent files
    # This catches old broken configs that try to source common.sh etc.
    local config_dir
    config_dir="$(dirname "$config_file")"

    # Extract all source commands and check if the files exist
    while IFS= read -r line; do
        # Skip comments and empty lines
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$line" ]] && continue

        # Check for source or . commands
        if [[ "$line" =~ (^|[[:space:]])(source|\.)[[:space:]]+[\"\']?([^\"\';]+) ]]; then
            local sourced_file="${BASH_REMATCH[3]}"
            # Remove trailing quotes/whitespace
            sourced_file="${sourced_file%%[\"\']*}"
            sourced_file="${sourced_file%% *}"

            # Skip variable expansions we can't resolve
            [[ "$sourced_file" =~ \$ ]] && continue

            # Resolve relative paths
            if [[ "$sourced_file" != /* ]]; then
                sourced_file="$config_dir/$sourced_file"
            fi

            # If the file doesn't exist, config is invalid
            if [ ! -f "$sourced_file" ]; then
                return 1
            fi
        fi
    done < "$config_file"

    return 0
}

# Check if a script file is customized (not a placeholder)
is_customized_script() {
    local script_file="$1"

    # Must exist
    [ -f "$script_file" ] || return 1

    # If it contains placeholder marker, it's not customized
    grep -q '<!-- PLACEHOLDER:' "$script_file" 2>/dev/null && return 1

    # If it contains the TODO marker from our templates, it's not customized
    grep -q '# TODO: Add your' "$script_file" 2>/dev/null && return 1

    return 0
}

# Check if a prompt file is customized (not a placeholder)
is_customized_prompt() {
    local prompt_file="$1"

    # Must exist
    [ -f "$prompt_file" ] || return 1

    # If it contains placeholder marker, it's not customized
    grep -q '<!-- PLACEHOLDER:' "$prompt_file" 2>/dev/null && return 1

    return 0
}

#==============================================================================
# UPDATE INSTALLATION (Idempotent)
#==============================================================================

update_ralph_installation() {
    local project_path="$1"
    local ralph_dir="$project_path/.ralph"
    local project_name=$(basename "$project_path")

    print_header "Updating Ralph Loop"

    #--------------------------------------------------------------------------
    # Step 1: Update core script (always)
    #--------------------------------------------------------------------------
    print_step "Updating core files..."
    echo ""

    # Always update ralph_loop.sh (the engine)
    if download_file "core/ralph_loop.sh" "$ralph_dir/ralph_loop.sh"; then
        chmod +x "$ralph_dir/ralph_loop.sh" 2>/dev/null
        print_success "Updated ralph_loop.sh"
    else
        print_error "Failed to download ralph_loop.sh"
        return 1
    fi

    # Only update base_prompt.txt if it has the version marker (not customized)
    # or if it doesn't exist
    if [ ! -f "$ralph_dir/base_prompt.txt" ]; then
        if download_file "core/base_prompt.txt" "$ralph_dir/base_prompt.txt"; then
            print_success "Created base_prompt.txt"
        fi
    elif grep -q 'RALPH_BASE_PROMPT_VERSION:' "$ralph_dir/base_prompt.txt" 2>/dev/null; then
        # Has version marker = not customized, safe to update
        if download_file "core/base_prompt.txt" "$ralph_dir/base_prompt.txt"; then
            print_success "Updated base_prompt.txt"
        fi
    else
        print_success "base_prompt.txt is customized - preserved"
    fi

    # Update templates (always)
    download_template_files "$ralph_dir"
    echo ""

    #--------------------------------------------------------------------------
    # Step 2: Fix/create config.sh if needed
    #--------------------------------------------------------------------------
    local config_file="$ralph_dir/config.sh"

    if ! is_valid_config "$config_file"; then
        print_warning "config.sh is missing or invalid - recreating..."

        # Try to preserve existing settings if possible
        local existing_agent="cursor"
        if [ -f "$config_file" ]; then
            existing_agent=$(grep 'AGENT_TYPE=' "$config_file" 2>/dev/null | cut -d'"' -f2 || echo "cursor")
        fi

        create_config_file "$ralph_dir" "$project_name" "${existing_agent:-cursor}" "50" "true"
        print_success "Recreated .ralph/config.sh"
    else
        print_success "config.sh is valid - preserved"
    fi

    #--------------------------------------------------------------------------
    # Step 3: Create missing user files (don't overwrite customized ones)
    #--------------------------------------------------------------------------

    # build.sh
    if [ ! -f "$ralph_dir/build.sh" ]; then
        create_build_script "$ralph_dir"
        print_success "Created missing .ralph/build.sh"
    elif ! is_customized_script "$ralph_dir/build.sh"; then
        # It's a placeholder, update it with latest template
        create_build_script "$ralph_dir"
        print_success "Updated placeholder .ralph/build.sh"
    else
        print_success "build.sh is customized - preserved"
    fi

    # test.sh
    if [ ! -f "$ralph_dir/test.sh" ]; then
        create_test_script "$ralph_dir"
        print_success "Created missing .ralph/test.sh"
    elif ! is_customized_script "$ralph_dir/test.sh"; then
        create_test_script "$ralph_dir"
        print_success "Updated placeholder .ralph/test.sh"
    else
        print_success "test.sh is customized - preserved"
    fi

    # platform_prompt.txt
    if [ ! -f "$ralph_dir/platform_prompt.txt" ]; then
        create_prompt_files "$ralph_dir" "$project_name"
        print_success "Created missing prompt files"
    elif ! is_customized_prompt "$ralph_dir/platform_prompt.txt"; then
        create_prompt_files "$ralph_dir" "$project_name"
        print_success "Updated placeholder prompt files"
    else
        print_success "Prompt files are customized - preserved"
    fi

    # TASKS.md - never overwrite, only create if missing
    if [ ! -f "$ralph_dir/TASKS.md" ]; then
        create_tasks_file "$ralph_dir"
        print_success "Created missing .ralph/TASKS.md"
    else
        print_success "TASKS.md exists - preserved"
    fi

    # docs/README.md - only create if missing
    if [ ! -f "$ralph_dir/docs/README.md" ]; then
        mkdir -p "$ralph_dir/docs"
        create_docs_readme "$ralph_dir"
        print_success "Created missing .ralph/docs/README.md"
    else
        print_success "docs/ exists - preserved"
    fi

    #--------------------------------------------------------------------------
    # Done
    #--------------------------------------------------------------------------
    echo ""
    print_header "Update Complete! 🎉"
    echo ""
    echo "Ralph Loop has been updated to version $RALPH_VERSION"
    echo ""
    echo "Your configuration files have been preserved."
    echo ""
    echo "To run Ralph Loop:"
    echo -e "  ${CYAN}cd $project_path && .ralph/ralph_loop.sh${NC}"
}

#==============================================================================
# FRESH INSTALLATION
#==============================================================================

fresh_install() {
    local project_path="$1"
    local ralph_dir="$project_path/.ralph"

    print_subheader "Installing Ralph Loop"

    mkdir -p "$ralph_dir"

    if ! download_ralph_files "$ralph_dir"; then
        print_error "Failed to download Ralph Loop files."
        echo "Check your network connection and GitHub authentication."
        exit 1
    fi

    # Download template files
    download_template_files "$ralph_dir"
    echo ""

    # Run setup wizard
    run_setup_wizard "$project_path"
}

#==============================================================================
# SETUP WIZARD
#==============================================================================

run_setup_wizard() {
    local project_path="$1"
    local ralph_dir="$project_path/.ralph"
    local step_num=1

    print_header "Ralph Loop Setup"

    #--------------------------------------------------------------------------
    # Step 1: Project Type
    #--------------------------------------------------------------------------
    local project_name=$(basename "$project_path")

    #--------------------------------------------------------------------------
    # Step 1: AI Agent
    #--------------------------------------------------------------------------
    print_step "Step $step_num: AI Agent"
    ((step_num++))
    echo ""

    local agent_type=$(detect_or_select_agent)
    print_success "Agent: $agent_type"
    echo ""

    #--------------------------------------------------------------------------
    # Step 2: Git Branch
    #--------------------------------------------------------------------------
    print_step "Step $step_num: Git Branch"
    ((step_num++))
    echo ""

    local current_branch=""
    local create_branch=false
    local branch_name=""

    current_branch=$(get_current_branch "$project_path")
    echo -e "Current branch: ${BOLD}$current_branch${NC}"

    if is_protected_branch "$project_path"; then
        print_warning "You're on $current_branch. Ralph Loop requires a feature branch."
        if ask_yes_no "Create a new branch?" "y"; then
            create_branch=true
            branch_name=$(ask "Branch name" "feature/ralph-automation")
        else
            print_warning "You'll need to create a branch before running Ralph Loop."
        fi
    else
        print_success "Branch '$current_branch' looks good for Ralph Loop."
    fi
    echo ""

    #--------------------------------------------------------------------------
    # Step 3: Task File
    #--------------------------------------------------------------------------
    print_step "Step $step_num: Task List"
    ((step_num++))
    echo ""

    local create_sample_tasks=false

    echo "Ralph Loop needs a TASKS.md file with your task checklist."
    echo ""

    if [ -f "$ralph_dir/TASKS.md" ]; then
        echo "Found existing task file."
        if ! ask_yes_no "Keep existing tasks?" "y"; then
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
    # Create Files
    #--------------------------------------------------------------------------
    print_header "Creating Configuration Files"

    # Create config.sh
    create_config_file "$ralph_dir" "$project_name" "$agent_type" "50" "true"
    print_success "Created .ralph/config.sh"

    # Create build.sh and test.sh scripts (placeholder templates)
    create_build_script "$ralph_dir"
    print_success "Created .ralph/build.sh"

    create_test_script "$ralph_dir"
    print_success "Created .ralph/test.sh"

    # Create prompt files (placeholder templates)
    create_prompt_files "$ralph_dir" "$project_name"
    print_success "Created .ralph/platform_prompt.txt"
    print_success "Created .ralph/project_prompt.txt"

    # Create docs README
    create_docs_readme "$ralph_dir"
    print_success "Created .ralph/docs/README.md"

    # Create TASKS.md if requested
    if [ "$create_sample_tasks" = true ]; then
        create_tasks_file "$ralph_dir"
        print_success "Created .ralph/TASKS.md"
    fi

    # Create branch if requested
    if [ "$create_branch" = true ] && [ -n "$branch_name" ]; then
        create_branch "$project_path" "$branch_name"
        print_success "Switched to branch: $branch_name"
    fi

    #--------------------------------------------------------------------------
    # Final Instructions
    #--------------------------------------------------------------------------
    print_header "Setup Complete! 🎉"

    echo "Ralph Loop is now configured for your project."
    echo ""
    echo -e "${BOLD}Files created:${NC}"
    echo "  $ralph_dir/ralph_loop.sh       - Main automation script"
    echo "  $ralph_dir/config.sh           - Loop settings and configuration"
    echo "  $ralph_dir/build.sh            - Build verification script (placeholder)"
    echo "  $ralph_dir/test.sh             - Test runner script (placeholder)"
    echo "  $ralph_dir/platform_prompt.txt - Platform guidelines (placeholder)"
    echo "  $ralph_dir/project_prompt.txt  - Project instructions (placeholder)"
    echo "  $ralph_dir/base_prompt.txt     - Core agent workflow"
    echo "  $ralph_dir/docs/               - Additional documentation for agents"
    if [ "$create_sample_tasks" = true ]; then
        echo "  $ralph_dir/TASKS.md            - Task checklist"
    fi
    echo ""

    local step=1
    echo -e "${BOLD}Next steps:${NC}"
    echo ""
    echo -e "  $step. ${YELLOW}Run the AI setup assistant${NC} to configure placeholder files"
    ((step++))
    echo ""
    echo -e "  $step. ${YELLOW}Or manually configure:${NC}"
    echo "     - build.sh and test.sh with your build/test commands"
    echo "     - platform_prompt.txt with platform guidelines"
    echo "     - project_prompt.txt with project-specific instructions"
    ((step++))
    echo ""

    if [ "$create_sample_tasks" = true ]; then
        echo "  $step. Edit TASKS.md with your actual tasks"
        ((step++))
        echo ""
    fi

    echo "  $step. Run Ralph Loop:"
    echo -e "     ${CYAN}cd $project_path${NC}"
    echo -e "     ${CYAN}.ralph/ralph_loop.sh${NC}"
    echo ""

    # Offer AI setup assistant
    offer_ai_setup_assistant "$project_path" "$agent_type"
}


#==============================================================================
# AI SETUP ASSISTANT OFFER
#==============================================================================

offer_ai_setup_assistant() {
    local project_path="$1"
    local agent_type="$2"

    # Check agent availability
    local agent_available=false
    if [ "$agent_type" = "cursor" ] && is_cursor_available; then
        agent_available=true
    elif [ "$agent_type" = "auggie" ] && is_auggie_available; then
        agent_available=true
    fi

    if [ "$agent_available" = false ]; then
        if [ "$agent_type" = "cursor" ]; then
            echo -e "${YELLOW}⚠ Action required:${NC} Install Cursor CLI to use the 'agent' command."
            echo "  Visit: https://cursor.sh"
        elif [ "$agent_type" = "auggie" ]; then
            echo -e "${YELLOW}⚠ Action required:${NC} Install Augment CLI to use the 'auggie' command."
            echo "  Visit: https://augmentcode.com"
        fi
        echo ""
        echo "Happy automating! 🤖"
        return
    fi

    echo -e "${GREEN}✓ Agent '$agent_type' is ready!${NC}"
    echo ""

    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}AI Setup Assistant${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "The AI agent can analyze your project and automatically configure:"
    echo "  • build.sh            - Build verification script"
    echo "  • test.sh             - Test runner script"
    echo "  • platform_prompt.txt - Platform guidelines"
    echo "  • project_prompt.txt  - Project-specific instructions"
    echo ""

    if ask_yes_no "Run AI setup assistant now?" "y"; then
        run_ai_setup_assistant "$project_path" "$agent_type"
    else
        echo ""
        echo "You can run the setup assistant later. Happy automating! 🤖"
    fi
}

#==============================================================================
# MAIN ENTRY POINT
#==============================================================================

main() {
    _print_header "🤖 Ralph Loop Installer"

    echo "Ralph Loop is an automated AI agent task runner."
    echo "Files will be installed into your project's .ralph/ directory."
    echo ""

    # Bootstrap libraries
    bootstrap_libraries

    # Step 1: Check prerequisites
    check_and_install_prerequisites

    # Step 2: Detect project directory
    print_subheader "Project Detection"

    local project_path=$(pwd)

    # Check if current directory is a git repo
    if is_git_repo "$project_path"; then
        echo -e "Current directory: ${BOLD}$project_path${NC}"
        echo -e "Git repository: ${GREEN}✓${NC}"
        echo ""

        if ! ask_yes_no "Set up Ralph Loop for this project?" "y"; then
            project_path=$(ask "Enter project path" "$project_path")
            project_path="${project_path/#\~/$HOME}"

            if [[ ! "$project_path" = /* ]]; then
                project_path="$(cd "$project_path" 2>/dev/null && pwd)"
            fi

            if [ ! -d "$project_path" ]; then
                print_error "Directory does not exist: $project_path"
                exit 1
            fi
        fi
    else
        print_warning "Current directory is not a git repository."
        echo ""

        project_path=$(ask "Enter path to your project" "")
        project_path="${project_path/#\~/$HOME}"

        if [ -z "$project_path" ]; then
            print_error "Project path is required."
            exit 1
        fi

        if [[ ! "$project_path" = /* ]]; then
            project_path="$(cd "$project_path" 2>/dev/null && pwd)"
        fi

        if [ ! -d "$project_path" ]; then
            print_error "Directory does not exist: $project_path"
            exit 1
        fi

        # Check if that path is a git repo
        if ! is_git_repo "$project_path"; then
            print_warning "This directory is not a git repository."
            if ask_yes_no "Initialize git?" "y"; then
                init_git_repo "$project_path"
                print_success "Git repository initialized"
            else
                print_error "Ralph Loop requires a git repository."
                exit 1
            fi
        fi
    fi

    echo ""
    print_success "Project: $project_path"
    echo ""

    # Step 3: Check for existing .ralph
    local ralph_dir="$project_path/.ralph"

    if [ -d "$ralph_dir" ]; then
        print_warning "This project already has Ralph Loop installed."
        echo ""
        echo "Options:"
        echo "  1) Update Ralph Loop (preserves your configuration)"
        echo "  2) Reconfigure from scratch (deletes everything)"
        echo "  3) Exit"
        echo ""

        local choice=$(ask "Choose an option" "1")

        case "$choice" in
            1)
                update_ralph_installation "$project_path"
                exit 0
                ;;
            2)
                rm -rf "$ralph_dir"
                print_success "Removed existing configuration"
                ;;
            3)
                print_success "Goodbye!"
                exit 0
                ;;
        esac
    fi

    # Fresh installation
    fresh_install "$project_path"
}

# Run main
main

