// Package build provides build and test verification logic for ralph.
// This file implements BUILD-004: TDD mode support with test baseline capture and comparison.
package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wexinc/ralph/internal/config"
)

// TestBaseline represents the captured test baseline state.
type TestBaseline struct {
	// CapturedAt is when the baseline was captured.
	CapturedAt time.Time `json:"captured_at"`
	// Scope is the baseline scope (global, session, task).
	Scope config.BaselineScope `json:"scope"`
	// Passing is the list of test names that were passing when baseline was captured.
	Passing []string `json:"passing"`
	// Failing is the list of test names that were failing when baseline was captured.
	Failing []string `json:"failing"`
	// Skipped is the list of test names that were skipped when baseline was captured.
	Skipped []string `json:"skipped,omitempty"`
	// BootstrapCompletedAt is when the bootstrap phase ended (first tests appeared).
	BootstrapCompletedAt *time.Time `json:"bootstrap_completed_at,omitempty"`
	// SessionID is the session ID when the baseline was captured (for session scope).
	SessionID string `json:"session_id,omitempty"`
}

// TDDResult represents the result of TDD mode verification.
type TDDResult struct {
	// Passed indicates whether the TDD check passed (no regressions).
	Passed bool `json:"passed"`
	// Skipped indicates whether TDD check was skipped (bootstrap phase, no tests).
	Skipped bool `json:"skipped"`
	// SkipReason explains why TDD check was skipped.
	SkipReason string `json:"skip_reason,omitempty"`
	// BaselineCaptured indicates whether a new baseline was captured this run.
	BaselineCaptured bool `json:"baseline_captured"`
	// Regressions is the list of tests that were passing but now fail.
	Regressions []string `json:"regressions,omitempty"`
	// NewlyPassing is the list of tests that were failing but now pass.
	NewlyPassing []string `json:"newly_passing,omitempty"`
	// TotalPassing is the count of currently passing tests.
	TotalPassing int `json:"total_passing"`
	// TotalFailing is the count of currently failing tests.
	TotalFailing int `json:"total_failing"`
	// Message is a human-readable summary of the result.
	Message string `json:"message"`
}

// TDDManager handles TDD mode baseline capture and comparison.
type TDDManager struct {
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// Config is the test configuration.
	Config config.TestConfig
	// Analysis is the AI-driven project analysis.
	Analysis *ProjectAnalysis
	// SessionID is the current session ID (for session scope).
	SessionID string
}

// NewTDDManager creates a new TDDManager.
func NewTDDManager(projectDir string, cfg config.TestConfig, analysis *ProjectAnalysis, sessionID string) *TDDManager {
	return &TDDManager{
		ProjectDir: projectDir,
		Config:     cfg,
		Analysis:   analysis,
		SessionID:  sessionID,
	}
}

// baselinePath returns the path to the baseline file.
func (m *TDDManager) baselinePath() string {
	baselineFile := m.Config.BaselineFile
	if baselineFile == "" {
		baselineFile = config.DefaultBaselineFile
	}
	return filepath.Join(m.ProjectDir, baselineFile)
}

// LoadBaseline loads the existing baseline from disk.
func (m *TDDManager) LoadBaseline() (*TestBaseline, error) {
	data, err := os.ReadFile(m.baselinePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No baseline exists yet
		}
		return nil, fmt.Errorf("failed to read baseline: %w", err)
	}

	var baseline TestBaseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("failed to parse baseline: %w", err)
	}

	return &baseline, nil
}

// SaveBaseline saves the baseline to disk.
func (m *TDDManager) SaveBaseline(baseline *TestBaseline) error {
	// Ensure directory exists
	dir := filepath.Dir(m.baselinePath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create baseline directory: %w", err)
	}

	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}

	if err := os.WriteFile(m.baselinePath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write baseline: %w", err)
	}

	return nil
}

