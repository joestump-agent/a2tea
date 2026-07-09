// Package component defines the typed union of A2UI components that a2tea
// understands. Each concrete type corresponds to one component "kind" in the
// A2UI JSON schema (see https://a2ui.org).
//
// The wire format implemented here is provisional and A2UI-inspired rather
// than a verified transcription of the a2ui.org specification — see
// docs/wire-format.md for the exact status and what is still open.
//
// The interface is deliberately small: a Component knows its Kind so that
// dispatch tables in sibling packages (render, event) can pick the right
// implementation without type switches sprinkled across the codebase, and it
// can Validate itself so malformed agent output is rejected at the edge
// instead of silently rendering as a blank component.
package component

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// Component is the closed union of all A2UI components a2tea can render.
//
// New component kinds MUST be added here AND in the Unmarshal switch below
// AND in render.For, otherwise they will be silently rejected (Unmarshal) or
// fail at runtime (render). The completeness test in render enforces the
// render.For half of that contract.
type Component interface {
	// Kind returns the A2UI component kind string ("card", "form", ...).
	// It is the dispatch key used by render.For and event routing.
	Kind() string
	// Validate reports whether the component's fields satisfy the schema's
	// required-field and range constraints. Unmarshal calls it so a decoded
	// component is always valid or the error is surfaced.
	Validate() error
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

// Sentinel errors returned by Unmarshal. Callers can match them with
// errors.Is to distinguish "the agent sent nothing", "the agent sent an
// unknown kind", and "the agent sent a structurally-valid but invalid
// document".
var (
	// ErrEmptyDocument is returned when the document is empty or only
	// whitespace. It is an explicit sentinel rather than a (nil, nil)
	// return so callers do not have to nil-check the component separately.
	ErrEmptyDocument = errors.New("a2tea/component: empty document")
	// ErrUnknownKind is returned when the "kind" discriminator does not
	// match any registered component type.
	ErrUnknownKind = errors.New("a2tea/component: unknown kind")
	// ErrValidation wraps every field-validation failure so callers can
	// errors.Is against it while still getting a descriptive message.
	ErrValidation = errors.New("a2tea/component: invalid component")
)

// invalid builds an ErrValidation-wrapped error with contextual detail.
func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrValidation}, args...)...)
}

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

// Validate implements Component. A card's buttons must each be valid.
func (c Card) Validate() error {
	for i, b := range c.Buttons {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("card button %d: %w", i, err)
		}
	}
	return nil
}

// Button is a clickable action embedded in a Card or Form.
type Button struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Validate reports whether the button has the required id and label. Button
// is not itself a Component (it never renders standalone), so this is a plain
// helper called by the components that embed buttons.
func (b Button) Validate() error {
	if b.ID == "" {
		return invalid("button id is required")
	}
	if b.Label == "" {
		return invalid("button %q label is required", b.ID)
	}
	return nil
}

// Form is an A2UI form: an ordered collection of input fields plus a submit
// action.
//
// Fields is a heterogeneous union: a form may mix single-line inputs and
// single-select choices (and, later, other input-like kinds). Each element is
// decoded from JSON by dispatching on its own "kind" discriminator — see
// UnmarshalJSON.
type Form struct {
	ID     string      `json:"id,omitempty"`
	Title  string      `json:"title,omitempty"`
	Fields []FormField `json:"fields,omitempty"`
	Submit Button      `json:"submit,omitempty"`
}

// Kind implements Component.
func (Form) Kind() string { return KindForm }

// Validate implements Component. Every field and the submit button (when
// present) must be valid.
func (f Form) Validate() error {
	for i, field := range f.Fields {
		if err := field.Validate(); err != nil {
			return fmt.Errorf("form field %d (%s): %w", i, field.Kind(), err)
		}
	}
	if f.Submit != (Button{}) {
		if err := f.Submit.Validate(); err != nil {
			return fmt.Errorf("form submit: %w", err)
		}
	}
	return nil
}

