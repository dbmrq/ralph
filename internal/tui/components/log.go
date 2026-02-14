// Package components provides reusable TUI components for ralph.
package components

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wexinc/ralph/internal/tui/styles"
)

// LogViewport is a scrollable log viewer with auto-follow support.
// Optimized for append-only log streaming with minimal memory allocation.
type LogViewport struct {
	viewport   viewport.Model
	lines      []string
	autoFollow bool
	focused    bool
	title      string
	width      int
	height     int
	// contentDirty tracks if we need to rebuild viewport content.
	contentDirty bool
	// cachedContent is the cached joined lines for the viewport.
	cachedContent string
	// lastLineComplete tracks if the last line ended with a newline.
	// When false, the next AppendText should append to the last line.
	lastLineComplete bool
}

// NewLogViewport creates a new LogViewport component.
func NewLogViewport() *LogViewport {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.BorderColor)

	return &LogViewport{
		viewport:         vp,
		lines:            make([]string, 0, 1024), // Pre-allocate for typical log size
		autoFollow:       true,
		focused:          false,
		title:            "Log Output",
		width:            80,
		height:           20,
		contentDirty:     false,
		lastLineComplete: true, // Start with no partial line
	}
}

// SetTitle sets the viewport title.
func (l *LogViewport) SetTitle(title string) {
	l.title = title
}

// SetSize sets the viewport dimensions.
func (l *LogViewport) SetSize(width, height int) {
	l.width = width
	l.height = height
	// Account for border and title
	l.viewport.Width = width - 2
	l.viewport.Height = height - 2
}

// SetFocused sets whether the viewport is focused.
func (l *LogViewport) SetFocused(focused bool) {
	l.focused = focused
	if focused {
		l.viewport.Style = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Primary)
	} else {
		l.viewport.Style = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.BorderColor)
	}
}

// SetAutoFollow enables or disables auto-follow mode.
func (l *LogViewport) SetAutoFollow(enabled bool) {
	l.autoFollow = enabled
}

// AutoFollow returns the current auto-follow state.
func (l *LogViewport) AutoFollow() bool {
	return l.autoFollow
}

// Clear clears all log content.
func (l *LogViewport) Clear() {
	l.lines = l.lines[:0] // Reuse backing array
	l.cachedContent = ""
	l.contentDirty = false
	l.lastLineComplete = true
	l.viewport.SetContent("")
}

// Write implements io.Writer for direct streaming.
func (l *LogViewport) Write(p []byte) (n int, err error) {
	l.AppendText(string(p))
	return len(p), nil
}

// AppendText appends text to the log.
// It parses newlines and appends individual lines efficiently.
// Partial lines (text without trailing newline) are appended to the last line.
// The viewport content is lazily rebuilt on the next View() or explicit refresh.
func (l *LogViewport) AppendText(text string) {
	if text == "" {
		return
	}

	// Check if text ends with newline
	endsWithNewline := strings.HasSuffix(text, "\n")

	// Split into lines
	newLines := strings.Split(text, "\n")

	for i, line := range newLines {
		// Skip trailing empty string from strings.Split on trailing newline
		if i == len(newLines)-1 && line == "" {
			continue
		}

		if i == 0 && !l.lastLineComplete && len(l.lines) > 0 {
			// Append first part to the last existing line (handles partial lines)
			l.lines[len(l.lines)-1] += line
		} else {
			l.lines = append(l.lines, line)
		}
	}

	l.lastLineComplete = endsWithNewline
	l.contentDirty = true
}

// AppendLine appends a line to the log.
// This is optimized for the common case of adding single lines.
// The viewport content is lazily rebuilt on the next View() or explicit refresh.
func (l *LogViewport) AppendLine(line string) {
	l.lines = append(l.lines, line)
	l.contentDirty = true
}

// ensureViewportContent rebuilds the viewport content if dirty.
// This is called lazily before rendering.
func (l *LogViewport) ensureViewportContent() {
	if l.contentDirty {
		l.cachedContent = strings.Join(l.lines, "\n")
		l.viewport.SetContent(l.cachedContent)
		l.contentDirty = false
		if l.autoFollow {
			l.viewport.GotoBottom()
		}
	}
}

// SetContent sets the entire log content.
func (l *LogViewport) SetContent(content string) {
	l.lines = strings.Split(content, "\n")
	l.cachedContent = content
	l.contentDirty = false
	l.lastLineComplete = strings.HasSuffix(content, "\n") || content == ""
	l.viewport.SetContent(content)

	if l.autoFollow {
		l.viewport.GotoBottom()
	}
}

// Content returns the full log content.
func (l *LogViewport) Content() string {
	if l.contentDirty {
		l.cachedContent = strings.Join(l.lines, "\n")
		l.contentDirty = false
	}
	return l.cachedContent
}

