<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from 'svelte'
  import type { HintResult, SessionView } from '../../lib/api'
  import { useNormalAPI } from '../../lib/api-context'
  import ChessBoard from '../chess/ChessBoard.svelte'

  export let session: SessionView
  const api = useNormalAPI()

  const dispatch = createEventDispatcher<{
    home: { completed: boolean }
    change: SessionView
  }>()
  let shownFen = session.current?.currentFen ?? ''
  let inputDisabled = false
  let statusMessage = ''
  let error = ''
  let hint: HintResult | null = null
  let lastMove: string[] = []
  let wrongMove: string[] = []
  let preludeTimer: ReturnType<typeof setTimeout> | undefined

  $: current = session.current

  function hasPrelude(view: SessionView): boolean {
    return Boolean(view.current?.sourceFen && view.current?.preludeUci)
  }

  function reducedMotion(): boolean {
    return typeof window !== 'undefined' && typeof window.matchMedia === 'function'
      && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  }

  function preparePuzzle(view: SessionView): void {
    clearTimeout(preludeTimer)
    hint = null
    lastMove = []
    wrongMove = []
    statusMessage = ''
    error = ''
    if (!view.current) {
      shownFen = ''
      inputDisabled = true
      return
    }
    if (hasPrelude(view) && !reducedMotion()) {
      shownFen = view.current.sourceFen!
      inputDisabled = true
      statusMessage = 'Watch the last move…'
      preludeTimer = setTimeout(() => {
        shownFen = view.current?.currentFen ?? ''
        lastMove = view.current?.preludeUci?.slice(0, 4).match(/.{2}/g) ?? []
        inputDisabled = false
        statusMessage = ''
      }, 550)
      return
    }
    shownFen = view.current.currentFen
    inputDisabled = false
  }

  function acceptSession(next: SessionView, move = ''): void {
    const oldFingerprint = session.current?.fingerprint
    session = next
    dispatch('change', next)
    if (!next.current) {
      clearTimeout(preludeTimer)
      shownFen = ''
      inputDisabled = true
      return
    }
    if (next.current.fingerprint !== oldFingerprint) {
      preparePuzzle(next)
      return
    }
    shownFen = next.current.currentFen
    lastMove = move.slice(0, 4).match(/.{2}/g) ?? []
    inputDisabled = false
  }

  async function play(event: CustomEvent<{ uci: string }>): Promise<void> {
    if (inputDisabled) return
    const uci = event.detail.uci
    inputDisabled = true
    statusMessage = ''
    error = ''
    wrongMove = []
    try {
      const result = await api.playMove(session.sessionId, uci)
      acceptSession(result.session, result.correct ? uci : '')
      if (!result.correct) {
        wrongMove = uci.slice(0, 4).match(/.{2}/g) ?? []
        statusMessage = result.message ?? 'Try again'
      }
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
      inputDisabled = false
    }
  }

  async function useHint(): Promise<void> {
    inputDisabled = true
    error = ''
    try {
      const result = await api.useHint(session.sessionId)
      acceptSession(result.session)
      hint = result.level > 0 ? result : null
      statusMessage = result.text
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
      inputDisabled = false
    }
  }

  async function reveal(): Promise<void> {
    inputDisabled = true
    error = ''
    try {
      const result = await api.revealSolution(session.sessionId)
      acceptSession(result.session)
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
      inputDisabled = false
    }
  }

  async function pause(): Promise<void> {
    inputDisabled = true
    error = ''
    try {
      await api.pauseSession(session.sessionId)
      dispatch('home', { completed: false })
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
      inputDisabled = false
    }
  }

  onMount(() => preparePuzzle(session))
  onDestroy(() => clearTimeout(preludeTimer))
</script>

