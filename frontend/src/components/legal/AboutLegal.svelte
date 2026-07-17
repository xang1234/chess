<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte'
  import type { BuildInfo } from '../../lib/api'
  import {
    loadLegalDocuments,
    type LegalDocuments
  } from '../../lib/legal-assets'
  import { openExternal as openExternalLink } from '../../lib/external-links'

  export let buildInfo: BuildInfo
  export let loadDocuments: () => Promise<LegalDocuments> = loadLegalDocuments
  export let openExternal: (url: string) => void = openExternalLink

  const dispatch = createEventDispatcher<{ back: void }>()
  let documents: LegalDocuments | null = null
  let error = ''

  onMount(() => {
    void load()
  })

  async function load(): Promise<void> {
    try {
      documents = await loadDocuments()
    } catch (cause) {
      error = errorMessage(cause)
    }
  }

  function openSource(): void {
    try {
      openExternal(buildInfo.sourceUrl)
    } catch (cause) {
      error = errorMessage(cause)
    }
  }

  function errorMessage(cause: unknown): string {
    return cause instanceof Error ? cause.message : String(cause)
  }
</script>

<section class="legal-panel panel" aria-labelledby="legal-title">
  <div class="legal-heading">
    <div>
      <p class="eyebrow">Source and licences</p>
      <h2 id="legal-title">About &amp; Legal</h2>
    </div>
    <button class="secondary" type="button" on:click={() => dispatch('back')}>Back</button>
  </div>

  <div class="rights-copy">
    <p>Copyright © 2026 David Ten and Chess Trainer contributors</p>
    <p>
      Chess Trainer is free software: you may modify and redistribute it under the
      GNU General Public License, version 3 or later (GPL-3.0-or-later).
    </p>
    <p><strong>WITHOUT ANY WARRANTY</strong>, to the extent permitted by law.</p>
  </div>

  <div class="attribution-grid">
    <article>
      <h3>@lichess-org/chessground 10.1.1</h3>
      <p>Lichess Team &lt;contact@lichess.org&gt;</p>
      <p>https://github.com/lichess-org/chessground</p>
      <p>GPL-3.0-or-later</p>
      <p>Preferred source: <code>third_party/source/chessground-v10.1.1.tar.gz</code></p>
    </article>
    <article>
      <h3>Nunito v16</h3>
      <p>Copyright 2016 The Nunito Project Authors (contact@sansoxygen.com)</p>
      <p>SIL Open Font License 1.1</p>
    </article>
  </div>

  <section class="build-identity" aria-labelledby="build-identity-title">
    <h3 id="build-identity-title">Matching source for this build</h3>
    <dl>
      <div><dt>Build commit</dt><dd><code>{buildInfo.commit}</code></dd></div>
      <div><dt>Source URL</dt><dd><code>{buildInfo.sourceUrl}</code></dd></div>
    </dl>
    <button class="primary" type="button" on:click={openSource}>Open matching source</button>
  </section>

  {#if error}
    <p class="error" role="alert">{error}</p>
  {/if}

  {#if documents}
    <div class="legal-documents">
      <details>
        <summary>Application license (GPL-3.0-or-later)</summary>
        <pre aria-label="Application license text">{documents.application}</pre>
      </details>
      <details>
        <summary>Third-party notices</summary>
        <pre aria-label="Third-party notices text">{documents.notices}</pre>
      </details>
      <details>
        <summary>Chessground license</summary>
        <pre aria-label="Chessground license text">{documents.chessground}</pre>
      </details>
      <details>
        <summary>Nunito SIL Open Font License 1.1</summary>
        <pre aria-label="Nunito license text">{documents.nunito}</pre>
      </details>
    </div>
  {:else if !error}
    <p class="loading" aria-live="polite">Loading bundled legal documents…</p>
  {/if}
</section>

<style>
  .legal-panel { width: min(980px, 100%); }
  .legal-heading { display: flex; gap: 20px; align-items: flex-start; justify-content: space-between; }
  .legal-heading h2 { margin-top: 6px; }
  .rights-copy { margin: 24px 0; }
  .rights-copy p { margin: 9px 0; }
  .attribution-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
  .attribution-grid article,
  .build-identity {
    padding: 18px;
    border-radius: var(--radius-medium);
    background: var(--ivory-100);
  }
  h3 { margin: 0 0 10px; }
  .attribution-grid p { margin: 5px 0; overflow-wrap: anywhere; }
  .build-identity { margin: 14px 0; }
  dl { display: grid; gap: 8px; }
  dl div { display: grid; grid-template-columns: 120px minmax(0, 1fr); gap: 10px; }
  dt { font-weight: 800; }
  dd { margin: 0; min-width: 0; overflow-wrap: anywhere; }
  .legal-documents { display: grid; gap: 10px; margin-top: 22px; }
  details { border: 1px solid var(--ivory-200); border-radius: 12px; background: var(--white); }
  summary { min-height: 44px; padding: 12px 14px; cursor: pointer; font-weight: 800; }
  pre {
    max-height: 360px;
    margin: 0;
    padding: 16px;
    overflow: auto;
    border-top: 1px solid var(--ivory-200);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }
  @media (max-width: 700px) {
    .attribution-grid { grid-template-columns: 1fr; }
    dl div { grid-template-columns: 1fr; gap: 2px; }
  }
</style>
