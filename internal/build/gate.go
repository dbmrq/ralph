// Package build provides build and test verification logic for ralph.
// This file implements BUILD-005: verification gate logic that orchestrates
// build and test verification with support for different modes and task overrides.
package build

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/task"
)

// GateStatus represents the overall result of gate verification.
type GateStatus string

const (
	// GateStatusPassed indicates all gates passed successfully.
	GateStatusPassed GateStatus = "passed"
	// GateStatusSkipped indicates the gate was skipped (bootstrap phase or task override).
	GateStatusSkipped GateStatus = "skipped"
	// GateStatusSkippedByTask indicates the gate was skipped due to task metadata.
	GateStatusSkippedByTask GateStatus = "skipped_by_task"
	// GateStatusFailed indicates the gate check failed.
	GateStatusFailed GateStatus = "failed"
)

// GateResult contains the overall result of verification gate checks.
type GateResult struct {
	// Status is the overall gate status.
	Status GateStatus `json:"status"`
	// Reason is a human-readable explanation of the result.
	Reason string `json:"reason"`
	// BuildResult contains the build verification result (nil if build not run).
	BuildResult *BuildResult `json:"build_result,omitempty"`
	// TestResult contains the test verification result (nil if tests not run).
	TestResult *TestResult `json:"test_result,omitempty"`
	// TDDResult contains the TDD mode result (nil if not in TDD mode or tests not run).
	TDDResult *TDDResult `json:"tdd_result,omitempty"`
	// BuildSkipped indicates whether the build was skipped.
	BuildSkipped bool `json:"build_skipped"`
	// BuildSkipReason explains why the build was skipped.
	BuildSkipReason string `json:"build_skip_reason,omitempty"`
	// TestSkipped indicates whether tests were skipped.
	TestSkipped bool `json:"test_skipped"`
	// TestSkipReason explains why tests were skipped.
	TestSkipReason string `json:"test_skip_reason,omitempty"`
}

// Passed returns true if the gate check passed (including skipped as a pass).
func (r *GateResult) Passed() bool {
	return r.Status == GateStatusPassed || r.Status == GateStatusSkipped || r.Status == GateStatusSkippedByTask
}

// VerificationGate orchestrates build and test verification for a task.
// It supports different test modes (gate, tdd, report) and task-level overrides.
type VerificationGate struct {
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// BuildConfig is the build configuration.
	BuildConfig config.BuildConfig
	// TestConfig is the test configuration.
	TestConfig config.TestConfig
	// Analysis is the AI-driven project analysis.
	Analysis *ProjectAnalysis
	// SessionID is the current session ID (for TDD baseline scoping).
	SessionID string
}

// NewVerificationGate creates a new VerificationGate.
func NewVerificationGate(projectDir string, buildCfg config.BuildConfig, testCfg config.TestConfig, analysis *ProjectAnalysis) *VerificationGate {
	return &VerificationGate{
		ProjectDir:  projectDir,
		BuildConfig: buildCfg,
		TestConfig:  testCfg,
		Analysis:    analysis,
	}
}

// TaskGateOverride represents parsed gate override from task metadata.
type TaskGateOverride struct {
	// BuildNotRequired indicates the task has marked build as not required.
	BuildNotRequired bool
	// TestNotRequired indicates the task has marked tests as not required.
	TestNotRequired bool
}

// Patterns for parsing task metadata gate overrides.
// Supported patterns (case-insensitive):
// - Tests: Not required / Tests: None / Tests: N/A / Tests: Skip
// - Build: Not required / Build: None / Build: N/A / Build: Skip
// - No tests needed / No tests required
// - No build needed / No build required
var (
	testNotRequiredPattern  = regexp.MustCompile(`(?i)(?:tests?\s*:\s*(?:not\s+required|none|n/?a|skip))|(?:no\s+tests?\s+(?:needed|required))`)
	buildNotRequiredPattern = regexp.MustCompile(`(?i)(?:build\s*:\s*(?:not\s+required|none|n/?a|skip))|(?:no\s+build\s+(?:needed|required))`)
)

