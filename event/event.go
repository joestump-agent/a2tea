// Package event defines the outbound messages a2tea emits when a user
// interacts with a rendered A2UI surface. Each event is a tea.Msg so it flows
// through the standard bubbletea Update loop, and the consuming agent picks
// them up from there.
//
// ButtonClicked carries the activated button's resolved Action so the host can
// round-trip it as a protocol-native a2ui.ClientMessage{Action *ActionEvent}.
// When a button has an EventAction, the emitted ButtonClicked also includes the
// a2ui.ClientMessage. Buttons with only a FunctionCall action (client-side
// functions with no server event) produce a ButtonClicked with nil Action —
// client-side function calls are out of scope for a2tea and are handled by the
// host.
//
// InputSubmitted, ChoiceSelected, and FormSubmitted are not emitted yet. They
// are provisional and will be re-grounded in A2UI catalog terms when wired.
//
// Source context. Every event embeds Source, which carries the IDs a consumer
// needs to tell interactions apart when more than one component (or the same
// component reused across a surface) is on screen. Without it, a bare
// element-local ID like "ok" is ambiguous. The originating renderer fills
// Source in when it dispatches the event.
package event

import a2ui "github.com/tmc/a2ui"

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

// ButtonClicked is emitted when the user activates an A2UI Button: with the
// surface focused, Tab / Shift+Tab cycle buttons and Enter dispatches this
// event with Source set.
//
// When the button has an EventAction, Action is non-nil and ClientMessage
// carries the protocol-native a2ui.ClientMessage{Action *ActionEvent} ready
// for the host to round-trip to the agent. When the button has only a
// FunctionCall action (or no action), Action is nil and ClientMessage is the
// zero value — client-side function calls are handled by the host, not a2tea.
//
// ActionEvent.Timestamp is left empty; the host stamps it before sending to
// the agent so that the renderer's Update path stays deterministic for tests.
type ButtonClicked struct {
	Source
	// ID is the a2ui.Component ID of the activated Button.
	ID string
	// Action is the button's resolved EventAction, or nil for buttons with
	// only a FunctionCall action (or no action at all).
	Action *a2ui.EventAction
	// ClientMessage is the protocol-native message carrying the ActionEvent.
	// It is non-zero only when Action is non-nil. The host can send this
	// directly to the agent without further translation.
	ClientMessage a2ui.ClientMessage
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
