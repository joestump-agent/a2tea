// Package a2tea bridges A2UI (https://a2ui.org) JSON messages into Bubble Tea
// models so AI agents can drive rich terminal UI from a structured message
// format.
//
// This is an early revision: the public API is fixed, but the rendering and
// event roundtrip logic is intentionally stubbed. See the Roadmap section of
// README.md for what is not yet implemented. The wire format is provisional
// and A2UI-inspired — see docs/wire-format.md.
package a2tea

import (
	"encoding/json"

	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
	"github.com/joestump/a2tea/render"
)

// Render parses a raw A2UI JSON document and returns a Bubble Tea model that
// will render the described UI when run inside a tea.Program.
//
// The returned model is a render.Model: an embeddable child component, not a
// standalone program. It does not handle quit or own the full terminal. To
// run one directly (examples, manual testing), wrap it with Standalone.
//
// Render surfaces every failure as a non-nil error so a host application can
// tell "the agent sent a bad document" from "the renderer isn't implemented
// yet" and fall back to plain-text rendering:
//   - an empty document returns component.ErrEmptyDocument,
//   - an unknown kind returns component.ErrUnknownKind,
//   - an invalid document returns an error wrapping component.ErrValidation,
//   - malformed JSON returns the underlying decode error.
//
// All are matchable with errors.Is.
func Render(raw json.RawMessage) (tea.Model, error) {
	c, err := component.Unmarshal(raw)
	if err != nil {
		return nil, err
	}
	m, err := render.For(c)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Standalone wraps a renderer so it can run as its own tea.Program. It owns
// the two responsibilities a renderer deliberately does not: it quits on
// Ctrl+C, q, or Esc, and it forwards terminal-size changes to the child via
// SetSize. Hosts that embed a renderer inside a larger TUI do NOT use this —
// they own quit and lay out the child themselves. Standalone exists for
// examples and manual testing of a single component.
func Standalone(child tea.Model) tea.Model {
	return standaloneModel{child: child}
}

// standaloneModel is the root wrapper returned by Standalone.
type standaloneModel struct {
	child tea.Model
}

// sizer is the subset of render.Model that Standalone needs to lay the child
// out. Accepting the interface (rather than render.Model) keeps Standalone
// usable with any size-aware model.
type sizer interface {
	SetSize(width, height int)
}

func (m standaloneModel) Init() tea.Cmd { return m.child.Init() }

func (m standaloneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if s, ok := m.child.(sizer); ok {
			s.SetSize(msg.Width, msg.Height)
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}
	}
	child, cmd := m.child.Update(msg)
	m.child = child
	return m, cmd
}

func (m standaloneModel) View() tea.View { return m.child.View() }
