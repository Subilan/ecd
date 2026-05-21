package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Subilan/ecd/internal/dict"
	"github.com/Subilan/ecd/internal/history"
	"github.com/Subilan/ecd/internal/i18n"
	"github.com/Subilan/ecd/internal/search"
	"github.com/Subilan/ecd/internal/sm2"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the top-level Bubble Tea model.
type Model struct {
	state  AppState
	width  int
	height int
	ready  bool
	err    error

	// Data sources
	DictDB    *dict.DB
	HistoryDB *history.DB

	// Shared state
	lang     i18n.Lang
	lastWord string
	autoAdd  bool

	// Sub-models
	search  searchModel
	detail  detailModel
	review  reviewModel
	deck    deckModel
	help    helpModel

	// Context for search operations
	searchCtx *search.Context

	// Status message shown below the footer
	statusMsg string
	statusSeq int // incremented on each new message; timer only clears if seq matches
}

func (m *Model) statusTimerCmd() tea.Cmd {
	m.statusSeq++
	seq := m.statusSeq
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return clearStatusMsg{seq: seq}
	})
}

// NewModel creates the main TUI model.
func NewModel(dictDB *dict.DB, historyDB *history.DB) Model {
	var lastWord string

	sm := newSearchModel()
	sm.input.Focus()

	return Model{
		state:     StateSearch,
		DictDB:    dictDB,
		HistoryDB: historyDB,
		lang:      i18n.GetLang(),
		search:    sm,
		detail:    newDetailModel(),
		review:    newReviewModel(),
		deck:      newDeckModel(),
		help:      newHelpModel(),
		searchCtx: &search.Context{
			DictDB:    dictDB,
			HistoryDB: historyDB,
			LastWord:  &lastWord,
		},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.search.Init(),
		func() tea.Msg { return tea.WindowSizeMsg{} },
	)
}

// ---- Messages ----

type clearStatusMsg struct{ seq int }

type searchDoneMsg struct {
	result *search.QueryResult
	query  string
}

