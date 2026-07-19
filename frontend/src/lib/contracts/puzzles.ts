import {
  array,
  boolean,
  enumeration,
  number,
  optionalString,
  record,
  string
} from './decoder'

export type PuzzleView = {
  fingerprint: string
  sourceFen?: string
  displayedFen: string
  currentFen: string
  preludeUci?: string
  solver: 'white' | 'black'
  currentPath: number[]
  puzzleNumber: number
  puzzleTotal: number
  hintLevel: number
  incorrectMoves: number
  canReveal: boolean
  legalMoves: string[]
}

export type AppliedMove = { uci: string; resultingFen: string }
export type AppliedMoveFrames = [AppliedMove, ...AppliedMove[]]

export type SessionSummary = {
  total: number
  firstTry: number
  retried: number
  usedHint: number
  revealed: number
  unavailable: number
}

export type SessionMode = 'guided' | 'practice'

type SessionBase = {
  sessionId: string
  mode: SessionMode
  currentIndex: number
  total: number
}

export type ActiveSessionView = SessionBase & {
  status: 'active'
  current: PuzzleView
  summary?: never
}

export type CompletedSessionView = SessionBase & {
  status: 'completed'
  current?: never
  summary: SessionSummary
}

export type SessionView = ActiveSessionView | CompletedSessionView

export type IncorrectMoveResult = {
  session: SessionView
  correct: false
  puzzleCompleted: false
  message?: string
  appliedMoves?: never
  finalFen?: never
}

export type ContinuingMoveResult = {
  session: ActiveSessionView
  correct: true
  puzzleCompleted: false
  message?: string
  appliedMoves: AppliedMoveFrames
  finalFen?: never
}

export type CompletedMoveResult = {
  session: SessionView
  correct: true
  puzzleCompleted: true
  message?: string
  appliedMoves: AppliedMoveFrames
  finalFen: string
}

export type MoveResult = IncorrectMoveResult | ContinuingMoveResult | CompletedMoveResult

export type HintResult = {
  session: SessionView
  level: number
  text: string
  sourceSquare?: string
  targetSquare?: string
  canReveal: boolean
}

function decodePuzzle(value: unknown, path: string): PuzzleView {
  const raw = record(value, path)
  return {
    fingerprint: string(raw.fingerprint, `${path}.fingerprint`),
    sourceFen: optionalString(raw.sourceFen, `${path}.sourceFen`),
    displayedFen: string(raw.displayedFen, `${path}.displayedFen`),
    currentFen: string(raw.currentFen, `${path}.currentFen`),
    preludeUci: optionalString(raw.preludeUci, `${path}.preludeUci`),
    solver: enumeration(raw.solver, ['white', 'black'], `${path}.solver`),
    currentPath: array(raw.currentPath, `${path}.currentPath`, number),
    puzzleNumber: number(raw.puzzleNumber, `${path}.puzzleNumber`),
    puzzleTotal: number(raw.puzzleTotal, `${path}.puzzleTotal`),
    hintLevel: number(raw.hintLevel, `${path}.hintLevel`),
    incorrectMoves: number(raw.incorrectMoves, `${path}.incorrectMoves`),
    canReveal: boolean(raw.canReveal, `${path}.canReveal`),
    legalMoves: array(raw.legalMoves, `${path}.legalMoves`, string)
  }
}

function decodeSummary(value: unknown, path: string): SessionSummary {
  const raw = record(value, path)
  return {
    total: number(raw.total, `${path}.total`),
    firstTry: number(raw.firstTry, `${path}.firstTry`),
    retried: number(raw.retried, `${path}.retried`),
    usedHint: number(raw.usedHint, `${path}.usedHint`),
    revealed: number(raw.revealed, `${path}.revealed`),
    unavailable: number(raw.unavailable, `${path}.unavailable`)
  }
}

export function decodeSession(value: unknown, path = 'session'): SessionView {
  const raw = record(value, path)
  const base = {
    sessionId: string(raw.sessionId, `${path}.sessionId`),
    mode: enumeration(raw.mode, ['guided', 'practice'], `${path}.mode`),
    currentIndex: number(raw.currentIndex, `${path}.currentIndex`),
    total: number(raw.total, `${path}.total`)
  }
  const status = enumeration(raw.status, ['active', 'completed'], `${path}.status`)
  if (status === 'active') {
    if (raw.current === undefined) throw new Error(`${path} active session has no current puzzle`)
    if (raw.summary !== undefined) throw new Error(`${path} active session must not include a summary`)
    return { ...base, status, current: decodePuzzle(raw.current, `${path}.current`) }
  }
  if (raw.summary === undefined) throw new Error(`${path} completed session has no summary`)
  if (raw.current !== undefined) throw new Error(`${path} completed session must not include a current puzzle`)
  return { ...base, status, summary: decodeSummary(raw.summary, `${path}.summary`) }
}

function requireActive(session: SessionView, path: string): ActiveSessionView {
  if (session.status !== 'active') throw new Error(`${path} must contain an active session`)
  return session
}

function decodeAppliedMove(value: unknown, path: string): AppliedMove {
  const raw = record(value, path)
  return {
    uci: string(raw.uci, `${path}.uci`),
    resultingFen: string(raw.resultingFen, `${path}.resultingFen`)
  }
}

export function decodeAppliedMoveFrames(value: unknown, path: string): AppliedMoveFrames {
  if (!Array.isArray(value)) throw new Error(`${path} must contain authoritative move frames`)
  const frames = array(value, path, decodeAppliedMove)
  if (frames.length === 0) throw new Error(`${path} must contain authoritative move frames`)
  return frames as AppliedMoveFrames
}

export function decodeMoveResult(value: unknown, path = 'move result'): MoveResult {
  const raw = record(value, path)
  const session = decodeSession(raw.session, `${path}.session`)
  const correct = boolean(raw.correct, `${path}.correct`)
  const puzzleCompleted = boolean(raw.puzzleCompleted, `${path}.puzzleCompleted`)
  const message = optionalString(raw.message, `${path}.message`)

  if (!correct) {
    if (puzzleCompleted) throw new Error(`${path} incorrect move cannot complete a puzzle`)
    if (raw.appliedMoves !== undefined || raw.finalFen !== undefined) {
      throw new Error(`${path} incorrect move must not include authoritative move frames or a final FEN`)
    }
    return { session, correct: false, puzzleCompleted: false, message }
  }

  const appliedMoves = decodeAppliedMoveFrames(raw.appliedMoves, `${path}.appliedMoves`)
  if (puzzleCompleted) {
    return {
      session,
      correct: true,
      puzzleCompleted: true,
      message,
      appliedMoves,
      finalFen: string(raw.finalFen, `${path} completed move final FEN`)
    }
  }
  if (raw.finalFen !== undefined) throw new Error(`${path} continuing result must not include a final FEN`)
  return {
    session: requireActive(session, path),
    correct: true,
    puzzleCompleted: false,
    message,
    appliedMoves
  }
}

export function decodeHintResult(value: unknown, path = 'hint result'): HintResult {
  const raw = record(value, path)
  return {
    session: decodeSession(raw.session, `${path}.session`),
    level: number(raw.level, `${path}.level`),
    text: string(raw.text, `${path}.text`),
    sourceSquare: optionalString(raw.sourceSquare, `${path}.sourceSquare`),
    targetSquare: optionalString(raw.targetSquare, `${path}.targetSquare`),
    canReveal: boolean(raw.canReveal, `${path}.canReveal`)
  }
}
