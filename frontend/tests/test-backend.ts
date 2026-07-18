import type { Page } from '@playwright/test'

export type PuzzleDefinition = {
  fingerprint: string
  fen: string
  solver: 'white' | 'black'
  legalMoves: string[]
  canReveal?: boolean
  sourceFen?: string
  preludeUci?: string
  currentPath?: number[]
}

type AppliedMoveDefinition = { uci: string; resultingFen: string }

export type ContinueResponse = {
  kind: 'continue'
  uci: string
  appliedMoves: AppliedMoveDefinition[]
  continuation: PuzzleDefinition
}

export type NextResponse = {
  kind: 'next'
  uci: string
  appliedMoves: AppliedMoveDefinition[]
  finalFen: string
  next: PuzzleDefinition
}

export type SummaryResponse = {
  kind: 'summary'
  uci: string
  appliedMoves: AppliedMoveDefinition[]
  finalFen: string
}

export type ResponseDefinition = ContinueResponse | NextResponse | SummaryResponse

export type BoardScenario = {
  kind: 'board'
  first: PuzzleDefinition
  wrongMoves?: string[]
  correct?: ResponseDefinition
  reveal?: ResponseDefinition
}

export type TrainerScenario = { kind: 'trainer' }
export type TestBackendScenario = BoardScenario | TrainerScenario

export type TrustedInput = { type: string; trusted: boolean }

type TestBackendState = {
  moves: string[]
  reveals: number
  trustedInputs: TrustedInput[]
  semanticFrames: string[][]
  holdImportOpen(): void
  selectedImportPath(): string
  reportImportProgress(rowsRead: number, bytesRead: number): void
}

type WireValue<Value> =
  Value extends Promise<infer Resolved> ? Promise<WireValue<Resolved>>
    : Value extends readonly (infer Entry)[] ? WireValue<Entry>[]
      : Value extends object ? {
        [Key in keyof Value as Value[Key] extends (...arguments_: never[]) => unknown
          ? never
          : Key]: WireValue<Value[Key]>
      }
        : Value

type BindingMock<Bindings> = {
  [Key in keyof Bindings]: Bindings[Key] extends (...args: infer Arguments) => infer Result
    ? (...args: Arguments) => WireValue<Result>
    : never
}

type ModeControllerMock = BindingMock<typeof import('../wailsjs/go/main/ModeController')>
type NormalBindings = typeof import('../wailsjs/go/main/NormalController')
type NormalControllerMock = Omit<BindingMock<NormalBindings>, 'GetProfile' | 'ResumeSession'> & {
  GetProfile: (...args: Parameters<NormalBindings['GetProfile']>) =>
    Promise<WireValue<Awaited<ReturnType<NormalBindings['GetProfile']>>> | null>
  ResumeSession: (...args: Parameters<NormalBindings['ResumeSession']>) =>
    Promise<WireValue<Awaited<ReturnType<NormalBindings['ResumeSession']>>> | null>
}
type RecoveryControllerMock = BindingMock<typeof import('../wailsjs/go/main/RecoveryController')>
type RuntimeBindings = typeof import('../wailsjs/runtime/runtime')
type WireSession = Exclude<Awaited<ReturnType<NormalControllerMock['StartGuided']>>, null>
type WirePuzzle = NonNullable<WireSession['current']>
type WireImportResult = Awaited<ReturnType<NormalControllerMock['GetImportResult']>>

type TestWindow = Window & {
  __testBackend: TestBackendState
  runtime: Pick<RuntimeBindings, 'EventsOnMultiple' | 'EventsOff' | 'EventsOffAll'>
  go: {
    main: {
      ModeController: ModeControllerMock
      NormalController: NormalControllerMock
      RecoveryController: RecoveryControllerMock
    }
  }
}

