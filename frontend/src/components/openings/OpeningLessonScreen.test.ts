import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import type { KeyPair } from '@lichess-org/chessground/types'
import type {
  ActiveOpeningSessionView,
  CompletedOpeningSessionView,
  OpeningSessionView,
  OpeningStepResult,
  RestartRequiredOpeningSessionView
} from '../../lib/api'
import type { SoundService } from '../../lib/sound'
import { fakeAPI, fakeOpeningSession, withNormalAPI } from '../../test-fakes'
import type { BoardEffects } from '../chess/board-effects'
import type {
  BoardCallbacks,
  BoardInteraction,
  ChessBoardAdapter,
  ChessgroundAdapterFactory
} from '../chess/chessground-adapter'
import OpeningLessonScreen from './OpeningLessonScreen.svelte'

function active(
  kind: ActiveOpeningSessionView['current']['kind'] = 'explain',
  overrides: Partial<ActiveOpeningSessionView['current']> = {}
): ActiveOpeningSessionView {
  return {
    ...fakeOpeningSession,
    current: {
      ...fakeOpeningSession.current,
      kind,
      variationName: 'Giuoco Piano',
      legalMoves: kind === 'try' || kind === 'branch' || kind === 'recall' ? ['c2c3'] : [],
      ...overrides
    }
  }
}

function completed(): CompletedOpeningSessionView {
  return {
    sessionId: fakeOpeningSession.sessionId,
    mode: 'lesson',
    status: 'completed',
    courseId: fakeOpeningSession.courseId,
    generationId: fakeOpeningSession.generationId,
    lessonId: fakeOpeningSession.lessonId,
    depth: fakeOpeningSession.depth,
    summary: {
      totalPrompts: 4,
      positionsRecalled: 2,
      branchesRecognized: 1,
      retried: 1,
      usedHint: 2,
      revealed: 1
    }
  }
}

function effects(): BoardEffects {
  const service: SoundService = {
    muted: false,
    unlock: async () => {},
    play: vi.fn(),
    setMuted: vi.fn(),
    toggleMuted: () => false,
    destroy: vi.fn()
  }
  return {
    createSound: () => service,
    delay: async () => {},
    prefersReducedMotion: () => true
  }
}

function boardHarness() {
  const callbacks: BoardCallbacks[] = []
  const interactions: BoardInteraction[] = []
  const positions: Array<[string, KeyPair | undefined, boolean | undefined]> = []
  const factory: ChessgroundAdapterFactory = vi.fn((_element, _fen, interaction, boardCallbacks) => {
    callbacks.push(boardCallbacks)
    interactions.push(interaction)
    const adapter: ChessBoardAdapter = {
      configure: vi.fn((next) => interactions.push(next)),
      setPosition: vi.fn((fen, lastMove, animate) => positions.push([fen, lastMove, animate])),
      selectSquare: vi.fn(),
      destroy: vi.fn()
    }
    return adapter
  })
  return { callbacks, factory, interactions, positions }
}

test('renders a board-first teaching step and defers the next step until Continue', async () => {
  const explain = active('explain', {
    title: 'The central plan',
    instruction: 'Prepare d4 without blocking the bishop.',
    noteTexts: ['The quiet c3 move supports a later d4.']
  })
  const watch = active('watch', { stepId: 'watch-c3', stepNumber: 2, title: 'Watch c3' })
  const advanceOpeningStep = vi.fn(async (): Promise<OpeningStepResult> => ({
    session: watch,
    stepCompleted: true,
    message: 'Plan understood.'
  }))
  const board = boardHarness()
  const { component } = render(OpeningLessonScreen, {
    session: explain,
    effects: effects(),
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ advanceOpeningStep })))
  const changes: OpeningSessionView[] = []
  component.$on('change', (event) => changes.push(event.detail))

  expect(await screen.findByRole('heading', { name: 'The central plan' })).toBeInTheDocument()
  expect(screen.getByText('Opening course · Giuoco Piano')).toBeInTheDocument()
  expect(screen.getByText('Step 1 of 5')).toBeInTheDocument()
  expect(screen.getByText('Prepare d4 without blocking the bishop.')).toBeInTheDocument()
  expect(screen.getByText('The quiet c3 move supports a later d4.')).toBeInTheDocument()
  expect(screen.getByText('Reference notes')).toBeInTheDocument()
  expect(screen.getByRole('grid', { name: 'Chess board, white side' }))
    .toHaveAttribute('aria-disabled', 'true')

  await fireEvent.click(screen.getByRole('button', { name: 'Continue' }))

  expect(await screen.findByText('Plan understood.')).toBeInTheDocument()
  expect(screen.getByRole('heading', { name: 'The central plan' })).toBeInTheDocument()
  expect(changes).toEqual([])

  await fireEvent.click(screen.getByRole('button', { name: 'Continue' }))

  expect(await screen.findByRole('heading', { name: 'Watch c3' })).toBeInTheDocument()
  expect(changes).toEqual([watch])
})

