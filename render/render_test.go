package render_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/event"
	"github.com/joestump-agent/a2tea/render"
)

func text(id, s string) a2ui.Component {
	return a2ui.Component{ID: id, Text: &a2ui.TextComponent{Text: a2ui.StringLiteral(s)}}
}

// renderPlain renders comps as a surface and returns the view with all ANSI
// styling stripped, so tests assert on structure rather than escape codes.
func renderPlain(comps []a2ui.Component) string {
	return ansi.Strip(render.NewSurface("s", comps).View().Content)
}

// surface: card(root) -> column(col) -> [title, body]
func sampleComponents() []a2ui.Component {
	return []a2ui.Component{
		{ID: "root", Card: &a2ui.CardComponent{Child: "col"}},
		{ID: "col", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"title", "body"}}}},
		text("title", "Title Line"),
		text("body", "Body line"),
	}
}

func TestSurfaceRendersTree(t *testing.T) {
	out := renderPlain(sampleComponents())

	// Both text leaves appear.
	titleIdx := strings.Index(out, "Title Line")
	bodyIdx := strings.Index(out, "Body line")
	if titleIdx < 0 || bodyIdx < 0 {
		t.Fatalf("rendered surface missing text: %q", out)
	}
	// The column stacks them in order on separate lines (the card border adds
	// chrome around each line, so exact adjacency is not asserted).
	if titleIdx > bodyIdx {
		t.Fatalf("title should render before body: %q", out)
	}
	if !strings.Contains(out[titleIdx:bodyIdx], "\n") {
		t.Fatalf("column should stack children on separate lines: %q", out)
	}
	// The card draws a rounded border around the column.
	if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
		t.Fatalf("card should draw a rounded border: %q", out)
	}
}

func TestSurfaceRootDetection(t *testing.T) {
	// Declaration order deliberately does NOT put the root first: the root is
	// the component nothing else references as a child.
	comps := []a2ui.Component{
		text("title", "Only Child"),
		{ID: "root", Card: &a2ui.CardComponent{Child: "title"}},
	}
	out := renderPlain(comps)
	if !strings.Contains(out, "Only Child") {
		t.Fatalf("root not resolved to the card: %q", out)
	}
}

func TestSurfaceSharedChildIsNotACycle(t *testing.T) {
	// "shared" is legally referenced by two parents (adjacency-list reuse). It
	// must render at both sites, not trip the cycle guard on the second.
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"row1", "row2"}}}},
		{ID: "row1", Row: &a2ui.RowComponent{Children: a2ui.ChildList{IDs: []string{"shared"}}}},
		{ID: "row2", Row: &a2ui.RowComponent{Children: a2ui.ChildList{IDs: []string{"shared"}}}},
		text("shared", "twice"),
	}
	out := renderPlain(comps)
	if strings.Contains(out, "cycle") {
		t.Fatalf("shared child wrongly flagged as a cycle: %q", out)
	}
	if got := strings.Count(out, "twice"); got != 2 {
		t.Fatalf("shared child should render at both parents; %q rendered %d times", "twice", got)
	}
}

// TestSurfaceSharedChildStillCatchesGenuineCycle ensures the path-scoped
// guard does not weaken cycle detection: a component referenced by two
// parents must still be caught when one of those parents is its own
// descendant (a real ancestor loop).
func TestSurfaceSharedChildStillCatchesGenuineCycle(t *testing.T) {
	// root -> a -> loop (ancestor cycle through "loop")
	// root -> b -> shared (harmless reuse)
	// "shared" appears under two parents but is not in a cycle; "loop"
	// is its own ancestor and must be flagged.
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"a", "b"}}}},
		{ID: "a", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"loop"}}}},
		{ID: "loop", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"a"}}}},
		{ID: "b", Row: &a2ui.RowComponent{Children: a2ui.ChildList{IDs: []string{"shared"}}}},
		text("shared", "ok"),
	}
	out := renderPlain(comps)
	if !strings.Contains(out, "cycle") {
		t.Fatalf("genuine ancestor cycle not caught: %q", out)
	}
	// The shared (non-cyclic) child must still render normally.
	if !strings.Contains(out, "ok") {
		t.Fatalf("non-cyclic shared child should still render: %q", out)
	}
}

