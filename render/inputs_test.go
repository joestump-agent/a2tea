package render_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/render"
)

// Key press builders for the input-editing keys. Space carries its text the
// way a real terminal delivers it (Code AND Text set); the renderer matches
// on Key.String(), which reports "space".
func pressSpace() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "} }
func pressKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

// checkBoxComp builds a CheckBox with a literal value.
func checkBoxComp(id string, checked bool) a2ui.Component {
	return a2ui.Component{
		ID: id,
		CheckBox: &a2ui.CheckBoxComponent{
			Label: a2ui.StringLiteral("Agree"),
			Value: a2ui.BoolLiteral(checked),
		},
	}
}

// pickerComp builds a ChoicePicker with options a/b/c and the given variant
// and literal selection.
func pickerComp(id string, variant a2ui.ChoicePickerVariant, selected []string) a2ui.Component {
	return a2ui.Component{
		ID: id,
		ChoicePicker: &a2ui.ChoicePickerComponent{
			Variant: variant,
			Value:   a2ui.StringListLiteral(selected),
			Options: []a2ui.ChoiceOption{
				{Label: a2ui.StringLiteral("Alpha"), Value: "a"},
				{Label: a2ui.StringLiteral("Beta"), Value: "b"},
				{Label: a2ui.StringLiteral("Gamma"), Value: "c"},
			},
		},
	}
}

// sliderComp builds a Slider with the given bounds and literal value.
func sliderComp(id string, min, max, value float64) a2ui.Component {
	return a2ui.Component{
		ID: id,
		Slider: &a2ui.SliderComponent{
			Min:   &min,
			Max:   max,
			Value: a2ui.NumberLiteral(value),
		},
	}
}

// dateTimeComp builds a DateTimeInput with a literal value.
func dateTimeComp(id, value string) a2ui.Component {
	return a2ui.Component{
		ID:            id,
		DateTimeInput: &a2ui.DateTimeInputComponent{Value: a2ui.StringLiteral(value)},
	}
}

// inputSurface wraps comps in a Column root, builds the surface, and focuses
// it (focus starts on the first focusable).
func inputSurface(t *testing.T, comps ...a2ui.Component) *render.Surface {
	t.Helper()
	ids := make([]string, len(comps))
	for i, c := range comps {
		ids[i] = c.ID
	}
	root := a2ui.Component{
		ID:     "root",
		Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: ids}},
	}
	s := render.NewSurface("s", append([]a2ui.Component{root}, comps...))
	s.Focus()
	return s
}

// TestAllInputsCollectedAsFocusables verifies that CheckBox, ChoicePicker,
// Slider, and DateTimeInput join Buttons and TextFields in the focus ring, in
// depth-first order.
func TestAllInputsCollectedAsFocusables(t *testing.T) {
	s := inputSurface(t,
		textFieldInput("tf", "x"),
		checkBoxComp("cb", false),
		pickerComp("cp", "", nil),
		sliderComp("sl", 0, 10, 5),
		dateTimeComp("dt", "2026-07-18"),
		actionButton("btn", "lbl", "go", nil),
		textLabel("lbl", "Go"),
	)
	want := []string{"tf", "cb", "cp", "sl", "dt", "btn"}
	if got := s.Focusables(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Focusables() = %v, want %v", got, want)
	}
}

// TestCheckBoxSpaceToggles verifies Space flips a focused CheckBox's value,
// re-rendering live and updating FieldValues.
func TestCheckBoxSpaceToggles(t *testing.T) {
	s := inputSurface(t, checkBoxComp("cb", false))

	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "[ ] Agree") {
		t.Fatalf("initial render = %q, want unchecked box", out)
	}

	s.Update(pressSpace())
	out = ansi.Strip(s.View().Content)
	if !strings.Contains(out, "[x] Agree") {
		t.Fatalf("after space, render = %q, want checked box", out)
	}
	if v := s.FieldValues()["cb"]; v != true {
		t.Fatalf("FieldValues['cb'] = %v, want true", v)
	}

	// Toggle back.
	s.Update(pressSpace())
	out = ansi.Strip(s.View().Content)
	if !strings.Contains(out, "[ ] Agree") {
		t.Fatalf("after second space, render = %q, want unchecked box", out)
	}
	if v := s.FieldValues()["cb"]; v != false {
		t.Fatalf("FieldValues['cb'] = %v, want false", v)
	}
}

