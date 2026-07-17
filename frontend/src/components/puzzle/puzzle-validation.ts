import type {
  AppliedMove,
  MoveResult,
  PuzzleView,
  SessionView
} from '../../lib/api'
import { parseFEN } from '../../lib/fen'
import { groupLegalMoves, parseUCI } from '../../lib/uci'
import type { PuzzleState } from './puzzle-state'

type SuccessfulMoveResult = Extract<MoveResult, { correct: true }>
type RequestingState = Extract<PuzzleState, { phase: 'requesting' }>

export function validatePuzzle(puzzle: PuzzleView): void {
  try {
    groupLegalMoves(puzzle.legalMoves)
  } catch (error) {
    throw new Error(`Invalid legal move data: ${errorMessage(error)}. Puzzle input is locked.`)
  }
  parseFEN(puzzle.currentFen)
  parseFEN(puzzle.displayedFen)
  if (puzzle.sourceFen) parseFEN(puzzle.sourceFen)
  if (puzzle.preludeUci) parseUCI(puzzle.preludeUci)
}

export function validateSuccessfulResult(
  result: SuccessfulMoveResult,
  request: RequestingState,
  optimisticUci?: string
): AppliedMove[] {
  if (!Array.isArray(result.appliedMoves) || result.appliedMoves.length === 0) {
    throw new Error('Successful puzzle response is missing authoritative move frames.')
  }
  for (const frame of result.appliedMoves) {
    parseUCI(frame.uci)
    if (!frame.resultingFen) throw new Error(`Move ${frame.uci} has no authoritative FEN.`)
    parseFEN(frame.resultingFen)
  }
  if (optimisticUci && result.appliedMoves[0].uci !== optimisticUci) {
    throw new Error('Authoritative move frames do not begin with the submitted move.')
  }
  if (result.puzzleCompleted) {
    parseFEN(result.finalFen)
    if (result.session.sessionId !== request.displaySession.sessionId ||
      result.session.currentIndex <= request.displaySession.currentIndex) {
      throw new Error(
        'Completed puzzle response did not advance within the same session to a higher puzzle index.'
      )
    }
    validatePendingSession(result.session)
  } else {
    if (responseAdvanced(request.displaySession, result.session)) {
      throw new Error('Incomplete correct response advanced to a different puzzle.')
    }
    validatePuzzle(result.session.current)
    parseFEN(result.session.current.currentFen)
  }
  return [...result.appliedMoves]
}

export function validatePendingSession(next: SessionView): void {
  if (next.status === 'active') {
    validatePuzzle(next.current)
    return
  }
  if (!next.summary) throw new Error('Completed response has neither a next puzzle nor results.')
}

export function responseAdvanced(before: SessionView, after: SessionView): boolean {
  if (before.sessionId !== after.sessionId || before.currentIndex !== after.currentIndex) return true
  if (before.status !== 'active' || after.status !== 'active') return true
  return before.current.fingerprint !== after.current.fingerprint
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
