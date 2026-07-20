import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { vi } from 'vitest'
import App from './App.svelte'
import type {
  ActiveSessionView,
  ActiveOpeningSessionView,
  CompletedOpeningSessionView,
  CompletedSessionView,
  ImportProgress,
  ImportResult,
  MoveResult,
  SessionView
} from './lib/api'
import {
  fakeAPI,
  fakeBuildInfo,
  fakeOpeningHome,
  fakeOpeningSession,
  fakeRecoveryAPI,
  normalApplication,
  recoveryApplication
} from './test-fakes'

afterEach(() => {
  vi.restoreAllMocks()
})

const puzzleFen = '4k3/8/8/4p3/8/8/4P3/4K3 w - - 0 2'
const solvedFen = '4k3/8/8/4p3/4P3/8/8/4K3 b - - 0 2'
const nextFen = '4k3/8/8/8/8/8/3P4/4K3 b - - 0 1'

function guidedPuzzle(total = 2): ActiveSessionView {
  return {
    sessionId: 'guided-session',
    mode: 'guided',
    status: 'active',
    currentIndex: 0,
    total,
    current: {
      fingerprint: 'puzzle-1',
      displayedFen: puzzleFen,
      currentFen: puzzleFen,
      solver: 'white',
      currentPath: [0],
      puzzleNumber: 1,
      puzzleTotal: total,
      hintLevel: 3,
      incorrectMoves: 0,
      canReveal: true,
      legalMoves: ['e2e4']
    }
  }
}

function nextGuidedPuzzle(): ActiveSessionView {
  return {
    sessionId: 'guided-session',
    mode: 'guided',
    status: 'active',
    currentIndex: 1,
    total: 2,
    current: {
      fingerprint: 'puzzle-2',
      displayedFen: nextFen,
      currentFen: nextFen,
      solver: 'black',
      currentPath: [],
      puzzleNumber: 2,
      puzzleTotal: 2,
      hintLevel: 0,
      incorrectMoves: 0,
      canReveal: false,
      legalMoves: ['e8d7']
    }
  }
}

function completedGuidedSession(): CompletedSessionView {
  return {
    sessionId: 'guided-session',
    mode: 'guided',
    status: 'completed',
    currentIndex: 1,
    total: 1,
    summary: {
      total: 1,
      firstTry: 1,
      retried: 0,
      usedHint: 0,
      revealed: 0,
      unavailable: 0
    }
  }
}

function revealResult(session: SessionView): MoveResult {
  return {
    session,
    correct: true,
    puzzleCompleted: true,
    appliedMoves: [{ uci: 'e2e4', resultingFen: solvedFen }],
    finalFen: solvedFen
  }
}

function guidedAPI(start: SessionView, pending: SessionView) {
  const revealSolution = vi.fn(async () => revealResult(pending))
  return {
    api: fakeAPI({
      getProfile: async () => ({ learnerRating: 1200, sessionSize: 5 }),
      startGuided: async () => start,
      revealSolution
    }),
    revealSolution
  }
}

async function openAndReveal(api: ReturnType<typeof fakeAPI>): Promise<void> {
  render(App, { loadAPI: async () => normalApplication(api) })
  await fireEvent.click(await screen.findByRole('button', { name: "Start today's training" }))
  await fireEvent.click(await screen.findByRole('button', { name: 'Show solution' }))
}

test('renders the product name and initial setup for a new learner', async () => {
  const api = fakeAPI({ getProfile: async () => null })
  render(App, { loadAPI: async () => normalApplication(api) })
  expect(screen.getByText('Chess Trainer')).toBeInTheDocument()
  await waitFor(() => expect(screen.getByText('Set up today’s training')).toBeInTheDocument())
})

test('shows Continue on the home hub when a session is active', async () => {
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 1200, sessionSize: 10 }),
    resumeSession: async () => guidedPuzzle(10)
  })
  render(App, { loadAPI: async () => normalApplication(api) })
  await waitFor(() => {
    expect(screen.getByRole('button', { name: "Continue today's training" })).toBeInTheDocument()
  })
})

