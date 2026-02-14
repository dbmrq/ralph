// Package components provides reusable TUI components for ralph.
package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wexinc/ralph/internal/task"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// TaskListConfirmedMsg is sent when the user confirms the task list.
type TaskListConfirmedMsg struct {
	Tasks []*task.Task
}

// TaskListReparseMsg is sent when the user wants to re-parse.
type TaskListReparseMsg struct{}

// TaskListForm displays parsed tasks for user confirmation.
type TaskListForm struct {
	tasks        []*task.Task
	selectedIdx  int
	scrollOffset int
	visibleRows  int
	width        int
	editMode     bool
	editInput    *TextInput

	confirmBtn *Button
	reparseBtn *Button
	focusIndex int // 0 = task list, 1 = confirm, 2 = reparse
}

// NewTaskListForm creates a new TaskListForm.
func NewTaskListForm() *TaskListForm {
	f := &TaskListForm{
		tasks:       []*task.Task{},
		visibleRows: 10,
		editInput:   NewTextInput("edit", "Edit Task"),
	}

	f.confirmBtn = NewButton("confirm", "Confirm & Save")
	f.confirmBtn.SetStyle(ButtonStylePrimary)

	f.reparseBtn = NewButton("reparse", "Re-parse")
	f.reparseBtn.SetStyle(ButtonStyleSecondary)

	return f
}

// SetTasks sets the tasks to display.
func (f *TaskListForm) SetTasks(tasks []*task.Task) {
	f.tasks = tasks
	f.selectedIdx = 0
	f.scrollOffset = 0
}

// Tasks returns the current tasks (possibly modified).
func (f *TaskListForm) Tasks() []*task.Task {
	return f.tasks
}

// SetWidth sets the form width.
func (f *TaskListForm) SetWidth(width int) {
	f.width = width
	f.editInput.SetWidth(width - 10)
}

// SetVisibleRows sets the number of visible task rows.
func (f *TaskListForm) SetVisibleRows(rows int) {
	f.visibleRows = rows
}

// Focus focuses the form.
func (f *TaskListForm) Focus() tea.Cmd {
	f.focusIndex = 0
	return nil
}

// Blur blurs the form.
func (f *TaskListForm) Blur() {
	f.confirmBtn.Blur()
	f.reparseBtn.Blur()
}

// Update handles messages.
func (f *TaskListForm) Update(msg tea.Msg) (*TaskListForm, tea.Cmd) {
	var cmds []tea.Cmd

	if f.editMode {
		return f.updateEditMode(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if f.focusIndex == 0 {
				if f.selectedIdx > 0 {
					f.selectedIdx--
					f.ensureVisible()
				}
			}
		case "down", "j":
			if f.focusIndex == 0 {
				if f.selectedIdx < len(f.tasks)-1 {
					f.selectedIdx++
					f.ensureVisible()
				}
			}
		case "tab":
			f.focusIndex = (f.focusIndex + 1) % 3
			f.updateButtonFocus()
		case "shift+tab":
			f.focusIndex--
			if f.focusIndex < 0 {
				f.focusIndex = 2
			}
			f.updateButtonFocus()
		case "e":
			// Edit selected task
			if f.focusIndex == 0 && len(f.tasks) > 0 {
				f.editMode = true
				f.editInput.SetValue(f.tasks[f.selectedIdx].Name)
				cmds = append(cmds, f.editInput.Focus())
			}
		case "d":
			// Delete selected task
			if f.focusIndex == 0 && len(f.tasks) > 0 {
				f.tasks = append(f.tasks[:f.selectedIdx], f.tasks[f.selectedIdx+1:]...)
				if f.selectedIdx >= len(f.tasks) && f.selectedIdx > 0 {
					f.selectedIdx--
				}
			}
		case "enter":
			if f.focusIndex == 1 {
				return f, func() tea.Msg {
					return TaskListConfirmedMsg{Tasks: f.tasks}
				}
			} else if f.focusIndex == 2 {
				return f, func() tea.Msg {
					return TaskListReparseMsg{}
				}
			}
		}
	}

	return f, tea.Batch(cmds...)
}

