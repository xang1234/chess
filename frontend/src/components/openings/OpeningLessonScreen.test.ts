import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import type { KeyPair } from '@lichess-org/chessground/types'
import type {
  ActiveOpeningSessionView,
  CompletedOpeningSessionView,
  OpeningSessionView,
  OpeningActivityResult,
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
  kind: ActiveOpeningSessionView['current']['kind'] = 'concept',
  overrides: Partial<ActiveOpeningSessionView['current']> = {}
): ActiveOpeningSessionView {
  return {
    ...fakeOpeningSession,
    current: {
      ...fakeOpeningSession.current,
      kind,
      variationName: 'Giuoco Piano',
      legalMoves: kind === 'decision' ? ['c2c3'] : [],
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
    depth: fakeOpeningSession.depth
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
  const explain = active('concept', {
    title: 'The central plan',
    instruction: 'Prepare d4 without blocking the bishop.',
    teachingNoteTexts: ['The quiet c3 move supports a later d4.'],
    referenceNoteTexts: ['A detailed reference remains available on demand.']
  })
  const watch = active('demonstration', { activityId: 'watch-c3', activityNumber: 2, title: 'Watch c3' })
  const advanceOpeningActivity = vi.fn(async (): Promise<OpeningActivityResult> => ({
    session: watch,
    activityCompleted: true,
    message: 'Plan understood.'
  }))
  const board = boardHarness()
  const { component } = render(OpeningLessonScreen, {
    session: explain,
    effects: effects(),
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ advanceOpeningActivity })))
  const changes: OpeningSessionView[] = []
  component.$on('change', (event) => changes.push(event.detail))

  expect(await screen.findByRole('heading', { name: 'The central plan' })).toBeInTheDocument()
  expect(screen.getByText('Opening course · Giuoco Piano')).toBeInTheDocument()
  expect(screen.getByText(/Idea 1 of 3/)).toBeInTheDocument()
  expect(screen.getByText('Prepare d4 without blocking the bishop.')).toBeInTheDocument()
  expect(screen.getByText('The quiet c3 move supports a later d4.')).toBeInTheDocument()
  expect(screen.getByText('Deeper analysis')).toBeInTheDocument()
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

test('keeps detailed reference notes collapsed until the learner opens them', async () => {
  const current = active('concept', {
    teachingNoteTexts: ['Keep this teaching note visible.'],
    referenceNoteTexts: [
      'Source note a records a detailed private reference.',
      'Source note b records another detailed private reference.'
    ]
  })
  render(OpeningLessonScreen, {
    session: current,
    effects: effects(),
    boardAdapterFactory: boardHarness().factory
  }, withNormalAPI(fakeAPI()))

  expect(await screen.findByText('Keep this teaching note visible.')).toBeVisible()
  const firstReference = screen.getByText('Source note a records a detailed private reference.')
  expect(firstReference).not.toBeVisible()

  await fireEvent.click(screen.getByText('Deeper analysis'))

  expect(firstReference).toBeVisible()
  expect(screen.getByText('Source note b records another detailed private reference.')).toBeVisible()
})

test('shows progressive hints and the reveal action only when allowed', async () => {
  const current = active('decision', { title: 'Find White’s setup', canReveal: false })
  const hinted = active('decision', { title: 'Find White’s setup', hintLevel: 3, canReveal: true })
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
  const current = active('decision', { title: 'Recognize the branch' })
  const result: OpeningActivityResult = {
    session: current,
    activityCompleted: false,
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

test('continues a completed lesson through the roadmap checkpoint without returning home', async () => {
  const current = active('decision', { title: 'Prepare the centre' })
  const done = completed()
  const result: OpeningActivityResult = {
    session: done,
    activityCompleted: true,
    feedback: 'expected',
    appliedMoves: [{
      uci: 'c2c3',
      resultingFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'
    }],
    finalFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4',
    checkpoint: {
      completedLessonId: 'giuoco-c3',
      path: [{ lessonId: 'giuoco-c3', title: 'Prepare d4 with c3' }],
      availableLessonIds: ['giuoco-d4'],
      recommendedLessonId: 'giuoco-d4',
      recommendedLessonTitle: 'Occupy the centre with d4',
      completedLessons: 1,
      totalLessons: 3
    }
  }
  const board = boardHarness()
  const { component } = render(OpeningLessonScreen, {
    session: current,
    path: [{ lessonId: 'giuoco-c3', title: 'Prepare d4 with c3' }],
    effects: effects(),
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({ playOpeningMove: async () => result })))
  const continuations: unknown[] = []
  component.$on('continue', (event) => continuations.push(event.detail))
  await waitFor(() => expect(board.callbacks).toHaveLength(1))

  board.callbacks[0].onRoute('c2', 'c3')
  await fireEvent.click(await screen.findByRole('button', { name: 'Continue' }))
  expect(await screen.findByRole('heading', { name: 'Prepare d4 with c3 complete' }))
    .toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', {
    name: 'Continue to Occupy the centre with d4'
  }))
  expect(continuations).toEqual([{
    courseId: fakeOpeningSession.courseId,
    lessonId: 'giuoco-d4'
  }])
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
  const checkpoint = active('concept', { title: 'Safe checkpoint' })
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
  expect(screen.getByText('Your place in the course has been saved.')).toBeInTheDocument()
})

test('keeps review results separate and returns to the course tree', async () => {
  const review: CompletedOpeningSessionView = {
    sessionId: 'review-session',
    mode: 'review',
    status: 'completed',
    courseId: fakeOpeningSession.courseId,
    generationId: fakeOpeningSession.generationId,
    lessonId: 'review',
    depth: fakeOpeningSession.depth,
    summary: {
      totalPrompts: 2,
      positionsRecalled: 2,
      branchesRecognized: 0,
      retried: 0,
      usedHint: 1,
      revealed: 0
    }
  }
  const { component } = render(OpeningLessonScreen, {
    session: review,
    effects: effects(),
    boardAdapterFactory: boardHarness().factory
  }, withNormalAPI(fakeAPI()))
  const tree = vi.fn()
  component.$on('tree', tree)

  expect(await screen.findByRole('heading', { name: 'Opening review complete!' }))
    .toBeInTheDocument()
  expect(screen.getByText('2 positions recalled')).toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', { name: 'Back to course' }))
  expect(tree).toHaveBeenCalledOnce()
})
