# Render() never returns an error — parse failures and unknown kinds are silently swallowed

**Severity:** high (API contract) · **Blocks crush integration:** yes

## Problem

`a2tea.Render` has the signature `(tea.Model, error)`, but it never returns a
non-nil error. Every failure mode is converted into a `notImplementedModel`
placeholder with `err == nil`:

- malformed JSON → `notImplementedModel{reason: "parse error: ..."}`, `nil` error
- unknown `kind` → `notImplementedModel{reason: "no renderer for ..."}`, `nil` error
- empty document → `notImplementedModel{reason: "empty document"}`, `nil` error

Verified empirically:

```go
m, err := a2tea.Render(json.RawMessage(`{{{not json`))
// m = a2tea.notImplementedModel, err = <nil>
m, err = a2tea.Render(json.RawMessage(`{"kind":"table"}`))
// m = a2tea.notImplementedModel, err = <nil>
```

There is a `TODO(a2tea)` acknowledging this, but it matters enough for the
crush integration to track: a host application cannot distinguish "the agent
sent a bad document" from "renderer not implemented yet" programmatically.
Crush will need to fall back to plain-text rendering when an A2UI document is
invalid, and it can only do that off a real error value.

## Suggested fix

- Return the `component.Unmarshal` error to the caller (`return nil, err`).
- Return `render.For`'s error for unknown kinds (or a wrapped
  `component.ErrUnknownKind` so callers can `errors.Is` it).
- Give "empty document" an explicit sentinel (`ErrEmptyDocument`) instead of a
  placeholder model.
- Keep the placeholder model only for the genuinely-not-yet-implemented
  rendering path, if at all.

This is an API-behavior change, so it should land before crush starts
consuming the library — the README calls the API surface "fixed", and the
error contract is part of that surface.
