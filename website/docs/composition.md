---
id: composition
title: Composition
sidebar_label: Composition
description: Renderers are built to be embedded as children of a larger TUI, not to own their own program.
---

# Composition

Renderers are built to be embedded as children of a larger TUI, not to own their
own program.

## The render.Model contract

`render.Model` extends `tea.Model` with sizing and focus, and no renderer ever
calls `tea.Quit` — quitting is the host's decision.

```go
type Model interface {
    tea.Model
    SetSize(width, height int)
    Focus() tea.Cmd
    Blur() tea.Cmd
    Focused() bool
}
```

## Standalone

`Standalone` wraps a renderer so it can run as its own `tea.Program`. It owns the
two responsibilities a renderer deliberately does not: it quits on `Ctrl+C` /
`Esc` (and on `q`, unless a text field is being edited) and forwards
terminal-size changes via `SetSize`. Hosts that embed a renderer do not use it.

:::note q vs. typing.
When the focused child reports it is editing a `TextField`, `Standalone`
forwards `q` as input instead of quitting — via the optional `EditingText()`
probe.
:::
