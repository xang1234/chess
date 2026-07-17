import {
  groupLegalMoves,
  moveKeyboardCursor,
  moveSquares,
  parseUCI,
  promotionChoices
} from './uci'

describe('parseUCI', () => {
  it('parses ordinary and promotion moves', () => {
    expect(parseUCI('e2e4')).toEqual({ from: 'e2', to: 'e4' })
    expect(parseUCI('a7a8q')).toEqual({ from: 'a7', to: 'a8', promotion: 'q' })
    expect(moveSquares('h2h1n')).toEqual(['h2', 'h1'])
  })

  it.each([
    'i2e4',
    'e0e4',
    'E2E4',
    'e2e4 trailing',
    'e2e4x',
    'e2',
    'a7a8k'
  ])('rejects malformed UCI %s with the offending value', (value) => {
    expect(() => parseUCI(value)).toThrow(value)
  })

  it('accepts a structurally valid four-character promotion route', () => {
    expect(parseUCI('a7a8')).toEqual({ from: 'a7', to: 'a8' })
  })
})

describe('keyboard cursor movement', () => {
  it('moves by screen direction for white and clamps at board edges', () => {
    expect(moveKeyboardCursor('e4', 'ArrowUp', 'white')).toBe('e5')
    expect(moveKeyboardCursor('e4', 'ArrowDown', 'white')).toBe('e3')
    expect(moveKeyboardCursor('e4', 'ArrowLeft', 'white')).toBe('d4')
    expect(moveKeyboardCursor('e4', 'ArrowRight', 'white')).toBe('f4')
    expect(moveKeyboardCursor('a8', 'ArrowUp', 'white')).toBe('a8')
    expect(moveKeyboardCursor('a8', 'ArrowLeft', 'white')).toBe('a8')
  })

  it('reverses screen directions for black orientation', () => {
    expect(moveKeyboardCursor('e4', 'ArrowUp', 'black')).toBe('e3')
    expect(moveKeyboardCursor('e4', 'ArrowDown', 'black')).toBe('e5')
    expect(moveKeyboardCursor('e4', 'ArrowLeft', 'black')).toBe('f4')
    expect(moveKeyboardCursor('e4', 'ArrowRight', 'black')).toBe('d4')
    expect(moveKeyboardCursor('h1', 'ArrowUp', 'black')).toBe('h1')
    expect(moveKeyboardCursor('h1', 'ArrowLeft', 'black')).toBe('h1')
  })
})

describe('legal-move grouping', () => {
  const legal = ['e2e4', 'e2e3', 'a7a8q', 'a7a8r', 'e2e4']

  it('sorts sources and deduplicates visual destinations', () => {
    const grouped = groupLegalMoves(legal)

    expect([...grouped.entries()]).toEqual([
      ['a7', ['a8']],
      ['e2', ['e3', 'e4']]
    ])
  })

  it('retains every deterministic promotion choice for a route', () => {
    expect(promotionChoices(legal, 'a7', 'a8')).toEqual(['q', 'r'])
    expect(promotionChoices(['a7a8r', 'a7a8b', 'a7a8n', 'a7a8q'], 'a7', 'a8'))
      .toEqual(['b', 'n', 'q', 'r'])
  })
})