// LineCount returns the number of lines.
func (l *LogViewport) LineCount() int {
	return len(l.lines)
}

// ScrollPercent returns the scroll position as a percentage.
func (l *LogViewport) ScrollPercent() float64 {
	return l.viewport.ScrollPercent()
}

// GotoTop scrolls to the top.
func (l *LogViewport) GotoTop() {
	l.viewport.GotoTop()
	l.autoFollow = false
}

// GoToTop is an alias for GotoTop for consistency.
func (l *LogViewport) GoToTop() {
	l.GotoTop()
}

// GotoBottom scrolls to the bottom.
func (l *LogViewport) GotoBottom() {
	l.viewport.GotoBottom()
	l.autoFollow = true
}

// GoToBottom is an alias for GotoBottom for consistency.
func (l *LogViewport) GoToBottom() {
	l.GotoBottom()
}

// ScrollUp scrolls up one line.
func (l *LogViewport) ScrollUp() {
	l.viewport.LineUp(1)
	l.autoFollow = false
}

// ScrollDown scrolls down one line.
func (l *LogViewport) ScrollDown() {
	l.viewport.LineDown(1)
}

// ToggleAutoFollow toggles auto-follow mode.
func (l *LogViewport) ToggleAutoFollow() {
	l.autoFollow = !l.autoFollow
	if l.autoFollow {
		l.viewport.GotoBottom()
	}
}

// Update handles keyboard events for scrolling.
func (l *LogViewport) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			l.autoFollow = false
			l.viewport.LineUp(1)
		case "down", "j":
			l.viewport.LineDown(1)
			// Re-enable auto-follow if at bottom
			if l.viewport.AtBottom() {
				l.autoFollow = true
			}
		case "pgup", "ctrl+u":
			l.autoFollow = false
			l.viewport.HalfViewUp()
		case "pgdown", "ctrl+d":
			l.viewport.HalfViewDown()
			if l.viewport.AtBottom() {
				l.autoFollow = true
			}
		case "home", "g":
			l.GotoTop()
		case "end", "G":
			l.GotoBottom()
		case "f":
			// Toggle auto-follow
			l.autoFollow = !l.autoFollow
			if l.autoFollow {
				l.viewport.GotoBottom()
			}
		default:
			l.viewport, cmd = l.viewport.Update(msg)
		}
	default:
		l.viewport, cmd = l.viewport.Update(msg)
	}

	return cmd
}

// View renders the log viewport.
func (l *LogViewport) View() string {
	// Ensure content is up-to-date before rendering
	l.ensureViewportContent()

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Foreground).
		Background(styles.Primary).
		Bold(true).
		Padding(0, 1).
		Width(l.width)

	title := l.title
	if l.autoFollow {
		title += " [auto-follow]"
	}

	// Scroll position indicator
	scrollInfo := lipgloss.NewStyle().
		Foreground(styles.MutedLight).
		Render(fmt.Sprintf(" %.0f%%", l.viewport.ScrollPercent()*100))

	titleLine := titleStyle.Render(title) + scrollInfo

	// Viewport content
	viewportStyle := lipgloss.NewStyle()
	if l.focused {
		viewportStyle = viewportStyle.
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Primary)
	} else {
		viewportStyle = viewportStyle.
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.BorderColor)
	}

	content := viewportStyle.Render(l.viewport.View())

	// Help line
	helpStyle := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Italic(true)
	help := helpStyle.Render("j/k: scroll  f: toggle follow  e: open in $EDITOR")

	return titleLine + "\n" + content + "\n" + help
}

// OpenInEditor opens the log content in the user's $EDITOR.
// Returns a tea.Cmd that executes the editor.
func (l *LogViewport) OpenInEditor() tea.Cmd {
	return func() tea.Msg {
		// Get editor from environment
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi" // Default fallback
		}

		// Create temp file
		tmpFile, err := os.CreateTemp("", "ralph-log-*.txt")
		if err != nil {
			return EditorErrorMsg{Error: err}
		}

		// Write content
		_, err = tmpFile.WriteString(l.Content())
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return EditorErrorMsg{Error: err}
		}
		tmpFile.Close()

		// Open editor
		cmd := exec.Command(editor, tmpFile.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Run()

		// Clean up temp file
		os.Remove(tmpFile.Name())

		if err != nil {
			return EditorErrorMsg{Error: err}
		}

		return EditorClosedMsg{}
	}
}

// EditorErrorMsg is sent when opening the editor fails.
type EditorErrorMsg struct {
	Error error
}

// EditorClosedMsg is sent when the editor is closed.
type EditorClosedMsg struct{}

