#!/bin/bash
#
# Ralph Loop - Universal Entry Point
#
# This is THE single entry point for Ralph Loop. Run this script to:
#   - Install prerequisites (Homebrew, GitHub CLI)
#   - Clone or update ralph-loop
#   - Configure a project
#   - Add tasks and custom instructions
#
# One-liner install (requires gh CLI already installed):
#   bash <(gh api repos/W508153_wexinc/ralph-loop/contents/install.sh --jq '.content' | base64 -d)
#
# Or if you have ralph-loop cloned, just run:
#   ./install.sh
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

REPO_URL="https://github.com/W508153_wexinc/ralph-loop.git"
REPO_NAME="W508153_wexinc/ralph-loop"

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

print_subheader() {
    echo ""
    echo -e "${CYAN}─────────────────────────────────────────────────────────────────${NC}"
    echo -e "${CYAN}   $1${NC}"
    echo -e "${CYAN}─────────────────────────────────────────────────────────────────${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_step() {
    echo -e "${CYAN}▶ $1${NC}"
}

print_info() {
    echo -e "${MAGENTA}ℹ $1${NC}"
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

#==============================================================================
# PREREQUISITE CHECKS AND INSTALLATION
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


#==============================================================================
# DETECT CONTEXT
#==============================================================================

detect_ralph_loop_location() {
    # Check if we're running from within a ralph-loop repo
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd)"

    # If BASH_SOURCE is empty (piped from curl/gh), we're running remotely
    if [ -z "$script_dir" ] || [ "$script_dir" = "." ]; then
        echo ""
        return
    fi

    # Check if this directory is the ralph-loop repo
    if [ -f "$script_dir/ralph_loop.sh" ] && [ -f "$script_dir/setup.sh" ]; then
        echo "$script_dir"
        return
    fi

    echo ""
}

#==============================================================================
# INSTALLATION / UPDATE
#==============================================================================

