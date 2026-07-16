import { expect, test } from '@playwright/test'

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    const listeners: Record<string, Array<(payload: unknown) => void>> = {}
    const fen = '4k3/8/8/4p3/8/8/4P3/4K3 w - - 0 2'
    const puzzle = (fingerprint: string, number: number, total: number) => ({
      fingerprint,
      displayedFen: fen,
      currentFen: fen,
      solver: 'white',
      currentPath: [],
      puzzleNumber: number,
      puzzleTotal: total,
      hintLevel: 0,
      incorrectMoves: 0,
      canReveal: false
    })
    const session = (fingerprint = 'guided-1', number = 1) => ({
      sessionId: 'guided-session',
      mode: 'guided',
      status: 'active',
      currentIndex: number - 1,
      total: 2,
      current: puzzle(fingerprint, number, 2)
    })
    let profile: { learnerRating: number; sessionSize: number } | null = null
    let activeSession: ReturnType<typeof session> | Record<string, unknown> | null = null
    let hintLevel = 0
    let importedPath = ''
    let holdImportOpen = false
    let importResult: Record<string, unknown> = {
      jobId: 'job-1', status: 'running',
      progress: { jobId: 'job-1', rowsRead: 0, bytesRead: 0 },
      report: { accepted: 0, duplicates: 0, rejected: 0 }
    }

    const emit = (name: string, payload: unknown) => {
      for (const listener of listeners[name] ?? []) listener(payload)
    }
    ;(window as any).runtime = {
      EventsOnMultiple(name: string, callback: (payload: unknown) => void) {
        listeners[name] = [...(listeners[name] ?? []), callback]
        return () => {
          listeners[name] = (listeners[name] ?? []).filter((value) => value !== callback)
        }
      },
      EventsOff(name: string) { delete listeners[name] },
      EventsOffAll() { for (const name of Object.keys(listeners)) delete listeners[name] }
    }
    ;(window as any).__importTest = {
      holdOpen() { holdImportOpen = true },
      selectedPath() { return importedPath },
      progress(rowsRead: number, bytesRead: number) {
        const progress = { jobId: 'job-1', rowsRead, bytesRead }
        importResult = { ...importResult, progress }
        emit('import:progress', progress)
      }
    }
    ;(window as any).go = { main: { App: {
      GetRecoveryState: async () => ({ required: false }),
      GetProfile: async () => profile,
      UpdateProfile: async (value: { learnerRating: number; sessionSize: number }) => { profile = value },
      ResumeSession: async () => activeSession,
      StartGuided: async () => {
        hintLevel = 0
        activeSession = session()
        return activeSession
      },
      PlayMove: async (_sessionId: string, uci: string) => {
        if (uci !== 'e2e4') {
          return { session: activeSession, correct: false, puzzleCompleted: false, message: 'Try again' }
        }
        activeSession = session('guided-2', 2)
        return { session: activeSession, correct: true, puzzleCompleted: true }
      },
      PauseSession: async () => {},
      UseHint: async () => {
        hintLevel++
        if (hintLevel === 1) {
          return { session: activeSession, level: 1, text: 'Look for: fork', canReveal: false }
        }
        if (hintLevel === 2) {
          return {
            session: activeSession,
            level: 2,
            text: 'Start with this piece.',
            sourceSquare: 'e2',
            canReveal: false
          }
        }
        return {
          session: activeSession,
          level: 3,
          text: 'Try this destination.',
          sourceSquare: 'e2',
          targetSquare: 'e4',
          canReveal: true
        }
      },
      RevealSolution: async () => {
        activeSession = {
          sessionId: 'guided-session', mode: 'guided', status: 'completed', currentIndex: 2, total: 2,
          summary: { total: 2, firstTry: 0, retried: 1, usedHint: 1, revealed: 1, unavailable: 0 }
        }
        return { session: activeSession, correct: true, puzzleCompleted: true }
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
        maximumSolutionPlies: 5
      }),
      GetParentSummary: async () => ({
        learnerRating: profile?.learnerRating ?? 1200,
        ratingTrend: [{ rating: 1150, recordedAt: 1 }, { rating: profile?.learnerRating ?? 1200, recordedAt: 2 }],
        firstAttemptAccuracy: 50,
        hintRate: 25,
        dueReviews: 1,
        themePerformance: [{ theme: 'fork', attempts: 4, accuracy: 50 }],
        recentSessions: [{
          sessionId: 'guided-session', mode: 'guided', status: 'completed', updatedAt: 2,
          total: 2, completed: 2, firstTry: 0, usedHint: 1, revealed: 1
        }]
      }),
      ChoosePuzzleImportFile: async () => '/Users/family/Downloads/lichess_db_puzzle.csv.zst',
      StartLichessImport: async (path: string) => {
        importedPath = path
        importResult = {
          jobId: 'job-1', status: 'running',
          progress: { jobId: 'job-1', rowsRead: 0, bytesRead: 0 },
          report: { accepted: 0, duplicates: 0, rejected: 0 }
        }
        if (holdImportOpen) return 'job-1'
        setTimeout(() => {
          emit('import:progress', { jobId: 'job-1', rowsRead: 10_000, bytesRead: 2048 })
          importResult = {
            jobId: 'job-1', status: 'succeeded',
            report: { accepted: 9800, duplicates: 150, rejected: 50 }
          }
          emit('import:finished', importResult)
        }, 20)
        return 'job-1'
      },
      CancelImport: async () => {
        importResult = {
          jobId: 'job-1', status: 'cancelled', report: { accepted: 0, duplicates: 0, rejected: 0 }
        }
        emit('import:finished', importResult)
      },
      GetImportResult: async () => importResult,
      CreateBackup: async () => '/tmp/backup.zip',
      RestoreBackup: async () => {},
      OpenDataFolder: async () => {},
      Quit: async () => {}
    } } }
  })
})

