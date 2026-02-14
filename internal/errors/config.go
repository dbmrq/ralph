// Package errors provides comprehensive error types for ralph.
// This file contains configuration-related errors.
package errors

import (
	"fmt"
	"strings"
)

// Configuration-related error constructors.

// ConfigNotFound creates an error for missing configuration.
func ConfigNotFound(configPath string) *RalphError {
	return &RalphError{
		Kind:    ErrConfig,
		Message: fmt.Sprintf("configuration file not found: %s", configPath),
		Details: map[string]string{
			"path": configPath,
		},
		Suggestion: `Initialize Ralph in your project:
  
  Option 1: Run setup interactively
    ralph init
  
  Option 2: Run with defaults
    ralph init --yes
  
  Option 3: Create config manually
    mkdir -p .ralph
    touch .ralph/config.yaml`,
		DocLink: "https://github.com/wexinc/ralph#configuration",
	}
}

// ConfigParseError creates an error for YAML parsing failures.
func ConfigParseError(configPath string, parseErr error) *RalphError {
	return &RalphError{
		Kind:    ErrConfig,
		Message: fmt.Sprintf("failed to parse configuration: %s", configPath),
		Cause:   parseErr,
		Details: map[string]string{
			"path": configPath,
		},
		Suggestion: `Check your config.yaml for syntax errors:
  1. Ensure proper YAML indentation (use spaces, not tabs)
  2. Check for missing colons or quotes
  3. Validate with: yamllint .ralph/config.yaml
  
Common issues:
  - Lists need proper '- ' prefix
  - String values with special chars need quotes
  - Nested keys need consistent indentation`,
	}
}

// ConfigValidationError creates an error for invalid configuration values.
func ConfigValidationError(field, message string, validOptions []string) *RalphError {
	suggestion := fmt.Sprintf("Fix the %q field in .ralph/config.yaml", field)
	if len(validOptions) > 0 {
		suggestion += fmt.Sprintf("\n  Valid options: %s", strings.Join(validOptions, ", "))
	}

	return &RalphError{
		Kind:    ErrConfig,
		Message: fmt.Sprintf("invalid configuration: %s", message),
		Details: map[string]string{
			"field": field,
		},
		Suggestion: suggestion,
	}
}

// ProjectNotInitialized creates an error when ralph is not set up.
func ProjectNotInitialized(projectDir string) *RalphError {
	return &RalphError{
		Kind:    ErrConfig,
		Message: "Ralph is not initialized in this project",
		Details: map[string]string{
			"directory": projectDir,
		},
		Suggestion: `Initialize Ralph:
  ralph init
  
This will:
  1. Create .ralph/ directory
  2. Analyze your project structure
  3. Set up configuration
  4. Help you create or import a task list`,
	}
}

// Task-related error constructors.

// NoTasksFound creates an error when no tasks are available.
func NoTasksFound(projectDir string) *RalphError {
	return &RalphError{
		Kind:    ErrTask,
		Message: "no tasks found",
		Details: map[string]string{
			"directory": projectDir,
		},
		Suggestion: `Create or import a task list:
  
  Option 1: Initialize with a task file
    ralph init --tasks ./TASKS.md
  
  Option 2: Run init interactively
    ralph init
    (Choose 'Describe your goal' to generate tasks from a description)
  
  Option 3: Create tasks.json manually
    Create .ralph/tasks.json with your task definitions`,
	}
}

// TaskNotFound creates an error when a specific task is not found.
func TaskNotFound(taskID string) *RalphError {
	return &RalphError{
		Kind:    ErrTask,
		Message: fmt.Sprintf("task not found: %s", taskID),
		Details: map[string]string{
			"task_id": taskID,
		},
		Suggestion: `Check that the task ID is correct.
  List available tasks: cat .ralph/tasks.json | jq '.[] | .id'`,
	}
}

// AllTasksComplete creates an informational "error" when all tasks are done.
func AllTasksComplete() *RalphError {
	return &RalphError{
		Kind:       ErrTask,
		Message:    "all tasks are complete",
		Suggestion: "Add new tasks or start a new project.",
	}
}

// SessionNotFound creates an error for missing session.
func SessionNotFound(sessionID string) *RalphError {
	return &RalphError{
		Kind:    ErrNotFound,
		Message: fmt.Sprintf("session not found: %s", sessionID),
		Details: map[string]string{
			"session_id": sessionID,
		},
		Suggestion: `The session may have been cleaned up or never existed.
  
  List available sessions:
    ls .ralph/sessions/
  
  Start a new session:
    ralph run`,
	}
}
