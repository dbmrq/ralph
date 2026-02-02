#!/bin/bash
#
# Ralph Loop - Universal Installer
#
# Installs Ralph Loop directly into your project's .ralph/ directory.
# The AI agent can see these files and understand the automation context.
#
# One-liner install (run from your project directory):
#   bash <(gh api repos/dbmrq/ralph/contents/install.sh --jq '.content' | base64 -d)
#
# What it does:
#   1. Detects if you're in a git repository
#   2. Downloads Ralph Loop files into .ralph/
#   3. Configures the project (type, agent, build commands)
#   4. Offers to run Ralph Loop when ready
#

set -e

# Version
RALPH_VERSION="2.0.0"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

# GitHub repo for downloading files
REPO_NAME="dbmrq/ralph"

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

ask_choice() {
    local prompt="$1"
    shift
    local options=("$@")
    local i=1

    echo "" >&2
    echo "$prompt" >&2
    for opt in "${options[@]}"; do
        echo "  $i) $opt" >&2
        ((i++))
    done
    echo "" >&2

    local choice
    echo -en "${BOLD}Select [1]: ${NC}" >&2
    read choice </dev/tty

    if [ -z "$choice" ]; then
        choice=1
    fi

    # Return the selected option
    echo "${options[$((choice-1))]}"
}

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

    # Download core files
    local files=(
        "ralph_loop.sh"
        "base_prompt.txt"
        "validate.sh"
    )

    for file in "${files[@]}"; do
        if download_file "$file" "$ralph_dir/$file"; then
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

#==============================================================================
# XCODE HELPERS
#==============================================================================

detect_xcode_schemes() {
    local project_dir="$1"
    local schemes=""

    # Method 1: Try XcodeGen project.yml first (fast, no xcodebuild needed)
    local project_yml=$(find "$project_dir" -maxdepth 2 -name "project.yml" -type f 2>/dev/null | head -1)

    if [ -n "$project_yml" ] && [ -f "$project_yml" ]; then
        # Parse scheme names from XcodeGen project.yml
        # Schemes are top-level keys under 'schemes:' (2-space indent, ending with just ':')
        schemes=$(sed -n '/^schemes:/,/^[a-zA-Z]/p' "$project_yml" | grep -E "^  [A-Za-z0-9_-]+:$" | sed 's/:$//' | sed 's/^  //')

        if [ -n "$schemes" ]; then
            echo "$schemes"
            return
        fi
    fi

    # Method 2: Use xcodebuild -list (slower, needs to resolve packages)
    local xcworkspace xcodeproj xcode_output

    xcworkspace=$(find "$project_dir" -maxdepth 2 -name "*.xcworkspace" -type d 2>/dev/null | grep -v ".xcodeproj" | head -1)
    xcodeproj=$(find "$project_dir" -maxdepth 2 -name "*.xcodeproj" -type d 2>/dev/null | head -1)

    if [ -n "$xcworkspace" ]; then
        xcode_output=$(xcodebuild -workspace "$xcworkspace" -list 2>&1 </dev/null)
    elif [ -n "$xcodeproj" ]; then
        xcode_output=$(xcodebuild -project "$xcodeproj" -list 2>&1 </dev/null)
    else
        return
    fi

    # Extract schemes from xcodebuild output
    echo "$xcode_output" | grep -A 100 "Schemes:" | tail -n +2 | grep -v "^$" | sed 's/^[[:space:]]*//' | grep -v "^$"
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
# CONFIG FILE GENERATION
#==============================================================================

create_config_file() {
    local ralph_dir="$1"
    local project_name="$2"
    local project_type="$3"
    local agent_type="$4"
    local xcode_scheme="$5"
    local xcode_project_dir="$6"
    local build_command="$7"
    local test_command="$8"
    local commit_scope="$9"

    local config_file="$ralph_dir/config.sh"

    # Map project type to platform type
    local platform_type="generic"
    case "$project_type" in
        ios) platform_type="ios" ;;
        python) platform_type="python" ;;
        web-react|node) platform_type="generic" ;;
        *) platform_type="generic" ;;
    esac

    cat > "$config_file" << EOF
