export type OpeningTrainingRole = 'repertoire' | 'opponent' | 'alternative'
export type OpeningEvaluationCode =
  | 'none'
  | 'equal'
  | 'unclear'
  | 'white_slight'
  | 'black_slight'
  | 'white_clear'
  | 'black_clear'
  | 'white_winning'
  | 'black_winning'

export type OpeningEvaluation = {
  code: OpeningEvaluationCode
  sourceSymbol?: string
}

export type OpeningSourceRef = {
  printedPage: number
  tableColumn?: string
  noteLabel?: string
  coverageId: string
}

export type OpeningExplorerMove = {
  moveId: string
  uci: string
  san: string
  toPositionId: string
  role: OpeningTrainingRole
  variationName?: string
  evaluation: OpeningEvaluation
  sourceRef: OpeningSourceRef
}

export type OpeningNoteView = {
  kind: string
  text: string
  sourceRef: OpeningSourceRef
}

export type OpeningPositionView = {
  courseId: string
  positionId: string
  fen: string
  label: string
  evaluation: OpeningEvaluation
  notes: OpeningNoteView[]
  moves: OpeningExplorerMove[]
  incomingPaths: number
}

type UnknownRecord = Record<string, unknown>

const evaluationCodes = [
  'none',
  'equal',
  'unclear',
  'white_slight',
  'black_slight',
  'white_clear',
  'black_clear',
  'white_winning',
  'black_winning'
] as const

function record(value: unknown, path: string): UnknownRecord {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    throw new Error(`${path} must be an object`)
  }
  return value as UnknownRecord
}

function string(value: unknown, path: string): string {
  if (typeof value !== 'string') throw new Error(`${path} must be a string`)
  return value
}

function optionalString(value: unknown, path: string): string | undefined {
  return value === undefined ? undefined : string(value, path)
}

function positiveInteger(value: unknown, path: string): number {
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value <= 0) {
    throw new Error(`${path} must be a positive integer`)
  }
  return value
}

function nonNegativeInteger(value: unknown, path: string): number {
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < 0) {
    throw new Error(`${path} must be a non-negative integer`)
  }
  return value
}

function array<Value>(
  value: unknown,
  path: string,
  decode: (entry: unknown, entryPath: string) => Value
): Value[] {
  if (!Array.isArray(value)) throw new Error(`${path} must be an array`)
  return value.map((entry, index) => decode(entry, `${path}[${index}]`))
}

function enumeration<Value extends string>(
  value: unknown,
  allowed: readonly Value[],
  path: string
): Value {
  if (typeof value !== 'string' || !allowed.includes(value as Value)) {
    throw new Error(`${path} has unknown value ${JSON.stringify(value)}`)
  }
  return value as Value
}

function decodeEvaluation(value: unknown, path: string): OpeningEvaluation {
  const raw = record(value, path)
  return {
    code: enumeration(raw.code, evaluationCodes, `${path}.code`),
    sourceSymbol: optionalString(raw.sourceSymbol, `${path}.sourceSymbol`)
  }
}

function decodeSourceRef(value: unknown, path: string): OpeningSourceRef {
  const raw = record(value, path)
  return {
    printedPage: positiveInteger(raw.printedPage, `${path}.printedPage`),
    tableColumn: optionalString(raw.tableColumn, `${path}.tableColumn`),
    noteLabel: optionalString(raw.noteLabel, `${path}.noteLabel`),
    coverageId: string(raw.coverageId, `${path}.coverageId`)
  }
}

export function decodeOpeningPosition(
  value: unknown,
  path = 'opening position'
): OpeningPositionView {
  const raw = record(value, path)
  return {
    courseId: string(raw.courseId, `${path}.courseId`),
    positionId: string(raw.positionId, `${path}.positionId`),
    fen: string(raw.fen, `${path}.fen`),
    label: string(raw.label, `${path}.label`),
    evaluation: decodeEvaluation(raw.evaluation, `${path}.evaluation`),
    notes: array(raw.notes, `${path}.notes`, (noteValue, notePath) => {
      const note = record(noteValue, notePath)
      return {
        kind: string(note.kind, `${notePath}.kind`),
        text: string(note.text, `${notePath}.text`),
        sourceRef: decodeSourceRef(note.sourceRef, `${notePath}.sourceRef`)
      }
    }),
    moves: array(raw.moves, `${path}.moves`, (moveValue, movePath) => {
      const move = record(moveValue, movePath)
      return {
        moveId: string(move.moveId, `${movePath}.moveId`),
        uci: string(move.uci, `${movePath}.uci`),
        san: string(move.san, `${movePath}.san`),
        toPositionId: string(move.toPositionId, `${movePath}.toPositionId`),
        role: enumeration(
          move.role,
          ['repertoire', 'opponent', 'alternative'],
          `${movePath}.role`
        ),
        variationName: optionalString(move.variationName, `${movePath}.variationName`),
        evaluation: decodeEvaluation(move.evaluation, `${movePath}.evaluation`),
        sourceRef: decodeSourceRef(move.sourceRef, `${movePath}.sourceRef`)
      }
    }),
    incomingPaths: nonNegativeInteger(raw.incomingPaths, `${path}.incomingPaths`)
  }
}
