import type { ReactNode } from 'react';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';
import styles from './index.module.css';

type Tile = {
  glyph: string;
  glyphColor: string;
  title: string;
  body: string;
  linkLabel: string;
  to: string;
};

const TILES: Tile[] = [
  {
    glyph: '◆',
    glyphColor: 'var(--charm-purple)',
    title: 'Detect A2UI in any reply',
    body:
      'Contains and Scan split an assistant message into ordered parts of prose and typed A2UI messages — no hand-rolled JSON sniffing.',
    linkLabel: 'Scan reference →',
    to: '/docs/api-reference',
  },
  {
    glyph: '╭╮',
    glyphColor: 'var(--neon-cyan)',
    title: 'Real Lip Gloss rendering',
    body:
      'Text variants, rounded Cards, Column / Row / List layout, Dividers and focusable Buttons — drawn with charm.land/lipgloss/v2.',
    linkLabel: 'Wire format →',
    to: '/docs/wire-format',
  },
  {
    glyph: '●',
    glyphColor: 'var(--charm-pink)',
    title: 'The real A2UI protocol',
    body:
      'Targets A2UI v0.9 via tmc/a2ui. a2tea invents no component types — it decodes and renders the actual A2UI catalog.',
    linkLabel: 'Read the spec notes →',
    to: '/docs/wire-format',
  },
  {
    glyph: '⇥',
    glyphColor: 'var(--neon-mint)',
    title: 'Wired interaction',
    body:
      'Tab / Shift+Tab cycle focusables; Enter emits event.ButtonClicked and a protocol-native ClientMessage back to the agent.',
    linkLabel: 'Events →',
    to: '/docs/api-reference',
  },
  {
    glyph: '▢',
    glyphColor: 'var(--neon-lilac)',
    title: 'Embeddable by design',
    body:
      'Render returns a render.Model child that never calls tea.Quit — the host owns the terminal, the layout, and the theme.',
    linkLabel: 'Composition →',
    to: '/docs/composition',
  },
  {
    glyph: '▶',
    glyphColor: 'var(--neon-gold)',
    title: 'Run it standalone',
    body:
      'Wrap any surface in a2tea.Standalone to quit on q / Esc and forward terminal size — perfect for examples and manual testing.',
    linkLabel: 'See examples →',
    to: '/docs/examples',
  },
];

function Hero(): ReactNode {
  return (
    <section className={styles.wrap}>
      <div className={styles.hero}>
        <div>
          <div className={`${styles.eyebrow} ${styles.eyebrowCyan}`}>
            // A2UI → BUBBLE TEA BRIDGE
          </div>
          <h1 className={styles.heroTitle}>
            render agent UIs
            <br />
            in the <span className={styles.gradTron}>terminal</span>
          </h1>
          <p className={styles.heroLede}>
            a2tea parses the{' '}
            <a href="https://a2ui.org" target="_blank" rel="noopener noreferrer">
              A2UI
            </a>{' '}
            an agent emits — interleaved with prose in an LLM reply — and draws
            the described surfaces as{' '}
            <a href="https://charm.land" target="_blank" rel="noopener noreferrer">
              Bubble Tea
            </a>{' '}
            models. JSON in, a live TUI out.
          </p>
          <div className={styles.heroCtas}>
            <Link className={`${styles.btn} ${styles.btnPrimary}`} to="/docs/intro">
              get started →
            </Link>
            <Link className={`${styles.btn} ${styles.btnGhost}`} to="/docs/quickstart">
              $ go get a2tea
            </Link>
          </div>
          <div className={styles.badges}>
            <span className={`${styles.badge} ${styles.badgeInfo}`}>Go</span>
            <span className={`${styles.badge} ${styles.badgePrimary}`}>A2UI v0.9</span>
            <span className={`${styles.badge} ${styles.badgeAccent}`}>lipgloss/v2</span>
            <span className={`${styles.badge} ${styles.badgeMuted}`}>Apache-2.0</span>
          </div>
        </div>
        <div>
          <div className={styles.term}>
            <div className={styles.termBar}>
              <i className={`${styles.dot} ${styles.dotPink}`} />
              <i className={`${styles.dot} ${styles.dotGold}`} />
              <i className={`${styles.dot} ${styles.dotMint}`} />
              <span className={styles.termTitle}>~/kyoto — crush</span>
            </div>
            <div className={styles.termBody}>
              <div className={styles.termPrompt}>$ crush "plan a weekend in kyoto"</div>
              <div style={{ marginTop: 9 }} className={styles.termDim}>
                Here's an option for your trip —
              </div>
              <div className={styles.termCard}>
                <div className={styles.termCardTitle}>Kyoto · Autumn Weekend</div>
                <div className={styles.termCardSub}>3 nights · ¥182,000 · 2 travelers</div>
                <div className={styles.termRule} />
                <div className={styles.termBtns}>
                  <span className={styles.termBtnGo}>[ Book it ]</span>
                  <span className={styles.termBtnOutline}>Adjust dates</span>
                </div>
              </div>
              <div className={styles.termHelp}>
                <span className={styles.termKey}>tab</span> cycle •{' '}
                <span className={styles.termKey}>enter</span> select •{' '}
                <span className={styles.termKey}>q</span> quit
                <span className={styles.cursor} />
              </div>
            </div>
          </div>
          <div className={styles.heroCaption}>
            a2tea turns an A2UI surface into a focusable Bubble Tea model.
          </div>
        </div>
      </div>
    </section>
  );
}

