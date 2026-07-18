package render_test

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/render"
)

// activateButtonCtx wraps comps in a Column root so the button is reachable
// from the focus ring, then focuses the surface, tabs to the button, and
// sends Enter. Activation emits a batch; this pulls the native
// a2ui.ClientMessage out of it and returns its ActionEvent.Context.
func activateButtonCtx(t *testing.T, comps []a2ui.Component) map[string]any {
	t.Helper()
	ids := make([]string, len(comps))
	for i, c := range comps {
		ids[i] = c.ID
	}
	root := a2ui.Component{
		ID:     "__root__",
		Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: ids}},
	}
	s := render.NewSurface("surf1", append([]a2ui.Component{root}, comps...))
	s.Focus()
	// Tab to the first button in the focus ring, pressing Enter only once
	// focus lands on it. The ring includes every input component, and Enter
	// is no longer inert on all of them (it toggles a focused CheckBox), so
	// blindly pressing Enter at each stop would corrupt the values under
	// test.
	isButton := make(map[string]bool, len(comps))
	for _, c := range comps {
		if c.Button != nil {
			isButton[c.ID] = true
		}
	}
	for _, id := range s.Focusables() {
		if isButton[id] {
			_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if cmd == nil {
				t.Fatalf("enter on button %q produced no cmd", id)
			}
			cm := findMsg[a2ui.ClientMessage](t, collectMsgs(t, cmd))
			if cm.Action == nil {
				t.Fatalf("button %q ClientMessage has nil Action", id)
			}
			return cm.Action.Context
		}
		s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	}
	t.Fatal("no button found in focus ring")
	return nil
}

func textInput(id, val string) a2ui.Component {
	ds := a2ui.StringLiteral(val)
	return a2ui.Component{
		ID:        id,
		TextField: &a2ui.TextFieldComponent{Value: &ds},
	}
}

func choiceInput(id string, vals []string) a2ui.Component {
	return a2ui.Component{
		ID:           id,
		ChoicePicker: &a2ui.ChoicePickerComponent{Value: a2ui.StringListLiteral(vals)},
	}
}

func checkBoxInput(id string, val bool) a2ui.Component {
	return a2ui.Component{
		ID:       id,
		CheckBox: &a2ui.CheckBoxComponent{Value: a2ui.BoolLiteral(val)},
	}
}

func actionButton(id, child, actionName string, ctx map[string]a2ui.DynamicValue) a2ui.Component {
	return a2ui.Component{
		ID: id,
		Button: &a2ui.ButtonComponent{
			Child: child,
			Action: a2ui.Action{
				Event: &a2ui.EventAction{Name: actionName, Context: ctx},
			},
		},
	}
}

func textLabel(id, val string) a2ui.Component {
	return a2ui.Component{ID: id, Text: &a2ui.TextComponent{Text: a2ui.StringLiteral(val)}}
}

// TestContextTextFieldValue verifies a TextField's literal value appears
// in ActionEvent.Context as a string keyed by component ID.
func TestContextTextFieldValue(t *testing.T) {
	comps := []a2ui.Component{
		textInput("name", "Alice"),
		actionButton("submit", "lbl", "save", nil),
		textLabel("lbl", "Save"),
	}
	ctx := activateButtonCtx(t, comps)

	v, ok := ctx["name"]
	if !ok {
		t.Fatal("Context missing key 'name'")
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("Context['name'] = %T, want string", v)
	}
	if s != "Alice" {
		t.Fatalf("Context['name'] = %q, want %q", s, "Alice")
	}
}

// TestContextChoicePickerIsList verifies a ChoicePicker's multi-value
// appears in ActionEvent.Context as a []string, not a single string.
func TestContextChoicePickerIsList(t *testing.T) {
	comps := []a2ui.Component{
		choiceInput("colors", []string{"red", "blue"}),
		actionButton("submit", "lbl", "save", nil),
		textLabel("lbl", "Save"),
	}
	ctx := activateButtonCtx(t, comps)

	v, ok := ctx["colors"]
	if !ok {
		t.Fatal("Context missing key 'colors'")
	}
	sl, ok := v.([]string)
	if !ok {
		t.Fatalf("Context['colors'] = %T, want []string", v)
	}
	if !reflect.DeepEqual(sl, []string{"red", "blue"}) {
		t.Fatalf("Context['colors'] = %v, want [red blue]", sl)
	}
}

// TestContextCheckBoxIsBool verifies a CheckBox's value appears in
// ActionEvent.Context as a bool.
func TestContextCheckBoxIsBool(t *testing.T) {
	comps := []a2ui.Component{
		checkBoxInput("agree", true),
		actionButton("submit", "lbl", "save", nil),
		textLabel("lbl", "Save"),
	}
	ctx := activateButtonCtx(t, comps)

	v, ok := ctx["agree"]
	if !ok {
		t.Fatal("Context missing key 'agree'")
	}
	b, ok := v.(bool)
	if !ok {
		t.Fatalf("Context['agree'] = %T, want bool", v)
	}
	if b != true {
		t.Fatalf("Context['agree'] = %v, want true", b)
	}
}

// TestContextActionBindings verifies that a button's declared
// Action.Event.Context bindings are merged into ActionEvent.Context.
func TestContextActionBindings(t *testing.T) {
	bindings := map[string]a2ui.DynamicValue{
		"mode": a2ui.ValueString("production"),
	}
	comps := []a2ui.Component{
		textInput("name", "Bob"),
		actionButton("submit", "lbl", "save", bindings),
		textLabel("lbl", "Save"),
	}
	ctx := activateButtonCtx(t, comps)

	// Field value from surface input.
	v, ok := ctx["name"]
	if !ok {
		t.Fatal("Context missing field key 'name'")
	}
	if v.(string) != "Bob" {
		t.Fatalf("Context['name'] = %q, want %q", v, "Bob")
	}

	// Declared action binding.
	v, ok = ctx["mode"]
	if !ok {
		t.Fatal("Context missing action binding key 'mode'")
	}
	if v.(string) != "production" {
		t.Fatalf("Context['mode'] = %q, want %q", v, "production")
	}
}

