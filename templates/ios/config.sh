#!/bin/bash
#
# Ralph Loop - iOS Project Configuration Template
#
# Copy this file to your project's .ralph/config.sh and customize it.
#

#==============================================================================
# PROJECT SETTINGS
#==============================================================================

# Project name (for display purposes)
PROJECT_NAME="My iOS App"

# Xcode scheme name
XCODE_SCHEME="MyApp"

# Simulator destination
SIMULATOR_DESTINATION="platform=iOS Simulator,name=iPhone 15"

# Path to Xcode project/workspace directory (relative to project root)
XCODE_PROJECT_DIR="."

#==============================================================================
# AGENT SETTINGS
#==============================================================================

# Default agent to use: cursor, auggie, or custom
AGENT_TYPE="cursor"

#==============================================================================
# LOOP SETTINGS
#==============================================================================

# Maximum iterations before stopping
MAX_ITERATIONS=50

# Seconds to pause between iterations
PAUSE_SECONDS=5

# Stop after this many consecutive failures
MAX_CONSECUTIVE_FAILURES=3

#==============================================================================
# GIT SETTINGS
#==============================================================================

# Require running on a non-main branch
REQUIRE_BRANCH=true

# Allowed branches (empty = any non-main branch)
ALLOWED_BRANCHES=""

# Auto-commit after each task
AUTO_COMMIT=true

# Commit message prefix (e.g., "feat", "fix", "chore")
COMMIT_PREFIX="feat"

# Commit message scope (e.g., "ios", "android", "web")
COMMIT_SCOPE="ios"

#==============================================================================
# BUILD SETTINGS
#==============================================================================

# Enable build verification between tasks
BUILD_GATE_ENABLED=true

# Number of attempts to fix a broken build before stopping
BUILD_FIX_ATTEMPTS=1

#==============================================================================
# BUILD COMMANDS
#==============================================================================

# Build command - called to verify the project builds
project_build() {
    cd "$XCODE_PROJECT_DIR"
    xcodebuild \
        -scheme "$XCODE_SCHEME" \
        -destination "$SIMULATOR_DESTINATION" \
        build \
        2>&1
}

# Test command - called to run tests (optional)
project_test() {
    cd "$XCODE_PROJECT_DIR"
    xcodebuild \
        -scheme "$XCODE_SCHEME" \
        -destination "$SIMULATOR_DESTINATION" \
        test \
        2>&1
}

#==============================================================================
# CUSTOM AGENT (optional)
#==============================================================================

# Uncomment and customize if using AGENT_TYPE="custom"
# run_agent_custom() {
#     local prompt="$1"
#     local log_file="$2"
#     
#     # Your custom agent command here
#     # Example: my-custom-agent --prompt "$prompt" > "$log_file" 2>&1
# }

