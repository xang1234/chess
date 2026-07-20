import type {
  ImportInspection,
  MoveResult,
  NormalAPI,
  RecoveryAPI,
  SessionView
} from './api'
import {
  decodeImportInspection,
  decodeImportProgress,
  decodeImportResult,
  decodeOpeningHintResult,
  decodeOpeningHome,
  decodeOpeningPosition,
  decodeOpeningSession,
  decodeOpeningStepResult
} from './api-contract'
import { decodeOpeningHome as decodeOpeningHomeContract } from './contracts/openings'
import { decodeImportInspection as decodeImportInspectionContract } from './contracts/imports'
import { loadPreviewApplicationAPI } from './api/preview'

type HasKey<Value, Key extends PropertyKey> = Key extends keyof Value ? true : false
type APIModule = typeof import('./api')
type BrowserWindow = Window & { go?: unknown }

const recoveryHasStartGuided: HasKey<RecoveryAPI, 'startGuided'> = false
const normalHasRecoveryState: HasKey<NormalAPI, 'getRecoveryState'> = false
const moduleHasNormalGetter: HasKey<APIModule, 'getAPI'> = false
const moduleHasRecoveryGetter: HasKey<APIModule, 'getRecoveryAPI'> = false

const production = vi.hoisted(() => ({
  getMode: vi.fn(),
  getBuildInfo: vi.fn(),
  getParentSummary: vi.fn(),
  resumeSession: vi.fn(),
  playMove: vi.fn(),
  pauseSession: vi.fn(),
  getOpeningHome: vi.fn(),
  getOpeningPosition: vi.fn(),
  setOpeningDepth: vi.fn(),
  startOpeningLesson: vi.fn(),
  resumeOpeningSession: vi.fn(),
  restartOpeningSession: vi.fn(),
  advanceOpeningStep: vi.fn(),
  playOpeningMove: vi.fn(),
  useOpeningHint: vi.fn(),
  revealOpeningMove: vi.fn(),
  pauseOpeningSession: vi.fn(),
  startOpeningReview: vi.fn(),
  createBackup: vi.fn(),
  restoreBackup: vi.fn(),
  openDataFolder: vi.fn(),
  quit: vi.fn(),
  choosePuzzleImportFile: vi.fn(),
  chooseOpeningCourseFile: vi.fn(),
  inspectPuzzleImport: vi.fn(),
  inspectOpeningCourseImport: vi.fn(),
  startPuzzleImport: vi.fn(),
  startOpeningCourseImport: vi.fn(),
  cancelImport: vi.fn()
}))

vi.mock('../../wailsjs/go/main/ModeController', () => ({
  GetApplicationMode: production.getMode,
  GetBuildInfo: production.getBuildInfo
}))
vi.mock('../../wailsjs/go/main/NormalController', () => ({
  GetParentSummary: production.getParentSummary,
  ResumeSession: production.resumeSession,
  PlayMove: production.playMove,
  PauseSession: production.pauseSession,
  GetOpeningHome: production.getOpeningHome,
  GetOpeningPosition: production.getOpeningPosition,
  SetOpeningDepth: production.setOpeningDepth,
  StartOpeningLesson: production.startOpeningLesson,
  ResumeOpeningSession: production.resumeOpeningSession,
  RestartOpeningSession: production.restartOpeningSession,
  AdvanceOpeningStep: production.advanceOpeningStep,
  PlayOpeningMove: production.playOpeningMove,
  UseOpeningHint: production.useOpeningHint,
  RevealOpeningMove: production.revealOpeningMove,
  PauseOpeningSession: production.pauseOpeningSession,
  StartOpeningReview: production.startOpeningReview,
  CreateBackup: production.createBackup,
  RestoreBackup: production.restoreBackup,
  OpenDataFolder: production.openDataFolder,
  Quit: production.quit,
  ChoosePuzzleImportFile: production.choosePuzzleImportFile,
  ChooseOpeningCourseFile: production.chooseOpeningCourseFile,
  InspectPuzzleImport: production.inspectPuzzleImport,
  InspectOpeningCourseImport: production.inspectOpeningCourseImport,
  StartPuzzleImport: production.startPuzzleImport,
  StartOpeningCourseImport: production.startOpeningCourseImport,
  CancelImport: production.cancelImport
}))
vi.mock('../../wailsjs/go/main/RecoveryController', () => ({
  CreateBackup: production.createBackup,
  RestoreBackup: production.restoreBackup,
  OpenDataFolder: production.openDataFolder,
  Quit: production.quit
}))
vi.mock('../../wailsjs/runtime/runtime', () => ({
  EventsOn: () => () => {}
}))

