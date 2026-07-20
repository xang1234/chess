import type {
  ActiveOpeningSessionView,
  CompletedOpeningSessionView,
  OpeningHintResult,
  OpeningMoveFeedback,
  OpeningSessionView,
  RestartRequiredOpeningSessionView
} from '../../lib/api'

export type OpeningOperation = 'advance' | 'move' | 'hint' | 'reveal' | 'pause' | 'restart'

export type OpeningState =
  | { phase: 'passive'; session: ActiveOpeningSessionView; fen: string }
  | { phase: 'ready'; session: ActiveOpeningSessionView; fen: string; hint: OpeningHintResult | null }
  | {
    phase: 'requesting'
    session: ActiveOpeningSessionView
    fen: string
    requestId: number
    operation: OpeningOperation
  }
  | {
    phase: 'animating'
    session: ActiveOpeningSessionView
    fen: string
    requestId: number
  }
  | {
    phase: 'step-complete'
    session: ActiveOpeningSessionView
    fen: string
    pending: OpeningSessionView
    message: string
  }
  | { phase: 'summary'; session: CompletedOpeningSessionView }
  | { phase: 'restart-required'; session: RestartRequiredOpeningSessionView }
  | {
    phase: 'failed'
    session: ActiveOpeningSessionView
    fen: string
    message: string
    recoverable: boolean
  }

function activeState(session: ActiveOpeningSessionView): OpeningState {
  if (session.current.kind === 'concept' || session.current.kind === 'demonstration') {
    return { phase: 'passive', session, fen: session.current.currentFen }
  }
  return { phase: 'ready', session, fen: session.current.currentFen, hint: null }
}

export function initialiseOpening(session: OpeningSessionView): OpeningState {
  switch (session.status) {
    case 'active': return activeState(session)
    case 'completed': return { phase: 'summary', session }
    case 'restart_required': return { phase: 'restart-required', session }
  }
}

export function acceptsOpeningInput(state: OpeningState): boolean {
  return state.phase === 'ready' || (state.phase === 'failed' && state.recoverable)
}

export function acceptsOpeningResponse(
  state: OpeningState,
  requestId: number
): state is Extract<OpeningState, { phase: 'requesting' }> {
  return state.phase === 'requesting' && state.requestId === requestId
}

export function beginOpeningRequest(
  state: OpeningState,
  operation: OpeningOperation,
  requestId: number
): OpeningState {
  if (!Number.isSafeInteger(requestId) || requestId <= 0) {
    throw new Error(`request ID must be a positive integer, got ${requestId}`)
  }

  const active = state.phase === 'passive' || state.phase === 'ready' ||
    (state.phase === 'failed' && state.recoverable)
  if (!active) throw new Error(`${state.phase} state is locked`)
  if (state.phase === 'passive' && operation !== 'advance' && operation !== 'pause') {
    throw new Error(`${operation} is unavailable for a teaching step`)
  }
  if (state.phase !== 'passive' && operation === 'advance') {
    throw new Error('advance is available only for a teaching step')
  }

  return {
    phase: 'requesting',
    session: state.session,
    fen: state.fen,
    requestId,
    operation
  }
}

export function beginOpeningAnimation(state: OpeningState, requestId: number): OpeningState {
  if (!acceptsOpeningResponse(state, requestId)) return state
  if (state.operation !== 'advance' && state.operation !== 'move' && state.operation !== 'reveal') {
    throw new Error(`${state.operation} response cannot begin move animation`)
  }
  return {
    phase: 'animating',
    session: state.session,
    fen: state.fen,
    requestId
  }
}

export function markOpeningFeedback(
  state: OpeningState,
  requestId: number,
  session: ActiveOpeningSessionView,
  feedback: Extract<OpeningMoveFeedback, 'alternative' | 'off_course'>
): OpeningState {
  if (!acceptsOpeningResponse(state, requestId)) return state
  if (state.operation !== 'move') throw new Error(`${state.operation} cannot return move feedback`)
  if (feedback !== 'alternative' && feedback !== 'off_course') {
    throw new Error(`unsupported feedback ${String(feedback)}`)
  }
  return {
    phase: 'ready',
    session,
    fen: session.current.currentFen,
    hint: null
  }
}

export function finishOpeningHint(
  state: OpeningState,
  requestId: number,
  hint: OpeningHintResult
): OpeningState {
  if (!acceptsOpeningResponse(state, requestId)) return state
  if (state.operation !== 'hint') throw new Error(`${state.operation} cannot return a hint`)
  return {
    phase: 'ready',
    session: hint.session,
    fen: hint.session.current.currentFen,
    hint
  }
}

export function completeOpeningStep(
  state: OpeningState,
  requestId: number,
  finalFen: string,
  pending: OpeningSessionView,
  message: string
): OpeningState {
  if (state.phase !== 'animating' || state.requestId !== requestId) return state
  if (!finalFen) throw new Error('completed opening step requires a final FEN')
  return {
    phase: 'step-complete',
    session: state.session,
    fen: finalFen,
    pending,
    message
  }
}

export function acknowledgeOpeningStep(state: OpeningState): OpeningState {
  if (state.phase !== 'step-complete') {
    throw new Error(`${state.phase} state cannot acknowledge a completed step`)
  }
  return initialiseOpening(state.pending)
}

export function failOpeningRequest(
  state: OpeningState,
  message: string,
  recoverable: boolean
): OpeningState {
  if (state.phase === 'summary' || state.phase === 'restart-required') return state
  if (state.phase === 'step-complete') {
    throw new Error('completed opening step cannot fail a request')
  }
  return {
    phase: 'failed',
    session: state.session,
    fen: state.fen,
    message,
    recoverable
  }
}
