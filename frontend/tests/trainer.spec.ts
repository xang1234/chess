import { expect, test } from '@playwright/test'
import { chessgroundBoard, mouseClickMove, pieceKeys } from './board-driver'
import {
  holdImportOpen,
  installTestBackend,
  reportImportProgress,
  selectedImportPath
} from './test-backend'

test.beforeEach(async ({ page }) => {
  await installTestBackend(page, { kind: 'trainer' })
})

test('runs setup, import, guided training, practice, and parent progress', async ({ page }) => {
  await page.goto('/')

  await expect(page.getByText('Set up today’s training')).toBeVisible()
  await page.getByLabel('Starting rating').fill('1250')
  await page.getByLabel('Puzzles per session').selectOption('5')
  await page.getByRole('button', { name: 'Save and continue' }).click()
  await expect(page.getByText('What would you like to play?')).toBeVisible()

  await page.getByRole('button', { name: 'Parent settings' }).click()
  await page.getByRole('button', { name: 'Import puzzles' }).click()
  await page.getByRole('button', { name: 'Choose puzzle database' }).click()
  await expect(page.getByText('lichess_db_puzzle.csv.zst', { exact: true })).toBeVisible()
  await expect(page.getByText('/Users/family/Downloads/lichess_db_puzzle.csv.zst')).toBeVisible()
  await page.getByRole('button', { name: 'Import puzzles' }).click()
  await expect(page.getByText('9,800 accepted')).toBeVisible()
  await expect(page.getByText('150 duplicates')).toBeVisible()
  await expect(page.getByText('50 rejected')).toBeVisible()

  await page.getByRole('button', { name: 'Chess Trainer home' }).click()
  await page.getByRole('button', { name: "Start today's training" }).click()
  const board = chessgroundBoard(page)
  await mouseClickMove(page, board, 'e2', 'e3', 'white')
  await expect(page.getByText('Try again')).toBeVisible()
  await mouseClickMove(page, board, 'e2', 'e4', 'white')
  await expect(page.getByText('Correct!')).toBeVisible()
  await expect(page.getByText('Puzzle 1 of 2', { exact: true })).toBeVisible()
  await expect.poll(() => pieceKeys(board, 'piece.white.pawn')).toContain('e4')
  await page.getByRole('button', { name: 'Next puzzle' }).click()
  await expect(page.getByText('Puzzle 2 of 2')).toBeVisible()

  await page.getByRole('button', { name: 'Pause' }).click()
  await expect(page.getByRole('button', { name: "Continue today's training" })).toBeVisible()
  await page.getByRole('button', { name: "Continue today's training" }).click()
  await page.getByRole('button', { name: 'Hint' }).click()
  await expect(page.getByText('Look for: fork')).toBeVisible()
  await page.getByRole('button', { name: 'Hint' }).click()
  await page.getByRole('button', { name: 'Hint' }).click()
  await expect(page.getByRole('button', { name: 'Show solution' })).toBeVisible()
  await page.getByRole('button', { name: 'Show solution' }).click()
  await expect(page.getByText('Solution shown')).toBeVisible()
  await expect(page.getByText('Puzzle 2 of 2', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'See results' }).click()
  await expect(page.getByText('Training complete!')).toBeVisible()
  await expect(page.getByText('1 retried')).toBeVisible()
  await expect(page.getByText('1 solution shown')).toBeVisible()
  await page.getByRole('button', { name: 'Back home' }).click()

  await page.getByRole('button', { name: 'Free Practice' }).click()
  await page.getByLabel('Limit by rating').check()
  await page.getByLabel('Minimum rating').fill('1100')
  await page.getByLabel('Maximum rating').fill('1500')
  await page.getByLabel('fork').check()
  await page.getByLabel('Maximum solution length').selectOption('3')
  await page.getByRole('button', { name: 'Start practice' }).click()
  await expect(page.getByText('Find the best move')).toBeVisible()

  await page.getByRole('button', { name: 'Chess Trainer home' }).click()
  await page.getByRole('button', { name: 'Parent settings' }).click()
  const metrics = page.locator('.metric-grid')
  await expect(metrics.getByText('50%', { exact: true })).toBeVisible()
  await expect(metrics.getByText('25%', { exact: true })).toBeVisible()
  await expect(page.getByRole('cell', { name: 'fork' })).toBeVisible()
  await expect(page.getByRole('cell', { name: 'Guided' })).toBeVisible()
})

test('keeps an active import observable across navigation and can cancel it', async ({ page }) => {
  await page.goto('/')
  await page.getByLabel('Starting rating').fill('1250')
  await page.getByRole('button', { name: 'Save and continue' }).click()
  await holdImportOpen(page)

  await page.getByRole('button', { name: 'Parent settings' }).click()
  await page.getByRole('button', { name: 'Import puzzles' }).click()
  await page.getByRole('button', { name: 'Choose puzzle database' }).click()
  await page.getByRole('button', { name: 'Import puzzles' }).click()
  await expect.poll(() => selectedImportPath(page))
    .toBe('/Users/family/Downloads/lichess_db_puzzle.csv.zst')

  await page.getByRole('button', { name: 'Chess Trainer home' }).click()
  await reportImportProgress(page, 10_000, 2048)
  await page.getByRole('button', { name: 'Parent settings' }).click()
  await page.getByRole('button', { name: 'Import puzzles' }).click()

  await expect(page.getByText('10,000 rows read')).toBeVisible()
  await page.getByRole('button', { name: 'Cancel import' }).click()
  await expect(page.getByText('Import cancelled.')).toBeVisible()
})
