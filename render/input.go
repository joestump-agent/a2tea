package render

import (
	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
)

// InputModel renders a component.Input.
//
// TODO(a2tea): back this with bubbles/textinput, surface placeholder/value,
// and emit event.InputSubmitted on Enter.
type InputModel struct {
	base
	c component.Input
}

// NewInput builds an InputModel for the given input.
func NewInput(c component.Input) *InputModel { return &InputModel{c: c} }

// Init implements tea.Model.
func (m *InputModel) Init() tea.Cmd { return nil }

// Update implements tea.Model. It is a no-op stub and, per the composition
// contract, never quits — the host owns program exit.
func (m *InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

// View implements tea.Model.
func (m *InputModel) View() tea.View { return placeholderView(component.KindInput) }
