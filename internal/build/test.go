// Package build provides build and test verification logic for ralph.
// This file implements BUILD-003: test verification with bootstrap awareness.
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

// TestResult contains the result of a test verification.
type TestResult struct {
	// Success indicates whether all tests passed.
	Success bool `json:"success"`
	// Skipped indicates whether tests were skipped (e.g., bootstrap phase, no test files).
	Skipped bool `json:"skipped"`
	// SkipReason explains why tests were skipped.
	SkipReason string `json:"skip_reason,omitempty"`
	// Command is the test command that was executed.
	Command string `json:"command,omitempty"`
	// Output is the raw output from the test command.
	Output string `json:"output,omitempty"`
	// Failures is a list of parsed test failures.
	Failures []TestFailure `json:"failures,omitempty"`
	// Duration is how long the tests took.
	Duration time.Duration `json:"duration"`
	// ExitCode is the exit code from the test command.
	ExitCode int `json:"exit_code"`
	// TotalTests is the total number of tests detected (if parseable).
	TotalTests int `json:"total_tests,omitempty"`
	// PassedTests is the number of passing tests detected (if parseable).
	PassedTests int `json:"passed_tests,omitempty"`
	// FailedTests is the number of failing tests detected (if parseable).
	FailedTests int `json:"failed_tests,omitempty"`
}

// TestFailure represents a single test failure.
type TestFailure struct {
	// TestName is the name of the failing test.
	TestName string `json:"test_name,omitempty"`
	// Package is the package containing the failing test.
	Package string `json:"package,omitempty"`
	// File is the file where the test is defined (if detected).
	File string `json:"file,omitempty"`
	// Line is the line number where the failure occurred (if detected).
	Line int `json:"line,omitempty"`
	// Message is the failure message or error.
	Message string `json:"message"`
}

// String returns a human-readable representation of the test failure.
func (f TestFailure) String() string {
	var sb strings.Builder
	if f.Package != "" {
		sb.WriteString(f.Package)
		sb.WriteString("/")
	}
	if f.TestName != "" {
		sb.WriteString(f.TestName)
	}
	if f.File != "" {
		sb.WriteString(" (")
		sb.WriteString(f.File)
		if f.Line > 0 {
			sb.WriteString(fmt.Sprintf(":%d", f.Line))
		}
		sb.WriteString(")")
	}
	if sb.Len() > 0 {
		sb.WriteString(": ")
	}
	sb.WriteString(f.Message)
	return sb.String()
}

// TestVerifier executes and verifies tests.
type TestVerifier struct {
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// Config is the test configuration.
	Config config.TestConfig
	// Analysis is the AI-driven project analysis.
	Analysis *ProjectAnalysis
}

// NewTestVerifier creates a new TestVerifier.
func NewTestVerifier(projectDir string, cfg config.TestConfig, analysis *ProjectAnalysis) *TestVerifier {
	return &TestVerifier{
		ProjectDir: projectDir,
		Config:     cfg,
		Analysis:   analysis,
	}
}

// Verify runs the test verification.
// It respects bootstrap state, config overrides, and parses test failures.
func (v *TestVerifier) Verify(ctx context.Context) (*TestResult, error) {
	// Check if tests should be skipped
	if skip, reason := v.shouldSkip(); skip {
		return &TestResult{
			Success:    true, // Skipping is considered success (not a blocking failure)
			Skipped:    true,
			SkipReason: reason,
		}, nil
	}

	// Determine the test command
	command := v.getCommand()
	if command == "" {
		return &TestResult{
			Success:    true,
			Skipped:    true,
			SkipReason: "no test command available",
		}, nil
	}

	// Execute the tests
	return v.executeTests(ctx, command)
}

// shouldSkip checks if tests should be skipped.
// Returns (should skip, reason for skipping).
func (v *TestVerifier) shouldSkip() (bool, string) {
	// Check analysis state
	if v.Analysis != nil {
		if v.Analysis.IsGreenfield {
			return true, "greenfield project (no test files yet)"
		}
		if !v.Analysis.Test.Ready {
			reason := "tests not ready"
			if v.Analysis.Test.Reason != "" {
				reason = v.Analysis.Test.Reason
			}
			return true, reason
		}
		if !v.Analysis.Test.HasTestFiles {
			return true, "no test files found"
		}
	}
	return false, ""
}

// getCommand returns the test command to use.
// Config override takes precedence over AI-detected command.
func (v *TestVerifier) getCommand() string {
	// Config override takes precedence
	if v.Config.Command != "" {
		return v.Config.Command
	}

	// Use AI-detected command
	if v.Analysis != nil && v.Analysis.Test.Command != nil {
		return *v.Analysis.Test.Command
	}

	return ""
}

