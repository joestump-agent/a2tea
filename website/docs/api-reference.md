---
id: api-reference
title: API reference
sidebar_label: API reference
description: Public entry points, types, packages, and events for github.com/joestump-agent/a2tea.
---

# API reference

`package github.com/joestump-agent/a2tea`

## Functions

| Signature | Description |
| --- | --- |
| `Contains(s string) bool` | Cheap probe: any A2UI present? |
| `Scan(s string) ([]Part, error)` | Split reply into text + messages |
| `Render(msgs []a2ui.ServerMessage, opts ...render.Option) (tea.Model, error)` | Build an embeddable surface model |
| `Standalone(child tea.Model) tea.Model` | Run one surface as its own program |

### Contains

`Contains(s string) bool`

Cheap two-stage probe for whether a reply contains any A2UI at all — a literal
key scan, then a real parse only on a hit. `Contains` and `Scan` always agree.

### Scan

`Scan(s string) ([]Part, error)`

Splits an LLM response into ordered parts of text and A2UI messages. The entry
point a host uses instead of hand-rolling detection.

### Render

`Render(msgs []a2ui.ServerMessage, opts ...render.Option) (tea.Model, error)`

Applies messages in order to build surface state and returns an embeddable
model. Returns `ErrNoRenderableSurface` when there is nothing to draw.

### Standalone

`Standalone(child tea.Model) tea.Model`

Wraps a renderer to run as its own program — owns quit and size forwarding. For
examples and manual testing of a single surface.

## Types

```go
type Part struct {
    Text     string
    Messages []a2ui.ServerMessage
}

var ErrNoRenderableSurface = errors.New("a2tea: no renderable surface in messages")
```

A `Part` is a segment of an LLM response: conversational text and the A2UI
messages that immediately followed it. Either field may be empty.

## Packages

| Package | Role |
| --- | --- |
| `a2tea` | Public entry: `Contains`, `Scan`, `Render`, `Standalone` |
| `a2tea/render` | Walks an A2UI surface into a `render.Model` |
| `a2tea/event` | Outbound `tea.Msg` types for interaction results |

A2UI message and component types come from `github.com/tmc/a2ui`.

## Events

Outbound `tea.Msg` types a host consumes for interaction results — each carrying
`Source` context.

| Event | Status | Emitted on |
| --- | --- | --- |
| `ButtonClicked` | ✓ emitted | A focused button is activated |
| `InputSubmitted` | ✓ emitted | Enter confirms a focused `TextField` / `DateTimeInput` value |
| `ChoiceSelected` | ✓ emitted | A focused `ChoicePicker`'s selection changes |
| `FormSubmitted` | ✗ deprecated | Never — see below |

`ButtonClicked` carries the button's resolved `*a2ui.EventAction` (nil for
buttons with no server-side event). Alongside it the renderer emits a native
`a2ui.ClientMessage` whose `ActionEvent.Context` is populated from the
surface's input component values (`TextField` / `DateTimeInput` → string,
`ChoicePicker` → `[]string`, `CheckBox` → bool, `Slider` → float64, keyed by
component ID) — that message is the round-trip contract a host consumes.

`InputSubmitted` carries the field's current value — the edited text when the
user has typed, else the literal / resolved-binding seed (an unresolved
`{binding}` placeholder submits `""`). `ChoiceSelected` carries the full
post-change selection as `Values []string` in option-declaration order, and is
emitted only when the selection actually changes — re-selecting an
already-selected single-select option emits nothing.

`FormSubmitted` is **deprecated** and never emitted: A2UI v0.9 has no Form
component, so a "form submit" is just a Button action whose
`ActionEvent.Context` carries the gathered field values.
