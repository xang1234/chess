import type {
  AppliedMove,
  NormalAPI,
  OpeningMoveFeedback,
  OpeningMoveFeedbackDetail,
  OpeningSessionView,
  OpeningActivityResult
} from '../../lib/api'
import { moveSquares, type Square } from '../../lib/uci'
import type { BoardEffects } from '../chess/board-effects'
import {
  InteractiveBoardRuntime,
  type InteractiveBoardPort
} from '../chess/interactive-board-runtime'
import { animateAppliedMoves } from '../chess/move-animation'
import type { RequestToken } from '../chess/request-owner'
import {
  acceptsOpeningInput,
  acceptsOpeningResponse,
  acknowledgeOpeningActivity,
  beginOpeningAnimation,
  beginOpeningRequest,
  completeOpeningActivity,
  completeOpeningActivityWithoutPause,
  failOpeningRequest,
  finishOpeningHint,
  initialiseOpening,
  markOpeningFeedback,
  type OpeningOperation,
  type OpeningState
} from './opening-state'
import { openingFeedbackCopy } from './opening-feedback-copy'

export type OpeningBoardPort = InteractiveBoardPort

export type OpeningControllerEvents = {
  home(completed: boolean): void
  change(session: OpeningSessionView): void
  persisted(session: OpeningSessionView): void
}

export type OpeningControllerView = {
  state: OpeningState
  message: string
  feedback: OpeningMoveFeedback | null
  feedbackDetail: OpeningMoveFeedbackDetail | null
  notice: string
  announcement: string
  boardGeneration: number
  reducedMotion: boolean
  soundMuted: boolean
  lastFrames: AppliedMove[]
  comparisonPreviewIndex: number | null
  comparisonPreviewPlaying: boolean
  lastMove?: [Square, Square]
}

type ControllerOptions = {
  api: NormalAPI
  effects: BoardEffects
  events: OpeningControllerEvents
  afterRender(): Promise<void>
}

type OwnedRequest = RequestToken & {
  request: Extract<OpeningState, { phase: 'requesting' }>
}

type ViewSubscriber = (view: OpeningControllerView) => void

export class OpeningController {
  private readonly api: NormalAPI
  private readonly events: OpeningControllerEvents
  private readonly afterRender: () => Promise<void>
  private readonly runtime: InteractiveBoardRuntime
  private readonly subscribers = new Set<ViewSubscriber>()
  private currentView: OpeningControllerView | undefined
  private mounted = false
  private observedSession: OpeningSessionView | undefined
  private failedMoveUci: string | undefined

  constructor(options: ControllerOptions) {
    this.api = options.api
    this.events = options.events
    this.afterRender = options.afterRender
    this.runtime = new InteractiveBoardRuntime(options.effects, {
      publishPosition: (fen, lastMove, replaceBoard) => this.updateStateFen(
        fen,
        lastMove,
        replaceBoard ? this.view.boardGeneration + 1 : this.view.boardGeneration
      )
    })
  }

  readonly subscribe = (subscriber: ViewSubscriber): (() => void) => {
    this.subscribers.add(subscriber)
    if (this.currentView) subscriber(this.currentView)
    return () => this.subscribers.delete(subscriber)
  }

  get view(): OpeningControllerView {
    if (!this.currentView) throw new Error('Opening controller is not mounted')
    return this.currentView
  }

  mount(session: OpeningSessionView): void {
    if (this.mounted) return
    this.mounted = true
    const preferences = this.runtime.mount()
    this.observedSession = session
    this.publish({
      state: initialiseOpening(session),
      message: '',
      feedback: null,
      feedbackDetail: null,
      notice: session.notice ?? '',
      announcement: '',
      boardGeneration: 0,
      reducedMotion: preferences.reducedMotion,
      soundMuted: preferences.soundMuted,
      lastFrames: [],
      comparisonPreviewIndex: null,
      comparisonPreviewPlaying: false
    })
  }

  destroy(): void {
    if (this.currentView) {
      this.update({
        comparisonPreviewIndex: null,
        comparisonPreviewPlaying: false,
        lastMove: undefined
      })
    }
    this.mounted = false
    this.runtime.destroy()
  }

