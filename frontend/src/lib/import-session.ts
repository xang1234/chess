import { writable, type Readable } from 'svelte/store'
import type {
  ImportInspection,
  ImportKind,
  ImportProgress,
  ImportResult,
  NormalAPI
} from './api'

export type IdleImportState = {
  phase: 'idle'
  error: string
}

export type ReadyImportState = {
  phase: 'ready'
  inspection: ImportInspection
  error: string
}

export type TerminalImportResult = Omit<ImportResult, 'status'> & {
  status: 'succeeded' | 'failed' | 'cancelled'
}

export type FinishedImportState = {
  phase: 'finished'
  inspection: ImportInspection
  jobId: string
  progress: ImportProgress
  result: TerminalImportResult
  error: string
}

export type SelectableImportState =
  | IdleImportState
  | ReadyImportState
  | FinishedImportState

export type ImportSessionState =
  | SelectableImportState
  | {
      phase: 'selecting'
      previous: SelectableImportState
      error: string
    }
  | {
      phase: 'inspecting'
      path: string
      error: string
    }
  | {
      phase: 'starting'
      inspection: ImportInspection
      error: string
    }
  | {
      phase: 'running'
      inspection: ImportInspection
      jobId: string
      progress: ImportProgress
      error: string
    }

export function canSelectImportFile(
  state: ImportSessionState
): state is SelectableImportState {
  return state.phase === 'idle' || state.phase === 'ready' || state.phase === 'finished'
}

export function canStartImport(
  state: ImportSessionState
): state is ReadyImportState | FinishedImportState {
  return state.phase === 'ready' || state.phase === 'finished'
}

export function selectedImportInspection(state: ImportSessionState): ImportInspection | null {
  switch (state.phase) {
    case 'selecting':
      return selectedImportInspection(state.previous)
    case 'ready':
    case 'starting':
    case 'running':
    case 'finished':
      return state.inspection
    default:
      return null
  }
}

export type ImportSession = Readable<ImportSessionState> & {
  connect(): () => void
  selectFile(): Promise<void>
  start(): Promise<void>
  cancel(): Promise<void>
  refresh(): Promise<void>
}

type ProgressSnapshot = Omit<ImportProgress, 'jobId'>

const phaseRank: Record<ImportProgress['phase'], number> = {
  detecting: 0,
  parsing: 1,
  sealing: 2,
  activating: 3
}

function initialProgress(jobId: string): ImportProgress {
  return {
    jobId,
    phase: 'detecting',
    rowsRead: 0,
    bytesRead: 0,
    totalBytes: 0
  }
}

function mergeProgress(
  currentProgress: ImportProgress,
  jobId: string,
  nextProgress: ProgressSnapshot
): ImportProgress {
  if (currentProgress.jobId !== jobId) return { jobId, ...nextProgress }
  return {
    jobId,
    phase: phaseRank[nextProgress.phase] >= phaseRank[currentProgress.phase]
      ? nextProgress.phase
      : currentProgress.phase,
    rowsRead: Math.max(currentProgress.rowsRead, nextProgress.rowsRead),
    bytesRead: Math.max(currentProgress.bytesRead, nextProgress.bytesRead),
    totalBytes: Math.max(currentProgress.totalBytes, nextProgress.totalBytes)
  }
}

function messageFrom(cause: unknown): string {
  return cause instanceof Error ? cause.message : String(cause)
}

export function createImportSession(
  api: () => NormalAPI,
  kind: ImportKind = 'puzzle'
): ImportSession {
  const initial: ImportSessionState = { phase: 'idle', error: '' }
  const state = writable<ImportSessionState>(initial)
  let current: ImportSessionState = initial
  state.subscribe((value) => { current = value })

  function applyResult(result: ImportResult): void {
    if (current.phase !== 'running' || result.jobId !== current.jobId) return
    const active = current
    const progress = mergeProgress(active.progress, result.jobId, result.progress)
    if (result.status === 'running') {
      state.set({ ...active, progress })
      return
    }
    const terminalResult: TerminalImportResult = { ...result, status: result.status }
    state.set({
      phase: 'finished',
      inspection: active.inspection,
      jobId: active.jobId,
      progress,
      result: terminalResult,
      error: result.status === 'failed' ? (result.error ?? 'Import failed') : ''
    })
  }

  async function refresh(): Promise<void> {
    if (current.phase !== 'running') return
    const jobId = current.jobId
    try {
      applyResult(await api().getImportResult(jobId))
    } catch (cause) {
      if (current.phase !== 'running' || current.jobId !== jobId) return
      state.set({ ...current, error: messageFrom(cause) })
    }
  }

  return {
    subscribe: state.subscribe,
    connect() {
      const stopProgress = api().onImportProgress((progress) => {
        if (current.phase !== 'running' || progress.jobId !== current.jobId) return
        state.set({
          ...current,
          progress: mergeProgress(current.progress, progress.jobId, progress)
        })
      })
      const stopFinished = api().onImportFinished(applyResult)
      return () => {
        stopProgress()
        stopFinished()
      }
    },
    async selectFile() {
      if (!canSelectImportFile(current)) return
      const previous = current
      state.set({ phase: 'selecting', previous, error: '' })

      let path: string
      try {
        path = kind === 'course'
          ? await api().chooseOpeningCourseFile()
          : await api().choosePuzzleImportFile()
      } catch (cause) {
        state.set({ ...previous, error: messageFrom(cause) })
        return
      }
      if (!path) {
        state.set(previous)
        return
      }

      state.set({ phase: 'inspecting', path, error: '' })
      try {
        const inspection = kind === 'course'
          ? await api().inspectOpeningCourseImport(path)
          : await api().inspectPuzzleImport(path)
        state.set({ phase: 'ready', inspection, error: '' })
      } catch (cause) {
        state.set({ phase: 'idle', error: messageFrom(cause) })
      }
    },
    async start() {
      if (!canStartImport(current)) return
      const inspection = current.inspection
      state.set({ phase: 'starting', inspection, error: '' })
      try {
        const jobId = kind === 'course'
          ? await api().startOpeningCourseImport(inspection)
          : await api().startPuzzleImport(inspection)
        if (!jobId) {
          throw new Error(`${kind === 'course' ? 'Course' : 'Puzzle'} import returned an empty job ID`)
        }
        state.set({
          phase: 'running',
          inspection,
          jobId,
          progress: initialProgress(jobId),
          error: ''
        })
        await refresh()
      } catch (cause) {
        state.set({ phase: 'ready', inspection, error: messageFrom(cause) })
      }
    },
    async cancel() {
      if (current.phase !== 'running') return
      const jobId = current.jobId
      try {
        await api().cancelImport(jobId)
        await refresh()
      } catch (cause) {
        if (current.phase !== 'running' || current.jobId !== jobId) return
        state.set({ ...current, error: messageFrom(cause) })
      }
    },
    refresh
  }
}
