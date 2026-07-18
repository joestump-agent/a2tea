---
id: quickstart
title: Quickstart
sidebar_label: Quickstart
description: Install a2tea, wire the Scan → Render loop into your host, and run the bundled example.
---

# Quickstart

Install the module, wire the Scan → Render loop into your host, and run the
bundled example.

## Install

```bash
go get github.com/joestump-agent/a2tea
```

## Wire up a host

Scan the reply into parts, render each part's prose, and hand its messages to
`Render`. `Render` returns an embeddable model — it never owns the terminal.

```go
parts, err := a2tea.Scan(reply)
if err != nil {
    // not valid A2UI — render reply as plain text
}
for _, p := range parts {
    if p.Text != "" {
        renderProse(p.Text)
    }
    if len(p.Messages) > 0 {
        model, err := a2tea.Render(p.Messages)
        if err != nil {
            continue
        }
        draw(model)
    }
}
```

:::tip
Use `a2tea.Contains(reply)` as a cheap gate before taking the `Scan` path at all.
:::

## Run the example

A runnable version of the whole flow lives in
[`examples/hello`](https://github.com/joestump-agent/a2tea/tree/main/examples/hello).
It wraps a single surface with `Standalone` so it runs on its own.

```bash
go run ./examples/hello
```

```text
╭──────────────────────────╮
│ Hello from A2UI          │
│ rendered by a2tea        │
│ ──────────────────────── │
│ [ OK ]                   │
╰──────────────────────────╯
tab cycle • enter select • q quit
```
