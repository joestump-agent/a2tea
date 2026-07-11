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
// Tab / Shift+Tab cycle the buttons and Enter emits event.ButtonClicked as a
// tea.Msg. Editing input components is not wired yet.
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

// Surface renders one A2UI surface: the component set from an updateComponents
// message, walked as a tree starting from its root.
type Surface struct {
	base
	id     string
	byID   map[string]a2ui.Component
	rootID string

	// focusables are the IDs of interactive components (buttons) in
	// depth-first tree order; focusIdx points at the one holding focus.
	focusables []string
	focusIdx   int
}

// NewSurface indexes components by ID and picks the surface root: the single
// component that no other component references as a child. If every component
// is referenced (or none is), it falls back to the first in declaration order.
// surfaceID is the A2UI surfaceId the components belong to; it is carried on
// the events the surface emits.
func NewSurface(surfaceID string, components []a2ui.Component) *Surface {
	byID := make(map[string]a2ui.Component, len(components))
	for _, c := range components {
		byID[c.ID] = c
	}
	s := &Surface{id: surfaceID, byID: byID, rootID: rootID(components)}
	s.focusables = s.collectFocusables()
	return s
}

// Init implements tea.Model.
func (s *Surface) Init() tea.Cmd { return nil }

// Update implements tea.Model. When the surface holds focus, Tab / Shift+Tab
// cycle button focus and Enter activates the focused button, emitting
// event.ButtonClicked. Per the composition contract it never quits — the host
// owns program exit.
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
		id := s.focusables[s.focusIdx]
		return s, func() tea.Msg {
			return event.ButtonClicked{
				Source: event.Source{ComponentID: id, SurfaceID: s.id},
				ID:     id,
			}
		}
	}
	return s, nil
}

// View implements tea.Model.
func (s *Surface) View() tea.View {
	if s.rootID == "" {
		return tea.NewView("[a2tea: empty surface]")
	}
	return tea.NewView(s.renderComponent(s.rootID, map[string]bool{}))
}

// isFocused reports whether the component with the given ID currently holds
// button focus (surface focused AND it is the selected focusable).
func (s *Surface) isFocused(id string) bool {
	return s.Focused() && len(s.focusables) > 0 && s.focusables[s.focusIdx] == id
}

// collectFocusables walks the tree from the root and returns the IDs of
// interactive components (buttons) in depth-first order. A component
// referenced by more than one parent (legal adjacency-list reuse) is
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
		if c.Button != nil && !collected[c.ID] {
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
		return []string{styleCaption.Render("[a2tea: dynamic children not yet supported]")}
	}
	parts := make([]string, 0, len(cl.IDs))
	for _, id := range cl.IDs {
		parts = append(parts, s.renderComponent(id, seen))
	}
	return parts
}

// dynString extracts a display string from a DynamicString: the literal when
// present, otherwise a placeholder marking a binding/function the data model
// would resolve (not wired yet).
func dynString(d a2ui.DynamicString) string {
	switch {
	case d.Literal != nil:
		return *d.Literal
	case d.Binding != nil:
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
