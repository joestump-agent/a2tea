package render

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Chrome styles shared by the component renderers.
//
// Deliberately monochrome: a2tea surfaces render inside a host TUI (crush)
// that owns the color theme, so component chrome sticks to attributes that
// read correctly on any background — borders, bold, faint, and reverse-video
// for focus. Do not add hardcoded colors here.
var (
	// styleCardBorder draws the rounded box around a Card's child.
	styleCardBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)

	// styleHeading renders Text variants h1–h3.
	styleHeading = lipgloss.NewStyle().Bold(true)
	// styleSubheading renders Text variants h4–h5.
	styleSubheading = lipgloss.NewStyle().Bold(true).Faint(true)
	// styleCaption renders caption-variant Text and other secondary chrome
	// (field labels, media placeholders).
	styleCaption = lipgloss.NewStyle().Faint(true)

	// styleButton renders an idle button's label.
	styleButton = lipgloss.NewStyle().Bold(true)
	// styleButtonFocused renders the focused button's label.
	styleButtonFocused = lipgloss.NewStyle().Bold(true).Reverse(true)
)

// wrapTo wraps s to width w when w > 0; otherwise returns s unchanged.
func wrapTo(s string, w int) string {
	if w <= 0 {
		return s
	}
	return lipgloss.NewStyle().Width(w).Render(s)
}

// hr draws a horizontal rule of width w (minimum 1 cell).
func hr(w int) string {
	if w < 1 {
		w = 1
	}
	return strings.Repeat("─", w)
}
