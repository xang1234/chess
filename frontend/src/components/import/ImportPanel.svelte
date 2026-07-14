<script lang="ts">
  import { onMount } from 'svelte'
  import { getAPI, type ImportProgress, type ImportResult } from '../../lib/api'

  let path = ''
  let jobId = ''
  let running = false
  let progress: ImportProgress = { jobId: '', rowsRead: 0, bytesRead: 0 }
  let result: ImportResult | null = null
  let error = ''

  const formatted = (value: number) => new Intl.NumberFormat().format(value)

  onMount(() => {
    const stopProgress = getAPI().onImportProgress((event) => {
      if (event.jobId === jobId) progress = event
    })
    const stopFinished = getAPI().onImportFinished((event) => {
      if (event.jobId !== jobId) return
      result = event
      running = false
      error = event.error ?? ''
    })
    return () => {
      stopProgress()
      stopFinished()
    }
  })

  async function start(): Promise<void> {
    error = ''
    result = null
    progress = { jobId: '', rowsRead: 0, bytesRead: 0 }
    try {
      jobId = await getAPI().startLichessImport(path)
      running = true
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    }
  }

  async function cancel(): Promise<void> {
    if (jobId) await getAPI().cancelImport(jobId)
  }
</script>

<section class="panel import-panel" aria-labelledby="import-title">
  <p class="eyebrow">Puzzle collection</p>
  <h2 id="import-title">Import Lichess puzzles</h2>
  <p class="muted">The compressed file is read directly. No decompressed copy is created.</p>

  <label>
    <span>Puzzle database file</span>
    <input type="text" bind:value={path} placeholder="/path/to/lichess_db_puzzle.csv.zst" disabled={running} />
  </label>

  {#if running}
    <div class="progress-card" aria-live="polite">
      <strong>{formatted(progress.rowsRead)} rows read</strong>
      <span>{formatted(progress.bytesRead)} compressed bytes</span>
    </div>
    <button class="secondary" type="button" on:click={cancel}>Cancel import</button>
  {:else}
    <button class="primary" type="button" on:click={start} disabled={!path}>Import puzzles</button>
  {/if}

  {#if result && result.status === 'succeeded'}
    <div class="report-grid" aria-label="Import report">
      <strong>{formatted(result.report.accepted)} accepted</strong>
      <span>{formatted(result.report.duplicates)} duplicates</span>
      <span>{formatted(result.report.rejected)} rejected</span>
    </div>
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
</section>
