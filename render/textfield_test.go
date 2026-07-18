package render_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/event"
	"github.com/joestump-agent/a2tea/render"
)

// typeKey builds a KeyPressMsg for a printable character the way a real
// terminal delivers it: both Code and Text are set. The renderer reads
// key.Text (not key.Code) for input, so tests must set it — Code alone is not
// what the wire produces and would silently type nothing.
func typeKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

// textFieldInput builds a TextField component with a literal value.
func textFieldInput(id, val string) a2ui.Component {
	ds := a2ui.StringLiteral(val)
	return a2ui.Component{
		ID:        id,
		TextField: &a2ui.TextFieldComponent{Value: &ds},
	}
}

// fieldSurface builds a surface with a text field and a button under a Column
// root, focuses it, and returns the surface.
func fieldSurface(t *testing.T) *render.Surface {
	t.Helper()
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"field", "btn"}}}},
		textFieldInput("field", "initial"),
		actionButton("btn", "lbl", "save", nil),
		textLabel("lbl", "Save"),
	}
	s := render.NewSurface("s", comps)
	s.Focus()
	return s
}

// TestTextFieldCollectedAsFocusable verifies that a TextField appears in the
// focus ring alongside Buttons, in depth-first order.
func TestTextFieldCollectedAsFocusable(t *testing.T) {
	s := fieldSurface(t)
	focusables := s.Focusables()
	// field comes before btn (depth-first).
	if len(focusables) != 2 || focusables[0] != "field" || focusables[1] != "btn" {
		t.Fatalf("focusables = %v, want [field btn]", focusables)
	}
}

// TestTabCyclesBetweenFieldAndButton verifies Tab moves focus from the text
// field to the button and back.
func TestTabCyclesBetweenFieldAndButton(t *testing.T) {
	s := fieldSurface(t)

	// Initially focused on "field" (index 0).
	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "▎") {
		t.Fatalf("text field should show focus cue initially: %q", out)
	}

	// Tab to button.
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	out = ansi.Strip(s.View().Content)
	// Button has reverse-video when focused; the text field should NOT have
	// the focus cue anymore.
	if strings.Contains(out, "▎") {
		t.Fatalf("text field should not show focus cue after tab: %q", out)
	}

	// Tab back to field.
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	out = ansi.Strip(s.View().Content)
	if !strings.Contains(out, "▎") {
		t.Fatalf("text field should show focus cue after tabbing back: %q", out)
	}
}

// TestTypingUpdatesRenderedValue verifies that typing runes into a focused
// text field updates the rendered value live.
func TestTypingUpdatesRenderedValue(t *testing.T) {
	s := fieldSurface(t)

	// Type "X" into the focused field.
	s.Update(typeKey('X'))

	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "initialX") {
		t.Fatalf("rendered value should include typed rune: %q", out)
	}

	// Type "Y".
	s.Update(typeKey('Y'))
	out = ansi.Strip(s.View().Content)
	if !strings.Contains(out, "initialXY") {
		t.Fatalf("rendered value should include both typed runes: %q", out)
	}
}

// TestShiftedAndSymbolInputInserted verifies that Shift-produced characters
// (uppercase letters and shifted symbols) are inserted. A real terminal
// delivers these with the base rune in Code, ModShift set, and the shifted
// character in Text — the renderer must read Text, not gate on Mod == 0.
func TestShiftedAndSymbolInputInserted(t *testing.T) {
	s := fieldSurface(t)

	// Backspace the literal away so we can read exactly what was typed.
	for range "initial" {
		s.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}

	// Shift+a → "A"; Shift+1 → "!". Both carry ModShift and a Code that is
	// NOT the shifted character.
	s.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModShift, Text: "A"})
	s.Update(tea.KeyPressMsg{Code: '1', Mod: tea.ModShift, Text: "!"})

	vals := s.FieldValues()
	if vals["field"].(string) != "A!" {
		t.Fatalf("FieldValues['field'] = %q, want %q (shifted input dropped)", vals["field"], "A!")
	}
}

