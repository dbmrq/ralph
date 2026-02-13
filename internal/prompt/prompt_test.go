package prompt

import (
	"testing"
)

func TestPromptLevel_String(t *testing.T) {
	tests := []struct {
		level    PromptLevel
		expected string
	}{
		{LevelBase, "base"},
		{LevelPlatform, "platform"},
		{LevelProject, "project"},
		{PromptLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("PromptLevel.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestVariables_ToMap(t *testing.T) {
	vars := &Variables{
		TaskID:          "TASK-001",
		TaskName:        "Test Task",
		TaskDescription: "A test task",
		TaskStatus:      "pending",
		Iteration:       3,
		ProjectDir:      "/project",
		AgentName:       "auggie",
		Model:           "opus-4",
		SessionID:       "abc123",
		Custom: map[string]string{
			"CUSTOM_VAR": "custom_value",
		},
	}

	m := vars.ToMap()

	expectations := map[string]string{
		"TASK_ID":          "TASK-001",
		"TASK_NAME":        "Test Task",
		"TASK_DESCRIPTION": "A test task",
		"TASK_STATUS":      "pending",
		"ITERATION":        "3",
		"PROJECT_DIR":      "/project",
		"AGENT_NAME":       "auggie",
		"MODEL":            "opus-4",
		"SESSION_ID":       "abc123",
		"CUSTOM_VAR":       "custom_value",
	}

	for key, expected := range expectations {
		if got, ok := m[key]; !ok {
			t.Errorf("Variables.ToMap() missing key %q", key)
		} else if got != expected {
			t.Errorf("Variables.ToMap()[%q] = %q, want %q", key, got, expected)
		}
	}
}

func TestVariables_ToMap_Empty(t *testing.T) {
	vars := &Variables{}
	m := vars.ToMap()

	if got := m["TASK_ID"]; got != "" {
		t.Errorf("Variables.ToMap()[TASK_ID] = %q, want empty string", got)
	}
	if got := m["ITERATION"]; got != "0" {
		t.Errorf("Variables.ToMap()[ITERATION] = %q, want %q", got, "0")
	}
}

func TestSubstitute(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		vars     map[string]string
		expected string
	}{
		{"simple", "Hello ${NAME}!", map[string]string{"NAME": "World"}, "Hello World!"},
		{"multiple", "${A} ${B}", map[string]string{"A": "1", "B": "2"}, "1 2"},
		{"unknown unchanged", "${UNKNOWN}", map[string]string{"NAME": "X"}, "${UNKNOWN}"},
		{"empty vars", "${NAME}", map[string]string{}, "${NAME}"},
		{"nil vars", "${NAME}", nil, "${NAME}"},
		{"no variables", "Hello World", map[string]string{"NAME": "X"}, "Hello World"},
		{"underscore", "${TASK_ID}", map[string]string{"TASK_ID": "T1"}, "T1"},
		{"numbers", "${VAR2}", map[string]string{"VAR2": "val"}, "val"},
		{"multiline", "A: ${X}\nB: ${Y}", map[string]string{"X": "1", "Y": "2"}, "A: 1\nB: 2"},
		{"adjacent", "${A}${B}", map[string]string{"A": "1", "B": "2"}, "12"},
		{"in path", "cd ${DIR}/src", map[string]string{"DIR": "/home"}, "cd /home/src"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Substitute(tt.content, tt.vars)
			if got != tt.expected {
				t.Errorf("Substitute() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSubstituteVariables(t *testing.T) {
	vars := &Variables{
		TaskID:     "TASK-001",
		TaskName:   "Test",
		Iteration:  5,
		ProjectDir: "/proj",
	}

	content := "${TASK_ID}: ${TASK_NAME} #${ITERATION} at ${PROJECT_DIR}"
	expected := "TASK-001: Test #5 at /proj"

	if got := SubstituteVariables(content, vars); got != expected {
		t.Errorf("SubstituteVariables() = %q, want %q", got, expected)
	}
}

func TestSubstituteVariables_Nil(t *testing.T) {
	content := "Hello ${NAME}!"
	if got := SubstituteVariables(content, nil); got != content {
		t.Errorf("SubstituteVariables(nil) = %q, want %q", got, content)
	}
}

func TestLoadError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *LoadError
		expected string
	}{
		{
			name:     "with underlying",
			err:      &LoadError{Path: "/file.txt", Message: "failed", Err: errTest},
			expected: "prompt template /file.txt: failed: test error",
		},
		{
			name:     "without underlying",
			err:      &LoadError{Path: "/file.txt", Message: "not found"},
			expected: "prompt template /file.txt: not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("LoadError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLoadError_Unwrap(t *testing.T) {
	err := &LoadError{Path: "/file.txt", Message: "test", Err: errTest}
	if got := err.Unwrap(); got != errTest {
		t.Errorf("LoadError.Unwrap() = %v, want %v", got, errTest)
	}
}

var errTest = &testError{"test error"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