test('keeps opening startup failures local to the opening card', async () => {
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 1200, sessionSize: 10 }),
    getOpeningHome: async () => { throw new Error('Reimport the private course pack.') },
    resumeOpeningSession: async () => { throw new Error('Opening course is unavailable.') }
  })
  render(App, { loadAPI: async () => normalApplication(api) })

  const openings = await screen.findByRole('button', { name: 'Learn Openings' })
  expect(openings).toHaveTextContent('Reimport the private course pack.')
  expect(screen.getByRole('button', { name: "Start today's training" })).toBeInTheDocument()
  expect(screen.queryByRole('alert')).not.toBeInTheDocument()
})

test('opens the course hub and starts a visible opening lesson', async () => {
  const setOpeningDepth = vi.fn(async () => {})
  const getOpeningHome = vi.fn(async () => fakeOpeningHome)
  const startOpeningLesson = vi.fn(async () => fakeOpeningSession)
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 1200, sessionSize: 10 }),
    getOpeningHome,
    setOpeningDepth,
    startOpeningLesson
  })
  render(App, { loadAPI: async () => normalApplication(api) })

  await fireEvent.click(await screen.findByRole('button', { name: 'Learn Openings' }))
  expect(await screen.findByRole('heading', { name: 'Italian Game for White' })).toBeInTheDocument()
  await fireEvent.change(screen.getByLabelText('Course depth'), { target: { value: 'quick' } })
  await waitFor(() => expect(setOpeningDepth).toHaveBeenCalledWith('synthetic-italian', 'quick'))
  expect(getOpeningHome).toHaveBeenCalledTimes(2)

  await fireEvent.click(screen.getByRole('button', { name: 'Study Giuoco Piano' }))
  expect(startOpeningLesson).toHaveBeenCalledWith('synthetic-italian', 'giuoco-c3')
  expect(await screen.findByRole('heading', { name: 'The central plan' })).toBeInTheDocument()
  expect(screen.getByRole('grid', { name: 'Chess board, white side' })).toBeInTheDocument()
})

test('keeps an explicit opening course selection across the app shell', async () => {
	const secondCourse = {
		...fakeOpeningHome.courses[0],
		courseId: 'queens-gambit-white',
		title: "Queen's Gambit for White"
	}
	const startOpeningLesson = vi.fn(async () => ({
		...fakeOpeningSession,
		courseId: secondCourse.courseId,
		courseTitle: secondCourse.title
	}))
	const api = fakeAPI({
		getProfile: async () => ({ learnerRating: 1200, sessionSize: 10 }),
		getOpeningHome: async () => ({ courses: [fakeOpeningHome.courses[0], secondCourse] }),
		startOpeningLesson
	})
	render(App, { loadAPI: async () => normalApplication(api) })

	await fireEvent.click(await screen.findByRole('button', { name: 'Learn Openings' }))
	await fireEvent.change(screen.getByLabelText('Opening course'), {
		target: { value: secondCourse.courseId }
	})
	expect(await screen.findByRole('heading', { name: secondCourse.title })).toBeInTheDocument()
	await fireEvent.click(screen.getByRole('button', { name: 'Study Giuoco Piano' }))
	expect(startOpeningLesson).toHaveBeenCalledWith(secondCourse.courseId, 'giuoco-c3')
})

