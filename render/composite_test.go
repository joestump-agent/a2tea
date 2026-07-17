package render_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/event"
	"github.com/joestump-agent/a2tea/render"
)

// TestApplyFocusPreserved verifies that focus is preserved across a
// component update when the focused button survives.
func TestApplyFocusPreserved(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"btn1", "btn2"}}}},
		{ID: "btn1", Button: &a2ui.ButtonComponent{Child: "l1"}},
		{ID: "btn2", Button: &a2ui.ButtonComponent{Child: "l2"}},
		{ID: "l1", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("First")}},
		{ID: "l2", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Second")}},
	}
	s := render.NewSurface("s1", comps)
	s.Focus()

	// Tab to the second button (index 1 = btn2).
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	// Apply an update that adds a new component but leaves both buttons intact.
	alive := s.Apply([]a2ui.ServerMessage{
		{
			UpdateComponents: &a2ui.UpdateComponents{
				SurfaceID: "s1",
				Components: []a2ui.Component{
					{ID: "extra", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Extra")}},
				},
			},
		},
	})
	if !alive {
		t.Fatal("Apply reported surface as not alive")
	}

	// Focus should still be on btn2. Activate it and check the emitted event.
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter after Apply produced nil cmd")
	}
	msg := cmd()
	ev, ok := msg.(event.ButtonClicked)
	if !ok {
		t.Fatalf("cmd produced %T, want event.ButtonClicked", msg)
	}
	if ev.ID != "btn2" {
		t.Fatalf("focus not preserved: activated %q, want btn2", ev.ID)
	}
}

// TestApplyFocusResetsWhenGone verifies that focus resets to the first
// button when the focused button is removed by an update.
func TestApplyFocusResetsWhenGone(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"btn1", "btn2"}}}},
		{ID: "btn1", Button: &a2ui.ButtonComponent{Child: "l1"}},
		{ID: "btn2", Button: &a2ui.ButtonComponent{Child: "l2"}},
		{ID: "l1", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("First")}},
		{ID: "l2", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Second")}},
	}
	s := render.NewSurface("s1", comps)
	s.Focus()

	// Tab to btn2 (index 1).
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	// Replace btn2 with a Text component — it's no longer a button.
	s.Apply([]a2ui.ServerMessage{
		{
			UpdateComponents: &a2ui.UpdateComponents{
				SurfaceID: "s1",
				Components: []a2ui.Component{
					{ID: "btn2", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Not a button")}},
				},
			},
		},
	})

	// Focus should reset to btn1 (the only remaining button).
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter produced nil cmd")
	}
	msg := cmd()
	ev, ok := msg.(event.ButtonClicked)
	if !ok {
		t.Fatalf("cmd produced %T, want event.ButtonClicked", msg)
	}
	if ev.ID != "btn1" {
		t.Fatalf("focus should reset to btn1; activated %q", ev.ID)
	}
}

// TestApplyDataModelResolvesBinding verifies that a data-model update on the
// surface resolves a bound Text component's value.
func TestApplyDataModelResolvesBinding(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Card: &a2ui.CardComponent{Child: "t"}},
		{ID: "t", Text: &a2ui.TextComponent{Text: a2ui.StringBinding("/status")}},
	}
	s := render.NewSurface("s1", comps)

	// Before data model update, binding renders as placeholder.
	view := s.View().Content
	if !strings.Contains(view, "{binding}") {
		t.Fatalf("expected {binding} before data model update; got %q", view)
	}

	// Apply data model update.
	s.Apply([]a2ui.ServerMessage{
		{
			UpdateDataModel: &a2ui.UpdateDataModel{
				SurfaceID: "s1",
				Path:      "/status",
				Value:     "online",
			},
		},
	})

	// After update, the binding should resolve.
	view = s.View().Content
	if !strings.Contains(view, "online") {
		t.Fatalf("expected resolved value 'online'; got %q", view)
	}
	if strings.Contains(view, "{binding}") {
		t.Fatalf("unexpected {binding} placeholder after data model update; got %q", view)
	}
}

