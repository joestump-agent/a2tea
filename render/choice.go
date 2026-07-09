package render

import (
	tea "charm.land/bubbletea/v2"

	"github.com/joestump-agent/a2tea/component"
)

// ChoiceModel renders a component.Choice.
//
// TODO(a2tea): back this with bubbles/list or huh.Select; emit
// event.ChoiceSelected when a value is chosen.
type ChoiceModel struct {
	base
	c component.Choice
}

// NewChoice builds a ChoiceModel for the given choice.
func NewChoice(c component.Choice) *ChoiceModel { return &ChoiceModel{c: c} }

// Init implements tea.Model.
func (m *ChoiceModel) Init() tea.Cmd { return nil }

// Update implements tea.Model. It is a no-op stub and, per the composition
// contract, never quits — the host owns program exit.
func (m *ChoiceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

// View implements tea.Model.
func (m *ChoiceModel) View() tea.View { return placeholderView(component.KindChoice) }
