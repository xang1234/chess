<script lang="ts">
  import { onMount } from 'svelte'
  import type { ImportSession } from '../../lib/import-session'

  export let session: ImportSession

  const formatted = (value: number) => new Intl.NumberFormat().format(value)
  const filename = (path: string) => path.split(/[\\/]/).pop() ?? path

  onMount(() => {
    void session.refresh()
  })
</script>

<section class="panel import-panel" aria-labelledby="import-title">
  <p class="eyebrow">Puzzle collection</p>
  <h2 id="import-title">Import Lichess puzzles</h2>
  <p class="muted">The compressed file is read directly. No decompressed copy is created.</p>

  <div class="file-picker" aria-labelledby="puzzle-file-label">
    <span id="puzzle-file-label">Puzzle database file</span>
    <button
      class="secondary"
      type="button"
      on:click={() => session.selectFile()}
      disabled={$session.running || $session.busy}
    >Choose puzzle database</button>
    {#if $session.path}
      <div class="selected-file" aria-live="polite">
        <strong>{filename($session.path)}</strong>
        <span>{$session.path}</span>
      </div>
    {:else}
      <p class="muted">No file selected</p>
    {/if}
  </div>

  {#if $session.running}
    <div class="progress-card" aria-live="polite">
      <strong>{formatted($session.progress.rowsRead)} rows read</strong>
      <span>{formatted($session.progress.bytesRead)} compressed bytes</span>
    </div>
    <button class="secondary" type="button" on:click={() => session.cancel()}>Cancel import</button>
  {:else}
    <button class="primary" type="button" on:click={() => session.start()} disabled={!$session.path || $session.busy}>Import puzzles</button>
  {/if}

  {#if $session.result && $session.result.status === 'succeeded'}
    <div class="report-grid" aria-label="Import report">
      <strong>{formatted($session.result.report.accepted)} accepted</strong>
      <span>{formatted($session.result.report.duplicates)} duplicates</span>
      <span>{formatted($session.result.report.rejected)} rejected</span>
    </div>
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
  .selected-file { display: grid; gap: 4px; min-width: 0; padding: 12px 14px; border-radius: 12px; background: var(--ivory-100); }
  .selected-file span { overflow-wrap: anywhere; color: var(--ink-700); font-size: 0.9rem; }
</style>
