package render_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/render"
)

// cardRowComps builds a Card containing a Row with two text children. This
// exercises both the compact card (no border) and compact row (vertical stack)
// paths.
func cardRowComps() []a2ui.Component {
	return []a2ui.Component{
		{ID: "root", Card: &a2ui.CardComponent{Child: "row"}},
		{ID: "row", Row: &a2ui.RowComponent{Children: a2ui.ChildList{IDs: []string{"a", "b"}}}},
		text("a", "AAA"),
		text("b", "BBB"),
	}
}

// renderAt renders comps as a surface with the given width and returns the
// stripped (ANSI-free) output.
func renderAt(comps []a2ui.Component, width int) string {
	surf := render.NewSurface("s", comps)
	surf.SetSize(width, 24)
	return ansi.Strip(surf.View().Content)
}

// renderAtOpts is renderAt with functional options.
func renderAtOpts(comps []a2ui.Component, width int, opts ...render.Option) string {
	surf := render.NewSurface("s", comps, opts...)
	surf.SetSize(width, 24)
	return ansi.Strip(surf.View().Content)
}

// maxLineWidth returns the number of cells occupied by the longest line in s.
func maxLineWidth(s string) int {
	max := 0
	for _, ln := range strings.Split(s, "\n") {
		w := utf8.RuneCountInString(ln)
		if w > max {
			max = w
		}
	}
	return max
}

// TestCompactCardDropsBorder verifies that at width 30 (below the 40-col
// threshold) a Card renders without its rounded border.
func TestCompactCardDropsBorder(t *testing.T) {
	out := renderAt(cardRowComps(), 30)

	if strings.Contains(out, "╭") || strings.Contains(out, "╰") {
		t.Fatalf("compact card should NOT draw a border: %q", out)
	}
	if !strings.Contains(out, "AAA") || !strings.Contains(out, "BBB") {
		t.Fatalf("compact card missing child text: %q", out)
	}
}

// TestCompactCardStaysWithinWidth verifies that a compact card's output does
// not exceed the allocated width budget.
func TestCompactCardStaysWithinWidth(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Card: &a2ui.CardComponent{Child: "body"}},
		text("body", "This is a long line of text that would overflow a narrow panel if not wrapped properly"),
	}
	out := renderAt(comps, 30)

	if w := maxLineWidth(out); w > 30 {
		t.Fatalf("compact card output is %d cols wide (budget 30): %q", w, out)
	}
}

// TestCompactRowStacksVertically verifies that a Row in compact mode stacks
// its children vertically (AAA on one line, BBB on the next) rather than
// joining them horizontally.
func TestCompactRowStacksVertically(t *testing.T) {
	out := renderAt(cardRowComps(), 30)

	// In vertical stacking, AAA and BBB appear on separate lines.
	aIdx := strings.Index(out, "AAA")
	bIdx := strings.Index(out, "BBB")
	if aIdx < 0 || bIdx < 0 {
		t.Fatalf("output missing children: %q", out)
	}
	if aIdx > bIdx {
		t.Fatalf("AAA should render before BBB: %q", out)
	}
	if !strings.Contains(out[aIdx:bIdx], "\n") {
		t.Fatalf("compact row should stack AAA and BBB on separate lines: %q", out)
	}
}

// rowComps builds a bare Row with two text children (no Card wrapper) so the
// row's own layout — horizontal vs vertical — can be asserted without card
// chrome interfering.
func rowComps() []a2ui.Component {
	return []a2ui.Component{
		{ID: "root", Row: &a2ui.RowComponent{Children: a2ui.ChildList{IDs: []string{"a", "b"}}}},
		text("a", "AAA"),
		text("b", "BBB"),
	}
}

// TestNormalCardAt80HasBorder verifies that at width 80 the card still draws
// its border (the non-compact path is unchanged).
func TestNormalCardAt80HasBorder(t *testing.T) {
	out := renderAt(cardRowComps(), 80)

	if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
		t.Fatalf("non-compact card should draw a rounded border: %q", out)
	}
}

// TestNormalRowAt80IsHorizontal verifies that a bare Row at width 80 lays out
// children horizontally (AAA and BBB on the same line).
func TestNormalRowAt80IsHorizontal(t *testing.T) {
	out := renderAt(rowComps(), 80)

	aIdx := strings.Index(out, "AAA")
	bIdx := strings.Index(out, "BBB")
	if aIdx < 0 || bIdx < 0 {
		t.Fatalf("output missing children: %q", out)
	}
	if strings.Contains(out[aIdx:bIdx], "\n") {
		t.Fatalf("non-compact row should join children horizontally: %q", out)
	}
}

// TestWithCompactTrueForcesCompactAtWidth100 verifies that WithCompact(true)
// activates compact mode even when the surface width is well above the
// threshold.
func TestWithCompactTrueForcesCompactAtWidth100(t *testing.T) {
	out := renderAtOpts(cardRowComps(), 100, render.WithCompact(true))

	if strings.Contains(out, "╭") {
		t.Fatalf("WithCompact(true) should force borderless card even at width 100: %q", out)
	}

	aIdx := strings.Index(out, "AAA")
	bIdx := strings.Index(out, "BBB")
	if aIdx < 0 || bIdx < 0 {
		t.Fatalf("output missing children: %q", out)
	}
	if !strings.Contains(out[aIdx:bIdx], "\n") {
		t.Fatalf("WithCompact(true) should stack row vertically even at width 100: %q", out)
	}
}

