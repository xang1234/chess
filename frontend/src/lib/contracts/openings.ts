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
export type OpeningActivityKind =
  | 'concept' | 'demonstration' | 'decision' | 'comparison' | 'recap' | 'reference'
export type OpeningLessonEdgeKind = 'continuation' | 'alternative' | 'reference'
export type OpeningNodeProgress = 'available' | 'in_progress' | 'completed'
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

export type OpeningTeachingNode = {
  lessonId: string
  chapterId: string
  title: string
  objective: string
  minimumDepth: OpeningDepth
  progress: OpeningNodeProgress
  completedActivities: number
  requiredActivities: number
  recommended: boolean
  reviewDue: boolean
  visible: boolean
}

export type OpeningTeachingEdge = {
  edgeId: string
  fromLessonId: string
  toLessonId: string
  ordinal: number
  kind: OpeningLessonEdgeKind
  label?: string
  minimumDepth: OpeningDepth
}

export type OpeningTeachingTree = {
  rootLessonId: string
  nodes: OpeningTeachingNode[]
  edges: OpeningTeachingEdge[]
}

export type OpeningPathItem = { lessonId: string; title: string }

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
  currentLessonId?: string
  currentActivityId?: string
  currentPath: OpeningPathItem[]
  recommendedLessonId?: string
  recommendedLessonTitle?: string
  hasResumable: boolean
  tree: OpeningTeachingTree
  chapters: OpeningChapterSummary[]
}

export type OpeningHomeView = { notice?: string; courses: OpeningCourseSummary[] }

export type OpeningActivityLine = { label: string; moveIds: string[] }
export type OpeningBoardAnnotation = {
  kind: 'square' | 'arrow'
  from: string
  to?: string
  label?: string
}

export type OpeningReferenceSection = {
  activityId: string
  title: string
  instruction: string
  positionId?: string
  noteTexts: string[]
  annotations: OpeningBoardAnnotation[]
}

export type OpeningActivityView = {
  activityId: string
  kind: OpeningActivityKind
  title: string
  instruction: string
  required: boolean
  variationName?: string
  positionId?: string
  currentFen: string
  orientation: OpeningPerspective
  legalMoves: string[]
  teachingNoteTexts: string[]
  referenceNoteTexts: string[]
  comparison: OpeningActivityLine[]
  annotations: OpeningBoardAnnotation[]
  movesToHere: AppliedMoveFrames | []
  activityNumber: number
  activityTotal: number
  completedIdeas: number
  requiredIdeas: number
  hintLevel: number
  canReveal: boolean
  referenceSections: OpeningReferenceSection[]
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
  current: OpeningActivityView
  summary?: never
  notice?: string
}

export type CompletedOpeningLessonView = OpeningSessionBase & {
  mode: 'lesson'
  status: 'completed'
  current?: never
  summary?: never
  notice?: string
}

