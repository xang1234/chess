import type { Locator, Page } from '@playwright/test'

export type BoardOrientation = 'white' | 'black'

export function chessgroundBoard(page: Page): Locator {
  return page.locator('.chessground-host cg-board')
}

export async function squarePoint(
  board: Locator,
  square: string,
  orientation: BoardOrientation
): Promise<{ x: number; y: number }> {
  if (!/^[a-h][1-8]$/.test(square)) {
    throw new Error(`invalid chess square: ${square}`)
  }
  const bounds = await board.boundingBox()
  if (!bounds || bounds.width <= 0 || bounds.height <= 0) {
    throw new Error('chess board has no visible bounds')
  }

  const file = square.charCodeAt(0) - 'a'.charCodeAt(0)
  const rank = Number(square[1]) - 1
  const column = orientation === 'white' ? file : 7 - file
  const row = orientation === 'white' ? 7 - rank : rank

  return {
    x: bounds.x + ((column + 0.5) * bounds.width) / 8,
    y: bounds.y + ((row + 0.5) * bounds.height) / 8
  }
}

export async function mouseClickMove(
  page: Page,
  board: Locator,
  from: string,
  to: string,
  orientation: BoardOrientation
): Promise<void> {
  const source = await squarePoint(board, from, orientation)
  const destination = await squarePoint(board, to, orientation)
  await page.mouse.click(source.x, source.y)
  await page.mouse.click(destination.x, destination.y)
}

export async function mouseDragMove(
  page: Page,
  board: Locator,
  from: string,
  to: string,
  orientation: BoardOrientation
): Promise<void> {
  const source = await squarePoint(board, from, orientation)
  const destination = await squarePoint(board, to, orientation)
  await page.mouse.move(source.x, source.y)
  await page.mouse.down()
  await page.mouse.move(destination.x, destination.y, { steps: 8 })
  await page.mouse.up()
}

export async function mouseCancelDrag(
  page: Page,
  board: Locator,
  from: string,
  orientation: BoardOrientation
): Promise<void> {
  const source = await squarePoint(board, from, orientation)
  const bounds = await board.boundingBox()
  if (!bounds) throw new Error('chess board has no visible bounds')
  const viewport = page.viewportSize()
  const right = bounds.x + bounds.width + 16
  const outsideX = !viewport || right < viewport.width
    ? right
    : Math.max(1, bounds.x - 16)
  const outsideY = bounds.y + bounds.height / 2

  await page.mouse.move(source.x, source.y)
  await page.mouse.down()
  await page.mouse.move(outsideX, outsideY, { steps: 8 })
  await page.mouse.up()
}

export async function touchTapMove(
  page: Page,
  board: Locator,
  from: string,
  to: string,
  orientation: BoardOrientation
): Promise<void> {
  const source = await squarePoint(board, from, orientation)
  const destination = await squarePoint(board, to, orientation)
  await page.touchscreen.tap(source.x, source.y)
  await page.touchscreen.tap(destination.x, destination.y)
}

export function markerForSquare(
  board: Locator,
  square: string,
  marker: string
): Locator {
  if (!/^[a-h][1-8]$/.test(square)) {
    throw new Error(`invalid chess square: ${square}`)
  }
  if (!/^[a-z][a-z0-9-]*$/.test(marker)) {
    throw new Error(`invalid marker class: ${marker}`)
  }
  return board.locator(`square.${marker}[data-key="${square}"]:visible`)
}

export async function pieceKeys(board: Locator, selector = 'piece'): Promise<string[]> {
  return board.locator(selector).evaluateAll((pieces) => pieces
    .filter((piece) => {
      const bounds = piece.getBoundingClientRect()
      return getComputedStyle(piece).display !== 'none' && bounds.width > 0 && bounds.height > 0
    })
    .map((piece) => (piece as Element & { cgKey?: string }).cgKey)
    .filter((key): key is string => Boolean(key)))
}
