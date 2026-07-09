# a2tea wire format status

**Status: provisional / A2UI-inspired. Not yet verified against the a2ui.org
specification.**

This document records what the JSON wire format `a2tea` currently implements,
how it diverges from the real [A2UI](https://a2ui.org) protocol, and what has
to happen before the README can claim plain "A2UI compatibility". It exists so
that anyone building against a2tea — especially the crush integration — knows
which parts are safe to depend on and which are placeholders.

## What is implemented today

A single, flat JSON object with a `kind` discriminator and inline fields, one
object per `a2tea.Render` call:

```json
{ "kind": "card", "id": "hello", "title": "Hi", "body": "…", "buttons": [ … ] }
```

Recognized kinds: `card`, `form`, `input`, `choice`, `progress`, `markdown`,
`stream`. Each maps to one concrete type in the `component` package and one
stub renderer in `render`.

## Known divergences from A2UI

The current shape was **invented** to get the pipeline wired end-to-end. It has
not been transcribed from a pinned version of the a2ui.org schema, and it
differs from the real protocol in ways that matter:

1. **Message-oriented vs. component-oriented.** A2UI is a stream of messages
   that create and update *surfaces* over time. a2tea decodes exactly one
   component per call and has no update-in-place message.
2. **Nesting vs. adjacency.** A2UI components reference `children` by ID in an
   adjacency-list style. a2tea has no container kind at all (no column / row /
   list) and no `children` handling.
3. **Data binding.** A2UI values can bind to a data model that updates
   independently of the component tree. a2tea has no data-model concept.

Because of (1)–(3), a document an agent produces for real A2UI will not decode
here unchanged, and vice versa.

## Decision required (tracked by issue #5)

Before crush consumes this library against the invented shape, the owner needs
to either:

1. **Target A2UI for real** — pin a specific a2ui.org schema version, vendor or
   commit-anchor a copy into this `docs/` directory, and redesign
   `component.Unmarshal` around the spec's message envelope and component
   catalog (containers, `children`, surfaces, data binding); or
2. **Scope A2UI out of v0** — keep this provisional shape and describe it
   honestly as "A2UI-inspired" so downstream expectations are correct.

Until that decision is made and this document names a concrete schema version,
treat the wire format as unstable. The README's compatibility claim has been
softened to "A2UI-inspired" to reflect (2) as the current default.
