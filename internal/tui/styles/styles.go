// Package styles provides Lip Gloss styles for the Ralph TUI.
package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette for the TUI.
var (
	// Primary colors
	Primary     = lipgloss.Color("#7C3AED") // Purple
	Secondary   = lipgloss.Color("#06B6D4") // Cyan
	Success     = lipgloss.Color("#10B981") // Green
	Warning     = lipgloss.Color("#F59E0B") // Amber
	Error       = lipgloss.Color("#EF4444") // Red
	Muted       = lipgloss.Color("#6B7280") // Gray
	MutedLight  = lipgloss.Color("#9CA3AF") // Light Gray
	Background  = lipgloss.Color("#1F2937") // Dark Gray
	Foreground  = lipgloss.Color("#F9FAFB") // White
	BorderColor = lipgloss.Color("#374151") // Border Gray
)

// Header styles.
var (
	// HeaderStyle is the main header container.
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Foreground).
			Background(Primary).
			Padding(0, 1)

	// HeaderLabelStyle is for header labels.
	HeaderLabelStyle = lipgloss.NewStyle().
				Foreground(MutedLight)

	// HeaderValueStyle is for header values.
	HeaderValueStyle = lipgloss.NewStyle().
				Foreground(Foreground).
				Bold(true)

	// TitleStyle is for the application title.
	TitleStyle = lipgloss.NewStyle().
			Foreground(Foreground).
			Background(Primary).
			Bold(true).
			Padding(0, 1)
)

// Progress bar styles.
var (
	// ProgressBarStyle is the progress bar container.
	ProgressBarStyle = lipgloss.NewStyle().
				Padding(0, 1)

	// ProgressFilledStyle is for the filled portion.
	ProgressFilledStyle = lipgloss.NewStyle().
				Foreground(Success).
				Bold(true)

	// ProgressEmptyStyle is for the empty portion.
	ProgressEmptyStyle = lipgloss.NewStyle().
				Foreground(Muted)

	// ProgressCountStyle is for the task count display.
	ProgressCountStyle = lipgloss.NewStyle().
				Foreground(Secondary)
)

// Task status styles and icons.
var (
	// StatusCompleted is the completed task style/icon.
	StatusCompleted = lipgloss.NewStyle().
			Foreground(Success).
			Render("✓")

	// StatusPending is the pending task style/icon.
	StatusPending = lipgloss.NewStyle().
			Foreground(Muted).
			Render("○")

	// StatusInProgress is the in-progress task style/icon.
	StatusInProgress = lipgloss.NewStyle().
				Foreground(Secondary).
				Render("→")

	// StatusSkipped is the skipped task style/icon.
	StatusSkipped = lipgloss.NewStyle().
			Foreground(Warning).
			Render("⊘")

	// StatusPaused is the paused task style/icon.
	StatusPaused = lipgloss.NewStyle().
			Foreground(Warning).
			Render("⏸")

	// StatusFailed is the failed task style/icon.
	StatusFailed = lipgloss.NewStyle().
			Foreground(Error).
			Render("✗")
)

// Box styles.
var (
	// BoxStyle is a standard box with border.
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(0, 1)

	// FocusedBoxStyle is a box that's currently focused.
	FocusedBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(0, 1)
)

// Text styles.
var (
	// MutedTextStyle is for de-emphasized text.
	MutedTextStyle = lipgloss.NewStyle().
			Foreground(Muted)

	// ErrorTextStyle is for error messages.
	ErrorTextStyle = lipgloss.NewStyle().
			Foreground(Error)

	// SuccessTextStyle is for success messages.
	SuccessTextStyle = lipgloss.NewStyle().
				Foreground(Success)

	// WarningTextStyle is for warning messages.
	WarningTextStyle = lipgloss.NewStyle().
				Foreground(Warning)
)

// Status bar styles.
var (
	// StatusBarStyle is the main status bar container.
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(MutedLight).
			Padding(0, 1)

	// KeyStyle is for keyboard shortcut keys.
	KeyStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Bold(true)

	// HelpStyle is for help text.
	HelpStyle = lipgloss.NewStyle().
			Foreground(Muted)
)

// Form component styles.
var (
	// FormTitleStyle is for form titles.
	FormTitleStyle = lipgloss.NewStyle().
			Foreground(Foreground).
			Bold(true).
			Padding(0, 1)

	// FormLabelStyle is for form field labels.
	FormLabelStyle = lipgloss.NewStyle().
			Foreground(MutedLight)

	// FormLabelFocusedStyle is for focused form field labels.
	FormLabelFocusedStyle = lipgloss.NewStyle().
				Foreground(Secondary).
				Bold(true)

	// FormInputStyle is for form text inputs (unfocused).
	FormInputStyle = lipgloss.NewStyle().
			Foreground(MutedLight).
			Padding(0, 1)

	// FormInputFocusedStyle is for focused form text inputs.
	FormInputFocusedStyle = lipgloss.NewStyle().
				Foreground(Foreground).
				Background(Background).
				Padding(0, 1)

	// CheckboxCheckedStyle is for checked checkboxes.
	CheckboxCheckedStyle = lipgloss.NewStyle().
				Foreground(Success)

	// CheckboxUncheckedStyle is for unchecked checkboxes.
	CheckboxUncheckedStyle = lipgloss.NewStyle().
				Foreground(Muted)

	// ButtonPrimaryStyle is for primary buttons (focused).
	ButtonPrimaryStyle = lipgloss.NewStyle().
				Foreground(Background).
				Background(Primary).
				Bold(true).
				Padding(0, 2)

	// ButtonPrimaryUnfocusedStyle is for primary buttons (unfocused).
	ButtonPrimaryUnfocusedStyle = lipgloss.NewStyle().
					Foreground(Primary).
					Border(lipgloss.NormalBorder()).
					BorderForeground(Primary).
					Padding(0, 1)

	// ButtonSecondaryStyle is for secondary buttons (focused).
	ButtonSecondaryStyle = lipgloss.NewStyle().
				Foreground(Background).
				Background(Secondary).
				Bold(true).
				Padding(0, 2)

	// ButtonSecondaryUnfocusedStyle is for secondary buttons (unfocused).
	ButtonSecondaryUnfocusedStyle = lipgloss.NewStyle().
					Foreground(MutedLight).
					Border(lipgloss.NormalBorder()).
					BorderForeground(Muted).
					Padding(0, 1)

	// ButtonDangerStyle is for danger buttons (focused).
	ButtonDangerStyle = lipgloss.NewStyle().
				Foreground(Foreground).
				Background(Error).
				Bold(true).
				Padding(0, 2)

	// ButtonDangerUnfocusedStyle is for danger buttons (unfocused).
	ButtonDangerUnfocusedStyle = lipgloss.NewStyle().
					Foreground(Error).
					Border(lipgloss.NormalBorder()).
					BorderForeground(Error).
					Padding(0, 1)
)
