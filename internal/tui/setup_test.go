// Package tui provides the terminal user interface for ralph.
package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dbmrq/ralph/internal/app"
)

// mockAgent is a minimal agent for testing.
type mockSetupAgent struct {
	name        string
	description string
	available   bool
	authErr     error
}

func (m *mockSetupAgent) Name() string                 { return m.name }
func (m *mockSetupAgent) Description() string          { return m.description }
func (m *mockSetupAgent) IsAvailable() bool            { return m.available }
func (m *mockSetupAgent) CheckAuth() error             { return m.authErr }
func (m *mockSetupAgent) ListModels() ([]interface{}, error) { return nil, nil }
func (m *mockSetupAgent) GetDefaultModel() interface{} { return nil }
func (m *mockSetupAgent) Run(ctx context.Context, prompt string, opts interface{}) (interface{}, error) {
	return nil, nil
}
func (m *mockSetupAgent) Continue(ctx context.Context, sessionID string, prompt string, opts interface{}) (interface{}, error) {
	return nil, nil
}
func (m *mockSetupAgent) GetSessionID() string { return "" }

func TestFormatProjectType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"go", "Go"},
		{"node", "Node.js"},
		{"python", "Python"},
		{"rust", "Rust"},
		{"ruby", "Ruby"},
		{"php", "PHP"},
		{"swift", "Swift"},
		{"xcode", "Xcode/iOS"},
		{"gradle", "Gradle (Java/Kotlin)"},
		{"maven", "Maven (Java)"},
		{"dotnet", ".NET"},
		{"make", "Make"},
		{"cmake", "CMake"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatProjectType(tt.input)
			if result != tt.expected {
				t.Errorf("formatProjectType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatMarkers(t *testing.T) {
	tests := []struct {
		name     string
		markers  []string
		expected string
	}{
		{
			name:     "empty",
			markers:  []string{},
			expected: "",
		},
		{
			name:     "only git",
			markers:  []string{".git"},
			expected: "",
		},
		{
			name:     "only ralph",
			markers:  []string{".ralph"},
			expected: "",
		},
		{
			name:     "git and ralph only",
			markers:  []string{".git", ".ralph"},
			expected: "",
		},
		{
			name:     "single marker",
			markers:  []string{"go.mod"},
			expected: "go.mod",
		},
		{
			name:     "multiple markers",
			markers:  []string{"go.mod", "go.sum"},
			expected: "go.mod, go.sum",
		},
		{
			name:     "filters git and ralph",
			markers:  []string{".git", "go.mod", ".ralph", "go.sum"},
			expected: "go.mod, go.sum",
		},
		{
			name:     "more than 4 markers",
			markers:  []string{"a", "b", "c", "d", "e"},
			expected: "a, b, c, d, ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMarkers(tt.markers)
			if result != tt.expected {
				t.Errorf("formatMarkers(%v) = %q, want %q", tt.markers, result, tt.expected)
			}
		})
	}
}

func TestWelcomeInfo(t *testing.T) {
	info := &WelcomeInfo{
		ProjectName:   "test-project",
		ProjectType:   "go",
		ProjectPath:   "/path/to/project",
		IsGitRepo:     true,
		Markers:       []string{".git", "go.mod", "go.sum"},
		SelectedAgent: "auggie",
		Agents: []AgentStatus{
			{
				Name:        "auggie",
				Description: "Augment AI agent",
				Available:   true,
				AuthError:   nil,
			},
		},
	}

	if info.ProjectName != "test-project" {
		t.Errorf("expected ProjectName 'test-project', got %q", info.ProjectName)
	}
	if info.ProjectType != "go" {
		t.Errorf("expected ProjectType 'go', got %q", info.ProjectType)
	}
	if !info.IsGitRepo {
		t.Error("expected IsGitRepo to be true")
	}
	if len(info.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(info.Agents))
	}
}

func TestNewSetupModelWithWelcomeInfo(t *testing.T) {
	ctx := context.Background()
	setup := app.NewSetup("/tmp/test-project", nil)

	model := NewSetupModel(ctx, setup)

	if model.Phase != PhaseWelcome {
		t.Errorf("expected phase PhaseWelcome, got %d", model.Phase)
	}

	if model.welcomeInfo == nil {
		t.Error("expected welcomeInfo to be initialized")
	}

	// Project name should be extracted from path
	if model.welcomeInfo.ProjectPath != "/tmp/test-project" {
		t.Errorf("expected ProjectPath '/tmp/test-project', got %q", model.welcomeInfo.ProjectPath)
	}
}