// ParseTaskGateOverride parses gate overrides from task description.
// It looks for patterns like "Tests: Not required" or "Build: Not required".
func ParseTaskGateOverride(t *task.Task) *TaskGateOverride {
	if t == nil {
		return &TaskGateOverride{}
	}

	// Check both description and metadata
	text := t.Description
	if name, ok := t.GetMetadata("test_gate"); ok {
		text += " " + name
	}
	if name, ok := t.GetMetadata("build_gate"); ok {
		text += " " + name
	}

	return &TaskGateOverride{
		BuildNotRequired: buildNotRequiredPattern.MatchString(text),
		TestNotRequired:  testNotRequiredPattern.MatchString(text),
	}
}

// Verify runs the verification gate for a given task.
// It orchestrates the flow: build verification → test verification → mode-specific logic.
func (g *VerificationGate) Verify(ctx context.Context, t *task.Task) (*GateResult, error) {
	override := ParseTaskGateOverride(t)
	return g.VerifyWithOverride(ctx, override)
}

// VerifyWithOverride runs the verification gate with explicit overrides.
// This is useful when the caller has already parsed task overrides or wants to
// provide custom override behavior.
func (g *VerificationGate) VerifyWithOverride(ctx context.Context, override *TaskGateOverride) (*GateResult, error) {
	result := &GateResult{
		Status: GateStatusPassed,
	}

	// Step 1: Run build verification
	if err := g.runBuildVerification(ctx, result, override); err != nil {
		return nil, err
	}

	// If build failed, return immediately (no point running tests)
	if result.Status == GateStatusFailed {
		return result, nil
	}

	// Step 2: Run test verification
	if err := g.runTestVerification(ctx, result, override); err != nil {
		return nil, err
	}

	return result, nil
}

// runBuildVerification runs build verification and updates the result.
func (g *VerificationGate) runBuildVerification(ctx context.Context, result *GateResult, override *TaskGateOverride) error {
	// Check for task-level override
	if override != nil && override.BuildNotRequired {
		result.BuildSkipped = true
		result.BuildSkipReason = "build not required for this task"
		if result.Status == GateStatusPassed {
			result.Status = GateStatusSkippedByTask
			result.Reason = "build skipped per task metadata"
		}
		return nil
	}

	// Run build verification
	verifier := NewBuildVerifier(g.ProjectDir, g.BuildConfig, g.Analysis)
	buildResult, err := verifier.Verify(ctx)
	if err != nil {
		return fmt.Errorf("build verification failed: %w", err)
	}

	result.BuildResult = buildResult

	// Handle build result
	if buildResult.Skipped {
		result.BuildSkipped = true
		result.BuildSkipReason = buildResult.SkipReason
		// Don't change status to skipped if we're already passed - build skip is normal
		return nil
	}

	if !buildResult.Success {
		result.Status = GateStatusFailed
		result.Reason = g.formatBuildFailure(buildResult)
		return nil
	}

	return nil
}

// runTestVerification runs test verification and updates the result.
func (g *VerificationGate) runTestVerification(ctx context.Context, result *GateResult, override *TaskGateOverride) error {
	// Check for task-level override
	if override != nil && override.TestNotRequired {
		result.TestSkipped = true
		result.TestSkipReason = "tests not required for this task"
		if result.Status == GateStatusPassed {
			result.Status = GateStatusSkippedByTask
			result.Reason = "tests skipped per task metadata"
		}
		return nil
	}

	// Run test verification
	testVerifier := NewTestVerifier(g.ProjectDir, g.TestConfig, g.Analysis)
	testResult, err := testVerifier.Verify(ctx)
	if err != nil {
		return fmt.Errorf("test verification failed: %w", err)
	}

	result.TestResult = testResult

	// Handle test result based on mode
	switch g.TestConfig.Mode {
	case config.TestModeGate:
		return g.handleGateMode(result, testResult)
	case config.TestModeTDD:
		return g.handleTDDMode(ctx, result, testResult)
	case config.TestModeReport:
		return g.handleReportMode(result, testResult)
	default:
		// Default to gate mode
		return g.handleGateMode(result, testResult)
	}
}

