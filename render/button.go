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
// the brackets but keeps the same styles so focus still reads.
//
// Label fallback: when Child is empty (common when the model puts "text"
// directly on the button — the A2UI schema ignores it so the label is lost),
// the component ID is used as a last-resort placeholder so the button still
// renders as a focusable element instead of [a2tea: missing component ""].
// The producer-side fix (crush#47) repairs childless buttons host-side so the
// real label survives; this fallback is defense-in-depth.
//
// The button's Action is not rendered; Surface.activate reads it on Enter and
// emits both event.ButtonClicked and a protocol-native a2ui.ClientMessage (see
// render.go).
func (s *Surface) renderButton(c a2ui.Component, seen map[string]bool) string {
	var label string
	if c.Button.Child != "" {
		label = strings.TrimSpace(s.renderComponent(c.Button.Child, seen))
	}
	if label == "" {
		label = c.ID
	}
	chrome := "[ " + label + " ]"
	if c.Button.Variant == a2ui.ButtonVariantBorderless {
		chrome = label
	}
	if s.isFocused(c.ID) {
		return styleButtonFocused.Render(chrome)
	}
	return styleButton.Render(chrome)
}
