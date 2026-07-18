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
// Rendering is real for the core catalog (see the render package): styled
// text, bordered cards, container layout, and focusable buttons that emit
// event.ButtonClicked. The remaining gaps — data-model bindings, editable
// inputs, surface lifecycle — are listed in the render package doc and
// docs/wire-format.md.
package a2tea

import (
	"errors"
	"fmt"
	"strings"

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

// Contains reports whether s contains at least one A2UI message, so a host
// can cheaply decide whether to take the Scan path at all. It detects both
// <a2ui-json>-tagged blocks and bare A2UI JSON objects — the same forms Scan
// parses, so Contains and Scan always agree.
//
// The check is two-stage: a cheap literal probe for A2UI's message keys,
// then — only on a probe hit — a real parse to confirm. Ordinary prose never
// pays the parse cost, and prose that merely mentions a key without valid
// A2UI JSON does not false-positive.
func Contains(s string) bool {
	if a2uistream.HasParts(s) {
		return true
	}
	if !mentionsA2UIKey(s) {
		return false
	}
	parts, err := a2uistream.ParseAndValidate(s, nil)
	if err != nil {
		return false
	}
	for _, p := range parts {
		if len(p.Messages) > 0 {
			return true
		}
	}
	return false
}

// a2uiMessageKeys are the quoted JSON keys naming the A2UI v0.9 server
// message types. A bare A2UI JSON object necessarily contains one of these,
// so a reply without any of them cannot contain an untagged message.
var a2uiMessageKeys = []string{
	`"createSurface"`,
	`"updateComponents"`,
	`"updateDataModel"`,
	`"deleteSurface"`,
}

// mentionsA2UIKey is the cheap first-stage probe used by Contains.
func mentionsA2UIKey(s string) bool {
	for _, k := range a2uiMessageKeys {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
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
// surface. Messages are composited: updateComponents merges components by ID
// (an update to one component leaves siblings intact), updateDataModel sets
// bound values that resolve on the next render, and deleteSurface removes the
// targeted surface.
//
// Optional functional options (e.g. render.WithStyles) may be appended after
// the messages argument; calling Render without any options produces
// byte-for-byte identical output to the pre-options behaviour.
//
// It returns ErrNoRenderableSurface when the messages describe no components
// to draw (or when the surface was deleted), so a host can fall back to
// plain text.
//
// The returned model is a render.Model — an embeddable child component that
// does not handle quit. To run one directly, wrap it with Standalone.
func Render(msgs []a2ui.ServerMessage, opts ...render.Option) (tea.Model, error) {
	var surfaceID string
	var firstComponents []a2ui.Component
	found := false

	// Find the first updateComponents to establish the surface.
	for _, m := range msgs {
		if m.UpdateComponents != nil {
			surfaceID = m.UpdateComponents.SurfaceID
			firstComponents = m.UpdateComponents.Components
			found = true
			break
		}
	}

	if !found {
		return nil, ErrNoRenderableSurface
	}

	s := render.NewSurface(surfaceID, firstComponents, opts...)

	// Apply all subsequent messages (including further updateComponents for
	// compositing, data-model updates, and deleteSurface).
	alive := s.Apply(msgs)
	if !alive {
		return nil, ErrNoRenderableSurface
	}
	return s, nil
}

// Standalone wraps a renderer so it can run as its own tea.Program. It owns the
// two responsibilities a renderer deliberately does not: it quits on Ctrl+C or
// Esc (unless the child reports an open modal — then Esc closes the modal; and
// on q, unless the child reports it is editing a text field — then q is
// typed), and it forwards terminal-size changes to the child via SetSize. Hosts
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

// focuser is the subset of render.Model that Standalone uses to grant its
// single child keyboard focus at startup, so Tab/Enter interaction works
// without a host focus router.
type focuser interface {
	Focus() tea.Cmd
}

// editingTexter is the optional probe Standalone uses to decide whether "q"
// is a quit command or text input. render.Surface implements it.
type editingTexter interface {
	EditingText() bool
}

// openModaler is the optional probe Standalone uses to decide whether Esc
// is a quit command or a close-the-modal command. render.Surface implements
// it (HasOpenModal); when a modal is open, Esc is forwarded to the child so
// the modal closes instead of the program quitting.
type openModaler interface {
	HasOpenModal() bool
}

func (m standaloneModel) Init() tea.Cmd {
	cmd := m.child.Init()
	if f, ok := m.child.(focuser); ok {
		return tea.Batch(cmd, f.Focus())
	}
	return cmd
}

func (m standaloneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if s, ok := m.child.(sizer); ok {
			s.SetSize(msg.Width, msg.Height)
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Esc quits only when no modal is open: an open modal consumes
			// Esc as its close command, so forward the key to the child.
			if e, ok := m.child.(openModaler); ok && e.HasOpenModal() {
				break
			}
			return m, tea.Quit
		case "q":
			// "q" quits only when it isn't text input: if the child reports
			// it is editing a text field, forward the key so the user can
			// type the letter q.
			if e, ok := m.child.(editingTexter); ok && e.EditingText() {
				break
			}
			return m, tea.Quit
		}
	}
	child, cmd := m.child.Update(msg)
	m.child = child
	return m, cmd
}

func (m standaloneModel) View() tea.View { return m.child.View() }
