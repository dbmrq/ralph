package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/build"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	name      string
	runResult agent.Result
	runErr    error
}

func (m *mockAgent) Name() string                        { return m.name }
func (m *mockAgent) Description() string                 { return "Mock agent for testing" }
func (m *mockAgent) IsAvailable() bool                   { return true }
func (m *mockAgent) CheckAuth() error                    { return nil }
func (m *mockAgent) ListModels() ([]agent.Model, error)  { return nil, nil }
func (m *mockAgent) GetDefaultModel() agent.Model        { return agent.Model{ID: "mock-model"} }
func (m *mockAgent) GetSessionID() string                { return "test-session" }

func (m *mockAgent) Run(ctx context.Context, prompt string, opts agent.RunOptions) (agent.Result, error) {
	return m.runResult, m.runErr
}

func (m *mockAgent) Continue(ctx context.Context, sessionID, prompt string, opts agent.RunOptions) (agent.Result, error) {
	return m.runResult, m.runErr
}

func newMockAgent() *mockAgent {
	return &mockAgent{
		name: "mock",
		runResult: agent.Result{
			ExitCode: 0,
		},
	}
}

func TestNeedsSetup(t *testing.T) {
	tests := []struct {
		name       string
		createDir  bool
		wantResult bool
	}{
		{
			name:       "needs setup when .ralph does not exist",
			createDir:  false,
			wantResult: true,
		},
		{
			name:       "does not need setup when .ralph exists",
			createDir:  true,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.createDir {
				if err := os.MkdirAll(filepath.Join(tmpDir, ".ralph"), 0755); err != nil {
					t.Fatalf("failed to create .ralph dir: %v", err)
				}
			}

			got := NeedsSetup(tmpDir)
			if got != tt.wantResult {
				t.Errorf("NeedsSetup() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestSetup_CreateRalphDir(t *testing.T) {
	tmpDir := t.TempDir()
	ag := newMockAgent()
	setup := NewSetup(tmpDir, ag)

	if err := setup.CreateRalphDir(); err != nil {
		t.Fatalf("CreateRalphDir() error = %v", err)
	}

	// Check that .ralph directory exists
	ralphDir := filepath.Join(tmpDir, ".ralph")
	if _, err := os.Stat(ralphDir); os.IsNotExist(err) {
		t.Error(".ralph directory was not created")
	}

	// Check that subdirectories exist
	subdirs := []string{"sessions", "logs"}
	for _, subdir := range subdirs {
		path := filepath.Join(ralphDir, subdir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("%s directory was not created", subdir)
		}
	}
}

func TestSetup_BuildConfigFromAnalysis(t *testing.T) {
	tmpDir := t.TempDir()
	ag := newMockAgent()
	setup := NewSetup(tmpDir, ag)
	setup.Model = "test-model"

	buildCmd := "go build ./..."
	testCmd := "go test ./..."

	analysis := &build.ProjectAnalysis{
		ProjectType: "go-cli",
		Languages:   []string{"go"},
		Build: build.BuildAnalysis{
			Command: &buildCmd,
			Ready:   true,
		},
		Test: build.TestAnalysis{
			Command: &testCmd,
			Ready:   true,
		},
	}

	cfg := setup.BuildConfigFromAnalysis(analysis)

	if cfg.Build.Command != buildCmd {
		t.Errorf("Build.Command = %v, want %v", cfg.Build.Command, buildCmd)
	}

	if cfg.Test.Command != testCmd {
		t.Errorf("Test.Command = %v, want %v", cfg.Test.Command, testCmd)
	}

	if cfg.Agent.Default != "mock" {
		t.Errorf("Agent.Default = %v, want %v", cfg.Agent.Default, "mock")
	}

	if cfg.Agent.Model != "test-model" {
		t.Errorf("Agent.Model = %v, want %v", cfg.Agent.Model, "test-model")
	}
}

func TestSetup_SaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	ag := newMockAgent()
	setup := NewSetup(tmpDir, ag)

	// Create .ralph directory first
	if err := setup.CreateRalphDir(); err != nil {
		t.Fatalf("CreateRalphDir() error = %v", err)
	}

	analysis := &build.ProjectAnalysis{
		ProjectType: "go-cli",
	}
	cfg := setup.BuildConfigFromAnalysis(analysis)

	if err := setup.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Check that config file exists
	configPath := filepath.Join(tmpDir, ".ralph", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config.yaml was not created")
	}
}

