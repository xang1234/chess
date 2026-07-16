import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { vi } from 'vitest'
import App from './App.svelte'
import type { ImportProgress, ImportResult } from './lib/api'
import {
  fakeAPI,
  fakeRecoveryAPI,
  normalApplication,
  recoveryApplication
} from './test-fakes'

test('renders the product name and initial setup for a new learner', async () => {
  const api = fakeAPI({ getProfile: async () => null })
  render(App, { loadAPI: async () => normalApplication(api) })
  expect(screen.getByText('Chess Trainer')).toBeInTheDocument()
  await waitFor(() => expect(screen.getByText('Set up today’s training')).toBeInTheDocument())
})

test('shows Continue on the home hub when a session is active', async () => {
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 1200, sessionSize: 10 }),
    resumeSession: async () => ({
      sessionId: 'active-session', mode: 'guided', status: 'active', currentIndex: 0, total: 10
    })
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
        canReveal: false
      }
    })
  })
  render(App, { loadAPI: async () => normalApplication(api) })

  await fireEvent.click(await screen.findByRole('button', { name: "Start today's training" }))
  await waitFor(() => expect(screen.getByText('Find the best move')).toBeInTheDocument())
  expect(screen.getByRole('grid', { name: 'Chess board, white side' })).toBeInTheDocument()
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

test('keeps monitoring an active import while navigating away and reconciles it on return', async () => {
  let progressListener: (progress: ImportProgress) => void = () => {}
  let finishedListener: (result: ImportResult) => void = () => {}
  const getImportResult = vi.fn(async (jobId: string): Promise<ImportResult> => ({
    jobId,
    status: 'running',
    progress: { rowsRead: 10_000, bytesRead: 2048 },
    report: { accepted: 0, duplicates: 0, rejected: 0 }
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
  await fireEvent.click(screen.getByRole('button', { name: 'Choose puzzle database' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Import puzzles' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Chess Trainer home' }))
  progressListener({ jobId: 'job-1', rowsRead: 10_000, bytesRead: 2048 })

  await fireEvent.click(screen.getByRole('button', { name: 'Parent settings' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Import puzzles' }))

  await waitFor(() => expect(getImportResult).toHaveBeenCalledWith('job-1'))
  expect(screen.getByText('10,000 rows read')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Cancel import' })).toBeInTheDocument()

  finishedListener({
    jobId: 'job-1',
    status: 'succeeded',
    report: { accepted: 9800, duplicates: 150, rejected: 50 }
  })
  await waitFor(() => expect(screen.getByText('9,800 accepted')).toBeInTheDocument())
})
