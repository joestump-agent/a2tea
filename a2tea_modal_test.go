package a2tea_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea"
)

// modalMessages builds the server messages for a surface whose root column
// holds a Modal (trigger text + content text).
func modalMessages() []a2ui.ServerMessage {
	return []a2ui.ServerMessage{{
		UpdateComponents: &a2ui.UpdateComponents{
			SurfaceID: "s",
			Components: []a2ui.Component{
				{ID: "root", Column: &a2ui.ColumnComponent{Children: a2ui.ChildList{IDs: []string{"m"}}}},
				{ID: "m", Modal: &a2ui.ModalComponent{Trigger: "trig", Content: "body"}},
				{ID: "trig", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("Open settings")}},
				{ID: "body", Text: &a2ui.TextComponent{Text: a2ui.StringLiteral("dialog body")}},
			},
		},
	}}
}

// quits reports whether cmd produces a tea.QuitMsg.
func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// TestStandaloneEscClosesModalBeforeQuit verifies the Esc interplay between an
// open modal and Standalone's esc-quit: while a modal is open, Esc closes the
// modal (no quit); once no modal is open, Esc quits as before.
func TestStandaloneEscClosesModalBeforeQuit(t *testing.T) {
	child, err := a2tea.Render(modalMessages())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	m := a2tea.Standalone(child)
	m.Init() // grants the child focus so Enter reaches the modal

	// Enter opens the modal.
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(m2.View().Content, "dialog body") {
		t.Fatalf("modal should be open after Enter: %q", m2.View().Content)
	}

	// Esc with the modal open: consumed by the modal, no quit.
	m3, cmd := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if quits(cmd) {
		t.Fatal("Esc with an open modal must close the modal, not quit")
	}
	if strings.Contains(m3.View().Content, "dialog body") {
		t.Fatalf("Esc should have closed the modal: %q", m3.View().Content)
	}

	// Esc with no modal open: Standalone quits.
	_, cmd = m3.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !quits(cmd) {
		t.Fatal("Esc with no open modal should quit")
	}
}

// TestStandaloneCtrlCQuitsWithOpenModal verifies Ctrl+C stays an unconditional
// quit even while a modal is open — only Esc is consumed by the modal.
func TestStandaloneCtrlCQuitsWithOpenModal(t *testing.T) {
	child, err := a2tea.Render(modalMessages())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	m := a2tea.Standalone(child)
	m.Init()

	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_, cmd := m2.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if !quits(cmd) {
		t.Fatal("Ctrl+C should quit even with an open modal")
	}
}
