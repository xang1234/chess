<script lang="ts">
  import { afterUpdate, createEventDispatcher, onDestroy, onMount, tick } from 'svelte'
  import type {
    OpeningBoardAnnotation,
    OpeningPathItem,
    OpeningSessionView
  } from '../../lib/api'
  import { useNormalAPI } from '../../lib/api-context'
  import type { Square } from '../../lib/uci'
  import ChessBoard from '../chess/ChessBoard.svelte'
  import { browserBoardEffects, type BoardEffects } from '../chess/board-effects'
  import {
    createChessgroundAdapter,
    type BoardAnnotation,
    type ChessgroundAdapterFactory
  } from '../chess/chessground-adapter'
  import {
    createOpeningController,
    type OpeningControllerView
  } from './opening-controller'
  import { acceptsOpeningInput } from './opening-state'
  import OpeningActivityContent from './OpeningActivityContent.svelte'
  import OpeningPathContext from './OpeningPathContext.svelte'
  import OpeningRoadmapCheckpoint from './OpeningRoadmapCheckpoint.svelte'

  export let session: OpeningSessionView
  export let path: OpeningPathItem[] = []
  export let effects: BoardEffects = browserBoardEffects
  export let boardAdapterFactory: ChessgroundAdapterFactory = createChessgroundAdapter

  const api = useNormalAPI()
  const dispatch = createEventDispatcher<{
    home: { completed: boolean }
    continue: { courseId: string; lessonId: string }
    tree: void
    change: OpeningSessionView
    persisted: OpeningSessionView
  }>()
  const controller = createOpeningController({
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
  let view: OpeningControllerView | undefined
  const unsubscribe = controller.subscribe((next) => { view = next })

  $: controller.attachBoard(boardComponent)
  $: state = view?.state
  $: active = state && state.phase !== 'summary' && state.phase !== 'checkpoint' &&
    state.phase !== 'restart-required'
    ? state.session
    : undefined
  $: current = active?.current
  $: inputEnabled = state ? acceptsOpeningInput(state) : false
  $: boardLegalMoves = inputEnabled ? current?.legalMoves ?? [] : []
  $: hint = state?.phase === 'ready' ? state.hint : null
  $: hintSource = optionalSquare(hint?.sourceSquare)
  $: hintTarget = optionalSquare(hint?.targetSquare)
  $: canReveal = Boolean(hint?.canReveal || current?.canReveal)
  $: annotations = toBoardAnnotations(current?.annotations ?? [])
  $: canReplayDemonstration = state?.phase === 'activity-complete' &&
    current?.kind === 'demonstration' && (view?.lastFrames.length ?? 0) > 0

  afterUpdate(() => controller.receiveSession(session))
  onMount(() => controller.mount(session))
  onDestroy(() => {
    unsubscribe()
    controller.destroy()
  })

  function optionalSquare(value: string | undefined): Square | undefined {
    return value && /^[a-h][1-8]$/.test(value) ? value as Square : undefined
  }

  function toBoardAnnotations(values: OpeningBoardAnnotation[]): BoardAnnotation[] {
    const result: BoardAnnotation[] = []
    for (const annotation of values) {
      const from = optionalSquare(annotation.from)
      if (!from) continue
      if (annotation.kind === 'square') {
        result.push({ kind: 'square', from })
        continue
      }
      const to = optionalSquare(annotation.to)
      if (to) result.push({ kind: 'arrow', from, to })
    }
    return result
  }

  function countLabel(count: number, singular: string, plural = `${singular}s`): string {
    return `${count} ${count === 1 ? singular : plural}`
  }
</script>

<div
  class="opening-lesson-root"
  on:pointerdown|capture={() => controller.startSoundUnlock()}
  on:keydown|capture={(event) => controller.unlockFromKeyboard(event.key)}
>
  {#if state?.phase === 'summary'}
    <section class="completion panel" aria-labelledby="opening-completion-title">
      {#if state.session.notice}<p class="terminal-notice">{state.session.notice}</p>{/if}
      <span class="celebration" aria-hidden="true">♝</span>
      <p class="eyebrow">Italian course</p>
      <h2 id="opening-completion-title">
        {state.session.mode === 'review' ? 'Opening review complete!' : 'Opening lesson complete!'}
      </h2>
      {#if state.session.mode === 'review'}
        <p>You worked through {countLabel(state.session.summary.totalPrompts, 'training prompt')}.</p>
        <div class="summary-grid">
          <strong>{countLabel(state.session.summary.positionsRecalled, 'position recalled', 'positions recalled')}</strong>
          <strong>{countLabel(state.session.summary.branchesRecognized, 'branch recognized')}</strong>
          <strong>{countLabel(state.session.summary.retried, 'retry', 'retries')}</strong>
          <strong>{countLabel(state.session.summary.usedHint, 'hint used', 'hints used')}</strong>
          <strong>{countLabel(state.session.summary.revealed, 'course move shown')}</strong>
        </div>
      {:else}
        <p>Your place in the course has been saved.</p>
      {/if}
      <button class="primary" type="button" on:click={() => dispatch('tree')}>
        Back to course
      </button>
    </section>
  {:else if state?.phase === 'checkpoint'}
    <OpeningRoadmapCheckpoint
      checkpoint={state.checkpoint}
      courseId={state.session.courseId}
      on:continue
      on:tree
      on:home
    />
  {:else if state?.phase === 'restart-required'}
    <section class="panel opening-terminal" aria-labelledby="opening-restart-title">
      <p class="eyebrow">Course update</p>
      <h2 id="opening-restart-title">Restart this lesson safely</h2>
      <p>{state.session.notice}</p>
      {#if view?.message}<p aria-live="polite">{view.message}</p>{/if}
      <button class="primary" type="button" on:click={() => controller.restart()}>
        Restart from checkpoint
      </button>
    </section>
  {:else if state && current && view}
    <section
      class="opening-lesson-layout"
      aria-label={`Opening lesson idea ${current.activityNumber} of ${current.activityTotal}`}
    >
      <div class="opening-board-stage">
        {#key view.boardGeneration}
          <ChessBoard
            bind:this={boardComponent}
            fen={state.fen}
            orientation={current.orientation}
            legalMoves={boardLegalMoves}
            {inputEnabled}
            lastMove={view.lastMove}
            {hintSource}
            {hintTarget}
            {annotations}
            reducedMotion={view.reducedMotion}
            adapterFactory={boardAdapterFactory}
            on:move={(event) => controller.play(event.detail.uci)}
            on:error={(event) => controller.handleBoardError(event.detail.message)}
            on:announce={(event) => controller.announce(event.detail.message)}
          />
        {/key}
      </div>

      <aside class="opening-lesson-panel">
        <OpeningPathContext {path} />
        <div class="opening-lesson-heading">
          <div>
            <p class="eyebrow">
              Opening course{current.variationName ? ` · ${current.variationName}` : ''}
            </p>
            <h2>{current.title}</h2>
          </div>
          <button
            class="sound-toggle"
            type="button"
            aria-label={view.soundMuted ? 'Turn sound on' : 'Mute sounds'}
            aria-pressed={view.soundMuted}
            on:click={() => controller.toggleSound()}
          >
            <span aria-hidden="true">{view.soundMuted ? '🔇' : '🔊'}</span>
          </button>
        </div>

        <div>
          <p class="progress-label">
            Idea {current.activityNumber} of {current.activityTotal}
            · {current.completedIdeas} learned
          </p>
          <div class="progress-track" aria-hidden="true">
            <span style={`width: ${(current.activityNumber / current.activityTotal) * 100}%`}></span>
          </div>
        </div>

        <OpeningActivityContent
          activity={current}
          {canReplayDemonstration}
          on:replayMoves={() => controller.replayMovesToHere()}
          on:replayDemonstration={() => controller.replayDemonstration()}
        />

        <div class="opening-feedback" aria-live="polite" aria-atomic="true">
          {#if view.message}<p class:neutral={view.feedback !== null}>{view.message}</p>{/if}
          {#if view.notice}<p class="notice">{view.notice}</p>{/if}
          {#if view.announcement && view.announcement !== view.message && view.announcement !== view.notice}
            <p class="visually-hidden">{view.announcement}</p>
          {/if}
        </div>

        <div class="opening-actions">
          {#if state.phase === 'activity-complete'}
            <button class="primary" type="button" on:click={() => controller.acknowledgeActivity()}>
              Continue
            </button>
          {:else if state.phase === 'failed' && state.recoverable && state.retryOperation}
            <button class="primary" type="button" on:click={() => controller.retry()}>
              Retry
            </button>
            {#if state.retryOperation !== 'pause'}
              <button class="quiet-action" type="button" on:click={() => controller.pause()}>
                Pause lesson
              </button>
            {/if}
          {:else if state.phase === 'passive'}
            <button class="primary" type="button" on:click={() => controller.advance()}>
              Continue
            </button>
            <button class="quiet-action" type="button" on:click={() => controller.pause()}>
              Pause lesson
            </button>
          {:else}
            <button class="primary" type="button" disabled={!inputEnabled} on:click={() => controller.useHint()}>
              Hint
            </button>
            {#if canReveal}
              <button class="secondary" type="button" disabled={!inputEnabled} on:click={() => controller.reveal()}>
                Show course move
              </button>
            {/if}
            <button class="quiet-action" type="button" disabled={!inputEnabled} on:click={() => controller.pause()}>
              Pause lesson
            </button>
          {/if}
        </div>
      </aside>
    </section>
  {:else}
    <p class="loading" aria-live="polite">Preparing the opening lesson…</p>
  {/if}
</div>

<style>
  .opening-lesson-root { width: 100%; }
  .opening-lesson-layout {
    display: grid;
    width: 100%;
    grid-template-columns: minmax(360px, 760px) minmax(310px, 370px);
    gap: clamp(22px, 3vw, 42px);
    align-items: center;
    justify-content: center;
  }
  .opening-board-stage { min-width: 0; }
  .opening-lesson-panel {
    display: flex;
    min-height: 520px;
    padding: 26px;
    flex-direction: column;
    gap: 20px;
    border: 1px solid #3b4843;
    border-radius: var(--radius-large);
    color: #f7f4e9;
    background: var(--charcoal-800);
    box-shadow: var(--shadow-soft);
  }
  .opening-lesson-panel .eyebrow { color: #a9d2b6; }
  .opening-lesson-heading { display: flex; gap: 16px; align-items: flex-start; justify-content: space-between; }
  .opening-lesson-panel h2 { margin: 7px 0 0; font-size: clamp(1.55rem, 3vw, 2.1rem); line-height: 1.08; }
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
  .opening-feedback { display: grid; min-height: 72px; margin-top: auto; align-content: center; gap: 7px; }
  .opening-feedback p { margin: 0; font-weight: 800; }
  .opening-feedback .neutral { color: #c8dfcf; }
  .opening-feedback .notice { color: #f4d27b; font-size: 0.94rem; }
  .opening-actions { display: grid; gap: 10px; }
  .opening-actions button { min-height: 46px; }
  .quiet-action { border: 0; color: #e8e4d8; background: transparent; font-weight: 800; }
  .opening-terminal { max-width: 680px; text-align: center; }
  .completion { color: var(--ink-900); text-align: center; }
  .completion .eyebrow { color: var(--forest-600); }
  .celebration { display: block; color: var(--amber-400); font-size: 4.5rem; line-height: 1; }
  .summary-grid { display: grid; margin: 24px 0; grid-template-columns: repeat(2, 1fr); gap: 10px; }
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

  @media (max-width: 860px) {
    .opening-lesson-layout { grid-template-columns: minmax(280px, 1fr); }
    .opening-lesson-panel { min-height: auto; }
  }
</style>
