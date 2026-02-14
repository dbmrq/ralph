package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewCheckbox(t *testing.T) {
	cb := NewCheckbox("cb-id", "Test Checkbox")
	if cb == nil {
		t.Fatal("NewCheckbox returned nil")
	}

	if cb.ID() != "cb-id" {
		t.Errorf("Expected ID 'cb-id', got '%s'", cb.ID())
	}

	if cb.Checked() {
		t.Error("Checkbox should not be checked initially")
	}
}

func TestCheckboxToggle(t *testing.T) {
	cb := NewCheckbox("cb-id", "Test Checkbox")

	if cb.Checked() {
		t.Error("Checkbox should start unchecked")
	}

	cb.Toggle()
	if !cb.Checked() {
		t.Error("Checkbox should be checked after Toggle()")
	}

	cb.Toggle()
	if cb.Checked() {
		t.Error("Checkbox should be unchecked after second Toggle()")
	}
}

func TestCheckboxSetChecked(t *testing.T) {
	cb := NewCheckbox("cb-id", "Test Checkbox")

	cb.SetChecked(true)
	if !cb.Checked() {
		t.Error("Checkbox should be checked")
	}

	cb.SetChecked(false)
	if cb.Checked() {
		t.Error("Checkbox should be unchecked")
	}
}

func TestCheckboxFocus(t *testing.T) {
	cb := NewCheckbox("cb-id", "Test Checkbox")

	if cb.Focused() {
		t.Error("Checkbox should not be focused initially")
	}

	cb.Focus()
	if !cb.Focused() {
		t.Error("Checkbox should be focused after Focus()")
	}

	cb.Blur()
	if cb.Focused() {
		t.Error("Checkbox should not be focused after Blur()")
	}
}

func TestCheckboxUpdateEnter(t *testing.T) {
	cb := NewCheckbox("cb-id", "Test Checkbox")
	cb.Focus()

	if cb.Checked() {
		t.Fatal("Checkbox should start unchecked")
	}

	// Simulate Enter key press
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedCb, _ := cb.Update(msg)

	if !updatedCb.Checked() {
		t.Error("Checkbox should be checked after Enter")
	}
}

func TestCheckboxUpdateSpace(t *testing.T) {
	cb := NewCheckbox("cb-id", "Test Checkbox")
	cb.Focus()

	if cb.Checked() {
		t.Fatal("Checkbox should start unchecked")
	}

	// Simulate Space key press
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	updatedCb, _ := cb.Update(msg)

	if !updatedCb.Checked() {
		t.Error("Checkbox should be checked after Space")
	}
}

func TestCheckboxUpdateWithoutFocus(t *testing.T) {
	cb := NewCheckbox("cb-id", "Test Checkbox")

	// Don't focus, try to update
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	cb.Update(msg)

	// Should not toggle without focus
	if cb.Checked() {
		t.Error("Checkbox should not toggle without focus")
	}
}

func TestCheckboxViewContainsLabel(t *testing.T) {
	cb := NewCheckbox("cb-id", "Accept Terms")
	view := cb.View()

	if !strings.Contains(view, "Accept Terms") {
		t.Error("View should contain the label 'Accept Terms'")
	}
}

func TestCheckboxViewCheckedIcon(t *testing.T) {
	cb := NewCheckbox("cb-id", "Test")

	viewUnchecked := cb.View()
	if !strings.Contains(viewUnchecked, "[ ]") {
		t.Error("Unchecked checkbox should show [ ]")
	}

	cb.SetChecked(true)
	viewChecked := cb.View()
	if !strings.Contains(viewChecked, "✓") {
		t.Error("Checked checkbox should show checkmark")
	}
}

func TestCheckboxValue(t *testing.T) {
	cb := NewCheckbox("cb-id", "Test")

	if cb.Value() != false {
		t.Error("Value() should return false for unchecked")
	}

	cb.SetChecked(true)
	if cb.Value() != true {
		t.Error("Value() should return true for checked")
	}
}

func TestCheckboxSetLabel(t *testing.T) {
	cb := NewCheckbox("cb-id", "Old Label")
	cb.SetLabel("New Label")

	view := cb.View()
	if !strings.Contains(view, "New Label") {
		t.Error("View should contain the new label")
	}
}
