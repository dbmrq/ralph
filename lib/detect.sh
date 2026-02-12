#!/bin/bash
#
# detect.sh - Project Detection Library
#
# This is a library file meant to be sourced by other scripts.
# It provides functions for detecting Xcode-specific configurations.
#
# Usage:
#   source "$(dirname "${BASH_SOURCE[0]}")/detect.sh"
#   schemes=$(detect_xcode_schemes "/path/to/project")
#

# Guard against double-sourcing
if [ -n "$__DETECT_SH_SOURCED__" ]; then
    return 0
fi
__DETECT_SH_SOURCED__=1

# Source common utilities
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

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

