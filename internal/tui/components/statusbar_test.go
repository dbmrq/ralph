package components

import (
	"strings"
	"testing"
	"time"
)

func TestNewStatusBar(t *testing.T) {
	sb := NewStatusBar()
	if sb == nil {
		t.Fatal("expected non-nil StatusBar")
	}
	if sb.data.BuildStatus != "pending" {
		t.Errorf("expected build status 'pending', got %s", sb.data.BuildStatus)
	}
	if sb.data.TestStatus != "pending" {
		t.Errorf("expected test status 'pending', got %s", sb.data.TestStatus)
	}
	if sb.data.LoopState != "running" {
		t.Errorf("expected loop state 'running', got %s", sb.data.LoopState)
	}
	if !sb.data.ShowShortcuts {
		t.Error("expected ShowShortcuts to be true by default")
	}
}

func TestStatusBar_SetData(t *testing.T) {
	sb := NewStatusBar()

	data := StatusBarData{
		ElapsedTime:   5 * time.Minute,
		Iteration:     3,
		BuildStatus:   "pass",
		TestStatus:    "fail",
		LoopState:     "paused",
		Message:       "Test message",
		ShowShortcuts: false,
	}

	sb.SetData(data)

	if sb.data.ElapsedTime != 5*time.Minute {
		t.Errorf("expected elapsed time 5m, got %v", sb.data.ElapsedTime)
	}
	if sb.data.Iteration != 3 {
		t.Errorf("expected iteration 3, got %d", sb.data.Iteration)
	}
	if sb.data.BuildStatus != "pass" {
		t.Errorf("expected build status 'pass', got %s", sb.data.BuildStatus)
	}
	if sb.data.TestStatus != "fail" {
		t.Errorf("expected test status 'fail', got %s", sb.data.TestStatus)
	}
	if sb.data.LoopState != "paused" {
		t.Errorf("expected loop state 'paused', got %s", sb.data.LoopState)
	}
	if sb.data.Message != "Test message" {
		t.Errorf("expected message 'Test message', got %s", sb.data.Message)
	}
}

func TestStatusBar_SetMethods(t *testing.T) {
	sb := NewStatusBar()

	sb.SetElapsedTime(10 * time.Minute)
	if sb.data.ElapsedTime != 10*time.Minute {
		t.Errorf("SetElapsedTime: expected 10m, got %v", sb.data.ElapsedTime)
	}

	sb.SetIteration(5)
	if sb.data.Iteration != 5 {
		t.Errorf("SetIteration: expected 5, got %d", sb.data.Iteration)
	}

	sb.SetBuildStatus("fail")
	if sb.data.BuildStatus != "fail" {
		t.Errorf("SetBuildStatus: expected 'fail', got %s", sb.data.BuildStatus)
	}

	sb.SetTestStatus("running")
	if sb.data.TestStatus != "running" {
		t.Errorf("SetTestStatus: expected 'running', got %s", sb.data.TestStatus)
	}

	sb.SetLoopState("completed")
	if sb.data.LoopState != "completed" {
		t.Errorf("SetLoopState: expected 'completed', got %s", sb.data.LoopState)
	}

	sb.SetMessage("Custom message")
	if sb.data.Message != "Custom message" {
		t.Errorf("SetMessage: expected 'Custom message', got %s", sb.data.Message)
	}

	sb.SetShowShortcuts(false)
	if sb.data.ShowShortcuts {
		t.Error("SetShowShortcuts: expected false")
	}

	sb.SetWidth(100)
	if sb.width != 100 {
		t.Errorf("SetWidth: expected 100, got %d", sb.width)
	}
}

func TestStatusBar_View(t *testing.T) {
	sb := NewStatusBar()
	sb.SetElapsedTime(5*time.Minute + 30*time.Second)
	sb.SetIteration(2)
	sb.SetBuildStatus("pass")
	sb.SetTestStatus("pass")
	sb.SetLoopState("running")

	view := sb.View()

	// Check for elapsed time
	if !strings.Contains(view, "05:30") {
		t.Errorf("expected '05:30' in view, got: %s", view)
	}

	// Check for iteration
	if !strings.Contains(view, "2") {
		t.Errorf("expected iteration '2' in view, got: %s", view)
	}

	// Check for keyboard shortcuts (default on)
	if !strings.Contains(view, "pause") || !strings.Contains(view, "skip") {
		t.Errorf("expected keyboard shortcuts in view, got: %s", view)
	}
}

func TestStatusBar_formatDuration(t *testing.T) {
	sb := NewStatusBar()

	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "00:00"},
		{30 * time.Second, "00:30"},
		{5 * time.Minute, "05:00"},
		{5*time.Minute + 45*time.Second, "05:45"},
		{1*time.Hour + 30*time.Minute + 15*time.Second, "01:30:15"},
	}

	for _, tt := range tests {
		result := sb.formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %s, want %s", tt.duration, result, tt.expected)
		}
	}
}

func TestStatusBar_View_NoShortcuts(t *testing.T) {
	sb := NewStatusBar()
	sb.SetShowShortcuts(false)

	view := sb.View()

	// Should not contain shortcut keys when disabled
	if strings.Contains(view, "pause") && strings.Contains(view, "skip") {
		t.Error("expected shortcuts to be hidden")
	}
}

func TestStatusBar_View_WithMessage(t *testing.T) {
	sb := NewStatusBar()
	sb.SetMessage("Processing task...")

	view := sb.View()

	if !strings.Contains(view, "Processing task...") {
		t.Errorf("expected message in view, got: %s", view)
	}
}

func TestStatusBar_View_StatusIndicators(t *testing.T) {
	sb := NewStatusBar()

	// Test pass status
	sb.SetBuildStatus("pass")
	sb.SetTestStatus("pass")
	view := sb.View()
	if !strings.Contains(view, "✓") {
		t.Error("expected checkmark for pass status")
	}

	// Test fail status
	sb.SetBuildStatus("fail")
	view = sb.View()
	if !strings.Contains(view, "✗") {
		t.Error("expected X for fail status")
	}

	// Test running status
	sb.SetBuildStatus("running")
	view = sb.View()
	if !strings.Contains(view, "◐") {
		t.Error("expected running indicator")
	}

	// Test pending (default)
	sb.SetBuildStatus("pending")
	view = sb.View()
	if !strings.Contains(view, "○") {
		t.Error("expected pending indicator")
	}
}

func TestStatusBar_View_LoopStateIcons(t *testing.T) {
	sb := NewStatusBar()

	tests := []struct {
		state    string
		contains string
	}{
		{"running", "Running"},
		{"paused", "Paused"},
		{"completed", "Complete"},
		{"failed", "Failed"},
		{"idle", "Idle"},
	}

	for _, tt := range tests {
		sb.SetLoopState(tt.state)
		view := sb.View()
		if !strings.Contains(view, tt.contains) {
			t.Errorf("expected '%s' for loop state '%s', got: %s", tt.contains, tt.state, view)
		}
	}
}

func TestStatusBar_View_WithWidth(t *testing.T) {
	sb := NewStatusBar()
	sb.SetWidth(100)
	sb.SetElapsedTime(1 * time.Minute)
	sb.SetIteration(1)

	view := sb.View()

	// Just verify it renders without panic
	if view == "" {
		t.Error("expected non-empty view")
	}
}
