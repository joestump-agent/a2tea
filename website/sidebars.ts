import type { SidebarsConfig } from '@docusaurus/plugin-content-docs';

// Explicit sidebar mirroring the design: Getting started / Guides / Reference.
const sidebars: SidebarsConfig = {
  docsSidebar: [
    {
      type: 'category',
      label: 'Getting started',
      collapsible: false,
      items: ['intro', 'quickstart'],
    },
    {
      type: 'category',
      label: 'Guides',
      collapsible: false,
      items: ['wire-format', 'composition'],
    },
    {
      type: 'category',
      label: 'Reference',
      collapsible: false,
      items: ['api-reference', 'examples'],
    },
  ],
};

export default sidebars;
