#!/bin/bash
#
# Ralph Loop - Build Script
#
# This script verifies the project builds successfully.
# Exit code 0 = success, non-zero = failure.
#
# ⚠️  PLACEHOLDER - Configure this for your project!
#

set -e

# Navigate to project root (parent of .ralph directory)
cd "$(dirname "$0")/.."

#==============================================================================
# BUILD COMMAND - Replace the placeholder below with your build command
#==============================================================================

# Examples:
#
# iOS / Xcode:
#   xcodebuild -scheme "MyApp" -destination 'platform=iOS Simulator,name=iPhone 16' build
#
# Swift Package:
#   swift build
#
# Node.js:
#   npm run build
#
# Python:
#   ruff check . || python -m py_compile **/*.py
#
# Rust:
#   cargo build
#
# Go:
#   go build ./...

# PLACEHOLDER: Remove this block and add your build command above
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "⚠️  BUILD SCRIPT NOT CONFIGURED"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Edit .ralph/build.sh to add your project's build command."
echo "Run the AI setup assistant to configure automatically:"
echo "  auggie \"Configure .ralph/build.sh for this project\""
echo ""
exit 1

