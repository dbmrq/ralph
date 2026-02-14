package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dbmrq/ralph/internal/task"
)

func TestNewTaskList(t *testing.T) {
	tl := NewTaskList()
	if tl == nil {
		t.Fatal("expected non-nil TaskList")
	}
	if len(tl.items) != 0 {
		t.Errorf("expected empty items, got %d", len(tl.items))
	}
	if tl.selected != 0 {
		t.Errorf("expected selected 0, got %d", tl.selected)
	}
	if !tl.focused {
		t.Error("expected focused to be true by default")
	}
}

func TestTaskList_SetItems(t *testing.T) {
	tl := NewTaskList()

	items := []TaskListItem{
		{ID: "TASK-001", Name: "First task", Status: task.StatusPending},
		{ID: "TASK-002", Name: "Second task", Status: task.StatusCompleted},
		{ID: "TASK-003", Name: "Third task", Status: task.StatusInProgress},
	}

	tl.SetItems(items)

	if len(tl.items) != 3 {
		t.Errorf("expected 3 items, got %d", len(tl.items))
	}
}

func TestTaskList_SetTasks(t *testing.T) {
	tl := NewTaskList()

	tasks := []*task.Task{
		task.NewTask("TASK-001", "First task", "Description 1"),
		task.NewTask("TASK-002", "Second task", "Description 2"),
	}
	tasks[0].Status = task.StatusInProgress
	tasks[0].StartIteration()

	tl.SetTasks(tasks, "TASK-001")

	if len(tl.items) != 2 {
		t.Errorf("expected 2 items, got %d", len(tl.items))
	}

	// Check that current task is marked
	if !tl.items[0].IsCurrent {
		t.Error("expected first task to be marked as current")
	}
	if tl.items[1].IsCurrent {
		t.Error("expected second task to not be marked as current")
	}
}

func TestTaskList_Navigation(t *testing.T) {
	tl := NewTaskList()
	items := []TaskListItem{
		{ID: "TASK-001", Name: "First"},
		{ID: "TASK-002", Name: "Second"},
		{ID: "TASK-003", Name: "Third"},
	}
	tl.SetItems(items)

	t.Run("MoveDown", func(t *testing.T) {
		tl.selected = 0
		tl.MoveDown()
		if tl.selected != 1 {
			t.Errorf("expected selected 1, got %d", tl.selected)
		}
	})

	t.Run("MoveUp", func(t *testing.T) {
		tl.selected = 2
		tl.MoveUp()
		if tl.selected != 1 {
			t.Errorf("expected selected 1, got %d", tl.selected)
		}
	})

	t.Run("MoveUp at start", func(t *testing.T) {
		tl.selected = 0
		tl.MoveUp()
		if tl.selected != 0 {
			t.Errorf("expected selected 0 (no change), got %d", tl.selected)
		}
	})

	t.Run("MoveDown at end", func(t *testing.T) {
		tl.selected = 2
		tl.MoveDown()
		if tl.selected != 2 {
			t.Errorf("expected selected 2 (no change), got %d", tl.selected)
		}
	})
}

func TestTaskList_Update(t *testing.T) {
	tl := NewTaskList()
	items := []TaskListItem{
		{ID: "TASK-001", Name: "First"},
		{ID: "TASK-002", Name: "Second"},
		{ID: "TASK-003", Name: "Third"},
	}
	tl.SetItems(items)

	tests := []struct {
		name        string
		key         string
		expectedSel int
		initialSel  int
	}{
		{"j moves down", "j", 1, 0},
		{"k moves up", "k", 1, 2},
		{"down arrow", "down", 1, 0},
		{"up arrow", "up", 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl.selected = tt.initialSel
			tl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			if tl.selected != tt.expectedSel {
				t.Errorf("expected selected %d, got %d", tt.expectedSel, tl.selected)
			}
		})
	}
}

func TestTaskList_View(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		tl := NewTaskList()
		view := tl.View()
		if !strings.Contains(view, "No tasks") {
			t.Errorf("expected 'No tasks' message, got: %s", view)
		}
	})

	t.Run("with items", func(t *testing.T) {
		tl := NewTaskList()
		items := []TaskListItem{
			{ID: "TASK-001", Name: "First task", Status: task.StatusCompleted},
			{ID: "TASK-002", Name: "Second task", Status: task.StatusInProgress, IsCurrent: true},
		}
		tl.SetItems(items)

		view := tl.View()

		// Should contain task names
		if !strings.Contains(view, "First task") {
			t.Errorf("expected 'First task' in view, got: %s", view)
		}
		if !strings.Contains(view, "Second task") {
			t.Errorf("expected 'Second task' in view, got: %s", view)
		}
	})
}

