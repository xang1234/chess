import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import type { Key, KeyPair } from '@lichess-org/chessground/types'
import type {
  ActiveSessionView,
  CompletedSessionView,
  HintResult,
  MoveResult,
  PuzzleView,
  SessionView
} from '../../lib/api'
import type { SoundService } from '../../lib/sound'
import { fakeAPI, withNormalAPI } from '../../test-fakes'
import PuzzleScreen from './PuzzleScreen.svelte'
import type {
  BoardCallbacks,
  BoardInteraction,
  ChessBoardAdapter,
  ChessgroundAdapterFactory
} from '../chess/chessground-adapter'

type PuzzleEffects = {
  createSound(): SoundService
  delay(milliseconds: number, signal: AbortSignal): Promise<void>
  prefersReducedMotion(): boolean
}

const sourceFen = '4k3/4p3/8/8/8/8/4P3/4K3 b - - 0 1'
const puzzleFen = '4k3/8/8/4p3/8/8/4P3/4K3 w - - 0 2'
const afterUserFen = '4k3/8/8/4p3/4P3/8/8/4K3 b - - 0 2'
const afterReplyFen = '4k3/8/8/8/4p3/8/8/4K3 w - - 0 3'
const nextFen = '4k3/8/8/8/8/8/3P4/4K3 b - - 0 1'
const puzzleLegalMoves = ['e1d1', 'e1d2', 'e1f1', 'e1f2', 'e2e3', 'e2e4']
const replyLegalMoves = ['e1d1', 'e1d2', 'e1e2', 'e1f1', 'e1f2']
const nextLegalMoves = ['e8d7', 'e8d8', 'e8e7', 'e8f7', 'e8f8']

function activeSession(
  puzzleOverrides: Partial<PuzzleView> = {},
  sessionOverrides: Partial<Pick<ActiveSessionView, 'sessionId' | 'mode' | 'currentIndex' | 'total'>> = {}
): ActiveSessionView {
  return {
    sessionId: 'session-1',
    mode: 'guided',
    status: 'active',
    currentIndex: 0,
    total: 2,
    current: {
      fingerprint: 'puzzle-1',
      sourceFen,
      displayedFen: puzzleFen,
      currentFen: puzzleFen,
      preludeUci: 'e7e5',
      solver: 'white',
      currentPath: [],
      puzzleNumber: 1,
      puzzleTotal: 2,
      hintLevel: 0,
      incorrectMoves: 0,
      canReveal: false,
      legalMoves: [...puzzleLegalMoves],
      ...puzzleOverrides
    },
    ...sessionOverrides
  }
}

function resumedSession(puzzleOverrides: Partial<PuzzleView> = {}): ActiveSessionView {
  return activeSession({
    sourceFen: undefined,
    preludeUci: undefined,
    currentPath: [0],
    ...puzzleOverrides
  })
}

function nextSession(): ActiveSessionView {
  return activeSession({
    fingerprint: 'puzzle-2',
    sourceFen: undefined,
    displayedFen: nextFen,
    currentFen: nextFen,
    preludeUci: undefined,
    solver: 'black',
    currentPath: [],
    puzzleNumber: 2,
    legalMoves: [...nextLegalMoves]
  }, { currentIndex: 1 })
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
      firstTry: 1,
      retried: 1,
      usedHint: 0,
      revealed: 0,
      unavailable: 0
    }
  }
}

function correctResult(pending: SessionView = nextSession()): MoveResult {
  return {
    session: pending,
    correct: true,
    puzzleCompleted: true,
    appliedMoves: [{ uci: 'e2e4', resultingFen: afterUserFen }],
    finalFen: afterUserFen
  }
}

function continuingResult(): MoveResult {
  return {
    session: activeSession({
      sourceFen: undefined,
      preludeUci: undefined,
      displayedFen: puzzleFen,
      currentFen: afterReplyFen,
      currentPath: [0, 0],
      legalMoves: [...replyLegalMoves]
    }),
    correct: true,
    puzzleCompleted: false,
    appliedMoves: [
      { uci: 'e2e4', resultingFen: afterUserFen },
      { uci: 'e5e4', resultingFen: afterReplyFen }
    ]
  }
}

function revealResult(pending: SessionView = nextSession()): MoveResult {
  return {
    session: pending,
    correct: true,
    puzzleCompleted: true,
    appliedMoves: [
      { uci: 'e2e4', resultingFen: afterUserFen },
      { uci: 'e5e4', resultingFen: afterReplyFen }
    ],
    finalFen: afterReplyFen
  }
}

function malformedMoveResult(value: unknown): MoveResult {
  return value as MoveResult
}