func TestViewWelcome(t *testing.T) {
	ctx := context.Background()
	setup := app.NewSetup("/tmp/test-project", nil)

	model := NewSetupModel(ctx, setup)
	model.welcomeInfo = &WelcomeInfo{
		ProjectName:   "my-project",
		ProjectType:   "go",
		ProjectPath:   "/tmp/test-project",
		IsGitRepo:     true,
		Markers:       []string{".git", "go.mod"},
		SelectedAgent: "auggie",
		Agents: []AgentStatus{
			{Name: "auggie", Available: true},
		},
	}

	view := model.viewWelcome()

	// Should contain the ASCII art logo
	if !containsString(view, "╦═╗") {
		t.Error("expected view to contain ASCII logo")
	}

	// Should contain project name
	if !containsString(view, "my-project") {
		t.Error("expected view to contain project name")
	}

	// Should contain formatted project type
	if !containsString(view, "Go") {
		t.Error("expected view to contain formatted project type 'Go'")
	}

	// Should contain agent info
	if !containsString(view, "auggie") {
		t.Error("expected view to contain agent name")
	}

	// Should contain setup steps
	if !containsString(view, "Analyze project") {
		t.Error("expected view to contain setup steps")
	}

	// Should contain action prompt
	if !containsString(view, "Enter") {
		t.Error("expected view to contain action prompt")
	}
}

func TestRenderLogo(t *testing.T) {
	ctx := context.Background()
	setup := app.NewSetup("/tmp/test", nil)
	model := NewSetupModel(ctx, setup)

	logo := model.renderLogo()

	// Should contain Ralph ASCII art characters
	if !containsString(logo, "╦═╗") || !containsString(logo, "┌─┐") {
		t.Error("expected logo to contain ASCII art characters")
	}
}

func TestRenderProjectInfo_NoInfo(t *testing.T) {
	ctx := context.Background()
	setup := app.NewSetup("/tmp/test", nil)
	model := NewSetupModel(ctx, setup)
	model.welcomeInfo = nil

	info := model.renderProjectInfo()

	// Should still contain the section header
	if !containsString(info, "Project") {
		t.Error("expected section header 'Project'")
	}
}

func TestRenderAgentStatus_NoAgents(t *testing.T) {
	ctx := context.Background()
	setup := app.NewSetup("/tmp/test", nil)
	model := NewSetupModel(ctx, setup)
	model.welcomeInfo = &WelcomeInfo{
		Agents: []AgentStatus{},
	}

	status := model.renderAgentStatus()

	// Should show warning
	if !containsString(status, "No agents") {
		t.Error("expected 'No agents' warning")
	}
}

func TestRenderWhatHappens(t *testing.T) {
	ctx := context.Background()
	setup := app.NewSetup("/tmp/test", nil)
	model := NewSetupModel(ctx, setup)

	content := model.renderWhatHappens()

	// Should contain all 4 steps
	if !containsString(content, "1.") || !containsString(content, "4.") {
		t.Error("expected content to contain step numbers")
	}
	if !containsString(content, "Analyze") {
		t.Error("expected content to mention analysis")
	}
}

func TestRenderQuickTips(t *testing.T) {
	ctx := context.Background()
	setup := app.NewSetup("/tmp/test", nil)
	model := NewSetupModel(ctx, setup)

	tips := model.renderQuickTips()

	// Should contain tips about headless mode
	if !containsString(tips, "headless") {
		t.Error("expected tips to mention headless mode")
	}
	// Should contain config path
	if !containsString(tips, "config.yaml") {
		t.Error("expected tips to mention config.yaml")
	}
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Tests for edge case handling (UX-007)

func TestSetupTUIOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := SetupTUIOptions{}
		if opts.IsLegacy {
			t.Error("IsLegacy should be false by default")
		}
		if opts.NoAgents {
			t.Error("NoAgents should be false by default")
		}
	})

	t.Run("legacy option set", func(t *testing.T) {
		opts := SetupTUIOptions{IsLegacy: true}
		if !opts.IsLegacy {
			t.Error("IsLegacy should be true")
		}
	})

	t.Run("no agents option set", func(t *testing.T) {
		opts := SetupTUIOptions{NoAgents: true}
		if !opts.NoAgents {
			t.Error("NoAgents should be true")
		}
	})
}

func TestPhaseNoAgents(t *testing.T) {
	ctx := context.Background()
	setup := app.NewSetup("/tmp/test-project", nil)
	model := NewSetupModel(ctx, setup)

	// Manually set to PhaseNoAgents (simulating no agents available)
	model.Phase = PhaseNoAgents

	t.Run("view contains no agents message", func(t *testing.T) {
		view := model.viewNoAgents()

		if !containsString(view, "No AI Agents") {
			t.Error("expected view to contain 'No AI Agents'")
		}
		if !containsString(view, "manual") || !containsString(view, "Manual") {
			t.Error("expected view to mention manual mode option")
		}
	})

	t.Run("phase is PhaseNoAgents", func(t *testing.T) {
		if model.Phase != PhaseNoAgents {
			t.Errorf("expected PhaseNoAgents, got %d", model.Phase)
		}
	})
}

