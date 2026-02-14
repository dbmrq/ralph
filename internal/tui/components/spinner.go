package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dbmrq/ralph/internal/tui/styles"
)

// Spinner is a component that displays an animated spinner with optional status text.
type Spinner struct {
	spinner    spinner.Model
	statusText string
	startTime  time.Time
	estimate   time.Duration
	width      int
	showTime   bool
}

// NewSpinner creates a new Spinner component with default styling.
func NewSpinner() *Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Secondary)
	return &Spinner{
		spinner:  s,
		showTime: true,
	}
}

// NewSpinnerWithStyle creates a new Spinner with a custom spinner style.
func NewSpinnerWithStyle(style spinner.Spinner) *Spinner {
	s := spinner.New()
	s.Spinner = style
	s.Style = lipgloss.NewStyle().Foreground(styles.Secondary)
	return &Spinner{
		spinner:  s,
		showTime: true,
	}
}

// SetStatusText sets the status text to display next to the spinner.
func (s *Spinner) SetStatusText(text string) {
	s.statusText = text
}

// SetEstimate sets the estimated time for the operation.
func (s *Spinner) SetEstimate(d time.Duration) {
	s.estimate = d
}

// SetShowTime controls whether elapsed time is shown.
func (s *Spinner) SetShowTime(show bool) {
	s.showTime = show
}

// SetWidth sets the width of the spinner component.
func (s *Spinner) SetWidth(width int) {
	s.width = width
}

// Start marks the start time for elapsed time tracking.
func (s *Spinner) Start() {
	s.startTime = time.Now()
}

// Elapsed returns the elapsed time since Start was called.
func (s *Spinner) Elapsed() time.Duration {
	if s.startTime.IsZero() {
		return 0
	}
	return time.Since(s.startTime)
}

// Init returns the initial command for the spinner animation.
func (s *Spinner) Init() tea.Cmd {
	return s.spinner.Tick
}

// Update handles spinner tick messages.
func (s *Spinner) Update(msg tea.Msg) (*Spinner, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// View renders the spinner with status text and optional timing info.
func (s *Spinner) View() string {
	// Main spinner and status
	spinnerView := s.spinner.View()
	statusStyle := lipgloss.NewStyle().Foreground(styles.Foreground)
	status := statusStyle.Render(s.statusText)

	// Build the main line
	line := fmt.Sprintf("%s %s", spinnerView, status)

	// Add elapsed time if enabled and started
	if s.showTime && !s.startTime.IsZero() {
		elapsed := time.Since(s.startTime)
		timeStyle := lipgloss.NewStyle().Foreground(styles.MutedLight)

		elapsedStr := formatSpinnerDuration(elapsed)
		if s.estimate > 0 {
			// Show estimated time remaining
			remaining := s.estimate - elapsed
			if remaining < 0 {
				remaining = 0
			}
			line = fmt.Sprintf("%s %s(elapsed: %s, est. remaining: %s)",
				line, timeStyle.Render(""), elapsedStr, formatSpinnerDuration(remaining))
		} else {
			line = fmt.Sprintf("%s %s", line, timeStyle.Render(fmt.Sprintf("(%s)", elapsedStr)))
		}
	}

	// Apply width if set
	if s.width > 0 {
		containerStyle := lipgloss.NewStyle().
			Width(s.width).
			Padding(0, 1)
		return containerStyle.Render(line)
	}

	return line
}

// formatSpinnerDuration formats a duration for display.
func formatSpinnerDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}
