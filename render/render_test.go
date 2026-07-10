package render_test

import (
	"strings"
	"testing"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/render"
)

func text(id, s string) a2ui.Component {
	return a2ui.Component{ID: id, Text: &a2ui.TextComponent{Text: a2ui.StringLiteral(s)}}
}

// surface: card(root) -> column(col) -> [title, body]
func sampleComponents() []a2ui.Component {
	return []a2ui.Component{
		{ID: "root", Card: &a2ui.CardComponent{Child: "col"}},
		{ID: "col", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"title", "body"}}}},
		text("title", "Title Line"),
		text("body", "Body line"),
	}
}

func TestSurfaceRendersTree(t *testing.T) {
	s := render.NewSurface("s", sampleComponents())
	out := s.View().Content

	// Both text leaves appear, and the column stacks them on separate lines.
	if !strings.Contains(out, "Title Line") || !strings.Contains(out, "Body line") {
		t.Fatalf("rendered surface missing text: %q", out)
	}
	if !strings.Contains(out, "Title Line\nBody line") {
		t.Fatalf("column should stack children on separate lines: %q", out)
	}
}

func TestSurfaceRootDetection(t *testing.T) {
	// Declaration order deliberately does NOT put the root first: the root is
	// the component nothing else references as a child.
	comps := []a2ui.Component{
		text("title", "Only Child"),
		{ID: "root", Card: &a2ui.CardComponent{Child: "title"}},
	}
	out := render.NewSurface("s", comps).View().Content
	if !strings.Contains(out, "Only Child") {
		t.Fatalf("root not resolved to the card: %q", out)
	}
}

func TestSurfaceSharedChildIsNotACycle(t *testing.T) {
	// "shared" is legally referenced by two parents (adjacency-list reuse). It
	// must render at both sites, not trip the cycle guard on the second.
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"row1", "row2"}}}},
		{ID: "row1", Row: &a2ui.RowComponent{Children: a2ui.ChildList{IDs: []string{"shared"}}}},
		{ID: "row2", Row: &a2ui.RowComponent{Children: a2ui.ChildList{IDs: []string{"shared"}}}},
		text("shared", "twice"),
	}
	out := render.NewSurface("s", comps).View().Content
	if strings.Contains(out, "cycle") {
		t.Fatalf("shared child wrongly flagged as a cycle: %q", out)
	}
	if got := strings.Count(out, "twice"); got != 2 {
		t.Fatalf("shared child should render at both parents; %q rendered %d times", "twice", got)
	}
}

func TestSurfaceGenuineCycleIsCaught(t *testing.T) {
	// root -> a -> root is a real ancestor loop and must still be caught.
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"a"}}}},
		{ID: "a", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"root"}}}},
	}
	out := render.NewSurface("s", comps).View().Content
	if !strings.Contains(out, "cycle") {
		t.Fatalf("genuine cycle not caught: %q", out)
	}
}

func TestSurfaceEmpty(t *testing.T) {
	out := render.NewSurface("s", nil).View().Content
	if !strings.Contains(out, "empty surface") {
		t.Fatalf("empty surface = %q, want a placeholder", out)
	}
}

func TestSurfaceMissingChildIsFlagged(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Card: &a2ui.CardComponent{Child: "nope"}},
	}
	out := render.NewSurface("s", comps).View().Content
	if !strings.Contains(out, "missing component") {
		t.Fatalf("dangling child ref should be flagged: %q", out)
	}
}

func TestKindOf(t *testing.T) {
	cases := []struct {
		c    a2ui.Component
		want string
	}{
		{a2ui.Component{Text: &a2ui.TextComponent{}}, "text"},
		{a2ui.Component{Card: &a2ui.CardComponent{}}, "card"},
		{a2ui.Component{Button: &a2ui.ButtonComponent{}}, "button"},
		{a2ui.Component{Row: &a2ui.RowComponent{}}, "row"},
		{a2ui.Component{Column: &a2ui.ColumnComponent{}}, "column"},
		{a2ui.Component{}, "unknown"},
	}
	for _, tc := range cases {
		if got := render.KindOf(tc.c); got != tc.want {
			t.Errorf("KindOf(%+v) = %q, want %q", tc.c, got, tc.want)
		}
	}
}
