package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Subilan/ecd/internal/config"
	"github.com/Subilan/ecd/internal/i18n"
	"github.com/Subilan/ecd/internal/render"
	"github.com/Subilan/ecd/internal/repl"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
}

// NewAIModel creates a new AI mode sub-model.
func NewAIModel(cfg *config.AIConfig) AIModel {
	ti := textinput.New()
	ti.Placeholder = i18n.T("ai.placeholder")
	ti.CharLimit = 200
	ti.Width = 50
	ti.Focus()

	vp := viewport.New(80, 20)

	return AIModel{
		Base: repl.NewBase(ti, vp),
		cfg:  cfg,
	}
}

// SetConfig updates the AI config reference.
func (m *AIModel) SetConfig(cfg *config.AIConfig) {
	m.cfg = cfg
}

// Update implements tea.Model.
func (m AIModel) Update(msg tea.Msg) (AIModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Base.HandleWindowSizeMsg(msg)
		if m.result != "" {
			m.Viewport.SetContent(render.WrapContent(m.result, m.Viewport.Width))
		}

	case repl.ClearStatusMsg:
		m.Base.HandleClearStatusMsg(msg)
		return m, nil

	case aiTickMsg:
		if m.waiting {
			m.spinnerIdx = (m.spinnerIdx + 1) % len(spinnerFrames)
			return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
				return aiTickMsg{}
			})
		}

	case AIResultMsg:
		m.waiting = false
		if msg.Err != nil {
			m.result = aiErrStyle.Render(msg.Err.Error())
		} else {
			m.result = formatAIResponse(msg)
		}
		m.Viewport.SetContent(render.WrapContent(m.result, m.Viewport.Width))
		m.Viewport.GotoTop()

	case tea.KeyMsg:
		if m.waiting {
			return m, nil
		}

		switch msg.String() {
		case "enter":
			query := strings.TrimSpace(m.Input.Value())
			if query == "" {
				return m, nil
			}
			m.Input.SetValue("")
			return m, m.executeCommand(query)

		case "esc":
			m.Input.SetValue("")
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m AIModel) View() string {
	if !m.Ready {
		return i18n.T("common.initializing")
	}

	var b strings.Builder

	b.WriteString(m.Input.View())
	b.WriteString("\n")

	if m.waiting {
		b.WriteString(aiDimStyle.Render(
			fmt.Sprintf(" %s %s\n", spinnerFrames[m.spinnerIdx], i18n.T("ai.waiting"))))
	} else if m.result != "" {
		b.WriteString(m.Viewport.View())
	}

	return lipgloss.JoinVertical(lipgloss.Left, b.String(), m.renderFooter())
}

