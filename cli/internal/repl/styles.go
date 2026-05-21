package repl

import "github.com/charmbracelet/lipgloss"

var (
	LabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	WordStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	DimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	WarnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	PronStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
)
