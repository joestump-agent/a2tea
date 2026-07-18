---
id: wire-format
title: Wire format
sidebar_label: Wire format
description: a2tea targets the real A2UI protocol (v0.9) via tmc/a2ui. It defines no component types of its own.
---

# Wire format

a2tea targets the real A2UI protocol (v0.9) via
[`github.com/tmc/a2ui`](https://pkg.go.dev/github.com/tmc/a2ui). It defines no
component types of its own.

:::warning Not an a2tea invention.
Earlier revisions used a provisional flat-object shape with a `kind`
discriminator. That has been removed — the wire format is A2UI.
:::

## What arrives

A2UI is message-oriented. An agent emits a stream of `ServerMessage`s —
`createSurface`, `updateComponents`, `updateDataModel`, `deleteSurface` —
usually interleaved with conversational text and wrapped in `<a2ui-json>` tags.

```json
<a2ui-json>
{
  "version": "v0.9",
  "updateComponents": {
    "surfaceId": "trip",
    "components": [
      { "component": "Card", "id": "root", "child": "col" },
      { "component": "Column", "id": "col",
        "children": ["title", "book"] },
      { "component": "Text", "id": "title",
        "text": "Kyoto · Autumn Weekend" },
      { "component": "Button", "id": "book", "child": "book-label" },
      { "component": "Text", "id": "book-label", "text": "Book it" }
    ]
  }
}
</a2ui-json>
```

Components live in a flat set that references children by ID (an adjacency
list); the renderer resolves the tree from its root. Note the shape rules the
catalog enforces: the type discriminator is `component`, a `Card` wraps exactly
one `child`, and a `Button` has **no label field** — its label is a child
`Text` component.

## Parsing & rendering

`Contains` / `Scan` wrap
[`a2uistream`](https://pkg.go.dev/github.com/tmc/a2ui/a2uistream) to split a
reply into ordered `Part`s of text and typed messages. `Render` applies the
messages to build surface state and returns an embeddable model — resolving the
component tree from the root (the component nothing else references as a child).

## Implemented vs. not yet

### ✓ Implemented

- Scan A2UI from LLM text (`<a2ui-json>` tags or bare JSON)
- `Text` variants, `Card`, `Column`/`Row`/`List`, `Divider`
- Focusable, styled `Button`s
- Editable inputs — all five: `TextField`, `CheckBox`, `ChoicePicker`,
  `Slider`, and `DateTimeInput`. Edited values round-trip to the agent in
  `ActionEvent.Context`
- `ChildList` templates: the dynamic form expands one template instance per
  element of the bound data-model list, with bindings inside each instance
  resolving against that element first
- `Modal`: `Enter` opens (content renders as a bordered in-flow block — the
  honest terminal equivalent of an overlay), `Esc` closes the innermost open
  modal
- `createSurface` as a documented no-op: a surface is established by its first
  `updateComponents`; theme hints and `catalogId` are deliberately ignored so
  the host theme wins and the compiled-in catalog stays authoritative
- Lifecycle: `updateComponents` / `updateDataModel` / `deleteSurface`
- Events: `event.ButtonClicked` + native `ClientMessage`, `InputSubmitted`
  (Enter on a `TextField` / `DateTimeInput`), and `ChoiceSelected` (a
  `ChoicePicker` selection change, carrying `Values []string`)

### ✗ Not yet

Each gap is tracked in the issue backlog:

- Tab switching (first tab is active) —
  [#45](https://github.com/joestump-agent/a2tea/issues/45)
