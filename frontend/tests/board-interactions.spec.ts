import { expect, test, type Locator, type Page, type TestInfo } from '@playwright/test'
import {
  chessgroundBoard,
  markerForSquare,
  mouseCancelDrag,
  mouseClickMove,
  mouseDragMove,
  pieceKeys,
  squarePoint,
  touchTapMove
} from './board-driver'
import {
  backendState,
  installTestBackend,
  observeSemanticFrames,
  type BoardScenario,
  type ContinueResponse,
  type NextResponse,
  type PuzzleDefinition,
  type SummaryResponse
} from './test-backend'

const startingFen = '4k3/8/8/4p3/8/8/4P3/4K3 w - - 0 2'
const afterE4 = '4k3/8/8/4p3/4P3/8/8/4K3 b - - 0 2'
const replyStartingFen = '4k3/8/8/3p4/8/8/4P3/4K3 w - - 0 2'
const afterReplyUserMove = '4k3/8/8/3p4/4P3/8/8/4K3 b - - 0 2'
const afterReplyCapture = '4k3/8/8/8/4p3/8/8/4K3 w - - 0 3'
const nextFen = '4k3/8/8/8/8/8/3P4/4K3 w - - 0 1'

type RGB = { red: number; green: number; blue: number }

function parseRGB(value: string): RGB {
  const channels = value.match(/[\d.]+/g)?.slice(0, 3).map(Number)
  if (!channels || channels.length !== 3) {
    throw new Error(`Expected an rgb() color, received ${value}`)
  }
  return { red: channels[0], green: channels[1], blue: channels[2] }
}

