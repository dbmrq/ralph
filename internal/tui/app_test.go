package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wexinc/ralph/internal/task"
)

func TestNew(t *testing.T) {
	m := New()
	if m == nil {
		t.Fatal("New() returned nil")
	}

	if m.header == nil {
		t.Error("Model should have a header component")
	}
	if m.progress == nil {
		t.Error("Model should have a progress component")
	}
	if m.loopState != LoopStateIdle {
		t.Errorf("Initial loop state should be idle, got %v", m.loopState)
	}
}

func TestModelInit(t *testing.T) {
	m := New()
	cmd := m.Init()

	if cmd == nil {
		t.Error("Init should return a tick command")
	}
}

func TestModelUpdateKeyQuit(t *testing.T) {
	m := New()

	// Test 'q' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, cmd := m.Update(msg)
	model := newModel.(*Model)

	if !model.quitting {
		t.Error("Model should be quitting after 'q' press")
	}
	if cmd == nil {
		t.Error("Should return a quit command")
	}
}

func TestModelUpdateKeyCtrlC(t *testing.T) {
	m := New()

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, _ := m.Update(msg)
	model := newModel.(*Model)

	if !model.quitting {
		t.Error("Model should be quitting after Ctrl+C")
	}
}

func TestModelUpdateWindowSize(t *testing.T) {
	m := New()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := m.Update(msg)
	model := newModel.(*Model)

	if model.width != 120 {
		t.Errorf("Width should be 120, got %d", model.width)
	}
	if model.height != 40 {
		t.Errorf("Height should be 40, got %d", model.height)
	}
}

func TestModelUpdateTasksUpdated(t *testing.T) {
	m := New()

	tasks := []*task.Task{
		{ID: "TASK-001", Name: "Task 1", Status: task.StatusCompleted},
		{ID: "TASK-002", Name: "Task 2", Status: task.StatusPending},
	}

	msg := TasksUpdatedMsg{
		Tasks:     tasks,
		Completed: 1,
		Total:     2,
	}

	newModel, _ := m.Update(msg)
	model := newModel.(*Model)

	if len(model.tasks) != 2 {
		t.Errorf("Should have 2 tasks, got %d", len(model.tasks))
	}
}

func TestModelUpdateTaskStarted(t *testing.T) {
	m := New()
	m.tasks = []*task.Task{
		{ID: "TASK-001", Name: "Task 1"},
	}

	msg := TaskStartedMsg{
		TaskID:    "TASK-001",
		TaskName:  "Task 1",
		Iteration: 3,
	}

	newModel, _ := m.Update(msg)
	model := newModel.(*Model)

	if model.iteration != 3 {
		t.Errorf("Iteration should be 3, got %d", model.iteration)
	}
	if model.currentTask == nil {
		t.Error("Current task should be set")
	} else if model.currentTask.ID != "TASK-001" {
		t.Errorf("Current task ID should be 'TASK-001', got %s", model.currentTask.ID)
	}
}

func TestModelUpdateSessionInfo(t *testing.T) {
	m := New()

	msg := SessionInfoMsg{
		SessionID:   "session-123",
		ProjectName: "my-project",
		AgentName:   "auggie",
		ModelName:   "opus-4",
	}

	newModel, _ := m.Update(msg)
	model := newModel.(*Model)

	if model.sessionID != "session-123" {
		t.Errorf("SessionID should be 'session-123', got %s", model.sessionID)
	}
	if model.projectName != "my-project" {
		t.Errorf("ProjectName should be 'my-project', got %s", model.projectName)
	}
}

func TestModelUpdateLoopState(t *testing.T) {
	m := New()

	msg := LoopStateMsg{
		State:     "running",
		Iteration: 5,
	}

	newModel, _ := m.Update(msg)
	model := newModel.(*Model)

	if model.loopState != LoopStateRunning {
		t.Errorf("Loop state should be running, got %v", model.loopState)
	}
}

func TestModelUpdateError(t *testing.T) {
	m := New()

	msg := ErrorMsg{Error: "something went wrong"}
	newModel, _ := m.Update(msg)
	model := newModel.(*Model)

	if model.lastError != "something went wrong" {
		t.Errorf("lastError should be 'something went wrong', got %s", model.lastError)
	}
}

func TestModelUpdateQuit(t *testing.T) {
	m := New()

	msg := QuitMsg{Reason: "user requested"}
	newModel, cmd := m.Update(msg)
	model := newModel.(*Model)

	if !model.quitting {
		t.Error("Model should be quitting after QuitMsg")
	}
	if cmd == nil {
		t.Error("Should return a quit command")
	}
}

func TestModelViewWhenQuitting(t *testing.T) {
	m := New()
	m.quitting = true

	view := m.View()

	if !strings.Contains(view, "Goodbye") {
		t.Error("Quitting view should contain 'Goodbye'")
	}
}

func TestModelView(t *testing.T) {
	m := New()
	m.width = 80
	m.height = 24

	view := m.View()

	// Should contain header elements
	if !strings.Contains(view, "RALPH LOOP") {
		t.Error("View should contain 'RALPH LOOP' title")
	}

	// Should contain progress
	if !strings.Contains(view, "Progress:") {
		t.Error("View should contain progress bar")
	}

	// Should contain help hint
	if !strings.Contains(view, "quit") {
		t.Error("View should contain quit hint")
	}
	if !strings.Contains(view, "help") {
		t.Error("View should contain help hint")
	}
}

func TestModelSetTasks(t *testing.T) {
	m := New()

	tasks := []*task.Task{
		{ID: "1", Status: task.StatusCompleted},
		{ID: "2", Status: task.StatusCompleted},
		{ID: "3", Status: task.StatusPending},
	}

	m.SetTasks(tasks)

	if len(m.tasks) != 3 {
		t.Errorf("Should have 3 tasks, got %d", len(m.tasks))
	}
}

func TestModelSetSessionInfo(t *testing.T) {
	m := New()
	m.SetSessionInfo("project", "agent", "model", "session")

	if m.projectName != "project" {
		t.Errorf("ProjectName should be 'project', got %s", m.projectName)
	}
	if m.agentName != "agent" {
		t.Errorf("AgentName should be 'agent', got %s", m.agentName)
	}
	if m.modelName != "model" {
		t.Errorf("ModelName should be 'model', got %s", m.modelName)
	}
	if m.sessionID != "session" {
		t.Errorf("SessionID should be 'session', got %s", m.sessionID)
	}
}

func TestModelSetLoopState(t *testing.T) {
	m := New()
	m.SetLoopState(LoopStatePaused)

	if m.loopState != LoopStatePaused {
		t.Errorf("LoopState should be paused, got %v", m.loopState)
	}
}

func TestLoopStateConstants(t *testing.T) {
	states := []LoopState{
		LoopStateIdle,
		LoopStateRunning,
		LoopStatePaused,
		LoopStateAwaitingFix,
		LoopStateCompleted,
		LoopStateFailed,
	}

	for _, state := range states {
		if state == "" {
			t.Error("Loop state should not be empty")
		}
	}
}

func TestFormatDuration(t *testing.T) {
	// formatDuration is tested indirectly through the View method.
	// Here we verify the status bar renders without error.
	m := New()
	m.width = 80
	m.height = 24

	view := m.View()

	// Should contain "Time:" from the status bar (provided by StatusBar component)
	if !strings.Contains(view, "Time:") {
		t.Error("View should contain 'Time:' from status bar")
	}

	// The elapsed time should be in MM:SS format (at least)
	// Since we just created the model, it should show something like "00:00"
	if !strings.Contains(view, ":") {
		t.Error("Elapsed time should contain colon separator")
	}
}

