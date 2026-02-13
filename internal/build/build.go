// Package build provides build and test verification logic for ralph.
// This file implements BUILD-002: build verification with bootstrap awareness.
package build

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/wexinc/ralph/internal/config"
)

// BuildResult contains the result of a build verification.
type BuildResult struct {
	// Success indicates whether the build succeeded.
	Success bool `json:"success"`
	// Skipped indicates whether the build was skipped (e.g., bootstrap phase).
	Skipped bool `json:"skipped"`
	// SkipReason explains why the build was skipped.
	SkipReason string `json:"skip_reason,omitempty"`
	// Command is the build command that was executed.
	Command string `json:"command,omitempty"`
	// Output is the raw output from the build command.
	Output string `json:"output,omitempty"`
	// Errors is a list of parsed error messages.
	Errors []BuildError `json:"errors,omitempty"`
	// Duration is how long the build took.
	Duration time.Duration `json:"duration"`
	// ExitCode is the exit code from the build command.
	ExitCode int `json:"exit_code"`
}

// BuildError represents a single error from the build output.
type BuildError struct {
	// File is the file where the error occurred (if detected).
	File string `json:"file,omitempty"`
	// Line is the line number where the error occurred (if detected).
	Line int `json:"line,omitempty"`
	// Column is the column number where the error occurred (if detected).
	Column int `json:"column,omitempty"`
	// Message is the error message.
	Message string `json:"message"`
}

// String returns a human-readable representation of the build error.
func (e BuildError) String() string {
	if e.File != "" {
		if e.Line > 0 {
			if e.Column > 0 {
				return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
			}
			return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
		}
		return fmt.Sprintf("%s: %s", e.File, e.Message)
	}
	return e.Message
}

// BuildVerifier executes and verifies builds.
type BuildVerifier struct {
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// Config is the build configuration.
	Config config.BuildConfig
	// Analysis is the AI-driven project analysis.
	Analysis *ProjectAnalysis
}

// NewBuildVerifier creates a new BuildVerifier.
func NewBuildVerifier(projectDir string, cfg config.BuildConfig, analysis *ProjectAnalysis) *BuildVerifier {
	return &BuildVerifier{
		ProjectDir: projectDir,
		Config:     cfg,
		Analysis:   analysis,
	}
}

// Verify runs the build verification.
// It respects bootstrap state, config overrides, and parses build errors.
func (v *BuildVerifier) Verify(ctx context.Context) (*BuildResult, error) {
	// Check if build should be skipped
	if skip, reason := v.shouldSkip(); skip {
		return &BuildResult{
			Success:    true, // Skipping is considered success (not a blocking failure)
			Skipped:    true,
			SkipReason: reason,
		}, nil
	}

	// Determine the build command
	command := v.getCommand()
	if command == "" {
		return &BuildResult{
			Success:    true,
			Skipped:    true,
			SkipReason: "no build command available",
		}, nil
	}

	// Execute the build
	return v.executeBuild(ctx, command)
}

// shouldSkip checks if the build should be skipped.
// Returns (should skip, reason for skipping).
func (v *BuildVerifier) shouldSkip() (bool, string) {
	// Check analysis state
	if v.Analysis != nil {
		if v.Analysis.IsGreenfield {
			return true, "greenfield project (no buildable code yet)"
		}
		if !v.Analysis.Build.Ready {
			reason := "build not ready"
			if v.Analysis.Build.Reason != "" {
				reason = v.Analysis.Build.Reason
			}
			return true, reason
		}
	}
	return false, ""
}

// getCommand returns the build command to use.
// Config override takes precedence over AI-detected command.
func (v *BuildVerifier) getCommand() string {
	// Config override takes precedence
	if v.Config.Command != "" {
		return v.Config.Command
	}

	// Use AI-detected command
	if v.Analysis != nil && v.Analysis.Build.Command != nil {
		return *v.Analysis.Build.Command
	}

	return ""
}

