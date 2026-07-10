package a2tea_test

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/joestump-agent/a2tea"
)

// sampleReply is an LLM-style reply: prose wrapping an <a2ui-json> block whose
// surface is a card containing a single text component.
const sampleReply = `intro text <a2ui-json>{"version":"v0.9","updateComponents":{"surfaceId":"s","components":[{"component":"Card","id":"root","child":"t"},{"component":"Text","id":"t","text":"Hi there"}]}}</a2ui-json> outro`

func TestContains(t *testing.T) {
	if !a2tea.Contains(sampleReply) {
		t.Error("Contains(reply with a2ui block) = false, want true")
	}
	if a2tea.Contains("just some prose, no ui") {
		t.Error("Contains(plain prose) = true, want false")
	}
}

func TestScanSplitsTextAndMessages(t *testing.T) {
	parts, err := a2tea.Scan(sampleReply)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	var texts []string
	msgCount := 0
	for _, p := range parts {
		if s := strings.TrimSpace(p.Text); s != "" {
			texts = append(texts, s)
		}
		msgCount += len(p.Messages)
	}
	if msgCount != 1 {
		t.Fatalf("message count = %d, want 1", msgCount)
	}
	joined := strings.Join(texts, "|")
	if !strings.Contains(joined, "intro text") || !strings.Contains(joined, "outro") {
		t.Fatalf("text parts = %q, want both intro and outro", joined)
	}
}

func TestRenderSurface(t *testing.T) {
	parts, err := a2tea.Scan(sampleReply)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	for _, p := range parts {
		if len(p.Messages) == 0 {
			continue
		}
		m, err := a2tea.Render(p.Messages)
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		out := m.View().Content
		if !strings.Contains(out, "Hi there") {
			t.Fatalf("rendered surface = %q, want the card's text", out)
		}
		return
	}
	t.Fatal("no message part found in scan")
}

func TestRenderNoRenderableSurface(t *testing.T) {
	// createSurface alone has nothing to draw.
	parts, err := a2tea.Scan(`<a2ui-json>{"version":"v0.9","createSurface":{"surfaceId":"s","catalogId":"c"}}</a2ui-json>`)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	for _, p := range parts {
		if len(p.Messages) == 0 {
			continue
		}
		if _, err := a2tea.Render(p.Messages); !errors.Is(err, a2tea.ErrNoRenderableSurface) {
			t.Fatalf("Render err = %v, want ErrNoRenderableSurface", err)
		}
		return
	}
	t.Fatal("no message part found in scan")
}

// sizeSpy is a tea.Model that records the last size it was given, so the
// Standalone tests can assert window-size forwarding.
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
