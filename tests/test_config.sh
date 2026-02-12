#!/bin/bash
#==============================================================================
# Test: Config Generation
#==============================================================================
# Tests for lib/config.sh configuration file generation.
#==============================================================================

# Source the libraries
unset __COMMON_SH_SOURCED__
unset RALPH_CONFIG_LIB_LOADED
unset __PROMPTS_SH_SOURCED__
source "$REPO_ROOT/lib/config.sh"
source "$REPO_ROOT/lib/prompts.sh"

# Test: create_config_file creates config.sh
test_create_config_file_exists() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph"
    mkdir -p "$ralph_dir"

    create_config_file "$ralph_dir" "TestProject" "cursor" "50" "true"

    assert_file_exists "$ralph_dir/config.sh" "config.sh should be created"
}

# Test: config.sh contains project name
test_config_contains_project_name() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph2"
    mkdir -p "$ralph_dir"

    create_config_file "$ralph_dir" "MyAwesomeProject" "cursor" "50" "true"

    local content=$(cat "$ralph_dir/config.sh")
    assert_contains "$content" 'PROJECT_NAME="MyAwesomeProject"' "Should contain project name"
}

# Test: config.sh contains agent type
test_config_contains_agent_type() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph3"
    mkdir -p "$ralph_dir"

    create_config_file "$ralph_dir" "TestProject" "auggie" "50" "true"

    local content=$(cat "$ralph_dir/config.sh")
    assert_contains "$content" 'AGENT_TYPE="auggie"' "Should contain agent type"
}

# Test: config.sh contains max iterations
test_config_max_iterations() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph4"
    mkdir -p "$ralph_dir"

    create_config_file "$ralph_dir" "TestProject" "cursor" "100" "true"

    local content=$(cat "$ralph_dir/config.sh")
    assert_contains "$content" "MAX_ITERATIONS=100" "Should contain max iterations"
}

# Test: config.sh contains build gate setting
test_config_build_gate() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph5"
    mkdir -p "$ralph_dir"

    create_config_file "$ralph_dir" "TestProject" "cursor" "50" "false"

    local content=$(cat "$ralph_dir/config.sh")
    assert_contains "$content" "BUILD_GATE_ENABLED=false" "Should contain build gate setting"
}

# Test: config.sh has valid bash syntax
test_config_valid_syntax() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph6"
    mkdir -p "$ralph_dir"

    create_config_file "$ralph_dir" "TestProject" "cursor" "50" "true"

    bash -n "$ralph_dir/config.sh"
}

# Test: create_build_script creates build.sh
test_create_build_script_exists() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph7"
    mkdir -p "$ralph_dir"

    create_build_script "$ralph_dir"

    assert_file_exists "$ralph_dir/build.sh" "build.sh should be created"
}

# Test: build.sh is executable
test_build_script_executable() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph8"
    mkdir -p "$ralph_dir"

    create_build_script "$ralph_dir"

    [ -x "$ralph_dir/build.sh" ]
}

# Test: create_test_script creates test.sh
test_create_test_script_exists() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph9"
    mkdir -p "$ralph_dir"

    create_test_script "$ralph_dir"

    assert_file_exists "$ralph_dir/test.sh" "test.sh should be created"
}

# Test: test.sh is executable
test_test_script_executable() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph10"
    mkdir -p "$ralph_dir"

    create_test_script "$ralph_dir"

    [ -x "$ralph_dir/test.sh" ]
}

# Test: build.sh contains placeholder warning
test_build_script_contains_placeholder() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph11"
    mkdir -p "$ralph_dir"

    create_build_script "$ralph_dir"

    local content=$(cat "$ralph_dir/build.sh")
    assert_contains "$content" "PLACEHOLDER" "build.sh should contain placeholder warning"
}

# Test: create_prompt_files creates both prompt files
test_create_prompt_files() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph12"
    mkdir -p "$ralph_dir"

    create_prompt_files "$ralph_dir" "TestProject"

    assert_file_exists "$ralph_dir/platform_prompt.txt" "platform_prompt.txt should be created"
    assert_file_exists "$ralph_dir/project_prompt.txt" "project_prompt.txt should be created"
}

# Test: build.sh has valid bash syntax
test_build_script_valid_syntax() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph13"
    mkdir -p "$ralph_dir"

    create_build_script "$ralph_dir"

    bash -n "$ralph_dir/build.sh"
}

# Test: test.sh has valid bash syntax
test_test_script_valid_syntax() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph14"
    mkdir -p "$ralph_dir"

    create_test_script "$ralph_dir"

    bash -n "$ralph_dir/test.sh"
}

# Test: RALPH_VERSION is defined
test_ralph_version_defined() {
    [ -n "$RALPH_VERSION" ]
}

# Run all tests
run_test "create_config_file creates config.sh" test_create_config_file_exists
run_test "config.sh contains project name" test_config_contains_project_name
run_test "config.sh contains agent type" test_config_contains_agent_type
run_test "config.sh contains max iterations" test_config_max_iterations
run_test "config.sh contains build gate setting" test_config_build_gate
run_test "config.sh has valid bash syntax" test_config_valid_syntax
run_test "create_build_script creates build.sh" test_create_build_script_exists
run_test "build.sh is executable" test_build_script_executable
run_test "create_test_script creates test.sh" test_create_test_script_exists
run_test "test.sh is executable" test_test_script_executable
run_test "build.sh contains placeholder" test_build_script_contains_placeholder
run_test "create_prompt_files creates both files" test_create_prompt_files
run_test "build.sh has valid bash syntax" test_build_script_valid_syntax
run_test "test.sh has valid bash syntax" test_test_script_valid_syntax
run_test "RALPH_VERSION is defined" test_ralph_version_defined

