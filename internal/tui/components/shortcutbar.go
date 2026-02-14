// Package components provides reusable TUI components for ralph.
package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/wexinc/ralph/internal/tui/styles"
)

// ShortcutDef defines a single keyboard shortcut.
type ShortcutDef struct {
	Key  string
	Desc string
}

// ShortcutBar is a component that displays contextual keyboard shortcuts.
// It provides a standardized way to show hints at the bottom of each TUI phase.
type ShortcutBar struct {
	shortcuts []ShortcutDef
	width     int
	centered  bool
}

// NewShortcutBar creates a new ShortcutBar with the given shortcuts.
func NewShortcutBar(shortcuts ...ShortcutDef) *ShortcutBar {
	return &ShortcutBar{
		shortcuts: shortcuts,
		centered:  false,
	}
}

// SetShortcuts replaces all shortcuts.
func (s *ShortcutBar) SetShortcuts(shortcuts ...ShortcutDef) {
	s.shortcuts = shortcuts
}

// SetWidth sets the bar width for alignment.
func (s *ShortcutBar) SetWidth(width int) {
	s.width = width
}

// SetCentered controls whether the bar content is centered.
func (s *ShortcutBar) SetCentered(centered bool) {
	s.centered = centered
}

// View renders the shortcut bar.
func (s *ShortcutBar) View() string {
	if len(s.shortcuts) == 0 {
		return ""
	}

	var parts []string
	for _, sc := range s.shortcuts {
		parts = append(parts, s.renderShortcut(sc))
	}

	content := strings.Join(parts, s.renderSeparator())

	if s.centered && s.width > 0 {
		containerStyle := lipgloss.NewStyle().
			Width(s.width).
			Align(lipgloss.Center)
		return containerStyle.Render(content)
	}

	return content
}

// renderShortcut renders a single shortcut (key: description).
func (s *ShortcutBar) renderShortcut(sc ShortcutDef) string {
	keyStyle := styles.KeyStyle
	helpStyle := styles.HelpStyle

	return keyStyle.Render(sc.Key) + helpStyle.Render(":") + helpStyle.Render(sc.Desc)
}

// renderSeparator renders the separator between shortcuts.
func (s *ShortcutBar) renderSeparator() string {
	return lipgloss.NewStyle().Foreground(styles.Muted).Render(" │ ")
}

// Predefined shortcut sets for common TUI phases.
var (
	// WelcomeShortcuts are shortcuts for the welcome screen.
	WelcomeShortcuts = []ShortcutDef{
		{"Enter", "begin"},
		{"q", "quit"},
		{"?", "help"},
	}

	// AnalysisShortcuts are shortcuts for the analysis confirmation form.
	AnalysisShortcuts = []ShortcutDef{
		{"Tab", "next"},
		{"Enter", "edit/toggle"},
		{"r", "re-analyze"},
		{"?", "help"},
	}

	// TaskInitShortcuts are shortcuts for the task init selector.
	TaskInitShortcuts = []ShortcutDef{
		{"↑↓", "select"},
		{"Enter", "choose"},
		{"Esc", "back"},
		{"?", "help"},
	}

	// TaskListShortcuts are shortcuts for the task list form.
	TaskListShortcuts = []ShortcutDef{
		{"↑↓", "select"},
		{"e", "edit"},
		{"d", "delete"},
		{"Tab", "buttons"},
		{"Enter", "confirm"},
	}

	// FileInputShortcuts are shortcuts for file path input.
	FileInputShortcuts = []ShortcutDef{
		{"Enter", "submit"},
		{"Esc", "cancel"},
		{"Tab", "autocomplete"},
	}

	// TextAreaShortcuts are shortcuts for paste/goal input.
	TextAreaShortcuts = []ShortcutDef{
		{"Ctrl+Enter", "submit"},
		{"Esc", "cancel"},
	}

	// MainLoopShortcuts are shortcuts for the main loop view.
	MainLoopShortcuts = []ShortcutDef{
		{"p", "pause"},
		{"s", "skip"},
		{"a", "abort"},
		{"l", "logs"},
		{"q", "quit"},
		{"?", "help"},
	}

	// MainLoopPausedShortcuts are shortcuts when the loop is paused.
	MainLoopPausedShortcuts = []ShortcutDef{
		{"p", "resume"},
		{"s", "skip"},
		{"a", "abort"},
		{"e", "add task"},
		{"m", "model"},
		{"?", "help"},
	}
)