beforeEach(() => {
  vi.resetModules()
  vi.clearAllMocks()
  delete (window as BrowserWindow).go
})

afterEach(() => {
  delete (window as BrowserWindow).go
})

const openingStepPayload = {
  stepId: 'try-c3',
  kind: 'try',
  title: 'Prepare the centre',
  instruction: 'Choose the course move.',
  variationName: 'Giuoco Piano',
  positionId: 'after-bc5',
  currentFen: 'opening-fen',
  orientation: 'white',
  legalMoves: ['c2c3', 'd2d3'],
  noteTexts: ['Prepare d4.'],
  referenceNoteTexts: ['A detailed source line.'],
  stepNumber: 3,
  stepTotal: 5,
  hintLevel: 0,
  canReveal: false
}

const activeOpeningPayload = {
  sessionId: 'opening-session',
  mode: 'lesson',
  status: 'active',
  courseId: 'italian-white',
  generationId: 'generation-1',
  lessonId: 'giuoco-c3',
  depth: 'reference',
  current: openingStepPayload
}

const openingHomePayload = {
  notice: 'Private course notice',
  courses: [{
    courseId: 'italian-white',
    title: 'Italian Game for White',
    perspective: 'white',
    depth: 'reference',
    rootPositionId: 'initial',
    completedLessons: 1,
    totalLessons: 3,
    dueReviews: 2,
    nextLessonId: 'giuoco-c3',
    nextLessonTitle: 'Prepare d4 with c3',
    hasResumable: true,
    chapters: [{
      chapterId: 'giuoco',
      title: 'Giuoco Piano',
      lessons: [{
        lessonId: 'giuoco-c3',
        title: 'Prepare d4 with c3',
        completedSteps: 2,
        totalSteps: 5,
        completed: false
      }]
    }]
  }]
}

const importInspectionPayload = {
  path: '/collections/club.pgn',
  filename: 'club.pgn',
  format: 'tactical-pgn',
  formatLabel: 'Tactical PGN',
  sourceId: 'club-tactics',
  sourceIdOrigin: 'embedded',
  sourceName: 'Club tactics',
  url: 'https://example.test/club',
  attribution: 'Club authors',
  replacesExisting: true
}

const openingPositionPayload = {
  courseId: 'italian-white',
  positionId: 'after-bc5',
  fen: 'opening-fen',
  label: 'Giuoco Piano',
  evaluation: { code: 'equal', sourceSymbol: '=' },
  notes: [{
    kind: 'overview',
    text: 'Prepare the centre.',
    sourceRef: { printedPage: 18, noteLabel: 'a', coverageId: 'p18-a' }
  }],
  moves: [{
    moveId: 'white-c3',
    uci: 'c2c3',
    san: 'c3',
    toPositionId: 'after-c3',
    role: 'repertoire',
    variationName: 'Giuoco Piano',
    evaluation: { code: 'white_slight', sourceSymbol: '+=' },
    sourceRef: { printedPage: 18, tableColumn: 'I', coverageId: 'p18-c3' }
  }],
  incomingPaths: 2
}

test('strictly decodes opening explorer positions, notes, sources, and moves', () => {
  expect(decodeOpeningPosition(openingPositionPayload)).toEqual(openingPositionPayload)
})

