package ai

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Subilan/ecd/internal/config"
	"github.com/Subilan/ecd/internal/i18n"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	openai "github.com/sashabaranov/go-openai"
)

// InitModel is the AI configuration form.
type InitModel struct {
	cfg *config.AIConfig

	inputs   [2]textinput.Model // 0=api_key, 1=base_url
	modelVal string             // local copy, only written to cfg on save
	cursor   int                // 0=api_key, 1=base_url, 2=model label, 3+=model list / custom
	width    int
	height   int
	ready    bool

	// Model list (inline, below model label row)
	models      []string
	modelsReady bool
	modelErr    string
	fetching    bool
	fetchKey    string // credentials used for last fetch attempt
	fetchURL    string
	selected    int // index in models, or len(models) for custom, -1 for none
	customInput textinput.Model

	// Connection test state
	testing    bool
	spinnerIdx int

	// Error shown after failed validation / connection test
	statusErr string
}

func newTextInput(placeholder string, echo textinput.EchoMode) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.EchoMode = echo
	ti.CharLimit = 200
	ti.Prompt = ""
	return ti
}

// NewInitModel creates the init form model.
func NewInitModel(cfg *config.AIConfig) InitModel {
	keyInput := newTextInput(i18n.T("ai.empty"), textinput.EchoPassword)
	keyInput.SetValue(cfg.APIKey)
	keyInput.Focus()

	urlInput := newTextInput(i18n.T("ai.empty"), textinput.EchoNormal)
	urlInput.SetValue(cfg.BaseURL)

	customTi := textinput.New()
	customTi.Placeholder = i18n.T("ai.custom_model")
	customTi.CharLimit = 100
	customTi.Prompt = ""

	return InitModel{
		cfg:         cfg,
		modelVal:    cfg.Model,
		inputs:      [2]textinput.Model{keyInput, urlInput},
		selected:    -1,
		customInput: customTi,
	}
}

func (m InitModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InitModel) Update(msg tea.Msg) (InitModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		for i := range m.inputs {
			m.inputs[i].Width = max(1, msg.Width-12)
		}
		m.customInput.Width = max(15, msg.Width-20)
		return m, nil

	case modelListMsg:
		m.fetching = false
		if msg.err != nil {
			m.modelErr = msg.err.Error()
		} else {
			m.models = msg.models
			m.modelsReady = true
			m.modelErr = ""
			// Restore selection from current modelVal (config value)
			m.selected = -1
			for i, name := range m.models {
				if name == m.modelVal {
					m.selected = i
					break
				}
			}
			if m.selected == -1 && m.modelVal != "" {
				m.selected = len(m.models)
				m.customInput.SetValue(m.modelVal)
				m.customInput.Focus()
			}
			// Clamp cursor if it exceeds the new max
			if maxCur := m.maxCursor(); m.cursor > maxCur {
				m.cursor = maxCur
			}
		}
		return m, nil

	case aiTickMsg:
		if m.testing {
			m.spinnerIdx = (m.spinnerIdx + 1) % len(spinnerFrames)
			return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
				return aiTickMsg{}
			})
		}
		return m, nil

	case AIResultMsg:
		m.testing = false
		if msg.Err != nil {
			m.statusErr = msg.Err.Error()
			return m, nil
		}
		// Connection OK — save and exit
		m.cfg.APIKey = m.inputs[0].Value()
		m.cfg.BaseURL = m.inputs[1].Value()
		m.cfg.Model = m.modelVal
		return m, func() tea.Msg { return SaveAndCloseMsg{} }

	case tea.KeyMsg:
		if m.testing {
			return m, nil
		}
		return m.updateForm(msg)
	}

	return m, nil
}

// ---- Cursor helpers ----

// maxCursor returns the highest valid cursor position.
func (m InitModel) maxCursor() int {
	if !m.modelsReady {
		return 2 // not yet fetched, fetching, or fetch failed — cursor locked to model label
	}
	if len(m.models) > 0 {
		return 3 + len(m.models) // model items + custom
	}
	return 3 // custom row (fetch succeeded but list is empty)
}

