import {
  decodeImportInspection,
  decodeImportProgress,
  decodeImportResult
} from './api-contract'

test('requires the backend-provided import format label', () => {
  expect(() => decodeImportInspection({
    path: '/collections/club.pgn',
    filename: 'club.pgn',
    format: 'tactical-pgn',
    sourceId: 'club-tactics',
    sourceIdOrigin: 'embedded',
    replacesExisting: false
  })).toThrow('import inspection.formatLabel must be a string')
})

test('strictly decodes import phases, byte totals, and rejection examples', () => {
  expect(decodeImportProgress({
    jobId: 'job-1', phase: 'sealing', rowsRead: 12, bytesRead: 500, totalBytes: 750
  })).toEqual({
    jobId: 'job-1', phase: 'sealing', rowsRead: 12, bytesRead: 500, totalBytes: 750
  })
  expect(decodeImportResult({
    jobId: 'job-1',
    status: 'succeeded',
    progress: { phase: 'activating', rowsRead: 12, bytesRead: 750, totalBytes: 750 },
    report: {
      accepted: 10, duplicates: 1, rejected: 1,
      examples: [{ ordinal: 7, reason: 'illegal move e2e5' }]
    }
  })).toMatchObject({
    progress: { phase: 'activating', totalBytes: 750 },
    report: { examples: [{ ordinal: 7, reason: 'illegal move e2e5' }] }
  })
  expect(() => decodeImportProgress({
    jobId: 'job-1', phase: 'uploading', rowsRead: 0, bytesRead: 0, totalBytes: 0
  })).toThrow('phase')
  expect(() => decodeImportResult({
    jobId: 'job-1',
    status: 'failed',
    progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 0 },
    report: { accepted: 0, duplicates: 0, rejected: 1, examples: [{ ordinal: 1 }] }
  })).toThrow('reason')
  expect(() => decodeImportResult({
    jobId: 'job-1',
    status: 'succeeded',
    progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 0 },
    report: { accepted: 1, duplicates: 0, rejected: 0 }
  })).toThrow('examples')
})

test('rejects an import result without its required progress snapshot', () => {
  expect(() => decodeImportResult({
    jobId: 'job-1', status: 'running',
    report: { accepted: 0, duplicates: 0, rejected: 0, examples: [] }
  })).toThrow('import result.progress must be an object')
})

test('normalizes a nil Go rejection sample to an empty decoded array', () => {
  expect(decodeImportResult({
    jobId: 'job-1',
    status: 'running',
    progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 100 },
    report: { accepted: 0, duplicates: 0, rejected: 0, examples: null }
  }).report.examples).toEqual([])
})

test('decodes course counts and defaults legacy puzzle counts to empty', () => {
  const base = {
    jobId: 'course-job',
    status: 'succeeded',
    progress: { phase: 'activating', rowsRead: 40, bytesRead: 800, totalBytes: 800 }
  }
  expect(decodeImportResult({
    ...base,
    report: {
      accepted: 1, duplicates: 0, rejected: 0, examples: [],
      counts: { chapters: 3, moves: 40, lessons: 8 }
    }
  }).report.counts).toEqual({ chapters: 3, moves: 40, lessons: 8 })
  expect(decodeImportResult({
    ...base,
    report: { accepted: 1, duplicates: 0, rejected: 0, examples: [] }
  }).report.counts).toEqual({})
})

test.each([
  [['not', 'a record'], 'object'],
  [{ moves: -1 }, 'non-negative integer'],
  [{ moves: 1.5 }, 'non-negative integer'],
  [{ moves: Number.POSITIVE_INFINITY }, 'non-negative integer']
])('rejects invalid import report counts %#', (counts, message) => {
  expect(() => decodeImportResult({
    jobId: 'course-job',
    status: 'succeeded',
    progress: { phase: 'activating', rowsRead: 1, bytesRead: 1, totalBytes: 1 },
    report: { accepted: 1, duplicates: 0, rejected: 0, examples: [], counts }
  })).toThrow(message)
})

test.each(['detecting', 'parsing', 'sealing', 'activating'] as const)(
  'accepts the supported import phase %s',
  (phase) => {
    expect(decodeImportProgress({
      jobId: 'job-1', phase, rowsRead: 0, bytesRead: 0, totalBytes: 0
    }).phase).toBe(phase)
  }
)
