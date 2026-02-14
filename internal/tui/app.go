// Package tui provides the terminal user interface for ralph.
package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dbmrq/ralph/internal/loop"
	"github.com/dbmrq/ralph/internal/task"
	"github.com/dbmrq/ralph/internal/tui/components"
	"github.com/dbmrq/ralph/internal/tui/styles"
)

// LoopState is an alias for loop.State for convenience within the TUI package.
// The authoritative state machine is defined in internal/loop/state.go.
type LoopState = loop.State

// Loop state constants - aliases for loop package constants.
const (
	LoopStateIdle        = loop.StateIdle
	LoopStateRunning     = loop.StateRunning
	LoopStatePaused      = loop.StatePaused
	LoopStateAwaitingFix = loop.StateAwaitingFix
	LoopStateCompleted   = loop.StateCompleted
	LoopStateFailed      = loop.StateFailed
)

// Model is the Bubble Tea model for the Ralph TUI.
type Model struct {
	// Components
	header      *components.Header
	progress    *components.Progress
	taskList    *components.TaskList
	statusBar   *components.StatusBar
	logView     *components.LogViewport
	taskEditor  *components.TaskEditor
	modelPicker *components.ModelPicker
	helpOverlay *components.HelpOverlay
	confirmDlg  *components.ConfirmDialog

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
	quitting    bool
	showLogs    bool
	focusedPane FocusedPane

	// Loop control callback (set by SetLoopController)
	loopController LoopController
}

// FocusedPane indicates which pane has focus.
type FocusedPane int

const (
	FocusTasks FocusedPane = iota
	FocusLogs
)

// LoopController provides control over the running loop.
type LoopController interface {
	Pause() error
	Resume() error
	Skip(taskID string) error
	Abort() error
}

