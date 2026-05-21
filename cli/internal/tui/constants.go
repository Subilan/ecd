package tui

import (
	"github.com/Subilan/ecd/internal/repl"
	"github.com/charmbracelet/lipgloss"
)

// AppState represents which screen is currently active.
type AppState int

const (
	StateSearch AppState = iota
	StateEntryDetail
	StateReview
	StateDeckStats
	StateHelp
	StateConfirmReset
	StateAI
	StateAIInit
)

var (
	// Base styles
	SourceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan

	// Shared styles (aliased from repl package)
	WordStyle  = repl.WordStyle
	LabelStyle = repl.LabelStyle
	DimStyle   = repl.DimStyle
	WarnStyle  = repl.WarnStyle
	TitleStyle = repl.TitleStyle
	PronStyle  = repl.PronStyle
	FooterStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(1, 0, 1, 0)
	StatusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Padding(0, 1)
	SelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("3"))

	// Bracket styles for word display
	BracketStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // gray
	FlashcardBracketStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	LeechBracketStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
)
