import type {
  AppliedMove,
  NormalAPI,
  OpeningMoveFeedback,
  OpeningSessionView,
  OpeningStepResult
} from '../../lib/api'
import type { SoundService } from '../../lib/sound'
import { moveSquares, type Square } from '../../lib/uci'
import type { BoardEffects } from '../chess/board-effects'
import { animateAppliedMoves, type PositionFrame } from '../chess/move-animation'
import { RequestOwner, type RequestToken } from '../chess/request-owner'
import {
  acceptsOpeningInput,
  acceptsOpeningResponse,
  acknowledgeOpeningStep,
  beginOpeningAnimation,
  beginOpeningRequest,
  completeOpeningStep,
  failOpeningRequest,
  finishOpeningHint,
  initialiseOpening,
  markOpeningFeedback,
  type OpeningOperation,
  type OpeningState
} from './opening-state'

export type OpeningBoardPort = {
  setPosition(fen: string, lastMove?: [Square, Square], animate?: boolean): void
}

export type OpeningControllerEvents = {
  home(completed: boolean): void
  change(session: OpeningSessionView): void
  persisted(session: OpeningSessionView): void
}

export type OpeningControllerView = {
  state: OpeningState
  message: string
  feedback: OpeningMoveFeedback | null
  notice: string
  announcement: string
  boardGeneration: number
  reducedMotion: boolean
  soundMuted: boolean
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
  private readonly effects: BoardEffects
  private readonly events: OpeningControllerEvents
  private readonly afterRender: () => Promise<void>
  private readonly requests = new RequestOwner()
  private readonly subscribers = new Set<ViewSubscriber>()
  private currentView: OpeningControllerView | undefined
  private board: OpeningBoardPort | undefined
  private mounted = false
  private observedSession: OpeningSessionView | undefined
  private sound: SoundService | undefined
  private soundUnlockStarted = false
  private recoveringBoard = false
  private pendingBoardWarning = ''

  constructor(options: ControllerOptions) {
    this.api = options.api
    this.effects = options.effects
    this.events = options.events
    this.afterRender = options.afterRender
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
    this.sound = this.effects.createSound()
    this.observedSession = session
    this.publish({
      state: initialiseOpening(session),
      message: '',
      feedback: null,
      notice: session.notice ?? '',
      announcement: '',
      boardGeneration: 0,
      reducedMotion: this.effects.prefersReducedMotion(),
      soundMuted: this.sound.muted
    })
  }

  destroy(): void {
    this.mounted = false
    this.requests.cancel()
    this.sound?.destroy()
    this.sound = undefined
    this.board = undefined
  }

  attachBoard(board: OpeningBoardPort | undefined): void {
    this.board = board
  }

  receiveSession(session: OpeningSessionView): void {
    if (!this.mounted || session === this.observedSession) return
    this.observedSession = session
    if (session === this.visibleSession()) return
    this.requests.cancel()
    this.adoptSession(session)
  }

  async advance(): Promise<void> {
    const owned = this.beginOwnedRequest('advance')
    if (!owned) return
    try {
      const result = await this.api.advanceOpeningStep(owned.request.session.sessionId)
      if (!this.acceptsOwnedResponse(owned)) return
      if (result.stepCompleted === false) {
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
    let result: OpeningStepResult
    try {
      result = await this.api.playOpeningMove(owned.request.session.sessionId, uci)
    } catch (error) {
      await this.failRequest(owned, error, true)
      return
    }
    if (!this.acceptsOwnedResponse(owned)) return
    if (result.stepCompleted === false) {
      await this.finishFeedback(owned, result)
      return
    }
    await this.finishSuccessful(owned, result, uci)
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
      if (result.stepCompleted === false) {
        throw new Error('Reveal did not return the authoritative course move.')
      }
      await this.finishSuccessful(owned, result)
    } catch (error) {
      await this.failRequest(owned, error, false)
    }
  }

  async pause(): Promise<void> {
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
      if (!this.isCurrent(owned)) return
      this.requests.cancel()
      this.events.home(false)
    } catch (error) {
      await this.failRequest(owned, error, false)
    }
  }

  async restart(): Promise<void> {
    const state = this.requireState()
    if (state.phase !== 'restart-required') return
    const token = this.requests.start()
    this.update({ message: 'Restarting from a safe checkpoint…', announcement: 'Restarting lesson.' })
    try {
      const next = await this.api.restartOpeningSession(state.session.sessionId)
      if (!this.isCurrent(token)) return
      this.requests.cancel()
      this.adoptSession(next)
      this.events.change(next)
    } catch (error) {
      if (!this.isCurrent(token)) return
      const message = errorMessage(error)
      this.update({ message, announcement: message })
    }
  }

  acknowledgeStep(): void {
    const state = this.requireState()
    if (state.phase !== 'step-complete') return
    const pending = state.pending
    this.requests.cancel()
    this.publishState(acknowledgeOpeningStep(state), {
      message: '',
      feedback: null,
      notice: pending.notice ?? '',
      announcement: pending.status === 'completed' ? 'Opening lesson results.' : 'Next lesson step.',
      boardGeneration: this.view.boardGeneration + 1,
      lastMove: undefined
    })
    this.events.change(pending)
  }

  finishHome(): void {
    this.events.home(true)
  }

  handleBoardError(message: string): void {
    if (this.recoveringBoard) {
      this.pendingBoardWarning = message
      return
    }
    const state = this.requireState()
    if (state.phase === 'step-complete') {
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
    if (!this.sound || this.soundUnlockStarted) return
    this.soundUnlockStarted = true
    try {
      void this.sound.unlock().catch(() => { this.soundUnlockStarted = false })
    } catch {
      this.soundUnlockStarted = false
    }
  }

  unlockFromKeyboard(key: string): void {
    if (key === 'Enter' || key === ' ') this.startSoundUnlock()
  }

  toggleSound(): void {
    if (!this.sound) return
    this.update({ soundMuted: this.sound.toggleMuted() })
  }

  private visibleSession(): OpeningSessionView | undefined {
    return this.currentView?.state.session
  }

  private adoptSession(session: OpeningSessionView): void {
    this.pendingBoardWarning = ''
    this.recoveringBoard = false
    this.publishState(initialiseOpening(session), {
      message: '',
      feedback: null,
      notice: session.notice ?? '',
      announcement: '',
      boardGeneration: this.currentView ? this.currentView.boardGeneration + 1 : 0,
      lastMove: undefined
    })
  }

  private beginOwnedRequest(operation: OpeningOperation): OwnedRequest | null {
    const state = this.requireState()
    const allowed = state.phase === 'passive' || acceptsOpeningInput(state)
    if (!allowed) return null
    const token = this.requests.start()
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
    return this.isCurrent(owned) && acceptsOpeningResponse(this.requireState(), owned.id)
  }

  private async finishFeedback(
    owned: OwnedRequest,
    result: Extract<OpeningStepResult, { stepCompleted: false }>
  ): Promise<void> {
    const authoritativeFen = result.session.current.currentFen
    const warning = await this.reconcilePosition(
      authoritativeFen,
      owned.controller,
      !this.view.reducedMotion
    )
    if (!this.acceptsOwnedResponse(owned)) return
    const message = result.message || feedbackMessage(result.feedback)
    const state = markOpeningFeedback(
      this.requireState(),
      owned.id,
      result.session,
      result.feedback
    )
    this.publishState(state, {
      message,
      feedback: result.feedback,
      announcement: message,
      notice: warning
    })
    this.applyBoardWarnings()
    this.events.change(result.session)
  }

  private async finishSuccessful(
    owned: OwnedRequest,
    result: Extract<OpeningStepResult, { stepCompleted: true }>,
    optimisticUci?: string
  ): Promise<void> {
    const frames: AppliedMove[] = result.appliedMoves ?? []
    const finalFen = result.finalFen ?? frames.at(-1)?.resultingFen ?? owned.request.fen
    const animating = beginOpeningAnimation(this.requireState(), owned.id)
    if (animating.phase !== 'animating') return
    this.publishState(animating, { message: 'Showing the course line…', feedback: null })

    const animation = await animateAppliedMoves({
      port: this.animationPort(),
      startingFen: owned.request.fen,
      appliedMoves: frames,
      ...(optimisticUci ? { optimisticUci } : {}),
      finalFen,
      reducedMotion: this.view.reducedMotion,
      signal: owned.controller.signal,
      onStep: (kind) => this.sound?.play(kind)
    })
    const state = this.requireState()
    if (!this.isCurrent(owned) || animation.status === 'aborted' ||
      state.phase !== 'animating' || state.requestId !== owned.id) return

    const message = result.message || (owned.request.operation === 'reveal'
      ? 'Course move shown.'
      : owned.request.operation === 'move' ? 'Course move found.' : 'Step complete.')
    const lastMove = frames.length > 0 ? moveSquares(frames[frames.length - 1].uci) : undefined
    const complete = completeOpeningStep(state, owned.id, finalFen, result.session, message)
    this.publishState(complete, {
      message,
      feedback: 'feedback' in result ? result.feedback ?? null : null,
      announcement: message,
      notice: animation.warning ?? '',
      ...(lastMove ? { lastMove } : {})
    })
    this.applyBoardWarnings()
    if (owned.request.operation === 'move') this.sound?.play('correct')
    this.events.persisted(result.session)
  }

  private async failRequest(owned: OwnedRequest, error: unknown, reconcile: boolean): Promise<void> {
    if (!this.acceptsOwnedResponse(owned)) return
    let warning = ''
    if (reconcile) {
      warning = await this.reconcilePosition(
        owned.request.fen,
        owned.controller,
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

  private async reconcilePosition(
    fen: string,
    controller: AbortController,
    animate: boolean
  ): Promise<string> {
    try {
      if (!this.board) throw new Error('Chess board is unavailable')
      this.updateStateFen(fen, undefined)
      this.board.setPosition(fen, undefined, animate)
      if (animate) await this.effects.delay(180, controller.signal)
      return ''
    } catch (error) {
      if (controller.signal.aborted) return ''
      this.recoverBoard(fen)
      return `Board reconciliation failed: ${errorMessage(error)}. The saved position was restored.`
    }
  }

  private animationPort() {
    return {
      setPosition: (frame: PositionFrame) => {
        if (!this.board) throw new Error('Chess board is unavailable')
        this.updateStateFen(frame.fen, frame.lastMove)
        this.board.setPosition(frame.fen, frame.lastMove, frame.animate)
      },
      delay: this.effects.delay,
      recover: (finalFen: string) => this.recoverBoard(finalFen)
    }
  }

  private recoverBoard(finalFen: string): void {
    this.recoveringBoard = true
    this.updateStateFen(finalFen, undefined, this.view.boardGeneration + 1)
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
    const messages = [warning, this.view.notice, this.pendingBoardWarning].filter(Boolean)
    this.pendingBoardWarning = ''
    this.recoveringBoard = false
    if (messages.length > 0) this.update({ notice: [...new Set(messages)].join(' ') })
  }

  private isCurrent(token: RequestToken): boolean {
    return this.mounted && this.requests.isCurrent(token)
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
      notice: overrides.notice ?? current?.notice ?? '',
      announcement: overrides.announcement ?? current?.announcement ?? '',
      boardGeneration: overrides.boardGeneration ?? current?.boardGeneration ?? 0,
      reducedMotion: overrides.reducedMotion ?? current?.reducedMotion ?? false,
      soundMuted: overrides.soundMuted ?? current?.soundMuted ?? false,
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

function requestMessage(operation: OpeningOperation): string {
  switch (operation) {
    case 'advance': return 'Preparing the next step…'
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
