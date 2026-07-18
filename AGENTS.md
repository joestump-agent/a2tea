# AGENTS.md

## What This Is

a2tea is a Go library that bridges the A2UI protocol (v0.9) into Bubble Tea. It parses A2UI JSON from LLM responses and renders the described surfaces as Bubble Tea models. The primary consumer is Joe's fork of crush, where a2tea detects A2UI in model output and draws it instead of dumping raw JSON.

## Commands

```sh
go build ./...          # build all packages
go test -race ./...     # run all tests with race detector (CI runs this)
go vet ./...            # vet
gofmt -l .              # check formatting (CI fails if any file is unformatted)
gofumpt -w .            # format before committing (per user preferences)
go run ./examples/hello # run the standalone example (quits on q/Esc/Ctrl+C)
```

CI (`.github/workflows/ci.yml`) runs gofmt check, go vet, go build, and `go test -race ./...`. All must pass before merge.

## Architecture

Three packages, strict separation of concerns:

- **Root package (`a2tea`)** — Public entry point. `Contains(reply)` is a cheap pre-check. `Scan(reply)` splits an LLM response into ordered `Part` structs (text + A2UI messages). `Render(msgs)` applies `ServerMessage`s and returns an embeddable `tea.Model`. `Standalone(child)` wraps a renderer to run as its own program for examples/testing only.
- **`render`** — Walks an A2UI surface's component tree and draws it with lipgloss. `Surface` is the core type, holding a flat `map[string]Component` (by ID) and resolving the tree from the root.
- **`event`** — Outbound `tea.Msg` types emitted on user interaction. Every event embeds `Source{ComponentID, SurfaceID}`. Only `ButtonClicked` is actually emitted today; the rest are provisional.

A2UI message/component types come from the external `github.com/tmc/a2ui` package. a2tea does **not** define its own component types.

### Component Model

A2UI surfaces are adjacency lists: a flat slice of `Component` structs, each referencing children by string ID. The root is the component no other component references as a child (falls back to first in declaration order). Components can be legally shared across multiple parents — this is not a cycle. Genuine ancestor cycles are caught during tree walks.

The `renderComponent` dispatch is a type switch on the component's concrete field (`c.Text`, `c.Card`, `c.Button`, etc.). `childIDs` extracts referenced child IDs per container type. `KindOf` returns the string kind name.

### Composition Contract (Critical)

Renderers are designed to be **embedded children** of a larger host TUI, never the root of their own program:

- **Never call `tea.Quit`** — quitting is the host's decision.
- Layout via `SetSize(w, h)` — the host allocates a region.
- Focus routing via `Focus()`/`Blur()`/`Focused()` — the host routes keys to at most one focused child.
- `render.Model` interface extends `tea.Model` with `SetSize`, `Focus`, `Blur`, `Focused`.
- `base` struct (embedded by value) provides the default `SetSize`/`Focus`/`Blur`/`Focused` implementation.
- `Standalone` exists only for examples and manual testing — it wraps a renderer with quit handling and size forwarding.

### Interaction

Only buttons are interactive. When the surface holds focus, `Tab`/`Shift+Tab` cycle through the focus ring (depth-first button collection, deduped), and `Enter` emits `event.ButtonClicked`. The focus ring is built once in `NewSurface` via `collectFocusables`.

### Styling

Deliberately monochrome: borders, bold, faint, and reverse-video for focus. **No hardcoded colors** — the host's theme must win. Styles live in `render/styles.go` as package-level `lipgloss.Style` vars.

### Width Budget

`withWidth` temporarily narrows `s.width` for a subtree render, restoring it after. This lets containers (Card's border+padding, List's bullet indent) narrow their children's wrapping budget. Rendering is single-goroutine depth-first, so the save/restore is safe.

## Conventions

- **Imports**: Charm ecosystem uses `charm.land/` prefixed v2 modules (`charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2`, `charm.land/glamour/v2`). The A2UI protocol library is imported as `a2ui "github.com/tmc/a2ui"` and Bubble Tea as `tea "charm.land/bubbletea/v2"`.
- **Tests** live alongside code (`_test.go` files in the same package directory). Tests use external test packages (`package render_test`, `package a2tea_test`). Test helpers like `text(id, s)` and `renderPlain(comps)` build minimal component trees and strip ANSI for structure assertions.
- **Test data**: JSON fixtures in `testdata/` are unmarshaled into `a2ui.ServerMessage` for integration-style tests (see `a2tea_button_test.go`).
- **Comments**: Doc comments on all exported symbols. Multi-paragraph doc comments explain design decisions and trade-offs, not just what the code does. Comments frequently reference issue numbers and TODOs with the `TODO(a2tea):` prefix.
- **`render/deps.go`**: Contains blank imports pinning rendering dependencies. Each blank import should be removed when its renderer starts using the package directly.

## Gotchas

- **Button label fallback**: When a Button's `Child` is empty (the model put "text" directly on the button, which the A2UI schema ignores), the component ID is used as the label fallback. This is pinned by `TestButtonRendersWithoutChild`.
- **`Button.Action` is not yet wired**: The `enter` handler emits `ButtonClicked` with only the ID/Source — the Action's `Event.Name` and `Context` are discarded. Tracked in issue #19 / epic #18.
- **DynamicString placeholders**: `DynamicString` bindings render as `{binding}` and function calls as `{fn}`. The data model (`updateDataModel`) is not applied.
- **Input components are read-only**: `TextField`, `CheckBox`, `ChoicePicker`, `Slider`, `DateTimeInput` draw current values but never mutate state or emit input events.
- **Tab switching and modals**: First tab is always active. Modal renders only its trigger; content stays hidden.
- **`huh` dependency blocked**: Cannot add `charmbracelet/huh` for editable fields because v1.0.0 conflicts with the v2 charm.land stack (noted in `render/deps.go`).
- **Go version**: `go.mod` specifies `go 1.26.3`.
