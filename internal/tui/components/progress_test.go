package components

import (
	"strings"
	"testing"
)

func TestNewProgress(t *testing.T) {
	p := NewProgress()
	if p == nil {
		t.Fatal("NewProgress returned nil")
	}

	// Check default values
	if p.PercentComplete() != 0 {
		t.Error("New progress should have 0% completion")
	}
}

func TestProgressSetProgress(t *testing.T) {
	p := NewProgress()

	tests := []struct {
		completed int
		total     int
		expected  float64
	}{
		{0, 10, 0.0},
		{5, 10, 0.5},
		{10, 10, 1.0},
		{0, 0, 0.0}, // Handle zero total
	}

	for _, tt := range tests {
		p.SetProgress(tt.completed, tt.total)
		got := p.PercentComplete()
		if got != tt.expected {
			t.Errorf("SetProgress(%d, %d): expected %v, got %v",
				tt.completed, tt.total, tt.expected, got)
		}
	}
}

func TestProgressSetData(t *testing.T) {
	p := NewProgress()
	p.SetData(ProgressData{
		Completed:  3,
		Total:      10,
		Iteration:  5,
		StatusText: "Building...",
	})

	if p.PercentComplete() != 0.3 {
		t.Errorf("Expected 30%% completion, got %v", p.PercentComplete())
	}
}

func TestProgressIsComplete(t *testing.T) {
	p := NewProgress()

	// Not complete
	p.SetProgress(5, 10)
	if p.IsComplete() {
		t.Error("5/10 tasks should not be complete")
	}

	// Complete
	p.SetProgress(10, 10)
	if !p.IsComplete() {
		t.Error("10/10 tasks should be complete")
	}

	// Empty (not complete)
	p.SetProgress(0, 0)
	if p.IsComplete() {
		t.Error("0/0 tasks should not be complete")
	}
}

func TestProgressView(t *testing.T) {
	p := NewProgress()
	p.SetProgress(5, 10)

	view := p.View()

	// Should contain progress label
	if !strings.Contains(view, "Progress:") {
		t.Error("View should contain 'Progress:' label")
	}

	// Should contain task count
	if !strings.Contains(view, "5/10") {
		t.Error("View should contain '5/10' task count")
	}

	if !strings.Contains(view, "tasks") {
		t.Error("View should contain 'tasks' label")
	}
}

func TestProgressViewWithIteration(t *testing.T) {
	p := NewProgress()
	p.SetProgress(5, 10)
	p.SetIteration(3)

	view := p.View()

	if !strings.Contains(view, "Iteration 3") {
		t.Error("View should contain 'Iteration 3'")
	}
}

func TestProgressViewWithStatusText(t *testing.T) {
	p := NewProgress()
	p.SetProgress(5, 10)
	p.SetStatusText("Building...")

	view := p.View()

	if !strings.Contains(view, "Building...") {
		t.Error("View should contain status text 'Building...'")
	}
}

func TestProgressSetWidth(t *testing.T) {
	p := NewProgress()
	p.SetProgress(5, 10)
	p.SetWidth(80)

	// Just verify it doesn't panic
	view := p.View()
	if view == "" {
		t.Error("Progress view should not be empty")
	}
}

func TestProgressBarCharacters(t *testing.T) {
	p := NewProgress()
	p.SetProgress(5, 10)
	p.SetWidth(60)

	view := p.View()

	// Should contain filled and empty bar characters
	if !strings.Contains(view, "█") {
		t.Error("View should contain filled bar character '█'")
	}
	if !strings.Contains(view, "░") {
		t.Error("View should contain empty bar character '░'")
	}
}