func (m InitModel) onModelItem() bool {
	return m.modelsReady && len(m.models) > 0 && m.cursor >= 3 && m.cursor < 3+len(m.models)
}
func (m InitModel) onCustomRow() bool      { return m.cursor == m.maxCursor() }
func (m InitModel) customIsSelected() bool { return m.selected == len(m.models) }

func (m *InitModel) blurCurrent() {
	if m.cursor <= 1 {
		m.inputs[m.cursor].Blur()
	} else if m.onCustomRow() {
		m.customInput.Blur()
	}
}

func (m *InitModel) focusCurrent() {
	if m.cursor <= 1 {
		m.inputs[m.cursor].Focus()
	} else if m.onCustomRow() && m.customIsSelected() {
		m.customInput.Focus()
	}
}

// maybeFetch triggers model list fetch when cursor reaches the model section.
func (m *InitModel) maybeFetch() tea.Cmd {
	if m.cursor < 2 {
		return nil
	}
	key := m.inputs[0].Value()
	url := m.inputs[1].Value()
	if key == "" || url == "" {
		return nil
	}
	if m.fetching {
		return nil
	}
	if m.modelErr != "" && key == m.fetchKey && url == m.fetchURL {
		return nil // already failed with these credentials
	}
	if m.modelsReady && key == m.fetchKey && url == m.fetchURL {
		return nil
	}
	m.fetching = true
	m.modelsReady = false
	m.modelErr = ""
	m.models = nil
	m.fetchKey = key
	m.fetchURL = url
	return fetchModels(key, url)
}

// ---- Form update ----

func (m InitModel) updateForm(msg tea.KeyMsg) (InitModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return TransitionToAIMsg{} }

	case "up":
		if m.cursor > 0 {
			m.blurCurrent()
			m.cursor--
			m.focusCurrent()
		}
		return m, m.maybeFetch()

	case "down", "tab":
		if m.cursor < m.maxCursor() {
			m.blurCurrent()
			m.cursor++
			m.focusCurrent()
		}
		return m, m.maybeFetch()

	case "enter":
		m.statusErr = ""
		return m.saveAndTest()

	case " ":
		if m.onModelItem() {
			m.selected = m.cursor - 3
			return m, nil
		}
		if m.onCustomRow() {
			if !m.customIsSelected() {
				m.customInput.SetValue("")
				m.customInput.Focus()
				m.selected = len(m.models)
				return m, nil
			}
			// Already selected — let Space through to textinput
		} else {
			return m, nil
		}
	}

	// Delegate typed keys to active textinput
	if m.cursor <= 1 {
		var cmd tea.Cmd
		m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
		m.statusErr = ""
		return m, cmd
	}
	if m.onCustomRow() && m.customIsSelected() {
		var cmd tea.Cmd
		m.customInput, cmd = m.customInput.Update(msg)
		m.statusErr = ""
		return m, cmd
	}

	return m, nil
}

// ---- Save & test ----

func (m *InitModel) saveAndTest() (InitModel, tea.Cmd) {
	key := strings.TrimSpace(m.inputs[0].Value())
	url := strings.TrimSpace(m.inputs[1].Value())

	// Resolve model: explicit selection first, fall back to configured value
	model := ""
	if m.customIsSelected() {
		model = strings.TrimSpace(m.customInput.Value())
	} else if m.selected >= 0 && m.selected < len(m.models) {
		model = m.models[m.selected]
	} else {
		model = strings.TrimSpace(m.modelVal)
	}

	if key == "" {
		m.statusErr = i18n.T("ai.err_api_key_empty")
		return *m, nil
	}
	if url == "" {
		m.statusErr = i18n.T("ai.err_base_url_empty")
		return *m, nil
	}
	if model == "" {
		m.statusErr = i18n.T("ai.err_model_empty")
		return *m, nil
	}

	// Persist selection into modelVal for display and save
	m.modelVal = model
	m.statusErr = ""
	m.testing = true
	m.spinnerIdx = 0

	cfg := config.AIConfig{
		APIKey:  key,
		BaseURL: url,
		Model:   model,
	}
	return *m, tea.Batch(
		func() tea.Msg {
			ctx := context.Background()
			_, err := CallAI(ctx, "You are a helpful assistant. Respond in JSON format.", `Say hello. Respond with a JSON object like {"message":"hello"}.`, cfg)
			if err != nil {
				return AIResultMsg{Err: err}
			}
			return AIResultMsg{Response: "ok"}
		},
		tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return aiTickMsg{}
		}),
	)
}

