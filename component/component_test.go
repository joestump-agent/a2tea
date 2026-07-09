package component_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/joestump-agent/a2tea/component"
)

// TestUnmarshalKinds asserts every registered kind decodes to the right
// concrete type. It also acts as a completeness guard: allKinds must list one
// case per Kind* constant, so adding a kind without a decode path fails here.
func TestUnmarshalKinds(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want component.Component
	}{
		{
			name: "card",
			doc:  `{"kind":"card","id":"c1","title":"Hi","buttons":[{"id":"ok","label":"OK"}]}`,
			want: component.Card{ID: "c1", Title: "Hi", Buttons: []component.Button{{ID: "ok", Label: "OK"}}},
		},
		{
			name: "input",
			doc:  `{"kind":"input","id":"name","label":"Name","placeholder":"you"}`,
			want: component.Input{ID: "name", Label: "Name", Placeholder: "you"},
		},
		{
			name: "choice",
			doc:  `{"kind":"choice","id":"pick","options":[{"value":"a","label":"A"}]}`,
			want: component.Choice{ID: "pick", Options: []component.ChoiceOption{{Value: "a", Label: "A"}}},
		},
		{
			name: "progress",
			doc:  `{"kind":"progress","id":"p","percent":0.5}`,
			want: component.Progress{ID: "p", Percent: 0.5},
		},
		{
			name: "markdown",
			doc:  `{"kind":"markdown","source":"# hi"}`,
			want: component.Markdown{Source: "# hi"},
		},
		{
			name: "stream",
			doc:  `{"kind":"stream","chunks":["a","b"]}`,
			want: component.Stream{Chunks: []string{"a", "b"}},
		},
		{
			name: "form with heterogeneous fields",
			doc: `{"kind":"form","id":"f","fields":[
				{"kind":"input","id":"name"},
				{"kind":"choice","id":"color","options":[{"value":"r"}]}
			],"submit":{"id":"go","label":"Go"}}`,
			want: component.Form{
				ID: "f",
				Fields: []component.FormField{
					component.Input{ID: "name"},
					component.Choice{ID: "color", Options: []component.ChoiceOption{{Value: "r"}}},
				},
				Submit: component.Button{ID: "go", Label: "Go"},
			},
		},
	}

	seen := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := component.Unmarshal(json.RawMessage(tc.doc))
			if err != nil {
				t.Fatalf("Unmarshal: unexpected error: %v", err)
			}
			if got.Kind() != tc.want.Kind() {
				t.Fatalf("Kind = %q, want %q", got.Kind(), tc.want.Kind())
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("decoded = %#v, want %#v", got, tc.want)
			}
			seen[tc.want.Kind()] = true
		})
	}

	for _, k := range allKinds {
		if !seen[k] {
			t.Errorf("kind %q has no TestUnmarshalKinds case — add one so decoding stays covered", k)
		}
	}
}

// allKinds is the full set of component kinds. Keeping it here (rather than
// deriving it) means adding a Kind* constant without covering it trips the
// completeness check above.
var allKinds = []string{
	component.KindCard,
	component.KindForm,
	component.KindInput,
	component.KindChoice,
	component.KindProgress,
	component.KindMarkdown,
	component.KindStream,
}

func TestUnmarshalEmptyDocument(t *testing.T) {
	for _, doc := range []string{"", "   ", "\n\t"} {
		got, err := component.Unmarshal(json.RawMessage(doc))
		if !errors.Is(err, component.ErrEmptyDocument) {
			t.Errorf("Unmarshal(%q): err = %v, want ErrEmptyDocument", doc, err)
		}
		if got != nil {
			t.Errorf("Unmarshal(%q): component = %#v, want nil", doc, got)
		}
	}
}

func TestUnmarshalUnknownKind(t *testing.T) {
	_, err := component.Unmarshal(json.RawMessage(`{"kind":"table"}`))
	if !errors.Is(err, component.ErrUnknownKind) {
		t.Fatalf("err = %v, want ErrUnknownKind", err)
	}
}

func TestUnmarshalMalformedJSON(t *testing.T) {
	_, err := component.Unmarshal(json.RawMessage(`{{{not json`))
	if err == nil {
		t.Fatal("expected an error for malformed JSON, got nil")
	}
	if errors.Is(err, component.ErrUnknownKind) || errors.Is(err, component.ErrEmptyDocument) {
		t.Fatalf("malformed JSON should not report a sentinel kind/empty error: %v", err)
	}
}

// TestUnmarshalTypeMismatchIsAnError is the regression for the original bug:
// a wrong field type used to degrade silently to a zero-valued component.
func TestUnmarshalTypeMismatchIsAnError(t *testing.T) {
	_, err := component.Unmarshal(json.RawMessage(`{"kind":"card","title":12345,"buttons":"nope"}`))
	if err == nil {
		t.Fatal("expected a decode error for a numeric title / string buttons, got nil")
	}
}

func TestValidation(t *testing.T) {
	cases := []struct {
		name string
		doc  string
	}{
		{"card button missing id", `{"kind":"card","buttons":[{"label":"OK"}]}`},
		{"card button missing label", `{"kind":"card","buttons":[{"id":"ok"}]}`},
		{"input missing id", `{"kind":"input","label":"Name"}`},
		{"choice missing id", `{"kind":"choice","options":[{"value":"a"}]}`},
		{"choice option missing value", `{"kind":"choice","id":"c","options":[{"label":"A"}]}`},
		{"progress percent too high", `{"kind":"progress","percent":42}`},
		{"progress percent negative", `{"kind":"progress","percent":-0.5}`},
		{"markdown missing source", `{"kind":"markdown"}`},
		{"form field missing id", `{"kind":"form","fields":[{"kind":"input"}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := component.Unmarshal(json.RawMessage(tc.doc))
			if !errors.Is(err, component.ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}

// TestProgressIndeterminateSkipsRange confirms an indeterminate bar isn't
// range-checked (Percent is meaningless when Indeterminate is true).
func TestProgressIndeterminateSkipsRange(t *testing.T) {
	_, err := component.Unmarshal(json.RawMessage(`{"kind":"progress","indeterminate":true,"percent":42}`))
	if err != nil {
		t.Fatalf("indeterminate progress should not be range-checked: %v", err)
	}
}

func TestFormFieldUnknownKindIsRejected(t *testing.T) {
	_, err := component.Unmarshal(json.RawMessage(`{"kind":"form","fields":[{"kind":"card","id":"x"}]}`))
	if !errors.Is(err, component.ErrUnknownKind) {
		t.Fatalf("err = %v, want ErrUnknownKind for a non-input form field", err)
	}
}

// TestFormFieldDefaultsToInput documents the terse-input ergonomic: a field
// with no "kind" decodes as an Input.
func TestFormFieldDefaultsToInput(t *testing.T) {
	got, err := component.Unmarshal(json.RawMessage(`{"kind":"form","id":"f","fields":[{"id":"name"}]}`))
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	form, ok := got.(component.Form)
	if !ok {
		t.Fatalf("got %T, want component.Form", got)
	}
	if len(form.Fields) != 1 {
		t.Fatalf("len(Fields) = %d, want 1", len(form.Fields))
	}
	if _, ok := form.Fields[0].(component.Input); !ok {
		t.Fatalf("field[0] = %T, want component.Input", form.Fields[0])
	}
}
