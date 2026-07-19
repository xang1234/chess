import type {
  AppliedMove,
  CompletedSessionView,
  HintResult,
  IncorrectMoveResult,
  MoveResult,
  NormalAPI,
  SessionView
} from '../../lib/api'
import { moveSquares } from '../../lib/uci'
import type { BoardEffects } from '../chess/board-effects'
import {
  InteractiveBoardRuntime,
  type InteractiveBoardPort
} from '../chess/interactive-board-runtime'
import { animateAppliedMoves } from '../chess/move-animation'
import type { RequestToken } from '../chess/request-owner'
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
import {
  acceptsInput,
  initialRenderState,
  requestMessage,
  type PuzzleRenderState,
  type RenderCommon
} from './puzzle-view'
import {
  errorMessage,
  responseAdvanced,
  validatePuzzle,
  validateSuccessfulResult
} from './puzzle-validation'
export type PuzzleBoardPort = InteractiveBoardPort

export type PuzzleControllerEvents = {
  home(completed: boolean): void
  change(session: SessionView): void
  persisted(session: SessionView): void
}

type ControllerOptions = {
  api: NormalAPI
  effects: BoardEffects
  events: PuzzleControllerEvents
  afterRender(): Promise<void>
}

type OwnedRequest = RequestToken & {
  request: Extract<PuzzleState, { phase: 'requesting' }>
}

type SuccessfulMoveResult = Extract<MoveResult, { correct: true }>
type ViewSubscriber = (view: PuzzleRenderState) => void

export class PuzzleController {
  private readonly api: NormalAPI
  private readonly events: PuzzleControllerEvents
  private readonly afterRender: () => Promise<void>
  private readonly runtime: InteractiveBoardRuntime
  private readonly subscribers = new Set<ViewSubscriber>()
  private currentView = initialRenderState()
  private mounted = false
  private observedSession: SessionView | undefined

  constructor(options: ControllerOptions) {
    this.api = options.api
    this.events = options.events
    this.afterRender = options.afterRender
    this.runtime = new InteractiveBoardRuntime(options.effects, {
      publishPosition: (fen, lastMove, replaceBoard) => {
        const state = this.requirePuzzleState()
        this.setPuzzle({ ...state, fen, lastMove }, replaceBoard
          ? { boardGeneration: this.currentView.boardGeneration + 1 }
          : {})
      }
    })
  }

  readonly subscribe = (subscriber: ViewSubscriber): (() => void) => {
    this.subscribers.add(subscriber)
    subscriber(this.currentView)
    return () => this.subscribers.delete(subscriber)
  }

  get view(): PuzzleRenderState {
    return this.currentView
  }

  mount(session: SessionView): void {
    if (this.mounted) return
    this.mounted = true
    this.updateCommon(this.runtime.mount())
    this.observedSession = session
    this.adoptVisibleSession(session)
  }

  destroy(): void {
    this.mounted = false
    this.runtime.destroy()
  }

  attachBoard(board: PuzzleBoardPort | undefined): void {
    this.runtime.attachBoard(board)
  }

  receiveSession(session: SessionView): void {
    if (!this.mounted || session === this.observedSession) return
    this.observedSession = session
    if (session !== this.visibleSession()) this.adoptVisibleSession(session)
  }

  async play(uci: string): Promise<void> {
    const owned = this.beginOwnedRequest('move', uci)
    if (!owned) return
    let result: MoveResult
    try {
      result = await this.api.playMove(owned.request.displaySession.sessionId, uci)
    } catch (error) {
      await this.failRequest(owned, error, true)
      return
    }
    if (!this.acceptsOwnedResponse(owned)) return
    if (result.correct === false && responseAdvanced(owned.request.displaySession, result.session)) {
      this.adoptVisibleSession(result.session, result.message || 'Puzzle unavailable', true)
      return
    }
    if (result.correct === false) {
      await this.finishIncorrect(owned, result, uci)
      return
    }
    await this.finishSuccessful(owned, result, 'correct', uci)
  }