// TestCheckBoxEnterToggles verifies Enter also toggles a focused CheckBox and
// does not emit any activation command.
func TestCheckBoxEnterToggles(t *testing.T) {
	s := inputSurface(t, checkBoxComp("cb", true))

	_, cmd := s.Update(pressKey(tea.KeyEnter))
	if cmd != nil {
		t.Fatalf("enter on a CheckBox should not produce a cmd, got %#v", cmd())
	}
	if v := s.FieldValues()["cb"]; v != false {
		t.Fatalf("FieldValues['cb'] = %v, want false after enter toggle", v)
	}
}

// TestCheckBoxToggleShadowsBindingPlaceholder verifies that toggling a
// CheckBox whose value is an unresolved binding replaces the "{binding}"
// placeholder with real local state: render drops the placeholder and
// FieldValues reports the toggled bool (bound values are otherwise skipped).
func TestCheckBoxToggleShadowsBindingPlaceholder(t *testing.T) {
	s := inputSurface(t, a2ui.Component{
		ID: "cb",
		CheckBox: &a2ui.CheckBoxComponent{
			Label: a2ui.StringLiteral("Agree"),
			Value: a2ui.BoolBinding("/flags/on"),
		},
	})

	if _, ok := s.FieldValues()["cb"]; ok {
		t.Fatal("unedited bound CheckBox should be absent from FieldValues")
	}

	s.Update(pressSpace())
	out := ansi.Strip(s.View().Content)
	if strings.Contains(out, "{binding}") {
		t.Fatalf("toggled bound CheckBox should drop the placeholder: %q", out)
	}
	if !strings.Contains(out, "[x] Agree") {
		t.Fatalf("bound CheckBox starts unchecked, so first toggle checks it: %q", out)
	}
	if v := s.FieldValues()["cb"]; v != true {
		t.Fatalf("FieldValues['cb'] = %v, want true", v)
	}
}

// TestChoicePickerSingleSelectSpaceSelects verifies Up/Down move the
// highlight and Space replaces the selection in a single-select picker.
func TestChoicePickerSingleSelectSpaceSelects(t *testing.T) {
	s := inputSurface(t, pickerComp("cp", "", []string{"a"}))

	// Move the highlight to Beta and select it.
	s.Update(pressKey(tea.KeyDown))
	s.Update(pressSpace())

	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "(•) Beta") {
		t.Fatalf("Beta should be selected: %q", out)
	}
	if strings.Contains(out, "(•) Alpha") {
		t.Fatalf("single-select must replace the previous selection: %q", out)
	}
	if got := s.FieldValues()["cp"]; !reflect.DeepEqual(got, []string{"b"}) {
		t.Fatalf("FieldValues['cp'] = %v, want [b]", got)
	}

	// Space again on the selected option keeps it selected (radio semantics).
	s.Update(pressSpace())
	if got := s.FieldValues()["cp"]; !reflect.DeepEqual(got, []string{"b"}) {
		t.Fatalf("re-pressing space deselected: FieldValues['cp'] = %v, want [b]", got)
	}
}

// TestChoicePickerMultiSelectToggles verifies Space toggles membership in a
// multipleSelection picker and the stored value keeps option-declaration
// order regardless of toggle order.
func TestChoicePickerMultiSelectToggles(t *testing.T) {
	s := inputSurface(t, pickerComp("cp", a2ui.ChoicePickerVariantMultipleSelection, nil))

	// Select Gamma first, then Alpha: value must still come out [a c].
	s.Update(pressKey(tea.KeyDown))
	s.Update(pressKey(tea.KeyDown))
	s.Update(pressSpace())
	s.Update(pressKey(tea.KeyUp))
	s.Update(pressKey(tea.KeyUp))
	s.Update(pressSpace())

	if got := s.FieldValues()["cp"]; !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Fatalf("FieldValues['cp'] = %v, want [a c] in option order", got)
	}
	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "[x] Alpha") || !strings.Contains(out, "[ ] Beta") || !strings.Contains(out, "[x] Gamma") {
		t.Fatalf("render = %q, want Alpha and Gamma checked", out)
	}

	// Toggle Alpha back off.
	s.Update(pressSpace())
	if got := s.FieldValues()["cp"]; !reflect.DeepEqual(got, []string{"c"}) {
		t.Fatalf("FieldValues['cp'] = %v, want [c] after untoggling Alpha", got)
	}
}

