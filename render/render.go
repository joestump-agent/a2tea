// Package render turns an A2UI surface — a flat set of components that
// reference their children by ID — into a Bubble Tea model that draws the
// component tree with lipgloss.
//
// Rendering is real for the core catalog: Text (with variants), Card, Column,
// Row, List, Divider, Button, and read-only visuals for the input components
// (TextField, CheckBox, ChoicePicker, Slider, DateTimeInput). Media components
// (Image, Icon, Video, AudioPlayer) draw compact placeholders, as do Tabs and
// Modal. DynamicString data bindings render as placeholders until the data
// model lands.
//
// Interaction: Buttons are focusable. When the host grants the surface focus,
// Tab / Shift+Tab cycle the buttons and Enter activates the focused button.
// Activation emits event.ButtonClicked (carrying the resolved *a2ui.EventAction)
// and, when the button has a server-side Event action, a protocol-native
// a2ui.ClientMessage whose ActionEvent carries Name, SurfaceID, and
// SourceComponentID. FunctionCall-only buttons emit no ClientMessage. Editing
// input components is not wired yet.
//
// Composition contract. A renderer is designed to be embedded as a child of a
// larger TUI (crush), not to be the root of its own program:
//   - It NEVER calls tea.Quit — quitting is the host's decision.
//   - It lays itself out inside a host-allocated region via SetSize.
//   - The host routes key events to at most one focused child via
//     Focus/Blur/Focused.
//
// Wrap a renderer with a2tea.Standalone to run it as its own program.
package render

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/event"
)

// Model is the contract every a2tea renderer satisfies. It extends tea.Model
// with the operations a parent component needs to compose a child: size
// allocation and focus routing.
type Model interface {
	tea.Model
	// SetSize tells the renderer the region the host has allocated to it.
	SetSize(width, height int)
	// Focus grants keyboard focus to the renderer.
	Focus() tea.Cmd
	// Blur revokes keyboard focus.
	Blur()
	// Focused reports whether the renderer currently holds focus.
	Focused() bool
}

// Option configures a Surface at construction time via NewSurface options.
type Option func(*Surface)

// WithStyles overrides the Surface's chrome styles with the provided set.
// Call DefaultStyles, override specific fields, and pass the result:
//
//	st := render.DefaultStyles()
//	st.Heading = st.Heading.Foreground(lipgloss.Color("99"))
//	surf := render.NewSurface(id, comps, render.WithStyles(st))
func WithStyles(st Styles) Option {
	return func(s *Surface) { s.styles = st }
}

// compactWidthThreshold is the default width below which the surface switches
// to compact rendering: cards drop borders, rows stack vertically. Widths of 0
// (unconstrained) or >= threshold use the normal rendering path.
const compactWidthThreshold = 40

// compactOverride is the tri-state result of an explicit WithCompact option.
type compactOverride int

const (
	compactAuto    compactOverride = iota // decide from width
	compactForce                          // WithCompact(true)
	compactDisable                        // WithCompact(false)
)

// WithCompact forces compact rendering on or off, overriding the automatic
// width-based detection. WithCompact(true) activates compact mode even at wide
// widths; WithCompact(false) keeps the normal path even below the threshold.
func WithCompact(on bool) Option {
	return func(s *Surface) {
		if on {
			s.compactOverride = compactForce
		} else {
			s.compactOverride = compactDisable
		}
	}
}

// WithCompactThreshold overrides the width below which compact rendering
// activates automatically. It has no effect when WithCompact has set an
// explicit override.
func WithCompactThreshold(w int) Option {
	return func(s *Surface) { s.compactThreshold = w }
}

