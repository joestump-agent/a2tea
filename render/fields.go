package render

// The field renderers in this file are READ-ONLY visuals: they draw each
// input component's current value, but editing is not wired — the surface
// never mutates field state or emits input events for these components.

import (
	"strconv"
	"strings"

	a2ui "github.com/tmc/a2ui"
)

// sliderCells is the width of a Slider's bar in terminal cells.
const sliderCells = 16

// labeled prefixes body with a caption-styled label line when label is
// non-empty; otherwise returns body unchanged.
func (s *Surface) labeled(label, body string) string {
	if label == "" {
		return body
	}
	return s.styles.Caption.Render(label) + "\n" + body
}

// dynNumString formats a DynamicNumber's literal as a plain decimal, or a
// placeholder marking a binding/function (mirrors dynString).
func dynNumString(d a2ui.DynamicNumber) string {
	switch {
	case d.Literal != nil:
		return strconv.FormatFloat(*d.Literal, 'f', -1, 64)
	case d.Binding != nil:
		return "{binding}"
	case d.FunctionCall != nil:
		return "{fn}"
	default:
		return ""
	}
}

// renderTextField renders a TextField: a caption label line, then the value
// behind a "▏" input cue. When the user has typed into this field, the edited
// value (from fieldValues) is shown instead of the static literal. A focused
// text field shows a cursor block "▎" at the end of the value. A nil or empty
// value renders a faint "(empty)".
func (s *Surface) renderTextField(c a2ui.Component) string {
	tf := c.TextField
	value := ""
	// An edited value shadows the static literal — including an edit to the
	// empty string, which means the user cleared the field and must render as
	// "(empty)", not fall back to the literal (that would disagree with the
	// value readout, which reports the cleared "").
	edited := false
	if s.fieldValues != nil {
		if v, ok := s.fieldValues[c.ID]; ok {
			value, edited = v, true
		}
	}
	if !edited && tf.Value != nil {
		value = s.dynString(*tf.Value)
	}
	if value == "" {
		value = s.styles.Caption.Render("(empty)")
	}
	cue := "▏"
	if s.isFocused(c.ID) {
		cue = "▎"
	}
	return wrapTo(s.labeled(s.dynString(tf.Label), cue+value), s.width)
}

// renderCheckBox renders a CheckBox as "[x] label" when the value's literal
// is true and "[ ] label" otherwise. A non-literal value (binding/function)
// is treated as unchecked, with its placeholder appended after the label.
func (s *Surface) renderCheckBox(c a2ui.Component) string {
	cb := c.CheckBox
	box := "[ ]"
	if cb.Value.Literal != nil && *cb.Value.Literal {
		box = "[x]"
	}
	line := box + " " + s.dynString(cb.Label)
	switch {
	case cb.Value.Binding != nil:
		line += " " + s.styles.Caption.Render("{binding}")
	case cb.Value.FunctionCall != nil:
		line += " " + s.styles.Caption.Render("{fn}")
	}
	return wrapTo(line, s.width)
}

// renderChoicePicker renders a ChoicePicker: an optional caption label line,
// then one line per option marked "(•)"/"( )" for single-select or
// "[x]"/"[ ]" for the multipleSelection variant. An option is selected when
// its value appears in the picker's Value list literal; a non-literal Value
// (binding/function) renders every option unselected. Option display text is
// the option's label, falling back to its value.
func (s *Surface) renderChoicePicker(c a2ui.Component) string {
	cp := c.ChoicePicker
	multi := cp.Variant == a2ui.ChoicePickerVariantMultipleSelection
	selected := make(map[string]bool, len(cp.Value.Literal))
	for _, v := range cp.Value.Literal {
		selected[v] = true
	}
	lines := make([]string, 0, len(cp.Options)+1)
	if cp.Label != nil {
		if label := s.dynString(*cp.Label); label != "" {
			lines = append(lines, s.styles.Caption.Render(label))
		}
	}
	for _, opt := range cp.Options {
		mark := "( )"
		switch {
		case multi && selected[opt.Value]:
			mark = "[x]"
		case multi:
			mark = "[ ]"
		case selected[opt.Value]:
			mark = "(•)"
		}
		text := s.dynString(opt.Label)
		if text == "" {
			text = opt.Value
		}
		lines = append(lines, mark+" "+text)
	}
	return wrapTo(strings.Join(lines, "\n"), s.width)
}

// renderSlider renders a Slider: an optional caption label line, then a
// sliderCells-wide bar of filled "█" and empty "─" cells proportional to
// (value-min)/(max-min), followed by the numeric value. A missing value
// literal or a degenerate range (max <= min, which would divide by zero)
// renders a faint placeholder bar instead.
func (s *Surface) renderSlider(c a2ui.Component) string {
	sl := c.Slider
	lo := 0.0
	if sl.Min != nil {
		lo = *sl.Min
	}
	span := sl.Max - lo
	bar := s.styles.Caption.Render(strings.Repeat("─", sliderCells))
	if sl.Value.Literal != nil && span > 0 {
		ratio := (*sl.Value.Literal - lo) / span
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		filled := int(ratio*sliderCells + 0.5)
		bar = strings.Repeat("█", filled) + strings.Repeat("─", sliderCells-filled)
	}
	body := bar
	if v := dynNumString(sl.Value); v != "" {
		body += " " + v
	}
	label := ""
	if sl.Label != nil {
		label = s.dynString(*sl.Label)
	}
	return wrapTo(s.labeled(label, body), s.width)
}

// renderDateTimeInput renders a DateTimeInput: an optional caption label
// line, then the value string; an absent value renders a faint "(unset)".
// EnableDate/EnableTime and the Min/Max bounds are not rendered.
func (s *Surface) renderDateTimeInput(c a2ui.Component) string {
	dt := c.DateTimeInput
	value := s.dynString(dt.Value)
	if value == "" {
		value = s.styles.Caption.Render("(unset)")
	}
	label := ""
	if dt.Label != nil {
		label = s.dynString(*dt.Label)
	}
	return wrapTo(s.labeled(label, value), s.width)
}