// TestContextBindingOverridesField verifies that when an action binding
// key matches a field component ID, the action binding wins (producer's
// explicit intent overrides the automatic field gather).
func TestContextBindingOverridesField(t *testing.T) {
	bindings := map[string]a2ui.DynamicValue{
		"name": a2ui.ValueString("override"),
	}
	comps := []a2ui.Component{
		textInput("name", "Alice"),
		actionButton("submit", "lbl", "save", bindings),
		textLabel("lbl", "Save"),
	}
	ctx := activateButtonCtx(t, comps)

	v := ctx["name"]
	if v.(string) != "override" {
		t.Fatalf("Context['name'] = %q, want %q (binding should override field)", v, "override")
	}
}

// TestContextEmptyForNoInputs verifies Context is empty when there are no
// input components and no action bindings.
func TestContextEmptyForNoInputs(t *testing.T) {
	comps := []a2ui.Component{
		actionButton("go", "lbl", "navigate", nil),
		textLabel("lbl", "Go"),
	}
	ctx := activateButtonCtx(t, comps)
	if len(ctx) != 0 {
		t.Fatalf("Context = %v, want empty map", ctx)
	}
}

// TestContextAllFieldTypesTogether verifies that a surface with TextField,
// ChoicePicker, and CheckBox all present their values in Context with
// correct types simultaneously.
func TestContextAllFieldTypesTogether(t *testing.T) {
	comps := []a2ui.Component{
		textInput("query", "hello"),
		choiceInput("tags", []string{"go", "a2ui"}),
		checkBoxInput("active", false),
		actionButton("submit", "lbl", "search", nil),
		textLabel("lbl", "Search"),
	}
	ctx := activateButtonCtx(t, comps)

	if ctx["query"].(string) != "hello" {
		t.Fatalf("Context['query'] = %v, want %q", ctx["query"], "hello")
	}
	if !reflect.DeepEqual(ctx["tags"], []string{"go", "a2ui"}) {
		t.Fatalf("Context['tags'] = %v, want [go a2ui]", ctx["tags"])
	}
	if ctx["active"].(bool) != false {
		t.Fatalf("Context['active'] = %v, want false", ctx["active"])
	}
}

// TestContextTextFieldEmptyStringIncluded verifies a TextField whose value is
// a literal empty string is still reported (an absent field and a cleared
// field are different signals the host must be able to tell apart).
func TestContextTextFieldEmptyStringIncluded(t *testing.T) {
	comps := []a2ui.Component{
		textInput("note", ""),
		actionButton("submit", "lbl", "save", nil),
		textLabel("lbl", "Save"),
	}
	ctx := activateButtonCtx(t, comps)

	v, ok := ctx["note"]
	if !ok {
		t.Fatal("Context missing key 'note' for a cleared (empty-literal) field")
	}
	if s, ok := v.(string); !ok || s != "" {
		t.Fatalf("Context['note'] = %#v, want empty string", v)
	}
}

// TestContextBindingFieldsSkipped verifies fields whose values are data-model
// bindings are omitted (the data model is not applied yet) and never leak a
// "{binding}" placeholder into Context.
func TestContextBindingFieldsSkipped(t *testing.T) {
	boundText := a2ui.StringBinding("user.name")
	comps := []a2ui.Component{
		{ID: "bt", TextField: &a2ui.TextFieldComponent{Value: &boundText}},
		{ID: "bc", CheckBox: &a2ui.CheckBoxComponent{Value: a2ui.BoolBinding("flags.on")}},
		{ID: "bp", ChoicePicker: &a2ui.ChoicePickerComponent{Value: a2ui.StringListBinding("sel")}},
		actionButton("submit", "lbl", "save", nil),
		textLabel("lbl", "Save"),
	}
	ctx := activateButtonCtx(t, comps)

	for _, k := range []string{"bt", "bc", "bp"} {
		if _, ok := ctx[k]; ok {
			t.Errorf("Context contains binding-valued field %q; bindings must be skipped", k)
		}
	}
	for k, v := range ctx {
		if s, ok := v.(string); ok && s == "{binding}" {
			t.Errorf("Context[%q] leaked a {binding} placeholder", k)
		}
	}
}

// TestContextActionBindingNonLiteralSkipped verifies a declared
// Action.Event.Context entry that is itself a data-model binding (not a
// literal) is skipped rather than resolved to a placeholder.
func TestContextActionBindingNonLiteralSkipped(t *testing.T) {
	bindings := map[string]a2ui.DynamicValue{
		"lit":   a2ui.ValueString("here"),
		"bound": a2ui.ValueBinding("model.path"),
	}
	comps := []a2ui.Component{
		actionButton("submit", "lbl", "save", bindings),
		textLabel("lbl", "Save"),
	}
	ctx := activateButtonCtx(t, comps)

	if ctx["lit"] != "here" {
		t.Fatalf("Context['lit'] = %#v, want \"here\"", ctx["lit"])
	}
	if _, ok := ctx["bound"]; ok {
		t.Fatalf("Context['bound'] present = %#v; a non-literal binding must be skipped", ctx["bound"])
	}
}
