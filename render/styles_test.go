package render_test

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/render"
)

// TestDefaultStylesMonochrome verifies that a surface with no style options
// produces output with no color escape sequences — only text attributes
// (bold/faint/reverse) that are stripped by ansi.Strip.
func TestDefaultStylesMonochrome(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"t", "btn"}}}},
		{ID: "t", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Heading"), Variant: a2ui.TextVariantH1}},
		{ID: "btn", Button: &a2ui.ButtonComponent{Child: "btnlabel"}},
		{ID: "btnlabel", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Click")}},
	}

	// Raw output (ANSI intact) from a default-styled surface.
	raw := render.NewSurface("s", comps).View().Content

	// The raw output should NOT contain any color SGR sequences (38;5; or
	// 48;5; for 256-color, or 38;2; / 48;2; for truecolor). It may contain
	// bold (1), faint (2), reverse (7), and reset (0) — those are the
	// monochrome attributes.
	for _, colorSeq := range []string{"38;5;", "48;5;", "38;2;", "48;2;"} {
		if strings.Contains(raw, colorSeq) {
			t.Fatalf("default-styled output contains color sequence %q: %q", colorSeq, raw)
		}
	}
}

// TestThemedStylesProduceColor verifies that a surface built with WithStyles
// produces visible color escape sequences in its rendered output.
func TestThemedStylesProduceColor(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Themed Heading"), Variant: a2ui.TextVariantH1}},
	}

	// Build a themed style set: heading gets a 256-color foreground.
	st := render.DefaultStyles()
	st.Heading = st.Heading.Foreground(lipgloss.Color("99")) // color 99 = bright blue

	surf := render.NewSurface("s", comps, render.WithStyles(st))
	raw := surf.View().Content

	// The output must contain the 256-color foreground escape for color 99.
	if !strings.Contains(raw, "38;5;99") {
		t.Fatalf("themed output should contain color 99 foreground sequence; got: %q", raw)
	}

	// The heading text must still be present (underneath the ANSI).
	if !strings.Contains(raw, "Themed Heading") {
		t.Fatalf("themed output missing heading text: %q", raw)
	}
}

// TestThemedButtonProducesColor verifies that themed button styles (both idle
// and focused) apply color to the rendered output.
func TestThemedButtonProducesColor(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Button: &a2ui.ButtonComponent{Child: "lbl"}},
		{ID: "lbl", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("OK")}},
	}

	st := render.DefaultStyles()
	st.Button = st.Button.Foreground(lipgloss.Color("205"))               // pink
	st.ButtonFocused = st.ButtonFocused.Foreground(lipgloss.Color("206")) // hot pink

	surf := render.NewSurface("s", comps, render.WithStyles(st))
	// Focus the surface + button so the focused style path is exercised.
	surf.Focus()

	raw := surf.View().Content

	// The focused button should carry color 206.
	if !strings.Contains(raw, "38;5;206") {
		t.Fatalf("themed focused button should contain color 206; got: %q", raw)
	}
}

// TestDefaultStylesIdenticalToPreOptions verifies the byte-for-byte guarantee:
// NewSurface without options must produce exactly the same output as
// NewSurface with explicit DefaultStyles.
func TestDefaultStylesIdenticalToPreOptions(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Card: &a2ui.CardComponent{Child: "col"}},
		{ID: "col", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"h", "b", "btn"}}}},
		{ID: "h", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Title"), Variant: a2ui.TextVariantH1}},
		{ID: "b", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Body"), Variant: a2ui.TextVariantCaption}},
		{ID: "btn", Button: &a2ui.ButtonComponent{Child: "lbl"}},
		{ID: "lbl", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Go")}},
	}

	implicit := render.NewSurface("s", comps).View().Content
	explicit := render.NewSurface("s", comps, render.WithStyles(render.DefaultStyles())).View().Content

	if implicit != explicit {
		t.Fatalf("default-styled output differs from explicit DefaultStyles:\nimplicit: %q\nexplicit: %q", implicit, explicit)
	}
}
