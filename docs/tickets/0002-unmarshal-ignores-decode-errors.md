# component.Unmarshal discards field-decode errors and its doc comment is wrong

**Severity:** high · **Blocks crush integration:** yes

## Problem

Every branch of the `kind` switch in `component.Unmarshal` discards the decode
error:

```go
case KindCard:
    var c Card
    // TODO(a2tea): actually decode the rest of the document into c.
    _ = json.Unmarshal(raw, &c)
    return c, nil
```

Two consequences, both verified empirically:

1. **Malformed documents silently degrade to zero values.** A document with
   wrong field types produces an empty component and no error:

   ```go
   c, err := component.Unmarshal(json.RawMessage(`{"kind":"card","title":12345,"buttons":"nope"}`))
   // c = component.Card{Title:"", Buttons:nil}, err = <nil>
   ```

   An agent bug that emits a number where a string belongs renders as a blank
   card instead of surfacing an error anywhere.

2. **The doc comment is factually wrong.** It claims the function "returns a
   zero-valued struct of the right kind", but `json.Unmarshal(raw, &c)` *does*
   decode all tagged fields — `{"kind":"card","title":"Hi"}` returns
   `Card{Title:"Hi"}`. The TODO "actually decode the rest of the document" is
   already done by accident. Anyone extending this code from the comments will
   have a wrong mental model.

## Also missing: validation

No required-field or range validation exists anywhere:

- `Button.ID`, `Input.ID`, `Choice.ID` are documented as required (no
  `omitempty`) but empty values pass through.
- `Progress.Percent` is documented as `[0.0, 1.0]` but nothing rejects `42.0`.
- `Markdown.Source` is required by its JSON tag but an empty document passes.

## Suggested fix

- Check and return the `json.Unmarshal` error in every branch (wrap with the
  kind for context). Consider `json.Decoder` with `DisallowUnknownFields`
  as a strict mode, given the payload author is an LLM.
- Add a `Validate() error` per component (or one central validator) enforcing
  required IDs and the Percent range; call it from `Unmarshal`.
- Fix the doc comment to describe the actual behavior.
- Decide the `Unmarshal(nil)` contract: it currently returns `(nil, nil)`,
  which forces every caller to nil-check the component as well as the error.
  An explicit `ErrEmptyDocument` would be harder to misuse.