type BoardCreation = {
  adapter: ChessBoardAdapter
  callbacks: BoardCallbacks
  fen: string
  interactions: BoardInteraction[]
  setPosition: ReturnType<typeof vi.fn>
}

function boardHarness(
  setPositionImpl: (creation: number, fen: string, lastMove?: KeyPair, animate?: boolean) => void = () => {}
) {
  const creations: BoardCreation[] = []
  const factory: ChessgroundAdapterFactory = vi.fn((_element, fen, interaction, callbacks) => {
    const index = creations.length
    const interactions = [interaction]
    const setPosition = vi.fn((nextFen: string, lastMove?: KeyPair, animate?: boolean) => {
      setPositionImpl(index, nextFen, lastMove, animate)
    })
    const adapter: ChessBoardAdapter = {
      configure: vi.fn((next) => { interactions.push(next) }),
      setPosition,
      selectSquare: vi.fn(),
      destroy: vi.fn()
    }
    creations.push({ adapter, callbacks, fen, interactions, setPosition })
    return adapter
  })
  return { creations, factory }
}

function soundHarness(initialMuted = false, sequence: string[] = []) {
  let muted = initialMuted
  const unlock = vi.fn(async () => { sequence.push('unlock') })
  const play = vi.fn((name: string) => { sequence.push(`sound:${name}`) })
  const destroy = vi.fn()
  const service: SoundService = {
    get muted() { return muted },
    unlock,
    play,
    setMuted: (value) => { muted = value },
    toggleMuted: () => {
      muted = !muted
      return muted
    },
    destroy
  }
  return { destroy, play, service, unlock }
}

function effectsHarness(options: {
  reducedMotion?: boolean
  delay?: (milliseconds: number, signal: AbortSignal) => Promise<void>
  sound?: ReturnType<typeof soundHarness>
} = {}) {
  const delays: number[] = []
  const signals: AbortSignal[] = []
  const sound = options.sound ?? soundHarness()
  const delay = vi.fn(async (milliseconds: number, signal: AbortSignal) => {
    delays.push(milliseconds)
    signals.push(signal)
    await options.delay?.(milliseconds, signal)
  })
  const effects: PuzzleEffects = {
    createSound: () => sound.service,
    delay,
    prefersReducedMotion: () => options.reducedMotion ?? true
  }
  return { delay, delays, effects, signals, sound }
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((accept, decline) => {
    resolve = accept
    reject = decline
  })
  return { promise, reject, resolve }
}

async function routeMove(
  board: ReturnType<typeof boardHarness>,
  from: Key = 'e2',
  to: Key = 'e4'
) {
  await waitFor(() => expect(board.creations).toHaveLength(1))
  board.creations[0].callbacks.onRoute(from, to)
}

test('plays a fresh-puzzle prelude before enabling input', async () => {
  const gate = deferred<void>()
  const board = boardHarness()
  const effects = effectsHarness({
    reducedMotion: false,
    delay: () => gate.promise
  })
  render(PuzzleScreen, {
    session: activeSession(),
    effects: effects.effects,
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI()))

  expect(await screen.findByText('Watch the last move…')).toBeInTheDocument()
  expect(screen.getByRole('grid')).toHaveAttribute('aria-disabled', 'true')
  expect(board.creations[0].fen).toBe(sourceFen)
  await waitFor(() => expect(board.creations[0].setPosition).toHaveBeenCalledWith(
    puzzleFen,
    ['e7', 'e5'],
    true
  ))
  expect(screen.getByRole('grid')).toHaveAttribute('aria-disabled', 'true')

  gate.resolve()

  await waitFor(() => expect(screen.getByRole('grid')).toHaveAttribute('aria-disabled', 'false'))
  expect(effects.delays).toEqual([180])
})

test('starts a resumed puzzle immediately without replaying the prelude', async () => {
  const board = boardHarness()
  const effects = effectsHarness({ reducedMotion: false })
  render(PuzzleScreen, {
    session: resumedSession({ currentFen: afterUserFen }),
    effects: effects.effects,
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI()))

  await waitFor(() => expect(board.creations).toHaveLength(1))
  expect(board.creations[0].fen).toBe(afterUserFen)
  expect(effects.delay).not.toHaveBeenCalled()
  expect(screen.getByRole('grid')).toHaveAttribute('aria-disabled', 'false')
  expect(screen.queryByText('Watch the last move…')).not.toBeInTheDocument()
})

