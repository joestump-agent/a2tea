package a2tea_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/joestump-agent/a2tea"
	a2ui "github.com/tmc/a2ui"
)

// TestButtonRendersWithoutChild is a regression test for issue #14.
// When a Button has no child component (Child is empty — the model put
// "text" on the button itself), the button must still render as a real
// focusable button, not [a2tea: missing component ""].
//
// Because the A2UI schema has no "text" field on ButtonComponent, the
// label is lost at unmarshal time. The fallback uses the component ID as
// the label — this test pins that behavior so it is explicit, not
// accidental. The producer-side fix (crush#47) repairs the label
// host-side so the real text survives.
func TestButtonRendersWithoutChild(t *testing.T) {
	data, err := os.ReadFile("testdata/button_form.json")
	if err != nil {
		t.Fatal(err)
	}

	var msg a2ui.ServerMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal ServerMessage: %v", err)
	}
	if msg.UpdateComponents == nil {
		t.Fatal("UpdateComponents is nil")
	}

	model, err := a2tea.Render([]a2ui.ServerMessage{msg})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	view := model.View().Content
	t.Logf("rendered:\n%s", view)

	if strings.Contains(view, "missing component") {
		t.Errorf("output contains missing component placeholder")
	}
	// The fallback renders the component ID since the real label is lost.
	if !strings.Contains(view, "[ btn1 ]") {
		t.Errorf("output should contain [ btn1 ] (ID fallback label), got: %s", view)
	}
	if !strings.Contains(view, "[ btn2 ]") {
		t.Errorf("output should contain [ btn2 ] (ID fallback label), got: %s", view)
	}
}

// TestButtonRendersWithChild verifies buttons that do use a child Text
// component for their label still render correctly.
func TestButtonRendersWithChild(t *testing.T) {
	data, err := os.ReadFile("testdata/button_with_child.json")
	if err != nil {
		t.Fatal(err)
	}

	var msg a2ui.ServerMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal ServerMessage: %v", err)
	}

	model, err := a2tea.Render([]a2ui.ServerMessage{msg})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	view := model.View().Content
	t.Logf("rendered:\n%s", view)

	if strings.Contains(view, "missing component") {
		t.Errorf("output contains missing component placeholder")
	}
	if !strings.Contains(view, "Send") || !strings.Contains(view, "Cancel") {
		t.Errorf("output does not contain expected button labels")
	}
}
