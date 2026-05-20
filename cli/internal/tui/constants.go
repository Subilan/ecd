package tui

import "github.com/charmbracelet/lipgloss"

// AppState represents which screen is currently active.
type AppState int

const (
	StateSearch AppState = iota
	StateEntryDetail
	StateReview
	StateDeckStats
	StateHelp
	StateConfirmReset
)

var (
	// Base styles
	SourceStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))  // cyan
	WordStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))  // yellow
	LabelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))  // green
	PronStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))  // purple
	DimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // dim
	WarnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))  // red
	TitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	FooterStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(1, 0, 1, 0)
	StatusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Padding(0, 1)
	SelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("3"))

	// Bracket styles for word display
	BracketStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // gray
	FlashcardBracketStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	LeechBracketStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
)
