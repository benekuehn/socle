package ui

import "github.com/charmbracelet/lipgloss"

// Colors holds predefined lipgloss styles for consistent UI elements.
var Colors = struct {
	SuccessStyle   lipgloss.Style
	FailureStyle   lipgloss.Style
	WarningStyle   lipgloss.Style
	InfoStyle      lipgloss.Style
	FaintStyle     lipgloss.Style
	UserInputStyle lipgloss.Style
}{
	SuccessStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true), // Bright Green
	FailureStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),  // Bright Red
	WarningStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true), // Bright Yellow
	InfoStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true), // Bright Blue
	FaintStyle:     lipgloss.NewStyle().Faint(true),
	UserInputStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("6")), // Cyan
}
