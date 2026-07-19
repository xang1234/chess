import type {
  ActiveSessionView,
  CompletedMoveResult,
  CompletedSessionView
} from '../../lib/api'
import type { SoundService } from '../../lib/sound'
import { fakeAPI } from '../../test-fakes'
import type { BoardEffects } from '../chess/board-effects'
import {
  createPuzzleController,
  type PuzzleBoardPort,
  type PuzzleControllerEvents
} from './puzzle-controller'
import type { PuzzleRenderState } from './puzzle-view'

const fen = '4k3/8/8/4p3/8/8/4P3/4K3 w - - 0 2'
const finalFen = '4k3/8/8/4p3/4P3/8/8/4K3 b - - 0 2'

function activeSession(fingerprint = 'puzzle-1', index = 0): ActiveSessionView {
  return {
    sessionId: 'session-1',
    mode: 'guided',
    status: 'active',
    currentIndex: index,
    total: 2,
    current: {
      fingerprint,
      displayedFen: fen,
      currentFen: fen,
      solver: 'white',
      currentPath: [0],
      puzzleNumber: index + 1,
      puzzleTotal: 2,
      hintLevel: 0,
      incorrectMoves: 0,
      canReveal: false,
      legalMoves: ['e2e3', 'e2e4']
    }
  }
}

function completedSession(): CompletedSessionView {
  return {
    sessionId: 'session-1',
    mode: 'guided',
    status: 'completed',
    currentIndex: 2,
    total: 2,
    summary: {
      total: 2,
      firstTry: 2,
      retried: 0,
      usedHint: 0,
      revealed: 0,
      unavailable: 0
    }
  }
}

function completedResult(): CompletedMoveResult {
  return {
    session: completedSession(),
    correct: true,
    puzzleCompleted: true,
    appliedMoves: [{ uci: 'e2e4', resultingFen: finalFen }],
    finalFen
  }
}

function sound(): SoundService {
  return {
    muted: false,
    unlock: async () => {},
    play: vi.fn(),
    setMuted: vi.fn(),
    toggleMuted: vi.fn(() => true),
    destroy: vi.fn()
  }
}

function effects(overrides: Partial<BoardEffects> = {}): BoardEffects {
  return {
    createSound: sound,
    delay: async () => {},
    prefersReducedMotion: () => true,
    ...overrides
  }
}

function events(): PuzzleControllerEvents & {
  change: ReturnType<typeof vi.fn>
  persisted: ReturnType<typeof vi.fn>
  home: ReturnType<typeof vi.fn>
} {
  return { change: vi.fn(), persisted: vi.fn(), home: vi.fn() }
}

function requireSummary(view: PuzzleRenderState) {
  if (view.kind !== 'summary') throw new Error('expected summary view')
  return view
}

test('owns the solved-to-summary transition as one discriminated render state', async () => {
  const emitted = events()
  const controller = createPuzzleController({
    api: fakeAPI({ playMove: async () => completedResult() }),
    effects: effects(),
    events: emitted,
    afterRender: async () => {}
  })
  const board: PuzzleBoardPort = { setPosition: vi.fn() }
  controller.attachBoard(board)
  controller.mount(activeSession())

  expect(controller.view.kind).toBe('puzzle')
  await controller.play('e2e4')

  expect(controller.view.kind).toBe('puzzle')
  if (controller.view.kind !== 'puzzle') throw new Error('expected solved puzzle view')
  expect(controller.view.state.phase).toBe('solved')
  expect(emitted.persisted).toHaveBeenCalledWith(completedSession())

  controller.acknowledgeSolution()

  const summary = requireSummary(controller.view)
  expect(summary.session.summary.total).toBe(2)
  expect(emitted.change).toHaveBeenLastCalledWith(completedSession())
})

