package repl

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const maxHistory = 100

// Base is a shared foundation for text-input + viewport REPL sub-models.
// Both the dictionary search REPL and the AI assistant REPL embed this.
type Base struct {
	Input    textinput.Model
	Viewport viewport.Model
	Width    int
	Height   int
	Ready    bool

	statusMsg string
	statusSeq int

	history      []string
	historyIdx   int
	historySaved string
}

// NewBase creates a new Base with the given text input and viewport.
func NewBase(ti textinput.Model, vp viewport.Model) Base {
	return Base{
		Input:      ti,
		Viewport:   vp,
		historyIdx: -1,
	}
}

// Init implements tea.Model.
func (b Base) Init() tea.Cmd {
	return textinput.Blink
}

// SetStatus sets a status message and returns a command that clears it after 5 seconds.
func (b *Base) SetStatus(msg string) tea.Cmd {
	b.statusMsg = msg
	b.statusSeq++
	seq := b.statusSeq
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return ClearStatusMsg{Seq: seq}
	})
}

// StatusMsg returns the current status message.
func (b Base) StatusMsg() string {
	return b.statusMsg
}

// HandleClearStatusMsg clears the status message if the sequence number matches.
func (b *Base) HandleClearStatusMsg(msg ClearStatusMsg) {
	if msg.Seq == b.statusSeq {
		b.statusMsg = ""
	}
}

// HandleWindowSizeMsg updates dimensions from a WindowSizeMsg.
func (b *Base) HandleWindowSizeMsg(msg tea.WindowSizeMsg) {
	b.Width = msg.Width
	b.Height = msg.Height
	b.Ready = true
	b.Input.Width = max(1, msg.Width-4)
	b.Viewport.Width = max(1, msg.Width-2)
	b.Viewport.Height = max(1, msg.Height-7)
}

// AddHistory appends a query to the history ring buffer.
func (b *Base) AddHistory(query string) {
	if query == "" {
		return
	}
	if len(b.history) > 0 && b.history[len(b.history)-1] == query {
		return
	}
	b.history = append(b.history, query)
	if len(b.history) > maxHistory {
		b.history = b.history[len(b.history)-maxHistory:]
	}
	b.historyIdx = -1
}

// HistoryUp moves backward through the history buffer.
func (b *Base) HistoryUp() {
	if len(b.history) == 0 {
		return
	}
	if b.historyIdx == -1 {
		b.historySaved = b.Input.Value()
		b.historyIdx = len(b.history) - 1
	} else if b.historyIdx > 0 {
		b.historyIdx--
	}
	b.Input.SetValue(b.history[b.historyIdx])
	b.Input.CursorEnd()
}

// HistoryDown moves forward through the history buffer.
func (b *Base) HistoryDown() {
	if b.historyIdx == -1 {
		return
	}
	if b.historyIdx < len(b.history)-1 {
		b.historyIdx++
		b.Input.SetValue(b.history[b.historyIdx])
		b.Input.CursorEnd()
	} else {
		b.historyIdx = -1
		b.Input.SetValue(b.historySaved)
		b.historySaved = ""
	}
}

// ResetHistoryIfChanged exits history navigation mode if the input no longer
// matches the current history entry (i.e. the user typed or deleted a character).
func (b *Base) ResetHistoryIfChanged(newQuery string) {
	if b.historyIdx >= 0 && b.historyIdx < len(b.history) &&
		b.history[b.historyIdx] != newQuery {
		b.historyIdx = -1
	}
}
