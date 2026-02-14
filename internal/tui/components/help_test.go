package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewHelpOverlay(t *testing.T) {
	h := NewHelpOverlay()

	if h.IsVisible() {
		t.Error("HelpOverlay should be hidden by default")
	}

	if len(h.groups) == 0 {
		t.Error("HelpOverlay should have default shortcut groups")
	}
}

func TestHelpOverlayVisibility(t *testing.T) {
	h := NewHelpOverlay()

	// Test Show
	h.Show()
	if !h.IsVisible() {
		t.Error("Show should make overlay visible")
	}

	// Test Hide
	h.Hide()
	if h.IsVisible() {
		t.Error("Hide should make overlay hidden")
	}

	// Test Toggle
	h.Toggle()
	if !h.IsVisible() {
		t.Error("Toggle from hidden should show overlay")
	}

	h.Toggle()
	if h.IsVisible() {
		t.Error("Toggle from visible should hide overlay")
	}
}

func TestHelpOverlaySetSize(t *testing.T) {
	h := NewHelpOverlay()

	h.SetSize(100, 50)

	if h.width != 100 {
		t.Errorf("Width should be 100, got %d", h.width)
	}
	if h.height != 50 {
		t.Errorf("Height should be 50, got %d", h.height)
	}
}

func TestHelpOverlaySetGroups(t *testing.T) {
	h := NewHelpOverlay()

	customGroups := []ShortcutGroup{
		{
			Title: "Custom",
			Shortcuts: []Shortcut{
				{Key: "x", Desc: "Do something"},
			},
		},
	}

	h.SetGroups(customGroups)

	if len(h.groups) != 1 {
		t.Errorf("Should have 1 group, got %d", len(h.groups))
	}
	if h.groups[0].Title != "Custom" {
		t.Errorf("Group title should be 'Custom', got %s", h.groups[0].Title)
	}
}

func TestHelpOverlayUpdateWhenHidden(t *testing.T) {
	h := NewHelpOverlay()

	// Update when hidden should return nil
	cmd := h.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if cmd != nil {
		t.Error("Update when hidden should return nil")
	}
}

func TestHelpOverlayUpdateClosesOnKey(t *testing.T) {
	closeKeys := []string{"esc", "h", "?", "q"}

	for _, key := range closeKeys {
		h := NewHelpOverlay()
		h.Show()

		var msg tea.KeyMsg
		if key == "esc" {
			msg = tea.KeyMsg{Type: tea.KeyEscape}
		} else {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}

		cmd := h.Update(msg)

		if h.IsVisible() {
			t.Errorf("Key '%s' should hide the overlay", key)
		}

		if cmd == nil {
			t.Errorf("Key '%s' should return a command", key)
			continue
		}

		// Execute command and check message type
		result := cmd()
		if _, ok := result.(HelpClosedMsg); !ok {
			t.Errorf("Key '%s' should return HelpClosedMsg", key)
		}
	}
}

func TestHelpOverlayViewWhenHidden(t *testing.T) {
	h := NewHelpOverlay()

	view := h.View()
	if view != "" {
		t.Error("View should be empty when hidden")
	}
}

func TestHelpOverlayViewWhenVisible(t *testing.T) {
	h := NewHelpOverlay()
	h.Show()

	view := h.View()

	// Should contain the title
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Error("View should contain title")
	}

	// Should contain some keyboard shortcuts
	if !strings.Contains(view, "Pause") || !strings.Contains(view, "Resume") {
		t.Error("View should contain pause/resume shortcut")
	}

	// Should contain footer
	if !strings.Contains(view, "Press any key to close") {
		t.Error("View should contain close instruction")
	}
}
