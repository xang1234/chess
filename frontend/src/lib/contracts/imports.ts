import {
  array,
  boolean,
  enumeration,
  number,
  numberRecord,
  optionalString,
  record,
  string
} from './decoder'

const importFormats = [
  'lichess',
  'tactical-pgn',
  'canonical-json',
  'lucas-fns',
  'linear-fen-uci',
  'coursepack'
] as const

export type ImportFormat = typeof importFormats[number]
export type ImportKind = 'puzzle' | 'course'
export type ImportSourceIDOrigin = 'fixed' | 'embedded' | 'path'
export type ImportInspection = {
  path: string
  filename: string
  format: ImportFormat
  formatLabel: string
  sourceId: string
  sourceIdOrigin: ImportSourceIDOrigin
  sourceName?: string
  url?: string
  attribution?: string
  replacesExisting: boolean
}

export type ImportPhase = 'detecting' | 'parsing' | 'sealing' | 'activating'
export type ImportProgress = {
  jobId: string
  phase: ImportPhase
  rowsRead: number
  bytesRead: number
  totalBytes: number
}

type ImportProgressSnapshot = Omit<ImportProgress, 'jobId'>
export type ImportRejection = { ordinal: number; reason: string }
export type ImportReport = {
  accepted: number
  duplicates: number
  rejected: number
  examples: ImportRejection[]
  counts: Record<string, number>
}

export type ImportResult = {
  jobId: string
  status: 'running' | 'succeeded' | 'failed' | 'cancelled'
  progress: ImportProgressSnapshot
  report: ImportReport
  error?: string
}

export function decodeImportInspection(value: unknown): ImportInspection {
  const raw = record(value, 'import inspection')
  return {
    path: string(raw.path, 'import inspection.path'),
    filename: string(raw.filename, 'import inspection.filename'),
    format: enumeration(raw.format, importFormats, 'import inspection.format'),
    formatLabel: string(raw.formatLabel, 'import inspection.formatLabel'),
    sourceId: string(raw.sourceId, 'import inspection.sourceId'),
    sourceIdOrigin: enumeration(
      raw.sourceIdOrigin,
      ['fixed', 'embedded', 'path'],
      'import inspection.sourceIdOrigin'
    ),
    sourceName: optionalString(raw.sourceName, 'import inspection.sourceName'),
    url: optionalString(raw.url, 'import inspection.url'),
    attribution: optionalString(raw.attribution, 'import inspection.attribution'),
    replacesExisting: boolean(raw.replacesExisting, 'import inspection.replacesExisting')
  }
}

function decodeImportPhase(value: unknown, path: string): ImportPhase {
  return enumeration(value, ['detecting', 'parsing', 'sealing', 'activating'], path)
}

function decodeImportProgressSnapshot(value: unknown, path: string): ImportProgressSnapshot {
  const raw = record(value, path)
  return {
    phase: decodeImportPhase(raw.phase, `${path}.phase`),
    rowsRead: number(raw.rowsRead, `${path}.rowsRead`),
    bytesRead: number(raw.bytesRead, `${path}.bytesRead`),
    totalBytes: number(raw.totalBytes, `${path}.totalBytes`)
  }
}

function decodeImportReport(value: unknown, path: string): ImportReport {
  const raw = record(value, path)
  const examples = raw.examples === null
    ? []
    : array(raw.examples, `${path}.examples`, (entry, entryPath) => {
        const example = record(entry, entryPath)
        return {
          ordinal: number(example.ordinal, `${entryPath}.ordinal`),
          reason: string(example.reason, `${entryPath}.reason`)
        }
      })
  return {
    accepted: number(raw.accepted, `${path}.accepted`),
    duplicates: number(raw.duplicates, `${path}.duplicates`),
    rejected: number(raw.rejected, `${path}.rejected`),
    examples,
    counts: raw.counts === undefined ? {} : numberRecord(raw.counts, `${path}.counts`)
  }
}

export function decodeImportProgress(value: unknown): ImportProgress {
  const raw = record(value, 'import progress')
  return {
    jobId: string(raw.jobId, 'import progress.jobId'),
    ...decodeImportProgressSnapshot(raw, 'import progress')
  }
}

export function decodeImportResult(value: unknown): ImportResult {
  const raw = record(value, 'import result')
  return {
    jobId: string(raw.jobId, 'import result.jobId'),
    status: enumeration(
      raw.status,
      ['running', 'succeeded', 'failed', 'cancelled'],
      'import result.status'
    ),
    progress: decodeImportProgressSnapshot(raw.progress, 'import result.progress'),
    report: decodeImportReport(raw.report, 'import result.report'),
    error: optionalString(raw.error, 'import result.error')
  }
}
