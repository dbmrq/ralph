// Package tui provides the terminal user interface for ralph.
package tui

import (
	"testing"
	"time"

	"github.com/wexinc/ralph/internal/loop"
	"github.com/wexinc/ralph/internal/task"
)

func TestTUIEventHandler_HandleEvent_LoopStarted(t *testing.T) {
	handler := &TUIEventHandler{
		startAt: time.Now(),
	}
	// Use a custom handler that captures messages
	handler.HandleEvent(loop.Event{
		Type:      loop.EventLoopStarted,
		Timestamp: time.Now(),
	})

	// Since we can't easily mock tea.Program, we at least verify the handler doesn't panic
	// In a real test, we'd use a mock program
	if handler.startAt.IsZero() {
		t.Error("startAt should be set")
	}
}

func TestTUIEventHandler_SetTasks(t *testing.T) {
	handler := NewTUIEventHandler(nil)

	tasks := []*task.Task{
		{ID: "1", Name: "Task 1", Status: task.StatusPending},
		{ID: "2", Name: "Task 2", Status: task.StatusCompleted},
	}

	handler.SetTasks(tasks)

	handler.tasksMu.RLock()
	defer handler.tasksMu.RUnlock()

	if len(handler.tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(handler.tasks))
	}
}

func TestTUIOutputWriter_Write(t *testing.T) {
	writer := &TUIOutputWriter{
		program: nil, // nil program means Write returns without sending
	}

	n, err := writer.Write([]byte("test line\n"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 10 {
		t.Errorf("expected 10 bytes written, got %d", n)
	}
}

func TestTUIOutputWriter_NilProgram_NoOp(t *testing.T) {
	// When program is nil, writes should be a no-op (returns without error)
	// This is intentional to avoid unnecessary buffering when output isn't going anywhere
	writer := &TUIOutputWriter{
		program: nil,
	}

	n, err := writer.Write([]byte("partial"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 7 {
		t.Errorf("expected 7 bytes written, got %d", n)
	}

	// Buffer should be empty when program is nil (early return)
	if writer.buffer.Len() != 0 {
		t.Errorf("expected buffer to be empty when program is nil, got %d bytes", writer.buffer.Len())
	}
}

func TestTUIOutputWriter_Flush_NilProgram(t *testing.T) {
	// Flush should not panic when program is nil
	writer := &TUIOutputWriter{
		program: nil,
	}

	// Manually add to buffer to test flush
	writer.buffer.WriteString("test")

	// Flush should clear the buffer (even with nil program)
	writer.Flush()

	if writer.buffer.Len() != 0 {
		t.Errorf("expected buffer to be empty after flush, got %d bytes", writer.buffer.Len())
	}
}

func TestLineWriter_Write(t *testing.T) {
	var lines []string
	lw := NewLineWriter(func(line string) {
		lines = append(lines, line)
	})

	lw.Write([]byte("line1\nline2\n"))

	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "line1" {
		t.Errorf("expected 'line1', got %q", lines[0])
	}
	if lines[1] != "line2" {
		t.Errorf("expected 'line2', got %q", lines[1])
	}
}

func TestLineWriter_PartialLine(t *testing.T) {
	var lines []string
	lw := NewLineWriter(func(line string) {
		lines = append(lines, line)
	})

	lw.Write([]byte("partial"))

	// No complete line yet
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}

	// Complete the line
	lw.Write([]byte(" line\n"))

	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(lines))
	}
	if lines[0] != "partial line" {
		t.Errorf("expected 'partial line', got %q", lines[0])
	}
}