// TestApplyComponentsMerge verifies that applying a second updateComponents
// merges by ID — new components appear alongside existing ones, and updates
// to existing components replace them.
func TestApplyComponentsMerge(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"a", "b"}}}},
		{ID: "a", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Alpha")}},
		{ID: "b", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Beta")}},
	}
	s := render.NewSurface("s1", comps)

	// Apply an update that replaces "a" and adds "c".
	s.Apply([]a2ui.ServerMessage{
		{
			UpdateComponents: &a2ui.UpdateComponents{
				SurfaceID: "s1",
				Components: []a2ui.Component{
					{ID: "a", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Updated")}},
					{ID: "c", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Charlie")}},
				},
			},
		},
	})

	view := s.View().Content

	// "Beta" should survive (not clobbered).
	if !strings.Contains(view, "Beta") {
		t.Errorf("sibling component was clobbered; view = %q", view)
	}
	// "Alpha" should be replaced by "Updated".
	if strings.Contains(view, "Alpha") {
		t.Errorf("old text should be gone; view = %q", view)
	}
	if !strings.Contains(view, "Updated") {
		t.Errorf("updated text missing; view = %q", view)
	}
}

// TestApplyDeleteSurface verifies that applying a deleteSurface clears the
// surface state and returns false.
func TestApplyDeleteSurface(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Hi")}},
	}
	s := render.NewSurface("s1", comps)

	alive := s.Apply([]a2ui.ServerMessage{
		{DeleteSurface: &a2ui.DeleteSurface{SurfaceID: "s1"}},
	})
	if alive {
		t.Fatal("Apply returned true after deleteSurface, want false")
	}

	view := s.View().Content
	if !strings.Contains(view, "empty surface") {
		t.Errorf("expected empty surface after delete; view = %q", view)
	}
}

// TestApplyDeleteDifferentSurface verifies that a deleteSurface for a
// different surface ID does not affect this surface.
func TestApplyDeleteDifferentSurface(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Hi")}},
	}
	s := render.NewSurface("s1", comps)

	alive := s.Apply([]a2ui.ServerMessage{
		{DeleteSurface: &a2ui.DeleteSurface{SurfaceID: "other"}},
	})
	if !alive {
		t.Fatal("Apply returned false for unrelated delete, want true")
	}

	view := s.View().Content
	if !strings.Contains(view, "Hi") {
		t.Errorf("surface should survive unrelated delete; view = %q", view)
	}
}

// TestApplyRootDeterministic verifies that when an update makes the previous
// root a referenced child and introduces multiple root-eligible components,
// the derived root is stable across renders (map iteration order is random).
func TestApplyRootDeterministic(t *testing.T) {
	var first string
	for i := 0; i < 50; i++ {
		s := render.NewSurface("s1", []a2ui.Component{
			{ID: "orig", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("ORIG")}},
		})
		// "wrap" now references "orig"; "loose" is a second unreferenced root.
		s.Apply([]a2ui.ServerMessage{
			{UpdateComponents: &a2ui.UpdateComponents{SurfaceID: "s1", Components: []a2ui.Component{
				{ID: "wrap", Card: &a2ui.CardComponent{Child: "orig"}},
				{ID: "loose", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("LOOSE")}},
			}}},
		})
		view := s.View().Content
		if first == "" {
			first = view
			continue
		}
		if view != first {
			t.Fatalf("non-deterministic root selection across renders:\nfirst:\n%q\ngot:\n%q", first, view)
		}
	}
}

// TestApplyIgnoresOtherSurfaceComponents verifies that an updateComponents
// for a different surface does not clobber this surface's components.
func TestApplyIgnoresOtherSurfaceComponents(t *testing.T) {
	s := render.NewSurface("s1", []a2ui.Component{
		{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("FROM-S1")}},
	})

	alive := s.Apply([]a2ui.ServerMessage{
		{UpdateComponents: &a2ui.UpdateComponents{SurfaceID: "s2", Components: []a2ui.Component{
			{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("FROM-S2")}},
		}}},
	})
	if !alive {
		t.Fatal("surface should stay alive after an unrelated update")
	}

	view := s.View().Content
	if !strings.Contains(view, "FROM-S1") {
		t.Errorf("s1 content clobbered by s2's update; view = %q", view)
	}
	if strings.Contains(view, "FROM-S2") {
		t.Errorf("s2's component leaked into s1; view = %q", view)
	}
}

// TestApplyIgnoresOtherSurfaceDataModel verifies that an updateDataModel for
// a different surface does not resolve bindings on this surface.
func TestApplyIgnoresOtherSurfaceDataModel(t *testing.T) {
	s := render.NewSurface("s1", []a2ui.Component{
		{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringBinding("/status")}},
	})

	s.Apply([]a2ui.ServerMessage{
		{UpdateDataModel: &a2ui.UpdateDataModel{SurfaceID: "OTHER", Path: "/status", Value: "leaked"}},
	})

	view := s.View().Content
	if strings.Contains(view, "leaked") {
		t.Errorf("other surface's data model leaked into s1; view = %q", view)
	}
	if !strings.Contains(view, "{binding}") {
		t.Errorf("binding should stay unresolved; view = %q", view)
	}
}

