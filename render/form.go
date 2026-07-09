package render

import (
	tea "charm.land/bubbletea/v2"

	"github.com/joestump/a2tea/component"
)

// FormModel renders a component.Form.
//
// TODO(a2tea): wire this to huh.Form so field navigation, validation, and
// submission Just Work. On submit, emit a single event.FormSubmitted carrying
// every field's value (not one InputSubmitted per field).
type FormModel struct {
	base
	c component.Form
}

// NewForm builds a FormModel for the given form.
func NewForm(c component.Form) *FormModel { return &FormModel{c: c} }

// Init implements tea.Model.
func (m *FormModel) Init() tea.Cmd { return nil }

// Update implements tea.Model. It is a no-op stub and, per the composition
// contract, never quits — the host owns program exit.
func (m *FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

// View implements tea.Model.
func (m *FormModel) View() tea.View { return placeholderView(component.KindForm) }
