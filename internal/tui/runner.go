// Package tui provides the terminal user interface for ralph.
// This file implements TUI-007: the bridge that runs the TUI alongside the Loop.
package tui

import (
	"bufio"
	"bytes"
	"io"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wexinc/ralph/internal/loop"
	"github.com/wexinc/ralph/internal/task"
)

// TUIEventHandler translates loop events to TUI messages.
// It implements the loop.EventHandler signature and sends appropriate
// TUI messages to the Bubble Tea program.
type TUIEventHandler struct {
	program  *tea.Program
	startAt  time.Time
	tasks    []*task.Task
	tasksMu  sync.RWMutex
}

// NewTUIEventHandler creates a new TUIEventHandler.
func NewTUIEventHandler(program *tea.Program) *TUIEventHandler {
	return &TUIEventHandler{
		program: program,
		startAt: time.Now(),
	}
}

// SetTasks sets the task list for progress tracking.
func (h *TUIEventHandler) SetTasks(tasks []*task.Task) {
	h.tasksMu.Lock()
	defer h.tasksMu.Unlock()
	h.tasks = tasks
}

// HandleEvent processes a loop event and sends appropriate TUI messages.
// This implements the loop.EventHandler signature.
func (h *TUIEventHandler) HandleEvent(event loop.Event) {
	if h.program == nil {
		return
	}

	switch event.Type {
	case loop.EventLoopStarted:
		h.program.Send(LoopStateMsg{
			State:      string(loop.StateRunning),
			ElapsedAt:  h.startAt,
			CurrentMsg: "Loop started",
		})

	case loop.EventLoopCompleted:
		h.program.Send(LoopStateMsg{
			State:      string(loop.StateCompleted),
			ElapsedAt:  h.startAt,
			CurrentMsg: "All tasks completed",
		})

	case loop.EventLoopFailed:
		errMsg := ""
		if event.Error != nil {
			errMsg = event.Error.Error()
		}
		h.program.Send(LoopStateMsg{
			State:      string(loop.StateFailed),
			ElapsedAt:  h.startAt,
			CurrentMsg: errMsg,
		})
		if errMsg != "" {
			h.program.Send(ErrorMsg{Error: errMsg})
		}

	case loop.EventLoopPaused:
		h.program.Send(LoopStateMsg{
			State:      string(loop.StatePaused),
			ElapsedAt:  h.startAt,
			CurrentMsg: event.Message,
		})

	case loop.EventTaskStarted:
		h.program.Send(TaskStartedMsg{
			TaskID:    event.TaskID,
			TaskName:  event.TaskName,
			Iteration: event.Iteration,
		})
		h.sendTasksUpdated()

	case loop.EventTaskCompleted:
		h.program.Send(TaskCompletedMsg{
			TaskID:   event.TaskID,
			TaskName: event.TaskName,
			Status:   "completed",
			Duration: time.Since(h.startAt),
		})
		h.sendTasksUpdated()

	case loop.EventTaskSkipped:
		h.program.Send(TaskCompletedMsg{
			TaskID:   event.TaskID,
			TaskName: event.TaskName,
			Status:   "skipped",
			Duration: time.Since(h.startAt),
		})
		h.sendTasksUpdated()

	case loop.EventTaskFailed:
		errMsg := ""
		if event.Error != nil {
			errMsg = event.Error.Error()
		}
		h.program.Send(TaskFailedMsg{
			TaskID:    event.TaskID,
			TaskName:  event.TaskName,
			Error:     errMsg,
			Iteration: event.Iteration,
		})
		h.sendTasksUpdated()

	case loop.EventIterationStarted, loop.EventIterationEnded:
		h.program.Send(LoopStateMsg{
			State:      string(loop.StateRunning),
			Iteration:  event.Iteration,
			ElapsedAt:  h.startAt,
			CurrentMsg: event.Message,
		})

	case loop.EventVerifyStarted:
		h.program.Send(BuildStatusMsg{Running: true})
		h.program.Send(TestStatusMsg{Running: true})

	case loop.EventVerifyPassed:
		h.program.Send(BuildStatusMsg{Passed: true})
		h.program.Send(TestStatusMsg{Passed: true})

	case loop.EventVerifyFailed:
		h.program.Send(BuildStatusMsg{Error: event.Message})
		h.program.Send(TestStatusMsg{Error: event.Message})

	case loop.EventAnalysisStarted:
		h.program.Send(AnalysisStatusMsg{
			Running: true,
			Status:  event.Message,
		})

	case loop.EventAnalysisCompleted:
		h.program.Send(AnalysisStatusMsg{
			Complete: true,
			Status:   "Analysis complete",
		})

	case loop.EventAnalysisFailed:
		errMsg := ""
		if event.Error != nil {
			errMsg = event.Error.Error()
		}
		h.program.Send(AnalysisStatusMsg{
			Error: errMsg,
		})

	case loop.EventError:
		errMsg := ""
		if event.Error != nil {
			errMsg = event.Error.Error()
		} else {
			errMsg = event.Message
		}
		h.program.Send(ErrorMsg{Error: errMsg})
	}
}

// sendTasksUpdated sends a TasksUpdatedMsg with current task status.
func (h *TUIEventHandler) sendTasksUpdated() {
	if h.program == nil {
		return
	}

	h.tasksMu.RLock()
	defer h.tasksMu.RUnlock()

	if h.tasks == nil {
		return
	}

	completed := 0
	for _, t := range h.tasks {
		if t.Status == task.StatusCompleted {
			completed++
		}
	}

	h.program.Send(TasksUpdatedMsg{
		Tasks:     h.tasks,
		Completed: completed,
		Total:     len(h.tasks),
	})
}

