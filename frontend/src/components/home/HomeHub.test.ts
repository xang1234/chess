import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
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

test('offers exact opening continuation separately from the course map', async () => {
  const { component } = render(HomeHub, {
    activeSession: true,
    openingHome: {
      ...fakeOpeningHome,
      courses: [{ ...fakeOpeningHome.courses[0], hasResumable: true, dueReviews: 0 }]
    }
  })
  const continuations: unknown[] = []
  const maps: unknown[] = []
  component.$on('continueOpening', (event) => continuations.push(event.detail))
  component.$on('openings', (event) => maps.push(event.detail))

  await fireEvent.click(screen.getByRole('button', { name: 'Continue opening lesson' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Course map' }))
  expect(continuations).toHaveLength(1)
  expect(maps).toHaveLength(1)
  expect(screen.getByRole('button', { name: "Continue today's training" })).toBeInTheDocument()

  component.$set({
    openingHome: {
      ...fakeOpeningHome,
      courses: [{ ...fakeOpeningHome.courses[0], hasResumable: false, dueReviews: 3 }]
    }
  })
  await waitFor(() => {
    expect(screen.getByRole('button', { name: 'Learn Openings' }))
      .toHaveTextContent('Next: Giuoco Piano')
  })
})

test('summarises multiple opening courses without silently choosing the first', () => {
	render(HomeHub, {
		openingHome: {
			courses: [
				fakeOpeningHome.courses[0],
				{ ...fakeOpeningHome.courses[0], courseId: 'queens-gambit', title: "Queen's Gambit" }
			]
		}
	})
	expect(screen.getByRole('button', { name: 'Learn Openings' }))
		.toHaveTextContent('Choose from 2 opening courses')
})
