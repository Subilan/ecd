package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Subilan/ecd/internal/i18n"
	tea "github.com/charmbracelet/bubbletea"
)

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
		return func() tea.Msg {
			return ShowHelpMsg{}
		}

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
