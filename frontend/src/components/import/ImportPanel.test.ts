import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import ImportPanel from './ImportPanel.svelte'
import type { ImportProgress, ImportResult } from '../../lib/api'
import { setAPIForTests } from '../../lib/api'
import { fakeAPI } from '../../test-fakes'

test('shows running progress and the completed import report', async () => {
  let progressListener: (progress: ImportProgress) => void = () => {}
  let finishedListener: (result: ImportResult) => void = () => {}
  setAPIForTests(fakeAPI({
    onImportProgress: (listener) => {
      progressListener = listener
      return () => {}
    },
    onImportFinished: (listener) => {
      finishedListener = listener
      return () => {}
    }
  }))
  render(ImportPanel)

  await fireEvent.input(screen.getByLabelText('Puzzle database file'), {
    target: { value: '/tmp/puzzles.csv.zst' }
  })
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
})
