import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import type { ImportInspection, ImportProgress, ImportResult } from '../../lib/api'
import { createImportSession } from '../../lib/import-session'
import { fakeAPI } from '../../test-fakes'
import ImportCard from './ImportCard.svelte'

const courseInspection: ImportInspection = {
  path: '/Users/family/Courses/italian.ctcourse',
  filename: 'italian.ctcourse',
  format: 'coursepack',
  formatLabel: 'Opening course',
  sourceId: 'italian-white',
  sourceIdOrigin: 'embedded',
  sourceName: '<strong>Italian Game for White</strong>',
  attribution: 'Private reference source',
  replacesExisting: true
}

test('imports a private course with course progress and structural totals', async () => {
  let progressListener: (progress: ImportProgress) => void = () => {}
  let finishedListener: (result: ImportResult) => void = () => {}
  const session = createImportSession(() => fakeAPI({
    inspectOpeningCourseImport: async () => courseInspection,
    onImportProgress: (listener) => {
      progressListener = listener
      return () => {}
    },
    onImportFinished: (listener) => {
      finishedListener = listener
      return () => {}
    }
  }), 'course')
  const disconnect = session.connect()
  render(ImportCard, {
    kind: 'course',
    session,
    heading: 'Import opening course',
    description: 'Private .ctcourse files stay on this Mac.',
    fileLabel: 'Opening course file',
    chooseLabel: 'Choose opening course',
    startLabel: 'Import course'
  })

  expect(screen.getByText('Private .ctcourse files stay on this Mac.')).toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', { name: 'Choose opening course' }))
  const selected = await screen.findByLabelText('Selected opening course')
  expect(selected).toHaveTextContent('italian-white')
  expect(selected).toHaveTextContent('<strong>Italian Game for White</strong>')
  expect(selected.querySelector('strong strong')).toBeNull()
  expect(screen.getByText(
    'This import will replace the active italian-white course'
  )).toBeInTheDocument()

  await fireEvent.click(screen.getByRole('button', { name: 'Import course' }))
  progressListener({
    jobId: 'course-job-1', phase: 'parsing', rowsRead: 28, bytesRead: 7_085, totalBytes: 7_085
  })
  await waitFor(() => expect(screen.getByText('Checking course records')).toBeInTheDocument())
  expect(screen.getByText('28 course records checked')).toBeInTheDocument()

  finishedListener({
    jobId: 'course-job-1',
    status: 'succeeded',
    progress: { phase: 'activating', rowsRead: 28, bytesRead: 7_085, totalBytes: 7_085 },
    report: {
      accepted: 1,
      duplicates: 0,
      rejected: 0,
      examples: [],
      counts: { chapters: 3, moves: 40, lessons: 8 }
    }
  })
  await waitFor(() => expect(screen.getByText('3 chapters')).toBeInTheDocument())
  expect(screen.getByText('40 moves')).toBeInTheDocument()
  expect(screen.getByText('8 lessons')).toBeInTheDocument()
  disconnect()
})