  attachBoard(board: OpeningBoardPort | undefined): void {
    this.runtime.attachBoard(board)
  }

  receiveSession(session: OpeningSessionView): void {
    if (!this.mounted || session === this.observedSession) return
    this.observedSession = session
    if (session === this.visibleSession()) return
    this.runtime.cancelRequest()
    this.adoptSession(session)
  }

  async advance(): Promise<void> {
    await this.resetComparisonPreviewIfNeeded()
    const owned = this.beginOwnedRequest('advance')
    if (!owned) return
    try {
      const result = await this.api.advanceOpeningActivity(owned.request.session.sessionId)
      if (!this.acceptsOwnedResponse(owned)) return
      if (result.activityCompleted === false) {
        throw new Error('Teaching step did not advance to the next course step.')
      }
      await this.finishSuccessful(owned, result)
    } catch (error) {
      await this.failRequest(owned, error, false)
    }
  }

  async play(uci: string): Promise<void> {
    const owned = this.beginOwnedRequest('move')
    if (!owned) return
    let result: OpeningActivityResult
    try {
      result = await this.api.playOpeningMove(owned.request.session.sessionId, uci)
    } catch (error) {
      this.failedMoveUci = uci
      await this.failRequest(owned, error, true)
      return
    }
    if (!this.acceptsOwnedResponse(owned)) return
    if (result.activityCompleted === false) {
      await this.finishFeedback(owned, result)
      return
    }
    await this.finishSuccessful(owned, result, uci)
    this.failedMoveUci = undefined
  }

  async retry(): Promise<void> {
    const state = this.requireState()
    if (state.phase !== 'failed' || !state.recoverable || !state.retryOperation) return
    switch (state.retryOperation) {
      case 'advance': await this.advance(); return
      case 'hint': await this.useHint(); return
      case 'reveal': await this.reveal(); return
      case 'pause': await this.pause(); return
      case 'move':
        if (this.failedMoveUci) await this.play(this.failedMoveUci)
        return
      case 'restart': return
    }
  }

  async useHint(): Promise<void> {
    const owned = this.beginOwnedRequest('hint')
    if (!owned) return
    try {
      const hint = await this.api.useOpeningHint(owned.request.session.sessionId)
      if (!this.acceptsOwnedResponse(owned)) return
      const state = finishOpeningHint(this.requireState(), owned.id, hint)
      this.publishState(state, {
        message: hint.text,
        feedback: null,
        announcement: hint.text,
        notice: hint.session.notice ?? ''
      })
      this.events.change(hint.session)
    } catch (error) {
      await this.failRequest(owned, error, false)
    }
  }

  async reveal(): Promise<void> {
    const owned = this.beginOwnedRequest('reveal')
    if (!owned) return
    try {
      const result = await this.api.revealOpeningMove(owned.request.session.sessionId)
      if (!this.acceptsOwnedResponse(owned)) return
      if (result.activityCompleted === false) {
        throw new Error('Reveal did not return the authoritative course move.')
      }
      await this.finishSuccessful(owned, result)
    } catch (error) {
      await this.failRequest(owned, error, false)
    }
  }

  async pause(): Promise<void> {
    await this.resetComparisonPreviewIfNeeded()
    const owned = this.beginOwnedRequest('pause')
    if (!owned) return
    try {
      await this.api.pauseOpeningSession(owned.request.session.sessionId)
      if (!this.acceptsOwnedResponse(owned)) return
      this.publishState(owned.request, {
        message: 'Lesson paused.',
        announcement: 'Lesson paused.'
      })
      await this.afterRender()
      if (!this.runtime.isCurrent(owned)) return
      this.runtime.cancelRequest()
      this.events.home(false)
    } catch (error) {
      await this.failRequest(owned, error, false)
    }
  }

