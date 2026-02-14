package components

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewFileInput(t *testing.T) {
	f := NewFileInput("/tmp/project")
	if f == nil {
		t.Fatal("expected non-nil FileInput")
	}
	if f.focused {
		t.Error("expected focused to be false initially")
	}
	if f.projectDir != "/tmp/project" {
		t.Errorf("expected projectDir '/tmp/project', got %q", f.projectDir)
	}
	if f.fileExists {
		t.Error("expected fileExists to be false initially")
	}
}

func TestFileInput_SetWidth(t *testing.T) {
	f := NewFileInput("/tmp")
	f.SetWidth(100)

	if f.width != 100 {
		t.Errorf("expected width 100, got %d", f.width)
	}
}

func TestFileInput_Focus(t *testing.T) {
	f := NewFileInput("/tmp")

	cmd := f.Focus()
	if !f.focused {
		t.Error("expected focused to be true after Focus()")
	}
	if cmd == nil {
		t.Error("expected command from Focus()")
	}
}

func TestFileInput_Blur(t *testing.T) {
	f := NewFileInput("/tmp")
	f.Focus()
	f.Blur()

	if f.focused {
		t.Error("expected focused to be false after Blur()")
	}
}

func TestFileInput_Value(t *testing.T) {
	f := NewFileInput("/tmp")
	f.SetValue("TASKS.md")

	value := f.Value()
	if value != "TASKS.md" {
		t.Errorf("expected value 'TASKS.md', got %q", value)
	}
}

func TestFileInput_FileExists_NonExistent(t *testing.T) {
	f := NewFileInput("/tmp")
	f.SetValue("nonexistent-file-12345.md")

	if f.FileExists() {
		t.Error("expected FileExists() to be false for nonexistent file")
	}
	if f.ParseError() == "" {
		t.Error("expected ParseError() to be set for nonexistent file")
	}
}

func TestFileInput_FileExists_ValidFile(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-tasks.md")
	err := os.WriteFile(tmpFile, []byte("- [ ] Task 1\n- [ ] Task 2"), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	f := NewFileInput(tmpDir)
	f.SetValue("test-tasks.md")

	if !f.FileExists() {
		t.Error("expected FileExists() to be true for valid file")
	}
	tasks := f.PreviewTasks()
	if len(tasks) != 2 {
		t.Errorf("expected 2 preview tasks, got %d", len(tasks))
	}
}

func TestFileInput_Update_Esc(t *testing.T) {
	f := NewFileInput("/tmp")

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if cmd == nil {
		t.Fatal("expected command from Escape")
	}

	msg := cmd()
	if _, ok := msg.(FileInputCanceledMsg); !ok {
		t.Fatalf("expected FileInputCanceledMsg, got %T", msg)
	}
}

func TestFileInput_Update_Enter_NoFile(t *testing.T) {
	f := NewFileInput("/tmp")
	f.SetValue("nonexistent.md")

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should not submit if file doesn't exist
	if cmd != nil {
		t.Error("expected no command when file doesn't exist")
	}
}

func TestFileInput_View(t *testing.T) {
	f := NewFileInput("/tmp")

	view := f.View()

	if !strings.Contains(view, "Import from File") {
		t.Error("expected title 'Import from File' in view")
	}
	if !strings.Contains(view, "Enter") {
		t.Error("expected help text mentioning Enter")
	}
	if !strings.Contains(view, "Common locations") {
		t.Error("expected 'Common locations' section in view")
	}
}

func TestFileInputSubmittedMsg(t *testing.T) {
	msg := FileInputSubmittedMsg{Path: "TASKS.md"}
	if msg.Path != "TASKS.md" {
		t.Errorf("expected Path 'TASKS.md', got %q", msg.Path)
	}
}

func TestFileInputCanceledMsg(t *testing.T) {
	msg := FileInputCanceledMsg{}
	_ = msg
}

