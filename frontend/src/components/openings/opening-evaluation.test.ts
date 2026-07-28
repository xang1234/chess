import { describe, expect, it } from 'vitest'
import type { OpeningEvaluationCode } from '../../lib/api'
import { formatOpeningEvaluation } from './opening-evaluation'

const rows: Array<{
  code: OpeningEvaluationCode
  symbol: string
  white: string
  black: string
}> = [
  { code: 'equal', symbol: '=', white: 'Equal chances', black: 'Equal chances' },
  { code: 'unclear', symbol: '∞', white: 'Unclear position', black: 'Unclear position' },
  { code: 'white_slight', symbol: '⩲', white: 'Slightly better for you', black: 'Slightly better for White' },
  { code: 'black_slight', symbol: '⩱', white: 'Slightly better for Black', black: 'Slightly better for you' },
  { code: 'white_clear', symbol: '±', white: 'Clearly better for you', black: 'Clearly better for White' },
  { code: 'black_clear', symbol: '∓', white: 'Clearly better for Black', black: 'Clearly better for you' },
  { code: 'white_winning', symbol: '+-', white: 'Winning for you', black: 'Winning for White' },
  { code: 'black_winning', symbol: '-+', white: 'Winning for Black', black: 'Winning for you' }
]

describe('formatOpeningEvaluation', () => {
  it.each(rows)('formats $code from both course perspectives', ({ code, symbol, white, black }) => {
    expect(formatOpeningEvaluation({ code }, 'white')).toEqual({
      symbol,
      label: white,
      text: `${symbol} ${white}`
    })
    expect(formatOpeningEvaluation({ code }, 'black')).toEqual({
      symbol,
      label: black,
      text: `${symbol} ${black}`
    })
  })

  it('omits an evaluation that has no authored meaning', () => {
    expect(formatOpeningEvaluation({ code: 'none' }, 'white')).toBeNull()
    expect(formatOpeningEvaluation({ code: 'none' }, 'black')).toBeNull()
  })

  it('uses a non-empty source symbol without changing the normalized label', () => {
    expect(formatOpeningEvaluation({ code: 'white_slight', sourceSymbol: '+=' }, 'white'))
      .toEqual({ symbol: '+=', label: 'Slightly better for you', text: '+= Slightly better for you' })
    expect(formatOpeningEvaluation({ code: 'equal', sourceSymbol: '   ' }, 'white'))
      .toEqual({ symbol: '=', label: 'Equal chances', text: '= Equal chances' })
  })
})
