package render

import a2ui "github.com/tmc/a2ui"

// gatherFieldValues walks every input component on the surface and returns a
// map of component ID → current value. Scope is the entire surface because
// A2UI v0.9 has no form grouping; if the producer wants a subset it
// references them explicitly in Action.Event.Context.
//
// Value types match the component kind so the host and agent see a consistent
// shape: TextField → string, ChoicePicker → []string, CheckBox → bool.
// Slider and DateTimeInput are omitted — they are not yet editable (#15) and
// their readout is speculative.
//
// Only literal values are resolved. Binding and FunctionCall DynamicX values
// are skipped because the data model is not applied yet; the host should
// resolve these from its own state.
func (s *Surface) gatherFieldValues() map[string]any {
	ctx := make(map[string]any)
	for _, c := range s.byID {
		switch {
		case c.TextField != nil && c.TextField.Value != nil:
			// Include the field whenever its value is a literal — even an
			// empty string, which is a meaningful "the user cleared this"
			// signal the host must be able to tell apart from an absent
			// field. Bindings resolve to nil and are skipped.
			if v := resolveDynamicString(*c.TextField.Value); v != nil {
				ctx[c.ID] = *v
			}
		case c.ChoicePicker != nil:
			if v := resolveDynamicStringList(c.ChoicePicker.Value); v != nil {
				ctx[c.ID] = v
			}
		case c.CheckBox != nil:
			if v := resolveDynamicBool(c.CheckBox.Value); v != nil {
				ctx[c.ID] = *v
			}
		}
	}
	return ctx
}

// resolveDynamicValue converts a DynamicValue to a concrete Go value.
// Only literal values are resolved; bindings and function calls return nil
// because the data model is not applied yet.
func resolveDynamicValue(d a2ui.DynamicValue) any {
	switch {
	case d.String != nil:
		return *d.String
	case d.Number != nil:
		return *d.Number
	case d.Bool != nil:
		return *d.Bool
	case d.Array != nil:
		return d.Array
	}
	return nil
}

// resolveDynamicString extracts a literal string from a DynamicString,
// returning nil for non-literal (binding/function) values. It returns a
// pointer so callers can distinguish a literal empty string from an absent
// value — mirroring resolveDynamicBool.
func resolveDynamicString(d a2ui.DynamicString) *string {
	return d.Literal
}

// resolveDynamicStringList extracts a concrete []string from a
// DynamicStringList. Returns nil for non-literal values.
func resolveDynamicStringList(d a2ui.DynamicStringList) []string {
	if d.Literal != nil {
		return d.Literal
	}
	return nil
}

// resolveDynamicBool extracts a concrete *bool from a DynamicBoolean.
// Returns nil for non-literal values.
func resolveDynamicBool(d a2ui.DynamicBoolean) *bool {
	return d.Literal
}
