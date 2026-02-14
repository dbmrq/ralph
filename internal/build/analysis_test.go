package build

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dbmrq/ralph/internal/agent"
)

// mockAgent is a test double for agent.Agent.
type mockAgent struct {
	name         string
	runResult    agent.Result
	runError     error
	runCallCount int
	lastPrompt   string
	lastOpts     agent.RunOptions
}

func (m *mockAgent) Name() string                       { return m.name }
func (m *mockAgent) Description() string                { return "Mock agent for testing" }
func (m *mockAgent) IsAvailable() bool                  { return true }
func (m *mockAgent) CheckAuth() error                   { return nil }
func (m *mockAgent) ListModels() ([]agent.Model, error) { return nil, nil }
func (m *mockAgent) GetDefaultModel() agent.Model       { return agent.Model{ID: "mock-model"} }
func (m *mockAgent) GetSessionID() string               { return "" }
func (m *mockAgent) Continue(ctx context.Context, sessionID string, prompt string, opts agent.RunOptions) (agent.Result, error) {
	return m.Run(ctx, prompt, opts)
}

func (m *mockAgent) Run(ctx context.Context, prompt string, opts agent.RunOptions) (agent.Result, error) {
	m.runCallCount++
	m.lastPrompt = prompt
	m.lastOpts = opts
	return m.runResult, m.runError
}

// Test JSON parsing with valid input
func TestParseAnalysisOutput_ValidJSON(t *testing.T) {
	input := `{
		"project_type": "go",
		"languages": ["go"],
		"is_greenfield": false,
		"is_monorepo": false,
		"build": {
			"ready": true,
			"command": "go build ./...",
			"reason": "go.mod found"
		},
		"test": {
			"ready": true,
			"command": "go test ./...",
			"has_test_files": true,
			"reason": "test files found"
		},
		"lint": {
			"command": "golangci-lint run ./...",
			"available": true
		},
		"dependencies": {
			"manager": "go mod",
			"installed": true
		},
		"task_list": {
			"detected": true,
			"path": "TASKS.md",
			"format": "markdown",
			"task_count": 5
		},
		"project_context": "A Go project"
	}`

	analysis, err := parseAnalysisOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if analysis.ProjectType != "go" {
		t.Errorf("expected project_type 'go', got %q", analysis.ProjectType)
	}
	if !analysis.Build.Ready {
		t.Error("expected Build.Ready to be true")
	}
	if analysis.Build.Command == nil || *analysis.Build.Command != "go build ./..." {
		t.Error("expected Build.Command to be 'go build ./...'")
	}
	if !analysis.TaskList.Detected {
		t.Error("expected TaskList.Detected to be true")
	}
	if analysis.TaskList.TaskCount != 5 {
		t.Errorf("expected TaskList.TaskCount 5, got %d", analysis.TaskList.TaskCount)
	}
}

// Test JSON parsing with wrapped JSON (markdown code block)
func TestParseAnalysisOutput_WrappedJSON(t *testing.T) {
	input := "Here is the analysis:\n```json\n" + `{
		"project_type": "node",
		"languages": ["typescript"],
		"is_greenfield": true,
		"is_monorepo": false,
		"build": {"ready": false, "command": null, "reason": "no src"},
		"test": {"ready": false, "command": null, "has_test_files": false, "reason": "no tests"},
		"lint": {"command": null, "available": false},
		"dependencies": {"manager": "npm", "installed": false},
		"task_list": {"detected": false, "path": "", "format": "", "task_count": 0},
		"project_context": "Empty Node project"
	}` + "\n```\n"

	analysis, err := parseAnalysisOutput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if analysis.ProjectType != "node" {
		t.Errorf("expected project_type 'node', got %q", analysis.ProjectType)
	}
	if !analysis.IsGreenfield {
		t.Error("expected IsGreenfield to be true")
	}
}

// Test JSON parsing with no JSON
func TestParseAnalysisOutput_NoJSON(t *testing.T) {
	input := "This is just some text with no JSON"
	_, err := parseAnalysisOutput(input)
	if err == nil {
		t.Error("expected error for input with no JSON")
	}
}