// ---- Update ----

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.search, _ = m.search.Update(msg)
		m.detail, _ = m.detail.Update(msg)
		return m, nil

	case clearStatusMsg:
		if msg.seq == m.statusSeq {
			m.statusMsg = ""
		}

	case tea.KeyMsg:
		// Global keybindings (work in any state)
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		// State-specific handling
		switch m.state {
		case StateSearch:
			return m.updateSearch(msg)

		case StateEntryDetail, StateDeckStats, StateHelp:
			switch msg.String() {
			case "esc", "q":
				m.state = StateSearch
				m.search.restoreScrollPos()
				m.search.input.Focus()
				return m, nil
			default:
				m.state = StateSearch
				m.search.restoreScrollPos()
				m.search.input.Focus()
				return m, nil
			}

		case StateConfirmReset:
			if msg.String() == "esc" {
				m.state = StateSearch
				m.statusMsg = i18n.T("reset.cancelled")
				return m, nil
			}
			if msg.String() == "enter" {
				query := strings.TrimSpace(m.search.input.Value())
				if strings.HasPrefix(query, "/") {
					m.search.input.SetValue("")
					return m.handleSlashCommand(query)
				}
			}
			// Pass to search for text input
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			return m, cmd

		case StateReview:
			return m.updateReview(msg)
		}

	case searchDoneMsg:
		if msg.result.Entries != nil {
			statuses := m.flashcardStatusesForEntries(msg.result.Entries, nil)
			m.search.results = buildSearchItems(msg.result.Entries, statuses)
		} else if msg.result.Chinese != nil {
			statuses := m.flashcardStatusesForChinese(msg.result.Chinese)
			m.search.results = buildChineseItems(msg.result.Chinese, statuses)
		} else if msg.result.Suggestions != nil {
			m.search.results = []searchResultItem{{
				header: LabelStyle.Render(
					i18n.T("search.did_you_mean",
						search.FormatSuggestions(msg.result.Suggestions))),
			}}
		} else if msg.result.NotFound {
			m.search.results = []searchResultItem{{
				header: DimStyle.Render(
					i18n.T("search.no_results", msg.query)),
			}}
		}
		m.search.loading = false
		m.search.viewport.SetContent(m.search.renderResults())
		m.search.viewport.GotoTop()

		// Auto-add
		if m.autoAdd && *m.searchCtx.LastWord != "" {
			m.HistoryDB.AddFlashcard(*m.searchCtx.LastWord)
		}
	}

	// Delegate non-key messages to active sub-model
	switch m.state {
	case StateSearch:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		if cmd != nil {
			return m, cmd
		}
	case StateEntryDetail:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		if cmd != nil {
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// ESC clears input
	if msg.String() == "esc" {
		m.search.input.SetValue("")
		return m, nil
	}

	// Enter: process input (command or search)
	if msg.String() == "enter" {
		query := strings.TrimSpace(m.search.input.Value())
		if query == "" {
			return m, nil
		}

		// Check for /command
		if strings.HasPrefix(query, "/") {
			return m.handleSlashCommand(query)
		}

		// Regular search — clear input after submit
		m.statusMsg = ""
		m.search.addHistory(query)
		m.search.input.SetValue("")
		return m, m.doSearch(query, nil)
	}

	// Pass all other keys to the search sub-model (textinput + viewport)
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	return m, cmd
}

func (m *Model) handleSlashCommand(query string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(query)
	if len(parts) == 0 {
		return m, nil
	}
	cmdName := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = strings.Join(parts[1:], " ")
	}

	m.search.input.SetValue("")
	m.statusMsg = ""

	switch cmdName {
	case "/exit", "/quit", "/q":
		return m, tea.Quit

	case "/help":
		m.state = StateHelp

	case "/lang":
		switch strings.ToLower(arg) {
		case "en":
			i18n.SetLang(i18n.LangEN)
			m.lang = i18n.LangEN
			m.statusMsg = i18n.T("interactive.lang_switched")
		case "zh":
			i18n.SetLang(i18n.LangZH)
			m.lang = i18n.LangZH
			m.statusMsg = i18n.T("interactive.lang_switched")
		default:
			m.statusMsg = fmt.Sprintf("Usage: /lang en|zh  (current: %s)", i18n.GetLang())
		}
		return m, m.statusTimerCmd()

	case "/syn":
		word := arg
		if word == "" {
			word = m.lastWord
		}
		if word == "" {
			return m, m.setStatus(i18n.T("synonym.usage"))
		}
		synResults, notFound := search.SearchSynonyms(m.searchCtx, word, nil)
		if notFound != nil {
			return m, m.setStatus(i18n.T("synonym.no_entries", word))
		}
		if len(synResults) == 0 {
			return m, m.setStatus(i18n.T("synonym.not_found", word))
		}
		var totalSyns int
		for _, sr := range synResults {
			totalSyns += len(sr.Items)
		}
		var items []searchResultItem
		items = append(items, searchResultItem{
			header: LabelStyle.Render(i18n.T("synonym.found", totalSyns)),
		})
		for _, sr := range synResults {
			for _, s := range sr.Items {
				items = append(items, searchResultItem{
					header: fmt.Sprintf("%s: %s",
						SourceStyle.Render(i18n.T("source."+sr.Source)),
						WordStyle.Render(s)),
				})
			}
		}
		m.search.results = items
		m.search.viewport.SetContent(m.search.renderResults())
		m.search.viewport.GotoTop()

	case "/ant":
		word := arg
		if word == "" {
			word = m.lastWord
		}
		if word == "" {
			return m, m.setStatus(i18n.T("antonym.usage"))
		}
		antResults, notFound := search.SearchAntonyms(m.searchCtx, word, nil)
		if notFound != nil {
			return m, m.setStatus(i18n.T("synonym.no_entries", word))
		}
		if len(antResults) == 0 {
			return m, m.setStatus(i18n.T("antonym.not_found", word))
		}
		var totalAnts int
		for _, ar := range antResults {
			totalAnts += len(ar.Items)
		}
		var items []searchResultItem
		items = append(items, searchResultItem{
			header: LabelStyle.Render(i18n.T("antonym.found", totalAnts)),
		})
		for _, ar := range antResults {
			for _, a := range ar.Items {
				items = append(items, searchResultItem{
					header: fmt.Sprintf("%s: %s",
						SourceStyle.Render(i18n.T("source."+ar.Source)),
						WordStyle.Render(a)),
				})
			}
		}
		m.search.results = items
		m.search.viewport.SetContent(m.search.renderResults())
		m.search.viewport.GotoTop()

	case "/add":
		word := arg
		if word == "" {
			word = m.lastWord
		}
		if word == "" {
			return m, m.setStatus(i18n.T("add.no_word"))
		}
		added, _ := m.HistoryDB.AddFlashcard(word)
		if added {
			m.statusMsg = i18n.T("add.added", word)
			m.lastWord = word
		} else {
			m.statusMsg = i18n.T("add.already", word)
		}
		return m, m.statusTimerCmd()

	case "/del":
		word := arg
		if word == "" {
			return m, m.setStatus(i18n.T("del.usage"))
		}
		deleted, _ := m.HistoryDB.DelFlashcard(word)
		if deleted {
			m.statusMsg = i18n.T("del.removed", word)
		} else {
			m.statusMsg = i18n.T("del.not_found", word)
		}
		return m, m.statusTimerCmd()

	case "/auto-add":
		switch strings.ToLower(arg) {
		case "on":
			m.autoAdd = true
		case "off":
			m.autoAdd = false
		default:
			m.autoAdd = !m.autoAdd
		}
		state := "OFF"
		if m.autoAdd {
			state = "ON"
		}
		return m, m.setStatus(i18n.T("interactive.auto_add", state))

	case "/review":
		m.startReview()

	case "/deck":
		m.showDeckStats()

	case "/reset":
		m.state = StateConfirmReset
		m.search.input.SetValue("")

	case "/reset-confirm":
		if m.state == StateConfirmReset {
			m.HistoryDB.ResetAll()
			m.statusMsg = i18n.T("reset.done")
			m.state = StateSearch
			return m, m.statusTimerCmd()
		}

	case "/random":
		word, err := m.DictDB.RandomWord(nil)
		if err == nil && word != "" {
			m.search.input.SetValue(word)
			return m, m.doSearch(word, nil)
		}
		return m, m.setStatus(i18n.T("search.no_words"))

	default:
		return m, m.setStatus(i18n.T("interactive.unknown_cmd", cmdName))
	}
	return m, nil
}

