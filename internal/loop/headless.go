// Package loop provides the main execution loop for ralph.
// This file implements LOOP-006: headless execution mode for CI/GitHub Actions.
// It provides the same functionality as TUI mode but with structured output
// suitable for non-interactive environments.
package loop

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// OutputFormat defines the output format for headless mode.
type OutputFormat string

const (
	// OutputFormatText is the default human-readable text output.
	OutputFormatText OutputFormat = "text"
	// OutputFormatJSON produces structured JSON output.
	OutputFormatJSON OutputFormat = "json"
)

// HeadlessConfig configures the headless runner.
type HeadlessConfig struct {
	// OutputFormat is the format for output (text or json).
	OutputFormat OutputFormat
	// Writer is the output writer (defaults to stdout).
	Writer io.Writer
	// ErrorWriter is the error writer (defaults to stderr).
	ErrorWriter io.Writer
	// Verbose enables detailed logging.
	Verbose bool
}

// DefaultHeadlessConfig returns a default configuration.
func DefaultHeadlessConfig() *HeadlessConfig {
	return &HeadlessConfig{
		OutputFormat: OutputFormatText,
		Verbose:      false,
	}
}

// HeadlessRunner executes the ralph loop in headless mode.
// It provides the same functionality as TUI mode without interactive UI.
type HeadlessRunner struct {
	config     *HeadlessConfig
	startTime  time.Time
	jsonEvents []JSONEvent
}

// NewHeadlessRunner creates a new headless runner with the given configuration.
func NewHeadlessRunner(config *HeadlessConfig) *HeadlessRunner {
	if config == nil {
		config = DefaultHeadlessConfig()
	}
	return &HeadlessRunner{
		config:     config,
		startTime:  time.Now(),
		jsonEvents: []JSONEvent{},
	}
}