test('runs setup, import, guided training, practice, and parent progress', async ({ page }) => {
  await page.goto('/')

  await expect(page.getByText('Set up today’s training')).toBeVisible()
  await page.getByLabel('Starting rating').fill('1250')
  await page.getByLabel('Puzzles per session').selectOption('5')
  await page.getByRole('button', { name: 'Save and continue' }).click()
  await expect(page.getByText('What would you like to play?')).toBeVisible()

  await page.getByRole('button', { name: 'Parent settings' }).click()
  await page.getByRole('button', { name: 'Import puzzles' }).click()
  await page.getByRole('button', { name: 'Choose puzzle database' }).click()
  await expect(page.getByText('lichess_db_puzzle.csv.zst', { exact: true })).toBeVisible()
  await expect(page.getByText('/Users/family/Downloads/lichess_db_puzzle.csv.zst')).toBeVisible()
  await page.getByRole('button', { name: 'Import puzzles' }).click()
  await expect(page.getByText('9,800 accepted')).toBeVisible()
  await expect(page.getByText('150 duplicates')).toBeVisible()
  await expect(page.getByText('50 rejected')).toBeVisible()

  await page.getByRole('button', { name: 'Chess Trainer home' }).click()
  await page.getByRole('button', { name: "Start today's training" }).click()
  await page.getByRole('gridcell', { name: 'White pawn on e2' }).click()
  await page.getByRole('gridcell', { name: 'Empty e3' }).click()
  await expect(page.getByText('Try again')).toBeVisible()
  await page.getByRole('gridcell', { name: 'White pawn on e2' }).click()
  await page.getByRole('gridcell', { name: 'Empty e4' }).click()
  await expect(page.getByText('Puzzle 2 of 2')).toBeVisible()

  await page.getByRole('button', { name: 'Pause' }).click()
  await expect(page.getByRole('button', { name: "Continue today's training" })).toBeVisible()
  await page.getByRole('button', { name: "Continue today's training" }).click()
  await page.getByRole('button', { name: 'Hint' }).click()
  await expect(page.getByText('Look for: fork')).toBeVisible()
  await page.getByRole('button', { name: 'Hint' }).click()
  await page.getByRole('button', { name: 'Hint' }).click()
  await expect(page.getByRole('button', { name: 'Show solution' })).toBeVisible()
  await page.getByRole('button', { name: 'Show solution' }).click()
  await expect(page.getByText('Training complete!')).toBeVisible()
  await expect(page.getByText('1 retried')).toBeVisible()
  await expect(page.getByText('1 solution shown')).toBeVisible()
  await page.getByRole('button', { name: 'Back home' }).click()

  await page.getByRole('button', { name: 'Free Practice' }).click()
  await page.getByLabel('Limit by rating').check()
  await page.getByLabel('Minimum rating').fill('1100')
  await page.getByLabel('Maximum rating').fill('1500')
  await page.getByLabel('fork').check()
  await page.getByLabel('Maximum solution length').selectOption('3')
  await page.getByRole('button', { name: 'Start practice' }).click()
  await expect(page.getByText('Find the best move')).toBeVisible()

  await page.getByRole('button', { name: 'Chess Trainer home' }).click()
  await page.getByRole('button', { name: 'Parent settings' }).click()
  const metrics = page.locator('.metric-grid')
  await expect(metrics.getByText('50%', { exact: true })).toBeVisible()
  await expect(metrics.getByText('25%', { exact: true })).toBeVisible()
  await expect(page.getByRole('cell', { name: 'fork' })).toBeVisible()
  await expect(page.getByRole('cell', { name: 'Guided' })).toBeVisible()
})

test('keeps an active import observable across navigation and can cancel it', async ({ page }) => {
  await page.goto('/')
  await page.getByLabel('Starting rating').fill('1250')
  await page.getByRole('button', { name: 'Save and continue' }).click()
  await page.evaluate(() => (window as any).__importTest.holdOpen())

  await page.getByRole('button', { name: 'Parent settings' }).click()
  await page.getByRole('button', { name: 'Import puzzles' }).click()
  await page.getByRole('button', { name: 'Choose puzzle database' }).click()
  await page.getByRole('button', { name: 'Import puzzles' }).click()
  await expect.poll(() => page.evaluate(() => (window as any).__importTest.selectedPath()))
    .toBe('/Users/family/Downloads/lichess_db_puzzle.csv.zst')

  await page.getByRole('button', { name: 'Chess Trainer home' }).click()
  await page.evaluate(() => (window as any).__importTest.progress(10_000, 2048))
  await page.getByRole('button', { name: 'Parent settings' }).click()
  await page.getByRole('button', { name: 'Import puzzles' }).click()

  await expect(page.getByText('10,000 rows read')).toBeVisible()
  await page.getByRole('button', { name: 'Cancel import' }).click()
  await expect(page.getByText('Import cancelled.')).toBeVisible()
})