#!/bin/bash
#
# Ralph Loop - Project Configuration
# Generated by Ralph Loop installer v$RALPH_VERSION
#

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

MAX_ITERATIONS=$DEFAULT_MAX_ITERATIONS
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

COMMIT_SCOPE="$commit_scope"
PROTECTED_BRANCHES="main master develop"

#==============================================================================
# BUILD VERIFICATION
#==============================================================================

BUILD_GATE_ENABLED=true
EOF

    # Add platform-specific settings
    if [ "$project_type" = "ios" ]; then
        cat >> "$config_file" << EOF

#==============================================================================
# iOS SETTINGS
#==============================================================================

XCODE_SCHEME="$xcode_scheme"
XCODE_PROJECT_DIR="$xcode_project_dir"

# Build verification command
verify_build() {
    local project_dir="\$1"
    cd "\$project_dir/\$XCODE_PROJECT_DIR"

    # Generate project if using XcodeGen
    if [ -f "project.yml" ]; then
        xcodegen generate 2>/dev/null || true
    fi

    # Build
    xcodebuild -scheme "\$XCODE_SCHEME" -destination 'platform=iOS Simulator,name=iPhone 16' build 2>&1
}
EOF
    elif [ -n "$build_command" ]; then
        cat >> "$config_file" << EOF

#==============================================================================
# BUILD SETTINGS
#==============================================================================

BUILD_COMMAND="$build_command"
TEST_COMMAND="$test_command"

verify_build() {
    local project_dir="\$1"
    cd "\$project_dir"
    $build_command
}
EOF
    fi
}

create_prompt_file() {
    local ralph_dir="$1"
    local project_type="$2"
    local project_name="$3"

    local prompt_file="$ralph_dir/project_prompt.txt"

    cat > "$prompt_file" << 'EOF'
# Project-Specific Instructions

<!--
  This file provides context about your project to AI agents.
  Fill in each section to help agents understand your codebase.
  The AI setup assistant will help populate this automatically.
-->

## Project Overview
<!-- Brief description of what this project does -->


## Architecture
<!-- Describe the architecture pattern (MVC, MVVM, Clean Architecture, etc.) -->
<!-- List key frameworks and libraries used -->


## Key Directories
<!-- Map out the important directories and their purposes -->
<!-- Example:
- `src/` - Main source code
- `tests/` - Test files
- `docs/` - Documentation
-->


## Coding Standards
<!-- Describe naming conventions, formatting rules, and style guidelines -->
<!-- Reference any linter configs or style guides -->


## Testing Requirements
<!-- How to run tests, what should be tested, coverage expectations -->


## Things to Avoid
<!-- Files that shouldn't be modified, anti-patterns, known pitfalls -->


## Additional Documentation
Check `.ralph/docs/` for additional project documentation.

EOF
}