// Surface renders one A2UI surface: the component set from an updateComponents
// message, walked as a tree starting from its root.
type Surface struct {
	base
	id     string
	byID   map[string]a2ui.Component
	rootID string

	// styles holds the chrome styles for this surface. Defaults to
	// DefaultStyles when no WithStyles option is provided.
	styles Styles

	// data holds resolved data-model values keyed by path, used to resolve
	// bound DynamicString/Value components at render time.
	data map[string]any

	// focusables are the IDs of interactive components (buttons and text
	// fields) in depth-first tree order; focusIdx points at the one holding
	// focus.
	focusables []string
	focusIdx   int

	// fieldValues holds the edited text of TextField components, keyed by
	// component ID. It is lazily initialized on first edit. An entry here
	// shadows the component's static literal value for both rendering and
	// value readout (gatherFieldValues / FieldValues).
	fieldValues map[string]string

	// compactOverride controls whether compact rendering is forced on,
	// forced off, or decided automatically from the surface width.
	compactOverride compactOverride

	// compactThreshold is the width below which compact rendering activates
	// automatically. Defaults to compactWidthThreshold.
	compactThreshold int

	// hostWidth is the width the host allocated via SetSize. The compact-mode
	// decision keys off this — a stable property of the panel — rather than
	// s.width, which withWidth narrows as the walk descends into cards/lists.
	hostWidth int
}

// SetSize records the host-allocated width for the compact-mode decision, then
// forwards to the embedded base (which the render walk reads and narrows via
// withWidth). Keeping the compact decision on hostWidth stops compact mode from
// leaking into nested subtrees whose per-subtree budget dips below the
// threshold on a surface that is wide overall.
func (s *Surface) SetSize(width, height int) {
	s.hostWidth = width
	s.base.SetSize(width, height)
}

// NewSurface indexes components by ID and picks the surface root: the single
// component that no other component references as a child. If every component
// is referenced (or none is), it falls back to the first in declaration order.
// surfaceID is the A2UI surfaceId the components belong to; it is carried on
// the events the surface emits.
//
// Optional configuration via functional options (e.g. WithStyles) may be
// appended after the two required arguments; calling NewSurface without any
// options produces a surface with DefaultStyles — byte-for-byte identical to
// the pre-options behaviour.
func NewSurface(surfaceID string, components []a2ui.Component, opts ...Option) *Surface {
	byID := make(map[string]a2ui.Component, len(components))
	for _, c := range components {
		byID[c.ID] = c
	}
	s := &Surface{
		id:               surfaceID,
		byID:             byID,
		rootID:           rootID(components),
		styles:           DefaultStyles(),
		compactThreshold: compactWidthThreshold,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.focusables = s.collectFocusables()
	return s
}

// Init implements tea.Model.
func (s *Surface) Init() tea.Cmd { return nil }

// Update implements tea.Model. When the surface holds focus, Tab / Shift+Tab
// cycle through buttons and text fields. Enter activates the focused button.
// When a text field holds focus, rune key presses append to its value and
// backspace deletes the last rune; Enter on a text field is a no-op (there is
// no form-submit concept).
//
// Button activation emits two messages via tea.Batch:
//   - event.ButtonClicked — the host-facing convenience event, carrying
//     the button's resolved *a2ui.EventAction (nil for buttons with no server
//     event).
//   - a2ui.ClientMessage — the protocol-native round-trip message whose
//     ActionEvent carries Name, SurfaceID, SourceComponentID, and Context.
//     Context is populated from gatherFieldValues, so typed text field edits
//     flow through. This is only emitted when the button has a server-side
//     Event action.
//
// The ActionEvent.Timestamp is left empty for the host to stamp. Calling
// time.Now here would make Update non-deterministic and break golden tests.
//
// Per the composition contract the surface never quits — the host owns program
// exit.
func (s *Surface) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !s.Focused() || len(s.focusables) == 0 {
		return s, nil
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}
	switch key.String() {
	case "tab":
		s.focusIdx = (s.focusIdx + 1) % len(s.focusables)
	case "shift+tab":
		s.focusIdx = (s.focusIdx - 1 + len(s.focusables)) % len(s.focusables)
	case "enter":
		// Enter only activates buttons; on a text field it is a no-op.
		if s.focusedIsButton() {
			return s, s.activate()
		}
	case "backspace":
		if s.focusedIsTextField() {
			s.deleteRune()
		}
	default:
		// Rune key presses edit the focused text field.
		if s.focusedIsTextField() && key.Mod == 0 && key.Code != 0 {
			s.appendRune(key.Code)
		}
	}
	return s, nil
}

