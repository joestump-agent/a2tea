package render

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	a2ui "github.com/tmc/a2ui"
)

// renderTemplateChildren expands the dynamic-template form of a ChildList:
// the template's Path resolves in the data model to a list, and the template
// component renders once per element with bindings scoped to that element.
//
// An unresolved path (the data model hasn't delivered the list yet) or an
// empty list expands to no children — the list grows when the data arrives,
// exactly as a later updateDataModel grows or shrinks it. A resolved value
// that is not a list is a producer error and renders a caption placeholder,
// matching the surface's other "[a2tea: ...]" diagnostics.
//
// Cycle safety comes from renderComponent's seen set: each instance renders
// through the same ancestor-path guard, so a template component that
// references its own container reports a cycle instead of recursing forever.
func (s *Surface) renderTemplateChildren(t *a2ui.ChildTemplate, seen map[string]bool) []string {
	v, ok := s.lookupBinding(t.Path)
	if !ok {
		return nil
	}
	list, ok := asList(v)
	if !ok {
		return []string{s.styles.Caption.Render(fmt.Sprintf("[a2tea: template path %q is not a list]", t.Path))}
	}
	parts := make([]string, 0, len(list))
	for _, el := range list {
		s.scope = append(s.scope, el)
		parts = append(parts, s.renderComponent(t.ComponentID, seen))
		s.scope = s.scope[:len(s.scope)-1]
	}
	return parts
}

// lookupBinding resolves a data-binding path. Template element scopes are
// consulted innermost-first — inside a template instance, a path resolves
// against the current list element (an empty path or "/" is the element
// itself) — before falling back to the surface data model, whose entries are
// keyed by the exact path an updateDataModel carried. Outside template
// expansion the scope stack is empty and only the data model is consulted,
// preserving the pre-template lookup behavior.
func (s *Surface) lookupBinding(path string) (any, bool) {
	key := strings.TrimPrefix(path, "/")
	for i := len(s.scope) - 1; i >= 0; i-- {
		if v, ok := resolveElementPath(s.scope[i], key); ok {
			return v, true
		}
	}
	if s.data != nil {
		if v, ok := s.data[key]; ok {
			return v, true
		}
	}
	return nil, false
}

// resolveElementPath resolves a slash-separated path (leading "/" already
// trimmed) against a data-model element. An empty path is the element itself.
// Map segments index string-keyed maps; numeric segments index lists.
func resolveElementPath(el any, key string) (any, bool) {
	if key == "" {
		return el, true
	}
	cur := el
	for _, seg := range strings.Split(key, "/") {
		switch node := cur.(type) {
		case map[string]any:
			v, ok := node[seg]
			if !ok {
				return nil, false
			}
			cur = v
		default:
			rv := reflect.ValueOf(cur)
			switch rv.Kind() {
			case reflect.Map:
				if rv.Type().Key().Kind() != reflect.String {
					return nil, false
				}
				v := rv.MapIndex(reflect.ValueOf(seg))
				if !v.IsValid() {
					return nil, false
				}
				cur = v.Interface()
			case reflect.Slice, reflect.Array:
				i, err := strconv.Atoi(seg)
				if err != nil || i < 0 || i >= rv.Len() {
					return nil, false
				}
				cur = rv.Index(i).Interface()
			default:
				return nil, false
			}
		}
	}
	return cur, true
}

// asList converts a data-model value to a []any list. JSON-decoded lists
// arrive as []any (the fast path); typed slices built in Go (e.g. []string)
// are converted via reflection. Non-list values report false.
func asList(v any) ([]any, bool) {
	if l, ok := v.([]any); ok {
		return l, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}
	out := make([]any, rv.Len())
	for i := range out {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}

// childListIDs returns the component IDs a ChildList references: the explicit
// IDs of the static form, or the template component of the dynamic form. The
// template component counts as referenced so root derivation never mistakes it
// for the surface root and the focus walk reaches interactive components
// inside the template subtree.
func childListIDs(cl a2ui.ChildList) []string {
	if cl.Template != nil {
		return []string{cl.Template.ComponentID}
	}
	return cl.IDs
}