// TestNavigationKeysDoNotInsert verifies that navigation and function keys —
// whose Code is a sentinel above unicode.MaxRune and whose Text is empty — are
// not inserted into the field. Inserting key.Code for these would append a
// U+FFFD replacement character.
func TestNavigationKeysDoNotInsert(t *testing.T) {
	s := fieldSurface(t)

	for _, code := range []rune{tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown, tea.KeyHome, tea.KeyEnd, tea.KeyF1} {
		s.Update(tea.KeyPressMsg{Code: code})
	}

	vals := s.FieldValues()
	if vals["field"].(string) != "initial" {
		t.Fatalf("FieldValues['field'] = %q, want unchanged %q (navigation key inserted)", vals["field"], "initial")
	}
	out := ansi.Strip(s.View().Content)
	if strings.Contains(out, "�") {
		t.Fatalf("rendered value contains a replacement character: %q", out)
	}
}

// TestBackspaceDeletesLastRune verifies that backspace removes the last rune
// from the focused text field's value.
func TestBackspaceDeletesLastRune(t *testing.T) {
	s := fieldSurface(t)

	// Type "AB".
	s.Update(typeKey('A'))
	s.Update(typeKey('B'))

	// Backspace removes "B".
	s.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})

	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "initialA") {
		t.Fatalf("after backspace, rendered value should end with 'A': %q", out)
	}
}

// TestBackspaceToEmptyFallsBackToLiteral verifies that when all typed runes
// are deleted, rendering falls back to the original literal.
func TestBackspaceToEmptyFallsBackToLiteral(t *testing.T) {
	s := fieldSurface(t)

	// Type "X".
	s.Update(typeKey('X'))
	// Delete it.
	s.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})

	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "initial") {
		t.Fatalf("after deleting all typed runes, should show literal: %q", out)
	}
}

// TestClearingLiteralFieldRendersEmpty verifies that backspacing a literal
// field all the way to empty renders "(empty)" — the cleared state — and the
// value readout reports the empty string, not the stale literal. Rendering the
// literal here would disagree with FieldValues and lose the "user cleared it"
// signal.
func TestClearingLiteralFieldRendersEmpty(t *testing.T) {
	s := fieldSurface(t)

	for range "initial" {
		s.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}

	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "(empty)") {
		t.Fatalf("cleared field should render (empty): %q", out)
	}
	if strings.Contains(out, "initial") {
		t.Fatalf("cleared field should not render the stale literal: %q", out)
	}
	vals := s.FieldValues()
	if v := vals["field"].(string); v != "" {
		t.Fatalf("FieldValues['field'] = %q, want cleared %q", v, "")
	}
}

// TestFieldValuesReturnsEditedValue verifies that FieldValues returns the
// typed value, not the static literal.
func TestFieldValuesReturnsEditedValue(t *testing.T) {
	s := fieldSurface(t)

	// Type "!" into the focused field.
	s.Update(typeKey('!'))

	vals := s.FieldValues()
	v, ok := vals["field"]
	if !ok {
		t.Fatal("FieldValues missing 'field' key")
	}
	if v.(string) != "initial!" {
		t.Fatalf("FieldValues['field'] = %v, want %q", v, "initial!")
	}
}

// TestFieldValuesReturnsLiteralWhenNotEdited verifies that FieldValues returns
// the static literal when no edits have been made.
func TestFieldValuesReturnsLiteralWhenNotEdited(t *testing.T) {
	s := fieldSurface(t)
	vals := s.FieldValues()
	v, ok := vals["field"]
	if !ok {
		t.Fatal("FieldValues missing 'field' key")
	}
	if v.(string) != "initial" {
		t.Fatalf("FieldValues['field'] = %v, want %q", v, "initial")
	}
}

