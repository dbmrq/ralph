#!/bin/bash
#==============================================================================
# Test: Library Sourcing
#==============================================================================
# Tests that all library files can be sourced correctly without errors.
#==============================================================================

# Test: All library files have valid bash syntax
test_lib_syntax() {
    local lib_dir="$REPO_ROOT/lib"
    local failed=0
    
    for lib in "$lib_dir"/*.sh; do
        if ! bash -n "$lib" 2>/dev/null; then
            echo "      Syntax error in: $(basename "$lib")"
            failed=1
        fi
    done
    
    [ $failed -eq 0 ]
}

# Test: common.sh can be sourced
test_source_common() {
    unset __COMMON_SH_SOURCED__
    source "$REPO_ROOT/lib/common.sh"
    
    # Verify key functions exist
    declare -f print_header >/dev/null && \
    declare -f print_success >/dev/null && \
    declare -f print_error >/dev/null && \
    declare -f ask >/dev/null && \
    declare -f ask_yes_no >/dev/null && \
    declare -f ask_choice >/dev/null
}

# Test: detect.sh can be sourced
test_source_detect() {
    unset __COMMON_SH_SOURCED__
    unset __DETECT_SH_SOURCED__
    source "$REPO_ROOT/lib/detect.sh"

    # Verify key functions exist (Xcode helpers)
    declare -f detect_xcode_schemes >/dev/null && \
    declare -f detect_xcode_project_dir >/dev/null
}

# Test: git.sh can be sourced
test_source_git() {
    unset __COMMON_SH_SOURCED__
    unset __GIT_LIB_SOURCED__
    source "$REPO_ROOT/lib/git.sh"
    
    # Verify key functions exist
    declare -f is_git_repo >/dev/null && \
    declare -f get_current_branch >/dev/null && \
    declare -f is_protected_branch >/dev/null && \
    declare -f create_branch >/dev/null && \
    declare -f init_git_repo >/dev/null
}

# Test: config.sh can be sourced
test_source_config() {
    unset __COMMON_SH_SOURCED__
    unset RALPH_CONFIG_LIB_LOADED
    source "$REPO_ROOT/lib/config.sh"

    # Verify key functions exist
    declare -f create_config_file >/dev/null && \
    declare -f create_build_script >/dev/null && \
    declare -f create_test_script >/dev/null
}

# Test: tasks.sh can be sourced
test_source_tasks() {
    unset __COMMON_SH_SOURCED__
    unset __TASKS_LIB_SOURCED__
    source "$REPO_ROOT/lib/tasks.sh"
    
    # Verify key functions exist
    declare -f create_tasks_file >/dev/null
}

# Test: Double-sourcing guards work
test_double_sourcing_guard() {
    unset __COMMON_SH_SOURCED__
    
    # Source twice - should not error
    source "$REPO_ROOT/lib/common.sh"
    source "$REPO_ROOT/lib/common.sh"
    
    # Verify guard variable is set
    [ -n "$__COMMON_SH_SOURCED__" ]
}

# Test: install.sh has valid syntax
test_install_syntax() {
    bash -n "$REPO_ROOT/install.sh" 2>/dev/null
}

# Test: core/ralph_loop.sh has valid syntax
test_ralph_loop_syntax() {
    bash -n "$REPO_ROOT/core/ralph_loop.sh" 2>/dev/null
}

# Run all tests
run_test "All library files have valid bash syntax" test_lib_syntax
run_test "common.sh can be sourced" test_source_common
run_test "detect.sh can be sourced" test_source_detect
run_test "git.sh can be sourced" test_source_git
run_test "config.sh can be sourced" test_source_config
run_test "tasks.sh can be sourced" test_source_tasks
run_test "Double-sourcing guards work" test_double_sourcing_guard
run_test "install.sh has valid syntax" test_install_syntax
run_test "core/ralph_loop.sh has valid syntax" test_ralph_loop_syntax

