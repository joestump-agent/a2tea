# a2tea

**A2UI ↔ Bubble Tea renderer bridge.**

`a2tea` maps [A2UI](https://a2ui.org) JSON messages onto
[Bubble Tea](https://github.com/charmbracelet/bubbletea) models so AI agents
can drive rich terminal UI from a structured wire format instead of dumping
raw text. The eventual consumer is Joe's fork of
[`charmbracelet/crush`](https://github.com/charmbracelet/crush) — this
library is the bridge that lets an agent speak A2UI and have crush draw the
result.

This repository is currently **scaffolding**. The public API surface is
fixed, but the rendering and event roundtrip are stubbed. See the
[Roadmap](#roadmap) for what is and is not yet implemented.

## Usage

```go
package main

import (
    "encoding/json"
    "log"
    "os"

    tea "charm.land/bubbletea/v2"

    "github.com/joestump/a2tea"
)

func main() {
    raw, err := os.ReadFile("sample.json")
    if err != nil {
        log.Fatal(err)
    }

    model, err := a2tea.Render(json.RawMessage(raw))
    if err != nil {
        log.Fatal(err)
    }

    if _, err := tea.NewProgram(model).Run(); err != nil {
        log.Fatal(err)
    }
}
```

A runnable version of the same flow lives in
[`examples/hello`](examples/hello/). Today it prints a `[a2tea: card]`
placeholder; once the renderers land it will draw a real card.

## Packages

- `github.com/joestump/a2tea` — public entry point. `Render(raw) (tea.Model, error)`.
- `github.com/joestump/a2tea/component` — typed union of A2UI components and the JSON unmarshaler.
- `github.com/joestump/a2tea/render` — one `tea.Model` per component kind, plus a `For(c)` dispatcher.
- `github.com/joestump/a2tea/event` — outbound `tea.Msg` types: `ButtonClicked`, `InputSubmitted`, `ChoiceSelected`.

## Roadmap

What is **not** yet implemented (and is currently marked with
`// TODO(a2tea):` in source):

- **JSON unmarshaling.** `component.Unmarshal` only reads the `kind`
  discriminator. Field-level decoding, schema validation, and nested
  `children` handling are stubs.
- **Real renderers.** Every `render/*Model` returns a `[a2tea: <kind>]`
  placeholder. The real implementations should use `charm.land/bubbles/v2`
  (textinput, list, progress), `charm.land/glamour/v2` (markdown), and
  `charm.land/lipgloss/v2` (layout).
- **Form support.** `FormModel` should wrap `huh.Form` so field
  navigation, validation, and submission work without bespoke code. The
  `huh` dependency is intentionally **not** in `go.mod` yet — the
  published `charmbracelet/huh v1.0.0` pins older
  `charm.land/x/ansi` internals that conflict with the v2 stack this
  module shares with crush. Pull it back in once a huh release compatible
  with the v2 ecosystem ships.
- **Event roundtrip.** The types in `event/` are defined but no renderer
  emits them yet. Once they do, agents will consume them from the standard
  `tea.Msg` channel inside their own `Update`.
- **Live streaming.** `StreamModel` only accepts a static `Chunks` slice;
  there's no channel-based live append yet.

## Versioning

This is pre-1.0 and the public API may change. Once the renderers and
event roundtrip are real and exercised by a downstream consumer (crush),
a `v0.1.0` tag will be cut.

## License

Apache 2.0 — see [LICENSE](LICENSE).
