import { waitFor } from '@testing-library/svelte'
import type { ImportInspection, ImportProgress, ImportResult } from './api'
import { createImportSession, type ImportSessionState } from './import-session'
import { fakeAPI } from '../test-fakes'

const embeddedInspection: ImportInspection = {
  path: '/normalized/club.pgn',
  filename: 'club.pgn',
  format: 'tactical-pgn',
  sourceId: 'club-tactics',
  sourceIdOrigin: 'embedded',
  sourceName: 'Club tactics',
  url: 'https://example.test/club',
  attribution: 'Club authors',
  replacesExisting: false
}

const emptyReport = () => ({ accepted: 0, duplicates: 0, rejected: 0, examples: [] })

function observe(session: ReturnType<typeof createImportSession>): {
  state: () => ImportSessionState
  stop: () => void
} {
  let current: ImportSessionState
  const stop = session.subscribe((value) => { current = value })
  return { state: () => current, stop }
}

test('starts in one explicit lifecycle phase without overlapping boolean guards', () => {
  const session = createImportSession(() => fakeAPI())
  const observed = observe(session)

  expect(observed.state().phase).toBe('idle')
  expect('busy' in observed.state()).toBe(false)
  expect('running' in observed.state()).toBe(false)

  observed.stop()
})

test('chooses then inspects a file and stores only the authoritative inspection', async () => {
  const calls: string[] = []
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile: async () => {
      calls.push('choose')
      return '/chosen/../club.pgn'
    },
    inspectPuzzleImport: async (path) => {
      calls.push(`inspect:${path}`)
      return embeddedInspection
    }
  }))
  const observed = observe(session)

  await session.selectFile()

  expect(calls).toEqual(['choose', 'inspect:/chosen/../club.pgn'])
  expect(observed.state()).toMatchObject({
    phase: 'ready',
    inspection: embeddedInspection,
    result: null,
    error: ''
  })
  observed.stop()
})

test('clears stale selection and result while inspecting a newly chosen file', async () => {
  let finishedListener: (result: ImportResult) => void = () => {}
  let resolveSecondInspection: (inspection: ImportInspection) => void = () => {}
  const secondInspection = new Promise<ImportInspection>((resolve) => {
    resolveSecondInspection = resolve
  })
  const chosen = ['/chosen/club.pgn', '/chosen/new.json']
  let inspections = 0
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile: async () => chosen.shift() ?? '',
    inspectPuzzleImport: async () => ++inspections === 1 ? embeddedInspection : secondInspection,
    getImportResult: async (jobId) => ({
      jobId,
      status: 'running',
      progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 50 },
      report: emptyReport()
    }),
    onImportFinished: (listener) => {
      finishedListener = listener
      return () => {}
    }
  }))
  const observed = observe(session)
  const disconnect = session.connect()

  await session.selectFile()
  await session.start()
  finishedListener({
    jobId: 'job-1',
    status: 'succeeded',
    progress: { phase: 'activating', rowsRead: 10, bytesRead: 50, totalBytes: 50 },
    report: { accepted: 10, duplicates: 0, rejected: 0, examples: [] }
  })
  expect(observed.state().result?.status).toBe('succeeded')

  const selecting = session.selectFile()
  await waitFor(() => expect(observed.state()).toMatchObject({
    phase: 'inspecting',
    jobId: '',
    inspection: null,
    result: null
  }))

  resolveSecondInspection({
    path: '/normalized/new.json',
    filename: 'new.json',
    format: 'canonical-json',
    sourceId: 'new-source',
    sourceIdOrigin: 'embedded',
    replacesExisting: false
  })
  await selecting
  expect(observed.state().inspection?.path).toBe('/normalized/new.json')
  disconnect()
  observed.stop()
})

test('treats chooser cancellation as no error and preserves the current inspection', async () => {
  const chosen = ['/chosen/club.pgn', '']
  const inspectPuzzleImport = vi.fn(async () => embeddedInspection)
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile: async () => chosen.shift() ?? '',
    inspectPuzzleImport
  }))
  const observed = observe(session)

  await session.selectFile()
  await session.selectFile()

  expect(inspectPuzzleImport).toHaveBeenCalledOnce()
  expect(observed.state()).toMatchObject({
    phase: 'ready',
    inspection: embeddedInspection,
    error: ''
  })
  observed.stop()
})

