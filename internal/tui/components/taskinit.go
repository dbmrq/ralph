// Package components provides reusable TUI components for ralph.
package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wexinc/ralph/internal/task"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// TaskInitMode represents how to initialize the task list.
type TaskInitMode string

const (
	// TaskInitModeFile imports from a file.
	TaskInitModeFile TaskInitMode = "file"
	// TaskInitModePaste parses pasted content.
	TaskInitModePaste TaskInitMode = "paste"
	// TaskInitModeGenerate generates from a goal description.
	TaskInitModeGenerate TaskInitMode = "generate"
	// TaskInitModeEmpty starts with an empty list.
	TaskInitModeEmpty TaskInitMode = "empty"
)

// TaskInitSelectedMsg is sent when the user selects an init mode.
type TaskInitSelectedMsg struct {
	Mode TaskInitMode
}

// TaskInitCanceledMsg is sent when the user cancels task init.
type TaskInitCanceledMsg struct{}

// TaskInitSelector shows options for initializing the task list.
type TaskInitSelector struct {
	options     []taskInitOption
	selectedIdx int
	width       int
	detection   *task.TaskListDetection
}

type taskInitOption struct {
	mode        TaskInitMode
	title       string
	description string
}

// NewTaskInitSelector creates a new TaskInitSelector.
func NewTaskInitSelector() *TaskInitSelector {
	return &TaskInitSelector{
		options: []taskInitOption{
			{TaskInitModeFile, "Point to a file", "Browse or enter path to existing task file"},
			{TaskInitModePaste, "Paste a list", "Paste tasks from clipboard or type them"},
			{TaskInitModeGenerate, "Describe your goal", "Describe what you want to build, AI generates tasks"},
			{TaskInitModeEmpty, "Start empty", "Begin with no tasks, add them manually"},
		},
		selectedIdx: 0,
	}
}

// SetDetection sets the detected task list info (if found).
// This enables the "Import detected" option.
func (s *TaskInitSelector) SetDetection(detection *task.TaskListDetection) {
	s.detection = detection
}

// SetWidth sets the component width.
func (s *TaskInitSelector) SetWidth(width int) {
	s.width = width
}

// Update handles input messages.
func (s *TaskInitSelector) Update(msg tea.Msg) (*TaskInitSelector, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.selectedIdx > 0 {
				s.selectedIdx--
			}
		case "down", "j":
			if s.selectedIdx < len(s.options)-1 {
				s.selectedIdx++
			}
		case "enter":
			selected := s.options[s.selectedIdx]
			return s, func() tea.Msg {
				return TaskInitSelectedMsg{Mode: selected.mode}
			}
		case "esc":
			return s, func() tea.Msg {
				return TaskInitCanceledMsg{}
			}
		}
	}
	return s, nil
}

// View renders the selector.
func (s *TaskInitSelector) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Bold(true).
		Padding(0, 1)

	if s.detection != nil && s.detection.Detected {
		b.WriteString(titleStyle.Render("📋 Task List Found"))
		b.WriteString("\n\n")

		detectedStyle := lipgloss.NewStyle().
			Foreground(styles.Success).
			PaddingLeft(2)
		b.WriteString(detectedStyle.Render("Found: " + s.detection.Path))
		b.WriteString("\n")
		b.WriteString(detectedStyle.Render("Format: " + s.detection.Format))
		b.WriteString("\n\n")

		b.WriteString(lipgloss.NewStyle().
			Foreground(styles.MutedLight).
			PaddingLeft(2).
			Render("Press Enter to import, or choose another option:"))
		b.WriteString("\n\n")
	} else {
		b.WriteString(titleStyle.Render("📋 No Task List Found"))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(styles.MutedLight).
			PaddingLeft(2).
			Render("How would you like to create your task list?"))
		b.WriteString("\n\n")
	}

	// Options
	for i, opt := range s.options {
		prefix := "  "
		if i == s.selectedIdx {
			prefix = "▶ "
		}

		var optStyle lipgloss.Style
		if i == s.selectedIdx {
			optStyle = lipgloss.NewStyle().
				Foreground(styles.Primary).
				Bold(true)
		} else {
			optStyle = lipgloss.NewStyle().
				Foreground(styles.Foreground)
		}

		descStyle := lipgloss.NewStyle().
			Foreground(styles.Muted).
			PaddingLeft(4)

		b.WriteString(prefix)
		b.WriteString(optStyle.Render(opt.title))
		b.WriteString("\n")
		b.WriteString(descStyle.Render(opt.description))
		b.WriteString("\n\n")
	}

	return b.String()
}