test('continues from a lesson checkpoint directly into the recommended branch', async () => {
  const baseCourse = fakeOpeningHome.courses[0]
  const courseHome = {
    courses: [{
      ...baseCourse,
      completedLessons: 0,
      totalLessons: 2,
      tree: {
        rootLessonId: 'giuoco-c3',
        nodes: [
          { ...baseCourse.tree.nodes[0], title: 'Giuoco Piano', requiredActivities: 3 },
          {
            ...baseCourse.tree.nodes[0],
            lessonId: 'giuoco-d4',
            title: 'Occupy the centre with d4',
            objective: 'Time the central break.',
            completedActivities: 0,
            requiredActivities: 3,
            recommended: false
          }
        ],
        edges: [{
          edgeId: 'c3-d4',
          fromLessonId: 'giuoco-c3',
          toLessonId: 'giuoco-d4',
          ordinal: 1,
          kind: 'continuation' as const,
          minimumDepth: 'quick' as const
        }]
      }
    }]
  }
  const first = interactiveOpening()
  const second: ActiveOpeningSessionView = {
    ...fakeOpeningSession,
    lessonId: 'giuoco-d4',
    current: {
      ...fakeOpeningSession.current,
      activityId: 'd4-concept',
      title: 'Occupy the centre with d4'
    }
  }
  const startOpeningLesson = vi.fn(async (_courseId: string, lessonId: string) => (
    lessonId === 'giuoco-d4' ? second : first
  ))
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 1200, sessionSize: 10 }),
    getOpeningHome: async () => courseHome,
    startOpeningLesson,
    revealOpeningMove: async () => ({
      session: completedOpening(),
      activityCompleted: true,
      feedback: 'expected',
      message: 'Course move shown.',
      appliedMoves: [{
        uci: 'c2c3',
        resultingFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'
      }],
      finalFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4',
      checkpoint: {
        completedLessonId: 'giuoco-c3',
        path: [{ lessonId: 'giuoco-c3', title: 'Giuoco Piano' }],
        availableLessonIds: ['giuoco-d4'],
        recommendedLessonId: 'giuoco-d4',
        recommendedLessonTitle: 'Occupy the centre with d4',
        completedLessons: 1,
        totalLessons: 2
      }
    })
  })
  render(App, { loadAPI: async () => normalApplication(api) })

  await fireEvent.click(await screen.findByRole('button', { name: 'Learn Openings' }))
  await fireEvent.click(await screen.findByRole('button', { name: 'Study Giuoco Piano' }))
  await fireEvent.click(await screen.findByRole('button', { name: 'Show course move' }))
  await fireEvent.click(await screen.findByRole('button', { name: 'Continue' }))
  await fireEvent.click(await screen.findByRole('button', {
    name: 'Continue to Occupy the centre with d4'
  }))

  expect(startOpeningLesson).toHaveBeenLastCalledWith('synthetic-italian', 'giuoco-d4')
  expect(await screen.findByRole('heading', { name: 'Occupy the centre with d4' }))
    .toBeInTheDocument()
  expect(screen.getByRole('navigation', { name: 'Opening course path' }))
    .toHaveTextContent('Giuoco Piano')
})

test('refreshes opening progress when Stop for now leaves a checkpoint', async () => {
  const getOpeningHome = vi.fn(async () => fakeOpeningHome)
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 1200, sessionSize: 10 }),
    getOpeningHome,
    startOpeningLesson: async () => interactiveOpening(),
    revealOpeningMove: async () => ({
      session: completedOpening(),
      activityCompleted: true,
      feedback: 'expected',
      appliedMoves: [{
        uci: 'c2c3',
        resultingFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'
      }],
      finalFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4',
      checkpoint: {
        completedLessonId: 'giuoco-c3',
        path: [{ lessonId: 'giuoco-c3', title: 'Giuoco Piano' }],
        availableLessonIds: [],
        completedLessons: 1,
        totalLessons: 1
      }
    })
  })
  render(App, { loadAPI: async () => normalApplication(api) })

  await fireEvent.click(await screen.findByRole('button', { name: 'Learn Openings' }))
  await fireEvent.click(await screen.findByRole('button', { name: 'Study Giuoco Piano' }))
  await fireEvent.click(await screen.findByRole('button', { name: 'Show course move' }))
  await fireEvent.click(await screen.findByRole('button', { name: 'Continue' }))
  await fireEvent.click(await screen.findByRole('button', { name: 'Stop for now' }))

  await waitFor(() => expect(getOpeningHome).toHaveBeenCalledTimes(3))
  expect(screen.getByRole('button', { name: 'Learn Openings' })).toBeInTheDocument()
})

test('opens the read-only variation explorer at the course depth', async () => {
  const getOpeningPosition = vi.fn(async (courseId: string, positionId: string) => ({
    courseId,
    positionId,
    fen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
    label: 'Initial position',
    evaluation: { code: 'none' as const },
    notes: [],
    moves: [],
    incomingPaths: 0
  }))
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 1200, sessionSize: 10 }),
    getOpeningHome: async () => fakeOpeningHome,
    getOpeningPosition
  })
  render(App, { loadAPI: async () => normalApplication(api) })

  await fireEvent.click(await screen.findByRole('button', { name: 'Learn Openings' }))
  await fireEvent.click(await screen.findByRole('button', { name: 'Explore variations' }))

  expect(await screen.findByRole('heading', { name: 'Variation explorer' })).toBeInTheDocument()
  expect(screen.getByRole('heading', { name: 'Initial position' })).toBeInTheDocument()
  expect(getOpeningPosition).toHaveBeenCalledWith(
    'synthetic-italian', 'initial', 'reference'
  )
})

