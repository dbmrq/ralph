package components

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wexinc/ralph/internal/project"
)

func TestNewDirPicker(t *testing.T) {
	d := NewDirPicker()
	if d == nil {
		t.Fatal("NewDirPicker returned nil")
	}
	if d.detector == nil {
		t.Error("detector should be initialized")
	}
	if d.textInput == nil {
		t.Error("textInput should be initialized")
	}
}

func TestDirPicker_Init(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a project marker
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)

	d := NewDirPicker()
	d.Init(tmpDir, nil)

	// Should have at least current directory and manual entry
	if len(d.items) < 2 {
		t.Errorf("expected at least 2 items, got %d", len(d.items))
	}

	// First item should be current directory
	if len(d.items) > 0 && !d.items[0].IsCurrent {
		t.Error("first item should be current directory")
	}

	// Last item should be manual entry
	lastItem := d.items[len(d.items)-1]
	if lastItem.Path != "" || !strings.Contains(lastItem.Name, "manually") {
		t.Error("last item should be manual entry option")
	}
}

func TestDirPicker_InitWithRecent(t *testing.T) {
	tmpDir := t.TempDir()
	proj1 := filepath.Join(tmpDir, "proj1")
	_ = os.Mkdir(proj1, 0755)

	recent := &project.RecentProjects{
		Projects: []project.RecentProject{
			{Path: proj1, Name: "proj1"},
		},
	}

	d := NewDirPicker()
	d.Init(tmpDir, recent)

	// Should include recent project
	found := false
	for _, item := range d.items {
		if item.Path == proj1 {
			found = true
			if !item.IsRecent {
				t.Error("recent project should have IsRecent=true")
			}
			break
		}
	}
	if !found {
		t.Error("recent project should be in items")
	}
}

func TestDirPicker_UpdateList(t *testing.T) {
	d := NewDirPicker()
	d.items = []DirPickerItem{
		{Path: "/path/1", Name: "p1"},
		{Path: "/path/2", Name: "p2"},
		{Path: "", Name: "Enter manually..."},
	}
	d.selectedIdx = 0

	// Test down navigation
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyDown})
	if d.selectedIdx != 1 {
		t.Errorf("selectedIdx = %d, want 1", d.selectedIdx)
	}

	// Test up navigation
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyUp})
	if d.selectedIdx != 0 {
		t.Errorf("selectedIdx = %d, want 0", d.selectedIdx)
	}

	// Test k/j navigation
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if d.selectedIdx != 1 {
		t.Errorf("selectedIdx after j = %d, want 1", d.selectedIdx)
	}

	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if d.selectedIdx != 0 {
		t.Errorf("selectedIdx after k = %d, want 0", d.selectedIdx)
	}
}

func TestDirPicker_SelectDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	d := NewDirPicker()
	d.items = []DirPickerItem{
		{Path: tmpDir, Name: "test"},
	}
	d.selectedIdx = 0

	// Select the directory
	_, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("expected command from selection")
	}

	msg := cmd()
	if selected, ok := msg.(DirSelectedMsg); ok {
		if selected.Path != tmpDir {
			t.Errorf("selected path = %q, want %q", selected.Path, tmpDir)
		}
	} else {
		t.Errorf("expected DirSelectedMsg, got %T", msg)
	}
}

func TestDirPicker_Cancel(t *testing.T) {
	d := NewDirPicker()
	d.items = []DirPickerItem{{Path: "/test", Name: "test"}}

	_, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected command from cancel")
	}

	msg := cmd()
	if _, ok := msg.(DirCanceledMsg); !ok {
		t.Errorf("expected DirCanceledMsg, got %T", msg)
	}
}

func TestDirPicker_ManualMode(t *testing.T) {
	tmpDir := t.TempDir()

	d := NewDirPicker()
	d.items = []DirPickerItem{
		{Path: "", Name: "Enter a path manually..."},
	}
	d.selectedIdx = 0

	// Enter manual mode
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if d.mode != DirPickerModeManual {
		t.Error("should be in manual mode")
	}

	// Type a path
	d.textInput.SetValue(tmpDir)

	// Submit
	d, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command from manual submission")
	}

	msg := cmd()
	if selected, ok := msg.(DirSelectedMsg); ok {
		if selected.Path != tmpDir {
			t.Errorf("selected path = %q, want %q", selected.Path, tmpDir)
		}
	} else {
		t.Errorf("expected DirSelectedMsg, got %T", msg)
	}
}

