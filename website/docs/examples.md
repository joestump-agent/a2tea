---
id: examples
title: Examples
sidebar_label: Examples
description: The same surface, three ways — the host loop, a standalone runner, and the A2UI that feeds them.
---

# Examples

The same surface, three ways: the host loop, a standalone runner, and the A2UI
that feeds them.

## Code

### Host loop

```go
if !a2tea.Contains(reply) {
    renderProse(reply)
    return
}
parts, _ := a2tea.Scan(reply)
for _, p := range parts {
    if model, err := a2tea.Render(p.Messages); err == nil {
        host.Mount(model) // embed the surface
    }
}
```

### Standalone

```go
parts, _ := a2tea.Scan(reply)
model, err := a2tea.Render(parts[0].Messages)
if err != nil {
    log.Fatal(err)
}
// quits on q/esc, forwards terminal size
p := tea.NewProgram(a2tea.Standalone(model))
if _, err := p.Run(); err != nil {
    log.Fatal(err)
}
```

### A2UI input

```json
<a2ui-json>
{
  "version": "v0.9",
  "updateComponents": {
    "surfaceId": "hello",
    "components": [
      { "component": "Card", "id": "root", "child": "col" },
      { "component": "Column", "id": "col",
        "children": ["t", "d", "ok"] },
      { "component": "Text", "id": "t", "text": "Hello from A2UI" },
      { "component": "Divider", "id": "d" },
      { "component": "Button", "id": "ok", "child": "ok-label" },
      { "component": "Text", "id": "ok-label", "text": "OK" }
    ]
  }
}
</a2ui-json>
```

## Rendered output

```text
╭──────────────────────────╮
│ Hello from A2UI          │
│ ──────────────────────── │
│ [ OK ]                   │
╰──────────────────────────╯
tab cycle • enter select • q quit
```

A runnable version lives in
[`examples/hello`](https://github.com/joestump-agent/a2tea/tree/main/examples/hello).