// TestChoicePickerCursorClampsAtEnds verifies the highlight does not wrap:
// Up at the top and Down past the bottom stay put.
func TestChoicePickerCursorClampsAtEnds(t *testing.T) {
	s := inputSurface(t, pickerComp("cp", "", nil))

	// Up at the top: still highlights Alpha; Space selects it.
	s.Update(pressKey(tea.KeyUp))
	s.Update(pressSpace())
	if got := s.FieldValues()["cp"]; !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("after up at top, FieldValues['cp'] = %v, want [a]", got)
	}

	// Down past the last option clamps to Gamma.
	for range 5 {
		s.Update(pressKey(tea.KeyDown))
	}
	s.Update(pressSpace())
	if got := s.FieldValues()["cp"]; !reflect.DeepEqual(got, []string{"c"}) {
		t.Fatalf("after down past end, FieldValues['cp'] = %v, want [c]", got)
	}
}

// TestChoicePickerFocusHighlightCue verifies a focused picker draws the "▎"
// cue on the highlighted option and the "▏" gutter on the rest, and that an
// unfocused picker draws no gutter.
func TestChoicePickerFocusHighlightCue(t *testing.T) {
	s := inputSurface(t,
		pickerComp("cp", "", nil),
		actionButton("btn", "lbl", "go", nil),
		textLabel("lbl", "Go"),
	)

	s.Update(pressKey(tea.KeyDown))
	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "▎( ) Beta") {
		t.Fatalf("focused picker should mark the highlighted option: %q", out)
	}
	if !strings.Contains(out, "▏( ) Alpha") {
		t.Fatalf("focused picker should gutter the other options: %q", out)
	}

	// Tab away: the gutter disappears.
	s.Update(pressKey(tea.KeyTab))
	out = ansi.Strip(s.View().Content)
	if strings.Contains(out, "▎") || strings.Contains(out, "▏") {
		t.Fatalf("unfocused picker should draw no gutter: %q", out)
	}
}

// TestSliderStepsWithinRange verifies Left/Right step a focused Slider by 1
// and the rendered readout tracks the edited value.
func TestSliderStepsWithinRange(t *testing.T) {
	s := inputSurface(t, sliderComp("sl", 0, 100, 50))

	s.Update(pressKey(tea.KeyRight))
	if v := s.FieldValues()["sl"]; v != 51.0 {
		t.Fatalf("FieldValues['sl'] = %v, want 51", v)
	}
	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "51") {
		t.Fatalf("render should show the stepped value: %q", out)
	}

	s.Update(pressKey(tea.KeyLeft))
	s.Update(pressKey(tea.KeyLeft))
	if v := s.FieldValues()["sl"]; v != 49.0 {
		t.Fatalf("FieldValues['sl'] = %v, want 49", v)
	}
}

// TestSliderClampsAtBounds verifies stepping never leaves [min, max].
func TestSliderClampsAtBounds(t *testing.T) {
	s := inputSurface(t, sliderComp("sl", 0, 3, 2))

	for range 5 {
		s.Update(pressKey(tea.KeyRight))
	}
	if v := s.FieldValues()["sl"]; v != 3.0 {
		t.Fatalf("FieldValues['sl'] = %v, want clamped max 3", v)
	}
	for range 10 {
		s.Update(pressKey(tea.KeyLeft))
	}
	if v := s.FieldValues()["sl"]; v != 0.0 {
		t.Fatalf("FieldValues['sl'] = %v, want clamped min 0", v)
	}
}

