import { render, screen, within } from '@testing-library/svelte'
import type { OpeningPositionContext as PositionContext } from '../../lib/api'
import { fakeOpeningPositionContext } from '../../test-fakes'
import OpeningPositionContext from './OpeningPositionContext.svelte'

test('explains the named position, its route, and authored continuations without controls', () => {
  render(OpeningPositionContext, {
    props: { context: fakeOpeningPositionContext, orientation: 'white' }
  })

  const card = screen.getByRole('region', { name: 'Position: Giuoco Piano' })
  expect(within(card).getByRole('heading', { name: 'Giuoco Piano' })).toBeVisible()
  expect(within(card).getByText('= Equal chances')).toBeVisible()
  expect(within(card).getByText('Prepare d4 while keeping the bishop active.')).toBeVisible()

  const arrivals = within(card).getByRole('list', { name: 'You arrived via' })
  expect(arrivals).toHaveTextContent('Bc5')
  expect(arrivals).toHaveTextContent('Giuoco Piano')
  expect(arrivals).toHaveTextContent('Played')
  expect(within(arrivals).getByRole('listitem')).toHaveAttribute('aria-current', 'true')

  const continuations = within(card).getByRole('list', { name: 'From here' })
  expect(continuations).toHaveTextContent('c3')
  expect(continuations).toHaveTextContent('Giuoco Piano')
  expect(within(card).queryAllByRole('button')).toHaveLength(0)
})

test('uses useful root and leaf labels without inventing a plan', () => {
  const context: PositionContext = {
    positionId: 'initial',
    label: '',
    evaluation: { code: 'none' },
    arrivedVia: [],
    nextMoves: []
  }
  render(OpeningPositionContext, { props: { context, orientation: 'black' } })

  const card = screen.getByRole('region', { name: 'Position: Current position' })
  expect(within(card).getByRole('heading', { name: 'Current position' })).toBeVisible()
  expect(within(card).getByText('Starting position')).toBeVisible()
  expect(within(card).getByText('End of this course line')).toBeVisible()
  expect(within(card).queryByText(/plan/i)).not.toBeInTheDocument()
  expect(within(card).queryAllByRole('button')).toHaveLength(0)
})
