// Package build provides build and test verification logic for ralph.
package build

import (
	"context"
	"os"
	"testing"

	"github.com/wexinc/ralph/internal/config"
)

func TestProjectAnalysis_ToBootstrapState(t *testing.T) {
	tests := []struct {
		name     string
		analysis ProjectAnalysis
		want     BootstrapState
	}{
		{
			name: "greenfield project",
			analysis: ProjectAnalysis{
				IsGreenfield: true,
				Build:        BuildAnalysis{Ready: false, Reason: "no source files"},
				Test:         TestAnalysis{Ready: false, Reason: "no test files"},
			},
			want: BootstrapState{
				BuildReady: false,
				TestReady:  false,
				Reason:     "greenfield project (no buildable code yet)",
			},
		},
		{
			name: "build and test ready",
			analysis: ProjectAnalysis{
				IsGreenfield: false,
				Build:        BuildAnalysis{Ready: true, Reason: "go.mod found"},
				Test:         TestAnalysis{Ready: true, Reason: "test files found"},
			},
			want: BootstrapState{
				BuildReady: true,
				TestReady:  true,
				Reason:     "build ready; tests ready",
			},
		},
		{
			name: "build ready but tests not ready",
			analysis: ProjectAnalysis{
				IsGreenfield: false,
				Build:        BuildAnalysis{Ready: true, Reason: ""},
				Test:         TestAnalysis{Ready: false, Reason: "no test files found"},
			},
			want: BootstrapState{
				BuildReady: true,
				TestReady:  false,
				Reason:     "build ready; tests not ready: no test files found",
			},
		},
		{
			name: "build not ready but tests ready",
			analysis: ProjectAnalysis{
				IsGreenfield: false,
				Build:        BuildAnalysis{Ready: false, Reason: "missing dependencies"},
				Test:         TestAnalysis{Ready: true, Reason: ""},
			},
			want: BootstrapState{
				BuildReady: false,
				TestReady:  true,
				Reason:     "build not ready: missing dependencies; tests ready",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.analysis.ToBootstrapState()
			if got.BuildReady != tt.want.BuildReady {
				t.Errorf("BuildReady = %v, want %v", got.BuildReady, tt.want.BuildReady)
			}
			if got.TestReady != tt.want.TestReady {
				t.Errorf("TestReady = %v, want %v", got.TestReady, tt.want.TestReady)
			}
			if got.Reason != tt.want.Reason {
				t.Errorf("Reason = %q, want %q", got.Reason, tt.want.Reason)
			}
		})
	}
}

func TestBootstrapDetector_Detect_Disabled(t *testing.T) {
	detector := NewBootstrapDetector("/tmp/test", config.BuildConfig{
		BootstrapDetection: config.BootstrapDetectionDisabled,
	})

	state, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !state.BuildReady {
		t.Error("expected BuildReady to be true when disabled")
	}
	if !state.TestReady {
		t.Error("expected TestReady to be true when disabled")
	}
	if state.Reason != "bootstrap detection disabled" {
		t.Errorf("unexpected reason: %q", state.Reason)
	}
}

