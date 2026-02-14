// Package components provides reusable TUI components for ralph.
package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dbmrq/ralph/internal/tui/styles"
)

// Shortcut represents a keyboard shortcut.
type Shortcut struct {
	Key  string
	Desc string
}

// ShortcutGroup represents a group of related shortcuts.
type ShortcutGroup struct {
	Title     string
	Shortcuts []Shortcut
}

// HelpOverlay displays keyboard shortcuts and help information.
type HelpOverlay struct {
	visible bool
	width   int
	height  int
	groups  []ShortcutGroup
}

// NewHelpOverlay creates a new HelpOverlay component.
func NewHelpOverlay() *HelpOverlay {
	return &HelpOverlay{
		visible: false,
		width:   60,
		height:  20,
		groups: []ShortcutGroup{
			{
				Title: "Loop Control",
				Shortcuts: []Shortcut{
					{"p", "Pause/Resume loop"},
					{"s", "Skip current task"},
					{"a", "Abort loop"},
				},
			},
			{
				Title: "Navigation",
				Shortcuts: []Shortcut{
					{"j/↓", "Move down"},
					{"k/↑", "Move up"},
					{"g", "Go to top"},
					{"G", "Go to bottom"},
				},
			},
			{
				Title: "Task Management",
				Shortcuts: []Shortcut{
					{"e", "Add/Edit task"},
					{"m", "Change model"},
					{"l", "View logs"},
				},
			},
			{
				Title: "General",
				Shortcuts: []Shortcut{
					{"h/?", "Toggle help"},
					{"q", "Quit"},
					{"Esc", "Close overlay/Cancel"},
				},
			},
		},
	}
}

// SetGroups sets custom shortcut groups.
func (h *HelpOverlay) SetGroups(groups []ShortcutGroup) {
	h.groups = groups
}

// SetSize sets the overlay dimensions.
func (h *HelpOverlay) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// Show makes the overlay visible.
func (h *HelpOverlay) Show() {
	h.visible = true
}

// Hide hides the overlay.
func (h *HelpOverlay) Hide() {
	h.visible = false
}

// Toggle toggles visibility.
func (h *HelpOverlay) Toggle() {
	h.visible = !h.visible
}

// IsVisible returns whether the overlay is visible.
func (h *HelpOverlay) IsVisible() bool {
	return h.visible
}

// Update handles input messages.
func (h *HelpOverlay) Update(msg tea.Msg) tea.Cmd {
	if !h.visible {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "h", "?", "q":
			h.Hide()
			return func() tea.Msg {
				return HelpClosedMsg{}
			}
		}
	}
	return nil
}

// View renders the help overlay.
func (h *HelpOverlay) View() string {
	if !h.visible {
		return ""
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Background(styles.Primary).
		Bold(true).
		Padding(0, 1).
		Width(h.width - 4)
	b.WriteString(titleStyle.Render("  Keyboard Shortcuts"))
	b.WriteString("\n\n")

	// Render each group
	for i, group := range h.groups {
		b.WriteString(h.renderGroup(group))
		if i < len(h.groups)-1 {
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	footerStyle := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Italic(true)
	b.WriteString(footerStyle.Render("Press any key to close"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(styles.Primary).
		Padding(1, 2)

	return boxStyle.Render(b.String())
}

// renderGroup renders a single shortcut group.
func (h *HelpOverlay) renderGroup(group ShortcutGroup) string {
	var b strings.Builder

	// Group title
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Secondary).
		Bold(true)
	b.WriteString(titleStyle.Render(group.Title))
	b.WriteString("\n")

	// Shortcuts
	for _, shortcut := range group.Shortcuts {
		keyStyle := lipgloss.NewStyle().
			Foreground(styles.Foreground).
			Bold(true).
			Width(8)
		descStyle := lipgloss.NewStyle().
			Foreground(styles.MutedLight)

		b.WriteString("  ")
		b.WriteString(keyStyle.Render(shortcut.Key))
		b.WriteString(" ")
		b.WriteString(descStyle.Render(shortcut.Desc))
		b.WriteString("\n")
	}

	return b.String()
}

// HelpClosedMsg is sent when the help overlay is closed.
type HelpClosedMsg struct{}
