import { parseFEN } from './fen'
import { parseUCI } from './uci'

export type MoveFeedback = 'move' | 'capture'

export function classifyMoveFeedback(
  beforeFen: string,
  afterFen: string,
  uci: string
): MoveFeedback {
  try {
    parseUCI(uci)
    const beforePieces = Object.keys(parseFEN(beforeFen)).length
    const afterPieces = Object.keys(parseFEN(afterFen)).length
    return afterPieces < beforePieces ? 'capture' : 'move'
  } catch (error) {
    const detail = error instanceof Error ? error.message : String(error)
    throw new Error(`classify move ${uci}: ${detail}`, { cause: error })
  }
}
