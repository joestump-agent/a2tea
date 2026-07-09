# Renderers assume they are the root tea.Model — no composition contract for embedding in crush

**Severity:** high (design) · **Blocks crush integration:** yes

## Problem

Every renderer (and `notImplementedModel`) is written as if it owns the whole
program:

- `Update` returns `tea.Quit` on **any** key press (`quitOnKey`). Embedded in
  crush's TUI, a stray keystroke inside a rendered card would terminate the
  entire application. Crush composes child components that never call
  `tea.Quit`; only the root model decides to exit.
- No `tea.WindowSizeMsg` handling and no width/height plumbing, so a renderer
  cannot lay itself out inside a parent-allocated region (crush allocates
  regions, it does not hand children the full terminal).
- No focus/blur concept. Crush needs to route key events to at most one
  focused component; a Card with focusable buttons next to an Input needs
  focus state and a way for the parent to grant/revoke it.
- `render.For` returns `tea.Model` (root-level interface with `View() tea.View`),
  while embedded components in the crush codebase render to strings/layers
  sized by the parent.

This is the single biggest integration risk: if the real renderers are built
against the "standalone program" assumption, they will all need rework to be
embeddable.

## Suggested fix

Define the composition contract *before* implementing the real renderers:

- Renderers implement a child-component interface (e.g. `Init/Update/View`
  plus `SetSize(w, h int)` and `Focus()/Blur()`), with a thin adapter to run
  one standalone for the examples.
- Remove `tea.Quit` from renderer `Update` paths entirely; quitting is the
  host's decision.
- Document which `tea.Msg`s a renderer consumes vs. passes through, and that
  interaction results are emitted as the `event` package types rather than by
  side effect.
