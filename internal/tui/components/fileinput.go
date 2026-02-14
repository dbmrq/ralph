// Package components provides reusable TUI components for ralph.
package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wexinc/ralph/internal/task"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// FileInputSubmittedMsg is sent when user submits the file path.
type FileInputSubmittedMsg struct {
	Path string
}

// FileInputCanceledMsg is sent when user cancels file input.
type FileInputCanceledMsg struct{}

// FileInput is a component for entering file paths to import tasks.
type FileInput struct {
	input        textinput.Model
	width        int
	focused      bool
	projectDir   string
	fileExists   bool
	previewTasks []*task.Task
	parseError   string
}

// NewFileInput creates a new FileInput component.
func NewFileInput(projectDir string) *FileInput {
	ti := textinput.New()
	ti.Placeholder = "Enter path to task file (e.g., TASKS.md, TODO.txt)"
	ti.CharLimit = 512
	ti.Width = 50

	return &FileInput{
		input:      ti,
		projectDir: projectDir,
	}
}

// SetWidth sets the component width.
func (f *FileInput) SetWidth(width int) {
	f.width = width
	f.input.Width = width - 4
}

// Focus focuses the input.
func (f *FileInput) Focus() tea.Cmd {
	f.focused = true
	return f.input.Focus()
}

// Blur removes focus from the input.
func (f *FileInput) Blur() {
	f.focused = false
	f.input.Blur()
}

// Value returns the current file path.
func (f *FileInput) Value() string {
	return f.input.Value()
}

// SetValue sets the file path.
func (f *FileInput) SetValue(value string) {
	f.input.SetValue(value)
	f.validateAndPreview()
}

// FileExists returns whether the current path points to a valid file.
func (f *FileInput) FileExists() bool {
	return f.fileExists
}

// PreviewTasks returns the currently parsed tasks.
func (f *FileInput) PreviewTasks() []*task.Task {
	return f.previewTasks
}

// ParseError returns any error from parsing.
func (f *FileInput) ParseError() string {
	return f.parseError
}

// validateAndPreview checks if the file exists and parses it.
func (f *FileInput) validateAndPreview() {
	path := strings.TrimSpace(f.input.Value())
	if path == "" {
		f.fileExists = false
		f.previewTasks = nil
		f.parseError = ""
		return
	}

	// Resolve path relative to project dir
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(f.projectDir, path)
	}

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		f.fileExists = false
		f.previewTasks = nil
		f.parseError = "File not found"
		return
	}

	if info.IsDir() {
		f.fileExists = false
		f.previewTasks = nil
		f.parseError = "Path is a directory, not a file"
		return
	}

	f.fileExists = true

	// Try to parse the file
	importer := task.NewImporter()
	result, err := importer.ImportFileAuto(fullPath)
	if err != nil {
		f.parseError = err.Error()
		f.previewTasks = nil
		return
	}

	f.parseError = ""
	f.previewTasks = result.Tasks
}

// Update handles messages for the component.
func (f *FileInput) Update(msg tea.Msg) (*FileInput, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			path := strings.TrimSpace(f.input.Value())
			if path == "" || !f.fileExists {
				return f, nil
			}
			return f, func() tea.Msg {
				return FileInputSubmittedMsg{Path: path}
			}
		case "esc":
			return f, func() tea.Msg {
				return FileInputCanceledMsg{}
			}
		}
	}

	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)

	// Validate after each change
	f.validateAndPreview()

	return f, cmd
}

// View renders the component.
func (f *FileInput) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Bold(true).
		Padding(0, 1)

	b.WriteString(titleStyle.Render("📂 Import from File"))
	b.WriteString("\n\n")

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(styles.MutedLight).
		PaddingLeft(1)

	b.WriteString(subtitleStyle.Render("Enter the path to your existing task file:"))
	b.WriteString("\n\n")

	// Input field
	b.WriteString("  ")
	b.WriteString(f.input.View())
	b.WriteString("\n\n")

	// Status/Preview
	if f.parseError != "" {
		errorStyle := lipgloss.NewStyle().Foreground(styles.Error)
		b.WriteString(errorStyle.Render("  ⚠ " + f.parseError))
		b.WriteString("\n")
	} else if len(f.previewTasks) > 0 {
		successStyle := lipgloss.NewStyle().Foreground(styles.Success)
		b.WriteString(successStyle.Render(fmt.Sprintf("  ✓ Found %d tasks", len(f.previewTasks))))
		b.WriteString("\n")

		// Show first few tasks
		mutedStyle := lipgloss.NewStyle().Foreground(styles.Muted)
		maxPreview := 3
		for i, tsk := range f.previewTasks {
			if i >= maxPreview {
				remaining := len(f.previewTasks) - maxPreview
				b.WriteString(mutedStyle.Render(fmt.Sprintf("    ... and %d more", remaining)))
				b.WriteString("\n")
				break
			}
			b.WriteString(mutedStyle.Render("    • " + tsk.Name))
			b.WriteString("\n")
		}
	} else if f.input.Value() != "" {
		mutedStyle := lipgloss.NewStyle().Foreground(styles.Muted)
		b.WriteString(mutedStyle.Render("  Validating path..."))
		b.WriteString("\n")
	}

	// Common file locations hint
	b.WriteString("\n")
	hintTitleStyle := lipgloss.NewStyle().
		Foreground(styles.Secondary).
		Bold(true)
	b.WriteString(hintTitleStyle.Render("  Common locations:"))
	b.WriteString("\n")

	hintStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	hints := []string{"TASKS.md", "TODO.md", "docs/TASKS.md", ".github/TASKS.md"}
	for _, hint := range hints {
		b.WriteString(hintStyle.Render("    " + hint))
		b.WriteString("\n")
	}

	// Shortcut bar
	b.WriteString("\n")
	shortcutBar := NewShortcutBar(FileInputShortcuts...)
	b.WriteString(shortcutBar.View())

	return b.String()
}

