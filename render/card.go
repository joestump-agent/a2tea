package render

import a2ui "github.com/tmc/a2ui"

// renderCard renders a Card: its child wrapped in a rounded, padded border
// (the CardBorder style). When the surface has a width budget (s.width > 0) the
// child subtree is rendered under a budget narrowed by the 4 cells of
// border+padding chrome (via withWidth, so text wraps once at the right
// width instead of being re-wrapped after the fact), and the result is
// padded to that inner width so the finished box fills the budget exactly.
// When unconstrained (s.width == 0) the box sizes to its content. A budget
// below 5 cells cannot be honored — the chrome alone costs 4 — so the child
// is clamped to a 1-cell column and the box renders at its 5-cell minimum.
func (s *Surface) renderCard(c a2ui.Component, seen map[string]bool) string {
	if s.width <= 0 {
		return s.styles.CardBorder.Render(s.renderComponent(c.Card.Child, seen))
	}
	w := s.width - 4
	if w < 1 {
		w = 1
	}
	inner := s.withWidth(w, func() string {
		return s.renderComponent(c.Card.Child, seen)
	})
	// Pad (and, for chrome that ignores the budget, wrap) to the inner width
	// so the box renders flush at exactly s.width.
	return s.styles.CardBorder.Render(wrapTo(inner, w))
}
