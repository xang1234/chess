<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte'
  import type {
    NormalAPI,
    OpeningDepth,
    OpeningEvaluation,
    OpeningPositionView,
    OpeningSourceRef
  } from '../../lib/api'
  import { useNormalAPI } from '../../lib/api-context'
  import ChessBoard from '../chess/ChessBoard.svelte'
  import {
    createChessgroundAdapter,
    type ChessgroundAdapterFactory
  } from '../chess/chessground-adapter'

  export let courseId: string
  export let rootPositionId: string
  export let depth: OpeningDepth
  export let boardAdapterFactory: ChessgroundAdapterFactory = createChessgroundAdapter

  const api: NormalAPI = useNormalAPI()
  const dispatch = createEventDispatcher<{ back: void }>()
  let history: OpeningPositionView[] = []
  let loading = true
  let error = ''
  let requestSequence = 0

  $: current = history.at(-1)

  onMount(() => {
    void loadPosition(rootPositionId, false)
    return () => { requestSequence++ }
  })

  async function loadPosition(positionId: string, append: boolean): Promise<void> {
    const requestId = ++requestSequence
    loading = true
    error = ''
    try {
      const position = await api.getOpeningPosition(courseId, positionId, depth)
      if (requestId !== requestSequence) return
      history = append ? [...history, position] : [position]
    } catch (cause) {
      if (requestId !== requestSequence) return
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      if (requestId === requestSequence) loading = false
    }
  }

  function backOneMove(): void {
    if (history.length <= 1) return
    requestSequence++
    loading = false
    error = ''
    history = history.slice(0, -1)
  }

  function reset(): void {
    if (history.length === 0) {
      void loadPosition(rootPositionId, false)
      return
    }
    requestSequence++
    loading = false
    error = ''
    history = [history[0]]
  }

  function returnTo(index: number): void {
    requestSequence++
    loading = false
    error = ''
    history = history.slice(0, index + 1)
  }

  function evaluationLabel(evaluation: OpeningEvaluation): string {
    const labels: Record<OpeningEvaluation['code'], string> = {
      none: 'No evaluation',
      equal: 'Equal',
      unclear: 'Unclear',
      white_slight: 'White is slightly better',
      black_slight: 'Black is slightly better',
      white_clear: 'White is clearly better',
      black_clear: 'Black is clearly better',
      white_winning: 'White is winning',
      black_winning: 'Black is winning'
    }
    return `${labels[evaluation.code]}${evaluation.sourceSymbol ? ` (${evaluation.sourceSymbol})` : ''}`
  }

  function sourceLabel(source: OpeningSourceRef): string {
    const coordinates = [
      source.tableColumn ? `column ${source.tableColumn}` : '',
      source.noteLabel ? `note ${source.noteLabel}` : ''
    ].filter(Boolean)
    return `Source page ${source.printedPage}${coordinates.length ? ` · ${coordinates.join(' · ')}` : ''}`
  }
</script>

