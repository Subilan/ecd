package tui

import (
	"strings"

	"github.com/Subilan/ecd/internal/dict"
	"github.com/Subilan/ecd/internal/i18n"
	"github.com/Subilan/ecd/internal/render"
	"github.com/Subilan/ecd/internal/repl"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type searchModel struct {
	repl.Base
	results      []searchResultItem
	query        string
	loading      bool
	err          string
	savedYOffset int

	focusInput bool // true = input receives arrows (history), false = viewport receives arrows (scroll)
}

type searchResultItem struct {
	header      string
	definition  string
	examples    []string
	synonyms    string
	antonyms    string
	notes       []string
	source      string
}

func newSearchModel() searchModel {
	ti := textinput.New()
	ti.Placeholder = i18n.T("common.search_placeholder")
	ti.CharLimit = 100
	ti.Width = 50

	vp := viewport.New(80, 20)
	// Remove single-character keybindings (j, k, h, l, f, b, d, u, space)
	// that conflict with typing in the search input. Keep only dedicated
	// navigation keys.
	vp.KeyMap = viewport.KeyMap{
		PageDown:     key.NewBinding(key.WithKeys("pgdown")),
		PageUp:       key.NewBinding(key.WithKeys("pgup")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u")),
	}

	return searchModel{
		Base:       repl.NewBase(ti, vp),
		focusInput: true,
	}
}

func (m searchModel) Update(msg tea.Msg) (searchModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Base.HandleWindowSizeMsg(msg)
		if len(m.results) > 0 {
			m.Viewport.SetContent(m.renderResults())
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.Input.SetValue("")
			m.query = ""
			m.results = nil
			m.Viewport.SetContent("")
			return m, nil

		case "tab":
			m.focusInput = !m.focusInput
			return m, nil

		case "up":
			if m.focusInput {
				m.Base.HistoryUp()
			} else {
				m.Viewport.ScrollUp(1)
			}
			return m, nil

		case "down":
			if m.focusInput {
				m.Base.HistoryDown()
			} else {
				m.Viewport.ScrollDown(1)
			}
			return m, nil
		}
	}

	// Handle input changes
	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	cmds = append(cmds, cmd)

	newQuery := strings.TrimSpace(m.Input.Value())
	if newQuery != m.query {
		m.query = newQuery
		m.Base.ResetHistoryIfChanged(newQuery)
	}

	m.Viewport, cmd = m.Viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m searchModel) View() string {
	var b strings.Builder

	b.WriteString(m.Input.View())
	b.WriteString("\n")

	if m.loading {
		b.WriteString(DimStyle.Render("  Searching..."))
	} else if m.err != "" {
		b.WriteString(WarnStyle.Render("  " + m.err))
	} else {
		b.WriteString(m.Viewport.View())
	}

	return b.String()
}

func (m *searchModel) renderResults() string {
	var b strings.Builder
	for _, r := range m.results {
		b.WriteString(r.header)
		b.WriteString("\n")
		if r.definition != "" {
			b.WriteString(r.definition)
			b.WriteString("\n")
		}
		for _, ex := range r.examples {
			b.WriteString(ex)
			b.WriteString("\n")
		}
		if r.synonyms != "" {
			b.WriteString(r.synonyms)
			b.WriteString("\n")
		}
		if r.antonyms != "" {
			b.WriteString(r.antonyms)
			b.WriteString("\n")
		}
		for _, n := range r.notes {
			b.WriteString(n)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return render.WrapContent(b.String(), m.Viewport.Width)
}

// bracketFor returns the bracket style for a word based on its flashcard status.
func bracketFor(status string) lipgloss.Style {
	switch status {
	case "leech":
		return LeechBracketStyle
	case "flashcard":
		return FlashcardBracketStyle
	default:
		return BracketStyle
	}
}

func buildSearchItems(entries []dict.Entry, statuses map[string]string) []searchResultItem {
	var items []searchResultItem
	for _, e := range entries {
		item := searchResultItem{source: e.Source}

		srcLabel := i18n.T("source." + e.Source)
		posStr := ""
		if e.Pos != "" {
			posStr = " " + e.Pos
		}

		pronStr := ""
		if len(e.Pronunciation) > 0 {
			joined := strings.Join(e.Pronunciation, " | ")
			pronStr = DimStyle.Render(" /") + PronStyle.Render(joined) + DimStyle.Render("/")
		}

		br := bracketFor(statuses[e.Word])
		item.header = SourceStyle.Render(srcLabel) + ": " +
			br.Render("[") + WordStyle.Render(" "+e.Word+" ") + br.Render("]") + pronStr + DimStyle.Render(posStr)

		if e.CnDefinition != "" {
			item.definition = LabelStyle.Render(i18n.T("label.definition")+":") + " " + e.CnDefinition
		}

		for _, ex := range e.Examples {
			var exStr string
			if ex.En != "" && ex.Cn != "" {
				exStr = LabelStyle.Render(i18n.T("label.example")+":") + " " + ex.En + " / " + ex.Cn
			} else if ex.En != "" {
				exStr = LabelStyle.Render(i18n.T("label.example")+":") + " " + ex.En
			} else if ex.Cn != "" {
				exStr = LabelStyle.Render(i18n.T("label.example_cn")+":") + " " + ex.Cn
			}
			item.examples = append(item.examples, exStr)
		}

		if len(e.Synonyms) > 0 {
			var parts []string
			for _, s := range e.Synonyms {
				parts = append(parts, WordStyle.Render(s))
			}
			item.synonyms = LabelStyle.Render(i18n.T("label.synonym")+":") + " " +
				strings.Join(parts, DimStyle.Render(", "))
		}

		if len(e.Antonyms) > 0 {
			var parts []string
			for _, a := range e.Antonyms {
				parts = append(parts, WordStyle.Render(a))
			}
			item.antonyms = LabelStyle.Render(i18n.T("label.antonym")+":") + " " +
				strings.Join(parts, DimStyle.Render(", "))
		}

		for _, note := range e.ExtraNotes {
			typeLabel := noteTypeLabel(note.Type)
			noteStr := LabelStyle.Render("["+typeLabel+"]") + "\n"
			enLines := strings.Split(note.En, "\n")
			cnLines := strings.Split(note.Cn, "\n")
			maxLines := len(enLines)
			if len(cnLines) > maxLines {
				maxLines = len(cnLines)
			}
			for i := 0; i < maxLines; i++ {
				var enPart, cnPart string
				if i < len(enLines) {
					enPart = strings.TrimRight(enLines[i], " ")
				}
				if i < len(cnLines) {
					cnPart = strings.TrimRight(cnLines[i], " ")
				}
				if enPart != "" && cnPart != "" {
					noteStr += enPart + " / " + cnPart + "\n"
				} else if enPart != "" {
					noteStr += enPart + "\n"
				} else if cnPart != "" {
					noteStr += cnPart + "\n"
				}
			}
			item.notes = append(item.notes, strings.TrimRight(noteStr, "\n"))
		}

		items = append(items, item)
	}
	return items
}

func (m *searchModel) saveScrollPos() {
	m.savedYOffset = m.Viewport.YOffset
}

func (m *searchModel) restoreScrollPos() {
	m.Viewport.SetYOffset(m.savedYOffset)
}

func buildChineseItems(results []dict.ChineseResult, statuses map[string]string) []searchResultItem {
	var items []searchResultItem
	for _, r := range results {
		item := searchResultItem{source: r.Source}
		srcLabel := i18n.T("source." + r.Source)
		br := bracketFor(statuses[r.Word])
		item.header = SourceStyle.Render(srcLabel) + ": " +
			br.Render("[") + WordStyle.Render(" "+r.Word+" ") + br.Render("]") + " " + r.CnDef
		for _, ex := range r.Examples {
			item.examples = append(item.examples, "  "+ex)
		}
		items = append(items, item)
	}
	return items
}

func noteTypeLabel(noteType string) string {
	labels := map[string]string{
		"usage":     i18n.T("note.usage"),
		"drv":       i18n.T("note.drv"),
		"regional":  i18n.T("note.regional"),
		"sense":     i18n.T("note.sense"),
		"quotation": i18n.T("note.quotation"),
		"phrase":    i18n.T("note.phrase"),
		"note":      i18n.T("note.general"),
	}
	if l, ok := labels[noteType]; ok {
		return l
	}
	return noteType
}
