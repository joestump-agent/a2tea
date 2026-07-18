# a2tea docs site

The [a2tea](https://github.com/joestump-agent/a2tea) documentation site, built
with [Docusaurus](https://docusaurus.io/) and the Bubble Tea TUI design system
(deep blue-black terminal surfaces lit by ANSI neon, with a lavender-paper light
mode).

Published to GitHub Pages at **https://joestump-agent.github.io/a2tea/** by the
[`Deploy docs`](../.github/workflows/deploy-docs.yml) workflow on every push to
`main` that touches `website/`.

## Local development

```bash
cd website
npm install
npm start          # dev server with hot reload
npm run build      # production build into website/build
npm run serve      # serve the production build locally
```

## Layout

- `src/pages/index.tsx` — the landing page (hero, feature tiles, flow, host code).
- `src/css/custom.css` — design tokens ported onto Infima, with `[data-theme]`
  light/dark variants.
- `docs/` — the documentation pages (intro, quickstart, wire format,
  composition, API reference, examples).
- `docusaurus.config.ts` / `sidebars.ts` — site config and the explicit sidebar.

The design tokens come from the Bubble Tea TUI design system; the palette,
typography, and spacing scales live inline in `src/css/custom.css`.
