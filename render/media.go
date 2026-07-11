package render

import a2ui "github.com/tmc/a2ui"

// Media components render as compact one-line placeholders — terminals do not
// display images, video, or audio. All placeholders use styleCaption and are
// left unwrapped/untruncated; the host wraps long URLs as it sees fit.

// renderImage renders an Image component as a placeholder: the glyph plus the
// best available description — the Description field when present and
// non-empty, otherwise the URL.
func (s *Surface) renderImage(c a2ui.Component) string {
	desc := ""
	if c.Image.Description != nil {
		desc = dynString(*c.Image.Description)
	}
	if desc == "" {
		desc = dynString(c.Image.URL)
	}
	return styleCaption.Render("🖼 " + desc)
}

// renderIcon renders an Icon component as its name wrapped in angle quotes,
// e.g. ⟨accountCircle⟩. IconNameOrPath is a union: a well-known name renders
// verbatim; a custom SVG path renders as "svg" (raw path data is not readable
// text); a binding renders as the "{binding}" placeholder used by dynString
// until the data model lands.
func (s *Surface) renderIcon(c a2ui.Component) string {
	name := "icon"
	switch n := c.Icon.Name; {
	case n.Name != nil:
		name = string(*n.Name)
	case n.SVGPath != nil:
		name = "svg"
	case n.Binding != nil:
		name = "{binding}"
	}
	return styleCaption.Render("⟨" + name + "⟩")
}

// renderVideo renders a Video component as a placeholder: a play glyph plus
// the URL (VideoComponent carries no title or description field).
func (s *Surface) renderVideo(c a2ui.Component) string {
	return styleCaption.Render("▶ " + dynString(c.Video.URL))
}

// renderAudio renders an AudioPlayer component as a placeholder: a note glyph
// plus the Description when present and non-empty, otherwise the URL.
func (s *Surface) renderAudio(c a2ui.Component) string {
	desc := ""
	if c.AudioPlayer.Description != nil {
		desc = dynString(*c.AudioPlayer.Description)
	}
	if desc == "" {
		desc = dynString(c.AudioPlayer.URL)
	}
	return styleCaption.Render("♪ " + desc)
}
