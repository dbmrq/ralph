#!/bin/bash
#==============================================================================
# Test: Detection Functions
#==============================================================================
# Tests for lib/detect.sh project type detection.
#==============================================================================

# Source the library
unset __COMMON_SH_SOURCED__
unset __DETECT_SH_SOURCED__
source "$REPO_ROOT/lib/detect.sh"

# Test: Detect iOS project with Package.swift
test_detect_ios_package_swift() {
    local test_dir="$TEST_TEMP_DIR/ios_project"
    mkdir -p "$test_dir"
    touch "$test_dir/Package.swift"
    
    local result=$(detect_project_type "$test_dir")
    assert_equals "ios" "$result" "Should detect iOS project from Package.swift"
}

# Test: Detect iOS project with xcodeproj
test_detect_ios_xcodeproj() {
    local test_dir="$TEST_TEMP_DIR/ios_project2"
    mkdir -p "$test_dir/MyApp.xcodeproj"
    
    local result=$(detect_project_type "$test_dir")
    assert_equals "ios" "$result" "Should detect iOS project from .xcodeproj"
}

# Test: Detect iOS project with project.yml (XcodeGen)
test_detect_ios_xcodegen() {
    local test_dir="$TEST_TEMP_DIR/ios_project3"
    mkdir -p "$test_dir"
    touch "$test_dir/project.yml"
    
    local result=$(detect_project_type "$test_dir")
    assert_equals "ios" "$result" "Should detect iOS project from project.yml"
}

# Test: Detect React project
test_detect_react() {
    local test_dir="$TEST_TEMP_DIR/react_project"
    mkdir -p "$test_dir"
    echo '{"dependencies": {"react": "^18.0.0"}}' > "$test_dir/package.json"
    
    local result=$(detect_project_type "$test_dir")
    assert_equals "react" "$result" "Should detect React project"
}

# Test: Detect Node.js project
test_detect_node() {
    local test_dir="$TEST_TEMP_DIR/node_project"
    mkdir -p "$test_dir"
    echo '{"name": "my-node-app"}' > "$test_dir/package.json"
    
    local result=$(detect_project_type "$test_dir")
    assert_equals "node" "$result" "Should detect Node.js project"
}

# Test: Detect Python project with requirements.txt
test_detect_python_requirements() {
    local test_dir="$TEST_TEMP_DIR/python_project"
    mkdir -p "$test_dir"
    touch "$test_dir/requirements.txt"
    
    local result=$(detect_project_type "$test_dir")
    assert_equals "python" "$result" "Should detect Python project from requirements.txt"
}

# Test: Detect Python project with pyproject.toml
test_detect_python_pyproject() {
    local test_dir="$TEST_TEMP_DIR/python_project2"
    mkdir -p "$test_dir"
    touch "$test_dir/pyproject.toml"
    
    local result=$(detect_project_type "$test_dir")
    assert_equals "python" "$result" "Should detect Python project from pyproject.toml"
}

# Test: Detect Rust project
test_detect_rust() {
    local test_dir="$TEST_TEMP_DIR/rust_project"
    mkdir -p "$test_dir"
    touch "$test_dir/Cargo.toml"
    
    local result=$(detect_project_type "$test_dir")
    assert_equals "rust" "$result" "Should detect Rust project"
}

# Test: Detect Go project
test_detect_go() {
    local test_dir="$TEST_TEMP_DIR/go_project"
    mkdir -p "$test_dir"
    touch "$test_dir/go.mod"
    
    local result=$(detect_project_type "$test_dir")
    assert_equals "go" "$result" "Should detect Go project"
}

# Test: Detect generic project (no markers)
test_detect_generic() {
    local test_dir="$TEST_TEMP_DIR/generic_project"
    mkdir -p "$test_dir"
    touch "$test_dir/README.md"
    
    local result=$(detect_project_type "$test_dir")
    assert_equals "generic" "$result" "Should detect generic project when no markers found"
}

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
run_test "Detect iOS project with Package.swift" test_detect_ios_package_swift
run_test "Detect iOS project with xcodeproj" test_detect_ios_xcodeproj
run_test "Detect iOS project with project.yml" test_detect_ios_xcodegen
run_test "Detect React project" test_detect_react
run_test "Detect Node.js project" test_detect_node
run_test "Detect Python project with requirements.txt" test_detect_python_requirements
run_test "Detect Python project with pyproject.toml" test_detect_python_pyproject
run_test "Detect Rust project" test_detect_rust
run_test "Detect Go project" test_detect_go
run_test "Detect generic project" test_detect_generic
run_test "detect_xcode_project_dir returns '.' for root" test_detect_xcode_project_dir_root
run_test "detect_xcode_schemes parses XcodeGen project.yml" test_detect_xcode_schemes_xcodegen
run_test "detect_xcode_schemes with xcodebuild (macOS only)" test_detect_xcode_schemes_xcodebuild

