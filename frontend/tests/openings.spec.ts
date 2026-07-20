import { expect, test } from '@playwright/test'
import { chessgroundBoard, mouseClickMove } from './board-driver'
import { installTestBackend } from './test-backend'
import { backendState, selectedCoursePath } from './test-backend-state'

test.beforeEach(async ({ page }) => {
  await installTestBackend(page, { kind: 'openings' })
})

test('learns a private two-node teaching tree and preserves the journey', async ({ page }) => {
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
  await expect(page.getByText('2 chapters', { exact: true })).toBeVisible()
  await expect(page.getByText('42 moves', { exact: true })).toBeVisible()
  await expect(page.getByText('2 lessons', { exact: true })).toBeVisible()
  await expect.poll(() => selectedCoursePath(page))
    .toBe('/Users/family/Documents/synthetic-italian.ctcourse')

  await page.getByRole('button', { name: 'Chess Trainer home' }).click()
  await page.getByRole('button', { name: 'Learn Openings' }).click()
  await expect(page.getByRole('heading', { name: 'Synthetic Italian for White' })).toBeVisible()
  await expect(page.getByRole('tree', { name: 'Synthetic Italian for White course roadmap' })).toBeVisible()
  await expect(page.getByRole('treeitem', { name: /Prepare d4 with c3.*Recommended.*0 of 3 ideas/i })).toBeVisible()
  await expect(page.getByRole('treeitem', { name: /Choose the quiet d3 setup.*0 of 1 ideas/i })).toBeVisible()

  await page.getByLabel('Course depth').selectOption('quick')
  const hiddenD3 = page.getByRole('treeitem', {
    name: /Choose the quiet d3 setup.*Hidden at this depth/i
  })
  await expect(hiddenD3).toBeVisible()
  await expect(hiddenD3.getByRole('button', { name: 'Study Choose the quiet d3 setup' })).toBeDisabled()

  await page.getByLabel('Course depth').selectOption('reference')
  await expect.poll(() => backendState<string[]>(page, 'openingDepths'))
    .toEqual(['quick', 'reference'])
  await page.getByRole('button', { name: 'Study Prepare d4 with c3' }).click()

  await expect(page.getByRole('heading', { name: 'Build the centre' })).toBeVisible()
  await expect(page.getByText('Idea 1 of 3 · 0 learned')).toBeVisible()
  await expect(page.getByText('Connect c3 with the later d4 break; this is one plan, not a move drill.')).toBeVisible()
  const referenceText = page.getByText(
    'Reference analysis: c3 supports d4 while preserving the active bishop.'
  )
  await expect(referenceText).not.toBeVisible()
  await page.getByText('Deeper analysis', { exact: true }).click()
  await expect(referenceText).toBeVisible()

  await page.getByRole('button', { name: 'Continue' }).click()
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page.getByRole('heading', { name: 'Choose the preparation' })).toBeVisible()

  const c3Board = chessgroundBoard(page)
  await mouseClickMove(page, c3Board, 'c2', 'c3', 'white')
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page.getByRole('heading', { name: 'Keep the plan' })).toBeVisible()
  await expect(page.getByText('You learned the purpose of c3; you do not need to replay it again here.')).toBeVisible()

  await page.getByRole('button', { name: 'Continue' }).click()
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page.getByRole('heading', { name: 'Prepare d4 with c3 complete' })).toBeVisible()
  await expect(page.getByText('1 of 2 lessons complete')).toBeVisible()
  await expect(page.getByRole('navigation', { name: 'Opening course path' })).toContainText('Prepare d4 with c3')

  await page.getByRole('button', { name: 'Continue to Choose the quiet d3 setup' }).click()
  await expect(page.getByRole('heading', { name: 'Choose the quiet setup' })).toBeVisible()
  const d3Path = page.getByRole('navigation', { name: 'Opening course path' })
  await expect(d3Path).toContainText('Prepare d4 with c3')
  await expect(d3Path).toContainText('Choose the quiet d3 setup')
  await page.getByRole('button', { name: 'Pause lesson' }).click()

  await expect(page.getByRole('button', { name: 'Learn Openings' })).toBeVisible()
  await page.getByRole('button', { name: 'Learn Openings' }).click()
  await expect(page.getByText('1 of 2 lessons complete')).toBeVisible()
  await expect(page.getByRole('treeitem', { name: /Prepare d4 with c3.*Complete.*Review due/i })).toBeVisible()
  await expect(page.getByRole('treeitem', { name: /Choose the quiet d3 setup.*In progress.*Recommended/i })).toBeVisible()

  await page.getByLabel('Course depth').selectOption('quick')
  await expect(page.getByText('1 of 1 lessons complete')).toBeVisible()
  await expect(page.getByRole('treeitem', {
    name: /Choose the quiet d3 setup.*In progress.*Hidden at this depth/i
  })).toBeVisible()
  await page.getByLabel('Course depth').selectOption('reference')
  await page.getByRole('button', { name: 'Continue learning — Choose the quiet d3 setup' }).click()
  await expect(page.getByRole('heading', { name: 'Choose the quiet setup' })).toBeVisible()
  await expect(page.getByText('Idea 1 of 1 · 0 learned')).toBeVisible()

  const d3Board = chessgroundBoard(page)
  await mouseClickMove(page, d3Board, 'd2', 'd3', 'white')
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page.getByRole('heading', { name: 'Choose the quiet d3 setup complete' })).toBeVisible()
  await expect(page.getByText('2 of 2 lessons complete')).toBeVisible()
  await page.getByRole('button', { name: 'View course tree' }).click()

  await expect(page.getByText('2 of 2 lessons complete')).toBeVisible()
  await page.getByRole('button', { name: 'Review 1 due position' }).click()
  await expect(page.getByRole('heading', { name: 'Review the Giuoco Piano' })).toBeVisible()
  const reviewBoard = chessgroundBoard(page)
  await mouseClickMove(page, reviewBoard, 'c2', 'c3', 'white')
  await page.getByRole('button', { name: 'Continue' }).click()
  await expect(page.getByRole('heading', { name: 'Opening review complete!' })).toBeVisible()
  await expect(page.getByText('1 position recalled')).toBeVisible()
  await page.getByRole('button', { name: 'Back to course' }).click()

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
    .toEqual(['c2c3', 'd2d3', 'c2c3'])
  await expect.poll(() => backendState<number[]>(page, 'openingHints'))
    .toEqual([])
})
