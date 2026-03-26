package tui

import "github.com/charmbracelet/lipgloss"

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)
	
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)

	statusOKStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusErrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63"))

	activeBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205"))

	inactiveBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("237"))

	searchInputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	keyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("251"))
	separatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("237")).Padding(0, 1)
)
