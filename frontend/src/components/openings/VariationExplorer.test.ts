import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import type { OpeningPositionView } from '../../lib/api'
import { fakeAPI, withNormalAPI } from '../../test-fakes'
import type {
  ChessBoardAdapter,
  ChessgroundAdapterFactory
} from '../chess/chessground-adapter'
import VariationExplorer from './VariationExplorer.svelte'

const rootFen = 'rnbqk1nr/pppp1ppp/2n5/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3'
const destinationFen = 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4'

const root: OpeningPositionView = {
  courseId: 'italian-white',
  positionId: 'after-bc4',
  fen: rootFen,
  label: 'Italian Game',
  evaluation: { code: 'equal', sourceSymbol: '=' },
  notes: [{
    kind: 'overview',
    text: 'Black chooses how to develop the king bishop.',
    sourceRef: { printedPage: 18, noteLabel: 'a', coverageId: 'p18-a' }
  }],
  moves: [{
    moveId: 'black-bc5',
    uci: 'f8c5',
    san: 'Bc5',
    toPositionId: 'after-bc5',
    role: 'opponent',
    variationName: 'Giuoco Piano',
    evaluation: { code: 'equal' },
    sourceRef: { printedPage: 18, tableColumn: 'I', coverageId: 'p18-bc5' }
  }],
  incomingPaths: 1
}

const destination: OpeningPositionView = {
  courseId: 'italian-white',
  positionId: 'after-bc5',
  fen: destinationFen,
  label: 'Giuoco Piano',
  evaluation: { code: 'white_slight', sourceSymbol: '+=' },
  notes: [{
    kind: 'plan',
    text: 'White prepares d4 with c3.',
    sourceRef: { printedPage: 19, coverageId: 'p19-plan' }
  }],
  moves: [],
  incomingPaths: 2
}

function boardHarness() {
  const fens: string[] = []
  const factory: ChessgroundAdapterFactory = vi.fn((_element, fen) => {
    fens.push(fen)
    const adapter: ChessBoardAdapter = {
      configure: vi.fn(),
      setPosition: vi.fn(),
      selectSquare: vi.fn(),
      destroy: vi.fn()
    }
    return adapter
  })
  return { factory, fens }
}

test('navigates read-only course branches with local Back history', async () => {
  const getOpeningPosition = vi.fn(async (_courseId: string, positionId: string) =>
    positionId === root.positionId ? root : destination)
  const startOpeningLesson = vi.fn()
  const playOpeningMove = vi.fn()
  const useOpeningHint = vi.fn()
  const board = boardHarness()
  render(VariationExplorer, {
    courseId: root.courseId,
    rootPositionId: root.positionId,
    depth: 'reference',
    boardAdapterFactory: board.factory
  }, withNormalAPI(fakeAPI({
    getOpeningPosition,
    startOpeningLesson,
    playOpeningMove,
    useOpeningHint
  })))

  expect(await screen.findByRole('heading', { name: 'Italian Game' })).toBeInTheDocument()
  expect(board.fens[0]).toBe(rootFen)
  expect(screen.getByText('Equal (=)')).toBeInTheDocument()
  expect(screen.getByText('Black chooses how to develop the king bishop.')).toBeInTheDocument()
  expect(screen.getByText('Source page 18 · note a')).toBeInTheDocument()
  expect(screen.getByText('Source page 18 · column I')).toBeInTheDocument()
  expect(screen.getByRole('grid')).toHaveAttribute('aria-disabled', 'true')

  await fireEvent.click(screen.getByRole('button', { name: 'Bc5 — Giuoco Piano' }))

  expect(getOpeningPosition).toHaveBeenLastCalledWith(
    'italian-white', 'after-bc5', 'reference'
  )
  expect(await screen.findByRole('heading', { name: 'Giuoco Piano' })).toBeInTheDocument()
  expect(screen.getByText('White is slightly better (+=)')).toBeInTheDocument()
  expect(screen.getByText('This position is reached by 2 move orders.')).toBeInTheDocument()
  expect(board.fens.at(-1)).toBe(destinationFen)

  await fireEvent.click(screen.getByRole('button', { name: 'Back one move' }))

  expect(await screen.findByRole('heading', { name: 'Italian Game' })).toBeInTheDocument()
  expect(getOpeningPosition).toHaveBeenCalledTimes(2)
  await waitFor(() => expect(board.fens.at(-1)).toBe(rootFen))
  expect(startOpeningLesson).not.toHaveBeenCalled()
  expect(playOpeningMove).not.toHaveBeenCalled()
  expect(useOpeningHint).not.toHaveBeenCalled()
})
