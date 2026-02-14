package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestConfigNotFound(t *testing.T) {
	err := ConfigNotFound("/path/to/config.yaml")

	if !errors.Is(err, ErrConfig) {
		t.Error("ConfigNotFound should return ErrConfig")
	}
	if err.Details["path"] != "/path/to/config.yaml" {
		t.Error("Should include path in details")
	}
	if !strings.Contains(err.Suggestion, "ralph init") {
		t.Error("Suggestion should mention init command")
	}
	if err.DocLink == "" {
		t.Error("Should include documentation link")
	}
}

func TestConfigParseError(t *testing.T) {
	parseErr := errors.New("unexpected end of file")
	err := ConfigParseError("/path/config.yaml", parseErr)

	if !errors.Is(err, ErrConfig) {
		t.Error("ConfigParseError should return ErrConfig")
	}
	if !errors.Is(err.Cause, parseErr) {
		t.Error("Should wrap the parse error")
	}
	if !strings.Contains(err.Suggestion, "YAML") {
		t.Error("Suggestion should mention YAML syntax")
	}
}

func TestConfigValidationError(t *testing.T) {
	err := ConfigValidationError("timeout.active", "must be positive", []string{"1h", "2h", "30m"})

	if !errors.Is(err, ErrConfig) {
		t.Error("ConfigValidationError should return ErrConfig")
	}
	if !strings.Contains(err.Suggestion, "1h, 2h, 30m") {
		t.Error("Suggestion should list valid options")
	}
	if err.Details["field"] != "timeout.active" {
		t.Error("Should include field name")
	}
}

func TestConfigValidationError_NoOptions(t *testing.T) {
	err := ConfigValidationError("agent.default", "invalid agent", nil)

	if !strings.Contains(err.Suggestion, "Fix the") {
		t.Error("Should still provide suggestion without options")
	}
}

func TestProjectNotInitialized(t *testing.T) {
	err := ProjectNotInitialized("/home/user/project")

	if !errors.Is(err, ErrConfig) {
		t.Error("ProjectNotInitialized should return ErrConfig")
	}
	if !strings.Contains(err.Suggestion, "ralph init") {
		t.Error("Suggestion should mention init command")
	}
	if err.Details["directory"] != "/home/user/project" {
		t.Error("Should include project directory")
	}
}

func TestNoTasksFound(t *testing.T) {
	err := NoTasksFound("/project")

	if !errors.Is(err, ErrTask) {
		t.Error("NoTasksFound should return ErrTask")
	}
	if !strings.Contains(err.Suggestion, "--tasks") {
		t.Error("Suggestion should mention --tasks flag")
	}
}

func TestTaskNotFound(t *testing.T) {
	err := TaskNotFound("TASK-001")

	if !errors.Is(err, ErrTask) {
		t.Error("TaskNotFound should return ErrTask")
	}
	if !strings.Contains(err.Message, "TASK-001") {
		t.Error("Error message should contain task ID")
	}
}

func TestAllTasksComplete(t *testing.T) {
	err := AllTasksComplete()

	if !errors.Is(err, ErrTask) {
		t.Error("AllTasksComplete should return ErrTask")
	}
	if !strings.Contains(err.Message, "complete") {
		t.Error("Message should indicate completion")
	}
}

func TestSessionNotFound(t *testing.T) {
	err := SessionNotFound("abc123")

	if !errors.Is(err, ErrNotFound) {
		t.Error("SessionNotFound should return ErrNotFound")
	}
	if err.Details["session_id"] != "abc123" {
		t.Error("Should include session ID")
	}
	if !strings.Contains(err.Suggestion, "ralph run") {
		t.Error("Suggestion should mention how to start a new session")
	}
}