func (m AIModel) renderFooter() string {
	var b strings.Builder
	b.WriteString(aiDimStyle.Render(i18n.T("ai.footer")))

	if m.cfg.IsConfigured() && m.cfg.CacheEnabled {
		b.WriteString("  " + aiDimStyle.Render("[cache]"))
	}

	if !m.cfg.IsConfigured() {
		b.WriteString("  " + aiErrStyle.Render(i18n.T("ai.footer_no_config")))
	}

	if s := m.StatusMsg(); s != "" {
		b.WriteString("\n\n" + aiLabelStyle.Render(s))
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(1, 0, 1, 0).Render(b.String())
}

// ---- Command execution ----

func (m *AIModel) executeCommand(input string) tea.Cmd {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	cmdRaw := strings.ToLower(parts[0])
	if !strings.HasPrefix(cmdRaw, "/") {
		return m.SetStatus(i18n.T("ai.not_a_command"))
	}

	hasBang := strings.HasSuffix(cmdRaw, "!")
	cmdName := strings.TrimSuffix(cmdRaw, "!")
	bypass := hasBang && m.cfg.CacheEnabled

	arg := ""
	if len(parts) > 1 {
		arg = strings.Join(parts[1:], " ")
	}

	switch cmdName {
	case "/back":
		return func() tea.Msg {
			return TransitionToSearchMsg{}
		}

	case "/init":
		return func() tea.Msg {
			return EnterInitMsg{}
		}

	case "/help":
		m.result = buildAIHelp()
		m.Viewport.SetContent(render.WrapContent(m.result, m.Viewport.Width))
		m.Viewport.GotoTop()
		return nil

	case "/cache":
		switch strings.ToLower(arg) {
		case "on":
			m.cfg.CacheEnabled = true
			return m.SetStatus(i18n.T("ai.cache_on"))
		case "off":
			m.cfg.CacheEnabled = false
			return m.SetStatus(i18n.T("ai.cache_off"))
		default:
			state := "off"
			if m.cfg.CacheEnabled {
				state = "on"
			}
			return m.SetStatus(i18n.T("ai.cache_status", state))
		}
	}

	// All remaining commands require AI configuration
	if !m.cfg.IsConfigured() {
		return m.SetStatus(i18n.T("ai.err_need_init"))
	}

	switch cmdName {
	case "/diff", "/diff!":
		return m.handleDiff(arg, input, bypass)

	case "/ant", "/ant!":
		return m.handleListCmd("ant", arg, input, bypass)

	case "/syn", "/syn!":
		return m.handleListCmd("syn", arg, input, bypass)

	case "/phr", "/phr!":
		return m.handleListCmd("phr", arg, input, bypass)

	case "/example", "/example!":
		return m.handleExample(arg, input, bypass)

	case "/explain", "/explain!":
		return m.handleExplain(arg, input, bypass)

	default:
		return m.SetStatus(i18n.T("ai.unknown_cmd", cmdRaw))
	}
}

func (m *AIModel) handleDiff(arg, input string, bypass bool) tea.Cmd {
	words := strings.Fields(arg)
	if len(words) < 2 {
		return m.SetStatus(i18n.T("ai.err_missing_word", "/diff"))
	}
	if len(words) > 5 {
		return m.SetStatus(i18n.T("ai.err_too_many", len(words)))
	}

	var nonAlpha []string
	for _, w := range words {
		if !alphaRE.MatchString(w) {
			nonAlpha = append(nonAlpha, w)
		}
	}
	if len(nonAlpha) > 0 {
		return m.SetStatus(i18n.T("ai.err_invalid_word", strings.Join(nonAlpha, ", ")))
	}

	systemPrompt, userPrompt := DiffPrompt(words)
	cmdDisplay := fmt.Sprintf("diff %s", strings.Join(words, ", "))
	return m.callAI(input, cmdDisplay, systemPrompt, userPrompt, "diff", bypass)
}

func (m *AIModel) handleListCmd(kind, arg, input string, bypass bool) tea.Cmd {
	parts := strings.Fields(arg)
	if len(parts) == 0 {
		return m.SetStatus(i18n.T("ai.err_missing_word", "/"+kind))
	}

	word := parts[0]
	level := "some"
	if len(parts) > 1 {
		switch strings.ToLower(parts[1]) {
		case "one", "some", "many":
			level = strings.ToLower(parts[1])
		}
	}

	var systemPrompt, userPrompt string
	switch kind {
	case "ant":
		systemPrompt, userPrompt = AntPrompt(word, level)
	case "syn":
		systemPrompt, userPrompt = SynPrompt(word, level)
	case "phr":
		systemPrompt, userPrompt = PhrPrompt(word, level)
	}

	cmdDisplay := fmt.Sprintf("%s %s (%s)", kind, word, level)
	return m.callAI(input, cmdDisplay, systemPrompt, userPrompt, "list", bypass)
}

func (m *AIModel) handleExample(arg, input string, bypass bool) tea.Cmd {
	word := strings.Fields(arg)
	if len(word) == 0 {
		return m.SetStatus(i18n.T("ai.err_missing_word", "/example"))
	}

	systemPrompt, userPrompt := ExamplePrompt(word[0])
	cmdDisplay := fmt.Sprintf("example %s", word[0])
	return m.callAI(input, cmdDisplay, systemPrompt, userPrompt, "example", bypass)
}

func (m *AIModel) handleExplain(arg, input string, bypass bool) tea.Cmd {
	word := strings.Fields(arg)
	if len(word) == 0 {
		return m.SetStatus(i18n.T("ai.err_missing_word", "/explain"))
	}

	systemPrompt, userPrompt := ExplainPrompt(word[0])
	cmdDisplay := fmt.Sprintf("explain %s", word[0])
	return m.callAI(input, cmdDisplay, systemPrompt, userPrompt, "explain", bypass)
}

func (m *AIModel) callAI(input, cmdDisplay, systemPrompt, userPrompt, kind string, bypass bool) tea.Cmd {
	m.waiting = true
	m.result = ""
	m.spinnerIdx = 0
	cfg := *m.cfg

	return tea.Batch(
		func() tea.Msg {
			ctx := context.Background()
			response, cached, err := CallAIWithCache(ctx, input, systemPrompt, userPrompt, cfg, bypass)
			return AIResultMsg{
				Response:   response,
				Cached:     cached,
				Err:        err,
				CmdDisplay: cmdDisplay,
				Kind:       kind,
			}
		},
		tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return aiTickMsg{}
		}),
	)
}

// ---- Response formatting ----

func formatAIResponse(msg AIResultMsg) string {
	parsed, err := ParseJSONResponse(msg.Response)
	if err != nil {
		return aiErrStyle.Render(err.Error())
	}

	if valid, ok := parsed["valid"].(bool); ok && !valid {
		reason, _ := parsed["reason"].(string)
		return formatInvalidResult(msg.CmdDisplay, reason, msg.Cached)
	}

	var b strings.Builder

	header := aiTitleStyle.Render(fmt.Sprintf("[%s]", msg.CmdDisplay))
	if msg.Cached {
		header += aiDimStyle.Render(" " + i18n.T("ai.cached_hint"))
	}
	b.WriteString(header)
	b.WriteString("\n\n")

	switch msg.Kind {
	case "diff":
		formatDiffResult(&b, parsed)
	case "list":
		formatListResult(&b, parsed)
	case "example":
		formatExampleResult(&b, parsed)
	case "explain":
		formatExplainResult(&b, parsed)
	}

	return b.String()
}

