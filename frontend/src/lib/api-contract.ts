export type Profile = {
  learnerRating: number
  sessionSize: 5 | 10 | 15
}

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

export type OpeningHomeView = {
  notice?: string
  courses: OpeningCourseSummary[]
}

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

export type PracticeSource = {
  id: string
  kind: string
  minimumRating: number
  maximumRating: number
  hasRatingRange: boolean
  maximumPlies: number
}

export type RatingBounds = { minimum: number; maximum: number }

export type PracticeFilters = {
  sources: PracticeSource[]
  themes: string[]
  maximumSolutionPlies: number
  learnerRatingBounds: RatingBounds
}

export type RatingPoint = { rating: number; recordedAt: number }
export type ThemePerformance = { theme: string; attempts: number; accuracy: number }
export type RecentSession = {
  sessionId: string
  mode: SessionMode
  status: 'active' | 'paused' | 'completed'
  updatedAt: number
  total: number
  completed: number
  firstTry: number
  usedHint: number
  revealed: number
}
export type ParentSummary = {
  learnerRating: number
  ratingTrend: RatingPoint[]
  firstAttemptAccuracy: number
  hintRate: number
  themePerformance: ThemePerformance[]
  dueReviews: number
  recentSessions: RecentSession[]
}

export type RecoveryState = { required: boolean; path?: string; detail?: string }
export type ApplicationMode = 'normal' | 'recovery'
export type BuildInfo = { name: string; commit: string; sourceUrl: string }
const importFormats = [
  'lichess',
  'tactical-pgn',
  'canonical-json',
  'lucas-fns',
  'linear-fen-uci',
  'coursepack'
] as const
export type ImportFormat = typeof importFormats[number]
export type ImportKind = 'puzzle' | 'course'
export type ImportSourceIDOrigin = 'fixed' | 'embedded' | 'path'
export type ImportInspection = {
  path: string
  filename: string
  format: ImportFormat
  formatLabel: string
  sourceId: string
  sourceIdOrigin: ImportSourceIDOrigin
  sourceName?: string
  url?: string
  attribution?: string
  replacesExisting: boolean
}
export type ImportPhase = 'detecting' | 'parsing' | 'sealing' | 'activating'
export type ImportProgress = {
  jobId: string
  phase: ImportPhase
  rowsRead: number
  bytesRead: number
  totalBytes: number
}
type ImportProgressSnapshot = Omit<ImportProgress, 'jobId'>
export type ImportRejection = { ordinal: number; reason: string }
export type ImportReport = {
  accepted: number
  duplicates: number
  rejected: number
  examples: ImportRejection[]
  counts: Record<string, number>
}
export type ImportResult = {
  jobId: string
  status: 'running' | 'succeeded' | 'failed' | 'cancelled'
  progress: ImportProgressSnapshot
  report: ImportReport
  error?: string
}

type UnknownRecord = Record<string, unknown>

function record(value: unknown, path: string): UnknownRecord {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    throw new Error(`${path} must be an object`)
  }
  return value as UnknownRecord
}

function numberRecord(value: unknown, path: string): Record<string, number> {
  const raw = record(value, path)
  const decoded: Record<string, number> = {}
  for (const [key, entry] of Object.entries(raw)) {
    if (typeof entry !== 'number' || !Number.isFinite(entry) ||
      !Number.isInteger(entry) || entry < 0) {
      throw new Error(`${path}.${key} must be a non-negative integer`)
    }
    decoded[key] = entry
  }
  return decoded
}

function string(value: unknown, path: string): string {
  if (typeof value !== 'string') throw new Error(`${path} must be a string`)
  return value
}

function optionalString(value: unknown, path: string): string | undefined {
  return value === undefined ? undefined : string(value, path)
}

function number(value: unknown, path: string): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    throw new Error(`${path} must be a finite number`)
  }
  return value
}

function nonNegativeInteger(value: unknown, path: string): number {
  const decoded = number(value, path)
  if (!Number.isInteger(decoded) || decoded < 0) {
    throw new Error(`${path} must be a non-negative integer`)
  }
  return decoded
}

function positiveInteger(value: unknown, path: string): number {
  const decoded = number(value, path)
  if (!Number.isInteger(decoded) || decoded <= 0) {
    throw new Error(`${path} must be a positive integer`)
  }
  return decoded
}

