package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wexinc/ralph/internal/task"
)

func TestNewTaskListForm(t *testing.T) {
	f := NewTaskListForm()
	if f == nil {
		t.Fatal("expected non-nil TaskListForm")
	}
	if len(f.tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(f.tasks))
	}
	if f.visibleRows != 10 {
		t.Errorf("expected visibleRows 10, got %d", f.visibleRows)
	}
	if f.confirmBtn == nil {
		t.Error("expected confirmBtn to be set")
	}
	if f.reparseBtn == nil {
		t.Error("expected reparseBtn to be set")
	}
}

func TestTaskListForm_SetTasks(t *testing.T) {
	f := NewTaskListForm()

	tasks := []*task.Task{
		task.NewTask("TASK-001", "First task", "Description 1"),
		task.NewTask("TASK-002", "Second task", "Description 2"),
		task.NewTask("TASK-003", "Third task", "Description 3"),
	}

	f.SetTasks(tasks)

	if len(f.tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(f.tasks))
	}
	if f.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0, got %d", f.selectedIdx)
	}
	if f.scrollOffset != 0 {
		t.Errorf("expected scrollOffset 0, got %d", f.scrollOffset)
	}
}

func TestTaskListForm_Tasks(t *testing.T) {
	f := NewTaskListForm()
	tasks := []*task.Task{
		task.NewTask("TASK-001", "First task", "Desc"),
	}
	f.SetTasks(tasks)

	retrieved := f.Tasks()
	if len(retrieved) != 1 {
		t.Errorf("expected 1 task, got %d", len(retrieved))
	}
	if retrieved[0].ID != "TASK-001" {
		t.Errorf("expected task ID 'TASK-001', got %s", retrieved[0].ID)
	}
}

func TestTaskListForm_SetWidth(t *testing.T) {
	f := NewTaskListForm()
	f.SetWidth(120)

	if f.width != 120 {
		t.Errorf("expected width 120, got %d", f.width)
	}
}

func TestTaskListForm_SetVisibleRows(t *testing.T) {
	f := NewTaskListForm()
	f.SetVisibleRows(15)

	if f.visibleRows != 15 {
		t.Errorf("expected visibleRows 15, got %d", f.visibleRows)
	}
}

func TestTaskListForm_Focus(t *testing.T) {
	f := NewTaskListForm()
	f.focusIndex = 1

	f.Focus()

	if f.focusIndex != 0 {
		t.Errorf("expected focusIndex 0 after Focus, got %d", f.focusIndex)
	}
}

func TestTaskListForm_Blur(t *testing.T) {
	f := NewTaskListForm()
	f.confirmBtn.Focus()
	f.reparseBtn.Focus()

	f.Blur()

	// Verify buttons are blurred (they won't be focused)
	if f.confirmBtn.Focused() {
		t.Error("expected confirmBtn to be blurred")
	}
	if f.reparseBtn.Focused() {
		t.Error("expected reparseBtn to be blurred")
	}
}

func TestTaskListForm_Update_Navigation(t *testing.T) {
	f := NewTaskListForm()
	tasks := []*task.Task{
		task.NewTask("TASK-001", "First", ""),
		task.NewTask("TASK-002", "Second", ""),
		task.NewTask("TASK-003", "Third", ""),
	}
	f.SetTasks(tasks)
	f.focusIndex = 0 // task list focused

	// Navigate down with j
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if f.selectedIdx != 1 {
		t.Errorf("expected selectedIdx 1 after j, got %d", f.selectedIdx)
	}

	// Navigate down with down arrow
	f.Update(tea.KeyMsg{Type: tea.KeyDown})
	if f.selectedIdx != 2 {
		t.Errorf("expected selectedIdx 2 after down, got %d", f.selectedIdx)
	}

	// Navigate up with k
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if f.selectedIdx != 1 {
		t.Errorf("expected selectedIdx 1 after k, got %d", f.selectedIdx)
	}

	// Navigate up with up arrow
	f.Update(tea.KeyMsg{Type: tea.KeyUp})
	if f.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0 after up, got %d", f.selectedIdx)
	}
}

