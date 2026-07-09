# Component schema is not verified against the actual A2UI specification

**Severity:** high (design) · **Blocks crush integration:** eventually

## Problem

The library's premise is "maps A2UI JSON messages onto Bubble Tea models",
but the wire format implemented here — a single flat object with a `kind`
discriminator and inline fields — is a provisional invention. The source
acknowledges this in several TODOs ("flesh out fields once the A2UI schema
for cards is pinned down"), but there is no ticket tracking the actual
conformance work, and nothing in the repo records *which* A2UI schema
version the shapes were derived from.

The real A2UI protocol (a2ui.org) is message-oriented rather than
single-component-oriented: a stream of messages creates and updates surfaces,
components reference `children` by ID in an adjacency-list style rather than
nesting, and values can be bound to a data model that updates independently
of the component tree. None of that is representable in the current
`component.Component` union — there is no container kind at all (no column /
row / list), no update-in-place message, and `Render` accepts exactly one
component per call.

If crush integration starts against the invented shape, every document the
agent produces will need a bespoke translation layer later — or a2tea's wire
format forks from A2UI permanently while the README still claims A2UI
compatibility.

## Suggested fix

1. Pin the A2UI spec version being targeted and vendor a copy (or link a
   commit-anchored URL) into `docs/`.
2. Redesign `component.Unmarshal` around the spec's actual message envelope
   and component catalog, including container components and `children`
   handling (already a TODO in the code).
3. If full A2UI is out of scope for v0, say so in the README and rename the
   claim to "A2UI-inspired" so downstream expectations are correct.
