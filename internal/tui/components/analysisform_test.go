package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dbmrq/ralph/internal/build"
)

func TestNewAnalysisForm(t *testing.T) {
	form := NewAnalysisForm()
	if form == nil {
		t.Fatal("NewAnalysisForm returned nil")
	}

	// Should have 9 fields (7 inputs/checkboxes + 2 buttons)
	if len(form.fields) != 9 {
		t.Errorf("Expected 9 fields, got %d", len(form.fields))
	}
}

func TestAnalysisFormSetAnalysis(t *testing.T) {
	form := NewAnalysisForm()
	buildCmd := "go build ./..."
	testCmd := "go test ./..."

	analysis := &build.ProjectAnalysis{
		ProjectType:  "go",
		Languages:    []string{"go", "shell"},
		IsGreenfield: false,
		Build: build.BuildAnalysis{
			Ready:   true,
			Command: &buildCmd,
			Reason:  "go.mod found",
		},
		Test: build.TestAnalysis{
			Ready:   true,
			Command: &testCmd,
			Reason:  "test files exist",
		},
		ProjectContext: "A Go CLI project",
	}

	form.SetAnalysis(analysis)

	// Verify values were set
	if form.projectType.Value() != "go" {
		t.Errorf("Expected project type 'go', got '%s'", form.projectType.Value())
	}

	if form.languages.Value() != "go, shell" {
		t.Errorf("Expected languages 'go, shell', got '%s'", form.languages.Value())
	}

	if form.buildCmd.Value() != buildCmd {
		t.Errorf("Expected build command '%s', got '%s'", buildCmd, form.buildCmd.Value())
	}

	if !form.buildReady.Checked() {
		t.Error("Expected build ready to be checked")
	}

	if form.testCmd.Value() != testCmd {
		t.Errorf("Expected test command '%s', got '%s'", testCmd, form.testCmd.Value())
	}

	if !form.testReady.Checked() {
		t.Error("Expected test ready to be checked")
	}

	if form.greenfield.Checked() {
		t.Error("Expected greenfield to be unchecked")
	}
}

func TestAnalysisFormGetAnalysis(t *testing.T) {
	form := NewAnalysisForm()
	buildCmd := "make build"
	testCmd := "make test"

	// Set up original analysis
	original := &build.ProjectAnalysis{
		ProjectType:  "node",
		Languages:    []string{"typescript"},
		IsGreenfield: true,
		Build: build.BuildAnalysis{
			Ready:   false,
			Command: nil,
			Reason:  "no build files",
		},
	}
	form.SetAnalysis(original)

	// Modify form values
	form.projectType.SetValue("go")
	form.languages.SetValue("go, rust")
	form.buildCmd.SetValue(buildCmd)
	form.buildReady.SetChecked(true)
	form.testCmd.SetValue(testCmd)
	form.testReady.SetChecked(true)
	form.greenfield.SetChecked(false)

	// Get modified analysis
	result := form.GetAnalysis()

	if result.ProjectType != "go" {
		t.Errorf("Expected project type 'go', got '%s'", result.ProjectType)
	}

	if len(result.Languages) != 2 || result.Languages[0] != "go" || result.Languages[1] != "rust" {
		t.Errorf("Expected languages [go, rust], got %v", result.Languages)
	}

	if result.Build.Command == nil || *result.Build.Command != buildCmd {
		t.Errorf("Expected build command '%s'", buildCmd)
	}

	if !result.Build.Ready {
		t.Error("Expected build ready to be true")
	}

	if result.Test.Command == nil || *result.Test.Command != testCmd {
		t.Errorf("Expected test command '%s'", testCmd)
	}

	if !result.Test.Ready {
		t.Error("Expected test ready to be true")
	}

	if result.IsGreenfield {
		t.Error("Expected greenfield to be false")
	}
}

func TestAnalysisFormNavigation(t *testing.T) {
	form := NewAnalysisForm()
	form.Focus()

	if form.FocusIndex() != 0 {
		t.Errorf("Expected focus index 0, got %d", form.FocusIndex())
	}

	form.NextField()
	if form.FocusIndex() != 1 {
		t.Errorf("Expected focus index 1, got %d", form.FocusIndex())
	}

	form.PrevField()
	if form.FocusIndex() != 0 {
		t.Errorf("Expected focus index 0, got %d", form.FocusIndex())
	}
}

func TestAnalysisFormTabNavigation(t *testing.T) {
	form := NewAnalysisForm()
	form.Focus()

	// Tab should move to next field
	msg := tea.KeyMsg{Type: tea.KeyTab}
	form.Update(msg)

	if form.FocusIndex() != 1 {
		t.Errorf("Tab should move focus, expected 1, got %d", form.FocusIndex())
	}
}

func TestAnalysisFormShiftTabNavigation(t *testing.T) {
	form := NewAnalysisForm()
	form.Focus()
	form.NextField() // Move to index 1

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	form.Update(msg)

	if form.FocusIndex() != 0 {
		t.Errorf("Shift+Tab should move to previous field, expected 0, got %d", form.FocusIndex())
	}
}

