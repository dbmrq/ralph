// Package components provides reusable TUI components for ralph.
package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wexinc/ralph/internal/task"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// TaskListItem represents a task item in the list.
type TaskListItem struct {
	ID          string
	Name        string
	Description string
	Status      task.TaskStatus
	Iteration   int
	IsCurrent   bool
}

// TaskList is a scrollable list of tasks with status icons.
type TaskList struct {
	items       []TaskListItem
	selected    int
	height      int
	width       int
	scrollStart int
	focused     bool
}

// NewTaskList creates a new TaskList component.
func NewTaskList() *TaskList {
	return &TaskList{
		items:       []TaskListItem{},
		selected:    0,
		height:      10,
		scrollStart: 0,
		focused:     true,
	}
}

// SetItems updates the task list items.
func (t *TaskList) SetItems(items []TaskListItem) {
	t.items = items
	if t.selected >= len(items) {
		t.selected = len(items) - 1
	}
	if t.selected < 0 {
		t.selected = 0
	}
	t.updateScroll()
}

// SetTasks converts task.Task slice to TaskListItems.
func (t *TaskList) SetTasks(tasks []*task.Task, currentTaskID string) {
	items := make([]TaskListItem, len(tasks))
	for i, tsk := range tasks {
		iter := 0
		if tsk.CurrentIteration() != nil {
			iter = tsk.CurrentIteration().Number
		}
		items[i] = TaskListItem{
			ID:          tsk.ID,
			Name:        tsk.Name,
			Description: tsk.Description,
			Status:      tsk.Status,
			Iteration:   iter,
			IsCurrent:   tsk.ID == currentTaskID,
		}
	}
	t.SetItems(items)
}

// SetHeight sets the visible height of the list.
func (t *TaskList) SetHeight(height int) {
	t.height = height
	t.updateScroll()
}

// SetWidth sets the width of the list.
func (t *TaskList) SetWidth(width int) {
	t.width = width
}

// SetFocused sets whether the list is focused.
func (t *TaskList) SetFocused(focused bool) {
	t.focused = focused
}

// Selected returns the currently selected item index.
func (t *TaskList) Selected() int {
	return t.selected
}

// SelectedItem returns the currently selected item, or nil if empty.
func (t *TaskList) SelectedItem() *TaskListItem {
	if len(t.items) == 0 || t.selected < 0 || t.selected >= len(t.items) {
		return nil
	}
	return &t.items[t.selected]
}

// MoveUp moves selection up.
func (t *TaskList) MoveUp() {
	if t.selected > 0 {
		t.selected--
		t.updateScroll()
	}
}

// MoveDown moves selection down.
func (t *TaskList) MoveDown() {
	if t.selected < len(t.items)-1 {
		t.selected++
		t.updateScroll()
	}
}

// GoToTop moves selection to the first item.
func (t *TaskList) GoToTop() {
	t.selected = 0
	t.updateScroll()
}

// GoToBottom moves selection to the last item.
func (t *TaskList) GoToBottom() {
	if len(t.items) > 0 {
		t.selected = len(t.items) - 1
		t.updateScroll()
	}
}

// SetSize sets both width and height.
func (t *TaskList) SetSize(width, height int) {
	t.width = width
	t.height = height
	t.updateScroll()
}

// SetSelected sets the selected index.
func (t *TaskList) SetSelected(index int) {
	if index >= 0 && index < len(t.items) {
		t.selected = index
		t.updateScroll()
	}
}

// updateScroll ensures the selected item is visible.
func (t *TaskList) updateScroll() {
	if t.selected < t.scrollStart {
		t.scrollStart = t.selected
	}
	if t.selected >= t.scrollStart+t.height {
		t.scrollStart = t.selected - t.height + 1
	}
	if t.scrollStart < 0 {
		t.scrollStart = 0
	}
}

// Update handles keyboard events for navigation.
func (t *TaskList) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			t.MoveUp()
		case "down", "j":
			t.MoveDown()
		case "home", "g":
			t.selected = 0
			t.updateScroll()
		case "end", "G":
			t.selected = len(t.items) - 1
			t.updateScroll()
		}
	}
	return nil
}

// View renders the task list.
func (t *TaskList) View() string {
	if len(t.items) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(styles.Muted).
			Italic(true).
			Padding(1, 2)
		return emptyStyle.Render("No tasks")
	}

	var lines []string
	endIndex := t.scrollStart + t.height
	if endIndex > len(t.items) {
		endIndex = len(t.items)
	}

	for i := t.scrollStart; i < endIndex; i++ {
		item := t.items[i]
		line := t.renderItem(item, i == t.selected)
		lines = append(lines, line)
	}

	// Add scroll indicators if needed
	content := strings.Join(lines, "\n")

	if t.scrollStart > 0 {
		content = "  ↑ more above\n" + content
	}
	if endIndex < len(t.items) {
		content = content + "\n  ↓ more below"
	}

	return content
}

// renderItem renders a single task item.
func (t *TaskList) renderItem(item TaskListItem, isSelected bool) string {
	// Status icon
	icon := t.statusIcon(item.Status)

	// Task ID and name
	idStyle := lipgloss.NewStyle().
		Foreground(styles.MutedLight).
		Width(12)

	nameStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground)

	// Current task indicator
	currentIndicator := " "
	if item.IsCurrent {
		currentIndicator = lipgloss.NewStyle().
			Foreground(styles.Secondary).
			Bold(true).
			Render("▶")
	}

	// Selection styling
	lineStyle := lipgloss.NewStyle()
	if isSelected && t.focused {
		lineStyle = lineStyle.
			Background(styles.Background).
			Bold(true)
	}

	// Build the line
	id := idStyle.Render(truncateString(item.ID, 11))
	name := nameStyle.Render(item.Name)

	// Add iteration number if applicable
	iterStr := ""
	if item.Iteration > 0 && (item.Status == task.StatusInProgress || item.Status == task.StatusPaused) {
		iterStyle := lipgloss.NewStyle().
			Foreground(styles.MutedLight)
		iterStr = iterStyle.Render(fmt.Sprintf(" (iter %d)", item.Iteration))
	}

	line := fmt.Sprintf("%s %s %s %s%s", currentIndicator, icon, id, name, iterStr)

	// Apply width constraint if set
	if t.width > 0 {
		lineStyle = lineStyle.Width(t.width)
	}

	return lineStyle.Render(line)
}

// statusIcon returns the appropriate icon for a task status.
func (t *TaskList) statusIcon(status task.TaskStatus) string {
	switch status {
	case task.StatusCompleted:
		return styles.StatusCompleted
	case task.StatusInProgress:
		return styles.StatusInProgress
	case task.StatusPending:
		return styles.StatusPending
	case task.StatusSkipped:
		return styles.StatusSkipped
	case task.StatusPaused:
		return styles.StatusPaused
	case task.StatusFailed:
		return styles.StatusFailed
	default:
		return styles.StatusPending
	}
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}
