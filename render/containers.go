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
// with a two-space gap between adjacent children. Align and Justify are
// ignored for now (see renderColumn).
func (s *Surface) renderRow(c a2ui.Component, seen map[string]bool) string {
	return joinRow(s.renderChildren(c.Row.Children, seen))
}

// renderList renders a List component. Vertical lists (the default when
// Direction is unset) bullet each child block — rendering the children under
// a budget narrowed by the 2-cell bullet indent so wrapped lines still fit —
// while horizontal lists lay children out like a Row.
func (s *Surface) renderList(c a2ui.Component, seen map[string]bool) string {
	if c.List.Direction == a2ui.ListDirectionHorizontal {
		return joinRow(s.renderChildren(c.List.Children, seen))
	}
	childWidth := s.width
	if childWidth > 2 {
		childWidth -= 2
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
		return styleCaption.Render("[a2tea: tabs with no tabs]")
	}
	titles := make([]string, len(tabs))
	for i, t := range tabs {
		titles[i] = s.dynString(t.Title)
	}
	titles[0] = styleHeading.Render(titles[0])
	return strings.Join(titles, " │ ") + "\n" + s.renderComponent(tabs[0].Child, seen)
}

// renderModal renders a Modal component: the trigger child plus a faint note
// that the modal content stays hidden until interaction support lands. Content
// is deliberately not rendered — a closed modal shows only its trigger.
func (s *Surface) renderModal(c a2ui.Component, seen map[string]bool) string {
	trigger := s.renderComponent(c.Modal.Trigger, seen)
	note := styleCaption.Render("[a2tea: modal content hidden until interaction support lands]")
	return trigger + "\n" + note
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
