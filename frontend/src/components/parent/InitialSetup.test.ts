import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import InitialSetup from './InitialSetup.svelte'
import { setAPIForTests } from '../../lib/api'
import { fakeAPI } from '../../test-fakes'

test('asks for a starting rating and session size', async () => {
  let saved: unknown
  setAPIForTests(fakeAPI({
    updateProfile: async (profile) => { saved = profile }
  }))
  render(InitialSetup)

  const rating = screen.getByLabelText('Starting rating')
  const size = screen.getByLabelText('Puzzles per session')
  await fireEvent.input(rating, { target: { value: '1250' } })
  await fireEvent.change(size, { target: { value: '15' } })
  await fireEvent.click(screen.getByRole('button', { name: 'Save and continue' }))

  await waitFor(() => expect(saved).toEqual({ learnerRating: 1250, sessionSize: 15 }))
})
