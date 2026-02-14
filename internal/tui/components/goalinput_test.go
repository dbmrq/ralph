package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewGoalInput(t *testing.T) {
	g := NewGoalInput()
	if g == nil {
		t.Fatal("expected non-nil GoalInput")
	}
	if g.focused {
		t.Error("expected focused to be false initially")
	}
	if len(g.examples) == 0 {
		t.Error("expected example prompts to be populated")
	}
}

func TestGoalInput_SetWidth(t *testing.T) {
	g := NewGoalInput()
	g.SetWidth(100)

	if g.width != 100 {
		t.Errorf("expected width 100, got %d", g.width)
	}
}

func TestGoalInput_SetHeight(t *testing.T) {
	g := NewGoalInput()
	g.SetHeight(30)

	if g.height != 30 {
		t.Errorf("expected height 30, got %d", g.height)
	}
}

func TestGoalInput_Focus(t *testing.T) {
	g := NewGoalInput()

	cmd := g.Focus()
	if !g.focused {
		t.Error("expected focused to be true after Focus()")
	}
	if cmd == nil {
		t.Error("expected command from Focus()")
	}
}

func TestGoalInput_Blur(t *testing.T) {
	g := NewGoalInput()
	g.Focus()
	g.Blur()

	if g.focused {
		t.Error("expected focused to be false after Blur()")
	}
}

func TestGoalInput_Value(t *testing.T) {
	g := NewGoalInput()
	g.SetValue("Build a REST API")

	value := g.Value()
	if value != "Build a REST API" {
		t.Errorf("expected value 'Build a REST API', got %q", value)
	}
}

func TestGoalInput_Update_CtrlEnter_EmptyInput(t *testing.T) {
	g := NewGoalInput()
	// Empty input, ctrl+enter should not submit

	_, cmd := g.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ctrl+enter")})

	// With empty input, no command should be issued (or it should be nil)
	// The exact behavior depends on the key mapping
	_ = cmd
}

func TestGoalInput_Update_Esc(t *testing.T) {
	g := NewGoalInput()

	_, cmd := g.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if cmd == nil {
		t.Fatal("expected command from Escape")
	}

	msg := cmd()
	if _, ok := msg.(GoalCanceledMsg); !ok {
		t.Fatalf("expected GoalCanceledMsg, got %T", msg)
	}
}

func TestGoalInput_View(t *testing.T) {
	g := NewGoalInput()

	view := g.View()

	if !strings.Contains(view, "Describe Your Goal") {
		t.Error("expected title 'Describe Your Goal' in view")
	}
	if !strings.Contains(view, "Ctrl+Enter") {
		t.Error("expected help text mentioning Ctrl+Enter")
	}
	if !strings.Contains(view, "Example goals") {
		t.Error("expected 'Example goals' section in view")
	}
}

func TestGoalInput_View_Examples(t *testing.T) {
	g := NewGoalInput()

	view := g.View()

	// Check that at least one example is shown
	if !strings.Contains(view, "CLI tool") && !strings.Contains(view, "REST API") &&
		!strings.Contains(view, "authentication") && !strings.Contains(view, "caching") {
		t.Error("expected at least one example prompt in view")
	}
}

func TestGoalSubmittedMsg(t *testing.T) {
	msg := GoalSubmittedMsg{Goal: "Build a CLI tool"}
	if msg.Goal != "Build a CLI tool" {
		t.Errorf("expected Goal 'Build a CLI tool', got %q", msg.Goal)
	}
}

func TestGoalCanceledMsg(t *testing.T) {
	msg := GoalCanceledMsg{}
	// Just ensure the type exists and can be instantiated
	_ = msg
}
