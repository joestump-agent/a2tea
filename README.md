# a2tea

**A2UI → Bubble Tea bridge.**

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

This repository is early: the parse path and component catalog are the real
protocol, but the renderers are still visual stubs — a surface's tree is walked
and text renders literally, while interactive/media components draw a
`[a2tea: <kind>]` placeholder.

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
- `github.com/joestump-agent/a2tea/event` — outbound `tea.Msg` types a host can consume for interaction results (`ButtonClicked`, `InputSubmitted`, `ChoiceSelected`, `FormSubmitted`), each carrying `Source`. Not emitted yet.

A2UI message and component types come from `github.com/tmc/a2ui`.

## Composition

Renderers are built to be embedded as children of a larger TUI (crush), not to
own their own program. The `render.Model` contract extends `tea.Model` with
`SetSize(w, h)` and `Focus()`/`Blur()`/`Focused()`, and **no renderer ever
calls `tea.Quit`** — quitting is the host's decision. Use `a2tea.Standalone` to
run a single surface on its own for examples and manual testing.

## Roadmap

The renderers are visual stubs. What is **not** yet implemented:

- **Real per-component rendering** with `charm.land/lipgloss/v2`,
  `charm.land/bubbles/v2`, and `charm.land/glamour/v2` instead of the
  `[a2tea: <kind>]` placeholders.
- **Data model.** `DynamicString` bindings/function calls render as
  `{binding}` / `{fn}`; `updateDataModel` is not applied.
- **Surface lifecycle.** Only the latest `updateComponents` is drawn;
  `createSurface` theming/catalog, `deleteSurface`, multi-surface compositing,
  and `ChildList` templates are not handled.
- **Interaction round-trip.** A2UI `Action`/`ClientMessage` events are not yet
  emitted back to the agent; the `event` types are defined but unused.

## Versioning

This is pre-1.0 and the public API may change. Once the renderers and event
round-trip are real and exercised by crush, a `v0.1.0` tag will be cut.

## License

Apache 2.0 — see [LICENSE](LICENSE).
