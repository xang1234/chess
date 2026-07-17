import type {
  MoveResult,
  NormalAPI,
  RecoveryAPI,
  SessionView
} from './api'

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
  createBackup: vi.fn(),
  restoreBackup: vi.fn(),
  openDataFolder: vi.fn(),
  quit: vi.fn(),
  choosePuzzleImportFile: vi.fn(),
  startLichessImport: vi.fn(),
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
  CreateBackup: production.createBackup,
  RestoreBackup: production.restoreBackup,
  OpenDataFolder: production.openDataFolder,
  Quit: production.quit,
  ChoosePuzzleImportFile: production.choosePuzzleImportFile,
  StartLichessImport: production.startLichessImport,
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
