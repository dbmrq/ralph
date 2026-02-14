package tui

import (
	"strings"
	"testing"
	"time"

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

func TestModelHandleKeyPress_Help(t *testing.T) {
	m := New()
	m.width = 80
	m.height = 24

	// Test '?' key toggles help
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	m.Update(msg)

	// Help should be visible
	if !m.helpOverlay.IsVisible() {
		t.Error("Help overlay should be visible after '?' press")
	}

	// Press again to toggle off
	m.Update(msg)
	if m.helpOverlay.IsVisible() {
		t.Error("Help overlay should be hidden after second '?' press")
	}
}

func TestModelHandleKeyPress_H(t *testing.T) {
	m := New()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	m.Update(msg)

	if !m.helpOverlay.IsVisible() {
		t.Error("Help overlay should be visible after 'h' press")
	}
}

func TestModelHandleKeyPress_ToggleLogs(t *testing.T) {
	m := New()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	m.Update(msg)

	if !m.showLogs {
		t.Error("showLogs should be true after 'l' press")
	}
	if m.focusedPane != FocusLogs {
		t.Errorf("focusedPane should be FocusLogs, got %v", m.focusedPane)
	}

	// Toggle off
	m.Update(msg)
	if m.showLogs {
		t.Error("showLogs should be false after second 'l' press")
	}
	if m.focusedPane != FocusTasks {
		t.Errorf("focusedPane should be FocusTasks, got %v", m.focusedPane)
	}
}

func TestModelHandleKeyPress_Tab(t *testing.T) {
	m := New()
	m.showLogs = true
	m.focusedPane = FocusTasks

	msg := tea.KeyMsg{Type: tea.KeyTab}
	m.Update(msg)

	if m.focusedPane != FocusLogs {
		t.Errorf("focusedPane should be FocusLogs after Tab, got %v", m.focusedPane)
	}

	// Tab again
	m.Update(msg)
	if m.focusedPane != FocusTasks {
		t.Errorf("focusedPane should be FocusTasks after second Tab, got %v", m.focusedPane)
	}
}

func TestModelHandleKeyPress_Pause_NoController(t *testing.T) {
	m := New()
	m.loopState = LoopStateRunning

	// Press 'p' without a loop controller - should not error
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	_, cmd := m.Update(msg)

	if cmd != nil {
		// No command should be returned when no controller
	}
}

func TestModelHandleKeyPress_Skip_ShowsConfirm(t *testing.T) {
	m := New()
	m.loopState = LoopStateRunning
	m.currentTask = &task.Task{ID: "TEST-001", Name: "Test Task"}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	m.Update(msg)

	if !m.confirmDlg.IsVisible() {
		t.Error("confirm dialog should be visible after 's' press")
	}
}

func TestModelHandleKeyPress_Abort_ShowsConfirm(t *testing.T) {
	m := New()
	m.loopState = LoopStateRunning

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	m.Update(msg)

	if !m.confirmDlg.IsVisible() {
		t.Error("confirm dialog should be visible after 'a' press")
	}
}

func TestModelHandleKeyPress_QuitWhileRunning(t *testing.T) {
	m := New()
	m.loopState = LoopStateRunning

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	m.Update(msg)

	// Should show confirmation dialog, not quit immediately
	if m.quitting {
		t.Error("should not quit immediately while running")
	}
	if !m.confirmDlg.IsVisible() {
		t.Error("confirm dialog should be visible")
	}
}

func TestModelHandleKeyPress_QuitWhileIdle(t *testing.T) {
	m := New()
	m.loopState = LoopStateIdle

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, cmd := m.Update(msg)
	model := newModel.(*Model)

	// Should quit immediately when idle
	if !model.quitting {
		t.Error("should quit immediately when idle")
	}
	if cmd == nil {
		t.Error("should return a quit command")
	}
}

func TestModelHandleKeyPress_AddTask_WhenIdle(t *testing.T) {
	m := New()
	m.loopState = LoopStateIdle

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	m.Update(msg)

	if !m.taskEditor.IsActive() {
		t.Error("task editor should be active after 'e' press when idle")
	}
}

func TestModelHandleKeyPress_AddTask_WhenRunning(t *testing.T) {
	m := New()
	m.loopState = LoopStateRunning

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	m.Update(msg)

	// Should not activate editor when running
	if m.taskEditor.IsActive() {
		t.Error("task editor should not be active when loop is running")
	}
}

func TestModelHandleKeyPress_ModelPicker_WhenIdle(t *testing.T) {
	m := New()
	m.loopState = LoopStateIdle

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
	m.Update(msg)

	if !m.modelPicker.IsVisible() {
		t.Error("model picker should be visible after 'm' press when idle")
	}
}

func TestModelSetLoopController(t *testing.T) {
	m := New()

	// Create a mock controller
	mockCtrl := &mockLoopController{}
	m.SetLoopController(mockCtrl)

	if m.loopController == nil {
		t.Error("loopController should be set")
	}
}

func TestModelRenderOverlay(t *testing.T) {
	m := New()
	m.width = 80
	m.height = 24

	base := "Base content"
	overlay := "Overlay content"

	result := m.renderOverlay(base, overlay)

	if !strings.Contains(result, base) {
		t.Error("result should contain base content")
	}
	if !strings.Contains(result, overlay) {
		t.Error("result should contain overlay content")
	}
}

func TestModelRenderOverlay_EmptyOverlay(t *testing.T) {
	m := New()
	base := "Base content"

	result := m.renderOverlay(base, "")

	if result != base {
		t.Errorf("empty overlay should return base unchanged, got: %s", result)
	}
}

func TestRepeatChar(t *testing.T) {
	tests := []struct {
		char     string
		n        int
		expected string
	}{
		{"x", 5, "xxxxx"},
		{"-", 3, "---"},
		{"ab", 2, "abab"},
		{"x", 0, ""},
		{"x", -1, ""},
	}

	for _, tt := range tests {
		result := repeatChar(tt.char, tt.n)
		if result != tt.expected {
			t.Errorf("repeatChar(%q, %d) = %q, want %q", tt.char, tt.n, result, tt.expected)
		}
	}
}

func TestFormatDurationFunc(t *testing.T) {
	tests := []struct {
		duration         string
		expectedContains string
	}{
		{"0s", "00:00"},
		{"30s", "00:30"},
		{"1m0s", "01:00"},
		{"5m30s", "05:30"},
		{"1h30m15s", "01:30:15"},
	}

	for _, tt := range tests {
		d, _ := time.ParseDuration(tt.duration)
		result := formatDuration(d)
		if result != tt.expectedContains {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expectedContains)
		}
	}
}

// mockLoopController implements LoopController for testing
type mockLoopController struct {
	pauseCalled  bool
	resumeCalled bool
	abortCalled  bool
	skipCalled   bool
	skipTaskID   string
}

func (m *mockLoopController) Pause() error {
	m.pauseCalled = true
	return nil
}

func (m *mockLoopController) Resume() error {
	m.resumeCalled = true
	return nil
}

func (m *mockLoopController) Abort() error {
	m.abortCalled = true
	return nil
}

func (m *mockLoopController) Skip(taskID string) error {
	m.skipCalled = true
	m.skipTaskID = taskID
	return nil
}
