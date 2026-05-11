// Package a2tea bridges A2UI (https://a2ui.org) JSON messages into Bubble Tea
// models so AI agents can drive rich terminal UI from a structured message
// format.
//
// This is the scaffolding revision: the public API is fixed, but the
// rendering and event roundtrip logic is intentionally stubbed. See the
// Roadmap section of README.md for what is not yet implemented.
package a2tea

import (
	"encoding/json"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
	"github.com/joestump/a2tea/render"
)

// Render parses a raw A2UI JSON document and returns a Bubble Tea model that
// will render the described UI when run inside a tea.Program.
//
// TODO(a2tea): implement real JSON -> component.Component decoding, then
// dispatch to the appropriate concrete renderer in the render package. The
// current implementation always returns a notImplementedModel that shows a
// placeholder string, even on parse error, so callers can wire the API in
// before the parser exists.
func Render(raw json.RawMessage) (tea.Model, error) {
	c, err := component.Unmarshal(raw)
	if err != nil {
		// TODO(a2tea): once Unmarshal is real, surface this error to the
		// caller instead of swallowing it into a placeholder model.
		return notImplementedModel{reason: fmt.Sprintf("parse error: %v", err)}, nil
	}
	if c == nil {
		return notImplementedModel{reason: "empty document"}, nil
	}
	m, err := render.For(c)
	if err != nil {
		return notImplementedModel{reason: fmt.Sprintf("no renderer for %q", c.Kind())}, nil
	}
	return m, nil
}

// notImplementedModel is the placeholder model returned while the real
// renderer pipeline is unimplemented. It satisfies tea.Model and exits on any
// key press so the example program is well-behaved under a timeout.
type notImplementedModel struct {
	reason string
}

func (m notImplementedModel) Init() tea.Cmd { return nil }

func (m notImplementedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyPressMsg, tea.QuitMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m notImplementedModel) View() tea.View {
	if m.reason == "" {
		return tea.NewView("[a2tea: not implemented yet — press any key to quit]")
	}
	return tea.NewView(fmt.Sprintf("[a2tea: not implemented yet — %s — press any key to quit]", m.reason))
}
