package repl

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

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
}

// NewBase creates a new Base with the given text input and viewport.
func NewBase(ti textinput.Model, vp viewport.Model) Base {
	return Base{
		Input:    ti,
		Viewport: vp,
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