func TestSurfaceGenuineCycleIsCaught(t *testing.T) {
	// root -> a -> root is a real ancestor loop and must still be caught.
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"a"}}}},
		{ID: "a", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"root"}}}},
	}
	out := renderPlain(comps)
	if !strings.Contains(out, "cycle") {
		t.Fatalf("genuine cycle not caught: %q", out)
	}
}

func TestSurfaceEmpty(t *testing.T) {
	out := renderPlain(nil)
	if !strings.Contains(out, "empty surface") {
		t.Fatalf("empty surface = %q, want a placeholder", out)
	}
}

func TestSurfaceMissingChildIsFlagged(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Card: &a2ui.CardComponent{Child: "nope"}},
	}
	out := renderPlain(comps)
	if !strings.Contains(out, "missing component") {
		t.Fatalf("dangling child ref should be flagged: %q", out)
	}
}

func TestButtonChrome(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "btn", Button: &a2ui.ButtonComponent{Child: "lbl"}},
		text("lbl", "OK"),
	}
	out := renderPlain(comps)
	if !strings.Contains(out, "[ OK ]") {
		t.Fatalf("button should render bracketed chrome: %q", out)
	}
}

func TestButtonFocusCycleAndActivate(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"b1", "b2"}}}},
		{ID: "b1", Button: &a2ui.ButtonComponent{Child: "l1"}},
		{ID: "b2", Button: &a2ui.ButtonComponent{Child: "l2"}},
		text("l1", "First"),
		text("l2", "Second"),
	}
	s := render.NewSurface("surf", comps)
	s.Focus()

	press := func(code rune) tea.Cmd {
		_, cmd := s.Update(tea.KeyPressMsg{Code: code})
		return cmd
	}
	clicked := func(t *testing.T, cmd tea.Cmd) event.ButtonClicked {
		t.Helper()
		if cmd == nil {
			t.Fatal("enter on a focused button returned nil cmd")
		}
		msg := cmd()
		ev, ok := msg.(event.ButtonClicked)
		if !ok {
			t.Fatalf("cmd produced %T, want event.ButtonClicked", msg)
		}
		return ev
	}

	// Initial focus is the first button in tree order.
	ev := clicked(t, press(tea.KeyEnter))
	if ev.ID != "b1" || ev.ComponentID != "b1" || ev.SurfaceID != "surf" {
		t.Fatalf("first activation = %+v, want ID/ComponentID b1 on surface surf", ev)
	}

	// Tab moves focus to the second button.
	if cmd := press(tea.KeyTab); cmd != nil {
		t.Fatalf("tab produced an unexpected cmd: %#v", cmd())
	}
	ev = clicked(t, press(tea.KeyEnter))
	if ev.ID != "b2" || ev.ComponentID != "b2" || ev.SurfaceID != "surf" {
		t.Fatalf("post-tab activation = %+v, want ID/ComponentID b2 on surface surf", ev)
	}
}

// collectMsgs executes a cmd (which may be a single cmd or a tea.Batch) and
// returns all messages it produces. For a Batch, it iterates BatchMsg's
// sub-cmds and calls each one.
func collectMsgs(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		msgs := make([]tea.Msg, 0, len(batch))
		for _, c := range batch {
			msgs = append(msgs, c())
		}
		return msgs
	}
	return []tea.Msg{msg}
}

// findMsg returns the first message of type T in msgs, or t.Fatals if none.
func findMsg[T any](t *testing.T, msgs []tea.Msg) T {
	t.Helper()
	for _, m := range msgs {
		if v, ok := m.(T); ok {
			return v
		}
	}
	t.Fatalf("no %T in %d messages", *new(T), len(msgs))
	return *new(T)
}

// hasMsg reports whether any message in msgs is of type T.
func hasMsg[T any](msgs []tea.Msg) bool {
	for _, m := range msgs {
		if _, ok := m.(T); ok {
			return true
		}
	}
	return false
}