function interactiveOpening(): ActiveOpeningSessionView {
  return {
    ...fakeOpeningSession,
    current: {
      ...fakeOpeningSession.current,
      kind: 'decision',
      title: 'Recall the quiet setup',
      legalMoves: ['c2c3'],
      canReveal: true,
      hintLevel: 3
    }
  }
}

function nextOpening(): ActiveOpeningSessionView {
  return {
    ...fakeOpeningSession,
    current: {
      ...fakeOpeningSession.current,
      activityId: 'watch-reply',
      kind: 'demonstration',
      title: 'Watch Black’s reply',
      activityNumber: 2,
      legalMoves: []
    }
  }
}

function completedOpening(): CompletedOpeningSessionView {
  return {
    sessionId: fakeOpeningSession.sessionId,
    mode: 'lesson',
    status: 'completed',
    courseId: fakeOpeningSession.courseId,
    generationId: fakeOpeningSession.generationId,
    lessonId: fakeOpeningSession.lessonId,
    courseTitle: fakeOpeningSession.courseTitle,
    path: fakeOpeningSession.path,
    depth: fakeOpeningSession.depth
  }
}

async function openAndRevealOpening(
  pending: ActiveOpeningSessionView | CompletedOpeningSessionView,
  puzzle: SessionView | null = null
): Promise<void> {
	let activityPersisted = false
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 1200, sessionSize: 10 }),
    resumeSession: async () => puzzle,
    getOpeningHome: async () => ({
		...fakeOpeningHome,
		courses: fakeOpeningHome.courses.map((course) => ({
			...course,
			hasResumable: activityPersisted && pending.status !== 'completed',
			currentLessonId: activityPersisted && pending.status !== 'completed' ? pending.lessonId : undefined,
			currentPath: activityPersisted && pending.status !== 'completed' ? pending.path : []
		}))
	}),
    startOpeningLesson: async () => interactiveOpening(),
		revealOpeningMove: async () => {
			activityPersisted = true
			return {
				session: pending,
				activityCompleted: true as const,
				feedback: 'expected' as const,
				message: 'Course move shown.',
				appliedMoves: [{
					uci: 'c2c3',
					resultingFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'
				}],
				finalFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'
			}
		}
  })
  render(App, { loadAPI: async () => normalApplication(api) })
  await fireEvent.click(await screen.findByRole('button', { name: 'Learn Openings' }))
  await fireEvent.click(await screen.findByRole('button', { name: 'Study Giuoco Piano' }))
  await fireEvent.click(await screen.findByRole('button', { name: 'Show course move' }))
}

test('keeps the persisted next opening step resumable when Home leaves its solved board', async () => {
  await openAndRevealOpening(nextOpening())
  expect(await screen.findByText('Course move shown.')).toBeInTheDocument()

  await fireEvent.click(screen.getByRole('button', { name: 'Chess Trainer home' }))

  expect(await screen.findByRole('button', { name: 'Learn Openings' }))
    .toHaveTextContent('Continue Italian Game for White')
})

test('finishing an opening lesson does not clear an active puzzle session', async () => {
  await openAndRevealOpening(completedOpening(), guidedPuzzle())
  expect(await screen.findByRole('button', { name: 'Continue' })).toBeInTheDocument()

  await fireEvent.click(screen.getByRole('button', { name: 'Chess Trainer home' }))

  expect(await screen.findByRole('button', { name: "Continue today's training" })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Learn Openings' }))
    .not.toHaveTextContent('Continue Italian Game for White')
})

test('uses catalogued learner bounds on the parent settings screen', async () => {
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 3300, sessionSize: 10 }),
    getPracticeFilters: async () => ({
      sources: [],
      themes: [],
      maximumSolutionPlies: 0,
      learnerRatingBounds: { minimum: 3200, maximum: 3600 }
    })
  })
  render(App, { loadAPI: async () => normalApplication(api) })

  await fireEvent.click(await screen.findByRole('button', { name: 'Parent settings' }))
  const rating = await screen.findByLabelText('Current learner rating')
  expect(rating).toHaveAttribute('min', '3200')
  expect(rating).toHaveAttribute('max', '3600')
})

