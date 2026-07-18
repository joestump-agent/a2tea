# a2tea

**A2UI → Bubble Tea bridge.**

📖 **Docs:** https://joestump-agent.github.io/a2tea/ — the full documentation
site (source in [`website/`](website/)).

`a2tea` lets an AI agent drive a terminal UI with [A2UI](https://a2ui.org): it
parses the A2UI messages an agent emits — interleaved with conversational text
in an LLM response — and renders the described surfaces as
[Bubble Tea](https://github.com/charmbracelet/bubbletea) models. The consumer is
Joe's fork of [`charmbracelet/crush`](https://github.com/charmbracelet/crush):
a2tea is the bridge that lets crush recognize A2UI in a model's reply and draw
it instead of dumping raw JSON.

a2tea targets the **real A2UI protocol (v0.9)** via
[`github.com/tmc/a2ui`](https://pkg.go.dev/github.com/tmc/a2ui) — it does not
define its own component types. It parses A2UI out of model output with
[`a2uistream`](https://pkg.go.dev/github.com/tmc/a2ui/a2uistream). See
[`docs/wire-format.md`](docs/wire-format.md).

This repository is early, but rendering is now real for the core catalog:
`Text` draws with its variant styles (headings, subheadings, captions), `Card`
gets a rounded border, `Column`/`Row`/`List` lay out their children, `Divider`
rules, and `Button`s render as styled, focusable chrome. The input components
(`TextField`, `CheckBox`, `ChoicePicker`, `Slider`, `DateTimeInput`) are
read-only visuals of their current values; media components (`Image`, `Icon`,
`Video`, `AudioPlayer`) draw compact one-line placeholders; `Tabs` render their
title bar plus the first tab's content, and `Modal` opens on `Enter` (its
content renders as a bordered in-flow block) and closes on `Esc`.

Buttons are the first wired interaction: when the host gives a surface focus,
`Tab` / `Shift+Tab` cycle its buttons and `Enter` emits `event.ButtonClicked`
with `Source` context. Component chrome is deliberately monochrome — borders,
bold, faint, and reverse-video focus — so the host's theme wins.

## Usage

A host feeds an assistant reply to `Scan`, renders each part's text as prose,
and hands each part's A2UI messages to `Render`:

```go
parts, err := a2tea.Scan(reply)
if err != nil {
    // not valid A2UI — render `reply` as plain text
}
for _, p := range parts {
    if p.Text != "" {
        renderProse(p.Text)
    }
    if len(p.Messages) > 0 {
        model, err := a2tea.Render(p.Messages)
        if err != nil {
            continue // no renderable surface in these messages
        }
        draw(model) // an embeddable tea.Model
    }
}
```

`a2tea.Contains(reply)` is a cheap check for whether a reply has any A2UI at all.

`Render` returns an *embeddable* child component (see [Composition](#composition)),
so it does not handle quit or own the terminal. To run one directly, wrap it
with `a2tea.Standalone`, which quits on `q` / `Esc` / `Ctrl+C` and forwards the
terminal size. A runnable version of the whole flow lives in
[`examples/hello`](examples/hello/).

## Packages

- `github.com/joestump-agent/a2tea` — public entry point: `Contains`, `Scan(reply) ([]Part, error)`, `Render(msgs) (tea.Model, error)`, and `Standalone`.
- `github.com/joestump-agent/a2tea/render` — walks an A2UI surface (components referencing children by ID) into an embeddable `render.Model`.
- `github.com/joestump-agent/a2tea/event` — outbound `tea.Msg` types a host can consume for interaction results (`ButtonClicked`, `InputSubmitted`, `ChoiceSelected`), each carrying `Source`. `ButtonClicked` is emitted when a focused button is activated; `InputSubmitted` when Enter confirms a focused `TextField`/`DateTimeInput` value; `ChoiceSelected` (carrying the full selection as `Values []string`) when a focused `ChoicePicker`'s selection changes. `FormSubmitted` is deprecated and never emitted.

A2UI message and component types come from `github.com/tmc/a2ui`.

## Composition

Renderers are built to be embedded as children of a larger TUI (crush), not to
own their own program. The `render.Model` contract extends `tea.Model` with
`SetSize(w, h)` and `Focus()`/`Blur()`/`Focused()`, and **no renderer ever
calls `tea.Quit`** — quitting is the host's decision. Use `a2tea.Standalone` to
run a single surface on its own for examples and manual testing.

## Roadmap

The core catalog renders for real with `charm.land/lipgloss/v2` (see above).
The message lifecycle is applied in order — `updateComponents` composites by
component ID, `updateDataModel` resolves `DynamicString` bindings,
`deleteSurface` clears the surface — `TextField` is editable (typed edits
update state and flow into `ActionEvent.Context`), and button activation sends
a protocol-native `a2ui.ClientMessage` back to the agent with `Name`,
`SurfaceID`, `SourceComponentID`, and a populated `Context`.

What is **not** yet implemented:

- **`ChildList` templates.** Children resolve from explicit ID lists only;
  the dynamic template form is not expanded.
- **`createSurface` theming/catalog.** The message is ignored — a surface is
  established by its first `updateComponents`, and theme/catalog payloads are
  not applied.
- **Tab switching.** Tabs are not focusable — the first tab is always the
  active one.
- **Editing beyond `TextField`.** `CheckBox`/`ChoicePicker`/`Slider`/
  `DateTimeInput` remain read-only visuals.

## Versioning

This is pre-1.0 and the public API may change. Once the event round-trip is
complete and exercised by crush, a `v0.1.0` tag will be cut.

## License

Apache 2.0 — see [LICENSE](LICENSE).
