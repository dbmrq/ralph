package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dbmrq/ralph/internal/tui/styles"
)

// SetupStep represents a step in the setup process.
type SetupStep struct {
	Name        string
	Description string
	Estimate    time.Duration // Estimated time for this step
}

// DefaultSetupSteps are the standard setup steps.
var DefaultSetupSteps = []SetupStep{
	{Name: "Analyze", Description: "Analyzing project with AI", Estimate: 30 * time.Second},
	{Name: "Confirm", Description: "Confirm analysis results", Estimate: 0},
	{Name: "Tasks", Description: "Import or create tasks", Estimate: 10 * time.Second},
	{Name: "Save", Description: "Save configuration", Estimate: 2 * time.Second},
}

// SetupProgress displays a multi-step progress indicator with spinner.
type SetupProgress struct {
	steps       []SetupStep
	currentStep int
	spinner     spinner.Model
	startTime   time.Time
	stepStart   time.Time
	width       int
	statusText  string
}

// NewSetupProgress creates a new SetupProgress component.
func NewSetupProgress(steps []SetupStep) *SetupProgress {
	if steps == nil {
		steps = DefaultSetupSteps
	}
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Secondary)
	return &SetupProgress{
		steps:       steps,
		currentStep: 0,
		spinner:     s,
	}
}

// SetCurrentStep sets the current step (0-indexed).
func (p *SetupProgress) SetCurrentStep(step int) {
	if step < 0 {
		step = 0
	}
	if step >= len(p.steps) {
		step = len(p.steps) - 1
	}
	p.currentStep = step
	p.stepStart = time.Now()
}

// SetStatusText sets additional status text for the current step.
func (p *SetupProgress) SetStatusText(text string) {
	p.statusText = text
}

// SetWidth sets the width of the component.
func (p *SetupProgress) SetWidth(width int) {
	p.width = width
}

// Start marks the start of the setup process.
func (p *SetupProgress) Start() {
	p.startTime = time.Now()
	p.stepStart = time.Now()
}

// Init returns the initial command for the spinner.
func (p *SetupProgress) Init() tea.Cmd {
	return p.spinner.Tick
}

// Update handles spinner tick messages.
func (p *SetupProgress) Update(msg tea.Msg) (*SetupProgress, tea.Cmd) {
	var cmd tea.Cmd
	p.spinner, cmd = p.spinner.Update(msg)
	return p, cmd
}

// View renders the setup progress display.
func (p *SetupProgress) View() string {
	var sections []string

	// Step indicator: "Step 2/4: Analyzing project with AI"
	stepNum := p.currentStep + 1
	totalSteps := len(p.steps)
	step := p.steps[p.currentStep]

	stepStyle := lipgloss.NewStyle().Foreground(styles.Secondary).Bold(true)
	stepLabel := stepStyle.Render(fmt.Sprintf("Step %d/%d", stepNum, totalSteps))

	// Progress dots: ● ● ○ ○
	var dots []string
	for i := 0; i < totalSteps; i++ {
		if i < p.currentStep {
			// Completed
			dots = append(dots, lipgloss.NewStyle().Foreground(styles.Success).Render("●"))
		} else if i == p.currentStep {
			// Current
			dots = append(dots, lipgloss.NewStyle().Foreground(styles.Secondary).Render("●"))
		} else {
			// Pending
			dots = append(dots, lipgloss.NewStyle().Foreground(styles.Muted).Render("○"))
		}
	}
	dotString := strings.Join(dots, " ")

	sections = append(sections, fmt.Sprintf("%s  %s", stepLabel, dotString))

	// Current step with spinner
	spinnerView := p.spinner.View()
	descStyle := lipgloss.NewStyle().Foreground(styles.Foreground)
	sections = append(sections, fmt.Sprintf("%s %s", spinnerView, descStyle.Render(step.Description)))

	// Additional status text
	if p.statusText != "" {
		statusStyle := lipgloss.NewStyle().Foreground(styles.MutedLight).Italic(true).PaddingLeft(2)
		sections = append(sections, statusStyle.Render(p.statusText))
	}

	// Time info
	if !p.startTime.IsZero() {
		elapsed := time.Since(p.startTime)
		timeStyle := lipgloss.NewStyle().Foreground(styles.MutedLight)
		timeInfo := fmt.Sprintf("Total: %s", formatSpinnerDuration(elapsed))
		if step.Estimate > 0 && !p.stepStart.IsZero() {
			stepElapsed := time.Since(p.stepStart)
			remaining := step.Estimate - stepElapsed
			if remaining > 0 {
				timeInfo += fmt.Sprintf("  •  Est. ~%s remaining", formatSpinnerDuration(remaining))
			}
		}
		sections = append(sections, timeStyle.Render(timeInfo))
	}

	content := strings.Join(sections, "\n")

	if p.width > 0 {
		containerStyle := lipgloss.NewStyle().Width(p.width).Padding(1, 2)
		return containerStyle.Render(content)
	}

	return content
}

