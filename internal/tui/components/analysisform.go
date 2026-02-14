// Package components provides reusable TUI components for ralph.
package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wexinc/ralph/internal/build"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// AnalysisConfirmedMsg is sent when the user confirms the analysis.
type AnalysisConfirmedMsg struct {
	Analysis *build.ProjectAnalysis
}

// ReanalyzeRequestedMsg is sent when the user requests re-analysis.
type ReanalyzeRequestedMsg struct{}

// AnalysisForm displays project analysis results for user confirmation.
type AnalysisForm struct {
	// Fields
	projectType *TextInput
	languages   *TextInput
	buildCmd    *TextInput
	buildReady  *Checkbox
	testCmd     *TextInput
	testReady   *Checkbox
	greenfield  *Checkbox

	// Buttons
	confirmBtn   *Button
	reanalyzeBtn *Button

	// State
	fields        []FormField
	focusIndex    int
	width         int
	showReasoning bool
	reasoning     string

	// Original analysis for reference
	original *build.ProjectAnalysis
}

// NewAnalysisForm creates a new AnalysisForm.
func NewAnalysisForm() *AnalysisForm {
	f := &AnalysisForm{}

	// Create text inputs
	f.projectType = NewTextInput("project_type", "Project Type")
	f.projectType.SetWidth(50)
	f.projectType.SetPlaceholder("e.g., go, node, python")

	f.languages = NewTextInput("languages", "Languages")
	f.languages.SetWidth(50)
	f.languages.SetPlaceholder("e.g., go, typescript")

	f.buildCmd = NewTextInput("build_cmd", "Build Command")
	f.buildCmd.SetWidth(50)
	f.buildCmd.SetPlaceholder("e.g., go build ./...")

	f.testCmd = NewTextInput("test_cmd", "Test Command")
	f.testCmd.SetWidth(50)
	f.testCmd.SetPlaceholder("e.g., go test ./...")

	// Create checkboxes
	f.buildReady = NewCheckbox("build_ready", "Build Ready")
	f.testReady = NewCheckbox("test_ready", "Tests Ready")
	f.greenfield = NewCheckbox("greenfield", "Greenfield Project")

	// Create buttons
	f.confirmBtn = NewButton("confirm", "Confirm & Start")
	f.confirmBtn.SetStyle(ButtonStylePrimary)

	f.reanalyzeBtn = NewButton("reanalyze", "Re-analyze")
	f.reanalyzeBtn.SetStyle(ButtonStyleSecondary)

	// Build field list (order matters for navigation)
	f.fields = []FormField{
		f.projectType,
		f.languages,
		f.buildCmd,
		f.buildReady,
		f.testCmd,
		f.testReady,
		f.greenfield,
		f.confirmBtn,
		f.reanalyzeBtn,
	}

	return f
}

// SetAnalysis populates the form with analysis data.
func (f *AnalysisForm) SetAnalysis(analysis *build.ProjectAnalysis) {
	f.original = analysis

	f.projectType.SetValue(analysis.ProjectType)
	f.languages.SetValue(strings.Join(analysis.Languages, ", "))

	if analysis.Build.Command != nil {
		f.buildCmd.SetValue(*analysis.Build.Command)
	}
	f.buildReady.SetChecked(analysis.Build.Ready)

	if analysis.Test.Command != nil {
		f.testCmd.SetValue(*analysis.Test.Command)
	}
	f.testReady.SetChecked(analysis.Test.Ready)

	f.greenfield.SetChecked(analysis.IsGreenfield)

	// Build reasoning string from analysis
	f.reasoning = f.buildReasoning(analysis)
}

// buildReasoning creates a summary of AI reasoning.
func (f *AnalysisForm) buildReasoning(analysis *build.ProjectAnalysis) string {
	var parts []string
	if analysis.Build.Reason != "" {
		parts = append(parts, "Build: "+analysis.Build.Reason)
	}
	if analysis.Test.Reason != "" {
		parts = append(parts, "Test: "+analysis.Test.Reason)
	}
	if analysis.ProjectContext != "" {
		parts = append(parts, "Context: "+analysis.ProjectContext)
	}
	return strings.Join(parts, "\n")
}

// GetAnalysis returns the modified analysis from form values.
func (f *AnalysisForm) GetAnalysis() *build.ProjectAnalysis {
	analysis := &build.ProjectAnalysis{}

	// Copy from original if available
	if f.original != nil {
		*analysis = *f.original
	}

	// Update with form values
	analysis.ProjectType = f.projectType.Value()
	analysis.Languages = parseLanguages(f.languages.Value())
	analysis.IsGreenfield = f.greenfield.Checked()

	buildCmd := f.buildCmd.Value()
	if buildCmd != "" {
		analysis.Build.Command = &buildCmd
	} else {
		analysis.Build.Command = nil
	}
	analysis.Build.Ready = f.buildReady.Checked()

	testCmd := f.testCmd.Value()
	if testCmd != "" {
		analysis.Test.Command = &testCmd
	} else {
		analysis.Test.Command = nil
	}
	analysis.Test.Ready = f.testReady.Checked()

	return analysis
}

