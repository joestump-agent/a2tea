// Package render maps each A2UI component type to a tea.Model that knows how
// to draw it. Every renderer here is a stub: Init does nothing, Update only
// handles quit, and View returns a "[a2tea: <Kind>]" placeholder.
//
// The contract worth preserving even at the stub stage:
//   - For(c) returns the right model for any component.Component.
//   - Every renderer is itself a tea.Model so it composes with bubbletea
//     programs without an adapter layer.
package render

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
)

// For returns the tea.Model responsible for rendering c. It is the single
// dispatch point from a2tea.Render into this package and is intentionally a
// thin switch — keep concrete renderer construction in the model's own
// constructor below, not in this function.
func For(c component.Component) (tea.Model, error) {
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

// quitOnKey is the shared Update() body for stub renderers — any key press
// or quit message exits the program. This keeps the example runnable under a
// timeout without each renderer reimplementing the same boilerplate.
func quitOnKey(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyPressMsg, tea.QuitMsg:
		return tea.Quit
	}
	return nil
}