func TestBootstrapDetector_Detect_Auto_NoAnalysis(t *testing.T) {
	detector := NewBootstrapDetector("/tmp/test", config.BuildConfig{
		BootstrapDetection: config.BootstrapDetectionAuto,
	})

	_, err := detector.Detect(context.Background())
	if err == nil {
		t.Error("expected error when no analysis is set in auto mode")
	}

	expectedErr := "bootstrap_detection is 'auto' but no ProjectAnalysis is available"
	if err != nil && err.Error()[:len(expectedErr)] != expectedErr {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBootstrapDetector_Detect_Auto_WithAnalysis(t *testing.T) {
	detector := NewBootstrapDetector("/tmp/test", config.BuildConfig{
		BootstrapDetection: config.BootstrapDetectionAuto,
	})

	analysis := &ProjectAnalysis{
		IsGreenfield: false,
		Build:        BuildAnalysis{Ready: true, Reason: "detected go.mod"},
		Test:         TestAnalysis{Ready: true, Reason: "found test files"},
	}
	detector.SetAnalysis(analysis)

	state, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !state.BuildReady {
		t.Error("expected BuildReady to be true")
	}
	if !state.TestReady {
		t.Error("expected TestReady to be true")
	}
}

func TestBootstrapDetector_Detect_Manual_Success(t *testing.T) {
	// Create a temp dir for the test
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	detector := NewBootstrapDetector(tmpDir, config.BuildConfig{
		BootstrapDetection: config.BootstrapDetectionManual,
		BootstrapCheck:     "exit 0", // Exit 0 = still bootstrapping
	})

	state, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.BuildReady {
		t.Error("expected BuildReady to be false when command exits 0")
	}
	if state.TestReady {
		t.Error("expected TestReady to be false when command exits 0")
	}
}

func TestBootstrapDetector_Detect_Manual_Ready(t *testing.T) {
	// Create a temp dir for the test
	tmpDir, err := os.MkdirTemp("", "ralph-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	detector := NewBootstrapDetector(tmpDir, config.BuildConfig{
		BootstrapDetection: config.BootstrapDetectionManual,
		BootstrapCheck:     "exit 1", // Non-zero = project ready
	})

	state, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !state.BuildReady {
		t.Error("expected BuildReady to be true when command exits non-zero")
	}
	if !state.TestReady {
		t.Error("expected TestReady to be true when command exits non-zero")
	}
}

func TestBootstrapDetector_Detect_Manual_NoCommand(t *testing.T) {
	detector := NewBootstrapDetector("/tmp/test", config.BuildConfig{
		BootstrapDetection: config.BootstrapDetectionManual,
		BootstrapCheck:     "", // No command set
	})

	_, err := detector.Detect(context.Background())
	if err == nil {
		t.Error("expected error when no bootstrap_check command is set")
	}
}

func TestBootstrapDetector_Detect_UnknownMode(t *testing.T) {
	detector := NewBootstrapDetector("/tmp/test", config.BuildConfig{
		BootstrapDetection: "invalid_mode",
	})

	_, err := detector.Detect(context.Background())
	if err == nil {
		t.Error("expected error for unknown bootstrap_detection mode")
	}
}

func TestBootstrapDetector_Detect_EmptyMode_DefaultsToAuto(t *testing.T) {
	detector := NewBootstrapDetector("/tmp/test", config.BuildConfig{
		BootstrapDetection: "", // Empty should default to auto
	})

	// Without analysis, should fail with the "no analysis" error
	_, err := detector.Detect(context.Background())
	if err == nil {
		t.Error("expected error when no analysis is set for empty (auto) mode")
	}

	expectedErr := "bootstrap_detection is 'auto' but no ProjectAnalysis is available"
	if err != nil && err.Error()[:len(expectedErr)] != expectedErr {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestJoinReasons(t *testing.T) {
	tests := []struct {
		name    string
		reasons []string
		want    string
	}{
		{
			name:    "empty",
			reasons: []string{},
			want:    "",
		},
		{
			name:    "single reason",
			reasons: []string{"only one"},
			want:    "only one",
		},
		{
			name:    "multiple reasons",
			reasons: []string{"first", "second", "third"},
			want:    "first; second; third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinReasons(tt.reasons)
			if got != tt.want {
				t.Errorf("joinReasons() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProjectAnalysis_Types(t *testing.T) {
	// Test that all types can be created and have the expected structure
	cmd := "go build ./..."
	analysis := ProjectAnalysis{
		ProjectType:  "go",
		Languages:    []string{"go", "shell"},
		IsGreenfield: false,
		IsMonorepo:   true,
		Build: BuildAnalysis{
			Ready:   true,
			Command: &cmd,
			Reason:  "detected go.mod",
		},
		Test: TestAnalysis{
			Ready:        true,
			Command:      &cmd,
			HasTestFiles: true,
			Reason:       "found 10 test files",
		},
		Lint: LintAnalysis{
			Command:   &cmd,
			Available: true,
		},
		Dependencies: DependencyAnalysis{
			Manager:   "go mod",
			Installed: true,
		},
		TaskList: TaskListAnalysis{
			Detected:  true,
			Path:      ".ralph/TASKS.md",
			Format:    "markdown",
			TaskCount: 15,
		},
		ProjectContext: "Go CLI application using Cobra",
	}

	if analysis.ProjectType != "go" {
		t.Errorf("ProjectType = %q, want %q", analysis.ProjectType, "go")
	}
	if len(analysis.Languages) != 2 {
		t.Errorf("Languages length = %d, want 2", len(analysis.Languages))
	}
	if analysis.Build.Command == nil || *analysis.Build.Command != cmd {
		t.Error("Build.Command not set correctly")
	}
	if !analysis.TaskList.Detected {
		t.Error("TaskList.Detected should be true")
	}
}
