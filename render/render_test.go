package render_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/joestump-agent/a2tea/component"
	"github.com/joestump-agent/a2tea/render"
)

// oneOfEachKind is one zero-valued component per registered kind. The
// completeness test below asserts render.For handles every one — turning
// "a new kind MUST be added to render.For too" from a comment into a test
// failure.
var oneOfEachKind = []component.Component{
	component.Card{},
	component.Form{},
	component.Input{},
	component.Choice{},
	component.Progress{},
	component.Markdown{},
	component.Stream{},
}

func TestForCoversEveryKind(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range oneOfEachKind {
		m, err := render.For(c)
		if err != nil {
			t.Errorf("For(%s): unexpected error: %v", c.Kind(), err)
			continue
		}
		if m == nil {
			t.Errorf("For(%s): returned a nil Model", c.Kind())
			continue
		}
		seen[c.Kind()] = true
	}

	for _, k := range []string{
		component.KindCard, component.KindForm, component.KindInput,
		component.KindChoice, component.KindProgress, component.KindMarkdown,
		component.KindStream,
	} {
		if !seen[k] {
			t.Errorf("render.For has no case for kind %q", k)
		}
	}
}

// unknownComponent is a Component whose Kind is not registered, used to prove
// For returns an error rather than a placeholder for kinds it cannot render.
type unknownComponent struct{}

func (unknownComponent) Kind() string    { return "table" }
func (unknownComponent) Validate() error { return nil }

func TestForUnknownKindErrors(t *testing.T) {
	m, err := render.For(unknownComponent{})
	if err == nil {
		t.Fatal("For(unknown): expected an error, got nil")
	}
	if m != nil {
		t.Fatalf("For(unknown): model = %#v, want nil", m)
	}
}

// TestModelContract exercises the embeddable-child contract every renderer
// must satisfy: it is a tea.Model, it accepts a size, its focus state is
// settable, and — critically — its Update never returns tea.Quit.
func TestModelContract(t *testing.T) {
	for _, c := range oneOfEachKind {
		m, err := render.For(c)
		if err != nil {
			t.Fatalf("For(%s): %v", c.Kind(), err)
		}

		m.SetSize(80, 24)
		if cmd := m.Focus(); cmd != nil {
			t.Errorf("%s: Focus returned a non-nil cmd from a stub renderer", c.Kind())
		}
		if !m.Focused() {
			t.Errorf("%s: Focused() = false after Focus()", c.Kind())
		}
		m.Blur()
		if m.Focused() {
			t.Errorf("%s: Focused() = true after Blur()", c.Kind())
		}

		// A stray key press must NOT quit an embedded renderer.
		_, cmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
		if cmd != nil {
			t.Errorf("%s: Update on a key returned a non-nil cmd (renderers must not quit): %#v", c.Kind(), cmd())
		}
	}
}
