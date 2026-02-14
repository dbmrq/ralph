// Package components provides reusable TUI components for ralph.
package components

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dbmrq/ralph/internal/project"
	"github.com/dbmrq/ralph/internal/tui/styles"
)

// DirPickerMode represents the current mode of the directory picker.
type DirPickerMode int

const (
	// DirPickerModeList shows the list of recent and suggested directories.
	DirPickerModeList DirPickerMode = iota
	// DirPickerModeManual shows a text input for manual path entry.
	DirPickerModeManual
)

// DirSelectedMsg is sent when a directory is selected.
type DirSelectedMsg struct {
	Path    string
	Project *project.ProjectInfo
}

// DirCanceledMsg is sent when directory selection is canceled.
type DirCanceledMsg struct{}

// DirPickerItem represents an item in the directory picker list.
type DirPickerItem struct {
	Path        string
	Name        string
	ProjectType string
	IsRecent    bool
	IsCurrent   bool
}

// DirPicker is a directory selection component.
type DirPicker struct {
	items       []DirPickerItem
	selectedIdx int
	mode        DirPickerMode
	textInput   *TextInput
	width       int
	height      int
	detector    *project.Detector
	currentDir  string
	errorMsg    string
}

// NewDirPicker creates a new DirPicker component.
func NewDirPicker() *DirPicker {
	return &DirPicker{
		items:     []DirPickerItem{},
		textInput: NewTextInput("path", ""),
		detector:  project.NewDetector(),
	}
}

// Init initializes the directory picker with recent projects and current directory.
func (d *DirPicker) Init(currentDir string, recent *project.RecentProjects) {
	d.currentDir = currentDir
	d.items = []DirPickerItem{}

	// Add current directory if it's a valid project
	if proj, err := d.detector.DetectProject(currentDir); err == nil && proj != nil {
		d.items = append(d.items, DirPickerItem{
			Path:        proj.Path,
			Name:        proj.Name + " (current)",
			ProjectType: proj.ProjectType,
			IsCurrent:   true,
		})
	}

	// Add recent projects
	if recent != nil {
		for _, rp := range recent.Projects {
			// Skip if same as current
			if rp.Path == currentDir {
				continue
			}
			// Skip if doesn't exist
			if _, err := os.Stat(rp.Path); err != nil {
				continue
			}
			d.items = append(d.items, DirPickerItem{
				Path:        rp.Path,
				Name:        rp.Name,
				ProjectType: rp.ProjectType,
				IsRecent:    true,
			})
		}
	}

	// Add suggested directories from home
	d.addSuggestedDirectories()

	// Always add manual entry option at the end
	d.items = append(d.items, DirPickerItem{
		Path: "",
		Name: "Enter a path manually...",
	})

	// Set placeholder for text input
	d.textInput.SetPlaceholder("/path/to/project")
}

// addSuggestedDirectories adds common project locations.
func (d *DirPicker) addSuggestedDirectories() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Common project directories
	suggestions := []string{
		filepath.Join(home, "Projects"),
		filepath.Join(home, "Code"),
		filepath.Join(home, "Developer"),
		filepath.Join(home, "Development"),
		filepath.Join(home, "work"),
		filepath.Join(home, "src"),
		filepath.Join(home, "repos"),
		filepath.Join(home, "git"),
		filepath.Join(home, "Documents", "Code"),
		filepath.Join(home, "Documents", "Projects"),
	}

	existingPaths := make(map[string]bool)
	for _, item := range d.items {
		existingPaths[item.Path] = true
	}

	// Check for subdirectories in common locations
	for _, dir := range suggestions {
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			subPath := filepath.Join(dir, entry.Name())
			if existingPaths[subPath] {
				continue
			}
			if proj, err := d.detector.DetectProject(subPath); err == nil && proj != nil {
				d.items = append(d.items, DirPickerItem{
					Path:        proj.Path,
					Name:        proj.Name,
					ProjectType: proj.ProjectType,
				})
				existingPaths[subPath] = true
			}
		}
	}
}

// SetSize sets the component dimensions.
func (d *DirPicker) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.textInput.SetWidth(width - 4)
}

// Update handles input messages.
func (d *DirPicker) Update(msg tea.Msg) (*DirPicker, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if d.mode == DirPickerModeManual {
			return d.updateManual(msg)
		}
		return d.updateList(msg)
	}
	return d, nil
}

// updateList handles input in list mode.
func (d *DirPicker) updateList(msg tea.KeyMsg) (*DirPicker, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if d.selectedIdx > 0 {
			d.selectedIdx--
		}
	case "down", "j":
		if d.selectedIdx < len(d.items)-1 {
			d.selectedIdx++
		}
	case "enter":
		if d.selectedIdx >= 0 && d.selectedIdx < len(d.items) {
			selected := d.items[d.selectedIdx]
			// Check if this is the manual entry option
			if selected.Path == "" && selected.Name == "Enter a path manually..." {
				d.mode = DirPickerModeManual
				d.textInput.Focus()
				return d, nil
			}
			return d.selectDirectory(selected.Path)
		}
	case "esc", "q":
		return d, func() tea.Msg { return DirCanceledMsg{} }
	}
	return d, nil
}

