<script lang="ts">
  import { onMount } from 'svelte'
  import type { ImportInspection, ImportPhase } from '../../lib/api'
  import type { ImportSession } from '../../lib/import-session'

  export let session: ImportSession

  const formatted = (value: number) => new Intl.NumberFormat().format(value)
  const formatLabels: Record<ImportInspection['format'], string> = {
    lichess: 'Lichess',
    'tactical-pgn': 'Tactical PGN',
    'canonical-json': 'Canonical JSON',
    'lucas-fns': 'Lucas FNS',
    'linear-fen-uci': 'Linear FEN/UCI'
  }
  const originLabels: Record<ImportInspection['sourceIdOrigin'], string> = {
    fixed: 'Fixed source ID',
    embedded: 'Embedded source ID',
    path: 'Fallback source ID (file path)'
  }
  const phaseLabels: Record<ImportPhase, string> = {
    detecting: 'Inspecting collection',
    parsing: 'Reading puzzles',
    sealing: 'Finalizing collection',
    activating: 'Activating collection'
  }

  onMount(() => {
    void session.refresh()
  })
</script>

<section class="panel import-panel" aria-labelledby="import-title">
  <p class="eyebrow">Puzzle collection</p>
  <h2 id="import-title">Import puzzle collection</h2>
  <p class="muted">
    Lichess, tactical PGN, canonical JSON, Lucas FNS, and linear FEN/UCI collections are
    streamed directly from this computer. No intermediate copy is created.
  </p>

  <div class="file-picker" aria-labelledby="puzzle-file-label">
    <span id="puzzle-file-label">Puzzle collection file</span>
    <button
      class="secondary"
      type="button"
      on:click={() => session.selectFile()}
      disabled={$session.running || $session.busy}
    >Choose puzzle collection</button>
    {#if $session.inspection}
      <div
        class="selected-source"
        aria-label="Selected puzzle collection"
        aria-live="polite"
      >
        <strong>{$session.inspection.sourceId}</strong>
        <span class="format-label">{formatLabels[$session.inspection.format]}</span>
        <span>{originLabels[$session.inspection.sourceIdOrigin]}</span>
        <span class="filename">{$session.inspection.filename}</span>
        <span class="path">{$session.inspection.path}</span>
        {#if $session.inspection.replacesExisting}
          <span class="replacement-warning">
            This import will replace the active {$session.inspection.sourceId} collection
          </span>
        {/if}
      </div>
    {:else}
      <p class="muted">No collection selected</p>
    {/if}
  </div>

  {#if $session.running}
    <div class="progress-card" aria-live="polite">
      <strong>{phaseLabels[$session.progress.phase]}</strong>
      <span>{formatted($session.progress.rowsRead)} rows read</span>
      <span>
        {#if $session.progress.totalBytes > 0}
          {formatted($session.progress.bytesRead)} of {formatted($session.progress.totalBytes)} bytes
        {:else}
          {formatted($session.progress.bytesRead)} bytes
        {/if}
      </span>
    </div>
    <button class="secondary" type="button" on:click={() => session.cancel()}>Cancel import</button>
  {:else}
    <button
      class="primary"
      type="button"
      on:click={() => session.start()}
      disabled={!$session.inspection || $session.busy}
    >Import puzzles</button>
  {/if}

  {#if $session.result && $session.result.status === 'succeeded'}
    <div class="report-grid" aria-label="Import report">
      <strong>{formatted($session.result.report.accepted)} accepted</strong>
      <span>{formatted($session.result.report.duplicates)} duplicates</span>
      <span>{formatted($session.result.report.rejected)} rejected</span>
    </div>
    {#if $session.result.report.examples.length > 0}
      <div class="rejection-examples" aria-labelledby="rejection-examples-title">
        <h3 id="rejection-examples-title">Rejection examples</h3>
        <ul>
          {#each $session.result.report.examples as example}
            <li>{example.ordinal}: {example.reason}</li>
          {/each}
        </ul>
      </div>
    {/if}
  {/if}
  {#if $session.result && $session.result.status === 'cancelled'}
    <p role="status">Import cancelled.</p>
  {/if}
  {#if $session.error}<p class="error" role="alert">{$session.error}</p>{/if}
</section>

<style>
  .file-picker { display: grid; gap: 10px; }
  .file-picker > span { font-weight: 700; }
  .file-picker button { justify-self: start; }
  .selected-source {
    display: grid;
    gap: 4px;
    min-width: 0;
    padding: 14px 16px;
    border-radius: 12px;
    background: var(--ivory-100);
  }
  .selected-source strong { overflow-wrap: anywhere; font-size: 1.15rem; }
  .selected-source span { color: var(--ink-700); font-size: 0.9rem; }
  .selected-source .format-label { color: var(--ink-900); font-weight: 800; }
  .selected-source .filename { margin-top: 4px; color: var(--ink-900); font-weight: 700; }
  .selected-source .path { overflow-wrap: anywhere; }
  .selected-source .replacement-warning {
    margin-top: 6px;
    color: var(--red-600);
    font-weight: 800;
  }
  .rejection-examples { margin-top: 20px; }
  .rejection-examples h3 { margin: 0 0 8px; font-size: 1rem; }
  .rejection-examples ul { margin: 0; padding-left: 22px; }
  .rejection-examples li { margin-top: 6px; overflow-wrap: anywhere; }
</style>
