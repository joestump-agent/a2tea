// Package render turns an A2UI surface — a flat set of components that
// reference their children by ID — into a Bubble Tea model that draws the
// component tree with lipgloss.
//
// Rendering is real for the core catalog: Text (with variants), Card, Column,
// Row, List, Divider, Button, and editable visuals for the input components
// (TextField, CheckBox, ChoicePicker, Slider, DateTimeInput). Tabs render a
// title bar plus the active tab's content. Media components (Image, Icon,
// Video, AudioPlayer) draw compact placeholders. DynamicString data bindings
// render as placeholders until the data model lands.
//
// Interaction: Buttons are focusable. When the host grants the surface focus,
// Tab / Shift+Tab cycle the buttons and Enter activates the focused button.
// Activation emits event.ButtonClicked (carrying the resolved *a2ui.EventAction)
// and, when the button has a server-side Event action, a protocol-native
// a2ui.ClientMessage whose ActionEvent carries Name, SurfaceID, and
// SourceComponentID. FunctionCall-only buttons emit no ClientMessage.
//
// Input components are editable and join Buttons in the focus ring. A focused
// TextField (and DateTimeInput, which shares the same rune-edit path against
// its string value) accepts printable keys and backspace, and Enter emits
// event.InputSubmitted with the field's current value. A focused CheckBox
// toggles with Space or Enter. A focused ChoicePicker moves its highlight with
// Up/Down and toggles the highlighted option with Space, emitting
// event.ChoiceSelected whenever the selection set changes. A focused Slider
// steps with Left/Right within its min/max bounds. Edited values are read back
// via FieldValues and flow into a button's ActionEvent Context.
//
// Tab bars also join the focus ring: when one holds focus, Left / Right (or
// h / l) switch the active tab, and the active tab survives Apply merges the
// same way focus does. Only the active tab's subtree joins the ring — focus
// never lands on a component hidden inside an inactive tab.
//
// Modals open and close: a Modal joins the focus ring as a single element
// drawn as its trigger, Enter toggles it open, and Esc closes the most
// recently opened modal. Open content renders as a bordered in-flow block —
// the honest terminal equivalent of an overlay — and its focusables join the
// ring only while it is open. Hosts use HasOpenModal to keep Esc routed to
// the surface while a modal is up (mirroring the EditingText probe).
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

	// scope is the stack of data-model elements for the ChildList template
	// instances currently being rendered: renderTemplateChildren pushes each
	// list element before rendering the template component and pops it after,
	// so bindings inside the instance resolve against that element first (see
	// lookupBinding). Empty outside template expansion. Rendering is a
	// single-goroutine depth-first pass, so the push/pop is safe.
	scope []any

	// focusables are the IDs of interactive components (buttons, input
	// components, modals, and tab bars) in depth-first tree order; focusIdx
	// points at the one holding focus.
	focusables []string
	focusIdx   int

	// activeTabs holds the active tab index of each Tabs component, keyed by
	// component ID. It is lazily initialized on first switch; a missing entry
	// means the first tab. Entries survive applyComponents merges — like
	// focus, the active tab is user state the server must not clobber — and
	// are clamped at read time (activeTab) so a merge that shrinks a tab list
	// below a previously selected index falls back to the first tab.
	activeTabs map[string]int

	// fieldValues holds the edited text of TextField and DateTimeInput
	// components, keyed by component ID. It is lazily initialized on first
	// edit. An entry here shadows the component's static literal value for
	// both rendering and value readout (gatherFieldValues / FieldValues).
	fieldValues map[string]string

	// checkValues holds the toggled state of CheckBox components, keyed by
	// component ID. Lazily initialized on first toggle; an entry shadows the
	// component's static literal, mirroring fieldValues.
	checkValues map[string]bool

	// choiceValues holds the edited selection of ChoicePicker components,
	// keyed by component ID. The value is the selected option values in
	// option-declaration order. An entry shadows the static literal.
	choiceValues map[string][]string

	// sliderValues holds the adjusted value of Slider components, keyed by
	// component ID. An entry shadows the static literal.
	sliderValues map[string]float64

	// choiceCursor holds each ChoicePicker's highlighted option index, keyed
	// by component ID. It is presentation state, not a value: it never flows
	// into FieldValues or ActionEvent.Context.
	choiceCursor map[string]int

	// openModals holds the component IDs of Modal components currently
	// open, in the order they were opened (a stack — Esc closes the most
	// recently opened first). Like fieldValues it is user-interaction state:
	// it survives Apply merges and is cleared by deleteSurface.
	openModals []string

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
// cycle through buttons, input components, modals, and tab bars. Enter
// activates the focused button and toggles the focused modal open/closed; Esc
// closes the most recently opened modal (and is otherwise ignored, so the
// host keeps its Esc semantics when no modal is open — see HasOpenModal).
// When a text-editable component (TextField or DateTimeInput) holds focus,
// rune key presses append to its value, backspace deletes the last rune, and
// Enter emits event.InputSubmitted carrying the field's current value. A
// focused CheckBox toggles on Space or Enter; a focused ChoicePicker moves
// its highlight with Up/Down and toggles the highlighted option with Space,
// emitting event.ChoiceSelected when the selection set changes; a focused
// Slider steps with Left/Right. When a tab bar holds focus, Left / Right (or
// h / l) switch its active tab; h and l reach the tab bar only when it is the
// focused component — on a focused text-editable component they are literal
// runes.
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
		// Enter toggles a focused modal open/closed, activates a focused
		// button, submits a focused text-editable component's value, and
		// toggles a focused checkbox.
		if s.focusedIsModal() {
			s.toggleModal(s.focusables[s.focusIdx])
			return s, nil
		}
		if s.focusedIsButton() {
			return s, s.activate()
		}
		if s.focusedIsTextEditable() {
			return s, s.submitInput()
		}
		if s.focusedIsCheckBox() {
			s.toggleCheckBox()
		}
	case "space":
		// Space is a command on toggleable components and text input on
		// text-editable ones. Key.String() reports "space" (never " "), so
		// the text-input branch cannot be reached from the default case and
		// must be handled here.
		switch {
		case s.focusedIsCheckBox():
			s.toggleCheckBox()
		case s.focusedIsChoicePicker():
			return s, s.togglePickerOption()
		case s.focusedIsTextEditable():
			s.appendText(" ")
		}
	case "up":
		if s.focusedIsChoicePicker() {
			s.movePickerCursor(-1)
		}
	case "down":
		if s.focusedIsChoicePicker() {
			s.movePickerCursor(1)
		}
	case "left":
		// Left steps a focused slider down and switches a focused tab bar
		// to the previous tab; the focused component type disambiguates.
		if s.focusedIsSlider() {
			s.stepSlider(-1)
		} else if s.focusedIsTabs() {
			s.switchTab(-1)
		}
	case "right":
		if s.focusedIsSlider() {
			s.stepSlider(1)
		} else if s.focusedIsTabs() {
			s.switchTab(1)
		}
	case "esc":
		// Esc closes the most recently opened modal, returning focus to it
		// (its trigger). With no modal open the key falls through untouched —
		// in Standalone that means Esc quits, gated by the HasOpenModal probe.
		if id, ok := s.closeTopModal(); ok {
			s.refreshFocusables(id)
		}
	case "backspace":
		if s.focusedIsTextEditable() {
			s.deleteRune()
		}
	case "h":
		// On a focused tab bar h/l switch tabs (vim-style aliases for
		// Left/Right); on a focused text-editable component they are
		// literal runes and the rune-edit path wins.
		if s.focusedIsTabs() {
			s.switchTab(-1)
		} else if s.focusedIsTextEditable() && key.Text != "" {
			s.appendText(key.Text)
		}
	case "l":
		if s.focusedIsTabs() {
			s.switchTab(1)
		} else if s.focusedIsTextEditable() && key.Text != "" {
			s.appendText(key.Text)
		}
	default:
		// Printable key presses edit the focused text-editable component.
		// key.Text is the actual characters produced by the key — it already
		// accounts for Shift (so "A" and "!" arrive as text) and is empty for
		// navigation and control keys (arrows, Home/End, F-keys), whose Code
		// is a sentinel above unicode.MaxRune that must never be inserted.
		if s.focusedIsTextEditable() && key.Text != "" {
			s.appendText(key.Text)
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

// submitInput dispatches event.InputSubmitted for the focused text-editable
// component (TextField or DateTimeInput). Value follows the same shadowing
// rule as rendering and FieldValues: the edited value when the user has
// typed, else the field's literal or resolved data-model seed (editSeed) — so
// an unedited field submits its pre-filled content and an unresolved binding
// submits "" rather than leaking a display placeholder.
func (s *Surface) submitInput() tea.Cmd {
	id := s.focusables[s.focusIdx]
	v, ok := s.fieldValues[id]
	if !ok {
		v = s.editSeed(id)
	}
	submitted := event.InputSubmitted{
		Source: event.Source{ComponentID: id, SurfaceID: s.id},
		ID:     id,
		Value:  v,
	}
	return func() tea.Msg { return submitted }
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

// Focusables returns the IDs of interactive components (buttons, input
// components, modals, and tab bars) in depth-first focus-ring order. The host
// can use
// this to inspect the focus ring; it is mainly intended for testing.
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

// focusedIsTextEditable reports whether the currently focused component takes
// the rune-edit path: a TextField, or a DateTimeInput (whose string value is
// edited the same way).
func (s *Surface) focusedIsTextEditable() bool {
	c := s.focusedComponent()
	return c.TextField != nil || c.DateTimeInput != nil
}

// focusedIsCheckBox reports whether the currently focused component is a
// CheckBox.
func (s *Surface) focusedIsCheckBox() bool {
	c := s.focusedComponent()
	return c.CheckBox != nil
}

// focusedIsChoicePicker reports whether the currently focused component is a
// ChoicePicker.
func (s *Surface) focusedIsChoicePicker() bool {
	c := s.focusedComponent()
	return c.ChoicePicker != nil
}

// focusedIsSlider reports whether the currently focused component is a Slider.
func (s *Surface) focusedIsSlider() bool {
	c := s.focusedComponent()
	return c.Slider != nil
}

// focusedIsModal reports whether the currently focused component is a Modal.
func (s *Surface) focusedIsModal() bool {
	c := s.focusedComponent()
	return c.Modal != nil
}

// focusedIsTabs reports whether the currently focused component is a Tabs
// component (i.e. the focus ring is on a tab bar).
func (s *Surface) focusedIsTabs() bool {
	c := s.focusedComponent()
	return c.Tabs != nil
}

// switchTab moves the focused tab bar's active tab by delta, wrapping around
// at either end (left from the first tab lands on the last, mirroring the
// focus ring's Tab/Shift+Tab cycling). Because the focus ring contains only
// the active tab's descendants, switching re-collects the ring and restores
// focus to the tab bar itself, which always survives its own switch.
func (s *Surface) switchTab(delta int) {
	id := s.focusables[s.focusIdx]
	c, ok := s.byID[id]
	if !ok || c.Tabs == nil {
		return
	}
	n := len(c.Tabs.Tabs)
	if n == 0 {
		return
	}
	if s.activeTabs == nil {
		s.activeTabs = make(map[string]int)
	}
	s.activeTabs[id] = ((s.activeTab(id, n)+delta)%n + n) % n

	s.focusables = s.collectFocusables()
	for i, fid := range s.focusables {
		if fid == id {
			s.focusIdx = i
			break
		}
	}
}

// activeTab returns the active tab index for the Tabs component with the
// given ID, clamped to [0, n). Out-of-range state — e.g. a component update
// shrank the tab list below a previously selected index — falls back to the
// first tab instead of indexing past the end.
func (s *Surface) activeTab(id string, n int) int {
	idx := s.activeTabs[id]
	if idx < 0 || idx >= n {
		return 0
	}
	return idx
}

// ActiveTab returns the active tab index of the Tabs component with the given
// ID, clamped to the component's current tab count. It returns 0 when the ID
// is unknown, not a Tabs component, or has no tabs.
func (s *Surface) ActiveTab(id string) int {
	c, ok := s.byID[id]
	if !ok || c.Tabs == nil || len(c.Tabs.Tabs) == 0 {
		return 0
	}
	return s.activeTab(id, len(c.Tabs.Tabs))
}

// EditingText reports whether the surface holds focus on a text-editable
// component (TextField or DateTimeInput), i.e. printable key presses are
// currently text input rather than commands. Hosts (and a2tea.Standalone) use
// this to decide whether keys like "q" should quit or be typed.
func (s *Surface) EditingText() bool {
	return s.Focused() && s.focusedIsTextEditable()
}

// editSeed returns the text an edit of the given TextField or DateTimeInput
// starts from: its literal value, or its binding's resolved data-model value,
// or "" when the value is absent or unresolved. A display placeholder like
// "{binding}" or "{fn}" is rendering chrome, not field content — it must never
// leak into edits, FieldValues, or the ActionEvent.Context round-tripped to
// the agent.
func (s *Surface) editSeed(id string) string {
	c, ok := s.byID[id]
	if !ok {
		return ""
	}
	var d a2ui.DynamicString
	switch {
	case c.TextField != nil && c.TextField.Value != nil:
		d = *c.TextField.Value
	case c.DateTimeInput != nil:
		d = c.DateTimeInput.Value
	default:
		return ""
	}
	switch {
	case d.Literal != nil:
		return *d.Literal
	case d.Binding != nil:
		if s.data != nil {
			if v, ok := s.data[strings.TrimPrefix(d.Binding.Path, "/")]; ok {
				if str, ok := v.(string); ok {
					return str
				}
				return fmt.Sprintf("%v", v)
			}
		}
	}
	return ""
}

// appendText appends printable text to the focused text-editable component's
// edited value, lazily initializing the fieldValues map on first edit. On the first edit,
// the field's current content (literal or resolved binding, via editSeed) is
// used as the starting point so typed characters extend the existing text.
// The argument is key.Text — the characters the key produced — which may be
// more than one rune (composed/IME input) and already reflects Shift.
func (s *Surface) appendText(text string) {
	id := s.focusables[s.focusIdx]
	if s.fieldValues == nil {
		s.fieldValues = make(map[string]string)
	}
	if _, ok := s.fieldValues[id]; !ok {
		s.fieldValues[id] = s.editSeed(id)
	}
	s.fieldValues[id] += text
}

// deleteRune removes the last rune from the focused text-editable component's
// edited value.
// On the first edit of a pristine field it seeds from the literal, so backspace
// can shorten (and ultimately clear) a pre-filled default — not just text the
// user typed this session. If the value drops back to exactly the original
// literal, the key is deleted so rendering falls back to the static literal.
func (s *Surface) deleteRune() {
	id := s.focusables[s.focusIdx]
	literal := s.editSeed(id)
	if s.fieldValues == nil {
		s.fieldValues = make(map[string]string)
	}
	v, ok := s.fieldValues[id]
	if !ok {
		// Seed from the literal so a pre-filled field is editable.
		v = literal
	}
	if len(v) == 0 {
		return
	}
	runes := []rune(v)
	v = string(runes[:len(runes)-1])
	// If the edited value matches the original literal, drop the key so
	// rendering falls back to the static value.
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

// isInteractive reports whether c joins the focus ring: Buttons, Modals,
// every editable input component (TextField, CheckBox, ChoicePicker, Slider,
// DateTimeInput), and Tabs bars (unless the tab list is empty — a bar with
// nothing to switch is not interactive).
func isInteractive(c a2ui.Component) bool {
	return c.Button != nil || c.TextField != nil || c.CheckBox != nil ||
		c.ChoicePicker != nil || c.Slider != nil || c.DateTimeInput != nil ||
		c.Modal != nil || (c.Tabs != nil && len(c.Tabs.Tabs) > 0)
}

// collectFocusables walks the tree from the root and returns the IDs of
// interactive components (buttons, input components, modals, and tab bars) in
// depth-first order. A component referenced by more than one parent (legal
// adjacency-list reuse) is collected once — it is one interactive element
// however many times it is drawn.
//
// A Tabs component contributes its own ID (the tab bar is focusable, unless
// it has no tabs to switch) and then only its ACTIVE tab's subtree: focus
// must never land on a component hidden inside an inactive tab — buttons,
// inputs, and modals in inactive tabs are all excluded. switchTab re-collects
// the ring after every switch to keep it in step.
//
// A Modal is its own focusable: the trigger child is the modal's chrome (like
// a button's label), so the trigger subtree never joins the ring separately —
// Enter on the modal is what "activates the trigger". Content focusables are
// reachable only while the modal is open, matching what is on screen.
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
		if isInteractive(c) && !collected[c.ID] {
			collected[c.ID] = true
			out = append(out, c.ID)
		}
		if c.Tabs != nil {
			// Only the ACTIVE tab's subtree joins the ring; components
			// hidden in inactive tabs must not be focusable.
			if tabs := c.Tabs.Tabs; len(tabs) > 0 {
				walk(tabs[s.activeTab(c.ID, len(tabs))].Child, seen)
			}
			return
		}
		if c.Modal != nil {
			// Content focusables join the ring only while the modal is
			// open; the trigger subtree never joins separately.
			if s.modalOpen(c.ID) {
				walk(c.Modal.Content, seen)
			}
			return
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

// renderChildren renders a ChildList: each explicit child ID in order for the
// static form, or one template-component instance per data-model list element
// for the dynamic form (see renderTemplateChildren).
func (s *Surface) renderChildren(cl a2ui.ChildList, seen map[string]bool) []string {
	if cl.Template != nil {
		return s.renderTemplateChildren(cl.Template, seen)
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
		if v, ok := s.lookupBinding(d.Binding.Path); ok {
			if str, ok := v.(string); ok {
				return str
			}
			return fmt.Sprintf("%v", v)
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
		return childListIDs(c.Column.Children)
	case c.Row != nil:
		return childListIDs(c.Row.Children)
	case c.List != nil:
		return childListIDs(c.List.Children)
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
