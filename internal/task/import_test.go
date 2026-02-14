package task

import (
	"os"
	"strings"
	"testing"
)

func TestNewImporter(t *testing.T) {
	imp := NewImporter()
	if imp.idPrefix != "TASK" {
		t.Errorf("expected idPrefix 'TASK', got %q", imp.idPrefix)
	}
	if imp.idCounter != 1 {
		t.Errorf("expected idCounter 1, got %d", imp.idCounter)
	}
}

func TestImporter_SetIDPrefix(t *testing.T) {
	imp := NewImporter()
	imp.SetIDPrefix("FEAT")
	if imp.idPrefix != "FEAT" {
		t.Errorf("expected idPrefix 'FEAT', got %q", imp.idPrefix)
	}
}

func TestImporter_SetIDStart(t *testing.T) {
	imp := NewImporter()
	imp.SetIDStart(100)
	if imp.idCounter != 100 {
		t.Errorf("expected idCounter 100, got %d", imp.idCounter)
	}
}

func TestImporter_GenerateID(t *testing.T) {
	imp := NewImporter()

	id1 := imp.generateID()
	if id1 != "TASK-001" {
		t.Errorf("expected 'TASK-001', got %q", id1)
	}

	id2 := imp.generateID()
	if id2 != "TASK-002" {
		t.Errorf("expected 'TASK-002', got %q", id2)
	}
}

func TestImporter_ImportFromMarkdown_BasicTasks(t *testing.T) {
	input := `# Tasks

- [ ] TASK-001: First task
- [ ] TASK-002: Second task
- [x] TASK-003: Completed task
`
	imp := NewImporter()
	result, err := imp.ImportFromMarkdown(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result.Tasks))
	}

	// Check first task
	if result.Tasks[0].ID != "TASK-001" {
		t.Errorf("expected ID 'TASK-001', got %q", result.Tasks[0].ID)
	}
	if result.Tasks[0].Name != "First task" {
		t.Errorf("expected name 'First task', got %q", result.Tasks[0].Name)
	}
	if result.Tasks[0].Status != StatusPending {
		t.Errorf("expected status pending, got %v", result.Tasks[0].Status)
	}
	if result.Tasks[0].Order != 1 {
		t.Errorf("expected order 1, got %d", result.Tasks[0].Order)
	}

	// Check completed task
	if result.Tasks[2].Status != StatusCompleted {
		t.Errorf("expected status completed, got %v", result.Tasks[2].Status)
	}
}

func TestImporter_ImportFromMarkdown_WithContext(t *testing.T) {
	input := `- [ ] TASK-001: First task
  > Goal: Complete the first task
  > Tests: Not required
  > Reference: See docs/spec.md
`
	imp := NewImporter()
	result, err := imp.ImportFromMarkdown(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result.Tasks))
	}

	task := result.Tasks[0]
	if !strings.Contains(task.Description, "Goal: Complete the first task") {
		t.Errorf("expected description to contain goal, got %q", task.Description)
	}
	if !strings.Contains(task.Description, "Tests: Not required") {
		t.Errorf("expected description to contain tests note, got %q", task.Description)
	}
}

func TestImporter_ImportFromMarkdown_AutoGenerateID(t *testing.T) {
	input := `- [ ] Task without ID
- [ ] Another task without ID
`
	imp := NewImporter()
	result, err := imp.ImportFromMarkdown(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(result.Tasks))
	}

	if result.Tasks[0].ID != "TASK-001" {
		t.Errorf("expected ID 'TASK-001', got %q", result.Tasks[0].ID)
	}
	if result.Tasks[1].ID != "TASK-002" {
		t.Errorf("expected ID 'TASK-002', got %q", result.Tasks[1].ID)
	}

	// Should have warnings about generated IDs
	if len(result.Warnings) != 2 {
		t.Errorf("expected 2 warnings, got %d", len(result.Warnings))
	}
}

