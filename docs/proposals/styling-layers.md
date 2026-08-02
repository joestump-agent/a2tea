# Proposal: a2tea as the terminal catalog in A2UI's styling layers

Status: draft, for discussion. Companion doc: `docs/proposals/a2ui-mcp.md` in the
crush fork, which covers the host-side theme bridge and A2UI-over-MCP.

## Where a2tea sits in the A2UI styling model

The A2UI theming guide (a2ui.org/guides/theming) defines four styling layers:

1. **Agent-provided styling** — semantic hints only (`usageHint: "h1"`, button
   `primary`/`borderless`), never visual properties. The `theme` property on
   `createSurface` is deliberately untyped (`z.any()`) and even the reference web
   basic catalog does not consume it.
2. **Catalog theming** — the catalog maps hints to styled components and owns the
   theming solution. On the web that is a set of CSS variables
   (`--a2ui-color-primary`, `--a2ui-card-background`, …) injected at minimal
   specificity so the host wins.
3. **Platform theming** — the host developer overrides those variables.
4. Rendered output.

a2tea is the **catalog implementation for terminals**. The mapping is direct:

| A2UI web basic catalog | a2tea |
| --- | --- |
| CSS variables with low-specificity defaults | `render.Styles` fields with monochrome defaults |
| Host overrides via stylesheet | Host overrides via `render.WithStyles` |
| Semantic hints → styled DOM | Semantic hints → lipgloss styles |
| `prefers-color-scheme` dark mode | Inherit the terminal palette (monochrome default) |

