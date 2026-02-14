// Package components provides reusable TUI components for ralph.
package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/dbmrq/ralph/internal/tui/styles"
)

// HeaderData contains the data to display in the header.
type HeaderData struct {
	ProjectName string
	AgentName   string
	ModelName   string
	SessionID   string
}

// Header is a component that displays project info in a header bar.
type Header struct {
	data  HeaderData
	width int
}

// NewHeader creates a new Header component.
func NewHeader() *Header {
	return &Header{
		data: HeaderData{
			ProjectName: "ralph",
			AgentName:   "-",
			ModelName:   "-",
			SessionID:   "-",
		},
	}
}

// SetData updates the header data.
func (h *Header) SetData(data HeaderData) {
	h.data = data
}

// SetProjectName sets the project name.
func (h *Header) SetProjectName(name string) {
	h.data.ProjectName = name
}

// SetAgentName sets the agent name.
func (h *Header) SetAgentName(name string) {
	h.data.AgentName = name
}

// SetModelName sets the model name.
func (h *Header) SetModelName(name string) {
	h.data.ModelName = name
}

// SetSessionID sets the session ID.
func (h *Header) SetSessionID(id string) {
	h.data.SessionID = id
}

// SetWidth sets the width for the header.
func (h *Header) SetWidth(width int) {
	h.width = width
}

// View renders the header.
func (h *Header) View() string {
	// Title
	title := styles.TitleStyle.Render("RALPH LOOP")

	// Separator
	sep := lipgloss.NewStyle().
		Foreground(styles.MutedLight).
		Render(" │ ")

	// Build info items
	projectLabel := styles.HeaderLabelStyle.Render("Project: ")
	projectValue := styles.HeaderValueStyle.Render(h.data.ProjectName)

	agentLabel := styles.HeaderLabelStyle.Render("Agent: ")
	agentValue := styles.HeaderValueStyle.Render(h.data.AgentName)

	modelLabel := styles.HeaderLabelStyle.Render("Model: ")
	modelValue := styles.HeaderValueStyle.Render(h.data.ModelName)

	// Build the header content
	content := fmt.Sprintf("%s%s%s%s%s%s%s%s%s%s",
		title, sep,
		projectLabel, projectValue, sep,
		agentLabel, agentValue, sep,
		modelLabel, modelValue,
	)

	// Add session ID if short enough
	if h.data.SessionID != "" && h.data.SessionID != "-" {
		shortSession := h.data.SessionID
		if len(shortSession) > 8 {
			shortSession = shortSession[:8]
		}
		sessionLabel := styles.HeaderLabelStyle.Render("Session: ")
		sessionValue := styles.HeaderValueStyle.Render(shortSession)
		content = fmt.Sprintf("%s%s%s%s", content, sep, sessionLabel, sessionValue)
	}

	// Apply header style and width
	headerStyle := lipgloss.NewStyle().
		Background(styles.Primary).
		Foreground(styles.Foreground).
		Padding(0, 1)

	if h.width > 0 {
		headerStyle = headerStyle.Width(h.width)
	}

	return headerStyle.Render(content)
}