// activate dispatches the activation messages for the focused button. It
// resolves the focused button's a2ui.Component from the surface's component
// map, reads its Action, and emits:
//
//   - event.ButtonClicked (always) — enriched with the resolved
//     *a2ui.EventAction, nil for buttons with no server event.
//   - a2ui.ClientMessage (only when Action.Event is non-nil) — the
//     protocol-native round-trip message whose ActionEvent carries Name,
//     SurfaceID, and SourceComponentID. A FunctionCall-only button produces
//     no ClientMessage; client-side functions are handled by the host, not
//     round-tripped to the agent.
//
// ActionEvent.Timestamp is left empty — the host stamps it before sending.
// Calling time.Now here would make Update non-deterministic.
func (s *Surface) activate() tea.Cmd {
	id := s.focusables[s.focusIdx]
	c, ok := s.byID[id]
	if !ok || c.Button == nil {
		// Not a button — nothing to activate. This path should not be
		// reached (Update guards enter for non-buttons) but is defensive.
		return nil
	}

	// Resolve the server-side event action, if any.
	var ea *a2ui.EventAction
	if c.Button.Action.Event != nil {
		ea = c.Button.Action.Event
	}

	clicked := event.ButtonClicked{
		Source: event.Source{ComponentID: id, SurfaceID: s.id},
		ID:     id,
		Action: ea,
	}

	// FunctionCall-only buttons have no server event to round-trip.
	if ea == nil {
		return func() tea.Msg { return clicked }
	}

	// Gather the ActionEvent.Context: the surface's input component values,
	// then the action's own declared context bindings (which override field
	// values on key collision — the producer's explicit intent wins).
	ctx := s.gatherFieldValues()
	for k, dv := range ea.Context {
		if v := resolveDynamicValue(dv); v != nil {
			ctx[k] = v
		}
	}

	// Emit both the host-facing event and the protocol-native ClientMessage.
	cm := a2ui.ClientMessage{
		Version: a2ui.Version,
		Action: &a2ui.ActionEvent{
			Name:              ea.Name,
			SurfaceID:         s.id,
			SourceComponentID: id,
			Context:           ctx,
		},
	}
	return tea.Batch(
		func() tea.Msg { return clicked },
		func() tea.Msg { return cm },
	)
}

// View implements tea.Model.
func (s *Surface) View() tea.View {
	if s.rootID == "" {
		return tea.NewView("[a2tea: empty surface]")
	}
	return tea.NewView(s.renderComponent(s.rootID, map[string]bool{}))
}

// isFocused reports whether the component with the given ID currently holds
// focus (surface focused AND it is the selected focusable).
func (s *Surface) isFocused(id string) bool {
	return s.Focused() && len(s.focusables) > 0 && s.focusables[s.focusIdx] == id
}

// Focusables returns the IDs of interactive components (buttons and text
// fields) in depth-first focus-ring order. The host can use this to inspect
// the focus ring; it is mainly intended for testing.
func (s *Surface) Focusables() []string {
	return s.focusables
}

// focusedComponent returns the component at the current focus index, or a
// zero Component when the focus ring is empty.
func (s *Surface) focusedComponent() a2ui.Component {
	if len(s.focusables) == 0 {
		return a2ui.Component{}
	}
	return s.byID[s.focusables[s.focusIdx]]
}

// focusedIsButton reports whether the currently focused component is a Button.
func (s *Surface) focusedIsButton() bool {
	c := s.focusedComponent()
	return c.Button != nil
}

// focusedIsTextField reports whether the currently focused component is a
// TextField.
func (s *Surface) focusedIsTextField() bool {
	c := s.focusedComponent()
	return c.TextField != nil
}

// appendRune appends a rune to the focused text field's edited value,
// lazily initializing the fieldValues map on first edit. On the first edit,
// the field's current display value (literal or binding placeholder) is used
// as the starting point so typed characters extend the existing text.
func (s *Surface) appendRune(r rune) {
	id := s.focusables[s.focusIdx]
	if s.fieldValues == nil {
		s.fieldValues = make(map[string]string)
	}
	// Seed with the current literal value on first edit so typed characters
	// extend the existing text rather than replacing it.
	if _, ok := s.fieldValues[id]; !ok {
		c := s.byID[id]
		if c.TextField != nil && c.TextField.Value != nil {
			s.fieldValues[id] = s.dynString(*c.TextField.Value)
		}
	}
	s.fieldValues[id] += string(r)
}