This is already the architecture we have (`Styles` + `DefaultStyles()` +
`WithStyles`, landed in #31). The proposal below is about **completing the token
set**, **honoring more semantic hints**, and **supporting live restyling** — not
changing the model.

Two invariants stay:

- **Monochrome by default.** `DefaultStyles()` never sets a color, pinned by the
  no-color-SGR test in `render/styles_test.go`. The host's theme wins by
  construction.
- **`createSurface.theme` stays a no-op.** This matches the spec today (the web
  basic catalog ignores it too; the design is open in a2ui-project/a2ui#1118).
  If that issue resolves toward agent-influenced theming, we can add an opt-in
  `WithThemeMapper(func(a2ui.Theme, Styles) Styles)` later without breaking
  anything. `Theme.AdditionalProperties` is the escape hatch to watch.

## 1. Complete the style token set

`render.Styles` today has six fields (CardBorder, Heading, Subheading, Caption,
Button, ButtonFocused). A host can theme those, but most chrome is hardcoded:
focus cues (`▏`/`▎`), button brackets `[ ]`, checkbox/radio marks, list bullets,
tab separators, divider/slider glyphs, media placeholders. The web catalog's
per-component variables (`--a2ui-card-background`, …) have no terminal
equivalent yet.

Proposed shape (all defaults byte-identical to today's output):

```go
type Styles struct {
    // existing
    CardBorder, Heading, Subheading, Caption lipgloss.Style
    Button, ButtonFocused                    lipgloss.Style

    // new: component slots
    ButtonPrimary        lipgloss.Style // default: = Button (see §2)
    ButtonPrimaryFocused lipgloss.Style // default: = ButtonFocused
    Label                lipgloss.Style // input labels; default: = Caption
    Placeholder          lipgloss.Style // "(empty)"/"(unset)"/bindings; default: = Caption
    InputText            lipgloss.Style // field values; default: unstyled
    InputFocused         lipgloss.Style // focused field cue styling; default: unstyled
    Divider              lipgloss.Style // default: unstyled
    ListBullet           lipgloss.Style // default: unstyled
    TabActive, TabInactive lipgloss.Style // default: Heading / unstyled
    SliderFilled, SliderEmpty lipgloss.Style // default: unstyled

    // new: chrome glyphs (the terminal analog of per-component CSS vars)
    Glyphs Glyphs
}

type Glyphs struct {
    FocusCue, FocusCueActive string // "▏", "▎"
    ButtonOpen, ButtonClose  string // "[ ", " ]"
    CheckOn, CheckOff        string // "[x]", "[ ]"
    RadioOn, RadioOff        string // "(•)", "( )"
    ListBullet               string // "• "
    TabSeparator             string // " │ "
    ModalCue, ModalCueOpen   string // "▹", "▸"
    DividerH, DividerV       string // "─", "│"
    SliderFilled, SliderEmpty string // "█", "─"
}
```

Rules:

- `DefaultStyles()` fills every slot with the current hardcoded values; existing
  goldens and the no-color test must pass unchanged.
- New slots that today share an existing style (labels currently render with
  `Caption` directly — `render/fields.go`) use a **render-time fallback**: the
  renderer honors `Label` when the host set it explicitly and falls back to
  `Caption` otherwise (zero-value check). This is what actually preserves
  today's behavior — a host overriding `Caption` alone keeps restyling labels,
  exactly as it does now — while still letting a host split the two. Aliasing
  at `DefaultStyles()` time would do the opposite of preserving behavior:
  `st := DefaultStyles(); st.Caption = themed` (the pattern crush's
  `a2uiThemeStyles` already uses) would leave `Label` holding the stale
  default, so Caption-only overrides would silently stop reaching labels. If
  implementation constraints ever force `DefaultStyles()`-time aliasing
  instead, that is a documented behavior break for Caption-only overrides,
  not a compatibility guarantee — say so in the changelog.
- Zero-value glyphs fall back to defaults in `NewSurface` so a partially
  populated `Styles` from an older host doesn't render blank chrome.

## 2. Honor more semantic hints

In order of value:

1. **`TextField.Variant == "obscured"` must mask input.** Today the value renders
   in cleartext — a correctness/privacy bug, not a styling gap. Mask stored value
   and echo with `•`. (Independent of theming; should land first.)
2. **`Button.Variant == "primary"` gets a visual identity.** The old rationale —
   monochrome leaves no free attribute — dissolves once hosts can inject color:
   `ButtonPrimary` defaults to `Button`, so monochrome output is unchanged, but a
   themed host (crush) maps its accent color onto it.
3. **Per-level headings.** `Heading` (h1–h3) and `Subheading` (h4–h5) stay as
   coarse defaults; optional `H1…H5 lipgloss.Style` overrides for hosts that want
   distinction (zero-value = fall back to Heading/Subheading).
4. **`Image.Variant`** at minimum distinguishes `icon`/`avatar` (inline glyph)
   from feature/header sizes (line placeholder). Cheap, honest.
5. **`Row`/`Column` `Align`/`Justify` and `Component.Weight`** stay deferred —
   they need a real per-child layout budget. Document them as deferred in
   wire-format.md instead of silently dropping (Weight and Accessibility are
   currently undocumented drops).

## 3. Live restyling: `Surface.SetStyles`

Hosts with runtime theme switching (crush swaps themes per provider) can only
restyle a surface today by rebuilding it via `Render`, which **loses user input
state** (typed text, selections). Add:

```go
// SetStyles replaces the surface's style set. Safe at any time; the next
// View() renders with the new styles. Component state is untouched.
func (s *Surface) SetStyles(st Styles)
```

and surface it on `render.Model` (alongside `SetSize`/`Focus`) so hosts can
restyle in place on theme change.

## 4. Documentation

`WithStyles`, `DefaultStyles`, and `Styles` appear nowhere on the website — the
api-reference mentions `opts ...render.Option` without saying what an Option is.
When the token set lands, add a "Theming" page documenting every slot and glyph
(the terminal analog of the web catalog's CSS-variable reference), and state the
two invariants (monochrome default, `createSurface.theme` no-op) with the
rationale.

Housekeeping discovered while researching this: `AGENTS.md` predates #31 and
misdescribes the styling architecture ("package-level lipgloss.Style vars"),
plus several other stale claims (events, interactivity, tabs/modals,
`render/deps.go`). It should be refreshed in the same series.

## Sequencing

1. `obscured` masking fix (bug, independent).
2. `Styles`/`Glyphs` expansion + `SetStyles` + aliasing rules + tests.
3. Hint upgrades (`primary`, heading levels, image variants).
4. Website theming page + AGENTS.md refresh.

Crush-side consumption (palette mapping, live swap, MCP) is specified in the
companion crush doc.
