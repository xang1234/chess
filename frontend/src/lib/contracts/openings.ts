import {
  array,
  boolean,
  enumeration,
  nonNegativeInteger,
  optionalString,
  positiveInteger,
  record,
  string
} from './decoder'
import type { AppliedMoveFrames } from './puzzles'

export type OpeningDepth = 'quick' | 'standard' | 'reference'
export type OpeningPerspective = 'white' | 'black'
export type OpeningStepKind = 'explain' | 'watch' | 'try' | 'branch' | 'recall'
export type OpeningMoveFeedback = 'expected' | 'alternative' | 'off_course'
export type OpeningSessionMode = 'lesson' | 'review'
export type OpeningSessionStatus = 'active' | 'completed' | 'restart_required'

export type OpeningLessonSummary = {
  lessonId: string
  title: string
  completedSteps: number
  totalSteps: number
  completed: boolean
}

export type OpeningChapterSummary = {
  chapterId: string
  title: string
  lessons: OpeningLessonSummary[]
}

export type OpeningCourseSummary = {
  courseId: string
  title: string
  perspective: OpeningPerspective
  depth: OpeningDepth
  rootPositionId: string
  completedLessons: number
  totalLessons: number
  dueReviews: number
  nextLessonId?: string
  nextLessonTitle?: string
  hasResumable: boolean
  chapters: OpeningChapterSummary[]
}

export type OpeningHomeView = { notice?: string; courses: OpeningCourseSummary[] }

export type OpeningStepView = {
  stepId: string
  kind: OpeningStepKind
  title: string
  instruction: string
  variationName?: string
  positionId: string
  currentFen: string
  orientation: OpeningPerspective
  legalMoves: string[]
  noteTexts: string[]
  referenceNoteTexts: string[]
  stepNumber: number
  stepTotal: number
  hintLevel: number
  canReveal: boolean
}

export type OpeningSummary = {
  totalPrompts: number
  positionsRecalled: number
  branchesRecognized: number
  retried: number
  usedHint: number
  revealed: number
}

type OpeningSessionBase = {
  sessionId: string
  mode: OpeningSessionMode
  courseId: string
  generationId: string
  lessonId: string
  depth: OpeningDepth
}

export type ActiveOpeningSessionView = OpeningSessionBase & {
  status: 'active'
  current: OpeningStepView
  summary?: never
  notice?: string
}

export type CompletedOpeningSessionView = OpeningSessionBase & {
  status: 'completed'
  current?: never
  summary: OpeningSummary
  notice?: string
}

export type RestartRequiredOpeningSessionView = OpeningSessionBase & {
  status: 'restart_required'
  current?: never
  summary?: never
  notice: string
}

export type OpeningSessionView =
  | ActiveOpeningSessionView
  | CompletedOpeningSessionView
  | RestartRequiredOpeningSessionView

export type ExpectedOpeningStepResult = {
  session: OpeningSessionView
  stepCompleted: true
  feedback: 'expected'
  message?: string
  appliedMoves: AppliedMoveFrames
  finalFen: string
}

export type OpeningFeedbackResult = {
  session: ActiveOpeningSessionView
  stepCompleted: false
  feedback: 'alternative' | 'off_course'
  message?: string
  appliedMoves?: never
  finalFen?: never
}

export type OpeningAdvanceResult = {
  session: ActiveOpeningSessionView
  stepCompleted: true
  feedback?: never
  message?: string
  appliedMoves?: AppliedMoveFrames
  finalFen?: string
}

export type OpeningStepResult =
  | ExpectedOpeningStepResult
  | OpeningFeedbackResult
  | OpeningAdvanceResult

export type OpeningHintResult = {
  session: ActiveOpeningSessionView
  level: number
  text: string
  sourceSquare?: string
  targetSquare?: string
  canReveal: boolean
}

export type OpeningTrainingRole = 'repertoire' | 'opponent' | 'alternative'
export type OpeningEvaluationCode =
  | 'none' | 'equal' | 'unclear' | 'white_slight' | 'black_slight'
  | 'white_clear' | 'black_clear' | 'white_winning' | 'black_winning'

export type OpeningEvaluation = { code: OpeningEvaluationCode; sourceSymbol?: string }
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
export type OpeningNoteView = { kind: string; text: string; sourceRef: OpeningSourceRef }
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

function decodeOpeningSummary(value: unknown, path: string): OpeningSummary {
  const raw = record(value, path)
  return {
    totalPrompts: nonNegativeInteger(raw.totalPrompts, `${path}.totalPrompts`),
    positionsRecalled: nonNegativeInteger(raw.positionsRecalled, `${path}.positionsRecalled`),
    branchesRecognized: nonNegativeInteger(raw.branchesRecognized, `${path}.branchesRecognized`),
    retried: nonNegativeInteger(raw.retried, `${path}.retried`),
    usedHint: nonNegativeInteger(raw.usedHint, `${path}.usedHint`),
    revealed: nonNegativeInteger(raw.revealed, `${path}.revealed`)
  }
}