// TestEnterOnTextFieldEmitsInputSubmitted verifies that pressing Enter while
// a TextField is focused emits event.InputSubmitted with Source set and the
// field's current value — and does NOT emit ButtonClicked (Enter still only
// activates buttons).
func TestEnterOnTextFieldEmitsInputSubmitted(t *testing.T) {
	s := fieldSurface(t)

	// Focus is on the text field (index 0). Press Enter without editing: the
	// submitted value is the field's literal.
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msgs := collectMsgs(t, cmd)

	if hasMsg[event.ButtonClicked](msgs) {
		t.Fatal("Enter on a TextField should not emit ButtonClicked")
	}
	sub := findMsg[event.InputSubmitted](t, msgs)
	if sub.ComponentID != "field" || sub.SurfaceID != "s" {
		t.Fatalf("InputSubmitted.Source = %#v, want ComponentID 'field', SurfaceID 's'", sub.Source)
	}
	if sub.ID != "field" || sub.Value != "initial" {
		t.Fatalf("InputSubmitted = %#v, want ID 'field', Value 'initial'", sub)
	}
}

// TestEnterSubmitsEditedValue verifies InputSubmitted carries the edited text,
// not the stale literal, after the user types into the field.
func TestEnterSubmitsEditedValue(t *testing.T) {
	s := fieldSurface(t)

	for _, r := range "!!" {
		s.Update(typeKey(r))
	}
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	sub := findMsg[event.InputSubmitted](t, collectMsgs(t, cmd))
	if sub.Value != "initial!!" {
		t.Fatalf("InputSubmitted.Value = %q, want %q", sub.Value, "initial!!")
	}
}

// TestEnterOnUnresolvedBoundFieldSubmitsEmpty verifies submitting an unedited
// TextField whose value is an unresolved binding carries "" — the "{binding}"
// display placeholder must not leak into the event.
func TestEnterOnUnresolvedBoundFieldSubmitsEmpty(t *testing.T) {
	bound := a2ui.StringBinding("/name")
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"f"}}}},
		{ID: "f", TextField: &a2ui.TextFieldComponent{Value: &bound}},
	}
	s := render.NewSurface("s", comps)
	s.Focus()

	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	sub := findMsg[event.InputSubmitted](t, collectMsgs(t, cmd))
	if sub.Value != "" {
		t.Fatalf("InputSubmitted.Value = %q, want empty (placeholder leaked)", sub.Value)
	}
}

// TestEnterOnButtonActivatesAfterTyping verifies that after typing into a
// field, tabbing to the button and pressing Enter carries the typed value in
// ActionEvent.Context.
func TestEnterOnButtonActivatesAfterTyping(t *testing.T) {
	s := fieldSurface(t)

	// Type "world" into the focused text field.
	for _, r := range "world" {
		s.Update(typeKey(r))
	}

	// Tab to the button.
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	// Press Enter on the button.
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on button should produce a command")
	}

	msgs := collectMsgs(t, cmd)

	// Should emit a ButtonClicked.
	clicked := findMsg[event.ButtonClicked](t, msgs)
	if clicked.ID != "btn" {
		t.Fatalf("ButtonClicked.ID = %q, want %q", clicked.ID, "btn")
	}

	// Should carry the typed value in ActionEvent.Context via ClientMessage.
	cm := findMsg[a2ui.ClientMessage](t, msgs)
	if cm.Action == nil {
		t.Fatal("ClientMessage.Action is nil")
	}
	v, ok := cm.Action.Context["field"]
	if !ok {
		t.Fatal("Context missing 'field' key")
	}
	if v.(string) != "initialworld" {
		t.Fatalf("Context['field'] = %v, want %q", v, "initialworld")
	}
}

// TestButtonOnlySurfaceUnchangedFocusRing verifies that a surface with only
// buttons (no text fields) has the same focus ring as before.
func TestButtonOnlySurfaceUnchangedFocusRing(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"btn1", "btn2"}}}},
		actionButton("btn1", "lbl1", "go", nil),
		actionButton("btn2", "lbl2", "stop", nil),
		textLabel("lbl1", "Go"),
		textLabel("lbl2", "Stop"),
	}
	s := render.NewSurface("s", comps)
	focusables := s.Focusables()
	if len(focusables) != 2 || focusables[0] != "btn1" || focusables[1] != "btn2" {
		t.Fatalf("button-only focusables = %v, want [btn1 btn2]", focusables)
	}
}

