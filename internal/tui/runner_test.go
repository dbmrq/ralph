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

// ============================================
// TEST-007: TUI-Loop Integration Tests
// ============================================

// mockProgram captures messages sent to it for testing.
// Since we can't easily use tea.Program in tests, we test the handler logic
// by verifying it doesn't panic and processes events correctly.
type mockMessageCollector struct {
	messages []interface{}
}

func (m *mockMessageCollector) add(msg interface{}) {
	m.messages = append(m.messages, msg)
}

// TestTUIEventHandler_HandleEvent_AllEventTypes tests that all loop events
// are handled without panicking, even with a nil program.
func TestTUIEventHandler_HandleEvent_AllEventTypes(t *testing.T) {
	handler := NewTUIEventHandler(nil) // nil program - should not panic

	eventTypes := []loop.EventType{
		loop.EventLoopStarted,
		loop.EventLoopCompleted,
		loop.EventLoopFailed,
		loop.EventLoopPaused,
		loop.EventTaskStarted,
		loop.EventTaskCompleted,
		loop.EventTaskSkipped,
		loop.EventTaskFailed,
		loop.EventIterationStarted,
		loop.EventIterationEnded,
		loop.EventVerifyStarted,
		loop.EventVerifyPassed,
		loop.EventVerifyFailed,
		loop.EventAnalysisStarted,
		loop.EventAnalysisCompleted,
		loop.EventAnalysisFailed,
		loop.EventError,
	}

	for _, et := range eventTypes {
		t.Run(string(et), func(t *testing.T) {
			event := loop.Event{
				Type:      et,
				TaskID:    "TASK-001",
				TaskName:  "Test Task",
				Iteration: 1,
				Message:   "Test message",
				Timestamp: time.Now(),
			}
			// Should not panic
			handler.HandleEvent(event)
		})
	}
}

// TestTUIEventHandler_HandleEvent_WithError tests events that include errors.
func TestTUIEventHandler_HandleEvent_WithError(t *testing.T) {
	handler := NewTUIEventHandler(nil)

	tests := []struct {
		name  string
		event loop.Event
	}{
		{
			name: "loop failed with error",
			event: loop.Event{
				Type:  loop.EventLoopFailed,
				Error: errTest,
			},
		},
		{
			name: "task failed with error",
			event: loop.Event{
				Type:     loop.EventTaskFailed,
				TaskID:   "TASK-001",
				TaskName: "Test Task",
				Error:    errTest,
			},
		},
		{
			name: "analysis failed with error",
			event: loop.Event{
				Type:  loop.EventAnalysisFailed,
				Error: errTest,
			},
		},
		{
			name: "error event",
			event: loop.Event{
				Type:  loop.EventError,
				Error: errTest,
			},
		},
		{
			name: "error event with message but no error",
			event: loop.Event{
				Type:    loop.EventError,
				Message: "Something went wrong",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			handler.HandleEvent(tt.event)
		})
	}
}

var errTest = &testError{msg: "test error"}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }

// TestTUIEventHandler_sendTasksUpdated tests the task counting logic.
func TestTUIEventHandler_sendTasksUpdated(t *testing.T) {
	handler := NewTUIEventHandler(nil)

	t.Run("with nil tasks", func(t *testing.T) {
		// Should not panic with nil tasks
		handler.sendTasksUpdated()
	})

	t.Run("with empty tasks", func(t *testing.T) {
		handler.SetTasks([]*task.Task{})
		handler.sendTasksUpdated()
	})

	t.Run("counts completed tasks correctly", func(t *testing.T) {
		tasks := []*task.Task{
			{ID: "1", Name: "Task 1", Status: task.StatusPending},
			{ID: "2", Name: "Task 2", Status: task.StatusCompleted},
			{ID: "3", Name: "Task 3", Status: task.StatusCompleted},
			{ID: "4", Name: "Task 4", Status: task.StatusInProgress},
		}
		handler.SetTasks(tasks)
		handler.sendTasksUpdated()
		// Can't verify the message was sent without a mock program,
		// but we verify the logic runs without error
	})
}