export type CompletedOpeningReviewView = OpeningSessionBase & {
  mode: 'review'
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
  | CompletedOpeningLessonView
  | CompletedOpeningReviewView
  | RestartRequiredOpeningSessionView

export type CompletedOpeningSessionView =
  | CompletedOpeningLessonView
  | CompletedOpeningReviewView

export type OpeningRoadmapCheckpoint = {
  completedLessonId: string
  path: OpeningPathItem[]
  availableLessonIds: string[]
  recommendedLessonId?: string
  recommendedLessonTitle?: string
  completedLessons: number
  totalLessons: number
}

type OpeningActivityResultBase = {
  session: OpeningSessionView
  message?: string
}

export type OpeningActivityResult =
  | OpeningActivityResultBase & {
    session: ActiveOpeningSessionView
    activityCompleted: false
    feedback: 'alternative' | 'off_course'
    appliedMoves?: never
    finalFen?: never
    checkpoint?: never
  }
  | OpeningActivityResultBase & {
    activityCompleted: true
    feedback?: 'expected'
    appliedMoves?: AppliedMoveFrames
    finalFen?: string
    checkpoint?: OpeningRoadmapCheckpoint
  }

export type OpeningHintResult = {
  session: ActiveOpeningSessionView
  level: number
  text: string
  sourceSquare?: string
  targetSquare?: string
  canReveal: boolean
}

// One-release source aliases while callers migrate to activity terminology.
export type OpeningStepView = OpeningActivityView
export type OpeningStepResult = OpeningActivityResult
export type OpeningStepKind = OpeningActivityKind

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

const depths = ['quick', 'standard', 'reference'] as const
const activityKinds = [
  'concept', 'demonstration', 'decision', 'comparison', 'recap', 'reference'
] as const

function decodePathItem(value: unknown, path: string): OpeningPathItem {
  const raw = record(value, path)
  return { lessonId: string(raw.lessonId, `${path}.lessonId`), title: string(raw.title, `${path}.title`) }
}

function decodeAnnotation(value: unknown, path: string): OpeningBoardAnnotation {
  const raw = record(value, path)
  const kind = enumeration(raw.kind, ['square', 'arrow'], `${path}.kind`)
  const to = optionalString(raw.to, `${path}.to`)
  if (kind === 'arrow' && to === undefined) throw new Error(`${path}.to is required for an arrow`)
  if (kind === 'square' && to !== undefined) throw new Error(`${path}.to is not allowed for a square`)
  return {
    kind,
    from: string(raw.from, `${path}.from`),
    to,
    label: optionalString(raw.label, `${path}.label`)
  }
}

function decodeFramesAllowEmpty(value: unknown, path: string): AppliedMoveFrames | [] {
  return array(value, path, (entry, entryPath) => {
    const move = record(entry, entryPath)
    return {
      uci: string(move.uci, `${entryPath}.uci`),
      resultingFen: string(move.resultingFen, `${entryPath}.resultingFen`)
    }
  }) as AppliedMoveFrames | []
}

function decodeFrames(value: unknown, path: string): AppliedMoveFrames {
  const frames = decodeFramesAllowEmpty(value, path)
  if (frames.length === 0) throw new Error(`${path} must contain authoritative move frames`)
  return frames as AppliedMoveFrames
}

function decodeReferenceSection(value: unknown, path: string): OpeningReferenceSection {
  const raw = record(value, path)
  return {
    activityId: string(raw.activityId, `${path}.activityId`),
    title: string(raw.title, `${path}.title`),
    instruction: string(raw.instruction, `${path}.instruction`),
    positionId: optionalString(raw.positionId, `${path}.positionId`),
    noteTexts: array(raw.noteTexts, `${path}.noteTexts`, string),
    annotations: array(raw.annotations, `${path}.annotations`, decodeAnnotation)
  }
}

function decodeOpeningActivity(value: unknown, path: string): OpeningActivityView {
  const raw = record(value, path)
  const activityNumber = positiveInteger(raw.activityNumber, `${path}.activityNumber`)
  const activityTotal = positiveInteger(raw.activityTotal, `${path}.activityTotal`)
  if (activityNumber > activityTotal) {
    throw new Error(`${path}.activityNumber must not exceed activityTotal`)
  }
  const completedIdeas = nonNegativeInteger(raw.completedIdeas, `${path}.completedIdeas`)
  const requiredIdeas = positiveInteger(raw.requiredIdeas, `${path}.requiredIdeas`)
  if (completedIdeas > requiredIdeas) {
    throw new Error(`${path}.completedIdeas must not exceed requiredIdeas`)
  }
  return {
    activityId: string(raw.activityId, `${path}.activityId`),
    kind: enumeration(raw.kind, activityKinds, `${path}.kind`),
    title: string(raw.title, `${path}.title`),
    instruction: string(raw.instruction, `${path}.instruction`),
    required: boolean(raw.required, `${path}.required`),
    variationName: optionalString(raw.variationName, `${path}.variationName`),
    positionId: optionalString(raw.positionId, `${path}.positionId`),
    currentFen: string(raw.currentFen, `${path}.currentFen`),
    orientation: enumeration(raw.orientation, ['white', 'black'], `${path}.orientation`),
    legalMoves: array(raw.legalMoves, `${path}.legalMoves`, string),
    teachingNoteTexts: array(raw.teachingNoteTexts, `${path}.teachingNoteTexts`, string),
    referenceNoteTexts: array(raw.referenceNoteTexts, `${path}.referenceNoteTexts`, string),
    comparison: array(raw.comparison, `${path}.comparison`, (entry, entryPath) => {
      const line = record(entry, entryPath)
      return {
        label: string(line.label, `${entryPath}.label`),
        moveIds: array(line.moveIds, `${entryPath}.moveIds`, string)
      }
    }),
    annotations: array(raw.annotations, `${path}.annotations`, decodeAnnotation),
    movesToHere: decodeFramesAllowEmpty(raw.movesToHere, `${path}.movesToHere`),
    activityNumber,
    activityTotal,
    completedIdeas,
    requiredIdeas,
    hintLevel: nonNegativeInteger(raw.hintLevel, `${path}.hintLevel`),
    canReveal: boolean(raw.canReveal, `${path}.canReveal`),
    referenceSections: array(
      raw.referenceSections, `${path}.referenceSections`, decodeReferenceSection
    )
  }
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

export function decodeOpeningSession(value: unknown, path = 'opening session'): OpeningSessionView {
  const raw = record(value, path)
  const base = {
    sessionId: string(raw.sessionId, `${path}.sessionId`),
    mode: enumeration(raw.mode, ['lesson', 'review'], `${path}.mode`),
    courseId: string(raw.courseId, `${path}.courseId`),
    generationId: string(raw.generationId, `${path}.generationId`),
    lessonId: string(raw.lessonId, `${path}.lessonId`),
    depth: enumeration(raw.depth, depths, `${path}.depth`)
  }
  const notice = optionalString(raw.notice, `${path}.notice`)
  const status = enumeration(raw.status, ['active', 'completed', 'restart_required'], `${path}.status`)
  if (status === 'active') {
    if (raw.current === undefined) throw new Error(`${path} active session has no current activity`)
    if (raw.summary !== undefined) throw new Error(`${path} active session must not include a summary`)
    return { ...base, status, current: decodeOpeningActivity(raw.current, `${path}.current`), notice }
  }
  if (status === 'completed') {
    if (raw.current !== undefined) throw new Error(`${path} completed session must not include a current activity`)
    if (base.mode === 'review') {
      if (raw.summary === undefined) throw new Error(`${path} completed review has no summary`)
      return { ...base, mode: 'review', status, summary: decodeOpeningSummary(raw.summary, `${path}.summary`), notice }
    }
    if (raw.summary !== undefined) throw new Error(`${path} completed lesson must not include a summary`)
    return { ...base, mode: 'lesson', status, notice }
  }
  if (raw.current !== undefined) throw new Error(`${path} restart-required session must not include a current activity`)
  if (raw.summary !== undefined) throw new Error(`${path} restart-required session must not include a summary`)
  if (notice === undefined) throw new Error(`${path} restart-required session has no notice`)
  return { ...base, status, notice }
}

function requireActiveOpening(session: OpeningSessionView, path: string): ActiveOpeningSessionView {
  if (session.status !== 'active') throw new Error(`${path} must contain an active session`)
  return session
}

function decodeCheckpoint(value: unknown, path: string): OpeningRoadmapCheckpoint {
  const raw = record(value, path)
  const completedLessons = nonNegativeInteger(raw.completedLessons, `${path}.completedLessons`)
  const totalLessons = positiveInteger(raw.totalLessons, `${path}.totalLessons`)
  if (completedLessons > totalLessons) {
    throw new Error(`${path}.completedLessons must not exceed totalLessons`)
  }
  return {
    completedLessonId: string(raw.completedLessonId, `${path}.completedLessonId`),
    path: array(raw.path, `${path}.path`, decodePathItem),
    availableLessonIds: array(raw.availableLessonIds, `${path}.availableLessonIds`, string),
    recommendedLessonId: optionalString(raw.recommendedLessonId, `${path}.recommendedLessonId`),
    recommendedLessonTitle: optionalString(raw.recommendedLessonTitle, `${path}.recommendedLessonTitle`),
    completedLessons,
    totalLessons
  }
}

export function decodeOpeningActivityResult(
  value: unknown,
  path = 'opening activity result'
): OpeningActivityResult {
  const raw = record(value, path)
  const session = decodeOpeningSession(raw.session, `${path}.session`)
  const activityCompleted = boolean(raw.activityCompleted, `${path}.activityCompleted`)
  const message = optionalString(raw.message, `${path}.message`)
  const feedback = raw.feedback === undefined
    ? undefined
    : enumeration(raw.feedback, ['expected', 'alternative', 'off_course'], `${path}.feedback`)
  const checkpoint = raw.checkpoint === undefined
    ? undefined
    : decodeCheckpoint(raw.checkpoint, `${path}.checkpoint`)

  if (feedback === 'alternative' || feedback === 'off_course') {
    if (activityCompleted) throw new Error(`${path} ${feedback} feedback cannot complete an activity`)
    if (raw.appliedMoves !== undefined || raw.finalFen !== undefined || checkpoint !== undefined) {
      throw new Error(`${path} ${feedback} feedback must not mutate lesson progress`)
    }
    return {
      session: requireActiveOpening(session, path), activityCompleted: false, feedback, message
    }
  }
  if (!activityCompleted) throw new Error(`${path} incomplete result requires move feedback`)
  if (feedback === 'expected') {
    return {
      session,
      activityCompleted: true,
      feedback,
      message,
      appliedMoves: decodeFrames(raw.appliedMoves, `${path}.appliedMoves`),
      finalFen: string(raw.finalFen, `${path}.finalFen`),
      checkpoint
    }
  }
  if (feedback !== undefined) throw new Error(`${path}.feedback is invalid`)
  if (raw.appliedMoves === undefined) {
    if (raw.finalFen !== undefined) throw new Error(`${path} passive result must not include only a final FEN`)
    return { session, activityCompleted: true, message, checkpoint }
  }
  return {
    session,
    activityCompleted: true,
    message,
    appliedMoves: decodeFrames(raw.appliedMoves, `${path}.appliedMoves`),
    finalFen: string(raw.finalFen, `${path}.finalFen`),
    checkpoint
  }
}

export const decodeOpeningStepResult = decodeOpeningActivityResult

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

function decodeTree(value: unknown, path: string): OpeningTeachingTree {
  const raw = record(value, path)
  return {
    rootLessonId: string(raw.rootLessonId, `${path}.rootLessonId`),
    nodes: array(raw.nodes, `${path}.nodes`, (nodeValue, nodePath) => {
      const node = record(nodeValue, nodePath)
      const completedActivities = nonNegativeInteger(
        node.completedActivities, `${nodePath}.completedActivities`
      )
      const requiredActivities = positiveInteger(
        node.requiredActivities, `${nodePath}.requiredActivities`
      )
      if (completedActivities > requiredActivities) {
        throw new Error(`${nodePath}.completedActivities must not exceed requiredActivities`)
      }
      return {
        lessonId: string(node.lessonId, `${nodePath}.lessonId`),
        chapterId: string(node.chapterId, `${nodePath}.chapterId`),
        title: string(node.title, `${nodePath}.title`),
        objective: string(node.objective, `${nodePath}.objective`),
        minimumDepth: enumeration(node.minimumDepth, depths, `${nodePath}.minimumDepth`),
        progress: enumeration(
          node.progress, ['available', 'in_progress', 'completed'], `${nodePath}.progress`
        ),
        completedActivities,
        requiredActivities,
        recommended: boolean(node.recommended, `${nodePath}.recommended`),
        reviewDue: boolean(node.reviewDue, `${nodePath}.reviewDue`),
        visible: boolean(node.visible, `${nodePath}.visible`)
      }
    }),
    edges: array(raw.edges, `${path}.edges`, (edgeValue, edgePath) => {
      const edge = record(edgeValue, edgePath)
      return {
        edgeId: string(edge.edgeId, `${edgePath}.edgeId`),
        fromLessonId: string(edge.fromLessonId, `${edgePath}.fromLessonId`),
        toLessonId: string(edge.toLessonId, `${edgePath}.toLessonId`),
        ordinal: positiveInteger(edge.ordinal, `${edgePath}.ordinal`),
        kind: enumeration(
          edge.kind, ['continuation', 'alternative', 'reference'], `${edgePath}.kind`
        ),
        label: optionalString(edge.label, `${edgePath}.label`),
        minimumDepth: enumeration(edge.minimumDepth, depths, `${edgePath}.minimumDepth`)
      }
    })
  }
}

export function decodeOpeningHome(value: unknown, path = 'opening home'): OpeningHomeView {
  const raw = record(value, path)
  return {
    notice: optionalString(raw.notice, `${path}.notice`),
    courses: array(raw.courses, `${path}.courses`, (courseValue, coursePath) => {
      const course = record(courseValue, coursePath)
      const completedLessons = nonNegativeInteger(course.completedLessons, `${coursePath}.completedLessons`)
      const totalLessons = nonNegativeInteger(course.totalLessons, `${coursePath}.totalLessons`)
      if (completedLessons > totalLessons) {
        throw new Error(`${coursePath}.completedLessons must not exceed totalLessons`)
      }
      return {
        courseId: string(course.courseId, `${coursePath}.courseId`),
        title: string(course.title, `${coursePath}.title`),
        perspective: enumeration(course.perspective, ['white', 'black'], `${coursePath}.perspective`),
        depth: enumeration(course.depth, depths, `${coursePath}.depth`),
        rootPositionId: string(course.rootPositionId, `${coursePath}.rootPositionId`),
        completedLessons,
        totalLessons,
        dueReviews: nonNegativeInteger(course.dueReviews, `${coursePath}.dueReviews`),
        nextLessonId: optionalString(course.nextLessonId, `${coursePath}.nextLessonId`),
        nextLessonTitle: optionalString(course.nextLessonTitle, `${coursePath}.nextLessonTitle`),
        currentLessonId: optionalString(course.currentLessonId, `${coursePath}.currentLessonId`),
        currentActivityId: optionalString(course.currentActivityId, `${coursePath}.currentActivityId`),
        currentPath: array(course.currentPath, `${coursePath}.currentPath`, decodePathItem),
        recommendedLessonId: optionalString(
          course.recommendedLessonId, `${coursePath}.recommendedLessonId`
        ),
        recommendedLessonTitle: optionalString(
          course.recommendedLessonTitle, `${coursePath}.recommendedLessonTitle`
        ),
        hasResumable: boolean(course.hasResumable, `${coursePath}.hasResumable`),
        tree: decodeTree(course.tree, `${coursePath}.tree`),
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
