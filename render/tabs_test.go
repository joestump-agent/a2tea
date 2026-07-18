package render_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	a2ui "github.com/tmc/a2ui"

	"github.com/joestump-agent/a2tea/render"
)

// threeTabs builds a Tabs component with three literal-titled tabs.
func threeTabs(id string) a2ui.Component {
	return a2ui.Component{ID: id, Tabs: &a2ui.TabsComponent{Tabs: []a2ui.TabDef{
		{Title: a2ui.StringLiteral("One"), Child: "c1"},
		{Title: a2ui.StringLiteral("Two"), Child: "c2"},
		{Title: a2ui.StringLiteral("Three"), Child: "c3"},
	}}}
}

// tabsSurface builds a focused surface whose root is a three-tab Tabs
// component with plain text contents.
func tabsSurface(t *testing.T) *render.Surface {
	t.Helper()
	comps := []a2ui.Component{
		threeTabs("tabs"),
		text("c1", "first content"),
		text("c2", "second content"),
		text("c3", "third content"),
	}
	s := render.NewSurface("s", comps)
	s.Focus()
	return s
}

// keyPress builds a KeyPressMsg for a non-printable key like tea.KeyLeft.
func keyPress(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

// TestTabsJoinFocusRing verifies that a Tabs component with tabs is
// focusable, and one without tabs is not.
func TestTabsJoinFocusRing(t *testing.T) {
	s := tabsSurface(t)
	if got := s.Focusables(); len(got) != 1 || got[0] != "tabs" {
		t.Fatalf("focusables = %v, want [tabs]", got)
	}

	empty := render.NewSurface("s", []a2ui.Component{
		{ID: "root", Tabs: &a2ui.TabsComponent{}},
	})
	if got := empty.Focusables(); len(got) != 0 {
		t.Fatalf("empty tabs must not be focusable; focusables = %v", got)
	}
}

// TestTabsLeftRightSwitch verifies that Right/Left on a focused tab bar
// switch the active tab and swap the rendered content.
func TestTabsLeftRightSwitch(t *testing.T) {
	s := tabsSurface(t)

	s.Update(keyPress(tea.KeyRight))
	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "second content") || strings.Contains(out, "first content") {
		t.Fatalf("right should activate tab 2 only: %q", out)
	}
	if got := s.ActiveTab("tabs"); got != 1 {
		t.Fatalf("ActiveTab = %d, want 1", got)
	}
	// The full title bar keeps rendering all titles.
	if !strings.Contains(out, "One │ Two │ Three") {
		t.Fatalf("title bar should list every tab: %q", out)
	}

	s.Update(keyPress(tea.KeyLeft))
	out = ansi.Strip(s.View().Content)
	if !strings.Contains(out, "first content") || strings.Contains(out, "second content") {
		t.Fatalf("left should return to tab 1 only: %q", out)
	}
}

// TestTabsHLSwitch verifies the vi-style h/l aliases switch tabs when the
// tab bar holds focus.
func TestTabsHLSwitch(t *testing.T) {
	s := tabsSurface(t)

	s.Update(typeKey('l'))
	if got := s.ActiveTab("tabs"); got != 1 {
		t.Fatalf("l should advance to tab index 1; ActiveTab = %d", got)
	}
	s.Update(typeKey('h'))
	if got := s.ActiveTab("tabs"); got != 0 {
		t.Fatalf("h should return to tab index 0; ActiveTab = %d", got)
	}
}

// TestTabsSwitchWrapsAround verifies switching wraps at both ends, mirroring
// the focus ring's cycling.
func TestTabsSwitchWrapsAround(t *testing.T) {
	s := tabsSurface(t)

	s.Update(keyPress(tea.KeyLeft)) // left from the first tab lands on the last
	if got := s.ActiveTab("tabs"); got != 2 {
		t.Fatalf("left from tab 0 should wrap to 2; ActiveTab = %d", got)
	}
	s.Update(keyPress(tea.KeyRight)) // right from the last wraps to the first
	if got := s.ActiveTab("tabs"); got != 0 {
		t.Fatalf("right from tab 2 should wrap to 0; ActiveTab = %d", got)
	}
}

// TestTabsSwitchRequiresFocus verifies arrow keys do nothing when the surface
// does not hold focus.
func TestTabsSwitchRequiresFocus(t *testing.T) {
	s := tabsSurface(t)
	s.Blur()

	s.Update(keyPress(tea.KeyRight))
	if got := s.ActiveTab("tabs"); got != 0 {
		t.Fatalf("blurred surface must not switch tabs; ActiveTab = %d", got)
	}
}

// TestTabsActiveTitleChrome verifies the monochrome focus chrome: the active
// title renders reverse-video (ButtonFocused) while the tab bar holds focus
// and bold (Heading) when it does not.
func TestTabsActiveTitleChrome(t *testing.T) {
	st := render.DefaultStyles()

	s := tabsSurface(t) // focused; the tab bar is the only focusable
	raw := s.View().Content
	if !strings.Contains(raw, st.ButtonFocused.Render("One")) {
		t.Fatalf("focused tab bar should reverse-video the active title: %q", raw)
	}

	s.Blur()
	raw = s.View().Content
	if strings.Contains(raw, st.ButtonFocused.Render("One")) {
		t.Fatalf("blurred tab bar must not reverse-video the title: %q", raw)
	}
	if !strings.Contains(raw, st.Heading.Render("One")) {
		t.Fatalf("blurred tab bar should bold the active title: %q", raw)
	}
}

