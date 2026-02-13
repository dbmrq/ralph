package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewButton(t *testing.T) {
	btn := NewButton("btn-id", "Submit")
	if btn == nil {
		t.Fatal("NewButton returned nil")
	}

	if btn.ID() != "btn-id" {
		t.Errorf("Expected ID 'btn-id', got '%s'", btn.ID())
	}

	if btn.Label() != "Submit" {
		t.Errorf("Expected label 'Submit', got '%s'", btn.Label())
	}
}

func TestButtonFocus(t *testing.T) {
	btn := NewButton("btn-id", "Submit")

	if btn.Focused() {
		t.Error("Button should not be focused initially")
	}

	btn.Focus()
	if !btn.Focused() {
		t.Error("Button should be focused after Focus()")
	}

	btn.Blur()
	if btn.Focused() {
		t.Error("Button should not be focused after Blur()")
	}
}

func TestButtonSetLabel(t *testing.T) {
	btn := NewButton("btn-id", "Old Label")
	btn.SetLabel("New Label")

	if btn.Label() != "New Label" {
		t.Errorf("Expected 'New Label', got '%s'", btn.Label())
	}
}

func TestButtonSetStyle(t *testing.T) {
	btn := NewButton("btn-id", "Submit")

	// Test setting different styles - should not panic
	btn.SetStyle(ButtonStylePrimary)
	view := btn.View()
	if view == "" {
		t.Error("View should not be empty")
	}

	btn.SetStyle(ButtonStyleSecondary)
	view = btn.View()
	if view == "" {
		t.Error("View should not be empty")
	}

	btn.SetStyle(ButtonStyleDanger)
	view = btn.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestButtonUpdateEnter(t *testing.T) {
	btn := NewButton("btn-id", "Submit")
	btn.Focus()

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, _, activated := btn.Update(msg)

	if !activated {
		t.Error("Button should be activated on Enter")
	}
}

func TestButtonUpdateSpace(t *testing.T) {
	btn := NewButton("btn-id", "Submit")
	btn.Focus()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, _, activated := btn.Update(msg)

	if !activated {
		t.Error("Button should be activated on Space")
	}
}

func TestButtonUpdateWithoutFocus(t *testing.T) {
	btn := NewButton("btn-id", "Submit")

	// Don't focus
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, _, activated := btn.Update(msg)

	if activated {
		t.Error("Button should not be activated without focus")
	}
}

func TestButtonViewContainsLabel(t *testing.T) {
	btn := NewButton("btn-id", "Confirm & Start")
	view := btn.View()

	if !strings.Contains(view, "Confirm & Start") {
		t.Error("View should contain the label")
	}
}

func TestButtonViewFocusedStyle(t *testing.T) {
	btn := NewButton("btn-id", "Submit")

	// Unfocused view
	viewUnfocused := btn.View()

	// Focused view
	btn.Focus()
	viewFocused := btn.View()

	// Views should be different (focused has different style)
	if viewUnfocused == viewFocused {
		t.Error("Focused and unfocused views should look different")
	}
}

func TestButtonStyleConstants(t *testing.T) {
	// Verify style constants have expected values
	if ButtonStylePrimary != 0 {
		t.Error("ButtonStylePrimary should be 0")
	}
	if ButtonStyleSecondary != 1 {
		t.Error("ButtonStyleSecondary should be 1")
	}
	if ButtonStyleDanger != 2 {
		t.Error("ButtonStyleDanger should be 2")
	}
}

