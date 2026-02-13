#!/bin/bash
# Test script for Ralph Go

set -e

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "🧪 Go not installed - skipping tests"
    echo "   (Install Go to enable test verification)"
    exit 0
fi

# Check if we're in bootstrap phase (no go.mod yet)
if [ ! -f "go.mod" ]; then
    echo "🧪 Bootstrap phase: No go.mod found yet - skipping tests"
    echo "   (This is expected for initial setup tasks)"
    exit 0
fi

# Check if there are any Go test files
if ! find . -name "*_test.go" -not -path "./vendor/*" | head -1 | grep -q .; then
    echo "🧪 No test files found yet - skipping tests"
    echo "   (Tests will be required once there is testable code)"
    exit 0
fi

echo "Running tests..."

# Run tests with race detection and coverage
# -race: Detect data races (important for concurrent code)
# -cover: Show coverage percentage
# -v: Verbose output
go test -race -cover -v ./...

echo "✓ All tests passed!"

