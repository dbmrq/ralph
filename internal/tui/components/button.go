// Package components provides reusable TUI components for ralph.
package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// ButtonStyle represents the visual style of a button.
type ButtonStyle int

const (
	// ButtonStylePrimary is the default button style.
	ButtonStylePrimary ButtonStyle = iota
	// ButtonStyleSecondary is a less prominent button style.
	ButtonStyleSecondary
	// ButtonStyleDanger is for destructive actions.
	ButtonStyleDanger
)

// Button is a clickable button component.
type Button struct {
	label   string
	focused bool
	id      string
	style   ButtonStyle
}

// NewButton creates a new Button component.
func NewButton(id, label string) *Button {
	return &Button{
		label: label,
		id:    id,
		style: ButtonStylePrimary,
	}
}

// ID returns the component's unique identifier.
func (b *Button) ID() string {
	return b.id
}

// Focus focuses the button.
func (b *Button) Focus() tea.Cmd {
	b.focused = true
	return nil
}

// Blur removes focus from the button.
func (b *Button) Blur() {
	b.focused = false
}

// Focused returns whether the button is focused.
func (b *Button) Focused() bool {
	return b.focused
}

// SetStyle sets the button style.
func (b *Button) SetStyle(style ButtonStyle) {
	b.style = style
}

// SetLabel sets the button label.
func (b *Button) SetLabel(label string) {
	b.label = label
}

// Label returns the button label.
func (b *Button) Label() string {
	return b.label
}

// Update handles messages for the button.
// Returns true if the button was activated.
func (b *Button) Update(msg tea.Msg) (*Button, tea.Cmd, bool) {
	if !b.focused {
		return b, nil, false
	}

	activated := false
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			activated = true
		}
	}

	return b, nil, activated
}

// View renders the button.
func (b *Button) View() string {
	var buttonStyle lipgloss.Style

	if b.focused {
		// Focused button - use inverted colors based on style
		switch b.style {
		case ButtonStylePrimary:
			buttonStyle = lipgloss.NewStyle().
				Foreground(styles.Background).
				Background(styles.Primary).
				Bold(true).
				Padding(0, 2)
		case ButtonStyleSecondary:
			buttonStyle = lipgloss.NewStyle().
				Foreground(styles.Background).
				Background(styles.Secondary).
				Bold(true).
				Padding(0, 2)
		case ButtonStyleDanger:
			buttonStyle = lipgloss.NewStyle().
				Foreground(styles.Foreground).
				Background(styles.Error).
				Bold(true).
				Padding(0, 2)
		}
	} else {
		// Unfocused button - show outline/muted version
		switch b.style {
		case ButtonStylePrimary:
			buttonStyle = lipgloss.NewStyle().
				Foreground(styles.Primary).
				Border(lipgloss.NormalBorder()).
				BorderForeground(styles.Primary).
				Padding(0, 1)
		case ButtonStyleSecondary:
			buttonStyle = lipgloss.NewStyle().
				Foreground(styles.MutedLight).
				Border(lipgloss.NormalBorder()).
				BorderForeground(styles.Muted).
				Padding(0, 1)
		case ButtonStyleDanger:
			buttonStyle = lipgloss.NewStyle().
				Foreground(styles.Error).
				Border(lipgloss.NormalBorder()).
				BorderForeground(styles.Error).
				Padding(0, 1)
		}
	}

	return buttonStyle.Render(b.label)
}

