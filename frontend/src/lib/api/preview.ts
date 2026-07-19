import type { BuildInfo, Profile } from '../contracts/application'
import type {
  ActiveOpeningSessionView,
  OpeningDepth,
  OpeningHomeView,
  OpeningSessionView,
  OpeningStepResult,
  OpeningStepView
} from '../contracts/openings'
import type {
  ActiveSessionView,
  AppliedMoveFrames,
  CompletedSessionView,
  MoveResult,
  PuzzleView,
  SessionView
} from '../contracts/puzzles'
import type { ApplicationAPI, NormalAPI } from './types'

const previewBuildInfo: BuildInfo = {
  name: 'Chess Trainer',
  commit: 'development',
  sourceUrl: 'https://github.com/xang1234/chess'
}

let previewProfile: Profile | null = null
const previewStartingFen = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'
const previewLegalMoves = [
  'a2a3', 'a2a4', 'b1a3', 'b1c3', 'b2b3', 'b2b4',
  'c2c3', 'c2c4', 'd2d3', 'd2d4', 'e2e3', 'e2e4',
  'f2f3', 'f2f4', 'g1f3', 'g1h3', 'g2g3', 'g2g4',
  'h2h3', 'h2h4'
]
const previewPuzzles = [
  {
    fingerprint: 'preview-puzzle-1',
    correctMove: 'e2e4',
    wrongMove: 'e2e3',
    finalFen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1'
  },
  {
    fingerprint: 'preview-puzzle-2',
    correctMove: 'd2d4',
    wrongMove: 'd2d3',
    finalFen: 'rnbqkbnr/pppppppp/8/8/3P4/8/PPP1PPPP/RNBQKBNR b KQkq d3 0 1'
  }
] as const

let previewSession: SessionView | null = null
let previewIncorrect = new Set<number>()

function previewPuzzle(index: number): PuzzleView {
  const puzzle = previewPuzzles[index]
  return {
    fingerprint: puzzle.fingerprint,
    displayedFen: previewStartingFen,
    currentFen: previewStartingFen,
    solver: 'white',
    currentPath: [],
    puzzleNumber: index + 1,
    puzzleTotal: previewPuzzles.length,
    hintLevel: 0,
    incorrectMoves: previewIncorrect.has(index) ? 1 : 0,
    canReveal: false,
    legalMoves: [...previewLegalMoves]
  }
}

function previewActiveSession(
  index: number,
  mode: 'guided' | 'practice' = 'guided'
): ActiveSessionView {
  return {
    sessionId: mode === 'practice' ? 'preview-practice' : 'preview-session',
    mode,
    status: 'active',
    currentIndex: index,
    total: previewPuzzles.length,
    current: previewPuzzle(index)
  }
}

function previewCompletedSession(session: ActiveSessionView): CompletedSessionView {
  return {
    sessionId: session.sessionId,
    mode: session.mode,
    status: 'completed',
    currentIndex: previewPuzzles.length,
    total: previewPuzzles.length,
    summary: {
      total: previewPuzzles.length,
      firstTry: previewPuzzles.length - previewIncorrect.size,
      retried: previewIncorrect.size,
      usedHint: 0,
      revealed: 0,
      unavailable: 0
    }
  }
}

function clonePreviewSession<Value>(value: Value): Value {
  return structuredClone(value)
}

function completePreviewPuzzle(): MoveResult {
  if (!previewSession || previewSession.status !== 'active') {
    throw new Error('preview session is not active')
  }
  const activeSession = previewSession
  const index = activeSession.currentIndex
  const puzzle = previewPuzzles[index]
  const appliedMoves: AppliedMoveFrames = [{
    uci: puzzle.correctMove,
    resultingFen: puzzle.finalFen
  }]
  previewSession = index + 1 < previewPuzzles.length
    ? previewActiveSession(index + 1, activeSession.mode)
    : previewCompletedSession(activeSession)
  return {
    session: clonePreviewSession(previewSession),
    correct: true,
    puzzleCompleted: true,
    appliedMoves,
    finalFen: puzzle.finalFen
  }
}

const previewOpeningFens = {
  initial: previewStartingFen,
  prompt: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4',
  afterC3: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'
}

let previewOpeningDepth: OpeningDepth = 'reference'
let previewOpeningSession: OpeningSessionView | null = null
let previewOpeningStepIndex = 0
let previewOpeningHintLevel = 0