  async restart(): Promise<void> {
    const state = this.requireState()
    if (state.phase !== 'restart-required') return
    const token = this.runtime.startRequest()
    this.update({
      message: 'Restarting from a safe checkpoint…',
      announcement: 'Restarting lesson.',
      comparisonPreviewIndex: null,
      comparisonPreviewPlaying: false,
      lastMove: undefined
    })
    try {
      const next = await this.api.restartOpeningSession(state.session.sessionId)
      if (!this.runtime.isCurrent(token)) return
      this.runtime.cancelRequest()
      this.adoptSession(next)
      this.events.change(next)
    } catch (error) {
      if (!this.runtime.isCurrent(token)) return
      const message = errorMessage(error)
      this.update({ message, announcement: message })
    }
  }

  acknowledgeActivity(): void {
    const state = this.requireState()
    if (state.phase !== 'activity-complete') return
    const pending = state.result.session
    const boardGeneration = this.generationForTargetFen(targetSessionFen(pending))
    this.runtime.cancelRequest()
    this.publishState(acknowledgeOpeningActivity(state), {
      message: '',
      feedback: null,
      notice: pending.notice ?? '',
      announcement: state.result.checkpoint ? 'Lesson roadmap checkpoint.' : 'Next lesson idea.',
      boardGeneration,
      lastFrames: [],
      comparisonPreviewIndex: null,
      comparisonPreviewPlaying: false,
      lastMove: undefined
    })
    this.events.change(pending)
  }

  async replayMovesToHere(): Promise<void> {
    const state = this.requireState()
    if (state.phase === 'summary' || state.phase === 'checkpoint' ||
      state.phase === 'restart-required') return
    await this.replay(state.session.current.movesToHere, state.session.current.currentFen)
  }

  async replayDemonstration(): Promise<void> {
    const state = this.requireState()
    if (state.phase !== 'activity-complete' || state.session.current.kind !== 'demonstration') return
    await this.replay(this.view.lastFrames, state.session.current.currentFen)
  }

  async previewComparison(index: number): Promise<void> {
    const state = this.requireState()
    if (state.phase !== 'passive' || state.session.current.kind !== 'comparison') return
    const line = state.session.current.comparison[index]
    if (!line) return

    const token = this.runtime.startRequest()
    this.update({
      comparisonPreviewIndex: index,
      comparisonPreviewPlaying: true,
      message: `Playing ${line.label}…`,
      announcement: `Playing ${line.label}.`
    })
    const resetWarning = await this.runtime.reconcilePosition(
      state.session.current.currentFen,
      token.controller.signal,
      false
    )
    if (!this.runtime.isCurrent(token)) return
    if (resetWarning) {
      await this.afterRender()
      if (!this.runtime.isCurrent(token)) return
    }

    const animation = await animateAppliedMoves({
      port: this.runtime.animationPort(),
      startingFen: state.session.current.currentFen,
      appliedMoves: line.frames,
      finalFen: line.frames[line.frames.length - 1].resultingFen,
      reducedMotion: this.view.reducedMotion,
      signal: token.controller.signal,
      onStep: (kind) => this.runtime.playSound(kind)
    })
    if (!this.runtime.isCurrent(token) || animation.status === 'aborted') return
    let warning = animation.warning ?? resetWarning
    if (animation.warning) {
      await this.afterRender()
      if (!this.runtime.isCurrent(token)) return
      const restoreWarning = await this.runtime.reconcilePosition(
        state.session.current.currentFen,
        token.controller.signal,
        false
      )
      if (!this.runtime.isCurrent(token)) return
      warning = [animation.warning, restoreWarning].filter(Boolean).join(' ')
    }
    this.runtime.cancelRequest()
    this.update({
      comparisonPreviewIndex: index,
      comparisonPreviewPlaying: false,
      message: `${line.label} complete.`,
      announcement: `${line.label} complete.`,
      notice: warning
    })
    this.applyBoardWarnings()
  }