// parseLanguages splits a comma-separated language string into a slice.
func parseLanguages(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// SetWidth sets the form width.
func (f *AnalysisForm) SetWidth(width int) {
	f.width = width
	for _, field := range f.fields {
		if ti, ok := field.(*TextInput); ok {
			ti.SetWidth(width - 4)
		}
	}
}

// ToggleReasoning toggles the reasoning visibility.
func (f *AnalysisForm) ToggleReasoning() {
	f.showReasoning = !f.showReasoning
}

// Focus focuses the first field in the form.
func (f *AnalysisForm) Focus() tea.Cmd {
	if len(f.fields) == 0 {
		return nil
	}
	f.focusIndex = 0
	return f.fields[0].Focus()
}

// Blur blurs all fields.
func (f *AnalysisForm) Blur() {
	for _, field := range f.fields {
		field.Blur()
	}
}

// NextField moves focus to the next field.
func (f *AnalysisForm) NextField() tea.Cmd {
	if len(f.fields) == 0 {
		return nil
	}
	if f.focusIndex >= 0 && f.focusIndex < len(f.fields) {
		f.fields[f.focusIndex].Blur()
	}
	f.focusIndex = (f.focusIndex + 1) % len(f.fields)
	return f.fields[f.focusIndex].Focus()
}

// PrevField moves focus to the previous field.
func (f *AnalysisForm) PrevField() tea.Cmd {
	if len(f.fields) == 0 {
		return nil
	}
	if f.focusIndex >= 0 && f.focusIndex < len(f.fields) {
		f.fields[f.focusIndex].Blur()
	}
	f.focusIndex--
	if f.focusIndex < 0 {
		f.focusIndex = len(f.fields) - 1
	}
	return f.fields[f.focusIndex].Focus()
}

// FocusIndex returns the current focus index.
func (f *AnalysisForm) FocusIndex() int {
	return f.focusIndex
}

// Update handles messages for the form.
func (f *AnalysisForm) Update(msg tea.Msg) (*AnalysisForm, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			cmds = append(cmds, f.NextField())
			return f, tea.Batch(cmds...)

		case "shift+tab":
			cmds = append(cmds, f.PrevField())
			return f, tea.Batch(cmds...)

		case "r":
			// Quick key for re-analyze
			return f, func() tea.Msg {
				return ReanalyzeRequestedMsg{}
			}
		}
	}

	// Delegate to focused field
	if f.focusIndex >= 0 && f.focusIndex < len(f.fields) {
		field := f.fields[f.focusIndex]
		switch typedField := field.(type) {
		case *TextInput:
			updated, cmd := typedField.Update(msg)
			f.fields[f.focusIndex] = updated
			cmds = append(cmds, cmd)

		case *Checkbox:
			updated, cmd := typedField.Update(msg)
			f.fields[f.focusIndex] = updated
			cmds = append(cmds, cmd)

		case *Button:
			updated, cmd, activated := typedField.Update(msg)
			f.fields[f.focusIndex] = updated
			cmds = append(cmds, cmd)
			if activated {
				if typedField.ID() == "confirm" {
					cmds = append(cmds, func() tea.Msg {
						return AnalysisConfirmedMsg{Analysis: f.GetAnalysis()}
					})
				} else if typedField.ID() == "reanalyze" {
					cmds = append(cmds, func() tea.Msg {
						return ReanalyzeRequestedMsg{}
					})
				}
			}
		}
	}

	return f, tea.Batch(cmds...)
}

// View renders the form.
func (f *AnalysisForm) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Bold(true).
		Padding(0, 1)
	b.WriteString(titleStyle.Render("📋 Project Analysis Results"))
	b.WriteString("\n\n")

	// Fields section
	b.WriteString("  ")
	b.WriteString(f.projectType.View())
	b.WriteString("\n  ")
	b.WriteString(f.languages.View())
	b.WriteString("\n\n")

	// Build section
	b.WriteString("  ")
	b.WriteString(f.buildCmd.View())
	b.WriteString("\n  ")
	b.WriteString(f.buildReady.View())
	b.WriteString("\n\n")

	// Test section
	b.WriteString("  ")
	b.WriteString(f.testCmd.View())
	b.WriteString("\n  ")
	b.WriteString(f.testReady.View())
	b.WriteString("\n\n")

	// Greenfield option
	b.WriteString("  ")
	b.WriteString(f.greenfield.View())
	b.WriteString("\n")

	// Reasoning section (collapsible)
	b.WriteString("\n  ")
	divider := lipgloss.NewStyle().Foreground(styles.Muted).Render(strings.Repeat("┄", 50))
	b.WriteString(divider)
	b.WriteString("\n")

	reasoningHeader := "  ▶ AI Reasoning (press 'r' to toggle)"
	if f.showReasoning {
		reasoningHeader = "  ▼ AI Reasoning"
	}
	reasoningStyle := lipgloss.NewStyle().Foreground(styles.MutedLight)
	b.WriteString(reasoningStyle.Render(reasoningHeader))
	b.WriteString("\n")

	if f.showReasoning && f.reasoning != "" {
		reasoningTextStyle := lipgloss.NewStyle().
			Foreground(styles.Muted).
			PaddingLeft(4)
		b.WriteString(reasoningTextStyle.Render(f.reasoning))
		b.WriteString("\n")
	}

	// Buttons
	b.WriteString("\n  ")
	b.WriteString(f.confirmBtn.View())
	b.WriteString("    ")
	b.WriteString(f.reanalyzeBtn.View())
	b.WriteString("\n")

	// Shortcut bar
	b.WriteString("\n  ")
	shortcutBar := NewShortcutBar(AnalysisShortcuts...)
	b.WriteString(shortcutBar.View())

	return b.String()
}
