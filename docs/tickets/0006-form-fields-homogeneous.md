# Form.Fields is []Input — heterogeneous forms are unrepresentable

**Severity:** medium (design, acknowledged in code TODO)

## Problem

```go
type Form struct {
    ...
    Fields []Input `json:"fields,omitempty"`
    Submit Button  `json:"submit,omitempty"`
}
```

A form containing anything other than single-line text inputs — a `Choice`,
a checkbox, a multi-line text area — cannot be expressed. The code TODO on
the type already flags this ("The current shape assumes homogeneous inputs,
which is wrong for real forms"); this ticket exists so the decision gets made
before the `huh`-backed `FormModel` is built on the wrong shape.

Note the JSON consequence too: because `Fields` is `[]Input`, a document
whose `fields` array contains a choice object decodes each entry as an
`Input` with whatever fields happen to overlap — no error (see ticket 0002).

## Suggested fix

- Make `Fields []Component` (or a dedicated `FormField` closed union of the
  input-like kinds) with a custom `UnmarshalJSON` that dispatches on each
  element's `kind`.
- This interacts with ticket 0005: the real A2UI schema's form/field shape
  should drive the choice.
