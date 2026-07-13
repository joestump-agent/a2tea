package render_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/event"
	"github.com/joestump-agent/a2tea/render"
)

// activateButton builds a surface from comps, focuses it, and sends Enter to
// the focused button (index 0). It returns the emitted event.ButtonClicked.
func activateButton(t *testing.T, comps []a2ui.Component) event.ButtonClicked {
	t.Helper()
	s := render.NewSurface("surf1", comps)
	s.Focus()
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on focused button returned nil cmd")
	}
	msg := cmd()
	ev, ok := msg.(event.ButtonClicked)
	if !ok {
		t.Fatalf("cmd produced %T, want event.ButtonClicked", msg)
	}
	return ev
}

// TestButtonEventActionEmitsClientMessage verifies that activating a button
// with an EventAction populates both the Action field and the protocol-native
// ClientMessage on the emitted event.ButtonClicked.
func TestButtonEventActionEmitsClientMessage(t *testing.T) {
	comps := []a2ui.Component{
		{
			ID: "submit",
			Button: &a2ui.ButtonComponent{
				Child: "lbl",
				Action: a2ui.Action{
					Event: &a2ui.EventAction{Name: "submitForm"},
				},
			},
		},
		{ID: "lbl", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Submit")}},
	}

	ev := activateButton(t, comps)

	if ev.ID != "submit" {
		t.Fatalf("ButtonClicked.ID = %q, want %q", ev.ID, "submit")
	}
	if ev.Action == nil {
		t.Fatal("ButtonClicked.Action is nil, want non-nil EventAction")
	}
	if ev.Action.Name != "submitForm" {
		t.Fatalf("Action.Name = %q, want %q", ev.Action.Name, "submitForm")
	}

	cm := ev.ClientMessage
	if cm.Version == "" {
		t.Fatal("ClientMessage.Version is empty, want protocol version")
	}
	if cm.Action == nil {
		t.Fatal("ClientMessage.Action is nil, want non-nil ActionEvent")
	}
	if cm.Action.Name != "submitForm" {
		t.Fatalf("ClientMessage.Action.Name = %q, want %q", cm.Action.Name, "submitForm")
	}
	if cm.Action.SurfaceID != "surf1" {
		t.Fatalf("ClientMessage.Action.SurfaceID = %q, want %q", cm.Action.SurfaceID, "surf1")
	}
	if cm.Action.SourceComponentID != "submit" {
		t.Fatalf("ClientMessage.Action.SourceComponentID = %q, want %q", cm.Action.SourceComponentID, "submit")
	}
	// Timestamp is intentionally left empty for the host to stamp.
	if cm.Action.Timestamp != "" {
		t.Fatalf("ClientMessage.Action.Timestamp = %q, want empty (host stamps it)", cm.Action.Timestamp)
	}
	// Context is intentionally left empty; populating it from field values is #20.
	if cm.Action.Context != nil {
		t.Fatalf("ClientMessage.Action.Context = %v, want nil (#20)", cm.Action.Context)
	}
}

// TestButtonFunctionCallOnlyNoActionEvent verifies that a button with only a
// FunctionCall action (no EventAction) does NOT produce an ActionEvent or
// ClientMessage — client-side function calls are handled by the host.
func TestButtonFunctionCallOnlyNoActionEvent(t *testing.T) {
	comps := []a2ui.Component{
		{
			ID: "calc",
			Button: &a2ui.ButtonComponent{
				Child: "lbl",
				Action: a2ui.Action{
					FunctionCall: &a2ui.FunctionCall{
						Call: "computeTotal",
					},
				},
			},
		},
		{ID: "lbl", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Calculate")}},
	}

	ev := activateButton(t, comps)

	if ev.ID != "calc" {
		t.Fatalf("ButtonClicked.ID = %q, want %q", ev.ID, "calc")
	}
	if ev.Action != nil {
		t.Fatalf("ButtonClicked.Action = %+v, want nil for FunctionCall-only button", ev.Action)
	}
	cm := ev.ClientMessage
	if cm.Action != nil {
		t.Fatalf("ClientMessage.Action = %+v, want nil for FunctionCall-only button", cm.Action)
	}
}

// TestButtonNoActionEmitsBasicEvent verifies that a button with no action at
// all still emits a ButtonClicked with correct IDs but nil Action.
func TestButtonNoActionEmitsBasicEvent(t *testing.T) {
	comps := []a2ui.Component{
		{
			ID: "plain",
			Button: &a2ui.ButtonComponent{
				Child: "lbl",
			},
		},
		{ID: "lbl", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Plain")}},
	}

	ev := activateButton(t, comps)

	if ev.ID != "plain" || ev.ComponentID != "plain" || ev.SurfaceID != "surf1" {
		t.Fatalf("ButtonClicked = %+v, want ID=plain on surf1", ev)
	}
	if ev.Action != nil {
		t.Fatalf("ButtonClicked.Action = %+v, want nil for actionless button", ev.Action)
	}
}
