<script lang="ts">
  import { afterUpdate, createEventDispatcher, onDestroy, onMount, tick } from 'svelte'
  import type { SessionView } from '../../lib/api'
  import { useNormalAPI } from '../../lib/api-context'
  import ChessBoard from '../chess/ChessBoard.svelte'
  import {
    createChessgroundAdapter,
    type ChessgroundAdapterFactory
  } from '../chess/chessground-adapter'
  import { createPuzzleController } from './puzzle-controller'
  import { browserPuzzleEffects, type PuzzleEffects } from './puzzle-effects'
  import { acceptsInput, feedbackMessage, optionalSquare } from './puzzle-view'

  export let session: SessionView
  export let effects: PuzzleEffects = browserPuzzleEffects
  export let boardAdapterFactory: ChessgroundAdapterFactory = createChessgroundAdapter

  const api = useNormalAPI()
  const dispatch = createEventDispatcher<{
    home: { completed: boolean }
    change: SessionView
    persisted: SessionView
  }>()
  const controller = createPuzzleController({
    api,
    effects,
    afterRender: tick,
    events: {
      home: (completed) => dispatch('home', { completed }),
      change: (next) => dispatch('change', next),
      persisted: (next) => dispatch('persisted', next)
    }
  })

  let boardComponent: ChessBoard | undefined

  $: controller.attachBoard(boardComponent)
  $: puzzle = $controller.kind === 'puzzle' ? $controller : undefined
  $: summary = $controller.kind === 'summary' ? $controller : undefined
  $: state = puzzle?.state
  $: current = state?.displaySession.current
  $: inputEnabled = state ? acceptsInput(state) : false
  $: boardLegalMoves = state?.phase === 'solved' ? [] : current?.legalMoves ?? []
  $: boardWrongMove = state?.phase === 'incorrect' ? state.wrongMove : undefined
  $: boardHintSource = optionalSquare(state?.hint?.sourceSquare)
  $: boardHintTarget = optionalSquare(state?.hint?.targetSquare)
  $: feedback = state ? feedbackMessage(state) : ''
  $: solvedAction = state?.phase === 'solved'
    ? state.pendingSession.status === 'active' ? 'Next puzzle' : 'See results'
    : ''

  afterUpdate(() => controller.receiveSession(session))
  onMount(() => controller.mount(session))
  onDestroy(() => controller.destroy())
</script>

<div
  class="puzzle-root"
  on:pointerdown|capture={() => controller.startSoundUnlock()}
  on:keydown|capture={(event) => controller.unlockFromKeyboard(event.key)}
