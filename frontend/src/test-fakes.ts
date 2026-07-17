import type {
  ApplicationAPI,
  ActiveSessionView,
  BuildInfo,
  ImportResult,
  NormalAPI,
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
    startLichessImport: async () => 'job-1',
    cancelImport: async () => {},
    getImportResult: async (): Promise<ImportResult> => ({
      jobId: 'job-1',
      status: 'running',
      report: { accepted: 0, duplicates: 0, rejected: 0 }
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
