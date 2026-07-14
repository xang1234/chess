import { render, screen, waitFor } from '@testing-library/svelte'
import HomeHub from './HomeHub.svelte'

test('shows the child-friendly home destinations', async () => {
  const { component } = render(HomeHub, { activeSession: false })
  expect(screen.getByRole('button', { name: "Start today's training" })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Free Practice' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Game Library' })).toBeInTheDocument()

  component.$set({ activeSession: true })
  await waitFor(() => {
    expect(screen.getByRole('button', { name: "Continue today's training" })).toBeInTheDocument()
  })
})
