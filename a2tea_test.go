package a2tea_test

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/joestump-agent/a2tea"
	"github.com/joestump-agent/a2tea/render"
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

// TestScanMalformedJSONInTags pins the ACTUAL behavior of Scan on malformed
// JSON inside <a2ui-json> tags: a2uistream.ParseAndValidate does NOT return
// an error — it silently drops the unparseable block and keeps the
// surrounding text as text parts with zero messages. (Verified empirically
// against a2uistream: truncated JSON and outright non-JSON both come back as
// err == nil with no ServerMessages.) If a2uistream ever starts erroring
// here, this test should be revisited deliberately, not just patched.
func TestScanMalformedJSONInTags(t *testing.T) {
	cases := []string{
		// Truncated JSON: the block opens but never closes its objects.
		`before <a2ui-json>{"version":"v0.9","updateComponents":{</a2ui-json> after`,
		// Not JSON at all.
		`before <a2ui-json>this is not json at all</a2ui-json> after`,
	}
	for _, in := range cases {
		parts, err := a2tea.Scan(in)
		if err != nil {
			t.Fatalf("Scan(%q) err = %v, want nil (malformed blocks are dropped, not errored)", in, err)
		}
		total := 0
		var texts []string
		for _, p := range parts {
			total += len(p.Messages)
			if s := strings.TrimSpace(p.Text); s != "" {
				texts = append(texts, s)
			}
		}
		if total != 0 {
			t.Fatalf("Scan(%q) produced %d messages from a malformed block, want 0", in, total)
		}
		joined := strings.Join(texts, "|")
		if !strings.Contains(joined, "before") || !strings.Contains(joined, "after") {
			t.Fatalf("Scan(%q) should keep surrounding text; got parts %q", in, joined)
		}
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

// TestRenderForwardsWithStyles verifies that render.Option values passed to the
// public a2tea.Render are threaded through to the Surface — the host-facing half
// of the theming API. sampleReply is a Card, so a themed border foreground shows
// up as a color SGR; the default (no options) path stays monochrome.
func TestRenderForwardsWithStyles(t *testing.T) {
	parts, err := a2tea.Scan(sampleReply)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	for _, p := range parts {
		if len(p.Messages) == 0 {
			continue
		}

		st := render.DefaultStyles()
		st.CardBorder = st.CardBorder.BorderForeground(lipgloss.Color("99"))
		themed, err := a2tea.Render(p.Messages, render.WithStyles(st))
		if err != nil {
			t.Fatalf("Render themed: %v", err)
		}
		if raw := themed.View().Content; !strings.Contains(raw, "38;5;99") {
			t.Fatalf("a2tea.Render did not forward WithStyles; want color 99 in border, got %q", raw)
		}

		plain, err := a2tea.Render(p.Messages)
		if err != nil {
			t.Fatalf("Render default: %v", err)
		}
		if raw := plain.View().Content; strings.Contains(raw, "38;5;99") {
			t.Fatalf("default a2tea.Render should be monochrome, got %q", raw)
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

// bareJSONReply is an LLM reply carrying an untagged A2UI JSON object — the
// bare form Scan parses; Contains must agree with Scan on it.
const bareJSONReply = `Here is your card. {"version":"v0.9","updateComponents":{"surfaceId":"s","components":[{"component":"Text","id":"root","text":"Hi"}]}} Enjoy!`

// TestContainsBareJSON verifies Contains detects untagged A2UI JSON that
// Scan successfully parses — the two must never disagree.
func TestContainsBareJSON(t *testing.T) {
	if !a2tea.Contains(bareJSONReply) {
		t.Fatal("Contains = false for bare A2UI JSON that Scan parses")
	}
	parts, err := a2tea.Scan(bareJSONReply)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	total := 0
	for _, p := range parts {
		total += len(p.Messages)
	}
	if total == 0 {
		t.Fatal("fixture no longer parses as bare A2UI — update the test")
	}
}

// TestContainsProseMentioningKeyIsFalse verifies the parse-confirm stage: a
// reply that merely mentions an A2UI key in prose (no valid JSON) is not
// reported as containing A2UI.
func TestContainsProseMentioningKeyIsFalse(t *testing.T) {
	prose := `The "updateComponents" message is how A2UI updates a surface.`
	if a2tea.Contains(prose) {
		t.Fatal("Contains = true for prose that merely mentions an A2UI key")
	}
}

// TestContainsPlainProseIsFalse pins the cheap path: no A2UI keys at all.
func TestContainsPlainProseIsFalse(t *testing.T) {
	if a2tea.Contains("Just a normal reply about the weather.") {
		t.Fatal("Contains = true for plain prose")
	}
}