// deleteRune removes the last rune from the focused text field's edited
// value. If the value drops back to the field's original literal, the key is
// deleted so rendering falls back to the static literal.
func (s *Surface) deleteRune() {
	id := s.focusables[s.focusIdx]
	if s.fieldValues == nil {
		return
	}
	v, ok := s.fieldValues[id]
	if !ok || len(v) == 0 {
		return
	}
	runes := []rune(v)
	v = string(runes[:len(runes)-1])
	// If the edited value matches the original literal, drop the key so
	// rendering falls back to the static value.
	literal := ""
	if c := s.byID[id]; c.TextField != nil && c.TextField.Value != nil {
		literal = s.dynString(*c.TextField.Value)
	}
	if v == literal {
		delete(s.fieldValues, id)
	} else {
		s.fieldValues[id] = v
	}
}

// compact reports whether the surface should render in compact mode. When an
// explicit override is set via WithCompact it wins; otherwise compact mode
// activates automatically when the host-allocated width is non-zero and below
// the threshold. Host width 0 (unconstrained) and >= threshold use the normal
// rendering path. The decision uses hostWidth, not s.width, so it stays uniform
// across the whole surface even as withWidth narrows the per-subtree budget.
func (s *Surface) compact() bool {
	switch s.compactOverride {
	case compactForce:
		return true
	case compactDisable:
		return false
	default:
		return s.hostWidth > 0 && s.hostWidth < s.compactThreshold
	}
}

// collectFocusables walks the tree from the root and returns the IDs of
// interactive components (buttons and text fields) in depth-first order. A
// component referenced by more than one parent (legal adjacency-list reuse) is
// collected once — it is one interactive element however many times it is
// drawn.
func (s *Surface) collectFocusables() []string {
	var out []string
	collected := map[string]bool{}
	var walk func(id string, seen map[string]bool)
	walk = func(id string, seen map[string]bool) {
		if seen[id] {
			return
		}
		seen[id] = true
		defer delete(seen, id)
		c, ok := s.byID[id]
		if !ok {
			return
		}
		if (c.Button != nil || c.TextField != nil) && !collected[c.ID] {
			collected[c.ID] = true
			out = append(out, c.ID)
		}
		for _, child := range childIDs(c) {
			walk(child, seen)
		}
	}
	if s.rootID != "" {
		walk(s.rootID, map[string]bool{})
	}
	return out
}

// renderComponent renders the component with the given ID, recursing into
// children. seen holds the IDs on the current ancestor path so genuine
// reference cycles are caught; it is unwound on return (delete below) so a
// component legally referenced by two parents — normal adjacency-list reuse —
// is not mistaken for a cycle on its second occurrence.
func (s *Surface) renderComponent(id string, seen map[string]bool) string {
	if seen[id] {
		return fmt.Sprintf("[a2tea: cycle at %q]", id)
	}
	seen[id] = true
	defer delete(seen, id)

	c, ok := s.byID[id]
	if !ok {
		return fmt.Sprintf("[a2tea: missing component %q]", id)
	}

	switch {
	case c.Text != nil:
		return s.renderText(c)
	case c.Card != nil:
		return s.renderCard(c, seen)
	case c.Column != nil:
		return s.renderColumn(c, seen)
	case c.Row != nil:
		return s.renderRow(c, seen)
	case c.List != nil:
		return s.renderList(c, seen)
	case c.Divider != nil:
		return s.renderDivider(c)
	case c.Tabs != nil:
		return s.renderTabs(c, seen)
	case c.Modal != nil:
		return s.renderModal(c, seen)
	case c.Button != nil:
		return s.renderButton(c, seen)
	case c.TextField != nil:
		return s.renderTextField(c)
	case c.CheckBox != nil:
		return s.renderCheckBox(c)
	case c.ChoicePicker != nil:
		return s.renderChoicePicker(c)
	case c.Slider != nil:
		return s.renderSlider(c)
	case c.DateTimeInput != nil:
		return s.renderDateTimeInput(c)
	case c.Image != nil:
		return s.renderImage(c)
	case c.Icon != nil:
		return s.renderIcon(c)
	case c.Video != nil:
		return s.renderVideo(c)
	case c.AudioPlayer != nil:
		return s.renderAudio(c)
	default:
		return fmt.Sprintf("[a2tea: %s]", KindOf(c))
	}
}