function decodeOpeningStep(value: unknown, path: string): OpeningStepView {
  const raw = record(value, path)
  const stepNumber = positiveInteger(raw.stepNumber, `${path}.stepNumber`)
  const stepTotal = positiveInteger(raw.stepTotal, `${path}.stepTotal`)
  if (stepNumber > stepTotal) throw new Error(`${path}.stepNumber must not exceed stepTotal`)
  return {
    stepId: string(raw.stepId, `${path}.stepId`),
    kind: enumeration(raw.kind, ['explain', 'watch', 'try', 'branch', 'recall'], `${path}.kind`),
    title: string(raw.title, `${path}.title`),
    instruction: string(raw.instruction, `${path}.instruction`),
    variationName: optionalString(raw.variationName, `${path}.variationName`),
    positionId: string(raw.positionId, `${path}.positionId`),
    currentFen: string(raw.currentFen, `${path}.currentFen`),
    orientation: enumeration(raw.orientation, ['white', 'black'], `${path}.orientation`),
    legalMoves: array(raw.legalMoves, `${path}.legalMoves`, string),
    noteTexts: array(raw.noteTexts, `${path}.noteTexts`, string),
    referenceNoteTexts: array(raw.referenceNoteTexts, `${path}.referenceNoteTexts`, string),
    stepNumber,
    stepTotal,
    hintLevel: nonNegativeInteger(raw.hintLevel, `${path}.hintLevel`),
    canReveal: boolean(raw.canReveal, `${path}.canReveal`)
  }
}

export function decodeOpeningSession(value: unknown, path = 'opening session'): OpeningSessionView {
  const raw = record(value, path)
  const base = {
    sessionId: string(raw.sessionId, `${path}.sessionId`),
    mode: enumeration(raw.mode, ['lesson', 'review'], `${path}.mode`),
    courseId: string(raw.courseId, `${path}.courseId`),
    generationId: string(raw.generationId, `${path}.generationId`),
    lessonId: string(raw.lessonId, `${path}.lessonId`),
    depth: enumeration(raw.depth, ['quick', 'standard', 'reference'], `${path}.depth`)
  }
  const notice = optionalString(raw.notice, `${path}.notice`)
  const status = enumeration(raw.status, ['active', 'completed', 'restart_required'], `${path}.status`)
  if (status === 'active') {
    if (raw.current === undefined) throw new Error(`${path} active session has no current step`)
    if (raw.summary !== undefined) throw new Error(`${path} active session must not include a summary`)
    return { ...base, status, current: decodeOpeningStep(raw.current, `${path}.current`), notice }
  }
  if (status === 'completed') {
    if (raw.summary === undefined) throw new Error(`${path} completed session has no summary`)
    if (raw.current !== undefined) throw new Error(`${path} completed session must not include a current step`)
    return { ...base, status, summary: decodeOpeningSummary(raw.summary, `${path}.summary`), notice }
  }
  if (raw.current !== undefined) throw new Error(`${path} restart-required session must not include a current step`)
  if (raw.summary !== undefined) throw new Error(`${path} restart-required session must not include a summary`)
  if (notice === undefined) throw new Error(`${path} restart-required session has no notice`)
  return { ...base, status, notice }
}

function requireActiveOpening(session: OpeningSessionView, path: string): ActiveOpeningSessionView {
  if (session.status !== 'active') throw new Error(`${path} must contain an active session`)
  return session
}

function decodeFrames(value: unknown, path: string): AppliedMoveFrames {
  if (!Array.isArray(value)) throw new Error(`${path} must contain authoritative move frames`)
  const frames = array(value, path, (entry, entryPath) => {
    const move = record(entry, entryPath)
    return {
      uci: string(move.uci, `${entryPath}.uci`),
      resultingFen: string(move.resultingFen, `${entryPath}.resultingFen`)
    }
  })
  if (frames.length === 0) throw new Error(`${path} must contain authoritative move frames`)
  return frames as AppliedMoveFrames
}

