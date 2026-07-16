import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { vi } from 'vitest'
import ImportPanel from './ImportPanel.svelte'
import type { ImportProgress, ImportResult } from '../../lib/api'
import { createImportSession } from '../../lib/import-session'
import { fakeAPI } from '../../test-fakes'

test('shows running progress and the completed import report', async () => {
  let progressListener: (progress: ImportProgress) => void = () => {}
  let finishedListener: (result: ImportResult) => void = () => {}
  const session = createImportSession(() => fakeAPI({
    onImportProgress: (listener) => {
      progressListener = listener
      return () => {}
    },
    onImportFinished: (listener) => {
      finishedListener = listener
      return () => {}
    }
  }))
  const disconnect = session.connect()
  render(ImportPanel, { session })

  await session.selectFile()
  await fireEvent.click(screen.getByRole('button', { name: 'Import puzzles' }))
  progressListener({ jobId: 'job-1', rowsRead: 10000, bytesRead: 2048 })

  await waitFor(() => expect(screen.getByText('10,000 rows read')).toBeInTheDocument())
  expect(screen.getByRole('button', { name: 'Cancel import' })).toBeInTheDocument()

  finishedListener({
    jobId: 'job-1',
    status: 'succeeded',
    report: { accepted: 9800, duplicates: 150, rejected: 50 }
  })
  await waitFor(() => expect(screen.getByText('9,800 accepted')).toBeInTheDocument())
  expect(screen.getByText('150 duplicates')).toBeInTheDocument()
  expect(screen.getByText('50 rejected')).toBeInTheDocument()
  disconnect()
})

test('selects a compressed puzzle database without typing a path', async () => {
  const startLichessImport = vi.fn(async () => 'job-1')
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile: async () => '/Users/family/Downloads/lichess_db_puzzle.csv.zst',
    startLichessImport
  }))
  render(ImportPanel, { session })

  await fireEvent.click(screen.getByRole('button', { name: 'Choose puzzle database' }))

  expect(screen.getByText('lichess_db_puzzle.csv.zst')).toBeInTheDocument()
  expect(screen.getByText('/Users/family/Downloads/lichess_db_puzzle.csv.zst')).toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', { name: 'Import puzzles' }))
  expect(startLichessImport).toHaveBeenCalledWith('/Users/family/Downloads/lichess_db_puzzle.csv.zst')
})

test('treats closing the file chooser as cancellation, not an error', async () => {
  const session = createImportSession(() => fakeAPI({ choosePuzzleImportFile: async () => '' }))
  render(ImportPanel, { session })

  await fireEvent.click(screen.getByRole('button', { name: 'Choose puzzle database' }))

  expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  expect(screen.getByText('No file selected')).toBeInTheDocument()
})

test('does not let a stale running snapshot overwrite a terminal event', async () => {
  let finishedListener: (result: ImportResult) => void = () => {}
  let resolveSnapshot: (result: ImportResult) => void = () => {}
  const snapshot = new Promise<ImportResult>((resolve) => { resolveSnapshot = resolve })
  const session = createImportSession(() => fakeAPI({
    getImportResult: async () => snapshot,
    onImportFinished: (listener) => {
      finishedListener = listener
      return () => {}
    }
  }))
  const disconnect = session.connect()
  render(ImportPanel, { session })

  await session.selectFile()
  await fireEvent.click(screen.getByRole('button', { name: 'Import puzzles' }))
  await waitFor(() => expect(screen.getByRole('button', { name: 'Cancel import' })).toBeInTheDocument())
  finishedListener({
    jobId: 'job-1', status: 'succeeded',
    report: { accepted: 10, duplicates: 0, rejected: 0 }
  })
  resolveSnapshot({
    jobId: 'job-1', status: 'running',
    progress: { rowsRead: 5, bytesRead: 100 },
    report: { accepted: 0, duplicates: 0, rejected: 0 }
  })

  await waitFor(() => expect(screen.getByText('10 accepted')).toBeInTheDocument())
  expect(screen.queryByRole('button', { name: 'Cancel import' })).not.toBeInTheDocument()
  disconnect()
})
