package build

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dbmrq/ralph/internal/config"
)

func TestTDDManager_BaselinePath(t *testing.T) {
	tests := []struct {
		name       string
		projectDir string
		config     config.TestConfig
		want       string
	}{
		{
			name:       "default baseline file",
			projectDir: "/project",
			config:     config.TestConfig{},
			want:       "/project/.ralph/test_baseline.json",
		},
		{
			name:       "custom baseline file",
			projectDir: "/project",
			config:     config.TestConfig{BaselineFile: "custom_baseline.json"},
			want:       "/project/custom_baseline.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewTDDManager(tt.projectDir, tt.config, nil, "session-1")
			got := m.baselinePath()
			if got != tt.want {
				t.Errorf("baselinePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTDDManager_LoadSaveBaseline(t *testing.T) {
	tmpDir := t.TempDir()

	m := NewTDDManager(tmpDir, config.TestConfig{
		BaselineFile: ".ralph/test_baseline.json",
	}, nil, "session-1")

	// Initially no baseline exists
	baseline, err := m.LoadBaseline()
	if err != nil {
		t.Fatalf("LoadBaseline() error = %v", err)
	}
	if baseline != nil {
		t.Error("expected nil baseline for empty directory")
	}

	// Save a baseline
	now := time.Now()
	testBaseline := &TestBaseline{
		CapturedAt: now,
		Scope:      config.BaselineScopeGlobal,
		Passing:    []string{"TestA", "TestB"},
		Failing:    []string{"TestC"},
		SessionID:  "session-1",
	}

	if err := m.SaveBaseline(testBaseline); err != nil {
		t.Fatalf("SaveBaseline() error = %v", err)
	}

	// Load it back
	loaded, err := m.LoadBaseline()
	if err != nil {
		t.Fatalf("LoadBaseline() after save error = %v", err)
	}
	if loaded == nil {
		t.Fatal("expected baseline after save")
	}
	if len(loaded.Passing) != 2 {
		t.Errorf("Passing = %v, want 2 items", loaded.Passing)
	}
	if len(loaded.Failing) != 1 {
		t.Errorf("Failing = %v, want 1 item", loaded.Failing)
	}
	if loaded.SessionID != "session-1" {
		t.Errorf("SessionID = %q, want %q", loaded.SessionID, "session-1")
	}
}

func TestTDDManager_CaptureBaseline(t *testing.T) {
	m := NewTDDManager("/project", config.TestConfig{
		BaselineScope: config.BaselineScopeSession,
	}, nil, "test-session")

	result := &TestResult{
		Success:     true,
		PassedTests: 5,
	}

	baseline := m.CaptureBaseline(result)

	if baseline.Scope != config.BaselineScopeSession {
		t.Errorf("Scope = %v, want %v", baseline.Scope, config.BaselineScopeSession)
	}
	if baseline.SessionID != "test-session" {
		t.Errorf("SessionID = %q, want %q", baseline.SessionID, "test-session")
	}
	if baseline.BootstrapCompletedAt == nil {
		t.Error("expected BootstrapCompletedAt to be set")
	}
}

func TestTDDManager_ShouldCaptureNewBaseline(t *testing.T) {
	tests := []struct {
		name     string
		scope    config.BaselineScope
		existing *TestBaseline
		session  string
		want     bool
	}{
		{
			name:     "no existing baseline",
			scope:    config.BaselineScopeGlobal,
			existing: nil,
			session:  "s1",
			want:     true,
		},
		{
			name:  "global scope with existing baseline",
			scope: config.BaselineScopeGlobal,
			existing: &TestBaseline{
				Scope:     config.BaselineScopeGlobal,
				SessionID: "s1",
			},
			session: "s1",
			want:    false,
		},
		{
			name:  "session scope with same session",
			scope: config.BaselineScopeSession,
			existing: &TestBaseline{
				Scope:     config.BaselineScopeSession,
				SessionID: "s1",
			},
			session: "s1",
			want:    false,
		},
		{
			name:  "session scope with different session",
			scope: config.BaselineScopeSession,
			existing: &TestBaseline{
				Scope:     config.BaselineScopeSession,
				SessionID: "s1",
			},
			session: "s2",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewTDDManager("/project", config.TestConfig{
				BaselineScope: tt.scope,
			}, nil, tt.session)

			got := m.ShouldCaptureNewBaseline(tt.existing)
			if got != tt.want {
				t.Errorf("ShouldCaptureNewBaseline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTDDManager_ShouldCaptureNewBaseline_TaskScope(t *testing.T) {
	// Task scope should always capture new baseline
	m := NewTDDManager("/project", config.TestConfig{
		BaselineScope: config.BaselineScopeTask,
	}, nil, "s1")

	existing := &TestBaseline{
		Scope:     config.BaselineScopeTask,
		SessionID: "s1",
	}

	if !m.ShouldCaptureNewBaseline(existing) {
		t.Error("task scope should always capture new baseline")
	}
}

func TestTDDManager_Evaluate_Bootstrap(t *testing.T) {
	tests := []struct {
		name     string
		analysis *ProjectAnalysis
		wantSkip bool
		wantMsg  string
	}{
		{
			name: "greenfield project",
			analysis: &ProjectAnalysis{
				IsGreenfield: true,
			},
			wantSkip: true,
			wantMsg:  "greenfield project (no tests yet)",
		},
		{
			name: "no test files",
			analysis: &ProjectAnalysis{
				IsGreenfield: false,
				Test: TestAnalysis{
					Ready:        true,
					HasTestFiles: false,
				},
			},
			wantSkip: true,
			wantMsg:  "no test files found",
		},
		{
			name: "tests not ready",
			analysis: &ProjectAnalysis{
				IsGreenfield: false,
				Test: TestAnalysis{
					Ready:        false,
					HasTestFiles: true,
					Reason:       "dependencies not installed",
				},
			},
			wantSkip: true,
			wantMsg:  "dependencies not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewTDDManager("/project", config.TestConfig{}, tt.analysis, "s1")

			result, err := m.Evaluate(&TestResult{})
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if !result.Skipped {
				t.Error("expected result to be skipped")
			}
			if !result.Passed {
				t.Error("skipped result should pass")
			}
			if result.SkipReason != tt.wantMsg {
				t.Errorf("SkipReason = %q, want %q", result.SkipReason, tt.wantMsg)
			}
		})
	}
}

func TestTDDManager_Evaluate_TestResultSkipped(t *testing.T) {
	analysis := &ProjectAnalysis{
		IsGreenfield: false,
		Test: TestAnalysis{
			Ready:        true,
			HasTestFiles: true,
		},
	}
	m := NewTDDManager("/project", config.TestConfig{}, analysis, "s1")

	result, err := m.Evaluate(&TestResult{
		Skipped:    true,
		SkipReason: "test command not available",
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !result.Skipped {
		t.Error("expected result to be skipped")
	}
	if !result.Passed {
		t.Error("skipped result should pass")
	}
}

func TestTDDManager_Evaluate_CapturesBaselineOnFirstRun(t *testing.T) {
	tmpDir := t.TempDir()
	analysis := &ProjectAnalysis{
		IsGreenfield: false,
		Test: TestAnalysis{
			Ready:        true,
			HasTestFiles: true,
		},
	}
	m := NewTDDManager(tmpDir, config.TestConfig{
		BaselineFile:  ".ralph/test_baseline.json",
		BaselineScope: config.BaselineScopeGlobal,
	}, analysis, "s1")

	testResult := &TestResult{
		Success:     true,
		PassedTests: 3,
	}

	result, err := m.Evaluate(testResult)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !result.Passed {
		t.Error("expected result to pass")
	}
	if !result.BaselineCaptured {
		t.Error("expected baseline to be captured")
	}

	// Verify baseline was saved
	baselinePath := filepath.Join(tmpDir, ".ralph", "test_baseline.json")
	if _, err := os.Stat(baselinePath); os.IsNotExist(err) {
		t.Error("baseline file was not created")
	}
}

func TestTDDManager_Evaluate_DetectsRegressions(t *testing.T) {
	tmpDir := t.TempDir()
	analysis := &ProjectAnalysis{
		IsGreenfield: false,
		Test: TestAnalysis{
			Ready:        true,
			HasTestFiles: true,
		},
	}
	m := NewTDDManager(tmpDir, config.TestConfig{
		BaselineFile:  ".ralph/test_baseline.json",
		BaselineScope: config.BaselineScopeGlobal,
	}, analysis, "s1")

	// Save a baseline with some passing tests
	baseline := &TestBaseline{
		CapturedAt: time.Now(),
		Scope:      config.BaselineScopeGlobal,
		Passing:    []string{"TestA", "TestB", "TestC"},
		Failing:    []string{"TestD"},
	}
	if err := m.SaveBaseline(baseline); err != nil {
		t.Fatalf("SaveBaseline() error = %v", err)
	}

	// Now run with TestB failing (regression)
	testResult := &TestResult{
		Success:     false,
		PassedTests: 2,
		FailedTests: 2,
		Failures: []TestFailure{
			{TestName: "TestB", Message: "assertion failed"},
			{TestName: "TestD", Message: "still failing"},
		},
	}

	result, err := m.Evaluate(testResult)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Passed {
		t.Error("expected result to fail due to regression")
	}
	if len(result.Regressions) == 0 {
		t.Error("expected regressions to be detected")
	}
}

func TestTDDManager_Evaluate_NoRegressions(t *testing.T) {
	tmpDir := t.TempDir()
	analysis := &ProjectAnalysis{
		IsGreenfield: false,
		Test: TestAnalysis{
			Ready:        true,
			HasTestFiles: true,
		},
	}
	m := NewTDDManager(tmpDir, config.TestConfig{
		BaselineFile:  ".ralph/test_baseline.json",
		BaselineScope: config.BaselineScopeGlobal,
	}, analysis, "s1")

	// Save a baseline with placeholder test names (matching what we capture)
	// This simulates a baseline captured with the same placeholder system
	baseline := &TestBaseline{
		CapturedAt: time.Now(),
		Scope:      config.BaselineScopeGlobal,
		Passing:    []string{"_all_tests_passed_"},
		Failing:    []string{},
	}
	if err := m.SaveBaseline(baseline); err != nil {
		t.Fatalf("SaveBaseline() error = %v", err)
	}

	// All tests still pass - using same placeholder
	testResult := &TestResult{
		Success:     true,
		PassedTests: 0, // Will use _all_tests_passed_ placeholder
	}

	result, err := m.Evaluate(testResult)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !result.Passed {
		t.Error("expected result to pass with no regressions")
	}
}

func TestFindRegressions(t *testing.T) {
	tests := []struct {
		name            string
		baselinePassing []string
		currentPassing  []string
		wantCount       int
	}{
		{
			name:            "no regressions - same tests",
			baselinePassing: []string{"TestA", "TestB"},
			currentPassing:  []string{"TestA", "TestB"},
			wantCount:       0,
		},
		{
			name:            "one regression",
			baselinePassing: []string{"TestA", "TestB"},
			currentPassing:  []string{"TestA"},
			wantCount:       1,
		},
		{
			name:            "all regressed",
			baselinePassing: []string{"TestA", "TestB"},
			currentPassing:  []string{},
			wantCount:       2,
		},
		{
			name:            "new tests added - not regression",
			baselinePassing: []string{"TestA"},
			currentPassing:  []string{"TestA", "TestB"},
			wantCount:       0,
		},
		{
			name:            "empty baseline",
			baselinePassing: []string{},
			currentPassing:  []string{"TestA"},
			wantCount:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regressions := findRegressions(tt.baselinePassing, tt.currentPassing)
			if len(regressions) != tt.wantCount {
				t.Errorf("findRegressions() = %d regressions, want %d", len(regressions), tt.wantCount)
			}
		})
	}
}

func TestFindNewlyPassing(t *testing.T) {
	tests := []struct {
		name            string
		baselineFailing []string
		currentPassing  []string
		wantCount       int
	}{
		{
			name:            "one newly passing",
			baselineFailing: []string{"TestA", "TestB"},
			currentPassing:  []string{"TestA"},
			wantCount:       1,
		},
		{
			name:            "all newly passing",
			baselineFailing: []string{"TestA", "TestB"},
			currentPassing:  []string{"TestA", "TestB"},
			wantCount:       2,
		},
		{
			name:            "none newly passing",
			baselineFailing: []string{"TestA", "TestB"},
			currentPassing:  []string{},
			wantCount:       0,
		},
		{
			name:            "empty baseline failing",
			baselineFailing: []string{},
			currentPassing:  []string{"TestA"},
			wantCount:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newlyPassing := findNewlyPassing(tt.baselineFailing, tt.currentPassing)
			if len(newlyPassing) != tt.wantCount {
				t.Errorf("findNewlyPassing() = %d, want %d", len(newlyPassing), tt.wantCount)
			}
		})
	}
}

func TestExtractPassingTestNames(t *testing.T) {
	tests := []struct {
		name   string
		result *TestResult
		want   int // count of names
	}{
		{
			name:   "nil result",
			result: nil,
			want:   0,
		},
		{
			name: "success with count",
			result: &TestResult{
				Success:     true,
				PassedTests: 5,
			},
			want: 5,
		},
		{
			name: "success without count",
			result: &TestResult{
				Success: true,
			},
			want: 1, // placeholder
		},
		{
			name: "failure with passed count",
			result: &TestResult{
				Success:     false,
				PassedTests: 3,
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := extractPassingTestNames(tt.result)
			if len(names) != tt.want {
				t.Errorf("extractPassingTestNames() = %d names, want %d", len(names), tt.want)
			}
		})
	}
}

func TestExtractFailingTestNames(t *testing.T) {
	tests := []struct {
		name   string
		result *TestResult
		want   int // count of names
	}{
		{
			name:   "nil result",
			result: nil,
			want:   0,
		},
		{
			name: "with parsed failures",
			result: &TestResult{
				Success: false,
				Failures: []TestFailure{
					{TestName: "TestA", Message: "failed"},
					{TestName: "TestB", Message: "failed"},
				},
			},
			want: 2,
		},
		{
			name: "failure with count only",
			result: &TestResult{
				Success:     false,
				FailedTests: 3,
			},
			want: 3, // placeholder names
		},
		{
			name: "failure without details",
			result: &TestResult{
				Success: false,
			},
			want: 1, // placeholder
		},
		{
			name: "success result",
			result: &TestResult{
				Success: true,
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := extractFailingTestNames(tt.result)
			if len(names) != tt.want {
				t.Errorf("extractFailingTestNames() = %d names, want %d", len(names), tt.want)
			}
		})
	}
}

func TestTDDResult_Message(t *testing.T) {
	// Test that messages are set appropriately
	tmpDir := t.TempDir()
	analysis := &ProjectAnalysis{
		IsGreenfield: false,
		Test: TestAnalysis{
			Ready:        true,
			HasTestFiles: true,
		},
	}
	m := NewTDDManager(tmpDir, config.TestConfig{
		BaselineFile:  ".ralph/test_baseline.json",
		BaselineScope: config.BaselineScopeGlobal,
	}, analysis, "s1")

	// First run - baseline captured
	result, err := m.Evaluate(&TestResult{Success: true})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Message == "" {
		t.Error("expected message to be set")
	}
	if !result.BaselineCaptured {
		t.Error("expected baseline to be captured on first run")
	}
}