test('reconciles and waits before showing a wrong-move retry', async () => {
  const gate = deferred<void>()
  const board = boardHarness()
  const effects = effectsHarness({ reducedMotion: false, delay: () => gate.promise })
  const playMove = vi.fn().mockResolvedValue({
    session: resumedSession({ incorrectMoves: 1 }),
    correct: false,
    puzzleCompleted: false,
    message: 'Try again'
  } satisfies MoveResult)
  render(PuzzleScreen, {
    session: resumedSession(),
    effects: effects.effects,
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ playMove })))

  await routeMove(board, 'e2', 'e3')
  await waitFor(() => expect(board.creations[0].setPosition).toHaveBeenCalledWith(
    puzzleFen,
    undefined,
    true
  ))
  expect(screen.queryByText('Try again')).not.toBeInTheDocument()
  expect(screen.getByRole('grid')).toHaveAttribute('aria-disabled', 'true')

  gate.resolve()

  expect(await screen.findByText('Try again')).toBeInTheDocument()
  await waitFor(() => expect(screen.getByRole('grid')).toHaveAttribute('aria-disabled', 'false'))
  expect(board.creations[0].interactions.at(-1)?.wrongMove).toEqual(['e2', 'e3'])
  expect(effects.sound.play).toHaveBeenCalledWith('incorrect')
  expect(playMove).toHaveBeenCalledTimes(1)
})

test('adopts an unavailable move response without treating it as incorrect', async () => {
  const board = boardHarness()
  const effects = effectsHarness()
  const playMove = vi.fn().mockResolvedValue({
    session: nextSession(),
    correct: false,
    puzzleCompleted: false,
    message: 'Puzzle unavailable'
  } satisfies MoveResult)
  render(PuzzleScreen, {
    session: resumedSession(),
    effects: effects.effects,
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ playMove })))

  await routeMove(board)

  expect(await screen.findByText('Puzzle 2 of 2')).toBeInTheDocument()
  expect(screen.getByText('Puzzle unavailable')).toBeInTheDocument()
  expect(effects.sound.play).not.toHaveBeenCalledWith('incorrect')
  expect(board.creations.at(-1)?.interactions.at(-1)?.wrongMove).toBeUndefined()
})

test('restores a rejected optimistic move and permits retry', async () => {
  const board = boardHarness()
  const effects = effectsHarness()
  const playMove = vi.fn()
    .mockRejectedValueOnce(new Error('connection lost'))
    .mockResolvedValueOnce({
      session: resumedSession({ incorrectMoves: 1 }),
      correct: false,
      puzzleCompleted: false,
      message: 'Try again'
    } satisfies MoveResult)
  render(PuzzleScreen, {
    session: resumedSession(),
    effects: effects.effects,
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ playMove })))

  await routeMove(board)

  expect(await screen.findByText(/connection lost/i)).toBeInTheDocument()
  expect(board.creations[0].setPosition).toHaveBeenCalledWith(puzzleFen, undefined, false)
  expect(screen.getByRole('grid')).toHaveAttribute('aria-disabled', 'false')

  board.creations[0].callbacks.onRoute('e2', 'e4')
  expect(await screen.findByText('Try again')).toBeInTheDocument()
  expect(playMove).toHaveBeenCalledTimes(2)
})

test('locks on a successful response missing authoritative frames', async () => {
  const board = boardHarness()
  const playMove = vi.fn().mockResolvedValue(malformedMoveResult({
    session: nextSession(),
    correct: true,
    puzzleCompleted: true
  }))
  render(PuzzleScreen, {
    session: resumedSession(),
    effects: effectsHarness().effects,
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ playMove })))

  await routeMove(board)

  expect(await screen.findByText(/authoritative move frames/i)).toBeInTheDocument()
  expect(screen.getByRole('grid')).toHaveAttribute('aria-disabled', 'true')
})

test('reconciles the optimistic frame, animates the automatic reply, and reopens input', async () => {
  const board = boardHarness()
  const effects = effectsHarness({ reducedMotion: false })
  const playMove = vi.fn().mockResolvedValue(continuingResult())
  render(PuzzleScreen, {
    session: resumedSession(),
    effects: effects.effects,
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ playMove })))

  await routeMove(board)

  await waitFor(() => expect(board.creations[0].setPosition).toHaveBeenCalledTimes(2))
  expect(board.creations[0].setPosition.mock.calls).toEqual([
    [afterUserFen, ['e2', 'e4'], false],
    [afterReplyFen, ['e5', 'e4'], true]
  ])
  expect(effects.delays).toEqual([220, 180])
  expect(effects.sound.play.mock.calls.map(([name]) => name)).toEqual(['move', 'capture'])
  await waitFor(() => expect(screen.getByRole('grid')).toHaveAttribute('aria-disabled', 'false'))
  expect(board.creations[0].interactions.at(-1)?.legalMoves).toEqual(replyLegalMoves)
})