function previewOpeningStep(index: number, mode: 'lesson' | 'review'): OpeningStepView {
  if (mode === 'review') {
    return {
      stepId: 'review-recall-c3', kind: 'recall', title: 'Review the Giuoco Piano',
      instruction: 'Play the course move from memory.', variationName: 'Giuoco Piano',
      positionId: 'after-bc5', currentFen: previewOpeningFens.prompt,
      orientation: 'white', legalMoves: ['b2b4', 'c2c3', 'd2d3'], noteTexts: [],
      stepNumber: 1, stepTotal: 1, hintLevel: previewOpeningHintLevel,
      canReveal: previewOpeningHintLevel >= 4
    }
  }
  const common = {
    orientation: 'white' as const,
    noteTexts: ['Develop quickly and prepare the centre.'],
    stepTotal: 5,
    hintLevel: previewOpeningHintLevel,
    canReveal: previewOpeningHintLevel >= 4
  }
  const steps: OpeningStepView[] = [
    {
      ...common, stepId: 'explain-plan', kind: 'explain', title: 'The central plan',
      instruction: 'White prepares d4 while keeping the position flexible.',
      positionId: 'after-bc5', currentFen: previewOpeningFens.prompt,
      legalMoves: [], stepNumber: 1
    },
    {
      ...common, stepId: 'watch-setup', kind: 'watch', title: 'Reach the Italian',
      instruction: 'Watch both sides develop toward the Italian Game.',
      variationName: 'Giuoco Piano', positionId: 'initial', currentFen: previewOpeningFens.initial,
      legalMoves: [], stepNumber: 2
    },
    {
      ...common, stepId: 'try-c3', kind: 'try', title: 'Prepare the centre',
      instruction: 'Choose White\'s preparation move.', variationName: 'Giuoco Piano',
      positionId: 'after-bc5', currentFen: previewOpeningFens.prompt,
      legalMoves: ['b2b4', 'c2c3', 'd2d3'], stepNumber: 3
    },
    {
      ...common, stepId: 'branch-giuoco', kind: 'branch', title: 'Recognise the branch',
      instruction: 'Choose White\'s setup after Black develops the bishop.',
      variationName: 'Giuoco Piano', positionId: 'after-bc5', currentFen: previewOpeningFens.prompt,
      legalMoves: ['b2b4', 'c2c3', 'd2d3'], stepNumber: 4
    },
    {
      ...common, stepId: 'recall-c3', kind: 'recall', title: 'Recall the Giuoco move',
      instruction: 'Play the course move from memory.', variationName: 'Giuoco Piano',
      positionId: 'after-bc5', currentFen: previewOpeningFens.prompt,
      legalMoves: ['b2b4', 'c2c3', 'd2d3'], stepNumber: 5
    }
  ]
  return steps[index]
}

function previewActiveOpening(
  index: number,
  mode: 'lesson' | 'review' = 'lesson'
): ActiveOpeningSessionView {
  return {
    sessionId: 'preview-opening-session', mode, status: 'active',
    courseId: 'synthetic-italian', generationId: 'preview-generation',
    lessonId: mode === 'review' ? 'review' : 'giuoco-c3', depth: previewOpeningDepth,
    current: previewOpeningStep(index, mode)
  }
}

function previewCompletedOpening(mode: 'lesson' | 'review'): OpeningSessionView {
  return {
    sessionId: 'preview-opening-session', mode, status: 'completed',
    courseId: 'synthetic-italian', generationId: 'preview-generation',
    lessonId: mode === 'review' ? 'review' : 'giuoco-c3', depth: previewOpeningDepth,
    summary: {
      totalPrompts: mode === 'review' ? 1 : 3,
      positionsRecalled: mode === 'review' ? 1 : 2,
      branchesRecognized: mode === 'review' ? 0 : 1,
      retried: 0, usedHint: previewOpeningHintLevel > 0 ? 1 : 0, revealed: 0
    }
  }
}

function previewOpeningHome(): OpeningHomeView {
  const completed = previewOpeningSession?.status === 'completed'
  return {
    courses: [{
      courseId: 'synthetic-italian', title: 'Synthetic Italian for White',
      perspective: 'white', depth: previewOpeningDepth, rootPositionId: 'initial',
      completedLessons: completed ? 1 : 0, totalLessons: 1, dueReviews: 0,
      nextLessonId: 'giuoco-c3', nextLessonTitle: 'Prepare d4 with c3',
      hasResumable: previewOpeningSession?.status === 'active',
      chapters: [{
        chapterId: 'giuoco', title: 'Giuoco Piano',
        lessons: [{
          lessonId: 'giuoco-c3', title: 'Prepare d4 with c3',
          completedSteps: completed ? 5 : 0, totalSteps: 5, completed
        }]
      }]
    }]
  }
}

function completePreviewOpeningMove(): OpeningStepResult {
  if (!previewOpeningSession || previewOpeningSession.status !== 'active') {
    throw new Error('preview opening session is not active')
  }
  const current = previewOpeningSession
  const nextIndex = previewOpeningStepIndex + 1
  previewOpeningSession = current.mode === 'review' || nextIndex >= 5
    ? previewCompletedOpening(current.mode)
    : previewActiveOpening(nextIndex, current.mode)
  previewOpeningStepIndex = nextIndex
  return {
    session: clonePreviewSession(previewOpeningSession),
    stepCompleted: true,
    feedback: 'expected',
    appliedMoves: [{ uci: 'c2c3', resultingFen: previewOpeningFens.afterC3 }],
    finalFen: previewOpeningFens.afterC3
  }
}

