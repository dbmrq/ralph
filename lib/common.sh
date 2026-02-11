#!/bin/bash
#
# lib/common.sh - Shared utility functions library
#
# This file contains common utility functions meant to be sourced by other scripts.
# It provides color definitions, print functions, and interactive input functions.
#
# Usage:
#   source "$(dirname "$0")/lib/common.sh"
#   or
#   source lib/common.sh
#

# Guard to prevent double-sourcing
if [ -n "$__COMMON_SH_SOURCED__" ]; then
    return 0
fi
__COMMON_SH_SOURCED__=1

#==============================================================================
# COLOR DEFINITIONS
#==============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

#==============================================================================
# PRINT FUNCTIONS
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

#==============================================================================
# INTERACTIVE FUNCTIONS
#==============================================================================

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