// TestButtonActivationEmitsActionEvent verifies that activating a focused
// button with a server-side Action.Event emits both event.ButtonClicked
// (carrying the resolved *a2ui.EventAction) and a protocol-native
// a2ui.ClientMessage whose ActionEvent has the right Name and source IDs.
func TestButtonActivationEmitsActionEvent(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"btn"}}}},
		{ID: "btn", Button: &a2ui.ButtonComponent{
			Child: "lbl",
			Action: a2ui.Action{Event: &a2ui.EventAction{
				Name: "setProvider",
			}},
		}},
		text("lbl", "Set Provider"),
	}
	s := render.NewSurface("surf", comps)
	s.Focus()

	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msgs := collectMsgs(t, cmd)

	// event.ButtonClicked carries the resolved Action.
	clicked := findMsg[event.ButtonClicked](t, msgs)
	if clicked.ID != "btn" || clicked.ComponentID != "btn" || clicked.SurfaceID != "surf" {
		t.Fatalf("ButtonClicked = %+v, want ID/ComponentID btn on surf", clicked)
	}
	if clicked.Action == nil || clicked.Action.Name != "setProvider" {
		t.Fatalf("ButtonClicked.Action = %+v, want Name setProvider", clicked.Action)
	}

	// a2ui.ClientMessage carries the protocol-native ActionEvent.
	cm := findMsg[a2ui.ClientMessage](t, msgs)
	if cm.Action == nil {
		t.Fatal("ClientMessage.Action is nil")
	}
	if cm.Action.Name != "setProvider" {
		t.Fatalf("ActionEvent.Name = %q, want setProvider", cm.Action.Name)
	}
	if cm.Action.SurfaceID != "surf" {
		t.Fatalf("ActionEvent.SurfaceID = %q, want surf", cm.Action.SurfaceID)
	}
	if cm.Action.SourceComponentID != "btn" {
		t.Fatalf("ActionEvent.SourceComponentID = %q, want btn", cm.Action.SourceComponentID)
	}
	if cm.Version != a2ui.Version {
		t.Fatalf("ClientMessage.Version = %q, want %s", cm.Version, a2ui.Version)
	}
	// Timestamp is left empty for the host to stamp (documented choice).
	if cm.Action.Timestamp != "" {
		t.Fatalf("ActionEvent.Timestamp = %q, want empty (host stamps it)", cm.Action.Timestamp)
	}
}

// TestButtonActivationFunctionCallNoClientMessage verifies that a button
// whose Action is a FunctionCall (client-side fn, no server event) emits
// event.ButtonClicked with a nil Action but does NOT produce a spurious
// a2ui.ClientMessage.
func TestButtonActivationFunctionCallNoClientMessage(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"btn"}}}},
		{ID: "btn", Button: &a2ui.ButtonComponent{
			Child: "lbl",
			Action: a2ui.Action{FunctionCall: &a2ui.FunctionCall{
				Call: "openUrl",
			}},
		}},
		text("lbl", "Open"),
	}
	s := render.NewSurface("surf", comps)
	s.Focus()

	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msgs := collectMsgs(t, cmd)

	// ButtonClicked is emitted with nil Action (no server event).
	clicked := findMsg[event.ButtonClicked](t, msgs)
	if clicked.ID != "btn" {
		t.Fatalf("ButtonClicked.ID = %q, want btn", clicked.ID)
	}
	if clicked.Action != nil {
		t.Fatalf("ButtonClicked.Action = %+v, want nil for FunctionCall-only button", clicked.Action)
	}

	// No ClientMessage should be produced for a client-side function call.
	if hasMsg[a2ui.ClientMessage](msgs) {
		t.Fatal("FunctionCall-only button should not emit a ClientMessage")
	}
}

func TestDividerRendersRule(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "d", Divider: &a2ui.DividerComponent{}},
	}
	out := renderPlain(comps)
	if !strings.Contains(out, "─") {
		t.Fatalf("horizontal divider should render a rule: %q", out)
	}
}

func TestVerticalListBullets(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", List: &a2ui.ListComponent{Children: a2ui.ChildList{IDs: []string{"i1", "i2"}}}},
		text("i1", "alpha"),
		text("i2", "beta"),
	}
	out := renderPlain(comps)
	if got := strings.Count(out, "• "); got != 2 {
		t.Fatalf("vertical list should bullet each item; got %d bullets in %q", got, out)
	}
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Fatalf("list items missing: %q", out)
	}
}

