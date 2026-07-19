import { render, screen, waitFor } from '@testing-library/svelte'
import HomeHub from './HomeHub.svelte'
import { fakeOpeningHome } from '../../test-fakes'

test('shows the child-friendly home destinations', async () => {
  const { component } = render(HomeHub, { activeSession: false })
  expect(screen.getByRole('button', { name: "Start today's training" })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Free Practice' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Game Library' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Learn Openings' })).toHaveTextContent('Import a private course')

  component.$set({ activeSession: true })
  await waitFor(() => {
    expect(screen.getByRole('button', { name: "Continue today's training" })).toBeInTheDocument()
  })
})

test('summarises resumable and due opening work without replacing puzzle training', async () => {
  const { component } = render(HomeHub, {
    activeSession: true,
    openingHome: {
      ...fakeOpeningHome,
      courses: [{ ...fakeOpeningHome.courses[0], hasResumable: true, dueReviews: 0 }]
    }
  })
  expect(screen.getByRole('button', { name: 'Learn Openings' }))
    .toHaveTextContent('Continue your Italian lesson')
  expect(screen.getByRole('button', { name: "Continue today's training" })).toBeInTheDocument()

  component.$set({
    openingHome: {
      ...fakeOpeningHome,
      courses: [{ ...fakeOpeningHome.courses[0], hasResumable: false, dueReviews: 3 }]
    }
  })
  await waitFor(() => {
    expect(screen.getByRole('button', { name: 'Learn Openings' }))
      .toHaveTextContent('3 opening reviews due')
  })
})
