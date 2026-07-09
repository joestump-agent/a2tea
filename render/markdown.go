package render

import (
	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
)

// MarkdownModel renders a component.Markdown.
//
// TODO(a2tea): run Source through glamour with a terminal-friendly style,
// cache the rendered string, and re-render when SetSize changes the width.
type MarkdownModel struct {
	base
	c component.Markdown
}

// NewMarkdown builds a MarkdownModel for the given markdown block.
func NewMarkdown(c component.Markdown) *MarkdownModel { return &MarkdownModel{c: c} }

// Init implements tea.Model.
func (m *MarkdownModel) Init() tea.Cmd { return nil }

// Update implements tea.Model. It is a no-op stub and, per the composition
// contract, never quits — the host owns program exit.
func (m *MarkdownModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

// View implements tea.Model.
func (m *MarkdownModel) View() tea.View { return placeholderView(component.KindMarkdown) }
