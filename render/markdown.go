package render

import (
	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
)

// MarkdownModel renders a component.Markdown.
//
// TODO(a2tea): run Source through glamour with a terminal-friendly style,
// cache the rendered string, and re-render on window resize.
type MarkdownModel struct {
	c component.Markdown
}

// NewMarkdown builds a MarkdownModel for the given markdown block.
func NewMarkdown(c component.Markdown) MarkdownModel { return MarkdownModel{c: c} }

// Init implements tea.Model.
func (m MarkdownModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m MarkdownModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, quitOnKey(msg)
}

// View implements tea.Model.
func (m MarkdownModel) View() tea.View { return placeholderView(component.KindMarkdown) }
