import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { vi } from 'vitest'
import type { SessionView } from '../../lib/api'
import { fakeAPI, withNormalAPI } from '../../test-fakes'
import FreePractice from './FreePractice.svelte'

test('starts practice with the selected source, rating, themes, and length filters', async () => {
  const started: SessionView = {
    sessionId: 'practice-1', mode: 'practice', status: 'active', currentIndex: 0, total: 5
  }
  const startFreePractice = vi.fn().mockResolvedValue(started)
  const api = fakeAPI({
    getPracticeFilters: async () => ({
      sources: [{
        id: 'lichess', kind: 'lichess', minimumRating: 800, maximumRating: 2200,
        hasRatingRange: true, maximumPlies: 7
      }],
      themes: ['fork', 'pin'],
      maximumSolutionPlies: 7,
      learnerRatingBounds: { minimum: 800, maximum: 2200 }
    }),
    startFreePractice
  })
  const { component } = render(FreePractice, {}, withNormalAPI(api))
  let emitted: SessionView | null = null
  component.$on('start', (event) => { emitted = event.detail })

  await screen.findByLabelText('Puzzle source')
  await fireEvent.click(screen.getByLabelText('Limit by rating'))
  await fireEvent.input(screen.getByLabelText('Minimum rating'), { target: { value: '1200' } })
  await fireEvent.input(screen.getByLabelText('Maximum rating'), { target: { value: '1600' } })
  await fireEvent.click(screen.getByLabelText('fork'))
  await fireEvent.change(screen.getByLabelText('Maximum solution length'), { target: { value: '3' } })
  await fireEvent.click(screen.getByRole('button', { name: 'Start practice' }))

  await waitFor(() => expect(startFreePractice).toHaveBeenCalledWith({
    sourceId: 'lichess',
    minimumRating: 1200,
    maximumRating: 1600,
    themes: ['fork'],
    maximumSolutionPlies: 3
  }))
  expect(emitted).toEqual(started)
})
