package render_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/render"
)

// modalComponents builds a surface tree with a Modal whose content holds a
// text line and a button:
//
//	column(root) -> [modal(m), text(after)]
//	  modal(m): trigger=text(trig), content=column(body) -> [text(bodytext), button(okbtn)]
func modalComponents() []a2ui.Component {
	return []a2ui.Component{
		{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"m", "after"}}}},
		{ID: "m", Modal: &a2ui.ModalComponent{Trigger: "trig", Content: "body"}},
		text("trig", "Open settings"),
		{ID: "body", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"bodytext", "okbtn"}}}},
		text("bodytext", "secret dialog body"),
		actionButton("okbtn", "oklbl", "ok", nil),
		textLabel("oklbl", "OK"),
		text("after", "after the modal"),
	}
}

// modalSurface builds the modal surface and grants it focus (the modal is the
// only focusable while closed, so it holds focus immediately).
func modalSurface(t *testing.T) *render.Surface {
	t.Helper()
	s := render.NewSurface("s", modalComponents())
	s.Focus()
	return s
}

// TestModalClosedRendersTriggerOnly verifies a closed modal renders its
// trigger (behind the idle cue glyph) and hides its content.
func TestModalClosedRendersTriggerOnly(t *testing.T) {
	out := renderPlain(modalComponents())

	if !strings.Contains(out, "▹ Open settings") {
		t.Fatalf("closed modal should render its cued trigger: %q", out)
	}
	if strings.Contains(out, "secret dialog body") {
		t.Fatalf("closed modal should NOT render its content: %q", out)
	}
	if strings.Contains(out, "[ OK ]") {
		t.Fatalf("closed modal should NOT render content buttons: %q", out)
	}
}

// TestModalJoinsFocusRing verifies the modal itself is the focusable element:
// it appears in the ring while its content's button (unreachable when closed)
// and its trigger subtree do not.
func TestModalJoinsFocusRing(t *testing.T) {
	s := modalSurface(t)
	focusables := s.Focusables()
	if len(focusables) != 1 || focusables[0] != "m" {
		t.Fatalf("closed-modal focusables = %v, want [m]", focusables)
	}
}

// TestModalTriggerButtonNotSeparatelyFocusable verifies that when the trigger
// child is itself a Button, it does not join the ring — the modal is the one
// focusable slot, drawn as its trigger.
func TestModalTriggerButtonNotSeparatelyFocusable(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "m", Modal: &a2ui.ModalComponent{Trigger: "trigbtn", Content: "body"}},
		actionButton("trigbtn", "triglbl", "open", nil),
		textLabel("triglbl", "Open"),
		text("body", "content"),
	}
	s := render.NewSurface("s", comps)
	focusables := s.Focusables()
	if len(focusables) != 1 || focusables[0] != "m" {
		t.Fatalf("focusables = %v, want [m] (trigger button must not join the ring)", focusables)
	}
}

// TestModalFocusCue verifies the trigger's cue glyph swaps from "▹" to "▸"
// when the modal holds focus.
func TestModalFocusCue(t *testing.T) {
	// Unfocused surface: idle cue.
	s := render.NewSurface("s", modalComponents())
	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "▹ Open settings") || strings.Contains(out, "▸") {
		t.Fatalf("unfocused modal should show the idle cue: %q", out)
	}

	// Focused surface: the modal is the first (only) focusable.
	s.Focus()
	out = ansi.Strip(s.View().Content)
	if !strings.Contains(out, "▸ Open settings") {
		t.Fatalf("focused modal should show the focus cue: %q", out)
	}
}

// TestEnterOpensModal verifies Enter on the focused modal renders the content
// child as a bordered block and brings the content's focusables into the ring.
func TestEnterOpensModal(t *testing.T) {
	s := modalSurface(t)

	s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if !s.HasOpenModal() {
		t.Fatal("HasOpenModal = false after Enter on the focused modal")
	}
	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "secret dialog body") {
		t.Fatalf("open modal should render its content: %q", out)
	}
	if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
		t.Fatalf("open modal content should be bordered: %q", out)
	}
	// The trigger still renders above the content.
	if strings.Index(out, "Open settings") > strings.Index(out, "secret dialog body") {
		t.Fatalf("trigger should precede the open content: %q", out)
	}

	focusables := s.Focusables()
	if len(focusables) != 2 || focusables[0] != "m" || focusables[1] != "okbtn" {
		t.Fatalf("open-modal focusables = %v, want [m okbtn]", focusables)
	}
}

// TestEnterTogglesModalClosed verifies a second Enter on the focused modal
// closes it again: content hidden, content focusables gone.
func TestEnterTogglesModalClosed(t *testing.T) {
	s := modalSurface(t)

	s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if s.HasOpenModal() {
		t.Fatal("HasOpenModal = true after toggling the modal closed")
	}
	out := ansi.Strip(s.View().Content)
	if strings.Contains(out, "secret dialog body") {
		t.Fatalf("toggled-closed modal should hide its content: %q", out)
	}
	if focusables := s.Focusables(); len(focusables) != 1 || focusables[0] != "m" {
		t.Fatalf("closed-modal focusables = %v, want [m]", focusables)
	}
}

