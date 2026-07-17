import { describeSquare, orientSquares, parseFEN } from './fen'

test('parses standard and sparse FEN placement', () => {
  const standard = parseFEN('rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1')
  expect(standard.a1).toEqual({ color: 'white', role: 'rook' })
  expect(standard.e8).toEqual({ color: 'black', role: 'king' })
  expect(standard.e4).toBeUndefined()

  const sparse = parseFEN('7k/8/8/8/8/8/8/K7 w - - 0 1')
  expect(Object.keys(sparse)).toHaveLength(2)
  expect(sparse.a1.role).toBe('king')
  expect(sparse.h8.color).toBe('black')
})

test('rejects malformed ranks', () => {
  expect(() => parseFEN('7/8/8/8/8/8/8/8 w - - 0 1')).toThrow('rank 8')
})

test('orients black from h1 through a8', () => {
  const squares = orientSquares('black')
  expect(squares).toHaveLength(64)
  expect(squares[0]).toBe('h1')
  expect(squares[63]).toBe('a8')
})

test('describes occupied and empty semantic squares', () => {
  const board = parseFEN('4k3/8/8/8/8/8/4P3/4K3 w - - 0 1')
  expect(describeSquare('e2', board.e2)).toBe('White pawn on e2')
  expect(describeSquare('d4', board.d4)).toBe('Empty d4')
})
