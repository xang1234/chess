import {
  groupLegalMoves,
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
