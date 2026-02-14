// Package tui provides the terminal user interface for ralph.
// This file implements the combined setup-to-loop TUI for seamless transitions.
package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/app"
	"github.com/wexinc/ralph/internal/loop"
	"github.com/wexinc/ralph/internal/task"
)

// CombinedPhase represents which phase the combined TUI is in.
type CombinedPhase int

const (
	CombinedPhaseSetup CombinedPhase = iota
	CombinedPhaseTransition
	CombinedPhaseLoop
)

// CombinedModel wraps both SetupModel and the main loop Model,
// allowing seamless transition from setup to loop within a single tea.Program.
type CombinedModel struct {
	phase CombinedPhase

	// Setup phase
	setupModel *SetupModel
	setupDone  bool

	// Loop phase
	loopModel *Model

	// State for transition
	setupResult *app.SetupResult
	setupErr    error

	// Configuration passed in
	ctx        context.Context
	projectDir string
	agent      agent.Agent

	// Callbacks to create loop after setup
	loopFactory func(*app.SetupResult) (*loop.Loop, error)
	onComplete  func(*app.SetupResult) // Called when setup completes (before loop starts)

	// Window dimensions
	width  int
	height int
}

// CombinedConfig configures the combined TUI.
type CombinedConfig struct {
	Context     context.Context
	Agent       agent.Agent
	ProjectDir  string
	LoopFactory func(*app.SetupResult) (*loop.Loop, error)
	OnComplete  func(*app.SetupResult)
}

// NewCombinedModel creates a new combined model that handles both setup and loop.
func NewCombinedModel(cfg CombinedConfig) *CombinedModel {
	setup := app.NewSetup(cfg.ProjectDir, cfg.Agent)
	setupModel := NewSetupModel(cfg.Context, setup)

	return &CombinedModel{
		phase:       CombinedPhaseSetup,
		setupModel:  setupModel,
		loopModel:   nil, // Created after setup completes
		ctx:         cfg.Context,
		projectDir:  cfg.ProjectDir,
		agent:       cfg.Agent,
		loopFactory: cfg.LoopFactory,
		onComplete:  cfg.OnComplete,
	}
}

// Init initializes the combined model.
func (m *CombinedModel) Init() tea.Cmd {
	return m.setupModel.Init()
}

// Update handles messages for the combined model.
func (m *CombinedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward to active model
		if m.phase == CombinedPhaseSetup {
			_, cmd := m.setupModel.Update(msg)
			return m, cmd
		} else if m.phase == CombinedPhaseLoop && m.loopModel != nil {
			_, cmd := m.loopModel.Update(msg)
			return m, cmd
		}
		return m, nil

	case SetupCompleteMsg:
		// Setup completed successfully - transition to loop
		m.setupResult = msg.Result
		m.setupDone = true
		m.phase = CombinedPhaseTransition

		// Call completion callback if provided
		if m.onComplete != nil {
			m.onComplete(msg.Result)
		}

		// Transition to loop
		return m, m.transitionToLoop()

	case SetupErrorMsg:
		// Setup failed - stay in setup phase to show error
		m.setupErr = msg.Error
		return m, nil

	case loopReadyMsg:
		// Loop is ready, switch to loop phase
		m.phase = CombinedPhaseLoop
		// Re-send window size to initialize loop model dimensions
		if m.width > 0 && m.height > 0 {
			return m, func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			}
		}
		return m, m.loopModel.Init()
	}

	// Forward to active model
	switch m.phase {
	case CombinedPhaseSetup:
		newModel, cmd := m.setupModel.Update(msg)
		if sm, ok := newModel.(*SetupModel); ok {
			m.setupModel = sm
		}
		return m, cmd

	case CombinedPhaseLoop:
		if m.loopModel == nil {
			return m, nil
		}
		newModel, cmd := m.loopModel.Update(msg)
		if lm, ok := newModel.(*Model); ok {
			m.loopModel = lm
		}
		return m, cmd
	}

	return m, nil
}

