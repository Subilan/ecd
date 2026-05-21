package ai

import (
	"regexp"

	"github.com/Subilan/ecd/internal/config"
	"github.com/Subilan/ecd/internal/i18n"
	"github.com/Subilan/ecd/internal/repl"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

var alphaRE = regexp.MustCompile(`^[a-zA-Z]+$`)

var (
	aiLabelStyle = repl.LabelStyle
	aiWordStyle  = repl.WordStyle
	aiDimStyle   = repl.DimStyle
	aiErrStyle   = repl.WarnStyle
	aiWarnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	aiTitleStyle = repl.TitleStyle
	aiPronStyle  = repl.PronStyle
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ---- Messages ----

// AIResultMsg is sent when an AI API call completes.
type AIResultMsg struct {
	Response   string
	Cached     bool
	Err        error
	CmdDisplay string
	Kind       string
}

type aiTickMsg struct{}

// ---- Model ----

// AIModel is the Bubble Tea sub-model for AI mode.
type AIModel struct {
	repl.Base
	cfg *config.AIConfig

	waiting    bool
	spinnerIdx int
	result     string
	focusInput bool // true = input receives keys, false = viewport scrolls
}

// NewAIModel creates a new AI mode sub-model.
func NewAIModel(cfg *config.AIConfig) AIModel {
	ti := textinput.New()
	ti.Placeholder = i18n.T("ai.placeholder")
	ti.CharLimit = 200
	ti.Width = 50
	ti.Focus()

	vp := viewport.New(80, 20)
	// Strip single-char bindings so they don't conflict with typing.
	vp.KeyMap = viewport.KeyMap{
		PageDown:     key.NewBinding(key.WithKeys("pgdown")),
		PageUp:       key.NewBinding(key.WithKeys("pgup")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u")),
	}

	return AIModel{
		Base:       repl.NewBase(ti, vp),
		cfg:        cfg,
		focusInput: true,
	}
}

// SetConfig updates the AI config reference.
func (m *AIModel) SetConfig(cfg *config.AIConfig) {
	m.cfg = cfg
}

// ---- Transition messages ----

// TransitionToSearchMsg signals the parent model to switch back to search.
type TransitionToSearchMsg struct{}

// TransitionToAIMsg signals the parent model to switch back to AI mode.
type TransitionToAIMsg struct{}

// EnterInitMsg signals the parent model to enter init config mode.
type EnterInitMsg struct{}

// ShowHelpMsg signals the parent model to show the help screen in AI mode.
type ShowHelpMsg struct{}
