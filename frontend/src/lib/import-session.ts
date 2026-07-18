import { writable, type Readable } from 'svelte/store'
import type { ImportInspection, ImportProgress, ImportResult, NormalAPI } from './api'

export type ImportSessionState = {
  path: string
  inspection: ImportInspection | null
  jobId: string
  running: boolean
  busy: boolean
  progress: ImportProgress
  result: ImportResult | null
  error: string
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
    path: '',
    inspection: null,
    jobId: '',
    running: false,
    busy: false,
    progress: emptyProgress(),
    result: null,
    error: ''
  })
  let current: ImportSessionState
  let operationInFlight = false
  state.subscribe((value) => { current = value })

  function applyResult(result: ImportResult): void {
    if (result.jobId !== current.jobId) return
    if (current.result && result.status === 'running') return
    state.update((value) => ({
      ...value,
      running: result.status === 'running',
      progress: result.progress
        ? mergeProgress(value.progress, result.jobId, result.progress)
        : value.progress,
      result: result.status === 'running' ? null : result,
      error: result.status === 'failed' ? (result.error ?? 'Import failed') : ''
    }))
  }

  return {
    subscribe: state.subscribe,
    connect() {
      const stopProgress = api().onImportProgress((progress) => {
        if (progress.jobId !== current.jobId || !current.running) return
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
      if (operationInFlight || current.running) return
      operationInFlight = true
      state.update((value) => ({ ...value, busy: true, error: '' }))
      try {
        const path = await api().choosePuzzleImportFile()
        if (!path) return
        state.update((value) => ({
          ...value,
          path: '',
          inspection: null,
          jobId: '',
          running: false,
          progress: emptyProgress(),
          result: null
        }))
        const inspection = await api().inspectPuzzleImport(path)
        state.update((value) => ({
          ...value,
          path: inspection.path,
          inspection,
          result: null
        }))
      } catch (cause) {
        state.update((value) => ({
          ...value,
          error: cause instanceof Error ? cause.message : String(cause)
        }))
      } finally {
        operationInFlight = false
        state.update((value) => ({ ...value, busy: false }))
      }
    },
    async start() {
      const inspection = current.inspection
      if (operationInFlight || !inspection || current.running) return
      operationInFlight = true
      state.update((value) => ({
        ...value,
        jobId: '',
        busy: true,
        error: '',
        result: null,
        progress: emptyProgress()
      }))
      try {
        const jobId = await api().startPuzzleImport(inspection.path)
        state.update((value) => ({
          ...value,
          jobId,
          running: true,
          progress: { jobId, phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 0 }
        }))
        await this.refresh()
      } catch (cause) {
        state.update((value) => ({
          ...value,
          running: false,
          error: cause instanceof Error ? cause.message : String(cause)
        }))
      } finally {
        operationInFlight = false
        state.update((value) => ({ ...value, busy: false }))
      }
    },
    async cancel() {
      if (!current.jobId || !current.running) return
      try {
        await api().cancelImport(current.jobId)
        await this.refresh()
      } catch (cause) {
        state.update((value) => ({
          ...value,
          error: cause instanceof Error ? cause.message : String(cause)
        }))
      }
    },
    async refresh() {
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
  }
}
