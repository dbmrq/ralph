package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewForm(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	if form == nil {
		t.Fatal("NewForm returned nil")
	}

	if form.ID() != "form-id" {
		t.Errorf("Expected ID 'form-id', got '%s'", form.ID())
	}
}

func TestFormAddField(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	ti := NewTextInput("ti-1", "Name")

	form.AddField(ti)

	if len(form.Fields()) != 1 {
		t.Errorf("Expected 1 field, got %d", len(form.Fields()))
	}
}

func TestFormAddFields(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	ti1 := NewTextInput("ti-1", "Name")
	ti2 := NewTextInput("ti-2", "Email")
	cb := NewCheckbox("cb-1", "Accept")

	form.AddFields(ti1, ti2, cb)

	if len(form.Fields()) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(form.Fields()))
	}
}

func TestFormFocus(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	ti := NewTextInput("ti-1", "Name")
	form.AddField(ti)

	form.Focus()

	if form.FocusIndex() != 0 {
		t.Errorf("Expected focus index 0, got %d", form.FocusIndex())
	}

	if !ti.Focused() {
		t.Error("First field should be focused")
	}
}

func TestFormNextField(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	ti1 := NewTextInput("ti-1", "Name")
	ti2 := NewTextInput("ti-2", "Email")
	form.AddFields(ti1, ti2)
	form.Focus()

	if form.FocusIndex() != 0 {
		t.Fatal("Should start at index 0")
	}

	form.NextField()
	if form.FocusIndex() != 1 {
		t.Errorf("Expected focus index 1, got %d", form.FocusIndex())
	}

	// Should wrap around
	form.NextField()
	if form.FocusIndex() != 0 {
		t.Errorf("Expected focus index 0 after wrap, got %d", form.FocusIndex())
	}
}

func TestFormPrevField(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	ti1 := NewTextInput("ti-1", "Name")
	ti2 := NewTextInput("ti-2", "Email")
	form.AddFields(ti1, ti2)
	form.Focus()

	// Go backwards, should wrap to last
	form.PrevField()
	if form.FocusIndex() != 1 {
		t.Errorf("Expected focus index 1, got %d", form.FocusIndex())
	}

	form.PrevField()
	if form.FocusIndex() != 0 {
		t.Errorf("Expected focus index 0, got %d", form.FocusIndex())
	}
}

func TestFormFocusField(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	ti1 := NewTextInput("ti-1", "Name")
	ti2 := NewTextInput("ti-2", "Email")
	ti3 := NewTextInput("ti-3", "Phone")
	form.AddFields(ti1, ti2, ti3)
	form.Focus()

	form.FocusField(2)
	if form.FocusIndex() != 2 {
		t.Errorf("Expected focus index 2, got %d", form.FocusIndex())
	}
}

func TestFormGetField(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	ti := NewTextInput("ti-1", "Name")
	form.AddField(ti)

	found := form.GetField("ti-1")
	if found == nil {
		t.Fatal("Should find field by ID")
	}
	if found.ID() != "ti-1" {
		t.Error("Found wrong field")
	}

	notFound := form.GetField("nonexistent")
	if notFound != nil {
		t.Error("Should return nil for nonexistent field")
	}
}

func TestFormFocusedField(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	ti1 := NewTextInput("ti-1", "Name")
	ti2 := NewTextInput("ti-2", "Email")
	form.AddFields(ti1, ti2)
	form.Focus()

	focused := form.FocusedField()
	if focused == nil {
		t.Fatal("Should have focused field")
	}
	if focused.ID() != "ti-1" {
		t.Error("First field should be focused")
	}
}

func TestFormViewContainsTitle(t *testing.T) {
	form := NewForm("form-id", "User Registration")
	view := form.View()

	if !strings.Contains(view, "User Registration") {
		t.Error("View should contain the title")
	}
}

func TestFormUpdateTabNavigation(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	ti1 := NewTextInput("ti-1", "Name")
	ti2 := NewTextInput("ti-2", "Email")
	form.AddFields(ti1, ti2)
	form.Focus()

	// Initial state
	if form.FocusIndex() != 0 {
		t.Fatal("Should start at index 0")
	}

	// Tab to next field
	msg := tea.KeyMsg{Type: tea.KeyTab}
	form.Update(msg)

	if form.FocusIndex() != 1 {
		t.Errorf("Tab should move to next field, expected 1, got %d", form.FocusIndex())
	}
}

func TestFormUpdateShiftTabNavigation(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	ti1 := NewTextInput("ti-1", "Name")
	ti2 := NewTextInput("ti-2", "Email")
	form.AddFields(ti1, ti2)
	form.Focus()
	form.NextField() // Move to index 1

	// Shift+Tab to previous field
	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	form.Update(msg)

	if form.FocusIndex() != 0 {
		t.Errorf("Shift+Tab should move to previous field, expected 0, got %d", form.FocusIndex())
	}
}

func TestFormUpdateEscCancel(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	ti := NewTextInput("ti-1", "Name")
	form.AddField(ti)
	form.Focus()

	if form.Canceled() {
		t.Fatal("Form should not be canceled initially")
	}

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	form.Update(msg)

	if !form.Canceled() {
		t.Error("Form should be canceled after Esc")
	}
}

func TestFormSubmitted(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	btn := NewButton("btn-1", "Submit")
	form.AddField(btn)
	form.Focus()

	if form.Submitted() {
		t.Fatal("Form should not be submitted initially")
	}

	// Press Enter on the button
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	form.Update(msg)

	if !form.Submitted() {
		t.Error("Form should be submitted after Enter on button")
	}
}

func TestFormReset(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	btn := NewButton("btn-1", "Submit")
	form.AddField(btn)
	form.Focus()

	// Submit the form
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	form.Update(msg)

	if !form.Submitted() {
		t.Fatal("Form should be submitted")
	}

	form.Reset()

	if form.Submitted() {
		t.Error("Form should not be submitted after Reset")
	}
	if form.Canceled() {
		t.Error("Form should not be canceled after Reset")
	}
	if form.FocusIndex() != 0 {
		t.Error("Focus index should be 0 after Reset")
	}
}

func TestFormBlur(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	ti1 := NewTextInput("ti-1", "Name")
	ti2 := NewTextInput("ti-2", "Email")
	form.AddFields(ti1, ti2)
	form.Focus()

	if !ti1.Focused() {
		t.Fatal("First field should be focused")
	}

	form.Blur()

	if ti1.Focused() {
		t.Error("Field should not be focused after form Blur")
	}
}

func TestFormViewContainsHelpText(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	form.SetShowHelp(true)
	view := form.View()

	if !strings.Contains(view, "Tab") {
		t.Error("View should contain Tab help text")
	}
	if !strings.Contains(view, "Esc") {
		t.Error("View should contain Esc help text")
	}
}

func TestFormViewHideHelp(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	form.SetShowHelp(false)
	view := form.View()

	// Help text should not appear
	if strings.Contains(view, "Tab: next field") {
		t.Error("View should not contain help text when hidden")
	}
}

func TestFormSetWidth(t *testing.T) {
	form := NewForm("form-id", "Test Form")
	form.SetWidth(80)

	// Should not panic
	view := form.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestFormEmptyFields(t *testing.T) {
	form := NewForm("form-id", "Test Form")

	// Operations on empty form should not panic
	form.Focus()
	form.NextField()
	form.PrevField()
	form.Blur()

	if form.FocusedField() != nil {
		t.Error("FocusedField should be nil for empty form")
	}
}
