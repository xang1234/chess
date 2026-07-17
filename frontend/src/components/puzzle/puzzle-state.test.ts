import type { HintResult, PuzzleView, SessionView } from '../../lib/api'
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
  markSolved
} from './puzzle-state'

const sourceFen = '4k3/4p3/8/8/8/8/4P3/4K3 b - - 0 1'
const currentFen = '4k3/8/8/4p3/8/8/4P3/4K3 w - - 0 2'
const finalFen = '4k3/8/8/4p3/4P3/8/8/4K3 b - - 0 2'
const legalMoves = ['e1d1', 'e1d2', 'e1f1', 'e1f2', 'e2e3', 'e2e4']

function activeSession(overrides: Partial<PuzzleView> = {}): SessionView {
  return {
    sessionId: 'session-1',
    mode: 'guided',
    status: 'active',
    currentIndex: 0,
    total: 2,
    current: {
      fingerprint: 'puzzle-1',
      sourceFen,
      displayedFen: currentFen,
      currentFen,
      preludeUci: 'e7e5',
      solver: 'white',
      currentPath: [],
      puzzleNumber: 1,
      puzzleTotal: 2,
      hintLevel: 0,
      incorrectMoves: 0,
      canReveal: false,
      legalMoves,
      ...overrides
    }
  }
}

function nextSession(): SessionView {
  const session = activeSession({
    fingerprint: 'puzzle-2',
    sourceFen: undefined,
    preludeUci: undefined,
    displayedFen: finalFen,
    currentFen: finalFen,
    puzzleNumber: 2,
    legalMoves: ['e8d7', 'e8d8', 'e8e7', 'e8f7', 'e8f8']
  })
  session.currentIndex = 1
  return session
}

function completedSession(): SessionView {
  return {
    sessionId: 'session-1',
    mode: 'guided',
    status: 'completed',
    currentIndex: 2,
    total: 2,
    summary: {
      total: 2,
      firstTry: 1,
      retried: 1,
      usedHint: 0,
      revealed: 0,
      unavailable: 0
    }
  }
}

test('initialises fresh, resumed, and reduced-motion puzzles correctly', () => {
  const fresh = initialisePuzzle(activeSession(), false)
  expect(fresh).toMatchObject({ phase: 'prelude', fen: sourceFen, hint: null })

  const afterPrelude = finishPrelude(fresh)
  expect(afterPrelude).toMatchObject({
    phase: 'ready',
    fen: currentFen,
    lastMove: ['e7', 'e5']
  })

  const resumed = initialisePuzzle(
    activeSession({ currentPath: [0, 0], currentFen: finalFen }),
    false
  )
  expect(resumed).toMatchObject({ phase: 'ready', fen: finalFen })

  const reduced = initialisePuzzle(activeSession(), true)
  expect(reduced).toMatchObject({ phase: 'ready', fen: currentFen })
})

test('records requests and accepts only the matching response identity', () => {
  const ready = { ...initialisePuzzle(activeSession(), true), notice: 'Board restored' }
  const request = beginRequest(ready, 'move', 7, 'e2e4')

  expect(request).toMatchObject({
    phase: 'requesting',
    operation: 'move',
    requestId: 7,
    authoritativeFen: currentFen,
    submittedUci: 'e2e4'
  })
  expect(acceptsResponse(request, 7)).toBe(true)
  expect(acceptsResponse(request, 6)).toBe(false)
  expect(acceptsResponse(ready, 7)).toBe(false)
  expect(request.notice).toBeUndefined()
  expect(() => beginRequest(request, 'hint', 8)).toThrow(/requesting state is locked/)
})

test('finishes a same-puzzle hint without replaying the prelude', () => {
  const ready = initialisePuzzle(activeSession(), true)
  const request = beginRequest(ready, 'hint', 1)
  const returned = activeSession({
    hintLevel: 2,
    incorrectMoves: 1,
    legalMoves: ['e1d1', 'e2e4']
  })
  const hint: HintResult = {
    session: returned,
    level: 2,
    text: 'Start with this piece.',
    sourceSquare: 'e2',
    canReveal: false
  }

  const finished = finishReadyRequest(request, 1, returned, hint)

  expect(finished).toMatchObject({
    phase: 'ready',
    fen: currentFen,
    hint,
    displaySession: returned
  })
})

test('marks an incorrect move only after authoritative reconciliation', () => {
  const request = beginRequest(initialisePuzzle(activeSession(), true), 'move', 2, 'e2e3')
  const returned = activeSession({
    incorrectMoves: 1,
    legalMoves: ['e1d1', 'e2e3', 'e2e4']
  })

  const incorrect = markIncorrect(request, 2, returned, 'e2e3', 'Try again')

  expect(incorrect).toMatchObject({
    phase: 'incorrect',
    fen: currentFen,
    wrongMove: ['e2', 'e3'],
    message: 'Try again',
    displaySession: returned
  })
  const retry = beginRequest(incorrect, 'move', 3, 'e2e4')
  expect(retry).toMatchObject({ phase: 'requesting', requestId: 3 })
})

test('retains the solved board until next puzzle acknowledgement', () => {
  const request = beginRequest(initialisePuzzle(activeSession(), true), 'move', 4, 'e2e4')
  const animating = beginAnimation(request, 4)
  const pending = nextSession()

  const solved = markSolved(animating, 4, 'correct', finalFen, pending, ['e2', 'e4'])

  expect(solved).toMatchObject({
    phase: 'solved',
    displaySession: request.displaySession,
    fen: finalFen,
    finalFen,
    pendingSession: pending,
    lastMove: ['e2', 'e4']
  })
  const acknowledged = acknowledgeSolved(solved, false)
  expect(acknowledged).toMatchObject({
    kind: 'puzzle',
    state: { phase: 'ready', displaySession: pending, fen: finalFen }
  })
})

test('acknowledges the final solved state as a summary', () => {
  const request = beginRequest(initialisePuzzle(activeSession(), true), 'reveal', 5)
  const solved = markSolved(
    beginAnimation(request, 5),
    5,
    'revealed',
    finalFen,
    completedSession()
  )

  expect(acknowledgeSolved(solved, false)).toEqual({
    kind: 'summary',
    session: completedSession()
  })
})

test('allows retry only for explicitly recoverable failures', () => {
  const request = beginRequest(initialisePuzzle(activeSession(), true), 'move', 6, 'e2e4')
  const recoverable = markFailed(request, 'Network request failed', true)
  expect(recoverable).toMatchObject({
    phase: 'failed',
    recoverable: true,
    fen: currentFen
  })
  expect(beginRequest(recoverable, 'move', 7, 'e2e4')).toMatchObject({
    phase: 'requesting',
    requestId: 7
  })

  const fatal = markFailed(
    initialisePuzzle(activeSession(), true),
    'Legal move data is malformed',
    false
  )
  expect(fatal).toMatchObject({ phase: 'failed', recoverable: false })
  expect(() => beginRequest(fatal, 'move', 8, 'e2e4')).toThrow(/fatal failed state is locked/)
})
