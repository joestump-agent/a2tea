# Event model gaps: no form-level submit event, no source-component context

**Severity:** medium (design)

## Problem

The `event` package declares its three types "stable from day one", so gaps
in their shape are worth settling now, before agent-side code is written
against them:

1. **No aggregate form submission.** The `FormModel` TODO says "On submit,
   emit event.InputSubmitted per field". A consumer then has to collect N
   loose `InputSubmitted` messages, know when the set is complete, and
   correlate them back to one submit action — racy and awkward through a
   `tea.Msg` channel. A single `FormSubmitted{FormID string, Values map[string]string}`
   (plus the submit button ID, if forms can have multiple actions) is the
   shape agents actually want.

2. **No originating-component context.** `ButtonClicked{ID}` carries only the
   button's component-local ID. When several components are on screen (or the
   same reusable document is rendered twice), the consumer cannot tell which
   card/form/surface the click came from. Each event should carry the parent
   component ID (and later, per A2UI, the surface ID).

3. **Single-select only.** `ChoiceSelected.Value` is one string; if a
   multi-select choice ever lands (likely, per the A2UI catalog), the type
   cannot carry it. Worth deciding now whether Value becomes `[]string` or a
   separate event is added.

## Suggested fix

Add `FormSubmitted`, add a `ComponentID` (and/or `SurfaceID`) field to all
events, and pin the multi-select answer — all cheap now, breaking changes
later.