<section class="variation-explorer" aria-labelledby="explorer-title">
  <header class="explorer-header">
    <div>
      <p class="eyebrow">Read-only course reference</p>
      <h2 id="explorer-title">Variation explorer</h2>
    </div>
    <div class="button-row">
      <button class="secondary" type="button" on:click={() => dispatch('back')}>Back to course</button>
      <button class="secondary" type="button" disabled={history.length <= 1} on:click={backOneMove}>
        Back one move
      </button>
      <button class="secondary" type="button" disabled={history.length <= 1} on:click={reset}>
        Reset
      </button>
    </div>
  </header>

  {#if error}
    <div class="course-notice" role="alert">
      <p>{error}</p>
      <button class="secondary" type="button" on:click={() => loadPosition(current?.positionId ?? rootPositionId, false)}>
        Try again
      </button>
    </div>
  {/if}

  {#if current}
    <div class="explorer-layout" aria-busy={loading}>
      <aside class="explorer-branches" aria-label="Move history and branches">
        <div>
          <p class="eyebrow">Move order</p>
          <ol class="explorer-history">
            {#each history as position, index (position.positionId + index)}
              <li>
                <button
                  type="button"
                  aria-current={index === history.length - 1 ? 'step' : undefined}
                  on:click={() => returnTo(index)}
                >{position.label || position.positionId}</button>
              </li>
            {/each}
          </ol>
        </div>

        <div>
          <p class="eyebrow">Branches</p>
          {#if current.moves.length === 0}
            <p class="muted explorer-end">No further moves at {depth} depth.</p>
          {:else}
            <div class="branch-list">
              {#each current.moves as move (move.moveId)}
                <button
                  class="branch-button"
                  type="button"
                  aria-label={`${move.san}${move.variationName ? ` — ${move.variationName}` : ''}`}
                  disabled={loading}
                  on:click={() => loadPosition(move.toPositionId, true)}
                >
                  <span><strong>{move.san}</strong>{move.variationName ? ` · ${move.variationName}` : ''}</span>
                  <small>{evaluationLabel(move.evaluation)}</small>
                  <small>{sourceLabel(move.sourceRef)}</small>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      </aside>

      <div class="explorer-board">
        {#key current.positionId + history.length}
          <ChessBoard
            fen={current.fen}
            orientation="white"
            legalMoves={[]}
            inputEnabled={false}
            reducedMotion={true}
            adapterFactory={boardAdapterFactory}
          />
        {/key}
        {#if loading}<p class="board-loading" aria-live="polite">Loading position…</p>{/if}
      </div>

      <aside class="explorer-notes">
        <p class="eyebrow">Position</p>
        <h3>{current.label || current.positionId}</h3>
        <p class="evaluation">{evaluationLabel(current.evaluation)}</p>
        {#if current.incomingPaths > 1}
          <p class="transposition">
            This position is reached by {current.incomingPaths} move orders.
          </p>
        {/if}
        {#if current.notes.length === 0}
          <p class="muted">No reference notes for this position.</p>
        {:else}
          <div class="position-notes">
            {#each current.notes as note, index (`${note.sourceRef.coverageId}-${index}`)}
              <article>
                <strong>{note.kind}</strong>
                <p>{note.text}</p>
                <small>{sourceLabel(note.sourceRef)}</small>
              </article>
            {/each}
          </div>
        {/if}
      </aside>
    </div>
  {:else if loading}
    <p class="loading" aria-live="polite">Loading the course tree…</p>
  {/if}
</section>

<style>
  .variation-explorer { width: min(1220px, 100%); }
  .explorer-header {
    display: flex;
    gap: 20px;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 22px;
  }
  .explorer-header h2 { margin: 4px 0 0; font-size: clamp(1.7rem, 4vw, 2.5rem); }
  .course-notice p { margin-top: 0; }
  .explorer-layout {
    display: grid;
    grid-template-columns: minmax(220px, 0.72fr) minmax(360px, 1.6fr) minmax(250px, 0.82fr);
    gap: clamp(16px, 2vw, 26px);
    align-items: start;
  }
  .explorer-branches,
  .explorer-notes {
    display: grid;
    gap: 22px;
    padding: 20px;
    border: 1px solid var(--ivory-200);
    border-radius: var(--radius-medium);
    background: var(--ivory-50);
    box-shadow: var(--shadow-soft);
  }
  .explorer-history { display: grid; margin: 10px 0 0; padding-left: 22px; gap: 5px; }
  .explorer-history button {
    min-height: 36px;
    padding: 4px 7px;
    border: 0;
    color: var(--forest-700);
    text-align: left;
    background: transparent;
    font-weight: 800;
  }
  .explorer-history button[aria-current="step"] { color: var(--ink-900); text-decoration: underline; }
  .branch-list { display: grid; margin-top: 10px; gap: 9px; }
  .branch-button {
    display: grid;
    min-height: 74px;
    padding: 10px 12px;
    gap: 3px;
    border: 1px solid #b9b09c;
    border-radius: 10px;
    color: var(--ink-900);
    text-align: left;
    background: var(--white);
  }
  .branch-button small,
  .position-notes small { color: var(--ink-700); }
  .explorer-end { margin-bottom: 0; }
  .explorer-board { position: relative; min-width: 0; }
  .board-loading {
    position: absolute;
    right: 12px;
    bottom: 12px;
    margin: 0;
    padding: 7px 10px;
    border-radius: 8px;
    background: rgba(255, 253, 247, 0.9);
    font-weight: 800;
  }
  .explorer-notes h3 { margin: -14px 0 0; font-size: 1.55rem; }
  .evaluation { margin: -12px 0 0; color: var(--forest-700); font-weight: 800; }
  .transposition { margin: -8px 0 0; padding: 10px; border-radius: 10px; background: #f4e5bd; }
  .position-notes { display: grid; gap: 12px; }
  .position-notes article { padding-top: 12px; border-top: 1px solid var(--ivory-200); }
  .position-notes article:first-child { padding-top: 0; border-top: 0; }
  .position-notes strong { text-transform: capitalize; }
  .position-notes p { line-height: 1.5; }

  @media (max-width: 980px) {
    .explorer-layout { grid-template-columns: minmax(220px, 0.8fr) minmax(360px, 1.4fr); }
    .explorer-notes { grid-column: 1 / -1; }
  }
  @media (max-width: 720px) {
    .explorer-header { flex-direction: column; }
    .explorer-layout { grid-template-columns: minmax(0, 1fr); }
    .explorer-board { grid-row: 1; }
    .explorer-notes { grid-column: auto; }
  }
</style>