  async resetComparisonPreview(): Promise<void> {
    const state = this.requireState()
    if (state.phase === 'summary' || state.phase === 'checkpoint' ||
      state.phase === 'restart-required') return
    if (this.view.comparisonPreviewIndex === null && !this.view.comparisonPreviewPlaying) return

    const token = this.runtime.startRequest()
    const warning = await this.runtime.reconcilePosition(
      state.session.current.currentFen,
      token.controller.signal,
      false
    )
    if (!this.runtime.isCurrent(token)) return
    this.runtime.cancelRequest()
    this.update({
      comparisonPreviewIndex: null,
      comparisonPreviewPlaying: false,
      message: 'Comparison position reset.',
      announcement: 'Comparison position reset.',
      notice: warning,
      lastMove: undefined
    })
    this.applyBoardWarnings()
  }

  finishHome(): void {
    this.events.home(true)
  }

  handleBoardError(message: string): void {
    const recovering = this.runtime.isRecovering()
    this.runtime.noteBoardError(message)
    if (recovering) return
    this.runtime.consumeWarnings()
    const state = this.requireState()
    if (state.phase === 'activity-complete' || this.view.comparisonPreviewPlaying ||
      this.view.comparisonPreviewIndex !== null) {
      this.update({ notice: message, announcement: message })
      return
    }
    this.publishState(failOpeningRequest(state, message, false), {
      message,
      announcement: message
    })
  }

  announce(message: string): void {
    this.update({ announcement: message })
  }

  startSoundUnlock(): void {
    this.runtime.unlockFromPointer()
  }

  unlockFromKeyboard(key: string): void {
    this.runtime.unlockFromKeyboard(key)
  }

  toggleSound(): void {
    this.update({ soundMuted: this.runtime.toggleSound() })
  }

  private visibleSession(): OpeningSessionView | undefined {
    return this.currentView?.state.session
  }

  private adoptSession(session: OpeningSessionView): void {
    const boardGeneration = this.currentView
      ? this.generationForTargetFen(targetSessionFen(session))
      : 0
    this.runtime.consumeWarnings()
    this.publishState(initialiseOpening(session), {
      message: '',
      feedback: null,
      notice: session.notice ?? '',
      announcement: '',
      boardGeneration,
      lastFrames: [],
      comparisonPreviewIndex: null,
      comparisonPreviewPlaying: false,
      lastMove: undefined
    })
  }

  private generationForTargetFen(targetFen: string | undefined): number {
    if (targetFen === undefined) return this.view.boardGeneration
    return visibleFen(this.requireState()) === targetFen
      ? this.view.boardGeneration
      : this.view.boardGeneration + 1
  }

  private beginOwnedRequest(operation: OpeningOperation): OwnedRequest | null {
    const state = this.requireState()
    const allowed = state.phase === 'passive' || acceptsOpeningInput(state) ||
      (state.phase === 'failed' && state.recoverable)
    if (!allowed) return null
    const token = this.runtime.startRequest()
    try {
      const request = beginOpeningRequest(state, operation, token.id)
      if (request.phase !== 'requesting') throw new Error('Opening request did not start.')
      const message = requestMessage(operation)
      this.publishState(request, { message, feedback: null, announcement: message })
      return { ...token, request }
    } catch (error) {
      const message = errorMessage(error)
      this.publishState(failOpeningRequest(state, message, false), { message, announcement: message })
      return null
    }
  }

  private acceptsOwnedResponse(owned: OwnedRequest): boolean {
    return this.runtime.isCurrent(owned) && acceptsOpeningResponse(this.requireState(), owned.id)
  }

  private async finishFeedback(
    owned: OwnedRequest,
    result: Extract<OpeningActivityResult, { activityCompleted: false }>
  ): Promise<void> {
    const authoritativeFen = result.session.current.currentFen
    const warning = await this.runtime.reconcilePosition(
      authoritativeFen,
      owned.controller.signal,
      !this.view.reducedMotion
    )
    if (!this.acceptsOwnedResponse(owned)) return
    const message = result.message || feedbackMessage(result.feedback)
    const announcement = result.feedbackDetail
      ? openingFeedbackCopy(result.feedbackDetail, result.session.current.orientation).announcement
      : message
    const state = markOpeningFeedback(
      this.requireState(),
      owned.id,
      result.session,
      result.feedback
    )
    this.publishState(state, {
      message,
      feedback: result.feedback,
      feedbackDetail: result.feedbackDetail ?? null,
      announcement,
      notice: warning
    })
    this.applyBoardWarnings()
    this.events.change(result.session)
  }