func TestPhaseLegacyMigration(t *testing.T) {
	ctx := context.Background()
	setup := app.NewSetup("/tmp/test-project", nil)
	model := NewSetupModel(ctx, setup)

	// Manually set to PhaseLegacyMigration
	model.Phase = PhaseLegacyMigration
	model.isLegacy = true

	t.Run("view contains legacy migration message", func(t *testing.T) {
		view := model.viewLegacyMigration()

		if !containsString(view, "Legacy") {
			t.Error("expected view to contain 'Legacy'")
		}
		// Should offer migration options
		if !containsString(view, "y") || !containsString(view, "n") {
			t.Error("expected view to show y/n options")
		}
	})

	t.Run("phase is PhaseLegacyMigration", func(t *testing.T) {
		if model.Phase != PhaseLegacyMigration {
			t.Errorf("expected PhaseLegacyMigration, got %d", model.Phase)
		}
	})

	t.Run("isLegacy flag is set", func(t *testing.T) {
		if !model.isLegacy {
			t.Error("expected isLegacy to be true")
		}
	})
}

func TestSetupModelWithOptions(t *testing.T) {
	ctx := context.Background()

	t.Run("no agents option sets correct initial phase", func(t *testing.T) {
		setup := app.NewSetup("/tmp/test", nil)
		model := NewSetupModel(ctx, setup)
		// Apply options (simulating what RunSetupTUIWithOptions does)
		opts := SetupTUIOptions{NoAgents: true}
		if opts.NoAgents {
			model.Phase = PhaseNoAgents
		}

		if model.Phase != PhaseNoAgents {
			t.Errorf("expected PhaseNoAgents, got %d", model.Phase)
		}
	})

	t.Run("legacy option sets correct initial phase", func(t *testing.T) {
		setup := app.NewSetup("/tmp/test", nil)
		model := NewSetupModel(ctx, setup)
		// Apply options (simulating what RunSetupTUIWithOptions does)
		opts := SetupTUIOptions{IsLegacy: true}
		if opts.IsLegacy {
			model.Phase = PhaseLegacyMigration
			model.isLegacy = true
		}

		if model.Phase != PhaseLegacyMigration {
			t.Errorf("expected PhaseLegacyMigration, got %d", model.Phase)
		}
		if !model.isLegacy {
			t.Error("expected isLegacy to be true")
		}
	})
}

func TestErrorPhaseRetry(t *testing.T) {
	ctx := context.Background()
	setup := app.NewSetup("/tmp/test-project", nil)
	model := NewSetupModel(ctx, setup)

	// Simulate error phase with retry enabled
	model.Phase = PhaseError
	model.errorMsg = "test error"
	model.canRetry = true
	model.canSkipAnalysis = true
	// Need to set retryFunc for retry option to appear
	model.retryFunc = func() tea.Cmd { return nil }

	t.Run("error view shows retry option when available", func(t *testing.T) {
		view := model.viewError()

		if !containsString(view, "test error") {
			t.Error("expected view to contain error message")
		}
		// Should show retry option (the view contains "Press 'r' to retry")
		if !containsString(view, "retry") {
			t.Error("expected view to show retry option")
		}
	})

	t.Run("error view shows manual mode when available", func(t *testing.T) {
		view := model.viewError()

		// Should show manual mode option
		if !containsString(view, "m") || !containsString(view, "anual") {
			t.Error("expected view to show manual mode option")
		}
	})

	t.Run("canRetry flag is set", func(t *testing.T) {
		if !model.canRetry {
			t.Error("expected canRetry to be true")
		}
	})

	t.Run("canSkipAnalysis flag is set", func(t *testing.T) {
		if !model.canSkipAnalysis {
			t.Error("expected canSkipAnalysis to be true")
		}
	})
}

func TestHandleCancel(t *testing.T) {
	ctx := context.Background()
	setup := app.NewSetup("/tmp/test", nil)
	model := NewSetupModel(ctx, setup)

	t.Run("ctrl+c from welcome phase", func(t *testing.T) {
		model.Phase = PhaseWelcome
		_, cmd := model.handleCancel()
		// Should return quit command
		if cmd == nil {
			t.Error("expected quit command, got nil")
		}
	})

	t.Run("ctrl+c from error phase", func(t *testing.T) {
		model.Phase = PhaseError
		_, cmd := model.handleCancel()
		// Should return quit command
		if cmd == nil {
			t.Error("expected quit command, got nil")
		}
	})

	t.Run("ctrl+c from no agents phase", func(t *testing.T) {
		model.Phase = PhaseNoAgents
		_, cmd := model.handleCancel()
		// Should return quit command
		if cmd == nil {
			t.Error("expected quit command, got nil")
		}
	})
}

