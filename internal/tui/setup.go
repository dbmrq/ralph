// Package tui provides the terminal user interface for ralph.
// This file implements the setup flow TUI for first-run experience.
package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/app"
	"github.com/wexinc/ralph/internal/build"
	"github.com/wexinc/ralph/internal/task"
	"github.com/wexinc/ralph/internal/tui/components"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// SetupPhase represents the current phase of the setup flow.
type SetupPhase int

const (
	PhaseWelcome SetupPhase = iota
	PhaseAnalyzing
	PhaseAnalysisConfirm
	PhaseTaskDetection
	PhaseTaskInit
	PhaseTaskFileInput  // New: file path input
	PhaseTaskPaste      // New: paste task list
	PhaseTaskGenerate   // New: generate from goal
	PhaseTaskGenerating // New: generating tasks in progress
	PhaseTaskConfirm
	PhaseComplete
	PhaseError
)

// SetupModel is the Bubble Tea model for the setup flow.
type SetupModel struct {
	// Phase is the current setup phase.
	Phase SetupPhase

	// Components
	analysisForm *components.AnalysisForm
	taskInit     *components.TaskInitSelector
	taskListForm *components.TaskListForm
	textInput    *components.TextInput
	fileInput    *components.FileInput
	taskPaste    *components.TaskPaste
	goalInput    *components.GoalInput

	// State
	setup     *app.Setup
	ctx       context.Context
	analysis  *build.ProjectAnalysis
	detection *task.TaskListDetection
	tasks     []*task.Task
	errorMsg  string
	statusMsg string
	initMode  components.TaskInitMode

	// Window
	width  int
	height int

	// Result channel for async operations
	resultChan chan interface{}
}

// SetupCompleteMsg is sent when setup completes successfully.
type SetupCompleteMsg struct {
	Result *app.SetupResult
}

// SetupErrorMsg is sent when setup fails.
type SetupErrorMsg struct {
	Error error
}

// analysisCompleteMsg is sent when analysis completes.
type analysisCompleteMsg struct {
	analysis *build.ProjectAnalysis
	err      error
}

// tasksImportedMsg is sent when tasks are imported.
type tasksImportedMsg struct {
	tasks []*task.Task
	err   error
}

// tasksGeneratedMsg is sent when tasks are generated from a goal.
type tasksGeneratedMsg struct {
	tasks []*task.Task
	err   error
}

// NewSetupModel creates a new SetupModel.
func NewSetupModel(ctx context.Context, setup *app.Setup) *SetupModel {
	m := &SetupModel{
		Phase:        PhaseWelcome,
		ctx:          ctx,
		setup:        setup,
		analysisForm: components.NewAnalysisForm(),
		taskInit:     components.NewTaskInitSelector(),
		taskListForm: components.NewTaskListForm(),
		textInput:    components.NewTextInput("input", ""),
		fileInput:    components.NewFileInput(setup.ProjectDir),
		taskPaste:    components.NewTaskPaste(),
		goalInput:    components.NewGoalInput(),
		resultChan:   make(chan interface{}, 1),
	}
	return m
}

// Init initializes the setup model.
func (m *SetupModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m *SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.analysisForm.SetWidth(msg.Width - 4)
		m.taskInit.SetWidth(msg.Width - 4)
		m.taskListForm.SetWidth(msg.Width - 4)
		m.fileInput.SetWidth(msg.Width - 4)
		m.taskPaste.SetWidth(msg.Width - 4)
		m.taskPaste.SetHeight(msg.Height - 10)
		m.goalInput.SetWidth(msg.Width - 4)
		m.goalInput.SetHeight(msg.Height - 10)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case analysisCompleteMsg:
		return m.handleAnalysisComplete(msg)

	case tasksImportedMsg:
		return m.handleTasksImported(msg)

	case tasksGeneratedMsg:
		return m.handleTasksGenerated(msg)

	case components.AnalysisConfirmedMsg:
		return m.handleAnalysisConfirmed(msg)

	case components.ReanalyzeRequestedMsg:
		return m.startAnalysis()

	case components.TaskInitSelectedMsg:
		return m.handleTaskInitSelected(msg)

	case components.TaskListConfirmedMsg:
		return m.handleTaskListConfirmed(msg)

	case components.TaskListReparseMsg:
		m.Phase = PhaseTaskInit
		return m, nil

	case components.FileInputSubmittedMsg:
		return m.handleFileInputSubmitted(msg)

	case components.FileInputCanceledMsg:
		m.Phase = PhaseTaskInit
		return m, nil

	case components.TaskPasteSubmittedMsg:
		return m.handleTaskPasteSubmitted(msg)

	case components.TaskPasteCanceledMsg:
		m.Phase = PhaseTaskInit
		return m, nil

	case components.GoalSubmittedMsg:
		return m.handleGoalSubmitted(msg)

	case components.GoalCanceledMsg:
		m.Phase = PhaseTaskInit
		return m, nil
	}

	// Delegate to current phase component
	return m.updateCurrentPhase(msg)
}

