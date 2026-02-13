// Package components provides reusable TUI components for ralph.
package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wexinc/ralph/internal/agent"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// ModelPicker is a component for selecting AI models.
type ModelPicker struct {
	models      []agent.Model
	selected    int
	current     string
	visible     bool
	width       int
	height      int
	scrollStart int
}

// NewModelPicker creates a new ModelPicker component.
func NewModelPicker() *ModelPicker {
	return &ModelPicker{
		models:   []agent.Model{},
		selected: 0,
		current:  "",
		visible:  false,
		width:    50,
		height:   10,
	}
}

// SetModels sets the available models.
func (p *ModelPicker) SetModels(models []agent.Model) {
	p.models = models
	p.selected = 0
	// Find and select the current model or default
	for i, m := range models {
		if m.ID == p.current || m.IsDefault {
			p.selected = i
			break
		}
	}
}

// SetCurrentModel sets the currently active model ID.
func (p *ModelPicker) SetCurrentModel(modelID string) {
	p.current = modelID
	// Update selection to match
	for i, m := range p.models {
		if m.ID == modelID {
			p.selected = i
			break
		}
	}
}

// SetSize sets the picker dimensions.
func (p *ModelPicker) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// CurrentModel returns the currently active model ID.
func (p *ModelPicker) CurrentModel() string {
	return p.current
}

// SelectedModel returns the currently highlighted model.
func (p *ModelPicker) SelectedModel() *agent.Model {
	if len(p.models) == 0 || p.selected >= len(p.models) {
		return nil
	}
	return &p.models[p.selected]
}

// Show makes the picker visible.
func (p *ModelPicker) Show() {
	p.visible = true
}

// Hide hides the picker.
func (p *ModelPicker) Hide() {
	p.visible = false
}

// IsVisible returns whether the picker is visible.
func (p *ModelPicker) IsVisible() bool {
	return p.visible
}

// Toggle toggles the picker visibility.
func (p *ModelPicker) Toggle() {
	p.visible = !p.visible
}

// MoveUp moves the selection up.
func (p *ModelPicker) MoveUp() {
	if p.selected > 0 {
		p.selected--
		p.ensureVisible()
	}
}

// MoveDown moves the selection down.
func (p *ModelPicker) MoveDown() {
	if p.selected < len(p.models)-1 {
		p.selected++
		p.ensureVisible()
	}
}

// ensureVisible ensures the selected item is visible in the scroll area.
func (p *ModelPicker) ensureVisible() {
	visibleHeight := p.height - 4 // Account for title and borders
	if visibleHeight < 1 {
		visibleHeight = 5
	}

	if p.selected < p.scrollStart {
		p.scrollStart = p.selected
	} else if p.selected >= p.scrollStart+visibleHeight {
		p.scrollStart = p.selected - visibleHeight + 1
	}
}

// Update handles input messages.
func (p *ModelPicker) Update(msg tea.Msg) tea.Cmd {
	if !p.visible {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			p.MoveUp()
		case "down", "j":
			p.MoveDown()
		case "enter":
			if model := p.SelectedModel(); model != nil {
				selected := *model
				p.current = selected.ID
				p.Hide()
				return func() tea.Msg {
					return ModelSelectedMsg{Model: selected}
				}
			}
		case "esc", "m", "q":
			p.Hide()
			return func() tea.Msg {
				return ModelPickerClosedMsg{}
			}
		}
	}
	return nil
}

// View renders the model picker.
func (p *ModelPicker) View() string {
	if !p.visible {
		return ""
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Background(styles.Secondary).
		Bold(true).
		Padding(0, 1)
	b.WriteString(titleStyle.Render("Select Model"))
	b.WriteString("\n\n")

	if len(p.models) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(styles.Muted).
			Italic(true)
		b.WriteString(emptyStyle.Render("  No models available"))
		b.WriteString("\n")
	} else {
		visibleHeight := p.height - 4
		if visibleHeight < 1 {
			visibleHeight = 5
		}
		endIdx := p.scrollStart + visibleHeight
		if endIdx > len(p.models) {
			endIdx = len(p.models)
		}

		// Scroll indicator (top)
		if p.scrollStart > 0 {
			b.WriteString(styles.MutedTextStyle.Render("  ↑ more above"))
			b.WriteString("\n")
		}

		for i := p.scrollStart; i < endIdx; i++ {
			model := p.models[i]
			b.WriteString(p.renderModel(model, i == p.selected))
			b.WriteString("\n")
		}

		// Scroll indicator (bottom)
		if endIdx < len(p.models) {
			b.WriteString(styles.MutedTextStyle.Render("  ↓ more below"))
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Italic(true)
	b.WriteString(helpStyle.Render("j/k: navigate  Enter: select  Esc: close"))

	boxStyle := styles.FocusedBoxStyle.Width(p.width - 2)
	return boxStyle.Render(b.String())
}

// renderModel renders a single model item.
func (p *ModelPicker) renderModel(model agent.Model, selected bool) string {
	// Selection indicator
	indicator := "  "
	if selected {
		indicator = lipgloss.NewStyle().
			Foreground(styles.Secondary).
			Bold(true).
			Render("▶ ")
	}

	// Current model indicator
	currentIndicator := ""
	if model.ID == p.current {
		currentIndicator = lipgloss.NewStyle().
			Foreground(styles.Success).
			Render(" ✓ current")
	}

	// Default indicator
	defaultIndicator := ""
	if model.IsDefault && model.ID != p.current {
		defaultIndicator = lipgloss.NewStyle().
			Foreground(styles.Muted).
			Render(" (default)")
	}

	// Model name
	nameStyle := lipgloss.NewStyle().Foreground(styles.Foreground)
	if selected {
		nameStyle = nameStyle.Bold(true)
	}

	// Model description
	descStr := ""
	if model.Description != "" {
		descStyle := lipgloss.NewStyle().Foreground(styles.MutedLight)
		descStr = " - " + descStyle.Render(model.Description)
	}

	name := model.Name
	if name == "" {
		name = model.ID
	}

	return indicator + nameStyle.Render(name) + currentIndicator + defaultIndicator + descStr
}

// ModelSelectedMsg is sent when a model is selected.
type ModelSelectedMsg struct {
	Model agent.Model
}

// ModelPickerClosedMsg is sent when the picker is closed without selection.
type ModelPickerClosedMsg struct{}