create_tasks_file() {
    local ralph_dir="$1"
    local project_type="$2"

    local tasks_file="$ralph_dir/TASKS.md"

    cat > "$tasks_file" << 'EOF'
# Ralph Loop Task List

This is your task checklist. Each uncompleted task (marked with `- [ ]`) will be
processed by the AI agent in order.

## How to Write Tasks

- Be specific and actionable
- Each task should be completable in one AI session
- Include relevant file paths or context
- Break large tasks into smaller steps

## Task Format

```
- [ ] Brief description of the task
  > Optional: Additional context or requirements
```

## Your Tasks

- [ ] TASK-001: Example task - Replace this with your first task
  > Add any helpful context or requirements here

- [ ] TASK-002: Second task - Add more tasks as needed

EOF
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
    print_step "Step $step_num: Project Type"
    ((step_num++))
    echo ""

    local project_type=$(detect_project_type "$project_path")
    local project_name=$(basename "$project_path")

    if [ "$project_type" != "generic" ]; then
        echo -e "Detected project type: ${BOLD}$project_type${NC}"
        if ! ask_yes_no "Is this correct?" "y"; then
            project_type=$(ask_choice "Select project type:" "ios" "web-react" "python" "node" "go" "rust" "generic")
        fi
    else
        project_type=$(ask_choice "Select project type:" "ios" "web-react" "python" "node" "go" "rust" "generic")
    fi

    print_success "Project type: $project_type"
    echo ""

    #--------------------------------------------------------------------------
    # Step 2: Build Configuration (for supported types)
    #--------------------------------------------------------------------------
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

        # Detect available schemes
        echo -e "${CYAN}Detecting Xcode schemes...${NC}"
        local schemes_list=$(detect_xcode_schemes "$project_path")

        if [ -n "$schemes_list" ]; then
            local scheme_count=$(echo "$schemes_list" | wc -l | tr -d ' ')

            if [ "$scheme_count" -eq 1 ]; then
                xcode_scheme="$schemes_list"
                echo -e "Xcode scheme: ${BOLD}$xcode_scheme${NC} (auto-detected)"
            else
                echo -e "${GREEN}Found $scheme_count schemes:${NC}"
                echo ""
                local i=1
                while IFS= read -r scheme; do
                    echo "  $i) $scheme"
                    ((i++))
                done <<< "$schemes_list"
                echo ""

                local scheme_choice
                echo -en "${BOLD}Select scheme [1]: ${NC}"
                read scheme_choice </dev/tty

                if [ -z "$scheme_choice" ]; then
                    scheme_choice=1
                fi

                xcode_scheme=$(echo "$schemes_list" | sed -n "${scheme_choice}p")

                if [ -z "$xcode_scheme" ]; then
                    xcode_scheme=$(echo "$schemes_list" | head -1)
                fi
                echo -e "Selected: ${BOLD}$xcode_scheme${NC}"
            fi
        else
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
    fi

    #--------------------------------------------------------------------------
    # Step 3: AI Agent
    #--------------------------------------------------------------------------
    print_step "Step $step_num: AI Agent"
    ((step_num++))
    echo ""

    local agent_type=$(detect_or_select_agent)
    print_success "Agent: $agent_type"
    echo ""

    #--------------------------------------------------------------------------
    # Step 4: Git Branch
    #--------------------------------------------------------------------------
    print_step "Step $step_num: Git Branch"
    ((step_num++))
    echo ""

    local current_branch=""
    local create_branch=false
    local branch_name=""

    cd "$project_path"

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

    cd - > /dev/null
    echo ""

    #--------------------------------------------------------------------------
    # Step 5: Task File
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
    create_config_file "$ralph_dir" "$project_name" "$project_type" "$agent_type" \
        "$xcode_scheme" "$xcode_project_dir" "$build_command" "$test_command" "$commit_scope"
    print_success "Created .ralph/config.sh"

    # Create project_prompt.txt
    create_prompt_file "$ralph_dir" "$project_type" "$project_name"
    print_success "Created .ralph/project_prompt.txt"

    # Create TASKS.md if requested
    if [ "$create_sample_tasks" = true ]; then
        create_tasks_file "$ralph_dir" "$project_type"
        print_success "Created .ralph/TASKS.md"
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
    echo "  $ralph_dir/ralph_loop.sh      - Main automation script"
    echo "  $ralph_dir/config.sh          - Build/test configuration"
    echo "  $ralph_dir/project_prompt.txt - Project-specific AI instructions"
    echo "  $ralph_dir/base_prompt.txt    - Core agent workflow"
    echo "  $ralph_dir/docs/              - Additional documentation for agents"
    if [ "$create_sample_tasks" = true ]; then
        echo "  $ralph_dir/TASKS.md           - Task checklist"
    fi
    echo ""

    local step=1

    echo -e "${BOLD}Next steps:${NC}"
    echo ""
    echo -e "  $step. ${YELLOW}Review project_prompt.txt${NC} - customize for your project:"
    echo -e "     ${CYAN}code $ralph_dir/project_prompt.txt${NC}"
    echo "     This file tells the AI agent about your project's architecture,"
    echo "     coding standards, and things to avoid."
    ((step++))
    echo ""

    if [ "$create_sample_tasks" = true ]; then
        echo "  $step. Edit TASKS.md with your actual tasks:"
        echo -e "     ${CYAN}code $ralph_dir/TASKS.md${NC}"
        ((step++))
        echo ""
    fi

    echo "  $step. (Optional) Add documentation to .ralph/docs/:"
    echo "     Place architecture docs, API references, or coding guides there."
    echo "     Agents will check this directory when they need more context."
    ((step++))
    echo ""

    echo "  $step. Run Ralph Loop:"
    echo -e "     ${CYAN}cd $project_path${NC}"
    echo -e "     ${CYAN}.ralph/ralph_loop.sh${NC}"
    echo ""

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
        elif [ "$agent_type" = "custom" ]; then
            echo -e "${YELLOW}Note:${NC} Define run_agent_custom() in .ralph/config.sh"
        fi
        echo ""
        echo "Once your agent is set up, run the setup assistant manually:"
        echo -e "  ${CYAN}cd $project_path${NC}"
        echo -e "  ${CYAN}$agent_type \"$(cat "$ralph_dir/.setup_prompt.txt" 2>/dev/null || echo "Help me set up Ralph Loop")\"${NC}"
        echo ""
        echo "Happy automating! 🤖"
        return
    fi

    echo -e "${GREEN}✓ Agent '$agent_type' is ready!${NC}"
    echo ""

    # Offer to run AI setup assistant
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}AI Setup Assistant${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "The AI agent can analyze your project and automatically fill in:"
    echo "  • project_prompt.txt - Project-specific instructions"
    echo "  • config.sh - Build and test commands"
    echo ""
    echo "It will also verify the setup is correct and show you how to run Ralph Loop."
    echo ""

    if ask_yes_no "Run AI setup assistant now?" "y"; then
        echo ""
        print_step "Starting AI setup assistant..."
        echo ""

        cd "$project_path"

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

    echo ""
    echo "Happy automating! 🤖"
}

