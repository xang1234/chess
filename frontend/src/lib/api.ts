import {
  CancelImport,
  GetImportResult,
  GetProfile,
  PauseSession,
  PlayMove,
  ResumeSession,
  RevealSolution,
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

export type SessionView = {
  sessionId: string
  mode: string
  status: string
  currentIndex: number
  total: number
  current?: unknown
  summary?: unknown
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
  playMove(sessionId: string, uci: string): Promise<unknown>
  useHint(sessionId: string): Promise<unknown>
  revealSolution(sessionId: string): Promise<unknown>
  pauseSession(sessionId: string): Promise<void>
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
  playMove: PlayMove,
  useHint: UseHint,
  revealSolution: RevealSolution,
  pauseSession: PauseSession,
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
  playMove: async () => ({}),
  useHint: async () => ({}),
  revealSolution: async () => ({}),
  pauseSession: async () => {},
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