>
  {#if summary}
    <section class="completion panel" aria-labelledby="completion-title">
      {#if summary.notice}<p class="terminal-notice">{summary.notice}</p>{/if}
      <span class="celebration" aria-hidden="true">♛</span>
      <p class="eyebrow">Nice work</p>
      <h2 id="completion-title">Training complete!</h2>
      <p>You finished {summary.session.summary.total} puzzles.</p>
      <div class="summary-grid">
        <strong>{summary.session.summary.firstTry} first try</strong>
        <strong>{summary.session.summary.retried} retried</strong>
        <strong>{summary.session.summary.usedHint} used a hint</strong>
        <strong>{summary.session.summary.revealed} solution shown</strong>
      </div>
      <button class="primary" type="button" on:click={() => controller.finishHome()}>
        Back home
      </button>
    </section>
  {:else if puzzle && state && current}
    <section class="puzzle-layout" aria-label={`Puzzle ${current.puzzleNumber} of ${current.puzzleTotal}`}>
      {#key puzzle.boardGeneration}
        <ChessBoard
          bind:this={boardComponent}
          fen={state.fen}
          orientation={current.solver}
          legalMoves={boardLegalMoves}
          {inputEnabled}
          lastMove={state.lastMove}
          wrongMove={boardWrongMove}
          hintSource={boardHintSource}
          hintTarget={boardHintTarget}
          reducedMotion={puzzle.reducedMotion}
          adapterFactory={boardAdapterFactory}
          on:move={(event) => controller.play(event.detail.uci)}
          on:error={(event) => controller.handleBoardError(event.detail.message)}
          on:announce={(event) => controller.announce(event.detail.message)}
        />
      {/key}

      <aside class="puzzle-panel">
        <div class="puzzle-heading">
          <div>
            <p class="eyebrow">{current.solver === 'white' ? 'White' : 'Black'} to move</p>
            <h2>Find the best move</h2>
          </div>
          <button
            class="sound-toggle"
            type="button"
            aria-label={puzzle.soundMuted ? 'Turn sound on' : 'Mute sounds'}
            aria-pressed={puzzle.soundMuted}
            on:click={() => controller.toggleSound()}
          >
            <span aria-hidden="true">{puzzle.soundMuted ? '🔇' : '🔊'}</span>
          </button>
        </div>

        <div>
          <p class="progress-label">Puzzle {current.puzzleNumber} of {current.puzzleTotal}</p>
          <div class="progress-track" aria-hidden="true">
            <span style={`width: ${(current.puzzleNumber / current.puzzleTotal) * 100}%`}></span>
          </div>
        </div>

        <div class="puzzle-feedback" aria-live="polite" aria-atomic="true">
          {#if feedback}
            <p class:retry={state.phase === 'incorrect'} class:error={state.phase === 'failed'}>
              {feedback}
            </p>
          {/if}
          {#if state.notice}<p class="notice">{state.notice}</p>{/if}
          {#if puzzle.announcement && puzzle.announcement !== feedback && puzzle.announcement !== state.notice}
            <p class="visually-hidden">{puzzle.announcement}</p>
          {/if}
        </div>

        <div class="puzzle-actions">
          {#if state.phase === 'solved'}
            <button class="primary next-action" type="button" on:click={() => controller.acknowledgeSolution()}>
              {solvedAction}
            </button>
          {:else}
            <button class="primary" type="button" disabled={!inputEnabled} on:click={() => controller.useHint()}>Hint</button>
            {#if state.hint?.canReveal || current.canReveal}
              <button class="secondary" type="button" disabled={!inputEnabled} on:click={() => controller.reveal()}>
                Show solution
              </button>
            {/if}
            <button class="quiet-action" type="button" disabled={!inputEnabled} on:click={() => controller.pause()}>Pause</button>
          {/if}
        </div>
      </aside>
    </section>
  {:else}
    <p class="loading" aria-live="polite">Preparing the puzzle…</p>
  {/if}
</div>

<style>
  .puzzle-root { width: 100%; }
  .puzzle-layout {
    display: grid;
    width: 100%;
    grid-template-columns: minmax(360px, 760px) minmax(300px, 340px);
    gap: clamp(22px, 3vw, 42px);
    align-items: center;
    justify-content: center;
  }
  .puzzle-panel {
    display: flex;
    min-height: 440px;
    padding: 26px;
    flex-direction: column;
    gap: 22px;
    border: 1px solid #3b4843;
    border-radius: var(--radius-large);
    color: #f7f4e9;
    background: var(--charcoal-800);
    box-shadow: var(--shadow-soft);
  }
  .puzzle-panel .eyebrow { color: #a9d2b6; }
  .puzzle-heading { display: flex; gap: 16px; align-items: flex-start; justify-content: space-between; }
  .puzzle-panel h2 { margin: 7px 0 0; font-size: clamp(1.65rem, 3vw, 2.2rem); line-height: 1.08; }
  .sound-toggle {
    width: 46px;
    min-width: 46px;
    padding: 0;
    border: 1px solid #66746e;
    border-radius: 12px;
    color: inherit;
    background: var(--charcoal-700);
  }
  .progress-label { margin: 0 0 8px; color: #e8e4d8; font-weight: 800; }
  .progress-track { height: 9px; overflow: hidden; border-radius: 999px; background: #4c5a54; }
  .progress-track span { display: block; height: 100%; border-radius: inherit; background: var(--amber-400); }
  .puzzle-feedback { display: grid; min-height: 116px; margin-top: auto; align-content: center; gap: 8px; }
  .puzzle-feedback p { margin: 0; font-size: 1.08rem; font-weight: 800; }
  .puzzle-feedback .retry,
  .puzzle-feedback .error { color: #ff9d91; }
  .puzzle-feedback .notice { color: #f4d27b; font-size: 0.94rem; }
  .puzzle-actions { display: grid; gap: 10px; }
  .puzzle-actions button { min-height: 46px; }
  .quiet-action { border: 0; color: #e8e4d8; background: transparent; font-weight: 800; }
  .next-action { min-height: 52px !important; font-size: 1.06rem; }
  .completion { text-align: center; }
  .celebration { display: block; color: var(--amber-400); font-size: 4.5rem; line-height: 1; }
  .summary-grid { display: grid; margin: 24px 0; grid-template-columns: 1fr 1fr; gap: 10px; }
  .summary-grid strong { padding: 14px 10px; border-radius: 12px; background: #f4e5bd; }
  .terminal-notice { color: var(--red-600); font-weight: 800; }
  .visually-hidden {
    position: absolute !important;
    width: 1px !important;
    height: 1px !important;
    padding: 0 !important;
    overflow: hidden !important;
    border: 0 !important;
    clip: rect(0 0 0 0) !important;
    clip-path: inset(50%) !important;
    white-space: nowrap !important;
  }

  @media (max-width: 820px) {
    .puzzle-layout { grid-template-columns: minmax(280px, 1fr); }
    .puzzle-panel { min-height: auto; }
    .puzzle-feedback { min-height: 76px; }
  }
</style>
