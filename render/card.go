package render

import a2ui "github.com/tmc/a2ui"

// renderCard renders a Card: its child wrapped in a rounded, padded border
// (styleCardBorder). When the surface has a width budget (s.width > 0) the
// child is wrapped to s.width minus the 4 cells of border+padding chrome so
// the finished box never exceeds the budget; when unconstrained (s.width == 0)
// the box sizes to its content. A budget below 5 cells cannot be honored —
// the chrome alone costs 4 — so the child is clamped to a 1-cell column and
// the box renders at its 5-cell minimum.
func (s *Surface) renderCard(c a2ui.Component, seen map[string]bool) string {
	inner := s.renderComponent(c.Card.Child, seen)
	if s.width > 0 {
		w := s.width - 4
		if w < 1 {
			w = 1
		}
		inner = wrapTo(inner, w)
	}
	return styleCardBorder.Render(inner)
}
