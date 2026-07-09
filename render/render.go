// Package render maps each A2UI component type to a Model that knows how to
// draw it. Every renderer here is a stub: Init does nothing, Update is a
// no-op, and View returns a "[a2tea: <Kind>]" placeholder.
//
// The contract worth preserving even at the stub stage:
//   - For(c) returns the right Model for any component.Component.
//   - Every renderer satisfies Model, the child-component contract a host
//     application (crush) needs to embed it inside a larger layout.
//
// Composition contract. Renderers are designed to be embedded as children of
// a larger TUI, not to be the root of their own program:
//   - A renderer NEVER calls tea.Quit. Quitting is the host's decision; a
//     stray keystroke inside an embedded card must not tear down the whole
//     application.
//   - A renderer lays itself out inside a parent-allocated region: the host
//     calls SetSize(w, h); the renderer does not assume it owns the terminal.
//   - The host routes key events to at most one focused child. Focus()/Blur()
//     grant and revoke that focus; Focused() reports it.
//   - Interaction results are emitted as event package tea.Msg values, not by
//     side effect.
//
// To run a single renderer as its own program (examples, manual testing), wrap
// it with a2tea.Standalone, which owns quit and forwards window size.
package render

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
)

// Model is the contract every a2tea renderer satisfies. It extends tea.Model
// with the operations a parent component needs to compose a child: size
// allocation and focus routing. A renderer implementing Model can be dropped
// into a host TUI (such as crush) that allocates regions and routes focus,
// without the "standalone program" assumptions that would otherwise force a
// rewrite.
type Model interface {
	tea.Model
	// SetSize tells the renderer the region the host has allocated to it.
	SetSize(width, height int)
	// Focus grants keyboard focus to the renderer and returns any command
	// it needs to run as a result (e.g. starting a cursor blink).
	Focus() tea.Cmd
	// Blur revokes keyboard focus.
	Blur()
	// Focused reports whether the renderer currently holds focus.
	Focused() bool
}

// For returns the Model responsible for rendering c. It is the single
// dispatch point from a2tea.Render into this package and is intentionally a
// thin switch — keep concrete renderer construction in the model's own
// constructor, not in this function.
//
// Every component.Component kind MUST have a case here; the completeness test
// in this package fails the build if a new kind is added without one.
func For(c component.Component) (Model, error) {
	switch v := c.(type) {
	case component.Card:
		return NewCard(v), nil
	case component.Form:
		return NewForm(v), nil
	case component.Input:
		return NewInput(v), nil
	case component.Choice:
		return NewChoice(v), nil
	case component.Progress:
		return NewProgress(v), nil
	case component.Markdown:
		return NewMarkdown(v), nil
	case component.Stream:
		return NewStream(v), nil
	default:
		return nil, fmt.Errorf("a2tea/render: no renderer for kind %q", c.Kind())
	}
}

// placeholderView is the shared View() body for every stub renderer. Keeping
// it centralized means there is exactly one place to flip the "real
// rendering" switch on later.
func placeholderView(kind string) tea.View {
	return tea.NewView(fmt.Sprintf("[a2tea: %s]", kind))
}

// base carries the size and focus state shared by every stub renderer so each
// one does not reimplement the SetSize/Focus/Blur/Focused boilerplate. Real
// renderers will read width/height when they lay out and focused when they
// decide whether to handle key input.
type base struct {
	width, height int
	focused       bool
}

// SetSize implements Model.
func (b *base) SetSize(width, height int) { b.width, b.height = width, height }

// Focus implements Model.
func (b *base) Focus() tea.Cmd { b.focused = true; return nil }

// Blur implements Model.
func (b *base) Blur() { b.focused = false }

// Focused implements Model.
func (b *base) Focused() bool { return b.focused }
