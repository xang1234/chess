import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import ChessBoard from './ChessBoard.svelte'

const initialFEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'

test('emits click and drag moves and flips orientation', async () => {
  const moves: string[] = []
  const { component, container } = render(ChessBoard, { fen: initialFEN, orientation: 'white' })
  component.$on('move', (event) => moves.push(event.detail.uci))

  await fireEvent.click(screen.getByRole('gridcell', { name: 'White pawn on e2' }))
  await fireEvent.click(screen.getByRole('gridcell', { name: 'Empty e4' }))
  expect(moves).toEqual(['e2e4'])

  await fireEvent.dragStart(screen.getByRole('gridcell', { name: 'White pawn on d2' }))
  await fireEvent.drop(screen.getByRole('gridcell', { name: 'Empty d4' }))
  expect(moves).toEqual(['e2e4', 'd2d4'])

  component.$set({ orientation: 'black' })
  await waitFor(() => {
    expect(container.querySelector<HTMLButtonElement>('[data-square]')?.dataset.square).toBe('h1')
  })

  component.$set({ disabled: true })
  await fireEvent.click(screen.getByRole('gridcell', { name: 'White pawn on e2' }))
  await fireEvent.click(screen.getByRole('gridcell', { name: 'Empty e4' }))
  expect(moves).toHaveLength(2)
})

test('adds the selected promotion piece to UCI', async () => {
  const moves: string[] = []
  const { component } = render(ChessBoard, {
    fen: '7k/4P3/8/8/8/8/8/K7 w - - 0 1',
    orientation: 'white'
  })
  component.$on('move', (event) => moves.push(event.detail.uci))

  await fireEvent.click(screen.getByRole('gridcell', { name: 'White pawn on e7' }))
  await fireEvent.click(screen.getByRole('gridcell', { name: 'Empty e8' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Promote to queen' }))
  expect(moves).toEqual(['e7e8q'])
})