// TestEscClosesModalAndRestoresFocus verifies Esc closes the open modal even
// when focus sits inside its content, and returns focus to the modal (so a
// further Enter reopens it).
func TestEscClosesModalAndRestoresFocus(t *testing.T) {
	s := modalSurface(t)

	// Open, then Tab into the content's button.
	s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	// Esc closes.
	s.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if s.HasOpenModal() {
		t.Fatal("HasOpenModal = true after Esc")
	}
	out := ansi.Strip(s.View().Content)
	if strings.Contains(out, "secret dialog body") {
		t.Fatalf("esc-closed modal should hide its content: %q", out)
	}

	// Focus returned to the modal: Enter reopens it directly.
	s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !s.HasOpenModal() {
		t.Fatal("Enter after Esc should reopen the modal (focus should have returned to it)")
	}
}

// TestEscWithoutOpenModalIsNoOp verifies Esc on a surface with no open modal
// changes nothing — the key is left for the host.
func TestEscWithoutOpenModalIsNoOp(t *testing.T) {
	s := modalSurface(t)

	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil {
		t.Fatal("esc with no open modal should produce no command")
	}
	if s.HasOpenModal() {
		t.Fatal("esc must not open a modal")
	}
	if focusables := s.Focusables(); len(focusables) != 1 || focusables[0] != "m" {
		t.Fatalf("focus ring changed on no-op esc: %v", focusables)
	}
}

// TestModalOpenStatePreservedAcrossApply verifies an updateComponents merge
// that leaves the modal in place does not close it.
func TestModalOpenStatePreservedAcrossApply(t *testing.T) {
	s := modalSurface(t)
	s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	alive := s.Apply([]a2ui.ServerMessage{{
		UpdateComponents: &a2ui.UpdateComponents{
			SurfaceID: "s",
			Components: []a2ui.Component{
				{ID: "after", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("updated tail")}},
			},
		},
	}})
	if !alive {
		t.Fatal("Apply reported surface as not alive")
	}

	if !s.HasOpenModal() {
		t.Fatal("Apply merge closed an open modal")
	}
	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "secret dialog body") {
		t.Fatalf("modal content should survive the Apply merge: %q", out)
	}
	if !strings.Contains(out, "updated tail") {
		t.Fatalf("merged component should render: %q", out)
	}
}

// TestApplyReplacingModalPrunesOpenState verifies that when an update replaces
// the modal with a non-modal component, its open state is pruned rather than
// lingering as a phantom entry.
func TestApplyReplacingModalPrunesOpenState(t *testing.T) {
	s := modalSurface(t)
	s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	s.Apply([]a2ui.ServerMessage{{
		UpdateComponents: &a2ui.UpdateComponents{
			SurfaceID: "s",
			Components: []a2ui.Component{
				{ID: "m", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("not a modal anymore")}},
			},
		},
	}})

	if s.HasOpenModal() {
		t.Fatal("open state should be pruned when the component stops being a modal")
	}
	out := ansi.Strip(s.View().Content)
	if strings.Contains(out, "secret dialog body") {
		t.Fatalf("replaced modal must not render its old content: %q", out)
	}
}

// TestNestedModalsEscClosesInnermostFirst verifies the open stack: with a
// modal open inside another modal's content, the first Esc closes only the
// inner one and the second Esc closes the outer.
func TestNestedModalsEscClosesInnermostFirst(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "outer", Modal: &a2ui.ModalComponent{Trigger: "otrig", Content: "obody"}},
		text("otrig", "Open outer"),
		{ID: "obody", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"otext", "inner"}}}},
		text("otext", "outer body"),
		{ID: "inner", Modal: &a2ui.ModalComponent{Trigger: "itrig", Content: "ibody"}},
		text("itrig", "Open inner"),
		text("ibody", "inner body"),
	}
	s := render.NewSurface("s", comps)
	s.Focus()

	// Open outer, Tab to the inner modal (now in the ring), open it.
	s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "outer body") || !strings.Contains(out, "inner body") {
		t.Fatalf("both modal bodies should render when nested-open: %q", out)
	}

	// First Esc: inner closes, outer stays open.
	s.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	out = ansi.Strip(s.View().Content)
	if strings.Contains(out, "inner body") {
		t.Fatalf("first esc should close the inner modal: %q", out)
	}
	if !strings.Contains(out, "outer body") {
		t.Fatalf("first esc must not close the outer modal: %q", out)
	}

	// Second Esc: outer closes too.
	s.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if s.HasOpenModal() {
		t.Fatal("second esc should close the outer modal")
	}
}
