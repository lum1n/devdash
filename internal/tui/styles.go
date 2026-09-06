package tui

import "charm.land/lipgloss/v2"

var (
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	selStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("236"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	tabOn      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("240")).Padding(0, 1)
	tabOff     = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Padding(0, 1)
	metricBig  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
)