export function decodeOpeningStepResult(value: unknown, path = 'opening step result'): OpeningStepResult {
  const raw = record(value, path)
  const session = decodeOpeningSession(raw.session, `${path}.session`)
  const stepCompleted = boolean(raw.stepCompleted, `${path}.stepCompleted`)
  const message = optionalString(raw.message, `${path}.message`)
  const feedback = raw.feedback === undefined
    ? undefined
    : enumeration(raw.feedback, ['expected', 'alternative', 'off_course'], `${path}.feedback`)

  if (feedback === 'alternative' || feedback === 'off_course') {
    if (stepCompleted) throw new Error(`${path} ${feedback} feedback cannot complete a step`)
    if (raw.appliedMoves !== undefined || raw.finalFen !== undefined) {
      throw new Error(`${path} ${feedback} feedback must not include move frames or a final FEN`)
    }
    return { session: requireActiveOpening(session, path), stepCompleted: false, feedback, message }
  }
  if (feedback === 'expected') {
    if (!stepCompleted) throw new Error(`${path} expected feedback must complete a step`)
    return {
      session,
      stepCompleted: true,
      feedback,
      message,
      appliedMoves: decodeFrames(raw.appliedMoves, `${path}.appliedMoves`),
      finalFen: string(raw.finalFen, `${path} expected result final FEN`)
    }
  }
  if (!stepCompleted) throw new Error(`${path} incomplete result requires move feedback`)
  const active = requireActiveOpening(session, path)
  if (raw.appliedMoves === undefined) {
    if (raw.finalFen !== undefined) throw new Error(`${path} passive result must not include only a final FEN`)
    return { session: active, stepCompleted: true, message }
  }
  return {
    session: active,
    stepCompleted: true,
    message,
    appliedMoves: decodeFrames(raw.appliedMoves, `${path}.appliedMoves`),
    finalFen: string(raw.finalFen, `${path} animated result final FEN`)
  }
}

export function decodeOpeningHintResult(value: unknown, path = 'opening hint result'): OpeningHintResult {
  const raw = record(value, path)
  return {
    session: requireActiveOpening(decodeOpeningSession(raw.session, `${path}.session`), path),
    level: positiveInteger(raw.level, `${path}.level`),
    text: string(raw.text, `${path}.text`),
    sourceSquare: optionalString(raw.sourceSquare, `${path}.sourceSquare`),
    targetSquare: optionalString(raw.targetSquare, `${path}.targetSquare`),
    canReveal: boolean(raw.canReveal, `${path}.canReveal`)
  }
}

export function decodeOpeningHome(value: unknown, path = 'opening home'): OpeningHomeView {
  const raw = record(value, path)
  return {
    notice: optionalString(raw.notice, `${path}.notice`),
    courses: array(raw.courses, `${path}.courses`, (courseValue, coursePath) => {
      const course = record(courseValue, coursePath)
      return {
        courseId: string(course.courseId, `${coursePath}.courseId`),
        title: string(course.title, `${coursePath}.title`),
        perspective: enumeration(course.perspective, ['white', 'black'], `${coursePath}.perspective`),
        depth: enumeration(course.depth, ['quick', 'standard', 'reference'], `${coursePath}.depth`),
        rootPositionId: string(course.rootPositionId, `${coursePath}.rootPositionId`),
        completedLessons: nonNegativeInteger(course.completedLessons, `${coursePath}.completedLessons`),
        totalLessons: nonNegativeInteger(course.totalLessons, `${coursePath}.totalLessons`),
        dueReviews: nonNegativeInteger(course.dueReviews, `${coursePath}.dueReviews`),
        nextLessonId: optionalString(course.nextLessonId, `${coursePath}.nextLessonId`),
        nextLessonTitle: optionalString(course.nextLessonTitle, `${coursePath}.nextLessonTitle`),
        hasResumable: boolean(course.hasResumable, `${coursePath}.hasResumable`),
        chapters: array(course.chapters, `${coursePath}.chapters`, (chapterValue, chapterPath) => {
          const chapter = record(chapterValue, chapterPath)
          return {
            chapterId: string(chapter.chapterId, `${chapterPath}.chapterId`),
            title: string(chapter.title, `${chapterPath}.title`),
            lessons: array(chapter.lessons, `${chapterPath}.lessons`, (lessonValue, lessonPath) => {
              const lesson = record(lessonValue, lessonPath)
              return {
                lessonId: string(lesson.lessonId, `${lessonPath}.lessonId`),
                title: string(lesson.title, `${lessonPath}.title`),
                completedSteps: nonNegativeInteger(lesson.completedSteps, `${lessonPath}.completedSteps`),
                totalSteps: positiveInteger(lesson.totalSteps, `${lessonPath}.totalSteps`),
                completed: boolean(lesson.completed, `${lessonPath}.completed`)
              }
            })
          }
        })
      }
    })
  }
}

const evaluationCodes = [
  'none', 'equal', 'unclear', 'white_slight', 'black_slight',
  'white_clear', 'black_clear', 'white_winning', 'black_winning'
] as const

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

export function decodeOpeningPosition(value: unknown, path = 'opening position'): OpeningPositionView {
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
        role: enumeration(move.role, ['repertoire', 'opponent', 'alternative'], `${movePath}.role`),
        variationName: optionalString(move.variationName, `${movePath}.variationName`),
        evaluation: decodeEvaluation(move.evaluation, `${movePath}.evaluation`),
        sourceRef: decodeSourceRef(move.sourceRef, `${movePath}.sourceRef`)
      }
    }),
    incomingPaths: nonNegativeInteger(raw.incomingPaths, `${path}.incomingPaths`)
  }
}
