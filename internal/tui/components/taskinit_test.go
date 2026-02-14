package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wexinc/ralph/internal/task"
)

func TestNewTaskInitSelector(t *testing.T) {
	s := NewTaskInitSelector()
	if s == nil {
		t.Fatal("expected non-nil TaskInitSelector")
	}
	if len(s.options) != 4 {
		t.Errorf("expected 4 options, got %d", len(s.options))
	}
	if s.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0, got %d", s.selectedIdx)
	}
	if s.detection != nil {
		t.Error("expected detection to be nil initially")
	}
}

func TestTaskInitSelector_SetDetection(t *testing.T) {
	s := NewTaskInitSelector()

	detection := &task.TaskListDetection{
		Detected: true,
		Path:     "TASKS.md",
		Format:   "markdown",
	}

	s.SetDetection(detection)

	if s.detection != detection {
		t.Error("expected detection to be set")
	}
	if !s.detection.Detected {
		t.Error("expected detection.Detected to be true")
	}
}

func TestTaskInitSelector_SetWidth(t *testing.T) {
	s := NewTaskInitSelector()
	s.SetWidth(100)

	if s.width != 100 {
		t.Errorf("expected width 100, got %d", s.width)
	}
}

func TestTaskInitSelector_Update_Navigation(t *testing.T) {
	s := NewTaskInitSelector()

	// Test down navigation
	s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if s.selectedIdx != 1 {
		t.Errorf("expected selectedIdx 1 after j, got %d", s.selectedIdx)
	}

	// Test down with arrow
	s.Update(tea.KeyMsg{Type: tea.KeyDown})
	if s.selectedIdx != 2 {
		t.Errorf("expected selectedIdx 2 after down, got %d", s.selectedIdx)
	}

	// Test up navigation
	s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if s.selectedIdx != 1 {
		t.Errorf("expected selectedIdx 1 after k, got %d", s.selectedIdx)
	}

	// Test up with arrow
	s.Update(tea.KeyMsg{Type: tea.KeyUp})
	if s.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0 after up, got %d", s.selectedIdx)
	}

	// Test boundary - can't go below 0
	s.Update(tea.KeyMsg{Type: tea.KeyUp})
	if s.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0 (boundary), got %d", s.selectedIdx)
	}

	// Test boundary - can't go past end
	s.selectedIdx = 3
	s.Update(tea.KeyMsg{Type: tea.KeyDown})
	if s.selectedIdx != 3 {
		t.Errorf("expected selectedIdx 3 (boundary), got %d", s.selectedIdx)
	}
}

func TestTaskInitSelector_Update_Enter(t *testing.T) {
	s := NewTaskInitSelector()

	// Select second option (paste)
	s.selectedIdx = 1

	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("expected command from Enter")
	}

	msg := cmd()
	selected, ok := msg.(TaskInitSelectedMsg)
	if !ok {
		t.Fatalf("expected TaskInitSelectedMsg, got %T", msg)
	}
	if selected.Mode != TaskInitModePaste {
		t.Errorf("expected mode TaskInitModePaste, got %v", selected.Mode)
	}
}

func TestTaskInitSelector_Update_Escape(t *testing.T) {
	s := NewTaskInitSelector()

	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if cmd == nil {
		t.Fatal("expected command from Escape")
	}

	msg := cmd()
	if _, ok := msg.(TaskInitCancelledMsg); !ok {
		t.Fatalf("expected TaskInitCancelledMsg, got %T", msg)
	}
}

func TestTaskInitSelector_View_NoDetection(t *testing.T) {
	s := NewTaskInitSelector()

	view := s.View()

	if !strings.Contains(view, "No Task List Found") {
		t.Error("expected 'No Task List Found' in view")
	}
	if !strings.Contains(view, "Point to a file") {
		t.Error("expected 'Point to a file' option in view")
	}
	if !strings.Contains(view, "Paste a list") {
		t.Error("expected 'Paste a list' option in view")
	}
	if !strings.Contains(view, "Describe your goal") {
		t.Error("expected 'Describe your goal' option in view")
	}
	if !strings.Contains(view, "Start empty") {
		t.Error("expected 'Start empty' option in view")
	}
}

func TestTaskInitSelector_View_WithDetection(t *testing.T) {
	s := NewTaskInitSelector()
	s.SetDetection(&task.TaskListDetection{
		Detected: true,
		Path:     "docs/TASKS.md",
		Format:   "markdown",
	})

	view := s.View()

	if !strings.Contains(view, "Task List Found") {
		t.Error("expected 'Task List Found' in view")
	}
	if !strings.Contains(view, "docs/TASKS.md") {
		t.Error("expected path 'docs/TASKS.md' in view")
	}
	if !strings.Contains(view, "markdown") {
		t.Error("expected format 'markdown' in view")
	}
}

func TestTaskInitSelector_View_SelectedIndicator(t *testing.T) {
	s := NewTaskInitSelector()
	s.selectedIdx = 2

	view := s.View()

	// The selected item should have the indicator
	if !strings.Contains(view, "▶") {
		t.Error("expected selection indicator '▶' in view")
	}
}

func TestTaskInitModeConstants(t *testing.T) {
	modes := []TaskInitMode{
		TaskInitModeFile,
		TaskInitModePaste,
		TaskInitModeGenerate,
		TaskInitModeEmpty,
	}

	for _, mode := range modes {
		if mode == "" {
			t.Error("TaskInitMode should not be empty")
		}
	}

	// Verify expected values
	if TaskInitModeFile != "file" {
		t.Errorf("expected TaskInitModeFile = 'file', got %s", TaskInitModeFile)
	}
	if TaskInitModePaste != "paste" {
		t.Errorf("expected TaskInitModePaste = 'paste', got %s", TaskInitModePaste)
	}
	if TaskInitModeGenerate != "generate" {
		t.Errorf("expected TaskInitModeGenerate = 'generate', got %s", TaskInitModeGenerate)
	}
	if TaskInitModeEmpty != "empty" {
		t.Errorf("expected TaskInitModeEmpty = 'empty', got %s", TaskInitModeEmpty)
	}
}
