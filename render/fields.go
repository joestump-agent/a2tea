package render

// The field renderers in this file draw each input component's current
// value. Every input component is editable while focused: TextField and
// DateTimeInput accept typed edits via the rune-edit path, CheckBox toggles,
// ChoicePicker moves a highlight and toggles options, and Slider steps its
// value (see the Update loop in render.go and the edit paths in inputs.go).
// Edited state shadows each component's static literal for both rendering and
// value readout. A focused component marks its value line with the "▎" cue —
// the same monochrome chrome language as the TextField cursor.

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

// renderCheckBox renders a CheckBox as "[x] label" when its current value is
// true and "[ ] label" otherwise. A toggled value (from checkValues) shadows
// the static literal. An unedited non-literal value (binding/function) is
// treated as unchecked, with its placeholder appended after the label; once
// the user toggles, the edited value replaces the placeholder. A focused
// CheckBox is prefixed with the "▎" cue.
func (s *Surface) renderCheckBox(c a2ui.Component) string {
	cb := c.CheckBox
	edited := false
	if s.checkValues != nil {
		_, edited = s.checkValues[c.ID]
	}
	box := "[ ]"
	if s.checkBoxValue(c) {
		box = "[x]"
	}
	line := box + " " + s.dynString(cb.Label)
	if !edited {
		switch {
		case cb.Value.Binding != nil:
			line += " " + s.styles.Caption.Render("{binding}")
		case cb.Value.FunctionCall != nil:
			line += " " + s.styles.Caption.Render("{fn}")
		}
	}
	if s.isFocused(c.ID) {
		line = "▎" + line
	}
	return wrapTo(line, s.width)
}

// renderChoicePicker renders a ChoicePicker: an optional caption label line,
// then one line per option marked "(•)"/"( )" for single-select or
// "[x]"/"[ ]" for the multipleSelection variant. An option is selected when
// its value appears in the current selection — the edited selection (from
// choiceValues) when present, else the Value list literal; an unedited
// non-literal Value (binding/function) renders every option unselected.
// Option display text is the option's label, falling back to its value. When
// the picker holds focus every option line gains a "▏" gutter, with "▎" on
// the highlighted option Up/Down moves and Space toggles.
func (s *Surface) renderChoicePicker(c a2ui.Component) string {
	cp := c.ChoicePicker
	multi := cp.Variant == a2ui.ChoicePickerVariantMultipleSelection
	selected := s.pickerSelection(c.ID, cp)
	focused := s.isFocused(c.ID)
	cursor := s.pickerCursor(c.ID, cp)
	lines := make([]string, 0, len(cp.Options)+1)
	if cp.Label != nil {
		if label := s.dynString(*cp.Label); label != "" {
			lines = append(lines, s.styles.Caption.Render(label))
		}
	}
	for i, opt := range cp.Options {
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
		line := mark + " " + text
		if focused {
			cue := "▏"
			if i == cursor {
				cue = "▎"
			}
			line = cue + line
		}
		lines = append(lines, line)
	}
	return wrapTo(strings.Join(lines, "\n"), s.width)
}

// renderSlider renders a Slider: an optional caption label line, then a
// sliderCells-wide bar of filled "█" and empty "─" cells proportional to
// (value-min)/(max-min), followed by the numeric value. A stepped value (from
// sliderValues) shadows the static literal. A missing value literal or a
// degenerate range (max <= min, which would divide by zero) renders a faint
// placeholder bar instead. A focused Slider's bar is prefixed with the "▎"
// cue.
func (s *Surface) renderSlider(c a2ui.Component) string {
	sl := c.Slider
	lo := 0.0
	if sl.Min != nil {
		lo = *sl.Min
	}
	span := sl.Max - lo
	value := sl.Value.Literal
	if s.sliderValues != nil {
		if v, ok := s.sliderValues[c.ID]; ok {
			value = &v
		}
	}
	bar := s.styles.Caption.Render(strings.Repeat("─", sliderCells))
	if value != nil && span > 0 {
		ratio := (*value - lo) / span
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
	readout := dynNumString(sl.Value)
	if value != nil {
		readout = strconv.FormatFloat(*value, 'f', -1, 64)
	}
	if readout != "" {
		body += " " + readout
	}
	if s.isFocused(c.ID) {
		body = "▎" + body
	}
	label := ""
	if sl.Label != nil {
		label = s.dynString(*sl.Label)
	}
	return wrapTo(s.labeled(label, body), s.width)
}

// renderDateTimeInput renders a DateTimeInput: an optional caption label
// line, then the value string; an absent (or cleared) value renders a faint
// "(unset)". An edited value (from fieldValues — DateTimeInput shares the
// TextField rune-edit path) shadows the static literal, including an edit to
// the empty string, which must render as "(unset)" rather than fall back to
// the literal. A focused DateTimeInput's value is prefixed with the "▎" cue.
// EnableDate/EnableTime and the Min/Max bounds are not rendered.
func (s *Surface) renderDateTimeInput(c a2ui.Component) string {
	dt := c.DateTimeInput
	value := ""
	edited := false
	if s.fieldValues != nil {
		if v, ok := s.fieldValues[c.ID]; ok {
			value, edited = v, true
		}
	}
	if !edited {
		value = s.dynString(dt.Value)
	}
	if value == "" {
		value = s.styles.Caption.Render("(unset)")
	}
	if s.isFocused(c.ID) {
		value = "▎" + value
	}
	label := ""
	if dt.Label != nil {
		label = s.dynString(*dt.Label)
	}
	return wrapTo(s.labeled(label, value), s.width)
}
