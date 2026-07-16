import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import InitialSetup from './InitialSetup.svelte'
import type { Profile } from '../../lib/api'
import { fakeAPI, withNormalAPI } from '../../test-fakes'

test('asks for a starting rating and session size', async () => {
  let saved: Profile | undefined
  const api = fakeAPI({
    getPracticeFilters: async () => ({
      sources: [],
      themes: [],
      maximumSolutionPlies: 0,
      learnerRatingBounds: { minimum: 3200, maximum: 3600 }
    }),
    updateProfile: async (profile) => { saved = profile }
  })
  render(InitialSetup, {}, withNormalAPI(api))

  const rating = await screen.findByLabelText('Starting rating')
  const size = screen.getByLabelText('Puzzles per session')
  expect(rating).toHaveAttribute('min', '3200')
  expect(rating).toHaveAttribute('max', '3600')
  expect(rating).toHaveValue(3200)
  expect(screen.getByText('Available rating range: 3200–3600')).toBeInTheDocument()

  await fireEvent.input(rating, { target: { value: '3100' } })
  await fireEvent.click(screen.getByRole('button', { name: 'Save and continue' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('Learner rating must be between 3200 and 3600')
  expect(saved).toBeUndefined()

  await fireEvent.input(rating, { target: { value: '3250' } })
  await fireEvent.change(size, { target: { value: '15' } })
  await fireEvent.click(screen.getByRole('button', { name: 'Save and continue' }))

  await waitFor(() => expect(saved).toEqual({ learnerRating: 3250, sessionSize: 15 }))
})