// executeBuild runs the build command and returns the result.
func (v *BuildVerifier) executeBuild(ctx context.Context, command string) (*BuildResult, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = v.ProjectDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	// Combine stdout and stderr for full output
	output := stdout.String() + stderr.String()

	result := &BuildResult{
		Command:  command,
		Output:   output,
		Duration: duration,
		ExitCode: 0,
	}

	// Handle exit code
	if err != nil {
		// Check for context cancellation/timeout first (takes precedence)
		if ctx.Err() == context.DeadlineExceeded {
			result.Success = false
			result.Errors = []BuildError{{Message: "build timed out"}}
			return result, nil
		} else if ctx.Err() == context.Canceled {
			result.Success = false
			result.Errors = []BuildError{{Message: "build was canceled"}}
			return result, nil
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Success = false
			result.Errors = parseBuildErrors(output)
		} else {
			return nil, fmt.Errorf("failed to execute build command: %w", err)
		}
	} else {
		result.Success = true
	}

	return result, nil
}

// parseBuildErrors extracts error messages from build output.
// It supports common error formats from various build tools.
func parseBuildErrors(output string) []BuildError {
	var errors []BuildError
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to parse as file:line:col: message format (common for Go, GCC, clang, etc.)
		if err := parseFileLineError(line); err != nil {
			errors = append(errors, *err)
			continue
		}

		// Detect common error keywords
		lineLower := strings.ToLower(line)
		if containsErrorKeyword(lineLower) {
			errors = append(errors, BuildError{Message: line})
		}
	}

	return errors
}

// parseFileLineError attempts to parse an error in file:line:col: message format.
func parseFileLineError(line string) *BuildError {
	// Common patterns:
	// file.go:10:5: error message
	// file.go:10: error message
	// file.go: error message

	// Find first colon after potential file path
	firstColon := strings.Index(line, ":")
	if firstColon <= 0 {
		return nil
	}

	// Check if this looks like a Windows drive letter (C:)
	if firstColon == 1 && len(line) > 2 && line[2] == '\\' {
		// Skip Windows drive letter, find next colon
		nextColon := strings.Index(line[2:], ":")
		if nextColon <= 0 {
			return nil
		}
		firstColon = 2 + nextColon
	}

	file := line[:firstColon]
	rest := line[firstColon+1:]

	// Check if file looks like a valid file path
	if !looksLikeFilePath(file) {
		return nil
	}

	err := &BuildError{File: file}

	// Try to parse line number
	if colonIdx := strings.Index(rest, ":"); colonIdx > 0 {
		lineStr := strings.TrimSpace(rest[:colonIdx])
		var lineNum int
		if _, parseErr := fmt.Sscanf(lineStr, "%d", &lineNum); parseErr == nil {
			err.Line = lineNum
			rest = rest[colonIdx+1:]

			// Try to parse column number
			if colonIdx2 := strings.Index(rest, ":"); colonIdx2 > 0 {
				colStr := strings.TrimSpace(rest[:colonIdx2])
				var colNum int
				if _, parseErr := fmt.Sscanf(colStr, "%d", &colNum); parseErr == nil {
					err.Column = colNum
					rest = rest[colonIdx2+1:]
				}
			}
		}
	}

	err.Message = strings.TrimSpace(rest)
	if err.Message == "" {
		return nil
	}

	return err
}

// looksLikeFilePath checks if a string looks like a file path.
func looksLikeFilePath(s string) bool {
	// Must contain a dot (extension) or slash (path separator)
	if !strings.Contains(s, ".") && !strings.Contains(s, "/") && !strings.Contains(s, "\\") {
		return false
	}
	// Must not contain certain characters that indicate it's not a path
	if strings.ContainsAny(s, "()[]{}") {
		return false
	}
	return true
}

// containsErrorKeyword checks if a line contains common error keywords.
func containsErrorKeyword(lineLower string) bool {
	errorKeywords := []string{
		"error:",
		"error[",
		"fatal:",
		"fatal error",
		"undefined:",
		"cannot find",
		"not found",
		"compilation failed",
		"build failed",
		"linker error",
		"undefined reference",
		"undefined symbol",
	}
	for _, keyword := range errorKeywords {
		if strings.Contains(lineLower, keyword) {
			return true
		}
	}
	return false
}

