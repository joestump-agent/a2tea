// Package event defines the outbound messages a2tea emits when a user
// interacts with a rendered component. Each event is a tea.Msg so it flows
// through the standard bubbletea Update loop, and the consuming agent picks
// them up from there.
//
// These types are stable from day one even though the renderers don't emit
// them yet — fixing the wire format early lets agent-side code be written
// against the real shape before the renderers catch up.
//
// Source context. Every event embeds Source, which carries the IDs a consumer
// needs to tell interactions apart when more than one component (or the same
// reusable document rendered twice) is on screen. Without it, a bare
// component-local ID like "ok" is ambiguous. The originating renderer fills
// Source in when it dispatches the event; a standalone single-component
// program can leave the fields empty.
package event

// Source identifies where an event originated. It is embedded in every event
// type so consumers can route an interaction back to the specific component
// (and, per A2UI, the surface) that produced it.
type Source struct {
	// ComponentID is the ID of the component that owns the interactive
	// element — e.g. the Card or Form that contains the clicked button.
	// It is distinct from the element-local ID carried by each event.
	ComponentID string
	// SurfaceID is the A2UI surface the component was rendered on. It is
	// zero until surfaces land (see docs/wire-format.md) but is fixed in
	// the type now so adding it later is not a breaking change.
	SurfaceID string
}

// ButtonClicked is emitted when the user activates a button (e.g. a Card
// action button or a Form submit button).
//
// TODO(a2tea): wire CardModel and FormModel to dispatch this with Source set.
type ButtonClicked struct {
	Source
	// ID is the button's element-local identifier, matching
	// component.Button.ID on the originating component.
	ID string
}

// InputSubmitted is emitted when the user confirms a standalone input value
// by pressing Enter. Form submission does NOT emit one of these per field —
// it emits a single FormSubmitted instead (see below).
//
// TODO(a2tea): wire InputModel to dispatch this with Source set.
type InputSubmitted struct {
	Source
	// ID is the input's element-local identifier.
	ID string
	// Value is the final string value the user submitted.
	Value string
}

// ChoiceSelected is emitted when the user picks a value from a single-select
// Choice.
//
// Multi-select decision: ChoiceSelected stays single-valued. When a
// multi-select component lands (likely, per the A2UI catalog), it gets its
// own event type carrying a []string rather than widening this Value field —
// widening would silently break every existing consumer that reads a single
// string. Keeping the two distinct also lets a consumer pattern-match on
// "single vs. multi" by type.
//
// TODO(a2tea): wire ChoiceModel to dispatch this with Source set.
type ChoiceSelected struct {
	Source
	// ID is the choice's element-local identifier.
	ID string
	// Value is the selected option's value (NOT its label).
	Value string
}

// FormSubmitted is emitted once when the user submits a Form. It carries every
// field's value keyed by field ID, so a consumer gets one atomic, correlated
// event instead of having to collect N loose InputSubmitted messages, know
// when the set is complete, and correlate them back to a single submit action.
//
// TODO(a2tea): wire FormModel to dispatch this on submit.
type FormSubmitted struct {
	Source
	// FormID is the submitted form's identifier (component.Form.ID). It is
	// also mirrored in Source.ComponentID; FormID is kept as the primary,
	// self-documenting name for a form submission.
	FormID string
	// SubmitID is the ID of the submit button that triggered the
	// submission, distinguishing multiple actions on the same form (e.g.
	// "save" vs. "save and close").
	SubmitID string
	// Values maps each field's element-local ID to its final string value.
	// Multi-value fields are out of scope until a multi-select field lands
	// alongside the multi-select event described on ChoiceSelected.
	Values map[string]string
}
