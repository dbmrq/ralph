package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewConfirmDialog(t *testing.T) {
	c := NewConfirmDialog()

	if c.IsVisible() {
		t.Error("ConfirmDialog should be hidden by default")
	}

	if c.width != 50 {
		t.Errorf("Default width should be 50, got %d", c.width)
	}
}

func TestConfirmDialogShow(t *testing.T) {
	c := NewConfirmDialog()

	c.Show(ConfirmActionAbort, "Test Title", "Test Message", true)

	if !c.IsVisible() {
		t.Error("Show should make dialog visible")
	}
	if c.Action() != ConfirmActionAbort {
		t.Error("Action should be abort")
	}
	if c.title != "Test Title" {
		t.Errorf("Title should be 'Test Title', got %s", c.title)
	}
	if c.message != "Test Message" {
		t.Errorf("Message should be 'Test Message', got %s", c.message)
	}
	if !c.destructive {
		t.Error("Destructive should be true")
	}
}

func TestConfirmDialogShowAbort(t *testing.T) {
	c := NewConfirmDialog()

	c.ShowAbort()

	if !c.IsVisible() {
		t.Error("ShowAbort should make dialog visible")
	}
	if c.Action() != ConfirmActionAbort {
		t.Error("Action should be abort")
	}
	if !c.destructive {
		t.Error("Abort should be destructive")
	}
}

func TestConfirmDialogShowSkip(t *testing.T) {
	c := NewConfirmDialog()

	c.ShowSkip("Test Task")

	if !c.IsVisible() {
		t.Error("ShowSkip should make dialog visible")
	}
	if c.Action() != ConfirmActionSkip {
		t.Error("Action should be skip")
	}
	if c.destructive {
		t.Error("Skip should not be destructive")
	}
	if !strings.Contains(c.message, "Test Task") {
		t.Error("Message should contain task name")
	}
}

func TestConfirmDialogShowQuit(t *testing.T) {
	c := NewConfirmDialog()

	c.ShowQuit()

	if !c.IsVisible() {
		t.Error("ShowQuit should make dialog visible")
	}
	if c.Action() != ConfirmActionQuit {
		t.Error("Action should be quit")
	}
	if c.destructive {
		t.Error("Quit should not be destructive")
	}
}

func TestConfirmDialogHide(t *testing.T) {
	c := NewConfirmDialog()
	c.ShowAbort()
	c.Hide()

	if c.IsVisible() {
		t.Error("Hide should make dialog hidden")
	}
}

func TestConfirmDialogSetSize(t *testing.T) {
	c := NewConfirmDialog()

	c.SetSize(80)

	if c.width != 80 {
		t.Errorf("Width should be 80, got %d", c.width)
	}
}

func TestConfirmDialogUpdateWhenHidden(t *testing.T) {
	c := NewConfirmDialog()

	cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd != nil {
		t.Error("Update when hidden should return nil")
	}
}

func TestConfirmDialogUpdateYes(t *testing.T) {
	yesKeys := []string{"y", "Y", "enter"}

	for _, key := range yesKeys {
		c := NewConfirmDialog()
		c.ShowAbort()

		var msg tea.KeyMsg
		if key == "enter" {
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		} else {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}

		cmd := c.Update(msg)

		if c.IsVisible() {
			t.Errorf("Key '%s' should hide the dialog", key)
		}

		if cmd == nil {
			t.Errorf("Key '%s' should return a command", key)
			continue
		}

		result := cmd()
		if yesMsg, ok := result.(ConfirmYesMsg); !ok {
			t.Errorf("Key '%s' should return ConfirmYesMsg", key)
		} else if yesMsg.Action != ConfirmActionAbort {
			t.Errorf("Action should be abort, got %s", yesMsg.Action)
		}
	}
}

func TestConfirmDialogUpdateNo(t *testing.T) {
	noKeys := []string{"n", "N", "esc"}

	for _, key := range noKeys {
		c := NewConfirmDialog()
		c.ShowAbort()

		var msg tea.KeyMsg
		if key == "esc" {
			msg = tea.KeyMsg{Type: tea.KeyEscape}
		} else {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}

		cmd := c.Update(msg)

		if c.IsVisible() {
			t.Errorf("Key '%s' should hide the dialog", key)
		}

		if cmd == nil {
			t.Errorf("Key '%s' should return a command", key)
			continue
		}

		result := cmd()
		if _, ok := result.(ConfirmNoMsg); !ok {
			t.Errorf("Key '%s' should return ConfirmNoMsg", key)
		}
	}
}

func TestConfirmDialogViewWhenHidden(t *testing.T) {
	c := NewConfirmDialog()

	view := c.View()
	if view != "" {
		t.Error("View should be empty when hidden")
	}
}

func TestConfirmDialogViewWhenVisible(t *testing.T) {
	c := NewConfirmDialog()
	c.ShowAbort()

	view := c.View()

	// Should contain the title
	if !strings.Contains(view, "Abort") {
		t.Error("View should contain 'Abort' in title")
	}

	// Should contain the message
	if !strings.Contains(view, "stop") || !strings.Contains(view, "loop") {
		t.Error("View should contain abort message")
	}

	// Should contain buttons
	if !strings.Contains(view, "[Y]es") {
		t.Error("View should contain Yes button")
	}
	if !strings.Contains(view, "[N]o") {
		t.Error("View should contain No button")
	}
}

