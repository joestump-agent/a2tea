package render_test

import (
	"strings"
	"testing"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/render"
)

// TestTabsRenderTitleBarAndFirstChild verifies the default Tabs rendering:
// all tab titles joined with " │ " on one line, followed by only the ACTIVE
// tab's child — the first tab until the user switches (see tabs_test.go for
// switching) — so the second tab's content must stay hidden.
func TestTabsRenderTitleBarAndFirstChild(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Tabs: &a2ui.TabsComponent{Tabs: []a2ui.TabDef{
			{Title: a2ui.StringLiteral("One"), Child: "c1"},
			{Title: a2ui.StringLiteral("Two"), Child: "c2"},
		}}},
		text("c1", "first content"),
		text("c2", "second content"),
	}
	out := renderPlain(comps)

	if !strings.Contains(out, "One │ Two") {
		t.Fatalf("tabs should render a ' │ '-separated title bar: %q", out)
	}
	if !strings.Contains(out, "first content") {
		t.Fatalf("tabs should render the first tab's child: %q", out)
	}
	if strings.Contains(out, "second content") {
		t.Fatalf("tabs should NOT render an inactive tab's child: %q", out)
	}
	// The title bar comes before the active tab's content.
	if strings.Index(out, "One │ Two") > strings.Index(out, "first content") {
		t.Fatalf("title bar should precede tab content: %q", out)
	}
}

// TestTabsEmptyPlaceholder verifies that a Tabs component with no tabs
// renders the explicit placeholder rather than panicking on tabs[0].
func TestTabsEmptyPlaceholder(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Tabs: &a2ui.TabsComponent{}},
	}
	out := renderPlain(comps)
	if !strings.Contains(out, "[a2tea: tabs with no tabs]") {
		t.Fatalf("empty tabs should render the placeholder: %q", out)
	}
}

// TestModalRendersTriggerAndHidesContent verifies the Modal placeholder: the
// trigger child renders, the content stays hidden, and the faint explanatory
// note follows the trigger.
func TestModalRendersTriggerAndHidesContent(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Modal: &a2ui.ModalComponent{Trigger: "trig", Content: "body"}},
		text("trig", "Open settings"),
		text("body", "secret dialog body"),
	}
	out := renderPlain(comps)

	if !strings.Contains(out, "Open settings") {
		t.Fatalf("modal should render its trigger: %q", out)
	}
	if strings.Contains(out, "secret dialog body") {
		t.Fatalf("closed modal should NOT render its content: %q", out)
	}
	if !strings.Contains(out, "[a2tea: modal content hidden until interaction support lands]") {
		t.Fatalf("modal should render the hidden-content note: %q", out)
	}
	if strings.Index(out, "Open settings") > strings.Index(out, "modal content hidden") {
		t.Fatalf("trigger should precede the note: %q", out)
	}
}

// TestImagePlaceholder verifies the Image placeholder: the glyph plus the
// Description when present, otherwise the URL.
func TestImagePlaceholder(t *testing.T) {
	desc := a2ui.StringLiteral("a cat photo")
	withDesc := []a2ui.Component{
		{ID: "img", Image: &a2ui.ImageComponent{
			URL:         a2ui.StringLiteral("http://x/cat.png"),
			Description: &desc,
		}},
	}
	if out := renderPlain(withDesc); !strings.Contains(out, "🖼 a cat photo") {
		t.Fatalf("image with description = %q, want glyph + description", out)
	}

	urlOnly := []a2ui.Component{
		{ID: "img", Image: &a2ui.ImageComponent{URL: a2ui.StringLiteral("http://x/cat.png")}},
	}
	if out := renderPlain(urlOnly); !strings.Contains(out, "🖼 http://x/cat.png") {
		t.Fatalf("image without description = %q, want glyph + URL", out)
	}
}

// TestIconPlaceholder verifies the Icon placeholder for all three variants of
// the IconNameOrPath union: a well-known name renders verbatim, a custom SVG
// path renders as "svg", and a binding renders the "{binding}" placeholder.
func TestIconPlaceholder(t *testing.T) {
	name := a2ui.IconName("accountCircle")
	svg := "M0 0L10 10Z"
	cases := []struct {
		label string
		icon  a2ui.IconNameOrPath
		want  string
	}{
		{"name", a2ui.IconNameOrPath{Name: &name}, "⟨accountCircle⟩"},
		{"svgPath", a2ui.IconNameOrPath{SVGPath: &svg}, "⟨svg⟩"},
		{"binding", a2ui.IconNameOrPath{Binding: &a2ui.DataBinding{Path: "/icon"}}, "⟨{binding}⟩"},
	}
	for _, tc := range cases {
		comps := []a2ui.Component{
			{ID: "ic", Icon: &a2ui.IconComponent{Name: tc.icon}},
		}
		if out := renderPlain(comps); !strings.Contains(out, tc.want) {
			t.Errorf("icon %s = %q, want %q", tc.label, out, tc.want)
		}
	}
}

