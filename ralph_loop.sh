#!/bin/bash
#
# Ralph Loop - Automated AI Agent Task Runner
#
# This script repeatedly calls an AI agent to complete tasks from a task file
# until all tasks are done, max iterations reached, or build failures occur.
#
# Usage: ./ralph_loop.sh <project_dir> [agent]
#   project_dir: Path to project containing .ralph/ config directory
#   agent: Agent name (default: from config or 'cursor')
#
# Examples:
#   ./ralph_loop.sh ../my-project
#   ./ralph_loop.sh ../my-project cursor
#   ./ralph_loop.sh ../my-project auggie
#
# Project Setup:
#   Your project needs a .ralph/ directory with:
#   - config.sh           (required) - Project configuration
#   - project_prompt.txt  (optional) - Project-specific instructions
#   - TASKS.md            (required) - Task checklist
#

set -e

# Script directory (where ralph_loop.sh lives)
RALPH_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default configuration (can be overridden by project config)
MAX_ITERATIONS=50
PAUSE_SECONDS=5
MAX_CONSECUTIVE_FAILURES=3
DEFAULT_AGENT="cursor"
REQUIRE_BRANCH=true
ALLOWED_BRANCHES=""  # Empty means any non-main branch
AUTO_COMMIT=true
COMMIT_PREFIX="feat"
COMMIT_SCOPE=""

# Build verification settings
BUILD_GATE_ENABLED=true
BUILD_FIX_ATTEMPTS=1

# Test run mode settings
# When enabled, runs first N tasks then pauses for user verification
TEST_RUN_ENABLED=true
TEST_RUN_TASKS=2

#==============================================================================
# ARGUMENT PARSING
#==============================================================================

if [ $# -lt 1 ]; then
    echo -e "${RED}Usage: $0 <project_dir> [agent]${NC}"
    echo ""
    echo "  project_dir: Path to project containing .ralph/ config directory"
    echo "  agent:       Agent to use (default: cursor)"
    echo ""
    echo "Example: $0 ../my-ios-app auggie"
    exit 1
fi

PROJECT_DIR="$(cd "$1" && pwd)"
AGENT_OVERRIDE="${2:-}"

#==============================================================================
# LOAD PROJECT CONFIGURATION
#==============================================================================

RALPH_CONFIG_DIR="$PROJECT_DIR/.ralph"
CONFIG_FILE="$RALPH_CONFIG_DIR/config.sh"
TASK_FILE="$RALPH_CONFIG_DIR/TASKS.md"

if [ ! -d "$RALPH_CONFIG_DIR" ]; then
    echo -e "${RED}ERROR: .ralph/ directory not found in $PROJECT_DIR${NC}"
    echo "Create it with: mkdir -p $PROJECT_DIR/.ralph"
    echo "Then add config.sh and TASKS.md"
    exit 1
fi

if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}ERROR: Config file not found: $CONFIG_FILE${NC}"
    exit 1
fi

# Source the project config (this can override defaults above)
source "$CONFIG_FILE"

# Apply agent override if provided
if [ -n "$AGENT_OVERRIDE" ]; then
    AGENT_TYPE="$AGENT_OVERRIDE"
else
    AGENT_TYPE="${AGENT_TYPE:-$DEFAULT_AGENT}"
fi

#==============================================================================
# VALIDATE CONFIGURATION
#==============================================================================

# Task file is required
if [ ! -f "$TASK_FILE" ]; then
    echo -e "${RED}ERROR: Task file not found: $TASK_FILE${NC}"
    exit 1
fi

# Prompt files are optional but we should have at least one
# (base_prompt.txt is always available in ralph-loop repo)

#==============================================================================
# SETUP LOGGING
#==============================================================================

LOG_DIR="$RALPH_CONFIG_DIR/logs"
mkdir -p "$LOG_DIR"

RUN_ID=$(date +"%Y%m%d_%H%M%S")
MASTER_LOG="$LOG_DIR/ralph_run_${RUN_ID}.log"
touch "$MASTER_LOG"