// View renders the combined model.
func (m *CombinedModel) View() string {
	switch m.phase {
	case CombinedPhaseSetup:
		return m.setupModel.View()
	case CombinedPhaseTransition:
		return "✓ Setup complete! Starting Ralph..."
	case CombinedPhaseLoop:
		if m.loopModel != nil {
			return m.loopModel.View()
		}
	}
	return ""
}

// loopReadyMsg signals that the loop model is ready.
type loopReadyMsg struct{}

// transitionToLoop creates the loop model and prepares for loop execution.
func (m *CombinedModel) transitionToLoop() tea.Cmd {
	return func() tea.Msg {
		// Create the loop model
		m.loopModel = New()
		return loopReadyMsg{}
	}
}

// LoopModel returns the loop model (available after setup completes).
func (m *CombinedModel) LoopModel() *Model {
	return m.loopModel
}

// SetupResult returns the setup result (available after setup completes).
func (m *CombinedModel) SetupResult() *app.SetupResult {
	return m.setupResult
}

// Phase returns the current phase.
func (m *CombinedModel) Phase() CombinedPhase {
	return m.phase
}

// CombinedResult contains the final result from the combined TUI.
type CombinedResult struct {
	SetupResult *app.SetupResult
	Canceled    bool
	Error       error
}

// LoopRunnerFunc is the function signature for the loop runner callback.
// It receives the setup result, loop model, and tea.Program to run the loop.
type LoopRunnerFunc func(setupResult *app.SetupResult, model *Model, program *tea.Program) error

// RunCombinedTUI runs the combined setup-to-loop TUI.
// It handles setup, then seamlessly transitions to the main loop.
// The loopRunner function is called after setup completes to start the loop.
func RunCombinedTUI(
	ctx context.Context,
	ag agent.Agent,
	projectDir string,
	tasks []*task.Task,
	sessionInfo SessionInfo,
	loopRunner LoopRunnerFunc,
) (*CombinedResult, error) {
	var setupResult *app.SetupResult

	cfg := CombinedConfig{
		Context:    ctx,
		Agent:      ag,
		ProjectDir: projectDir,
		OnComplete: func(result *app.SetupResult) {
			setupResult = result
		},
	}

	model := NewCombinedModel(cfg)
	program := tea.NewProgram(model, tea.WithAltScreen())

	// Track if loop started
	loopStarted := false
	loopDone := make(chan error, 1)

	// Custom run loop to intercept transition
	go func() {
		for {
			// Check if we've transitioned to loop phase
			if model.Phase() == CombinedPhaseLoop && !loopStarted && model.LoopModel() != nil {
				loopStarted = true

				// Configure loop model with session info
				loopModel := model.LoopModel()
				loopModel.SetSessionInfo(
					sessionInfo.ProjectName,
					sessionInfo.AgentName,
					sessionInfo.ModelName,
					sessionInfo.SessionID,
				)

				// Set tasks from setup result
				if setupResult != nil && len(setupResult.Tasks) > 0 {
					loopModel.SetTasks(setupResult.Tasks)
				} else if len(tasks) > 0 {
					loopModel.SetTasks(tasks)
				}

				// Run the loop with setup result
				go func() {
					if loopRunner != nil {
						loopDone <- loopRunner(setupResult, loopModel, program)
					} else {
						loopDone <- nil
					}
				}()
			}
		}
	}()

	// Run the TUI
	finalModel, err := program.Run()
	if err != nil {
		return nil, err
	}

	// Wait for loop completion (non-blocking check)
	select {
	case loopErr := <-loopDone:
		if loopErr != nil {
			return &CombinedResult{
				SetupResult: setupResult,
				Error:       loopErr,
			}, loopErr
		}
	default:
		// Loop not finished or not started
	}

	// Check final model state
	if cm, ok := finalModel.(*CombinedModel); ok {
		if cm.setupErr != nil {
			return &CombinedResult{
				Error: cm.setupErr,
			}, cm.setupErr
		}
		if !cm.setupDone {
			return &CombinedResult{
				Canceled: true,
			}, nil
		}
		return &CombinedResult{
			SetupResult: cm.setupResult,
		}, nil
	}

	return &CombinedResult{Canceled: true}, nil
}