func TestImporter_ImportFromMarkdown_LowercaseX(t *testing.T) {
	input := `- [x] TASK-001: Lowercase completed
- [X] TASK-002: Uppercase completed
`
	imp := NewImporter()
	result, err := imp.ImportFromMarkdown(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, task := range result.Tasks {
		if task.Status != StatusCompleted {
			t.Errorf("task %d: expected completed, got %v", i, task.Status)
		}
	}
}

func TestImporter_ImportFromMarkdown_EmptyInput(t *testing.T) {
	input := ""
	imp := NewImporter()
	result, err := imp.ImportFromMarkdown(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(result.Tasks))
	}
}

func TestImporter_ImportFromPlainText_Numbered(t *testing.T) {
	input := `1. TASK-001: First task
2. TASK-002: Second task
3) Third task without ID
`
	imp := NewImporter()
	result, err := imp.ImportFromPlainText(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result.Tasks))
	}

	if result.Tasks[0].ID != "TASK-001" {
		t.Errorf("expected ID 'TASK-001', got %q", result.Tasks[0].ID)
	}
	if result.Tasks[2].ID != "TASK-001" {
		t.Errorf("expected generated ID 'TASK-001', got %q", result.Tasks[2].ID)
	}
	if result.Tasks[2].Name != "Third task without ID" {
		t.Errorf("expected name 'Third task without ID', got %q", result.Tasks[2].Name)
	}
}

func TestImporter_ImportFromPlainText_Bulleted(t *testing.T) {
	input := `* First task
• Second task
- Third task
`
	imp := NewImporter()
	result, err := imp.ImportFromPlainText(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result.Tasks))
	}

	// All should have generated IDs
	if len(result.Warnings) != 3 {
		t.Errorf("expected 3 warnings, got %d", len(result.Warnings))
	}
}

func TestImporter_ImportFromPlainText_SkipsMarkdown(t *testing.T) {
	input := `- [ ] TASK-001: Markdown task
- [x] TASK-002: Completed markdown task
* Normal bullet
`
	imp := NewImporter()
	result, err := imp.ImportFromPlainText(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only parse the normal bullet, not the markdown checkboxes
	if len(result.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result.Tasks))
	}

	if result.Tasks[0].Name != "Normal bullet" {
		t.Errorf("expected 'Normal bullet', got %q", result.Tasks[0].Name)
	}
}

func TestImporter_ImportFromString(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		format   ImportFormat
		expected int
	}{
		{
			name:     "markdown",
			content:  "- [ ] TASK-001: Test\n- [x] TASK-002: Done",
			format:   FormatMarkdown,
			expected: 2,
		},
		{
			name:     "plaintext",
			content:  "1. First\n2. Second",
			format:   FormatPlainText,
			expected: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			imp := NewImporter()
			result, err := imp.ImportFromString(tc.content, tc.format)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Tasks) != tc.expected {
				t.Errorf("expected %d tasks, got %d", tc.expected, len(result.Tasks))
			}
		})
	}
}

func TestImporter_ImportFromString_UnsupportedFormat(t *testing.T) {
	imp := NewImporter()
	_, err := imp.ImportFromString("test", ImportFormat("unknown"))
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected ImportFormat
	}{
		{
			name:     "markdown with unchecked",
			content:  "- [ ] TASK-001: Test",
			expected: FormatMarkdown,
		},
		{
			name:     "markdown with checked",
			content:  "- [x] TASK-001: Test",
			expected: FormatMarkdown,
		},
		{
			name:     "plain numbered",
			content:  "1. First task\n2. Second task",
			expected: FormatPlainText,
		},
		{
			name:     "plain bulleted",
			content:  "* First task\n* Second task",
			expected: FormatPlainText,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			format := DetectFormat(tc.content)
			if format != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, format)
			}
		})
	}
}