test('opens the board-first puzzle screen from the home hub', async () => {
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 1200, sessionSize: 5 }),
    startGuided: async () => ({
      sessionId: 'session-1', mode: 'guided', status: 'active', currentIndex: 0, total: 1,
      current: {
        fingerprint: 'puzzle-1',
        displayedFen: '4k3/8/8/4p3/8/8/4P3/4K3 w - - 0 2',
        currentFen: '4k3/8/8/4p3/8/8/4P3/4K3 w - - 0 2',
        solver: 'white',
        currentPath: [],
        puzzleNumber: 1,
        puzzleTotal: 1,
        hintLevel: 0,
        incorrectMoves: 0,
        canReveal: false,
        legalMoves: ['e1d1', 'e1d2', 'e1f1', 'e1f2', 'e2e3', 'e2e4']
      }
    })
  })
  render(App, { loadAPI: async () => normalApplication(api) })

  await fireEvent.click(await screen.findByRole('button', { name: "Start today's training" }))
  await waitFor(() => expect(screen.getByText('Find the best move')).toBeInTheDocument())
  expect(screen.getByRole('grid', { name: 'Chess board, white side' })).toBeInTheDocument()
})

test('opens matching source from About & Legal and returns to the normal home hub', async () => {
  const open = vi.spyOn(window, 'open').mockImplementation(() => null)
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 1200, sessionSize: 5 })
  })
  render(App, { loadAPI: async () => normalApplication(api, fakeBuildInfo) })

  await fireEvent.click(await screen.findByRole('button', { name: 'About & Legal' }))
  expect(await screen.findByRole('heading', { name: 'About & Legal' })).toBeInTheDocument()
  expect(screen.getByText(fakeBuildInfo.sourceUrl)).toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', { name: 'Open matching source' }))
  expect(open).toHaveBeenCalledWith(
    fakeBuildInfo.sourceUrl,
    '_blank',
    'noopener,noreferrer'
  )

  await fireEvent.click(screen.getByRole('button', { name: 'Back' }))
  expect(await screen.findByRole('heading', { name: 'What would you like to play?' })).toBeInTheDocument()
})

test('keeps the persisted next puzzle available when Home leaves a solved board', async () => {
  const { api } = guidedAPI(guidedPuzzle(), nextGuidedPuzzle())
  await openAndReveal(api)
  expect(await screen.findByRole('button', { name: 'Next puzzle' })).toBeInTheDocument()

  await fireEvent.click(screen.getByRole('button', { name: 'Chess Trainer home' }))

  expect(await screen.findByRole('button', { name: "Continue today's training" })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: "Start today's training" })).not.toBeInTheDocument()
})

test('does not offer a dead completed session when Home leaves the final solved board', async () => {
  const { api } = guidedAPI(guidedPuzzle(1), completedGuidedSession())
  await openAndReveal(api)
  expect(await screen.findByRole('button', { name: 'See results' })).toBeInTheDocument()

  await fireEvent.click(screen.getByRole('button', { name: 'Chess Trainer home' }))

  expect(await screen.findByRole('button', { name: "Start today's training" })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: "Continue today's training" })).not.toBeInTheDocument()
})

test('shows the next puzzle only after explicit acknowledgement', async () => {
  const { api, revealSolution } = guidedAPI(guidedPuzzle(), nextGuidedPuzzle())
  await openAndReveal(api)

  expect(await screen.findByRole('button', { name: 'Next puzzle' })).toBeInTheDocument()
  expect(screen.getByText('Puzzle 1 of 2')).toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', { name: 'Next puzzle' }))

  expect(await screen.findByText('Puzzle 2 of 2')).toBeInTheDocument()
  expect(revealSolution).toHaveBeenCalledTimes(1)
})

test('shows the summary only after explicit results acknowledgement', async () => {
  const { api } = guidedAPI(guidedPuzzle(1), completedGuidedSession())
  await openAndReveal(api)

  expect(await screen.findByRole('button', { name: 'See results' })).toBeInTheDocument()
  expect(screen.queryByText('Training complete!')).not.toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', { name: 'See results' }))

  expect(await screen.findByText('Training complete!')).toBeInTheDocument()
})