// JSONEvent represents a single event in JSON output format.
type JSONEvent struct {
	Timestamp string    `json:"timestamp"`
	Type      EventType `json:"type"`
	TaskID    string    `json:"task_id,omitempty"`
	TaskName  string    `json:"task_name,omitempty"`
	Iteration int       `json:"iteration,omitempty"`
	Message   string    `json:"message,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// JSONOutput is the complete JSON output for headless mode.
type JSONOutput struct {
	SessionID     string            `json:"session_id"`
	StartTime     string            `json:"start_time"`
	EndTime       string            `json:"end_time"`
	Duration      string            `json:"duration"`
	FinalState    string            `json:"final_state"`
	TasksTotal    int               `json:"tasks_total"`
	TasksComplete int               `json:"tasks_complete"`
	TasksFailed   int               `json:"tasks_failed"`
	TasksSkipped  int               `json:"tasks_skipped"`
	Events        []JSONEvent       `json:"events"`
	Summary       map[string]string `json:"summary,omitempty"`
}

// HandleEvent processes a loop event for headless output.
// This is designed to be used as the loop.Options.OnEvent callback.
func (h *HeadlessRunner) HandleEvent(event Event) {
	switch h.config.OutputFormat {
	case OutputFormatJSON:
		h.handleEventJSON(event)
	default:
		h.handleEventText(event)
	}
}

// handleEventText outputs an event in human-readable text format.
func (h *HeadlessRunner) handleEventText(event Event) {
	w := h.config.Writer
	if w == nil {
		return
	}

	elapsed := time.Since(h.startTime).Round(time.Second)
	prefix := fmt.Sprintf("[%s]", formatElapsed(elapsed))

	var message string
	switch event.Type {
	case EventLoopStarted:
		message = fmt.Sprintf("🚀 %s Loop started", prefix)
	case EventLoopCompleted:
		message = fmt.Sprintf("✅ %s Loop completed - all tasks done!", prefix)
	case EventLoopFailed:
		message = fmt.Sprintf("❌ %s Loop failed: %s", prefix, h.errorStr(event.Error))
	case EventLoopPaused:
		message = fmt.Sprintf("⏸️  %s Loop paused: %s", prefix, event.Message)

	case EventAnalysisStarted:
		message = fmt.Sprintf("🔍 %s %s", prefix, event.Message)
	case EventAnalysisCompleted:
		message = fmt.Sprintf("✓  %s Project analysis complete", prefix)
	case EventAnalysisFailed:
		message = fmt.Sprintf("❌ %s Analysis failed: %s", prefix, h.errorStr(event.Error))

	case EventTaskStarted:
		message = fmt.Sprintf("▶️  %s Starting task: %s", prefix, event.TaskName)
	case EventTaskCompleted:
		message = fmt.Sprintf("✅ %s Task completed: %s", prefix, event.TaskName)
	case EventTaskSkipped:
		message = fmt.Sprintf("⏭️  %s Task skipped: %s - %s", prefix, event.TaskName, event.Message)
	case EventTaskFailed:
		message = fmt.Sprintf("❌ %s Task failed: %s - %s", prefix, event.TaskName, h.errorStr(event.Error))

	case EventIterationStarted:
		if h.config.Verbose {
			message = fmt.Sprintf("   %s Iteration %d started", prefix, event.Iteration)
		}
	case EventIterationEnded:
		if h.config.Verbose {
			message = fmt.Sprintf("   %s Iteration %d: %s", prefix, event.Iteration, event.Message)
		}

	case EventVerifyStarted:
		if h.config.Verbose {
			message = fmt.Sprintf("   %s Running verification", prefix)
		}
	case EventVerifyPassed:
		message = fmt.Sprintf("✓  %s Verification passed", prefix)
	case EventVerifyFailed:
		message = fmt.Sprintf("⚠️  %s Verification failed: %s", prefix, event.Message)

	default:
		if h.config.Verbose {
			message = fmt.Sprintf("   %s %s: %s", prefix, event.Type, event.Message)
		}
	}

	if message != "" {
		fmt.Fprintln(w, message)
	}
}

// handleEventJSON collects events for JSON output.
func (h *HeadlessRunner) handleEventJSON(event Event) {
	jsonEvent := JSONEvent{
		Timestamp: event.Timestamp.Format(time.RFC3339),
		Type:      event.Type,
		TaskID:    event.TaskID,
		TaskName:  event.TaskName,
		Iteration: event.Iteration,
		Message:   event.Message,
	}
	if event.Error != nil {
		jsonEvent.Error = event.Error.Error()
	}
	h.jsonEvents = append(h.jsonEvents, jsonEvent)
}

// WriteJSONOutput writes the complete JSON output.
// This should be called after the loop completes.
func (h *HeadlessRunner) WriteJSONOutput(ctx *LoopContext) error {
	if h.config.OutputFormat != OutputFormatJSON {
		return nil
	}

	w := h.config.Writer
	if w == nil {
		return nil
	}

	endTime := time.Now()
	output := JSONOutput{
		SessionID:     ctx.SessionID,
		StartTime:     h.startTime.Format(time.RFC3339),
		EndTime:       endTime.Format(time.RFC3339),
		Duration:      endTime.Sub(h.startTime).Round(time.Second).String(),
		FinalState:    string(ctx.State),
		TasksTotal:    ctx.TasksCompleted + ctx.TasksFailed + ctx.TasksSkipped,
		TasksComplete: ctx.TasksCompleted,
		TasksFailed:   ctx.TasksFailed,
		TasksSkipped:  ctx.TasksSkipped,
		Events:        h.jsonEvents,
	}

	// Add summary based on final state
	output.Summary = make(map[string]string)
	switch ctx.State {
	case StateCompleted:
		output.Summary["status"] = "success"
		output.Summary["message"] = "All tasks completed successfully"
	case StateFailed:
		output.Summary["status"] = "failure"
		output.Summary["message"] = ctx.LastError
	case StatePaused:
		output.Summary["status"] = "paused"
		output.Summary["message"] = fmt.Sprintf("Session paused, resume with: ralph run --continue %s", ctx.SessionID)
	default:
		output.Summary["status"] = string(ctx.State)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// PrintSummary prints a summary at the end of the run.
// For text output, this provides a final status summary.
func (h *HeadlessRunner) PrintSummary(ctx *LoopContext) {
	if h.config.OutputFormat == OutputFormatJSON {
		return // JSON output handles its own summary
	}

	w := h.config.Writer
	if w == nil {
		return
	}

	elapsed := time.Since(h.startTime).Round(time.Second)

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, strings.Repeat("─", 60))
	fmt.Fprintf(w, "Session: %s\n", ctx.SessionID)
	fmt.Fprintf(w, "Duration: %s\n", elapsed)
	fmt.Fprintf(w, "Tasks: %d completed, %d failed, %d skipped\n",
		ctx.TasksCompleted, ctx.TasksFailed, ctx.TasksSkipped)

	switch ctx.State {
	case StateCompleted:
		fmt.Fprintln(w, "Status: ✅ All tasks completed successfully")
	case StateFailed:
		fmt.Fprintf(w, "Status: ❌ Failed - %s\n", ctx.LastError)
	case StatePaused:
		fmt.Fprintf(w, "Status: ⏸️  Paused - resume with: ralph run --continue %s\n", ctx.SessionID)
	default:
		fmt.Fprintf(w, "Status: %s\n", ctx.State)
	}
	fmt.Fprintln(w, strings.Repeat("─", 60))
}

// errorStr safely converts an error to string.
func (h *HeadlessRunner) errorStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// formatElapsed formats duration as MM:SS or HH:MM:SS.
func formatElapsed(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