// UnmarshalJSON decodes a form, dispatching each entry of the "fields" array
// to the concrete input-like type named by its "kind". A field with no
// "kind" is treated as a single-line input, the common case an agent is most
// likely to emit tersely.
func (f *Form) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID     string            `json:"id"`
		Title  string            `json:"title"`
		Fields []json.RawMessage `json:"fields"`
		Submit Button            `json:"submit"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	f.ID = raw.ID
	f.Title = raw.Title
	f.Submit = raw.Submit
	f.Fields = nil
	for i, fr := range raw.Fields {
		field, err := decodeFormField(fr)
		if err != nil {
			return fmt.Errorf("form field %d: %w", i, err)
		}
		f.Fields = append(f.Fields, field)
	}
	return nil
}

// FormField is the subset of components that can appear inside a Form. It is
// a sealed union: only the input-like kinds in this package implement the
// unexported marker method, so a caller cannot smuggle an arbitrary
// Component (e.g. a nested Card) into a form's fields.
type FormField interface {
	Component
	formField()
}

// decodeFormField decodes a single form-field element by its "kind".
func decodeFormField(raw json.RawMessage) (FormField, error) {
	var head struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("decode field discriminator: %w", err)
	}
	switch head.Kind {
	case KindInput, "":
		var in Input
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, fmt.Errorf("decode input field: %w", err)
		}
		return in, nil
	case KindChoice:
		var ch Choice
		if err := json.Unmarshal(raw, &ch); err != nil {
			return nil, fmt.Errorf("decode choice field: %w", err)
		}
		return ch, nil
	default:
		return nil, fmt.Errorf("%w: form field %q", ErrUnknownKind, head.Kind)
	}
}

// Input is a single-line text input.
type Input struct {
	ID          string `json:"id"`
	Label       string `json:"label,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Value       string `json:"value,omitempty"`
}

// Kind implements Component.
func (Input) Kind() string { return KindInput }

// Validate implements Component.
func (i Input) Validate() error {
	if i.ID == "" {
		return invalid("input id is required")
	}
	return nil
}

func (Input) formField() {}

// Choice is a single-select picker over a list of options.
type Choice struct {
	ID      string         `json:"id"`
	Label   string         `json:"label,omitempty"`
	Options []ChoiceOption `json:"options,omitempty"`
}

// Kind implements Component.
func (Choice) Kind() string { return KindChoice }

// Validate implements Component.
func (c Choice) Validate() error {
	if c.ID == "" {
		return invalid("choice id is required")
	}
	for i, o := range c.Options {
		if o.Value == "" {
			return invalid("choice %q option %d value is required", c.ID, i)
		}
	}
	return nil
}

func (Choice) formField() {}

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

// Validate implements Component. A determinate progress bar's Percent must be
// in [0.0, 1.0].
func (p Progress) Validate() error {
	if !p.Indeterminate && (p.Percent < 0 || p.Percent > 1) {
		return invalid("progress percent %v is out of range [0.0, 1.0]", p.Percent)
	}
	return nil
}

// Markdown is a block of rendered markdown content.
type Markdown struct {
	ID     string `json:"id,omitempty"`
	Source string `json:"source"`
}

// Kind implements Component.
func (Markdown) Kind() string { return KindMarkdown }

// Validate implements Component.
func (m Markdown) Validate() error {
	if m.Source == "" {
		return invalid("markdown source is required")
	}
	return nil
}

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

// Validate implements Component. A stream has no required fields.
func (Stream) Validate() error { return nil }

// Unmarshal decodes a raw A2UI JSON document into a concrete, validated
// Component.
//
// It peeks at the "kind" discriminator, decodes the full document into the
// matching concrete type (returning any decode error, wrapped with the kind,
// rather than swallowing it), and then calls Validate on the result. An empty
// document returns ErrEmptyDocument; an unrecognized kind returns
// ErrUnknownKind. Both are matchable with errors.Is.
//
// Decoding is not strict about unknown JSON fields: an agent may include
// forward-compatible extras without failing. Type mismatches on known fields
// are still errors — a "title" of 12345 is rejected, not silently dropped.
func Unmarshal(raw json.RawMessage) (Component, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, ErrEmptyDocument
	}
	var head struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("a2tea/component: decode discriminator: %w", err)
	}
	c, err := decode(head.Kind, raw)
	if err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// decode dispatches on kind and fully decodes raw into the matching concrete
// type, wrapping any decode error with the kind for context.
func decode(kind string, raw json.RawMessage) (Component, error) {
	switch kind {
	case KindCard:
		return decodeInto[Card](kind, raw)
	case KindForm:
		return decodeInto[Form](kind, raw)
	case KindInput:
		return decodeInto[Input](kind, raw)
	case KindChoice:
		return decodeInto[Choice](kind, raw)
	case KindProgress:
		return decodeInto[Progress](kind, raw)
	case KindMarkdown:
		return decodeInto[Markdown](kind, raw)
	case KindStream:
		return decodeInto[Stream](kind, raw)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}
}

// decodeInto decodes raw into a fresh T and returns it as a Component,
// wrapping any decode error with the kind for context.
func decodeInto[T Component](kind string, raw json.RawMessage) (Component, error) {
	var c T
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("a2tea/component: decode %s: %w", kind, err)
	}
	return c, nil
}
