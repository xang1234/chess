import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { vi } from 'vitest'
import ImportPanel from './ImportPanel.svelte'
import type { ImportInspection, ImportProgress, ImportResult } from '../../lib/api'
import { createImportSession } from '../../lib/import-session'
import { fakeAPI } from '../../test-fakes'

const embeddedInspection: ImportInspection = {
  path: '/Users/family/Puzzles/club-tactics.pgn',
  filename: 'club-tactics.pgn',
  format: 'tactical-pgn',
  sourceId: 'club-tactics',
  sourceIdOrigin: 'embedded',
  replacesExisting: true
}

test('shows the source identity first with format, origin, file details, and replacement warning', async () => {
  const session = createImportSession(() => fakeAPI({
    inspectPuzzleImport: async () => embeddedInspection
  }))
  render(ImportPanel, { session })

  await fireEvent.click(screen.getByRole('button', { name: 'Choose puzzle collection' }))

  const card = await screen.findByLabelText('Selected puzzle collection')
  const sourceId = screen.getByText('club-tactics')
  expect(sourceId.tagName).toBe('STRONG')
  expect(card.textContent).toContain('Tactical PGN')
  expect(card.textContent).toContain('Embedded source ID')
  expect(card.textContent).toContain('club-tactics.pgn')
  expect(card.textContent).toContain('/Users/family/Puzzles/club-tactics.pgn')
  expect(screen.getByText(
    'This import will replace the active club-tactics collection'
  )).toBeInTheDocument()

  const text = card.textContent ?? ''
  expect(text.indexOf('club-tactics')).toBeLessThan(text.indexOf('Tactical PGN'))
  expect(text.indexOf('Tactical PGN')).toBeLessThan(text.indexOf('club-tactics.pgn'))
})

test('labels a path-derived identity as the fallback source ID', async () => {
  const pathInspection: ImportInspection = {
    path: '/Users/family/Puzzles/pin-lines.txt',
    filename: 'pin-lines.txt',
    format: 'linear-fen-uci',
    sourceId: '/Users/family/Puzzles/pin-lines.txt',
    sourceIdOrigin: 'path',
    replacesExisting: false
  }
  const session = createImportSession(() => fakeAPI({
    inspectPuzzleImport: async () => pathInspection
  }))
  render(ImportPanel, { session })

  await fireEvent.click(screen.getByRole('button', { name: 'Choose puzzle collection' }))

  expect(await screen.findByText('Fallback source ID (file path)')).toBeInTheDocument()
  expect(screen.getByText('Linear FEN/UCI')).toBeInTheDocument()
})

test('keeps import disabled until authoritative inspection succeeds', async () => {
  let resolveInspection: (inspection: ImportInspection) => void = () => {}
  const pendingInspection = new Promise<ImportInspection>((resolve) => {
    resolveInspection = resolve
  })
  const session = createImportSession(() => fakeAPI({
    inspectPuzzleImport: async () => pendingInspection
  }))
  render(ImportPanel, { session })

  const importButton = screen.getByRole('button', { name: 'Import puzzles' })
  expect(importButton).toBeDisabled()
  await fireEvent.click(screen.getByRole('button', { name: 'Choose puzzle collection' }))
  expect(importButton).toBeDisabled()

  resolveInspection(embeddedInspection)
  await waitFor(() => expect(importButton).toBeEnabled())
})

test('shows phase-aware progress using ordinary source bytes', async () => {
  let progressListener: (progress: ImportProgress) => void = () => {}
  const session = createImportSession(() => fakeAPI({
    inspectPuzzleImport: async () => embeddedInspection,
    onImportProgress: (listener) => {
      progressListener = listener
      return () => {}
    }
  }))
  const disconnect = session.connect()
  render(ImportPanel, { session })

  await session.selectFile()
  await fireEvent.click(screen.getByRole('button', { name: 'Import puzzles' }))
  progressListener({
    jobId: 'job-1', phase: 'parsing', rowsRead: 10_000, bytesRead: 2_048, totalBytes: 4_096
  })

  await waitFor(() => expect(screen.getByText('Reading puzzles')).toBeInTheDocument())
  expect(screen.getByText('10,000 rows read')).toBeInTheDocument()
  expect(screen.getByText('2,048 of 4,096 bytes')).toBeInTheDocument()
  expect(screen.queryByText(/compressed bytes/i)).not.toBeInTheDocument()

  progressListener({
    jobId: 'job-1', phase: 'sealing', rowsRead: 10_000, bytesRead: 4_096, totalBytes: 4_096
  })
  await waitFor(() => expect(screen.getByText('Finalizing collection')).toBeInTheDocument())

  progressListener({
    jobId: 'job-1', phase: 'activating', rowsRead: 10_000, bytesRead: 4_096, totalBytes: 4_096
  })
  await waitFor(() => expect(screen.getByText('Activating collection')).toBeInTheDocument())
  expect(screen.getByRole('button', { name: 'Cancel import' })).toBeInTheDocument()
  disconnect()
})

test('renders only backend-provided rejection examples as plain text', async () => {
  let finishedListener: (result: ImportResult) => void = () => {}
  const session = createImportSession(() => fakeAPI({
    inspectPuzzleImport: async () => embeddedInspection,
    onImportFinished: (listener) => {
      finishedListener = listener
      return () => {}
    }
  }))
  const disconnect = session.connect()
  render(ImportPanel, { session })

  await session.selectFile()
  await fireEvent.click(screen.getByRole('button', { name: 'Import puzzles' }))
  finishedListener({
    jobId: 'job-1',
    status: 'succeeded',
    progress: { phase: 'activating', rowsRead: 10_000, bytesRead: 4_096, totalBytes: 4_096 },
    report: {
      accepted: 9_800,
      duplicates: 150,
      rejected: 2,
      examples: [
        { ordinal: 17, reason: 'illegal move e2e5' },
        { ordinal: 23, reason: '<strong>untrusted source text</strong>' }
      ]
    }
  })

  await waitFor(() => expect(screen.getByText('9,800 accepted')).toBeInTheDocument())
  expect(screen.getByText('150 duplicates')).toBeInTheDocument()
  expect(screen.getByText('2 rejected')).toBeInTheDocument()
  const examples = screen.getAllByRole('listitem')
  expect(examples).toHaveLength(2)
  expect(examples[0]).toHaveTextContent('17: illegal move e2e5')
  expect(examples[1]).toHaveTextContent('23: <strong>untrusted source text</strong>')
  expect(examples[1].querySelector('strong')).toBeNull()
  disconnect()
})

test('treats closing the file chooser as cancellation, not an error', async () => {
  const inspectPuzzleImport = vi.fn(async () => embeddedInspection)
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile: async () => '',
    inspectPuzzleImport
  }))
  render(ImportPanel, { session })

  await fireEvent.click(screen.getByRole('button', { name: 'Choose puzzle collection' }))

  expect(inspectPuzzleImport).not.toHaveBeenCalled()
  expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  expect(screen.getByText('No collection selected')).toBeInTheDocument()
})
