<script context="module" lang="ts">
  import { createSoundService, type SoundService } from '../../lib/sound'

  export type PuzzleEffects = {
    createSound(): SoundService
    delay(milliseconds: number, signal: AbortSignal): Promise<void>
    prefersReducedMotion(): boolean
  }

  function browserDelay(milliseconds: number, signal: AbortSignal): Promise<void> {
    if (signal.aborted) return Promise.resolve()
    return new Promise((resolve) => {
      const timer = window.setTimeout(finish, milliseconds)
      signal.addEventListener('abort', finish, { once: true })

      function finish(): void {
        window.clearTimeout(timer)
        signal.removeEventListener('abort', finish)
        resolve()
      }
    })
  }

  export const browserPuzzleEffects: PuzzleEffects = {
    createSound: createSoundService,
    delay: browserDelay,
    prefersReducedMotion: () => typeof window !== 'undefined' &&
      typeof window.matchMedia === 'function' &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches
  }
</script>

<script lang="ts">
  import { afterUpdate, createEventDispatcher, onDestroy, onMount, tick } from 'svelte'
  import type {
    AppliedMove,
    HintResult,
    MoveResult,
    PuzzleView,
    SessionView
  } from '../../lib/api'
  import { useNormalAPI } from '../../lib/api-context'
  import { parseFEN } from '../../lib/fen'
  import { groupLegalMoves, moveSquares, parseUCI, type Square } from '../../lib/uci'
  import ChessBoard from '../chess/ChessBoard.svelte'
  import {
    createChessgroundAdapter,
    type ChessgroundAdapterFactory
  } from '../chess/chessground-adapter'
  import { animateAppliedMoves, type PositionFrame } from './move-animation'
  import {
    acceptsResponse,
    acknowledgeSolved,
    beginAnimation,
    beginRequest,
    finishPrelude,
    finishReadyRequest,
    initialisePuzzle,
    markFailed,
    markIncorrect,
    markSolved,
    type Operation,
    type PuzzleState,
    type SolvedOutcome
  } from './puzzle-state'

  export let session: SessionView
  export let effects: PuzzleEffects = browserPuzzleEffects
  export let boardAdapterFactory: ChessgroundAdapterFactory = createChessgroundAdapter

  const api = useNormalAPI()
  const dispatch = createEventDispatcher<{
    home: { completed: boolean }
    change: SessionView
    persisted: SessionView
  }>()

  type OwnedRequest = {
    controller: AbortController
    id: number
    request: Extract<PuzzleState, { phase: 'requesting' }>
  }

  let state: PuzzleState | null = null
  let terminalSession: SessionView | null = null
  let terminalNotice = ''
  let boardComponent: ChessBoard
  let boardGeneration = 0
  let abortController: AbortController | undefined
  let requestSequence = 0
  let mounted = false
  let observedSession = session
  let reducedMotion = false
  let sound: SoundService | undefined
  let soundMuted = false
  let soundUnlockStarted = false
  let announcement = ''
  let recoveringBoard = false
  let pendingBoardWarning = ''

  $: current = state?.displaySession.current
  $: summary = terminalSession?.summary
  $: inputEnabled = acceptsInput(state)
  $: boardLegalMoves = state?.phase === 'solved' ? [] : current?.legalMoves ?? []
  $: boardLastMove = state?.lastMove
  $: boardWrongMove = state?.phase === 'incorrect' ? state.wrongMove : undefined
  $: boardHintSource = optionalSquare(state?.hint?.sourceSquare)
  $: boardHintTarget = optionalSquare(state?.hint?.targetSquare)
  $: feedback = feedbackMessage(state)
  $: solvedAction = state?.phase === 'solved'
    ? state.pendingSession.current ? 'Next puzzle' : 'See results'
    : ''

  afterUpdate(() => {
    if (!mounted || session === observedSession) return
    observedSession = session
    if (session !== visibleSession()) adoptVisibleSession(session)
  })

  function visibleSession(): SessionView | null {
    return state?.displaySession ?? terminalSession
  }

  function acceptsInput(value: PuzzleState | null): boolean {
    return value?.phase === 'ready' || value?.phase === 'incorrect' ||
      (value?.phase === 'failed' && value.recoverable)
  }

  function feedbackMessage(value: PuzzleState | null): string {
    if (!value) return ''
    switch (value.phase) {
      case 'prelude': return 'Watch the last move…'
      case 'ready': return value.hint?.text ?? ''
      case 'requesting': return requestMessage(value.operation)
      case 'animating': return 'Good move…'
      case 'incorrect': return value.message
      case 'solved': return value.outcome === 'correct' ? 'Correct!' : 'Solution shown'
      case 'failed': return value.message
    }
  }

  function requestMessage(operation: Operation): string {
    switch (operation) {
      case 'move': return 'Checking that move…'
      case 'hint': return 'Finding a hint…'
      case 'reveal': return 'Preparing the solution…'
      case 'pause': return 'Pausing…'
    }
  }

  function startOwnedWork(): { controller: AbortController; id: number } {
    abortController?.abort()
    requestSequence += 1
    const controller = new AbortController()
    abortController = controller
    return { controller, id: requestSequence }
  }

  function cancelOwnedWork(): void {
    abortController?.abort()
    abortController = undefined
    requestSequence += 1
  }

  function isCurrent(controller: AbortController, id: number): boolean {
    return mounted && !controller.signal.aborted && abortController === controller &&
      requestSequence === id
  }

  function beginOwnedRequest(operation: Operation, submittedUci?: string): OwnedRequest | null {
    if (!state || !acceptsInput(state)) return null
    const { controller, id } = startOwnedWork()
    try {
      let request = beginRequest(state, operation, id, submittedUci)
      if (request.phase !== 'requesting') throw new Error('Puzzle request did not start.')
      if (operation === 'move' || operation === 'reveal') request = { ...request, hint: null }
      state = request
      announcement = requestMessage(operation)
      return { controller, id, request }
    } catch (error) {
      failFatal(errorMessage(error))
      return null
    }
  }

  function acceptsOwnedResponse(owned: OwnedRequest): boolean {
    return isCurrent(owned.controller, owned.id) && Boolean(state) &&
      acceptsResponse(state!, owned.id)
  }

  function adoptVisibleSession(next: SessionView, notice = '', notify = false): void {
    cancelOwnedWork()
    terminalNotice = notice
    announcement = notice
    boardGeneration += 1
    pendingBoardWarning = ''
    recoveringBoard = false

    if (!next.current) {
      state = null
      terminalSession = next
      if (notify) dispatch('change', next)
      return
    }

    terminalSession = null
    try {
      validatePuzzle(next.current)
      state = initialisePuzzle(next, reducedMotion)
      if (notice) state = { ...state, notice }
    } catch (error) {
      const initial = initialisePuzzle(next, true)
      state = markFailed(initial, errorMessage(error), false)
    }
    if (notify) dispatch('change', next)
    if (state.phase === 'prelude') void playPrelude()
  }

  async function playPrelude(): Promise<void> {
    if (!state || state.phase !== 'prelude' || !state.displaySession.current?.preludeUci) return
    const { controller, id } = startOwnedWork()
    await tick()
    if (!isCurrent(controller, id) || !state || state.phase !== 'prelude') return
    const currentPuzzle = state.displaySession.current
    const result = await animateAppliedMoves({
      port: animationPort(),
      startingFen: state.fen,
      appliedMoves: [{
        uci: currentPuzzle.preludeUci,
        resultingFen: currentPuzzle.displayedFen
      }],
      finalFen: currentPuzzle.displayedFen,
      reducedMotion: false,
      signal: controller.signal,
      onStep: (kind) => sound?.play(kind)
    })
    if (!isCurrent(controller, id) || result.status === 'aborted' ||
      !state || state.phase !== 'prelude') return
    try {
      state = finishPrelude(state)
      applyBoardWarnings(result.warning)
      announcement = 'The puzzle is ready.'
    } catch (error) {
      failFatal(errorMessage(error))
    }
  }

  async function play(event: CustomEvent<{ uci: string }>): Promise<void> {
    const uci = event.detail.uci
    const owned = beginOwnedRequest('move', uci)
    if (!owned) return

    let result: MoveResult
    try {
      result = await api.playMove(owned.request.displaySession.sessionId, uci)
    } catch (error) {
      await failRequest(owned, error, true)
      return
    }
    if (!acceptsOwnedResponse(owned)) return

    if (!result.correct && responseAdvanced(owned.request.displaySession, result.session)) {
      adoptVisibleSession(result.session, result.message || 'Puzzle unavailable', true)
      return
    }
    if (!result.correct) {
      await finishIncorrect(owned, result, uci)
      return
    }
    await finishSuccessful(owned, result, 'correct', uci)
  }

  async function finishIncorrect(
    owned: OwnedRequest,
    result: MoveResult,
    uci: string
  ): Promise<void> {
    try {
      if (!result.session.current) throw new Error('Incorrect response has no current puzzle.')
      validatePuzzle(result.session.current)
    } catch (error) {
      failFatal(errorMessage(error))
      return
    }

    const warning = await reconcilePosition(
      owned.request.authoritativeFen,
      owned.controller,
      !reducedMotion
    )
    if (!acceptsOwnedResponse(owned)) return
    try {
      state = markIncorrect(
        state!,
        owned.id,
        result.session,
        uci,
        result.message || 'Try again'
      )
      applyBoardWarnings(warning)
      sound?.play('incorrect')
      announcement = result.message || 'Try again'
      dispatch('change', result.session)
    } catch (error) {
      failFatal(errorMessage(error))
    }
  }

  async function finishSuccessful(
    owned: OwnedRequest,
    result: MoveResult,
    outcome: SolvedOutcome,
    optimisticUci?: string
  ): Promise<void> {
    let frames: AppliedMove[]
    let target: string
    let readyAfter: PuzzleState | undefined
    try {
      frames = validateSuccessfulResult(result, owned.request, optimisticUci)
      target = result.puzzleCompleted ? result.finalFen! : result.session.current!.currentFen
      if (!result.puzzleCompleted) {
        readyAfter = finishReadyRequest(
          owned.request,
          owned.id,
          result.session,
          null
        )
      }
      state = beginAnimation(state!, owned.id)
    } catch (error) {
      failFatal(errorMessage(error))
      return
    }

    const animation = await animateAppliedMoves({
      port: animationPort(),
      startingFen: owned.request.authoritativeFen,
      appliedMoves: frames,
      optimisticUci,
      finalFen: target,
      reducedMotion,
      signal: owned.controller.signal,
      onStep: (kind) => sound?.play(kind)
    })
    if (!isCurrent(owned.controller, owned.id) || animation.status === 'aborted' ||
      !state || state.phase !== 'animating' || state.requestId !== owned.id) return

    const lastMove = frames.length > 0 ? moveSquares(frames[frames.length - 1].uci) : undefined
    try {
      if (result.puzzleCompleted) {
        state = markSolved(state, owned.id, outcome, target, result.session, lastMove)
        applyBoardWarnings(animation.warning)
        announcement = outcome === 'correct' ? 'Correct!' : 'Solution shown'
        if (outcome === 'correct') sound?.play('correct')
        dispatch('persisted', result.session)
      } else {
        state = {
          ...readyAfter!,
          ...(lastMove ? { lastMove } : {}),
          ...(animation.warning ? { notice: animation.warning } : {})
        }
        applyBoardWarnings()
        announcement = 'Good move. Find the next move.'
        dispatch('change', result.session)
      }
    } catch (error) {
      failFatal(errorMessage(error))
    }
  }

  async function useHint(): Promise<void> {
    const owned = beginOwnedRequest('hint')
    if (!owned) return
    let result: HintResult
    try {
      result = await api.useHint(owned.request.displaySession.sessionId)
    } catch (error) {
      await failRequest(owned, error, false)
      return
    }
    if (!acceptsOwnedResponse(owned)) return
    if (responseAdvanced(owned.request.displaySession, result.session)) {
      adoptVisibleSession(result.session, result.text || 'Puzzle unavailable', true)
      return
    }
    try {
      if (!result.session.current) throw new Error('Hint response has no current puzzle.')
      validatePuzzle(result.session.current)
      state = finishReadyRequest(
        state!,
        owned.id,
        result.session,
        result.level > 0 ? result : null
      )
      announcement = result.text
      dispatch('change', result.session)
    } catch (error) {
      failFatal(errorMessage(error))
    }
  }

  async function reveal(): Promise<void> {
    const owned = beginOwnedRequest('reveal')
    if (!owned) return
    let result: MoveResult
    try {
      result = await api.revealSolution(owned.request.displaySession.sessionId)
    } catch (error) {
      await failRequest(owned, error, false)
      return
    }
    if (!acceptsOwnedResponse(owned)) return
    if (!result.correct && responseAdvanced(owned.request.displaySession, result.session)) {
      adoptVisibleSession(result.session, result.message || 'Puzzle unavailable', true)
      return
    }
    if (!result.correct || !result.puzzleCompleted) {
      failFatal('Reveal response did not include a completed authoritative solution.')
      return
    }
    await finishSuccessful(owned, result, 'revealed')
  }

  async function pause(): Promise<void> {
    const owned = beginOwnedRequest('pause')
    if (!owned) return
    try {
      await api.pauseSession(owned.request.displaySession.sessionId)
    } catch (error) {
      await failRequest(owned, error, false)
      return
    }
    if (!acceptsOwnedResponse(owned)) return
    announcement = 'Training paused.'
    cancelOwnedWork()
    await tick()
    if (mounted) dispatch('home', { completed: false })
  }

  async function failRequest(
    owned: OwnedRequest,
    error: unknown,
    reconcile: boolean
  ): Promise<void> {
    if (!acceptsOwnedResponse(owned)) return
    let warning = ''
    if (reconcile) {
      warning = await reconcilePosition(
        owned.request.authoritativeFen,
        owned.controller,
        !reducedMotion
      )
    }
    if (!acceptsOwnedResponse(owned)) return
    const message = errorMessage(error)
    state = markFailed(state!, message, true)
    applyBoardWarnings(warning)
    announcement = message
  }

  async function reconcilePosition(
    fen: string,
    controller: AbortController,
    animate: boolean
  ): Promise<string> {
    try {
      if (!state || !boardComponent) throw new Error('Chess board is unavailable')
      state = { ...state, fen, lastMove: undefined }
      boardComponent.setPosition(fen, undefined, animate)
      if (animate) await effects.delay(180, controller.signal)
      return ''
    } catch (error) {
      if (controller.signal.aborted) return ''
      recoverBoard(fen)
      return `Board reconciliation failed: ${errorMessage(error)}. The saved position was restored.`
    }
  }

  function animationPort() {
    return {
      setPosition: (frame: PositionFrame) => {
        if (!state || !boardComponent) throw new Error('Chess board is unavailable')
        state = { ...state, fen: frame.fen, lastMove: frame.lastMove }
        boardComponent.setPosition(frame.fen, frame.lastMove, frame.animate)
      },
      delay: effects.delay,
      recover: recoverBoard
    }
  }

  function recoverBoard(finalFen: string): void {
    if (!state) throw new Error('Puzzle state is unavailable')
    state = { ...state, fen: finalFen, lastMove: undefined }
    recoveringBoard = true
    boardGeneration += 1
  }

  function applyBoardWarnings(warning = ''): void {
    const messages = [warning, pendingBoardWarning].filter(Boolean)
    pendingBoardWarning = ''
    recoveringBoard = false
    if (messages.length > 0 && state) {
      state = { ...state, notice: messages.join(' ') }
    }
  }

  function validateSuccessfulResult(
    result: MoveResult,
    request: Extract<PuzzleState, { phase: 'requesting' }>,
    optimisticUci?: string
  ): AppliedMove[] {
    if (!result.appliedMoves || result.appliedMoves.length === 0) {
      throw new Error('Successful puzzle response is missing authoritative move frames.')
    }
    for (const frame of result.appliedMoves) {
      parseUCI(frame.uci)
      if (!frame.resultingFen) throw new Error(`Move ${frame.uci} has no authoritative FEN.`)
      parseFEN(frame.resultingFen)
    }
    if (optimisticUci && result.appliedMoves[0].uci !== optimisticUci) {
      throw new Error('Authoritative move frames do not begin with the submitted move.')
    }
    if (result.puzzleCompleted) {
      if (!result.finalFen) throw new Error('Completed puzzle response has no final FEN.')
      parseFEN(result.finalFen)
      validatePendingSession(result.session)
    } else {
      if (responseAdvanced(request.displaySession, result.session) || !result.session.current) {
        throw new Error('Incomplete correct response advanced to a different puzzle.')
      }
      validatePuzzle(result.session.current)
      parseFEN(result.session.current.currentFen)
    }
    return [...result.appliedMoves]
  }

  function validatePendingSession(next: SessionView): void {
    if (next.current) {
      validatePuzzle(next.current)
      return
    }
    if (!next.summary) throw new Error('Completed response has neither a next puzzle nor results.')
  }

  function validatePuzzle(puzzle: PuzzleView): void {
    try {
      groupLegalMoves(puzzle.legalMoves)
    } catch (error) {
      throw new Error(`Invalid legal move data: ${errorMessage(error)}. Puzzle input is locked.`)
    }
    parseFEN(puzzle.currentFen)
    parseFEN(puzzle.displayedFen)
    if (puzzle.sourceFen) parseFEN(puzzle.sourceFen)
    if (puzzle.preludeUci) parseUCI(puzzle.preludeUci)
  }

  function responseAdvanced(before: SessionView, after: SessionView): boolean {
    return before.sessionId !== after.sessionId || before.currentIndex !== after.currentIndex ||
      !before.current || !after.current ||
      before.current.fingerprint !== after.current.fingerprint
  }

  function failFatal(message: string): void {
    announcement = message
    if (!state) return
    if (state.phase === 'solved') {
      state = { ...state, notice: message }
      return
    }
    state = markFailed(state, message, false)
  }

  function handleBoardError(event: CustomEvent<{ message: string }>): void {
    const message = event.detail.message
    if (recoveringBoard) {
      pendingBoardWarning = message
      return
    }
    failFatal(message)
  }

  function acknowledgeSolution(): void {
    if (!state || state.phase !== 'solved') return
    try {
      const acknowledged = acknowledgeSolved(state, reducedMotion)
      cancelOwnedWork()
      terminalNotice = ''
      announcement = acknowledged.kind === 'puzzle'
        ? 'Next puzzle.'
        : 'Training results.'
      boardGeneration += 1
      if (acknowledged.kind === 'puzzle') {
        state = acknowledged.state
        terminalSession = null
        dispatch('change', state.displaySession)
        if (state.phase === 'prelude') void playPrelude()
      } else {
        state = null
        terminalSession = acknowledged.session
        dispatch('change', acknowledged.session)
      }
    } catch (error) {
      failFatal(errorMessage(error))
    }
  }

  function startSoundUnlock(): void {
    if (!sound || soundUnlockStarted) return
    soundUnlockStarted = true
    try {
      void sound.unlock().catch(() => { soundUnlockStarted = false })
    } catch {
      soundUnlockStarted = false
    }
  }

  function unlockFromKeyboard(event: KeyboardEvent): void {
    if (event.key === 'Enter' || event.key === ' ') startSoundUnlock()
  }

  function toggleSound(): void {
    if (!sound) return
    soundMuted = sound.toggleMuted()
  }

  function optionalSquare(value: string | undefined): Square | undefined {
    return value && /^[a-h][1-8]$/.test(value) ? value as Square : undefined
  }

  function errorMessage(error: unknown): string {
    return error instanceof Error ? error.message : String(error)
  }

  onMount(() => {
    mounted = true
    reducedMotion = effects.prefersReducedMotion()
    sound = effects.createSound()
    soundMuted = sound.muted
    observedSession = session
    adoptVisibleSession(session)
  })

  onDestroy(() => {
    mounted = false
    cancelOwnedWork()
    sound?.destroy()
    sound = undefined
  })
