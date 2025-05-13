package ui

import "github.com/charmbracelet/lipgloss"

// Colors holds predefined lipgloss styles for consistent UI elements.
var Colors = struct {
	SuccessStyle    lipgloss.Style
	FailureStyle    lipgloss.Style
	WarningStyle    lipgloss.Style
	InfoStyle       lipgloss.Style
	FaintStyle      lipgloss.Style
	UserInputStyle  lipgloss.Style
	DotStyle        lipgloss.Style
	DotFilledStyle  lipgloss.Style
	DotWarningStyle lipgloss.Style
	MutedStyle      lipgloss.Style
}{
	SuccessStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#16a34a")).Bold(true),
	FailureStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#dc2626")).Bold(true),
	WarningStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#d97706")).Bold(true),
	InfoStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("#fafafa")).Bold(true),
	FaintStyle:      lipgloss.NewStyle().Faint(true),
	UserInputStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#2563eb")),
	DotStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("#fafafa")),
	DotFilledStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#16a34a")),
	DotWarningStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#d97706")),
	MutedStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#a1a1aa")),
}