test('opens only the recovery surface when startup integrity fails', async () => {
  const api = fakeRecoveryAPI({
    getRecoveryState: async () => ({
      required: true,
      path: '/data/user.sqlite',
      detail: 'database disk image is malformed'
    })
  })
  render(App, { loadAPI: async () => recoveryApplication(api) })

  await waitFor(() => expect(screen.getByText('Your chess data needs recovery')).toBeInTheDocument())
  expect(screen.queryByRole('button', { name: 'Chess Trainer home' })).not.toBeInTheDocument()
})

test('opens matching source from recovery About & Legal and returns to recovery', async () => {
  const open = vi.spyOn(window, 'open').mockImplementation(() => null)
  const api = fakeRecoveryAPI({
    getRecoveryState: async () => ({
      required: true,
      path: '/data/user.sqlite',
      detail: 'database disk image is malformed'
    })
  })
  render(App, { loadAPI: async () => recoveryApplication(api, fakeBuildInfo) })

  await fireEvent.click(await screen.findByRole('button', { name: 'About & Legal' }))
  expect(await screen.findByRole('heading', { name: 'About & Legal' })).toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', { name: 'Open matching source' }))
  expect(open).toHaveBeenCalledWith(
    fakeBuildInfo.sourceUrl,
    '_blank',
    'noopener,noreferrer'
  )

  await fireEvent.click(screen.getByRole('button', { name: 'Back' }))
  expect(await screen.findByText('Your chess data needs recovery')).toBeInTheDocument()
})

test('keeps About & Legal available when recovery-state loading fails', async () => {
  const api = fakeRecoveryAPI({
    getRecoveryState: async () => { throw new Error('database startup failed') }
  })
  render(App, { loadAPI: async () => recoveryApplication(api, fakeBuildInfo) })
  expect(await screen.findByRole('alert')).toHaveTextContent('database startup failed')

  await fireEvent.click(screen.getByRole('button', { name: 'About & Legal' }))
  expect(await screen.findByText('Copyright © 2026 David Ten and Chess Trainer contributors'))
    .toBeInTheDocument()

  await fireEvent.click(screen.getByRole('button', { name: 'Back' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('database startup failed')
})

test('keeps monitoring an active import while navigating away and reconciles it on return', async () => {
  let progressListener: (progress: ImportProgress) => void = () => {}
  let finishedListener: (result: ImportResult) => void = () => {}
  const getImportResult = vi.fn(async (jobId: string): Promise<ImportResult> => ({
    jobId,
    status: 'running',
    progress: { phase: 'parsing', rowsRead: 10_000, bytesRead: 2048, totalBytes: 4096 },
    report: { accepted: 0, duplicates: 0, rejected: 0, examples: [], counts: {} }
  }))
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 1200, sessionSize: 5 }),
    choosePuzzleImportFile: async () => '/tmp/lichess.csv.zst',
    getImportResult,
    onImportProgress: (listener) => {
      progressListener = listener
      return () => {}
    },
    onImportFinished: (listener) => {
      finishedListener = listener
      return () => {}
    }
  })
  render(App, { loadAPI: async () => normalApplication(api) })

  await fireEvent.click(await screen.findByRole('button', { name: 'Parent settings' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Import content' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Choose puzzle collection' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Import puzzles' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Chess Trainer home' }))
  progressListener({
    jobId: 'job-1', phase: 'parsing', rowsRead: 10_000, bytesRead: 2048, totalBytes: 4096
  })

  await fireEvent.click(screen.getByRole('button', { name: 'Parent settings' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Import content' }))

  await waitFor(() => expect(getImportResult).toHaveBeenCalledWith('job-1'))
  expect(screen.getByText('10,000 rows read')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Cancel import' })).toBeInTheDocument()

  finishedListener({
    jobId: 'job-1',
    status: 'succeeded',
    progress: { phase: 'activating', rowsRead: 10_000, bytesRead: 4096, totalBytes: 4096 },
    report: { accepted: 9800, duplicates: 150, rejected: 50, examples: [], counts: {} }
  })
  await waitFor(() => expect(screen.getByText('9,800 accepted')).toBeInTheDocument())
})
