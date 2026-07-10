// Package render turns an A2UI surface — a flat set of components that
// reference their children by ID — into a Bubble Tea model that draws the
// component tree.
//
// The renderers here are still visual stubs: leaf text renders literally,
// containers recurse into their children, and interactive/media components
// render a "[a2tea: <kind>]" placeholder. What is real is the tree walk over
// the actual A2UI v0.9 component catalog (github.com/tmc/a2ui) and the
// embeddable Model contract below.
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
	byID   map[string]a2ui.Component
	rootID string
}

// NewSurface indexes components by ID and picks the surface root: the single
// component that no other component references as a child. If every component
// is referenced (or none is), it falls back to the first in declaration order.
func NewSurface(components []a2ui.Component) *Surface {
	byID := make(map[string]a2ui.Component, len(components))
	for _, c := range components {
		byID[c.ID] = c
	}
	return &Surface{byID: byID, rootID: rootID(components)}
}

// Init implements tea.Model.
func (s *Surface) Init() tea.Cmd { return nil }

// Update implements tea.Model. It is a no-op and, per the composition
// contract, never quits — the host owns program exit.
func (s *Surface) Update(tea.Msg) (tea.Model, tea.Cmd) { return s, nil }

// View implements tea.Model.
func (s *Surface) View() tea.View {
	if s.rootID == "" {
		return tea.NewView("[a2tea: empty surface]")
	}
	return tea.NewView(s.renderComponent(s.rootID, map[string]bool{}))
}

// renderComponent renders the component with the given ID, recursing into
// children. seen guards against reference cycles in malformed documents.
func (s *Surface) renderComponent(id string, seen map[string]bool) string {
	if seen[id] {
		return fmt.Sprintf("[a2tea: cycle at %q]", id)
	}
	seen[id] = true

	c, ok := s.byID[id]
	if !ok {
		return fmt.Sprintf("[a2tea: missing component %q]", id)
	}

	switch {
	case c.Text != nil:
		return dynString(c.Text.Text)
	case c.Card != nil:
		return s.renderComponent(c.Card.Child, seen)
	case c.Column != nil:
		return s.joinChildren(c.Column.Children, "\n", seen)
	case c.List != nil:
		return s.joinChildren(c.List.Children, "\n", seen)
	case c.Row != nil:
		return s.joinChildren(c.Row.Children, "  ", seen)
	case c.Button != nil:
		return fmt.Sprintf("[ %s ]", strings.TrimSpace(s.renderComponent(c.Button.Child, seen)))
	case c.TextField != nil:
		return fmt.Sprintf("[a2tea: textField %q]", dynString(c.TextField.Label))
	default:
		return fmt.Sprintf("[a2tea: %s]", KindOf(c))
	}
}

// joinChildren renders each child ID in a ChildList and joins them with sep.
// The dynamic-template form of ChildList is not yet supported (no data model).
func (s *Surface) joinChildren(cl a2ui.ChildList, sep string, seen map[string]bool) string {
	parts := make([]string, 0, len(cl.IDs))
	for _, id := range cl.IDs {
		parts = append(parts, s.renderComponent(id, seen))
	}
	return strings.Join(parts, sep)
}

// dynString extracts a display string from a DynamicString: the literal when
// present, otherwise a placeholder marking a binding/function the data model
// would resolve (not wired at the stub stage).
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
