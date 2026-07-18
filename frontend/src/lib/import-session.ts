import { writable, type Readable } from 'svelte/store'
import type { ImportInspection, ImportProgress, ImportResult, NormalAPI } from './api'

export type ImportSessionState = {
  phase: ImportSessionPhase
  inspection: ImportInspection | null
  jobId: string
  progress: ImportProgress
  result: ImportResult | null
  error: string
}

export type ImportSessionPhase =
  | 'idle'
  | 'selecting'
  | 'inspecting'
  | 'ready'
  | 'starting'
  | 'running'
  | 'finished'

export function canSelectImportFile(phase: ImportSessionPhase): boolean {
  return phase === 'idle' || phase === 'ready' || phase === 'finished'
}

export function canStartImport(phase: ImportSessionPhase): boolean {
  return phase === 'ready' || phase === 'finished'
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

const emptyProgress = (): ImportProgress => ({
  jobId: '',
  phase: 'detecting',
  rowsRead: 0,
  bytesRead: 0,
  totalBytes: 0
})

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

export function createImportSession(api: () => NormalAPI): ImportSession {
  const state = writable<ImportSessionState>({
    phase: 'idle',
    inspection: null,
    jobId: '',
    progress: emptyProgress(),
    result: null,
    error: ''
  })
  let current: ImportSessionState
  state.subscribe((value) => { current = value })

  function applyResult(result: ImportResult): void {
    if (result.jobId !== current.jobId) return
    if (current.result && result.status === 'running') return
    state.update((value) => ({
      ...value,
      phase: result.status === 'running' ? 'running' : 'finished',
      progress: mergeProgress(value.progress, result.jobId, result.progress),
      result: result.status === 'running' ? null : result,
      error: result.status === 'failed' ? (result.error ?? 'Import failed') : ''
    }))
  }

  async function refresh(): Promise<void> {
    const jobId = current.jobId
    if (!jobId) return
    try {
      applyResult(await api().getImportResult(jobId))
    } catch (cause) {
      if (jobId !== current.jobId) return
      state.update((value) => ({
        ...value,
        error: cause instanceof Error ? cause.message : String(cause)
      }))
    }
  }

  return {
    subscribe: state.subscribe,
    connect() {
      const stopProgress = api().onImportProgress((progress) => {
        if (progress.jobId !== current.jobId || current.phase !== 'running') return
        state.update((value) => ({
          ...value,
          progress: mergeProgress(value.progress, progress.jobId, progress)
        }))
      })
      const stopFinished = api().onImportFinished(applyResult)
      return () => {
        stopProgress()
        stopFinished()
      }
    },
    async selectFile() {
      if (!canSelectImportFile(current.phase)) return
      const returnPhase = current.phase
      let inspecting = false
      state.update((value) => ({ ...value, phase: 'selecting', error: '' }))
      try {
        const path = await api().choosePuzzleImportFile()
        if (!path) {
          state.update((value) => ({ ...value, phase: returnPhase }))
          return
        }
        inspecting = true
        state.update((value) => ({
          ...value,
          phase: 'inspecting',
          inspection: null,
          jobId: '',
          progress: emptyProgress(),
          result: null
        }))
        const inspection = await api().inspectPuzzleImport(path)
        state.update((value) => ({
          ...value,
          phase: 'ready',
          inspection,
          result: null
        }))
      } catch (cause) {
        state.update((value) => ({
          ...value,
          phase: inspecting ? 'idle' : returnPhase,
          inspection: inspecting ? null : value.inspection,
          error: cause instanceof Error ? cause.message : String(cause)
        }))
      }
    },
    async start() {
      const inspection = current.inspection
      if (!inspection || !canStartImport(current.phase)) return
      state.update((value) => ({
        ...value,
        phase: 'starting',
        jobId: '',
        error: '',
        result: null,
        progress: emptyProgress()
      }))
      try {
        const jobId = await api().startPuzzleImport(inspection)
        state.update((value) => ({
          ...value,
          phase: 'running',
          jobId,
          progress: { jobId, phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 0 }
        }))
        await refresh()
      } catch (cause) {
        state.update((value) => ({
          ...value,
          phase: 'ready',
          error: cause instanceof Error ? cause.message : String(cause)
        }))
      }
    },
    async cancel() {
      if (!current.jobId || current.phase !== 'running') return
      try {
        await api().cancelImport(current.jobId)
        await refresh()
      } catch (cause) {
        state.update((value) => ({
          ...value,
          error: cause instanceof Error ? cause.message : String(cause)
        }))
      }
    },
    refresh
  }
}