// TestSliderFractionalSpanSteps verifies a slider whose span is below 1 steps
// by span/16 (one bar cell) instead of jumping the whole range.
func TestSliderFractionalSpanSteps(t *testing.T) {
	s := inputSurface(t, sliderComp("sl", 0, 0.5, 0))

	s.Update(pressKey(tea.KeyRight))
	v, ok := s.FieldValues()["sl"].(float64)
	if !ok {
		t.Fatalf("FieldValues['sl'] = %T, want float64", s.FieldValues()["sl"])
	}
	if want := 0.5 / 16; math.Abs(v-want) > 1e-12 {
		t.Fatalf("FieldValues['sl'] = %v, want %v", v, want)
	}
}

// TestSliderBoundValueSeedsFromMin verifies stepping a slider whose value is
// an unresolved binding starts from min — the placeholder is chrome, not a
// number — and that the unedited bound slider is absent from FieldValues.
func TestSliderBoundValueSeedsFromMin(t *testing.T) {
	min := 10.0
	s := inputSurface(t, a2ui.Component{
		ID: "sl",
		Slider: &a2ui.SliderComponent{
			Min:   &min,
			Max:   20,
			Value: a2ui.NumberBinding("/volume"),
		},
	})

	if _, ok := s.FieldValues()["sl"]; ok {
		t.Fatal("unedited bound Slider should be absent from FieldValues")
	}

	s.Update(pressKey(tea.KeyRight))
	if v := s.FieldValues()["sl"]; v != 11.0 {
		t.Fatalf("FieldValues['sl'] = %v, want 11 (min + one step)", v)
	}
	out := ansi.Strip(s.View().Content)
	if strings.Contains(out, "{binding}") {
		t.Fatalf("stepped bound slider should drop the placeholder readout: %q", out)
	}
}

// TestSliderDegenerateRangeIgnoresSteps verifies a slider whose max <= min
// cannot be stepped (nothing to adjust, and no divide-by-zero).
func TestSliderDegenerateRangeIgnoresSteps(t *testing.T) {
	s := inputSurface(t, sliderComp("sl", 5, 5, 5))

	s.Update(pressKey(tea.KeyRight))
	if v := s.FieldValues()["sl"]; v != 5.0 {
		t.Fatalf("FieldValues['sl'] = %v, want untouched literal 5", v)
	}
}

// TestSliderUneditedLiteralInFieldValues verifies an unedited Slider reports
// its literal in FieldValues, matching the other input kinds.
func TestSliderUneditedLiteralInFieldValues(t *testing.T) {
	s := inputSurface(t, sliderComp("sl", 0, 100, 42))
	if v := s.FieldValues()["sl"]; v != 42.0 {
		t.Fatalf("FieldValues['sl'] = %v, want literal 42", v)
	}
}

// TestDateTimeTypingEditsValue verifies a focused DateTimeInput accepts the
// TextField rune-edit path: typing appends, backspace deletes, and the
// rendered value tracks the edit.
func TestDateTimeTypingEditsValue(t *testing.T) {
	s := inputSurface(t, dateTimeComp("dt", "2026-07-1"))

	s.Update(typeKey('8'))
	if v := s.FieldValues()["dt"]; v != "2026-07-18" {
		t.Fatalf("FieldValues['dt'] = %v, want %q", v, "2026-07-18")
	}
	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "2026-07-18") {
		t.Fatalf("render should show the edited value: %q", out)
	}

	s.Update(pressKey(tea.KeyBackspace))
	s.Update(pressKey(tea.KeyBackspace))
	if v := s.FieldValues()["dt"]; v != "2026-07-" {
		t.Fatalf("FieldValues['dt'] = %v, want %q", v, "2026-07-")
	}
}

// TestDateTimeClearedRendersUnset verifies backspacing a DateTimeInput to
// empty renders "(unset)" and reports the cleared empty string, not the stale
// literal.
func TestDateTimeClearedRendersUnset(t *testing.T) {
	s := inputSurface(t, dateTimeComp("dt", "07"))

	s.Update(pressKey(tea.KeyBackspace))
	s.Update(pressKey(tea.KeyBackspace))

	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "(unset)") {
		t.Fatalf("cleared DateTimeInput should render (unset): %q", out)
	}
	if v := s.FieldValues()["dt"]; v != "" {
		t.Fatalf("FieldValues['dt'] = %v, want cleared empty string", v)
	}
}

