#!/bin/bash
# Ralph Loop Configuration for Ralph Go Rewrite

# Project settings
PROJECT_NAME="ralph-go"
TASKS_FILE="$SCRIPT_DIR/.ralph/TASKS.md"

# Agent settings (use cursor or auggie)
AGENT_TYPE="auggie"
DEFAULT_MODEL="opus4.5"

# Iteration limits
MAX_ITERATIONS=100
MAX_ITERATIONS_PER_TASK=5

# Build verification
BUILD_ENABLED=true
BUILD_COMMAND="go build ./..."

# Test verification
TEST_ENABLED=true
TEST_COMMAND="go test ./..."

# Git settings
AUTO_COMMIT=true
COMMIT_MESSAGE_PREFIX="[ralph]"

# Prompt files (relative to project root)
BASE_PROMPT="core/base_prompt.txt"
PLATFORM_PROMPT=".ralph/platform_prompt.txt"
PROJECT_PROMPT=".ralph/project_prompt.txt"

# Log settings
LOG_DIR="$SCRIPT_DIR/.ralph/logs"