test('reveals the complete line and retains a non-rewarding solved board', async () => {
  const board = boardHarness()
  const effects = effectsHarness({ reducedMotion: false })
  const revealSolution = vi.fn().mockResolvedValue(revealResult())
  const persisted: SessionView[] = []
  const { component } = render(PuzzleScreen, {
    session: resumedSession({ hintLevel: 3, canReveal: true }),
    effects: effects.effects,
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ revealSolution })))
  component.$on('persisted', (event) => persisted.push(event.detail))

  await fireEvent.click(await screen.findByRole('button', { name: 'Show solution' }))

  expect(await screen.findByText('Solution shown')).toBeInTheDocument()
  expect(screen.getByText('Puzzle 1 of 2')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Next puzzle' })).toBeInTheDocument()
  expect(board.creations[0].setPosition.mock.calls).toEqual([
    [afterUserFen, ['e2', 'e4'], true],
    [afterReplyFen, ['e5', 'e4'], true]
  ])
  expect(effects.delays).toEqual([180, 220, 180])
  expect(effects.sound.play).not.toHaveBeenCalledWith('correct')
  expect(persisted).toEqual([nextSession()])
})

test('recovers a throwing adapter at the final FEN and still exposes Next', async () => {
  const board = boardHarness(() => { throw new Error('adapter failed') })
  const effects = effectsHarness({ reducedMotion: false })
  render(PuzzleScreen, {
    session: resumedSession(),
    effects: effects.effects,
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ playMove: async () => correctResult() })))

  await routeMove(board)

  expect(await screen.findByRole('button', { name: 'Next puzzle' })).toBeInTheDocument()
  await waitFor(() => expect(board.creations).toHaveLength(2))
  expect(board.creations[1].fen).toBe(afterUserFen)
  expect(board.creations[0].setPosition).toHaveBeenCalledTimes(1)
  expect(board.creations[1].setPosition).not.toHaveBeenCalled()
  expect(screen.getByText(/animation failed/i)).toBeInTheDocument()
})

test('ignores a stale response after an externally supplied session change', async () => {
  const response = deferred<MoveResult>()
  const board = boardHarness()
  const effects = effectsHarness()
  const { component } = render(PuzzleScreen, {
    session: resumedSession(),
    effects: effects.effects,
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ playMove: () => response.promise })))

  await routeMove(board)
  component.$set({ session: nextSession() })
  expect(await screen.findByText('Puzzle 2 of 2')).toBeInTheDocument()

  response.resolve(correctResult())

  await new Promise((resolve) => setTimeout(resolve, 0))
  expect(screen.getByText('Puzzle 2 of 2')).toBeInTheDocument()
  expect(screen.queryByText('Correct!')).not.toBeInTheDocument()
})

test('keeps a completed puzzle visible until explicit Next without another mutation', async () => {
  const board = boardHarness()
  const playMove = vi.fn().mockResolvedValue(correctResult())
  const changes: SessionView[] = []
  const persisted: SessionView[] = []
  const { component } = render(PuzzleScreen, {
    session: resumedSession(),
    effects: effectsHarness().effects,
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ playMove })))
  component.$on('change', (event) => changes.push(event.detail))
  component.$on('persisted', (event) => persisted.push(event.detail))

  await routeMove(board)

  expect(await screen.findByText('Correct!')).toBeInTheDocument()
  expect(screen.getByText('Puzzle 1 of 2')).toBeInTheDocument()
  expect(screen.queryByText('Puzzle 2 of 2')).not.toBeInTheDocument()
  expect(persisted).toEqual([nextSession()])
  expect(changes).toEqual([])

  await fireEvent.click(screen.getByRole('button', { name: 'Next puzzle' }))

  expect(await screen.findByText('Puzzle 2 of 2')).toBeInTheDocument()
  expect(changes).toEqual([nextSession()])
  expect(playMove).toHaveBeenCalledTimes(1)
})

test('requires See results before showing the final summary', async () => {
  const board = boardHarness()
  const playMove = vi.fn().mockResolvedValue(correctResult(completedSession()))
  render(PuzzleScreen, {
    session: resumedSession(),
    effects: effectsHarness().effects,
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ playMove })))

  await routeMove(board)

  expect(await screen.findByRole('button', { name: 'See results' })).toBeInTheDocument()
  expect(screen.queryByText('Training complete!')).not.toBeInTheDocument()
  expect(screen.getByText('Puzzle 1 of 2')).toBeInTheDocument()

  await fireEvent.click(screen.getByRole('button', { name: 'See results' }))

  expect(await screen.findByText('Training complete!')).toBeInTheDocument()
})

