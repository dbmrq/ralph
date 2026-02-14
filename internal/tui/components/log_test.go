package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewLogViewport(t *testing.T) {
	lv := NewLogViewport()
	if lv == nil {
		t.Fatal("expected non-nil LogViewport")
	}
	if !lv.autoFollow {
		t.Error("expected auto-follow to be true by default")
	}
	if lv.title != "Log Output" {
		t.Errorf("expected title 'Log Output', got %s", lv.title)
	}
	if len(lv.lines) != 0 {
		t.Errorf("expected empty lines, got %d", len(lv.lines))
	}
}

func TestLogViewport_SetTitle(t *testing.T) {
	lv := NewLogViewport()
	lv.SetTitle("Agent Output")
	if lv.title != "Agent Output" {
		t.Errorf("expected title 'Agent Output', got %s", lv.title)
	}
}

func TestLogViewport_SetSize(t *testing.T) {
	lv := NewLogViewport()
	lv.SetSize(100, 30)
	if lv.width != 100 {
		t.Errorf("expected width 100, got %d", lv.width)
	}
	if lv.height != 30 {
		t.Errorf("expected height 30, got %d", lv.height)
	}
}

func TestLogViewport_SetFocused(t *testing.T) {
	lv := NewLogViewport()

	lv.SetFocused(true)
	if !lv.focused {
		t.Error("expected focused to be true")
	}

	lv.SetFocused(false)
	if lv.focused {
		t.Error("expected focused to be false")
	}
}

func TestLogViewport_AutoFollow(t *testing.T) {
	lv := NewLogViewport()

	// Default should be true
	if !lv.AutoFollow() {
		t.Error("expected auto-follow to be true by default")
	}

	lv.SetAutoFollow(false)
	if lv.AutoFollow() {
		t.Error("expected auto-follow to be false after SetAutoFollow(false)")
	}

	lv.SetAutoFollow(true)
	if !lv.AutoFollow() {
		t.Error("expected auto-follow to be true after SetAutoFollow(true)")
	}
}

func TestLogViewport_Clear(t *testing.T) {
	lv := NewLogViewport()

	lv.AppendLine("Line 1")
	lv.AppendLine("Line 2")

	if lv.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", lv.LineCount())
	}

	lv.Clear()

	if lv.LineCount() != 0 {
		t.Errorf("expected 0 lines after clear, got %d", lv.LineCount())
	}
	if lv.Content() != "" {
		t.Errorf("expected empty content after clear, got %q", lv.Content())
	}
}

func TestLogViewport_AppendLine(t *testing.T) {
	lv := NewLogViewport()

	lv.AppendLine("First line")
	lv.AppendLine("Second line")
	lv.AppendLine("Third line")

	if lv.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", lv.LineCount())
	}

	if !strings.Contains(lv.Content(), "First line") {
		t.Error("expected content to contain 'First line'")
	}
	if !strings.Contains(lv.Content(), "Third line") {
		t.Error("expected content to contain 'Third line'")
	}
}

func TestLogViewport_AppendText(t *testing.T) {
	lv := NewLogViewport()

	lv.AppendText("Hello ")
	lv.AppendText("World\n")
	lv.AppendText("New line")

	content := lv.Content()
	if !strings.Contains(content, "Hello World") {
		t.Errorf("expected content to contain 'Hello World', got %q", content)
	}
}

func TestLogViewport_SetContent(t *testing.T) {
	lv := NewLogViewport()

	content := "Line 1\nLine 2\nLine 3"
	lv.SetContent(content)

	if lv.Content() != content {
		t.Errorf("expected content %q, got %q", content, lv.Content())
	}
	if lv.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", lv.LineCount())
	}
}

