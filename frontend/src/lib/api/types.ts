import type {
  BuildInfo,
  ParentSummary,
  PracticeFilters,
  Profile,
  RecoveryState
} from '../contracts/application'
import type { ImportInspection, ImportProgress, ImportResult } from '../contracts/imports'
import type {
  OpeningDepth,
  OpeningHintResult,
  OpeningHomeView,
  OpeningPositionView,
  OpeningSessionView,
  OpeningStepResult
} from '../contracts/openings'
import type { HintResult, MoveResult, SessionView } from '../contracts/puzzles'

export type PracticeRequest = {
  sourceId: string
  minimumRating?: number
  maximumRating?: number
  themes: string[]
  maximumSolutionPlies?: number
}

export interface BackupAPI {
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
  getOpeningHome(): Promise<OpeningHomeView>
  getOpeningPosition(
    courseId: string,
    positionId: string,
    depth: OpeningDepth
  ): Promise<OpeningPositionView>
  setOpeningDepth(courseId: string, depth: OpeningDepth): Promise<void>
  startOpeningLesson(courseId: string, lessonId: string): Promise<OpeningSessionView>
  resumeOpeningSession(): Promise<OpeningSessionView | null>
  restartOpeningSession(sessionId: string): Promise<OpeningSessionView>
  advanceOpeningStep(sessionId: string): Promise<OpeningStepResult>
  playOpeningMove(sessionId: string, uci: string): Promise<OpeningStepResult>
  useOpeningHint(sessionId: string): Promise<OpeningHintResult>
  revealOpeningMove(sessionId: string): Promise<OpeningStepResult>
  pauseOpeningSession(sessionId: string): Promise<void>
  startOpeningReview(courseId: string): Promise<OpeningSessionView>
  getParentSummary(): Promise<ParentSummary>
  getPracticeFilters(): Promise<PracticeFilters>
  choosePuzzleImportFile(): Promise<string>
  inspectPuzzleImport(path: string): Promise<ImportInspection>
  startPuzzleImport(inspection: ImportInspection): Promise<string>
  chooseOpeningCourseFile(): Promise<string>
  inspectOpeningCourseImport(path: string): Promise<ImportInspection>
  startOpeningCourseImport(inspection: ImportInspection): Promise<string>
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
