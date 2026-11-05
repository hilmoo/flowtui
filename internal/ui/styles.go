package ui

import "charm.land/lipgloss/v2"

var (
	dim         = lipgloss.NewStyle().Foreground(gray)
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	selStyle    = lipgloss.NewStyle().Bold(true).Reverse(true)
	selDimStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	titleStyle  = lipgloss.NewStyle().Bold(true)
	errStyle    = lipgloss.NewStyle().Foreground(red).Bold(true)
	warnStyle   = lipgloss.NewStyle().Foreground(yellow)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
)
