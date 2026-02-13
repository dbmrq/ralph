package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wexinc/ralph/internal/task"
)

func TestNewTaskEditor(t *testing.T) {
	e := NewTaskEditor()
	if e == nil {
		t.Fatal("expected non-nil TaskEditor")
	}
	if e.mode != TaskEditorModeView {
		t.Errorf("expected mode TaskEditorModeView, got %v", e.mode)
	}
	if e.IsActive() {
		t.Error("expected editor to not be active initially")
	}
}

func TestTaskEditor_StartAdd(t *testing.T) {
	e := NewTaskEditor()

	e.StartAdd()

	if e.Mode() != TaskEditorModeAdd {
		t.Errorf("expected mode TaskEditorModeAdd, got %v", e.Mode())
	}
	if !e.IsActive() {
		t.Error("expected editor to be active after StartAdd")
	}
	if e.EditingTask() != nil {
		t.Error("expected EditingTask to be nil when adding")
	}
}

func TestTaskEditor_StartEdit(t *testing.T) {
	e := NewTaskEditor()
	testTask := task.NewTask("TEST-001", "Test Task", "Test Description")

	e.StartEdit(testTask)

	if e.Mode() != TaskEditorModeEdit {
		t.Errorf("expected mode TaskEditorModeEdit, got %v", e.Mode())
	}
	if !e.IsActive() {
		t.Error("expected editor to be active after StartEdit")
	}
	if e.EditingTask() != testTask {
		t.Error("expected EditingTask to match the provided task")
	}
	if e.Name() != "Test Task" {
		t.Errorf("expected name 'Test Task', got '%s'", e.Name())
	}
	if e.Description() != "Test Description" {
		t.Errorf("expected description 'Test Description', got '%s'", e.Description())
	}
}

func TestTaskEditor_Cancel(t *testing.T) {
	e := NewTaskEditor()

	e.StartAdd()
	if !e.IsActive() {
		t.Error("expected editor to be active")
	}

	e.Cancel()

	if e.IsActive() {
		t.Error("expected editor to not be active after Cancel")
	}
	if e.Mode() != TaskEditorModeView {
		t.Errorf("expected mode TaskEditorModeView, got %v", e.Mode())
	}
}

func TestTaskEditor_IsValid(t *testing.T) {
	e := NewTaskEditor()
	e.StartAdd()

	// Empty name should be invalid
	if e.IsValid() {
		t.Error("expected IsValid to be false with empty name")
	}

	// Set a name
	e.nameInput.SetValue("Valid Name")
	if !e.IsValid() {
		t.Error("expected IsValid to be true with non-empty name")
	}

	// Whitespace only should be invalid
	e.nameInput.SetValue("   ")
	if e.IsValid() {
		t.Error("expected IsValid to be false with whitespace-only name")
	}
}

func TestTaskEditor_SetSize(t *testing.T) {
	e := NewTaskEditor()
	e.SetSize(100, 10)

	if e.width != 100 {
		t.Errorf("expected width 100, got %d", e.width)
	}
	if e.height != 10 {
		t.Errorf("expected height 10, got %d", e.height)
	}
}

func TestTaskEditor_Update_EscCancels(t *testing.T) {
	e := NewTaskEditor()
	e.StartAdd()

	cmd := e.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if e.IsActive() {
		t.Error("expected editor to not be active after Esc")
	}

	// Should return a cancel message
	if cmd == nil {
		t.Fatal("expected a command to be returned")
	}
	msg := cmd()
	if _, ok := msg.(TaskEditorCancelMsg); !ok {
		t.Errorf("expected TaskEditorCancelMsg, got %T", msg)
	}
}

func TestTaskEditor_Update_NotActiveReturnsNil(t *testing.T) {
	e := NewTaskEditor()

	cmd := e.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if cmd != nil {
		t.Error("expected nil command when editor is not active")
	}
}

func TestTaskEditor_View_NotActiveReturnsEmpty(t *testing.T) {
	e := NewTaskEditor()

	view := e.View()

	if view != "" {
		t.Errorf("expected empty view when not active, got: %s", view)
	}
}

func TestTaskEditor_View_ShowsTitle(t *testing.T) {
	e := NewTaskEditor()

	e.StartAdd()
	view := e.View()
	if !strings.Contains(view, "Add New Task") {
		t.Errorf("expected view to contain 'Add New Task' when adding")
	}
}

