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

# Test: is_git_repo returns true for git repository
test_is_git_repo_true() {
    local test_dir="$TEST_TEMP_DIR/git_repo"
    mkdir -p "$test_dir"
    (cd "$test_dir" && git init >/dev/null 2>&1)
    
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
    mkdir -p "$test_dir"
    (cd "$test_dir" && git init >/dev/null 2>&1 && git checkout -b main >/dev/null 2>&1)
    
    # Need at least one commit for branch to exist
    (cd "$test_dir" && touch file.txt && git add . && git commit -m "init" >/dev/null 2>&1)
    
    local branch=$(get_current_branch "$test_dir")
    assert_equals "main" "$branch" "Should return 'main' as current branch"
}

# Test: is_protected_branch returns true for main
test_is_protected_branch_main() {
    local test_dir="$TEST_TEMP_DIR/protected_test"
    mkdir -p "$test_dir"
    (cd "$test_dir" && git init >/dev/null 2>&1 && git checkout -b main >/dev/null 2>&1)
    (cd "$test_dir" && touch file.txt && git add . && git commit -m "init" >/dev/null 2>&1)
    
    is_protected_branch "$test_dir"
}

# Test: is_protected_branch returns true for master
test_is_protected_branch_master() {
    local test_dir="$TEST_TEMP_DIR/protected_test2"
    mkdir -p "$test_dir"
    (cd "$test_dir" && git init >/dev/null 2>&1 && git checkout -b master >/dev/null 2>&1)
    (cd "$test_dir" && touch file.txt && git add . && git commit -m "init" >/dev/null 2>&1)
    
    is_protected_branch "$test_dir"
}

# Test: is_protected_branch returns false for feature branch
test_is_protected_branch_feature() {
    local test_dir="$TEST_TEMP_DIR/feature_test"
    mkdir -p "$test_dir"
    (cd "$test_dir" && git init >/dev/null 2>&1 && git checkout -b feature/test >/dev/null 2>&1)
    (cd "$test_dir" && touch file.txt && git add . && git commit -m "init" >/dev/null 2>&1)
    
    ! is_protected_branch "$test_dir"
}

# Test: create_branch creates and switches to new branch
test_create_branch() {
    local test_dir="$TEST_TEMP_DIR/create_branch_test"
    mkdir -p "$test_dir"
    (cd "$test_dir" && git init >/dev/null 2>&1 && git checkout -b main >/dev/null 2>&1)
    (cd "$test_dir" && touch file.txt && git add . && git commit -m "init" >/dev/null 2>&1)
    
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
    
    # Add a commit so branch exists
    (cd "$test_dir" && touch file.txt && git add . && git commit -m "init" >/dev/null 2>&1)
    
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

