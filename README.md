# a2tea

**A2UI ↔ Bubble Tea renderer bridge.**

`a2tea` maps [A2UI](https://a2ui.org)-inspired JSON messages onto
[Bubble Tea](https://github.com/charmbracelet/bubbletea) models so AI agents
can drive rich terminal UI from a structured wire format instead of dumping
raw text. The eventual consumer is Joe's fork of
[`charmbracelet/crush`](https://github.com/charmbracelet/crush) — this
library is the bridge that lets an agent speak a structured UI format and have
crush draw the result.

> **Wire format is provisional.** The JSON shape implemented here is
> *A2UI-inspired*, not a verified transcription of the a2ui.org spec — no
> containers, `children`, surfaces, or data binding yet. See
> [`docs/wire-format.md`](docs/wire-format.md) for exactly how it diverges and
> the decision still owed before a plain "A2UI compatible" claim is accurate.

This repository is early. The public API surface is fixed and the input path
(decode + validate + dispatch) is real, but the renderers themselves are still
stubs that draw a `[a2tea: <kind>]` placeholder, and no renderer emits
interaction events yet. See the [Roadmap](#roadmap) for what is and is not yet
implemented.

## Usage

```go
package main

import (
    "encoding/json"
    "log"
    "os"

    tea "charm.land/bubbletea/v2"

    "github.com/joestump-agent/a2tea"
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

`Render` returns an *embeddable* child component (see
[Composition](#composition)), so the model it returns does not handle quit or
own the terminal. To run one directly — as the snippet above does — wrap it
with `a2tea.Standalone`, which quits on `q` / `Esc` / `Ctrl+C` and forwards the
terminal size. A runnable version of this flow lives in
[`examples/hello`](examples/hello/). Today it prints a `[a2tea: card]`
placeholder; once the renderers land it will draw a real card.

`Render` also surfaces every failure as a non-nil error, so a host can tell a
bad document from an unimplemented renderer and fall back to plain text:
`component.ErrEmptyDocument`, `component.ErrUnknownKind`, and validation errors
(wrapping `component.ErrValidation`) are all matchable with `errors.Is`.

## Packages

- `github.com/joestump-agent/a2tea` — public entry point. `Render(raw) (tea.Model, error)` plus `Standalone` for running one renderer as its own program.
- `github.com/joestump-agent/a2tea/component` — typed union of components with a validating JSON unmarshaler (`Unmarshal`, `ErrEmptyDocument`, `ErrUnknownKind`, `ErrValidation`).
- `github.com/joestump-agent/a2tea/render` — one renderer per component kind, the embeddable `Model` contract, and a `For(c)` dispatcher.
- `github.com/joestump-agent/a2tea/event` — outbound `tea.Msg` types: `ButtonClicked`, `InputSubmitted`, `ChoiceSelected`, `FormSubmitted`, each carrying `Source`.

## Composition

Renderers are built to be embedded as children of a larger TUI (crush), not to
own their own program. The `render.Model` contract extends `tea.Model` with
`SetSize(w, h)` and `Focus()`/`Blur()`/`Focused()`, and **no renderer ever
calls `tea.Quit`** — quitting is the host's decision. Interaction results are
reported as `event` messages rather than by side effect. Use `a2tea.Standalone`
to run a single renderer on its own for examples and manual testing.

## Roadmap

What **is** done: `component.Unmarshal` decodes and validates every field,
returns real errors (no more silent zero-value degradation), and decodes
heterogeneous form fields by their own `kind`; `a2tea.Render` propagates those
errors; the `event` types carry `Source` context and include `FormSubmitted`;
and every renderer satisfies the embeddable `render.Model` contract.

What is **not** yet implemented (and is currently marked with
`// TODO(a2tea):` in source):

- **Container kinds & spec conformance.** No `children`/container support and
  no verification against a pinned a2ui.org schema — see
  [`docs/wire-format.md`](docs/wire-format.md).
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
- **Event roundtrip.** The types in `event/` are defined (and now carry
  `Source` context) but no renderer emits them yet. Once they do, agents will
  consume them from the standard `tea.Msg` channel inside their own `Update`.
- **Live streaming.** `StreamModel` only accepts a static `Chunks` slice;
  there's no channel-based live append yet.

## Versioning

This is pre-1.0 and the public API may change. Once the renderers and
event roundtrip are real and exercised by a downstream consumer (crush),
a `v0.1.0` tag will be cut.

## License

Apache 2.0 — see [LICENSE](LICENSE).
