#!/bin/bash
#
# Ralph Loop - Test Script
#
# This script runs the project's test suite.
# Exit code 0 = success, non-zero = failure.
#
# ⚠️  PLACEHOLDER - Configure this for your project!
#

set -e

# Navigate to project root (parent of .ralph directory)
cd "$(dirname "$0")/.."

#==============================================================================
# TEST COMMAND - Replace the placeholder below with your test command
#==============================================================================

# Examples:
#
# iOS / Xcode:
#   xcodebuild -scheme "MyApp" -destination 'platform=iOS Simulator,name=iPhone 16' test
#
# Swift Package:
#   swift test
#
# Node.js:
#   npm test
#
# Python:
#   pytest
#
# Rust:
#   cargo test
#
# Go:
#   go test ./...

# PLACEHOLDER: Remove this block and add your test command above
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "⚠️  TEST SCRIPT NOT CONFIGURED"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Edit .ralph/test.sh to add your project's test command."
echo "Run the AI setup assistant to configure automatically:"
echo "  auggie \"Configure .ralph/test.sh for this project\""
echo ""
exit 1

