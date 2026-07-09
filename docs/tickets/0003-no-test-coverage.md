# No test coverage at all

**Severity:** medium (grows to high as renderers land)

## Problem

The module has zero `*_test.go` files. Even at the scaffold stage there is
real, cheaply-testable behavior whose regressions would be silent:

- `component.Unmarshal` discriminator dispatch — every registered `kind`
  returns the right concrete type; unknown kinds return `ErrUnknownKind`;
  malformed JSON returns an error.
- `render.For` covers **every** component type in the union. The package doc
  itself calls this "the contract worth preserving even at the stub stage",
  but nothing enforces it — adding a component kind and forgetting the
  `render.For` case compiles fine and fails at runtime.
- `a2tea.Render` end-to-end: document in, expected model type out.
- The `event` types' shapes, which the README declares "stable from day one".

## Suggested fix

- Table-driven `component/component_test.go` over all kinds + error cases.
- A completeness test that constructs one value of each Component and asserts
  `render.For` returns a non-nil model with no error — this turns "new kind
  MUST be added in both places" from a comment into a CI failure.
- A golden/round-trip test for `a2tea.Render` on `examples/hello/sample.json`.
- CI (GitHub Actions) running `go test ./... && go vet ./...` on push, so the
  contract holds as the real renderers land.
