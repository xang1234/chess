<script lang="ts">
  import { onMount } from 'svelte'
  import type {
    ImportKind,
    ImportPhase,
    ImportSourceIDOrigin
  } from '../../lib/api'
  import {
    canSelectImportFile,
    canStartImport,
    selectedImportInspection
  } from '../../lib/import-session'
  import type { ImportSession } from '../../lib/import-session'

  export let kind: ImportKind
  export let session: ImportSession
  export let heading: string
  export let description: string
  export let fileLabel: string
  export let chooseLabel: string
  export let startLabel: string

  const formatted = (value: number) => new Intl.NumberFormat().format(value)
  const originLabels: Record<ImportSourceIDOrigin, string> = {
    fixed: 'Fixed source ID',
    embedded: 'Embedded source ID',
    path: 'Fallback source ID (file path)'
  }
  const phaseLabels: Record<ImportKind, Record<ImportPhase, string>> = {
    puzzle: {
      detecting: 'Inspecting collection',
      parsing: 'Reading puzzles',
      sealing: 'Finalizing collection',
      activating: 'Activating collection'
    },
    course: {
      detecting: 'Inspecting course',
      parsing: 'Checking course records',
      sealing: 'Finalizing course',
      activating: 'Activating course'
    }
  }

  $: inspection = selectedImportInspection($session)
  $: runningState = $session.phase === 'running' ? $session : null
  $: finishedState = $session.phase === 'finished' ? $session : null
  $: selectedLabel = kind === 'course'
    ? 'Selected opening course'
    : 'Selected puzzle collection'
  $: replacementNoun = kind === 'course' ? 'course' : 'collection'
  $: recordProgress = kind === 'course'
    ? `${formatted(runningState?.progress.rowsRead ?? 0)} course records checked`
    : `${formatted(runningState?.progress.rowsRead ?? 0)} rows read`

  onMount(() => {
    void session.refresh()
  })
</script>

<article class="import-card" aria-labelledby={`${kind}-import-title`}>
  <h3 id={`${kind}-import-title`}>{heading}</h3>
  <p class="muted">{description}</p>

  <div class="file-picker" aria-labelledby={`${kind}-file-label`}>
    <span id={`${kind}-file-label`}>{fileLabel}</span>
    <button
      class="secondary"
      type="button"
      on:click={() => session.selectFile()}
      disabled={!canSelectImportFile($session)}
    >{chooseLabel}</button>
    {#if inspection}
      <div class="selected-source" aria-label={selectedLabel} aria-live="polite">
        <strong>{inspection.sourceId}</strong>
        {#if inspection.sourceName}<span class="source-name">{inspection.sourceName}</span>{/if}
        <span class="format-label">{inspection.formatLabel}</span>
        <span>{originLabels[inspection.sourceIdOrigin]}</span>
        {#if inspection.attribution}<span>{inspection.attribution}</span>{/if}
        <span class="filename">{inspection.filename}</span>
        <span class="path">{inspection.path}</span>
        {#if inspection.replacesExisting}
          <span class="replacement-warning">
            This import will replace the active {inspection.sourceId} {replacementNoun}
          </span>
        {/if}
      </div>
    {:else}
      <p class="muted">No {kind === 'course' ? 'course' : 'collection'} selected</p>
    {/if}
  </div>

  {#if runningState}
    <div class="progress-card" aria-live="polite">
      <strong>{phaseLabels[kind][runningState.progress.phase]}</strong>
      <span>{recordProgress}</span>
      <span>
        {#if runningState.progress.totalBytes > 0}
          {formatted(runningState.progress.bytesRead)} of {formatted(runningState.progress.totalBytes)} bytes
        {:else}
          {formatted(runningState.progress.bytesRead)} bytes
        {/if}
      </span>
    </div>
    <button class="secondary" type="button" on:click={() => session.cancel()}>Cancel import</button>
  {:else}
    <button
      class="primary"
      type="button"
      on:click={() => session.start()}
      disabled={!canStartImport($session)}
    >{startLabel}</button>
  {/if}

  {#if finishedState && finishedState.result.status === 'succeeded'}
    {#if kind === 'course'}
      <div class="report-grid course-report" aria-label="Course import report">
        <strong>{formatted(finishedState.result.report.counts.chapters ?? 0)} chapters</strong>
        <span>{formatted(finishedState.result.report.counts.moves ?? 0)} moves</span>
        <span>{formatted(finishedState.result.report.counts.lessons ?? 0)} lessons</span>
      </div>
    {:else}
      <div class="report-grid" aria-label="Import report">
        <strong>{formatted(finishedState.result.report.accepted)} accepted</strong>
        <span>{formatted(finishedState.result.report.duplicates)} duplicates</span>
        <span>{formatted(finishedState.result.report.rejected)} rejected</span>
      </div>
    {/if}
    {#if finishedState.result.report.examples.length > 0}
      <div class="rejection-examples" aria-labelledby={`${kind}-rejection-examples-title`}>
        <h3 id={`${kind}-rejection-examples-title`}>Rejection examples</h3>
        <ul>
          {#each finishedState.result.report.examples as example}
            <li>{example.ordinal}: {example.reason}</li>
          {/each}
        </ul>
      </div>
    {/if}
  {/if}
  {#if finishedState && finishedState.result.status === 'cancelled'}
    <p role="status">Import cancelled.</p>
  {/if}
  {#if $session.error}<p class="error" role="alert">{$session.error}</p>{/if}
</article>

<style>
  .import-card {
    display: grid;
    gap: 16px;
    min-width: 0;
    padding: 22px;
    border: 1px solid var(--ivory-200);
    border-radius: var(--radius-medium);
    background: var(--ivory-50);
    box-shadow: var(--shadow-soft);
  }
  .import-card h3, .import-card p { margin: 0; }
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
  .selected-source .source-name { color: var(--ink-900); font-weight: 700; }
  .selected-source .format-label { color: var(--ink-900); font-weight: 800; }
  .selected-source .filename { margin-top: 4px; color: var(--ink-900); font-weight: 700; }
  .selected-source .path { overflow-wrap: anywhere; }
  .selected-source .replacement-warning {
    margin-top: 6px;
    color: var(--red-600);
    font-weight: 800;
  }
  .rejection-examples { margin-top: 4px; }
  .rejection-examples h3 { margin: 0 0 8px; font-size: 1rem; }
  .rejection-examples ul { margin: 0; padding-left: 22px; }
  .rejection-examples li { margin-top: 6px; overflow-wrap: anywhere; }
</style>
