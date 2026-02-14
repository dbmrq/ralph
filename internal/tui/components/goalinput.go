// Package components provides reusable TUI components for ralph.
package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wexinc/ralph/internal/tui/styles"
)

// GoalSubmittedMsg is sent when user submits the goal for task generation.
type GoalSubmittedMsg struct {
	Goal string
}

// GoalCanceledMsg is sent when user cancels goal input.
type GoalCanceledMsg struct{}

// GoalInput is a component for entering a project goal to generate tasks.
type GoalInput struct {
	textarea  textarea.Model
	width     int
	height    int
	focused   bool
	examples  []string
}

// NewGoalInput creates a new GoalInput component.
func NewGoalInput() *GoalInput {
	ta := textarea.New()
	ta.Placeholder = "Describe what you want to build or accomplish...\n\nExample: Build a REST API with user authentication,\nproduct CRUD operations, and Stripe payment integration."
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(5)
	ta.ShowLineNumbers = false

	return &GoalInput{
		textarea: ta,
		examples: []string{
			"Build a CLI tool that converts markdown files to PDF",
			"Create a web scraper that monitors price changes",
			"Implement user authentication with OAuth2 support",
			"Add a caching layer to the existing API",
		},
	}
}

// SetWidth sets the component width.
func (g *GoalInput) SetWidth(width int) {
	g.width = width
	g.textarea.SetWidth(width - 4)
}

// SetHeight sets the component height.
func (g *GoalInput) SetHeight(height int) {
	g.height = height
	// Reserve space for title, examples, help
	textareaHeight := height - 16
	if textareaHeight < 3 {
		textareaHeight = 3
	}
	g.textarea.SetHeight(textareaHeight)
}

// Focus focuses the textarea.
func (g *GoalInput) Focus() tea.Cmd {
	g.focused = true
	return g.textarea.Focus()
}

// Blur removes focus from the textarea.
func (g *GoalInput) Blur() {
	g.focused = false
	g.textarea.Blur()
}

// Value returns the current goal text.
func (g *GoalInput) Value() string {
	return g.textarea.Value()
}

// SetValue sets the goal text.
func (g *GoalInput) SetValue(value string) {
	g.textarea.SetValue(value)
}

// Update handles messages for the component.
func (g *GoalInput) Update(msg tea.Msg) (*GoalInput, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+enter":
			goal := strings.TrimSpace(g.textarea.Value())
			if goal == "" {
				return g, nil
			}
			return g, func() tea.Msg {
				return GoalSubmittedMsg{Goal: goal}
			}
		case "esc":
			return g, func() tea.Msg {
				return GoalCanceledMsg{}
			}
		}
	}

	var cmd tea.Cmd
	g.textarea, cmd = g.textarea.Update(msg)
	return g, cmd
}

// View renders the component.
func (g *GoalInput) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Bold(true).
		Padding(0, 1)

	b.WriteString(titleStyle.Render("🎯 Describe Your Goal"))
	b.WriteString("\n\n")

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(styles.MutedLight).
		PaddingLeft(1)

	b.WriteString(subtitleStyle.Render("Describe what you want to build. AI will generate a task list."))
	b.WriteString("\n\n")

	// Textarea
	b.WriteString(g.textarea.View())
	b.WriteString("\n\n")

	// Example prompts
	exampleTitleStyle := lipgloss.NewStyle().
		Foreground(styles.Secondary).
		Bold(true)

	b.WriteString(exampleTitleStyle.Render("Example goals:"))
	b.WriteString("\n")

	mutedStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	for _, example := range g.examples {
		b.WriteString(mutedStyle.Render("  • " + example))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	keyStyle := lipgloss.NewStyle().Foreground(styles.Secondary)
	b.WriteString(keyStyle.Render("Ctrl+Enter") + helpStyle.Render(": generate tasks") +
		helpStyle.Render(" │ ") +
		keyStyle.Render("Esc") + helpStyle.Render(": cancel"))

	return b.String()
}

