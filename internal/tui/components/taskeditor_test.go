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

func TestTaskEditor_View_EditMode(t *testing.T) {
	e := NewTaskEditor()
	testTask := task.NewTask("TEST-001", "Test Task", "Test Description")

	e.StartEdit(testTask)
	view := e.View()

	if !strings.Contains(view, "Edit Task") {
		t.Errorf("expected view to contain 'Edit Task' when editing")
	}
}

func TestTaskEditor_Update_TabNavigation(t *testing.T) {
	e := NewTaskEditor()
	e.StartAdd()

	// Tab should navigate to next field
	e.Update(tea.KeyMsg{Type: tea.KeyTab})
	if e.focusField != 1 {
		t.Errorf("expected focusField 1 after Tab, got %d", e.focusField)
	}

	// Tab again should wrap to first field
	e.Update(tea.KeyMsg{Type: tea.KeyTab})
	if e.focusField != 0 {
		t.Errorf("expected focusField 0 after Tab wrap, got %d", e.focusField)
	}
}

func TestTaskEditor_Update_ShiftTabNavigation(t *testing.T) {
	e := NewTaskEditor()
	e.StartAdd()
	e.focusField = 0

	// Shift+Tab should navigate to previous field (wrap around)
	e.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if e.focusField != 1 {
		t.Errorf("expected focusField 1 after Shift+Tab, got %d", e.focusField)
	}
}

func TestTaskEditor_Update_DownNavigation(t *testing.T) {
	e := NewTaskEditor()
	e.StartAdd()

	// Down should navigate like Tab
	e.Update(tea.KeyMsg{Type: tea.KeyDown})
	if e.focusField != 1 {
		t.Errorf("expected focusField 1 after Down, got %d", e.focusField)
	}
}

func TestTaskEditor_Update_UpNavigation(t *testing.T) {
	e := NewTaskEditor()
	e.StartAdd()
	e.focusField = 1

	// Up should navigate like Shift+Tab
	e.Update(tea.KeyMsg{Type: tea.KeyUp})
	if e.focusField != 0 {
		t.Errorf("expected focusField 0 after Up, got %d", e.focusField)
	}
}

func TestTaskEditor_Update_EnterSubmits(t *testing.T) {
	e := NewTaskEditor()
	e.StartAdd()
	e.nameInput.SetValue("Valid Task Name")

	cmd := e.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should return a submit message
	if cmd == nil {
		t.Fatal("expected a command from Enter")
	}
	msg := cmd()
	submitMsg, ok := msg.(TaskEditorSubmitMsg)
	if !ok {
		t.Fatalf("expected TaskEditorSubmitMsg, got %T", msg)
	}
	if submitMsg.Name != "Valid Task Name" {
		t.Errorf("expected name 'Valid Task Name', got '%s'", submitMsg.Name)
	}
	if submitMsg.Mode != TaskEditorModeAdd {
		t.Errorf("expected mode TaskEditorModeAdd, got %v", submitMsg.Mode)
	}
}

func TestTaskEditor_Update_EnterInvalidDoesNotSubmit(t *testing.T) {
	e := NewTaskEditor()
	e.StartAdd()
	// Leave name empty

	cmd := e.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should return nil (no submission)
	if cmd != nil {
		t.Error("expected no command when name is empty")
	}
	if !e.IsActive() {
		t.Error("expected editor to still be active")
	}
}

func TestTaskEditor_Update_TextInput(t *testing.T) {
	e := NewTaskEditor()
	e.StartAdd()

	// Type a character
	e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("T")})

	// The name input should have received the input
	// (Bubble Tea text inputs handle this automatically)
}

func TestTaskEditorModeConstants(t *testing.T) {
	// Verify mode constants have expected values
	if TaskEditorModeView != 0 {
		t.Errorf("expected TaskEditorModeView = 0, got %d", TaskEditorModeView)
	}
	if TaskEditorModeAdd != 1 {
		t.Errorf("expected TaskEditorModeAdd = 1, got %d", TaskEditorModeAdd)
	}
	if TaskEditorModeEdit != 2 {
		t.Errorf("expected TaskEditorModeEdit = 2, got %d", TaskEditorModeEdit)
	}
}