func TestImporter_ImportAuto(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "auto detect markdown",
			content:  "- [ ] TASK-001: Test\n- [x] TASK-002: Done",
			expected: 2,
		},
		{
			name:     "auto detect plaintext",
			content:  "1. First\n2. Second",
			expected: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			imp := NewImporter()
			result, err := imp.ImportAuto(tc.content)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Tasks) != tc.expected {
				t.Errorf("expected %d tasks, got %d", tc.expected, len(result.Tasks))
			}
		})
	}
}

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name        string
		description string
		expected    map[string]string
	}{
		{
			name:        "goal only",
			description: "Goal: Complete the task",
			expected:    map[string]string{"goal": "Complete the task"},
		},
		{
			name:        "multiple keys",
			description: "Goal: Complete it\nTests: Not required\nBuild: Skip",
			expected:    map[string]string{"goal": "Complete it", "tests": "Not required", "build": "Skip"},
		},
		{
			name:        "reference and notes",
			description: "Reference: docs/spec.md\nNotes: Be careful",
			expected:    map[string]string{"reference": "docs/spec.md", "notes": "Be careful"},
		},
		{
			name:        "custom key preserved",
			description: "CustomKey: custom value",
			expected:    map[string]string{"CustomKey": "custom value"},
		},
		{
			name:        "empty description",
			description: "",
			expected:    map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractMetadata(tc.description)
			if len(result) != len(tc.expected) {
				t.Errorf("expected %d keys, got %d", len(tc.expected), len(result))
			}
			for k, v := range tc.expected {
				if result[k] != v {
					t.Errorf("key %q: expected %q, got %q", k, v, result[k])
				}
			}
		})
	}
}

func TestTask_ParseTaskMetadata(t *testing.T) {
	task := NewTask("TEST-001", "Test task", "Goal: Complete it\nTests: Not required")
	task.ParseTaskMetadata()

	goal, ok := task.GetMetadata("goal")
	if !ok || goal != "Complete it" {
		t.Errorf("expected goal 'Complete it', got %q (ok=%v)", goal, ok)
	}

	tests, ok := task.GetMetadata("tests")
	if !ok || tests != "Not required" {
		t.Errorf("expected tests 'Not required', got %q (ok=%v)", tests, ok)
	}
}

func TestTask_ParseTaskMetadata_EmptyDescription(t *testing.T) {
	task := NewTask("TEST-001", "Test task", "")
	task.ParseTaskMetadata()

	if len(task.Metadata) != 0 {
		t.Errorf("expected empty metadata, got %v", task.Metadata)
	}
}

func TestImporter_ImportFromMarkdown_MixedContent(t *testing.T) {
	input := `# Task List

Some introductory text.

## Phase 1

- [ ] INIT-001: Initialize project
  > Goal: Setup the project structure
  > Tests: Not required (setup task)

- [x] INIT-002: Add dependencies
  > Goal: Add required packages

## Phase 2

- [ ] FEAT-001: Add feature
  > Goal: Implement the new feature
  > Reference: specs/feature.md
`
	imp := NewImporter()
	result, err := imp.ImportFromMarkdown(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result.Tasks))
	}

	// Check INIT-001
	if result.Tasks[0].ID != "INIT-001" {
		t.Errorf("expected ID 'INIT-001', got %q", result.Tasks[0].ID)
	}
	if !strings.Contains(result.Tasks[0].Description, "Tests: Not required") {
		t.Errorf("expected description to contain tests note")
	}

	// Check INIT-002 is completed
	if result.Tasks[1].Status != StatusCompleted {
		t.Errorf("expected INIT-002 to be completed")
	}

	// Check FEAT-001
	if result.Tasks[2].ID != "FEAT-001" {
		t.Errorf("expected ID 'FEAT-001', got %q", result.Tasks[2].ID)
	}
}