func TestDirPicker_ManualModeEscape(t *testing.T) {
	d := NewDirPicker()
	d.mode = DirPickerModeManual

	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if d.mode != DirPickerModeList {
		t.Error("should return to list mode on escape")
	}
}

func TestDirPicker_ManualModeInvalidPath(t *testing.T) {
	d := NewDirPicker()
	d.mode = DirPickerModeManual
	d.textInput.SetValue("/nonexistent/path/that/does/not/exist")

	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if d.errorMsg == "" {
		t.Error("expected error message for invalid path")
	}
}

func TestDirPicker_ManualModeEmptyPath(t *testing.T) {
	d := NewDirPicker()
	d.mode = DirPickerModeManual
	d.textInput.SetValue("")

	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if d.errorMsg == "" {
		t.Error("expected error message for empty path")
	}
}

func TestDirPicker_View(t *testing.T) {
	d := NewDirPicker()
	d.items = []DirPickerItem{
		{Path: "/path/1", Name: "project1", ProjectType: "go", IsCurrent: true},
		{Path: "/path/2", Name: "project2", IsRecent: true},
		{Path: "", Name: "Enter a path manually..."},
	}

	view := d.View()

	// Should contain title
	if !strings.Contains(view, "Select Project Directory") {
		t.Error("view should contain title")
	}

	// Should contain project names
	if !strings.Contains(view, "project1") {
		t.Error("view should contain project1")
	}

	// Should contain project type
	if !strings.Contains(view, "go") {
		t.Error("view should contain project type")
	}
}

func TestDirPicker_ViewManualMode(t *testing.T) {
	d := NewDirPicker()
	d.mode = DirPickerModeManual

	view := d.View()

	if !strings.Contains(view, "Enter the path") {
		t.Error("view should show path prompt in manual mode")
	}
}

func TestDirPicker_ViewWithError(t *testing.T) {
	d := NewDirPicker()
	d.mode = DirPickerModeManual
	d.errorMsg = "Test error message"

	view := d.View()

	if !strings.Contains(view, "Test error message") {
		t.Error("view should show error message")
	}
}

func TestDirPicker_SetSize(t *testing.T) {
	d := NewDirPicker()
	d.SetSize(100, 50)

	if d.width != 100 {
		t.Errorf("width = %d, want 100", d.width)
	}
	if d.height != 50 {
		t.Errorf("height = %d, want 50", d.height)
	}
}

func TestDirPicker_Mode(t *testing.T) {
	d := NewDirPicker()
	if d.Mode() != DirPickerModeList {
		t.Error("default mode should be list")
	}

	d.mode = DirPickerModeManual
	if d.Mode() != DirPickerModeManual {
		t.Error("Mode() should return current mode")
	}
}

func TestDirPicker_Items(t *testing.T) {
	d := NewDirPicker()
	d.items = []DirPickerItem{{Path: "/test", Name: "test"}}

	items := d.Items()
	if len(items) != 1 {
		t.Errorf("Items() returned %d items, want 1", len(items))
	}
}

func TestDirPicker_SelectedIndex(t *testing.T) {
	d := NewDirPicker()
	d.selectedIdx = 5

	if d.SelectedIndex() != 5 {
		t.Errorf("SelectedIndex() = %d, want 5", d.SelectedIndex())
	}
}

func TestDirPicker_TildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("could not get home directory")
	}

	d := NewDirPicker()
	d.mode = DirPickerModeManual
	d.textInput.SetValue("~")

	d, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command")
	}

	msg := cmd()
	if selected, ok := msg.(DirSelectedMsg); ok {
		if selected.Path != home {
			t.Errorf("path = %q, want %q", selected.Path, home)
		}
	}
}

