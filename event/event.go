// Package event defines the outbound messages a2tea emits when a user
// interacts with a rendered component. Each event is a tea.Msg so it flows
// through the standard bubbletea Update loop, and the consuming agent picks
// them up from there.
//
// These types are stable from day one even though the renderers don't emit
// them yet — fixing the wire format early lets agent-side code be written
// against the real shape before the renderers catch up.
package event

// ButtonClicked is emitted when the user activates a button (e.g. a Card
// action button or a Form submit button).
//
// TODO(a2tea): wire CardModel and FormModel to dispatch this.
type ButtonClicked struct {
	// ID is the button's component-local identifier, matching
	// component.Button.ID on the originating component.
	ID string
}

// InputSubmitted is emitted when the user confirms an input value, either
// by pressing Enter on a standalone Input or by submitting a containing
// Form.
//
// TODO(a2tea): wire InputModel and FormModel to dispatch this.
type InputSubmitted struct {
	// ID is the input's component-local identifier.
	ID string
	// Value is the final string value the user submitted.
	Value string
}

// ChoiceSelected is emitted when the user picks a value from a Choice.
//
// TODO(a2tea): wire ChoiceModel to dispatch this.
type ChoiceSelected struct {
	// ID is the choice's component-local identifier.
	ID string
	// Value is the selected option's value (NOT its label).
	Value string
}