function Features(): ReactNode {
  return (
    <section className={styles.section}>
      <div className={`${styles.eyebrow} ${styles.eyebrowPink}`}>// WHAT IT DOES</div>
      <h2 className={styles.sectionTitle}>a real renderer, not a mock</h2>
      <p className={styles.sectionLede}>
        Everything a host needs to recognize A2UI in a model's reply and draw it —
        detection, real Lip Gloss rendering, and the first wired interaction
        round-trip.
      </p>
      <div className={styles.tiles}>
        {TILES.map((t) => (
          <div className={styles.tile} key={t.title}>
            <div className={styles.tileGlyph} style={{ color: t.glyphColor }}>
              {t.glyph}
            </div>
            <h3 className={styles.tileTitle}>{t.title}</h3>
            <p className={styles.tileBody}>{t.body}</p>
            <Link className={styles.tileLink} to={t.to}>
              {t.linkLabel}
            </Link>
          </div>
        ))}
      </div>
    </section>
  );
}

function Flow(): ReactNode {
  return (
    <section className={styles.section}>
      <div className={`${styles.eyebrow} ${styles.eyebrowCyan}`}>// THE FLOW</div>
      <h2 className={styles.sectionTitle} style={{ marginBottom: 40 }}>
        reply in, surface out
      </h2>
      <div className={styles.flow}>
        <div className={styles.flowCard}>
          <div className={styles.flowNum}>01</div>
          <h3 className={styles.flowTitle}>Agent emits A2UI</h3>
          <p className={styles.flowBody}>
            Messages arrive interleaved with prose, wrapped in{' '}
            <span className={`${styles.mono} ${styles.cPink}`}>&lt;a2ui-json&gt;</span> tags.
          </p>
        </div>
        <div className={styles.flowArrow}>→</div>
        <div className={styles.flowCard}>
          <div className={styles.flowNum}>02</div>
          <h3 className={styles.flowTitle}>Scan splits it</h3>
          <p className={styles.flowBody}>
            You get ordered <span className={`${styles.mono} ${styles.cLilac}`}>[]Part</span> —
            each a chunk of Text plus typed Messages.
          </p>
        </div>
        <div className={styles.flowArrow}>→</div>
        <div className={styles.flowCard}>
          <div className={styles.flowNum}>03</div>
          <h3 className={styles.flowTitle}>Render draws it</h3>
          <p className={styles.flowBody}>
            Messages composite into an embeddable{' '}
            <span className={`${styles.mono} ${styles.cLilac}`}>render.Model</span> the host draws.
          </p>
        </div>
      </div>
      <p className={styles.flowNote}>
        The message lifecycle is applied in order — updateComponents composites by
        ID, updateDataModel resolves bindings, deleteSurface clears the surface.
      </p>
    </section>
  );
}

function HostSplit(): ReactNode {
  return (
    <section className={styles.section} style={{ paddingBottom: 72 }}>
      <div className={styles.split}>
        <div>
          <div className={`${styles.eyebrow} ${styles.eyebrowPink}`}>// FOR HOSTS</div>
          <h2 className={styles.sectionTitle} style={{ fontSize: 30 }}>
            a host is a handful of lines
          </h2>
          <p className={styles.sectionLede} style={{ marginBottom: 20 }}>
            Feed an assistant reply to Scan, render each part's text as prose, and
            hand each part's messages to Render. No quit handling — the model embeds
            into your program.
          </p>
          <Link className={styles.tileLink} to="/docs/quickstart">
            Full quickstart →
          </Link>
        </div>
        <div className={styles.codePanel}>
          <div className={styles.codePanelBar}>
            <span className={styles.cPink}>❯</span>
            <span>host.go</span>
            <span className={styles.codePanelLang}>GO</span>
          </div>
          <pre className={styles.code}>
            <span className={styles.var}>parts</span>, err := <span className={styles.pkg}>a2tea</span>.
            <span className={styles.fn}>Scan</span>(reply){'\n'}
            <span className={styles.kw}>for</span> _, p := <span className={styles.kw}>range</span> parts {'{'}
            {'\n'}    <span className={styles.kw}>if</span> p.Text != <span className={styles.str}>""</span> {'{'}
            {'\n'}        <span className={styles.fn}>renderProse</span>(p.Text){'\n'}    {'}'}
            {'\n'}    model, err := <span className={styles.pkg}>a2tea</span>.
            <span className={styles.fn}>Render</span>(p.Messages){'\n'}    <span className={styles.kw}>if</span> err !={' '}
            <span className={styles.kw}>nil</span> {'{'}
            {'\n'}        <span className={styles.kw}>continue</span>{' '}
            <span className={styles.cmt}>// no renderable surface</span>
            {'\n'}    {'}'}
            {'\n'}    <span className={styles.fn}>draw</span>(model){' '}
            <span className={styles.cmt}>// an embeddable tea.Model</span>
            {'\n'}
            {'}'}
          </pre>
        </div>
      </div>
    </section>
  );
}

export default function Home(): ReactNode {
  return (
    <Layout
      title="render agent UIs in the terminal"
      description="a2tea parses the A2UI an agent emits and draws the described surfaces as Bubble Tea models. JSON in, a live TUI out.">
      <main className={styles.page}>
        <Hero />
        <Features />
        <Flow />
        <HostSplit />
      </main>
    </Layout>
  );
}
