package a2tea_test

import (
	"encoding/json"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/joestump-agent/a2tea"
	"github.com/joestump-agent/a2tea/component"
	"github.com/joestump-agent/a2tea/render"
)

const sampleCard = `{"kind":"card","id":"hello","title":"Hi","body":"b","buttons":[{"id":"ok","label":"OK"}]}`

func TestRenderReturnsRightModel(t *testing.T) {
	m, err := a2tea.Render(json.RawMessage(sampleCard))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if _, ok := m.(*render.CardModel); !ok {
		t.Fatalf("Render returned %T, want *render.CardModel", m)
	}
}

func TestRenderErrorsArePropagated(t *testing.T) {
	cases := []struct {
		name    string
		doc     string
		wantErr error // nil means "just non-nil"
	}{
		{"empty", ``, component.ErrEmptyDocument},
		{"unknown kind", `{"kind":"table"}`, component.ErrUnknownKind},
		{"invalid", `{"kind":"input"}`, component.ErrValidation},
		{"malformed", `{{{`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := a2tea.Render(json.RawMessage(tc.doc))
			if err == nil {
				t.Fatalf("Render(%s): expected an error, got model %#v", tc.name, m)
			}
			if m != nil {
				t.Fatalf("Render(%s): model = %#v, want nil on error", tc.name, m)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("Render(%s): err = %v, want errors.Is %v", tc.name, err, tc.wantErr)
			}
		})
	}
}

// sizeSpy is a tea.Model that records the last size it was given, so the
// Standalone test can assert window-size forwarding.
type sizeSpy struct {
	w, h int
}

func (sizeSpy) Init() tea.Cmd                         { return nil }
func (s sizeSpy) Update(tea.Msg) (tea.Model, tea.Cmd) { return s, nil }
func (sizeSpy) View() tea.View                        { return tea.NewView("") }
func (s *sizeSpy) SetSize(w, h int)                   { s.w, s.h = w, h }

func TestStandaloneQuitsOnKeys(t *testing.T) {
	for _, key := range []string{"q", "esc", "ctrl+c"} {
		var code rune
		var text string
		mod := tea.KeyMod(0)
		switch key {
		case "q":
			code, text = 'q', "q"
		case "esc":
			code = tea.KeyEscape
		case "ctrl+c":
			code, mod = 'c', tea.ModCtrl
		}
		m := a2tea.Standalone(&sizeSpy{})
		_, cmd := m.Update(tea.KeyPressMsg{Code: code, Text: text, Mod: mod})
		if cmd == nil {
			t.Fatalf("%q: Update returned nil cmd, want tea.Quit", key)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("%q: cmd produced %T, want tea.QuitMsg", key, cmd())
		}
	}
}

func TestStandaloneForwardsWindowSize(t *testing.T) {
	spy := &sizeSpy{}
	m := a2tea.Standalone(spy)
	if _, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40}); cmd != nil {
		t.Fatalf("WindowSizeMsg produced an unexpected cmd: %#v", cmd())
	}
	if spy.w != 120 || spy.h != 40 {
		t.Fatalf("child size = (%d,%d), want (120,40)", spy.w, spy.h)
	}
}

func TestStandalonePassesOtherKeysToChild(t *testing.T) {
	// A non-quit key must not quit; it is forwarded to the child.
	m := a2tea.Standalone(&sizeSpy{})
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"}); cmd != nil {
		t.Fatalf("non-quit key produced a cmd: %#v", cmd())
	}
}
