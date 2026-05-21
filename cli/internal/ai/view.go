package ai

import (
	"fmt"
	"strings"

	"github.com/Subilan/ecd/internal/i18n"
	"github.com/charmbracelet/lipgloss"
)

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

	if !m.focusInput {
		b.WriteString("  " + aiPronStyle.Render("SCROLL"))
	}

	if s := m.StatusMsg(); s != "" {
		b.WriteString("\n\n" + aiLabelStyle.Render(s))
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Padding(1, 0, 1, 0).Render(b.String())
}

// BuildAIHelp builds the AI command help text.
func BuildAIHelp() string {
	lines := []string{
		"",
		aiTitleStyle.Render("  " + i18n.T("ai.help_header")),
		"",
		"  " + aiLabelStyle.Render(i18n.T("ai.help_section_commands")),
		"    " + aiWordStyle.Render("/back") + aiDimStyle.Render("  —  "+i18n.T("ai.help_back")),
		"    " + aiWordStyle.Render("/init") + aiDimStyle.Render("  —  "+i18n.T("ai.help_init")),
		"    " + aiWordStyle.Render("/cache on|off") + aiDimStyle.Render("  —  "+i18n.T("ai.help_cache")),
		"    " + aiWordStyle.Render("/diff <w1> <w2> ...") + aiDimStyle.Render("  —  "+i18n.T("ai.help_diff")),
		"    " + aiWordStyle.Render("/ant <word> [one|some|many]") + aiDimStyle.Render("  —  "+i18n.T("ai.help_ant")),
		"    " + aiWordStyle.Render("/syn <word> [one|some|many]") + aiDimStyle.Render("  —  "+i18n.T("ai.help_syn")),
		"    " + aiWordStyle.Render("/phr <word> [one|some|many]") + aiDimStyle.Render("  —  "+i18n.T("ai.help_phr")),
		"    " + aiWordStyle.Render("/example <word>") + aiDimStyle.Render("  —  "+i18n.T("ai.help_example")),
		"    " + aiWordStyle.Render("/explain <word>") + aiDimStyle.Render("  —  "+i18n.T("ai.help_explain")),
		"    " + aiWordStyle.Render("/help") + aiDimStyle.Render("  —  "+i18n.T("ai.help_help")),
		"",
		aiDimStyle.Render("  " + i18n.T("ai.help_cache_hint")),
		"",
		aiDimStyle.Render("  " + i18n.T("common.press_any_key")),
	}
	return strings.Join(lines, "\n")
}