// CaptureBaseline captures a new baseline from the test result.
func (m *TDDManager) CaptureBaseline(result *TestResult) *TestBaseline {
	now := time.Now()
	baseline := &TestBaseline{
		CapturedAt: now,
		Scope:      m.Config.BaselineScope,
		Passing:    extractPassingTestNames(result),
		Failing:    extractFailingTestNames(result),
		Skipped:    []string{},
		SessionID:  m.SessionID,
	}

	// Set bootstrap completed time if this is the first baseline
	baseline.BootstrapCompletedAt = &now

	return baseline
}

// ShouldCaptureNewBaseline determines if a new baseline should be captured.
func (m *TDDManager) ShouldCaptureNewBaseline(existingBaseline *TestBaseline) bool {
	// No baseline exists - always capture
	if existingBaseline == nil {
		return true
	}

	// For task scope, always capture new baseline
	if m.Config.BaselineScope == config.BaselineScopeTask {
		return true
	}

	// For session scope, capture if session ID changed
	if m.Config.BaselineScope == config.BaselineScopeSession {
		return existingBaseline.SessionID != m.SessionID
	}

	// For global scope, only capture if no baseline exists (handled above)
	return false
}

// Evaluate compares current test results against the baseline.
// Returns TDDResult with pass/fail status and details.
func (m *TDDManager) Evaluate(testResult *TestResult) (*TDDResult, error) {
	// Check if we're in bootstrap phase (no tests exist yet)
	if m.shouldSkip() {
		return m.createSkipResult()
	}

	// If test result was skipped (e.g., test command not ready), treat as skip
	if testResult.Skipped {
		return &TDDResult{
			Passed:     true,
			Skipped:    true,
			SkipReason: testResult.SkipReason,
			Message:    fmt.Sprintf("TDD check skipped: %s", testResult.SkipReason),
		}, nil
	}

	// Load existing baseline
	baseline, err := m.LoadBaseline()
	if err != nil {
		return nil, fmt.Errorf("failed to load baseline: %w", err)
	}

	// Check if we should capture a new baseline
	if m.ShouldCaptureNewBaseline(baseline) {
		return m.captureAndPass(testResult)
	}

	// Compare current results against baseline
	return m.compareAgainstBaseline(testResult, baseline)
}

// shouldSkip checks if TDD evaluation should be skipped.
func (m *TDDManager) shouldSkip() bool {
	if m.Analysis == nil {
		return false
	}
	if m.Analysis.IsGreenfield {
		return true
	}
	if !m.Analysis.Test.Ready {
		return true
	}
	if !m.Analysis.Test.HasTestFiles {
		return true
	}
	return false
}

// createSkipResult creates a skip result for bootstrap phase.
func (m *TDDManager) createSkipResult() (*TDDResult, error) {
	reason := "bootstrap phase"
	if m.Analysis != nil {
		if m.Analysis.IsGreenfield {
			reason = "greenfield project (no tests yet)"
		} else if !m.Analysis.Test.HasTestFiles {
			reason = "no test files found"
		} else if !m.Analysis.Test.Ready {
			reason = m.Analysis.Test.Reason
			if reason == "" {
				reason = "tests not ready"
			}
		}
	}

	return &TDDResult{
		Passed:     true,
		Skipped:    true,
		SkipReason: reason,
		Message:    fmt.Sprintf("TDD check skipped: %s", reason),
	}, nil
}

// captureAndPass captures a new baseline and returns a passing result.
func (m *TDDManager) captureAndPass(testResult *TestResult) (*TDDResult, error) {
	baseline := m.CaptureBaseline(testResult)

	if err := m.SaveBaseline(baseline); err != nil {
		return nil, fmt.Errorf("failed to save baseline: %w", err)
	}

	passing := len(baseline.Passing)
	failing := len(baseline.Failing)

	var message string
	if passing == 0 && failing == 0 {
		message = "Baseline captured: no tests detected yet"
	} else {
		message = fmt.Sprintf("Baseline captured: %d passing, %d failing", passing, failing)
	}

	return &TDDResult{
		Passed:           true,
		BaselineCaptured: true,
		TotalPassing:     passing,
		TotalFailing:     failing,
		Message:          message,
	}, nil
}

