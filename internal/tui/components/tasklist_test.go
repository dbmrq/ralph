package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wexinc/ralph/internal/task"
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
		name         string
		key          string
		expectedSel  int
		initialSel   int
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