// TestTabsActiveSurvivesApply verifies active-tab state is preserved across
// an updateComponents merge, the same way focus is.
func TestTabsActiveSurvivesApply(t *testing.T) {
	s := tabsSurface(t)
	s.Update(keyPress(tea.KeyRight)) // activate tab 2

	alive := s.Apply([]a2ui.ServerMessage{
		{UpdateComponents: &a2ui.UpdateComponents{SurfaceID: "s", Components: []a2ui.Component{
			text("c2", "second content v2"), // update the active tab's content
		}}},
	})
	if !alive {
		t.Fatal("Apply reported surface as not alive")
	}

	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "second content v2") {
		t.Fatalf("active tab should survive the merge and show updated content: %q", out)
	}
	if strings.Contains(out, "first content") {
		t.Fatalf("merge must not reset the active tab to the first: %q", out)
	}
	if got := s.ActiveTab("tabs"); got != 1 {
		t.Fatalf("ActiveTab = %d after merge, want 1", got)
	}
}

// TestTabsOutOfRangeIndexClampsToFirst verifies that when a component update
// shrinks the tab list below the previously active index, rendering clamps to
// the first tab instead of panicking or pointing past the end.
func TestTabsOutOfRangeIndexClampsToFirst(t *testing.T) {
	s := tabsSurface(t)
	s.Update(keyPress(tea.KeyLeft)) // wrap to the last tab (index 2)

	s.Apply([]a2ui.ServerMessage{
		{UpdateComponents: &a2ui.UpdateComponents{SurfaceID: "s", Components: []a2ui.Component{
			{ID: "tabs", Tabs: &a2ui.TabsComponent{Tabs: []a2ui.TabDef{
				{Title: a2ui.StringLiteral("One"), Child: "c1"},
				{Title: a2ui.StringLiteral("Two"), Child: "c2"},
			}}},
		}}},
	})

	if got := s.ActiveTab("tabs"); got != 0 {
		t.Fatalf("out-of-range active tab should clamp to 0; ActiveTab = %d", got)
	}
	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "first content") {
		t.Fatalf("clamped tabs should render the first tab's content: %q", out)
	}
	if strings.Contains(out, "third content") {
		t.Fatalf("removed tab's content must not render: %q", out)
	}
}

// TestTabsInactiveChildrenLeaveFocusRing verifies that focusables inside
// inactive tabs are excluded from the focus ring, and switching tabs swaps
// which ones are reachable while keeping focus on the tab bar.
func TestTabsInactiveChildrenLeaveFocusRing(t *testing.T) {
	comps := []a2ui.Component{
		{ID: "tabs", Tabs: &a2ui.TabsComponent{Tabs: []a2ui.TabDef{
			{Title: a2ui.StringLiteral("One"), Child: "btn1"},
			{Title: a2ui.StringLiteral("Two"), Child: "btn2"},
		}}},
		actionButton("btn1", "l1", "a1", nil),
		actionButton("btn2", "l2", "a2", nil),
		textLabel("l1", "First"),
		textLabel("l2", "Second"),
	}
	s := render.NewSurface("s", comps)
	s.Focus()

	if got := s.Focusables(); len(got) != 2 || got[0] != "tabs" || got[1] != "btn1" {
		t.Fatalf("focusables = %v, want [tabs btn1]", got)
	}

	s.Update(keyPress(tea.KeyRight))
	if got := s.Focusables(); len(got) != 2 || got[0] != "tabs" || got[1] != "btn2" {
		t.Fatalf("focusables after switch = %v, want [tabs btn2]", got)
	}
	// Focus stays on the tab bar across the switch.
	s.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if got := s.Focusables()[1]; got != "btn2" {
		t.Fatalf("tab should reach the active tab's button; got %q", got)
	}
}

// TestTabsHLStillTypeIntoTextField verifies the h/l aliases do not steal
// keystrokes from a focused text field.
func TestTabsHLStillTypeIntoTextField(t *testing.T) {
	s := fieldSurface(t) // focus starts on the text field

	s.Update(typeKey('h'))
	s.Update(typeKey('l'))
	if got := s.FieldValues()["field"]; got != "initialhl" {
		t.Fatalf("h/l should type into the focused field; value = %q", got)
	}
	// Arrow keys remain non-typing no-ops on a text field.
	s.Update(keyPress(tea.KeyLeft))
	s.Update(keyPress(tea.KeyRight))
	if got := s.FieldValues()["field"]; got != "initialhl" {
		t.Fatalf("arrow keys must not edit the field; value = %q", got)
	}
}

// TestTabsDeleteSurfaceResetsActiveTab verifies deleteSurface wipes active-tab
// state, so a re-created surface starts back on the first tab.
func TestTabsDeleteSurfaceResetsActiveTab(t *testing.T) {
	s := tabsSurface(t)
	s.Update(keyPress(tea.KeyRight)) // activate tab 2

	s.Apply([]a2ui.ServerMessage{
		{DeleteSurface: &a2ui.DeleteSurface{SurfaceID: "s"}},
		{UpdateComponents: &a2ui.UpdateComponents{SurfaceID: "s", Components: []a2ui.Component{
			threeTabs("tabs"),
			text("c1", "first content"),
			text("c2", "second content"),
			text("c3", "third content"),
		}}},
	})

	if got := s.ActiveTab("tabs"); got != 0 {
		t.Fatalf("re-created surface should start on tab 0; ActiveTab = %d", got)
	}
	out := ansi.Strip(s.View().Content)
	if !strings.Contains(out, "first content") || strings.Contains(out, "second content") {
		t.Fatalf("re-created tabs should render the first tab only: %q", out)
	}
}
