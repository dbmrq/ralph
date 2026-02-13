// Package task provides task data model and management for ralph.
package task

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// osOpen wraps os.Open for testability
func osOpen(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

// osReadFile wraps os.ReadFile for testability
func osReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ImportFormat represents the format of task input data.
type ImportFormat string

const (
	// FormatMarkdown represents markdown task list format (- [ ] or - [x]).
	FormatMarkdown ImportFormat = "markdown"
	// FormatPlainText represents plain text format (numbered or bulleted lists).
	FormatPlainText ImportFormat = "plaintext"
)

// ImportResult contains the result of an import operation.
type ImportResult struct {
	Tasks    []*Task
	Warnings []string
	Errors   []error
}

// markdownTaskPattern matches markdown task lines like "- [ ] TASK-001: Description"
// Groups: 1=checkbox content (x or space), 2=task ID (optional), 3=task name
var markdownTaskPattern = regexp.MustCompile(`^\s*-\s*\[([ xX])\]\s*(?:([A-Za-z]+-\d+):?\s*)?(.+)$`)

// markdownContextPattern matches context lines like "> Goal: ..."
var markdownContextPattern = regexp.MustCompile(`^\s*>\s*(.+)$`)

// plainTextNumberedPattern matches numbered tasks like "1. Task description" or "1) Task description"
var plainTextNumberedPattern = regexp.MustCompile(`^\s*(\d+)[.)]\s+(?:([A-Za-z]+-\d+):?\s*)?(.+)$`)

// plainTextBulletPattern matches bulleted tasks like "* Task description" or "• Task description"
var plainTextBulletPattern = regexp.MustCompile(`^\s*[*•-]\s+(?:([A-Za-z]+-\d+):?\s*)?(.+)$`)

// Importer handles importing tasks from various formats.
type Importer struct {
	idPrefix  string
	idCounter int
}

// NewImporter creates a new Importer with default settings.
func NewImporter() *Importer {
	return &Importer{
		idPrefix:  "TASK",
		idCounter: 1,
	}
}

// SetIDPrefix sets the prefix for auto-generated task IDs.
func (i *Importer) SetIDPrefix(prefix string) {
	i.idPrefix = prefix
}

// SetIDStart sets the starting number for auto-generated task IDs.
func (i *Importer) SetIDStart(start int) {
	i.idCounter = start
}

// generateID generates a new unique task ID.
func (i *Importer) generateID() string {
	id := fmt.Sprintf("%s-%03d", i.idPrefix, i.idCounter)
	i.idCounter++
	return id
}

// ImportFromMarkdown parses markdown format task lists.
// Supports:
// - [ ] TASK-ID: Task name
// - [x] TASK-ID: Task name (completed)
// - [ ] Task name without ID (ID will be generated)
// > Context lines are added to description
func (i *Importer) ImportFromMarkdown(reader io.Reader) (*ImportResult, error) {
	result := &ImportResult{
		Tasks:    []*Task{},
		Warnings: []string{},
		Errors:   []error{},
	}

	scanner := bufio.NewScanner(reader)
	var currentTask *Task
	var contextLines []string
	order := 1

	for scanner.Scan() {
		line := scanner.Text()

		// Try to match a markdown task line
		if matches := markdownTaskPattern.FindStringSubmatch(line); matches != nil {
			// Finalize previous task if any
			if currentTask != nil {
				if len(contextLines) > 0 {
					currentTask.Description = strings.Join(contextLines, "\n")
				}
				result.Tasks = append(result.Tasks, currentTask)
			}

			checkboxContent := matches[1]
			taskID := matches[2]
			taskName := strings.TrimSpace(matches[3])

			// Generate ID if not provided
			if taskID == "" {
				taskID = i.generateID()
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Generated ID %q for task: %s", taskID, taskName))
			}

			currentTask = NewTask(taskID, taskName, "")
			currentTask.Order = order
			order++

			// Check if task is marked as completed
			if checkboxContent == "x" || checkboxContent == "X" {
				currentTask.MarkCompleted()
			}

			contextLines = []string{}
			continue
		}

		// Try to match a context line
		if currentTask != nil {
			if matches := markdownContextPattern.FindStringSubmatch(line); matches != nil {
				contextLines = append(contextLines, strings.TrimSpace(matches[1]))
				continue
			}
		}
	}

	// Finalize last task
	if currentTask != nil {
		if len(contextLines) > 0 {
			currentTask.Description = strings.Join(contextLines, "\n")
		}
		result.Tasks = append(result.Tasks, currentTask)
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("error reading input: %w", err)
	}

	return result, nil
}

// ImportFromPlainText parses plain text format task lists.
// Supports:
// 1. TASK-ID: Task name (numbered)
// 1) Task name without ID (numbered, ID generated)
// * Task name (bulleted)
// • Task name (bulleted)
// - Task name (bulleted, when not followed by [ ])
func (i *Importer) ImportFromPlainText(reader io.Reader) (*ImportResult, error) {
	result := &ImportResult{
		Tasks:    []*Task{},
		Warnings: []string{},
		Errors:   []error{},
	}

	scanner := bufio.NewScanner(reader)
	order := 1

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		var taskID, taskName string

		// Try numbered pattern first
		if matches := plainTextNumberedPattern.FindStringSubmatch(line); matches != nil {
			taskID = matches[2]
			taskName = strings.TrimSpace(matches[3])
		} else if matches := plainTextBulletPattern.FindStringSubmatch(line); matches != nil {
			// Skip if it looks like a markdown checkbox
			if strings.Contains(line, "[ ]") || strings.Contains(line, "[x]") || strings.Contains(line, "[X]") {
				continue
			}
			taskID = matches[1]
			taskName = strings.TrimSpace(matches[2])
		} else {
			// Not a task line, skip
			continue
		}

		// Generate ID if not provided
		if taskID == "" {
			taskID = i.generateID()
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Generated ID %q for task: %s", taskID, taskName))
		}

		task := NewTask(taskID, taskName, "")
		task.Order = order
		order++
		result.Tasks = append(result.Tasks, task)
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("error reading input: %w", err)
	}

	return result, nil
}

