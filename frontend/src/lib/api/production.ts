import { GetApplicationMode, GetBuildInfo } from '../../../wailsjs/go/main/ModeController'
import * as Normal from '../../../wailsjs/go/main/NormalController'
import * as Recovery from '../../../wailsjs/go/main/RecoveryController'
import { EventsOn } from '../../../wailsjs/runtime/runtime'
import {
  decodeApplicationMode,
  decodeBuildInfo,
  decodeParentSummary,
  decodePracticeFilters,
  decodeProfile,
  decodeRecoveryState,
  type ApplicationMode,
  type BuildInfo
} from '../contracts/application'
import {
  decodeImportInspection,
  decodeImportProgress,
  decodeImportResult
} from '../contracts/imports'
import {
  decodeOpeningActivityResult,
  decodeOpeningHintResult,
  decodeOpeningHome,
  decodeOpeningPosition,
  decodeOpeningSession
} from '../contracts/openings'
import {
  decodeHintResult,
  decodeMoveResult,
  decodeSession
} from '../contracts/puzzles'
import type { ApplicationAPI, NormalAPI, RecoveryAPI } from './types'

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
  getOpeningHome: async () => decodeOpeningHome(await Normal.GetOpeningHome()),
  getOpeningPosition: async (courseId, positionId, depth) => decodeOpeningPosition(
    await Normal.GetOpeningPosition(courseId, positionId, depth)
  ),
  setOpeningDepth: Normal.SetOpeningDepth,
  startOpeningLesson: async (courseId, lessonId) => decodeOpeningSession(
    await Normal.StartOpeningLesson(courseId, lessonId)
  ),
  resumeOpeningSession: async () => {
    const session = await Normal.ResumeOpeningSession()
    return session == null ? null : decodeOpeningSession(session)
  },
  restartOpeningSession: async (sessionId) => decodeOpeningSession(
    await Normal.RestartOpeningSession(sessionId)
  ),
  advanceOpeningActivity: async (sessionId) => decodeOpeningActivityResult(
    await Normal.AdvanceOpeningActivity(sessionId)
  ),
  advanceOpeningStep: async (sessionId) => decodeOpeningActivityResult(
    await Normal.AdvanceOpeningStep(sessionId)
  ),
  playOpeningMove: async (sessionId, uci) => decodeOpeningActivityResult(
    await Normal.PlayOpeningMove(sessionId, uci)
  ),
  useOpeningHint: async (sessionId) => decodeOpeningHintResult(
    await Normal.UseOpeningHint(sessionId)
  ),
  revealOpeningMove: async (sessionId) => decodeOpeningActivityResult(
    await Normal.RevealOpeningMove(sessionId)
  ),
  pauseOpeningSession: Normal.PauseOpeningSession,
  startOpeningReview: async (courseId) => decodeOpeningSession(
    await Normal.StartOpeningReview(courseId)
  ),
  getParentSummary: async () => decodeParentSummary(await Normal.GetParentSummary()),
  getPracticeFilters: async () => decodePracticeFilters(await Normal.GetPracticeFilters()),
  createBackup: Normal.CreateBackup,
  restoreBackup: Normal.RestoreBackup,
  openDataFolder: Normal.OpenDataFolder,
  quit: Normal.Quit,
  choosePuzzleImportFile: Normal.ChoosePuzzleImportFile,
  inspectPuzzleImport: async (path) => decodeImportInspection(await Normal.InspectPuzzleImport(path)),
  startPuzzleImport: Normal.StartPuzzleImport,
  chooseOpeningCourseFile: Normal.ChooseOpeningCourseFile,
  inspectOpeningCourseImport: async (path) => decodeImportInspection(
    await Normal.InspectOpeningCourseImport(path)
  ),
  startOpeningCourseImport: Normal.StartOpeningCourseImport,
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

export async function loadProductionApplicationAPI(): Promise<ApplicationAPI> {
  const [mode, buildInfo] = await getProductionBootstrap()
  return mode === 'recovery'
    ? { mode, buildInfo, api: productionRecoveryAPI }
    : { mode, buildInfo, api: productionNormalAPI }
}
