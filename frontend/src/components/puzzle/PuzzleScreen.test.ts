import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { vi } from 'vitest'
import type { HintResult, MoveResult, SessionView } from '../../lib/api'
import { setAPIForTests } from '../../lib/api'
import { fakeAPI } from '../../test-fakes'
import PuzzleScreen from './PuzzleScreen.svelte'

const sourceFen = '4k3/4p3/8/8/8/8/4P3/4K3 b - - 0 1'
const puzzleFen = '4k3/8/8/4p3/8/8/4P3/4K3 w - - 0 2'
const advancedFen = '4k3/8/8/4p3/4P3/8/8/4K3 b - - 0 2'

function session(currentFen = puzzleFen): SessionView {
  return {
    sessionId: 'session-1',
    mode: 'guided',
    status: 'active',
    currentIndex: 0,
    total: 1,
    current: {
      fingerprint: 'puzzle-1',
      sourceFen,
      displayedFen: puzzleFen,
      currentFen,
      preludeUci: 'e7e5',
      solver: 'white',
      currentPath: [],
      puzzleNumber: 1,
      puzzleTotal: 1,
      hintLevel: 0,
      incorrectMoves: 0,
      canReveal: false
    }
  }
}

test('shows the source position before enabling the puzzle position', async () => {
  render(PuzzleScreen, { session: session() })

  expect(screen.getByRole('gridcell', { name: 'Black pawn on e7' })).toBeDisabled()
  expect(screen.getByText('Watch the last move…')).toBeInTheDocument()

  await waitFor(() => {
    expect(screen.getByRole('gridcell', { name: 'Black pawn on e5' })).not.toBeDisabled()
  }, { timeout: 1500 })
  expect(screen.getByText('Find the best move')).toBeInTheDocument()
})

test('keeps the position after a wrong move and applies the returned FEN after a correct move', async () => {
  const playMove = vi.fn<Parameters<ReturnType<typeof fakeAPI>['playMove']>, ReturnType<ReturnType<typeof fakeAPI>['playMove']>>()
  const wrong: MoveResult = {
    session: session(),
    correct: false,
    puzzleCompleted: false,
    message: 'Try again'
  }
  const correct: MoveResult = {
    session: session(advancedFen),
    correct: true,
    puzzleCompleted: false
  }
  playMove.mockResolvedValueOnce(wrong).mockResolvedValueOnce(correct)
  setAPIForTests(fakeAPI({ playMove }))
  render(PuzzleScreen, { session: { ...session(), current: { ...session().current!, sourceFen: undefined, preludeUci: undefined } } })

  await fireEvent.click(screen.getByRole('gridcell', { name: 'White pawn on e2' }))
  await fireEvent.click(screen.getByRole('gridcell', { name: 'Empty e3' }))
  await waitFor(() => expect(screen.getByText('Try again')).toBeInTheDocument())
  expect(screen.getByRole('gridcell', { name: 'White pawn on e2' })).toBeInTheDocument()

  await fireEvent.click(screen.getByRole('gridcell', { name: 'White pawn on e2' }))
  await fireEvent.click(screen.getByRole('gridcell', { name: 'Empty e4' }))
  await waitFor(() => {
    expect(screen.getByRole('gridcell', { name: 'White pawn on e4' })).toBeInTheDocument()
  })
  expect(playMove).toHaveBeenNthCalledWith(1, 'session-1', 'e2e3')
  expect(playMove).toHaveBeenNthCalledWith(2, 'session-1', 'e2e4')
})

test('reveals hints progressively and only offers the solution after level three', async () => {
  const hints: HintResult[] = [
    { level: 1, text: 'Look for: fork', canReveal: false },
    { level: 2, text: 'Start with this piece.', sourceSquare: 'e2', canReveal: false },
    { level: 3, text: 'Try this destination.', sourceSquare: 'e2', targetSquare: 'e4', canReveal: true }
  ]
  const useHint = vi.fn()
    .mockResolvedValueOnce(hints[0])
    .mockResolvedValueOnce(hints[1])
    .mockResolvedValueOnce(hints[2])
  setAPIForTests(fakeAPI({ useHint }))
  const { container } = render(PuzzleScreen, {
    session: { ...session(), current: { ...session().current!, sourceFen: undefined, preludeUci: undefined } }
  })

  expect(screen.queryByRole('button', { name: 'Show solution' })).not.toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', { name: 'Hint' }))
  expect(await screen.findByText('Look for: fork')).toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', { name: 'Hint' }))
  await waitFor(() => expect(container.querySelector('[data-square="e2"]')).toHaveClass('hint-source'))
  expect(screen.queryByRole('button', { name: 'Show solution' })).not.toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', { name: 'Hint' }))
  await waitFor(() => expect(container.querySelector('[data-square="e4"]')).toHaveClass('hint-target'))
  expect(screen.getByRole('button', { name: 'Show solution' })).toBeInTheDocument()
})

test('pauses to home and presents a child-friendly completion summary', async () => {
  const pauseSession = vi.fn().mockResolvedValue(undefined)
  setAPIForTests(fakeAPI({ pauseSession }))
  const { component } = render(PuzzleScreen, {
    session: { ...session(), current: { ...session().current!, sourceFen: undefined, preludeUci: undefined } }
  })
  let wentHome = false
  component.$on('home', () => { wentHome = true })

  await fireEvent.click(screen.getByRole('button', { name: 'Pause' }))
  await waitFor(() => expect(wentHome).toBe(true))
  expect(pauseSession).toHaveBeenCalledWith('session-1')

  component.$set({
    session: {
      sessionId: 'session-1', mode: 'guided', status: 'complete', currentIndex: 5, total: 5,
      summary: { total: 5, firstTry: 2, retried: 1, usedHint: 1, revealed: 1, unavailable: 0 }
    }
  })
  await waitFor(() => expect(screen.getByText('Training complete!')).toBeInTheDocument())
  expect(screen.getByText('2 first try')).toBeInTheDocument()
  expect(screen.getByText('1 retried')).toBeInTheDocument()
  expect(screen.getByText('1 used a hint')).toBeInTheDocument()
  expect(screen.getByText('1 solution shown')).toBeInTheDocument()
})
