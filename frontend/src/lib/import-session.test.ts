import { waitFor } from '@testing-library/svelte'
import type { ImportInspection, ImportProgress, ImportResult } from './api'
import {
  canSelectImportFile,
  canStartImport,
  createImportSession,
  selectedImportInspection,
  type ImportSessionState
} from './import-session'
import { fakeAPI } from '../test-fakes'

const embeddedInspection: ImportInspection = {
  path: '/normalized/club.pgn',
  filename: 'club.pgn',
  format: 'tactical-pgn',
  formatLabel: 'Tactical PGN',
  sourceId: 'club-tactics',
  sourceIdOrigin: 'embedded',
  sourceName: 'Club tactics',
  url: 'https://example.test/club',
  attribution: 'Club authors',
  replacesExisting: false
}

const jsonInspection: ImportInspection = {
  path: '/normalized/new.json',
  filename: 'new.json',
  format: 'canonical-json',
  formatLabel: 'Canonical JSON',
  sourceId: 'new-source',
  sourceIdOrigin: 'embedded',
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

function requirePhase<Phase extends ImportSessionState['phase']>(
  state: ImportSessionState,
  expected: Phase
): Extract<ImportSessionState, { phase: Phase }> {
  expect(state.phase).toBe(expected)
  return state as Extract<ImportSessionState, { phase: Phase }>
}

test('starts in the exact idle shape and derives actions from the whole state', () => {
  const session = createImportSession(() => fakeAPI())
  const observed = observe(session)
  const state = observed.state()

  expect(state).toEqual({ phase: 'idle', error: '' })
  expect(canSelectImportFile(state)).toBe(true)
  expect(canStartImport(state)).toBe(false)
  expect(selectedImportInspection(state)).toBeNull()

  observed.stop()
})

test('inspection owns only the chosen path before ready owns the inspection', async () => {
  let resolveInspection: (inspection: ImportInspection) => void = () => {}
  const pendingInspection = new Promise<ImportInspection>((resolve) => {
    resolveInspection = resolve
  })
  const calls: string[] = []
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile: async () => {
      calls.push('choose')
      return '/chosen/../club.pgn'
    },
    inspectPuzzleImport: async (path) => {
      calls.push(`inspect:${path}`)
      return pendingInspection
    }
  }))
  const observed = observe(session)

  const selecting = session.selectFile()
  await waitFor(() => expect(observed.state().phase).toBe('inspecting'))
  expect(observed.state()).toEqual({
    phase: 'inspecting', path: '/chosen/../club.pgn', error: ''
  })
  expect(selectedImportInspection(observed.state())).toBeNull()

  resolveInspection(embeddedInspection)
  await selecting
  expect(calls).toEqual(['choose', 'inspect:/chosen/../club.pgn'])
  expect(observed.state()).toEqual({
    phase: 'ready', inspection: embeddedInspection, error: ''
  })
  expect(canStartImport(observed.state())).toBe(true)
  observed.stop()
})

test('selecting owns the prior selectable state and cancellation restores it exactly', async () => {
  let resolveChooser: (path: string) => void = () => {}
  const pendingChooser = new Promise<string>((resolve) => { resolveChooser = resolve })
  const choosePuzzleImportFile = vi.fn()
    .mockResolvedValueOnce('/chosen/club.pgn')
    .mockImplementationOnce(async () => pendingChooser)
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile,
    inspectPuzzleImport: async () => embeddedInspection
  }))
  const observed = observe(session)

  await session.selectFile()
  const previous = requirePhase(observed.state(), 'ready')
  const selecting = session.selectFile()
  await waitFor(() => expect(observed.state().phase).toBe('selecting'))
  expect(observed.state()).toEqual({ phase: 'selecting', previous, error: '' })
  expect(selectedImportInspection(observed.state())).toBe(embeddedInspection)

  resolveChooser('')
  await selecting
  expect(observed.state()).toEqual(previous)
  observed.stop()
})

