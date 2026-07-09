package render

import (
	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
)

// CardModel renders a component.Card.
//
// TODO(a2tea): draw a real bordered box with title, body, and button row
// using lipgloss, sized to the host-allocated region. Buttons should be
// focusable and emit event.ButtonClicked (carrying the card's ID as the
// originating ComponentID).
type CardModel struct {
	base
	c component.Card
}

// NewCard builds a CardModel for the given card.
func NewCard(c component.Card) *CardModel { return &CardModel{c: c} }

// Init implements tea.Model.
func (m *CardModel) Init() tea.Cmd { return nil }

// Update implements tea.Model. It is a no-op stub and, per the composition
// contract, never quits — the host owns program exit.
func (m *CardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

// View implements tea.Model.
func (m *CardModel) View() tea.View { return placeholderView(component.KindCard) }
