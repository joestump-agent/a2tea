package render_test

import (
	"strings"
	"testing"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/render"
)

// templateSurface builds a Column whose children come from a ChildList
// template: one "item" Text per element of the /items data-model list, with
// the item's text bound to the element's "name" field.
func templateSurface() *render.Surface {
	return render.NewSurface("s1", []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{
			Template: &a2ui.ChildTemplate{ComponentID: "item", Path: "/items"},
		}}},
		{ID: "item", Text: &a2ui.TextComponent{Text: a2ui.StringBinding("/name")}},
	})
}

// itemsUpdate builds an updateDataModel message setting /items to the given
// list of element names.
func itemsUpdate(names ...string) a2ui.ServerMessage {
	items := make([]any, len(names))
	for i, n := range names {
		items[i] = map[string]any{"name": n}
	}
	return a2ui.ServerMessage{UpdateDataModel: &a2ui.UpdateDataModel{
		SurfaceID: "s1", Path: "/items", Value: items,
	}}
}

// TestTemplateChildrenRender verifies that a Column with a template child
// list renders one template-component instance per data-model element, with
// bindings resolved against each element.
func TestTemplateChildrenRender(t *testing.T) {
	s := templateSurface()
	s.Apply([]a2ui.ServerMessage{itemsUpdate("Ada", "Grace", "Katherine")})

	view := s.View().Content
	for _, want := range []string{"Ada", "Grace", "Katherine"} {
		if !strings.Contains(view, want) {
			t.Errorf("missing template instance %q; view = %q", want, view)
		}
	}
	if strings.Contains(view, "{binding}") {
		t.Errorf("unresolved binding placeholder in template instance; view = %q", view)
	}
	// Instances stack vertically in the Column: Ada above Grace.
	if strings.Index(view, "Ada") > strings.Index(view, "Grace") {
		t.Errorf("template instances out of list order; view = %q", view)
	}
}

// TestTemplateChildrenGrowShrink verifies that updateDataModel on the bound
// list re-renders with added and removed children.
func TestTemplateChildrenGrowShrink(t *testing.T) {
	s := templateSurface()
	s.Apply([]a2ui.ServerMessage{itemsUpdate("Ada", "Grace")})
	view := s.View().Content
	if !strings.Contains(view, "Ada") || !strings.Contains(view, "Grace") {
		t.Fatalf("initial instances missing; view = %q", view)
	}

	// Grow: a third element appears.
	s.Apply([]a2ui.ServerMessage{itemsUpdate("Ada", "Grace", "Katherine")})
	view = s.View().Content
	if !strings.Contains(view, "Katherine") {
		t.Errorf("added element did not render; view = %q", view)
	}

	// Shrink: only one element remains.
	s.Apply([]a2ui.ServerMessage{itemsUpdate("Grace")})
	view = s.View().Content
	if strings.Contains(view, "Ada") || strings.Contains(view, "Katherine") {
		t.Errorf("removed elements still render; view = %q", view)
	}
	if !strings.Contains(view, "Grace") {
		t.Errorf("surviving element missing; view = %q", view)
	}
}

// TestTemplateChildrenEmptyList verifies that an empty bound list — and a
// list whose data hasn't arrived at all — renders no children and no
// placeholder chrome.
func TestTemplateChildrenEmptyList(t *testing.T) {
	s := templateSurface()

	// No data model yet: the template expands to nothing.
	if view := s.View().Content; strings.Contains(view, "a2tea:") || strings.Contains(view, "{binding}") {
		t.Errorf("template with absent data should render nothing; view = %q", view)
	}

	// Explicitly empty list: still nothing.
	s.Apply([]a2ui.ServerMessage{itemsUpdate()})
	if view := s.View().Content; strings.Contains(view, "a2tea:") || strings.Contains(view, "{binding}") {
		t.Errorf("template with empty list should render nothing; view = %q", view)
	}
}

// TestTemplateChildrenCycle verifies that a template component referencing
// its own container trips the cycle guard instead of recursing forever.
func TestTemplateChildrenCycle(t *testing.T) {
	s := render.NewSurface("s1", []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{
			Template: &a2ui.ChildTemplate{ComponentID: "item", Path: "/items"},
		}}},
		// The template component wraps its own container: root -> item -> root.
		{ID: "item", Card: &a2ui.CardComponent{Child: "root"}},
	})
	s.Apply([]a2ui.ServerMessage{{UpdateDataModel: &a2ui.UpdateDataModel{
		SurfaceID: "s1", Path: "/items", Value: []any{"a", "b"},
	}}})

	view := s.View().Content
	if !strings.Contains(view, "cycle") {
		t.Fatalf("template cycle through the container not caught; view = %q", view)
	}
}

