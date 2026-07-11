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
func labeled(label, body string) string {
	if label == "" {
		return body
	}
	return styleCaption.Render(label) + "\n" + body
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
// behind a "▏" input cue. A nil or empty value renders a faint "(empty)".
func (s *Surface) renderTextField(c a2ui.Component) string {
	tf := c.TextField
	value := ""
	if tf.Value != nil {
		value = dynString(*tf.Value)
	}
	if value == "" {
		value = styleCaption.Render("(empty)")
	}
	return wrapTo(labeled(dynString(tf.Label), "▏"+value), s.width)
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
	line := box + " " + dynString(cb.Label)
	switch {
	case cb.Value.Binding != nil:
		line += " " + styleCaption.Render("{binding}")
	case cb.Value.FunctionCall != nil:
		line += " " + styleCaption.Render("{fn}")
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
		if label := dynString(*cp.Label); label != "" {
			lines = append(lines, styleCaption.Render(label))
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
		text := dynString(opt.Label)
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
	bar := styleCaption.Render(strings.Repeat("─", sliderCells))
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
		label = dynString(*sl.Label)
	}
	return wrapTo(labeled(label, body), s.width)
}

// renderDateTimeInput renders a DateTimeInput: an optional caption label
// line, then the value string; an absent value renders a faint "(unset)".
// EnableDate/EnableTime and the Min/Max bounds are not rendered.
func (s *Surface) renderDateTimeInput(c a2ui.Component) string {
	dt := c.DateTimeInput
	value := dynString(dt.Value)
	if value == "" {
		value = styleCaption.Render("(unset)")
	}
	label := ""
	if dt.Label != nil {
		label = dynString(*dt.Label)
	}
	return wrapTo(labeled(label, value), s.width)
}