function relativeLuminance({ red, green, blue }: RGB): number {
  const linear = [red, green, blue].map((channel) => {
    const value = channel / 255
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * linear[0] + 0.7152 * linear[1] + 0.0722 * linear[2]
}

async function contrastRatio(foreground: Locator, background: Locator): Promise<number> {
  const [foregroundColor, backgroundColor] = await Promise.all([
    foreground.evaluate((element) => getComputedStyle(element).color),
    background.evaluate((element) => getComputedStyle(element).backgroundColor)
  ])
  const foregroundLuminance = relativeLuminance(parseRGB(foregroundColor))
  const backgroundLuminance = relativeLuminance(parseRGB(backgroundColor))
  const lighter = Math.max(foregroundLuminance, backgroundLuminance)
  const darker = Math.min(foregroundLuminance, backgroundLuminance)
  return (lighter + 0.05) / (darker + 0.05)
}

function basicPuzzle(overrides: Partial<PuzzleDefinition> = {}): PuzzleDefinition {
  return {
    fingerprint: 'board-puzzle-1',
    fen: startingFen,
    solver: 'white',
    legalMoves: ['e2e3', 'e2e4'],
    ...overrides
  }
}

function nextPuzzle(): PuzzleDefinition {
  return {
    fingerprint: 'board-puzzle-2',
    fen: nextFen,
    solver: 'white',
    legalMoves: ['d2d3', 'd2d4']
  }
}

function completedMove(
  overrides: Partial<Omit<NextResponse, 'kind'>> = {}
): NextResponse {
  return {
    kind: 'next',
    uci: 'e2e4',
    appliedMoves: [{ uci: 'e2e4', resultingFen: afterE4 }],
    finalFen: afterE4,
    next: nextPuzzle(),
    ...overrides
  }
}

function summaryMove(
  overrides: Partial<Omit<SummaryResponse, 'kind'>> = {}
): SummaryResponse {
  return { ...completedMove(overrides), kind: 'summary' }
}

function continuationMove(
  continuation: PuzzleDefinition,
  overrides: Partial<Omit<ContinueResponse, 'kind' | 'continuation'>> = {}
): ContinueResponse {
  const base = completedMove(overrides)
  return {
    kind: 'continue',
    uci: base.uci,
    appliedMoves: base.appliedMoves,
    continuation
  }
}

function twoPuzzleScenario(): BoardScenario {
  return {
    kind: 'board',
    first: basicPuzzle(),
    wrongMoves: ['e2e3'],
    correct: completedMove()
  }
}

async function startScenario(page: Page, scenario: BoardScenario): Promise<void> {
  await installTestBackend(page, scenario)
  await page.goto('/')
  await page.getByRole('button', { name: "Start today's training" }).click()
  await expect(page.getByRole('grid')).toBeVisible()
}

function skipTouchProject(testInfo: TestInfo): void {
  test.skip(
    testInfo.project.name === 'webkit-touch',
    'desktop pointer behavior is covered by the desktop browser projects'
  )
}

test('uses trusted click-to-move and retains the solved board until Next puzzle', async ({ page }, testInfo) => {
  skipTouchProject(testInfo)
  await startScenario(page, twoPuzzleScenario())
  const board = chessgroundBoard(page)

  await mouseClickMove(page, board, 'e2', 'e4', 'white')

  await expect(page.getByText('Correct!')).toBeVisible()
  await expect(page.getByText('Puzzle 1 of 2', { exact: true })).toBeVisible()
  await expect.poll(() => pieceKeys(board, 'piece.white.pawn')).toContain('e4')
  expect(await backendState<string[]>(page, 'moves')).toEqual(['e2e4'])
  expect(await backendState<Array<{ trusted: boolean }>>(page, 'trustedInputs'))
    .toEqual(expect.arrayContaining([expect.objectContaining({ trusted: true })]))

  await page.getByRole('button', { name: 'Next puzzle' }).click()
  await expect(page.getByText('Puzzle 2 of 2', { exact: true })).toBeVisible()
  await expect.poll(() => pieceKeys(chessgroundBoard(page), 'piece.white.pawn')).toContain('d2')
})

test('uses trusted drag-to-move', async ({ page }, testInfo) => {
  skipTouchProject(testInfo)
  await startScenario(page, {
    kind: 'board',
    first: basicPuzzle(),
    correct: summaryMove()
  })
  const board = chessgroundBoard(page)

  await mouseDragMove(page, board, 'e2', 'e4', 'white')

  await expect(page.getByText('Correct!')).toBeVisible()
  expect(await backendState<string[]>(page, 'moves')).toEqual(['e2e4'])
  expect(await backendState<Array<{ type: string; trusted: boolean }>>(page, 'trustedInputs'))
    .toContainEqual({ type: 'mousedown', trusted: true })
})

test('cancels an outside drag and snaps a legal-but-wrong drag back', async ({ page }, testInfo) => {
  skipTouchProject(testInfo)
  await startScenario(page, twoPuzzleScenario())
  const board = chessgroundBoard(page)

  await mouseCancelDrag(page, board, 'e2', 'white')
  expect(await backendState<string[]>(page, 'moves')).toEqual([])
  await expect.poll(() => pieceKeys(board, 'piece.white.pawn')).toContain('e2')

  await mouseDragMove(page, board, 'e2', 'e3', 'white')
  await expect(page.getByText('Try again')).toBeVisible()
  await expect.poll(() => pieceKeys(board, 'piece.white.pawn')).toContain('e2')
  await expect(markerForSquare(board, 'e2', 'wrong-source')).toHaveCount(1)
  await expect(markerForSquare(board, 'e3', 'wrong-target')).toHaveCount(1)
})

test('maps black orientation coordinates and accepts the black move', async ({ page }, testInfo) => {
  skipTouchProject(testInfo)
  const blackFen = '4k3/4p3/8/8/8/8/8/4K3 b - - 0 1'
  const afterE5 = '4k3/8/8/4p3/8/8/8/4K3 w - - 0 2'
  await startScenario(page, {
    kind: 'board',
    first: basicPuzzle({
      fingerprint: 'black-puzzle',
      fen: blackFen,
      solver: 'black',
      legalMoves: ['e7e5']
    }),
    correct: summaryMove({
      uci: 'e7e5',
      appliedMoves: [{ uci: 'e7e5', resultingFen: afterE5 }],
      finalFen: afterE5
    })
  })
  const board = chessgroundBoard(page)
  const bounds = await board.boundingBox()
  const source = await squarePoint(board, 'e7', 'black')
  expect(bounds).not.toBeNull()
  expect(source.x).toBeCloseTo(bounds!.x + bounds!.width * 3.5 / 8, 3)
  expect(source.y).toBeCloseTo(bounds!.y + bounds!.height * 6.5 / 8, 3)

  await mouseClickMove(page, board, 'e7', 'e5', 'black')
  await expect(page.getByText('Correct!')).toBeVisible()
  expect(await backendState<string[]>(page, 'moves')).toEqual(['e7e5'])
})

test('shows selected, quiet, and capture markers and clears an immobile friendly piece', async ({ page }, testInfo) => {
  skipTouchProject(testInfo)
  await startScenario(page, {
    kind: 'board',
    first: basicPuzzle({
      fingerprint: 'marker-puzzle',
      fen: 'k7/8/3p4/8/3R4/8/P7/7K w - - 0 1',
      legalMoves: ['d4d5', 'd4d6']
    })
  })
  const board = chessgroundBoard(page)
  const source = await squarePoint(board, 'd4', 'white')
  await page.mouse.click(source.x, source.y)

  const selected = markerForSquare(board, 'd4', 'selected')
  const quiet = markerForSquare(board, 'd5', 'move-dest')
  const capture = markerForSquare(board, 'd6', 'move-dest')
  await expect(selected).toHaveCount(1)
  expect(await selected.evaluate((element) => getComputedStyle(element).backgroundColor))
    .toMatch(/rgba?\(246, 246, 84/)
  await expect(quiet).not.toHaveClass(/\boc\b/)
  await expect(capture).toHaveClass(/\boc\b/)
  const quietBackground = await quiet.evaluate((element) => getComputedStyle(element).backgroundImage)
  const captureBackground = await capture.evaluate((element) => getComputedStyle(element).backgroundImage)
  expect(quietBackground).toContain('radial-gradient')
  expect(captureBackground).toContain('radial-gradient')
  expect(captureBackground).not.toBe(quietBackground)

  const immobile = await squarePoint(board, 'a2', 'white')
  await page.mouse.click(immobile.x, immobile.y)
  await expect(selected).toHaveCount(0)
  await expect(markerForSquare(board, 'a2', 'selected')).toHaveCount(0)
  await expect(page.getByRole('gridcell', { name: 'White pawn on a2' }))
    .toHaveAttribute('aria-selected', 'false')
})

test('opens the promotion chooser and submits the complete UCI suffix', async ({ page }, testInfo) => {
  skipTouchProject(testInfo)
  const promotionFen = 'k7/4P3/8/8/8/8/8/7K w - - 0 1'
  const knightFen = 'k3N3/8/8/8/8/8/8/7K b - - 0 1'
  await startScenario(page, {
    kind: 'board',
    first: basicPuzzle({
      fingerprint: 'promotion-puzzle',
      fen: promotionFen,
      legalMoves: ['e7e8q', 'e7e8n']
    }),
    correct: summaryMove({
      uci: 'e7e8n',
      appliedMoves: [{ uci: 'e7e8n', resultingFen: knightFen }],
      finalFen: knightFen
    })
  })
  const board = chessgroundBoard(page)

  await mouseClickMove(page, board, 'e7', 'e8', 'white')
  await expect(page.getByRole('dialog', { name: 'Choose a promotion' })).toBeVisible()
  await page.getByRole('button', { name: 'Promote to knight' }).click()

  await expect(page.getByText('Correct!')).toBeVisible()
  expect(await backendState<string[]>(page, 'moves')).toEqual(['e7e8n'])
  await expect.poll(() => pieceKeys(board, 'piece.white.knight')).toContain('e8')
})

test('supports focus, arrows, Enter, Space, Escape, and live announcements', async ({ page }, testInfo) => {
  skipTouchProject(testInfo)
  const afterD3 = '4k3/8/8/8/8/3P4/8/4K3 b - - 0 1'
  await startScenario(page, {
    kind: 'board',
    first: basicPuzzle({
      fingerprint: 'keyboard-puzzle',
      fen: nextFen,
      legalMoves: ['d2d3']
    }),
    correct: summaryMove({
      uci: 'd2d3',
      appliedMoves: [{ uci: 'd2d3', resultingFen: afterD3 }],
      finalFen: afterD3
    })
  })
  const grid = page.getByRole('grid')
  const board = chessgroundBoard(page)
  await grid.focus()
  await expect(grid).toBeFocused()
  expect(await grid.evaluate((element) => getComputedStyle(element).outlineWidth)).not.toBe('0px')

  await page.keyboard.press('Enter')
  await expect(markerForSquare(board, 'd2', 'selected')).toHaveCount(1)
  await expect(page.getByRole('gridcell', { name: 'White pawn on d2' }))
    .toHaveAttribute('aria-selected', 'true')
  await expect(page.getByText('Selected White pawn on d2.', { exact: true })).toHaveCount(1)

  await page.keyboard.press('Escape')
  await expect(markerForSquare(board, 'd2', 'selected')).toHaveCount(0)
  await expect(page.getByText('Board selection cleared.', { exact: true })).toHaveCount(1)

  await page.keyboard.press('Enter')
  await page.keyboard.press('ArrowUp')
  await page.keyboard.press('Space')
  await expect(page.getByText('Correct!')).toBeVisible()
  expect(await backendState<string[]>(page, 'moves')).toEqual(['d2d3'])
})

test('applies an accepted automatic reply in authoritative order', async ({ page }, testInfo) => {
  skipTouchProject(testInfo)
  await startScenario(page, {
    kind: 'board',
    first: basicPuzzle({ fen: replyStartingFen }),
    correct: continuationMove(basicPuzzle({
      fen: afterReplyCapture,
      currentPath: [0, 0],
      legalMoves: ['e1d1']
    }), {
      appliedMoves: [
        { uci: 'e2e4', resultingFen: afterReplyUserMove },
        { uci: 'd5e4', resultingFen: afterReplyCapture }
      ]
    })
  })
  await observeSemanticFrames(page)

  await mouseClickMove(page, chessgroundBoard(page), 'e2', 'e4', 'white')
  await expect(page.getByText('Good move. Find the next move.')).toBeVisible()

  const frames = await backendState<string[][]>(page, 'semanticFrames')
  const userFrame = frames.findIndex((frame) =>
    frame.includes('White pawn on e4') && frame.includes('Black pawn on d5'))
  const replyFrame = frames.findIndex((frame) =>
    frame.includes('Black pawn on e4') && !frame.includes('White pawn on e4'))
  expect(userFrame).toBeGreaterThan(-1)
  expect(replyFrame).toBeGreaterThan(userFrame)
})

test('reveals in order and requires See results before the summary', async ({ page }, testInfo) => {
  skipTouchProject(testInfo)
  await startScenario(page, {
    kind: 'board',
    first: basicPuzzle({ fen: replyStartingFen, canReveal: true }),
    reveal: summaryMove({
      appliedMoves: [
        { uci: 'e2e4', resultingFen: afterReplyUserMove },
        { uci: 'd5e4', resultingFen: afterReplyCapture }
      ],
      finalFen: afterReplyCapture
    })
  })
  await observeSemanticFrames(page)

  await page.getByRole('button', { name: 'Show solution' }).click()
  await expect(page.getByText('Solution shown')).toBeVisible()
  await expect(page.getByText('Puzzle 1 of 1', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'See results' })).toBeVisible()
  await expect(page.getByText('Training complete!')).toHaveCount(0)

  const frames = await backendState<string[][]>(page, 'semanticFrames')
  const userFrame = frames.findIndex((frame) => frame.includes('White pawn on e4'))
  const replyFrame = frames.findIndex((frame) => frame.includes('Black pawn on e4'))
  expect(userFrame).toBeGreaterThan(-1)
  expect(replyFrame).toBeGreaterThan(userFrame)

  await page.getByRole('button', { name: 'See results' }).click()
  await expect(page.getByText('Training complete!')).toBeVisible()

  const completion = page.locator('.completion')
  const heading = page.getByRole('heading', { name: 'Training complete!' })
  const bodyCopy = page.getByText('You finished 1 puzzles.')
  const summaryCards = page.locator('.summary-grid strong')

  expect(await contrastRatio(heading, completion)).toBeGreaterThanOrEqual(4.5)
  expect(await contrastRatio(bodyCopy, completion)).toBeGreaterThanOrEqual(4.5)
  await expect(summaryCards).toHaveCount(4)
  for (const card of await summaryCards.all()) {
    expect(await contrastRatio(card, card)).toBeGreaterThanOrEqual(4.5)
  }
})

test('persists mute across a reload', async ({ page }, testInfo) => {
  skipTouchProject(testInfo)
  await startScenario(page, twoPuzzleScenario())
  await page.getByRole('button', { name: 'Mute sounds' }).click()
  await expect(page.getByRole('button', { name: 'Turn sound on' }))
    .toHaveAttribute('aria-pressed', 'true')

  await page.reload()
  await page.getByRole('button', { name: "Start today's training" }).click()
  await expect(page.getByRole('button', { name: 'Turn sound on' }))
    .toHaveAttribute('aria-pressed', 'true')
})

test('reduced motion skips prelude and reply frames but keeps the final FEN', async ({ page }, testInfo) => {
  skipTouchProject(testInfo)
  await page.emulateMedia({ reducedMotion: 'reduce' })
  const source = '4k3/3p4/8/8/8/8/4P3/4K3 b - - 0 1'
  await startScenario(page, {
    kind: 'board',
    first: basicPuzzle({
      fen: replyStartingFen,
      sourceFen: source,
      preludeUci: 'd7d5'
    }),
    correct: continuationMove(basicPuzzle({
      fen: afterReplyCapture,
      currentPath: [0, 0],
      legalMoves: ['e1d1']
    }), {
      appliedMoves: [
        { uci: 'e2e4', resultingFen: afterReplyUserMove },
        { uci: 'd5e4', resultingFen: afterReplyCapture }
      ]
    })
  })
  await expect(page.getByText('Watch the last move…')).toHaveCount(0)
  await observeSemanticFrames(page)

  await mouseClickMove(page, chessgroundBoard(page), 'e2', 'e4', 'white')
  await expect(page.getByText('Good move. Find the next move.')).toBeVisible()
  await expect(page.getByRole('gridcell', { name: 'Black pawn on e4' })).toHaveCount(1)

  const frames = await backendState<string[][]>(page, 'semanticFrames')
  expect(frames.some((frame) => frame.includes('White pawn on e4'))).toBe(false)
  expect(frames.at(-1)).toContain('Black pawn on e4')
})

test('keeps desktop geometry side-by-side and narrow geometry stacked without clipping', async ({ page }, testInfo) => {
  skipTouchProject(testInfo)
  await page.setViewportSize({ width: 1280, height: 900 })
  await startScenario(page, twoPuzzleScenario())

  const boardWrap = page.locator('.board-wrap')
  const panel = page.locator('.puzzle-panel')
  const desktopBoard = await boardWrap.boundingBox()
  const desktopPanel = await panel.boundingBox()
  expect(desktopBoard).not.toBeNull()
  expect(desktopPanel).not.toBeNull()
  expect(desktopBoard!.x + desktopBoard!.width).toBeLessThanOrEqual(desktopPanel!.x)
  expect(desktopPanel!.x + desktopPanel!.width).toBeLessThanOrEqual(1280)

  await page.setViewportSize({ width: 390, height: 844 })
  const narrowBoard = await boardWrap.boundingBox()
  const narrowPanel = await panel.boundingBox()
  expect(narrowBoard).not.toBeNull()
  expect(narrowPanel).not.toBeNull()
  expect(narrowPanel!.y).toBeGreaterThanOrEqual(narrowBoard!.y + narrowBoard!.height)
  await expect.poll(() => page.evaluate(() =>
    document.documentElement.scrollWidth - document.documentElement.clientWidth))
    .toBeLessThanOrEqual(0)
  const files = await page.locator('.chessground-host coords.files').boundingBox()
  expect(files).not.toBeNull()
  expect(files!.x).toBeGreaterThan(narrowBoard!.x)
  expect(files!.x + files!.width).toBeLessThanOrEqual(narrowBoard!.x + narrowBoard!.width)
  for (const button of await panel.getByRole('button').all()) {
    const bounds = await button.boundingBox()
    expect(bounds).not.toBeNull()
    expect(bounds!.x).toBeGreaterThanOrEqual(0)
    expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(390)
  }
})

test('uses trusted touchscreen taps in the WebKit touch project', async ({ page }, testInfo) => {
  test.skip(
    testInfo.project.name !== 'webkit-touch',
    'trusted touch input is covered by the touch-enabled WebKit project'
  )
  await startScenario(page, {
    kind: 'board',
    first: basicPuzzle(),
    correct: summaryMove()
  })

  await touchTapMove(page, chessgroundBoard(page), 'e2', 'e4', 'white')

  await expect(page.getByText('Correct!')).toBeVisible()
  expect(await backendState<string[]>(page, 'moves')).toEqual(['e2e4'])
  expect(await backendState<Array<{ type: string; trusted: boolean }>>(page, 'trustedInputs'))
    .toContainEqual({ type: 'touchstart', trusted: true })
})
