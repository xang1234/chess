import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import App from './App.svelte'
import { setAPIForTests } from './lib/api'
import { fakeAPI } from './test-fakes'

test('renders the product name and initial setup for a new learner', async () => {
  setAPIForTests(fakeAPI({ getProfile: async () => null }))
  render(App)
  expect(screen.getByText('Chess Trainer')).toBeInTheDocument()
  await waitFor(() => expect(screen.getByText('Set up today’s training')).toBeInTheDocument())
})

test('shows Continue on the home hub when a session is active', async () => {
  setAPIForTests(fakeAPI({
    getProfile: async () => ({ learnerRating: 1200, sessionSize: 10 }),
    resumeSession: async () => ({
      sessionId: 'active-session', mode: 'guided', status: 'active', currentIndex: 0, total: 10
    })
  }))
  render(App)
  await waitFor(() => {
    expect(screen.getByRole('button', { name: "Continue today's training" })).toBeInTheDocument()
  })
})

test('opens the board-first puzzle screen from the home hub', async () => {
  setAPIForTests(fakeAPI({
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
  }))
  render(App)

  await fireEvent.click(await screen.findByRole('button', { name: "Start today's training" }))
  await waitFor(() => expect(screen.getByText('Find the best move')).toBeInTheDocument())
  expect(screen.getByRole('grid', { name: 'Chess board, white side' })).toBeInTheDocument()
})

test('opens only the recovery surface when startup integrity fails', async () => {
  setAPIForTests(fakeAPI({
    getRecoveryState: async () => ({
      required: true,
      path: '/data/user.sqlite',
      detail: 'database disk image is malformed'
    })
  }))
  render(App)

  await waitFor(() => expect(screen.getByText('Your chess data needs recovery')).toBeInTheDocument())
  expect(screen.queryByRole('button', { name: 'Chess Trainer home' })).not.toBeInTheDocument()
})
