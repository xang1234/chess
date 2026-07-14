import {
  CancelImport,
  GetImportResult,
  GetParentSummary,
  GetPracticeFilters,
  GetProfile,
  PauseSession,
  PlayMove,
  ResumeSession,
  RevealSolution,
  StartFreePractice,
  StartGuided,
  StartLichessImport,
  UpdateProfile,
  UseHint
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

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
}

export type SessionSummary = {
  total: number
  firstTry: number
  retried: number
  usedHint: number
  revealed: number
  unavailable: number
}

export type SessionView = {
  sessionId: string
  mode: string
  status: string
  currentIndex: number
  total: number
  current?: PuzzleView
  summary?: SessionSummary
}

export type MoveResult = {
  session: SessionView
  correct: boolean
  puzzleCompleted: boolean
  message?: string
}

export type HintResult = {
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

export type PracticeFilters = {
  sources: PracticeSource[]
  themes: string[]
  maximumSolutionPlies: number
}

export type PracticeRequest = {
  sourceId: string
  minimumRating?: number
  maximumRating?: number
  themes: string[]
  maximumSolutionPlies?: number
}

export type RatingPoint = { rating: number; recordedAt: number }
export type ThemePerformance = { theme: string; attempts: number; accuracy: number }
export type RecentSession = {
  sessionId: string
  mode: string
  status: string
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

export type ImportProgress = {
  jobId: string
  rowsRead: number
  bytesRead: number
}

export type ImportReport = {
  accepted: number
  duplicates: number
  rejected: number
}

export type ImportResult = {
  jobId: string
  status: 'running' | 'succeeded' | 'failed' | 'cancelled'
  report: ImportReport
  error?: string
}

export interface AppAPI {
  getProfile(): Promise<Profile | null>
  updateProfile(profile: Profile): Promise<void>
  resumeSession(): Promise<SessionView | null>
  startGuided(): Promise<SessionView>
  startFreePractice(request: PracticeRequest): Promise<SessionView>
  playMove(sessionId: string, uci: string): Promise<MoveResult>
  useHint(sessionId: string): Promise<HintResult>
  revealSolution(sessionId: string): Promise<MoveResult>
  pauseSession(sessionId: string): Promise<void>
  getParentSummary(): Promise<ParentSummary>
  getPracticeFilters(): Promise<PracticeFilters>
  startLichessImport(path: string): Promise<string>
  cancelImport(jobId: string): Promise<void>
  getImportResult(jobId: string): Promise<ImportResult>
  onImportProgress(listener: (progress: ImportProgress) => void): () => void
  onImportFinished(listener: (result: ImportResult) => void): () => void
}

const productionAPI: AppAPI = {
  getProfile: async () => (await GetProfile()) as Profile | null,
  updateProfile: async (profile) => UpdateProfile(profile),
  resumeSession: async () => (await ResumeSession()) as SessionView | null,
  startGuided: async () => (await StartGuided()) as SessionView,
  startFreePractice: async (request) => (await StartFreePractice(request)) as SessionView,
  playMove: async (sessionId, uci) => (await PlayMove(sessionId, uci)) as MoveResult,
  useHint: async (sessionId) => (await UseHint(sessionId)) as HintResult,
  revealSolution: async (sessionId) => (await RevealSolution(sessionId)) as MoveResult,
  pauseSession: PauseSession,
  getParentSummary: async () => (await GetParentSummary()) as ParentSummary,
  getPracticeFilters: async () => (await GetPracticeFilters()) as PracticeFilters,
  startLichessImport: StartLichessImport,
  cancelImport: CancelImport,
  getImportResult: async (jobId) => (await GetImportResult(jobId)) as ImportResult,
  onImportProgress: (listener) => EventsOn('import:progress', listener),
  onImportFinished: (listener) => EventsOn('import:finished', listener)
}

let previewProfile: Profile | null = null
const previewAPI: AppAPI = {
  getProfile: async () => previewProfile,
  updateProfile: async (profile) => { previewProfile = profile },
  resumeSession: async () => null,
  startGuided: async () => ({
    sessionId: 'preview-session', mode: 'guided', status: 'active', currentIndex: 0, total: 10
  }),
  startFreePractice: async () => ({
    sessionId: 'preview-practice', mode: 'practice', status: 'active', currentIndex: 0, total: 5
  }),
  playMove: async () => ({
    session: {
      sessionId: 'preview-session', mode: 'guided', status: 'active', currentIndex: 0, total: 10
    },
    correct: false,
    puzzleCompleted: false,
    message: 'Try again'
  }),
  useHint: async () => ({ level: 1, text: 'Look for a forcing move.', canReveal: false }),
  revealSolution: async () => ({
    session: {
      sessionId: 'preview-session', mode: 'guided', status: 'complete', currentIndex: 10, total: 10,
      summary: { total: 10, firstTry: 7, retried: 2, usedHint: 1, revealed: 0, unavailable: 0 }
    },
    correct: true,
    puzzleCompleted: true
  }),
  pauseSession: async () => {},
  getParentSummary: async () => ({
    learnerRating: previewProfile?.learnerRating ?? 1200,
    ratingTrend: [{ rating: 1150, recordedAt: 1 }, { rating: previewProfile?.learnerRating ?? 1200, recordedAt: 2 }],
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
    maximumSolutionPlies: 12
  }),
  startLichessImport: async () => 'preview-import',
  cancelImport: async () => {},
  getImportResult: async (jobId) => ({
    jobId, status: 'running', report: { accepted: 0, duplicates: 0, rejected: 0 }
  }),
  onImportProgress: () => () => {},
  onImportFinished: () => () => {}
}

const defaultAPI = typeof window !== 'undefined' && window.go ? productionAPI : previewAPI
let currentAPI = defaultAPI

export function getAPI(): AppAPI {
  return currentAPI
}

export function setAPIForTests(api: AppAPI): void {
  currentAPI = api
}

export function resetAPIForTests(): void {
  currentAPI = defaultAPI
}