test('aborts an owned request before adopting a newer parent session', async () => {
  let resolveMove!: (result: CompletedMoveResult) => void
  const move = new Promise<CompletedMoveResult>((resolve) => { resolveMove = resolve })
  const controller = createPuzzleController({
    api: fakeAPI({ playMove: async () => move }),
    effects: effects(),
    events: events(),
    afterRender: async () => {}
  })
  controller.attachBoard({ setPosition: vi.fn() })
  controller.mount(activeSession())

  const pending = controller.play('e2e4')
  controller.receiveSession(activeSession('puzzle-2', 1))
  resolveMove(completedResult())
  await pending

  expect(controller.view.kind).toBe('puzzle')
  if (controller.view.kind !== 'puzzle') throw new Error('expected puzzle view')
  expect(controller.view.state.displaySession.current.fingerprint).toBe('puzzle-2')
  expect(controller.view.state.phase).toBe('ready')
})

test('does not navigate home when a newer session arrives during pause rendering', async () => {
  let finishRender!: () => void
  const renderBoundary = new Promise<void>((resolve) => { finishRender = resolve })
  const emitted = events()
  const controller = createPuzzleController({
    api: fakeAPI({ pauseSession: async () => {} }),
    effects: effects(),
    events: emitted,
    afterRender: async () => renderBoundary
  })
  controller.attachBoard({ setPosition: vi.fn() })
  controller.mount(activeSession())

  const pending = controller.pause()
  await Promise.resolve()
  controller.receiveSession(activeSession('puzzle-2', 1))
  finishRender()
  await pending

  expect(emitted.home).not.toHaveBeenCalled()
  expect(controller.view.kind).toBe('puzzle')
  if (controller.view.kind !== 'puzzle') throw new Error('expected puzzle view')
  expect(controller.view.state.displaySession.current.fingerprint).toBe('puzzle-2')
})

test('rejects a completed move that points back to the same puzzle', async () => {
  const samePuzzle: CompletedMoveResult = {
    ...completedResult(),
    session: activeSession()
  }
  const controller = createPuzzleController({
    api: fakeAPI({ playMove: async () => samePuzzle }),
    effects: effects(),
    events: events(),
    afterRender: async () => {}
  })
  controller.attachBoard({ setPosition: vi.fn() })
  controller.mount(activeSession())

  await controller.play('e2e4')

  expect(controller.view.kind).toBe('puzzle')
  if (controller.view.kind !== 'puzzle') throw new Error('expected puzzle view')
  expect(controller.view.state.phase).toBe('failed')
  if (controller.view.state.phase !== 'failed') throw new Error('expected failed state')
  expect(controller.view.state.message).toMatch(/did not advance/i)
})

test.each([
  {
    name: 'changes only the puzzle fingerprint',
    session: activeSession('replacement-puzzle', 0)
  },
  {
    name: 'changes only the session identifier',
    session: { ...activeSession(), sessionId: 'replacement-session' }
  }
])('rejects a completed move that $name without increasing the index', async ({ session }) => {
  const malformed: CompletedMoveResult = {
    ...completedResult(),
    session
  }
  const controller = createPuzzleController({
    api: fakeAPI({ playMove: async () => malformed }),
    effects: effects(),
    events: events(),
    afterRender: async () => {}
  })
  controller.attachBoard({ setPosition: vi.fn() })
  controller.mount(activeSession())

  await controller.play('e2e4')

  expect(controller.view.kind).toBe('puzzle')
  if (controller.view.kind !== 'puzzle') throw new Error('expected puzzle view')
  expect(controller.view.state.phase).toBe('failed')
  if (controller.view.state.phase !== 'failed') throw new Error('expected failed state')
  expect(controller.view.state.message).toMatch(/same session.*higher puzzle index/i)
})

test('recovers a detached board through generation replacement and retains the warning', async () => {
  const controller = createPuzzleController({
    api: fakeAPI({ playMove: async () => completedResult() }),
    effects: effects(),
    events: events(),
    afterRender: async () => {}
  })
  const board: PuzzleBoardPort = {
    setPosition: vi.fn(() => { throw new Error('board detached') })
  }
  controller.attachBoard(board)
  controller.mount(activeSession())
  const initialGeneration = controller.view.boardGeneration

  await controller.play('e2e4')

  expect(controller.view.kind).toBe('puzzle')
  if (controller.view.kind !== 'puzzle') throw new Error('expected puzzle view')
  expect(controller.view.boardGeneration).toBeGreaterThan(initialGeneration)
  expect(controller.view.state.notice).toMatch(/board detached.*final position was restored/i)
})
