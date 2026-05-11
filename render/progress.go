package render

import (
	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
)

// ProgressModel renders a component.Progress.
//
// TODO(a2tea): back this with bubbles/progress. Drive the bar from a
// tea.Msg that the agent emits as work advances.
type ProgressModel struct {
	c component.Progress
}

// NewProgress builds a ProgressModel for the given progress component.
func NewProgress(c component.Progress) ProgressModel { return ProgressModel{c: c} }

// Init implements tea.Model.
func (m ProgressModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, quitOnKey(msg)
}

// View implements tea.Model.
func (m ProgressModel) View() tea.View { return placeholderView(component.KindProgress) }
