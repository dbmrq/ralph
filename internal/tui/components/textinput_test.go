package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewTextInput(t *testing.T) {
	ti := NewTextInput("test-id", "Test Label")
	if ti == nil {
		t.Fatal("NewTextInput returned nil")
	}

	if ti.ID() != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", ti.ID())
	}

	if ti.Value() != "" {
		t.Errorf("Expected empty value, got '%s'", ti.Value())
	}
}

func TestTextInputSetValue(t *testing.T) {
	ti := NewTextInput("test-id", "Test Label")
	ti.SetValue("hello world")

	if ti.Value() != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", ti.Value())
	}
}

func TestTextInputFocus(t *testing.T) {
	ti := NewTextInput("test-id", "Test Label")

	if ti.Focused() {
		t.Error("TextInput should not be focused initially")
	}

	ti.Focus()
	if !ti.Focused() {
		t.Error("TextInput should be focused after Focus()")
	}

	ti.Blur()
	if ti.Focused() {
		t.Error("TextInput should not be focused after Blur()")
	}
}

func TestTextInputSetPlaceholder(t *testing.T) {
	ti := NewTextInput("test-id", "Test Label")
	ti.SetPlaceholder("Enter text here")

	// The placeholder is set on the internal model
	// We can verify the view contains hints about the placeholder
	view := ti.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestTextInputSetWidth(t *testing.T) {
	ti := NewTextInput("test-id", "Test Label")
	ti.SetWidth(40)

	// Verify it doesn't panic and renders
	view := ti.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestTextInputSetCharLimit(t *testing.T) {
	ti := NewTextInput("test-id", "Test Label")
	ti.SetCharLimit(10)

	// Set a value and verify it works
	ti.SetValue("short")
	if ti.Value() != "short" {
		t.Errorf("Expected 'short', got '%s'", ti.Value())
	}
}

func TestTextInputUpdate(t *testing.T) {
	ti := NewTextInput("test-id", "Test Label")

	// Update without focus should not change anything
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updatedTi, cmd := ti.Update(msg)

	if cmd != nil {
		t.Error("Update without focus should return nil cmd")
	}
	if updatedTi == nil {
		t.Error("Update should return non-nil TextInput")
	}

	// Focus and update
	ti.Focus()
	updatedTi, _ = ti.Update(msg)
	if updatedTi == nil {
		t.Error("Update should return non-nil TextInput")
	}
}

func TestTextInputViewContainsLabel(t *testing.T) {
	ti := NewTextInput("test-id", "Username")
	view := ti.View()

	if !strings.Contains(view, "Username") {
		t.Error("View should contain the label 'Username'")
	}
}

func TestTextInputReset(t *testing.T) {
	ti := NewTextInput("test-id", "Test Label")
	ti.SetValue("some value")

	if ti.Value() != "some value" {
		t.Fatal("Value should be set")
	}

	ti.Reset()
	if ti.Value() != "" {
		t.Errorf("Value should be empty after Reset, got '%s'", ti.Value())
	}
}
