// Package a2tea bridges A2UI (https://a2ui.org) into Bubble Tea: it parses the
// A2UI messages an agent emits — interleaved with conversational text in an
// LLM response — and renders the described surfaces as Bubble Tea models so a
// host TUI (crush) can draw them.
//
// A2UI parsing lives here on purpose. A host should not hand-roll detection of
// A2UI payloads in model output; it calls Scan (or Contains) and gets back the
// text and the typed A2UI messages, using the real A2UI wire format
// (github.com/tmc/a2ui): JSON wrapped in <a2ui-json> tags or bare A2UI JSON.
//
// The renderers are still visual stubs (see the render package), but the parse
// path and the component catalog are the real A2UI v0.9 protocol.
package a2tea

import (
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	a2ui "github.com/tmc/a2ui"
	"github.com/tmc/a2ui/a2uistream"

	"github.com/joestump-agent/a2tea/render"
)

// ErrNoRenderableSurface is returned by Render when the given messages contain
// nothing to draw (no updateComponents).
var ErrNoRenderableSurface = errors.New("a2tea: no renderable surface in messages")

// Part is a segment of an LLM response: conversational text and the A2UI
// messages that immediately followed it. Either field may be empty — a part is
// text-only, messages-only, or both (the text that preceded a JSON block plus
// the messages extracted from it).
type Part struct {
	Text     string
	Messages []a2ui.ServerMessage
}

// Contains reports whether s contains at least one complete A2UI message block,
// so a host can cheaply decide whether to take the Scan path at all.
func Contains(s string) bool {
	return a2uistream.HasParts(s)
}

// Scan splits an LLM response into ordered parts of text and A2UI messages. It
// is the entry point a host uses instead of hand-rolling detection: feed it the
// assistant's reply, render each part's Text as prose and hand each part's
// Messages to Render.
func Scan(s string) ([]Part, error) {
	raw, err := a2uistream.ParseAndValidate(s, nil)
	if err != nil {
		return nil, fmt.Errorf("a2tea: scan A2UI: %w", err)
	}
	parts := make([]Part, 0, len(raw))
	for _, p := range raw {
		parts = append(parts, Part{Text: p.Text, Messages: p.Messages})
	}
	return parts, nil
}

// Render applies a sequence of A2UI server messages in order to build surface
// state and returns an embeddable Bubble Tea model that draws the resulting
// surface. The latest updateComponents wins; surface compositing, data-model
// updates, and deletions are not yet applied (see the render package).
//
// It returns ErrNoRenderableSurface when the messages describe no components to
// draw, so a host can fall back to plain text.
//
// The returned model is a render.Model — an embeddable child component that
// does not handle quit. To run one directly, wrap it with Standalone.
func Render(msgs []a2ui.ServerMessage) (tea.Model, error) {
	var components []a2ui.Component
	found := false
	for _, m := range msgs {
		if m.UpdateComponents != nil {
			components = m.UpdateComponents.Components
			found = true
		}
	}
	if !found {
		return nil, ErrNoRenderableSurface
	}
	return render.NewSurface(components), nil
}

// Standalone wraps a renderer so it can run as its own tea.Program. It owns the
// two responsibilities a renderer deliberately does not: it quits on Ctrl+C, q,
// or Esc, and it forwards terminal-size changes to the child via SetSize. Hosts
// that embed a renderer inside a larger TUI do NOT use this — they own quit and
// lay out the child themselves. Standalone exists for examples and manual
// testing of a single surface.
func Standalone(child tea.Model) tea.Model {
	return standaloneModel{child: child}
}

// standaloneModel is the root wrapper returned by Standalone.
type standaloneModel struct {
	child tea.Model
}

// sizer is the subset of render.Model that Standalone needs to lay the child
// out.
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