install_or_update_ralph_loop() {
    local ralph_dir="$1"

    if [ -n "$ralph_dir" ]; then
        # Already have ralph-loop, offer to update
        print_success "Ralph Loop is installed at: $ralph_dir" >&2
        echo "" >&2
        if ask_yes_no "Check for updates?" "y"; then
            print_step "Updating ralph-loop..." >&2
            cd "$ralph_dir"
            git pull origin main >/dev/null 2>&1 || git pull origin master >/dev/null 2>&1 || true
            print_success "Updated to latest version!" >&2
        fi
        echo "$ralph_dir"
        return
    fi

    # Need to clone ralph-loop
    print_subheader "Installing Ralph Loop" >&2

    echo "Where should ralph-loop be installed?" >&2
    echo "" >&2
    echo "Enter a parent directory - ralph-loop will be created inside it." >&2
    echo "For example, if you enter ~/Code, it will install to ~/Code/ralph-loop" >&2
    echo "" >&2

    # Suggest a default based on current directory
    local current_dir=$(pwd)
    local parent_dir=$(dirname "$current_dir")
    local default_install="$parent_dir"

    local install_parent=$(ask "Parent directory" "$default_install")

    # Expand ~ if present
    install_parent="${install_parent/#\~/$HOME}"

    # Convert to absolute path
    if [[ ! "$install_parent" = /* ]]; then
        if [ -d "$install_parent" ]; then
            install_parent="$(cd "$install_parent" && pwd)"
        else
            mkdir -p "$install_parent"
            install_parent="$(cd "$install_parent" && pwd)"
        fi
    fi

    # The actual install path is parent/ralph-loop
    local install_path="$install_parent/ralph-loop"

    # Check if ralph-loop directory already exists
    if [ -d "$install_path" ]; then
        if [ -f "$install_path/ralph_loop.sh" ]; then
            print_success "ralph-loop already installed at $install_path" >&2
            cd "$install_path"
            if ask_yes_no "Update to latest version?" "y"; then
                git pull origin main >/dev/null 2>&1 || git pull origin master >/dev/null 2>&1 || true
            fi
            echo "$install_path"
            return
        else
            print_error "Directory $install_path exists but is not a ralph-loop installation." >&2
            echo "Please remove it or choose a different parent directory." >&2
            exit 1
        fi
    fi

    # Clone the repository
    print_step "Cloning ralph-loop to $install_path..." >&2
    echo "" >&2

    mkdir -p "$install_parent"
    git clone "$REPO_URL" "$install_path" >&2

    # Make scripts executable
    chmod +x "$install_path/ralph_loop.sh"
    chmod +x "$install_path/setup.sh"
    chmod +x "$install_path/install.sh"

    print_success "Installed to $install_path" >&2
    echo "$install_path"
}

#==============================================================================
# PROJECT SETUP (integrated from setup.sh)
#==============================================================================

detect_project_type() {
    local project_dir="$1"

    # iOS/macOS detection: Xcode projects, Swift packages, or XcodeGen
    if [ -f "$project_dir/Package.swift" ] || \
       [ -f "$project_dir/project.yml" ] || \
       [ -f "$project_dir/project.yaml" ] || \
       ls "$project_dir"/*.xcodeproj &>/dev/null 2>&1 || \
       ls "$project_dir"/*.xcworkspace &>/dev/null 2>&1 || \
       find "$project_dir" -maxdepth 2 -name "*.xcodeproj" 2>/dev/null | grep -q . || \
       find "$project_dir" -maxdepth 2 -name "*.xcworkspace" 2>/dev/null | grep -q . || \
       find "$project_dir" -maxdepth 2 -name "project.yml" 2>/dev/null | grep -q .; then
        echo "ios"
    elif [ -f "$project_dir/package.json" ]; then
        if grep -q '"react"' "$project_dir/package.json" 2>/dev/null; then
            echo "react"
        elif grep -q '"next"' "$project_dir/package.json" 2>/dev/null; then
            echo "nextjs"
        else
            echo "node"
        fi
    elif [ -f "$project_dir/requirements.txt" ] || [ -f "$project_dir/setup.py" ] || [ -f "$project_dir/pyproject.toml" ]; then
        echo "python"
    elif [ -f "$project_dir/Cargo.toml" ]; then
        echo "rust"
    elif [ -f "$project_dir/go.mod" ]; then
        echo "go"
    else
        echo "generic"
    fi
}

setup_project() {
    local ralph_dir="$1"

    print_subheader "Project Setup"

    echo "Which project do you want to set up with Ralph Loop?"
    echo ""
    echo "Enter the path to your project directory:"
    echo ""

    local default_project=$(pwd)
    # Don't suggest ralph-loop itself as the project
    if [ "$default_project" = "$ralph_dir" ]; then
        default_project=$(dirname "$ralph_dir")
    fi

    local project_path=$(ask "Project path" "$default_project")

    # Expand ~ if present
    project_path="${project_path/#\~/$HOME}"

    # Convert to absolute path
    if [[ ! "$project_path" = /* ]]; then
        project_path="$(cd "$project_path" 2>/dev/null && pwd)"
    fi

    if [ ! -d "$project_path" ]; then
        print_error "Directory does not exist: $project_path"
        exit 1
    fi

    # Check if it's a git repository
    if [ ! -d "$project_path/.git" ]; then
        print_warning "This directory is not a git repository."
        if ask_yes_no "Initialize git?" "y"; then
            cd "$project_path"
            git init
            print_success "Git repository initialized"
        else
            print_error "Ralph Loop requires a git repository."
            exit 1
        fi
    fi

    # Detect project type
    local project_type=$(detect_project_type "$project_path")
    print_info "Detected project type: $project_type"
    echo ""

    # Check if .ralph already exists
    if [ -d "$project_path/.ralph" ]; then
        print_warning "This project already has a .ralph configuration."
        if ask_yes_no "Reconfigure?" "n"; then
            rm -rf "$project_path/.ralph"
        else
            echo "$project_path"
            return
        fi
    fi

    # Run the full setup wizard
    export RALPH_PROJECT_PATH="$project_path"
    export RALPH_PROJECT_TYPE="$project_type"
    exec "$ralph_dir/setup.sh"
}



#==============================================================================
# MAIN MENU
#==============================================================================

show_menu() {
    local ralph_dir="$1"

    print_subheader "What would you like to do?"

    echo "  1) Set up a new project"
    echo "  2) Add/edit tasks for an existing project"
    echo "  3) Edit instructions (3-level system)"
    echo "  4) Run Ralph Loop on a project"
    echo "  5) Update ralph-loop to latest version"
    echo "  6) Exit"
    echo ""

    local choice=$(ask "Choose an option" "1")

    case "$choice" in
        1)
            setup_project "$ralph_dir"
            ;;
        2)
            edit_tasks "$ralph_dir"
            ;;
        3)
            edit_instructions "$ralph_dir"
            ;;
        4)
            run_ralph_loop "$ralph_dir"
            ;;
        5)
            print_step "Updating ralph-loop..."
            cd "$ralph_dir"
            git pull origin main 2>/dev/null || git pull origin master 2>/dev/null || true
            print_success "Updated!"
            show_menu "$ralph_dir"
            ;;
        6)
            echo ""
            print_success "Goodbye!"
            exit 0
            ;;
        *)
            print_error "Invalid choice"
            show_menu "$ralph_dir"
            ;;
    esac
}

#==============================================================================
# TASK EDITING
#==============================================================================

select_project() {
    local ralph_dir="$1"
    local prompt="${2:-Select a project}"

    # Output prompts to stderr since this function returns a value via stdout
    echo "$prompt" >&2
    echo "" >&2

    local default_project=$(pwd)
    if [ "$default_project" = "$ralph_dir" ]; then
        default_project=$(dirname "$ralph_dir")
    fi

    local project_path=$(ask "Project path" "$default_project")
    project_path="${project_path/#\~/$HOME}"

    if [[ ! "$project_path" = /* ]]; then
        project_path="$(cd "$project_path" 2>/dev/null && pwd)"
    fi

    if [ ! -d "$project_path/.ralph" ]; then
        print_error "This project doesn't have Ralph Loop set up." >&2
        if ask_yes_no "Set it up now?" "y"; then
            setup_project "$ralph_dir"
            return
        fi
        exit 1
    fi

    echo "$project_path"
}

edit_tasks() {
    local ralph_dir="$1"

    print_subheader "Edit Tasks"

    local project_path=$(select_project "$ralph_dir" "Which project's tasks do you want to edit?")
    local tasks_file="$project_path/.ralph/TASKS.md"

    echo ""
    echo "You can:"
    echo "  1) Open tasks file in your editor"
    echo "  2) Add tasks interactively here"
    echo "  3) View current tasks"
    echo ""

    local choice=$(ask "Choose an option" "2")

    case "$choice" in
        1)
            local editor="${EDITOR:-${VISUAL:-nano}}"
            print_step "Opening $tasks_file in $editor..."
            "$editor" "$tasks_file"
            ;;
        2)
            add_tasks_interactively "$tasks_file"
            ;;
        3)
            echo ""
            print_step "Current tasks:"
            echo ""
            cat "$tasks_file"
            echo ""
            if ask_yes_no "Edit tasks?" "y"; then
                edit_tasks "$ralph_dir"
            fi
            ;;
    esac

    print_success "Tasks updated!"
    show_menu "$ralph_dir"
}

add_tasks_interactively() {
    local tasks_file="$1"

    echo ""
    echo "Enter your tasks one by one."
    echo "Format: Brief description of what the agent should do"
    echo "Type 'done' when finished."
    echo ""

    local task_num=1

    # Find the highest existing task number
    if [ -f "$tasks_file" ]; then
        local max_num=$(grep -o 'TASK-[0-9]*' "$tasks_file" 2>/dev/null | grep -o '[0-9]*' | sort -n | tail -1)
        if [ -n "$max_num" ]; then
            task_num=$((max_num + 1))
        fi
    fi

    while true; do
        echo -en "${BOLD}Task $task_num${NC}: "
        read task_desc </dev/tty

        if [ "$task_desc" = "done" ] || [ -z "$task_desc" ]; then
            break
        fi

        # Ask for optional details
        echo -en "  ${CYAN}Details (optional, press Enter to skip)${NC}: "
        read task_details </dev/tty

        # Append to tasks file
        echo "" >> "$tasks_file"
        printf "- [ ] TASK-%03d: %s\n" "$task_num" "$task_desc" >> "$tasks_file"

        if [ -n "$task_details" ]; then
            echo "  > $task_details" >> "$tasks_file"
        fi

        print_success "Added TASK-$(printf '%03d' $task_num)"
        task_num=$((task_num + 1))
    done
}


#==============================================================================
# INSTRUCTIONS (3-Level System)
#==============================================================================

edit_instructions() {
    local ralph_dir="$1"

    print_subheader "Edit Instructions (3-Level System)"

    echo ""
    echo "Ralph Loop uses a 3-level instruction system:"
    echo ""
    echo -e "  ${BOLD}Level 1: Global${NC} - Ralph Loop workflow (base_prompt.txt)"
    echo "           Applies to all projects. Rarely needs editing."
    echo ""
    echo -e "  ${BOLD}Level 2: Platform${NC} - Platform guidelines (templates/{platform}/platform_prompt.txt)"
    echo "           iOS, Python, generic, etc. Edit to customize platform standards."
    echo ""
    echo -e "  ${BOLD}Level 3: Project${NC} - Project-specific (.ralph/project_prompt.txt)"
    echo "           Your project's unique requirements. Most commonly edited."
    echo ""
    echo "Which level do you want to edit?"
    echo "  1) Project instructions (Level 3) - most common"
    echo "  2) Platform instructions (Level 2)"
    echo "  3) Global instructions (Level 1)"
    echo "  4) View all levels combined"
    echo ""

    local level_choice=$(ask "Choose a level" "1")

    case "$level_choice" in
        1)
            edit_project_instructions "$ralph_dir"
            ;;
        2)
            edit_platform_instructions "$ralph_dir"
            ;;
        3)
            edit_global_instructions "$ralph_dir"
            ;;
        4)
            view_combined_instructions "$ralph_dir"
            ;;
        *)
            edit_project_instructions "$ralph_dir"
            ;;
    esac

    show_menu "$ralph_dir"
}

edit_project_instructions() {
    local ralph_dir="$1"

    print_subheader "Project Instructions (Level 3)"

    local project_path=$(select_project "$ralph_dir" "Which project's instructions do you want to edit?")
    local prompt_file="$project_path/.ralph/project_prompt.txt"

    echo ""
    echo "Project instructions describe YOUR project's unique requirements:"
    echo "  - Project structure and key files"
    echo "  - Coding conventions specific to this project"
    echo "  - Things to avoid"
    echo "  - Reference materials"
    echo ""
    echo "You can:"
    echo "  1) Open in your editor"
    echo "  2) Add instructions interactively"
    echo "  3) View current instructions"
    echo ""

    local choice=$(ask "Choose an option" "1")

    case "$choice" in
        1)
            local editor="${EDITOR:-${VISUAL:-nano}}"
            print_step "Opening $prompt_file in $editor..."
            "$editor" "$prompt_file"
            print_success "Project instructions updated!"
            ;;
        2)
            add_instructions_interactively "$prompt_file"
            ;;
        3)
            echo ""
            if [ -f "$prompt_file" ]; then
                print_step "Current project instructions:"
                echo ""
                cat "$prompt_file"
            else
                print_warning "No project instructions file found at $prompt_file"
            fi
            echo ""
            if ask_yes_no "Edit instructions?" "y"; then
                edit_project_instructions "$ralph_dir"
            fi
            ;;
    esac
}

edit_platform_instructions() {
    local ralph_dir="$1"

    print_subheader "Platform Instructions (Level 2)"

    echo ""
    echo "Available platforms:"
    local platforms=()
    for dir in "$ralph_dir/templates"/*/; do
        if [ -d "$dir" ]; then
            local platform=$(basename "$dir")
            platforms+=("$platform")
            echo "  - $platform"
        fi
    done
    echo ""

    local platform=$(ask "Which platform?" "ios")
    local prompt_file="$ralph_dir/templates/$platform/platform_prompt.txt"

    if [ ! -f "$prompt_file" ]; then
        print_warning "Platform '$platform' not found."
        if ask_yes_no "Create it?" "y"; then
            mkdir -p "$ralph_dir/templates/$platform"
            cp "$ralph_dir/templates/generic/platform_prompt.txt" "$prompt_file"
            print_success "Created $prompt_file from generic template"
        else
            return
        fi
    fi

    local editor="${EDITOR:-${VISUAL:-nano}}"
    print_step "Opening $prompt_file in $editor..."
    "$editor" "$prompt_file"
    print_success "Platform instructions updated!"
}

edit_global_instructions() {
    local ralph_dir="$1"

    print_subheader "Global Instructions (Level 1)"

    local prompt_file="$ralph_dir/base_prompt.txt"

    echo ""
    print_warning "Global instructions affect ALL projects using Ralph Loop."
    echo "These define the core workflow: task format, status markers, rules."
    echo ""

    if ask_yes_no "Are you sure you want to edit global instructions?" "n"; then
        local editor="${EDITOR:-${VISUAL:-nano}}"
        print_step "Opening $prompt_file in $editor..."
        "$editor" "$prompt_file"
        print_success "Global instructions updated!"
    fi
}

view_combined_instructions() {
    local ralph_dir="$1"

    print_subheader "View Combined Instructions"

    local project_path=$(select_project "$ralph_dir" "Which project?")

    # Get platform type from config
    local config_file="$project_path/.ralph/config.sh"
    local platform_type="generic"
    if [ -f "$config_file" ]; then
        source "$config_file"
        platform_type="${PLATFORM_TYPE:-generic}"
    fi

    echo ""
    echo "Showing combined instructions for: $project_path"
    echo "Platform type: $platform_type"
    echo ""
    echo -e "${CYAN}════════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  Level 1: Global Instructions${NC}"
    echo -e "${CYAN}════════════════════════════════════════════════════════════════${NC}"
    if [ -f "$ralph_dir/base_prompt.txt" ]; then
        cat "$ralph_dir/base_prompt.txt"
    else
        echo "(not found)"
    fi
    echo ""
    echo -e "${CYAN}════════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  Level 2: Platform Instructions ($platform_type)${NC}"
    echo -e "${CYAN}════════════════════════════════════════════════════════════════${NC}"
    if [ -f "$ralph_dir/templates/$platform_type/platform_prompt.txt" ]; then
        cat "$ralph_dir/templates/$platform_type/platform_prompt.txt"
    else
        echo "(not found)"
    fi
    echo ""
    echo -e "${CYAN}════════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  Level 3: Project Instructions${NC}"
    echo -e "${CYAN}════════════════════════════════════════════════════════════════${NC}"
    if [ -f "$project_path/.ralph/project_prompt.txt" ]; then
        cat "$project_path/.ralph/project_prompt.txt"
    else
        echo "(not found)"
    fi
    echo ""

    echo ""
    read -p "Press Enter to continue..." </dev/tty
}

add_instructions_interactively() {
    local prompt_file="$1"

    echo ""
    echo "Enter your custom instructions for the AI agent."
    echo ""
    echo "These could include:"
    echo "  - Project structure and key files"
    echo "  - Coding conventions specific to this project"
    echo "  - Things to avoid"
    echo "  - Reference materials"
    echo ""
    echo "Type your instructions below. When finished, type 'END' on a new line."
    echo ""
    print_step "Custom Instructions:"
    echo ""

    local instructions=""
    while IFS= read -r line </dev/tty; do
        if [ "$line" = "END" ]; then
            break
        fi
        instructions+="$line"$'\n'
    done

    if [ -n "$instructions" ]; then
        # Append to existing file with a separator
        if [ -f "$prompt_file" ]; then
            echo "" >> "$prompt_file"
        fi
        echo "# Custom Instructions (added $(date +%Y-%m-%d))" >> "$prompt_file"
        echo "" >> "$prompt_file"
        echo "$instructions" >> "$prompt_file"
        print_success "Instructions added to $prompt_file"
    else
        print_warning "No instructions entered."
    fi
}

#==============================================================================
# RUN RALPH LOOP
#==============================================================================

run_ralph_loop() {
    local ralph_dir="$1"

    print_subheader "Run Ralph Loop"

    local project_path=$(select_project "$ralph_dir" "Which project do you want to run Ralph Loop on?")

    echo ""
    print_step "Ready to run Ralph Loop!"
    echo ""
    echo "This will start the automated task runner on your project."
    echo "The agent will work through tasks in .ralph/TASKS.md"
    echo ""

    # Check current branch
    cd "$project_path"
    local current_branch=$(git branch --show-current)

    if [ "$current_branch" = "main" ] || [ "$current_branch" = "master" ]; then
        print_warning "You're on the $current_branch branch."
        echo "Ralph Loop won't run on main/master for safety."
        echo ""
        if ask_yes_no "Create a new branch?" "y"; then
            local new_branch=$(ask "Branch name" "feature/ralph-automation")
            git checkout -b "$new_branch"
            print_success "Created and switched to branch: $new_branch"
        else
            print_error "Please switch to a feature branch first."
            show_menu "$ralph_dir"
            return
        fi
    fi

    echo ""
    if ask_yes_no "Start Ralph Loop now?" "y"; then
        echo ""
        print_step "Starting Ralph Loop..."
        echo ""
        exec "$ralph_dir/ralph_loop.sh" "$project_path"
    else
        echo ""
        echo "To run Ralph Loop later, use:"
        echo ""
        echo -e "  ${BOLD}$ralph_dir/ralph_loop.sh $project_path${NC}"
        echo ""
        show_menu "$ralph_dir"
    fi
}

#==============================================================================
# MAIN ENTRY POINT
#==============================================================================

main() {
    print_header "🤖 Ralph Loop"

    echo "Ralph Loop is an automated AI agent task runner that helps"
    echo "you automate repetitive development tasks."
    echo ""

    # Step 1: Check and install prerequisites
    check_and_install_prerequisites

    # Step 2: Detect if we're already in ralph-loop repo
    local ralph_dir=$(detect_ralph_loop_location)

    # Step 3: Install or update ralph-loop
    ralph_dir=$(install_or_update_ralph_loop "$ralph_dir")

    # Step 4: Show menu
    show_menu "$ralph_dir"
}

# Run main
main