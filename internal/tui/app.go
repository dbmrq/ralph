// Package tui provides the terminal user interface for ralph.
package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wexinc/ralph/internal/task"
	"github.com/wexinc/ralph/internal/tui/components"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// LoopState represents the current state of the Ralph loop.
type LoopState string

const (
	LoopStateIdle        LoopState = "idle"
	LoopStateRunning     LoopState = "running"
	LoopStatePaused      LoopState = "paused"
	LoopStateAwaitingFix LoopState = "awaiting_fix"
	LoopStateCompleted   LoopState = "completed"
	LoopStateFailed      LoopState = "failed"
)

// Model is the Bubble Tea model for the Ralph TUI.
type Model struct {
	// Components
	header   *components.Header
	progress *components.Progress

	// State
	loopState   LoopState
	tasks       []*task.Task
	currentTask *task.Task
	iteration   int
	startTime   time.Time
	lastError   string

	// Session info
	sessionID   string
	projectName string
	agentName   string
	modelName   string

	// Window dimensions
	width  int
	height int

	// Flags
	quitting bool
}

// New creates a new TUI model.
func New() *Model {
	return &Model{
		header:      components.NewHeader(),
		progress:    components.NewProgress(),
		loopState:   LoopStateIdle,
		tasks:       []*task.Task{},
		iteration:   0,
		startTime:   time.Now(),
		projectName: "ralph",
	}
}

// Init is the Bubble Tea initialization function.
func (m *Model) Init() tea.Cmd {
	// Start with a tick for time-based updates
	return tickCmd()
}

// tickCmd returns a command that sends a tick message every second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg{Time: t}
	})
}

// Update handles messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.header.SetWidth(msg.Width)
		m.progress.SetWidth(msg.Width)
		return m, nil

	case TickMsg:
		// Update time-based displays
		return m, tickCmd()

	case TasksUpdatedMsg:
		m.tasks = msg.Tasks
		m.progress.SetProgress(msg.Completed, msg.Total)
		return m, nil

	case TaskStartedMsg:
		m.iteration = msg.Iteration
		m.progress.SetIteration(msg.Iteration)
		for _, t := range m.tasks {
			if t.ID == msg.TaskID {
				m.currentTask = t
				break
			}
		}
		return m, nil

	case SessionInfoMsg:
		m.sessionID = msg.SessionID
		m.projectName = msg.ProjectName
		m.agentName = msg.AgentName
		m.modelName = msg.ModelName
		m.updateHeader()
		return m, nil

	case LoopStateMsg:
		m.loopState = LoopState(msg.State)
		m.iteration = msg.Iteration
		return m, nil

	case ErrorMsg:
		m.lastError = msg.Error
		return m, nil

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// handleKeyPress handles keyboard input.
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "?", "h":
		// TODO: Show help overlay (TUI-006)
		return m, nil
	}

	return m, nil
}

// updateHeader updates the header component with current state.
func (m *Model) updateHeader() {
	m.header.SetData(components.HeaderData{
		ProjectName: m.projectName,
		AgentName:   m.agentName,
		ModelName:   m.modelName,
		SessionID:   m.sessionID,
	})
}

// View renders the TUI.
func (m *Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	// Build the view
	var view string

	// Header
	view += m.header.View() + "\n"

	// Progress bar
	view += m.progress.View() + "\n"

	// Divider
	if m.width > 0 {
		divider := lipgloss.NewStyle().
			Foreground(styles.BorderColor).
			Render(repeatChar("─", m.width))
		view += divider + "\n"
	}

	// Main content area (placeholder for now - will be task list in TUI-003)
	if m.loopState == LoopStateIdle {
		view += styles.MutedTextStyle.Render("Press 'q' to quit, 'h' for help") + "\n"
	} else if m.loopState == LoopStateRunning && m.currentTask != nil {
		view += styles.MutedTextStyle.Render("Running: "+m.currentTask.Name) + "\n"
	}

	// Error display
	if m.lastError != "" {
		view += styles.ErrorTextStyle.Render("Error: "+m.lastError) + "\n"
	}

	// Status bar at bottom
	view += m.renderStatusBar()

	return view
}

// renderStatusBar renders the bottom status bar.
func (m *Model) renderStatusBar() string {
	// Elapsed time
	elapsed := time.Since(m.startTime)
	elapsedStr := formatDuration(elapsed)

	// Build status items
	var items []string

	// Keyboard shortcuts
	shortcuts := []struct {
		key  string
		desc string
	}{
		{"q", "quit"},
		{"h", "help"},
	}

	for _, s := range shortcuts {
		key := styles.KeyStyle.Render("[" + s.key + "]")
		desc := styles.HelpStyle.Render(s.desc)
		items = append(items, key+desc)
	}

	// Join shortcuts
	shortcutStr := ""
	for i, item := range items {
		if i > 0 {
			shortcutStr += "  "
		}
		shortcutStr += item
	}

	// Elapsed time on the right
	timeStr := styles.MutedTextStyle.Render("Elapsed: " + elapsedStr)

	// Combine
	sep := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Render(" │ ")

	return "\n" + shortcutStr + sep + timeStr + "\n"
}

// repeatChar repeats a character n times.
func repeatChar(char string, n int) string {
	if n <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < n; i++ {
		result += char
	}
	return result
}

// formatDuration formats a duration as HH:MM:SS or MM:SS.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// SetTasks updates the task list.
func (m *Model) SetTasks(tasks []*task.Task) {
	m.tasks = tasks
	completed := 0
	for _, t := range tasks {
		if t.Status == task.StatusCompleted {
			completed++
		}
	}
	m.progress.SetProgress(completed, len(tasks))
}

// SetSessionInfo updates session information.
func (m *Model) SetSessionInfo(projectName, agentName, modelName, sessionID string) {
	m.projectName = projectName
	m.agentName = agentName
	m.modelName = modelName
	m.sessionID = sessionID
	m.updateHeader()
}

// SetLoopState updates the loop state.
func (m *Model) SetLoopState(state LoopState) {
	m.loopState = state
}

// Run starts the TUI.
func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

