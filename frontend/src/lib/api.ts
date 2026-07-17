import { GetApplicationMode } from '../../wailsjs/go/main/ModeController'
import * as Normal from '../../wailsjs/go/main/NormalController'
import * as Recovery from '../../wailsjs/go/main/RecoveryController'
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
  legalMoves: string[]
}

export type AppliedMove = { uci: string; resultingFen: string }

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
  appliedMoves?: AppliedMove[]
  finalFen?: string
}

export type HintResult = {
  session: SessionView
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

export type RatingBounds = {
  minimum: number
  maximum: number
}

export type PracticeFilters = {
  sources: PracticeSource[]
  themes: string[]
  maximumSolutionPlies: number
  learnerRatingBounds: RatingBounds
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

export type RecoveryState = {
  required: boolean
  path?: string
  detail?: string
}

export type ApplicationMode = 'normal' | 'recovery'

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
  progress?: { rowsRead: number; bytesRead: number }
  report: ImportReport
  error?: string
}

interface BackupAPI {
  createBackup(includeLibrary: boolean): Promise<string>
  restoreBackup(path: string): Promise<void>
  openDataFolder(): Promise<void>
  quit(): Promise<void>
}

export interface NormalAPI extends BackupAPI {
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
  choosePuzzleImportFile(): Promise<string>
  startLichessImport(path: string): Promise<string>
  cancelImport(jobId: string): Promise<void>
  getImportResult(jobId: string): Promise<ImportResult>
  onImportProgress(listener: (progress: ImportProgress) => void): () => void
  onImportFinished(listener: (result: ImportResult) => void): () => void
}

export interface RecoveryAPI extends BackupAPI {
  getRecoveryState(): Promise<RecoveryState>
}

export type ApplicationAPI =
  | { mode: 'normal'; api: NormalAPI }
  | { mode: 'recovery'; api: RecoveryAPI }

let productionMode: Promise<ApplicationMode> | null = null

function getProductionMode(): Promise<ApplicationMode> {
  productionMode ??= GetApplicationMode() as Promise<ApplicationMode>
  return productionMode
}

const productionNormalAPI: NormalAPI = {
  getProfile: async () => (await Normal.GetProfile()) as Profile | null,
  updateProfile: async (profile) => Normal.UpdateProfile(profile),
  resumeSession: async () => (await Normal.ResumeSession()) as SessionView | null,
  startGuided: async () => (await Normal.StartGuided()) as SessionView,
  startFreePractice: async (request) => (await Normal.StartFreePractice(request)) as SessionView,
  playMove: async (sessionId, uci) => (await Normal.PlayMove(sessionId, uci)) as MoveResult,
  useHint: async (sessionId) => (await Normal.UseHint(sessionId)) as HintResult,
  revealSolution: async (sessionId) => (await Normal.RevealSolution(sessionId)) as MoveResult,
  pauseSession: Normal.PauseSession,
  getParentSummary: async () => (await Normal.GetParentSummary()) as ParentSummary,
  getPracticeFilters: async () => (await Normal.GetPracticeFilters()) as PracticeFilters,
  createBackup: Normal.CreateBackup,
  restoreBackup: Normal.RestoreBackup,
  openDataFolder: Normal.OpenDataFolder,
  quit: Normal.Quit,
  choosePuzzleImportFile: Normal.ChoosePuzzleImportFile,
  startLichessImport: Normal.StartLichessImport,
  cancelImport: Normal.CancelImport,
  getImportResult: async (jobId) => (await Normal.GetImportResult(jobId)) as ImportResult,
  onImportProgress: (listener) => EventsOn('import:progress', listener),
  onImportFinished: (listener) => EventsOn('import:finished', listener)
}

const productionRecoveryAPI: RecoveryAPI = {
  getRecoveryState: async () => (await Recovery.GetRecoveryState()) as RecoveryState,
  createBackup: Recovery.CreateBackup,
  restoreBackup: Recovery.RestoreBackup,
  openDataFolder: Recovery.OpenDataFolder,
  quit: Recovery.Quit
}

async function loadProductionApplicationAPI(): Promise<ApplicationAPI> {
  const mode = await getProductionMode()
  return mode === 'recovery'
    ? { mode, api: productionRecoveryAPI }
    : { mode, api: productionNormalAPI }
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
): SessionView {
  return {
    sessionId: mode === 'practice' ? 'preview-practice' : 'preview-session',
    mode,
    status: 'active',
    currentIndex: index,
    total: previewPuzzles.length,
    current: previewPuzzle(index)
  }
}

function previewCompletedSession(session: SessionView): SessionView {
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

function clonePreviewSession(session: SessionView): SessionView {
  return JSON.parse(JSON.stringify(session)) as SessionView
}

function completePreviewPuzzle(): MoveResult {
  if (!previewSession?.current || previewSession.status !== 'active') {
    throw new Error('preview session is not active')
  }
  const index = previewSession.currentIndex
  const puzzle = previewPuzzles[index]
  const appliedMoves = [{ uci: puzzle.correctMove, resultingFen: puzzle.finalFen }]
  previewSession = index + 1 < previewPuzzles.length
    ? previewActiveSession(index + 1, previewSession.mode as 'guided' | 'practice')
    : previewCompletedSession(previewSession)
  return {
    session: clonePreviewSession(previewSession),
    correct: true,
    puzzleCompleted: true,
    appliedMoves,
    finalFen: puzzle.finalFen
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
    if (!previewSession?.current || previewSession.sessionId !== sessionId ||
      previewSession.status !== 'active') {
      throw new Error('preview session is not active')
    }
    const puzzle = previewPuzzles[previewSession.currentIndex]
    if (uci === puzzle.correctMove) return completePreviewPuzzle()
    if (uci !== puzzle.wrongMove) {
      throw new Error(`move ${uci} is not configured in the preview puzzle`)
    }
    previewIncorrect.add(previewSession.currentIndex)
    previewSession.current.incorrectMoves = 1
    return {
      session: clonePreviewSession(previewSession),
      correct: false,
      puzzleCompleted: false,
      message: 'Try again'
    }
  },
  useHint: async () => ({
    session: clonePreviewSession(previewSession ?? previewActiveSession(0)),
    level: 1,
    text: 'Look for a forcing move.',
    canReveal: false
  }),
  revealSolution: async () => completePreviewPuzzle(),
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
    maximumSolutionPlies: 12,
    learnerRatingBounds: { minimum: 400, maximum: 3000 }
  }),
  createBackup: async () => '/Users/preview/Chess Trainer Backup.zip',
  restoreBackup: async () => {},
  openDataFolder: async () => {},
  quit: async () => {},
  choosePuzzleImportFile: async () => '/Users/preview/Downloads/lichess_db_puzzle.csv.zst',
  startLichessImport: async () => 'preview-import',
  cancelImport: async () => {},
  getImportResult: async (jobId) => ({
    jobId, status: 'running', report: { accepted: 0, duplicates: 0, rejected: 0 }
  }),
  onImportProgress: () => () => {},
  onImportFinished: () => () => {}
}

const isProduction = typeof window !== 'undefined' && window.go

export function loadApplicationAPI(): Promise<ApplicationAPI> {
  return isProduction
    ? loadProductionApplicationAPI()
    : Promise.resolve({ mode: 'normal', api: previewNormalAPI })
}