func (m Model) updateReview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.review, cmd = m.review.Update(msg)

	switch m.review.state {
	case reviewComplete:
		// Any key returns to search
		if msg.String() != "" {
			m.state = StateSearch
			m.search.restoreScrollPos()
			m.search.input.Focus()
			return m, m.setStatus(fmt.Sprintf(i18n.T("review.complete"), len(m.review.cards)))
		}
	case reviewFront, reviewBack:
		// Continue review
	}

	return m, cmd
}

func (m *Model) setStatus(msg string) tea.Cmd {
	m.statusMsg = msg
	return m.statusTimerCmd()
}

// ---- View ----

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}
	if m.err != nil {
		return WarnStyle.Render(fmt.Sprintf("Error: %s", m.err))
	}

	var content string
	switch m.state {
	case StateSearch:
		content = m.search.View()
	case StateEntryDetail:
		content = m.detail.View()
	case StateReview:
		content = m.review.View()
	case StateDeckStats:
		content = m.deck.View()
	case StateHelp:
		content = m.help.View()
	case StateConfirmReset:
		content = m.confirmResetView()
	default:
		content = ""
	}

	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, content, footer)
}

func (m Model) renderFooter() string {
	var b strings.Builder
	b.WriteString(DimStyle.Render(i18n.T("footer.hint")))

	if m.autoAdd {
		b.WriteString("  " + DimStyle.Render("[auto]"))
	}

	if !m.search.focusInput {
		b.WriteString("  " + PronStyle.Render("SCROLL"))
	}

	if m.statusMsg != "" {
		b.WriteString("\n\n" + LabelStyle.Render(m.statusMsg))
	}

	return FooterStyle.Render(b.String())
}

func (m Model) confirmResetView() string {
	return lipgloss.NewStyle().Padding(2).Render(
		i18n.T("reset.confirm") + "\n\n" +
			"  /reset-confirm  — " + DimStyle.Render("confirm and delete all cards") + "\n" +
			"  Esc            — " + DimStyle.Render("cancel"),
	)
}

