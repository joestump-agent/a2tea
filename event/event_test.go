package event_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/event"
)

// TestEventsAreMessages pins that every event type is usable as a tea.Msg, so
// they flow through the standard bubbletea Update loop.
func TestEventsAreMessages(t *testing.T) {
	var _ tea.Msg = event.ButtonClicked{}
	var _ tea.Msg = event.InputSubmitted{}
	var _ tea.Msg = event.ChoiceSelected{}
	var _ tea.Msg = event.FormSubmitted{}
}

// TestSourceContext pins the Source fields on every event — the shape agents
// use to disambiguate interactions from multiple on-screen components.
func TestSourceContext(t *testing.T) {
	b := event.ButtonClicked{Source: event.Source{ComponentID: "card1", SurfaceID: "s1"}, ID: "ok"}
	if b.ComponentID != "card1" || b.SurfaceID != "s1" || b.ID != "ok" {
		t.Fatalf("ButtonClicked shape unexpected: %#v", b)
	}

	i := event.InputSubmitted{Source: event.Source{ComponentID: "form1"}, ID: "name", Value: "joe"}
	if i.ComponentID != "form1" || i.ID != "name" || i.Value != "joe" {
		t.Fatalf("InputSubmitted shape unexpected: %#v", i)
	}

	c := event.ChoiceSelected{Source: event.Source{ComponentID: "form1"}, ID: "color", Value: "red"}
	if c.Value != "red" {
		t.Fatalf("ChoiceSelected shape unexpected: %#v", c)
	}
}

// TestFormSubmittedCarriesAllValues pins the aggregate-submit shape: one event
// with every field value, keyed by field ID.
func TestFormSubmittedCarriesAllValues(t *testing.T) {
	fs := event.FormSubmitted{
		Source:   event.Source{ComponentID: "signup"},
		FormID:   "signup",
		SubmitID: "create",
		Values:   map[string]string{"name": "joe", "color": "red"},
	}
	if fs.FormID != "signup" || fs.SubmitID != "create" {
		t.Fatalf("FormSubmitted ids unexpected: %#v", fs)
	}
	if fs.Values["name"] != "joe" || fs.Values["color"] != "red" {
		t.Fatalf("FormSubmitted values unexpected: %#v", fs.Values)
	}
}