test.each([
  ['role', { ...openingPositionPayload, moves: [{ ...openingPositionPayload.moves[0], role: 'primary' }] }],
  ['evaluation', { ...openingPositionPayload, evaluation: { code: 'edge' } }],
  ['source', {
    ...openingPositionPayload,
    moves: [{ ...openingPositionPayload.moves[0], sourceRef: { printedPage: 0, coverageId: 'bad' } }]
  }],
  ['coverageId', {
    ...openingPositionPayload,
    notes: [{ ...openingPositionPayload.notes[0], sourceRef: { printedPage: 18 } }]
  }],
  ['notes', { ...openingPositionPayload, notes: null }],
  ['san', { ...openingPositionPayload, moves: [{ ...openingPositionPayload.moves[0], san: 3 }] }],
  ['incomingPaths', { ...openingPositionPayload, incomingPaths: -1 }]
])('opening explorer decoder rejects malformed %s data', (_field, value) => {
  expect(() => decodeOpeningPosition(value)).toThrow()
})

test('strictly decodes the opening home hierarchy', () => {
  expect(decodeOpeningHome(openingHomePayload)).toMatchObject({
    notice: 'Private course notice',
    courses: [{ depth: 'reference', dueReviews: 2, chapters: [{ lessons: [{ totalSteps: 5 }] }] }]
  })
})

test.each([
  [{ ...activeOpeningPayload, current: undefined }, 'current step'],
  [{ ...activeOpeningPayload, summary: { totalPrompts: 1 } }, 'must not include a summary'],
  [{ ...activeOpeningPayload, status: 'completed', current: undefined }, 'summary'],
  [{
    ...activeOpeningPayload,
    status: 'completed',
    summary: {
      totalPrompts: 1, positionsRecalled: 1, branchesRecognized: 0,
      retried: 0, usedHint: 0, revealed: 0
    }
  }, 'must not include a current step'],
  [{ ...activeOpeningPayload, status: 'restart_required', current: undefined }, 'notice'],
  [{ ...activeOpeningPayload, status: 'restart_required', notice: 'Updated' }, 'must not include a current step']
])('opening session decoder rejects an invalid discriminated shape', (value, message) => {
  expect(() => decodeOpeningSession(value)).toThrow(message)
})

test.each([
  ['depth', { ...activeOpeningPayload, depth: 'encyclopedic' }],
  ['mode', { ...activeOpeningPayload, mode: 'puzzle' }],
  ['status', { ...activeOpeningPayload, status: 'paused' }],
  ['kind', { ...activeOpeningPayload, current: { ...openingStepPayload, kind: 'quiz' } }],
  ['orientation', { ...activeOpeningPayload, current: { ...openingStepPayload, orientation: 'sideways' } }]
])('opening session decoder rejects unknown %s values', (field, value) => {
  expect(() => decodeOpeningSession(value)).toThrow(field)
})

test.each([
  [{ ...openingStepPayload, stepNumber: 0 }, 'stepNumber'],
  [{ ...openingStepPayload, stepTotal: 1.5 }, 'stepTotal'],
  [{ ...openingStepPayload, legalMoves: ['c2c3', 42] }, 'legalMoves[1]'],
  [{ ...openingStepPayload, noteTexts: undefined }, 'noteTexts'],
  [{ ...openingStepPayload, referenceNoteTexts: undefined }, 'referenceNoteTexts']
])('opening step decoder rejects malformed exact fields', (current, message) => {
  expect(() => decodeOpeningSession({ ...activeOpeningPayload, current })).toThrow(message)
})

test('opening result decoder requires authoritative frames for an expected prompt completion', () => {
  expect(() => decodeOpeningStepResult({
    session: activeOpeningPayload,
    stepCompleted: true,
    feedback: 'expected'
  })).toThrow('authoritative move frames')
  expect(() => decodeOpeningStepResult({
    session: activeOpeningPayload,
    stepCompleted: true,
    feedback: 'expected',
    appliedMoves: [{ uci: 'c2c3', resultingFen: 'after-c3' }]
  })).toThrow('final FEN')
  expect(decodeOpeningStepResult({
    session: activeOpeningPayload,
    stepCompleted: true,
    feedback: 'expected',
    appliedMoves: [{ uci: 'c2c3', resultingFen: 'after-c3' }],
    finalFen: 'after-c3'
  })).toMatchObject({ feedback: 'expected', finalFen: 'after-c3' })
})

