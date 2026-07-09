# Review tickets

Findings from a full review of the a2tea scaffold (2026-07-09), filed as
files because GitHub Issues is disabled on this repository. Once Issues is
enabled these should be transferred there and this directory removed.

The review covered every source file, an empirical probe of the
`Unmarshal`/`Render` error paths, `go build` / `go vet` / `gofmt` (all
clean), and a dependency comparison against crush (the intended consumer).

| # | Title | Severity |
|---|-------|----------|
| [0001](0001-render-swallows-errors.md) | `Render()` never returns an error | high |
| [0002](0002-unmarshal-ignores-decode-errors.md) | `Unmarshal` discards decode errors; doc comment wrong; no validation | high |
| [0003](0003-no-test-coverage.md) | No test coverage at all | medium |
| [0004](0004-renderers-assume-root-model.md) | Renderers assume root-model usage; no composition contract for crush | high |
| [0005](0005-a2ui-spec-conformance.md) | Component schema unverified against the real A2UI spec | high |
| [0006](0006-form-fields-homogeneous.md) | `Form.Fields []Input` can't represent heterogeneous forms | medium |
| [0007](0007-event-model-gaps.md) | Event model gaps: no `FormSubmitted`, no source context | medium |
| [0008](0008-chores-deps-and-example.md) | Chores: dep versions behind crush; example path bug; deps.go | low |

Suggested order of attack for the crush integration: 0005 (pin the wire
format) and 0004 (pin the composition contract) first, since every renderer
depends on both; 0001/0002 alongside, since they change the public error
contract; 0003 continuously as the above land.
