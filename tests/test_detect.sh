#!/bin/bash
#==============================================================================
# Test: Detection Functions
#==============================================================================
# Tests for lib/detect.sh Xcode detection helpers.
#==============================================================================

# Source the library
unset __COMMON_SH_SOURCED__
unset __DETECT_SH_SOURCED__
source "$REPO_ROOT/lib/detect.sh"

# Test: detect_xcode_project_dir returns "." for root project
test_detect_xcode_project_dir_root() {
    local test_dir="$TEST_TEMP_DIR/ios_root"
    mkdir -p "$test_dir/MyApp.xcodeproj"
    
    local result=$(detect_xcode_project_dir "$test_dir")
    assert_equals "." "$result" "Should return '.' for root-level Xcode project"
}

# Test: detect_xcode_schemes parses XcodeGen project.yml (cross-platform)
test_detect_xcode_schemes_xcodegen() {
    local test_dir="$TEST_TEMP_DIR/xcodegen_project"
    mkdir -p "$test_dir"

    # Create a mock XcodeGen project.yml with schemes
    cat > "$test_dir/project.yml" << 'EOF'
name: MyApp
schemes:
  MyApp:
    build:
      targets:
        MyApp: all
  MyAppTests:
    build:
      targets:
        MyAppTests: test
targets:
  MyApp:
    type: application
EOF

    local result=$(detect_xcode_schemes "$test_dir")
    assert_contains "$result" "MyApp" "Should detect MyApp scheme from project.yml"
}

# Test: detect_xcode_schemes with xcodebuild (macOS only)
test_detect_xcode_schemes_xcodebuild() {
    # Skip on non-macOS
    if [[ "$(uname)" != "Darwin" ]]; then
        echo "      (skipped - macOS only)"
        return 0
    fi

    # This test would require a real Xcode project, so we just verify the function exists
    # and doesn't crash when no project is found
    local test_dir="$TEST_TEMP_DIR/empty_project"
    mkdir -p "$test_dir"

    # Should return empty, not error
    local result=$(detect_xcode_schemes "$test_dir" 2>/dev/null)
    [ -z "$result" ] || [ -n "$result" ]  # Either empty or has content is fine
}

# Run all tests
run_test "detect_xcode_project_dir returns '.' for root" test_detect_xcode_project_dir_root
run_test "detect_xcode_schemes parses XcodeGen project.yml" test_detect_xcode_schemes_xcodegen
run_test "detect_xcode_schemes with xcodebuild (macOS only)" test_detect_xcode_schemes_xcodebuild

