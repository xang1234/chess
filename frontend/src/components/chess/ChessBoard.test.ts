import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import type { Key } from '@lichess-org/chessground/types'
import ChessBoard from './ChessBoard.svelte'
import type {
  BoardCallbacks,
  BoardInteraction,
  ChessBoardAdapter,
  ChessgroundAdapterFactory
} from './chessground-adapter'

const initialFen = '4k3/8/8/8/8/8/3PP3/4K3 w - - 0 1'

function adapterHarness() {
  const interactions: BoardInteraction[] = []
  const creations: Array<{ element: HTMLElement; fen: string; callbacks: BoardCallbacks }> = []
  const adapter: ChessBoardAdapter = {
    configure: vi.fn((interaction) => { interactions.push(interaction) }),
    setPosition: vi.fn(),
    selectSquare: vi.fn(),
    destroy: vi.fn()
  }
  const factory: ChessgroundAdapterFactory = vi.fn((element, fen, interaction, callbacks) => {
    interactions.push(interaction)
    creations.push({ element, fen, callbacks })
    return adapter
  })
  return { adapter, creations, factory, interactions }
}

function boardProps(factory: ChessgroundAdapterFactory) {
  return {
    fen: initialFen,
    orientation: 'white' as const,
    legalMoves: ['d2d3', 'd2d4', 'e2e3', 'e2e4'],
    inputEnabled: true,
    adapterFactory: factory
  }
}

test('owns adapter creation, interaction updates, explicit FEN reconciliation, and destruction', async () => {
  const { adapter, creations, factory, interactions } = adapterHarness()
  const { component, unmount } = render(ChessBoard, boardProps(factory))

  await waitFor(() => expect(creations).toHaveLength(1))
  expect(creations[0].fen).toBe(initialFen)
  expect(creations[0].element).toHaveAttribute('aria-hidden', 'true')
  expect(interactions[0]).toMatchObject({
    orientation: 'white',
    legalMoves: boardProps(factory).legalMoves,
    inputEnabled: true,
    keyboardCursor: 'd2'
  })

  component.$set({
    orientation: 'black',
    inputEnabled: false,
    hintSource: 'e2',
    hintTarget: 'e4'
  })
  await waitFor(() => expect(adapter.configure).toHaveBeenCalled())
  expect(interactions.at(-1)).toMatchObject({
    orientation: 'black',
    inputEnabled: false,
    hintSource: 'e2',
    hintTarget: 'e4'
  })

  component.setPosition('4k3/8/8/8/4P3/8/3P4/4K3 b - - 0 1', ['e2', 'e4'], true)
  expect(adapter.setPosition).toHaveBeenCalledWith(
    '4k3/8/8/8/4P3/8/3P4/4K3 b - - 0 1',
    ['e2', 'e4'],
    true
  )

  unmount()
  expect(adapter.destroy).toHaveBeenCalledTimes(1)
})

test('emits only complete legal UCI routes and reports rejected adapter routes', async () => {
  const { adapter, creations, factory } = adapterHarness()
  const moves: string[] = []
  const errors: string[] = []
  const { component } = render(ChessBoard, boardProps(factory))
  component.$on('move', (event) => moves.push(event.detail.uci))
  component.$on('error', (event) => errors.push(event.detail.message))
  await waitFor(() => expect(creations).toHaveLength(1))

  creations[0].callbacks.onRoute('e2', 'e4')
  creations[0].callbacks.onRoute('e2', 'e5')

  expect(moves).toEqual(['e2e4'])
  expect(errors[0]).toMatch(/e2e5.*not legal/i)
  expect(adapter.setPosition).toHaveBeenCalledWith(initialFen, undefined, false)
})

test('restricts promotion suffixes and emits only after a legal choice', async () => {
  const { creations, factory } = adapterHarness()
  const moves: string[] = []
  const { component } = render(ChessBoard, {
    fen: '4k3/4P3/8/8/8/8/8/4K3 w - - 0 1',
    orientation: 'white',
    legalMoves: ['e7e8q', 'e7e8n'],
    inputEnabled: true,
    adapterFactory: factory
  })
  component.$on('move', (event) => moves.push(event.detail.uci))
  await waitFor(() => expect(creations).toHaveLength(1))

  creations[0].callbacks.onRoute('e7', 'e8')
  expect(moves).toEqual([])
  const knight = await screen.findByRole('button', { name: 'Promote to knight' })
  expect(screen.queryByRole('button', { name: 'Promote to rook' })).not.toBeInTheDocument()
  await fireEvent.click(knight)

  expect(moves).toEqual(['e7e8n'])
})

test('cancelling promotion restores the authoritative position and board focus', async () => {
  const { adapter, creations, factory } = adapterHarness()
  render(ChessBoard, {
    fen: '4k3/4P3/8/8/8/8/8/4K3 w - - 0 1',
    orientation: 'white',
    legalMoves: ['e7e8q'],
    inputEnabled: true,
    adapterFactory: factory
  })
  await waitFor(() => expect(creations).toHaveLength(1))
  const board = screen.getByRole('grid', { name: 'Chess board, white side' })
  board.focus()

  creations[0].callbacks.onRoute('e7', 'e8')
  await fireEvent.keyDown(await screen.findByRole('dialog'), { key: 'Escape' })

  expect(adapter.setPosition).toHaveBeenCalledWith(
    '4k3/4P3/8/8/8/8/8/4K3 w - - 0 1',
    undefined,
    false
  )
  expect(board).toHaveFocus()
})

