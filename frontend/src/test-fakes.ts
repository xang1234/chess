import type {
  ApplicationAPI,
  ActiveSessionView,
  BuildInfo,
  ImportResult,
  NormalAPI,
  OpeningHomeView,
  ActiveOpeningSessionView,
  Profile,
  RecoveryAPI
} from './lib/api'
import NormalAPIProvider from './test-providers/NormalAPIProvider.svelte'
import RecoveryAPIProvider from './test-providers/RecoveryAPIProvider.svelte'

export const fakeStartingFen = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'
export const fakeLegalMoves = [
  'a2a3', 'a2a4', 'b1a3', 'b1c3', 'b2b3', 'b2b4',
  'c2c3', 'c2c4', 'd2d3', 'd2d4', 'e2e3', 'e2e4',
  'f2f3', 'f2f4', 'g1f3', 'g1h3', 'g2g3', 'g2g4',
  'h2h3', 'h2h4'
]

export const fakeBuildInfo: BuildInfo = {
  name: 'Chess Trainer',
  commit: 'development',
  sourceUrl: 'https://github.com/xang1234/chess'
}

const emptySession: ActiveSessionView = {
  sessionId: 'session-1',
  mode: 'guided',
  status: 'active',
  currentIndex: 0,
  total: 1,
  current: {
    fingerprint: 'puzzle-1',
    displayedFen: fakeStartingFen,
    currentFen: fakeStartingFen,
    solver: 'white',
    currentPath: [],
    puzzleNumber: 1,
    puzzleTotal: 1,
    hintLevel: 0,
    incorrectMoves: 0,
    canReveal: false,
    legalMoves: [...fakeLegalMoves]
  }
}

export const fakeOpeningHome: OpeningHomeView = {
  courses: [{
    courseId: 'synthetic-italian',
    title: 'Italian Game for White',
    perspective: 'white',
    depth: 'reference',
    rootPositionId: 'initial',
    completedLessons: 1,
    totalLessons: 3,
    dueReviews: 3,
    nextLessonId: 'giuoco-c3',
    nextLessonTitle: 'Giuoco Piano',
    currentPath: [],
    recommendedLessonId: 'giuoco-c3',
    recommendedLessonTitle: 'Giuoco Piano',
    hasResumable: false,
    hasResumableReview: false,
    tree: {
      rootLessonId: 'giuoco-c3',
      nodes: [{
        lessonId: 'giuoco-c3',
        chapterId: 'giuoco',
        title: 'Giuoco Piano',
        objective: 'Prepare the central break with c3.',
        minimumDepth: 'quick',
        progress: 'available',
        completedActivities: 0,
        requiredActivities: 3,
        recommended: true,
        reviewDue: false,
        visible: true
      }],
      edges: []
    },
    chapters: [{
      chapterId: 'giuoco',
      title: 'Giuoco Piano',
      lessons: [{
        lessonId: 'giuoco-c3',
        title: 'Giuoco Piano',
        completedSteps: 0,
        totalSteps: 5,
        completed: false
      }]
    }]
  }]
}

export const fakeOpeningSession: ActiveOpeningSessionView = {
  sessionId: 'opening-session-1',
  mode: 'lesson',
  status: 'active',
  courseId: 'synthetic-italian',
  generationId: 'generation-1',
  lessonId: 'giuoco-c3',
  courseTitle: 'Italian Game for White',
  path: [{ lessonId: 'giuoco-c3', title: 'Giuoco Piano' }],
  depth: 'reference',
  current: {
    activityId: 'explain-plan',
    kind: 'concept',
    title: 'The central plan',
    instruction: 'White prepares d4 while keeping the position flexible.',
    required: true,
    positionId: 'after-bc5',
    currentFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4',
    orientation: 'white',
    legalMoves: [],
    teachingNoteTexts: ['Develop quickly and prepare the centre.'],
    referenceNoteTexts: [],
    comparison: [],
    annotations: [],
    movesToHere: [],
    activityNumber: 1,
    activityTotal: 3,
    completedIdeas: 0,
    requiredIdeas: 3,
    hintLevel: 0,
    canReveal: false,
    referenceSections: []
  }
}

