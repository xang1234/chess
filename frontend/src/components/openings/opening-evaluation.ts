import type { OpeningEvaluation, OpeningPerspective } from '../../lib/api'

export type FormattedOpeningEvaluation = {
  symbol: string
  label: string
  text: string
}

export function formatOpeningEvaluation(
  evaluation: OpeningEvaluation,
  perspective: OpeningPerspective
): FormattedOpeningEvaluation | null {
  let symbol: string
  let label: string

  switch (evaluation.code) {
    case 'none':
      return null
    case 'equal':
      symbol = '='
      label = 'Equal chances'
      break
    case 'unclear':
      symbol = '∞'
      label = 'Unclear position'
      break
    case 'white_slight':
      symbol = '⩲'
      label = perspective === 'white' ? 'Slightly better for you' : 'Slightly better for White'
      break
    case 'black_slight':
      symbol = '⩱'
      label = perspective === 'black' ? 'Slightly better for you' : 'Slightly better for Black'
      break
    case 'white_clear':
      symbol = '±'
      label = perspective === 'white' ? 'Clearly better for you' : 'Clearly better for White'
      break
    case 'black_clear':
      symbol = '∓'
      label = perspective === 'black' ? 'Clearly better for you' : 'Clearly better for Black'
      break
    case 'white_winning':
      symbol = '+-'
      label = perspective === 'white' ? 'Winning for you' : 'Winning for White'
      break
    case 'black_winning':
      symbol = '-+'
      label = perspective === 'black' ? 'Winning for you' : 'Winning for Black'
      break
    default: {
      const exhaustive: never = evaluation.code
      return exhaustive
    }
  }

  const sourceSymbol = evaluation.sourceSymbol?.trim()
  if (sourceSymbol) symbol = sourceSymbol

  return { symbol, label, text: `${symbol} ${label}` }
}
