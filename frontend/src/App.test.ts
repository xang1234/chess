import { render, screen, waitFor } from '@testing-library/svelte'
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