test('guards an overlapping selection with one authoritative lifecycle phase', async () => {
  let resolveFirstInspection: (inspection: ImportInspection) => void = () => {}
  const firstInspection = new Promise<ImportInspection>((resolve) => {
    resolveFirstInspection = resolve
  })
  const choosePuzzleImportFile = vi.fn()
    .mockResolvedValueOnce('/chosen/first.pgn')
    .mockResolvedValueOnce('/chosen/second.json')
  const inspectPuzzleImport = vi.fn(async (path: string) => path.endsWith('first.pgn')
    ? firstInspection
    : {
        path: '/normalized/second.json',
        filename: 'second.json',
        format: 'canonical-json' as const,
        sourceId: 'second-source',
        sourceIdOrigin: 'embedded' as const,
        replacesExisting: false
      })
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile,
    inspectPuzzleImport
  }))
  const observed = observe(session)

  const selecting = session.selectFile()
  await waitFor(() => expect(inspectPuzzleImport).toHaveBeenCalledWith('/chosen/first.pgn'))
  await session.selectFile()
  const phaseWhileFirstInspectionWasPending = observed.state().phase

  resolveFirstInspection(embeddedInspection)
  await selecting

  expect(choosePuzzleImportFile).toHaveBeenCalledOnce()
  expect(inspectPuzzleImport).toHaveBeenCalledOnce()
  expect(phaseWhileFirstInspectionWasPending).toBe('inspecting')
  expect(observed.state()).toMatchObject({
    phase: 'ready',
    inspection: embeddedInspection,
    error: ''
  })
  observed.stop()
})

test('does not start a stale inspection while another file selection is pending', async () => {
  let resolveChooser: (path: string) => void = () => {}
  const pendingChooser = new Promise<string>((resolve) => { resolveChooser = resolve })
  const choosePuzzleImportFile = vi.fn()
    .mockResolvedValueOnce('/chosen/club.pgn')
    .mockImplementationOnce(async () => pendingChooser)
  const startPuzzleImport = vi.fn(async () => 'job-1')
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile,
    inspectPuzzleImport: async () => embeddedInspection,
    startPuzzleImport
  }))
  const observed = observe(session)

  await session.selectFile()
  const selecting = session.selectFile()
  await waitFor(() => expect(observed.state().phase).toBe('selecting'))
  await session.start()
  const phaseWhileChooserWasPending = observed.state().phase

  resolveChooser('')
  await selecting

  expect(startPuzzleImport).not.toHaveBeenCalled()
  expect(phaseWhileChooserWasPending).toBe('selecting')
  expect(observed.state()).toMatchObject({
    phase: 'ready',
    inspection: embeddedInspection,
    error: ''
  })
  observed.stop()
})

test('surfaces inspection errors without retaining a stale selection', async () => {
  const chosen = ['/chosen/club.pgn', '/chosen/broken.txt']
  let inspections = 0
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile: async () => chosen.shift() ?? '',
    inspectPuzzleImport: async () => {
      if (++inspections === 1) return embeddedInspection
      throw new Error('unsupported puzzle collection')
    }
  }))
  const observed = observe(session)

  await session.selectFile()
  await session.selectFile()

  expect(observed.state()).toMatchObject({
    phase: 'idle',
    inspection: null,
    result: null,
    error: 'unsupported puzzle collection'
  })
  observed.stop()
})

test('starts only after inspection and passes the confirmed inspection once', async () => {
  const startPuzzleImport = vi.fn(async () => 'job-1')
  const session = createImportSession(() => fakeAPI({
    inspectPuzzleImport: async () => embeddedInspection,
    startPuzzleImport,
    getImportResult: async (jobId) => ({
      jobId,
      status: 'running',
      progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 500 },
      report: emptyReport()
    })
  }))
  const observed = observe(session)

  await session.start()
  expect(startPuzzleImport).not.toHaveBeenCalled()

  await session.selectFile()
  await session.start()

  expect(startPuzzleImport).toHaveBeenCalledOnce()
  expect(startPuzzleImport).toHaveBeenCalledWith(embeddedInspection)
  expect(observed.state()).toMatchObject({ phase: 'running', jobId: 'job-1' })
  observed.stop()
})