// updateEditMode handles input in edit mode.
func (f *TaskListForm) updateEditMode(msg tea.Msg) (*TaskListForm, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			f.tasks[f.selectedIdx].Name = f.editInput.Value()
			f.editMode = false
			return f, nil
		case "esc":
			f.editMode = false
			return f, nil
		}
	}

	updated, cmd := f.editInput.Update(msg)
	f.editInput = updated
	return f, cmd
}

// ensureVisible adjusts scroll offset to keep selected task visible.
func (f *TaskListForm) ensureVisible() {
	if f.selectedIdx < f.scrollOffset {
		f.scrollOffset = f.selectedIdx
	} else if f.selectedIdx >= f.scrollOffset+f.visibleRows {
		f.scrollOffset = f.selectedIdx - f.visibleRows + 1
	}
}

// updateButtonFocus updates button focus state.
func (f *TaskListForm) updateButtonFocus() {
	f.confirmBtn.Blur()
	f.reparseBtn.Blur()
	if f.focusIndex == 1 {
		f.confirmBtn.Focus()
	} else if f.focusIndex == 2 {
		f.reparseBtn.Focus()
	}
}

// View renders the form.
func (f *TaskListForm) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Bold(true).
		Padding(0, 1)

	b.WriteString(titleStyle.Render(fmt.Sprintf("📋 Task List (%d tasks)", len(f.tasks))))
	b.WriteString("\n\n")

	if len(f.tasks) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(styles.Muted).
			PaddingLeft(2)
		b.WriteString(emptyStyle.Render("No tasks. Press 'a' to add a task."))
		b.WriteString("\n\n")
	} else {
		// Task list
		listStyle := lipgloss.NewStyle().PaddingLeft(2)
		b.WriteString(listStyle.Render(f.renderTaskList()))
		b.WriteString("\n")
	}

	// Edit mode
	if f.editMode {
		b.WriteString("\n  ")
		b.WriteString(f.editInput.View())
		b.WriteString("\n")
	}

	// Buttons
	b.WriteString("\n  ")
	b.WriteString(f.confirmBtn.View())
	b.WriteString("    ")
	b.WriteString(f.reparseBtn.View())
	b.WriteString("\n")

	// Shortcut bar
	b.WriteString("\n  ")
	shortcutBar := NewShortcutBar(TaskListShortcuts...)
	b.WriteString(shortcutBar.View())

	return b.String()
}

// renderTaskList renders the scrollable task list.
func (f *TaskListForm) renderTaskList() string {
	var lines []string

	start := f.scrollOffset
	end := start + f.visibleRows
	if end > len(f.tasks) {
		end = len(f.tasks)
	}

	for i := start; i < end; i++ {
		t := f.tasks[i]
		prefix := "  "
		if i == f.selectedIdx && f.focusIndex == 0 {
			prefix = "▶ "
		}

		// Status icon
		var icon string
		switch t.Status {
		case task.StatusCompleted:
			icon = "✓"
		case task.StatusInProgress:
			icon = "→"
		case task.StatusFailed:
			icon = "✗"
		case task.StatusSkipped:
			icon = "⊘"
		case task.StatusPaused:
			icon = "⏸"
		default:
			icon = "○"
		}

		var lineStyle lipgloss.Style
		if i == f.selectedIdx && f.focusIndex == 0 {
			lineStyle = lipgloss.NewStyle().
				Foreground(styles.Primary).
				Bold(true)
		} else {
			lineStyle = lipgloss.NewStyle().
				Foreground(styles.Foreground)
		}

		line := fmt.Sprintf("%s%s %s: %s", prefix, icon, t.ID, t.Name)
		lines = append(lines, lineStyle.Render(line))
	}

	// Scroll indicator
	if len(f.tasks) > f.visibleRows {
		indicator := fmt.Sprintf("... (%d more)", len(f.tasks)-f.visibleRows)
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.Muted).Render(indicator))
	}

	return strings.Join(lines, "\n")
}
