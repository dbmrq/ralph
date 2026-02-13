package build

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wexinc/ralph/internal/config"
	"github.com/wexinc/ralph/internal/task"
)

// TestParseTaskGateOverride tests the parsing of gate overrides from task metadata.
func TestParseTaskGateOverride(t *testing.T) {
	tests := []struct {
		name             string
		description      string
		metadata         map[string]string
		wantTestSkip     bool
		wantBuildSkip    bool
	}{
		{
			name:             "no overrides",
			description:      "Implement feature X",
			wantTestSkip:     false,
			wantBuildSkip:    false,
		},
		{
			name:             "tests not required",
			description:      "Setup task\nTests: Not required",
			wantTestSkip:     true,
			wantBuildSkip:    false,
		},
		{
			name:             "build not required",
			description:      "Config only task\nBuild: Not required",
			wantTestSkip:     false,
			wantBuildSkip:    true,
		},
		{
			name:             "both not required",
			description:      "Init task\nTests: Not required\nBuild: Not required",
			wantTestSkip:     true,
			wantBuildSkip:    true,
		},
		{
			name:             "tests none",
			description:      "Setup only (Tests: None)",
			wantTestSkip:     true,
			wantBuildSkip:    false,
		},
		{
			name:             "tests n/a",
			description:      "Documentation task (Tests: N/A)",
			wantTestSkip:     true,
			wantBuildSkip:    false,
		},
		{
			name:             "tests skip",
			description:      "Tests: Skip - no test infrastructure yet",
			wantTestSkip:     true,
			wantBuildSkip:    false,
		},
		{
			name:             "no tests needed phrase",
			description:      "Config change - no tests needed for this",
			wantTestSkip:     true,
			wantBuildSkip:    false,
		},
		{
			name:             "no tests required phrase",
			description:      "Documentation update - no tests required",
			wantTestSkip:     true,
			wantBuildSkip:    false,
		},
		{
			name:             "no build needed phrase",
			description:      "Metadata change - no build needed",
			wantTestSkip:     false,
			wantBuildSkip:    true,
		},
		{
			name:             "case insensitive",
			description:      "TESTS: NOT REQUIRED\nBUILD: NOT REQUIRED",
			wantTestSkip:     true,
			wantBuildSkip:    true,
		},
		{
			name:             "metadata override for tests",
			description:      "Regular task",
			metadata:        map[string]string{"test_gate": "Tests: skip"},
			wantTestSkip:     true,
			wantBuildSkip:    false,
		},
		{
			name:             "test singular form",
			description:      "Setup task (Test: Not required)",
			wantTestSkip:     true,
			wantBuildSkip:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := task.NewTask("TEST-001", "Test Task", tt.description)
			if tt.metadata != nil {
				for k, v := range tt.metadata {
					tk.SetMetadata(k, v)
				}
			}

			override := ParseTaskGateOverride(tk)

			if override.TestNotRequired != tt.wantTestSkip {
				t.Errorf("TestNotRequired = %v, want %v", override.TestNotRequired, tt.wantTestSkip)
			}
			if override.BuildNotRequired != tt.wantBuildSkip {
				t.Errorf("BuildNotRequired = %v, want %v", override.BuildNotRequired, tt.wantBuildSkip)
			}
		})
	}
}

// TestParseTaskGateOverride_NilTask tests handling of nil task.
func TestParseTaskGateOverride_NilTask(t *testing.T) {
	override := ParseTaskGateOverride(nil)
	if override.TestNotRequired || override.BuildNotRequired {
		t.Error("expected no overrides for nil task")
	}
}