// ---- View ----

// rowCursor returns a fixed-width cursor indicator (always 3 visible chars).
func rowCursor(active bool) string {
	if active {
		return aiWordStyle.Render("> ") + " "
	}
	return "   "
}

// radioMarker returns a fixed-width selection marker (always 5 visible chars).
func radioMarker(selected bool) string {
	if selected {
		return aiLabelStyle.Render("(*) ") + " "
	}
	return "( )  "
}

func (m InitModel) View() string {
	if !m.ready {
		return i18n.T("common.initializing")
	}

	var b strings.Builder
	b.WriteString(aiTitleStyle.Render(i18n.T("ai.init_title")))
	b.WriteString("\n\n")

	// Row 0: API Key
	b.WriteString(rowCursor(m.cursor == 0))
	b.WriteString(aiLabelStyle.Render(i18n.T("ai.init_label_key")))
	b.WriteString("\n   ")
	b.WriteString(m.inputs[0].View())
	b.WriteString("\n\n")

	// Row 1: Base URL
	b.WriteString(rowCursor(m.cursor == 1))
	b.WriteString(aiLabelStyle.Render(i18n.T("ai.init_label_url")))
	b.WriteString("\n   ")
	b.WriteString(m.inputs[1].View())
	b.WriteString("\n\n")

	// Row 2: Model label (no value appended)
	b.WriteString(rowCursor(m.cursor == 2))
	b.WriteString(aiLabelStyle.Render(i18n.T("ai.init_label_model")))
	b.WriteString("\n")

	// Model list area (inline, only shown after successful fetch)
	if m.fetching {
		b.WriteString("   ")
		b.WriteString(aiDimStyle.Render(i18n.T("ai.loading_models")))
		b.WriteString("\n")
	} else if m.modelErr != "" {
		b.WriteString("   ")
		b.WriteString(aiErrStyle.Render(m.modelErr))
		b.WriteString("\n")
	} else if !m.modelsReady {
		b.WriteString("   ")
		b.WriteString(aiDimStyle.Render(i18n.T("ai.model_pending")))
		b.WriteString("\n")
	} else {
		if len(m.models) > 0 {
			for i, name := range m.models {
				b.WriteString("   ")
				b.WriteString(rowCursor(m.cursor == 3+i))
				b.WriteString(radioMarker(m.selected == i))
				b.WriteString(name)
				b.WriteString("\n")
			}
		} else {
			b.WriteString("   ")
			b.WriteString(aiDimStyle.Render(i18n.T("ai.no_models")))
			b.WriteString("\n")
		}
		// Custom row (only after successful fetch)
		b.WriteString("   ")
		b.WriteString(rowCursor(m.cursor == m.maxCursor()))
		b.WriteString(radioMarker(m.selected == len(m.models)))
		b.WriteString(m.customInput.View())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.testing {
		b.WriteString(aiDimStyle.Render(
			fmt.Sprintf(" %s %s", spinnerFrames[m.spinnerIdx], i18n.T("ai.testing_connection"))))
	} else if m.statusErr != "" {
		b.WriteString(aiErrStyle.Render(m.statusErr))
	} else {
		b.WriteString(aiDimStyle.Render(i18n.T("ai.init_footer_line")))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

type SaveAndCloseMsg struct{}

type modelListMsg struct {
	models []string
	err    error
}

func fetchModels(apiKey, baseURL string) tea.Cmd {
	return func() tea.Msg {
		cfg := openai.DefaultConfig(apiKey)
		cfg.BaseURL = baseURL
		client := openai.NewClientWithConfig(cfg)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		list, err := client.ListModels(ctx)
		if err != nil {
			return modelListMsg{err: err}
		}

		var names []string
		for _, m := range list.Models {
			names = append(names, m.ID)
		}
		sort.Strings(names)
		return modelListMsg{models: names}
	}
}