func TestImporter_ImportFromMarkdown_TaskIDVariations(t *testing.T) {
	input := `- [ ] TASK-001: With colon
- [ ] TASK-002 Without colon
- [ ] A-1: Single digit
- [ ] ABC-999: Triple digit
`
	imp := NewImporter()
	result, err := imp.ImportFromMarkdown(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(result.Tasks))
	}

	expectedIDs := []string{"TASK-001", "TASK-002", "A-1", "ABC-999"}
	for i, expected := range expectedIDs {
		if result.Tasks[i].ID != expected {
			t.Errorf("task %d: expected ID %q, got %q", i, expected, result.Tasks[i].ID)
		}
	}
}

func TestImporter_ImportToStore(t *testing.T) {
	// Create a temporary store
	dir := t.TempDir()
	store := NewStoreInDir(dir)

	content := `- [ ] TASK-001: First task
  > Goal: Do the first thing
- [ ] TASK-002: Second task
`

	// Create a temp file with the content
	tempFile := dir + "/tasks.md"
	if err := writeTestFile(tempFile, content); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	imp := NewImporter()
	result, err := imp.ImportToStore(store, tempFile, FormatMarkdown)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 2 {
		t.Errorf("expected 2 tasks in result, got %d", len(result.Tasks))
	}

	// Check store has the tasks
	if store.Count() != 2 {
		t.Errorf("expected 2 tasks in store, got %d", store.Count())
	}

	// Check metadata was parsed
	task, _ := store.Get("TASK-001")
	goal, ok := task.GetMetadata("goal")
	if !ok || goal != "Do the first thing" {
		t.Errorf("expected goal metadata, got %q (ok=%v)", goal, ok)
	}
}

func TestImporter_ImportFromFile(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] FILE-001: Test file import
  > Goal: Verify file reading works
`
	tempFile := dir + "/test.md"
	if err := writeTestFile(tempFile, content); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	imp := NewImporter()
	result, err := imp.ImportFromFile(tempFile, FormatMarkdown)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(result.Tasks))
	}

	if result.Tasks[0].ID != "FILE-001" {
		t.Errorf("expected ID 'FILE-001', got %q", result.Tasks[0].ID)
	}
}

func TestImporter_ImportFromFile_NotFound(t *testing.T) {
	imp := NewImporter()
	_, err := imp.ImportFromFile("/nonexistent/path/file.md", FormatMarkdown)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestImporter_ImportFileAuto(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] AUTO-001: Auto detected markdown
- [x] AUTO-002: Completed
`
	tempFile := dir + "/auto.md"
	if err := writeTestFile(tempFile, content); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	imp := NewImporter()
	result, err := imp.ImportFileAuto(tempFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(result.Tasks))
	}
}

func TestImporter_ImportFileAuto_NotFound(t *testing.T) {
	imp := NewImporter()
	_, err := imp.ImportFileAuto("/nonexistent/path/file.md")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestImporter_ImportFromPlainText_WithIDs(t *testing.T) {
	input := `1. FEAT-001: Feature one
2. FEAT-002: Feature two
3. BUG-100: Bug fix
`
	imp := NewImporter()
	result, err := imp.ImportFromPlainText(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result.Tasks))
	}

	expectedIDs := []string{"FEAT-001", "FEAT-002", "BUG-100"}
	for i, expected := range expectedIDs {
		if result.Tasks[i].ID != expected {
			t.Errorf("task %d: expected ID %q, got %q", i, expected, result.Tasks[i].ID)
		}
	}

	// Should have no warnings since all IDs were provided
	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
	}
}

func TestImporter_CustomIDPrefix(t *testing.T) {
	input := `- [ ] First task without ID
- [ ] Second task without ID
`
	imp := NewImporter()
	imp.SetIDPrefix("CUSTOM")
	imp.SetIDStart(100)

	result, err := imp.ImportFromMarkdown(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Tasks[0].ID != "CUSTOM-100" {
		t.Errorf("expected ID 'CUSTOM-100', got %q", result.Tasks[0].ID)
	}
	if result.Tasks[1].ID != "CUSTOM-101" {
		t.Errorf("expected ID 'CUSTOM-101', got %q", result.Tasks[1].ID)
	}
}

// writeTestFile is a helper for tests
func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