test('shows progressive hints and the reveal action only when allowed', async () => {
  const current = active('try', { title: 'Find White’s setup', canReveal: false })
  const hinted = active('try', { title: 'Find White’s setup', hintLevel: 3, canReveal: true })
  const board = boardHarness()
  render(OpeningLessonScreen, {
    session: current,
    effects: effects(),
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({
    useOpeningHint: async () => ({
      session: hinted,
      level: 3,
      text: 'Move the c-pawn one square.',
      sourceSquare: 'c2',
      targetSquare: 'c3',
      canReveal: true
    })
  })))

  expect(screen.queryByRole('button', { name: 'Show course move' })).not.toBeInTheDocument()
  await fireEvent.click(await screen.findByRole('button', { name: 'Hint' }))

  expect(await screen.findByText('Move the c-pawn one square.')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Show course move' })).toBeInTheDocument()
  await waitFor(() => expect(board.interactions.at(-1)).toMatchObject({
    hintSource: 'c2',
    hintTarget: 'c3'
  }))
})

test('restores alternative moves with neutral course feedback', async () => {
  const current = active('branch', { title: 'Recognize the branch' })
  const result: OpeningStepResult = {
    session: current,
    stepCompleted: false,
    feedback: 'alternative'
  }
  const board = boardHarness()
  render(OpeningLessonScreen, {
    session: current,
    effects: effects(),
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ playOpeningMove: async () => result })))
  await waitFor(() => expect(board.callbacks).toHaveLength(1))

  board.callbacks[0].onRoute('c2', 'c3')

  expect(await screen.findByText(/Playable alternative/)).toBeInTheDocument()
  expect(screen.getByRole('grid')).toHaveAttribute('aria-disabled', 'false')
})

test('renders update restart and completion summaries', async () => {
  const restartRequired: RestartRequiredOpeningSessionView = {
    sessionId: fakeOpeningSession.sessionId,
    mode: 'lesson',
    status: 'restart_required',
    courseId: fakeOpeningSession.courseId,
    generationId: fakeOpeningSession.generationId,
    lessonId: fakeOpeningSession.lessonId,
    depth: fakeOpeningSession.depth,
    notice: 'The course changed since this lesson began.'
  }
  const checkpoint = active('explain', { title: 'Safe checkpoint' })
  const restartOpeningSession = vi.fn(async () => checkpoint)
  const restarted = render(OpeningLessonScreen, {
    session: restartRequired,
    effects: effects(),
    boardAdapterFactory: boardHarness().factory
  }, withNormalAPI(fakeAPI({ restartOpeningSession })))

  expect(await screen.findByText('The course changed since this lesson began.')).toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', { name: 'Restart from checkpoint' }))
  expect(await screen.findByRole('heading', { name: 'Safe checkpoint' })).toBeInTheDocument()
  restarted.unmount()

  render(OpeningLessonScreen, {
    session: completed(),
    effects: effects(),
    boardAdapterFactory: boardHarness().factory
  }, withNormalAPI(fakeAPI()))
  expect(await screen.findByRole('heading', { name: 'Opening lesson complete!' })).toBeInTheDocument()
  expect(screen.getByText('2 positions recalled')).toBeInTheDocument()
  expect(screen.getByText('1 branch recognized')).toBeInTheDocument()
  expect(screen.getByText('1 retry')).toBeInTheDocument()
  expect(screen.getByText('2 hints used')).toBeInTheDocument()
  expect(screen.getByText('1 course move shown')).toBeInTheDocument()
})