// withWidth renders f under a temporarily narrowed width budget and restores
// the previous budget afterwards. s.width therefore always holds the budget
// for the subtree currently being rendered — the host-allocated width at the
// root, minus each enclosing container's chrome (a Card's border+padding, a
// List's bullet indent). Rendering is a single-goroutine depth-first pass, so
// the save/restore is safe.
func (s *Surface) withWidth(w int, f func() string) string {
	old := s.width
	s.width = w
	defer func() { s.width = old }()
	return f()
}

// renderChildren renders each child ID in a ChildList in order. The
// dynamic-template form of ChildList is not yet supported (no data model);
// it renders a single placeholder.
func (s *Surface) renderChildren(cl a2ui.ChildList, seen map[string]bool) []string {
	if cl.Template != nil {
		return []string{s.styles.Caption.Render("[a2tea: dynamic children not yet supported]")}
	}
	parts := make([]string, 0, len(cl.IDs))
	for _, id := range cl.IDs {
		parts = append(parts, s.renderComponent(id, seen))
	}
	return parts
}

// dynString extracts a display string from a DynamicString: the literal when
// present, the resolved data-model value when bound, otherwise a placeholder
// marking an unresolved binding or function call.
func (s *Surface) dynString(d a2ui.DynamicString) string {
	switch {
	case d.Literal != nil:
		return *d.Literal
	case d.Binding != nil:
		if s.data != nil {
			key := strings.TrimPrefix(d.Binding.Path, "/")
			if v, ok := s.data[key]; ok {
				if str, ok := v.(string); ok {
					return str
				}
				return fmt.Sprintf("%v", v)
			}
		}
		return "{binding}"
	case d.FunctionCall != nil:
		return "{fn}"
	default:
		return ""
	}
}

// rootID returns the ID of the component that is not referenced as any other
// component's child, falling back to the first component.
func rootID(components []a2ui.Component) string {
	if len(components) == 0 {
		return ""
	}
	referenced := make(map[string]bool)
	for _, c := range components {
		for _, id := range childIDs(c) {
			referenced[id] = true
		}
	}
	for _, c := range components {
		if !referenced[c.ID] {
			return c.ID
		}
	}
	return components[0].ID
}

// childIDs returns the IDs a container component references. Leaf components
// return nil.
func childIDs(c a2ui.Component) []string {
	switch {
	case c.Card != nil:
		return []string{c.Card.Child}
	case c.Button != nil:
		return []string{c.Button.Child}
	case c.Column != nil:
		return c.Column.Children.IDs
	case c.Row != nil:
		return c.Row.Children.IDs
	case c.List != nil:
		return c.List.Children.IDs
	case c.Modal != nil:
		return []string{c.Modal.Content, c.Modal.Trigger}
	case c.Tabs != nil:
		ids := make([]string, 0, len(c.Tabs.Tabs))
		for _, t := range c.Tabs.Tabs {
			ids = append(ids, t.Child)
		}
		return ids
	}
	return nil
}

// KindOf reports the A2UI component kind of c ("text", "card", "button", ...),
// or "unknown" when no concrete field is set.
func KindOf(c a2ui.Component) string {
	switch {
	case c.Text != nil:
		return "text"
	case c.Image != nil:
		return "image"
	case c.Icon != nil:
		return "icon"
	case c.Video != nil:
		return "video"
	case c.AudioPlayer != nil:
		return "audioPlayer"
	case c.Row != nil:
		return "row"
	case c.Column != nil:
		return "column"
	case c.List != nil:
		return "list"
	case c.Card != nil:
		return "card"
	case c.Tabs != nil:
		return "tabs"
	case c.Modal != nil:
		return "modal"
	case c.Divider != nil:
		return "divider"
	case c.Button != nil:
		return "button"
	case c.TextField != nil:
		return "textField"
	case c.CheckBox != nil:
		return "checkBox"
	case c.ChoicePicker != nil:
		return "choicePicker"
	case c.Slider != nil:
		return "slider"
	case c.DateTimeInput != nil:
		return "dateTimeInput"
	}
	return "unknown"
}

// ensure strings is referenced by this file's helpers even before the
// component files land; hr and wrapTo live in styles.go.
var _ = strings.TrimSpace