export function fakeAPI(overrides: Partial<NormalAPI> = {}): NormalAPI {
  return {
    getProfile: async () => null,
    updateProfile: async (_profile: Profile) => {},
    resumeSession: async () => null,
    startGuided: async () => emptySession,
    startFreePractice: async () => emptySession,
    playMove: async () => ({ session: emptySession, correct: false, puzzleCompleted: false }),
    useHint: async () => ({
      session: emptySession,
      level: 1,
      text: 'Look for a forcing move.',
      canReveal: false
    }),
    revealSolution: async () => ({
      session: emptySession,
      correct: true,
      puzzleCompleted: true,
      appliedMoves: [{
        uci: 'e2e4',
        resultingFen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1'
      }],
      finalFen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1'
    }),
    pauseSession: async () => {},
    getOpeningHome: async () => fakeOpeningHome,
    getOpeningPosition: async (courseId, positionId) => ({
      courseId,
      positionId,
      fen: fakeStartingFen,
      label: 'Initial position',
      evaluation: { code: 'none' },
      notes: [],
      moves: [{
        moveId: 'white-e4',
        uci: 'e2e4',
        san: 'e4',
        toPositionId: 'after-e4',
        role: 'repertoire',
        variationName: 'Italian setup',
        evaluation: { code: 'equal' },
        sourceRef: { printedPage: 1, coverageId: 'p1-e4' }
      }],
      incomingPaths: 0
    }),
    setOpeningDepth: async () => {},
    startOpeningLesson: async () => fakeOpeningSession,
    resumeOpeningSession: async () => null,
    restartOpeningSession: async () => fakeOpeningSession,
    advanceOpeningActivity: async () => ({
      session: fakeOpeningSession,
      activityCompleted: true
    }),
    advanceOpeningStep: async () => ({
      session: fakeOpeningSession,
      activityCompleted: true
    }),
    playOpeningMove: async () => ({
      session: fakeOpeningSession,
      activityCompleted: false,
      feedback: 'off_course',
      message: 'Try the course move.'
    }),
    useOpeningHint: async () => ({
      session: fakeOpeningSession,
      level: 1,
      text: 'Develop quickly and prepare the centre.',
      canReveal: false
    }),
    revealOpeningMove: async () => ({
      session: fakeOpeningSession,
      activityCompleted: true,
      feedback: 'expected',
      appliedMoves: [{
        uci: 'c2c3',
        resultingFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'
      }],
      finalFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'
    }),
    pauseOpeningSession: async () => {},
    startOpeningReview: async () => ({ ...fakeOpeningSession, mode: 'review', lessonId: 'review' }),
    getParentSummary: async () => ({
      learnerRating: 1200,
      ratingTrend: [],
      firstAttemptAccuracy: 0,
      hintRate: 0,
      themePerformance: [],
      dueReviews: 0,
      recentSessions: []
    }),
    getPracticeFilters: async () => ({
      sources: [],
      themes: [],
      maximumSolutionPlies: 1,
      learnerRatingBounds: { minimum: 400, maximum: 3000 }
    }),
    createBackup: async () => '/tmp/Chess Trainer Backup.zip',
    restoreBackup: async () => {},
    openDataFolder: async () => {},
    quit: async () => {},
    choosePuzzleImportFile: async () => '/tmp/puzzles.csv.zst',
    inspectPuzzleImport: async () => ({
      path: '/tmp/puzzles.csv.zst',
      filename: 'puzzles.csv.zst',
      format: 'lichess',
      formatLabel: 'Lichess',
      sourceId: 'lichess',
      sourceIdOrigin: 'fixed',
      replacesExisting: false
    }),
    startPuzzleImport: async () => 'job-1',
    chooseOpeningCourseFile: async () => '/tmp/italian.ctcourse',
    inspectOpeningCourseImport: async () => ({
      path: '/tmp/italian.ctcourse',
      filename: 'italian.ctcourse',
      format: 'coursepack',
      formatLabel: 'Opening course',
      sourceId: 'italian-white',
      sourceIdOrigin: 'embedded',
      replacesExisting: false
    }),
    startOpeningCourseImport: async () => 'course-job-1',
    cancelImport: async () => {},
    getImportResult: async (): Promise<ImportResult> => ({
      jobId: 'job-1',
      status: 'running',
      progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 0 },
      report: { accepted: 0, duplicates: 0, rejected: 0, examples: [], counts: {} }
    }),
    onImportProgress: () => () => {},
    onImportFinished: () => () => {},
    ...overrides
  }
}

export function fakeRecoveryAPI(overrides: Partial<RecoveryAPI> = {}): RecoveryAPI {
  return {
    getRecoveryState: async () => ({ required: true }),
    createBackup: async () => '/tmp/Chess Trainer Backup.zip',
    restoreBackup: async () => {},
    openDataFolder: async () => {},
    quit: async () => {},
    ...overrides
  }
}

export function normalApplication(
  api: NormalAPI,
  buildInfo: BuildInfo = fakeBuildInfo
): ApplicationAPI {
  return { mode: 'normal', buildInfo, api }
}

export function recoveryApplication(
  api: RecoveryAPI,
  buildInfo: BuildInfo = fakeBuildInfo
): ApplicationAPI {
  return { mode: 'recovery', buildInfo, api }
}

export function withNormalAPI(api: NormalAPI) {
  return {
    wrapper: NormalAPIProvider,
    wrapperProps: { api }
  }
}

export function withRecoveryAPI(api: RecoveryAPI) {
  return {
    wrapper: RecoveryAPIProvider,
    wrapperProps: { api }
  }
}
