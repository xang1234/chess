import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { vi } from 'vitest'
import { fakeAPI, withNormalAPI } from '../../test-fakes'
import ParentDashboard from './ParentDashboard.svelte'

test('shows progress and saves parent settings', async () => {
  const updateProfile = vi.fn().mockResolvedValue(undefined)
  const api = fakeAPI({
    getProfile: async () => ({ learnerRating: 3250, sessionSize: 10 }),
    getPracticeFilters: async () => ({
      sources: [],
      themes: [],
      maximumSolutionPlies: 0,
      learnerRatingBounds: { minimum: 3200, maximum: 3600 }
    }),
    updateProfile,
    getParentSummary: async () => ({
      learnerRating: 1250,
      ratingTrend: [
        { rating: 1100, recordedAt: 1 },
        { rating: 1200, recordedAt: 2 },
        { rating: 1250, recordedAt: 3 }
      ],
      firstAttemptAccuracy: 67.5,
      hintRate: 12.5,
      dueReviews: 4,
      themePerformance: [
        { theme: 'fork', attempts: 8, accuracy: 75 },
        { theme: 'pin', attempts: 4, accuracy: 50 }
      ],
      recentSessions: [{
        sessionId: 'session-1', mode: 'guided', status: 'completed', updatedAt: 3,
        total: 10, completed: 10, firstTry: 7, usedHint: 1, revealed: 0
      }]
    })
  })
  const { component } = render(ParentDashboard, {}, withNormalAPI(api))
  let openImport = false
  component.$on('import', () => { openImport = true })

  expect(await screen.findByText('67.5%')).toBeInTheDocument()
  expect(screen.getByText('12.5%')).toBeInTheDocument()
  expect(screen.getByText('4 due')).toBeInTheDocument()
  expect(screen.getByRole('img', { name: 'Learner rating trend' })).toBeInTheDocument()
  expect(screen.getByText('Minimum 1100')).toBeInTheDocument()
  expect(screen.getByText('Current 1250')).toBeInTheDocument()
  expect(screen.getByText('Maximum 1250')).toBeInTheDocument()
  expect(screen.getByRole('cell', { name: 'fork' })).toBeInTheDocument()
  expect(screen.getByRole('cell', { name: 'Guided' })).toBeInTheDocument()

  const rating = screen.getByLabelText('Current learner rating')
  expect(rating).toHaveAttribute('min', '3200')
  expect(rating).toHaveAttribute('max', '3600')
  expect(screen.getByText('Available rating range: 3200–3600')).toBeInTheDocument()

  await fireEvent.input(rating, { target: { value: '3100' } })
  await fireEvent.click(screen.getByRole('button', { name: 'Save settings' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('Learner rating must be between 3200 and 3600')
  expect(updateProfile).not.toHaveBeenCalled()

  await fireEvent.input(rating, { target: { value: '3300' } })
  await fireEvent.change(screen.getByLabelText('Puzzles per guided session'), { target: { value: '5' } })
  await fireEvent.click(screen.getByRole('button', { name: 'Save settings' }))
  await waitFor(() => expect(updateProfile).toHaveBeenCalledWith({ learnerRating: 3300, sessionSize: 5 }))
  expect(screen.getByText('Settings saved')).toBeInTheDocument()

  await fireEvent.click(screen.getByRole('button', { name: 'Import content' }))
  expect(openImport).toBe(true)
})