// ImportFromString is a convenience function to import from a string.
func (i *Importer) ImportFromString(content string, format ImportFormat) (*ImportResult, error) {
	reader := strings.NewReader(content)
	switch format {
	case FormatMarkdown:
		return i.ImportFromMarkdown(reader)
	case FormatPlainText:
		return i.ImportFromPlainText(reader)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// DetectFormat attempts to detect the format of the input content.
func DetectFormat(content string) ImportFormat {
	// Check for markdown task checkboxes line by line
	// since the regex uses ^ which doesn't work with multiline strings by default
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if markdownTaskPattern.MatchString(line) {
			return FormatMarkdown
		}
	}
	// Default to plain text
	return FormatPlainText
}

// ImportAuto automatically detects format and imports tasks.
func (i *Importer) ImportAuto(content string) (*ImportResult, error) {
	format := DetectFormat(content)
	return i.ImportFromString(content, format)
}

// ExtractMetadata parses task description lines for metadata.
// Recognizes patterns like:
// > Tests: Not required
// > Build: Skip
// > Goal: Description
func ExtractMetadata(description string) map[string]string {
	metadata := make(map[string]string)

	lines := strings.Split(description, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for key: value pattern
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			// Normalize common keys
			keyLower := strings.ToLower(key)
			switch keyLower {
			case "goal", "tests", "build", "reference", "notes":
				metadata[keyLower] = value
			default:
				metadata[key] = value
			}
		}
	}

	return metadata
}

// ParseTaskMetadata extracts and applies metadata from task description.
func (t *Task) ParseTaskMetadata() {
	if t.Description == "" {
		return
	}

	metadata := ExtractMetadata(t.Description)
	for k, v := range metadata {
		t.SetMetadata(k, v)
	}
}

// ImportFromFile imports tasks from a file.
func (i *Importer) ImportFromFile(path string, format ImportFormat) (*ImportResult, error) {
	file, err := openFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	switch format {
	case FormatMarkdown:
		return i.ImportFromMarkdown(file)
	case FormatPlainText:
		return i.ImportFromPlainText(file)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// ImportFileAuto imports tasks from a file, auto-detecting the format.
func (i *Importer) ImportFileAuto(path string) (*ImportResult, error) {
	content, err := readFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	format := DetectFormat(content)
	return i.ImportFromString(content, format)
}

// ImportToStore imports tasks from a file into the given store.
// Returns the import result and any error.
func (i *Importer) ImportToStore(store *Store, path string, format ImportFormat) (*ImportResult, error) {
	result, err := i.ImportFromFile(path, format)
	if err != nil {
		return result, err
	}

	// Parse metadata for each task
	for _, task := range result.Tasks {
		task.ParseTaskMetadata()
	}

	// Add all tasks to the store
	if err := store.AddAll(result.Tasks); err != nil {
		result.Errors = append(result.Errors, err)
		return result, err
	}

	return result, nil
}

// openFile is a variable for testing
var openFile = func(path string) (io.ReadCloser, error) {
	return osOpen(path)
}

// readFile is a variable for testing
var readFile = func(path string) (string, error) {
	return osReadFile(path)
}
