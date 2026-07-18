import { GetApplicationMode, GetBuildInfo } from '../../wailsjs/go/main/ModeController'
import * as Normal from '../../wailsjs/go/main/NormalController'
import * as Recovery from '../../wailsjs/go/main/RecoveryController'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import {
  decodeApplicationMode,
  decodeBuildInfo,
  decodeHintResult,
  decodeImportInspection,
  decodeImportProgress,
  decodeImportResult,
  decodeMoveResult,
  decodeParentSummary,
  decodePracticeFilters,
  decodeProfile,
  decodeRecoveryState,
  decodeSession,
  type ActiveSessionView,
  type AppliedMoveFrames,
  type ApplicationMode,
  type BuildInfo,
  type CompletedSessionView,
  type HintResult,
  type ImportInspection,
  type ImportProgress,
  type ImportResult,
  type MoveResult,
  type ParentSummary,
  type PracticeFilters,
  type Profile,
  type PuzzleView,
  type RecoveryState,
  type SessionView
} from './api-contract'

export type {
  ActiveSessionView,
  AppliedMove,
  AppliedMoveFrames,
  ApplicationMode,
  BuildInfo,
  CompletedMoveResult,
  CompletedSessionView,
  ContinuingMoveResult,
  HintResult,
  ImportFormat,
  ImportInspection,
  ImportPhase,
  ImportProgress,
  ImportRejection,
  ImportReport,
  ImportResult,
  ImportSourceIDOrigin,
  IncorrectMoveResult,
  MoveResult,
  ParentSummary,
  PracticeFilters,
  PracticeSource,
  Profile,
  PuzzleView,
  RatingBounds,
  RatingPoint,
  RecentSession,
  RecoveryState,
  SessionMode,
  SessionSummary,
  SessionView,
  ThemePerformance
} from './api-contract'

export type PracticeRequest = {
  sourceId: string
  minimumRating?: number
  maximumRating?: number
  themes: string[]
  maximumSolutionPlies?: number
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
  inspectPuzzleImport(path: string): Promise<ImportInspection>
  startPuzzleImport(path: string): Promise<string>
  cancelImport(jobId: string): Promise<void>
  getImportResult(jobId: string): Promise<ImportResult>
  onImportProgress(listener: (progress: ImportProgress) => void): () => void
  onImportFinished(listener: (result: ImportResult) => void): () => void
}

export interface RecoveryAPI extends BackupAPI {
  getRecoveryState(): Promise<RecoveryState>
}

export type ApplicationAPI =
  | { mode: 'normal'; buildInfo: BuildInfo; api: NormalAPI }
  | { mode: 'recovery'; buildInfo: BuildInfo; api: RecoveryAPI }

let productionBootstrap: Promise<[ApplicationMode, BuildInfo]> | null = null

function getProductionBootstrap(): Promise<[ApplicationMode, BuildInfo]> {
  productionBootstrap ??= Promise.all([
    GetApplicationMode().then(decodeApplicationMode),
    GetBuildInfo().then(decodeBuildInfo)
  ])
  return productionBootstrap
}

const productionNormalAPI: NormalAPI = {
  getProfile: async () => decodeProfile(await Normal.GetProfile()),
  updateProfile: async (profile) => Normal.UpdateProfile(profile),
  resumeSession: async () => {
    const session = await Normal.ResumeSession()
    return session == null ? null : decodeSession(session)
  },
  startGuided: async () => decodeSession(await Normal.StartGuided()),
  startFreePractice: async (request) => decodeSession(await Normal.StartFreePractice(request)),
  playMove: async (sessionId, uci) => decodeMoveResult(await Normal.PlayMove(sessionId, uci)),
  useHint: async (sessionId) => decodeHintResult(await Normal.UseHint(sessionId)),
  revealSolution: async (sessionId) => decodeMoveResult(await Normal.RevealSolution(sessionId)),
  pauseSession: Normal.PauseSession,
  getParentSummary: async () => decodeParentSummary(await Normal.GetParentSummary()),
  getPracticeFilters: async () => decodePracticeFilters(await Normal.GetPracticeFilters()),
  createBackup: Normal.CreateBackup,
  restoreBackup: Normal.RestoreBackup,
  openDataFolder: Normal.OpenDataFolder,
  quit: Normal.Quit,
  choosePuzzleImportFile: Normal.ChoosePuzzleImportFile,
  inspectPuzzleImport: async (path) => decodeImportInspection(await Normal.InspectPuzzleImport(path)),
  startPuzzleImport: Normal.StartPuzzleImport,
  cancelImport: Normal.CancelImport,
  getImportResult: async (jobId) => decodeImportResult(await Normal.GetImportResult(jobId)),
  onImportProgress: (listener) => EventsOn('import:progress', (payload: unknown) => {
    listener(decodeImportProgress(payload))
  }),
  onImportFinished: (listener) => EventsOn('import:finished', (payload: unknown) => {
    listener(decodeImportResult(payload))
  })
}

const productionRecoveryAPI: RecoveryAPI = {
  getRecoveryState: async () => decodeRecoveryState(await Recovery.GetRecoveryState()),
  createBackup: Recovery.CreateBackup,
  restoreBackup: Recovery.RestoreBackup,
  openDataFolder: Recovery.OpenDataFolder,
  quit: Recovery.Quit
}

async function loadProductionApplicationAPI(): Promise<ApplicationAPI> {
  const [mode, buildInfo] = await getProductionBootstrap()
  return mode === 'recovery'
    ? { mode, buildInfo, api: productionRecoveryAPI }
    : { mode, buildInfo, api: productionNormalAPI }
}

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

function clonePreviewSession<Session extends SessionView>(session: Session): Session {
  return structuredClone(session)
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
  inspectPuzzleImport: async () => ({
    path: '/Users/preview/Downloads/lichess_db_puzzle.csv.zst',
    filename: 'lichess_db_puzzle.csv.zst',
    format: 'lichess',
    sourceId: 'lichess',
    sourceIdOrigin: 'fixed',
    replacesExisting: false
  }),
  startPuzzleImport: async () => 'preview-import',
  cancelImport: async () => {},
  getImportResult: async (jobId) => ({
    jobId,
    status: 'running',
    progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 0 },
    report: { accepted: 0, duplicates: 0, rejected: 0, examples: [] }
  }),
  onImportProgress: () => () => {},
  onImportFinished: () => () => {}
}

const isProduction = typeof window !== 'undefined' && window.go

export function loadApplicationAPI(): Promise<ApplicationAPI> {
  return isProduction
    ? loadProductionApplicationAPI()
    : Promise.resolve({ mode: 'normal', buildInfo: previewBuildInfo, api: previewNormalAPI })
}
