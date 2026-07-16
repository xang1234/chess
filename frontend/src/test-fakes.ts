import type {
  ApplicationAPI,
  ImportResult,
  NormalAPI,
  Profile,
  RecoveryAPI,
  SessionView
} from './lib/api'
import NormalAPIProvider from './test-providers/NormalAPIProvider.svelte'
import RecoveryAPIProvider from './test-providers/RecoveryAPIProvider.svelte'

const emptySession: SessionView = {
  sessionId: 'session-1',
  mode: 'guided',
  status: 'active',
  currentIndex: 0,
  total: 1
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
    revealSolution: async () => ({ session: emptySession, correct: true, puzzleCompleted: true }),
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

export function normalApplication(api: NormalAPI): ApplicationAPI {
  return { mode: 'normal', api }
}

export function recoveryApplication(api: RecoveryAPI): ApplicationAPI {
  return { mode: 'recovery', api }
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