// ---- Commands ----

func (m *Model) doSearch(query string, source *string) tea.Cmd {
	m.search.loading = true
	return func() tea.Msg {
		result := search.HandleQuery(m.searchCtx, query, source)
		return searchDoneMsg{result: result, query: query}
	}
}

func (m *Model) startReview() {
	cards, err := m.HistoryDB.GetDueCards(20)
	if err != nil || len(cards) == 0 {
		m.statusMsg = i18n.T("review.no_due")
		return
	}
	m.review = newReviewModel()
	m.review.dictDB = m.DictDB
	m.review.historyDB = m.HistoryDB
	m.review.cards = cards
	m.review.state = reviewFront
	m.review.currentIdx = 0
	m.review.loadCurrentCard()
	m.state = StateReview
}

func (m *Model) showDeckStats() {
	stats, err := m.HistoryDB.GetDeckStats()
	if err != nil {
		m.statusMsg = err.Error()
		return
	}
	m.deck.stats = stats
	m.deck.err = ""
	m.state = StateDeckStats
}

// ---- Review Model ----

type reviewState int

const (
	reviewFront reviewState = iota
	reviewBack
	reviewComplete
)

type reviewModel struct {
	state      reviewState
	dictDB     *dict.DB
	historyDB  *history.DB
	cards      []history.Flashcard
	currentIdx int
	entries    []dict.Entry
	entryIdx   int
	button     int
}

func newReviewModel() reviewModel {
	return reviewModel{}
}

func (m *reviewModel) loadCurrentCard() {
	if m.currentIdx >= len(m.cards) {
		return
	}
	card := m.cards[m.currentIdx]
	entries, _ := m.dictDB.SearchExact(card.Word, nil)
	m.entries = entries
	m.entryIdx = 0
}

func (m *reviewModel) Init() tea.Cmd { return nil }

func (m reviewModel) Update(msg tea.Msg) (reviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case reviewFront:
			if msg.String() == "enter" {
				m.state = reviewBack
			}

		case reviewBack:
			switch msg.String() {
			case "left":
				if len(m.entries) > 0 {
					m.entryIdx = (m.entryIdx - 1 + len(m.entries)) % len(m.entries)
				}
			case "right":
				if len(m.entries) > 0 {
					m.entryIdx = (m.entryIdx + 1) % len(m.entries)
				}
			case "0", "1", "2", "3":
				m.button = int(msg.String()[0] - '0')
				m.applyRating()
				m.currentIdx++
				if m.currentIdx >= len(m.cards) {
					m.state = reviewComplete
				} else {
					m.loadCurrentCard()
					m.state = reviewFront
				}
			}

		case reviewComplete:
			// Any key handled by parent
		}
	}
	return m, nil
}

func (m *reviewModel) applyRating() {
	card := m.cards[m.currentIdx]
	params := sm2.SM2Params{
		Repetitions:  card.Repetitions,
		IntervalDays: card.IntervalDays,
		EaseFactor:   card.EaseFactor,
	}
	outcome := sm2.ReviewOutcome(m.button)
	newParams := sm2.Schedule(outcome, params)

	correct := 0
	if m.button >= 2 {
		correct = 1
	}
	m.historyDB.UpdateFlashcard(card.Word, newParams.EaseFactor,
		newParams.IntervalDays, newParams.Repetitions, correct)
}