// executeTests runs the test command and returns the result.
func (v *TestVerifier) executeTests(ctx context.Context, command string) (*TestResult, error) {
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

	result := &TestResult{
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
			result.Failures = []TestFailure{{Message: "tests timed out"}}
			return result, nil
		} else if ctx.Err() == context.Canceled {
			result.Success = false
			result.Failures = []TestFailure{{Message: "tests were canceled"}}
			return result, nil
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Success = false
			result.Failures = parseTestFailures(output)
		} else {
			return nil, fmt.Errorf("failed to execute test command: %w", err)
		}
	} else {
		result.Success = true
	}

	return result, nil
}

// parseTestFailures extracts test failures from test output.
// It supports common test output formats from various test frameworks.
func parseTestFailures(output string) []TestFailure {
	var failures []TestFailure
	lines := strings.Split(output, "\n")

	// Track current package for Go test output
	var currentPackage string
	// Track lines that have been processed as part of Go test failures
	processedLines := make(map[int]bool)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Go test output patterns
		if failure, usedLines := parseGoTestFailure(line, currentPackage, lines, i); failure != nil {
			failures = append(failures, *failure)
			// Mark all lines used by this failure as processed
			for _, lineIdx := range usedLines {
				processedLines[lineIdx] = true
			}
			continue
		}

		// Skip if this line was already processed as part of a Go test failure
		if processedLines[i] {
			continue
		}

		// Track package changes in Go test output
		if strings.HasPrefix(line, "--- FAIL:") || strings.HasPrefix(line, "=== RUN") {
			// Reset for next test
			continue
		}

		// Detect Go package failures
		if strings.HasPrefix(line, "FAIL\t") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentPackage = parts[1]
			}
			continue
		}

		// Generic failure patterns
		lineLower := strings.ToLower(line)
		if containsTestFailureKeyword(lineLower) && !isKnownNonFailure(lineLower) {
			failures = append(failures, TestFailure{Message: line})
		}
	}

	return failures
}

// parseGoTestFailure attempts to parse a Go test failure line.
// Returns the failure and a list of line indices that were processed.
func parseGoTestFailure(line, currentPackage string, lines []string, lineIndex int) (*TestFailure, []int) {
	// Pattern: --- FAIL: TestName (duration)
	if strings.HasPrefix(line, "--- FAIL:") {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			testName := parts[2]
			failure := &TestFailure{
				TestName: testName,
				Package:  currentPackage,
			}
			usedLines := []int{lineIndex}

			// Look for failure message in subsequent lines
			for j := lineIndex + 1; j < len(lines) && j < lineIndex+10; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" || strings.HasPrefix(nextLine, "---") || strings.HasPrefix(nextLine, "===") {
					break
				}
				usedLines = append(usedLines, j)
				// Check for file:line pattern
				if loc := parseGoTestLocation(nextLine); loc != nil {
					failure.File = loc.file
					failure.Line = loc.line
					failure.Message = loc.message
					break
				}
				if failure.Message == "" {
					failure.Message = nextLine
				}
			}
			return failure, usedLines
		}
	}

	return nil, nil
}

type testLocation struct {
	file    string
	line    int
	message string
}

// parseGoTestLocation extracts file:line:message from Go test output.
func parseGoTestLocation(line string) *testLocation {
	// Go test format: file_test.go:123: message
	if !strings.Contains(line, "_test.go:") {
		return nil
	}

	colonIdx := strings.Index(line, ":")
	if colonIdx <= 0 {
		return nil
	}

	file := line[:colonIdx]
	rest := line[colonIdx+1:]

	// Parse line number
	nextColonIdx := strings.Index(rest, ":")
	if nextColonIdx <= 0 {
		return nil
	}

	lineStr := rest[:nextColonIdx]
	var lineNum int
	if _, err := fmt.Sscanf(lineStr, "%d", &lineNum); err != nil {
		return nil
	}

	message := strings.TrimSpace(rest[nextColonIdx+1:])

	return &testLocation{
		file:    file,
		line:    lineNum,
		message: message,
	}
}

// containsTestFailureKeyword checks if a line contains common test failure keywords.
func containsTestFailureKeyword(lineLower string) bool {
	failureKeywords := []string{
		"fail:",
		"failed:",
		"failure:",
		"error:",
		"assertion failed",
		"expected",
		"actual",
		"not equal",
		"timeout",
	}
	for _, keyword := range failureKeywords {
		if strings.Contains(lineLower, keyword) {
			return true
		}
	}
	return false
}

// isKnownNonFailure checks if a line is a known non-failure message.
func isKnownNonFailure(lineLower string) bool {
	// Some lines contain failure keywords but aren't actual failures
	nonFailures := []string{
		"no test files",
		"build successful",
		"0 failures",
		"0 failed",
		"all tests passed",
		"tests passed",
	}
	for _, nf := range nonFailures {
		if strings.Contains(lineLower, nf) {
			return true
		}
	}
	return false
}
