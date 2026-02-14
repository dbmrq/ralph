package loop

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewHeadlessRunner(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		runner := NewHeadlessRunner(nil)
		if runner == nil {
			t.Fatal("expected non-nil runner")
		}
		if runner.config.OutputFormat != OutputFormatText {
			t.Errorf("expected text output format, got %s", runner.config.OutputFormat)
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &HeadlessConfig{
			OutputFormat: OutputFormatJSON,
			Verbose:      true,
		}
		runner := NewHeadlessRunner(cfg)
		if runner.config.OutputFormat != OutputFormatJSON {
			t.Errorf("expected JSON output format, got %s", runner.config.OutputFormat)
		}
		if !runner.config.Verbose {
			t.Error("expected verbose to be true")
		}
	})
}

func TestDefaultHeadlessConfig(t *testing.T) {
	cfg := DefaultHeadlessConfig()
	if cfg.OutputFormat != OutputFormatText {
		t.Errorf("expected text output format, got %s", cfg.OutputFormat)
	}
	if cfg.Verbose {
		t.Error("expected verbose to be false by default")
	}
}

func TestHeadlessRunner_HandleEvent_Text(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &HeadlessConfig{
		OutputFormat: OutputFormatText,
		Writer:       buf,
	}
	runner := NewHeadlessRunner(cfg)

	tests := []struct {
		name       string
		event      Event
		wantOutput string
	}{
		{
			name:       "loop started",
			event:      Event{Type: EventLoopStarted},
			wantOutput: "🚀",
		},
		{
			name:       "loop completed",
			event:      Event{Type: EventLoopCompleted},
			wantOutput: "✅",
		},
		{
			name:       "loop failed",
			event:      Event{Type: EventLoopFailed, Error: errors.New("test error")},
			wantOutput: "test error",
		},
		{
			name:       "task started",
			event:      Event{Type: EventTaskStarted, TaskName: "TASK-001"},
			wantOutput: "TASK-001",
		},
		{
			name:       "task completed",
			event:      Event{Type: EventTaskCompleted, TaskName: "TASK-002"},
			wantOutput: "TASK-002",
		},
		{
			name:       "verify passed",
			event:      Event{Type: EventVerifyPassed},
			wantOutput: "Verification passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			runner.HandleEvent(tt.event)
			if !strings.Contains(buf.String(), tt.wantOutput) {
				t.Errorf("output = %q, want to contain %q", buf.String(), tt.wantOutput)
			}
		})
	}
}

func TestHeadlessRunner_HandleEvent_JSON(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &HeadlessConfig{
		OutputFormat: OutputFormatJSON,
		Writer:       buf,
	}
	runner := NewHeadlessRunner(cfg)

	event := Event{
		Type:      EventTaskStarted,
		TaskID:    "TASK-001",
		TaskName:  "Test Task",
		Timestamp: time.Now(),
	}

	runner.HandleEvent(event)

	// JSON mode collects events, doesn't write immediately
	if buf.Len() > 0 {
		t.Error("JSON mode should not write immediately")
	}

	if len(runner.jsonEvents) != 1 {
		t.Errorf("expected 1 collected event, got %d", len(runner.jsonEvents))
	}

	if runner.jsonEvents[0].TaskID != "TASK-001" {
		t.Errorf("expected task ID TASK-001, got %s", runner.jsonEvents[0].TaskID)
	}
}

func TestHeadlessRunner_WriteJSONOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &HeadlessConfig{
		OutputFormat: OutputFormatJSON,
		Writer:       buf,
	}
	runner := NewHeadlessRunner(cfg)

	// Add some events
	runner.jsonEvents = []JSONEvent{
		{Type: EventLoopStarted, Timestamp: time.Now().Format(time.RFC3339)},
		{Type: EventTaskStarted, TaskID: "TASK-001", TaskName: "Test Task"},
		{Type: EventLoopCompleted},
	}

	ctx := &LoopContext{
		SessionID:      "test-session",
		State:          StateCompleted,
		TasksCompleted: 1,
	}

	err := runner.WriteJSONOutput(ctx)
	if err != nil {
		t.Fatalf("WriteJSONOutput() error = %v", err)
	}

	var output JSONOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v", err)
	}

	if output.SessionID != "test-session" {
		t.Errorf("SessionID = %q, want %q", output.SessionID, "test-session")
	}
	if output.TasksComplete != 1 {
		t.Errorf("TasksComplete = %d, want 1", output.TasksComplete)
	}
	if len(output.Events) != 3 {
		t.Errorf("Events count = %d, want 3", len(output.Events))
	}
}

func TestHeadlessRunner_PrintSummary(t *testing.T) {
	tests := []struct {
		name       string
		ctx        *LoopContext
		wantOutput string
	}{
		{
			name: "completed state",
			ctx: &LoopContext{
				SessionID:      "test-session",
				State:          StateCompleted,
				TasksCompleted: 3,
			},
			wantOutput: "completed successfully",
		},
		{
			name: "failed state",
			ctx: &LoopContext{
				SessionID:   "test-session",
				State:       StateFailed,
				TasksFailed: 1,
				LastError:   "something went wrong",
			},
			wantOutput: "something went wrong",
		},
		{
			name: "paused state",
			ctx: &LoopContext{
				SessionID: "test-session",
				State:     StatePaused,
			},
			wantOutput: "--continue test-session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			cfg := &HeadlessConfig{
				OutputFormat: OutputFormatText,
				Writer:       buf,
			}
			runner := NewHeadlessRunner(cfg)
			runner.PrintSummary(tt.ctx)

			if !strings.Contains(buf.String(), tt.wantOutput) {
				t.Errorf("output = %q, want to contain %q", buf.String(), tt.wantOutput)
			}
		})
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "zero",
			duration: 0,
			want:     "00:00",
		},
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			want:     "00:45",
		},
		{
			name:     "minutes and seconds",
			duration: 5*time.Minute + 30*time.Second,
			want:     "05:30",
		},
		{
			name:     "hours minutes seconds",
			duration: 2*time.Hour + 15*time.Minute + 45*time.Second,
			want:     "02:15:45",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatElapsed(tt.duration)
			if got != tt.want {
				t.Errorf("formatElapsed(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestHeadlessRunner_VerboseMode(t *testing.T) {
	// Non-verbose mode should suppress iteration events
	t.Run("non-verbose suppresses iteration events", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cfg := &HeadlessConfig{
			OutputFormat: OutputFormatText,
			Writer:       buf,
			Verbose:      false,
		}
		runner := NewHeadlessRunner(cfg)

		runner.HandleEvent(Event{Type: EventIterationStarted, Iteration: 1})

		if buf.Len() > 0 {
			t.Error("iteration events should be suppressed in non-verbose mode")
		}
	})

	// Verbose mode should include iteration events
	t.Run("verbose includes iteration events", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cfg := &HeadlessConfig{
			OutputFormat: OutputFormatText,
			Writer:       buf,
			Verbose:      true,
		}
		runner := NewHeadlessRunner(cfg)

		runner.HandleEvent(Event{Type: EventIterationStarted, Iteration: 1})

		if !strings.Contains(buf.String(), "Iteration 1") {
			t.Errorf("expected iteration event in verbose mode, got: %q", buf.String())
		}
	})
}
