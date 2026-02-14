// Package task provides task data model and management for ralph.
package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wexinc/ralph/internal/agent"
)

// InitMode represents how the task list should be initialized.
type InitMode string

const (
	// InitModeAuto uses AI to detect and parse existing task lists.
	InitModeAuto InitMode = "auto"
	// InitModeFile imports tasks from a specified file.
	InitModeFile InitMode = "file"
	// InitModePaste parses tasks from pasted content.
	InitModePaste InitMode = "paste"
	// InitModeGenerate generates tasks from a goal description.
	InitModeGenerate InitMode = "generate"
	// InitModeEmpty starts with an empty task list.
	InitModeEmpty InitMode = "empty"
)

// TaskListDetection contains results from AI task list detection.
type TaskListDetection struct {
	// Detected indicates whether a task list was found.
	Detected bool `json:"detected"`
	// Path is the path to the detected task list file.
	Path string `json:"path"`
	// Format is the format of the task list (e.g., "markdown", "json", "plaintext").
	Format string `json:"format"`
	// TaskCount is the estimated number of tasks.
	TaskCount int `json:"task_count"`
}

// InitializerProgress is a callback for reporting initialization progress.
type InitializerProgress func(status string)

// Initializer handles task list detection and initialization.
type Initializer struct {
	// ProjectDir is the root directory of the project.
	ProjectDir string
	// Agent is the AI agent to use for parsing/generation.
	Agent agent.Agent
	// Model is the model to use (optional).
	Model string
	// OnProgress is called with status updates.
	OnProgress InitializerProgress
	// LogWriter receives real-time agent output (optional).
	LogWriter io.Writer
}

// NewInitializer creates a new Initializer.
func NewInitializer(projectDir string, ag agent.Agent) *Initializer {
	return &Initializer{
		ProjectDir: projectDir,
		Agent:      ag,
		OnProgress: func(status string) {}, // noop by default
	}
}

// DetectTaskList checks for existing task lists in the project.
// Returns detection info, or nil if no task list was found.
func (i *Initializer) DetectTaskList() *TaskListDetection {
	// Check common locations for task lists
	locations := []struct {
		path   string
		format string
	}{
		{".ralph/tasks.json", "json"},
		{"TASKS.md", "markdown"},
		{"TODO.md", "markdown"},
		{"ROADMAP.md", "markdown"},
		{".ralph/TASKS.md", "markdown"},
		{"docs/TASKS.md", "markdown"},
		{"docs/TODO.md", "markdown"},
	}

	for _, loc := range locations {
		fullPath := filepath.Join(i.ProjectDir, loc.path)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			count := i.countTasksInFile(fullPath, loc.format)
			return &TaskListDetection{
				Detected:  true,
				Path:      loc.path,
				Format:    loc.format,
				TaskCount: count,
			}
		}
	}
	return nil
}

// countTasksInFile counts tasks in a file (approximate).
func (i *Initializer) countTasksInFile(path string, format string) int {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	text := string(content)
	switch format {
	case "markdown":
		// Count markdown checkboxes
		return strings.Count(text, "- [ ]") + strings.Count(text, "- [x]") + strings.Count(text, "- [X]")
	case "json":
		var data map[string]interface{}
		if err := json.Unmarshal(content, &data); err == nil {
			if tasks, ok := data["tasks"].([]interface{}); ok {
				return len(tasks)
			}
		}
		return 0
	default:
		// Count lines that look like tasks
		lines := strings.Split(text, "\n")
		count := 0
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "-") ||
				strings.HasPrefix(strings.TrimSpace(line), "*") ||
				(len(strings.TrimSpace(line)) > 2 && strings.TrimSpace(line)[1] == '.') {
				count++
			}
		}
		return count
	}
}

// report calls the progress callback.
func (i *Initializer) report(status string) {
	if i.OnProgress != nil {
		i.OnProgress(status)
	}
}

