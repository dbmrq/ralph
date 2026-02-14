// Package errors provides comprehensive error types for ralph.
// This file contains build, test, and verification-related errors.
package errors

import (
	"fmt"
	"strings"
)

// Build-related error constructors.

// BuildFailed creates an error for build failures.
func BuildFailed(command string, exitCode int, errorCount int) *RalphError {
	err := &RalphError{
		Kind:    ErrBuild,
		Message: "build failed",
		Details: map[string]string{
			"command":   command,
			"exit_code": fmt.Sprintf("%d", exitCode),
		},
	}
	if errorCount > 0 {
		err.Details["errors"] = fmt.Sprintf("%d errors found", errorCount)
	}
	err.Suggestion = `Review the build errors above and fix them.
  
Common fixes:
  • Missing imports: Add the required import statements
  • Type errors: Check variable types and function signatures
  • Missing dependencies: Run your package manager (go mod tidy, npm install, etc.)
  
Ralph will attempt to fix build errors automatically if enabled in config.`
	return err
}

// TestFailed creates an error for test failures.
func TestFailed(command string, failedCount, totalCount int) *RalphError {
	err := &RalphError{
		Kind:    ErrTest,
		Message: fmt.Sprintf("%d of %d tests failed", failedCount, totalCount),
		Details: map[string]string{
			"command":      command,
			"failed":       fmt.Sprintf("%d", failedCount),
			"total":        fmt.Sprintf("%d", totalCount),
			"passing_rate": fmt.Sprintf("%.1f%%", float64(totalCount-failedCount)/float64(totalCount)*100),
		},
	}
	err.Suggestion = `Review the test failures and fix them.
  
If using TDD mode:
  • Only regressions (previously passing tests now failing) block progress
  • Pre-existing failures are tracked but don't block
  
Configure test behavior in .ralph/config.yaml:
  test:
    mode: tdd    # gate | tdd | report`
	return err
}

// TestRegression creates an error specifically for TDD regressions.
func TestRegression(regressedTests []string) *RalphError {
	testList := strings.Join(regressedTests, "\n  • ")
	if len(regressedTests) > 5 {
		testList = strings.Join(regressedTests[:5], "\n  • ")
		testList += fmt.Sprintf("\n  ... and %d more", len(regressedTests)-5)
	}

	return &RalphError{
		Kind:    ErrTest,
		Message: fmt.Sprintf("%d test regressions detected", len(regressedTests)),
		Details: map[string]string{
			"regressed_tests": testList,
		},
		Suggestion: `These tests were passing before but are now failing.
  
Fix the regressions before continuing:
  1. Review the failing tests
  2. Identify what changed that broke them
  3. Fix the code or update the tests if behavior changed intentionally
  
Ralph will attempt to fix regressions automatically if enabled.`,
	}
}

// NoTestsFound creates an informational message when no tests exist.
func NoTestsFound(projectDir string) *RalphError {
	return &RalphError{
		Kind:    ErrTest,
		Message: "no test files found (bootstrap phase)",
		Details: map[string]string{
			"directory": projectDir,
		},
		Suggestion: `This is normal for new projects.
  
Test verification is skipped during the bootstrap phase.
Once you add test files, Ralph will:
  1. Detect them automatically
  2. Capture a test baseline
  3. Start enforcing test gates`,
	}
}

// Git-related error constructors.

// GitNotInitialized creates an error when git is not set up.
func GitNotInitialized(projectDir string) *RalphError {
	return &RalphError{
		Kind:    ErrGit,
		Message: "git repository not initialized",
		Details: map[string]string{
			"directory": projectDir,
		},
		Suggestion: `Initialize a git repository:
  git init
  
Ralph uses git for:
  • Automatic commits after task completion
  • Change tracking and rollback
  • Session state persistence`,
	}
}

// GitDirtyState creates an error for uncommitted changes.
func GitDirtyState(changedFiles int) *RalphError {
	return &RalphError{
		Kind:    ErrGit,
		Message: fmt.Sprintf("uncommitted changes detected (%d files)", changedFiles),
		Suggestion: `Commit or stash your changes before running Ralph:
  
  Option 1: Commit changes
    git add .
    git commit -m "WIP: save current state"
  
  Option 2: Stash changes
    git stash
    # After Ralph finishes:
    git stash pop
  
  Option 3: Disable auto-commit in config
    git:
      auto_commit: false`,
	}
}

// GitConflict creates an error for merge conflicts.
func GitConflict(conflictFiles []string) *RalphError {
	fileList := strings.Join(conflictFiles, "\n  • ")
	if len(conflictFiles) > 5 {
		fileList = strings.Join(conflictFiles[:5], "\n  • ")
		fileList += fmt.Sprintf("\n  ... and %d more", len(conflictFiles)-5)
	}

	return &RalphError{
		Kind:    ErrGit,
		Message: "merge conflicts detected",
		Details: map[string]string{
			"conflict_files": fileList,
		},
		Suggestion: `Resolve the merge conflicts before continuing:
  
  1. Open the conflicting files
  2. Look for <<<<<<< and >>>>>>> markers
  3. Edit to resolve the conflicts
  4. git add <file> for each resolved file
  5. git commit to complete the merge`,
	}
}

// CommitFailed creates an error for git commit failures.
func CommitFailed(stderr string) *RalphError {
	err := &RalphError{
		Kind:    ErrGit,
		Message: "failed to create commit",
	}
	if stderr != "" {
		err.Details = map[string]string{"output": stderr}
	}
	err.Suggestion = `Check git configuration:
  git config user.name "Your Name"
  git config user.email "your@email.com"
  
Or disable auto-commit in .ralph/config.yaml:
  git:
    auto_commit: false`
	return err
}

