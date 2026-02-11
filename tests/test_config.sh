#!/bin/bash
#==============================================================================
# Test: Config Generation
#==============================================================================
# Tests for lib/config.sh configuration file generation.
#==============================================================================

# Source the library
unset __COMMON_SH_SOURCED__
unset RALPH_CONFIG_LIB_LOADED
source "$REPO_ROOT/lib/config.sh"

# Test: create_config_file creates config.sh
test_create_config_file_exists() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph"
    mkdir -p "$ralph_dir"
    
    create_config_file "$ralph_dir" "TestProject" "generic" "cursor" "" "." "" "" "" "50" "true"
    
    assert_file_exists "$ralph_dir/config.sh" "config.sh should be created"
}

# Test: config.sh contains project name
test_config_contains_project_name() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph2"
    mkdir -p "$ralph_dir"
    
    create_config_file "$ralph_dir" "MyAwesomeProject" "generic" "cursor" "" "." "" "" "" "50" "true"
    
    local content=$(cat "$ralph_dir/config.sh")
    assert_contains "$content" 'PROJECT_NAME="MyAwesomeProject"' "Should contain project name"
}

# Test: config.sh contains agent type
test_config_contains_agent_type() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph3"
    mkdir -p "$ralph_dir"
    
    create_config_file "$ralph_dir" "TestProject" "generic" "auggie" "" "." "" "" "" "50" "true"
    
    local content=$(cat "$ralph_dir/config.sh")
    assert_contains "$content" 'AGENT_TYPE="auggie"' "Should contain agent type"
}

# Test: iOS config contains Xcode scheme
test_config_ios_xcode_scheme() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph4"
    mkdir -p "$ralph_dir"
    
    create_config_file "$ralph_dir" "iOSApp" "ios" "cursor" "MyScheme" "." "" "" "ios" "50" "true"
    
    local content=$(cat "$ralph_dir/config.sh")
    assert_contains "$content" 'XCODE_SCHEME="MyScheme"' "Should contain Xcode scheme"
}

# Test: iOS config contains project_build function
test_config_ios_build_function() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph5"
    mkdir -p "$ralph_dir"
    
    create_config_file "$ralph_dir" "iOSApp" "ios" "cursor" "MyScheme" "." "" "" "ios" "50" "true"
    
    local content=$(cat "$ralph_dir/config.sh")
    assert_contains "$content" "project_build()" "Should contain project_build function"
}

# Test: config.sh contains max iterations
test_config_max_iterations() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph6"
    mkdir -p "$ralph_dir"
    
    create_config_file "$ralph_dir" "TestProject" "generic" "cursor" "" "." "" "" "" "100" "true"
    
    local content=$(cat "$ralph_dir/config.sh")
    assert_contains "$content" "MAX_ITERATIONS=100" "Should contain max iterations"
}

# Test: config.sh contains build gate setting
test_config_build_gate() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph7"
    mkdir -p "$ralph_dir"
    
    create_config_file "$ralph_dir" "TestProject" "generic" "cursor" "" "." "" "" "" "50" "false"
    
    local content=$(cat "$ralph_dir/config.sh")
    assert_contains "$content" "BUILD_GATE_ENABLED=false" "Should contain build gate setting"
}

# Test: config.sh has valid bash syntax
test_config_valid_syntax() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph8"
    mkdir -p "$ralph_dir"
    
    create_config_file "$ralph_dir" "TestProject" "ios" "cursor" "MyScheme" "." "" "" "ios" "50" "true"
    
    bash -n "$ralph_dir/config.sh"
}

# Test: Python config contains build command
test_config_python_build_command() {
    local ralph_dir="$TEST_TEMP_DIR/.ralph9"
    mkdir -p "$ralph_dir"
    
    create_config_file "$ralph_dir" "PyProject" "python" "cursor" "" "." "pytest" "pytest --cov" "python" "50" "true"
    
    local content=$(cat "$ralph_dir/config.sh")
    assert_contains "$content" "project_build()" "Should contain project_build function"
}

# Test: RALPH_VERSION is defined
test_ralph_version_defined() {
    [ -n "$RALPH_VERSION" ]
}

# Run all tests
run_test "create_config_file creates config.sh" test_create_config_file_exists
run_test "config.sh contains project name" test_config_contains_project_name
run_test "config.sh contains agent type" test_config_contains_agent_type
run_test "iOS config contains Xcode scheme" test_config_ios_xcode_scheme
run_test "iOS config contains project_build function" test_config_ios_build_function
run_test "config.sh contains max iterations" test_config_max_iterations
run_test "config.sh contains build gate setting" test_config_build_gate
run_test "config.sh has valid bash syntax" test_config_valid_syntax
run_test "Python config contains build command" test_config_python_build_command
run_test "RALPH_VERSION is defined" test_ralph_version_defined

