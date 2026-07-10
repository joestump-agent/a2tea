package render

import tea "charm.land/bubbletea/v2"

// base carries the size and focus state shared by every renderer so each one
// does not reimplement the SetSize/Focus/Blur/Focused boilerplate. Embed it by
// value; the promoted pointer methods satisfy the size/focus half of Model.
// Real renderers will read width/height when they lay out and focused when they
// decide whether to handle key input.
type base struct {
	width, height int
	focused       bool
}

// SetSize implements Model.
func (b *base) SetSize(width, height int) { b.width, b.height = width, height }

// Focus implements Model.
func (b *base) Focus() tea.Cmd { b.focused = true; return nil }

// Blur implements Model.
func (b *base) Blur() { b.focused = false }

// Focused implements Model.
func (b *base) Focused() bool { return b.focused }