log() {
    echo -e "$1" | tee -a "$MASTER_LOG"
}

log_only() {
    echo -e "$1" >> "$MASTER_LOG"
}

#==============================================================================
# BRANCH VERIFICATION
#==============================================================================

verify_branch() {
    if [ "$REQUIRE_BRANCH" != "true" ]; then
        return 0
    fi
    
    cd "$PROJECT_DIR"
    local current_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
    
    if [ -z "$current_branch" ]; then
        log "${RED}ERROR: Not a git repository: $PROJECT_DIR${NC}"
        exit 1
    fi
    
    # Check if on main/master (not allowed)
    if [ "$current_branch" = "main" ] || [ "$current_branch" = "master" ]; then
        log "${RED}ERROR: Cannot run on '$current_branch' branch${NC}"
        log "${YELLOW}Create a feature branch first: git checkout -b feature/my-feature${NC}"
        exit 1
    fi
    
    # Check allowed branches if specified
    if [ -n "$ALLOWED_BRANCHES" ]; then
        if ! echo "$ALLOWED_BRANCHES" | grep -qw "$current_branch"; then
            log "${RED}ERROR: Branch '$current_branch' not in allowed list${NC}"
            log "${YELLOW}Allowed branches: $ALLOWED_BRANCHES${NC}"
            exit 1
        fi
    fi
    
    log "${GREEN}✓ On branch: $current_branch${NC}"
    cd - > /dev/null
}

#==============================================================================
# BUILD PROMPT (3-Level System)
#==============================================================================
#
# Prompts are combined from 3 levels:
#   1. Global (base_prompt.txt) - Ralph Loop workflow instructions
#   2. Platform (templates/{platform}/platform_prompt.txt) - Platform guidelines
#   3. Project (.ralph/project_prompt.txt) - Project-specific instructions
#
# Each level can be edited independently without affecting the others.
#

build_prompt() {
    local base_prompt_file="$RALPH_DIR/base_prompt.txt"
    local platform_prompt_file="$RALPH_DIR/templates/${PLATFORM_TYPE:-generic}/platform_prompt.txt"
    local project_prompt_file="$RALPH_CONFIG_DIR/project_prompt.txt"

    # Level 1: Global/Ralph Loop instructions
    if [ -f "$base_prompt_file" ]; then
        echo "# Level 1: Ralph Loop Instructions"
        echo ""
        cat "$base_prompt_file"
        echo ""
        echo "---"
        echo ""
    fi

    # Level 2: Platform-specific guidelines
    if [ -f "$platform_prompt_file" ]; then
        echo "# Level 2: Platform Guidelines (${PLATFORM_TYPE:-generic})"
        echo ""
        cat "$platform_prompt_file"
        echo ""
        echo "---"
        echo ""
    else
        log "${YELLOW}Note: No platform prompt found for '${PLATFORM_TYPE:-generic}'${NC}"
    fi

    # Level 3: Project-specific instructions
    if [ -f "$project_prompt_file" ]; then
        echo "# Level 3: Project-Specific Instructions"
        echo ""
        cat "$project_prompt_file"
    else
        log "${YELLOW}Note: No project prompt found at $project_prompt_file${NC}"
    fi
}

#==============================================================================
# AGENT COMMANDS
#==============================================================================

# Default agent commands - can be overridden in config.sh
run_agent_cursor() {
    local prompt="$1"
    local log_file="$2"

    if ! command -v agent &> /dev/null; then
        log "${RED}ERROR: 'agent' command not found. Please install Cursor CLI.${NC}"
        return 1
    fi

    echo "$prompt" | agent --print --model "claude-4.5-opus" > "$log_file" 2>&1
    cat "$log_file"
}

run_agent_auggie() {
    local prompt="$1"
    local log_file="$2"

    if ! command -v auggie &> /dev/null; then
        log "${RED}ERROR: 'auggie' command not found. Please install Augment CLI.${NC}"
        return 1
    fi

    auggie --print --quiet "$prompt" > "$log_file" 2>&1
    cat "$log_file"
}

