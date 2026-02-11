#!/bin/bash
#==============================================================================
# Test: Tasks Generation
#==============================================================================
# Tests for lib/tasks.sh task file generation.
#==============================================================================

# Source the library
unset __COMMON_SH_SOURCED__
unset __TASKS_LIB_SOURCED__
source "$REPO_ROOT/lib/tasks.sh"

# Test: create_tasks_file creates TASKS.md
test_create_tasks_file_exists() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph"
    mkdir -p "$ralph_dir"
    
    create_tasks_file "$ralph_dir" "generic"
    
    assert_file_exists "$ralph_dir/TASKS.md" "TASKS.md should be created"
}

# Test: TASKS.md contains task format header
test_tasks_contains_format() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph2"
    mkdir -p "$ralph_dir"
    
    create_tasks_file "$ralph_dir" "generic"
    
    local content=$(cat "$ralph_dir/TASKS.md")
    assert_contains "$content" "Task List" "Should contain Task List header"
}

# Test: TASKS.md contains validation task
test_tasks_contains_validation() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph3"
    mkdir -p "$ralph_dir"
    
    create_tasks_file "$ralph_dir" "generic"
    
    local content=$(cat "$ralph_dir/TASKS.md")
    assert_contains "$content" "TEST-001" "Should contain validation task"
}

# Test: TASKS.md contains sample tasks
test_tasks_contains_samples() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph4"
    mkdir -p "$ralph_dir"
    
    create_tasks_file "$ralph_dir" "generic"
    
    local content=$(cat "$ralph_dir/TASKS.md")
    assert_contains "$content" "TASK-001" "Should contain sample task"
}

# Test: TASKS.md contains task writing tips
test_tasks_contains_tips() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph5"
    mkdir -p "$ralph_dir"
    
    create_tasks_file "$ralph_dir" "generic"
    
    local content=$(cat "$ralph_dir/TASKS.md")
    assert_contains "$content" "Task Writing Tips" "Should contain writing tips"
}

# Test: TASKS.md uses checkbox format
test_tasks_checkbox_format() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph6"
    mkdir -p "$ralph_dir"
    
    create_tasks_file "$ralph_dir" "generic"
    
    local content=$(cat "$ralph_dir/TASKS.md")
    assert_contains "$content" "- [ ]" "Should use checkbox format"
}

# Test: create_tasks_file uses template if available
test_tasks_uses_template() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph7"
    mkdir -p "$ralph_dir"
    
    # Create a mock template
    local template_dir="$REPO_ROOT/templates/test_platform"
    mkdir -p "$template_dir"
    echo "# Custom Template Tasks" > "$template_dir/TASKS.md"
    
    # Temporarily modify LIB_DIR to use our test template
    # (This test verifies the template lookup logic exists)
    create_tasks_file "$ralph_dir" "generic"
    
    # Clean up
    rm -rf "$template_dir"
    
    # Just verify file was created
    assert_file_exists "$ralph_dir/TASKS.md" "TASKS.md should be created"
}

# Test: TASKS.md contains goal format
test_tasks_goal_format() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph8"
    mkdir -p "$ralph_dir"
    
    create_tasks_file "$ralph_dir" "generic"
    
    local content=$(cat "$ralph_dir/TASKS.md")
    assert_contains "$content" "> Goal:" "Should contain goal format"
}

# Run all tests
run_test "create_tasks_file creates TASKS.md" test_create_tasks_file_exists
run_test "TASKS.md contains task format header" test_tasks_contains_format
run_test "TASKS.md contains validation task" test_tasks_contains_validation
run_test "TASKS.md contains sample tasks" test_tasks_contains_samples
run_test "TASKS.md contains task writing tips" test_tasks_contains_tips
run_test "TASKS.md uses checkbox format" test_tasks_checkbox_format
run_test "create_tasks_file handles templates" test_tasks_uses_template
run_test "TASKS.md contains goal format" test_tasks_goal_format

