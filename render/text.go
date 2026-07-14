package render

import a2ui "github.com/tmc/a2ui"

// renderText renders a Text component: the resolved text styled per variant
// (h1–h3 heading, h4–h5 subheading, caption faint; body and unknown variants
// plain), wrapped to the surface width.
func (s *Surface) renderText(c a2ui.Component) string {
	text := s.dynString(c.Text.Text)
	switch c.Text.Variant {
	case a2ui.TextVariantH1, a2ui.TextVariantH2, a2ui.TextVariantH3:
		text = s.styles.Heading.Render(text)
	case a2ui.TextVariantH4, a2ui.TextVariantH5:
		text = s.styles.Subheading.Render(text)
	case a2ui.TextVariantCaption:
		text = s.styles.Caption.Render(text)
	}
	return wrapTo(text, s.width)
}
