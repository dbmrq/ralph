// Package tui provides the terminal user interface for ralph.
package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/wexinc/ralph/internal/app"
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

