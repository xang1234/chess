import { render, screen } from '@testing-library/svelte'
import OpeningMoveRibbon from './OpeningMoveRibbon.svelte'

test('renders variation, numbered SAN, and current ply', () => {
  render(OpeningMoveRibbon, {
    variationName: 'Italian Game',
    moves: ['e4', 'e5', 'Nf3', 'Nc6', 'Bc4']
  })

  expect(screen.getByText('Italian Game')).toBeVisible()
  expect(screen.getByRole('list', { name: 'Moves to this position' }))
    .toHaveTextContent('1. e4 e5 2. Nf3 Nc6 3. Bc4')
  expect(screen.getByText('Bc4').closest('li')).toHaveAttribute('aria-current', 'step')
})

test('labels an empty route as the starting position', () => {
  render(OpeningMoveRibbon, { moves: [] })

  expect(screen.getByText('Starting position')).toBeVisible()
  expect(screen.queryByRole('button')).not.toBeInTheDocument()
})