func TestLogViewport_Write(t *testing.T) {
	lv := NewLogViewport()

	n, err := lv.Write([]byte("Test output"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != 11 {
		t.Errorf("Write() returned %d, expected 11", n)
	}
	if !strings.Contains(lv.Content(), "Test output") {
		t.Error("expected content to contain 'Test output'")
	}
}

func TestLogViewport_Navigation(t *testing.T) {
	lv := NewLogViewport()
	lv.SetSize(80, 10)

	// Add enough content to scroll
	for i := 0; i < 50; i++ {
		lv.AppendLine(strings.Repeat("Line ", 10))
	}

	t.Run("GotoTop disables auto-follow", func(t *testing.T) {
		lv.SetAutoFollow(true)
		lv.GotoTop()
		if lv.AutoFollow() {
			t.Error("expected auto-follow to be false after GotoTop")
		}
	})

	t.Run("GotoBottom enables auto-follow", func(t *testing.T) {
		lv.SetAutoFollow(false)
		lv.GotoBottom()
		if !lv.AutoFollow() {
			t.Error("expected auto-follow to be true after GotoBottom")
		}
	})
}

func TestLogViewport_Update(t *testing.T) {
	lv := NewLogViewport()
	lv.SetSize(80, 10)

	// Add content
	for i := 0; i < 30; i++ {
		lv.AppendLine("Test line")
	}

	tests := []struct {
		name             string
		key              string
		expectAutoFollow bool
	}{
		{"up disables auto-follow", "up", false},
		{"k disables auto-follow", "k", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lv.SetAutoFollow(true)
			lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			if lv.AutoFollow() != tt.expectAutoFollow {
				t.Errorf("expected auto-follow %v, got %v", tt.expectAutoFollow, lv.AutoFollow())
			}
		})
	}

	t.Run("f toggles auto-follow", func(t *testing.T) {
		lv.SetAutoFollow(true)
		lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
		if lv.AutoFollow() {
			t.Error("expected auto-follow to be toggled off")
		}
		lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
		if !lv.AutoFollow() {
			t.Error("expected auto-follow to be toggled on")
		}
	})
}

func TestLogViewport_View(t *testing.T) {
	lv := NewLogViewport()
	lv.SetTitle("Test Log")
	lv.SetSize(80, 10)
	lv.AppendLine("Test content line")

	view := lv.View()

	// Should contain title
	if !strings.Contains(view, "Test Log") {
		t.Errorf("expected view to contain 'Test Log', got: %s", view)
	}

	// Should contain help text
	if !strings.Contains(view, "scroll") {
		t.Errorf("expected view to contain help text with 'scroll', got: %s", view)
	}

	// Test auto-follow indicator
	lv.SetAutoFollow(true)
	view = lv.View()
	if !strings.Contains(view, "auto-follow") {
		t.Errorf("expected view to contain 'auto-follow' indicator, got: %s", view)
	}
}

func TestLogViewport_ScrollPercent(t *testing.T) {
	lv := NewLogViewport()
	lv.SetSize(80, 5)

	// With no content, scroll percent should be valid
	percent := lv.ScrollPercent()
	// Just verify it doesn't panic and returns a valid value
	if percent < 0 || percent > 1 {
		t.Errorf("expected scroll percent between 0 and 1, got %f", percent)
	}
}

func TestLogViewport_GoToTop(t *testing.T) {
	lv := NewLogViewport()
	lv.SetSize(80, 5)

	// Add enough content to scroll
	for i := 0; i < 30; i++ {
		lv.AppendLine("Line content")
	}

	lv.SetAutoFollow(true)
	lv.GoToTop()

	if lv.AutoFollow() {
		t.Error("expected auto-follow to be false after GoToTop")
	}
}

func TestLogViewport_GoToBottom(t *testing.T) {
	lv := NewLogViewport()
	lv.SetSize(80, 5)

	// Add content
	for i := 0; i < 30; i++ {
		lv.AppendLine("Line content")
	}

	lv.SetAutoFollow(false)
	lv.GoToBottom()

	if !lv.AutoFollow() {
		t.Error("expected auto-follow to be true after GoToBottom")
	}
}

func TestLogViewport_ScrollUp(t *testing.T) {
	lv := NewLogViewport()
	lv.SetSize(80, 5)

	// Add content
	for i := 0; i < 30; i++ {
		lv.AppendLine("Line content")
	}

	lv.SetAutoFollow(true)
	lv.ScrollUp()

	if lv.AutoFollow() {
		t.Error("expected auto-follow to be false after ScrollUp")
	}
}

func TestLogViewport_ScrollDown(t *testing.T) {
	lv := NewLogViewport()
	lv.SetSize(80, 5)

	// Add content
	for i := 0; i < 30; i++ {
		lv.AppendLine("Line content")
	}

	// ScrollDown should not error and should work
	lv.ScrollDown()
	// Just verify no panic
}

func TestLogViewport_ToggleAutoFollow(t *testing.T) {
	lv := NewLogViewport()
	lv.SetSize(80, 5)

	// Add content
	for i := 0; i < 30; i++ {
		lv.AppendLine("Line content")
	}

	lv.SetAutoFollow(true)
	lv.ToggleAutoFollow()
	if lv.AutoFollow() {
		t.Error("expected auto-follow to be toggled off")
	}

	lv.ToggleAutoFollow()
	if !lv.AutoFollow() {
		t.Error("expected auto-follow to be toggled on")
	}
}

func TestLogViewport_Update_PageNavigation(t *testing.T) {
	lv := NewLogViewport()
	lv.SetSize(80, 5)

	// Add content
	for i := 0; i < 50; i++ {
		lv.AppendLine("Line content")
	}

	// Test pgup disables auto-follow
	lv.SetAutoFollow(true)
	lv.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if lv.AutoFollow() {
		t.Error("expected auto-follow to be false after pgup")
	}

	// Test home key
	lv.SetAutoFollow(true)
	lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if lv.AutoFollow() {
		t.Error("expected auto-follow to be false after home (g)")
	}

	// Test end key enables auto-follow
	lv.SetAutoFollow(false)
	lv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if !lv.AutoFollow() {
		t.Error("expected auto-follow to be true after end (G)")
	}
}