test.each(['alternative', 'off_course'] as const)(
  'opening result decoder keeps %s feedback non-mutating',
  (feedback) => {
    const base = {
      session: activeOpeningPayload,
      stepCompleted: false,
      feedback,
      message: 'Try the course move.'
    }
    expect(decodeOpeningStepResult(base)).toMatchObject({ feedback, stepCompleted: false })
    expect(() => decodeOpeningStepResult({
      ...base,
      appliedMoves: [{ uci: 'd2d3', resultingFen: 'wrong-fen' }],
      finalFen: 'wrong-fen'
    })).toThrow('must not include')
  }
)

test('opening result decoder rejects unknown feedback and permits omitted passive frames only', () => {
  expect(() => decodeOpeningStepResult({
    session: activeOpeningPayload,
    stepCompleted: false,
    feedback: 'almost'
  })).toThrow('feedback')
  expect(decodeOpeningStepResult({
    session: activeOpeningPayload,
    stepCompleted: true
  }).appliedMoves).toBeUndefined()
})

test('opening hint decoder requires an active session and exact hint fields', () => {
  expect(decodeOpeningHintResult({
    session: activeOpeningPayload,
    level: 2,
    text: 'The course move starts on c2.',
    sourceSquare: 'c2',
    canReveal: false
  })).toMatchObject({ level: 2, sourceSquare: 'c2' })
  expect(() => decodeOpeningHintResult({
    session: { ...activeOpeningPayload, status: 'restart_required', current: undefined, notice: 'Updated' },
    level: 1,
    text: 'Plan',
    canReveal: false
  })).toThrow('active session')
})

test('API module exposes only the mode-discriminated bootstrap', async () => {
  const { loadApplicationAPI } = await import('./api')

  expect(recoveryHasStartGuided).toBe(false)
  expect(normalHasRecoveryState).toBe(false)
  expect(moduleHasNormalGetter).toBe(false)
  expect(moduleHasRecoveryGetter).toBe(false)
  expect(loadApplicationAPI).toBeTypeOf('function')
})

test('preview API runs a deterministic two-puzzle session', async () => {
  const { loadApplicationAPI } = await import('./api')
  const application = await loadApplicationAPI()
  expect(application.mode).toBe('normal')
  expect(application.buildInfo).toEqual({
    name: 'Chess Trainer',
    commit: 'development',
    sourceUrl: 'https://github.com/xang1234/chess'
  })
  if (application.mode !== 'normal') throw new Error('expected normal preview')

  const started = await application.api.startGuided()
  expect(started.current?.legalMoves).toHaveLength(20)
  expect(started.current?.legalMoves).toContain('e2e3')

  const wrong = await application.api.playMove(started.sessionId, 'e2e3')
  expect(wrong).toMatchObject({
    correct: false,
    puzzleCompleted: false,
    message: 'Try again'
  })
  expect(wrong.appliedMoves).toBeUndefined()
  expect(wrong.finalFen).toBeUndefined()

  const first = await application.api.playMove(started.sessionId, 'e2e4')
  expect(first.correct).toBe(true)
  expect(first.puzzleCompleted).toBe(true)
  expect(first.appliedMoves).toEqual([{
    uci: 'e2e4',
    resultingFen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1'
  }])
  expect(first.finalFen).toBe(first.appliedMoves?.[0].resultingFen)
  expect(first.session.currentIndex).toBe(1)
  expect(first.session.current?.fingerprint).toBe('preview-puzzle-2')

  const resumed = await application.api.resumeSession()
  expect(resumed).toEqual(first.session)

  const final = await application.api.playMove(started.sessionId, 'd2d4')
  expect(final.puzzleCompleted).toBe(true)
  expect(final.finalFen).toBe(
    'rnbqkbnr/pppppppp/8/8/3P4/8/PPP1PPPP/RNBQKBNR b KQkq d3 0 1'
  )
  expect(final.session.status).toBe('completed')
  expect(final.session.current).toBeUndefined()
  expect(final.session.summary).toEqual({
    total: 2,
    firstTry: 1,
    retried: 1,
    usedHint: 0,
    revealed: 0,
    unavailable: 0
  })
})