run_agent() {
    local log_file="$1"
    local prompt_override="$2"  # Optional: for build fix prompts

    local prompt
    if [ -n "$prompt_override" ]; then
        prompt="$prompt_override"
    else
        prompt=$(build_prompt)
    fi

    cd "$PROJECT_DIR"

    set +e  # Temporarily disable exit on error

    case "$AGENT_TYPE" in
        cursor)
            run_agent_cursor "$prompt" "$log_file"
            ;;
        auggie)
            run_agent_auggie "$prompt" "$log_file"
            ;;
        custom)
            # Custom agent command should be defined in config.sh as run_agent_custom()
            if type run_agent_custom &> /dev/null; then
                run_agent_custom "$prompt" "$log_file"
            else
                log "${RED}ERROR: Custom agent selected but run_agent_custom() not defined in config.sh${NC}"
                set -e
                return 1
            fi
            ;;
        *)
            log "${RED}ERROR: Unknown agent type '$AGENT_TYPE'. Use 'cursor', 'auggie', or 'custom'.${NC}"
            set -e
            return 1
            ;;
    esac

    local agent_exit=$?
    set -e

    cd - > /dev/null
    return 0  # We check log content, not exit code
}

#==============================================================================
# TASK COUNTING
#==============================================================================

count_remaining() {
    grep -c "^\- \[ \]" "$TASK_FILE" 2>/dev/null || echo "0"
}

count_completed() {
    grep -c "^\- \[x\]" "$TASK_FILE" 2>/dev/null || echo "0"
}

get_next_task() {
    grep "^\- \[ \]" "$TASK_FILE" | head -1 | sed -E 's/- \[ \] //'
}

get_last_completed_task_id() {
    grep "^\- \[x\]" "$TASK_FILE" | tail -1 | sed -E 's/.*\[x\] ([A-Za-z0-9_-]+):.*/\1/'
}

get_last_completed_task_description() {
    grep "^\- \[x\]" "$TASK_FILE" | tail -1 | sed -E 's/.*\[x\] [A-Za-z0-9_-]+: (.*)/\1/'
}

#==============================================================================
# BUILD VERIFICATION
#==============================================================================

# Default build command - should be overridden in config.sh
run_build() {
    if type project_build &> /dev/null; then
        project_build
    else
        log "${YELLOW}⚠ No build command defined (define project_build() in config.sh)${NC}"
        return 0
    fi
}

# Default test command - should be overridden in config.sh
run_tests() {
    if type project_test &> /dev/null; then
        project_test
    else
        log "${YELLOW}⚠ No test command defined (define project_test() in config.sh)${NC}"
        return 0
    fi
}

verify_build() {
    if [ "$BUILD_GATE_ENABLED" != "true" ]; then
        return 0
    fi

    log "${CYAN}🔨 Verifying build...${NC}"

    cd "$PROJECT_DIR"
    set +e
    run_build
    local build_result=$?
    set -e
    cd - > /dev/null

    if [ $build_result -ne 0 ]; then
        log "${RED}❌ Build failed${NC}"
        return 1
    fi

    log "${GREEN}✓ Build succeeded${NC}"
    return 0
}

#==============================================================================
# BUILD FIX MODE
#==============================================================================

BUILD_FIX_PROMPT="CRITICAL: The build is broken and must be fixed before any other work.

Your ONLY task right now is to fix the build. Do not work on any tasks from the task list.

Steps:
1. Run the build command to see the errors
2. Analyze the error messages
3. Fix the issues causing the build to fail
4. Verify the build succeeds
5. If tests are required, run them too

When the build is fixed and passing, output: FIXED
If you cannot fix the build, output: ERROR: <description of the problem>

Do NOT output NEXT or DONE - only FIXED or ERROR."