// Test JSON parsing with invalid JSON
func TestParseAnalysisOutput_InvalidJSON(t *testing.T) {
	input := `{"project_type": "go", "invalid": }`
	_, err := parseAnalysisOutput(input)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// Test extractJSON helper
func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple object", `{"key": "value"}`, `{"key": "value"}`},
		{"nested object", `{"outer": {"inner": "value"}}`, `{"outer": {"inner": "value"}}`},
		{"with prefix", `prefix {"key": "value"}`, `{"key": "value"}`},
		{"with suffix", `{"key": "value"} suffix`, `{"key": "value"}`},
		{"with both", `prefix {"key": "value"} suffix`, `{"key": "value"}`},
		{"empty", "", ""},
		{"no json", "just text", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractJSON(tc.input)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// Test ProjectAnalyzer.Analyze with successful response
func TestProjectAnalyzer_Analyze_Success(t *testing.T) {
	mockAg := &mockAgent{
		name: "test-agent",
		runResult: agent.Result{
			Output: `{"project_type": "go", "languages": ["go"], "is_greenfield": false, "is_monorepo": false,
				"build": {"ready": true, "command": "go build", "reason": "ok"},
				"test": {"ready": true, "command": "go test", "has_test_files": true, "reason": "ok"},
				"lint": {"command": null, "available": false},
				"dependencies": {"manager": "go mod", "installed": true},
				"task_list": {"detected": false, "path": "", "format": "", "task_count": 0},
				"project_context": "Go project"}`,
			ExitCode: 0,
		},
	}

	pa := NewProjectAnalyzer("/tmp/test-project", mockAg)

	var progressMessages []string
	pa.OnProgress = func(status string) {
		progressMessages = append(progressMessages, status)
	}

	analysis, err := pa.Analyze(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockAg.runCallCount != 1 {
		t.Errorf("expected 1 agent call, got %d", mockAg.runCallCount)
	}

	if analysis.ProjectType != "go" {
		t.Errorf("expected project_type 'go', got %q", analysis.ProjectType)
	}

	if len(progressMessages) < 3 {
		t.Errorf("expected at least 3 progress messages, got %d", len(progressMessages))
	}
}

// Test ProjectAnalyzer.AnalyzeWithFallback when analysis fails
func TestProjectAnalyzer_AnalyzeWithFallback(t *testing.T) {
	mockAg := &mockAgent{
		name: "test-agent",
		runResult: agent.Result{
			Output:   "error: something went wrong",
			ExitCode: 1,
			Error:    "failed",
		},
	}

	pa := NewProjectAnalyzer("/tmp/test-project", mockAg)

	analysis, err := pa.AnalyzeWithFallback(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return fallback analysis
	if analysis.ProjectType != "unknown" {
		t.Errorf("expected fallback project_type 'unknown', got %q", analysis.ProjectType)
	}
	if analysis.Build.Ready {
		t.Error("expected fallback Build.Ready to be false")
	}
}

// Test cache operations
func TestProjectAnalyzer_Cache(t *testing.T) {
	tmpDir := t.TempDir()
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatalf("failed to create .ralph dir: %v", err)
	}

	mockAg := &mockAgent{name: "test-agent"}
	pa := NewProjectAnalyzer(tmpDir, mockAg)

	// Initially no cache
	cached, err := pa.LoadCached()
	if err != nil {
		t.Fatalf("unexpected error loading empty cache: %v", err)
	}
	if cached != nil {
		t.Error("expected nil cache for non-existent file")
	}

	// Save analysis to cache
	buildCmd := "go build ./..."
	analysis := &ProjectAnalysis{
		ProjectType: "go",
		Languages:   []string{"go"},
		Build:       BuildAnalysis{Ready: true, Command: &buildCmd, Reason: "ok"},
	}

	if err := pa.SaveCache(analysis); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	// Load cache
	cached, err = pa.LoadCached()
	if err != nil {
		t.Fatalf("failed to load cache: %v", err)
	}
	if cached == nil {
		t.Fatal("expected cached analysis, got nil")
	}
	if cached.Analysis.ProjectType != "go" {
		t.Errorf("expected cached project_type 'go', got %q", cached.Analysis.ProjectType)
	}
	if cached.AgentName != "test-agent" {
		t.Errorf("expected agent name 'test-agent', got %q", cached.AgentName)
	}
	if !cached.IsCacheFresh() {
		t.Error("expected cache to be fresh")
	}
}

// Test cache freshness
func TestCachedAnalysis_IsCacheFresh(t *testing.T) {
	fresh := &CachedAnalysis{CachedAt: time.Now()}
	if !fresh.IsCacheFresh() {
		t.Error("expected recent cache to be fresh")
	}

	stale := &CachedAnalysis{CachedAt: time.Now().Add(-25 * time.Hour)}
	if stale.IsCacheFresh() {
		t.Error("expected old cache to be stale")
	}
}

// Test fallback analysis structure
func TestProjectAnalyzer_FallbackAnalysis(t *testing.T) {
	pa := &ProjectAnalyzer{}
	fallback := pa.fallbackAnalysis()

	if fallback.ProjectType != "unknown" {
		t.Errorf("expected project_type 'unknown', got %q", fallback.ProjectType)
	}
	if fallback.Build.Ready {
		t.Error("expected Build.Ready to be false")
	}
	if fallback.Test.Ready {
		t.Error("expected Test.Ready to be false")
	}
	if !fallback.IsGreenfield {
		t.Error("expected IsGreenfield to be true for fallback")
	}
}

// Test loading corrupted cache
func TestProjectAnalyzer_LoadCached_Corrupted(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, CacheFile)

	// Create directory structure
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Write corrupted JSON
	if err := os.WriteFile(cachePath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	mockAg := &mockAgent{name: "test-agent"}
	pa := NewProjectAnalyzer(tmpDir, mockAg)

	_, err := pa.LoadCached()
	if err == nil {
		t.Error("expected error for corrupted cache")
	}
}

// Test that analysis prompt contains key instructions
func TestProjectAnalyzer_AnalysisPrompt(t *testing.T) {
	pa := &ProjectAnalyzer{}
	prompt := pa.buildAnalysisPrompt()

	// Verify key elements are in the prompt
	mustContain := []string{
		"project_type",
		"build",
		"test",
		"is_greenfield",
		"task_list",
		"JSON",
	}

	for _, keyword := range mustContain {
		if !containsString(prompt, keyword) {
			t.Errorf("expected prompt to contain %q", keyword)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || containsString(s[1:], substr)))
}