// TestEditingDoesNotAffectOtherField verifies that typing into one field does
// not leak into another.
func TestEditingDoesNotAffectOtherField(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"f1", "f2"}}}},
		textFieldInput("f1", "one"),
		textFieldInput("f2", "two"),
	}
	s := render.NewSurface("s", comps)
	s.Focus()

	// Type "X" into f1 (focused).
	s.Update(typeKey('X'))

	vals := s.FieldValues()
	if vals["f1"].(string) != "oneX" {
		t.Fatalf("f1 = %v, want %q", vals["f1"], "oneX")
	}
	if vals["f2"].(string) != "two" {
		t.Fatalf("f2 = %v, want %q", vals["f2"], "two")
	}

	// Tab to f2, type "Y".
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	s.Update(typeKey('Y'))

	vals = s.FieldValues()
	if vals["f1"].(string) != "oneX" {
		t.Fatalf("f1 = %v, want %q", vals["f1"], "oneX")
	}
	if vals["f2"].(string) != "twoY" {
		t.Fatalf("f2 = %v, want %q", vals["f2"], "twoY")
	}
}

// TestEditingBoundFieldDoesNotLeakPlaceholder verifies that typing into a
// TextField whose value is an unresolved data-model binding starts from an
// empty seed — the "{binding}" display placeholder must not leak into
// FieldValues (and from there into ActionEvent.Context).
func TestEditingBoundFieldDoesNotLeakPlaceholder(t *testing.T) {
	bound := a2ui.StringBinding("/name")
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"f"}}}},
		{ID: "f", TextField: &a2ui.TextFieldComponent{Value: &bound}},
	}
	s := render.NewSurface("s", comps)
	s.Focus()

	s.Update(typeKey('X'))

	vals := s.FieldValues()
	if got := vals["f"].(string); got != "X" {
		t.Fatalf("FieldValues[f] = %q, want %q (placeholder leaked into edit seed)", got, "X")
	}
}

// TestEditingResolvedBoundFieldSeedsFromDataModel verifies that when the
// binding IS resolved in the data model, editing extends the resolved value.
func TestEditingResolvedBoundFieldSeedsFromDataModel(t *testing.T) {
	bound := a2ui.StringBinding("/name")
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"f"}}}},
		{ID: "f", TextField: &a2ui.TextFieldComponent{Value: &bound}},
	}
	s := render.NewSurface("s", comps)
	s.Apply([]a2ui.ServerMessage{
		{UpdateDataModel: &a2ui.UpdateDataModel{SurfaceID: "s", Path: "/name", Value: "Bob"}},
	})
	s.Focus()

	s.Update(typeKey('X'))

	vals := s.FieldValues()
	if got := vals["f"].(string); got != "BobX" {
		t.Fatalf("FieldValues[f] = %q, want %q (resolved binding should seed the edit)", got, "BobX")
	}
}

// TestBackspaceOnUnresolvedBoundFieldIsNoOp verifies backspace on a bound
// field with no resolved value does nothing (there is no real text to
// delete — the placeholder is not content).
func TestBackspaceOnUnresolvedBoundFieldIsNoOp(t *testing.T) {
	bound := a2ui.StringBinding("/name")
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"f"}}}},
		{ID: "f", TextField: &a2ui.TextFieldComponent{Value: &bound}},
	}
	s := render.NewSurface("s", comps)
	s.Focus()

	s.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})

	vals := s.FieldValues()
	if got, ok := vals["f"]; ok && got != "" {
		t.Fatalf("FieldValues[f] = %v after backspace on unresolved binding, want empty/absent", got)
	}
}