func TestTaskList_SetHeight(t *testing.T) {
	tl := NewTaskList()
	tl.SetHeight(20)

	if tl.height != 20 {
		t.Errorf("expected height 20, got %d", tl.height)
	}
}

func TestTaskList_SetWidth(t *testing.T) {
	tl := NewTaskList()
	tl.SetWidth(100)

	if tl.width != 100 {
		t.Errorf("expected width 100, got %d", tl.width)
	}
}

func TestTaskList_SetFocused(t *testing.T) {
	tl := NewTaskList()

	tl.SetFocused(false)
	if tl.focused {
		t.Error("expected focused to be false")
	}

	tl.SetFocused(true)
	if !tl.focused {
		t.Error("expected focused to be true")
	}
}

func TestTaskList_Selected(t *testing.T) {
	tl := NewTaskList()
	items := []TaskListItem{
		{ID: "TASK-001", Name: "First"},
		{ID: "TASK-002", Name: "Second"},
	}
	tl.SetItems(items)
	tl.selected = 1

	if tl.Selected() != 1 {
		t.Errorf("expected Selected() to return 1, got %d", tl.Selected())
	}
}

func TestTaskList_SelectedItem(t *testing.T) {
	tl := NewTaskList()
	items := []TaskListItem{
		{ID: "TASK-001", Name: "First"},
		{ID: "TASK-002", Name: "Second"},
	}
	tl.SetItems(items)
	tl.selected = 1

	selected := tl.SelectedItem()
	if selected == nil {
		t.Fatal("expected SelectedItem to return non-nil")
	}
	if selected.ID != "TASK-002" {
		t.Errorf("expected selected item ID 'TASK-002', got %s", selected.ID)
	}
}

func TestTaskList_SelectedItem_Empty(t *testing.T) {
	tl := NewTaskList()
	selected := tl.SelectedItem()
	if selected != nil {
		t.Error("expected SelectedItem to return nil for empty list")
	}
}

func TestTaskList_GoToTop(t *testing.T) {
	tl := NewTaskList()
	items := []TaskListItem{
		{ID: "TASK-001", Name: "First"},
		{ID: "TASK-002", Name: "Second"},
		{ID: "TASK-003", Name: "Third"},
	}
	tl.SetItems(items)
	tl.selected = 2

	tl.GoToTop()

	if tl.selected != 0 {
		t.Errorf("expected selected 0 after GoToTop, got %d", tl.selected)
	}
}

func TestTaskList_GoToBottom(t *testing.T) {
	tl := NewTaskList()
	items := []TaskListItem{
		{ID: "TASK-001", Name: "First"},
		{ID: "TASK-002", Name: "Second"},
		{ID: "TASK-003", Name: "Third"},
	}
	tl.SetItems(items)
	tl.selected = 0

	tl.GoToBottom()

	if tl.selected != 2 {
		t.Errorf("expected selected 2 after GoToBottom, got %d", tl.selected)
	}
}

func TestTaskList_SetSize(t *testing.T) {
	tl := NewTaskList()
	tl.SetSize(80, 24)

	if tl.width != 80 {
		t.Errorf("expected width 80, got %d", tl.width)
	}
	if tl.height != 24 {
		t.Errorf("expected height 24, got %d", tl.height)
	}
}

func TestTaskList_SetSelected(t *testing.T) {
	tl := NewTaskList()
	items := []TaskListItem{
		{ID: "TASK-001", Name: "First"},
		{ID: "TASK-002", Name: "Second"},
	}
	tl.SetItems(items)

	tl.SetSelected(1)
	if tl.selected != 1 {
		t.Errorf("expected selected 1, got %d", tl.selected)
	}

	// Test out of bounds - should clamp
	tl.SetSelected(10)
	if tl.selected != 1 {
		t.Errorf("expected selected to remain 1, got %d", tl.selected)
	}
}

func TestTaskList_StatusIcons(t *testing.T) {
	tl := NewTaskList()

	// Test different status icons by rendering view
	items := []TaskListItem{
		{ID: "TASK-001", Name: "Completed", Status: task.StatusCompleted},
		{ID: "TASK-002", Name: "In Progress", Status: task.StatusInProgress},
		{ID: "TASK-003", Name: "Pending", Status: task.StatusPending},
		{ID: "TASK-004", Name: "Skipped", Status: task.StatusSkipped},
		{ID: "TASK-005", Name: "Failed", Status: task.StatusFailed},
		{ID: "TASK-006", Name: "Paused", Status: task.StatusPaused},
	}
	tl.SetItems(items)
	tl.SetHeight(10)

	view := tl.View()

	// Verify the view contains the task names
	if !strings.Contains(view, "Completed") {
		t.Error("expected 'Completed' task in view")
	}
	if !strings.Contains(view, "Pending") {
		t.Error("expected 'Pending' task in view")
	}
}
