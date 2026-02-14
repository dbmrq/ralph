// Package components provides reusable TUI components for ralph.
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/wexinc/ralph/internal/tui/styles"
)

// StatusBarData contains the data to display in the status bar.
type StatusBarData struct {
	ElapsedTime   time.Duration
	Iteration     int
	BuildStatus   string // "pass", "fail", "running", "pending"
	TestStatus    string // "pass", "fail", "running", "pending"
	LoopState     string // "running", "paused", "completed", "failed"
	Message       string // Optional status message
	ShowShortcuts bool
	Shortcuts     []ShortcutDef // Optional custom shortcuts (overrides context defaults)
}

// StatusBar is a component that displays loop status and keyboard shortcuts.
type StatusBar struct {
	data  StatusBarData
	width int
}

// NewStatusBar creates a new StatusBar component.
func NewStatusBar() *StatusBar {
	return &StatusBar{
		data: StatusBarData{
			BuildStatus:   "pending",
			TestStatus:    "pending",
			LoopState:     "running",
			ShowShortcuts: true,
		},
	}
}

// SetData updates the status bar data.
func (s *StatusBar) SetData(data StatusBarData) {
	s.data = data
}

// SetElapsedTime sets the elapsed time.
func (s *StatusBar) SetElapsedTime(d time.Duration) {
	s.data.ElapsedTime = d
}

// SetIteration sets the current iteration number.
func (s *StatusBar) SetIteration(iteration int) {
	s.data.Iteration = iteration
}

// SetBuildStatus sets the build status.
func (s *StatusBar) SetBuildStatus(status string) {
	s.data.BuildStatus = status
}

// SetTestStatus sets the test status.
func (s *StatusBar) SetTestStatus(status string) {
	s.data.TestStatus = status
}

// SetLoopState sets the loop state.
func (s *StatusBar) SetLoopState(state string) {
	s.data.LoopState = state
}

// SetMessage sets an optional status message.
func (s *StatusBar) SetMessage(message string) {
	s.data.Message = message
}

// SetShowShortcuts sets whether to show keyboard shortcuts.
func (s *StatusBar) SetShowShortcuts(show bool) {
	s.data.ShowShortcuts = show
}

// SetWidth sets the width of the status bar.
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// View renders the status bar.
func (s *StatusBar) View() string {
	sep := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Render(" │ ")

	// Elapsed time
	elapsed := s.formatDuration(s.data.ElapsedTime)
	elapsedLabel := lipgloss.NewStyle().
		Foreground(styles.MutedLight).
		Render("Time: ")
	elapsedValue := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Render(elapsed)

	// Iteration count
	iterLabel := lipgloss.NewStyle().
		Foreground(styles.MutedLight).
		Render("Iter: ")
	iterValue := lipgloss.NewStyle().
		Foreground(styles.Secondary).
		Render(fmt.Sprintf("%d", s.data.Iteration))

	// Build status
	buildLabel := lipgloss.NewStyle().
		Foreground(styles.MutedLight).
		Render("Build: ")
	buildValue := s.renderStatusIndicator(s.data.BuildStatus)

	// Test status
	testLabel := lipgloss.NewStyle().
		Foreground(styles.MutedLight).
		Render("Test: ")
	testValue := s.renderStatusIndicator(s.data.TestStatus)

	// Build left side content
	leftContent := fmt.Sprintf("%s%s%s%s%s%s%s%s%s%s%s%s",
		elapsedLabel, elapsedValue, sep,
		iterLabel, iterValue, sep,
		buildLabel, buildValue, sep,
		testLabel, testValue, sep,
	)

	// Loop state indicator
	stateIcon := s.renderLoopStateIcon(s.data.LoopState)
	leftContent += stateIcon

	// Add message if present
	if s.data.Message != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(styles.MutedLight).
			Italic(true)
		leftContent += sep + msgStyle.Render(s.data.Message)
	}

	// Build right side (keyboard shortcuts)
	rightContent := ""
	if s.data.ShowShortcuts {
		rightContent = s.renderShortcuts()
	}

	// Combine left and right
	containerStyle := lipgloss.NewStyle().
		Background(styles.Background).
		Padding(0, 1)

	if s.width > 0 {
		containerStyle = containerStyle.Width(s.width)

		// Calculate spacing
		leftWidth := lipgloss.Width(leftContent)
		rightWidth := lipgloss.Width(rightContent)
		padding := s.width - leftWidth - rightWidth - 2 // -2 for container padding
		if padding > 0 {
			return containerStyle.Render(leftContent + strings.Repeat(" ", padding) + rightContent)
		}
	}

	return containerStyle.Render(leftContent + "  " + rightContent)
}

// renderStatusIndicator renders a status indicator (pass/fail/running/pending).
func (s *StatusBar) renderStatusIndicator(status string) string {
	switch status {
	case "pass":
		return lipgloss.NewStyle().Foreground(styles.Success).Render("✓")
	case "fail":
		return lipgloss.NewStyle().Foreground(styles.Error).Render("✗")
	case "running":
		return lipgloss.NewStyle().Foreground(styles.Secondary).Render("◐")
	default:
		return lipgloss.NewStyle().Foreground(styles.Muted).Render("○")
	}
}

// renderLoopStateIcon renders the loop state icon.
func (s *StatusBar) renderLoopStateIcon(state string) string {
	switch state {
	case "running":
		return lipgloss.NewStyle().Foreground(styles.Success).Render("● Running")
	case "paused":
		return lipgloss.NewStyle().Foreground(styles.Warning).Render("⏸ Paused")
	case "completed":
		return lipgloss.NewStyle().Foreground(styles.Success).Render("✓ Complete")
	case "failed":
		return lipgloss.NewStyle().Foreground(styles.Error).Render("✗ Failed")
	default:
		return lipgloss.NewStyle().Foreground(styles.Muted).Render("○ Idle")
	}
}

// renderShortcuts renders the keyboard shortcuts based on context.
func (s *StatusBar) renderShortcuts() string {
	// Use custom shortcuts if provided
	if len(s.data.Shortcuts) > 0 {
		bar := NewShortcutBar(s.data.Shortcuts...)
		return bar.View()
	}

	// Use context-aware defaults based on loop state
	var shortcuts []ShortcutDef
	switch s.data.LoopState {
	case "paused":
		shortcuts = MainLoopPausedShortcuts
	case "completed", "failed":
		shortcuts = []ShortcutDef{
			{"Enter", "continue"},
			{"q", "quit"},
			{"?", "help"},
		}
	default: // "running" or other
		shortcuts = MainLoopShortcuts
	}

	bar := NewShortcutBar(shortcuts...)
	return bar.View()
}

// formatDuration formats a duration as HH:MM:SS or MM:SS.
func (s *StatusBar) formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%02d:%02d", m, sec)
}
