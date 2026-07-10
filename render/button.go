package render

import (
	"strings"

	a2ui "github.com/tmc/a2ui"
)

// renderButton renders a Button as bracketed chrome around its child's
// rendered, trimmed content: "[ label ]". The whole bracketed run goes
// through one style — styleButton idle, styleButtonFocused when focused —
// so focus (reverse-video) reads as a single solid highlight.
//
// Variant decision (default / primary / borderless): default and primary
// render identically — styleButton is already bold and reverse-video is
// reserved for focus, so monochrome leaves no attribute for a primary
// distinction that would not collide with existing chrome. borderless drops
// the brackets but keeps the same styles so focus still reads; it falls back
// to brackets when the label is empty so the button stays visible.
//
// The button's Action is not rendered; Surface.Update emits
// event.ButtonClicked when the focused button is activated.
func (s *Surface) renderButton(c a2ui.Component, seen map[string]bool) string {
	label := strings.TrimSpace(s.renderComponent(c.Button.Child, seen))
	chrome := "[ " + label + " ]"
	if c.Button.Variant == a2ui.ButtonVariantBorderless && label != "" {
		chrome = label
	}
	if s.isFocused(c.ID) {
		return styleButtonFocused.Render(chrome)
	}
	return styleButton.Render(chrome)
}
