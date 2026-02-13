package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// ProgressData contains the data to display in the progress bar.
type ProgressData struct {
	Completed  int
	Total      int
	Iteration  int
	StatusText string // Optional additional status text
}

// Progress is a component that displays task progress as a bar.
type Progress struct {
	data  ProgressData
	width int
}

// NewProgress creates a new Progress component.
func NewProgress() *Progress {
	return &Progress{
		data: ProgressData{
			Completed: 0,
			Total:     0,
			Iteration: 0,
		},
	}
}

// SetData updates the progress data.
func (p *Progress) SetData(data ProgressData) {
	p.data = data
}

// SetProgress sets completed and total counts.
func (p *Progress) SetProgress(completed, total int) {
	p.data.Completed = completed
	p.data.Total = total
}

// SetIteration sets the current iteration number.
func (p *Progress) SetIteration(iteration int) {
	p.data.Iteration = iteration
}

// SetStatusText sets optional status text.
func (p *Progress) SetStatusText(text string) {
	p.data.StatusText = text
}

// SetWidth sets the width for the progress bar.
func (p *Progress) SetWidth(width int) {
	p.width = width
}

// View renders the progress bar.
func (p *Progress) View() string {
	// Calculate progress percentage
	var percent float64
	if p.data.Total > 0 {
		percent = float64(p.data.Completed) / float64(p.data.Total)
	}

	// Progress bar width (leave room for count and iteration)
	barWidth := 20
	if p.width > 60 {
		barWidth = 30
	}
	if p.width > 80 {
		barWidth = 40
	}

	// Build the progress bar
	filled := int(percent * float64(barWidth))
	empty := barWidth - filled

	filledStr := styles.ProgressFilledStyle.Render(strings.Repeat("█", filled))
	emptyStr := styles.ProgressEmptyStyle.Render(strings.Repeat("░", empty))
	bar := filledStr + emptyStr

	// Task count
	countStyle := lipgloss.NewStyle().
		Foreground(styles.Secondary)
	count := countStyle.Render(fmt.Sprintf("%d/%d tasks", p.data.Completed, p.data.Total))

	// Iteration
	iterStyle := lipgloss.NewStyle().
		Foreground(styles.MutedLight)
	iteration := ""
	if p.data.Iteration > 0 {
		iteration = iterStyle.Render(fmt.Sprintf("Iteration %d", p.data.Iteration))
	}

	// Separator
	sep := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Render(" │ ")

	// Build the content
	content := fmt.Sprintf("Progress: %s%s%s", bar, sep, count)
	if iteration != "" {
		content = fmt.Sprintf("%s%s%s", content, sep, iteration)
	}

	// Add optional status text
	if p.data.StatusText != "" {
		statusStyle := lipgloss.NewStyle().
			Foreground(styles.MutedLight).
			Italic(true)
		content = fmt.Sprintf("%s%s%s", content, sep, statusStyle.Render(p.data.StatusText))
	}

	// Apply container style
	containerStyle := lipgloss.NewStyle().
		Padding(0, 1)

	if p.width > 0 {
		containerStyle = containerStyle.Width(p.width)
	}

	return containerStyle.Render(content)
}

// PercentComplete returns the completion percentage (0.0 - 1.0).
func (p *Progress) PercentComplete() float64 {
	if p.data.Total == 0 {
		return 0
	}
	return float64(p.data.Completed) / float64(p.data.Total)
}

// IsComplete returns true if all tasks are completed.
func (p *Progress) IsComplete() bool {
	return p.data.Total > 0 && p.data.Completed >= p.data.Total
}

