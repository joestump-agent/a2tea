# a2tea wire format

**a2tea targets the real A2UI protocol (v0.9) via
[`github.com/tmc/a2ui`](https://pkg.go.dev/github.com/tmc/a2ui).**

Earlier revisions of a2tea used a provisional, invented flat-object shape with a
`kind` discriminator. That has been removed. a2tea no longer defines its own
component types — it decodes and renders the actual A2UI message and component
catalog from `github.com/tmc/a2ui`, and it parses A2UI out of LLM output with
[`github.com/tmc/a2ui/a2uistream`](https://pkg.go.dev/github.com/tmc/a2ui/a2uistream).

This closes the conformance question that the old `docs/wire-format.md` tracked
(a2tea issue #5): the wire format is A2UI, not an a2tea invention.

## What arrives, and how a2tea handles it

A2UI is message-oriented. An agent emits a stream of `ServerMessage`s —
`createSurface`, `updateComponents`, `updateDataModel`, `deleteSurface` —
usually interleaved with conversational text and wrapped in `<a2ui-json>` tags.

- **Parsing.** `a2tea.Contains` / `a2tea.Scan` wrap `a2uistream` to split a
  reply into ordered `Part`s of text and typed `[]a2ui.ServerMessage`. Hosts
  call this instead of hand-rolling detection.
- **Rendering.** `a2tea.Render(msgs)` applies the messages to build surface
  state and returns an embeddable Bubble Tea model. Components live in a flat
  set that references children by ID (adjacency list); the renderer resolves the
  tree from its root (the component nothing else references as a child) and
  walks it.

## Implemented vs. not yet

**Implemented**
- Scan/extract A2UI messages from LLM text (`<a2ui-json>` tags or bare JSON).
- Render one surface's component tree: `Text` renders its literal, containers
  (`Card`, `Column`, `Row`, `List`) recurse, `Button` resolves its label.

**Not yet** (tracked as follow-ups; the renderers remain visual stubs)
- Real per-component rendering with lipgloss/bubbles/glamour instead of the
  `[a2tea: <kind>]` placeholders for interactive and media components.
- The data model: `DynamicString` bindings/function calls render as
  `{binding}` / `{fn}` placeholders; `updateDataModel` is not applied.
- Surface lifecycle across messages: only the latest `updateComponents` is
  drawn; `createSurface` theming/catalog, `deleteSurface`, multi-surface
  compositing, and `ChildList` templates are not handled.
- The interaction round-trip: A2UI `Action`/`ClientMessage` events are not
  emitted back to the agent yet.