// TestWithCompactFalseForcesNormalAtWidth20 verifies that WithCompact(false)
// keeps the normal rendering path even below the threshold.
func TestWithCompactFalseForcesNormalAtWidth20(t *testing.T) {
	out := renderAtOpts(cardRowComps(), 20, render.WithCompact(false))

	if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
		t.Fatalf("WithCompact(false) should draw border even at width 20: %q", out)
	}
}

// TestWithCompactThreshold verifies that WithCompactThreshold moves the
// auto-activation boundary. At width 50 with a threshold of 50, compact mode
// should be inactive; with a threshold of 60, it should be active.
func TestWithCompactThreshold(t *testing.T) {
	// Width 50, threshold 50: not compact (50 is not < 50).
	out := renderAtOpts(cardRowComps(), 50, render.WithCompactThreshold(50))
	if !strings.Contains(out, "╭") {
		t.Fatalf("width 50 with threshold 50 should not be compact: %q", out)
	}

	// Width 50, threshold 60: compact (50 < 60).
	out = renderAtOpts(cardRowComps(), 50, render.WithCompactThreshold(60))
	if strings.Contains(out, "╭") {
		t.Fatalf("width 50 with threshold 60 should be compact: %q", out)
	}
}

// TestCompactUnconstrainedWidth verifies that width 0 (unconstrained) never
// activates compact mode, regardless of the threshold.
func TestCompactUnconstrainedWidth(t *testing.T) {
	surf := render.NewSurface("s", cardRowComps(), render.WithCompactThreshold(100))
	out := ansi.Strip(surf.View().Content)

	if !strings.Contains(out, "╭") {
		t.Fatalf("unconstrained width (0) should not be compact even with low threshold: %q", out)
	}
}

// TestNonCompactOutputUnchanged verifies the byte-for-byte guarantee: adding
// WithCompact(false) at a wide width produces identical output to no options.
func TestNonCompactOutputUnchanged(t *testing.T) {
	comps := cardRowComps()

	// No options at width 80.
	plain := renderAt(comps, 80)

	// WithCompact(false) at width 80 — should be identical.
	disabled := renderAtOpts(comps, 80, render.WithCompact(false))

	if plain != disabled {
		t.Fatalf("WithCompact(false) changed wide output:\nplain:    %q\ndisabled: %q", plain, disabled)
	}
}

// TestCompactDoesNotLeakIntoNestedSubtree verifies the compact decision keys
// off the host-allocated width, not the per-subtree budget that withWidth
// narrows. At a top-level width of 43 (>= the 40-col threshold) a nested
// card→card→row must render fully normal: both cards keep their borders and
// the row stays horizontal, even though the inner budget dips below 40.
func TestCompactDoesNotLeakIntoNestedSubtree(t *testing.T) {
	// Card borders are the clean signal for whether compact activated: compact
	// drops them. (Row horizontality can't be asserted here — the nested card
	// budget width-wraps any row regardless of compact.)
	comps := []a2ui.Component{
		{ID: "root", Card: &a2ui.CardComponent{Child: "inner"}},
		{ID: "inner", Card: &a2ui.CardComponent{Child: "leaf"}},
		text("leaf", "hi"),
	}

	// Width 43 >= threshold: NOT compact anywhere — both cards keep borders,
	// even though the inner card's per-subtree budget (43-4-4=35) is below 40.
	out := renderAt(comps, 43)
	if got := strings.Count(out, "╭"); got != 2 {
		t.Fatalf("compact leaked into nested subtree: expected 2 borders at width 43, got %d:\n%s", got, out)
	}

	// Width 30 < threshold: compact throughout — both cards drop their borders.
	compactOut := renderAt(comps, 30)
	if got := strings.Count(compactOut, "╭"); got != 0 {
		t.Fatalf("expected 0 borders at compact width 30, got %d:\n%s", got, compactOut)
	}
}

// TestCompactHorizontalListStacks verifies a horizontal List stacks vertically
// in compact mode instead of overflowing the width budget.
func TestCompactHorizontalListStacks(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", List: &a2ui.ListComponent{
			Direction: a2ui.ListDirectionHorizontal,
			Children:  a2ui.ChildList{IDs: []string{"a", "b", "c"}},
		}},
		text("a", "AAAAAAAAAA"),
		text("b", "BBBBBBBBBB"),
		text("c", "CCCCCCCCCC"),
	}

	compactOut := renderAt(comps, 30)
	if w := maxLineWidth(compactOut); w > 30 {
		t.Fatalf("compact horizontal list should fit width 30, got max line %d:\n%s", w, compactOut)
	}
	if sameLine(compactOut, "AAAAAAAAAA", "BBBBBBBBBB") {
		t.Fatalf("compact horizontal list should stack, not stay on one line:\n%s", compactOut)
	}

	// At a wide width it stays horizontal (unchanged): all children on one line.
	wideOut := renderAt(comps, 80)
	if !sameLine(wideOut, "AAAAAAAAAA", "BBBBBBBBBB", "CCCCCCCCCC") {
		t.Fatalf("wide horizontal list should stay horizontal (one line):\n%s", wideOut)
	}
}

// sameLine reports whether some single line of s contains every substring.
func sameLine(s string, subs ...string) bool {
	for _, ln := range strings.Split(s, "\n") {
		all := true
		for _, sub := range subs {
			if !strings.Contains(ln, sub) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}