const previewNormalAPI: NormalAPI = {
  getProfile: async () => previewProfile,
  updateProfile: async (profile) => { previewProfile = profile },
  resumeSession: async () => previewSession ? clonePreviewSession(previewSession) : null,
  startGuided: async () => {
    previewIncorrect = new Set()
    previewSession = previewActiveSession(0)
    return clonePreviewSession(previewSession)
  },
  startFreePractice: async () => {
    previewIncorrect = new Set()
    previewSession = previewActiveSession(0, 'practice')
    return clonePreviewSession(previewSession)
  },
  playMove: async (sessionId, uci) => {
    if (!previewSession || previewSession.status !== 'active' ||
      previewSession.sessionId !== sessionId) {
      throw new Error('preview session is not active')
    }
    const activeSession = previewSession
    const puzzle = previewPuzzles[activeSession.currentIndex]
    if (uci === puzzle.correctMove) return completePreviewPuzzle()
    if (uci !== puzzle.wrongMove) {
      throw new Error(`move ${uci} is not configured in the preview puzzle`)
    }
    previewIncorrect.add(activeSession.currentIndex)
    activeSession.current.incorrectMoves = 1
    return {
      session: clonePreviewSession(activeSession),
      correct: false,
      puzzleCompleted: false,
      message: 'Try again'
    }
  },
  useHint: async () => ({
    session: clonePreviewSession(
      previewSession?.status === 'active' ? previewSession : previewActiveSession(0)
    ),
    level: 1,
    text: 'Look for a forcing move.',
    canReveal: false
  }),
  revealSolution: async () => completePreviewPuzzle(),
  pauseSession: async () => {},
  getOpeningHome: async () => structuredClone(previewOpeningHome()),
  getOpeningPosition: async (courseId, positionId, _depth) => {
    if (courseId !== 'synthetic-italian') throw new Error('preview opening course was not found')
    if (positionId === 'initial') {
      return {
        courseId,
        positionId,
        fen: previewOpeningFens.initial,
        label: 'Initial position',
        evaluation: { code: 'none' },
        notes: [],
        moves: [{
          moveId: 'white-e4', uci: 'e2e4', san: 'e4', toPositionId: 'after-e4',
          role: 'repertoire', variationName: 'Italian setup', evaluation: { code: 'equal' },
          sourceRef: { printedPage: 1, coverageId: 'p1-e4' }
        }],
        incomingPaths: 0
      }
    }
    if (positionId === 'after-e4') {
      return {
        courseId,
        positionId,
        fen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1',
        label: 'King pawn opening',
        evaluation: { code: 'equal' },
        notes: [{
          kind: 'overview', text: 'Black meets the centre directly.',
          sourceRef: { printedPage: 1, noteLabel: 'overview', coverageId: 'p1-overview' }
        }],
        moves: [],
        incomingPaths: 1
      }
    }
    throw new Error(`preview opening position ${positionId} was not found`)
  },
  setOpeningDepth: async (_courseId, depth) => { previewOpeningDepth = depth },
  startOpeningLesson: async () => {
    previewOpeningStepIndex = 0
    previewOpeningHintLevel = 0
    previewOpeningSession = previewActiveOpening(0)
    return clonePreviewSession(previewOpeningSession)
  },
  resumeOpeningSession: async () => previewOpeningSession
    ? clonePreviewSession(previewOpeningSession)
    : null,
  restartOpeningSession: async () => {
    previewOpeningStepIndex = 0
    previewOpeningHintLevel = 0
    previewOpeningSession = previewActiveOpening(0)
    return clonePreviewSession(previewOpeningSession)
  },
  advanceOpeningStep: async () => {
    if (!previewOpeningSession || previewOpeningSession.status !== 'active' ||
      (previewOpeningSession.current.kind !== 'explain' &&
        previewOpeningSession.current.kind !== 'watch')) {
      throw new Error('preview opening step requires a move')
    }
    const wasWatch = previewOpeningSession.current.kind === 'watch'
    previewOpeningStepIndex++
    previewOpeningSession = previewActiveOpening(previewOpeningStepIndex)
    return {
      session: clonePreviewSession(previewOpeningSession),
      stepCompleted: true,
      ...(wasWatch
        ? {
            appliedMoves: [{ uci: 'f8c5', resultingFen: previewOpeningFens.prompt }] as AppliedMoveFrames,
            finalFen: previewOpeningFens.prompt
          }
        : {})
    }
  },
  playOpeningMove: async (_sessionId, uci) => {
    if (!previewOpeningSession || previewOpeningSession.status !== 'active') {
      throw new Error('preview opening session is not active')
    }
    if (uci === 'c2c3') return completePreviewOpeningMove()
    if (uci === 'b2b4') {
      return {
        session: clonePreviewSession(previewOpeningSession),
        stepCompleted: false,
        feedback: 'alternative',
        message: 'That is a playable course alternative. Return to the lesson position.'
      }
    }
    return {
      session: clonePreviewSession(previewOpeningSession),
      stepCompleted: false,
      feedback: 'off_course',
      message: 'That move is playable, but this lesson is practising c3.'
    }
  },
  useOpeningHint: async () => {
    if (!previewOpeningSession || previewOpeningSession.status !== 'active') {
      throw new Error('preview opening session is not active')
    }
    previewOpeningHintLevel = Math.min(4, previewOpeningHintLevel + 1)
    previewOpeningSession = previewActiveOpening(
      previewOpeningStepIndex,
      previewOpeningSession.mode
    )
    return {
      session: clonePreviewSession(previewOpeningSession),
      level: previewOpeningHintLevel,
      text: previewOpeningHintLevel === 4
        ? 'Show the course move.'
        : 'Develop quickly and prepare the centre.',
      sourceSquare: previewOpeningHintLevel >= 2 ? 'c2' : undefined,
      targetSquare: previewOpeningHintLevel >= 3 ? 'c3' : undefined,
      canReveal: previewOpeningHintLevel >= 4
    }
  },
  revealOpeningMove: async () => completePreviewOpeningMove(),
  pauseOpeningSession: async () => {},
  startOpeningReview: async () => {
    previewOpeningStepIndex = 0
    previewOpeningHintLevel = 0
    previewOpeningSession = previewActiveOpening(0, 'review')
    return clonePreviewSession(previewOpeningSession)
  },
  getParentSummary: async () => ({
    learnerRating: previewProfile?.learnerRating ?? 1200,
    ratingTrend: [
      { rating: 1150, recordedAt: 1 },
      { rating: previewProfile?.learnerRating ?? 1200, recordedAt: 2 }
    ],
    firstAttemptAccuracy: 68.4,
    hintRate: 18.2,
    themePerformance: [{ theme: 'fork', attempts: 12, accuracy: 75 }],
    dueReviews: 3,
    recentSessions: []
  }),
  getPracticeFilters: async () => ({
    sources: [{
      id: 'lichess', kind: 'lichess', minimumRating: 400, maximumRating: 3000,
      hasRatingRange: true, maximumPlies: 12
    }],
    themes: ['fork', 'mate', 'pin'],
    maximumSolutionPlies: 12,
    learnerRatingBounds: { minimum: 400, maximum: 3000 }
  }),
  createBackup: async () => '/Users/preview/Chess Trainer Backup.zip',
  restoreBackup: async () => {},
  openDataFolder: async () => {},
  quit: async () => {},
  choosePuzzleImportFile: async () => '/Users/preview/Downloads/lichess_db_puzzle.csv.zst',
  inspectPuzzleImport: async () => ({
    path: '/Users/preview/Downloads/lichess_db_puzzle.csv.zst',
    filename: 'lichess_db_puzzle.csv.zst',
    format: 'lichess',
    formatLabel: 'Lichess',
    sourceId: 'lichess',
    sourceIdOrigin: 'fixed',
    replacesExisting: false
  }),
  startPuzzleImport: async () => 'preview-import',
  chooseOpeningCourseFile: async () => '/Users/preview/Documents/italian.ctcourse',
  inspectOpeningCourseImport: async () => ({
    path: '/Users/preview/Documents/italian.ctcourse',
    filename: 'italian.ctcourse',
    format: 'coursepack',
    formatLabel: 'Opening course',
    sourceId: 'italian-white',
    sourceIdOrigin: 'embedded',
    sourceName: 'Italian Game for White',
    replacesExisting: false
  }),
  startOpeningCourseImport: async () => 'preview-course-import',
  cancelImport: async () => {},
  getImportResult: async (jobId) => ({
    jobId,
    status: 'running',
    progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 0 },
    report: { accepted: 0, duplicates: 0, rejected: 0, examples: [], counts: {} }
  }),
  onImportProgress: () => () => {},
  onImportFinished: () => () => {}
}

export function resetPreviewStateForTest(): void {
  previewProfile = null
  previewSession = null
  previewIncorrect = new Set<number>()
  previewOpeningDepth = 'reference'
  previewOpeningSession = null
  previewOpeningStepIndex = 0
  previewOpeningHintLevel = 0
}

export async function loadPreviewApplicationAPI(): Promise<ApplicationAPI> {
  return { mode: 'normal', buildInfo: previewBuildInfo, api: previewNormalAPI }
}