test('unlocks sound from a Hint gesture, persists mute state, and keeps text feedback', async () => {
  const sequence: string[] = []
  const sound = soundHarness(false, sequence)
  const effects = effectsHarness({ sound })
  const hint: HintResult = {
    session: resumedSession({ hintLevel: 1 }),
    level: 1,
    text: 'Look for a forcing move.',
    canReveal: false
  }
  const useHint = vi.fn(async () => {
    sequence.push('hint')
    return hint
  })
  render(PuzzleScreen, {
    session: resumedSession(),
    effects: effects.effects,
    boardAdapterFactory: boardHarness().factory
  }, withNormalAPI(fakeAPI({ useHint })))
  const hintButton = await screen.findByRole('button', { name: 'Hint' })

  await fireEvent.pointerDown(hintButton)
  await fireEvent.click(hintButton)

  expect(sequence.slice(0, 2)).toEqual(['unlock', 'hint'])
  expect(await screen.findByText('Look for a forcing move.')).toBeInTheDocument()
  const mute = screen.getByRole('button', { name: 'Mute sounds' })
  expect(mute).toHaveAttribute('aria-pressed', 'false')

  await fireEvent.click(mute)

  expect(screen.getByRole('button', { name: 'Turn sound on' })).toHaveAttribute('aria-pressed', 'true')
  expect(screen.getByText('Look for a forcing move.')).toBeInTheDocument()
})

test('unlocks sound synchronously on board keyboard activation', async () => {
  const sequence: string[] = []
  const sound = soundHarness(false, sequence)
  render(PuzzleScreen, {
    session: resumedSession(),
    effects: effectsHarness({ sound }).effects,
    boardAdapterFactory: boardHarness().factory
  }, withNormalAPI(fakeAPI()))

  await fireEvent.keyDown(await screen.findByRole('grid'), { key: 'Enter' })

  expect(sequence[0]).toBe('unlock')
})

test('locks malformed legal moves with an actionable visible error', async () => {
  render(PuzzleScreen, {
    session: resumedSession({ legalMoves: ['not-a-uci'] }),
    effects: effectsHarness().effects,
    boardAdapterFactory: boardHarness().factory
  }, withNormalAPI(fakeAPI()))

  expect(await screen.findByText(
    /invalid legal move data.*not-a-uci/i,
    { selector: '.error' }
  )).toBeInTheDocument()
  expect(screen.getByRole('grid')).toHaveAttribute('aria-disabled', 'true')
})

test('adopts an unavailable hint with no current puzzle without a false solved state', async () => {
  const unavailable: HintResult = {
    session: completedSession(),
    level: 0,
    text: 'Puzzle unavailable',
    canReveal: false
  }
  render(PuzzleScreen, {
    session: resumedSession(),
    effects: effectsHarness().effects,
    boardAdapterFactory: boardHarness().factory
  }, withNormalAPI(fakeAPI({ useHint: async () => unavailable })))

  await fireEvent.click(await screen.findByRole('button', { name: 'Hint' }))

  expect(await screen.findByText('Puzzle unavailable')).toBeInTheDocument()
  expect(screen.queryByText('Correct!')).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Next puzzle' })).not.toBeInTheDocument()
})

test('pauses to home and aborts owned work on unmount', async () => {
  const gate = deferred<void>()
  const effects = effectsHarness({ reducedMotion: false, delay: () => gate.promise })
  const pauseSession = vi.fn().mockResolvedValue(undefined)
  const { component, unmount } = render(PuzzleScreen, {
    session: activeSession(),
    effects: effects.effects,
    boardAdapterFactory: boardHarness().factory
  }, withNormalAPI(fakeAPI({ pauseSession })))
  let wentHome = false
  component.$on('home', () => { wentHome = true })
  await waitFor(() => expect(effects.signals).toHaveLength(1))

  unmount()

  expect(effects.signals[0].aborted).toBe(true)
  expect(effects.sound.destroy).toHaveBeenCalledTimes(1)

  const paused = render(PuzzleScreen, {
    session: resumedSession(),
    effects: effectsHarness().effects,
    boardAdapterFactory: boardHarness().factory
  }, withNormalAPI(fakeAPI({ pauseSession })))
  paused.component.$on('home', () => { wentHome = true })
  await fireEvent.click(await screen.findByRole('button', { name: 'Pause' }))
  await waitFor(() => expect(wentHome).toBe(true))
  expect(pauseSession).toHaveBeenCalledWith('session-1')
})
