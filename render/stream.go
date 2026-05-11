package render

import (
	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
)

// StreamModel renders a component.Stream.
//
// TODO(a2tea): support live chunk delivery via tea.Msg, autoscroll, and
// optional glamour rendering when the stream is declared as markdown.
type StreamModel struct {
	c component.Stream
}

// NewStream builds a StreamModel for the given stream.
func NewStream(c component.Stream) StreamModel { return StreamModel{c: c} }

// Init implements tea.Model.
func (m StreamModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m StreamModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, quitOnKey(msg)
}

// View implements tea.Model.
func (m StreamModel) View() tea.View { return placeholderView(component.KindStream) }