func formatInvalidResult(cmdDisplay, reason string, cached bool) string {
	var b strings.Builder
	header := aiTitleStyle.Render(fmt.Sprintf("[%s]", cmdDisplay))
	if cached {
		header += aiDimStyle.Render(" " + i18n.T("ai.cached_hint"))
	}
	b.WriteString(header)
	b.WriteString("\n\n")
	b.WriteString(aiWarnStyle.Render("⚠ " + i18n.T("ai.invalid_request")))
	b.WriteString("\n")
	if reason != "" {
		b.WriteString(reason)
		b.WriteString("\n")
	}
	return b.String()
}

func formatDiffResult(b *strings.Builder, parsed map[string]any) {
	if expl, ok := parsed["explanation"].(string); ok {
		b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.explanation")))
		b.WriteString("\n")
		b.WriteString(expl)
		b.WriteString("\n\n")
	}

	if examples, ok := parsed["examples"].([]any); ok {
		b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.examples")))
		b.WriteString("\n")
		for i, ex := range examples {
			if s, ok := ex.(string); ok {
				b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, s))
			}
		}
	}
}

func formatListResult(b *strings.Builder, parsed map[string]any) {
	var items []any
	var ok bool

	for _, key := range []string{"antonyms", "synonyms", "phrases"} {
		if items, ok = parsed[key].([]any); ok {
			break
		}
	}

	if !ok {
		b.WriteString(aiErrStyle.Render("unexpected response format"))
		return
	}

	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		word, _ := entry["word"].(string)
		def, _ := entry["definition"].(string)
		b.WriteString(fmt.Sprintf("  %s", aiWordStyle.Render(word)))
		if def != "" {
			b.WriteString(aiDimStyle.Render("  —  " + def))
		}
		b.WriteString("\n")
	}
}

func formatExampleResult(b *strings.Builder, parsed map[string]any) {
	examples, ok := parsed["examples"].([]any)
	if !ok {
		return
	}
	b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.examples")))
	b.WriteString("\n")
	for i, ex := range examples {
		entry, ok := ex.(map[string]any)
		if !ok {
			continue
		}
		en, _ := entry["en"].(string)
		zh, _ := entry["zh"].(string)
		if en == "" && zh == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("  %d. %s", i+1, en))
		if zh != "" {
			b.WriteString(fmt.Sprintf(" / %s", zh))
		}
		b.WriteString("\n")
	}
}

func formatExplainResult(b *strings.Builder, parsed map[string]any) {
	if def, ok := parsed["definition"].(string); ok && def != "" {
		b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.definition")))
		b.WriteString("\n")
		b.WriteString(def)
		b.WriteString("\n\n")
	}
	if etym, ok := parsed["etymology"].(string); ok && etym != "" {
		b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.etymology")))
		b.WriteString("\n")
		b.WriteString(etym)
		b.WriteString("\n\n")
	}
	if usage, ok := parsed["usage_notes"].(string); ok && usage != "" {
		b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.usage_notes")))
		b.WriteString("\n")
		b.WriteString(usage)
		b.WriteString("\n\n")
	}
	if examples, ok := parsed["example_sentences"].([]any); ok && len(examples) > 0 {
		b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.examples")))
		b.WriteString("\n")
		for i, ex := range examples {
			if s, ok := ex.(string); ok {
				b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, s))
			}
		}
	}
}

func buildAIHelp() string {
	var b strings.Builder
	b.WriteString(aiTitleStyle.Render(i18n.T("ai.help_header")))
	b.WriteString("\n\n")

	items := []struct{ cmd, desc string }{
		{"/back", i18n.T("ai.help_back")},
		{"/init", i18n.T("ai.help_init")},
		{"/cache on|off", i18n.T("ai.help_cache")},
		{"/diff <w1> <w2> ...", i18n.T("ai.help_diff")},
		{"/ant <word> [one|some|many]", i18n.T("ai.help_ant")},
		{"/syn <word> [one|some|many]", i18n.T("ai.help_syn")},
		{"/phr <word> [one|some|many]", i18n.T("ai.help_phr")},
		{"/example <word>", i18n.T("ai.help_example")},
		{"/explain <word>", i18n.T("ai.help_explain")},
		{"/help", i18n.T("ai.help_help")},
	}
	for _, item := range items {
		b.WriteString(fmt.Sprintf("  %s", aiWordStyle.Render(item.cmd)))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("    %s\n", aiDimStyle.Render(item.desc)))
	}
	b.WriteString("\n")
	b.WriteString(aiDimStyle.Render(i18n.T("ai.help_cache_hint")))
	return b.String()
}

// TransitionToSearchMsg signals the parent model to switch back to search.
type TransitionToSearchMsg struct{}

// TransitionToAIMsg signals the parent model to switch back to AI mode.
type TransitionToAIMsg struct{}

// EnterInitMsg signals the parent model to enter init config mode.
type EnterInitMsg struct{}
