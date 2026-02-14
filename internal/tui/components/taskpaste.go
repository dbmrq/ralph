// Package components provides reusable TUI components for ralph.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wexinc/ralph/internal/task"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// TaskPasteSubmittedMsg is sent when user submits the pasted task list.
type TaskPasteSubmittedMsg struct {
	Content string
	Tasks   []*task.Task
}

// TaskPasteCanceledMsg is sent when user cancels paste input.
type TaskPasteCanceledMsg struct{}

// TaskPaste is a component for pasting task lists.
type TaskPaste struct {
	textarea    textarea.Model
	width       int
	height      int
	focused     bool
	previewTasks []*task.Task
	parseError  string
}

// NewTaskPaste creates a new TaskPaste component.
func NewTaskPaste() *TaskPaste {
	ta := textarea.New()
	ta.Placeholder = "Paste your task list here...\n\nSupported formats:\n- [ ] Task with checkbox\n- Task with dash\n1. Numbered task\n* Bulleted task"
	ta.CharLimit = 10000
	ta.SetWidth(60)
	ta.SetHeight(10)
	ta.ShowLineNumbers = false

	return &TaskPaste{
		textarea: ta,
	}
}

// SetWidth sets the component width.
func (t *TaskPaste) SetWidth(width int) {
	t.width = width
	t.textarea.SetWidth(width - 4)
}

// SetHeight sets the component height.
func (t *TaskPaste) SetHeight(height int) {
	t.height = height
	// Reserve space for title, help, preview
	textareaHeight := height - 15
	if textareaHeight < 5 {
		textareaHeight = 5
	}
	t.textarea.SetHeight(textareaHeight)
}

// Focus focuses the textarea.
func (t *TaskPaste) Focus() tea.Cmd {
	t.focused = true
	return t.textarea.Focus()
}

// Blur removes focus from the textarea.
func (t *TaskPaste) Blur() {
	t.focused = false
	t.textarea.Blur()
}

// Value returns the current textarea content.
func (t *TaskPaste) Value() string {
	return t.textarea.Value()
}

// SetValue sets the textarea content.
func (t *TaskPaste) SetValue(value string) {
	t.textarea.SetValue(value)
	t.updatePreview()
}

// PreviewTasks returns the currently parsed tasks.
func (t *TaskPaste) PreviewTasks() []*task.Task {
	return t.previewTasks
}

// ParseError returns any error from parsing.
func (t *TaskPaste) ParseError() string {
	return t.parseError
}

// updatePreview parses the current content and updates the preview.
func (t *TaskPaste) updatePreview() {
	content := t.textarea.Value()
	if strings.TrimSpace(content) == "" {
		t.previewTasks = nil
		t.parseError = ""
		return
	}

	importer := task.NewImporter()
	result, err := importer.ImportAuto(content)
	if err != nil {
		t.parseError = err.Error()
		t.previewTasks = nil
		return
	}

	t.parseError = ""
	t.previewTasks = result.Tasks
}

// Update handles messages for the component.
func (t *TaskPaste) Update(msg tea.Msg) (*TaskPaste, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+enter":
			// Submit with parsed tasks
			content := t.textarea.Value()
			tasks := t.previewTasks
			return t, func() tea.Msg {
				return TaskPasteSubmittedMsg{Content: content, Tasks: tasks}
			}
		case "esc":
			return t, func() tea.Msg {
				return TaskPasteCanceledMsg{}
			}
		}
	}

	var cmd tea.Cmd
	t.textarea, cmd = t.textarea.Update(msg)

	// Update preview after each change
	t.updatePreview()

	return t, cmd
}

// View renders the component.
func (t *TaskPaste) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Bold(true).
		Padding(0, 1)

	b.WriteString(titleStyle.Render("📋 Paste Task List"))
	b.WriteString("\n\n")

	// Textarea
	b.WriteString(t.textarea.View())
	b.WriteString("\n\n")

	// Preview section
	previewTitleStyle := lipgloss.NewStyle().
		Foreground(styles.Secondary).
		Bold(true)

	if t.parseError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(styles.Error)
		b.WriteString(errorStyle.Render("⚠ Parse error: " + t.parseError))
		b.WriteString("\n")
	} else if len(t.previewTasks) > 0 {
		b.WriteString(previewTitleStyle.Render("Preview: "))
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Success).
			Render(fmt.Sprintf("%d tasks detected", len(t.previewTasks))))
		b.WriteString("\n")

		// Show first few tasks
		mutedStyle := lipgloss.NewStyle().Foreground(styles.Muted)
		maxPreview := 3
		for i, tsk := range t.previewTasks {
			if i >= maxPreview {
				remaining := len(t.previewTasks) - maxPreview
				b.WriteString(mutedStyle.Render(fmt.Sprintf("  ... and %d more", remaining)))
				b.WriteString("\n")
				break
			}
			b.WriteString(mutedStyle.Render("  • " + tsk.Name))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Muted).
			Render("Start typing or paste tasks to see preview"))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	keyStyle := lipgloss.NewStyle().Foreground(styles.Secondary)
	b.WriteString(keyStyle.Render("Ctrl+Enter") + helpStyle.Render(": submit") +
		helpStyle.Render(" │ ") +
		keyStyle.Render("Esc") + helpStyle.Render(": cancel"))

	return b.String()
}

