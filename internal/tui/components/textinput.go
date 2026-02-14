// Package components provides reusable TUI components for ralph.
package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wexinc/ralph/internal/tui/styles"
)

// TextInput is a wrapper around the bubbles textinput component
// that integrates with our form system.
type TextInput struct {
	model       textinput.Model
	label       string
	placeholder string
	focused     bool
	width       int
	id          string
}

// NewTextInput creates a new TextInput component.
func NewTextInput(id, label string) *TextInput {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 30

	return &TextInput{
		model: ti,
		label: label,
		id:    id,
	}
}

// ID returns the component's unique identifier.
func (t *TextInput) ID() string {
	return t.id
}

// Focus focuses the text input.
func (t *TextInput) Focus() tea.Cmd {
	t.focused = true
	return t.model.Focus()
}

// Blur removes focus from the text input.
func (t *TextInput) Blur() {
	t.focused = false
	t.model.Blur()
}

// Focused returns whether the text input is focused.
func (t *TextInput) Focused() bool {
	return t.focused
}

// SetValue sets the text input value.
func (t *TextInput) SetValue(value string) {
	t.model.SetValue(value)
}

// Value returns the current text input value.
func (t *TextInput) Value() string {
	return t.model.Value()
}

// SetPlaceholder sets the placeholder text.
func (t *TextInput) SetPlaceholder(placeholder string) {
	t.placeholder = placeholder
	t.model.Placeholder = placeholder
}

// SetWidth sets the width of the text input.
func (t *TextInput) SetWidth(width int) {
	t.width = width
	t.model.Width = width - len(t.label) - 5 // Account for label and padding
	if t.model.Width < 10 {
		t.model.Width = 10
	}
}

// SetCharLimit sets the character limit.
func (t *TextInput) SetCharLimit(limit int) {
	t.model.CharLimit = limit
}

// Update handles messages for the text input.
func (t *TextInput) Update(msg tea.Msg) (*TextInput, tea.Cmd) {
	if !t.focused {
		return t, nil
	}

	var cmd tea.Cmd
	t.model, cmd = t.model.Update(msg)
	return t, cmd
}

// View renders the text input.
func (t *TextInput) View() string {
	labelStyle := styles.HeaderLabelStyle
	if t.focused {
		labelStyle = lipgloss.NewStyle().
			Foreground(styles.Secondary).
			Bold(true)
	}

	label := labelStyle.Render(t.label + ": ")

	// Use different styles for focused vs unfocused
	inputView := t.model.View()

	// Add brackets around input
	var inputStyle lipgloss.Style
	if t.focused {
		inputStyle = lipgloss.NewStyle().
			Foreground(styles.Foreground).
			Background(styles.Background).
			Padding(0, 1)
	} else {
		inputStyle = lipgloss.NewStyle().
			Foreground(styles.MutedLight).
			Padding(0, 1)
	}

	return label + inputStyle.Render(inputView)
}

// Reset clears the text input value.
func (t *TextInput) Reset() {
	t.model.Reset()
}