attempt_build_fix() {
    local fix_log="$LOG_DIR/build_fix_${RUN_ID}_$(date +%H%M%S).log"

    log "${YELLOW}🔧 Attempting to fix build...${NC}"
    log "   Log: $fix_log"

    if run_agent "$fix_log" "$BUILD_FIX_PROMPT"; then
        local output=$(cat "$fix_log")

        if echo "$output" | grep -q "^FIXED$\|FIXED$"; then
            log "${GREEN}✓ Build fix reported success${NC}"

            # Verify the fix actually worked
            if verify_build; then
                # Commit the fix
                if [ "$AUTO_COMMIT" = "true" ]; then
                    commit_changes "BUILD-FIX" "Fix broken build"
                fi
                return 0
            else
                log "${RED}❌ Build still failing after fix attempt${NC}"
                return 1
            fi
        elif echo "$output" | grep -q "^ERROR:\|ERROR:"; then
            local error_msg=$(echo "$output" | grep "ERROR:" | head -1)
            log "${RED}❌ Build fix failed: $error_msg${NC}"
            return 1
        else
            log "${YELLOW}⚠ No status marker from build fix attempt${NC}"
            # Check if build works anyway
            if verify_build; then
                return 0
            fi
            return 1
        fi
    else
        log "${RED}❌ Build fix agent failed${NC}"
        return 1
    fi
}

#==============================================================================
# GIT OPERATIONS
#==============================================================================

commit_changes() {
    local task_id="$1"
    local task_desc="$2"

    if [ "$AUTO_COMMIT" != "true" ]; then
        return 0
    fi

    cd "$PROJECT_DIR"

    # Check for uncommitted changes
    if git diff --quiet HEAD 2>/dev/null && git diff --cached --quiet 2>/dev/null; then
        if [ -z "$(git ls-files --others --exclude-standard)" ]; then
            log "${YELLOW}No changes to commit${NC}"
            cd - > /dev/null
            return 0
        fi
    fi

    # Stage all changes
    git add -A

    # Build commit message
    local commit_msg
    if [ -n "$COMMIT_SCOPE" ]; then
        commit_msg="${COMMIT_PREFIX}(${COMMIT_SCOPE}): ${task_id} - ${task_desc}"
    else
        commit_msg="${COMMIT_PREFIX}: ${task_id} - ${task_desc}"
    fi

    if git commit -m "$commit_msg" 2>&1; then
        log "${GREEN}✓ Committed: ${commit_msg}${NC}"
    else
        log "${YELLOW}⚠ Git commit returned non-zero${NC}"
    fi

    cd - > /dev/null
}

#==============================================================================
# MAIN LOOP
#==============================================================================