// TestVideoPlaceholder verifies the Video placeholder: a play glyph plus the
// URL (VideoComponent has no description field to prefer).
func TestVideoPlaceholder(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "v", Video: &a2ui.VideoComponent{URL: a2ui.StringLiteral("http://x/clip.mp4")}},
	}
	if out := renderPlain(comps); !strings.Contains(out, "▶ http://x/clip.mp4") {
		t.Fatalf("video = %q, want play glyph + URL", out)
	}
}

// TestAudioPlaceholder verifies the AudioPlayer placeholder: a note glyph plus
// the Description when present, otherwise the URL.
func TestAudioPlaceholder(t *testing.T) {
	desc := a2ui.StringLiteral("theme song")
	withDesc := []a2ui.Component{
		{ID: "a", AudioPlayer: &a2ui.AudioPlayerComponent{
			URL:         a2ui.StringLiteral("http://x/a.mp3"),
			Description: &desc,
		}},
	}
	if out := renderPlain(withDesc); !strings.Contains(out, "♪ theme song") {
		t.Fatalf("audio with description = %q, want glyph + description", out)
	}

	urlOnly := []a2ui.Component{
		{ID: "a", AudioPlayer: &a2ui.AudioPlayerComponent{URL: a2ui.StringLiteral("http://x/a.mp3")}},
	}
	if out := renderPlain(urlOnly); !strings.Contains(out, "♪ http://x/a.mp3") {
		t.Fatalf("audio without description = %q, want glyph + URL", out)
	}
}

// TestDateTimeInputRendersLabelAndValue verifies the DateTimeInput renderer:
// a caption label line followed by the value string.
func TestDateTimeInputRendersLabelAndValue(t *testing.T) {
	label := a2ui.StringLiteral("Due date")
	comps := []a2ui.Component{
		{ID: "dt", DateTimeInput: &a2ui.DateTimeInputComponent{
			Label: &label,
			Value: a2ui.StringLiteral("2026-07-17T09:00"),
		}},
	}
	out := renderPlain(comps)
	if !strings.Contains(out, "Due date") {
		t.Fatalf("datetime should render its label: %q", out)
	}
	if !strings.Contains(out, "2026-07-17T09:00") {
		t.Fatalf("datetime should render its value: %q", out)
	}
	if strings.Index(out, "Due date") > strings.Index(out, "2026-07-17") {
		t.Fatalf("label line should precede the value: %q", out)
	}
}

// TestDateTimeInputUnset verifies that a DateTimeInput with no value renders
// the "(unset)" placeholder.
func TestDateTimeInputUnset(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "dt", DateTimeInput: &a2ui.DateTimeInputComponent{}},
	}
	if out := renderPlain(comps); !strings.Contains(out, "(unset)") {
		t.Fatalf("valueless datetime = %q, want \"(unset)\"", out)
	}
}

// choicePicker builds a ChoicePicker with a label, three options (the third
// deliberately without a label, to exercise the fall-back-to-value path), and
// the given variant/selected values.
func choicePicker(variant a2ui.ChoicePickerVariant, selected []string) []a2ui.Component {
	label := a2ui.StringLiteral("Pick one")
	return []a2ui.Component{
		{ID: "cp", ChoicePicker: &a2ui.ChoicePickerComponent{
			Label:   &label,
			Variant: variant,
			Value:   a2ui.StringListLiteral(selected),
			Options: []a2ui.ChoiceOption{
				{Label: a2ui.StringLiteral("Alpha"), Value: "a"},
				{Label: a2ui.StringLiteral("Beta"), Value: "b"},
				{Value: "c"}, // no label: display text falls back to the value
			},
		}},
	}
}

// TestChoicePickerSingleSelect verifies the single-select rendering: the
// caption label line, "(•)" on the selected option, "( )" on the rest, and
// the label-less option displayed by its value.
func TestChoicePickerSingleSelect(t *testing.T) {
	out := renderPlain(choicePicker("", []string{"b"}))

	if !strings.Contains(out, "Pick one") {
		t.Fatalf("picker should render its label: %q", out)
	}
	if !strings.Contains(out, "( ) Alpha") {
		t.Fatalf("unselected option = %q, want \"( ) Alpha\"", out)
	}
	if !strings.Contains(out, "(•) Beta") {
		t.Fatalf("selected option = %q, want \"(•) Beta\"", out)
	}
	if !strings.Contains(out, "( ) c") {
		t.Fatalf("label-less option should display its value: %q", out)
	}
	if strings.Contains(out, "[x]") || strings.Contains(out, "[ ]") {
		t.Fatalf("single-select should not use checkbox marks: %q", out)
	}
}

