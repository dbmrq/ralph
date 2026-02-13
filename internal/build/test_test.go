package build

import (
	"context"
	"testing"
	"time"

	"github.com/wexinc/ralph/internal/config"
)

func TestTestVerifier_Verify_GreenfieldProject(t *testing.T) {
	verifier := NewTestVerifier("/tmp/test", config.TestConfig{}, &ProjectAnalysis{
		IsGreenfield: true,
		Test: TestAnalysis{
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
		t.Error("expected tests to be skipped for greenfield project")
	}
	if result.SkipReason != "greenfield project (no test files yet)" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}

func TestTestVerifier_Verify_TestsNotReady(t *testing.T) {
	verifier := NewTestVerifier("/tmp/test", config.TestConfig{}, &ProjectAnalysis{
		IsGreenfield: false,
		Test: TestAnalysis{
			Ready:  false,
			Reason: "no test framework detected",
		},
	})

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success when tests not ready")
	}
	if !result.Skipped {
		t.Error("expected tests to be skipped when not ready")
	}
	if result.SkipReason != "no test framework detected" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}

func TestTestVerifier_Verify_NoTestFiles(t *testing.T) {
	verifier := NewTestVerifier("/tmp/test", config.TestConfig{}, &ProjectAnalysis{
		IsGreenfield: false,
		Test: TestAnalysis{
			Ready:        true,
			HasTestFiles: false,
		},
	})

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success when no test files")
	}
	if !result.Skipped {
		t.Error("expected tests to be skipped when no test files")
	}
	if result.SkipReason != "no test files found" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}

func TestTestVerifier_Verify_NoTestCommand(t *testing.T) {
	verifier := NewTestVerifier("/tmp/test", config.TestConfig{}, &ProjectAnalysis{
		IsGreenfield: false,
		Test: TestAnalysis{
			Ready:        true,
			HasTestFiles: true,
			Command:      nil, // No command detected
		},
	})

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success when no test command")
	}
	if !result.Skipped {
		t.Error("expected tests to be skipped when no command")
	}
	if result.SkipReason != "no test command available" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}

