import type { HintResult, SessionView } from '../../lib/api'
import { moveSquares, type Square } from '../../lib/uci'

export type Operation = 'move' | 'hint' | 'reveal' | 'pause'
export type SolvedOutcome = 'correct' | 'revealed'

type Common = {
  displaySession: SessionView
  fen: string
  hint: HintResult | null
  lastMove?: [Square, Square]
  notice?: string
}

export type PuzzleState =
  | (Common & { phase: 'prelude' })
  | (Common & { phase: 'ready' })
  | (Common & {
    phase: 'requesting'
    operation: Operation
    requestId: number
    authoritativeFen: string
    submittedUci?: string
  })
  | (Common & {
    phase: 'animating'
    operation: 'move' | 'reveal'
    requestId: number
  })
  | (Common & {
    phase: 'incorrect'
    wrongMove: [Square, Square]
    message: string
  })
  | (Common & {
    phase: 'solved'
    outcome: SolvedOutcome
    finalFen: string
    pendingSession: SessionView
  })
  | (Common & {
    phase: 'failed'
    message: string
    recoverable: boolean
  })

export type SolvedAcknowledgement =
  | { kind: 'puzzle'; state: PuzzleState }
  | { kind: 'summary'; session: SessionView }

function currentPuzzle(session: SessionView) {
  if (!session.current) throw new Error('active puzzle state requires a current puzzle')
  return session.current
}

function common(
  state: PuzzleState,
  overrides: Partial<Common> = {}
): Common {
  const replacesLastMove = Object.prototype.hasOwnProperty.call(overrides, 'lastMove')
  const replacesNotice = Object.prototype.hasOwnProperty.call(overrides, 'notice')
  return {
    displaySession: overrides.displaySession ?? state.displaySession,
    fen: overrides.fen ?? state.fen,
    hint: overrides.hint === undefined ? state.hint : overrides.hint,
    ...(replacesLastMove
      ? overrides.lastMove === undefined ? {} : { lastMove: overrides.lastMove }
      : state.lastMove === undefined ? {} : { lastMove: state.lastMove }),
    ...(replacesNotice
      ? overrides.notice === undefined ? {} : { notice: overrides.notice }
      : state.notice === undefined ? {} : { notice: state.notice })
  }
}

function requireRequest(state: PuzzleState, requestId: number) {
  if (state.phase !== 'requesting') {
    throw new Error(`${state.phase} state has no pending response`)
  }
  if (state.requestId !== requestId) {
    throw new Error(`stale response ${requestId}; waiting for ${state.requestId}`)
  }
  return state
}

export function initialisePuzzle(session: SessionView, reducedMotion: boolean): PuzzleState {
  const current = currentPuzzle(session)
  const showPrelude = !reducedMotion && current.currentPath.length === 0 &&
    Boolean(current.sourceFen && current.preludeUci)
  return {
    phase: showPrelude ? 'prelude' : 'ready',
    displaySession: session,
    fen: showPrelude ? current.sourceFen! : current.currentFen,
    hint: null
  }
}

export function finishPrelude(state: PuzzleState): PuzzleState {
  if (state.phase !== 'prelude') {
    throw new Error(`${state.phase} state cannot finish a prelude`)
  }
  const current = currentPuzzle(state.displaySession)
  if (!current.preludeUci) throw new Error('prelude state has no prelude move')
  return {
    ...common(state, { fen: current.currentFen }),
    phase: 'ready',
    lastMove: moveSquares(current.preludeUci)
  }
}

export function beginRequest(
  state: PuzzleState,
  operation: Operation,
  requestId: number,
  submittedUci?: string
): PuzzleState {
  const allowed = state.phase === 'ready' || state.phase === 'incorrect' ||
    (state.phase === 'failed' && state.recoverable)
  if (!allowed) {
    if (state.phase === 'failed') throw new Error('fatal failed state is locked')
    throw new Error(`${state.phase} state is locked`)
  }
  if (!Number.isSafeInteger(requestId) || requestId <= 0) {
    throw new Error(`request ID must be a positive integer, got ${requestId}`)
  }
  return {
    ...common(state, { notice: undefined }),
    phase: 'requesting',
    operation,
    requestId,
    authoritativeFen: state.fen,
    ...(submittedUci ? { submittedUci } : {})
  }
}