// TestChoicePickerMultiSelect verifies the multipleSelection variant: "[x]"
// on each selected option and "[ ]" on the rest, with no radio marks.
func TestChoicePickerMultiSelect(t *testing.T) {
	out := renderPlain(choicePicker(a2ui.ChoicePickerVariantMultipleSelection, []string{"a", "c"}))

	if !strings.Contains(out, "[x] Alpha") {
		t.Fatalf("selected option = %q, want \"[x] Alpha\"", out)
	}
	if !strings.Contains(out, "[ ] Beta") {
		t.Fatalf("unselected option = %q, want \"[ ] Beta\"", out)
	}
	if !strings.Contains(out, "[x] c") {
		t.Fatalf("selected label-less option = %q, want \"[x] c\"", out)
	}
	if strings.Contains(out, "(•)") || strings.Contains(out, "( )") {
		t.Fatalf("multi-select should not use radio marks: %q", out)
	}
}

// TestChoicePickerNonLiteralValueUnselected verifies that a bound (non-literal)
// Value renders every option unselected.
func TestChoicePickerNonLiteralValueUnselected(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "cp", ChoicePicker: &a2ui.ChoicePickerComponent{
			Value: a2ui.StringListBinding("/choices"),
			Options: []a2ui.ChoiceOption{
				{Label: a2ui.StringLiteral("Alpha"), Value: "a"},
				{Label: a2ui.StringLiteral("Beta"), Value: "b"},
			},
		}},
	}
	out := renderPlain(comps)
	if strings.Contains(out, "(•)") {
		t.Fatalf("bound picker value should leave every option unselected: %q", out)
	}
	if got := strings.Count(out, "( )"); got != 2 {
		t.Fatalf("bound picker should render %d unselected marks, got %d: %q", 2, got, out)
	}
}

// TestSliderDegenerateRangePlaceholder verifies that a slider whose range
// cannot be normalized (max <= min would divide by zero) renders the
// all-empty placeholder bar — never a filled cell — while still appending the
// numeric value.
func TestSliderDegenerateRangePlaceholder(t *testing.T) {
	five := 5.0
	comps := []a2ui.Component{
		{ID: "sl", Slider: &a2ui.SliderComponent{
			Min:   &five,
			Max:   5, // max == min: degenerate range
			Value: a2ui.NumberLiteral(5),
		}},
	}
	out := renderPlain(comps)
	if !strings.Contains(out, strings.Repeat("─", 16)) {
		t.Fatalf("degenerate slider should render a full 16-cell empty bar: %q", out)
	}
	if strings.Contains(out, "█") {
		t.Fatalf("degenerate slider must not render filled cells: %q", out)
	}
	if !strings.Contains(out, "5") {
		t.Fatalf("degenerate slider should still append its value: %q", out)
	}
}

// TestSliderMissingLiteralPlaceholder verifies that a slider with a bound
// (non-literal) value renders the placeholder bar plus the "{binding}"
// readout from dynNumString.
func TestSliderMissingLiteralPlaceholder(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "sl", Slider: &a2ui.SliderComponent{
			Max:   100,
			Value: a2ui.NumberBinding("/volume"),
		}},
	}
	out := renderPlain(comps)
	if !strings.Contains(out, strings.Repeat("─", 16)) {
		t.Fatalf("bound slider should render a full 16-cell empty bar: %q", out)
	}
	if strings.Contains(out, "█") {
		t.Fatalf("bound slider must not render filled cells: %q", out)
	}
	if !strings.Contains(out, "{binding}") {
		t.Fatalf("bound slider should render the binding placeholder readout: %q", out)
	}
}

// TestDynStringFunctionCallPlaceholder verifies that a Text whose
// DynamicString is a FunctionCall renders the "{fn}" placeholder.
func TestDynStringFunctionCallPlaceholder(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "t", Text: &a2ui.TextComponent{
			Text: a2ui.StringFunc(a2ui.FunctionCall{Call: "computeGreeting"}),
		}},
	}
	if out := renderPlain(comps); !strings.Contains(out, "{fn}") {
		t.Fatalf("function-call text = %q, want the {fn} placeholder", out)
	}
}

// TestDynStringNonStringBoundValue verifies that a Text bound to a
// non-string data-model value renders the value formatted with %v (an int 42
// renders as "42"), not the "{binding}" placeholder.
func TestDynStringNonStringBoundValue(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "t", Text: &a2ui.TextComponent{Text: a2ui.StringBinding("/count")}},
	}
	s := render.NewSurface("s", comps)

	// Before the data model lands, the binding is unresolved.
	if out := renderPlain(comps); !strings.Contains(out, "{binding}") {
		t.Fatalf("unresolved binding = %q, want the {binding} placeholder", out)
	}

	alive := s.Apply([]a2ui.ServerMessage{{
		UpdateDataModel: &a2ui.UpdateDataModel{SurfaceID: "s", Path: "/count", Value: 42},
	}})
	if !alive {
		t.Fatal("Apply reported surface as not alive")
	}
	out := s.View().Content
	if !strings.Contains(out, "42") {
		t.Fatalf("bound int should render via %%v as \"42\": %q", out)
	}
	if strings.Contains(out, "{binding}") {
		t.Fatalf("resolved binding should not render the placeholder: %q", out)
	}
}