</script>

<div
  class="puzzle-root"
  on:pointerdown|capture={startSoundUnlock}
  on:keydown|capture={unlockFromKeyboard}
>
  {#if summary}
    <section class="completion panel" aria-labelledby="completion-title">
      {#if terminalNotice}<p class="terminal-notice">{terminalNotice}</p>{/if}
      <span class="celebration" aria-hidden="true">♛</span>
      <p class="eyebrow">Nice work</p>
      <h2 id="completion-title">Training complete!</h2>
      <p>You finished {summary.total} puzzles.</p>
      <div class="summary-grid">
        <strong>{summary.firstTry} first try</strong>
        <strong>{summary.retried} retried</strong>
        <strong>{summary.usedHint} used a hint</strong>
        <strong>{summary.revealed} solution shown</strong>
      </div>
      <button class="primary" type="button" on:click={() => dispatch('home', { completed: true })}>
        Back home
      </button>
    </section>
  {:else if state && current}
    <section class="puzzle-layout" aria-label={`Puzzle ${current.puzzleNumber} of ${current.puzzleTotal}`}>
      {#key boardGeneration}
        <ChessBoard
          bind:this={boardComponent}
          fen={state.fen}
          orientation={current.solver}
          legalMoves={boardLegalMoves}
          {inputEnabled}
          lastMove={boardLastMove}
          wrongMove={boardWrongMove}
          hintSource={boardHintSource}
          hintTarget={boardHintTarget}
          {reducedMotion}
          adapterFactory={boardAdapterFactory}
          on:move={play}
          on:error={handleBoardError}
          on:announce={(event) => { announcement = event.detail.message }}
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
            aria-label={soundMuted ? 'Turn sound on' : 'Mute sounds'}
            aria-pressed={soundMuted}
            on:click={toggleSound}
          >
            <span aria-hidden="true">{soundMuted ? '🔇' : '🔊'}</span>
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
          {#if announcement && announcement !== feedback && announcement !== state.notice}
            <p class="visually-hidden">{announcement}</p>
          {/if}
        </div>

        <div class="puzzle-actions">
          {#if state.phase === 'solved'}
            <button class="primary next-action" type="button" on:click={acknowledgeSolution}>
              {solvedAction}
            </button>
          {:else}
            <button class="primary" type="button" disabled={!inputEnabled} on:click={useHint}>Hint</button>
            {#if state.hint?.canReveal || current.canReveal}
              <button class="secondary" type="button" disabled={!inputEnabled} on:click={reveal}>
                Show solution
              </button>
            {/if}
            <button class="quiet-action" type="button" disabled={!inputEnabled} on:click={pause}>Pause</button>
          {/if}
        </div>
      </aside>
    </section>
  {:else if terminalSession}
    <section class="panel" role="alert">
      <h2>Puzzle unavailable</h2>
      <p>{terminalNotice || 'This puzzle is no longer in the local library.'}</p>
      <button class="secondary" type="button" on:click={() => dispatch('home', { completed: false })}>
        Back home
      </button>
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
