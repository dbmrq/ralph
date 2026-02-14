// Package components provides reusable TUI components for ralph.
package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// FormField is the interface that all form fields must implement.
type FormField interface {
	ID() string
	Focus() tea.Cmd
	Blur()
	Focused() bool
	View() string
}

// FormSubmittedMsg is sent when a form is submitted.
type FormSubmittedMsg struct {
	FormID string
}

// FormCanceledMsg is sent when a form is canceled.
type FormCanceledMsg struct {
	FormID string
}

// Form is a container for form fields with navigation support.
type Form struct {
	id         string
	title      string
	fields     []FormField
	focusIndex int
	width      int
	submitted  bool
	canceled   bool
	showHelp   bool
}

// NewForm creates a new Form container.
func NewForm(id, title string) *Form {
	return &Form{
		id:       id,
		title:    title,
		fields:   []FormField{},
		showHelp: true,
	}
}

// ID returns the form's unique identifier.
func (f *Form) ID() string {
	return f.id
}

// AddField adds a field to the form.
func (f *Form) AddField(field FormField) {
	f.fields = append(f.fields, field)
}

// AddFields adds multiple fields to the form.
func (f *Form) AddFields(fields ...FormField) {
	f.fields = append(f.fields, fields...)
}

// SetWidth sets the form width.
func (f *Form) SetWidth(width int) {
	f.width = width
}

// SetShowHelp sets whether to show help text.
func (f *Form) SetShowHelp(show bool) {
	f.showHelp = show
}

// FocusIndex returns the current focus index.
func (f *Form) FocusIndex() int {
	return f.focusIndex
}

// FocusedField returns the currently focused field, or nil if none.
func (f *Form) FocusedField() FormField {
	if f.focusIndex >= 0 && f.focusIndex < len(f.fields) {
		return f.fields[f.focusIndex]
	}
	return nil
}

// GetField returns a field by ID.
func (f *Form) GetField(id string) FormField {
	for _, field := range f.fields {
		if field.ID() == id {
			return field
		}
	}
	return nil
}

// Fields returns all form fields.
func (f *Form) Fields() []FormField {
	return f.fields
}

// Focus focuses the form (focuses first field).
func (f *Form) Focus() tea.Cmd {
	if len(f.fields) == 0 {
		return nil
	}
	f.focusIndex = 0
	return f.fields[0].Focus()
}

// Blur blurs all fields in the form.
func (f *Form) Blur() {
	for _, field := range f.fields {
		field.Blur()
	}
}

// NextField moves focus to the next field.
func (f *Form) NextField() tea.Cmd {
	if len(f.fields) == 0 {
		return nil
	}

	// Blur current field
	if f.focusIndex >= 0 && f.focusIndex < len(f.fields) {
		f.fields[f.focusIndex].Blur()
	}

	// Move to next field (wrap around)
	f.focusIndex = (f.focusIndex + 1) % len(f.fields)

	return f.fields[f.focusIndex].Focus()
}

// PrevField moves focus to the previous field.
func (f *Form) PrevField() tea.Cmd {
	if len(f.fields) == 0 {
		return nil
	}

	// Blur current field
	if f.focusIndex >= 0 && f.focusIndex < len(f.fields) {
		f.fields[f.focusIndex].Blur()
	}

	// Move to previous field (wrap around)
	f.focusIndex--
	if f.focusIndex < 0 {
		f.focusIndex = len(f.fields) - 1
	}

	return f.fields[f.focusIndex].Focus()
}

// FocusField focuses a specific field by index.
func (f *Form) FocusField(index int) tea.Cmd {
	if index < 0 || index >= len(f.fields) {
		return nil
	}

	// Blur current field
	if f.focusIndex >= 0 && f.focusIndex < len(f.fields) {
		f.fields[f.focusIndex].Blur()
	}

	f.focusIndex = index
	return f.fields[f.focusIndex].Focus()
}

// Submitted returns whether the form was submitted.
func (f *Form) Submitted() bool {
	return f.submitted
}

// Canceled returns whether the form was canceled.
func (f *Form) Canceled() bool {
	return f.canceled
}

// Reset resets the form state.
func (f *Form) Reset() {
	f.submitted = false
	f.canceled = false
	f.focusIndex = 0
}

// Update handles messages for the form.
// It handles Tab/Shift+Tab navigation and delegates other messages to focused field.
func (f *Form) Update(msg tea.Msg) (*Form, tea.Cmd) {
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

		case "esc":
			f.canceled = true
			return f, func() tea.Msg {
				return FormCanceledMsg{FormID: f.id}
			}
		}
	}

	// Delegate message to focused field based on type
	if f.focusIndex >= 0 && f.focusIndex < len(f.fields) {
		field := f.fields[f.focusIndex]
		switch typedField := field.(type) {
		case *TextInput:
			updatedField, cmd := typedField.Update(msg)
			f.fields[f.focusIndex] = updatedField
			cmds = append(cmds, cmd)
		case *Checkbox:
			updatedField, cmd := typedField.Update(msg)
			f.fields[f.focusIndex] = updatedField
			cmds = append(cmds, cmd)
		case *Button:
			updatedField, cmd, activated := typedField.Update(msg)
			f.fields[f.focusIndex] = updatedField
			cmds = append(cmds, cmd)
			if activated {
				f.submitted = true
				cmds = append(cmds, func() tea.Msg {
					return FormSubmittedMsg{FormID: f.id}
				})
			}
		}
	}

	return f, tea.Batch(cmds...)
}

// View renders the form.
func (f *Form) View() string {
	var b strings.Builder

	// Title
	if f.title != "" {
		titleStyle := lipgloss.NewStyle().
			Foreground(styles.Foreground).
			Bold(true).
			Padding(0, 1)
		b.WriteString(titleStyle.Render(f.title))
		b.WriteString("\n\n")
	}

	// Render each field
	for i, field := range f.fields {
		b.WriteString("  ") // Indent
		b.WriteString(field.View())
		if i < len(f.fields)-1 {
			b.WriteString("\n")
		}
	}

	// Help text
	if f.showHelp {
		b.WriteString("\n\n")
		helpStyle := lipgloss.NewStyle().
			Foreground(styles.Muted)

		keyStyle := lipgloss.NewStyle().
			Foreground(styles.Secondary)

		help := keyStyle.Render("Tab") + helpStyle.Render(": next field") +
			helpStyle.Render(" │ ") +
			keyStyle.Render("Shift+Tab") + helpStyle.Render(": prev field") +
			helpStyle.Render(" │ ") +
			keyStyle.Render("Enter") + helpStyle.Render(": activate") +
			helpStyle.Render(" │ ") +
			keyStyle.Render("Esc") + helpStyle.Render(": cancel")

		b.WriteString("  " + help)
	}

	return b.String()
}