main() {
    # Verify we're on an appropriate branch
    verify_branch

    # Header
    log ""
    log "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    log "${BLUE}   Ralph Loop - Automated AI Agent Task Runner${NC}"
    log "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    log ""
    log "Run ID:         ${RUN_ID}"
    log "Project:        ${PROJECT_DIR}"
    log "Agent:          ${AGENT_TYPE}"
    log "Max iterations: ${MAX_ITERATIONS}"
    log "Task file:      ${TASK_FILE}"
    log "Log directory:  ${LOG_DIR}"
    if [ "$TEST_RUN_ENABLED" = "true" ]; then
        log "Test run mode:  ${GREEN}ON${NC} (checkpoint after ${TEST_RUN_TASKS} tasks)"
    fi
    log ""

    # Initial build check
    if [ "$BUILD_GATE_ENABLED" = "true" ]; then
        log "${CYAN}Checking initial build state...${NC}"
        if ! verify_build; then
            log "${YELLOW}Build is broken - attempting fix before starting...${NC}"
            if ! attempt_build_fix; then
                log "${RED}═══════════════════════════════════════════════════════════════${NC}"
                log "${RED}STOPPING: Could not fix initial build failure${NC}"
                log "${RED}═══════════════════════════════════════════════════════════════${NC}"
                exit 1
            fi
        fi
        log ""
    fi

    INITIAL_REMAINING=$(count_remaining)
    INITIAL_COMPLETED=$(count_completed)
    log "Initial state: ${INITIAL_COMPLETED} completed, ${INITIAL_REMAINING} remaining"
    log ""

    local iteration=1
    local consecutive_failures=0
    local tasks_completed_this_run=0
    local checkpoint_passed=false

    while [ $iteration -le $MAX_ITERATIONS ]; do
        local REMAINING=$(count_remaining)
        local COMPLETED=$(count_completed)

        log ""
        log "${YELLOW}══════════════════════════════════════════════════════════════${NC}"
        log "${YELLOW}  Iteration ${iteration}/${MAX_ITERATIONS} │ ✅ ${COMPLETED} done │ 📋 ${REMAINING} remaining${NC}"
        log "${YELLOW}══════════════════════════════════════════════════════════════${NC}"

        if [ "$REMAINING" -eq 0 ]; then
            log "${GREEN}✓ All tasks completed!${NC}"
            break
        fi

        # Test run checkpoint: pause after first N tasks for user verification
        if [ "$TEST_RUN_ENABLED" = "true" ] && [ "$checkpoint_passed" = "false" ]; then
            if [ $tasks_completed_this_run -ge $TEST_RUN_TASKS ]; then
                log ""
                log "${CYAN}══════════════════════════════════════════════════════════════${NC}"
                log "${CYAN}   🔍 Test Run Checkpoint${NC}"
                log "${CYAN}══════════════════════════════════════════════════════════════${NC}"
                log ""
                log "The first ${TEST_RUN_TASKS} tasks have been completed."
                log "Please review the changes and verify everything is going according to plan."
                log ""
                log "You can check:"
                log "  • Git log: git log --oneline -${TEST_RUN_TASKS}"
                log "  • Git diff: git diff HEAD~${TEST_RUN_TASKS}"
                log "  • Build: run your build command"
                log ""
                echo -en "${BOLD}Continue with the remaining ${REMAINING} tasks? [y/N]: ${NC}"
                # Read from /dev/tty to handle piped execution scenarios
                read -r checkpoint_response </dev/tty
                checkpoint_response=$(echo "$checkpoint_response" | tr '[:upper:]' '[:lower:]')

                if [ "$checkpoint_response" = "y" ] || [ "$checkpoint_response" = "yes" ]; then
                    checkpoint_passed=true
                    log ""
                    log "${GREEN}✓ Checkpoint approved - continuing with remaining tasks${NC}"
                else
                    log ""
                    log "${YELLOW}Checkpoint not approved - stopping run${NC}"
                    log "You can review the changes and run Ralph Loop again when ready."
                    break
                fi
            fi
        fi

        # Show next task
        local NEXT_TASK=$(get_next_task)
        log "${BLUE}📌 Next task: ${NEXT_TASK}${NC}"
        log ""

        # Create iteration log
        local ITER_LOG="$LOG_DIR/iteration_${RUN_ID}_$(printf "%03d" $iteration).log"

        log "⏳ Starting agent at $(date '+%Y-%m-%d %H:%M:%S')..."
        log "   Log: ${ITER_LOG}"
        log ""

        local START_TIME=$(date +%s)

        if run_agent "$ITER_LOG"; then
            local END_TIME=$(date +%s)
            local DURATION=$((END_TIME - START_TIME))
            local MINUTES=$((DURATION / 60))
            local SECONDS=$((DURATION % 60))

            local OUTPUT=$(cat "$ITER_LOG")

            if echo "$OUTPUT" | grep -q "^NEXT$\|NEXT$"; then
                local TASK_ID=$(get_last_completed_task_id)
                local TASK_DESC=$(get_last_completed_task_description)
                log ""
                log "${GREEN}✅ SUCCESS: ${TASK_ID} completed in ${MINUTES}m ${SECONDS}s${NC}"
                consecutive_failures=0
                tasks_completed_this_run=$((tasks_completed_this_run + 1))

                # Verify build after task completion
                if [ "$BUILD_GATE_ENABLED" = "true" ]; then
                    if ! verify_build; then
                        log "${YELLOW}Build broken after task - attempting fix...${NC}"
                        if ! attempt_build_fix; then
                            log "${RED}═══════════════════════════════════════════════════════════════${NC}"
                            log "${RED}STOPPING: Build broken and could not be fixed${NC}"
                            log "${RED}═══════════════════════════════════════════════════════════════${NC}"
                            exit 1
                        fi
                    fi
                fi

                # Commit changes
                if [ -n "$TASK_ID" ] && [ "$AUTO_COMMIT" = "true" ]; then
                    commit_changes "$TASK_ID" "$TASK_DESC"
                fi

            elif echo "$OUTPUT" | grep -q "^DONE$\|DONE$"; then
                local TASK_ID=$(get_last_completed_task_id)
                local TASK_DESC=$(get_last_completed_task_description)
                log ""
                log "${GREEN}🎉 ALL DONE! Final task ${TASK_ID} completed in ${MINUTES}m ${SECONDS}s${NC}"
                tasks_completed_this_run=$((tasks_completed_this_run + 1))

                # Final build check
                if [ "$BUILD_GATE_ENABLED" = "true" ]; then
                    verify_build
                fi

                # Commit final changes
                if [ -n "$TASK_ID" ] && [ "$AUTO_COMMIT" = "true" ]; then
                    commit_changes "$TASK_ID" "$TASK_DESC"
                fi

                break

            elif echo "$OUTPUT" | grep -q "^ERROR:\|ERROR:"; then
                local ERROR_MSG=$(echo "$OUTPUT" | grep "ERROR:" | head -1)
                log ""
                log "${RED}❌ ERROR after ${MINUTES}m ${SECONDS}s: ${ERROR_MSG}${NC}"
                consecutive_failures=$((consecutive_failures + 1))
            else
                log ""
                log "${YELLOW}⚠️  No status marker found after ${MINUTES}m ${SECONDS}s${NC}"
                consecutive_failures=0

                # Still try to commit if there were changes
                local TASK_ID=$(get_last_completed_task_id)
                local TASK_DESC=$(get_last_completed_task_description)
                if [ -n "$TASK_ID" ] && [ "$AUTO_COMMIT" = "true" ]; then
                    commit_changes "$TASK_ID" "$TASK_DESC"
                fi
            fi
        else
            local END_TIME=$(date +%s)
            local DURATION=$((END_TIME - START_TIME))
            local MINUTES=$((DURATION / 60))
            local SECONDS=$((DURATION % 60))
            log ""
            log "${RED}❌ Agent process failed after ${MINUTES}m ${SECONDS}s${NC}"
            log "${RED}   Check log: ${ITER_LOG}${NC}"
            consecutive_failures=$((consecutive_failures + 1))
        fi

        # Check for too many consecutive failures
        if [ $consecutive_failures -ge $MAX_CONSECUTIVE_FAILURES ]; then
            log "${RED}═══════════════════════════════════════════════════════════════${NC}"
            log "${RED}STOPPING: ${MAX_CONSECUTIVE_FAILURES} consecutive failures detected${NC}"
            log "${RED}Check logs for details: ${ITER_LOG}${NC}"
            log "${RED}═══════════════════════════════════════════════════════════════${NC}"
            exit 1
        fi

        iteration=$((iteration + 1))

        if [ $iteration -le $MAX_ITERATIONS ] && [ "$REMAINING" -gt 0 ]; then
            log "Pausing ${PAUSE_SECONDS}s before next iteration..."
            sleep $PAUSE_SECONDS
        fi
    done

    # Final summary
    log ""
    log "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    log "${BLUE}   Run Complete${NC}"
    log "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    local FINAL_REMAINING=$(count_remaining)
    local FINAL_COMPLETED=$(count_completed)
    local TASKS_DONE=$((FINAL_COMPLETED - INITIAL_COMPLETED))
    log "Tasks completed this run: ${TASKS_DONE}"
    log "Total completed: ${FINAL_COMPLETED}"
    log "Remaining: ${FINAL_REMAINING}"
    log "Iterations used: $((iteration - 1))"
    log "Master log: ${MASTER_LOG}"
    log ""

    if [ "$FINAL_REMAINING" -eq 0 ]; then
        log "${GREEN}🎉 All tasks are complete!${NC}"
    else
        log "${YELLOW}Run again to continue with remaining tasks.${NC}"
    fi
}

# Run main
main
