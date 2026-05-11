// Package component defines the typed union of A2UI components that a2tea
// understands. Each concrete type corresponds to one component "kind" in the
// A2UI JSON schema (see https://a2ui.org).
//
// The interface is deliberately small: a Component knows its Kind so that
// dispatch tables in sibling packages (render, event) can pick the right
// implementation without type switches sprinkled across the codebase.
package component

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Component is the closed union of all A2UI components a2tea can render.
//
// New component kinds MUST be added here AND in the Unmarshal switch below,
// otherwise they will be silently rejected.
type Component interface {
	// Kind returns the A2UI component kind string ("card", "form", ...).
	// It is the dispatch key used by render.For and event routing.
	Kind() string
}

// Common kind constants. Keep these as a single source of truth so renderers
// and tests can reference them without stringly-typed drift.
const (
	KindCard     = "card"
	KindForm     = "form"
	KindInput    = "input"
	KindChoice   = "choice"
	KindProgress = "progress"
	KindMarkdown = "markdown"
	KindStream   = "stream"
)

// Card is an A2UI card: a titled container with body content and optional
// action buttons.
//
// TODO(a2tea): flesh out fields once the A2UI schema for cards is pinned
// down (title styling, body markdown vs plain, button variants, etc).
type Card struct {
	ID      string   `json:"id,omitempty"`
	Title   string   `json:"title,omitempty"`
	Body    string   `json:"body,omitempty"`
	Buttons []Button `json:"buttons,omitempty"`
}

// Kind implements Component.
func (Card) Kind() string { return KindCard }

// Button is a clickable action embedded in a Card or Form.
type Button struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Form is an A2UI form: an ordered collection of input fields plus a submit
// action.
//
// TODO(a2tea): decide whether Fields should be a typed slice of Input/Choice
// or a heterogenous []Component. The current shape assumes homogeneous
// inputs, which is wrong for real forms.
type Form struct {
	ID     string  `json:"id,omitempty"`
	Title  string  `json:"title,omitempty"`
	Fields []Input `json:"fields,omitempty"`
	Submit Button  `json:"submit,omitempty"`
}

// Kind implements Component.
func (Form) Kind() string { return KindForm }

// Input is a single-line text input.
type Input struct {
	ID          string `json:"id"`
	Label       string `json:"label,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Value       string `json:"value,omitempty"`
}

// Kind implements Component.
func (Input) Kind() string { return KindInput }

// Choice is a single-select picker over a list of options.
type Choice struct {
	ID      string         `json:"id"`
	Label   string         `json:"label,omitempty"`
	Options []ChoiceOption `json:"options,omitempty"`
}

// Kind implements Component.
func (Choice) Kind() string { return KindChoice }

// ChoiceOption is one option in a Choice.
type ChoiceOption struct {
	Value string `json:"value"`
	Label string `json:"label,omitempty"`
}

// Progress is a determinate or indeterminate progress indicator. Percent is
// in [0.0, 1.0] when Indeterminate is false.
type Progress struct {
	ID            string  `json:"id,omitempty"`
	Label         string  `json:"label,omitempty"`
	Percent       float64 `json:"percent,omitempty"`
	Indeterminate bool    `json:"indeterminate,omitempty"`
}

// Kind implements Component.
func (Progress) Kind() string { return KindProgress }

// Markdown is a block of rendered markdown content.
type Markdown struct {
	ID     string `json:"id,omitempty"`
	Source string `json:"source"`
}

// Kind implements Component.
func (Markdown) Kind() string { return KindMarkdown }

// Stream represents an append-only stream of text chunks, the way an agent
// would emit a streaming response.
//
// TODO(a2tea): add a channel or callback for live chunk delivery; the static
// Chunks slice here only covers the "replay a finished stream" case.
type Stream struct {
	ID     string   `json:"id,omitempty"`
	Chunks []string `json:"chunks,omitempty"`
}

// Kind implements Component.
func (Stream) Kind() string { return KindStream }

// ErrUnknownKind is returned by Unmarshal when the "kind" discriminator on
// the incoming JSON does not match any registered component type.
var ErrUnknownKind = errors.New("a2tea/component: unknown kind")

// Unmarshal decodes a raw A2UI JSON document into a concrete Component.
//
// TODO(a2tea): this is a stub. The real implementation needs to:
//   - peek at the "kind" discriminator,
//   - dispatch to the correct concrete type's UnmarshalJSON,
//   - validate required fields per the A2UI schema,
//   - and probably support a nested "children" array for container kinds.
//
// For now it recognizes the discriminator and returns a zero-valued struct
// of the right kind, which is enough to wire the rest of the pipeline.
func Unmarshal(raw json.RawMessage) (Component, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var head struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("a2tea/component: decode discriminator: %w", err)
	}
	switch head.Kind {
	case KindCard:
		var c Card
		// TODO(a2tea): actually decode the rest of the document into c.
		_ = json.Unmarshal(raw, &c)
		return c, nil
	case KindForm:
		var c Form
		_ = json.Unmarshal(raw, &c)
		return c, nil
	case KindInput:
		var c Input
		_ = json.Unmarshal(raw, &c)
		return c, nil
	case KindChoice:
		var c Choice
		_ = json.Unmarshal(raw, &c)
		return c, nil
	case KindProgress:
		var c Progress
		_ = json.Unmarshal(raw, &c)
		return c, nil
	case KindMarkdown:
		var c Markdown
		_ = json.Unmarshal(raw, &c)
		return c, nil
	case KindStream:
		var c Stream
		_ = json.Unmarshal(raw, &c)
		return c, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, head.Kind)
	}
}
