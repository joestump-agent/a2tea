package render

import (
	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
)

// ChoiceModel renders a component.Choice.
//
// TODO(a2tea): back this with bubbles/list or huh.Select; emit
// event.ChoiceSelected when a value is chosen.
type ChoiceModel struct {
	c component.Choice
}

// NewChoice builds a ChoiceModel for the given choice.
func NewChoice(c component.Choice) ChoiceModel { return ChoiceModel{c: c} }

// Init implements tea.Model.
func (m ChoiceModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m ChoiceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, quitOnKey(msg)
}

// View implements tea.Model.
func (m ChoiceModel) View() tea.View { return placeholderView(component.KindChoice) }
