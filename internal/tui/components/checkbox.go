// Package components provides reusable TUI components for ralph.
package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// Checkbox is a toggle checkbox component.
type Checkbox struct {
	label   string
	checked bool
	focused bool
	id      string
}

// NewCheckbox creates a new Checkbox component.
func NewCheckbox(id, label string) *Checkbox {
	return &Checkbox{
		label: label,
		id:    id,
	}
}

// ID returns the component's unique identifier.
func (c *Checkbox) ID() string {
	return c.id
}

// Focus focuses the checkbox.
func (c *Checkbox) Focus() tea.Cmd {
	c.focused = true
	return nil
}

// Blur removes focus from the checkbox.
func (c *Checkbox) Blur() {
	c.focused = false
}

// Focused returns whether the checkbox is focused.
func (c *Checkbox) Focused() bool {
	return c.focused
}

// Toggle toggles the checkbox state.
func (c *Checkbox) Toggle() {
	c.checked = !c.checked
}

// SetChecked sets the checkbox state.
func (c *Checkbox) SetChecked(checked bool) {
	c.checked = checked
}

// Checked returns whether the checkbox is checked.
func (c *Checkbox) Checked() bool {
	return c.checked
}

// Value returns the checkbox value as an interface.
func (c *Checkbox) Value() bool {
	return c.checked
}

// Update handles messages for the checkbox.
func (c *Checkbox) Update(msg tea.Msg) (*Checkbox, tea.Cmd) {
	if !c.focused {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			c.Toggle()
		}
	}

	return c, nil
}

// View renders the checkbox.
func (c *Checkbox) View() string {
	labelStyle := styles.HeaderLabelStyle
	if c.focused {
		labelStyle = lipgloss.NewStyle().
			Foreground(styles.Secondary).
			Bold(true)
	}

	label := labelStyle.Render(c.label + ": ")

	// Checkbox icon
	var checkbox string
	if c.checked {
		checkStyle := lipgloss.NewStyle().Foreground(styles.Success)
		checkbox = checkStyle.Render("[✓]")
	} else {
		uncheckStyle := lipgloss.NewStyle().Foreground(styles.Muted)
		checkbox = uncheckStyle.Render("[ ]")
	}

	// Add focus indicator
	if c.focused {
		focusStyle := lipgloss.NewStyle().
			Foreground(styles.Secondary).
			Bold(true)
		checkbox = focusStyle.Render(checkbox)
	}

	return label + checkbox
}

// SetLabel sets the checkbox label.
func (c *Checkbox) SetLabel(label string) {
	c.label = label
}
