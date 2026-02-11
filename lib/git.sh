#!/bin/bash
#==============================================================================
# Ralph Loop - Git Operations Library
#==============================================================================
#
# This is a bash library meant to be sourced by other scripts.
# It provides utility functions for common git operations.
#
# Usage:
#   source "$(dirname "${BASH_SOURCE[0]}")/lib/git.sh"
#
# Functions:
#   is_git_repo(dir)           - Check if directory is a git repository
#   get_current_branch(dir)    - Get the current git branch name
#   is_protected_branch(dir)   - Check if current branch is main/master
#   create_branch(dir, name)   - Create and switch to a new branch
#   init_git_repo(dir)         - Initialize a new git repository
#

# Guard against double-sourcing
if [ -n "$__GIT_LIB_SOURCED__" ]; then
    return 0
fi
__GIT_LIB_SOURCED__=1

# Source common library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

#==============================================================================
# is_git_repo - Check if a directory is a git repository
#==============================================================================
# Arguments:
#   $1 - Directory path (defaults to current directory)
# Returns:
#   0 if directory is a git repository, 1 otherwise
#==============================================================================
is_git_repo() {
    local dir="${1:-.}"
    
    if [ -d "$dir/.git" ]; then
        return 0
    fi
    
    # Also check using git command (handles submodules)
    (cd "$dir" && git rev-parse --git-dir >/dev/null 2>&1)
}

#==============================================================================
# get_current_branch - Get the current git branch name
#==============================================================================
# Arguments:
#   $1 - Directory path (defaults to current directory)
# Returns:
#   Prints the current branch name to stdout
#   Returns 1 if not a git repository
#==============================================================================
get_current_branch() {
    local dir="${1:-.}"
    
    if ! is_git_repo "$dir"; then
        return 1
    fi
    
    (cd "$dir" && git rev-parse --abbrev-ref HEAD 2>/dev/null)
}

#==============================================================================
# is_protected_branch - Check if current branch is main/master
#==============================================================================
# Arguments:
#   $1 - Directory path (defaults to current directory)
# Returns:
#   0 if on main/master, 1 otherwise
#==============================================================================
is_protected_branch() {
    local dir="${1:-.}"
    local current_branch
    
    current_branch=$(get_current_branch "$dir") || return 1
    
    if [ "$current_branch" = "main" ] || [ "$current_branch" = "master" ]; then
        return 0
    fi
    
    return 1
}

#==============================================================================
# create_branch - Create and switch to a new branch
#==============================================================================
# Arguments:
#   $1 - Directory path
#   $2 - Branch name
# Returns:
#   0 on success, 1 on failure
#==============================================================================
create_branch() {
    local dir="$1"
    local branch_name="$2"
    
    if [ -z "$dir" ] || [ -z "$branch_name" ]; then
        return 1
    fi
    
    if ! is_git_repo "$dir"; then
        return 1
    fi
    
    (cd "$dir" && git checkout -b "$branch_name" >/dev/null 2>&1) || \
    (cd "$dir" && git checkout "$branch_name" >/dev/null 2>&1)
}

#==============================================================================
# init_git_repo - Initialize a new git repository
#==============================================================================
# Arguments:
#   $1 - Directory path
# Returns:
#   0 on success, 1 on failure
#==============================================================================
init_git_repo() {
    local dir="$1"
    
    if [ -z "$dir" ]; then
        return 1
    fi
    
    if ! [ -d "$dir" ]; then
        return 1
    fi
    
    (cd "$dir" && git init >/dev/null 2>&1 && git checkout -b main >/dev/null 2>&1)
}

