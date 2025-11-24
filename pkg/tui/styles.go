package tui

import (
	"github.com/gdamore/tcell/v2"
)

// Color scheme for the TUI
var (
	ColorBackground = tcell.ColorDefault
	ColorForeground = tcell.ColorDefault
	ColorBorder     = tcell.ColorBlue
	ColorTitle      = tcell.ColorWhite
	ColorHighlight  = tcell.ColorDarkCyan
	ColorSuccess    = tcell.ColorGreen
	ColorWarning    = tcell.ColorYellow
	ColorError      = tcell.ColorRed
	ColorInfo       = tcell.ColorBlue
	ColorMuted      = tcell.ColorGray
)

// Status colors
const (
	StatusHealthy   = "healthy"
	StatusDegraded  = "degraded"
	StatusUnhealthy = "unhealthy"
	StatusRunning   = "running"
	StatusStopped   = "stopped"
	StatusPending   = "pending"
)

// GetStatusColor returns the color for a given status
func GetStatusColor(status string) tcell.Color {
	switch status {
	case StatusHealthy, StatusRunning:
		return ColorSuccess
	case StatusDegraded, StatusPending:
		return ColorWarning
	case StatusUnhealthy, StatusStopped:
		return ColorError
	default:
		return ColorMuted
	}
}

// GetStatusIcon returns the icon for a given status
func GetStatusIcon(status string) string {
	switch status {
	case StatusHealthy, StatusRunning:
		return "✓"
	case StatusDegraded, StatusPending:
		return "⚠"
	case StatusUnhealthy, StatusStopped:
		return "✗"
	default:
		return "?"
	}
}