test('chooser failure preserves selection while inspection failure returns to idle', async () => {
  const choosePuzzleImportFile = vi.fn()
    .mockResolvedValueOnce('/chosen/club.pgn')
    .mockRejectedValueOnce(new Error('dialog unavailable'))
    .mockResolvedValueOnce('/chosen/broken.txt')
  let inspections = 0
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile,
    inspectPuzzleImport: async () => {
      if (++inspections === 1) return embeddedInspection
      throw new Error('unsupported puzzle collection')
    }
  }))
  const observed = observe(session)

  await session.selectFile()
  await session.selectFile()
  expect(observed.state()).toEqual({
    phase: 'ready', inspection: embeddedInspection, error: 'dialog unavailable'
  })

  await session.selectFile()
  expect(observed.state()).toEqual({
    phase: 'idle', error: 'unsupported puzzle collection'
  })
  observed.stop()
})

test('one inspecting phase rejects overlapping selection and stale start attempts', async () => {
  let resolveInspection: (inspection: ImportInspection) => void = () => {}
  const pendingInspection = new Promise<ImportInspection>((resolve) => {
    resolveInspection = resolve
  })
  const choosePuzzleImportFile = vi.fn(async () => '/chosen/club.pgn')
  const startPuzzleImport = vi.fn(async () => 'job-1')
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile,
    inspectPuzzleImport: async () => pendingInspection,
    startPuzzleImport
  }))
  const observed = observe(session)

  const selecting = session.selectFile()
  await waitFor(() => expect(observed.state().phase).toBe('inspecting'))
  await session.selectFile()
  await session.start()
  expect(choosePuzzleImportFile).toHaveBeenCalledOnce()
  expect(startPuzzleImport).not.toHaveBeenCalled()

  resolveInspection(embeddedInspection)
  await selecting
  expect(observed.state()).toEqual({
    phase: 'ready', inspection: embeddedInspection, error: ''
  })
  observed.stop()
})

test('starting owns the confirmed inspection and guards concurrent starts', async () => {
  let resolveStart: (jobId: string) => void = () => {}
  const pendingStart = new Promise<string>((resolve) => { resolveStart = resolve })
  const startPuzzleImport = vi.fn(async () => pendingStart)
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
  await waitFor(() => expect(observed.state().phase).toBe('starting'))
  expect(observed.state()).toEqual({
    phase: 'starting', inspection: embeddedInspection, error: ''
  })
  await session.start()
  expect(startPuzzleImport).toHaveBeenCalledOnce()
  expect(startPuzzleImport).toHaveBeenCalledWith(embeddedInspection)

  resolveStart('job-1')
  await starting
  expect(observed.state()).toEqual({
    phase: 'running',
    inspection: embeddedInspection,
    jobId: 'job-1',
    progress: {
      jobId: 'job-1', phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 500
    },
    error: ''
  })
  observed.stop()
})