test('preview API exposes an original guided opening lesson', async () => {
  const application = await (await import('./api')).loadApplicationAPI()
  if (application.mode !== 'normal') throw new Error('expected normal preview')

  const home = await application.api.getOpeningHome()
  expect(home.courses[0]).toMatchObject({
    title: 'Synthetic Italian for White',
    depth: 'reference',
    nextLessonId: 'giuoco-c3'
  })
  const started = await application.api.startOpeningLesson('synthetic-italian', 'giuoco-c3')
  expect(started.current?.kind).toBe('explain')
  const watch = await application.api.advanceOpeningStep(started.sessionId)
  expect(watch.session.current?.kind).toBe('watch')
  const prompt = await application.api.advanceOpeningStep(started.sessionId)
  expect(prompt.appliedMoves).toHaveLength(1)
  expect(prompt.session.current?.kind).toBe('try')
  const alternative = await application.api.playOpeningMove(started.sessionId, 'b2b4')
  expect(alternative).toMatchObject({ feedback: 'alternative', stepCompleted: false })
  const expected = await application.api.playOpeningMove(started.sessionId, 'c2c3')
  expect(expected).toMatchObject({ feedback: 'expected', stepCompleted: true })
  expect(expected.finalFen).toBe(expected.appliedMoves?.[0].resultingFen)
  const position = await application.api.getOpeningPosition(
    'synthetic-italian', 'initial', 'reference'
  )
  expect(position).toMatchObject({ positionId: 'initial', moves: [{ san: 'e4' }] })
})

test('production opening adapters decode every returned boundary', async () => {
  enableProduction()
  production.getOpeningHome.mockResolvedValue({ courses: [] })
  production.getOpeningPosition.mockResolvedValue(openingPositionPayload)
  production.startOpeningLesson.mockResolvedValue(activeOpeningPayload)
  production.resumeOpeningSession.mockResolvedValue(activeOpeningPayload)
  production.playOpeningMove.mockResolvedValue({
    session: activeOpeningPayload,
    stepCompleted: true,
    feedback: 'expected',
    appliedMoves: [{ uci: 'c2c3', resultingFen: 'after-c3' }],
    finalFen: 'after-c3'
  })

  const application = await (await import('./api')).loadApplicationAPI()
  if (application.mode !== 'normal') throw new Error('expected normal production API')
  await expect(application.api.getOpeningHome()).resolves.toEqual({ courses: [] })
  await expect(application.api.getOpeningPosition('italian-white', 'after-bc5', 'reference'))
    .resolves.toEqual(openingPositionPayload)
  await expect(application.api.startOpeningLesson('italian-white', 'giuoco-c3')).resolves
    .toMatchObject({ status: 'active', current: { kind: 'try' } })
  await expect(application.api.resumeOpeningSession()).resolves
    .toMatchObject({ sessionId: 'opening-session' })
  await expect(application.api.playOpeningMove('opening-session', 'c2c3')).resolves
    .toMatchObject({ feedback: 'expected', finalFen: 'after-c3' })
  expect(production.playOpeningMove).toHaveBeenCalledWith('opening-session', 'c2c3')
  expect(production.getOpeningPosition).toHaveBeenCalledWith(
    'italian-white', 'after-bc5', 'reference'
  )
})