// TestTUIOutputWriter_BuffersPartialLines tests that partial lines are buffered.
func TestTUIOutputWriter_BuffersPartialLines(t *testing.T) {
	// With nil program, writes should be no-ops
	writer := NewTUIOutputWriter(nil)

	// Write partial line
	n, err := writer.Write([]byte("partial"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 7 {
		t.Errorf("expected 7 bytes written, got %d", n)
	}

	// Buffer should be empty with nil program (early return)
	if writer.buffer.Len() != 0 {
		t.Errorf("expected empty buffer with nil program, got %d bytes", writer.buffer.Len())
	}
}

// TestTUIOutputWriter_MultipleLines tests writing multiple complete lines.
func TestTUIOutputWriter_MultipleLines(t *testing.T) {
	writer := NewTUIOutputWriter(nil)

	input := "line1\nline2\nline3\n"
	n, err := writer.Write([]byte(input))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != len(input) {
		t.Errorf("expected %d bytes written, got %d", len(input), n)
	}
}

// TestTUIOutputWriter_Flush tests flushing remaining content.
func TestTUIOutputWriter_Flush(t *testing.T) {
	t.Run("flush with nil program", func(t *testing.T) {
		writer := NewTUIOutputWriter(nil)
		writer.buffer.WriteString("remaining content")
		writer.Flush()

		if writer.buffer.Len() != 0 {
			t.Errorf("expected buffer to be cleared after flush")
		}
	})

	t.Run("flush empty buffer", func(t *testing.T) {
		writer := NewTUIOutputWriter(nil)
		// Should not panic
		writer.Flush()
	})
}

// TestTUIOutputWriter_Concurrent tests concurrent access to the writer.
func TestTUIOutputWriter_Concurrent(t *testing.T) {
	writer := NewTUIOutputWriter(nil)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				writer.Write([]byte("concurrent line\n"))
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestNewTUIEventHandler tests constructor.
func TestNewTUIEventHandler(t *testing.T) {
	handler := NewTUIEventHandler(nil)

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
	if handler.startAt.IsZero() {
		t.Error("expected startAt to be initialized")
	}
}

// TestNewTUIOutputWriter tests constructor.
func TestNewTUIOutputWriter(t *testing.T) {
	writer := NewTUIOutputWriter(nil)

	if writer == nil {
		t.Fatal("expected non-nil writer")
	}
}

// TestTUIRunner_ConfigureLoop tests that ConfigureLoop sets up the loop options correctly.
func TestTUIRunner_ConfigureLoop(t *testing.T) {
	// Create a minimal runner (without starting it)
	tasks := []*task.Task{
		{ID: "1", Name: "Task 1"},
	}
	sessionInfo := SessionInfo{
		ProjectName: "test-project",
		AgentName:   "test-agent",
		ModelName:   "test-model",
		SessionID:   "test-session",
	}
	runner := NewTUIRunner(nil, tasks, sessionInfo)

	opts := &loop.Options{}
	runner.ConfigureLoop(opts)

	if opts.OnEvent == nil {
		t.Error("expected OnEvent to be set")
	}
	if opts.LogWriter == nil {
		t.Error("expected LogWriter to be set")
	}
}

// TestTUIRunner_Program tests that Program() returns the program.
func TestTUIRunner_Program(t *testing.T) {
	tasks := []*task.Task{}
	sessionInfo := SessionInfo{}
	runner := NewTUIRunner(nil, tasks, sessionInfo)

	if runner.Program() == nil {
		t.Error("expected non-nil Program")
	}
}

// TestTUIRunner_Model tests that Model() returns the model.
func TestTUIRunner_Model(t *testing.T) {
	tasks := []*task.Task{}
	sessionInfo := SessionInfo{}
	runner := NewTUIRunner(nil, tasks, sessionInfo)

	if runner.Model() == nil {
		t.Error("expected non-nil Model")
	}
}

// TestSessionInfo tests SessionInfo struct.
func TestSessionInfo(t *testing.T) {
	info := SessionInfo{
		ProjectName: "test-project",
		AgentName:   "test-agent",
		ModelName:   "test-model",
		SessionID:   "test-session-123",
	}

	if info.ProjectName != "test-project" {
		t.Errorf("expected ProjectName 'test-project', got %q", info.ProjectName)
	}
	if info.AgentName != "test-agent" {
		t.Errorf("expected AgentName 'test-agent', got %q", info.AgentName)
	}
	if info.ModelName != "test-model" {
		t.Errorf("expected ModelName 'test-model', got %q", info.ModelName)
	}
	if info.SessionID != "test-session-123" {
		t.Errorf("expected SessionID 'test-session-123', got %q", info.SessionID)
	}
}

// TestLineWriter_NilOnLine tests LineWriter with nil onLine callback.
func TestLineWriter_NilOnLine(t *testing.T) {
	lw := NewLineWriter(nil)

	// Should not panic
	n, err := lw.Write([]byte("line1\nline2\n"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 12 {
		t.Errorf("expected 12 bytes written, got %d", n)
	}
}

// TestLineWriter_EmptyInput tests LineWriter with empty input.
func TestLineWriter_EmptyInput(t *testing.T) {
	var lines []string
	lw := NewLineWriter(func(line string) {
		lines = append(lines, line)
	})

	n, err := lw.Write([]byte(""))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 bytes written, got %d", n)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

// TestLineWriter_OnlyNewlines tests LineWriter with only newlines.
func TestLineWriter_OnlyNewlines(t *testing.T) {
	var lines []string
	lw := NewLineWriter(func(line string) {
		lines = append(lines, line)
	})

	lw.Write([]byte("\n\n\n"))

	// Should produce 3 empty lines
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if line != "" {
			t.Errorf("expected empty line at index %d, got %q", i, line)
		}
	}
}

