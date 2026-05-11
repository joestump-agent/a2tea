package render

import (
	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
)

// CardModel renders a component.Card.
//
// TODO(a2tea): draw a real bordered box with title, body, and button row
// using lipgloss. Buttons should be focusable and emit event.ButtonClicked.
type CardModel struct {
	c component.Card
}

// NewCard builds a CardModel for the given card.
func NewCard(c component.Card) CardModel { return CardModel{c: c} }

// Init implements tea.Model.
func (m CardModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m CardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, quitOnKey(msg)
}

// View implements tea.Model.
func (m CardModel) View() tea.View { return placeholderView(component.KindCard) }