function boolean(value: unknown, path: string): boolean {
  if (typeof value !== 'boolean') throw new Error(`${path} must be a boolean`)
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

function decodeFrames(value: unknown, path: string): AppliedMoveFrames {
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

  const appliedMoves = decodeFrames(raw.appliedMoves, `${path}.appliedMoves`)
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
    kind: enumeration(
      raw.kind,
      ['explain', 'watch', 'try', 'branch', 'recall'],
      `${path}.kind`
    ),
    title: string(raw.title, `${path}.title`),
    instruction: string(raw.instruction, `${path}.instruction`),
    variationName: optionalString(raw.variationName, `${path}.variationName`),
    positionId: string(raw.positionId, `${path}.positionId`),
    currentFen: string(raw.currentFen, `${path}.currentFen`),
    orientation: enumeration(raw.orientation, ['white', 'black'], `${path}.orientation`),
    legalMoves: array(raw.legalMoves, `${path}.legalMoves`, string),
    noteTexts: array(raw.noteTexts, `${path}.noteTexts`, string),
    stepNumber,
    stepTotal,
    hintLevel: nonNegativeInteger(raw.hintLevel, `${path}.hintLevel`),
    canReveal: boolean(raw.canReveal, `${path}.canReveal`)
  }
}

export function decodeOpeningSession(
  value: unknown,
  path = 'opening session'
): OpeningSessionView {
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
  const status = enumeration(
    raw.status,
    ['active', 'completed', 'restart_required'],
    `${path}.status`
  )
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

function requireActiveOpening(
  session: OpeningSessionView,
  path: string
): ActiveOpeningSessionView {
  if (session.status !== 'active') throw new Error(`${path} must contain an active session`)
  return session
}

export function decodeOpeningStepResult(
  value: unknown,
  path = 'opening step result'
): OpeningStepResult {
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
    return {
      session: requireActiveOpening(session, path),
      stepCompleted: false,
      feedback,
      message
    }
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

export function decodeOpeningHintResult(
  value: unknown,
  path = 'opening hint result'
): OpeningHintResult {
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

export function decodeApplicationMode(value: unknown): ApplicationMode {
  return enumeration(value, ['normal', 'recovery'], 'application mode')
}

export function decodeBuildInfo(value: unknown): BuildInfo {
  const raw = record(value, 'build info')
  return {
    name: string(raw.name, 'build info.name'),
    commit: string(raw.commit, 'build info.commit'),
    sourceUrl: string(raw.sourceUrl, 'build info.sourceUrl')
  }
}

export function decodeProfile(value: unknown): Profile | null {
  if (value === null || value === undefined) return null
  const raw = record(value, 'profile')
  const sessionSize = number(raw.sessionSize, 'profile.sessionSize')
  if (sessionSize !== 5 && sessionSize !== 10 && sessionSize !== 15) {
    throw new Error(`profile.sessionSize has unsupported value ${sessionSize}`)
  }
  return { learnerRating: number(raw.learnerRating, 'profile.learnerRating'), sessionSize }
}

export function decodePracticeFilters(value: unknown): PracticeFilters {
  const raw = record(value, 'practice filters')
  return {
    sources: array(raw.sources, 'practice filters.sources', (entry, path) => {
      const source = record(entry, path)
      return {
        id: string(source.id, `${path}.id`),
        kind: string(source.kind, `${path}.kind`),
        minimumRating: number(source.minimumRating, `${path}.minimumRating`),
        maximumRating: number(source.maximumRating, `${path}.maximumRating`),
        hasRatingRange: boolean(source.hasRatingRange, `${path}.hasRatingRange`),
        maximumPlies: number(source.maximumPlies, `${path}.maximumPlies`)
      }
    }),
    themes: array(raw.themes, 'practice filters.themes', string),
    maximumSolutionPlies: number(raw.maximumSolutionPlies, 'practice filters.maximumSolutionPlies'),
    learnerRatingBounds: (() => {
      const bounds = record(raw.learnerRatingBounds, 'practice filters.learnerRatingBounds')
      return {
        minimum: number(bounds.minimum, 'practice filters.learnerRatingBounds.minimum'),
        maximum: number(bounds.maximum, 'practice filters.learnerRatingBounds.maximum')
      }
    })()
  }
}

export function decodeParentSummary(value: unknown): ParentSummary {
  const raw = record(value, 'parent summary')
  return {
    learnerRating: number(raw.learnerRating, 'parent summary.learnerRating'),
    ratingTrend: array(raw.ratingTrend, 'parent summary.ratingTrend', (entry, path) => {
      const point = record(entry, path)
      return { rating: number(point.rating, `${path}.rating`), recordedAt: number(point.recordedAt, `${path}.recordedAt`) }
    }),
    firstAttemptAccuracy: number(raw.firstAttemptAccuracy, 'parent summary.firstAttemptAccuracy'),
    hintRate: number(raw.hintRate, 'parent summary.hintRate'),
    themePerformance: array(raw.themePerformance, 'parent summary.themePerformance', (entry, path) => {
      const performance = record(entry, path)
      return {
        theme: string(performance.theme, `${path}.theme`),
        attempts: number(performance.attempts, `${path}.attempts`),
        accuracy: number(performance.accuracy, `${path}.accuracy`)
      }
    }),
    dueReviews: number(raw.dueReviews, 'parent summary.dueReviews'),
    recentSessions: array(raw.recentSessions, 'parent summary.recentSessions', (entry, path) => {
      const session = record(entry, path)
      return {
        sessionId: string(session.sessionId, `${path}.sessionId`),
        mode: enumeration(session.mode, ['guided', 'practice'], `${path}.mode`),
        status: enumeration(session.status, ['active', 'paused', 'completed'], `${path}.status`),
        updatedAt: number(session.updatedAt, `${path}.updatedAt`),
        total: number(session.total, `${path}.total`),
        completed: number(session.completed, `${path}.completed`),
        firstTry: number(session.firstTry, `${path}.firstTry`),
        usedHint: number(session.usedHint, `${path}.usedHint`),
        revealed: number(session.revealed, `${path}.revealed`)
      }
    })
  }
}

export function decodeRecoveryState(value: unknown): RecoveryState {
  const raw = record(value, 'recovery state')
  return {
    required: boolean(raw.required, 'recovery state.required'),
    path: optionalString(raw.path, 'recovery state.path'),
    detail: optionalString(raw.detail, 'recovery state.detail')
  }
}

export function decodeImportInspection(value: unknown): ImportInspection {
  const raw = record(value, 'import inspection')
  return {
    path: string(raw.path, 'import inspection.path'),
    filename: string(raw.filename, 'import inspection.filename'),
    format: enumeration(
      raw.format,
      importFormats,
      'import inspection.format'
    ),
    formatLabel: string(raw.formatLabel, 'import inspection.formatLabel'),
    sourceId: string(raw.sourceId, 'import inspection.sourceId'),
    sourceIdOrigin: enumeration(
      raw.sourceIdOrigin,
      ['fixed', 'embedded', 'path'],
      'import inspection.sourceIdOrigin'
    ),
    sourceName: optionalString(raw.sourceName, 'import inspection.sourceName'),
    url: optionalString(raw.url, 'import inspection.url'),
    attribution: optionalString(raw.attribution, 'import inspection.attribution'),
    replacesExisting: boolean(raw.replacesExisting, 'import inspection.replacesExisting')
  }
}

function decodeImportPhase(value: unknown, path: string): ImportPhase {
  return enumeration(value, ['detecting', 'parsing', 'sealing', 'activating'], path)
}

function decodeImportProgressSnapshot(value: unknown, path: string): ImportProgressSnapshot {
  const raw = record(value, path)
  return {
    phase: decodeImportPhase(raw.phase, `${path}.phase`),
    rowsRead: number(raw.rowsRead, `${path}.rowsRead`),
    bytesRead: number(raw.bytesRead, `${path}.bytesRead`),
    totalBytes: number(raw.totalBytes, `${path}.totalBytes`)
  }
}

function decodeImportReport(value: unknown, path: string): ImportReport {
  const raw = record(value, path)
  const examples = raw.examples === null
    ? []
    : array(raw.examples, `${path}.examples`, (entry, entryPath) => {
        const example = record(entry, entryPath)
        return {
          ordinal: number(example.ordinal, `${entryPath}.ordinal`),
          reason: string(example.reason, `${entryPath}.reason`)
        }
      })
  return {
    accepted: number(raw.accepted, `${path}.accepted`),
    duplicates: number(raw.duplicates, `${path}.duplicates`),
    rejected: number(raw.rejected, `${path}.rejected`),
    examples,
    counts: raw.counts === undefined ? {} : numberRecord(raw.counts, `${path}.counts`)
  }
}

export function decodeImportProgress(value: unknown): ImportProgress {
  const raw = record(value, 'import progress')
  return {
    jobId: string(raw.jobId, 'import progress.jobId'),
    ...decodeImportProgressSnapshot(raw, 'import progress')
  }
}

export function decodeImportResult(value: unknown): ImportResult {
  const raw = record(value, 'import result')
  return {
    jobId: string(raw.jobId, 'import result.jobId'),
    status: enumeration(raw.status, ['running', 'succeeded', 'failed', 'cancelled'], 'import result.status'),
    progress: decodeImportProgressSnapshot(raw.progress, 'import result.progress'),
    report: decodeImportReport(raw.report, 'import result.report'),
    error: optionalString(raw.error, 'import result.error')
  }
}
