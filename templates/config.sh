#!/bin/bash
#==============================================================================
# Ralph Loop Configuration
#==============================================================================
# Project settings for Ralph Loop automation.
# Edit these values to customize behavior.
#==============================================================================

#------------------------------------------------------------------------------
# Project Identity
#------------------------------------------------------------------------------
PROJECT_NAME="__PROJECT_NAME__"

#------------------------------------------------------------------------------
# Agent Configuration
#------------------------------------------------------------------------------
AGENT_TYPE="__AGENT_TYPE__"      # cursor, auggie, or custom
DEFAULT_MODEL=""                  # Leave empty to prompt at startup

#------------------------------------------------------------------------------
# Loop Behavior
#------------------------------------------------------------------------------
MAX_ITERATIONS=__MAX_ITERATIONS__
PAUSE_SECONDS=5
MAX_CONSECUTIVE_FAILURES=3

#------------------------------------------------------------------------------
# Test Run Mode
#------------------------------------------------------------------------------
# Pause after completing first N tasks for verification
TEST_RUN_ENABLED=true
TEST_RUN_TASKS=2

#------------------------------------------------------------------------------
# Branch Protection
#------------------------------------------------------------------------------
REQUIRE_BRANCH=true               # Require non-main branch
ALLOWED_BRANCHES=""               # Specific branches (empty = any non-main)

#------------------------------------------------------------------------------
# Auto-Commit Settings
#------------------------------------------------------------------------------
AUTO_COMMIT=true
COMMIT_PREFIX="feat"
COMMIT_SCOPE="__COMMIT_SCOPE__"

#------------------------------------------------------------------------------
# Build & Test Gates
#------------------------------------------------------------------------------
BUILD_GATE_ENABLED=__BUILD_GATE_ENABLED__
BUILD_FIX_ATTEMPTS=1

TEST_GATE_ENABLED=false
TEST_FIX_ATTEMPTS=1

