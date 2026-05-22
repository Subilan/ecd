package ai

import (
	"strings"
	"time"

	"github.com/Subilan/ecd/internal/render"
	"github.com/Subilan/ecd/internal/repl"
	tea "github.com/charmbracelet/bubbletea"
)

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
			m.Base.AddHistory(query)
			m.Input.SetValue("")
			return m, m.executeCommand(query)

		case "esc":
			m.Input.SetValue("")
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

	var cmds []tea.Cmd

	// Always forward to text input
	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	cmds = append(cmds, cmd)

	m.Base.ResetHistoryIfChanged(strings.TrimSpace(m.Input.Value()))

	// Forward to viewport (scroll keys work when !focusInput, or pgup/pgdown/non-key msgs)
	if !m.focusInput {
		m.Viewport, cmd = m.Viewport.Update(msg)
	} else {
		_, cmd = m.Viewport.Update(msg) // still handle non-key msgs
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}