// TestTemplateScalarElements verifies that a list of plain strings works: a
// binding path of "/" inside the template resolves to the element itself.
func TestTemplateScalarElements(t *testing.T) {
	s := render.NewSurface("s1", []a2ui.Component{
		{ID: "root", List: &a2ui.ListComponent{Children: a2ui.ChildList{
			Template: &a2ui.ChildTemplate{ComponentID: "item", Path: "/tags"},
		}}},
		{ID: "item", Text: &a2ui.TextComponent{Text: a2ui.StringBinding("/")}},
	})
	// []string exercises the reflection path in asList — a Go host can feed
	// typed slices, not just JSON-decoded []any.
	s.Apply([]a2ui.ServerMessage{{UpdateDataModel: &a2ui.UpdateDataModel{
		SurfaceID: "s1", Path: "/tags", Value: []string{"alpha", "beta"},
	}}})

	view := s.View().Content
	if !strings.Contains(view, "alpha") || !strings.Contains(view, "beta") {
		t.Errorf("scalar elements did not render; view = %q", view)
	}
	// Vertical List chrome still applies to template instances.
	if !strings.Contains(view, "• alpha") {
		t.Errorf("template instance missing list bullet; view = %q", view)
	}
}

// TestTemplateScopeFallsBackToDataModel verifies that a binding that does not
// resolve against the current element falls back to the surface data model,
// so instances can mix per-element and shared values.
func TestTemplateScopeFallsBackToDataModel(t *testing.T) {
	s := render.NewSurface("s1", []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{
			Template: &a2ui.ChildTemplate{ComponentID: "item", Path: "/items"},
		}}},
		{ID: "item", Row: &a2ui.RowComponent{Children: a2ui.ChildList{IDs: []string{"name", "shared"}}}},
		{ID: "name", Text: &a2ui.TextComponent{Text: a2ui.StringBinding("/name")}},
		{ID: "shared", Text: &a2ui.TextComponent{Text: a2ui.StringBinding("/unit")}},
	})
	s.Apply([]a2ui.ServerMessage{
		{UpdateDataModel: &a2ui.UpdateDataModel{SurfaceID: "s1", Path: "/unit", Value: "kg"}},
		itemsUpdate("Ada"),
	})

	view := s.View().Content
	if !strings.Contains(view, "Ada") {
		t.Errorf("element-scoped binding missing; view = %q", view)
	}
	if !strings.Contains(view, "kg") {
		t.Errorf("surface data model fallback missing; view = %q", view)
	}
}

// TestTemplateNestedPath verifies that a binding path with multiple segments
// traverses nested maps inside the element.
func TestTemplateNestedPath(t *testing.T) {
	s := render.NewSurface("s1", []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{
			Template: &a2ui.ChildTemplate{ComponentID: "item", Path: "/items"},
		}}},
		{ID: "item", Text: &a2ui.TextComponent{Text: a2ui.StringBinding("/meta/label")}},
	})
	s.Apply([]a2ui.ServerMessage{{UpdateDataModel: &a2ui.UpdateDataModel{
		SurfaceID: "s1", Path: "/items", Value: []any{
			map[string]any{"meta": map[string]any{"label": "deep"}},
		},
	}}})

	if view := s.View().Content; !strings.Contains(view, "deep") {
		t.Errorf("nested element path did not resolve; view = %q", view)
	}
}

// TestTemplateNonListValue verifies that a template path resolving to a
// non-list value renders the diagnostic placeholder instead of children.
func TestTemplateNonListValue(t *testing.T) {
	s := templateSurface()
	s.Apply([]a2ui.ServerMessage{{UpdateDataModel: &a2ui.UpdateDataModel{
		SurfaceID: "s1", Path: "/items", Value: "not-a-list",
	}}})

	if view := s.View().Content; !strings.Contains(view, "is not a list") {
		t.Errorf("non-list template value should render a diagnostic; view = %q", view)
	}
}

// TestTemplateRootDerivation verifies that the template component counts as
// referenced when deriving the surface root, so the container — not the
// template component — is the root even though no explicit ID references the
// template.
func TestTemplateRootDerivation(t *testing.T) {
	// Declare the template component first so a naive "first component" root
	// fallback would pick it over the container.
	s := render.NewSurface("s1", []a2ui.Component{
		{ID: "item", Text: &a2ui.TextComponent{Text: a2ui.StringBinding("/name")}},
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{
			Template: &a2ui.ChildTemplate{ComponentID: "item", Path: "/items"},
		}}},
	})
	s.Apply([]a2ui.ServerMessage{itemsUpdate("Ada", "Grace")})

	view := s.View().Content
	// If "item" were the root, the view would be a single unscoped "{binding}".
	if !strings.Contains(view, "Ada") || !strings.Contains(view, "Grace") {
		t.Errorf("column with template children should be the root; view = %q", view)
	}
}

// TestStaticChildrenUnaffected verifies the static ChildList form still
// renders explicit IDs in order with no data model present.
func TestStaticChildrenUnaffected(t *testing.T) {
	s := render.NewSurface("s1", []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"a", "b"}}}},
		{ID: "a", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Alpha")}},
		{ID: "b", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Beta")}},
	})
	view := s.View().Content
	if !strings.Contains(view, "Alpha") || !strings.Contains(view, "Beta") {
		t.Errorf("static children missing; view = %q", view)
	}
	if strings.Index(view, "Alpha") > strings.Index(view, "Beta") {
		t.Errorf("static children out of order; view = %q", view)
	}
}