// TUIOutputWriter wraps a tea.Program to send agent output as TUI messages.
// It implements io.Writer and buffers partial lines before sending.
type TUIOutputWriter struct {
	program *tea.Program
	buffer  bytes.Buffer
	mu      sync.Mutex
}

// NewTUIOutputWriter creates a new TUIOutputWriter.
func NewTUIOutputWriter(program *tea.Program) *TUIOutputWriter {
	return &TUIOutputWriter{
		program: program,
	}
}

// Write implements io.Writer. It buffers data and sends complete lines
// as AgentOutputMsg messages to the TUI.
func (w *TUIOutputWriter) Write(p []byte) (n int, err error) {
	if w.program == nil {
		return len(p), nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Write to buffer
	n, err = w.buffer.Write(p)
	if err != nil {
		return n, err
	}

	// Process complete lines
	for {
		line, readErr := w.buffer.ReadString('\n')
		if readErr == io.EOF {
			// No complete line yet, put the partial line back
			if len(line) > 0 {
				w.buffer.WriteString(line)
			}
			break
		}
		if readErr != nil {
			return n, readErr
		}

		// Send the line (without trailing newline)
		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		w.program.Send(AgentOutputMsg{Line: line})
	}

	return n, nil
}

// Flush sends any remaining buffered content.
func (w *TUIOutputWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.buffer.Len() > 0 {
		if w.program != nil {
			w.program.Send(AgentOutputMsg{Line: w.buffer.String()})
		}
		w.buffer.Reset()
	}
}

// TUIRunner coordinates running the TUI and Loop together.
type TUIRunner struct {
	model        *Model
	program      *tea.Program
	loop         *loop.Loop
	eventHandler *TUIEventHandler
	outputWriter *TUIOutputWriter
}

// NewTUIRunner creates a new TUIRunner.
func NewTUIRunner(mainLoop *loop.Loop, tasks []*task.Task, sessionInfo SessionInfo) *TUIRunner {
	model := New()

	// Set up session info
	model.SetSessionInfo(
		sessionInfo.ProjectName,
		sessionInfo.AgentName,
		sessionInfo.ModelName,
		sessionInfo.SessionID,
	)

	// Set initial tasks
	model.SetTasks(tasks)

	// Create the program (don't start yet)
	program := tea.NewProgram(model, tea.WithAltScreen())

	// Create event handler and output writer
	eventHandler := NewTUIEventHandler(program)
	eventHandler.SetTasks(tasks)
	outputWriter := NewTUIOutputWriter(program)

	return &TUIRunner{
		model:        model,
		program:      program,
		loop:         mainLoop,
		eventHandler: eventHandler,
		outputWriter: outputWriter,
	}
}

// SessionInfo holds session information for the TUI.
type SessionInfo struct {
	ProjectName string
	AgentName   string
	ModelName   string
	SessionID   string
}

// ConfigureLoop configures the loop with TUI event handling.
func (r *TUIRunner) ConfigureLoop(opts *loop.Options) {
	opts.OnEvent = r.eventHandler.HandleEvent
	opts.LogWriter = r.outputWriter
}

// Run runs the TUI and Loop concurrently.
// The Loop runs in a goroutine while the TUI runs on the main thread.
// Returns when either the Loop completes or the TUI exits.
func (r *TUIRunner) Run(loopRun func() error) error {
	// Channel to communicate loop completion
	loopDone := make(chan error, 1)

	// Run the loop in a goroutine
	go func() {
		err := loopRun()
		loopDone <- err

		// Send quit message to TUI when loop completes
		if r.program != nil {
			r.outputWriter.Flush()
			r.program.Send(QuitMsg{Reason: "Loop completed"})
		}
	}()

	// Run the TUI on main thread (blocking)
	_, tuiErr := r.program.Run()

	// Wait for loop to complete (with timeout to avoid blocking forever)
	select {
	case loopErr := <-loopDone:
		if loopErr != nil {
			return loopErr
		}
	default:
		// Loop still running, that's fine - TUI was quit early
	}

	if tuiErr != nil {
		return tuiErr
	}

	return nil
}

// Program returns the tea.Program for external access (e.g., for SetLoopController).
func (r *TUIRunner) Program() *tea.Program {
	return r.program
}

// Model returns the TUI model for external access.
func (r *TUIRunner) Model() *Model {
	return r.model
}

// LineWriter wraps a writer and sends each line individually.
// This is useful when you need line-by-line output handling.
// Optimized to avoid O(n²) buffer growth with partial lines.
type LineWriter struct {
	w       io.Writer
	onLine  func(string)
	scanner *bufio.Scanner
	buf     []byte // Use slice instead of bytes.Buffer for efficient partial line handling
}

// NewLineWriter creates a writer that calls onLine for each complete line.
func NewLineWriter(onLine func(string)) *LineWriter {
	return &LineWriter{
		onLine: onLine,
		buf:    make([]byte, 0, 256), // Pre-allocate reasonable initial capacity
	}
}

// Write implements io.Writer.
func (lw *LineWriter) Write(p []byte) (n int, err error) {
	n = len(p)

	// Append new data to buffer
	lw.buf = append(lw.buf, p...)

	// Process complete lines
	for {
		idx := bytes.IndexByte(lw.buf, '\n')
		if idx < 0 {
			break // No complete line
		}

		// Extract line (without newline)
		line := string(lw.buf[:idx])
		if lw.onLine != nil {
			lw.onLine(line)
		}

		// Remove processed line from buffer
		lw.buf = lw.buf[idx+1:]
	}

	return n, nil
}

