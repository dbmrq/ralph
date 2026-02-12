#!/bin/bash
#==============================================================================
# Ralph Loop - Task File Generation Library
#==============================================================================
#
# This is a library file meant to be sourced by other scripts.
# It provides functions for generating and managing Ralph Loop task files.
#
# Usage:
#   source "$(dirname "${BASH_SOURCE[0]}")/lib/tasks.sh"
#   create_tasks_file "/path/to/.ralph" "ios"
#

# Guard against double-sourcing
if [ -n "$__TASKS_LIB_SOURCED__" ]; then
    return 0
fi
__TASKS_LIB_SOURCED__=1

# Source common library
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

#==============================================================================
# CREATE TASKS FILE
#==============================================================================
# Generates .ralph/TASKS.md with sample tasks and guidelines.
# Uses templates from .ralph/templates/ (downloaded by download_template_files).
#
# Parameters:
#   $1 - ralph_dir: Path to .ralph directory
#
# Behavior:
#   - Checks for template in .ralph/templates/TASKS.md
#   - Falls back to inline template if not found
#   - Creates file with validation task, sample tasks, and writing tips
#
create_tasks_file() {
    local ralph_dir="$1"

    local tasks_file="$ralph_dir/TASKS.md"
    local template_file="$ralph_dir/templates/TASKS.md"

    if [ -f "$template_file" ]; then
        cp "$template_file" "$tasks_file"
    else
        # Fallback: create inline template
        # Create generic tasks file
        cat > "$tasks_file" << 'EOF'
# Task List

**Purpose:** Atomic tasks for automated agent completion
**Format:** `- [ ] TASK-ID: Description` (unchecked) or `- [x] TASK-ID: Description` (done)

---

## 🧪 Validation Task

- [ ] TEST-001: Verify the project builds successfully
  > Goal: Run the build command and ensure it passes
  > This validates the Ralph Loop setup is working correctly

---

## 📋 Your Tasks

<!-- Add your tasks here -->

- [ ] TASK-001: Your first task
  > Goal: Describe what this task should accomplish
  > Reference: Link to any relevant documentation

- [ ] TASK-002: Your second task
  > Goal: Describe what this task should accomplish

---

## Task Writing Tips

1. **One atomic change per task** - Completable in one agent run
2. **Clear success criteria** - Agent knows when it's done
3. **Include references** - Links to designs, specs, examples
4. **Order matters** - Dependencies come first
5. **Use consistent IDs** - Format: `PREFIX-###` (e.g., FEAT-001, AUTH-015)

Example of a well-written task:
```
- [ ] AUTH-003: Add password validation to signup form
  > Goal: Validate password meets requirements (8+ chars, 1 uppercase, 1 number)
  > Reference: See login form validation in Sources/Auth/LoginViewModel.swift
  > Notes: Show inline error messages, disable submit until valid
```
EOF
    fi
}