// New creates a new TUI model.
func New() *Model {
	return &Model{
		header:      components.NewHeader(),
		progress:    components.NewProgress(),
		taskList:    components.NewTaskList(),
		statusBar:   components.NewStatusBar(),
		logView:     components.NewLogViewport(),
		taskEditor:  components.NewTaskEditor(),
		modelPicker: components.NewModelPicker(),
		helpOverlay: components.NewHelpOverlay(),
		confirmDlg:  components.NewConfirmDialog(),
		loopState:   LoopStateIdle,
		tasks:       []*task.Task{},
		iteration:   0,
		startTime:   time.Now(),
		projectName: "ralph",
		focusedPane: FocusTasks,
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
	// Handle overlay/dialog components first (they capture input when visible)
	if m.confirmDlg.IsVisible() {
		if cmd := m.confirmDlg.Update(msg); cmd != nil {
			return m, cmd
		}
		return m, nil
	}
	if m.helpOverlay.IsVisible() {
		if cmd := m.helpOverlay.Update(msg); cmd != nil {
			return m, cmd
		}
		return m, nil
	}
	if m.modelPicker.IsVisible() {
		if cmd := m.modelPicker.Update(msg); cmd != nil {
			return m, cmd
		}
		return m, nil
	}
	if m.taskEditor.IsActive() {
		if cmd := m.taskEditor.Update(msg); cmd != nil {
			return m, cmd
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.header.SetWidth(msg.Width)
		m.progress.SetWidth(msg.Width)
		m.statusBar.SetWidth(msg.Width)
		// Calculate heights for task list and log view
		taskListHeight := (msg.Height - 8) / 2
		logHeight := msg.Height - 8 - taskListHeight
		m.taskList.SetSize(msg.Width, taskListHeight)
		m.logView.SetSize(msg.Width, logHeight)
		m.taskEditor.SetSize(msg.Width-4, 8)
		m.modelPicker.SetSize(50, 15)
		m.helpOverlay.SetSize(60, 25)
		m.confirmDlg.SetSize(50)
		return m, nil

	case TickMsg:
		// Update time-based displays
		m.statusBar.SetElapsedTime(time.Since(m.startTime))
		return m, tickCmd()

	case TasksUpdatedMsg:
		m.tasks = msg.Tasks
		m.progress.SetProgress(msg.Completed, msg.Total)
		currentTaskID := ""
		if m.currentTask != nil {
			currentTaskID = m.currentTask.ID
		}
		m.taskList.SetTasks(msg.Tasks, currentTaskID)
		return m, nil

	case TaskStartedMsg:
		m.iteration = msg.Iteration
		m.progress.SetIteration(msg.Iteration)
		m.statusBar.SetIteration(msg.Iteration)
		for i, t := range m.tasks {
			if t.ID == msg.TaskID {
				m.currentTask = t
				m.taskList.SetSelected(i)
				break
			}
		}
		return m, nil

	case AgentOutputMsg:
		m.logView.AppendLine(msg.Line)
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
		m.statusBar.SetLoopState(msg.State)
		return m, nil

	case BuildStatusMsg:
		status := "pending"
		if msg.Running {
			status = "running"
		} else if msg.Passed {
			status = "pass"
		} else if msg.Error != "" {
			status = "fail"
		}
		m.statusBar.SetBuildStatus(status)
		return m, nil

	case TestStatusMsg:
		status := "pending"
		if msg.Running {
			status = "running"
		} else if msg.Passed {
			status = "pass"
		} else if msg.Error != "" || msg.Failed_ > 0 {
			status = "fail"
		}
		m.statusBar.SetTestStatus(status)
		return m, nil

	case components.ConfirmYesMsg:
		return m.handleConfirmYes(msg.Action)

	case components.ConfirmNoMsg:
		// Just dismissed, do nothing
		return m, nil

	case components.ModelSelectedMsg:
		m.modelName = msg.Model.Name
		m.updateHeader()
		return m, nil

	case components.TaskEditorSubmitMsg:
		return m.handleTaskEditorSubmit(msg)

	case ErrorMsg:
		m.lastError = msg.Error
		m.statusBar.SetMessage(msg.Error)
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
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "q":
		// Quit requires confirmation if loop is running
		if m.loopState == LoopStateRunning || m.loopState == LoopStatePaused {
			m.confirmDlg.ShowQuit()
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit

	case "?", "h":
		m.helpOverlay.Toggle()
		return m, nil

	case "p":
		// Pause/Resume toggle
		if m.loopController == nil {
			return m, nil
		}
		if m.loopState == LoopStateRunning {
			if err := m.loopController.Pause(); err != nil {
				m.lastError = err.Error()
			}
		} else if m.loopState == LoopStatePaused {
			if err := m.loopController.Resume(); err != nil {
				m.lastError = err.Error()
			}
		}
		return m, nil

	case "s":
		// Skip current task (with confirmation)
		if m.loopState == LoopStateRunning && m.currentTask != nil {
			m.confirmDlg.ShowSkip(m.currentTask.Name)
		}
		return m, nil

	case "a":
		// Abort loop (with confirmation)
		if m.loopState == LoopStateRunning || m.loopState == LoopStatePaused {
			m.confirmDlg.ShowAbort()
		}
		return m, nil

	case "l":
		// Toggle log view
		m.showLogs = !m.showLogs
		if m.showLogs {
			m.focusedPane = FocusLogs
		} else {
			m.focusedPane = FocusTasks
		}
		return m, nil

	case "e":
		// Add/Edit task
		if m.loopState == LoopStateIdle || m.loopState == LoopStatePaused {
			m.taskEditor.StartAdd()
		}
		return m, nil

	case "m":
		// Model picker
		if m.loopState == LoopStateIdle || m.loopState == LoopStatePaused {
			m.modelPicker.Show()
		}
		return m, nil

	case "tab":
		// Toggle focus between panes
		if m.showLogs {
			if m.focusedPane == FocusTasks {
				m.focusedPane = FocusLogs
			} else {
				m.focusedPane = FocusTasks
			}
		}
		return m, nil

	case "j", "down":
		// Navigate down
		if m.focusedPane == FocusTasks {
			m.taskList.MoveDown()
		} else if m.focusedPane == FocusLogs {
			m.logView.ScrollDown()
		}
		return m, nil

	case "k", "up":
		// Navigate up
		if m.focusedPane == FocusTasks {
			m.taskList.MoveUp()
		} else if m.focusedPane == FocusLogs {
			m.logView.ScrollUp()
		}
		return m, nil

	case "g":
		// Go to top
		if m.focusedPane == FocusTasks {
			m.taskList.GoToTop()
		} else if m.focusedPane == FocusLogs {
			m.logView.GoToTop()
		}
		return m, nil

	case "G":
		// Go to bottom
		if m.focusedPane == FocusTasks {
			m.taskList.GoToBottom()
		} else if m.focusedPane == FocusLogs {
			m.logView.GoToBottom()
		}
		return m, nil

	case "f":
		// Toggle auto-follow in logs
		if m.focusedPane == FocusLogs {
			m.logView.ToggleAutoFollow()
		}
		return m, nil

	case "esc":
		// Close any overlay or cancel action
		if m.showLogs && m.focusedPane == FocusLogs {
			m.showLogs = false
			m.focusedPane = FocusTasks
		}
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

// handleConfirmYes handles confirmed actions.
func (m *Model) handleConfirmYes(action components.ConfirmAction) (tea.Model, tea.Cmd) {
	switch action {
	case components.ConfirmActionAbort:
		if m.loopController != nil {
			if err := m.loopController.Abort(); err != nil {
				m.lastError = err.Error()
			}
		}
	case components.ConfirmActionSkip:
		if m.loopController != nil && m.currentTask != nil {
			if err := m.loopController.Skip(m.currentTask.ID); err != nil {
				m.lastError = err.Error()
			}
		}
	case components.ConfirmActionQuit:
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

// handleTaskEditorSubmit handles task editor submissions.
func (m *Model) handleTaskEditorSubmit(msg components.TaskEditorSubmitMsg) (tea.Model, tea.Cmd) {
	// The task editor submission will be handled by the parent
	// which has access to the task manager.
	// For now, we just emit a message that can be handled externally.
	return m, nil
}

// SetLoopController sets the loop controller for pause/resume/skip/abort operations.
func (m *Model) SetLoopController(controller LoopController) {
	m.loopController = controller
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

	// Main content area - task list
	view += m.taskList.View() + "\n"

	// Log view (if enabled)
	if m.showLogs {
		view += lipgloss.NewStyle().
			Foreground(styles.BorderColor).
			Render(repeatChar("─", m.width)) + "\n"
		view += m.logView.View() + "\n"
	}

	// Error display
	if m.lastError != "" {
		view += styles.ErrorTextStyle.Render("Error: "+m.lastError) + "\n"
	}

	// Status bar at bottom
	view += m.statusBar.View()

	// Render overlays on top (centered)
	if m.helpOverlay.IsVisible() {
		view = m.renderOverlay(view, m.helpOverlay.View())
	}
	if m.confirmDlg.IsVisible() {
		view = m.renderOverlay(view, m.confirmDlg.View())
	}
	if m.modelPicker.IsVisible() {
		view = m.renderOverlay(view, m.modelPicker.View())
	}
	if m.taskEditor.IsActive() {
		view = m.renderOverlay(view, m.taskEditor.View())
	}

	return view
}

// renderOverlay renders an overlay component centered on top of the base view.
func (m *Model) renderOverlay(base, overlay string) string {
	if overlay == "" {
		return base
	}

	// Calculate center position
	overlayStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	// For simplicity, just append the overlay at the bottom
	// A more sophisticated approach would use lipgloss.Place
	return base + "\n" + overlayStyle.Render(overlay)
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
	currentTaskID := ""
	if m.currentTask != nil {
		currentTaskID = m.currentTask.ID
	}
	m.taskList.SetTasks(tasks, currentTaskID)
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
