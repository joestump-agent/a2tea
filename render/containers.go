package render

import (
	"strings"

	"charm.land/lipgloss/v2"

	a2ui "github.com/tmc/a2ui"
)

// defaultDividerWidth sizes a horizontal divider when the surface has no
// width constraint (s.width == 0).
const defaultDividerWidth = 24

// renderColumn renders a Column component: children stacked vertically,
// left-aligned. Align and Justify are ignored for now — honoring them needs
// a height/width budget per child, which the layout model doesn't carry yet.
func (s *Surface) renderColumn(c a2ui.Component, seen map[string]bool) string {
	parts := s.renderChildren(c.Column.Children, seen)
	if len(parts) == 0 {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderRow renders a Row component: children joined horizontally, top-aligned,
// with a two-space gap between adjacent children. In compact mode the children
// fall back to vertical stacking (JoinVertical/Left) so they don't overflow a
// narrow width budget. Align and Justify are ignored for now (see renderColumn).
func (s *Surface) renderRow(c a2ui.Component, seen map[string]bool) string {
	parts := s.renderChildren(c.Row.Children, seen)
	if s.compact() {
		return lipgloss.JoinVertical(lipgloss.Left, parts...)
	}
	return joinRow(parts)
}

// renderList renders a List component. Vertical lists (the default when
// Direction is unset) bullet each child block — rendering the children under
// a budget narrowed by the 2-cell bullet indent so wrapped lines still fit —
// while horizontal lists lay children out like a Row. In compact mode a
// horizontal list falls back to vertical stacking (like a compact Row) so it
// doesn't overflow a narrow width budget.
func (s *Surface) renderList(c a2ui.Component, seen map[string]bool) string {
	if c.List.Direction == a2ui.ListDirectionHorizontal {
		parts := s.renderChildren(c.List.Children, seen)
		if s.compact() {
			return lipgloss.JoinVertical(lipgloss.Left, parts...)
		}
		return joinRow(parts)
	}
	// Narrow the child budget by the 2-cell "• " indent so bulleted lines
	// still fit; when constrained, floor at 1 so a width of 1-2 doesn't
	// leave children with a zero (= unconstrained) budget and overflow.
	childWidth := s.width
	if childWidth > 0 {
		childWidth -= 2
		if childWidth < 1 {
			childWidth = 1
		}
	}
	parts := s.withWidthParts(childWidth, c.List.Children, seen)
	if len(parts) == 0 {
		return ""
	}
	for i, p := range parts {
		parts[i] = bullet(p)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// withWidthParts renders a ChildList under a narrowed width budget.
func (s *Surface) withWidthParts(w int, cl a2ui.ChildList, seen map[string]bool) []string {
	var parts []string
	s.withWidth(w, func() string {
		parts = s.renderChildren(cl, seen)
		return ""
	})
	return parts
}

// renderDivider renders a Divider component: a horizontal rule sized to the
// surface width (a modest default when unconstrained), or a single "│" cell
// for the vertical axis. A vertical divider cannot know its neighbors'
// heights here — the parent Row's top-aligned join keeps it on the first line.
func (s *Surface) renderDivider(c a2ui.Component) string {
	if c.Divider.Axis == a2ui.DividerAxisVertical {
		return "│"
	}
	w := s.width
	if w <= 0 {
		w = defaultDividerWidth
	}
	return hr(w)
}

// renderTabs renders a Tabs component: a bar of tab titles separated by " │ "
// followed by the active tab's child. Tab switching isn't wired yet (tabs are
// not focusable), so the first tab is always the active one — its title is
// bolded and only its child renders.
func (s *Surface) renderTabs(c a2ui.Component, seen map[string]bool) string {
	tabs := c.Tabs.Tabs
	if len(tabs) == 0 {
		return s.styles.Caption.Render("[a2tea: tabs with no tabs]")
	}
	titles := make([]string, len(tabs))
	for i, t := range tabs {
		titles[i] = s.dynString(t.Title)
	}
	titles[0] = s.styles.Heading.Render(titles[0])
	return strings.Join(titles, " │ ") + "\n" + s.renderComponent(tabs[0].Child, seen)
}

// renderModal renders a Modal component. The modal itself is the focusable
// element; its trigger child draws as the modal's chrome behind a cue glyph —
// "▹" idle, "▸" when the modal holds focus (the same glyph-swap convention as
// TextField's input cue). Closed, only the cued trigger renders. Open (Enter
// on the focused modal), the content child renders below the trigger as a
// bordered in-flow block: a true overlay is out of scope for a string-rendered
// surface, so in-flow expansion is the honest terminal equivalent. Esc closes
// (see Surface.Update).
//
// The content border reuses the CardBorder chrome and the same width
// budgeting as renderCard, but it is NOT dropped in compact mode — the border
// is the only signifier separating modal content from the surrounding flow.
func (s *Surface) renderModal(c a2ui.Component, seen map[string]bool) string {
	cue := "▹ "
	if s.isFocused(c.ID) {
		cue = "▸ "
	}
	trigger := cue + s.renderComponent(c.Modal.Trigger, seen)
	if !s.modalOpen(c.ID) {
		return trigger
	}
	if s.width <= 0 {
		return trigger + "\n" + s.styles.CardBorder.Render(s.renderComponent(c.Modal.Content, seen))
	}
	w := s.width - 4
	if w < 1 {
		w = 1
	}
	inner := s.withWidth(w, func() string {
		return s.renderComponent(c.Modal.Content, seen)
	})
	return trigger + "\n" + s.styles.CardBorder.Render(wrapTo(inner, w))
}

// joinRow joins pre-rendered blocks horizontally, top-aligned, with a
// two-space gap between adjacent blocks.
func joinRow(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	gapped := make([]string, 0, 2*len(parts)-1)
	for i, p := range parts {
		if i > 0 {
			gapped = append(gapped, "  ")
		}
		gapped = append(gapped, p)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, gapped...)
}

// bullet prefixes a rendered block for a vertical list: "• " on the first
// line and a matching two-space indent on continuation lines.
func bullet(block string) string {
	lines := strings.Split(block, "\n")
	for i, ln := range lines {
		if i == 0 {
			lines[i] = "• " + ln
		} else {
			lines[i] = "  " + ln
		}
	}
	return strings.Join(lines, "\n")
}