{#if session.summary}
  <section class="completion panel" aria-labelledby="completion-title">
    <span class="celebration" aria-hidden="true">♛</span>
    <p class="eyebrow">Nice work</p>
    <h2 id="completion-title">Training complete!</h2>
    <p>You finished {session.summary.total} puzzles.</p>
    <div class="summary-grid">
      <strong>{session.summary.firstTry} first try</strong>
      <strong>{session.summary.retried} retried</strong>
      <strong>{session.summary.usedHint} used a hint</strong>
      <strong>{session.summary.revealed} solution shown</strong>
    </div>
    <button class="primary" type="button" on:click={() => dispatch('home', { completed: true })}>Back home</button>
  </section>
{:else if current}
  <section class="puzzle-layout" aria-label={`Puzzle ${current.puzzleNumber} of ${current.puzzleTotal}`}>
    <ChessBoard
      fen={shownFen}
      orientation={current.solver}
      disabled={inputDisabled}
      {lastMove}
      {wrongMove}
      hintSource={hint?.sourceSquare ?? ''}
      hintTarget={hint?.targetSquare ?? ''}
      on:move={play}
    />

    <aside class="puzzle-panel">
      <div>
        <p class="eyebrow">{current.solver === 'white' ? 'White' : 'Black'} to move</p>
        <h2>Find the best move</h2>
        <p class="progress-label">Puzzle {current.puzzleNumber} of {current.puzzleTotal}</p>
        <div class="progress-track" aria-hidden="true">
          <span style={`width: ${(current.puzzleNumber / current.puzzleTotal) * 100}%`}></span>
        </div>
      </div>

      <div class="puzzle-feedback" aria-live="polite">
        {#if statusMessage}<p class:retry={statusMessage === 'Try again'}>{statusMessage}</p>{/if}
        {#if error}<p class="error">{error}</p>{/if}
      </div>

      <div class="puzzle-actions">
        <button class="primary" type="button" disabled={inputDisabled} on:click={useHint}>Hint</button>
        {#if hint?.canReveal || current.canReveal}
          <button class="secondary" type="button" disabled={inputDisabled} on:click={reveal}>Show solution</button>
        {/if}
        <button class="quiet-action" type="button" disabled={inputDisabled} on:click={pause}>Pause</button>
      </div>
    </aside>
  </section>
{:else}
  <section class="panel" role="alert">
    <h2>Puzzle unavailable</h2>
    <p>This puzzle is no longer in the local library.</p>
    <button class="secondary" type="button" on:click={() => dispatch('home', { completed: false })}>Back home</button>
  </section>
{/if}

<style>
  .puzzle-layout {
    display: grid;
    width: 100%;
    grid-template-columns: minmax(360px, 680px) minmax(230px, 290px);
    gap: clamp(24px, 4vw, 48px);
    align-items: center;
    justify-content: center;
  }
  .puzzle-panel {
    display: flex;
    min-height: 420px;
    padding: 28px;
    flex-direction: column;
    border: 1px solid var(--ivory-200);
    border-radius: var(--radius-large);
    background: var(--ivory-50);
    box-shadow: var(--shadow-soft);
  }
  .puzzle-panel h2 { margin: 8px 0 12px; font-size: clamp(1.7rem, 3vw, 2.25rem); line-height: 1.08; }
  .progress-label { margin: 0 0 8px; color: var(--ink-700); font-weight: 800; }
  .progress-track { height: 9px; overflow: hidden; border-radius: 999px; background: var(--ivory-200); }
  .progress-track span { display: block; height: 100%; border-radius: inherit; background: var(--amber-400); }
  .puzzle-feedback { display: grid; min-height: 112px; place-items: center start; margin-top: auto; }
  .puzzle-feedback p { margin: 0; font-size: 1.08rem; font-weight: 800; }
  .puzzle-feedback .retry { color: var(--red-600); }
  .puzzle-actions { display: grid; gap: 10px; }
  .quiet-action { border: 0; color: var(--ink-700); background: transparent; font-weight: 800; }
  .completion { text-align: center; }
  .celebration { display: block; color: var(--amber-400); font-size: 4.5rem; line-height: 1; }
  .summary-grid { display: grid; margin: 24px 0; grid-template-columns: 1fr 1fr; gap: 10px; }
  .summary-grid strong { padding: 14px 10px; border-radius: 12px; background: #f4e5bd; }

  @media (max-width: 820px) {
    .puzzle-layout { grid-template-columns: minmax(280px, 620px); }
    .puzzle-panel { min-height: auto; }
    .puzzle-feedback { min-height: 72px; }
  }
</style>