export function acceptsResponse(state: PuzzleState, requestId: number): boolean {
  return state.phase === 'requesting' && state.requestId === requestId
}

export function beginAnimation(state: PuzzleState, requestId: number): PuzzleState {
  const request = requireRequest(state, requestId)
  if (request.operation !== 'move' && request.operation !== 'reveal') {
    throw new Error(`${request.operation} response cannot begin move animation`)
  }
  return {
    ...common(request),
    phase: 'animating',
    operation: request.operation,
    requestId
  }
}

export function finishReadyRequest(
  state: PuzzleState,
  requestId: number,
  returnedSession: SessionView,
  hint: HintResult | null
): PuzzleState {
  const request = requireRequest(state, requestId)
  if (request.operation !== 'hint' && request.operation !== 'move') {
    throw new Error(`${request.operation} response cannot return directly to ready`)
  }
  const returned = currentPuzzle(returnedSession)
  const displayed = currentPuzzle(request.displaySession)
  if (returned.fingerprint !== displayed.fingerprint) {
    throw new Error('ready response advanced to a different puzzle')
  }
  return {
    ...common(request, {
      displaySession: returnedSession,
      fen: returned.currentFen,
      hint,
      notice: undefined
    }),
    phase: 'ready'
  }
}

export function markIncorrect(
  state: PuzzleState,
  requestId: number,
  returnedSession: SessionView,
  uci: string,
  message: string
): PuzzleState {
  const request = requireRequest(state, requestId)
  if (request.operation !== 'move') {
    throw new Error(`${request.operation} response cannot be marked incorrect`)
  }
  const returned = currentPuzzle(returnedSession)
  const displayed = currentPuzzle(request.displaySession)
  if (returned.fingerprint !== displayed.fingerprint) {
    throw new Error('incorrect response advanced to a different puzzle')
  }
  if (returned.currentFen !== request.authoritativeFen) {
    throw new Error('incorrect response did not preserve the authoritative FEN')
  }
  return {
    ...common(request, {
      displaySession: returnedSession,
      fen: request.authoritativeFen,
      notice: undefined
    }),
    phase: 'incorrect',
    wrongMove: moveSquares(uci),
    message
  }
}

export function markSolved(
  state: PuzzleState,
  requestId: number,
  outcome: SolvedOutcome,
  finalFen: string,
  pendingSession: SessionView,
  lastMove?: [Square, Square]
): PuzzleState {
  if (state.phase !== 'animating' || state.requestId !== requestId) {
    throw new Error(`${state.phase} state cannot finish solved request ${requestId}`)
  }
  if (!finalFen) throw new Error('solved state requires a final FEN')
  return {
    ...common(state, {
      fen: finalFen,
      ...(lastMove ? { lastMove } : {}),
      notice: undefined
    }),
    phase: 'solved',
    outcome,
    finalFen,
    pendingSession
  }
}

export function acknowledgeSolved(
  state: PuzzleState,
  reducedMotion: boolean
): SolvedAcknowledgement {
  if (state.phase !== 'solved') {
    throw new Error(`${state.phase} state cannot acknowledge a solution`)
  }
  if (state.pendingSession.current) {
    return {
      kind: 'puzzle',
      state: initialisePuzzle(state.pendingSession, reducedMotion)
    }
  }
  if (state.pendingSession.summary) {
    return { kind: 'summary', session: state.pendingSession }
  }
  throw new Error('solved pending session has neither a puzzle nor a summary')
}

export function markFailed(
  state: PuzzleState,
  message: string,
  recoverable: boolean
): PuzzleState {
  if (state.phase === 'solved') throw new Error('solved state cannot fail a request')
  const fen = state.phase === 'requesting' ? state.authoritativeFen : state.fen
  return {
    ...common(state, { fen }),
    phase: 'failed',
    message,
    recoverable
  }
}