// TestSpaceStillInsertsIntoTextField guards the space key's dual role: on a
// focused TextField it is text input, not a toggle command.
func TestSpaceStillInsertsIntoTextField(t *testing.T) {
	s := inputSurface(t, textFieldInput("tf", "a"))

	s.Update(pressSpace())
	s.Update(typeKey('b'))
	if v := s.FieldValues()["tf"]; v != "a b" {
		t.Fatalf("FieldValues['tf'] = %q, want %q", v, "a b")
	}
}

// TestButtonContextCarriesEditedInputValues is the end-to-end dispatch check:
// edit every input kind, activate the button, and verify the emitted
// ClientMessage's ActionEvent.Context carries the edited bool, []string,
// float64, and string values.
func TestButtonContextCarriesEditedInputValues(t *testing.T) {
	s := inputSurface(t,
		checkBoxComp("cb", false),
		pickerComp("cp", a2ui.ChoicePickerVariantMultipleSelection, nil),
		sliderComp("sl", 0, 10, 5),
		dateTimeComp("dt", "2026-07-1"),
		actionButton("btn", "lbl", "submit", nil),
		textLabel("lbl", "Submit"),
	)

	// CheckBox: toggle on.
	s.Update(pressSpace())
	// ChoicePicker: select Beta.
	s.Update(pressKey(tea.KeyTab))
	s.Update(pressKey(tea.KeyDown))
	s.Update(pressSpace())
	// Slider: two steps right.
	s.Update(pressKey(tea.KeyTab))
	s.Update(pressKey(tea.KeyRight))
	s.Update(pressKey(tea.KeyRight))
	// DateTimeInput: complete the date.
	s.Update(pressKey(tea.KeyTab))
	s.Update(typeKey('8'))
	// Button: activate.
	s.Update(pressKey(tea.KeyTab))
	_, cmd := s.Update(pressKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("enter on the button produced no cmd")
	}

	cm := findMsg[a2ui.ClientMessage](t, collectMsgs(t, cmd))
	if cm.Action == nil {
		t.Fatal("ClientMessage.Action is nil")
	}
	ctx := cm.Action.Context

	if v := ctx["cb"]; v != true {
		t.Fatalf("Context['cb'] = %v, want true", v)
	}
	if v := ctx["cp"]; !reflect.DeepEqual(v, []string{"b"}) {
		t.Fatalf("Context['cp'] = %v, want [b]", v)
	}
	if v := ctx["sl"]; v != 7.0 {
		t.Fatalf("Context['sl'] = %v, want 7", v)
	}
	if v := ctx["dt"]; v != "2026-07-18" {
		t.Fatalf("Context['dt'] = %v, want %q", v, "2026-07-18")
	}
}

// TestInputEditsSurviveApplyMerge verifies edited input values survive an
// updateComponents merge that touches a sibling (edit state is keyed by ID
// and merges do not clear it).
func TestInputEditsSurviveApplyMerge(t *testing.T) {
	s := inputSurface(t,
		checkBoxComp("cb", false),
		sliderComp("sl", 0, 10, 5),
	)

	s.Update(pressSpace()) // toggle cb on
	s.Update(pressKey(tea.KeyTab))
	s.Update(pressKey(tea.KeyRight)) // sl -> 6

	s.Apply([]a2ui.ServerMessage{
		{UpdateComponents: &a2ui.UpdateComponents{
			SurfaceID:  "s",
			Components: []a2ui.Component{textLabel("note", "updated")},
		}},
	})

	vals := s.FieldValues()
	if v := vals["cb"]; v != true {
		t.Fatalf("FieldValues['cb'] = %v after merge, want true", v)
	}
	if v := vals["sl"]; v != 6.0 {
		t.Fatalf("FieldValues['sl'] = %v after merge, want 6", v)
	}
}
