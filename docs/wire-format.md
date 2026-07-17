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
- Real lipgloss rendering of one surface's component tree, core catalog first:
  `Text` with its variant styles (h1–h3 heading, h4–h5 subheading, caption),
  `Card` in a rounded border, `Column`/`Row`/`List` layout, `Divider`, and
  styled focusable `Button`s.
- Input components: `TextField` is editable — a focused field accepts typed
  edits that update its state and flow into `ActionEvent.Context` — while
  `CheckBox`, `ChoicePicker`, `Slider`, and `DateTimeInput` are read-only
  visuals that draw their current values.
- Compact placeholders for media (`Image`, `Icon`, `Video`, `AudioPlayer`);
  `Tabs` render their title bar plus the first tab's content; `Modal` renders
  only its trigger.
- The first wired event: when the host focuses a surface, `Tab` / `Shift+Tab`
  cycle its focusables (buttons and text fields) and `Enter` activates the
  focused button. Activation emits
  `event.ButtonClicked` (carrying the resolved `*a2ui.EventAction`) and, when
  the button has a server-side `Action.Event`, a protocol-native
  `a2ui.ClientMessage` whose `ActionEvent` carries `Name`, `SurfaceID`,
  `SourceComponentID`, and `Context`. `FunctionCall`-only buttons emit no
  `ClientMessage`.
  The `ActionEvent.Timestamp` is left empty for the host to stamp.
- Deliberately monochrome chrome (borders, bold, faint, reverse-video focus)
  so the host theme wins.

Also implemented since earlier revisions of this doc:
- The message lifecycle is composited in order: `updateComponents` merges
  components by ID (siblings survive), `updateDataModel` sets bound values so
  `DynamicString` bindings resolve (unresolved bindings/function calls still
  render as `{binding}` / `{fn}` placeholders), and `deleteSurface` clears the
  surface.
- `ActionEvent.Context` is populated from the surface's input component
  values, so typed `TextField` edits round-trip to the agent.

**Not yet** (tracked as follow-ups)
- `ChildList` templates: children resolve from explicit ID lists only; the
  dynamic template form is not expanded.
- `createSurface` theming/catalog: the message is ignored — a surface is
  established by its first `updateComponents`, and theme/catalog payloads are
  not applied.
- Tab switching: tabs are not focusable, so the first tab is always active.
- Modal content: a modal renders only its trigger; its content stays hidden.
- Editing beyond `TextField`: `CheckBox`, `ChoicePicker`, `Slider`, and
  `DateTimeInput` remain read-only visuals.
- The remaining host-facing event types: `InputSubmitted`/`ChoiceSelected`
  are defined but never dispatched.
