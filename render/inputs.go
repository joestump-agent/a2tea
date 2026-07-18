package render

import (
	"slices"

	tea "charm.land/bubbletea/v2"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/event"
)

// This file holds the edit paths for the non-text input components: CheckBox
// (Space/Enter toggles), ChoicePicker (Up/Down moves the highlight, Space
// toggles the highlighted option), and Slider (Left/Right steps within
// min/max). Each mirrors the hand-rolled TextField pattern in render.go:
// edits live in a lazily initialized map keyed by component ID and shadow the
// component's static literal for both rendering and value readout
// (gatherFieldValues / FieldValues). DateTimeInput needs no code here — it
// reuses the TextField rune-edit path (appendText / deleteRune) against its
// string value.

// checkBoxValue returns the CheckBox's current value: the toggled value when
// edited, else the literal, else false (a bound/function value has no local
// state to start from — the data model is not applied to booleans yet).
func (s *Surface) checkBoxValue(c a2ui.Component) bool {
	if s.checkValues != nil {
		if v, ok := s.checkValues[c.ID]; ok {
			return v
		}
	}
	return c.CheckBox.Value.Literal != nil && *c.CheckBox.Value.Literal
}

// toggleCheckBox flips the focused CheckBox's value, lazily initializing the
// checkValues map on first toggle.
func (s *Surface) toggleCheckBox() {
	id := s.focusables[s.focusIdx]
	c, ok := s.byID[id]
	if !ok || c.CheckBox == nil {
		return
	}
	if s.checkValues == nil {
		s.checkValues = make(map[string]bool)
	}
	s.checkValues[id] = !s.checkBoxValue(c)
}

// pickerSelection returns the ChoicePicker's current selection as a set: the
// edited selection when present, else the literal value list. A bound or
// function value starts empty — placeholders are chrome, not selection state.
func (s *Surface) pickerSelection(id string, cp *a2ui.ChoicePickerComponent) map[string]bool {
	var vals []string
	edited := false
	if s.choiceValues != nil {
		if v, ok := s.choiceValues[id]; ok {
			vals, edited = v, true
		}
	}
	if !edited {
		vals = cp.Value.Literal
	}
	sel := make(map[string]bool, len(vals))
	for _, v := range vals {
		sel[v] = true
	}
	return sel
}

// pickerCursor returns the picker's highlighted option index, clamped to its
// options list (options can shrink across Apply merges).
func (s *Surface) pickerCursor(id string, cp *a2ui.ChoicePickerComponent) int {
	cur := s.choiceCursor[id]
	if max := len(cp.Options) - 1; cur > max {
		cur = max
	}
	if cur < 0 {
		cur = 0
	}
	return cur
}

// movePickerCursor moves the focused ChoicePicker's highlight by delta,
// clamping at the ends of the options list (no wrap-around).
func (s *Surface) movePickerCursor(delta int) {
	id := s.focusables[s.focusIdx]
	c, ok := s.byID[id]
	if !ok || c.ChoicePicker == nil || len(c.ChoicePicker.Options) == 0 {
		return
	}
	next := s.pickerCursor(id, c.ChoicePicker) + delta
	if next < 0 {
		next = 0
	}
	if max := len(c.ChoicePicker.Options) - 1; next > max {
		next = max
	}
	if s.choiceCursor == nil {
		s.choiceCursor = make(map[string]int)
	}
	s.choiceCursor[id] = next
}

// togglePickerOption toggles the highlighted option of the focused
// ChoicePicker. The multipleSelection variant toggles membership in the
// selection; the single-select variants replace the selection with the
// highlighted option (radio semantics — pressing Space on the already
// selected option keeps it selected). The stored value is normalized to
// option-declaration order, so literal values that name no declared option
// are dropped once the user edits.
//
// When the toggle changes the selection set it returns a command dispatching
// event.ChoiceSelected with the picker's full post-toggle selection; a toggle
// that leaves the (normalized) selection unchanged — re-selecting a
// single-select picker's already selected option — returns nil.
func (s *Surface) togglePickerOption() tea.Cmd {
	id := s.focusables[s.focusIdx]
	c, ok := s.byID[id]
	if !ok || c.ChoicePicker == nil || len(c.ChoicePicker.Options) == 0 {
		return nil
	}
	cp := c.ChoicePicker
	optVal := cp.Options[s.pickerCursor(id, cp)].Value
	sel := s.pickerSelection(id, cp)
	// Normalize the pre-toggle selection to option-declaration order so the
	// changed check compares like with like (a literal naming no declared
	// option is dropped by normalization, not by the user's toggle).
	before := make([]string, 0, len(sel))
	for _, opt := range cp.Options {
		if sel[opt.Value] {
			before = append(before, opt.Value)
		}
	}
	if cp.Variant == a2ui.ChoicePickerVariantMultipleSelection {
		sel[optVal] = !sel[optVal]
	} else {
		sel = map[string]bool{optVal: true}
	}
	out := make([]string, 0, len(sel))
	for _, opt := range cp.Options {
		if sel[opt.Value] {
			out = append(out, opt.Value)
		}
	}
	if s.choiceValues == nil {
		s.choiceValues = make(map[string][]string)
	}
	s.choiceValues[id] = out
	if slices.Equal(before, out) {
		return nil
	}
	selected := event.ChoiceSelected{
		Source: event.Source{ComponentID: id, SurfaceID: s.id},
		ID:     id,
		Values: out,
	}
	return func() tea.Msg { return selected }
}

// sliderStep is the per-keypress increment for a slider spanning span. The
// v0.9 schema has no step field, so use 1 for spans of at least 1 (integer
// sliders step predictably) and span/sliderCells for smaller spans, so a
// fractional slider like 0..1 stays adjustable — one press moves the bar
// roughly one cell.
func sliderStep(span float64) float64 {
	if span >= 1 {
		return 1
	}
	return span / sliderCells
}

// stepSlider adjusts the focused Slider by dir (±1) steps, clamped into
// [min, max]. The first step seeds from the value literal, falling back to
// min when the value is bound/function (the data model is not applied to
// numbers yet). A degenerate range (max <= min) has nothing to step.
func (s *Surface) stepSlider(dir float64) {
	id := s.focusables[s.focusIdx]
	c, ok := s.byID[id]
	if !ok || c.Slider == nil {
		return
	}
	sl := c.Slider
	lo := 0.0
	if sl.Min != nil {
		lo = *sl.Min
	}
	span := sl.Max - lo
	if span <= 0 {
		return
	}
	cur, edited := s.sliderValues[id]
	if !edited {
		cur = lo
		if sl.Value.Literal != nil {
			cur = *sl.Value.Literal
		}
	}
	next := cur + dir*sliderStep(span)
	if next < lo {
		next = lo
	}
	if next > sl.Max {
		next = sl.Max
	}
	if s.sliderValues == nil {
		s.sliderValues = make(map[string]float64)
	}
	s.sliderValues[id] = next
}
