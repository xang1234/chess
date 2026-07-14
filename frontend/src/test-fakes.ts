import type { AppAPI, ImportResult, Profile, SessionView } from './lib/api'

const emptySession: SessionView = {
  sessionId: 'session-1',
  mode: 'guided',
  status: 'active',
  currentIndex: 0,
  total: 1
}

export function fakeAPI(overrides: Partial<AppAPI> = {}): AppAPI {
  return {
    getProfile: async () => null,
    updateProfile: async (_profile: Profile) => {},
    resumeSession: async () => null,
    startGuided: async () => emptySession,
    playMove: async () => ({}),
    useHint: async () => ({}),
    revealSolution: async () => ({}),
    pauseSession: async () => {},
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