// compareAgainstBaseline compares current test results against the baseline.
func (m *TDDManager) compareAgainstBaseline(testResult *TestResult, baseline *TestBaseline) (*TDDResult, error) {
	currentPassing := extractPassingTestNames(testResult)
	currentFailing := extractFailingTestNames(testResult)

	// Find regressions: tests that were passing but now fail
	regressions := findRegressions(baseline.Passing, currentPassing)

	// Find newly passing: tests that were failing but now pass
	newlyPassing := findNewlyPassing(baseline.Failing, currentPassing)

	result := &TDDResult{
		Passed:       len(regressions) == 0,
		Regressions:  regressions,
		NewlyPassing: newlyPassing,
		TotalPassing: len(currentPassing),
		TotalFailing: len(currentFailing),
	}

	// Build message
	if result.Passed {
		if len(newlyPassing) > 0 {
			result.Message = fmt.Sprintf("TDD check passed: %d tests now passing (no regressions)", len(newlyPassing))
		} else {
			result.Message = "TDD check passed: no regressions"
		}
	} else {
		result.Message = fmt.Sprintf("TDD check failed: %d regression(s) detected", len(regressions))
	}

	return result, nil
}

// findRegressions finds tests that were in the baseline passing list but are no longer passing.
func findRegressions(baselinePassing, currentPassing []string) []string {
	currentSet := make(map[string]bool)
	for _, t := range currentPassing {
		currentSet[t] = true
	}

	var regressions []string
	for _, t := range baselinePassing {
		if !currentSet[t] {
			regressions = append(regressions, t)
		}
	}
	return regressions
}

// findNewlyPassing finds tests that were in the baseline failing list but are now passing.
func findNewlyPassing(baselineFailing, currentPassing []string) []string {
	currentSet := make(map[string]bool)
	for _, t := range currentPassing {
		currentSet[t] = true
	}

	var newlyPassing []string
	for _, t := range baselineFailing {
		if currentSet[t] {
			newlyPassing = append(newlyPassing, t)
		}
	}
	return newlyPassing
}

// extractPassingTestNames extracts test names that passed from test result.
// If individual test names aren't available, returns a placeholder for test count.
func extractPassingTestNames(result *TestResult) []string {
	if result == nil {
		return []string{}
	}

	// If we have detailed test names from parsing, use them
	// For now, we use a simple approach based on exit code
	// TODO: Enhance to parse individual test names from output when available

	if result.Success {
		// All tests passed - if we know the count, generate placeholder names
		if result.PassedTests > 0 {
			names := make([]string, result.PassedTests)
			for i := 0; i < result.PassedTests; i++ {
				names[i] = fmt.Sprintf("test_%d", i+1)
			}
			return names
		}
		// We know tests passed but don't have details
		// Use a single placeholder to indicate "some tests passed"
		return []string{"_all_tests_passed_"}
	}

	// Some tests failed - extract from the result
	// PassedTests contains count of passing tests
	if result.PassedTests > 0 {
		names := make([]string, result.PassedTests)
		for i := 0; i < result.PassedTests; i++ {
			names[i] = fmt.Sprintf("test_%d", i+1)
		}
		return names
	}

	return []string{}
}

// extractFailingTestNames extracts test names that failed from test result.
func extractFailingTestNames(result *TestResult) []string {
	if result == nil {
		return []string{}
	}

	// Extract from parsed failures
	if len(result.Failures) > 0 {
		names := make([]string, 0, len(result.Failures))
		for _, f := range result.Failures {
			if f.TestName != "" {
				name := f.TestName
				if f.Package != "" {
					name = f.Package + "/" + f.TestName
				}
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			return names
		}
	}

	// Fallback: if we know the count
	if result.FailedTests > 0 {
		names := make([]string, result.FailedTests)
		for i := 0; i < result.FailedTests; i++ {
			names[i] = fmt.Sprintf("failed_test_%d", i+1)
		}
		return names
	}

	// If tests failed but we don't have details, use placeholder
	if !result.Success && !result.Skipped {
		return []string{"_some_tests_failed_"}
	}

	return []string{}
}