  async useHint(): Promise<void> {
    const owned = this.beginOwnedRequest('hint')
    if (!owned) return
    let result: HintResult
    try {
      result = await this.api.useHint(owned.request.displaySession.sessionId)
    } catch (error) {
      await this.failRequest(owned, error, false)
      return
    }
    if (!this.acceptsOwnedResponse(owned)) return
    if (responseAdvanced(owned.request.displaySession, result.session)) {
      this.adoptVisibleSession(result.session, result.text || 'Puzzle unavailable', true)
      return
    }
    if (result.session.status !== 'active') {
      this.failFatal('Hint response has no current puzzle.')
      return
    }
    try {
      validatePuzzle(result.session.current)
      const state = this.requirePuzzleState()
      this.setPuzzle(finishReadyRequest(
        state,
        owned.id,
        result.session,
        result.level > 0 ? result : null
      ), { announcement: result.text })
      this.events.change(result.session)
    } catch (error) {
      this.failFatal(errorMessage(error))
    }
  }

  async reveal(): Promise<void> {
    const owned = this.beginOwnedRequest('reveal')
    if (!owned) return
    let result: MoveResult
    try {
      result = await this.api.revealSolution(owned.request.displaySession.sessionId)
    } catch (error) {
      await this.failRequest(owned, error, false)
      return
    }
    if (!this.acceptsOwnedResponse(owned)) return
    if (result.correct === false && responseAdvanced(owned.request.displaySession, result.session)) {
      this.adoptVisibleSession(result.session, result.message || 'Puzzle unavailable', true)
      return
    }
    if (result.correct === false || result.puzzleCompleted === false) {
      this.failFatal('Reveal response did not include a completed authoritative solution.')
      return
    }
    await this.finishSuccessful(owned, result, 'revealed')
  }

  async pause(): Promise<void> {
    const owned = this.beginOwnedRequest('pause')
    if (!owned) return
    try {
      await this.api.pauseSession(owned.request.displaySession.sessionId)
    } catch (error) {
      await this.failRequest(owned, error, false)
      return
    }
    if (!this.acceptsOwnedResponse(owned)) return
    this.updateCommon({ announcement: 'Training paused.' })
    await this.afterRender()
    if (!this.acceptsOwnedResponse(owned)) return
    this.runtime.cancelRequest()
    this.events.home(false)
  }

  acknowledgeSolution(): void {
    const state = this.puzzleState()
    if (!state || state.phase !== 'solved') return
    try {
      const acknowledged = acknowledgeSolved(state, this.currentView.reducedMotion)
      this.runtime.cancelRequest()
      if (acknowledged.kind === 'puzzle') {
        this.setPuzzle(acknowledged.state, {
          announcement: 'Next puzzle.',
          boardGeneration: this.currentView.boardGeneration + 1
        })
        this.events.change(acknowledged.state.displaySession)
        if (acknowledged.state.phase === 'prelude') void this.playPrelude()
      } else {
        this.setSummary(acknowledged.session, '', {
          announcement: 'Training results.',
          boardGeneration: this.currentView.boardGeneration + 1
        })
        this.events.change(acknowledged.session)
      }
    } catch (error) {
      this.failFatal(errorMessage(error))
    }
  }

  handleBoardError(message: string): void {
    const recovering = this.runtime.isRecovering()
    this.runtime.noteBoardError(message)
    if (recovering) return
    this.runtime.consumeWarnings()
    this.failFatal(message)
  }

  announce(message: string): void {
    this.updateCommon({ announcement: message })
  }

  startSoundUnlock(): void {
    this.runtime.unlockFromPointer()
  }

  unlockFromKeyboard(key: string): void {
    this.runtime.unlockFromKeyboard(key)
  }

  toggleSound(): void {
    this.updateCommon({ soundMuted: this.runtime.toggleSound() })
  }

  finishHome(): void {
    this.events.home(true)
  }

  private visibleSession(): SessionView | undefined {
    if (this.currentView.kind === 'puzzle') return this.currentView.state.displaySession
    if (this.currentView.kind === 'summary') return this.currentView.session
    return undefined
  }

  private adoptVisibleSession(next: SessionView, notice = '', notify = false): void {
    this.runtime.cancelRequest()
    this.runtime.consumeWarnings()
    const common = {
      announcement: notice,
      boardGeneration: this.currentView.boardGeneration + 1
    }
    if (next.status === 'completed') {
      this.setSummary(next, notice, common)
      if (notify) this.events.change(next)
      return
    }

    let state: PuzzleState
    try {
      validatePuzzle(next.current)
      state = initialisePuzzle(next, this.currentView.reducedMotion)
      if (notice) state = { ...state, notice }
    } catch (error) {
      state = markFailed(initialisePuzzle(next, true), errorMessage(error), false)
    }
    this.setPuzzle(state, common)
    if (notify) this.events.change(next)
    if (state.phase === 'prelude') void this.playPrelude()
  }

