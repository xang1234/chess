import { writable, type Readable } from 'svelte/store'
import type { AppAPI, ImportProgress, ImportResult } from './api'

export type ImportSessionState = {
  path: string
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

const emptyProgress = (): ImportProgress => ({ jobId: '', rowsRead: 0, bytesRead: 0 })

function mergeProgress(
  currentProgress: ImportProgress,
  jobId: string,
  nextProgress: { rowsRead: number; bytesRead: number }
): ImportProgress {
  if (currentProgress.jobId !== jobId) return { jobId, ...nextProgress }
  return {
    jobId,
    rowsRead: Math.max(currentProgress.rowsRead, nextProgress.rowsRead),
    bytesRead: Math.max(currentProgress.bytesRead, nextProgress.bytesRead)
  }
}

export function createImportSession(api: () => AppAPI): ImportSession {
  const state = writable<ImportSessionState>({
    path: '',
    jobId: '',
    running: false,
    busy: false,
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
      state.update((value) => ({ ...value, busy: true, error: '' }))
      try {
        const path = await api().choosePuzzleImportFile()
        if (path) state.update((value) => ({ ...value, path }))
      } catch (cause) {
        state.update((value) => ({
          ...value,
          error: cause instanceof Error ? cause.message : String(cause)
        }))
      } finally {
        state.update((value) => ({ ...value, busy: false }))
      }
    },
    async start() {
      if (!current.path || current.running) return
      state.update((value) => ({
        ...value,
        busy: true,
        error: '',
        result: null,
        progress: emptyProgress()
      }))
      try {
        const jobId = await api().startLichessImport(current.path)
        state.update((value) => ({
          ...value,
          jobId,
          running: true,
          progress: { jobId, rowsRead: 0, bytesRead: 0 }
        }))
        await this.refresh()
      } catch (cause) {
        state.update((value) => ({
          ...value,
          running: false,
          error: cause instanceof Error ? cause.message : String(cause)
        }))
      } finally {
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
