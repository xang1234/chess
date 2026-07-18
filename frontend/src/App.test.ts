import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { vi } from 'vitest'
import App from './App.svelte'
import type {
  ActiveSessionView,
  CompletedSessionView,
  ImportProgress,
  ImportResult,
  MoveResult,
  SessionView
} from './lib/api'
import {
  fakeAPI,
  fakeBuildInfo,
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
    report: { accepted: 0, duplicates: 0, rejected: 0, examples: [] }
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
  await fireEvent.click(screen.getByRole('button', { name: 'Import puzzles' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Choose puzzle collection' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Import puzzles' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Chess Trainer home' }))
  progressListener({
    jobId: 'job-1', phase: 'parsing', rowsRead: 10_000, bytesRead: 2048, totalBytes: 4096
  })

  await fireEvent.click(screen.getByRole('button', { name: 'Parent settings' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Import puzzles' }))

  await waitFor(() => expect(getImportResult).toHaveBeenCalledWith('job-1'))
  expect(screen.getByText('10,000 rows read')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Cancel import' })).toBeInTheDocument()

  finishedListener({
    jobId: 'job-1',
    status: 'succeeded',
    report: { accepted: 9800, duplicates: 150, rejected: 50, examples: [] }
  })
  await waitFor(() => expect(screen.getByText('9,800 accepted')).toBeInTheDocument())
})