  private async playPrelude(): Promise<void> {
    const state = this.puzzleState()
    if (!state || state.phase !== 'prelude' || !state.displaySession.current.preludeUci) return
    const token = this.runtime.startRequest()
    await this.afterRender()
    const currentState = this.puzzleState()
    if (!this.runtime.isCurrent(token) || !currentState || currentState.phase !== 'prelude') return
    const currentPuzzle = currentState.displaySession.current
    const result = await animateAppliedMoves({
      port: this.runtime.animationPort(),
      startingFen: currentState.fen,
      appliedMoves: [{
        uci: currentPuzzle.preludeUci!,
        resultingFen: currentPuzzle.displayedFen
      }],
      finalFen: currentPuzzle.displayedFen,
      reducedMotion: false,
      signal: token.controller.signal,
      onStep: (kind) => this.runtime.playSound(kind)
    })
    const latest = this.puzzleState()
    if (!this.runtime.isCurrent(token) || result.status === 'aborted' ||
      !latest || latest.phase !== 'prelude') return
    try {
      this.setPuzzle(finishPrelude(latest), { announcement: 'The puzzle is ready.' })
      this.applyBoardWarnings(result.warning)
    } catch (error) {
      this.failFatal(errorMessage(error))
    }
  }

  private beginOwnedRequest(operation: Operation, submittedUci?: string): OwnedRequest | null {
    const state = this.puzzleState()
    if (!state || !acceptsInput(state)) return null
    const token = this.runtime.startRequest()
    try {
      let request = beginRequest(state, operation, token.id, submittedUci)
      if (request.phase !== 'requesting') throw new Error('Puzzle request did not start.')
      if (operation === 'move' || operation === 'reveal') request = { ...request, hint: null }
      this.setPuzzle(request, { announcement: requestMessage(operation) })
      return { ...token, request }
    } catch (error) {
      this.failFatal(errorMessage(error))
      return null
    }
  }

  private acceptsOwnedResponse(owned: OwnedRequest): boolean {
    const state = this.puzzleState()
    return this.runtime.isCurrent(owned) && Boolean(state && acceptsResponse(state, owned.id))
  }

  private async finishIncorrect(
    owned: OwnedRequest,
    result: IncorrectMoveResult,
    uci: string
  ): Promise<void> {
    if (result.session.status !== 'active') {
      this.failFatal('Incorrect response has no current puzzle.')
      return
    }
    try {
      validatePuzzle(result.session.current)
    } catch (error) {
      this.failFatal(errorMessage(error))
      return
    }
    const warning = await this.runtime.reconcilePosition(
      owned.request.authoritativeFen,
      owned.controller.signal,
      !this.currentView.reducedMotion
    )
    if (!this.acceptsOwnedResponse(owned)) return
    try {
      const next = markIncorrect(
        this.requirePuzzleState(),
        owned.id,
        result.session,
        uci,
        result.message || 'Try again'
      )
      this.setPuzzle(next, { announcement: result.message || 'Try again' })
      this.applyBoardWarnings(warning)
      this.runtime.playSound('incorrect')
      this.events.change(result.session)
    } catch (error) {
      this.failFatal(errorMessage(error))
    }
  }

