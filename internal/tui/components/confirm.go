// Package components provides reusable TUI components for ralph.
package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// ConfirmAction represents the action being confirmed.
type ConfirmAction string

const (
	// ConfirmActionAbort is for aborting the loop.
	ConfirmActionAbort ConfirmAction = "abort"
	// ConfirmActionSkip is for skipping a task.
	ConfirmActionSkip ConfirmAction = "skip"
	// ConfirmActionQuit is for quitting.
	ConfirmActionQuit ConfirmAction = "quit"
)

// ConfirmDialog displays a confirmation prompt for destructive actions.
type ConfirmDialog struct {
	visible     bool
	action      ConfirmAction
	title       string
	message     string
	confirmKey  string
	cancelKey   string
	width       int
	destructive bool
}

// NewConfirmDialog creates a new ConfirmDialog component.
func NewConfirmDialog() *ConfirmDialog {
	return &ConfirmDialog{
		visible:    false,
		confirmKey: "y",
		cancelKey:  "n",
		width:      50,
	}
}

// Show displays the dialog with the given action, title, and message.
func (c *ConfirmDialog) Show(action ConfirmAction, title, message string, destructive bool) {
	c.visible = true
	c.action = action
	c.title = title
	c.message = message
	c.destructive = destructive
}

// ShowAbort shows abort confirmation.
func (c *ConfirmDialog) ShowAbort() {
	c.Show(ConfirmActionAbort, "Abort Loop?",
		"This will stop the loop immediately. Any in-progress work may be lost.",
		true)
}

// ShowSkip shows skip confirmation.
func (c *ConfirmDialog) ShowSkip(taskName string) {
	c.Show(ConfirmActionSkip, "Skip Task?",
		"Skip task: "+taskName+"\nThe task will be marked as skipped and won't be executed.",
		false)
}

// ShowQuit shows quit confirmation.
func (c *ConfirmDialog) ShowQuit() {
	c.Show(ConfirmActionQuit, "Quit Ralph?",
		"The current session will be saved. You can resume later with --continue.",
		false)
}

// Hide hides the dialog.
func (c *ConfirmDialog) Hide() {
	c.visible = false
}

// IsVisible returns whether the dialog is visible.
func (c *ConfirmDialog) IsVisible() bool {
	return c.visible
}

// Action returns the current action being confirmed.
func (c *ConfirmDialog) Action() ConfirmAction {
	return c.action
}

// SetSize sets the dialog width.
func (c *ConfirmDialog) SetSize(width int) {
	c.width = width
}

// Update handles input messages.
func (c *ConfirmDialog) Update(msg tea.Msg) tea.Cmd {
	if !c.visible {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch strings.ToLower(msg.String()) {
		case "y", "enter":
			action := c.action
			c.Hide()
			return func() tea.Msg {
				return ConfirmYesMsg{Action: action}
			}
		case "n", "esc":
			c.Hide()
			return func() tea.Msg {
				return ConfirmNoMsg{}
			}
		}
	}
	return nil
}

// View renders the confirmation dialog.
func (c *ConfirmDialog) View() string {
	if !c.visible {
		return ""
	}

	var b strings.Builder

	// Title
	titleBg := styles.Warning
	if c.destructive {
		titleBg = styles.Error
	}
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Background(titleBg).
		Bold(true).
		Padding(0, 1).
		Width(c.width - 4)
	b.WriteString(titleStyle.Render("  " + c.title))
	b.WriteString("\n\n")

	// Message
	msgStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Width(c.width - 8)
	b.WriteString(msgStyle.Render(c.message))
	b.WriteString("\n\n")

	// Buttons
	yesStyle := styles.ButtonDangerStyle
	if !c.destructive {
		yesStyle = styles.ButtonPrimaryStyle
	}
	noStyle := styles.ButtonSecondaryUnfocusedStyle

	b.WriteString(yesStyle.Render("[Y]es"))
	b.WriteString("  ")
	b.WriteString(noStyle.Render("[N]o"))

	// Box
	borderColor := styles.Warning
	if c.destructive {
		borderColor = styles.Error
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(borderColor).
		Padding(1, 2)

	return boxStyle.Render(b.String())
}

// ConfirmYesMsg is sent when the user confirms.
type ConfirmYesMsg struct {
	Action ConfirmAction
}

// ConfirmNoMsg is sent when the user cancels.
type ConfirmNoMsg struct{}

