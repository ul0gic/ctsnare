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
