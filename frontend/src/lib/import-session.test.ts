import { waitFor } from '@testing-library/svelte'
import type { ImportProgress, ImportResult } from './api'
import { createImportSession, type ImportSessionState } from './import-session'
import { fakeAPI } from '../test-fakes'

test('keeps newer event progress when a lower running snapshot resolves later', async () => {
  let progressListener: (progress: ImportProgress) => void = () => {}
  let resolveSnapshot: (result: ImportResult) => void = () => {}
  const snapshot = new Promise<ImportResult>((resolve) => { resolveSnapshot = resolve })
  const session = createImportSession(() => fakeAPI({
    getImportResult: async () => snapshot,
    onImportProgress: (listener) => {
      progressListener = listener
      return () => {}
    }
  }))
  let state: ImportSessionState
  const stop = session.subscribe((value) => { state = value })
  const disconnect = session.connect()

  await session.selectFile()
  const starting = session.start()
  await waitFor(() => expect(state.jobId).toBe('job-1'))
  progressListener({ jobId: 'job-1', rowsRead: 100, bytesRead: 1_200 })
  resolveSnapshot({
    jobId: 'job-1',
    status: 'running',
    progress: { rowsRead: 90, bytesRead: 1_100 },
    report: { accepted: 0, duplicates: 0, rejected: 0 }
  })
  await starting

  expect(state.progress).toEqual({ jobId: 'job-1', rowsRead: 100, bytesRead: 1_200 })
  disconnect()
  stop()
})

test('merges progress counters independently and resets them for a new job', async () => {
  let progressListener: (progress: ImportProgress) => void = () => {}
  let finishedListener: (result: ImportResult) => void = () => {}
  const jobIds = ['job-1', 'job-2']
  const session = createImportSession(() => fakeAPI({
    startLichessImport: async () => jobIds.shift() ?? 'unexpected-job',
    getImportResult: async (jobId) => ({
      jobId,
      status: 'running',
      progress: { rowsRead: 0, bytesRead: 0 },
      report: { accepted: 0, duplicates: 0, rejected: 0 }
    }),
    onImportProgress: (listener) => {
      progressListener = listener
      return () => {}
    },
    onImportFinished: (listener) => {
      finishedListener = listener
      return () => {}
    }
  }))
  let state: ImportSessionState
  const stopState = session.subscribe((value) => { state = value })
  const disconnect = session.connect()

  await session.selectFile()
  await session.start()
  progressListener({ jobId: 'job-1', rowsRead: 100, bytesRead: 1_000 })
  progressListener({ jobId: 'job-1', rowsRead: 90, bytesRead: 1_200 })
  progressListener({ jobId: 'job-1', rowsRead: 110, bytesRead: 1_100 })
  expect(state.progress).toEqual({ jobId: 'job-1', rowsRead: 110, bytesRead: 1_200 })

  finishedListener({
    jobId: 'job-1',
    status: 'succeeded',
    report: { accepted: 100, duplicates: 0, rejected: 0 }
  })
  await session.start()

  expect(state.progress).toEqual({ jobId: 'job-2', rowsRead: 0, bytesRead: 0 })
  disconnect()
  stopState()
})