// updateCurrentPhase delegates to the component for the current phase.
func (m *SetupModel) updateCurrentPhase(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.Phase {
	case PhaseAnalysisConfirm:
		var cmd tea.Cmd
		m.analysisForm, cmd = m.analysisForm.Update(msg)
		return m, cmd

	case PhaseTaskInit:
		var cmd tea.Cmd
		m.taskInit, cmd = m.taskInit.Update(msg)
		return m, cmd

	case PhaseTaskFileInput:
		var cmd tea.Cmd
		m.fileInput, cmd = m.fileInput.Update(msg)
		return m, cmd

	case PhaseTaskPaste:
		var cmd tea.Cmd
		m.taskPaste, cmd = m.taskPaste.Update(msg)
		return m, cmd

	case PhaseTaskGenerate:
		var cmd tea.Cmd
		m.goalInput, cmd = m.goalInput.Update(msg)
		return m, cmd

	case PhaseTaskConfirm:
		var cmd tea.Cmd
		m.taskListForm, cmd = m.taskListForm.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handleKeyPress handles keyboard input.
func (m *SetupModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.Phase == PhaseWelcome || m.Phase == PhaseError {
			return m, tea.Quit
		}
	case "enter":
		if m.Phase == PhaseWelcome {
			return m.startSetup()
		}
	case "esc":
		if m.Phase == PhaseError {
			return m, tea.Quit
		}
	}
	return m, nil
}

// startSetup begins the setup flow.
func (m *SetupModel) startSetup() (tea.Model, tea.Cmd) {
	// Create .ralph directory
	if err := m.setup.CreateRalphDir(); err != nil {
		m.Phase = PhaseError
		m.errorMsg = fmt.Sprintf("Failed to create .ralph directory: %v", err)
		return m, nil
	}

	// Start analysis
	return m.startAnalysis()
}

// startAnalysis starts the project analysis.
func (m *SetupModel) startAnalysis() (tea.Model, tea.Cmd) {
	m.Phase = PhaseAnalyzing
	m.statusMsg = "Running AI analysis..."

	return m, func() tea.Msg {
		analysis, err := m.setup.RunAnalysis(m.ctx)
		return analysisCompleteMsg{analysis: analysis, err: err}
	}
}

// handleAnalysisComplete handles analysis completion.
func (m *SetupModel) handleAnalysisComplete(msg analysisCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.Phase = PhaseError
		m.errorMsg = fmt.Sprintf("Analysis failed: %v", msg.err)
		return m, nil
	}

	m.analysis = msg.analysis
	m.analysisForm.SetAnalysis(m.analysis)
	m.Phase = PhaseAnalysisConfirm
	return m, m.analysisForm.Focus()
}

// handleAnalysisConfirmed handles when the user confirms analysis.
func (m *SetupModel) handleAnalysisConfirmed(msg components.AnalysisConfirmedMsg) (tea.Model, tea.Cmd) {
	m.analysis = msg.Analysis

	// Save analysis
	if err := m.setup.SaveAnalysis(m.analysis); err != nil {
		// Non-fatal, log and continue
		m.statusMsg = fmt.Sprintf("Warning: failed to cache analysis: %v", err)
	}

	// Check for task list detection
	if m.analysis.TaskList.Detected {
		m.detection = &task.TaskListDetection{
			Detected:  m.analysis.TaskList.Detected,
			Path:      m.analysis.TaskList.Path,
			Format:    m.analysis.TaskList.Format,
			TaskCount: m.analysis.TaskList.TaskCount,
		}
		m.taskInit.SetDetection(m.detection)
	}

	m.Phase = PhaseTaskInit
	return m, nil
}