func (m reviewModel) View() string {
	var b strings.Builder

	switch m.state {
	case reviewFront:
		if m.currentIdx >= len(m.cards) {
			return ""
		}
		card := m.cards[m.currentIdx]
		b.WriteString("\n")
		b.WriteString(TitleStyle.Render(
			fmt.Sprintf(i18n.T("review.header_card"), m.currentIdx+1, len(m.cards))))
		b.WriteString("\n\n")

		b.WriteString(WordStyle.Render(card.Word))

		if len(m.entries) > 0 && len(m.entries[0].Pronunciation) > 0 {
			joined := strings.Join(m.entries[0].Pronunciation, " | ")
			b.WriteString(DimStyle.Render(" /"))
			b.WriteString(PronStyle.Render(joined))
			b.WriteString(DimStyle.Render("/"))
		}
		b.WriteString("\n\n")
		b.WriteString(DimStyle.Render(i18n.T("review.press_enter")))

	case reviewBack:
		if m.currentIdx >= len(m.cards) {
			return ""
		}
		card := m.cards[m.currentIdx]

		b.WriteString("\n")
		b.WriteString(TitleStyle.Render(
			fmt.Sprintf(i18n.T("review.header_card"), m.currentIdx+1, len(m.cards))))
		b.WriteString("\n\n")
		b.WriteString(WordStyle.Render(card.Word))
		b.WriteString("\n\n")

		if len(m.entries) == 0 {
			b.WriteString(DimStyle.Render(i18n.T("review.word_not_found")))
			b.WriteString("\n\n")
		} else {
			entry := m.entries[m.entryIdx]
			src := i18n.T("source." + entry.Source)
			posStr := entry.Pos
			if posStr == "" {
				posStr = "(none)"
			}
			b.WriteString(DimStyle.Render(
				fmt.Sprintf(i18n.T("review.entry_n_of"), m.entryIdx+1, len(m.entries), src)))
			b.WriteString("\n")
			b.WriteString(LabelStyle.Render(i18n.T("label.pos")+":") + " " + posStr)
			b.WriteString("\n")

			if entry.CnDefinition != "" {
				b.WriteString(LabelStyle.Render(i18n.T("label.definition")+":") + " " + entry.CnDefinition)
				b.WriteString("\n")
			}
			for _, ex := range entry.Examples {
				if ex.En != "" && ex.Cn != "" {
					b.WriteString(fmt.Sprintf("%s: %s / %s\n",
						LabelStyle.Render(i18n.T("label.example")), ex.En, ex.Cn))
				} else if ex.En != "" {
					b.WriteString(fmt.Sprintf("%s: %s\n",
						LabelStyle.Render(i18n.T("label.example")), ex.En))
				} else if ex.Cn != "" {
					b.WriteString(fmt.Sprintf("%s: %s\n",
						LabelStyle.Render(i18n.T("label.example_cn")), ex.Cn))
				}
			}
			for _, note := range entry.ExtraNotes {
				typeLabel := noteTypeLabel(note.Type)
				b.WriteString(LabelStyle.Render("[" + typeLabel + "]"))
				b.WriteString("\n")
				if note.En != "" && note.Cn != "" {
					b.WriteString(note.En + " / " + note.Cn + "\n")
				} else if note.En != "" {
					b.WriteString(note.En + "\n")
				} else if note.Cn != "" {
					b.WriteString(note.Cn + "\n")
				}
			}
		}

		b.WriteString("\n")
		rating := fmt.Sprintf("0=%s 1=%s 2=%s 3=%s",
			i18n.T("review.again"), i18n.T("review.hard"),
			i18n.T("review.good"), i18n.T("review.easy"))
		if len(m.entries) > 1 {
			b.WriteString(DimStyle.Render(i18n.T("review.switch_entry") + "  |  " + rating))
		} else {
			b.WriteString(DimStyle.Render(rating))
		}

	case reviewComplete:
		b.WriteString("\n")
		b.WriteString(LabelStyle.Render(
			fmt.Sprintf(i18n.T("review.complete"), len(m.cards))))
		b.WriteString("\n\n")
		b.WriteString(DimStyle.Render("Press any key to return..."))
	}

	return b.String()
}

// ---- Detail Model ----

type detailModel struct {
	viewport viewport.Model
	entry    *dict.Entry
}

func newDetailModel() detailModel {
	return detailModel{
		viewport: viewport.New(80, 20),
	}
}

func (m detailModel) Init() tea.Cmd { return nil }

func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width - 2
		m.viewport.Height = msg.Height - 4
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m detailModel) View() string {
	return m.viewport.View()
}

// ---- Deck Model ----

type deckModel struct {
	stats *history.DeckStats
	err   string
}

func newDeckModel() deckModel {
	return deckModel{}
}

func (m deckModel) Init() tea.Cmd { return nil }

func (m deckModel) Update(msg tea.Msg) (deckModel, tea.Cmd) {
	return m, nil
}

