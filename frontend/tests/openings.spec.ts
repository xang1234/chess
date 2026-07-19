import { expect, test } from '@playwright/test'
import { chessgroundBoard, mouseClickMove } from './board-driver'
import {
  backendState,
  installTestBackend,
  selectedCoursePath
} from './test-backend'

test.beforeEach(async ({ page }) => {
  await installTestBackend(page, { kind: 'openings' })
})

test('imports and learns a private opening course without disturbing puzzles', async ({ page }) => {
  await page.goto('/')

  await page.getByLabel('Starting rating').fill('1250')
  await page.getByLabel('Puzzles per session').selectOption('5')
  await page.getByRole('button', { name: 'Save and continue' }).click()

  await page.getByRole('button', { name: 'Parent settings' }).click()
  await page.getByRole('button', { name: 'Import content' }).click()
  await page.getByRole('button', { name: 'Choose opening course' }).click()
  const selectedCourse = page.getByLabel('Selected opening course')
  await expect(selectedCourse.getByText('synthetic-italian', { exact: true })).toBeVisible()
  await expect(selectedCourse.getByText('Synthetic Italian for White', { exact: true })).toBeVisible()
  await expect(selectedCourse.getByText('/Users/family/Documents/synthetic-italian.ctcourse')).toBeVisible()
  await page.getByRole('button', { name: 'Import course' }).click()
  await expect(page.getByText('3 chapters', { exact: true })).toBeVisible()
  await expect(page.getByText('42 moves', { exact: true })).toBeVisible()
  await expect(page.getByText('5 lessons', { exact: true })).toBeVisible()
  await expect.poll(() => selectedCoursePath(page))
    .toBe('/Users/family/Documents/synthetic-italian.ctcourse')

  await page.getByRole('button', { name: 'Chess Trainer home' }).click()
  await page.getByRole('button', { name: 'Learn Openings' }).click()
  await expect(page.getByRole('heading', { name: 'Synthetic Italian for White' })).toBeVisible()
  await page.getByLabel('Course depth').selectOption('reference')
  await expect.poll(() => backendState<string[]>(page, 'openingDepths'))
    .toEqual(['reference'])

  await page.getByRole('button', { name: 'Start Prepare d4 with c3' }).click()
  await expect(page.getByText('Step 1 of 5')).toBeVisible()

  await page.getByRole('button', { name: 'Continue' }).click()
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page.getByText('Reach the Italian')).toBeVisible()
  await page.getByRole('button', { name: 'Continue' }).click()
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page.getByRole('heading', { name: 'Prepare the centre' })).toBeVisible()

  const openingBoard = chessgroundBoard(page)
  await mouseClickMove(page, openingBoard, 'b2', 'b4', 'white')
  await expect(page.getByText('Playable alternative')).toBeVisible()
  await mouseClickMove(page, openingBoard, 'd2', 'd3', 'white')
  await expect(page.getByText('This lesson is practising c3.')).toBeVisible()

  await page.getByRole('button', { name: 'Hint' }).click()
  await expect(page.getByText('Plan: prepare d4 while keeping the centre flexible.')).toBeVisible()
  await page.getByRole('button', { name: 'Hint' }).click()
  await expect(page.getByText('Start with the c-pawn.')).toBeVisible()
  await page.getByRole('button', { name: 'Hint' }).click()
  await expect(page.getByText('Move it one square.')).toBeVisible()
  await page.getByRole('button', { name: 'Hint' }).click()
  await expect(page.getByText('The course move is ready to show.')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Show course move' })).toBeVisible()
  await page.getByRole('button', { name: 'Show course move' }).click()
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page.getByText('Recognise the branch')).toBeVisible()
  await page.getByRole('button', { name: 'Pause lesson' }).click()

  await expect(page.getByRole('button', { name: 'Learn Openings' })).toBeVisible()
  await page.getByRole('button', { name: 'Learn Openings' }).click()
  await page.getByRole('button', { name: 'Continue Prepare d4 with c3' }).click()
  await expect(page.getByText('Recognise the branch')).toBeVisible()

  const resumedBoard = chessgroundBoard(page)
  await mouseClickMove(page, resumedBoard, 'c2', 'c3', 'white')
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page.getByText('Recall the Giuoco move')).toBeVisible()
  await mouseClickMove(page, resumedBoard, 'c2', 'c3', 'white')
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page.getByRole('heading', { name: 'Opening lesson complete!' })).toBeVisible()
  await expect(page.getByText('1 position recalled')).toBeVisible()
  await page.getByRole('button', { name: 'Back home' }).click()

  await page.getByRole('button', { name: 'Learn Openings' }).click()
  await page.getByRole('button', { name: 'Review 1 due position' }).click()
  await expect(page.getByText('Review the Giuoco Piano')).toBeVisible()
  const reviewBoard = chessgroundBoard(page)
  await mouseClickMove(page, reviewBoard, 'c2', 'c3', 'white')
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page.getByRole('heading', { name: 'Opening lesson complete!' })).toBeVisible()
  await page.getByRole('button', { name: 'Back home' }).click()

  await page.getByRole('button', { name: 'Learn Openings' }).click()
  await page.getByLabel('Course depth').selectOption('quick')
  await expect(page.getByText('1 of 1 lessons complete')).toBeVisible()
  await expect.poll(() => backendState<string[]>(page, 'openingDepths'))
    .toEqual(['reference', 'quick'])

  await page.getByRole('button', { name: 'Explore variations' }).click()
  await expect(page.getByRole('heading', { name: 'Variation explorer' })).toBeVisible()
  await page.getByRole('button', { name: 'e4 — Italian setup' }).click()
  await expect(page.getByRole('heading', { name: 'King pawn opening' })).toBeVisible()
  await page.getByRole('button', { name: 'Back one move' }).click()
  await expect(page.getByRole('heading', { name: 'Initial position' })).toBeVisible()
  await page.getByRole('button', { name: 'Back to course' }).click()
  await page.getByRole('button', { name: 'Back' }).click()

  await page.getByRole('button', { name: "Start today's training" }).click()
  const puzzleBoard = chessgroundBoard(page)
  await mouseClickMove(page, puzzleBoard, 'e2', 'e4', 'white')
  await expect(page.getByText('Correct!')).toBeVisible()

  await expect.poll(() => backendState<string[]>(page, 'openingMoves'))
    .toEqual(['b2b4', 'd2d3', 'c2c3', 'c2c3', 'c2c3'])
  await expect.poll(() => backendState<number[]>(page, 'openingHints'))
    .toEqual([1, 2, 3, 4])
})
