package components

import (
	"strings"
	"testing"
)

func TestNewHeader(t *testing.T) {
	h := NewHeader()
	if h == nil {
		t.Fatal("NewHeader returned nil")
	}

	// Check default values
	view := h.View()
	if !strings.Contains(view, "ralph") {
		t.Error("Header should contain default project name 'ralph'")
	}
}

func TestHeaderSetData(t *testing.T) {
	h := NewHeader()
	h.SetData(HeaderData{
		ProjectName: "test-project",
		AgentName:   "auggie",
		ModelName:   "opus-4",
		SessionID:   "abc12345",
	})

	view := h.View()

	if !strings.Contains(view, "test-project") {
		t.Error("Header should contain project name 'test-project'")
	}
	if !strings.Contains(view, "auggie") {
		t.Error("Header should contain agent name 'auggie'")
	}
	if !strings.Contains(view, "opus-4") {
		t.Error("Header should contain model name 'opus-4'")
	}
	// Session ID should be truncated to 8 chars
	if !strings.Contains(view, "abc12345") {
		t.Error("Header should contain session ID 'abc12345'")
	}
}

func TestHeaderSetProjectName(t *testing.T) {
	h := NewHeader()
	h.SetProjectName("my-project")

	view := h.View()
	if !strings.Contains(view, "my-project") {
		t.Error("Header should contain project name 'my-project'")
	}
}

func TestHeaderSetAgentName(t *testing.T) {
	h := NewHeader()
	h.SetAgentName("cursor")

	view := h.View()
	if !strings.Contains(view, "cursor") {
		t.Error("Header should contain agent name 'cursor'")
	}
}

func TestHeaderSetModelName(t *testing.T) {
	h := NewHeader()
	h.SetModelName("gpt-4")

	view := h.View()
	if !strings.Contains(view, "gpt-4") {
		t.Error("Header should contain model name 'gpt-4'")
	}
}

func TestHeaderSetSessionID(t *testing.T) {
	h := NewHeader()

	// Test with short session ID
	h.SetSessionID("abc123")
	view := h.View()
	if !strings.Contains(view, "abc123") {
		t.Error("Header should contain session ID 'abc123'")
	}

	// Test with long session ID (should be truncated)
	h.SetSessionID("abcdefghijklmnop")
	view = h.View()
	if !strings.Contains(view, "abcdefgh") {
		t.Error("Header should contain truncated session ID 'abcdefgh'")
	}
	if strings.Contains(view, "abcdefghijklmnop") {
		t.Error("Header should not contain full long session ID")
	}
}

func TestHeaderSetWidth(t *testing.T) {
	h := NewHeader()
	h.SetWidth(80)

	// Just verify it doesn't panic
	view := h.View()
	if view == "" {
		t.Error("Header view should not be empty")
	}
}

func TestHeaderContainsTitle(t *testing.T) {
	h := NewHeader()
	view := h.View()

	if !strings.Contains(view, "RALPH LOOP") {
		t.Error("Header should contain title 'RALPH LOOP'")
	}
}