func (m deckModel) View() string {
	if m.err != "" {
		return WarnStyle.Render(m.err)
	}
	if m.stats == nil {
		return ""
	}
	if m.stats.Total == 0 {
		return DimStyle.Render(i18n.T("deck.empty"))
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(LabelStyle.Render(i18n.T("deck.stats_title")))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  %s        %d\n", LabelStyle.Render(i18n.T("deck.total")+":"), m.stats.Total))
	b.WriteString(fmt.Sprintf("  %s          %d\n", LabelStyle.Render(i18n.T("deck.due")+":"), m.stats.Due))
	b.WriteString(fmt.Sprintf("  %s          %d\n", LabelStyle.Render(i18n.T("deck.new")+":"), m.stats.New))
	b.WriteString(fmt.Sprintf("  %s       %d\n", LabelStyle.Render(i18n.T("deck.mature")+":"), m.stats.Mature))

	if m.stats.NextDelta != "" {
		if m.stats.Overdue {
			b.WriteString(fmt.Sprintf("  %s         %s%s\n",
				LabelStyle.Render(i18n.T("deck.next")+":"),
				m.stats.NextDelta,
				WarnStyle.Render(" ago")))
		} else {
			b.WriteString(fmt.Sprintf("  %s         in %s\n",
				LabelStyle.Render(i18n.T("deck.next")+":"),
				m.stats.NextDelta))
		}
	}

	if m.stats.Leeches > 0 {
		b.WriteString(fmt.Sprintf("  %s      %d\n",
			LabelStyle.Render(i18n.T("deck.leeches")+":"), m.stats.Leeches))
	}

	b.WriteString(fmt.Sprintf("  %s     %.0f%%\n",
		LabelStyle.Render(i18n.T("deck.avg_ease")+":"), m.stats.AvgEase*100))
	b.WriteString("\n")
	b.WriteString(DimStyle.Render("Press any key to return..."))
	return b.String()
}

// ---- Helpers ----

func (m *Model) flashcardStatusesForEntries(entries []dict.Entry, extraWords []string) map[string]string {
	words := make([]string, 0, len(entries)+len(extraWords))
	for _, e := range entries {
		words = append(words, e.Word)
	}
	words = append(words, extraWords...)
	if m.HistoryDB == nil || len(words) == 0 {
		return nil
	}
	return m.HistoryDB.GetFlashcardStatuses(words)
}

func (m *Model) flashcardStatusesForChinese(results []dict.ChineseResult) map[string]string {
	words := make([]string, len(results))
	for i, r := range results {
		words[i] = r.Word
	}
	if m.HistoryDB == nil || len(words) == 0 {
		return nil
	}
	return m.HistoryDB.GetFlashcardStatuses(words)
}

// ---- Help Model ----

type helpModel struct{}

func newHelpModel() helpModel     { return helpModel{} }
func (m helpModel) Init() tea.Cmd { return nil }

func (m helpModel) Update(msg tea.Msg) (helpModel, tea.Cmd) {
	return m, nil
}

func (m helpModel) View() string {
	lines := []string{
		"",
		TitleStyle.Render("  " + i18n.T("help.title")),
		"",
		"  " + i18n.T("help.desc"),
		"",
		"  " + LabelStyle.Render(i18n.T("help.section_search")),
		"    " + i18n.T("help.item_random"),
		"    " + i18n.T("help.item_syn"),
		"    " + i18n.T("help.item_ant"),
		"",
		"  " + LabelStyle.Render(i18n.T("help.section_flashcards")),
		"    " + i18n.T("help.item_add"),
		"    " + i18n.T("help.item_del"),
		"    " + i18n.T("help.item_auto_add"),
		"    " + i18n.T("help.item_review"),
		"    " + i18n.T("help.item_deck"),
		"    " + i18n.T("help.item_reset"),
		"",
		"  " + LabelStyle.Render(i18n.T("help.section_general")),
		"    " + i18n.T("help.item_help"),
		"    " + i18n.T("help.item_lang"),
		"    " + i18n.T("help.item_exit"),
		"    " + i18n.T("help.item_ctrlc"),
		"    " + i18n.T("help.item_esc"),
		"",
		DimStyle.Render("  " + i18n.T("help.close")),
	}
	return strings.Join(lines, "\n")
}