func TestAnalysisFormView(t *testing.T) {
	form := NewAnalysisForm()
	buildCmd := "go build ./..."

	analysis := &build.ProjectAnalysis{
		ProjectType: "go",
		Languages:   []string{"go"},
		Build: build.BuildAnalysis{
			Ready:   true,
			Command: &buildCmd,
			Reason:  "go.mod found",
		},
		ProjectContext: "A Go project",
	}
	form.SetAnalysis(analysis)

	view := form.View()

	// Check title
	if !strings.Contains(view, "Project Analysis Results") {
		t.Error("View should contain title")
	}

	// Check buttons
	if !strings.Contains(view, "Confirm & Start") {
		t.Error("View should contain Confirm button")
	}

	if !strings.Contains(view, "Re-analyze") {
		t.Error("View should contain Re-analyze button")
	}
}

func TestAnalysisFormToggleReasoning(t *testing.T) {
	form := NewAnalysisForm()
	buildCmd := "go build ./..."

	analysis := &build.ProjectAnalysis{
		ProjectType: "go",
		Languages:   []string{"go"},
		Build: build.BuildAnalysis{
			Ready:   true,
			Command: &buildCmd,
			Reason:  "go.mod found",
		},
		ProjectContext: "Test context",
	}
	form.SetAnalysis(analysis)

	// Initially reasoning is hidden
	if form.showReasoning {
		t.Error("Reasoning should be hidden initially")
	}

	form.ToggleReasoning()

	if !form.showReasoning {
		t.Error("Reasoning should be visible after toggle")
	}

	form.ToggleReasoning()

	if form.showReasoning {
		t.Error("Reasoning should be hidden after second toggle")
	}
}

func TestParseLanguages(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"go", []string{"go"}},
		{"go, rust", []string{"go", "rust"}},
		{"go,rust,python", []string{"go", "rust", "python"}},
		{"  go  ,  rust  ", []string{"go", "rust"}},
		{",,,", []string{}},
	}

	for _, tt := range tests {
		result := parseLanguages(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseLanguages(%q): expected %v, got %v", tt.input, tt.expected, result)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("parseLanguages(%q): expected %v, got %v", tt.input, tt.expected, result)
				break
			}
		}
	}
}

func TestAnalysisFormGetAnalysisEmptyCommands(t *testing.T) {
	form := NewAnalysisForm()
	form.SetAnalysis(&build.ProjectAnalysis{
		ProjectType: "unknown",
	})

	// Leave commands empty
	form.buildCmd.SetValue("")
	form.testCmd.SetValue("")

	result := form.GetAnalysis()

	if result.Build.Command != nil {
		t.Error("Expected nil build command for empty string")
	}

	if result.Test.Command != nil {
		t.Error("Expected nil test command for empty string")
	}
}

func TestAnalysisFormConfirmButton(t *testing.T) {
	form := NewAnalysisForm()
	form.SetAnalysis(&build.ProjectAnalysis{
		ProjectType: "go",
		Languages:   []string{"go"},
	})

	// Focus the confirm button (index 7)
	form.Focus()
	for i := 0; i < 7; i++ {
		form.NextField()
	}

	if form.FocusIndex() != 7 {
		t.Fatalf("Expected focus on confirm button (index 7), got %d", form.FocusIndex())
	}

	// Press Enter on confirm button
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := form.Update(msg)

	// Should produce a command that returns AnalysisConfirmedMsg
	if cmd == nil {
		t.Fatal("Expected a command from pressing Enter on confirm button")
	}

	// Execute the command and check message type
	result := cmd()
	if _, ok := result.(AnalysisConfirmedMsg); !ok {
		t.Errorf("Expected AnalysisConfirmedMsg, got %T", result)
	}
}

func TestAnalysisFormReanalyzeButton(t *testing.T) {
	form := NewAnalysisForm()
	form.SetAnalysis(&build.ProjectAnalysis{
		ProjectType: "go",
	})

	// Focus the reanalyze button (index 8)
	form.Focus()
	for i := 0; i < 8; i++ {
		form.NextField()
	}

	if form.FocusIndex() != 8 {
		t.Fatalf("Expected focus on reanalyze button (index 8), got %d", form.FocusIndex())
	}

	// Press Enter on reanalyze button
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := form.Update(msg)

	if cmd == nil {
		t.Fatal("Expected a command from pressing Enter on reanalyze button")
	}

	// Execute the command and check message type
	result := cmd()
	if _, ok := result.(ReanalyzeRequestedMsg); !ok {
		t.Errorf("Expected ReanalyzeRequestedMsg, got %T", result)
	}
}

func TestAnalysisFormReanalyzeKey(t *testing.T) {
	form := NewAnalysisForm()
	form.SetAnalysis(&build.ProjectAnalysis{
		ProjectType: "go",
	})
	form.Focus()

	// Press 'r' key for quick re-analyze
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := form.Update(msg)

	if cmd == nil {
		t.Fatal("Expected a command from pressing 'r' key")
	}

	result := cmd()
	if _, ok := result.(ReanalyzeRequestedMsg); !ok {
		t.Errorf("Expected ReanalyzeRequestedMsg, got %T", result)
	}
}

func TestAnalysisFormSetWidth(t *testing.T) {
	form := NewAnalysisForm()
	form.SetWidth(100)

	// Should not panic
	view := form.View()
	if view == "" {
		t.Error("View should not be empty after SetWidth")
	}
}

func TestAnalysisFormBlur(t *testing.T) {
	form := NewAnalysisForm()
	form.Focus()

	if !form.projectType.Focused() {
		t.Fatal("First field should be focused")
	}

	form.Blur()

	if form.projectType.Focused() {
		t.Error("Field should not be focused after form Blur")
	}
}