  private async finishSuccessful(
    owned: OwnedRequest,
    result: Extract<OpeningActivityResult, { activityCompleted: true }>,
    optimisticUci?: string
  ): Promise<void> {
    const frames: AppliedMove[] = result.appliedMoves ?? []
    if (frames.length === 0 && result.checkpoint === undefined &&
      result.session.status === 'active') {
      const next = completeOpeningActivityWithoutPause(this.requireState(), owned.id, result)
      this.runtime.cancelRequest()
      this.publishState(next, {
        message: '',
        feedback: null,
        notice: result.session.notice ?? '',
        announcement: `Next idea: ${result.session.current.title}.`,
        boardGeneration: this.generationForTargetFen(result.session.current.currentFen),
        lastFrames: [],
        comparisonPreviewIndex: null,
        comparisonPreviewPlaying: false,
        lastMove: undefined
      })
      this.events.persisted(result.session)
      this.events.change(result.session)
      return
    }
    const finalFen = result.finalFen ?? frames.at(-1)?.resultingFen ?? owned.request.fen
    const animating = beginOpeningAnimation(this.requireState(), owned.id)
    if (animating.phase !== 'animating') return
    this.publishState(animating, { message: 'Showing the course line…', feedback: null })

    const animation = await animateAppliedMoves({
      port: this.runtime.animationPort(),
      startingFen: owned.request.fen,
      appliedMoves: frames,
      ...(optimisticUci ? { optimisticUci } : {}),
      finalFen,
      reducedMotion: this.view.reducedMotion,
      signal: owned.controller.signal,
      onStep: (kind) => this.runtime.playSound(kind)
    })
    const state = this.requireState()
    if (!this.runtime.isCurrent(owned) || animation.status === 'aborted' ||
      state.phase !== 'animating' || state.requestId !== owned.id) return

    const message = result.message || (owned.request.operation === 'reveal'
      ? 'Course move shown.'
      : owned.request.operation === 'move' ? 'Course move found.' : 'Idea complete.')
    const lastMove = frames.length > 0 ? moveSquares(frames[frames.length - 1].uci) : undefined
    const complete = completeOpeningActivity(state, owned.id, finalFen, result, message)
    this.publishState(complete, {
      message,
      feedback: 'feedback' in result ? result.feedback ?? null : null,
      announcement: message,
      notice: animation.warning ?? '',
      lastFrames: frames,
      comparisonPreviewIndex: null,
      comparisonPreviewPlaying: false,
      ...(lastMove ? { lastMove } : {})
    })
    this.applyBoardWarnings()
    if (owned.request.operation === 'move') this.runtime.playSound('correct')
    this.events.persisted(result.session)
  }

  private async failRequest(owned: OwnedRequest, error: unknown, reconcile: boolean): Promise<void> {
    if (!this.acceptsOwnedResponse(owned)) return
    let warning = ''
    if (reconcile) {
      warning = await this.runtime.reconcilePosition(
        owned.request.fen,
        owned.controller.signal,
        !this.view.reducedMotion
      )
    }
    if (!this.acceptsOwnedResponse(owned)) return
    const message = errorMessage(error)
    this.publishState(failOpeningRequest(this.requireState(), message, true), {
      message,
      announcement: message,
      notice: warning
    })
    this.applyBoardWarnings()
  }

  private async replay(frames: readonly AppliedMove[], startingFen: string): Promise<void> {
    if (frames.length === 0) {
      this.update({ message: 'This position has no earlier moves to replay.' })
      return
    }
    const token = this.runtime.startRequest()
    this.update({ message: 'Replaying the line…', announcement: 'Replaying the line.' })
    const finalFen = frames[frames.length - 1].resultingFen
    const animation = await animateAppliedMoves({
      port: this.runtime.animationPort(),
      startingFen,
      appliedMoves: [...frames],
      finalFen,
      reducedMotion: this.view.reducedMotion,
      signal: token.controller.signal,
      onStep: (kind) => this.runtime.playSound(kind)
    })
    if (!this.runtime.isCurrent(token) || animation.status === 'aborted') return
    this.runtime.cancelRequest()
    this.update({
      message: 'Replay complete.',
      announcement: 'Replay complete.',
      notice: animation.warning ?? ''
    })
    this.applyBoardWarnings()
  }

