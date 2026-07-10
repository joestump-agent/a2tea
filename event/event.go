// Package event defines the outbound messages a2tea emits when a user
// interacts with a rendered A2UI surface. Each event is a tea.Msg so it flows
// through the standard bubbletea Update loop, and the consuming agent picks
// them up from there.
//
// Status. These types predate a2tea's adoption of the real A2UI protocol
// (github.com/tmc/a2ui) and no renderer emits them yet — the renderers are
// still visual stubs. They are a provisional host-facing vocabulary and will
// be re-grounded in A2UI catalog terms when the interaction round-trip is
// built: A2UI models interactions as a Button's Action / a ClientMessage, has
// TextField (not "input") and ChoicePicker (not "choice"), and has no Form
// component at all (see the note on FormSubmitted). Treat the shapes here as
// not yet stable.
//
// Source context. Every event embeds Source, which carries the IDs a consumer
// needs to tell interactions apart when more than one component (or the same
// component reused across a surface) is on screen. Without it, a bare
// element-local ID like "ok" is ambiguous. The originating renderer fills
// Source in when it dispatches the event.
package event

// Source identifies where an event originated. It is embedded in every event
// type so consumers can route an interaction back to the specific A2UI
// component and the surface that produced it.
type Source struct {
	// ComponentID is the a2ui.Component ID of the interactive element (or the
	// container it belongs to). It is distinct from any element-local ID an
	// event carries.
	ComponentID string
	// SurfaceID is the A2UI surface the component was rendered on (the
	// surfaceId from createSurface / updateComponents).
	SurfaceID string
}

// ButtonClicked is emitted when the user activates an A2UI Button.
//
// TODO(a2tea): dispatch this from the renderer with Source set; map it to the
// button's A2UI Action.
type ButtonClicked struct {
	Source
	// ID is the a2ui.Component ID of the activated Button.
	ID string
}

// InputSubmitted is emitted when the user confirms an A2UI TextField value.
//
// TODO(a2tea): dispatch this from the renderer with Source set.
type InputSubmitted struct {
	Source
	// ID is the a2ui.Component ID of the TextField.
	ID string
	// Value is the final string value the user submitted.
	Value string
}

// ChoiceSelected is emitted when the user picks from an A2UI ChoicePicker.
//
// A2UI's ChoicePicker is natively multi-value (its value is a string list), so
// when this event is wired it will likely carry a []string rather than a single
// Value — one of the re-grounding tasks noted in the package doc. It is kept
// single-valued here only until then.
//
// TODO(a2tea): dispatch this from the renderer with Source set; reconcile the
// value shape with ChoicePicker's list value.
type ChoiceSelected struct {
	Source
	// ID is the a2ui.Component ID of the ChoicePicker.
	ID string
	// Value is the selected option's value (NOT its label).
	Value string
}

// FormSubmitted is emitted once when the user submits a group of fields as a
// unit, carrying every field's value keyed by field ID — so a consumer gets one
// atomic, correlated event instead of collecting N loose InputSubmitted
// messages and correlating them to a single submit action.
//
// Note: A2UI v0.9 has no Form component. A "form submit" corresponds to a
// Button Action gathering the values of nearby TextField/ChoicePicker
// components; this type will be re-grounded on that shape when wired.
//
// TODO(a2tea): dispatch this from the renderer on the submitting Button's
// Action.
type FormSubmitted struct {
	Source
	// FormID identifies the submitted group (mirrored in Source.ComponentID;
	// kept as a self-documenting primary name).
	FormID string
	// SubmitID is the a2ui.Component ID of the Button that triggered the
	// submission, distinguishing multiple actions in the same group (e.g.
	// "save" vs. "save and close").
	SubmitID string
	// Values maps each field's a2ui.Component ID to its final string value.
	Values map[string]string
}