test('production adaptation preserves authoritative board fields exactly', async () => {
  const session: SessionView = {
    sessionId: 'production-session',
    mode: 'guided',
    status: 'active',
    currentIndex: 0,
    total: 1,
    current: {
      fingerprint: 'fingerprint',
      displayedFen: 'displayed',
      currentFen: 'current',
      solver: 'white',
      currentPath: [],
      puzzleNumber: 1,
      puzzleTotal: 1,
      hintLevel: 0,
      incorrectMoves: 0,
      canReveal: false,
      legalMoves: ['e2e3', 'e2e4']
    }
  }
  const move: MoveResult = {
    session,
    correct: true,
    puzzleCompleted: true,
    appliedMoves: [{ uci: 'e2e4', resultingFen: 'after' }],
    finalFen: 'after'
  }
  production.getMode.mockResolvedValue('normal')
  production.getBuildInfo.mockResolvedValue({
    name: 'Chess Trainer',
    commit: '0123456789abcdef0123456789abcdef01234567',
    sourceUrl: 'https://github.com/xang1234/chess/tree/0123456789abcdef0123456789abcdef01234567'
  })
  production.resumeSession.mockResolvedValue(session)
  production.playMove.mockResolvedValue(move)
  Object.defineProperty(window, 'go', {
    configurable: true,
    value: { main: {} }
  })

  const { loadApplicationAPI } = await import('./api')
  const application = await loadApplicationAPI()
  if (application.mode !== 'normal') throw new Error('expected normal production API')

  expect(production.getMode).toHaveBeenCalledOnce()
  expect(production.getBuildInfo).toHaveBeenCalledOnce()
  expect(application.buildInfo).toEqual({
    name: 'Chess Trainer',
    commit: '0123456789abcdef0123456789abcdef01234567',
    sourceUrl: 'https://github.com/xang1234/chess/tree/0123456789abcdef0123456789abcdef01234567'
  })

  await expect(application.api.resumeSession()).resolves.toEqual(session)
  await expect(application.api.playMove(session.sessionId, 'e2e4')).resolves.toEqual(move)
})

function enableProduction(mode: string = 'normal'): void {
  production.getMode.mockResolvedValue(mode)
  production.getBuildInfo.mockResolvedValue({
    name: 'Chess Trainer',
    commit: '0123456789abcdef0123456789abcdef01234567',
    sourceUrl: 'https://github.com/xang1234/chess/tree/0123456789abcdef0123456789abcdef01234567'
  })
  Object.defineProperty(window, 'go', {
    configurable: true,
    value: { main: {} }
  })
}

test('production bootstrap rejects an unknown application mode', async () => {
  enableProduction('future-mode')

  const { loadApplicationAPI } = await import('./api')

  await expect(loadApplicationAPI()).rejects.toThrow('application mode')
})

test('production session boundary rejects an unknown puzzle solver', async () => {
  enableProduction()
  production.resumeSession.mockResolvedValue({
    sessionId: 'session', mode: 'guided', status: 'active', currentIndex: 0, total: 1,
    current: {
      fingerprint: 'puzzle', displayedFen: 'displayed', currentFen: 'current',
      solver: 'sideways', currentPath: [], puzzleNumber: 1, puzzleTotal: 1,
      hintLevel: 0, incorrectMoves: 0, canReveal: false, legalMoves: ['e2e4']
    }
  })

  const application = await (await import('./api')).loadApplicationAPI()
  if (application.mode !== 'normal') throw new Error('expected normal production API')

  await expect(application.api.resumeSession()).rejects.toThrow('solver')
})

test.each([
  {
    name: 'active session without a current puzzle',
    value: { sessionId: 'session', mode: 'guided', status: 'active', currentIndex: 0, total: 1 },
    message: 'current puzzle'
  },
  {
    name: 'completed session without a summary',
    value: { sessionId: 'session', mode: 'guided', status: 'completed', currentIndex: 1, total: 1 },
    message: 'summary'
  }
])('production session boundary rejects $name', async ({ value, message }) => {
  enableProduction()
  production.resumeSession.mockResolvedValue(value)

  const application = await (await import('./api')).loadApplicationAPI()
  if (application.mode !== 'normal') throw new Error('expected normal production API')

  await expect(application.api.resumeSession()).rejects.toThrow(message)
})

test('production move boundary rejects a completed move without authoritative frames', async () => {
  enableProduction()
  production.playMove.mockResolvedValue({
    session: {
      sessionId: 'session', mode: 'guided', status: 'completed', currentIndex: 1, total: 1,
      summary: { total: 1, firstTry: 1, retried: 0, usedHint: 0, revealed: 0, unavailable: 0 }
    },
    correct: true,
    puzzleCompleted: true,
    finalFen: 'final'
  })

  const application = await (await import('./api')).loadApplicationAPI()
  if (application.mode !== 'normal') throw new Error('expected normal production API')

  await expect(application.api.playMove('session', 'e2e4')).rejects.toThrow('move frames')
})

