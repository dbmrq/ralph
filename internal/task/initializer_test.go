package task

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewInitializer(t *testing.T) {
	init := NewInitializer("/project", nil)
	if init == nil {
		t.Fatal("NewInitializer returned nil")
	}
	if init.ProjectDir != "/project" {
		t.Errorf("ProjectDir = %q, want %q", init.ProjectDir, "/project")
	}
	if init.Agent != nil {
		t.Error("Agent should be nil")
	}
}

func TestDetectTaskList_NoTasks(t *testing.T) {
	// Create temp dir with no task files
	dir := t.TempDir()
	init := NewInitializer(dir, nil)

	detection := init.DetectTaskList()
	if detection != nil {
		t.Errorf("DetectTaskList() = %v, want nil", detection)
	}
}

func TestDetectTaskList_FoundTasksMD(t *testing.T) {
	dir := t.TempDir()
	tasksFile := filepath.Join(dir, "TASKS.md")
	content := `# Tasks
- [ ] TASK-001: First task
- [ ] TASK-002: Second task
- [x] TASK-003: Completed task
`
	if err := os.WriteFile(tasksFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	init := NewInitializer(dir, nil)
	detection := init.DetectTaskList()

	if detection == nil {
		t.Fatal("DetectTaskList() returned nil, expected detection")
	}
	if !detection.Detected {
		t.Error("Detected = false, want true")
	}
	if detection.Path != "TASKS.md" {
		t.Errorf("Path = %q, want %q", detection.Path, "TASKS.md")
	}
	if detection.Format != "markdown" {
		t.Errorf("Format = %q, want %q", detection.Format, "markdown")
	}
	if detection.TaskCount != 3 {
		t.Errorf("TaskCount = %d, want %d", detection.TaskCount, 3)
	}
}

func TestDetectTaskList_FoundTasksJSON(t *testing.T) {
	dir := t.TempDir()
	ralphDir := filepath.Join(dir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatal(err)
	}
	tasksFile := filepath.Join(ralphDir, "tasks.json")
	content := `{"tasks": [{"id": "TASK-001"}, {"id": "TASK-002"}]}`
	if err := os.WriteFile(tasksFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	init := NewInitializer(dir, nil)
	detection := init.DetectTaskList()

	if detection == nil {
		t.Fatal("DetectTaskList() returned nil")
	}
	if detection.Path != ".ralph/tasks.json" {
		t.Errorf("Path = %q, want %q", detection.Path, ".ralph/tasks.json")
	}
	if detection.Format != "json" {
		t.Errorf("Format = %q, want %q", detection.Format, "json")
	}
	if detection.TaskCount != 2 {
		t.Errorf("TaskCount = %d, want %d", detection.TaskCount, 2)
	}
}

func TestImportFromDetection_Markdown(t *testing.T) {
	dir := t.TempDir()
	tasksFile := filepath.Join(dir, "TASKS.md")
	content := `- [ ] TASK-001: First task
  > Description for task 1
- [ ] TASK-002: Second task
`
	if err := os.WriteFile(tasksFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	init := NewInitializer(dir, nil)
	detection := init.DetectTaskList()
	if detection == nil {
		t.Fatal("DetectTaskList() returned nil")
	}

	result, err := init.ImportFromDetection(detection)
	if err != nil {
		t.Fatalf("ImportFromDetection() error = %v", err)
	}
	if len(result.Tasks) != 2 {
		t.Errorf("len(Tasks) = %d, want %d", len(result.Tasks), 2)
	}
	if result.Tasks[0].ID != "TASK-001" {
		t.Errorf("Tasks[0].ID = %q, want %q", result.Tasks[0].ID, "TASK-001")
	}
	if result.Tasks[0].Name != "First task" {
		t.Errorf("Tasks[0].Name = %q, want %q", result.Tasks[0].Name, "First task")
	}
}

func TestImportFromFile(t *testing.T) {
	dir := t.TempDir()
	tasksFile := filepath.Join(dir, "mytasks.md")
	content := `- [ ] Do thing one
- [ ] Do thing two
`
	if err := os.WriteFile(tasksFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	init := NewInitializer(dir, nil)
	result, err := init.ImportFromFile("mytasks.md")
	if err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}
	if len(result.Tasks) != 2 {
		t.Errorf("len(Tasks) = %d, want %d", len(result.Tasks), 2)
	}
}

func TestImportFromContent(t *testing.T) {
	init := NewInitializer(".", nil)
	content := `- [ ] Task one
- [ ] Task two
- [x] Task three
`
	result, err := init.ImportFromContent(content)
	if err != nil {
		t.Fatalf("ImportFromContent() error = %v", err)
	}
	if len(result.Tasks) != 3 {
		t.Errorf("len(Tasks) = %d, want %d", len(result.Tasks), 3)
	}
}

func TestCreateEmpty(t *testing.T) {
	init := NewInitializer(".", nil)
	result := init.CreateEmpty()

	if result == nil {
		t.Fatal("CreateEmpty() returned nil")
	}
	if len(result.Tasks) != 0 {
		t.Errorf("len(Tasks) = %d, want 0", len(result.Tasks))
	}
	if len(result.Warnings) != 0 {
		t.Errorf("len(Warnings) = %d, want 0", len(result.Warnings))
	}
	if len(result.Errors) != 0 {
		t.Errorf("len(Errors) = %d, want 0", len(result.Errors))
	}
}

func TestSaveToStore(t *testing.T) {
	dir := t.TempDir()
	init := NewInitializer(dir, nil)

	tasks := []*Task{
		NewTask("TASK-001", "First task", "Description 1"),
		NewTask("TASK-002", "Second task", "Description 2"),
	}

	storePath := ".ralph/tasks.json"
	err := init.SaveToStore(tasks, storePath)
	if err != nil {
		t.Fatalf("SaveToStore() error = %v", err)
	}

	// Verify file was created
	fullPath := filepath.Join(dir, storePath)
	if _, err := os.Stat(fullPath); err != nil {
		t.Errorf("Store file not created: %v", err)
	}

	// Load and verify
	store := NewStore(fullPath)
	if err := store.Load(); err != nil {
		t.Fatalf("Failed to load saved store: %v", err)
	}
	if len(store.Tasks()) != 2 {
		t.Errorf("len(Tasks) = %d, want 2", len(store.Tasks()))
	}
}

func TestImportFromDetection_NilDetection(t *testing.T) {
	init := NewInitializer(".", nil)
	_, err := init.ImportFromDetection(nil)
	if err == nil {
		t.Error("ImportFromDetection(nil) should return error")
	}
}

func TestImportFromDetection_NotDetected(t *testing.T) {
	init := NewInitializer(".", nil)
	detection := &TaskListDetection{Detected: false}
	_, err := init.ImportFromDetection(detection)
	if err == nil {
		t.Error("ImportFromDetection with Detected=false should return error")
	}
}

func TestImportFromFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	init := NewInitializer(dir, nil)
	_, err := init.ImportFromFile("nonexistent.md")
	if err == nil {
		t.Error("ImportFromFile() should return error for non-existent file")
	}
}

func TestExtractJSONArray(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple array",
			input: `[{"id": "1"}]`,
			want:  `[{"id": "1"}]`,
		},
		{
			name:  "array with surrounding text",
			input: `Here are tasks: [{"id": "1"}, {"id": "2"}] done`,
			want:  `[{"id": "1"}, {"id": "2"}]`,
		},
		{
			name:  "nested arrays",
			input: `[{"id": "1", "items": ["a", "b"]}]`,
			want:  `[{"id": "1", "items": ["a", "b"]}]`,
		},
		{
			name:  "no array",
			input: `just text`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONArray(tt.input)
			if got != tt.want {
				t.Errorf("extractJSONArray() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseGeneratedTasks(t *testing.T) {
	output := `[
		{"id": "TASK-001", "name": "First task", "description": "Do the first thing"},
		{"id": "TASK-002", "name": "Second task", "description": "Do the second thing"}
	]`

	tasks, err := parseGeneratedTasks(output)
	if err != nil {
		t.Fatalf("parseGeneratedTasks() error = %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("len(tasks) = %d, want 2", len(tasks))
	}
	if tasks[0].ID != "TASK-001" {
		t.Errorf("tasks[0].ID = %q, want %q", tasks[0].ID, "TASK-001")
	}
	if tasks[0].Name != "First task" {
		t.Errorf("tasks[0].Name = %q, want %q", tasks[0].Name, "First task")
	}
	if tasks[0].Order != 1 {
		t.Errorf("tasks[0].Order = %d, want 1", tasks[0].Order)
	}
}

func TestParseGeneratedTasks_NoJSON(t *testing.T) {
	_, err := parseGeneratedTasks("no json here")
	if err == nil {
		t.Error("parseGeneratedTasks() should error on invalid input")
	}
}

func TestOnProgress(t *testing.T) {
	init := NewInitializer(".", nil)
	var messages []string
	init.OnProgress = func(status string) {
		messages = append(messages, status)
	}

	// Create empty will trigger progress callback
	init.CreateEmpty()

	if len(messages) != 1 {
		t.Errorf("len(messages) = %d, want 1", len(messages))
	}
	if messages[0] != "Creating empty task list..." {
		t.Errorf("message = %q", messages[0])
	}
}

func TestImportFromDetection_JSON(t *testing.T) {
	dir := t.TempDir()
	ralphDir := filepath.Join(dir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a valid JSON task file
	content := `{
		"metadata": {"version": "1.0"},
		"tasks": [
			{
				"id": "TASK-001",
				"name": "First task",
				"description": "Description 1",
				"status": "pending",
				"order": 1
			},
			{
				"id": "TASK-002",
				"name": "Second task",
				"description": "Description 2",
				"status": "completed",
				"order": 2
			}
		]
	}`
	tasksFile := filepath.Join(ralphDir, "tasks.json")
	if err := os.WriteFile(tasksFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	init := NewInitializer(dir, nil)
	detection := init.DetectTaskList()

	if detection == nil {
		t.Fatal("DetectTaskList() returned nil")
	}
	if detection.Format != "json" {
		t.Errorf("Format = %q, want %q", detection.Format, "json")
	}

	result, err := init.ImportFromDetection(detection)
	if err != nil {
		t.Fatalf("ImportFromDetection() error = %v", err)
	}
	if len(result.Tasks) != 2 {
		t.Errorf("len(Tasks) = %d, want 2", len(result.Tasks))
	}
	if result.Tasks[0].ID != "TASK-001" {
		t.Errorf("Tasks[0].ID = %q, want %q", result.Tasks[0].ID, "TASK-001")
	}
	if result.Tasks[1].Status != "completed" {
		t.Errorf("Tasks[1].Status = %q, want %q", result.Tasks[1].Status, "completed")
	}
}

func TestImportFromDetection_PlainText(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a plain text task file (not markdown)
	content := `1. TASK-001: First task
2. TASK-002: Second task
3. Third task
`
	tasksFile := filepath.Join(dir, "TASKS.txt")
	if err := os.WriteFile(tasksFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	init := NewInitializer(dir, nil)
	// Manually call ImportFromFile with auto-detection
	result, err := init.ImportFromFile("TASKS.txt")
	if err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}
	if len(result.Tasks) != 3 {
		t.Errorf("len(Tasks) = %d, want 3", len(result.Tasks))
	}
}

func TestCountTasksInFile_PlainText(t *testing.T) {
	dir := t.TempDir()
	tasksFile := filepath.Join(dir, "tasks.txt")

	// Plain text format
	content := `- Task one
- Task two
* Task three
1. Task four
`
	if err := os.WriteFile(tasksFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	init := NewInitializer(dir, nil)
	count := init.countTasksInFile(tasksFile, "plaintext")

	// Should count lines starting with -, *, or number.
	if count < 4 {
		t.Errorf("count = %d, want at least 4", count)
	}
}

func TestCountTasksInFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	tasksFile := filepath.Join(dir, "tasks.json")

	// Invalid JSON
	content := `not valid json`
	if err := os.WriteFile(tasksFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	init := NewInitializer(dir, nil)
	count := init.countTasksInFile(tasksFile, "json")

	// Should return 0 for invalid JSON
	if count != 0 {
		t.Errorf("count = %d, want 0 for invalid JSON", count)
	}
}

func TestCountTasksInFile_NonExistent(t *testing.T) {
	init := NewInitializer("/tmp", nil)
	count := init.countTasksInFile("/nonexistent/file.md", "markdown")

	if count != 0 {
		t.Errorf("count = %d, want 0 for non-existent file", count)
	}
}

func TestImportFromFile_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	tasksFile := filepath.Join(dir, "tasks.md")
	content := `- [ ] TASK-001: Test task`
	if err := os.WriteFile(tasksFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize with different project dir but use absolute path
	init := NewInitializer("/different/project", nil)
	result, err := init.ImportFromFile(tasksFile)
	if err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}
	if len(result.Tasks) != 1 {
		t.Errorf("len(Tasks) = %d, want 1", len(result.Tasks))
	}
}

func TestExtractJSONArray_StringsWithBrackets(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "string containing open bracket",
			input: `[{"text": "use [x] syntax"}]`,
			want:  `[{"text": "use [x] syntax"}]`,
		},
		{
			name:  "string containing close bracket",
			input: `[{"text": "array] example"}]`,
			want:  `[{"text": "array] example"}]`,
		},
		{
			name:  "escaped quotes in string",
			input: `[{"text": "say \"hello\""}]`,
			want:  `[{"text": "say \"hello\""}]`,
		},
		{
			name:  "multiple brackets in strings",
			input: `[{"id": "1", "text": "[a][b]"}, {"id": "2", "text": "[c]"}]`,
			want:  `[{"id": "1", "text": "[a][b]"}, {"id": "2", "text": "[c]"}]`,
		},
		{
			name:  "unmatched bracket",
			input: `[{"id": "1"} extra text`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONArray(tt.input)
			if got != tt.want {
				t.Errorf("extractJSONArray() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseGeneratedTasks_WithSurroundingText(t *testing.T) {
	output := `Here are the tasks I generated:

[
	{"id": "TASK-001", "name": "Setup project", "description": "Initialize the project"},
	{"id": "TASK-002", "name": "Implement feature", "description": "Add the new feature"}
]

Let me know if you need any changes.`

	tasks, err := parseGeneratedTasks(output)
	if err != nil {
		t.Fatalf("parseGeneratedTasks() error = %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("len(tasks) = %d, want 2", len(tasks))
	}
}

func TestParseGeneratedTasks_InvalidJSON(t *testing.T) {
	output := `[{"id": "TASK-001", "name": "Missing close brace"]`
	_, err := parseGeneratedTasks(output)
	if err == nil {
		t.Error("parseGeneratedTasks() should error on invalid JSON")
	}
}

func TestSaveToStore_AbsolutePath(t *testing.T) {
	dir := t.TempDir()
	init := NewInitializer("/different/project", nil)

	tasks := []*Task{
		NewTask("TASK-001", "Test task", "Description"),
	}

	// Use absolute path
	storePath := filepath.Join(dir, "tasks.json")
	err := init.SaveToStore(tasks, storePath)
	if err != nil {
		t.Fatalf("SaveToStore() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(storePath); err != nil {
		t.Errorf("Store file not created: %v", err)
	}
}

func TestDetectTaskList_Priority(t *testing.T) {
	dir := t.TempDir()
	ralphDir := filepath.Join(dir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create both .ralph/tasks.json and TASKS.md
	jsonContent := `{"tasks": [{"id": "TASK-001"}]}`
	mdContent := `- [ ] TASK-001: From markdown`

	if err := os.WriteFile(filepath.Join(ralphDir, "tasks.json"), []byte(jsonContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "TASKS.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	init := NewInitializer(dir, nil)
	detection := init.DetectTaskList()

	// .ralph/tasks.json should have priority
	if detection == nil {
		t.Fatal("DetectTaskList() returned nil")
	}
	if detection.Path != ".ralph/tasks.json" {
		t.Errorf("Path = %q, want .ralph/tasks.json (highest priority)", detection.Path)
	}
}

func TestImportFromContent_PlainText(t *testing.T) {
	init := NewInitializer(".", nil)
	content := `1. First task
2. Second task
3. Third task
`
	result, err := init.ImportFromContent(content)
	if err != nil {
		t.Fatalf("ImportFromContent() error = %v", err)
	}
	if len(result.Tasks) != 3 {
		t.Errorf("len(Tasks) = %d, want 3", len(result.Tasks))
	}
}