  private async finishSuccessful(
    owned: OwnedRequest,
    result: SuccessfulMoveResult,
    outcome: SolvedOutcome,
    optimisticUci?: string
  ): Promise<void> {
    let frames: AppliedMove[]
    let target: string
    let readyAfter: PuzzleState | undefined
    try {
      frames = validateSuccessfulResult(result, owned.request, optimisticUci)
      if (result.puzzleCompleted === true) {
        target = result.finalFen
      } else {
        target = result.session.current.currentFen
        readyAfter = finishReadyRequest(owned.request, owned.id, result.session, null)
      }
      this.setPuzzle(beginAnimation(this.requirePuzzleState(), owned.id))
    } catch (error) {
      this.failFatal(errorMessage(error))
      return
    }

    const animation = await animateAppliedMoves({
      port: this.runtime.animationPort(),
      startingFen: owned.request.authoritativeFen,
      appliedMoves: frames,
      optimisticUci,
      finalFen: target,
      reducedMotion: this.currentView.reducedMotion,
      signal: owned.controller.signal,
      onStep: (kind) => this.runtime.playSound(kind)
    })
    const state = this.puzzleState()
    if (!this.runtime.isCurrent(owned) || animation.status === 'aborted' ||
      !state || state.phase !== 'animating' || state.requestId !== owned.id) return

    const lastMove = moveSquares(frames[frames.length - 1].uci)
    try {
      if (result.puzzleCompleted === true) {
        this.setPuzzle(markSolved(state, owned.id, outcome, target, result.session, lastMove), {
          announcement: outcome === 'correct' ? 'Correct!' : 'Solution shown'
        })
        this.applyBoardWarnings(animation.warning)
        if (outcome === 'correct') this.runtime.playSound('correct')
        this.events.persisted(result.session)
      } else {
        this.setPuzzle({
          ...readyAfter!,
          lastMove,
          ...(animation.warning ? { notice: animation.warning } : {})
        }, { announcement: 'Good move. Find the next move.' })
        this.applyBoardWarnings()
        this.events.change(result.session)
      }
    } catch (error) {
      this.failFatal(errorMessage(error))
    }
  }

  private async failRequest(
    owned: OwnedRequest,
    error: unknown,
    reconcile: boolean
  ): Promise<void> {
    if (!this.acceptsOwnedResponse(owned)) return
    let warning = ''
    if (reconcile) {
      warning = await this.runtime.reconcilePosition(
        owned.request.authoritativeFen,
        owned.controller.signal,
        !this.currentView.reducedMotion
      )
    }
    if (!this.acceptsOwnedResponse(owned)) return
    const message = errorMessage(error)
    this.setPuzzle(markFailed(this.requirePuzzleState(), message, true), { announcement: message })
    this.applyBoardWarnings(warning)
  }

  private applyBoardWarnings(warning = ''): void {
    const message = this.runtime.consumeWarnings(warning)
    const state = this.puzzleState()
    if (message && state) this.setPuzzle({ ...state, notice: message })
  }

  private failFatal(message: string): void {
    const state = this.puzzleState()
    this.updateCommon({ announcement: message })
    if (!state) return
    if (state.phase === 'solved') {
      this.setPuzzle({ ...state, notice: message })
      return
    }
    this.setPuzzle(markFailed(state, message, false))
  }

  private puzzleState(): PuzzleState | undefined {
    return this.currentView.kind === 'puzzle' ? this.currentView.state : undefined
  }

  private requirePuzzleState(): PuzzleState {
    const state = this.puzzleState()
    if (!state) throw new Error('Puzzle state is unavailable')
    return state
  }

  private common(overrides: Partial<RenderCommon> = {}): RenderCommon {
    return {
      announcement: overrides.announcement ?? this.currentView.announcement,
      boardGeneration: overrides.boardGeneration ?? this.currentView.boardGeneration,
      reducedMotion: overrides.reducedMotion ?? this.currentView.reducedMotion,
      soundMuted: overrides.soundMuted ?? this.currentView.soundMuted
    }
  }

  private updateCommon(overrides: Partial<RenderCommon>): void {
    const common = this.common(overrides)
    switch (this.currentView.kind) {
      case 'loading': this.publish({ ...common, kind: 'loading' }); break
      case 'puzzle': this.publish({ ...common, kind: 'puzzle', state: this.currentView.state }); break
      case 'summary': this.publish({
        ...common,
        kind: 'summary',
        session: this.currentView.session,
        notice: this.currentView.notice
      }); break
    }
  }

  private setPuzzle(state: PuzzleState, overrides: Partial<RenderCommon> = {}): void {
    this.publish({ ...this.common(overrides), kind: 'puzzle', state })
  }

  private setSummary(
    session: CompletedSessionView,
    notice: string,
    overrides: Partial<RenderCommon> = {}
  ): void {
    this.publish({ ...this.common(overrides), kind: 'summary', session, notice })
  }

  private publish(view: PuzzleRenderState): void {
    this.currentView = view
    for (const subscriber of this.subscribers) subscriber(view)
  }
}

export function createPuzzleController(options: ControllerOptions): PuzzleController {
  return new PuzzleController(options)
}