test('guards concurrent starts so a losing call cannot clobber the accepted job', async () => {
  let resolveAcceptedStart: (jobId: string) => void = () => {}
  const acceptedStart = new Promise<string>((resolve) => { resolveAcceptedStart = resolve })
  const startPuzzleImport = vi.fn(async () => {
    if (startPuzzleImport.mock.calls.length === 1) return acceptedStart
    throw new Error('puzzle import is already running')
  })
  const session = createImportSession(() => fakeAPI({
    inspectPuzzleImport: async () => embeddedInspection,
    startPuzzleImport,
    getImportResult: async (jobId) => ({
      jobId,
      status: 'running',
      progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 500 },
      report: emptyReport()
    })
  }))
  const observed = observe(session)

  await session.selectFile()
  const starting = session.start()
  await waitFor(() => expect(startPuzzleImport).toHaveBeenCalledOnce())
  await session.start()
  const stateWhileAcceptedStartWasPending = observed.state()

  resolveAcceptedStart('accepted-job')
  await starting

  expect(startPuzzleImport).toHaveBeenCalledOnce()
  expect(stateWhileAcceptedStartWasPending).toMatchObject({ phase: 'starting', error: '' })
  expect(observed.state()).toMatchObject({
    phase: 'running',
    jobId: 'accepted-job',
    error: ''
  })
  observed.stop()
})

test('keeps monotonic phases and independent counter maxima when a stale snapshot resolves later', async () => {
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
  const observed = observe(session)
  const disconnect = session.connect()

  await session.selectFile()
  const starting = session.start()
  await waitFor(() => expect(observed.state().jobId).toBe('job-1'))
  progressListener({
    jobId: 'job-1', phase: 'sealing', rowsRead: 100, bytesRead: 1_200, totalBytes: 2_000
  })
  resolveSnapshot({
    jobId: 'job-1',
    status: 'running',
    progress: { phase: 'parsing', rowsRead: 90, bytesRead: 1_500, totalBytes: 1_900 },
    report: emptyReport()
  })
  await starting

  expect(observed.state().progress).toEqual({
    jobId: 'job-1', phase: 'sealing', rowsRead: 100, bytesRead: 1_500, totalBytes: 2_000
  })
  disconnect()
  observed.stop()
})

test('resets progress for a new job and never lets a running snapshot replace a terminal result', async () => {
  let progressListener: (progress: ImportProgress) => void = () => {}
  let finishedListener: (result: ImportResult) => void = () => {}
  let resolveFirstSnapshot: (result: ImportResult) => void = () => {}
  const firstSnapshot = new Promise<ImportResult>((resolve) => { resolveFirstSnapshot = resolve })
  const jobIds = ['job-1', 'job-2']
  const session = createImportSession(() => fakeAPI({
    startPuzzleImport: async () => jobIds.shift() ?? 'unexpected-job',
    getImportResult: async (jobId) => jobId === 'job-1'
      ? firstSnapshot
      : {
          jobId,
          status: 'running',
          progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 2_500 },
          report: emptyReport()
        },
    onImportProgress: (listener) => {
      progressListener = listener
      return () => {}
    },
    onImportFinished: (listener) => {
      finishedListener = listener
      return () => {}
    }
  }))
  const observed = observe(session)
  const disconnect = session.connect()

  await session.selectFile()
  const firstStart = session.start()
  await waitFor(() => expect(observed.state().jobId).toBe('job-1'))
  progressListener({
    jobId: 'job-1', phase: 'activating', rowsRead: 110, bytesRead: 2_000, totalBytes: 2_000
  })
  finishedListener({
    jobId: 'job-1',
    status: 'succeeded',
    progress: { phase: 'activating', rowsRead: 110, bytesRead: 2_000, totalBytes: 2_000 },
    report: { accepted: 100, duplicates: 0, rejected: 0, examples: [] }
  })
  resolveFirstSnapshot({
    jobId: 'job-1',
    status: 'running',
    progress: { phase: 'parsing', rowsRead: 50, bytesRead: 900, totalBytes: 2_000 },
    report: emptyReport()
  })
  await firstStart

  expect(observed.state().result?.status).toBe('succeeded')
  expect(observed.state().phase).toBe('finished')

  await session.start()
  expect(observed.state().progress).toEqual({
    jobId: 'job-2', phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 2_500
  })
  disconnect()
  observed.stop()
})
