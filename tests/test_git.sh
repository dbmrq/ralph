#!/bin/bash
#==============================================================================
# Test: Git Functions
#==============================================================================
# Tests for lib/git.sh git operations.
#==============================================================================

# Source the library
unset __COMMON_SH_SOURCED__
unset __GIT_LIB_SOURCED__
source "$REPO_ROOT/lib/git.sh"

# Helper: Initialize a git repo with proper config for CI environments
init_test_git_repo() {
    local dir="$1"
    local branch="${2:-main}"
    mkdir -p "$dir"
    (
        cd "$dir" || return 1
        git init >/dev/null 2>&1
        # Configure git user for CI environments that don't have it
        git config user.email "test@example.com"
        git config user.name "Test User"
        git checkout -b "$branch" >/dev/null 2>&1
    )
}

# Helper: Make a commit in a git repo
make_test_commit() {
    local dir="$1"
    local filename="${2:-file.txt}"
    (
        cd "$dir" || return 1
        touch "$filename"
        git add .
        git commit -m "test commit" >/dev/null 2>&1
    )
}

# Test: is_git_repo returns true for git repository
test_is_git_repo_true() {
    local test_dir="$TEST_TEMP_DIR/git_repo"
    init_test_git_repo "$test_dir"

    is_git_repo "$test_dir"
}

# Test: is_git_repo returns false for non-git directory
test_is_git_repo_false() {
    local test_dir="$TEST_TEMP_DIR/not_git"
    mkdir -p "$test_dir"

    ! is_git_repo "$test_dir"
}

# Test: get_current_branch returns branch name
test_get_current_branch() {
    local test_dir="$TEST_TEMP_DIR/branch_test"
    init_test_git_repo "$test_dir" "main"
    make_test_commit "$test_dir"

    local branch=$(get_current_branch "$test_dir")
    assert_equals "main" "$branch" "Should return 'main' as current branch"
}

# Test: is_protected_branch returns true for main
test_is_protected_branch_main() {
    local test_dir="$TEST_TEMP_DIR/protected_test"
    init_test_git_repo "$test_dir" "main"
    make_test_commit "$test_dir"

    is_protected_branch "$test_dir"
}

# Test: is_protected_branch returns true for master
test_is_protected_branch_master() {
    local test_dir="$TEST_TEMP_DIR/protected_test2"
    init_test_git_repo "$test_dir" "master"
    make_test_commit "$test_dir"

    is_protected_branch "$test_dir"
}

# Test: is_protected_branch returns false for feature branch
test_is_protected_branch_feature() {
    local test_dir="$TEST_TEMP_DIR/feature_test"
    init_test_git_repo "$test_dir" "feature/test"
    make_test_commit "$test_dir"

    ! is_protected_branch "$test_dir"
}

# Test: create_branch creates and switches to new branch
test_create_branch() {
    local test_dir="$TEST_TEMP_DIR/create_branch_test"
    init_test_git_repo "$test_dir" "main"
    make_test_commit "$test_dir"

    create_branch "$test_dir" "feature/new-branch"

    local branch=$(get_current_branch "$test_dir")
    assert_equals "feature/new-branch" "$branch" "Should be on new branch"
}

# Test: init_git_repo initializes a new repository
test_init_git_repo() {
    local test_dir="$TEST_TEMP_DIR/init_test"
    mkdir -p "$test_dir"

    init_git_repo "$test_dir"

    is_git_repo "$test_dir"
}

# Test: init_git_repo creates main branch
test_init_git_repo_main_branch() {
    local test_dir="$TEST_TEMP_DIR/init_test2"
    mkdir -p "$test_dir"

    init_git_repo "$test_dir"

    # Configure git user and add a commit so branch exists
    (
        cd "$test_dir" || return 1
        git config user.email "test@example.com"
        git config user.name "Test User"
        touch file.txt
        git add .
        git commit -m "init" >/dev/null 2>&1
    )

    local branch=$(get_current_branch "$test_dir")
    assert_equals "main" "$branch" "Should initialize with main branch"
}

# Test: create_branch fails for non-git directory
test_create_branch_fails_non_git() {
    local test_dir="$TEST_TEMP_DIR/not_git2"
    mkdir -p "$test_dir"
    
    ! create_branch "$test_dir" "feature/test"
}

# Test: init_git_repo fails for non-existent directory
test_init_git_repo_fails_nonexistent() {
    ! init_git_repo "$TEST_TEMP_DIR/does_not_exist"
}

# Run all tests
run_test "is_git_repo returns true for git repository" test_is_git_repo_true
run_test "is_git_repo returns false for non-git directory" test_is_git_repo_false
run_test "get_current_branch returns branch name" test_get_current_branch
run_test "is_protected_branch returns true for main" test_is_protected_branch_main
run_test "is_protected_branch returns true for master" test_is_protected_branch_master
run_test "is_protected_branch returns false for feature branch" test_is_protected_branch_feature
run_test "create_branch creates and switches to new branch" test_create_branch
run_test "init_git_repo initializes a new repository" test_init_git_repo
run_test "init_git_repo creates main branch" test_init_git_repo_main_branch
run_test "create_branch fails for non-git directory" test_create_branch_fails_non_git
run_test "init_git_repo fails for non-existent directory" test_init_git_repo_fails_nonexistent

