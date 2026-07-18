import { themes as prismThemes } from 'prism-react-renderer';
import type { Config } from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

// a2tea documentation site.
// Deployed to GitHub Pages at https://joestump-agent.github.io/a2tea/.

const config: Config = {
  title: 'a2tea',
  tagline: 'render agent UIs in the terminal',
  favicon: 'img/logo-boba.svg',

  future: {
    v4: true,
  },

  url: 'https://joestump-agent.github.io',
  baseUrl: '/a2tea/',

  organizationName: 'joestump-agent',
  projectName: 'a2tea',
  trailingSlash: false,

  onBrokenLinks: 'throw',

  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: 'docs',
          editUrl: 'https://github.com/joestump-agent/a2tea/tree/main/website/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  // Terminal-first typefaces: JetBrains Mono (workhorse), Space Mono (display),
  // Silkscreen (pixel eyebrows). See the Bubble Tea TUI design system.
  stylesheets: [
    'https://fonts.googleapis.com/css2?family=JetBrains+Mono:ital,wght@0,400;0,500;0,700;0,800;1,400&family=Space+Mono:ital,wght@0,400;0,700;1,400&family=Silkscreen:wght@400;700&display=swap',
  ],

  themeConfig: {
    image: 'img/logo-boba.svg',
    colorMode: {
      defaultMode: 'dark',
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'a2tea',
      logo: {
        alt: 'a2tea boba cup logo',
        src: 'img/logo-boba.svg',
      },
      items: [
        { to: '/docs/intro', label: 'docs', position: 'left' },
        { to: '/docs/api-reference', label: 'api', position: 'left' },
        { to: '/docs/examples', label: 'examples', position: 'left' },
        { to: '/docs/wire-format', label: 'wire format', position: 'left' },
        {
          href: 'https://github.com/joestump-agent/a2tea',
          label: 'github ↗',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            { label: 'Introduction', to: '/docs/intro' },
            { label: 'Quickstart', to: '/docs/quickstart' },
            { label: 'Wire format', to: '/docs/wire-format' },
          ],
        },
        {
          title: 'Reference',
          items: [
            { label: 'API reference', to: '/docs/api-reference' },
            { label: 'Examples', to: '/docs/examples' },
            { label: 'Composition', to: '/docs/composition' },
          ],
        },
        {
          title: 'Ecosystem',
          items: [
            { label: 'GitHub ↗', href: 'https://github.com/joestump-agent/a2tea' },
            { label: 'A2UI ↗', href: 'https://a2ui.org' },
            { label: 'charm.land ↗', href: 'https://charm.land' },
          ],
        },
      ],
      copyright:
        'Apache-2.0 · pre-1.0, API may change · an independent bridge, not affiliated with Google or Charm 🫧',
    },
    prism: {
      theme: prismThemes.oneLight,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['go', 'bash', 'json'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