// TestGateResult_Passed tests the Passed helper method.
func TestGateResult_Passed(t *testing.T) {
	tests := []struct {
		status GateStatus
		want   bool
	}{
		{GateStatusPassed, true},
		{GateStatusSkipped, true},
		{GateStatusSkippedByTask, true},
		{GateStatusFailed, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := &GateResult{Status: tt.status}
			if got := result.Passed(); got != tt.want {
				t.Errorf("Passed() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestVerificationGate_GateMode tests gate mode verification.
func TestVerificationGate_GateMode(t *testing.T) {
	// Create temp directory with passing build/test setup
	tmpDir := t.TempDir()

	// Analysis indicating ready project
	analysis := &ProjectAnalysis{
		ProjectType:  "test",
		Languages:    []string{"sh"},
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready:   true,
			Command: stringPtr("exit 0"),
			Reason:  "test build",
		},
		Test: TestAnalysis{
			Ready:        true,
			Command:      stringPtr("exit 0"),
			HasTestFiles: true,
			Reason:       "test tests",
		},
	}

	gate := NewVerificationGate(tmpDir, config.BuildConfig{}, config.TestConfig{Mode: config.TestModeGate}, analysis)

	result, err := gate.VerifyWithOverride(context.Background(), &TaskGateOverride{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != GateStatusPassed {
		t.Errorf("expected status %s, got %s: %s", GateStatusPassed, result.Status, result.Reason)
	}
}

// TestVerificationGate_GateMode_BuildFails tests gate mode with build failure.
func TestVerificationGate_GateMode_BuildFails(t *testing.T) {
	tmpDir := t.TempDir()

	analysis := &ProjectAnalysis{
		ProjectType:  "test",
		Languages:    []string{"sh"},
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready:   true,
			Command: stringPtr("exit 1"),
			Reason:  "failing build",
		},
		Test: TestAnalysis{
			Ready:        true,
			Command:      stringPtr("exit 0"),
			HasTestFiles: true,
		},
	}

	gate := NewVerificationGate(tmpDir, config.BuildConfig{}, config.TestConfig{Mode: config.TestModeGate}, analysis)

	result, err := gate.VerifyWithOverride(context.Background(), &TaskGateOverride{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != GateStatusFailed {
		t.Errorf("expected status %s, got %s", GateStatusFailed, result.Status)
	}
	if result.BuildResult == nil {
		t.Error("expected BuildResult to be populated")
	}
	if result.TestResult != nil {
		t.Error("expected TestResult to be nil (should not run tests after build failure)")
	}
}

// TestVerificationGate_GateMode_TestFails tests gate mode with test failure.
func TestVerificationGate_GateMode_TestFails(t *testing.T) {
	tmpDir := t.TempDir()

	analysis := &ProjectAnalysis{
		ProjectType:  "test",
		Languages:    []string{"sh"},
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready:   true,
			Command: stringPtr("exit 0"),
		},
		Test: TestAnalysis{
			Ready:        true,
			Command:      stringPtr("exit 1"),
			HasTestFiles: true,
		},
	}

	gate := NewVerificationGate(tmpDir, config.BuildConfig{}, config.TestConfig{Mode: config.TestModeGate}, analysis)

	result, err := gate.VerifyWithOverride(context.Background(), &TaskGateOverride{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != GateStatusFailed {
		t.Errorf("expected status %s, got %s", GateStatusFailed, result.Status)
	}
}

// TestVerificationGate_ReportMode tests report mode (never fails).
func TestVerificationGate_ReportMode(t *testing.T) {
	tmpDir := t.TempDir()

	analysis := &ProjectAnalysis{
		ProjectType:  "test",
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready:   true,
			Command: stringPtr("exit 0"),
		},
		Test: TestAnalysis{
			Ready:        true,
			Command:      stringPtr("exit 1"), // Tests fail
			HasTestFiles: true,
		},
	}

	gate := NewVerificationGate(tmpDir, config.BuildConfig{}, config.TestConfig{Mode: config.TestModeReport}, analysis)

	result, err := gate.VerifyWithOverride(context.Background(), &TaskGateOverride{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Report mode should pass even with test failures
	if result.Status != GateStatusPassed {
		t.Errorf("expected status %s, got %s: %s", GateStatusPassed, result.Status, result.Reason)
	}
}

// TestVerificationGate_Greenfield tests behavior with greenfield project.
func TestVerificationGate_Greenfield(t *testing.T) {
	tmpDir := t.TempDir()

	analysis := &ProjectAnalysis{
		ProjectType:  "unknown",
		IsGreenfield: true, // Greenfield project
		Build: BuildAnalysis{
			Ready:  false,
			Reason: "no buildable code yet",
		},
		Test: TestAnalysis{
			Ready:        false,
			HasTestFiles: false,
			Reason:       "no test files yet",
		},
	}

	gate := NewVerificationGate(tmpDir, config.BuildConfig{}, config.TestConfig{Mode: config.TestModeGate}, analysis)

	result, err := gate.VerifyWithOverride(context.Background(), &TaskGateOverride{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should skip in greenfield mode
	if result.Status != GateStatusSkipped {
		t.Errorf("expected status %s, got %s: %s", GateStatusSkipped, result.Status, result.Reason)
	}
	if !result.BuildSkipped {
		t.Error("expected build to be skipped for greenfield project")
	}
	if !result.TestSkipped {
		t.Error("expected tests to be skipped for greenfield project")
	}
}

// TestVerificationGate_TaskOverride_TestsNotRequired tests task-level test skip.
func TestVerificationGate_TaskOverride_TestsNotRequired(t *testing.T) {
	tmpDir := t.TempDir()

	analysis := &ProjectAnalysis{
		ProjectType:  "test",
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready:   true,
			Command: stringPtr("exit 0"),
		},
		Test: TestAnalysis{
			Ready:        true,
			Command:      stringPtr("exit 1"), // Would fail
			HasTestFiles: true,
		},
	}

	gate := NewVerificationGate(tmpDir, config.BuildConfig{}, config.TestConfig{Mode: config.TestModeGate}, analysis)

	// Create task with tests not required
	tk := task.NewTask("INIT-001", "Setup", "Initialize project\nTests: Not required")
	result, err := gate.Verify(context.Background(), tk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != GateStatusSkippedByTask {
		t.Errorf("expected status %s, got %s: %s", GateStatusSkippedByTask, result.Status, result.Reason)
	}
	if !result.TestSkipped {
		t.Error("expected tests to be skipped")
	}
	if result.TestSkipReason != "tests not required for this task" {
		t.Errorf("unexpected skip reason: %s", result.TestSkipReason)
	}
}

// TestVerificationGate_TaskOverride_BuildNotRequired tests task-level build skip.
func TestVerificationGate_TaskOverride_BuildNotRequired(t *testing.T) {
	tmpDir := t.TempDir()

	analysis := &ProjectAnalysis{
		ProjectType:  "test",
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready:   true,
			Command: stringPtr("exit 1"), // Would fail
		},
		Test: TestAnalysis{
			Ready:        true,
			Command:      stringPtr("exit 0"),
			HasTestFiles: true,
		},
	}

	gate := NewVerificationGate(tmpDir, config.BuildConfig{}, config.TestConfig{Mode: config.TestModeGate}, analysis)

	// Create task with build not required
	tk := task.NewTask("DOC-001", "Documentation", "Update docs\nBuild: Not required")
	result, err := gate.Verify(context.Background(), tk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != GateStatusSkippedByTask {
		t.Errorf("expected status %s, got %s: %s", GateStatusSkippedByTask, result.Status, result.Reason)
	}
	if !result.BuildSkipped {
		t.Error("expected build to be skipped")
	}
}

// TestVerificationGate_TDDMode tests TDD mode with baseline capture.
func TestVerificationGate_TDDMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .ralph directory for baseline
	if err := os.MkdirAll(filepath.Join(tmpDir, ".ralph"), 0755); err != nil {
		t.Fatalf("failed to create .ralph dir: %v", err)
	}

	analysis := &ProjectAnalysis{
		ProjectType:  "test",
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready:   true,
			Command: stringPtr("exit 0"),
		},
		Test: TestAnalysis{
			Ready:        true,
			Command:      stringPtr("exit 0"),
			HasTestFiles: true,
		},
	}

	gate := NewVerificationGate(tmpDir, config.BuildConfig{}, config.TestConfig{
		Mode:          config.TestModeTDD,
		BaselineFile:  ".ralph/test_baseline.json",
		BaselineScope: config.BaselineScopeGlobal,
	}, analysis)

	result, err := gate.VerifyWithOverride(context.Background(), &TaskGateOverride{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != GateStatusPassed {
		t.Errorf("expected status %s, got %s: %s", GateStatusPassed, result.Status, result.Reason)
	}
	if result.TDDResult == nil {
		t.Error("expected TDDResult to be populated in TDD mode")
	}
	// First run should capture baseline
	if result.TDDResult != nil && !result.TDDResult.BaselineCaptured {
		t.Error("expected baseline to be captured on first TDD run")
	}
}

// stringPtr returns a pointer to a string.
func stringPtr(s string) *string {
	return &s
}

