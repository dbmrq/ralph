package build

import (
	"context"
	"testing"
	"time"

	"github.com/wexinc/ralph/internal/config"
)

func TestBuildVerifier_Verify_GreenfieldProject(t *testing.T) {
	verifier := NewBuildVerifier("/tmp/test", config.BuildConfig{}, &ProjectAnalysis{
		IsGreenfield: true,
		Build: BuildAnalysis{
			Ready: false,
		},
	})

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success for greenfield project")
	}
	if !result.Skipped {
		t.Error("expected build to be skipped for greenfield project")
	}
	if result.SkipReason != "greenfield project (no buildable code yet)" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}

func TestBuildVerifier_Verify_BuildNotReady(t *testing.T) {
	verifier := NewBuildVerifier("/tmp/test", config.BuildConfig{}, &ProjectAnalysis{
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready:  false,
			Reason: "no source files found",
		},
	})

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success when build not ready")
	}
	if !result.Skipped {
		t.Error("expected build to be skipped when not ready")
	}
	if result.SkipReason != "no source files found" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}

func TestBuildVerifier_Verify_NoBuildCommand(t *testing.T) {
	verifier := NewBuildVerifier("/tmp/test", config.BuildConfig{}, &ProjectAnalysis{
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready:   true,
			Command: nil, // No command detected
		},
	})

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success when no build command")
	}
	if !result.Skipped {
		t.Error("expected build to be skipped when no command")
	}
	if result.SkipReason != "no build command available" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}

func TestBuildVerifier_Verify_ConfigOverride(t *testing.T) {
	// Create a temp directory for the test
	tmpDir := t.TempDir()

	aiCommand := "go build ./..."
	configCommand := "echo 'using config command'"

	verifier := NewBuildVerifier(tmpDir, config.BuildConfig{
		Command: configCommand, // This should override AI command
	}, &ProjectAnalysis{
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready:   true,
			Command: &aiCommand,
		},
	})

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Command != configCommand {
		t.Errorf("expected config command to override AI command, got: %s", result.Command)
	}
}

func TestBuildVerifier_Verify_SuccessfulBuild(t *testing.T) {
	tmpDir := t.TempDir()
	buildCmd := "echo 'build successful'"

	verifier := NewBuildVerifier(tmpDir, config.BuildConfig{
		Command: buildCmd,
	}, &ProjectAnalysis{
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready: true,
		},
	})

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected build to succeed")
	}
	if result.Skipped {
		t.Error("expected build to not be skipped")
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", result.ExitCode)
	}
	if result.Duration == 0 {
		t.Error("expected non-zero duration")
	}
}

func TestBuildVerifier_Verify_FailedBuild(t *testing.T) {
	tmpDir := t.TempDir()
	buildCmd := "sh -c 'echo \"error: build failed\" && exit 1'"

	verifier := NewBuildVerifier(tmpDir, config.BuildConfig{
		Command: buildCmd,
	}, &ProjectAnalysis{
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready: true,
		},
	})

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected build to fail")
	}
	if result.Skipped {
		t.Error("expected build to not be skipped")
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code")
	}
	if len(result.Errors) == 0 {
		t.Error("expected at least one parsed error")
	}
}

func TestBuildVerifier_Verify_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	buildCmd := "sleep 10"

	verifier := NewBuildVerifier(tmpDir, config.BuildConfig{
		Command: buildCmd,
	}, &ProjectAnalysis{
		IsGreenfield: false,
		Build: BuildAnalysis{
			Ready: true,
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := verifier.Verify(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected build to fail due to timeout")
	}
	if len(result.Errors) == 0 || result.Errors[0].Message != "build timed out" {
		t.Errorf("expected timeout error, got: %v", result.Errors)
	}
}

func TestBuildVerifier_Verify_NilAnalysis(t *testing.T) {
	tmpDir := t.TempDir()

	verifier := NewBuildVerifier(tmpDir, config.BuildConfig{
		Command: "echo 'build'",
	}, nil) // No analysis

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With no analysis and a config command, build should run
	if result.Skipped {
		t.Error("expected build to run with config command even without analysis")
	}
	if !result.Success {
		t.Error("expected build to succeed")
	}
}

func TestParseBuildErrors(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []BuildError
	}{
		{
			name:   "go error format",
			output: "main.go:10:5: undefined: foo",
			expected: []BuildError{
				{File: "main.go", Line: 10, Column: 5, Message: "undefined: foo"},
			},
		},
		{
			name:   "go error without column",
			output: "main.go:10: syntax error",
			expected: []BuildError{
				{File: "main.go", Line: 10, Message: "syntax error"},
			},
		},
		{
			name:   "generic error keyword",
			output: "error: something went wrong",
			expected: []BuildError{
				{Message: "error: something went wrong"},
			},
		},
		{
			name:   "multiple errors",
			output: "main.go:10: error one\nmain.go:20: error two",
			expected: []BuildError{
				{File: "main.go", Line: 10, Message: "error one"},
				{File: "main.go", Line: 20, Message: "error two"},
			},
		},
		{
			name:     "no errors",
			output:   "build successful\nall tests passed",
			expected: []BuildError{},
		},
		{
			name:   "path with directory",
			output: "internal/build/main.go:5:10: undefined reference",
			expected: []BuildError{
				{File: "internal/build/main.go", Line: 5, Column: 10, Message: "undefined reference"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parseBuildErrors(tt.output)

			if len(errors) != len(tt.expected) {
				t.Fatalf("expected %d errors, got %d: %v", len(tt.expected), len(errors), errors)
			}

			for i, err := range errors {
				exp := tt.expected[i]
				if err.File != exp.File {
					t.Errorf("error %d: expected file %q, got %q", i, exp.File, err.File)
				}
				if err.Line != exp.Line {
					t.Errorf("error %d: expected line %d, got %d", i, exp.Line, err.Line)
				}
				if err.Column != exp.Column {
					t.Errorf("error %d: expected column %d, got %d", i, exp.Column, err.Column)
				}
				if err.Message != exp.Message {
					t.Errorf("error %d: expected message %q, got %q", i, exp.Message, err.Message)
				}
			}
		})
	}
}

func TestBuildError_String(t *testing.T) {
	tests := []struct {
		name     string
		err      BuildError
		expected string
	}{
		{
			name:     "full location",
			err:      BuildError{File: "main.go", Line: 10, Column: 5, Message: "error"},
			expected: "main.go:10:5: error",
		},
		{
			name:     "file and line only",
			err:      BuildError{File: "main.go", Line: 10, Message: "error"},
			expected: "main.go:10: error",
		},
		{
			name:     "file only",
			err:      BuildError{File: "main.go", Message: "error"},
			expected: "main.go: error",
		},
		{
			name:     "message only",
			err:      BuildError{Message: "error"},
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.String(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