test('production move boundary rejects a continuing result that includes a final FEN', async () => {
  enableProduction()
  production.playMove.mockResolvedValue({
    session: {
      sessionId: 'session', mode: 'guided', status: 'active', currentIndex: 0, total: 1,
      current: {
        fingerprint: 'puzzle', displayedFen: 'displayed', currentFen: 'current', solver: 'white',
        currentPath: [0], puzzleNumber: 1, puzzleTotal: 1, hintLevel: 0,
        incorrectMoves: 0, canReveal: false, legalMoves: ['e1d1']
      }
    },
    correct: true,
    puzzleCompleted: false,
    appliedMoves: [{ uci: 'e2e4', resultingFen: 'after' }],
    finalFen: 'after'
  })

  const application = await (await import('./api')).loadApplicationAPI()
  if (application.mode !== 'normal') throw new Error('expected normal production API')

  await expect(application.api.playMove('session', 'e2e4')).rejects.toThrow('final FEN')
})

test('production parent summary accepts paused recent sessions', async () => {
  enableProduction()
  production.getParentSummary.mockResolvedValue({
    learnerRating: 1200,
    ratingTrend: [],
    firstAttemptAccuracy: 0,
    hintRate: 0,
    themePerformance: [],
    dueReviews: 0,
    recentSessions: [{
      sessionId: 'paused-session',
      mode: 'guided',
      status: 'paused',
      updatedAt: 1,
      total: 5,
      completed: 2,
      firstTry: 1,
      usedHint: 0,
      revealed: 0
    }]
  })

  const application = await (await import('./api')).loadApplicationAPI()
  if (application.mode !== 'normal') throw new Error('expected normal production API')

  await expect(application.api.getParentSummary()).resolves.toMatchObject({
    recentSessions: [{ status: 'paused' }]
  })
})

test('decodes an authoritative puzzle import inspection', () => {
  expect(decodeImportInspection(importInspectionPayload)).toEqual({
    path: '/collections/club.pgn',
    filename: 'club.pgn',
    format: 'tactical-pgn',
    formatLabel: 'Tactical PGN',
    sourceId: 'club-tactics',
    sourceIdOrigin: 'embedded',
    sourceName: 'Club tactics',
    url: 'https://example.test/club',
    attribution: 'Club authors',
    replacesExisting: true
  })
})

test('loads split domain contracts and the preview adapter directly', async () => {
  expect(decodeOpeningHomeContract(openingHomePayload)).toEqual(openingHomePayload)
  expect(decodeImportInspectionContract(importInspectionPayload)).toEqual(importInspectionPayload)
  await expect(loadPreviewApplicationAPI()).resolves.toMatchObject({ mode: 'normal' })
})

test.each([
  'lichess',
  'tactical-pgn',
  'canonical-json',
  'lucas-fns',
  'linear-fen-uci',
  'coursepack'
] as const)('accepts the supported import format %s', (format) => {
  expect(decodeImportInspection({
    path: '/collections/puzzles',
    filename: 'puzzles',
    format,
    formatLabel: 'Supported format',
    sourceId: 'source',
    sourceIdOrigin: 'fixed',
    replacesExisting: false
  }).format).toBe(format)
})

test.each(['fixed', 'embedded', 'path'] as const)(
  'accepts the supported source ID origin %s',
  (sourceIdOrigin) => {
    expect(decodeImportInspection({
      path: '/collections/puzzles',
      filename: 'puzzles',
      format: 'lichess',
      formatLabel: 'Lichess',
      sourceId: 'source',
      sourceIdOrigin,
      replacesExisting: false
    }).sourceIdOrigin).toBe(sourceIdOrigin)
  }
)

test.each([
  ['format', 'future-format'],
  ['sourceIdOrigin', 'guessed']
])('rejects an unknown import inspection %s', (field, value) => {
  expect(() => decodeImportInspection({
    path: '/collections/club.pgn',
    filename: 'club.pgn',
    format: 'tactical-pgn',
    formatLabel: 'Tactical PGN',
    sourceId: 'club-tactics',
    sourceIdOrigin: 'embedded',
    replacesExisting: false,
    [field]: value
  })).toThrow(field)
})