func TestTaskListForm_Update_TabNavigation(t *testing.T) {
	f := NewTaskListForm()
	f.focusIndex = 0

	// Tab should cycle through focus: tasklist -> confirm -> reparse -> tasklist
	f.Update(tea.KeyMsg{Type: tea.KeyTab})
	if f.focusIndex != 1 {
		t.Errorf("expected focusIndex 1 after Tab, got %d", f.focusIndex)
	}

	f.Update(tea.KeyMsg{Type: tea.KeyTab})
	if f.focusIndex != 2 {
		t.Errorf("expected focusIndex 2 after Tab, got %d", f.focusIndex)
	}

	f.Update(tea.KeyMsg{Type: tea.KeyTab})
	if f.focusIndex != 0 {
		t.Errorf("expected focusIndex 0 after Tab (wrap), got %d", f.focusIndex)
	}

	// Shift+Tab should go backwards
	f.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if f.focusIndex != 2 {
		t.Errorf("expected focusIndex 2 after Shift+Tab, got %d", f.focusIndex)
	}
}

func TestTaskListForm_Update_Delete(t *testing.T) {
	f := NewTaskListForm()
	tasks := []*task.Task{
		task.NewTask("TASK-001", "First", ""),
		task.NewTask("TASK-002", "Second", ""),
		task.NewTask("TASK-003", "Third", ""),
	}
	f.SetTasks(tasks)
	f.focusIndex = 0
	f.selectedIdx = 1 // Select "Second"

	// Delete with 'd'
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	if len(f.tasks) != 2 {
		t.Errorf("expected 2 tasks after delete, got %d", len(f.tasks))
	}
	// Verify "Second" was removed
	for _, tsk := range f.tasks {
		if tsk.Name == "Second" {
			t.Error("expected 'Second' task to be deleted")
		}
	}
}

func TestTaskListForm_Update_ConfirmButton(t *testing.T) {
	f := NewTaskListForm()
	tasks := []*task.Task{
		task.NewTask("TASK-001", "First", ""),
	}
	f.SetTasks(tasks)
	f.focusIndex = 1 // confirm button

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("expected command from Enter on confirm button")
	}

	msg := cmd()
	confirmed, ok := msg.(TaskListConfirmedMsg)
	if !ok {
		t.Fatalf("expected TaskListConfirmedMsg, got %T", msg)
	}
	if len(confirmed.Tasks) != 1 {
		t.Errorf("expected 1 task in confirmed message, got %d", len(confirmed.Tasks))
	}
}

func TestTaskListForm_Update_ReparseButton(t *testing.T) {
	f := NewTaskListForm()
	f.focusIndex = 2 // reparse button

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("expected command from Enter on reparse button")
	}

	msg := cmd()
	if _, ok := msg.(TaskListReparseMsg); !ok {
		t.Fatalf("expected TaskListReparseMsg, got %T", msg)
	}
}

func TestTaskListForm_View_Empty(t *testing.T) {
	f := NewTaskListForm()

	view := f.View()

	if !strings.Contains(view, "0 tasks") {
		t.Errorf("expected '0 tasks' in view, got: %s", view)
	}
	if !strings.Contains(view, "No tasks") {
		t.Errorf("expected 'No tasks' message in view, got: %s", view)
	}
}

func TestTaskListForm_View_WithTasks(t *testing.T) {
	f := NewTaskListForm()
	tasks := []*task.Task{
		task.NewTask("TASK-001", "First task", ""),
		task.NewTask("TASK-002", "Second task", ""),
	}
	f.SetTasks(tasks)

	view := f.View()

	if !strings.Contains(view, "2 tasks") {
		t.Errorf("expected '2 tasks' in view, got: %s", view)
	}
	if !strings.Contains(view, "First task") {
		t.Errorf("expected 'First task' in view")
	}
	if !strings.Contains(view, "TASK-001") {
		t.Errorf("expected 'TASK-001' in view")
	}
	// Check for buttons
	if !strings.Contains(view, "Confirm") {
		t.Errorf("expected 'Confirm' button in view")
	}
}

func TestTaskListForm_EnsureVisible(t *testing.T) {
	f := NewTaskListForm()
	f.visibleRows = 3

	tasks := make([]*task.Task, 10)
	for i := 0; i < 10; i++ {
		tasks[i] = task.NewTask("TASK", "Task", "")
	}
	f.SetTasks(tasks)

	// Select item beyond visible range
	f.selectedIdx = 5
	f.ensureVisible()

	if f.scrollOffset != 3 {
		t.Errorf("expected scrollOffset 3, got %d", f.scrollOffset)
	}

	// Select item before scroll range
	f.selectedIdx = 1
	f.ensureVisible()

	if f.scrollOffset != 1 {
		t.Errorf("expected scrollOffset 1, got %d", f.scrollOffset)
	}
}