test('locks input and emits an actionable error when legal-move data becomes malformed', async () => {
  const { creations, factory, interactions } = adapterHarness()
  const errors: string[] = []
  const moves: string[] = []
  const { component } = render(ChessBoard, boardProps(factory))
  component.$on('error', (event) => errors.push(event.detail.message))
  component.$on('move', (event) => moves.push(event.detail.uci))
  await waitFor(() => expect(creations).toHaveLength(1))

  component.$set({ legalMoves: ['not-UCI'] })
  await waitFor(() => expect(errors).toHaveLength(1))

  expect(errors[0]).toMatch(/not-UCI.*locked/i)
  expect(interactions.at(-1)).toMatchObject({ inputEnabled: false, legalMoves: [] })
  creations[0].callbacks.onRoute('e2', 'e4')
  expect(moves).toEqual([])
})

test('locks the focusable board when the adapter cannot start', async () => {
  const factory: ChessgroundAdapterFactory = () => {
    throw new Error('mount failed')
  }
  render(ChessBoard, boardProps(factory))

  expect(await screen.findByRole('alert')).toHaveTextContent(/mount failed.*locked/i)
  expect(screen.getByRole('grid', { name: 'Chess board, white side' }))
    .toHaveAttribute('aria-disabled', 'true')
})

test('moves the keyboard cursor by orientation and exposes its semantic square', async () => {
  const { factory, interactions } = adapterHarness()
  const { component } = render(ChessBoard, boardProps(factory))
  const board = await screen.findByRole('grid', { name: 'Chess board, white side' })

  await fireEvent.keyDown(board, { key: 'ArrowUp' })
  await waitFor(() => expect(interactions.at(-1)?.keyboardCursor).toBe('d3'))
  expect(board.getAttribute('aria-activedescendant')).toMatch(/d3$/)

  component.$set({ orientation: 'black' })
  await fireEvent.keyDown(board, { key: 'ArrowUp' })
  await waitFor(() => expect(interactions.at(-1)?.keyboardCursor).toBe('d2'))
})

test('clears wrapper selection when a selected source is clicked again', async () => {
  const { adapter, creations, factory } = adapterHarness()
  render(ChessBoard, boardProps(factory))
  await waitFor(() => expect(creations).toHaveLength(1))
  const source = screen.getByRole('gridcell', { name: 'White pawn on d2' })

  creations[0].callbacks.onSelect('d2')
  await waitFor(() => expect(source).toHaveAttribute('aria-selected', 'true'))

  creations[0].callbacks.onSelect('d2')

  await waitFor(() => expect(source).toHaveAttribute('aria-selected', 'false'))
  expect(adapter.selectSquare).toHaveBeenLastCalledWith(null)
})

test('uses Enter and Space to select, change source, submit destinations, and Escape to clear', async () => {
  const { adapter, creations, factory } = adapterHarness()
  const moves: string[] = []
  const { component } = render(ChessBoard, boardProps(factory))
  component.$on('move', (event) => moves.push(event.detail.uci))
  await waitFor(() => expect(creations).toHaveLength(1))
  const board = screen.getByRole('grid', { name: 'Chess board, white side' })

  await fireEvent.keyDown(board, { key: 'Enter' })
  expect(adapter.selectSquare).toHaveBeenLastCalledWith('d2')

  await fireEvent.keyDown(board, { key: 'ArrowRight' })
  await fireEvent.keyDown(board, { key: ' ' })
  expect(adapter.selectSquare).toHaveBeenLastCalledWith('e2')

  await fireEvent.keyDown(board, { key: 'ArrowUp' })
  await fireEvent.keyDown(board, { key: 'ArrowUp' })
  await fireEvent.keyDown(board, { key: 'Enter' })
  expect(adapter.selectSquare).toHaveBeenLastCalledWith('e4')
  creations[0].callbacks.onRoute('e2', 'e4')
  expect(moves).toEqual(['e2e4'])

  creations[0].callbacks.onSelect('a1' as Key)
  expect(adapter.selectSquare).toHaveBeenLastCalledWith(null)
  await fireEvent.keyDown(board, { key: 'Escape' })
  expect(adapter.selectSquare).toHaveBeenLastCalledWith(null)
})

test('keeps an offscreen semantic grid backed by the authoritative FEN', async () => {
  const { factory } = adapterHarness()
  const { component } = render(ChessBoard, boardProps(factory))

  const rows = screen.getAllByRole('row')
  expect(rows).toHaveLength(8)
  rows.forEach((row) => expect(row.querySelectorAll('[role="gridcell"]')).toHaveLength(8))
  expect(screen.getByRole('gridcell', { name: 'White pawn on e2' })).toBeInTheDocument()
  expect(screen.getByRole('gridcell', { name: 'Empty e4' })).toBeInTheDocument()

  component.setPosition('4k3/8/8/8/4P3/8/3P4/4K3 b - - 0 1', ['e2', 'e4'], false)
  await waitFor(() => {
    expect(screen.getByRole('gridcell', { name: 'White pawn on e4' })).toBeInTheDocument()
  })
})