test.each([
  [{}, 'path'],
  [{
    path: '/collections/club.pgn',
    filename: 42,
    format: 'tactical-pgn',
    formatLabel: 'Tactical PGN',
    sourceId: 'club-tactics',
    sourceIdOrigin: 'embedded',
    replacesExisting: false
  }, 'filename'],
  [{
    path: '/collections/club.pgn',
    filename: 'club.pgn',
    format: 'tactical-pgn',
    formatLabel: 'Tactical PGN',
    sourceId: 'club-tactics',
    sourceIdOrigin: 'embedded',
    replacesExisting: 'yes'
  }, 'replacesExisting']
])('rejects a malformed import inspection', (value, message) => {
  expect(() => decodeImportInspection(value)).toThrow(message)
})

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
    jobId: 'job-1',
    phase: 'sealing',
    rowsRead: 12,
    bytesRead: 500,
    totalBytes: 750
  })).toEqual({
    jobId: 'job-1',
    phase: 'sealing',
    rowsRead: 12,
    bytesRead: 500,
    totalBytes: 750
  })

  expect(decodeImportResult({
    jobId: 'job-1',
    status: 'succeeded',
    progress: {
      phase: 'activating',
      rowsRead: 12,
      bytesRead: 750,
      totalBytes: 750
    },
    report: {
      accepted: 10,
      duplicates: 1,
      rejected: 1,
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
    jobId: 'job-1',
    status: 'running',
    report: { accepted: 0, duplicates: 0, rejected: 0, examples: [] }
  })).toThrow('import result.progress must be an object')
})

test('normalizes a nil Go rejection sample to an empty decoded array', () => {
  expect(decodeImportResult({
    jobId: 'job-1',
    status: 'running',
    progress: {
      phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 100
    },
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

test('production API inspects and starts puzzle imports through generic generated bindings', async () => {
  const inspection: ImportInspection = {
    path: '/normalized/club.pgn',
    filename: 'club.pgn',
    format: 'tactical-pgn',
    formatLabel: 'Tactical PGN',
    sourceId: 'club-tactics',
    sourceIdOrigin: 'embedded',
    replacesExisting: false
  }
  enableProduction()
  production.inspectPuzzleImport.mockResolvedValue(inspection)
  production.startPuzzleImport.mockResolvedValue('job-42')

  const application = await (await import('./api')).loadApplicationAPI()
  if (application.mode !== 'normal') throw new Error('expected normal production API')

  await expect(application.api.inspectPuzzleImport('/chosen/club.pgn')).resolves.toEqual(inspection)
  await expect(application.api.startPuzzleImport(inspection)).resolves.toBe('job-42')
  expect(production.inspectPuzzleImport).toHaveBeenCalledWith('/chosen/club.pgn')
  expect(production.startPuzzleImport).toHaveBeenCalledWith(inspection)
  expect('startLichessImport' in application.api).toBe(false)
})

test('production API keeps the confirmed course inspection intact', async () => {
  const inspection: ImportInspection = {
    path: '/normalized/italian.ctcourse',
    filename: 'italian.ctcourse',
    format: 'coursepack',
    formatLabel: 'Opening course',
    sourceId: 'italian-white',
    sourceIdOrigin: 'embedded',
    sourceName: 'Italian Game for White',
    replacesExisting: true
  }
  enableProduction()
  production.chooseOpeningCourseFile.mockResolvedValue(inspection.path)
  production.inspectOpeningCourseImport.mockResolvedValue(inspection)
  production.startOpeningCourseImport.mockResolvedValue('course-job')

  const application = await (await import('./api')).loadApplicationAPI()
  if (application.mode !== 'normal') throw new Error('expected normal production API')

  await expect(application.api.chooseOpeningCourseFile()).resolves.toBe(inspection.path)
  await expect(application.api.inspectOpeningCourseImport(inspection.path)).resolves.toEqual(inspection)
  await expect(application.api.startOpeningCourseImport(inspection)).resolves.toBe('course-job')
  expect(production.inspectOpeningCourseImport).toHaveBeenCalledWith(inspection.path)
  expect(production.startOpeningCourseImport).toHaveBeenCalledWith(inspection)
})