// handleTaskInitSelected handles task init mode selection.
func (m *SetupModel) handleTaskInitSelected(msg components.TaskInitSelectedMsg) (tea.Model, tea.Cmd) {
	m.initMode = msg.Mode

	switch msg.Mode {
	case components.TaskInitModeFile:
		// File mode: If we have detection, import directly; otherwise show file input
		if m.detection != nil && m.detection.Detected {
			return m.importDetectedTasks()
		}
		// Show file input component
		m.Phase = PhaseTaskFileInput
		return m, m.fileInput.Focus()

	case components.TaskInitModePaste:
		// Paste mode: Show textarea for pasting task list
		m.Phase = PhaseTaskPaste
		return m, m.taskPaste.Focus()

	case components.TaskInitModeGenerate:
		// Generate mode: Show goal input for AI task generation
		m.Phase = PhaseTaskGenerate
		return m, m.goalInput.Focus()

	case components.TaskInitModeEmpty:
		m.tasks = []*task.Task{}
		return m.finalizeSetup()
	}

	// If we have detection and no specific mode, import detected tasks
	if m.detection != nil && m.detection.Detected {
		return m.importDetectedTasks()
	}

	return m, nil
}

// importDetectedTasks imports tasks from the detected task list.
func (m *SetupModel) importDetectedTasks() (tea.Model, tea.Cmd) {
	m.Phase = PhaseTaskDetection
	m.statusMsg = fmt.Sprintf("Importing tasks from %s...", m.detection.Path)

	return m, func() tea.Msg {
		tasks, err := m.setup.ImportTasks(m.ctx, m.detection)
		return tasksImportedMsg{tasks: tasks, err: err}
	}
}

// handleTasksImported handles when tasks are imported.
func (m *SetupModel) handleTasksImported(msg tasksImportedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.Phase = PhaseError
		m.errorMsg = fmt.Sprintf("Failed to import tasks: %v", msg.err)
		return m, nil
	}

	m.tasks = msg.tasks
	m.taskListForm.SetTasks(m.tasks)
	m.Phase = PhaseTaskConfirm
	return m, m.taskListForm.Focus()
}

// handleTaskListConfirmed handles when user confirms the task list.
func (m *SetupModel) handleTaskListConfirmed(msg components.TaskListConfirmedMsg) (tea.Model, tea.Cmd) {
	m.tasks = msg.Tasks
	return m.finalizeSetup()
}

// handleFileInputSubmitted handles when user submits a file path.
func (m *SetupModel) handleFileInputSubmitted(msg components.FileInputSubmittedMsg) (tea.Model, tea.Cmd) {
	m.Phase = PhaseTaskDetection
	m.statusMsg = fmt.Sprintf("Importing tasks from %s...", msg.Path)

	return m, func() tea.Msg {
		tasks, err := m.setup.ImportTasksFromFile(msg.Path)
		return tasksImportedMsg{tasks: tasks, err: err}
	}
}

// handleTaskPasteSubmitted handles when user submits pasted task content.
func (m *SetupModel) handleTaskPasteSubmitted(msg components.TaskPasteSubmittedMsg) (tea.Model, tea.Cmd) {
	// Use the already-parsed tasks from the component
	m.tasks = msg.Tasks
	m.taskListForm.SetTasks(m.tasks)
	m.Phase = PhaseTaskConfirm
	return m, m.taskListForm.Focus()
}

// handleGoalSubmitted handles when user submits a goal for task generation.
func (m *SetupModel) handleGoalSubmitted(msg components.GoalSubmittedMsg) (tea.Model, tea.Cmd) {
	m.Phase = PhaseTaskGenerating
	m.statusMsg = "Generating tasks from goal..."

	return m, func() tea.Msg {
		tasks, err := m.setup.GenerateTasks(m.ctx, msg.Goal)
		return tasksGeneratedMsg{tasks: tasks, err: err}
	}
}

// handleTasksGenerated handles when AI task generation completes.
func (m *SetupModel) handleTasksGenerated(msg tasksGeneratedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.Phase = PhaseError
		m.errorMsg = fmt.Sprintf("Failed to generate tasks: %v", msg.err)
		return m, nil
	}

	m.tasks = msg.tasks
	m.taskListForm.SetTasks(m.tasks)
	m.Phase = PhaseTaskConfirm
	return m, m.taskListForm.Focus()
}

// finalizeSetup saves everything and completes the setup.
func (m *SetupModel) finalizeSetup() (tea.Model, tea.Cmd) {
	// Save tasks
	if err := m.setup.SaveTasks(m.tasks); err != nil {
		m.Phase = PhaseError
		m.errorMsg = fmt.Sprintf("Failed to save tasks: %v", err)
		return m, nil
	}

	// Build and save config
	cfg := m.setup.BuildConfigFromAnalysis(m.analysis)
	if err := m.setup.SaveConfig(cfg); err != nil {
		// Non-fatal, continue
		m.statusMsg = fmt.Sprintf("Warning: failed to save config: %v", err)
	}

	m.Phase = PhaseComplete

	return m, func() tea.Msg {
		return SetupCompleteMsg{
			Result: &app.SetupResult{
				Config:   cfg,
				Analysis: m.analysis,
				Tasks:    m.tasks,
			},
		}
	}
}

