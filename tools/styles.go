package tools

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Color palette - modern dashboard aesthetic
var (
	// Primary colors
	colorPrimary   = lipgloss.Color("#7C3AED") // violet
	colorSecondary = lipgloss.Color("#06B6D4") // cyan
	colorAccent    = lipgloss.Color("#F59E0B") // amber

	// Semantic colors
	colorSuccess = lipgloss.Color("#10B981") // emerald
	colorWarning = lipgloss.Color("#F59E0B") // amber
	colorError   = lipgloss.Color("#EF4444") // red
	colorInfo    = lipgloss.Color("#3B82F6") // blue

	// Neutral colors
	colorMuted   = lipgloss.Color("#6B7280") // gray-500
	colorSubtle  = lipgloss.Color("#374151") // gray-700
	colorBorder  = lipgloss.Color("#4B5563") // gray-600
	colorText    = lipgloss.Color("#F9FAFB") // gray-50
	colorDimText = lipgloss.Color("#9CA3AF") // gray-400
)

// Styles
var (
	// Base text styles
	textStyle  = lipgloss.NewStyle().Foreground(colorText)
	dimStyle   = lipgloss.NewStyle().Foreground(colorDimText)
	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)

	// Badge styles (compact inline indicators)
	badgeBase = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true)

	successBadge = badgeBase.Background(colorSuccess).Foreground(lipgloss.Color("#000"))
	warningBadge = badgeBase.Background(colorWarning).Foreground(lipgloss.Color("#000"))
	errorBadge   = badgeBase.Background(colorError).Foreground(lipgloss.Color("#fff"))
	infoBadge    = badgeBase.Background(colorInfo).Foreground(lipgloss.Color("#fff"))
	toolBadge    = badgeBase.Background(colorPrimary).Foreground(lipgloss.Color("#fff"))
	planBadge    = badgeBase.Background(colorSecondary).Foreground(lipgloss.Color("#000"))

	// Role prefixes
	userStyle  = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	agentStyle = lipgloss.NewStyle().Foreground(colorSecondary).Bold(true)

	// Box styles for panels/sections
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	// Input prompt
	promptStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	// Separator
	separatorStyle = lipgloss.NewStyle().Foreground(colorSubtle)
)

// Status formats a status message with info badge
func Status(msg string) string {
	return infoBadge.Render(msg)
}

// Error formats an error message with error badge
func Error(msg string) string {
	return errorBadge.Render("error") + " " + dimStyle.Render(msg)
}

// Success formats a success message
func Success(msg string) string {
	return successBadge.Render("ok") + " " + dimStyle.Render(msg)
}

// Warning formats a warning message
func Warning(msg string) string {
	return warningBadge.Render("warn") + " " + dimStyle.Render(msg)
}

// Tool formats a tool header badge
func Tool(name string) string {
	return toolBadge.Render("tool") + " " + dimStyle.Render(name)
}

// Plan formats a plan mode indicator
func Plan(msg string) string {
	return planBadge.Render(msg)
}

// User formats user prefix
func User() string {
	return userStyle.Render("▸ You")
}

// Agent formats agent prefix
func Agent() string {
	return agentStyle.Render("▸ Agent")
}

// Prompt returns the input prompt string
func Prompt() string {
	return promptStyle.Render("› ")
}

// Separator returns a horizontal separator
func Separator() string {
	return separatorStyle.Render("─────────────────────────────────────────")
}

// Dim returns dimmed text
func Dim(s string) string {
	return dimStyle.Render(s)
}

// Muted returns muted text
func Muted(s string) string {
	return mutedStyle.Render(s)
}

// Box wraps content in a rounded border box
func Box(title, content string) string {
	titleLine := ""
	if title != "" {
		titleLine = infoBadge.Render(title) + "\n"
	}
	return boxStyle.Render(titleLine + content)
}

// Checkbox styles for todo items
func Checkbox(status string) string {
	switch status {
	case "completed":
		return successBadge.Render("✓")
	case "in_progress":
		return warningBadge.Render("▸")
	default:
		return mutedStyle.Render("○")
	}
}

// OptionNumber formats a selection number
func OptionNumber(n int) string {
	return dimStyle.Render(fmt.Sprintf("%d.", n))
}

// Highlight returns highlighted text
func Highlight(s string) string {
	return lipgloss.NewStyle().Foreground(colorAccent).Render(s)
}

// KeyValue formats a key-value pair for permission dialogs
func KeyValue(key, value string) string {
	return fmt.Sprintf("  %s %s", mutedStyle.Render(key+":"), textStyle.Render(value))
}

// DiffAdd formats added lines in diffs
func DiffAdd(s string) string {
	return lipgloss.NewStyle().Foreground(colorSuccess).Render("+ " + s)
}

// DiffRemove formats removed lines in diffs
func DiffRemove(s string) string {
	return lipgloss.NewStyle().Foreground(colorError).Render("- " + s)
}

// Thinking formats thinking/reasoning text (dimmed)
func Thinking(s string) string {
	return mutedStyle.Render(s)
}
