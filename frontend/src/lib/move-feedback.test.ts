import { classifyMoveFeedback } from './move-feedback'

test('classifies ordinary and en-passant captures by occupancy change', () => {
  expect(classifyMoveFeedback(
    '4k3/8/8/3p4/4P3/8/8/4K3 w - - 0 1',
    '4k3/8/8/3P4/8/8/8/4K3 b - - 0 1',
    'e4d5'
  )).toBe('capture')

  expect(classifyMoveFeedback(
    '4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1',
    '4k3/8/3P4/8/8/8/8/4K3 b - - 0 1',
    'e5d6'
  )).toBe('capture')
})

test('classifies quiet moves, castling, and non-capturing promotion as moves', () => {
  expect(classifyMoveFeedback(
    '4k3/8/8/8/8/8/4P3/4K3 w - - 0 1',
    '4k3/8/8/8/4P3/8/8/4K3 b - - 0 1',
    'e2e4'
  )).toBe('move')

  expect(classifyMoveFeedback(
    '4k2r/8/8/8/8/8/8/4K3 b k - 0 1',
    '5rk1/8/8/8/8/8/8/4K3 w - - 1 2',
    'e8g8'
  )).toBe('move')

  expect(classifyMoveFeedback(
    '7k/P7/8/8/8/8/8/K7 w - - 0 1',
    'Q6k/8/8/8/8/8/8/K7 b - - 0 1',
    'a7a8q'
  )).toBe('move')
})

test('rejects malformed authoritative FEN input', () => {
  expect(() => classifyMoveFeedback('not a FEN', 'also not a FEN', 'e2e4'))
    .toThrow(/classify move e2e4/)
})