// TestApplyEmptySurfaceIDTargetsThisSurface verifies the lenient path: a
// message with an empty SurfaceID applies to the current surface, since some
// producers omit the field when only one surface exists.
func TestApplyEmptySurfaceIDTargetsThisSurface(t *testing.T) {
	s := render.NewSurface("s1", []a2ui.Component{
		{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringBinding("/status")}},
	})

	s.Apply([]a2ui.ServerMessage{
		{UpdateDataModel: &a2ui.UpdateDataModel{Path: "/status", Value: "online"}},
	})

	view := s.View().Content
	if !strings.Contains(view, "online") {
		t.Errorf("empty-SurfaceID data-model update should apply; view = %q", view)
	}
}

// TestApplyEmptySurfaceIDDeleteIsIgnored verifies the strict half of the
// scoping rule: deleteSurface never fires on an empty SurfaceID.
func TestApplyEmptySurfaceIDDeleteIsIgnored(t *testing.T) {
	s := render.NewSurface("s1", []a2ui.Component{
		{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Hi")}},
	})

	alive := s.Apply([]a2ui.ServerMessage{
		{DeleteSurface: &a2ui.DeleteSurface{}},
	})
	if !alive {
		t.Fatal("empty-SurfaceID delete must not clear the surface")
	}
}

// TestApplyDeleteThenRecreate verifies that a delete followed by an
// updateComponents in the same batch re-creates the surface instead of the
// batch dead-ending at the delete.
func TestApplyDeleteThenRecreate(t *testing.T) {
	s := render.NewSurface("s1", []a2ui.Component{
		{ID: "root", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("OLD")}},
	})

	alive := s.Apply([]a2ui.ServerMessage{
		{DeleteSurface: &a2ui.DeleteSurface{SurfaceID: "s1"}},
		{UpdateComponents: &a2ui.UpdateComponents{SurfaceID: "s1", Components: []a2ui.Component{
			{ID: "root2", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("RECREATED")}},
		}}},
	})
	if !alive {
		t.Fatal("delete-then-recreate should leave the surface alive")
	}

	view := s.View().Content
	if !strings.Contains(view, "RECREATED") {
		t.Errorf("recreated content missing; view = %q", view)
	}
	if strings.Contains(view, "OLD") {
		t.Errorf("pre-delete content survived; view = %q", view)
	}
}

// TestApplyDeleteClearsDataModelAndEdits verifies delete wipes the data model
// and pending field edits, so a re-created surface starts clean.
func TestApplyDeleteClearsDataModelAndEdits(t *testing.T) {
	val := a2ui.StringLiteral("seed")
	s := render.NewSurface("s1", []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"f", "b"}}}},
		{ID: "f", TextField: &a2ui.TextFieldComponent{Value: &val}},
		{ID: "b", Text: &a2ui.TextComponent{Text: a2ui.StringBinding("/status")}},
	})
	s.Focus()
	s.Update(tea.KeyPressMsg{Code: 'X', Text: "X"}) // edit the field
	s.Apply([]a2ui.ServerMessage{
		{UpdateDataModel: &a2ui.UpdateDataModel{SurfaceID: "s1", Path: "/status", Value: "online"}},
	})

	s.Apply([]a2ui.ServerMessage{
		{DeleteSurface: &a2ui.DeleteSurface{SurfaceID: "s1"}},
		{UpdateComponents: &a2ui.UpdateComponents{SurfaceID: "s1", Components: []a2ui.Component{
			{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"f", "b"}}}},
			{ID: "f", TextField: &a2ui.TextFieldComponent{Value: &val}},
			{ID: "b", Text: &a2ui.TextComponent{Text: a2ui.StringBinding("/status")}},
		}}},
	})

	if vals := s.FieldValues(); vals["f"] != "seed" {
		t.Errorf("field edit survived delete: FieldValues[f] = %v, want seed literal", vals["f"])
	}
	view := s.View().Content
	if strings.Contains(view, "online") {
		t.Errorf("data model survived delete; view = %q", view)
	}
}