  private async resetComparisonPreviewIfNeeded(): Promise<void> {
    if (this.view.comparisonPreviewIndex === null && !this.view.comparisonPreviewPlaying) return
    await this.resetComparisonPreview()
  }

  private updateStateFen(
    fen: string,
    lastMove?: [Square, Square],
    boardGeneration = this.view.boardGeneration
  ): void {
    const state = this.requireState()
    if (!('fen' in state)) throw new Error(`${state.phase} state has no board position`)
    this.publishState({ ...state, fen }, {
      boardGeneration,
      lastMove
    })
  }

  private applyBoardWarnings(warning = ''): void {
    const message = this.runtime.consumeWarnings(warning, this.view.notice)
    if (message) this.update({ notice: message })
  }

  private requireState(): OpeningState {
    return this.view.state
  }

  private publishState(
    state: OpeningState,
    overrides: Partial<Omit<OpeningControllerView, 'state'>> = {}
  ): void {
    const current = this.currentView
    this.publish({
      state,
      message: overrides.message ?? current?.message ?? '',
      feedback: overrides.feedback === undefined ? current?.feedback ?? null : overrides.feedback,
      feedbackDetail: overrides.feedbackDetail === undefined
        ? overrides.feedback === null ? null : current?.feedbackDetail ?? null
        : overrides.feedbackDetail,
      notice: overrides.notice ?? current?.notice ?? '',
      announcement: overrides.announcement ?? current?.announcement ?? '',
      boardGeneration: overrides.boardGeneration ?? current?.boardGeneration ?? 0,
      reducedMotion: overrides.reducedMotion ?? current?.reducedMotion ?? false,
      soundMuted: overrides.soundMuted ?? current?.soundMuted ?? false,
      lastFrames: overrides.lastFrames ?? current?.lastFrames ?? [],
      comparisonPreviewIndex: Object.prototype.hasOwnProperty.call(
        overrides,
        'comparisonPreviewIndex'
      )
        ? overrides.comparisonPreviewIndex ?? null
        : current?.comparisonPreviewIndex ?? null,
      comparisonPreviewPlaying: overrides.comparisonPreviewPlaying ??
        current?.comparisonPreviewPlaying ?? false,
      ...(Object.prototype.hasOwnProperty.call(overrides, 'lastMove')
        ? overrides.lastMove ? { lastMove: overrides.lastMove } : {}
        : current?.lastMove ? { lastMove: current.lastMove } : {})
    })
  }

  private update(overrides: Partial<Omit<OpeningControllerView, 'state'>>): void {
    this.publishState(this.requireState(), overrides)
  }

  private publish(view: OpeningControllerView): void {
    this.currentView = view
    for (const subscriber of this.subscribers) subscriber(view)
  }
}

function visibleFen(state: OpeningState): string | undefined {
  return 'fen' in state ? state.fen : undefined
}

function targetSessionFen(session: OpeningSessionView): string | undefined {
  return session.status === 'active' ? session.current.currentFen : undefined
}

function requestMessage(operation: OpeningOperation): string {
  switch (operation) {
    case 'advance': return 'Preparing the next idea…'
    case 'move': return 'Checking that move…'
    case 'hint': return 'Finding the next hint…'
    case 'reveal': return 'Preparing the course move…'
    case 'pause': return 'Pausing lesson…'
    case 'restart': return 'Restarting lesson…'
  }
}

function feedbackMessage(feedback: 'alternative' | 'off_course'): string {
  return feedback === 'alternative'
    ? 'Playable alternative. Return to the course line when you are ready.'
    : 'Outside this course line. Try the position again.'
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

export function createOpeningController(options: ControllerOptions): OpeningController {
  return new OpeningController(options)
}
