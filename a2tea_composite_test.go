package a2tea_test

import (
	"errors"
	"strings"
	"testing"

	a2ui "github.com/tmc/a2ui"

	"github.com/charmbracelet/x/ansi"
	"github.com/joestump-agent/a2tea"
)

func updateMsg(sid string, comps ...a2ui.Component) a2ui.ServerMessage {
	return a2ui.ServerMessage{
		UpdateComponents: &a2ui.UpdateComponents{
			SurfaceID:  sid,
			Components: comps,
		},
	}
}

func dataMsg(sid, path string, value any) a2ui.ServerMessage {
	return a2ui.ServerMessage{
		UpdateDataModel: &a2ui.UpdateDataModel{
			SurfaceID: sid,
			Path:      path,
			Value:     value,
		},
	}
}

func deleteMsg(sid string) a2ui.ServerMessage {
	return a2ui.ServerMessage{
		DeleteSurface: &a2ui.DeleteSurface{SurfaceID: sid},
	}
}

func txt(id, val string) a2ui.Component {
	return a2ui.Component{ID: id, Text: &a2ui.TextComponent{Text: a2ui.StringLiteral(val)}}
}

func boundText(id, path string) a2ui.Component {
	return a2ui.Component{ID: id, Text: &a2ui.TextComponent{Text: a2ui.StringBinding(path)}}
}

func renderStr(m a2ui.ServerMessage) string {
	model, err := a2tea.Render([]a2ui.ServerMessage{m})
	if err != nil {
		return ""
	}
	return ansi.Strip(model.View().Content)
}

// TestCompositeTwoUpdatesNoClobber verifies that two updateComponents
// messages touching different components both take effect — the second
// update does not clobber the first.
func TestCompositeTwoUpdatesNoClobber(t *testing.T) {
	msgs := []a2ui.ServerMessage{
		updateMsg("s",
			a2ui.Component{ID: "root", Card: &a2ui.CardComponent{Child: "col"}},
			a2ui.Component{ID: "col", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"a", "b"}}}},
			txt("a", "First"),
		),
		// Second update only touches "b" — "a" must survive.
		updateMsg("s",
			txt("b", "Second"),
		),
	}

	model, err := a2tea.Render(msgs)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	view := ansi.Strip(model.View().Content)

	if !strings.Contains(view, "First") {
		t.Errorf("first component was clobbered; view = %q", view)
	}
	if !strings.Contains(view, "Second") {
		t.Errorf("second component missing; view = %q", view)
	}
}

// TestCompositeUpdateExistingComponent verifies that updating an existing
// component replaces it while leaving siblings intact.
func TestCompositeUpdateExistingComponent(t *testing.T) {
	msgs := []a2ui.ServerMessage{
		updateMsg("s",
			a2ui.Component{ID: "root", Card: &a2ui.CardComponent{Child: "col"}},
			a2ui.Component{ID: "col", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"a"}}}},
			txt("a", "Original"),
		),
		// Replace component "a" with new text.
		updateMsg("s",
			txt("a", "Updated"),
		),
	}

	model, err := a2tea.Render(msgs)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	view := ansi.Strip(model.View().Content)

	if strings.Contains(view, "Original") {
		t.Errorf("old text should be gone; view = %q", view)
	}
	if !strings.Contains(view, "Updated") {
		t.Errorf("new text missing; view = %q", view)
	}
}

// TestCompositeDataModelOnlyUpdate verifies that a data-model-only update
// (no updateComponents) changes a bound value on the next render.
func TestCompositeDataModelOnlyUpdate(t *testing.T) {
	msgs := []a2ui.ServerMessage{
		updateMsg("s",
			a2ui.Component{ID: "root", Card: &a2ui.CardComponent{Child: "t"}},
			boundText("t", "/provider"),
		),
		// No updateComponents — just set the bound value.
		dataMsg("s", "/provider", "anthropic"),
	}

	model, err := a2tea.Render(msgs)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	view := ansi.Strip(model.View().Content)

	if !strings.Contains(view, "anthropic") {
		t.Errorf("bound value not resolved; view = %q", view)
	}
	if strings.Contains(view, "{binding}") {
		t.Errorf("unresolved placeholder found; view = %q", view)
	}
}

// TestCompositeDataModelBeforeValue verifies that a bound Text component
// without a data-model update renders the {binding} placeholder.
func TestCompositeDataModelNoUpdate(t *testing.T) {
	msgs := []a2ui.ServerMessage{
		updateMsg("s",
			a2ui.Component{ID: "root", Card: &a2ui.CardComponent{Child: "t"}},
			boundText("t", "/provider"),
		),
	}

	model, err := a2tea.Render(msgs)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	view := ansi.Strip(model.View().Content)

	if !strings.Contains(view, "{binding}") {
		t.Errorf("expected {binding} placeholder; view = %q", view)
	}
}

// TestCompositeDeleteSurface verifies that deleteSurface removes the
// targeted surface and Render returns ErrNoRenderableSurface.
func TestCompositeDeleteSurface(t *testing.T) {
	msgs := []a2ui.ServerMessage{
		updateMsg("s", txt("root", "Hello")),
		deleteMsg("s"),
	}

	_, err := a2tea.Render(msgs)
	if !errors.Is(err, a2tea.ErrNoRenderableSurface) {
		t.Fatalf("err = %v, want ErrNoRenderableSurface", err)
	}
}

// TestCompositeDeleteDifferentSurface verifies that a deleteSurface for a
// different surface ID does not affect the rendered surface.
func TestCompositeDeleteDifferentSurface(t *testing.T) {
	msgs := []a2ui.ServerMessage{
		updateMsg("s1", txt("root", "Hello")),
		deleteMsg("s2"),
	}

	model, err := a2tea.Render(msgs)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	view := ansi.Strip(model.View().Content)
	if !strings.Contains(view, "Hello") {
		t.Errorf("surface should survive unrelated delete; view = %q", view)
	}
}

// TestRenderNoMessages verifies that empty messages returns
// ErrNoRenderableSurface.
func TestRenderNoMessages(t *testing.T) {
	_, err := a2tea.Render(nil)
	if !errors.Is(err, a2tea.ErrNoRenderableSurface) {
		t.Fatalf("err = %v, want ErrNoRenderableSurface", err)
	}
}
