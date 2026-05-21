package tui

import (
	"strings"

	"github.com/Subilan/ecd/internal/dict"
	"github.com/Subilan/ecd/internal/i18n"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type searchModel struct {
	input        textinput.Model
	viewport     viewport.Model
	results      []searchResultItem
	query        string
	loading      bool
	err          string
	width        int
	height       int
	savedYOffset int

	history      []string // search history ring buffer
	historyIdx   int      // -1 = not navigating, 0..len-1 = active
	historySaved string   // input text before history navigation started

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
		input:      ti,
		viewport:   vp,
		focusInput: true,
	}
}

func (m searchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m searchModel) Update(msg tea.Msg) (searchModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = max(1, msg.Width-4)
		m.viewport.Width = max(1, msg.Width-2)
		m.viewport.Height = max(1, msg.Height-7)

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.input.SetValue("")
			m.query = ""
			m.results = nil
			m.viewport.SetContent("")
			return m, nil

		case "tab":
			m.focusInput = !m.focusInput
			return m, nil

		case "up":
			if m.focusInput {
				m.historyUp()
			} else {
				m.viewport.ScrollUp(1)
			}
			return m, nil

		case "down":
			if m.focusInput {
				m.historyDown()
			} else {
				m.viewport.ScrollDown(1)
			}
			return m, nil
		}
	}

	// Handle input changes
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	newQuery := strings.TrimSpace(m.input.Value())
	if newQuery != m.query {
		m.query = newQuery
		// If the query no longer matches the history entry at the current
		// index (user typed or deleted a character), exit history mode.
		if m.historyIdx >= 0 && m.historyIdx < len(m.history) &&
			m.history[m.historyIdx] != newQuery {
			m.historyIdx = -1
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m searchModel) View() string {
	var b strings.Builder

	b.WriteString(m.input.View())
	b.WriteString("\n")

	if m.loading {
		b.WriteString(DimStyle.Render("  Searching..."))
	} else if m.err != "" {
		b.WriteString(WarnStyle.Render("  " + m.err))
	} else {
		b.WriteString(m.viewport.View())
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
	return b.String()
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
	m.savedYOffset = m.viewport.YOffset
}

func (m *searchModel) restoreScrollPos() {
	m.viewport.SetYOffset(m.savedYOffset)
}

const maxHistory = 100

func (m *searchModel) addHistory(query string) {
	if query == "" {
		return
	}
	// Dedup consecutive duplicates.
	if len(m.history) > 0 && m.history[len(m.history)-1] == query {
		return
	}
	m.history = append(m.history, query)
	// Trim from front if over capacity.
	if len(m.history) > maxHistory {
		m.history = m.history[len(m.history)-maxHistory:]
	}
	m.historyIdx = -1
}

func (m *searchModel) historyUp() {
	if len(m.history) == 0 {
		return
	}
	if m.historyIdx == -1 {
		m.historySaved = m.input.Value()
		m.historyIdx = len(m.history) - 1
	} else if m.historyIdx > 0 {
		m.historyIdx--
	}
	m.input.SetValue(m.history[m.historyIdx])
	m.input.CursorEnd()
}

func (m *searchModel) historyDown() {
	if m.historyIdx == -1 {
		return
	}
	if m.historyIdx < len(m.history)-1 {
		m.historyIdx++
		m.input.SetValue(m.history[m.historyIdx])
		m.input.CursorEnd()
	} else {
		m.historyIdx = -1
		m.input.SetValue(m.historySaved)
		m.historySaved = ""
	}
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