func TestCheckBoxMarks(t *testing.T) {
	checkbox := func(checked bool) []a2ui.Component {
		return []a2ui.Component{
			{ID: "cb", CheckBox: &a2ui.CheckBoxComponent{
				Label: a2ui.StringLiteral("Done"),
				Value: a2ui.BoolLiteral(checked),
			}},
		}
	}
	if out := renderPlain(checkbox(true)); !strings.Contains(out, "[x] Done") {
		t.Fatalf("checked box = %q, want \"[x] Done\"", out)
	}
	if out := renderPlain(checkbox(false)); !strings.Contains(out, "[ ] Done") {
		t.Fatalf("unchecked box = %q, want \"[ ] Done\"", out)
	}
}

func TestSliderRendersBar(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "sl", Slider: &a2ui.SliderComponent{
			Max:   100,
			Value: a2ui.NumberLiteral(50),
		}},
	}
	out := renderPlain(comps)
	// Mid-range value: bar has both filled and empty cells, value appended.
	if !strings.Contains(out, "█") || !strings.Contains(out, "─") {
		t.Fatalf("slider should render a partially filled bar: %q", out)
	}
	if !strings.Contains(out, "50") {
		t.Fatalf("slider should append its numeric value: %q", out)
	}
}

func TestTextH1KeepsText(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "t", Text: &a2ui.TextComponent{
			Text:    a2ui.StringLiteral("Big Title"),
			Variant: a2ui.TextVariantH1,
		}},
	}
	if out := renderPlain(comps); !strings.Contains(out, "Big Title") {
		t.Fatalf("h1 text lost its content: %q", out)
	}
}

func TestKindOf(t *testing.T) {
	cases := []struct {
		c    a2ui.Component
		want string
	}{
		{a2ui.Component{Text: &a2ui.TextComponent{}}, "text"},
		{a2ui.Component{Card: &a2ui.CardComponent{}}, "card"},
		{a2ui.Component{Button: &a2ui.ButtonComponent{}}, "button"},
		{a2ui.Component{Row: &a2ui.RowComponent{}}, "row"},
		{a2ui.Component{Column: &a2ui.ColumnComponent{}}, "column"},
		{a2ui.Component{}, "unknown"},
	}
	for _, tc := range cases {
		if got := render.KindOf(tc.c); got != tc.want {
			t.Errorf("KindOf(%+v) = %q, want %q", tc.c, got, tc.want)
		}
	}
}

// TestFocusablesDedupSharedButton pins that a button referenced by two
// parents (legal adjacency-list reuse) occupies one slot in the focus ring,
// not two — Tab must not visit the same interactive element twice.
func TestFocusablesDedupSharedButton(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"r1", "r2", "other"}}}},
		{ID: "r1", Row: &a2ui.RowComponent{Children: a2ui.ChildList{IDs: []string{"shared-btn"}}}},
		{ID: "r2", Row: &a2ui.RowComponent{Children: a2ui.ChildList{IDs: []string{"shared-btn"}}}},
		{ID: "shared-btn", Button: &a2ui.ButtonComponent{Child: "lbl"}},
		{ID: "other", Button: &a2ui.ButtonComponent{Child: "lbl2"}},
		text("lbl", "Go"),
		text("lbl2", "Stop"),
	}
	s := render.NewSurface("s", comps)
	s.Focus()

	// Tab through the ring: with dedup there are exactly two stops, so two
	// tabs return to the start. Activate and check we land on each button
	// exactly once per cycle.
	seen := map[string]int{}
	for i := 0; i < 2; i++ {
		_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		if cmd == nil {
			t.Fatal("enter on focused surface returned no cmd")
		}
		click, ok := cmd().(event.ButtonClicked)
		if !ok {
			t.Fatalf("cmd yielded %T, want event.ButtonClicked", cmd())
		}
		seen[click.ID]++
		s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	}
	if seen["shared-btn"] != 1 || seen["other"] != 1 {
		t.Fatalf("focus ring should visit each button once per cycle, got %v", seen)
	}
}
