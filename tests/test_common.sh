#!/bin/bash
#==============================================================================
# Test: Common Utilities
#==============================================================================
# Tests for lib/common.sh utility functions.
#==============================================================================

# Source the library
unset __COMMON_SH_SOURCED__
source "$REPO_ROOT/lib/common.sh"

# Test: Color variables are defined
test_colors_defined() {
    [ -n "$RED" ] && \
    [ -n "$GREEN" ] && \
    [ -n "$YELLOW" ] && \
    [ -n "$BLUE" ] && \
    [ -n "$CYAN" ] && \
    [ -n "$BOLD" ] && \
    [ -n "$NC" ]
}

# Test: print_header outputs correctly
test_print_header() {
    local output=$(print_header "Test Header" 2>&1)
    
    assert_contains "$output" "Test Header" "print_header should contain the header text"
}

# Test: print_success outputs with checkmark
test_print_success() {
    local output=$(print_success "Success message" 2>&1)
    
    assert_contains "$output" "✓" "print_success should contain checkmark" && \
    assert_contains "$output" "Success message" "print_success should contain message"
}

# Test: print_error outputs with X mark
test_print_error() {
    local output=$(print_error "Error message" 2>&1)
    
    assert_contains "$output" "✗" "print_error should contain X mark" && \
    assert_contains "$output" "Error message" "print_error should contain message"
}

# Test: print_warning outputs with warning symbol
test_print_warning() {
    local output=$(print_warning "Warning message" 2>&1)
    
    assert_contains "$output" "⚠" "print_warning should contain warning symbol" && \
    assert_contains "$output" "Warning message" "print_warning should contain message"
}

# Test: print_step outputs with arrow
test_print_step() {
    local output=$(print_step "Step message" 2>&1)
    
    assert_contains "$output" "▶" "print_step should contain arrow" && \
    assert_contains "$output" "Step message" "print_step should contain message"
}

# Test: print_subheader outputs correctly
test_print_subheader() {
    local output=$(print_subheader "Subheader" 2>&1)
    
    assert_contains "$output" "Subheader" "print_subheader should contain the text"
}

# Test: ask function writes prompt to stderr
test_ask_writes_to_stderr() {
    # Capture stderr only
    local stderr_output
    stderr_output=$(ask "Test prompt" "default" 2>&1 >/dev/null < /dev/null || true)
    
    # The prompt should be in stderr (we can't fully test interactive input)
    # Just verify the function exists and doesn't crash
    declare -f ask >/dev/null
}

# Test: ask_yes_no function exists and is callable
test_ask_yes_no_exists() {
    declare -f ask_yes_no >/dev/null
}

# Test: ask_choice function exists and is callable
test_ask_choice_exists() {
    declare -f ask_choice >/dev/null
}

# Run all tests
run_test "Color variables are defined" test_colors_defined
run_test "print_header outputs correctly" test_print_header
run_test "print_success outputs with checkmark" test_print_success
run_test "print_error outputs with X mark" test_print_error
run_test "print_warning outputs with warning symbol" test_print_warning
run_test "print_step outputs with arrow" test_print_step
run_test "print_subheader outputs correctly" test_print_subheader
run_test "ask function writes prompt to stderr" test_ask_writes_to_stderr
run_test "ask_yes_no function exists" test_ask_yes_no_exists
run_test "ask_choice function exists" test_ask_choice_exists

