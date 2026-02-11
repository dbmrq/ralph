#!/bin/bash
#==============================================================================
# Ralph Loop - Test Runner
#==============================================================================
# Runs all tests for the Ralph Loop modular installation system.
#
# Usage:
#   ./tests/test_runner.sh
#
# Exit codes:
#   0 - All tests passed
#   1 - One or more tests failed
#==============================================================================

# Note: We don't use 'set -e' here because:
# 1. Arithmetic operations like ((var++)) return 1 when var is 0
# 2. We want tests to continue even if individual tests fail
# 3. We handle exit codes explicitly in the test framework

# Get the directory containing this script
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$TESTS_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Temporary directory for test fixtures
TEST_TEMP_DIR=""

#==============================================================================
# TEST FRAMEWORK
#==============================================================================

setup_test_env() {
    TEST_TEMP_DIR=$(mktemp -d)
    export TEST_TEMP_DIR
}

teardown_test_env() {
    if [ -n "$TEST_TEMP_DIR" ] && [ -d "$TEST_TEMP_DIR" ]; then
        rm -rf "$TEST_TEMP_DIR"
    fi
}

assert_equals() {
    local expected="$1"
    local actual="$2"
    local message="${3:-Values should be equal}"

    if [ "$expected" = "$actual" ]; then
        return 0
    else
        echo -e "    ${RED}✗ $message${NC}"
        echo -e "      Expected: '$expected'"
        echo -e "      Actual:   '$actual'"
        return 1
    fi
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local message="${3:-String should contain substring}"

    if [[ "$haystack" == *"$needle"* ]]; then
        return 0
    else
        echo -e "    ${RED}✗ $message${NC}"
        echo -e "      String: '$haystack'"
        echo -e "      Should contain: '$needle'"
        return 1
    fi
}

assert_file_exists() {
    local file="$1"
    local message="${2:-File should exist}"

    if [ -f "$file" ]; then
        return 0
    else
        echo -e "    ${RED}✗ $message${NC}"
        echo -e "      File not found: '$file'"
        return 1
    fi
}

assert_dir_exists() {
    local dir="$1"
    local message="${2:-Directory should exist}"

    if [ -d "$dir" ]; then
        return 0
    else
        echo -e "    ${RED}✗ $message${NC}"
        echo -e "      Directory not found: '$dir'"
        return 1
    fi
}

assert_true() {
    local condition="$1"
    local message="${2:-Condition should be true}"

    if eval "$condition"; then
        return 0
    else
        echo -e "    ${RED}✗ $message${NC}"
        return 1
    fi
}

assert_false() {
    local condition="$1"
    local message="${2:-Condition should be false}"

    if ! eval "$condition"; then
        return 0
    else
        echo -e "    ${RED}✗ $message${NC}"
        return 1
    fi
}

run_test() {
    local test_name="$1"
    local test_func="$2"

    ((TESTS_RUN++))

    echo -e "  ${CYAN}▶${NC} $test_name"

    # Run test in subshell to isolate failures
    if (setup_test_env && $test_func && teardown_test_env); then
        ((TESTS_PASSED++))
        echo -e "    ${GREEN}✓ Passed${NC}"
    else
        ((TESTS_FAILED++))
        echo -e "    ${RED}✗ Failed${NC}"
        teardown_test_env 2>/dev/null || true
    fi
}

run_test_suite() {
    local suite_name="$1"
    local suite_file="$2"

    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}$suite_name${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if [ -f "$suite_file" ]; then
        source "$suite_file"
    else
        echo -e "  ${RED}✗ Test file not found: $suite_file${NC}"
        ((TESTS_FAILED++))
    fi
}

#==============================================================================
# MAIN
#==============================================================================

main() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}   Ralph Loop Test Suite${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"

    # Run all test suites
    run_test_suite "Library Sourcing Tests" "$TESTS_DIR/test_sourcing.sh"
    run_test_suite "Common Utilities Tests" "$TESTS_DIR/test_common.sh"
    run_test_suite "Detection Tests" "$TESTS_DIR/test_detect.sh"
    run_test_suite "Git Functions Tests" "$TESTS_DIR/test_git.sh"
    run_test_suite "Config Generation Tests" "$TESTS_DIR/test_config.sh"
    run_test_suite "Tasks Generation Tests" "$TESTS_DIR/test_tasks.sh"

    # Summary
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}Test Summary${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "  Tests run:    $TESTS_RUN"
    echo -e "  ${GREEN}Passed:       $TESTS_PASSED${NC}"

    if [ $TESTS_FAILED -gt 0 ]; then
        echo -e "  ${RED}Failed:       $TESTS_FAILED${NC}"
        echo ""
        echo -e "${RED}✗ Some tests failed${NC}"
        exit 1
    else
        echo -e "  Failed:       0"
        echo ""
        echo -e "${GREEN}✓ All tests passed!${NC}"
        exit 0
    fi
}

main "$@"