test('keeps monotonic progress when an older polling snapshot resolves later', async () => {
  let progressListener: (progress: ImportProgress) => void = () => {}
  let resolveSnapshot: (result: ImportResult) => void = () => {}
  const snapshot = new Promise<ImportResult>((resolve) => { resolveSnapshot = resolve })
  const session = createImportSession(() => fakeAPI({
    inspectPuzzleImport: async () => embeddedInspection,
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
  await waitFor(() => expect(observed.state().phase).toBe('running'))
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

  const running = requirePhase(observed.state(), 'running')
  expect(running.progress).toEqual({
    jobId: 'job-1', phase: 'sealing', rowsRead: 100, bytesRead: 1_500, totalBytes: 2_000
  })
  disconnect()
  observed.stop()
})

test('ignores stale job events while preserving the active running state', async () => {
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
  const observed = observe(session)
  const disconnect = session.connect()

  await session.selectFile()
  await session.start()
  const active = observed.state()
  progressListener({
    jobId: 'stale-job', phase: 'activating', rowsRead: 999, bytesRead: 999, totalBytes: 999
  })
  finishedListener({
    jobId: 'stale-job',
    status: 'succeeded',
    progress: { phase: 'activating', rowsRead: 999, bytesRead: 999, totalBytes: 999 },
    report: { accepted: 999, duplicates: 0, rejected: 0, examples: [] }
  })
  expect(observed.state()).toEqual(active)
  disconnect()
  observed.stop()
})

test('terminal state owns its result and ignores a later running poll', async () => {
  let finishedListener: (result: ImportResult) => void = () => {}
  let resolveSnapshot: (result: ImportResult) => void = () => {}
  const snapshot = new Promise<ImportResult>((resolve) => { resolveSnapshot = resolve })
  const session = createImportSession(() => fakeAPI({
    inspectPuzzleImport: async () => embeddedInspection,
    getImportResult: async () => snapshot,
    onImportFinished: (listener) => {
      finishedListener = listener
      return () => {}
    }
  }))
  const observed = observe(session)
  const disconnect = session.connect()

  await session.selectFile()
  const starting = session.start()
  await waitFor(() => expect(observed.state().phase).toBe('running'))
  const terminal: ImportResult = {
    jobId: 'job-1',
    status: 'succeeded',
    progress: { phase: 'activating', rowsRead: 110, bytesRead: 2_000, totalBytes: 2_000 },
    report: { accepted: 100, duplicates: 5, rejected: 5, examples: [] }
  }
  finishedListener(terminal)
  resolveSnapshot({
    jobId: 'job-1',
    status: 'running',
    progress: { phase: 'parsing', rowsRead: 50, bytesRead: 900, totalBytes: 2_000 },
    report: emptyReport()
  })
  await starting

  expect(observed.state()).toEqual({
    phase: 'finished',
    inspection: embeddedInspection,
    jobId: 'job-1',
    progress: {
      jobId: 'job-1', phase: 'activating', rowsRead: 110, bytesRead: 2_000, totalBytes: 2_000
    },
    result: terminal,
    error: ''
  })
  expect(canSelectImportFile(observed.state())).toBe(true)
  expect(canStartImport(observed.state())).toBe(true)
  disconnect()
  observed.stop()
})

test('restarting a finished inspection resets progress for the new job', async () => {
  let finishedListener: (result: ImportResult) => void = () => {}
  const jobIds = ['job-1', 'job-2']
  const session = createImportSession(() => fakeAPI({
    startPuzzleImport: async () => jobIds.shift() ?? 'unexpected-job',
    getImportResult: async (jobId) => ({
      jobId,
      status: 'running',
      progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 2_500 },
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
    progress: { phase: 'activating', rowsRead: 10, bytesRead: 1_000, totalBytes: 1_000 },
    report: { accepted: 10, duplicates: 0, rejected: 0, examples: [] }
  })
  requirePhase(observed.state(), 'finished')

  await session.start()
  expect(requirePhase(observed.state(), 'running').progress).toEqual({
    jobId: 'job-2', phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 2_500
  })
  disconnect()
  observed.stop()
})

test('new selection clears a finished result before accepting another inspection', async () => {
  let finishedListener: (result: ImportResult) => void = () => {}
  let resolveInspection: (inspection: ImportInspection) => void = () => {}
  const pendingInspection = new Promise<ImportInspection>((resolve) => {
    resolveInspection = resolve
  })
  const chosen = ['/chosen/club.pgn', '/chosen/new.json']
  let inspections = 0
  const session = createImportSession(() => fakeAPI({
    choosePuzzleImportFile: async () => chosen.shift() ?? '',
    inspectPuzzleImport: async () => ++inspections === 1 ? embeddedInspection : pendingInspection,
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
    progress: { phase: 'activating', rowsRead: 1, bytesRead: 1, totalBytes: 1 },
    report: { accepted: 1, duplicates: 0, rejected: 0, examples: [] }
  })
  requirePhase(observed.state(), 'finished')

  const selecting = session.selectFile()
  await waitFor(() => expect(observed.state()).toEqual({
    phase: 'inspecting', path: '/chosen/new.json', error: ''
  }))
  resolveInspection(jsonInspection)
  await selecting
  expect(observed.state()).toEqual({
    phase: 'ready', inspection: jsonInspection, error: ''
  })
  disconnect()
  observed.stop()
})