func TestTestVerifier_Verify_ConfigOverride(t *testing.T) {
	tmpDir := t.TempDir()

	aiCommand := "go test ./..."
	configCommand := "echo 'using config command'"

	verifier := NewTestVerifier(tmpDir, config.TestConfig{
		Command: configCommand, // This should override AI command
	}, &ProjectAnalysis{
		IsGreenfield: false,
		Test: TestAnalysis{
			Ready:        true,
			HasTestFiles: true,
			Command:      &aiCommand,
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

func TestTestVerifier_Verify_SuccessfulTests(t *testing.T) {
	tmpDir := t.TempDir()
	testCmd := "echo 'all tests passed'"

	verifier := NewTestVerifier(tmpDir, config.TestConfig{
		Command: testCmd,
	}, &ProjectAnalysis{
		IsGreenfield: false,
		Test: TestAnalysis{
			Ready:        true,
			HasTestFiles: true,
		},
	})

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected tests to succeed")
	}
	if result.Skipped {
		t.Error("expected tests to not be skipped")
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got: %d", result.ExitCode)
	}
	if result.Duration == 0 {
		t.Error("expected non-zero duration")
	}
}

func TestTestVerifier_Verify_FailedTests(t *testing.T) {
	tmpDir := t.TempDir()
	testCmd := "sh -c 'echo \"--- FAIL: TestSomething\" && exit 1'"

	verifier := NewTestVerifier(tmpDir, config.TestConfig{
		Command: testCmd,
	}, &ProjectAnalysis{
		IsGreenfield: false,
		Test: TestAnalysis{
			Ready:        true,
			HasTestFiles: true,
		},
	})

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected tests to fail")
	}
	if result.Skipped {
		t.Error("expected tests to not be skipped")
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code")
	}
	if len(result.Failures) == 0 {
		t.Error("expected at least one parsed failure")
	}
}

func TestTestVerifier_Verify_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	testCmd := "sleep 10"

	verifier := NewTestVerifier(tmpDir, config.TestConfig{
		Command: testCmd,
	}, &ProjectAnalysis{
		IsGreenfield: false,
		Test: TestAnalysis{
			Ready:        true,
			HasTestFiles: true,
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := verifier.Verify(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected tests to fail due to timeout")
	}
	if len(result.Failures) == 0 || result.Failures[0].Message != "tests timed out" {
		t.Errorf("expected timeout failure, got: %v", result.Failures)
	}
}

func TestTestVerifier_Verify_NilAnalysis(t *testing.T) {
	tmpDir := t.TempDir()

	verifier := NewTestVerifier(tmpDir, config.TestConfig{
		Command: "echo 'test'",
	}, nil) // No analysis

	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With no analysis and a config command, tests should run
	if result.Skipped {
		t.Error("expected tests to run with config command even without analysis")
	}
	if !result.Success {
		t.Error("expected tests to succeed")
	}
}

func TestParseTestFailures(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []TestFailure
	}{
		{
			name:   "go test FAIL pattern",
			output: "--- FAIL: TestSomething (0.00s)",
			expected: []TestFailure{
				{TestName: "TestSomething"},
			},
		},
		{
			name:   "go test with file location",
			output: "--- FAIL: TestSomething (0.00s)\n    main_test.go:10: expected true, got false",
			expected: []TestFailure{
				{TestName: "TestSomething", File: "main_test.go", Line: 10, Message: "expected true, got false"},
			},
		},
		{
			name:   "generic failure keyword",
			output: "FAIL: something went wrong",
			expected: []TestFailure{
				{Message: "FAIL: something went wrong"},
			},
		},
		{
			name:     "no failures",
			output:   "ok  	github.com/test/pkg	0.001s",
			expected: []TestFailure{},
		},
		{
			name:     "all tests passed message",
			output:   "all tests passed\n0 failures",
			expected: []TestFailure{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failures := parseTestFailures(tt.output)

			if len(failures) != len(tt.expected) {
				t.Fatalf("expected %d failures, got %d: %v", len(tt.expected), len(failures), failures)
			}

			for i, fail := range failures {
				exp := tt.expected[i]
				if fail.TestName != exp.TestName {
					t.Errorf("failure %d: expected test name %q, got %q", i, exp.TestName, fail.TestName)
				}
				if fail.File != exp.File {
					t.Errorf("failure %d: expected file %q, got %q", i, exp.File, fail.File)
				}
				if fail.Line != exp.Line {
					t.Errorf("failure %d: expected line %d, got %d", i, exp.Line, fail.Line)
				}
				if exp.Message != "" && fail.Message != exp.Message {
					t.Errorf("failure %d: expected message %q, got %q", i, exp.Message, fail.Message)
				}
			}
		})
	}
}

func TestTestFailure_String(t *testing.T) {
	tests := []struct {
		name     string
		failure  TestFailure
		expected string
	}{
		{
			name:     "full details",
			failure:  TestFailure{Package: "pkg", TestName: "TestFoo", File: "foo_test.go", Line: 10, Message: "failed"},
			expected: "pkg/TestFoo (foo_test.go:10): failed",
		},
		{
			name:     "test name only",
			failure:  TestFailure{TestName: "TestFoo", Message: "failed"},
			expected: "TestFoo: failed",
		},
		{
			name:     "message only",
			failure:  TestFailure{Message: "failed"},
			expected: "failed",
		},
		{
			name:     "package and test",
			failure:  TestFailure{Package: "pkg", TestName: "TestFoo", Message: "failed"},
			expected: "pkg/TestFoo: failed",
		},
		{
			name:     "with file no line",
			failure:  TestFailure{TestName: "TestFoo", File: "foo_test.go", Message: "failed"},
			expected: "TestFoo (foo_test.go): failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.failure.String(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