// ImportFromDetection imports tasks from a detected task list.
func (i *Initializer) ImportFromDetection(detection *TaskListDetection) (*ImportResult, error) {
	if detection == nil || !detection.Detected {
		return nil, fmt.Errorf("no task list detected")
	}

	i.report(fmt.Sprintf("Importing tasks from %s...", detection.Path))

	fullPath := filepath.Join(i.ProjectDir, detection.Path)
	importer := NewImporter()

	var result *ImportResult
	var err error

	switch detection.Format {
	case "json":
		result, err = i.importFromJSON(fullPath)
	case "markdown":
		result, err = importer.ImportFromFile(fullPath, FormatMarkdown)
	default:
		result, err = importer.ImportFileAuto(fullPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to import tasks: %w", err)
	}

	i.report(fmt.Sprintf("Imported %d tasks", len(result.Tasks)))
	return result, nil
}

// importFromJSON imports tasks from our native JSON format.
func (i *Initializer) importFromJSON(path string) (*ImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var store struct {
		Tasks []*Task `json:"tasks"`
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}

	return &ImportResult{
		Tasks:    store.Tasks,
		Warnings: []string{},
		Errors:   []error{},
	}, nil
}

// ImportFromFile imports tasks from a specified file path.
func (i *Initializer) ImportFromFile(path string) (*ImportResult, error) {
	i.report(fmt.Sprintf("Importing tasks from %s...", path))

	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(i.ProjectDir, path)
	}

	// Check file exists
	if _, err := os.Stat(fullPath); err != nil {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	importer := NewImporter()
	result, err := importer.ImportFileAuto(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to import: %w", err)
	}

	i.report(fmt.Sprintf("Imported %d tasks", len(result.Tasks)))
	return result, nil
}

// ImportFromContent parses tasks from text content (paste mode).
func (i *Initializer) ImportFromContent(content string) (*ImportResult, error) {
	i.report("Parsing task content...")

	importer := NewImporter()
	result, err := importer.ImportAuto(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	i.report(fmt.Sprintf("Parsed %d tasks", len(result.Tasks)))
	return result, nil
}

// GenerateFromGoal uses AI to generate a task list from a goal description.
func (i *Initializer) GenerateFromGoal(ctx context.Context, goal string) (*ImportResult, error) {
	if i.Agent == nil {
		return nil, fmt.Errorf("no agent available for task generation")
	}

	i.report("Generating task list from goal...")

	prompt := buildTaskGenerationPrompt(goal)

	opts := agent.RunOptions{
		Model:     i.Model,
		WorkDir:   i.ProjectDir,
		Force:     true,
		LogWriter: i.LogWriter,
	}

	result, err := i.Agent.Run(ctx, prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("agent failed: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("agent error: %s", result.Error)
	}

	i.report("Parsing generated tasks...")
	tasks, err := parseGeneratedTasks(result.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated tasks: %w", err)
	}

	i.report(fmt.Sprintf("Generated %d tasks", len(tasks)))
	return &ImportResult{
		Tasks:    tasks,
		Warnings: []string{},
		Errors:   []error{},
	}, nil
}

// buildTaskGenerationPrompt creates the prompt for task generation.
func buildTaskGenerationPrompt(goal string) string {
	return fmt.Sprintf(`Generate a task list for the following goal. Return a JSON array of tasks.

Goal: %s

Return ONLY a JSON array with this structure, no other text:
[
  {
    "id": "TASK-001",
    "name": "Short task name",
    "description": "Detailed description of what needs to be done"
  },
  ...
]

Guidelines:
- Break down the goal into logical, sequential tasks
- Each task should be completable in 1-2 hours
- Use descriptive IDs (TASK-001, TASK-002, etc.)
- Include setup tasks if needed
- End with testing/verification tasks

Return ONLY the JSON array:`, goal)
}

// parseGeneratedTasks parses AI-generated task JSON.
func parseGeneratedTasks(output string) ([]*Task, error) {
	// Extract JSON array from output
	jsonStr := extractJSONArray(output)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON array found in output")
	}

	var taskData []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &taskData); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	tasks := make([]*Task, len(taskData))
	for i, td := range taskData {
		tasks[i] = NewTask(td.ID, td.Name, td.Description)
		tasks[i].Order = i + 1
	}

	return tasks, nil
}

// extractJSONArray finds and extracts a JSON array from text.
func extractJSONArray(text string) string {
	start := -1
	bracketCount := 0
	inString := false
	escape := false

	for i, c := range text {
		if escape {
			escape = false
			continue
		}
		if c == '\\' && inString {
			escape = true
			continue
		}
		if c == '"' && !escape {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if c == '[' {
			if start == -1 {
				start = i
			}
			bracketCount++
		} else if c == ']' {
			bracketCount--
			if bracketCount == 0 && start != -1 {
				return text[start : i+1]
			}
		}
	}

	return ""
}

// CreateEmpty creates an empty task store.
func (i *Initializer) CreateEmpty() *ImportResult {
	i.report("Creating empty task list...")
	return &ImportResult{
		Tasks:    []*Task{},
		Warnings: []string{},
		Errors:   []error{},
	}
}

// SaveToStore saves imported tasks to a store file.
func (i *Initializer) SaveToStore(tasks []*Task, path string) error {
	storePath := path
	if !filepath.IsAbs(path) {
		storePath = filepath.Join(i.ProjectDir, path)
	}

	store := NewStore(storePath)
	store.SetTasks(tasks)
	return store.Save()
}