// View renders the setup UI.
func (m *SetupModel) View() string {
	switch m.Phase {
	case PhaseWelcome:
		return m.viewWelcome()
	case PhaseAnalyzing:
		return m.viewAnalyzing()
	case PhaseAnalysisConfirm:
		return m.viewAnalysisConfirm()
	case PhaseTaskInit:
		return m.viewTaskInit()
	case PhaseTaskFileInput:
		return m.viewTaskFileInput()
	case PhaseTaskPaste:
		return m.viewTaskPaste()
	case PhaseTaskGenerate:
		return m.viewTaskGenerate()
	case PhaseTaskGenerating:
		return m.viewTaskGenerating()
	case PhaseTaskDetection:
		return m.viewTaskDetection()
	case PhaseTaskConfirm:
		return m.viewTaskConfirm()
	case PhaseComplete:
		return m.viewComplete()
	case PhaseError:
		return m.viewError()
	}
	return ""
}

// viewWelcome renders the welcome screen.
func (m *SetupModel) viewWelcome() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		Padding(1, 2)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Padding(0, 2)

	actionStyle := lipgloss.NewStyle().
		Foreground(styles.Success).
		Padding(1, 2)

	return fmt.Sprintf(
		"%s\n%s\n%s\n\n%s",
		titleStyle.Render("🐺 Welcome to Ralph!"),
		subtitleStyle.Render("Ralph helps you automate task execution with AI agents."),
		subtitleStyle.Render("Let's set up your project..."),
		actionStyle.Render("Press Enter to continue, or q to quit"),
	)
}

// viewAnalyzing renders the analyzing screen.
func (m *SetupModel) viewAnalyzing() string {
	return fmt.Sprintf("🔍 %s", m.statusMsg)
}

// viewAnalysisConfirm renders the analysis confirmation form.
func (m *SetupModel) viewAnalysisConfirm() string {
	return m.analysisForm.View()
}

// viewTaskInit renders the task initialization selector.
func (m *SetupModel) viewTaskInit() string {
	return m.taskInit.View()
}

// viewTaskFileInput renders the file path input.
func (m *SetupModel) viewTaskFileInput() string {
	return m.fileInput.View()
}

// viewTaskPaste renders the task paste textarea.
func (m *SetupModel) viewTaskPaste() string {
	return m.taskPaste.View()
}

// viewTaskGenerate renders the goal input for task generation.
func (m *SetupModel) viewTaskGenerate() string {
	return m.goalInput.View()
}

// viewTaskGenerating renders the task generation progress.
func (m *SetupModel) viewTaskGenerating() string {
	return fmt.Sprintf("🤖 %s", m.statusMsg)
}

// viewTaskDetection renders the task detection progress.
func (m *SetupModel) viewTaskDetection() string {
	return fmt.Sprintf("📋 %s", m.statusMsg)
}

// viewTaskConfirm renders the task list confirmation form.
func (m *SetupModel) viewTaskConfirm() string {
	return m.taskListForm.View()
}

// viewComplete renders the completion screen.
func (m *SetupModel) viewComplete() string {
	successStyle := lipgloss.NewStyle().
		Foreground(styles.Success).
		Bold(true).
		Padding(1, 2)

	return successStyle.Render("✓ Setup complete! Starting Ralph...")
}

// viewError renders the error screen.
func (m *SetupModel) viewError() string {
	errorStyle := lipgloss.NewStyle().
		Foreground(styles.Error).
		Bold(true).
		Padding(1, 2)

	return fmt.Sprintf(
		"%s\n\n%s",
		errorStyle.Render("✗ Setup failed"),
		m.errorMsg,
	)
}

// RunSetupTUI runs the setup flow TUI and returns the result.
func RunSetupTUI(ctx context.Context, ag agent.Agent, projectDir string) (*app.SetupResult, error) {
	setup := app.NewSetup(projectDir, ag)
	model := NewSetupModel(ctx, setup)

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}

	// Check the final model for result
	if m, ok := finalModel.(*SetupModel); ok {
		if m.Phase == PhaseComplete {
			return &app.SetupResult{
				Config:   m.setup.BuildConfigFromAnalysis(m.analysis),
				Analysis: m.analysis,
				Tasks:    m.tasks,
			}, nil
		}
		if m.Phase == PhaseError {
			return nil, fmt.Errorf("setup error: %s", m.errorMsg)
		}
	}

	return nil, fmt.Errorf("setup canceled")
}
