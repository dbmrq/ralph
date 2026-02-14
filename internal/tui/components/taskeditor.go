// Package components provides reusable TUI components for ralph.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dbmrq/ralph/internal/task"
	"github.com/dbmrq/ralph/internal/tui/styles"
)

// TaskEditorMode represents the current editing mode.
type TaskEditorMode int

const (
	// TaskEditorModeView is for viewing tasks (not editing).
	TaskEditorModeView TaskEditorMode = iota
	// TaskEditorModeAdd is for adding a new task.
	TaskEditorModeAdd
	// TaskEditorModeEdit is for editing an existing task.
	TaskEditorModeEdit
)

// TaskEditor is a component for adding and editing tasks inline.
type TaskEditor struct {
	mode        TaskEditorMode
	nameInput   textinput.Model
	descInput   textinput.Model
	focusField  int // 0 = name, 1 = description
	editingTask *task.Task
	width       int
	height      int
}

// NewTaskEditor creates a new TaskEditor component.
func NewTaskEditor() *TaskEditor {
	nameInput := textinput.New()
	nameInput.Placeholder = "Task name"
	nameInput.CharLimit = 200
	nameInput.Width = 40

	descInput := textinput.New()
	descInput.Placeholder = "Task description (optional)"
	descInput.CharLimit = 500
	descInput.Width = 60

	return &TaskEditor{
		mode:       TaskEditorModeView,
		nameInput:  nameInput,
		descInput:  descInput,
		focusField: 0,
		width:      80,
		height:     6,
	}
}

// SetSize sets the editor dimensions.
func (e *TaskEditor) SetSize(width, height int) {
	e.width = width
	e.height = height
	e.nameInput.Width = width - 20
	e.descInput.Width = width - 20
}

// Mode returns the current editing mode.
func (e *TaskEditor) Mode() TaskEditorMode {
	return e.mode
}

// IsActive returns true if the editor is in add or edit mode.
func (e *TaskEditor) IsActive() bool {
	return e.mode == TaskEditorModeAdd || e.mode == TaskEditorModeEdit
}

// StartAdd begins adding a new task.
func (e *TaskEditor) StartAdd() {
	e.mode = TaskEditorModeAdd
	e.editingTask = nil
	e.nameInput.Reset()
	e.descInput.Reset()
	e.nameInput.Focus()
	e.focusField = 0
}

// StartEdit begins editing an existing task.
func (e *TaskEditor) StartEdit(t *task.Task) {
	e.mode = TaskEditorModeEdit
	e.editingTask = t
	e.nameInput.SetValue(t.Name)
	e.descInput.SetValue(t.Description)
	e.nameInput.Focus()
	e.focusField = 0
}

// Cancel cancels the current edit operation.
func (e *TaskEditor) Cancel() {
	e.mode = TaskEditorModeView
	e.editingTask = nil
	e.nameInput.Blur()
	e.descInput.Blur()
}

// Name returns the current task name value.
func (e *TaskEditor) Name() string {
	return strings.TrimSpace(e.nameInput.Value())
}

// Description returns the current task description value.
func (e *TaskEditor) Description() string {
	return strings.TrimSpace(e.descInput.Value())
}

// EditingTask returns the task being edited (nil if adding).
func (e *TaskEditor) EditingTask() *task.Task {
	return e.editingTask
}

// IsValid returns true if the current input is valid (name is non-empty).
func (e *TaskEditor) IsValid() bool {
	return e.Name() != ""
}

// nextField moves focus to the next field.
func (e *TaskEditor) nextField() {
	e.focusField = (e.focusField + 1) % 2
	e.updateFocus()
}

// prevField moves focus to the previous field.
func (e *TaskEditor) prevField() {
	e.focusField = (e.focusField + 1) % 2
	e.updateFocus()
}

// updateFocus updates input focus based on focusField.
func (e *TaskEditor) updateFocus() {
	if e.focusField == 0 {
		e.nameInput.Focus()
		e.descInput.Blur()
	} else {
		e.nameInput.Blur()
		e.descInput.Focus()
	}
}

// Update handles input messages.
func (e *TaskEditor) Update(msg tea.Msg) tea.Cmd {
	if e.mode == TaskEditorModeView {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			e.nextField()
			return nil
		case "shift+tab", "up":
			e.prevField()
			return nil
		case "enter":
			if e.IsValid() {
				name := e.Name()
				desc := e.Description()
				editingTask := e.editingTask
				mode := e.mode
				e.Cancel()
				return func() tea.Msg {
					return TaskEditorSubmitMsg{
						Mode:        mode,
						Name:        name,
						Description: desc,
						EditingTask: editingTask,
					}
				}
			}
			return nil
		case "esc":
			e.Cancel()
			return func() tea.Msg {
				return TaskEditorCancelMsg{}
			}
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	if e.focusField == 0 {
		e.nameInput, cmd = e.nameInput.Update(msg)
	} else {
		e.descInput, cmd = e.descInput.Update(msg)
	}
	return cmd
}

// View renders the task editor.
func (e *TaskEditor) View() string {
	if e.mode == TaskEditorModeView {
		return ""
	}

	var title string
	if e.mode == TaskEditorModeAdd {
		title = "Add New Task"
	} else {
		title = "Edit Task"
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Background(styles.Primary).
		Bold(true).
		Padding(0, 1)

	// Name field
	nameLabel := "Name: "
	if e.focusField == 0 {
		nameLabel = styles.FormLabelFocusedStyle.Render(nameLabel)
	} else {
		nameLabel = styles.FormLabelStyle.Render(nameLabel)
	}

	// Description field
	descLabel := "Description: "
	if e.focusField == 1 {
		descLabel = styles.FormLabelFocusedStyle.Render(descLabel)
	} else {
		descLabel = styles.FormLabelStyle.Render(descLabel)
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Italic(true)
	help := helpStyle.Render("Tab: next field  Enter: save  Esc: cancel")

	// Validation indicator
	validIndicator := ""
	if e.Name() == "" {
		validIndicator = styles.ErrorTextStyle.Render(" (name required)")
	}

	content := fmt.Sprintf("%s\n\n  %s%s%s\n  %s%s\n\n%s",
		titleStyle.Render(title),
		nameLabel, e.nameInput.View(), validIndicator,
		descLabel, e.descInput.View(),
		help,
	)

	boxStyle := styles.FocusedBoxStyle.Width(e.width - 2)
	return boxStyle.Render(content)
}

// TaskEditorSubmitMsg is sent when the user submits the editor.
type TaskEditorSubmitMsg struct {
	Mode        TaskEditorMode
	Name        string
	Description string
	EditingTask *task.Task // Non-nil if editing
}

// TaskEditorCancelMsg is sent when the user cancels the editor.
type TaskEditorCancelMsg struct{}
