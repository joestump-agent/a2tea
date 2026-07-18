// Package event defines the outbound messages a2tea emits when a user
// interacts with a rendered A2UI surface. Each event is a tea.Msg so it flows
// through the standard bubbletea Update loop, and the consuming agent picks
// them up from there.
//
// Status. ButtonClicked is emitted by the surface renderer and now carries the
// button's resolved *a2ui.EventAction (nil for buttons with no server event).
// Alongside it the renderer emits a native a2ui.ClientMessage whose
// ActionEvent.Context is populated from the surface's input component values
// (TextField/DateTimeInput → string, ChoicePicker → []string, CheckBox →
// bool, Slider → float64, keyed by component ID) merged with the action's
// own declared context bindings.
//
// FormSubmitted is deprecated: A2UI v0.9 has no Form component, so a "form
// submit" is just a Button Action whose ActionEvent.Context carries the
// gathered field values — the host reads Context directly. InputSubmitted and
// ChoiceSelected are not emitted yet; they are a provisional host-facing
// vocabulary that will be re-grounded in A2UI catalog terms (TextField, not
// "input"; ChoicePicker, not "choice") when wired. Treat those shapes as not
// yet stable.
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
// event with Source set. The Action field carries the button's resolved
// *a2ui.EventAction when the button has a server-side event — nil for buttons
// with only a client-side FunctionCall or no action at all. The renderer also
// emits a native a2ui.ClientMessage alongside this event so the host can
// round-trip the ActionEvent to the agent without a translation layer.
type ButtonClicked struct {
	Source
	// ID is the a2ui.Component ID of the activated Button.
	ID string
	// Action is the button's resolved server-side event action, or nil when
	// the button has no server event (a FunctionCall-only button or one with
	// no action).
	Action *a2ui.EventAction
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

// Deprecated: A2UI v0.9 has no Form component. A "form submit" is a Button
// Action whose emitted a2ui.ClientMessage carries the gathered field values in
// ActionEvent.Context (TextField/ChoicePicker/CheckBox, keyed by component ID);
// the host reads ActionEvent.Context directly — see ButtonClicked. This type
// is retained for backward compatibility but is never emitted by the renderer.
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