#==============================================================================
# MAIN ENTRY POINT
#==============================================================================

main() {
    print_header "🤖 Ralph Loop Installer"

    echo "Ralph Loop is an automated AI agent task runner."
    echo "Files will be installed into your project's .ralph/ directory."
    echo ""

    # Step 1: Check prerequisites
    check_and_install_prerequisites

    # Step 2: Detect project directory
    print_subheader "Project Detection"

    local project_path=$(pwd)
    local is_git_repo=false

    # Check if current directory is a git repo
    if [ -d "$project_path/.git" ]; then
        is_git_repo=true
        echo -e "Current directory: ${BOLD}$project_path${NC}"
        echo -e "Git repository: ${GREEN}✓${NC}"
        echo ""

        if ! ask_yes_no "Set up Ralph Loop for this project?" "y"; then
            # Let them specify a different path
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
        if [ ! -d "$project_path/.git" ]; then
            print_warning "This directory is not a git repository."
            if ask_yes_no "Initialize git?" "y"; then
                cd "$project_path"
                git init
                git checkout -b main
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
        echo "  1) Update to latest version"
        echo "  2) Reconfigure from scratch"
        echo "  3) Exit"
        echo ""

        local choice=$(ask "Choose an option" "1")

        case "$choice" in
            1)
                print_step "Updating Ralph Loop files..."
                download_ralph_files "$ralph_dir"
                print_success "Updated to version $RALPH_VERSION!"
                echo ""
                echo "To run Ralph Loop:"
                echo -e "  ${CYAN}cd $project_path && .ralph/ralph_loop.sh${NC}"
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

    # Step 4: Download Ralph Loop files
    print_subheader "Installing Ralph Loop"

    mkdir -p "$ralph_dir"

    if ! download_ralph_files "$ralph_dir"; then
        print_error "Failed to download Ralph Loop files."
        echo "Check your network connection and GitHub authentication."
        exit 1
    fi

    # Download templates for the detected project type
    local project_type=$(detect_project_type "$project_path")
    local platform_type="generic"
    case "$project_type" in
        ios) platform_type="ios" ;;
        python) platform_type="python" ;;
        *) platform_type="generic" ;;
    esac

    download_template_files "$ralph_dir" "$platform_type"
    echo ""

    # Step 5: Run setup wizard
    run_setup_wizard "$project_path"
}

# Run main
main