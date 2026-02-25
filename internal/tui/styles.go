package tui

import "github.com/charmbracelet/lipgloss"

// Adaptive colors for light/dark terminal support.
var (
	colorHighSeverity = lipgloss.AdaptiveColor{Light: "#FF0000", Dark: "#FF4444"}
	colorMedSeverity  = lipgloss.AdaptiveColor{Light: "#CC8800", Dark: "#FFAA00"}
	colorLowSeverity  = lipgloss.AdaptiveColor{Light: "#008800", Dark: "#44CC44"}
	colorMuted        = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}
	colorSubtle       = lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#555555"}
	colorText         = lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#FAFAFA"}
	colorStatusBg     = lipgloss.AdaptiveColor{Light: "#DDDDDD", Dark: "#333333"}
	colorStatusFg     = lipgloss.AdaptiveColor{Light: "#333333", Dark: "#DDDDDD"}
	colorLive         = lipgloss.AdaptiveColor{Light: "#008800", Dark: "#22DD22"}
	colorDiscarded    = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#555555"}
	colorBookmark     = lipgloss.AdaptiveColor{Light: "#CC8800", Dark: "#FFD700"}
)

// StyleHighSeverity renders high-severity items in bold red.
var StyleHighSeverity = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorHighSeverity)

// StyleMedSeverity renders medium-severity items in bold yellow.
var StyleMedSeverity = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorMedSeverity)

// StyleLowSeverity renders low-severity items in green.
var StyleLowSeverity = lipgloss.NewStyle().
	Foreground(colorLowSeverity)

// StyleHeader renders the header bar with bold text and a bottom border.
var StyleHeader = lipgloss.NewStyle().
	Bold(true).
	BorderStyle(lipgloss.NormalBorder()).
	BorderBottom(true).
	BorderForeground(colorSubtle)

// StyleStatusBar renders the bottom status bar with a dark background.
var StyleStatusBar = lipgloss.NewStyle().
	Background(colorStatusBg).
	Foreground(colorStatusFg).
	Padding(0, 1)

// StyleSelectedRow renders the currently selected row with inverted colors.
var StyleSelectedRow = lipgloss.NewStyle().
	Bold(true).
	Background(colorText).
	Foreground(colorStatusBg)

// StyleHelpKey renders keybinding keys in bold muted text.
var StyleHelpKey = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorMuted)

// StyleHelpDesc renders keybinding descriptions in muted text.
var StyleHelpDesc = lipgloss.NewStyle().
	Foreground(colorMuted)

// StyleTitle renders section titles with bold text and horizontal padding.
var StyleTitle = lipgloss.NewStyle().
	Bold(true).
	Padding(0, 1)

// StyleBorder renders a container with a rounded border.
var StyleBorder = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorSubtle)

// StyleLiveDomain renders live domains in bold green to indicate successful liveness probe.
var StyleLiveDomain = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorLive)

// StyleDiscardedDomain renders discarded (zero-score) domains in dim gray.
var StyleDiscardedDomain = lipgloss.NewStyle().
	Foreground(colorDiscarded)

// StyleBookmarked renders the bookmark indicator in gold/yellow.
var StyleBookmarked = lipgloss.NewStyle().
	Foreground(colorBookmark)

// StyleSelectedCheckbox renders the multi-select checkbox indicator for explorer rows.
var StyleSelectedCheckbox = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorLive)

// --- Option B panel styles ---

// colorAppName is used for the "ctsnare" brand name in the tab bar.
var colorAppName = lipgloss.AdaptiveColor{Light: "#008888", Dark: "#00AAAA"}

// StyleAppName renders the application name in cyan bold.
var StyleAppName = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorAppName)

// StyleTabActive renders the active tab label with inverted colors.
var StyleTabActive = lipgloss.NewStyle().
	Bold(true).
	Background(colorStatusBg).
	Foreground(colorText).
	Padding(0, 1)

// StyleTabInactive renders inactive tab labels in muted text.
var StyleTabInactive = lipgloss.NewStyle().
	Foreground(colorMuted).
	Padding(0, 1)

// StylePanel renders a titled panel with a rounded border in subtle gray.
// Use .Width() to set the panel width before rendering.
var StylePanel = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorSubtle)

// StyleDottedSep renders a dotted separator line for detail view sections.
var StyleDottedSep = lipgloss.NewStyle().
	Foreground(colorSubtle)

// StyleConfirmOverlay renders the confirmation prompt with a red background.
var StyleConfirmOverlay = lipgloss.NewStyle().
	Bold(true).
	Background(colorHighSeverity).
	Foreground(lipgloss.Color("#FFFFFF")).
	Padding(0, 1)

// SeverityStyle returns the appropriate style for the given severity level.
func SeverityStyle(severity string) lipgloss.Style {
	switch severity {
	case "HIGH":
		return StyleHighSeverity
	case "MED":
		return StyleMedSeverity
	case "LOW":
		return StyleLowSeverity
	default:
		return lipgloss.NewStyle()
	}
}