export async function installTestBackend(
  page: Page,
  scenario: TestBackendScenario
): Promise<void> {
  await page.addInitScript((configured: TestBackendScenario) => {
    const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T
    const listeners: Record<string, Array<(payload: unknown) => void>> = {}
    const state = {
      moves: [] as string[],
      reveals: 0,
      trustedInputs: [] as TrustedInput[],
      semanticFrames: [] as string[][],
      holdImportOpen: () => {},
      selectedImportPath: () => '',
      reportImportProgress: (_rowsRead: number, _bytesRead: number) => {}
    }
    const testWindow = window as unknown as TestWindow
    testWindow.__testBackend = state

    const emit = (name: string, payload: unknown) => {
      for (const listener of listeners[name] ?? []) listener(payload)
    }
    testWindow.runtime = {
      EventsOnMultiple(name: string, callback: (payload: unknown) => void) {
        listeners[name] = [...(listeners[name] ?? []), callback]
        return () => {
          listeners[name] = (listeners[name] ?? []).filter((value) => value !== callback)
        }
      },
      EventsOff(name: string) { delete listeners[name] },
      EventsOffAll() { for (const name of Object.keys(listeners)) delete listeners[name] }
    }

    document.addEventListener('mousedown', (event) => {
      if ((event.target as Element | null)?.closest?.('cg-board')) {
        state.trustedInputs.push({ type: event.type, trusted: event.isTrusted })
      }
    }, true)
    document.addEventListener('touchstart', (event) => {
      if ((event.target as Element | null)?.closest?.('cg-board')) {
        state.trustedInputs.push({ type: event.type, trusted: event.isTrusted })
      }
    }, true)

    const buildInfo = {
      name: 'Chess Trainer',
      commit: 'development',
      sourceUrl: 'https://github.com/xang1234/chess'
    }
    const emptyImportResult: WireImportResult = {
      jobId: 'unused',
      request: { kind: 'lichess', sourceId: 'lichess', path: '' },
      status: 'cancelled',
      progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 0 },
      report: { accepted: 0, duplicates: 0, rejected: 0, examples: [] }
    }
    const normalController: NormalControllerMock = {
      CancelImport: async () => {},
      ChoosePuzzleImportFile: async () => '',
      CreateBackup: async () => '',
      GetImportResult: async () => emptyImportResult,
      GetParentSummary: async () => ({
        learnerRating: 1200, ratingTrend: [], firstAttemptAccuracy: 0, hintRate: 0,
        dueReviews: 0, themePerformance: [], recentSessions: []
      }),
      GetPracticeFilters: async () => ({
        sources: [], themes: [], maximumSolutionPlies: 1,
        learnerRatingBounds: { minimum: 800, maximum: 2200 }
      }),
      GetProfile: async () => ({ learnerRating: 1200, sessionSize: 5 }),
      InspectPuzzleImport: async () => ({
        path: '', filename: '', format: 'tactical-pgn', sourceId: '',
        sourceIdOrigin: 'embedded', replacesExisting: false
      }),
      OpenDataFolder: async () => {},
      PauseSession: async () => {},
      PlayMove: async () => { throw new Error('test backend has no move response') },
      Quit: async () => {},
      RestoreBackup: async () => {},
      ResumeSession: async () => null,
      RevealSolution: async () => { throw new Error('test backend has no reveal response') },
      StartFreePractice: async () => { throw new Error('test backend has no practice session') },
      StartGuided: async () => { throw new Error('test backend has no guided session') },
      StartLichessImport: async () => 'unused',
      StartPuzzleImport: async () => 'unused',
      UpdateProfile: async () => {},
      UseHint: async () => { throw new Error('test backend has no hint response') }
    }
    const modeController: ModeControllerMock = {
      GetApplicationMode: async () => 'normal',
      GetBuildInfo: async () => buildInfo
    }
    const recoveryController: RecoveryControllerMock = {
      CreateBackup: async () => '',
      GetRecoveryState: async () => ({ required: false }),
      OpenDataFolder: async () => {},
      Quit: async () => {},
      RestoreBackup: async () => {}
    }
    const installNormalController = (overrides: Partial<NormalControllerMock>) => {
      testWindow.go = { main: {
        ModeController: modeController,
        NormalController: { ...normalController, ...overrides },
        RecoveryController: recoveryController
      } }
    }

    if (configured.kind === 'board') {
      const configuredNext = [configured.correct, configured.reveal]
        .find((response): response is NextResponse => response?.kind === 'next')?.next
      const total = configuredNext ? 2 : 1
      const puzzleView = (definition: PuzzleDefinition, index: number) => ({
        fingerprint: definition.fingerprint,
        ...(definition.sourceFen ? { sourceFen: definition.sourceFen } : {}),
        displayedFen: definition.fen,
        currentFen: definition.fen,
        ...(definition.preludeUci ? { preludeUci: definition.preludeUci } : {}),
        solver: definition.solver,
        currentPath: definition.currentPath ?? [],
        puzzleNumber: index + 1,
        puzzleTotal: total,
        hintLevel: 0,
        incorrectMoves: 0,
        canReveal: definition.canReveal ?? false,
        legalMoves: [...definition.legalMoves]
      })
      const active = (definition: PuzzleDefinition, index: number) => ({
        sessionId: 'board-session',
        mode: 'guided',
        status: 'active',
        currentIndex: index,
        total,
        current: puzzleView(definition, index)
      })
      const completed = (revealed: boolean) => ({
        sessionId: 'board-session',
        mode: 'guided',
        status: 'completed',
        currentIndex: total,
        total,
        summary: {
          total,
          firstTry: revealed ? 0 : total,
          retried: 0,
          usedHint: 0,
          revealed: revealed ? 1 : 0,
          unavailable: 0
        }
      })

      let session: ReturnType<typeof active> | ReturnType<typeof completed> | null = null
      const applyResponse = (response: ResponseDefinition, revealed: boolean) => {
        if (response.kind === 'next') {
          session = active(response.next, 1)
        } else if (response.kind === 'continue') {
          session = active(response.continuation, 0)
        } else {
          session = completed(revealed)
        }
        return {
          session: clone(session),
          correct: true,
          puzzleCompleted: response.kind !== 'continue',
          appliedMoves: clone(response.appliedMoves),
          ...(response.kind === 'continue' ? {} : { finalFen: response.finalFen })
        }
      }

      installNormalController({
        StartGuided: async () => {
          session = active(configured.first, 0)
          return clone(session)
        },
        PlayMove: async (_sessionId: string, uci: string) => {
          state.moves.push(uci)
          if (configured.correct && uci === configured.correct.uci) {
            return applyResponse(configured.correct, false)
          }
          if (!configured.wrongMoves?.includes(uci) || !session || !('current' in session)) {
            throw new Error(`unexpected test move ${uci}`)
          }
          session.current.incorrectMoves += 1
          return {
            session: clone(session),
            correct: false,
            puzzleCompleted: false,
            message: 'Try again'
          }
        },
        UseHint: async () => ({
          session: clone(session),
          level: 3,
          text: 'Try this destination.',
          sourceSquare: 'e2',
          targetSquare: 'e4',
          canReveal: true
        }),
        RevealSolution: async () => {
          const response = configured.reveal ?? configured.correct
          if (!response) throw new Error('test scenario has no reveal response')
          state.reveals += 1
          return applyResponse(response, true)
        },
        StartFreePractice: async () => {
          session = active(configured.first, 0)
          return clone(session)
        }
      })
      return
    }

    const fen = '4k3/8/8/4p3/8/8/4P3/4K3 w - - 0 2'
    const finalFen = '4k3/8/8/4p3/4P3/8/8/4K3 b - - 0 2'
    const puzzle = (fingerprint: string, number: number, total: number): WirePuzzle => ({
      fingerprint,
      displayedFen: fen,
      currentFen: fen,
      solver: 'white',
      currentPath: [],
      puzzleNumber: number,
      puzzleTotal: total,
      hintLevel: 0,
      incorrectMoves: 0,
      canReveal: false,
      legalMoves: ['e2e3', 'e2e4']
    })
    const active = (fingerprint = 'guided-1', number = 1): WireSession => ({
      sessionId: 'guided-session',
      mode: 'guided',
      status: 'active',
      currentIndex: number - 1,
      total: 2,
      current: puzzle(fingerprint, number, 2)
    })
    let profile: { learnerRating: number; sessionSize: number } | null = null
    let activeSession: WireSession | null = null
    let hintLevel = 0
    let importedPath = ''
    let holdImportOpen = false
    const importPath = '/Users/family/Downloads/club-tactics.pgn'
    const importBytes = 4096
    let importResult: WireImportResult = {
      jobId: 'job-1',
      request: { kind: 'tactical-pgn', sourceId: 'club-tactics', path: '' },
      status: 'running',
      progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: importBytes },
      report: { accepted: 0, duplicates: 0, rejected: 0, examples: [] }
    }

    const requireActiveSession = (): WireSession => {
      if (!activeSession) throw new Error('test backend has no active session')
      return activeSession
    }

    state.holdImportOpen = () => { holdImportOpen = true }
    state.selectedImportPath = () => importedPath
    state.reportImportProgress = (rowsRead: number, bytesRead: number) => {
      const progress = {
        jobId: 'job-1', phase: 'parsing', rowsRead, bytesRead, totalBytes: importBytes
      }
      importResult = {
        ...importResult,
        progress: { phase: 'parsing', rowsRead, bytesRead, totalBytes: importBytes }
      }
      emit('import:progress', progress)
    }

    installNormalController({
      GetProfile: async () => profile,
      UpdateProfile: async (value: { learnerRating: number; sessionSize: number }) => {
        profile = value
      },
      ResumeSession: async () => activeSession,
      StartGuided: async () => {
        hintLevel = 0
        activeSession = active()
        return activeSession
      },
      PlayMove: async (_sessionId: string, uci: string) => {
        state.moves.push(uci)
        if (uci !== 'e2e4') {
          return {
            session: requireActiveSession(),
            correct: false,
            puzzleCompleted: false,
            message: 'Try again'
          }
        }
        activeSession = active('guided-2', 2)
        return {
          session: activeSession,
          correct: true,
          puzzleCompleted: true,
          appliedMoves: [{ uci: 'e2e4', resultingFen: finalFen }],
          finalFen
        }
      },
      PauseSession: async () => {},
      UseHint: async () => {
        const session = requireActiveSession()
        hintLevel++
        if (hintLevel === 1) {
          return { session, level: 1, text: 'Look for: fork', canReveal: false }
        }
        if (hintLevel === 2) {
          return {
            session, level: 2, text: 'Start with this piece.',
            sourceSquare: 'e2', canReveal: false
          }
        }
        return {
          session, level: 3, text: 'Try this destination.',
          sourceSquare: 'e2', targetSquare: 'e4', canReveal: true
        }
      },
      RevealSolution: async () => {
        activeSession = {
          sessionId: 'guided-session', mode: 'guided', status: 'completed', currentIndex: 2, total: 2,
          summary: { total: 2, firstTry: 0, retried: 1, usedHint: 1, revealed: 1, unavailable: 0 }
        }
        return {
          session: activeSession,
          correct: true,
          puzzleCompleted: true,
          appliedMoves: [{ uci: 'e2e4', resultingFen: finalFen }],
          finalFen
        }
      },
      StartFreePractice: async () => {
        activeSession = {
          sessionId: 'practice-session', mode: 'practice', status: 'active', currentIndex: 0, total: 1,
          current: puzzle('practice-1', 1, 1)
        }
        return activeSession
      },
      GetPracticeFilters: async () => ({
        sources: [{
          id: 'lichess', kind: 'lichess', minimumRating: 800, maximumRating: 2200,
          hasRatingRange: true, maximumPlies: 5
        }],
        themes: ['fork', 'pin'],
        maximumSolutionPlies: 5,
        learnerRatingBounds: { minimum: 800, maximum: 2200 }
      }),
      GetParentSummary: async () => ({
        learnerRating: profile?.learnerRating ?? 1200,
        ratingTrend: [
          { rating: 1150, recordedAt: 1 },
          { rating: profile?.learnerRating ?? 1200, recordedAt: 2 }
        ],
        firstAttemptAccuracy: 50,
        hintRate: 25,
        dueReviews: 1,
        themePerformance: [{ theme: 'fork', attempts: 4, accuracy: 50 }],
        recentSessions: [{
          sessionId: 'guided-session', mode: 'guided', status: 'completed', updatedAt: 2,
          total: 2, completed: 2, firstTry: 0, usedHint: 1, revealed: 1
        }]
      }),
      ChoosePuzzleImportFile: async () => importPath,
      InspectPuzzleImport: async (path: string) => {
        if (path !== importPath) throw new Error(`unexpected inspection path ${path}`)
        return {
          path: importPath,
          filename: 'club-tactics.pgn',
          format: 'tactical-pgn',
          sourceId: 'club-tactics',
          sourceIdOrigin: 'embedded',
          replacesExisting: false
        }
      },
      StartPuzzleImport: async (path: string) => {
        importedPath = path
        importResult = {
          jobId: 'job-1',
          request: { kind: 'tactical-pgn', sourceId: 'club-tactics', path },
          status: 'running',
          progress: {
            phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: importBytes
          },
          report: { accepted: 0, duplicates: 0, rejected: 0, examples: [] }
        }
        if (holdImportOpen) return 'job-1'
        setTimeout(() => {
          emit('import:progress', {
            jobId: 'job-1', phase: 'parsing', rowsRead: 10_000,
            bytesRead: 2048, totalBytes: importBytes
          })
          importResult = {
            ...importResult,
            status: 'succeeded',
            progress: {
              phase: 'activating', rowsRead: 10_000,
              bytesRead: importBytes, totalBytes: importBytes
            },
            report: { accepted: 9800, duplicates: 150, rejected: 50, examples: [] }
          }
          emit('import:finished', importResult)
        }, 20)
        return 'job-1'
      },
      CancelImport: async () => {
        importResult = {
          ...importResult,
          status: 'cancelled',
          report: { accepted: 0, duplicates: 0, rejected: 0, examples: [] }
        }
        emit('import:finished', importResult)
      },
      GetImportResult: async () => importResult,
      CreateBackup: async () => '/tmp/backup.zip',
      RestoreBackup: async () => {},
      OpenDataFolder: async () => {},
      Quit: async () => {}
    })
  }, scenario)
}