// updateManual handles input in manual entry mode.
func (d *DirPicker) updateManual(msg tea.KeyMsg) (*DirPicker, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := d.textInput.Value()
		if path == "" {
			d.errorMsg = "Please enter a path"
			return d, nil
		}
		return d.selectDirectory(path)
	case "esc":
		d.mode = DirPickerModeList
		d.textInput.Blur()
		d.errorMsg = ""
		return d, nil
	default:
		// Forward to text input
		newInput, _ := d.textInput.Update(msg)
		d.textInput = newInput
	}
	return d, nil
}

// selectDirectory validates and selects a directory.
func (d *DirPicker) selectDirectory(path string) (*DirPicker, tea.Cmd) {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		d.errorMsg = "Invalid path: " + err.Error()
		return d, nil
	}

	info, err := os.Stat(absPath)
	if err != nil {
		d.errorMsg = "Path not found: " + absPath
		return d, nil
	}
	if !info.IsDir() {
		d.errorMsg = "Not a directory: " + absPath
		return d, nil
	}

	// Detect project info
	proj, _ := d.detector.DetectProject(absPath)
	if proj == nil {
		// Create minimal project info for non-project directories
		proj = &project.ProjectInfo{
			Path: absPath,
			Name: filepath.Base(absPath),
		}
	}

	return d, func() tea.Msg {
		return DirSelectedMsg{Path: absPath, Project: proj}
	}
}

// View renders the directory picker.
func (d *DirPicker) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Bold(true).
		Padding(0, 1)

	b.WriteString(titleStyle.Render("📁 Select Project Directory"))
	b.WriteString("\n\n")

	if d.mode == DirPickerModeManual {
		return d.renderManualMode(&b)
	}
	return d.renderListMode(&b)
}

// renderListMode renders the list selection mode.
func (d *DirPicker) renderListMode(b *strings.Builder) string {
	helpStyle := lipgloss.NewStyle().
		Foreground(styles.MutedLight).
		PaddingLeft(2)

	b.WriteString(helpStyle.Render("Use ↑↓ to navigate, Enter to select, Q to quit"))
	b.WriteString("\n\n")

	// Render items
	for i, item := range d.items {
		prefix := "  "
		if i == d.selectedIdx {
			prefix = "▶ "
		}

		var nameStyle lipgloss.Style
		if i == d.selectedIdx {
			nameStyle = lipgloss.NewStyle().
				Foreground(styles.Primary).
				Bold(true)
		} else {
			nameStyle = lipgloss.NewStyle().
				Foreground(styles.Foreground)
		}

		// Build item display
		name := item.Name
		if item.IsCurrent {
			name = "• " + name
		} else if item.IsRecent {
			name = "◷ " + name
		}

		b.WriteString(prefix)
		b.WriteString(nameStyle.Render(name))

		// Show project type badge
		if item.ProjectType != "" {
			badgeStyle := lipgloss.NewStyle().
				Foreground(styles.Muted).
				PaddingLeft(1)
			b.WriteString(badgeStyle.Render("[" + item.ProjectType + "]"))
		}
		b.WriteString("\n")

		// Show path for selected item (except manual entry)
		if i == d.selectedIdx && item.Path != "" {
			pathStyle := lipgloss.NewStyle().
				Foreground(styles.Muted).
				PaddingLeft(4)
			b.WriteString(pathStyle.Render(item.Path))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderManualMode renders the manual path entry mode.
func (d *DirPicker) renderManualMode(b *strings.Builder) string {
	helpStyle := lipgloss.NewStyle().
		Foreground(styles.MutedLight).
		PaddingLeft(2)

	b.WriteString(helpStyle.Render("Enter the path to your project directory:"))
	b.WriteString("\n\n")

	// Text input
	b.WriteString("  ")
	b.WriteString(d.textInput.View())
	b.WriteString("\n\n")

	// Error message
	if d.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(styles.Error).
			PaddingLeft(2)
		b.WriteString(errorStyle.Render("⚠ " + d.errorMsg))
		b.WriteString("\n\n")
	}

	// Help text
	b.WriteString(helpStyle.Render("Press Enter to confirm, Esc to go back"))
	b.WriteString("\n")

	return b.String()
}

// Mode returns the current mode of the directory picker.
func (d *DirPicker) Mode() DirPickerMode {
	return d.mode
}

// Items returns the list of items.
func (d *DirPicker) Items() []DirPickerItem {
	return d.items
}

// SelectedIndex returns the currently selected index.
func (d *DirPicker) SelectedIndex() int {
	return d.selectedIdx
}
