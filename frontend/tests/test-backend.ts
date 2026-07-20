import type { Page } from '@playwright/test'
import type {
  BoardScenario, NextResponse,
  ModeControllerMock,
  NormalControllerMock,
  PuzzleDefinition,
  RecoveryControllerMock,
  ResponseDefinition,
  TestBackendScenario,
  TestWindow,
  TrustedInput,
  WireImportResult,
  WireOpeningActivity,
  WireOpeningSession,
  WirePuzzle,
  WireSession
} from './test-backend-types'

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
      openingMoves: [] as string[],
      openingHints: [] as number[],
      openingDepths: [] as string[],
      holdImportOpen: () => {},
      selectedImportPath: () => '',
      selectedCoursePath: () => '',
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
      status: 'cancelled',
      progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 0 },
      report: { accepted: 0, duplicates: 0, rejected: 0, examples: [] }
    }
    const normalController: NormalControllerMock = {
      AdvanceOpeningActivity: async () => { throw new Error('test backend has no opening activity') },
      AdvanceOpeningStep: async () => { throw new Error('test backend has no opening step') },
      CancelImport: async () => {},
      ChooseOpeningCourseFile: async () => '',
      ChoosePuzzleImportFile: async () => '',
      CreateBackup: async () => '',
      GetImportResult: async () => emptyImportResult,
      GetOpeningHome: async () => ({ courses: [] }),
      GetOpeningPosition: async (courseId: string, positionId: string) => ({
        courseId,
        positionId,
        fen: '8/8/8/8/8/8/8/8 w - - 0 1',
        label: 'Test position',
        evaluation: { code: 'none' },
        notes: [],
        moves: [],
        incomingPaths: 0
      }),
      GetParentSummary: async () => ({
        learnerRating: 1200, ratingTrend: [], firstAttemptAccuracy: 0, hintRate: 0,
        dueReviews: 0, themePerformance: [], recentSessions: []
      }),
      GetPracticeFilters: async () => ({
        sources: [], themes: [], maximumSolutionPlies: 1,
        learnerRatingBounds: { minimum: 800, maximum: 2200 }
      }),
      GetProfile: async () => ({ learnerRating: 1200, sessionSize: 5 }),
      InspectOpeningCourseImport: async () => ({
        path: '', filename: '', format: 'coursepack', formatLabel: 'Opening course', sourceId: '',
        sourceIdOrigin: 'embedded', replacesExisting: false
      }),
      InspectPuzzleImport: async () => ({
        path: '', filename: '', format: 'tactical-pgn', formatLabel: 'Tactical PGN', sourceId: '',
        sourceIdOrigin: 'embedded', replacesExisting: false
      }),
      OpenDataFolder: async () => {},
      PauseOpeningSession: async () => {},
      PauseSession: async () => {},
      PlayOpeningMove: async () => { throw new Error('test backend has no opening move') },
      PlayMove: async () => { throw new Error('test backend has no move response') },
      Quit: async () => {},
      RestoreBackup: async () => {},
      RestartOpeningSession: async () => { throw new Error('test backend has no opening restart') },
      ResumeOpeningSession: async () => null,
      ResumeSession: async () => null,
      RevealOpeningMove: async () => { throw new Error('test backend has no opening reveal') },
      RevealSolution: async () => { throw new Error('test backend has no reveal response') },
      SetOpeningDepth: async () => {},
      StartFreePractice: async () => { throw new Error('test backend has no practice session') },
      StartGuided: async () => { throw new Error('test backend has no guided session') },
      StartOpeningCourseImport: async () => 'unused',
      StartOpeningLesson: async () => { throw new Error('test backend has no opening lesson') },
      StartOpeningReview: async () => { throw new Error('test backend has no opening review') },
      StartPuzzleImport: async () => 'unused',
      UpdateProfile: async () => {},
      UseOpeningHint: async () => { throw new Error('test backend has no opening hint') },
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
    let installedNormalController = normalController
    const installNormalController = (overrides: Partial<NormalControllerMock>) => {
      installedNormalController = { ...installedNormalController, ...overrides }
      testWindow.go = { main: {
        ModeController: modeController,
        NormalController: installedNormalController,
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
    const chooserImportPath = '/Users/family/Downloads/../Puzzles/club-tactics.pgn'
    const importPath = '/Users/family/Puzzles/club-tactics.pgn'
    const importBytes = 4096
    const importInspection = {
      path: importPath,
      filename: 'club-tactics.pgn',
      format: 'tactical-pgn',
      formatLabel: 'Tactical PGN',
      sourceId: 'club-tactics',
      sourceIdOrigin: 'embedded',
      replacesExisting: false
    }
    let importResult: WireImportResult = {
      jobId: 'job-1',
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
      ChoosePuzzleImportFile: async () => chooserImportPath,
      InspectPuzzleImport: async (path: string) => {
        if (path !== chooserImportPath) throw new Error(`unexpected inspection path ${path}`)
        return importInspection
      },
      StartPuzzleImport: async (inspection) => {
        if (inspection.path !== importPath || inspection.sourceId !== 'club-tactics') {
          throw new Error(`unexpected import inspection ${JSON.stringify(inspection)}`)
        }
        importedPath = inspection.path
        importResult = {
          jobId: 'job-1',
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

    if (configured.kind === 'openings') {
      const initialFen = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'
      const promptFen = 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4'
      const afterC3Fen = 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'
      const watchFrames = [
        { uci: 'e2e4', resultingFen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1' },
        { uci: 'e7e5', resultingFen: 'rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2' },
        { uci: 'g1f3', resultingFen: 'rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2' },
        { uci: 'b8c6', resultingFen: 'r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3' },
        { uci: 'f1c4', resultingFen: 'r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3' },
        { uci: 'f8c5', resultingFen: promptFen }
      ]
      const courseChooserPath = '/Users/family/Documents/synthetic-italian.ctcourse'
      const courseInspection = {
        path: courseChooserPath,
        filename: 'synthetic-italian.ctcourse',
        format: 'coursepack',
        formatLabel: 'Opening course',
        sourceId: 'synthetic-italian',
        sourceIdOrigin: 'embedded',
        sourceName: 'Synthetic Italian for White',
        replacesExisting: false
      } as const
      let selectedCourse = ''
      let courseImported = false
      const completedLessonIds = new Set<string>()
      let reviewCompleted = false
      let openingDepth = 'standard'
      let openingLessonId = 'giuoco-c3'
      const openingActivityIndexes: Record<string, number> = {
        'giuoco-c3': 0,
        'giuoco-d3': 0
      }
      let openingHintLevel = 0
      let openingMode: 'lesson' | 'review' = 'lesson'
      let openingSession: WireOpeningSession | null = null
      let courseImportResult: WireImportResult = {
        jobId: 'course-job-1',
        status: 'running',
        progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 8192 },
        report: { accepted: 0, duplicates: 0, rejected: 0, examples: [], counts: {} }
      }

      const referenceSections = () => openingDepth === 'reference' ? [{
        activityId: 'giuoco-reference',
        title: 'Why c3 is flexible',
        instruction: 'Compare the immediate centre occupation with the quieter setup.',
        positionId: 'after-bc5',
        noteTexts: ['Reference analysis: c3 supports d4 while preserving the active bishop.'],
        annotations: [{ kind: 'arrow', from: 'c2', to: 'c3' }]
      }] : []
      const lessonActivities = (lessonId: string): WireOpeningActivity[] => {
        if (lessonId === 'giuoco-d3') {
          return [{
            activityId: 'giuoco-d3-decision', kind: 'decision', title: 'Choose the quiet setup',
            instruction: 'Play the move that supports e4 and keeps the centre compact.',
            variationName: 'Quiet Italian', required: true,
            positionId: 'after-bc5', currentFen: promptFen, orientation: 'white',
            legalMoves: ['c2c3', 'd2d3'], teachingNoteTexts: ['Use d3 when you want to develop before opening the centre.'],
            referenceNoteTexts: [], comparison: [],
            annotations: [{ kind: 'square', from: 'd3' }], movesToHere: [],
            completedIdeas: 0, requiredIdeas: 1, referenceSections: [],
            activityNumber: 1, activityTotal: 1, hintLevel: openingHintLevel,
            canReveal: openingHintLevel >= 4
          }]
        }
        const common = {
          orientation: 'white' as const,
          required: true,
          referenceNoteTexts: [] as string[],
          comparison: [],
          movesToHere: [],
          activityTotal: 3,
          requiredIdeas: 3,
          hintLevel: openingHintLevel,
          canReveal: openingHintLevel >= 4
        }
        return [
          {
            ...common, activityId: 'giuoco-concept', kind: 'concept', title: 'Build the centre',
            instruction: 'White prepares d4 without blocking the active bishop.',
            positionId: 'after-bc5', currentFen: promptFen, legalMoves: [],
            teachingNoteTexts: [
              'Connect c3 with the later d4 break; this is one plan, not a move drill.',
              ...Array.from(
                { length: 16 },
                (_, index) => `Extended lesson detail ${index + 1}: compare the central plan before continuing.`
              )
            ],
            annotations: [{ kind: 'arrow', from: 'c2', to: 'c3' }],
            referenceSections: referenceSections(), activityNumber: 1, completedIdeas: 0
          },
          {
            ...common, activityId: 'giuoco-c3-decision', kind: 'decision', title: 'Choose the preparation',
            instruction: 'Play the move that supports a later d4.', variationName: 'Giuoco Piano',
            positionId: 'after-bc5', currentFen: promptFen,
            legalMoves: ['b2b4', 'c2c3', 'd2d3'],
            teachingNoteTexts: ['This is the lesson\'s only c3 decision.'],
            annotations: [], referenceSections: [], activityNumber: 2, completedIdeas: 1
          },
          {
            ...common, activityId: 'giuoco-recap', kind: 'recap', title: 'Keep the plan',
            instruction: 'Develop, prepare d4, then choose the right moment to occupy the centre.',
            positionId: 'after-c3', currentFen: afterC3Fen, legalMoves: [],
            teachingNoteTexts: ['You learned the purpose of c3; you do not need to replay it again here.'],
            annotations: [{ kind: 'square', from: 'd4' }], referenceSections: [],
            activityNumber: 3, completedIdeas: 2
          }
        ]
      }
      const openingActivity = (
        lessonId: string,
        index: number,
        mode: 'lesson' | 'review'
      ): WireOpeningActivity => {
        if (mode === 'review') {
          return {
            activityId: 'review-recall-c3', kind: 'decision', title: 'Review the Giuoco Piano',
            instruction: 'Play the course move from memory.', variationName: 'Giuoco Piano',
            required: true,
            positionId: 'after-bc5', currentFen: promptFen, orientation: 'white',
            legalMoves: ['b2b4', 'c2c3', 'd2d3'], teachingNoteTexts: [],
            referenceNoteTexts: ['The quiet c3 move supports a later d4.'],
            comparison: [], annotations: [], movesToHere: [], completedIdeas: 0,
            requiredIdeas: 1, referenceSections: [], activityNumber: 1,
            activityTotal: 1, hintLevel: openingHintLevel, canReveal: openingHintLevel >= 4
          }
        }
        return lessonActivities(lessonId)[index]
      }
      const activeOpening = (
        lessonId = openingLessonId,
        index = openingActivityIndexes[lessonId],
        mode: 'lesson' | 'review' = openingMode
      ): WireOpeningSession => ({
        sessionId: mode === 'review' ? 'opening-review-session' : 'opening-lesson-session',
        mode,
        status: 'active',
        courseId: 'synthetic-italian',
        generationId: 'synthetic-generation-1',
        lessonId: mode === 'review' ? 'review-c3' : lessonId,
        courseTitle: 'Synthetic Italian for White',
        path: mode === 'review' ? [] : lessonPath(lessonId),
        depth: openingDepth,
        current: openingActivity(lessonId, index, mode)
      })
      const completedOpening = (
        mode: 'lesson' | 'review',
        lessonId = openingLessonId
      ): WireOpeningSession => ({
        sessionId: mode === 'review' ? 'opening-review-session' : 'opening-lesson-session',
        mode, status: 'completed', courseId: 'synthetic-italian',
        generationId: 'synthetic-generation-1',
        lessonId: mode === 'review' ? 'review-c3' : lessonId,
        courseTitle: 'Synthetic Italian for White',
        path: mode === 'review' ? [] : lessonPath(lessonId), depth: openingDepth,
        ...(mode === 'review' ? { summary: {
          totalPrompts: 1,
          positionsRecalled: 1,
          branchesRecognized: 0,
          retried: 0,
          usedHint: 0,
          revealed: 0
        } } : {})
      })
      const lessonPath = (lessonId: string) => lessonId === 'giuoco-d3'
        ? [
            { lessonId: 'giuoco-c3', title: 'Prepare d4 with c3' },
            { lessonId: 'giuoco-d3', title: 'Choose the quiet d3 setup' }
          ]
        : [{ lessonId: 'giuoco-c3', title: 'Prepare d4 with c3' }]
      const lessonProgress = (lessonId: string) => completedLessonIds.has(lessonId)
        ? 'completed'
        : openingSession?.status === 'active' && openingMode === 'lesson' && openingLessonId === lessonId
          ? 'in_progress'
          : 'available'
      const openingHome = () => ({
        courses: courseImported ? [{
          courseId: 'synthetic-italian',
          title: 'Synthetic Italian for White',
          perspective: 'white',
          depth: openingDepth,
          rootPositionId: 'initial',
          completedLessons: Number(completedLessonIds.has('giuoco-c3')) +
            Number(openingDepth !== 'quick' && completedLessonIds.has('giuoco-d3')),
          totalLessons: openingDepth === 'quick' ? 1 : 2,
          dueReviews: completedLessonIds.has('giuoco-c3') && !reviewCompleted ? 1 : 0,
          nextLessonId: completedLessonIds.has('giuoco-c3') ? 'giuoco-d3' : 'giuoco-c3',
          nextLessonTitle: completedLessonIds.has('giuoco-c3')
            ? 'Choose the quiet d3 setup'
            : 'Prepare d4 with c3',
          currentLessonId: openingSession?.status === 'active' && openingMode === 'lesson'
            ? openingLessonId
            : undefined,
          currentActivityId: openingSession?.status === 'active'
            ? openingSession.current?.activityId
            : undefined,
          currentPath: openingSession?.status === 'active' && openingMode === 'lesson'
            ? lessonPath(openingLessonId)
            : [],
          recommendedLessonId: completedLessonIds.has('giuoco-c3') ? 'giuoco-d3' : 'giuoco-c3',
          recommendedLessonTitle: completedLessonIds.has('giuoco-c3')
            ? 'Choose the quiet d3 setup'
            : 'Prepare d4 with c3',
          hasResumable: openingSession?.status === 'active' && openingMode === 'lesson',
          hasResumableReview: openingSession?.status === 'active' && openingMode === 'review',
          tree: {
            rootLessonId: 'giuoco-c3',
            nodes: [
              {
                lessonId: 'giuoco-c3', chapterId: 'giuoco', title: 'Prepare d4 with c3',
                objective: 'Connect c3 with the later d4 break.', minimumDepth: 'quick',
                progress: lessonProgress('giuoco-c3'),
                completedActivities: completedLessonIds.has('giuoco-c3')
                  ? 3
                  : openingActivityIndexes['giuoco-c3'],
                requiredActivities: 3,
                recommended: !completedLessonIds.has('giuoco-c3'),
                reviewDue: completedLessonIds.has('giuoco-c3') && !reviewCompleted,
                visible: true
              },
              {
                lessonId: 'giuoco-d3', chapterId: 'quiet-italian',
                title: 'Choose the quiet d3 setup',
                objective: 'Recognize when to stabilize e4 before opening the centre.',
                minimumDepth: 'standard', progress: lessonProgress('giuoco-d3'),
                completedActivities: completedLessonIds.has('giuoco-d3')
                  ? 1
                  : openingActivityIndexes['giuoco-d3'],
                requiredActivities: 1,
                recommended: completedLessonIds.has('giuoco-c3') && !completedLessonIds.has('giuoco-d3'),
                reviewDue: false, visible: openingDepth !== 'quick'
              }
            ],
            edges: [{
              edgeId: 'c3-to-d3', fromLessonId: 'giuoco-c3', toLessonId: 'giuoco-d3',
              ordinal: 1, kind: 'continuation', label: 'Keep the centre flexible',
              minimumDepth: 'standard'
            }]
          },
          chapters: [
            {
              chapterId: 'giuoco', title: 'Giuoco Piano',
              lessons: [{
                lessonId: 'giuoco-c3', title: 'Prepare d4 with c3',
                completedSteps: completedLessonIds.has('giuoco-c3') ? 3 : openingActivityIndexes['giuoco-c3'],
                totalSteps: 3, completed: completedLessonIds.has('giuoco-c3')
              }]
            },
            {
              chapterId: 'quiet-italian', title: 'Quiet Italian',
              lessons: [{
                lessonId: 'giuoco-d3', title: 'Choose the quiet d3 setup',
                completedSteps: completedLessonIds.has('giuoco-d3') ? 1 : openingActivityIndexes['giuoco-d3'],
                totalSteps: 1, completed: completedLessonIds.has('giuoco-d3')
              }]
            }
          ]
        }] : []
      })
      const requireOpening = (): WireOpeningSession => {
        if (!openingSession || openingSession.status !== 'active') {
          throw new Error('test backend has no active opening session')
        }
        return openingSession
      }
      const checkpointFor = (lessonId: string) => ({
        completedLessonId: lessonId,
        path: lessonPath(lessonId),
        availableLessonIds: lessonId === 'giuoco-c3' ? ['giuoco-d3'] : [],
        ...(lessonId === 'giuoco-c3' ? {
          recommendedLessonId: 'giuoco-d3',
          recommendedLessonTitle: 'Choose the quiet d3 setup'
        } : {}),
        completedLessons: completedLessonIds.size,
        totalLessons: openingDepth === 'quick' ? 1 : 2
      })
      const completeActivity = (appliedMove?: { uci: string; resultingFen: string }) => {
        requireOpening()
        if (openingMode === 'review') {
          reviewCompleted = true
          openingSession = completedOpening('review')
        } else {
          const nextIndex = openingActivityIndexes[openingLessonId] + 1
          if (nextIndex >= lessonActivities(openingLessonId).length) {
            completedLessonIds.add(openingLessonId)
            openingActivityIndexes[openingLessonId] = nextIndex
            openingSession = completedOpening('lesson', openingLessonId)
          } else {
            openingActivityIndexes[openingLessonId] = nextIndex
            openingSession = activeOpening(openingLessonId, nextIndex, 'lesson')
          }
        }
        const checkpoint = openingMode === 'lesson' && openingSession.status === 'completed'
          ? checkpointFor(openingLessonId)
          : undefined
        return {
          session: clone(openingSession),
          activityCompleted: true,
          ...(appliedMove ? {
            feedback: 'expected',
            appliedMoves: [appliedMove],
            finalFen: appliedMove.resultingFen
          } : {}),
          ...(checkpoint ? { checkpoint } : {})
        }
      }

      state.selectedCoursePath = () => selectedCourse

      installNormalController({
        ChooseOpeningCourseFile: async () => courseChooserPath,
        InspectOpeningCourseImport: async (path: string) => {
          if (path !== courseChooserPath) throw new Error(`unexpected course path ${path}`)
          return courseInspection
        },
        StartOpeningCourseImport: async (inspection) => {
          if (inspection.path !== courseChooserPath || inspection.sourceId !== 'synthetic-italian') {
            throw new Error(`unexpected course inspection ${JSON.stringify(inspection)}`)
          }
          selectedCourse = inspection.path
          courseImportResult = {
            jobId: 'course-job-1',
            status: 'running',
            progress: { phase: 'detecting', rowsRead: 0, bytesRead: 0, totalBytes: 8192 },
            report: { accepted: 0, duplicates: 0, rejected: 0, examples: [], counts: {} }
          }
          setTimeout(() => {
            courseImported = true
            courseImportResult = {
              ...courseImportResult,
              status: 'succeeded',
              progress: { phase: 'activating', rowsRead: 84, bytesRead: 8192, totalBytes: 8192 },
              report: {
                accepted: 1, duplicates: 0, rejected: 0, examples: [],
                counts: { chapters: 2, moves: 42, lessons: 2, activities: 4, lessonEdges: 1 }
              }
            }
            emit('import:finished', courseImportResult)
          }, 20)
          return 'course-job-1'
        },
        GetImportResult: async (jobId: string) => jobId === 'course-job-1'
          ? courseImportResult
          : importResult,
        GetOpeningHome: async () => clone(openingHome()),
        SetOpeningDepth: async (_courseId: string, depth: string) => {
          openingDepth = depth
          state.openingDepths.push(depth)
          if (openingSession?.status === 'active') {
            openingSession = activeOpening()
          }
        },
        StartOpeningLesson: async (_courseId: string, lessonId: string) => {
          if (lessonId !== 'giuoco-c3' && lessonId !== 'giuoco-d3') {
            throw new Error(`unexpected lesson ${lessonId}`)
          }
          openingMode = 'lesson'
          openingLessonId = lessonId
          openingActivityIndexes[lessonId] = 0
          openingHintLevel = 0
          openingSession = activeOpening(lessonId, 0, 'lesson')
          return clone(openingSession)
        },
        ResumeOpeningSession: async () => openingSession ? clone(openingSession) : null,
        RestartOpeningSession: async () => {
          openingActivityIndexes[openingLessonId] = 0
          openingHintLevel = 0
          openingSession = activeOpening(openingLessonId, 0, openingMode)
          return clone(openingSession)
        },
        AdvanceOpeningActivity: async () => {
          const activeSession = requireOpening()
          if (activeSession.current?.kind === 'decision') {
            throw new Error('opening step requires a learner move')
          }
          const watched = activeSession.current?.kind === 'demonstration'
          const result = completeActivity()
          return watched
            ? { ...result, appliedMoves: watchFrames, finalFen: promptFen }
            : result
        },
        PlayOpeningMove: async (_sessionId: string, uci: string) => {
          state.openingMoves.push(uci)
          if (openingMode === 'review' && uci === 'c2c3') {
            return completeActivity({ uci, resultingFen: afterC3Fen })
          }
          if (openingLessonId === 'giuoco-c3' && uci === 'c2c3') {
            return completeActivity({ uci, resultingFen: afterC3Fen })
          }
          if (openingLessonId === 'giuoco-d3' && uci === 'd2d3') {
            return completeActivity({
              uci,
              resultingFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/3P1N2/PPP2PPP/RNBQK2R b KQkq - 0 4'
            })
          }
          if (uci === 'b2b4') {
            return {
              session: clone(requireOpening()),
              activityCompleted: false,
              feedback: 'alternative',
              message: 'Playable alternative. Return to the lesson position.'
            }
          }
          if (openingLessonId === 'giuoco-c3' && uci === 'd2d3') {
            return {
              session: clone(requireOpening()),
              activityCompleted: false,
              feedback: 'off_course',
              message: 'This lesson is practising c3.'
            }
          }
          throw new Error(`unexpected opening move ${uci}`)
        },
        UseOpeningHint: async () => {
          requireOpening()
          openingHintLevel = Math.min(4, openingHintLevel + 1)
          state.openingHints.push(openingHintLevel)
          openingSession = activeOpening()
          const text = [
            '',
            'Plan: prepare d4 while keeping the centre flexible.',
            'Start with the c-pawn.',
            'Move it one square.',
            'The course move is ready to show.'
          ][openingHintLevel]
          return {
            session: clone(openingSession),
            level: openingHintLevel,
            text,
            ...(openingHintLevel >= 2 ? { sourceSquare: 'c2' } : {}),
            ...(openingHintLevel >= 3 ? { targetSquare: 'c3' } : {}),
            canReveal: openingHintLevel >= 4
          }
        },
        RevealOpeningMove: async () => openingLessonId === 'giuoco-d3'
          ? completeActivity({
              uci: 'd2d3',
              resultingFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/3P1N2/PPP2PPP/RNBQK2R b KQkq - 0 4'
            })
          : completeActivity({ uci: 'c2c3', resultingFen: afterC3Fen }),
        PauseOpeningSession: async () => {},
        StartOpeningReview: async () => {
          openingMode = 'review'
          openingHintLevel = 0
          openingSession = activeOpening(openingLessonId, 0, 'review')
          return clone(openingSession)
        },
        GetOpeningPosition: async (courseId: string, positionId: string) => {
          if (courseId !== 'synthetic-italian') throw new Error(`unexpected course ${courseId}`)
          if (positionId === 'initial') {
            return {
              courseId, positionId, fen: initialFen, label: 'Initial position',
              evaluation: { code: 'none' }, notes: [], incomingPaths: 0,
              moves: [{
                moveId: 'white-e4', uci: 'e2e4', san: 'e4',
                toPositionId: 'after-e4', role: 'repertoire',
                variationName: 'Italian setup', evaluation: { code: 'equal' },
                sourceRef: { printedPage: 1, coverageId: 'synthetic-p1-e4' }
              }]
            }
          }
          if (positionId === 'after-e4') {
            return {
              courseId, positionId,
              fen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1',
              label: 'King pawn opening', evaluation: { code: 'equal' },
              notes: [{
                kind: 'overview', text: 'Black can now contest the centre.',
                sourceRef: {
                  printedPage: 1, noteLabel: 'overview', coverageId: 'synthetic-p1-overview'
                }
              }],
              moves: [], incomingPaths: 1
            }
          }
          throw new Error(`unexpected opening position ${positionId}`)
        }
      })
    }
  }, scenario)
}
