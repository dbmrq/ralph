#!/bin/bash
# Build script for Ralph Go

set -e

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "📦 Go not installed - skipping build"
    echo "   (Install Go to enable build verification)"
    exit 0
fi

# Check if we're in bootstrap phase (no go.mod yet)
if [ ! -f "go.mod" ]; then
    echo "📦 Bootstrap phase: No go.mod found yet - skipping build"
    echo "   (This is expected for initial setup tasks)"
    exit 0
fi

# Check if there are any Go files to build
if ! find . -name "*.go" -not -path "./vendor/*" | head -1 | grep -q .; then
    echo "📦 Bootstrap phase: No Go files found yet - skipping build"
    exit 0
fi

echo "Building Ralph Go..."

# First, verify the module is tidy
go mod tidy

# Run linter if available (catches issues early)
if command -v golangci-lint &> /dev/null; then
    echo "Running linter..."
    golangci-lint run ./... || {
        echo "⚠️  Linter found issues (see above)"
        # Don't fail the build on lint errors, but warn
        # Remove this || block to make lint errors fail the build
    }
else
    echo "ℹ️  golangci-lint not installed - skipping lint check"
    echo "   Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi

# Build all packages (catches compile errors everywhere)
go build ./...

# If cmd/ralph exists, build the binary
if [ -d "cmd/ralph" ] && [ -f "cmd/ralph/main.go" ]; then
    echo "Building ralph binary..."
    go build -o ralph ./cmd/ralph
fi

echo "✓ Build successful!"