// handleGateMode handles test results in gate mode (fail on any test failure).
func (g *VerificationGate) handleGateMode(result *GateResult, testResult *TestResult) error {
	// Handle skipped tests
	if testResult.Skipped {
		result.TestSkipped = true
		result.TestSkipReason = testResult.SkipReason
		if result.Status == GateStatusPassed {
			result.Status = GateStatusSkipped
			result.Reason = "tests skipped: " + testResult.SkipReason
		}
		return nil
	}

	// In gate mode, any test failure blocks
	if !testResult.Success {
		result.Status = GateStatusFailed
		result.Reason = g.formatTestFailure(testResult)
		return nil
	}

	// All tests passed
	if result.Status == GateStatusPassed && result.Reason == "" {
		result.Reason = "all checks passed"
	}
	return nil
}

// handleTDDMode handles test results in TDD mode (baseline comparison).
func (g *VerificationGate) handleTDDMode(ctx context.Context, result *GateResult, testResult *TestResult) error {
	// Handle skipped tests
	if testResult.Skipped {
		result.TestSkipped = true
		result.TestSkipReason = testResult.SkipReason
		if result.Status == GateStatusPassed {
			result.Status = GateStatusSkipped
			result.Reason = "tests skipped: " + testResult.SkipReason
		}
		return nil
	}

	// Run TDD evaluation
	tddManager := NewTDDManager(g.ProjectDir, g.TestConfig, g.Analysis, g.SessionID)
	tddResult, err := tddManager.Evaluate(testResult)
	if err != nil {
		return fmt.Errorf("TDD evaluation failed: %w", err)
	}

	result.TDDResult = tddResult

	// Handle TDD result
	if tddResult.Skipped {
		result.TestSkipped = true
		result.TestSkipReason = tddResult.SkipReason
		if result.Status == GateStatusPassed {
			result.Status = GateStatusSkipped
			result.Reason = tddResult.Message
		}
		return nil
	}

	if !tddResult.Passed {
		result.Status = GateStatusFailed
		result.Reason = tddResult.Message
		return nil
	}

	// TDD check passed
	if result.Status == GateStatusPassed && result.Reason == "" {
		result.Reason = tddResult.Message
	}
	return nil
}

// handleReportMode handles test results in report mode (never fail, just report).
func (g *VerificationGate) handleReportMode(result *GateResult, testResult *TestResult) error {
	// Handle skipped tests
	if testResult.Skipped {
		result.TestSkipped = true
		result.TestSkipReason = testResult.SkipReason
		if result.Status == GateStatusPassed {
			result.Status = GateStatusSkipped
			result.Reason = "tests skipped: " + testResult.SkipReason
		}
		return nil
	}

	// In report mode, we never fail, just report the results
	if !testResult.Success {
		// Report the failure but don't block
		if result.Status == GateStatusPassed && result.Reason == "" {
			result.Reason = fmt.Sprintf("tests failed (%d failures), continuing in report mode",
				len(testResult.Failures))
		}
	} else {
		if result.Status == GateStatusPassed && result.Reason == "" {
			result.Reason = "all tests passed"
		}
	}
	return nil
}

// formatBuildFailure formats build failure for the result reason.
func (g *VerificationGate) formatBuildFailure(buildResult *BuildResult) string {
	if len(buildResult.Errors) == 0 {
		return "build failed"
	}
	if len(buildResult.Errors) == 1 {
		return fmt.Sprintf("build failed: %s", buildResult.Errors[0].Message)
	}
	// Summarize multiple errors
	var msgs []string
	for _, e := range buildResult.Errors[:min(3, len(buildResult.Errors))] {
		msgs = append(msgs, e.Message)
	}
	summary := strings.Join(msgs, "; ")
	if len(buildResult.Errors) > 3 {
		summary += fmt.Sprintf(" (+%d more)", len(buildResult.Errors)-3)
	}
	return fmt.Sprintf("build failed: %s", summary)
}

// formatTestFailure formats test failure for the result reason.
func (g *VerificationGate) formatTestFailure(testResult *TestResult) string {
	if len(testResult.Failures) == 0 {
		return "tests failed"
	}
	if len(testResult.Failures) == 1 {
		f := testResult.Failures[0]
		if f.TestName != "" {
			return fmt.Sprintf("test failed: %s", f.TestName)
		}
		return fmt.Sprintf("test failed: %s", f.Message)
	}
	return fmt.Sprintf("%d tests failed", len(testResult.Failures))
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