export async function backendState<Value>(page: Page, key: keyof TestBackendState): Promise<Value> {
  return page.evaluate((name) => {
    const state = (window as unknown as TestWindow).__testBackend
    return state[name] as Value
  }, key)
}

export async function holdImportOpen(page: Page): Promise<void> {
  await page.evaluate(() => (window as unknown as TestWindow).__testBackend.holdImportOpen())
}

export async function selectedImportPath(page: Page): Promise<string> {
  return page.evaluate(() => (window as unknown as TestWindow).__testBackend.selectedImportPath())
}

export async function reportImportProgress(
  page: Page,
  rowsRead: number,
  bytesRead: number
): Promise<void> {
  await page.evaluate(([rows, bytes]) => {
    (window as unknown as TestWindow).__testBackend.reportImportProgress(rows, bytes)
  }, [rowsRead, bytesRead])
}

export async function observeSemanticFrames(page: Page): Promise<void> {
  await page.evaluate(() => {
    const root = document.querySelector('[role="grid"]')
    if (!root) throw new Error('semantic chess grid is missing')
    const state = (window as unknown as TestWindow).__testBackend
    const snapshot = () => {
      const labels = [...root.querySelectorAll('[role="gridcell"]')]
        .map((cell) => cell.getAttribute('aria-label') ?? '')
        .filter((label) => !label.startsWith('Empty '))
        .sort()
      const previous = state.semanticFrames.at(-1)
      if (JSON.stringify(previous) !== JSON.stringify(labels)) {
        state.semanticFrames.push(labels)
      }
    }
    let queued = false
    new MutationObserver(() => {
      if (queued) return
      queued = true
      queueMicrotask(() => {
        queued = false
        snapshot()
      })
    }).observe(root, { attributes: true, subtree: true, attributeFilter: ['aria-label'] })
    snapshot()
  })
}
