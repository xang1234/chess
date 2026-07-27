import type {
  ActiveOpeningSessionView,
  AppliedMoveFrames,
  CompletedOpeningSessionView,
  NormalAPI,
  OpeningHintResult,
  OpeningSessionView,
  OpeningActivityResult
} from '../../lib/api'
import type { SoundService } from '../../lib/sound'
import { fakeAPI, fakeOpeningSession } from '../../test-fakes'
import type { BoardEffects } from '../chess/board-effects'
import {
  createOpeningController,
  type OpeningBoardPort,
  type OpeningControllerView
} from './opening-controller'

const startFen = fakeOpeningSession.current.currentFen
const afterC3 = 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'
const afterD6 = 'r1bqk1nr/ppp2ppp/2np4/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R w KQkq - 0 5'
const comparisonLines: ActiveOpeningSessionView['current']['comparison'] = [
  {
    label: 'Active centre',
    moves: ['c3', 'd4'],
    frames: [
      { uci: 'c2c3', resultingFen: afterC3 },
      { uci: 'd7d6', resultingFen: afterD6 }
    ]
  },
  {
    label: 'Quiet setup',
    moves: ['d3'],
    frames: [{ uci: 'd2d3', resultingFen: afterC3 }]
  }
]

function active(
  kind: ActiveOpeningSessionView['current']['kind'] = 'decision',
  overrides: Partial<ActiveOpeningSessionView['current']> = {}
): ActiveOpeningSessionView {
  return {
    ...fakeOpeningSession,
    current: {
      ...fakeOpeningSession.current,
      kind,
      legalMoves: kind === 'decision' ? ['c2c3'] : [],
      ...overrides
    }
  }
}

function completed(): CompletedOpeningSessionView {
  return {
    sessionId: fakeOpeningSession.sessionId,
    mode: 'lesson',
    status: 'completed',
    courseId: fakeOpeningSession.courseId,
    generationId: fakeOpeningSession.generationId,
    lessonId: fakeOpeningSession.lessonId,
    courseTitle: fakeOpeningSession.courseTitle,
    path: fakeOpeningSession.path,
    depth: fakeOpeningSession.depth
  }
}

function expected(
  session: OpeningSessionView,
  appliedMoves: AppliedMoveFrames = [{ uci: 'c2c3', resultingFen: afterC3 }],
  finalFen = afterC3
): OpeningActivityResult {
  return {
    session,
    activityCompleted: true,
    feedback: 'expected',
    message: 'Course move found.',
    appliedMoves,
    finalFen
  }
}

function sound(): SoundService {
  return {
    muted: false,
    unlock: async () => {},
    play: vi.fn(),
    setMuted: vi.fn(),
    toggleMuted: () => false,
    destroy: vi.fn()
  }
}

function effects(reducedMotion = false): BoardEffects {
  return {
    createSound: sound,
    delay: vi.fn(async () => {}),
    prefersReducedMotion: () => reducedMotion
  }
}

function board(implementation?: (fen: string) => void): OpeningBoardPort & {
  setPosition: ReturnType<typeof vi.fn>
} {
  return {
    setPosition: vi.fn((fen: string) => implementation?.(fen))
  }
}

function harness(
  session: OpeningSessionView,
  overrides: Partial<NormalAPI> = {},
  boardEffects = effects(false),
  boardPort = board(),
  order: string[] = []
) {
  const changes: OpeningSessionView[] = []
  const persisted: OpeningSessionView[] = []
  const homes: boolean[] = []
  const views: OpeningControllerView[] = []
  const controller = createOpeningController({
    api: fakeAPI(overrides),
    effects: boardEffects,
    afterRender: async () => {},
    events: {
      change: (next) => { changes.push(next); order.push('change') },
      persisted: (next) => { persisted.push(next); order.push('persisted') },
      home: (done) => homes.push(done)
    }
  })
  controller.subscribe((view) => views.push(view))
  controller.mount(session)
  controller.attachBoard(boardPort)
  return { board: boardPort, changes, controller, homes, persisted, views }
}

function deferred<Value>() {
  let resolve!: (value: Value) => void
  const promise = new Promise<Value>((accept) => { resolve = accept })
  return { promise, resolve }
}

test('advances a zero-frame teaching result immediately and orders session events', async () => {
  const explain = active('concept')
  const watch = active('demonstration', {
    activityId: 'watch-c3', activityNumber: 2, title: 'Watch c3'
  })
  const result: OpeningActivityResult = { session: watch, activityCompleted: true }
  const order: string[] = []
  const subject = harness(
    explain,
    { advanceOpeningActivity: async () => result },
    effects(false),
    board(),
    order
  )

  await subject.controller.advance()

  expect(subject.controller.view.state).toMatchObject({
    phase: 'passive', session: watch
  })
  expect(subject.controller.view.message).toBe('')
  expect(subject.controller.view.announcement).toBe('Next idea: Watch c3.')
  expect(subject.persisted).toEqual([watch])
  expect(subject.changes).toEqual([watch])
  expect(order).toEqual(['persisted', 'change'])
  expect(subject.views.map((view) => view.state.phase)).not.toContain('activity-complete')
  expect(subject.views.map((view) => view.state.phase)).not.toContain('animating')
})

test('preserves the board generation for the same FEN and replaces it for a different FEN', async () => {
  const explain = active('concept')
  const same = harness(explain, {
    advanceOpeningActivity: async () => ({
      session: active('recap', {
        activityId: 'same-fen',
        currentFen: explain.current.currentFen
      }),
      activityCompleted: true
    })
  })
  const sameGeneration = same.controller.view.boardGeneration
  await same.controller.advance()
  expect(same.controller.view.boardGeneration).toBe(sameGeneration)

  const different = harness(explain, {
    advanceOpeningActivity: async () => ({
      session: active('recap', { activityId: 'different-fen', currentFen: afterC3 }),
      activityCompleted: true
    })
  })
  const differentGeneration = different.controller.view.boardGeneration
  await different.controller.advance()
  expect(different.controller.view.boardGeneration).toBe(differentGeneration + 1)
})

test('adopts an external same-FEN activity without replacing the board', () => {
  const current = active('concept')
  const subject = harness(current)
  const generation = subject.controller.view.boardGeneration

  subject.controller.receiveSession(active('recap', {
    activityId: 'external',
    currentFen: current.current.currentFen
  }))

  expect(subject.controller.view.boardGeneration).toBe(generation)
})

test('retries a failed passive activity save without leaving the activity', async () => {
  const concept = active('concept')
  const next = active('recap', { activityId: 'recap', activityNumber: 2 })
  const advanceOpeningActivity = vi.fn()
    .mockRejectedValueOnce(new Error('save failed'))
    .mockResolvedValueOnce({ session: next, activityCompleted: true })
  const subject = harness(concept, { advanceOpeningActivity })

  await subject.controller.advance()
  expect(subject.controller.view.state).toMatchObject({
    phase: 'failed',
    session: concept,
    retryOperation: 'advance'
  })

  await subject.controller.retry()
  expect(advanceOpeningActivity).toHaveBeenCalledTimes(2)
  expect(subject.controller.view.state).toMatchObject({
    phase: 'passive',
    session: next
  })
})

test('can pause a lesson after a recoverable activity save failure', async () => {
  const pauseOpeningSession = vi.fn(async () => {})
  const subject = harness(active('concept'), {
    advanceOpeningActivity: async () => { throw new Error('save failed') },
    pauseOpeningSession
  })

  await subject.controller.advance()
  await subject.controller.pause()

  expect(pauseOpeningSession).toHaveBeenCalledWith(fakeOpeningSession.sessionId)
  expect(subject.homes).toEqual([false])
})

test('animates every authoritative watch frame', async () => {
  const watch = active('demonstration')
  const next = active('decision', { activityId: 'try-c3', activityNumber: 2, currentFen: afterD6 })
  const result: OpeningActivityResult = {
    session: next,
    activityCompleted: true,
    appliedMoves: [
      { uci: 'c2c3', resultingFen: afterC3 },
      { uci: 'd7d6', resultingFen: afterD6 }
    ],
    finalFen: afterD6
  }
  const subject = harness(watch, { advanceOpeningActivity: async () => result })

  await subject.controller.advance()

  expect(subject.board.setPosition.mock.calls).toEqual([
    [afterC3, ['c2', 'c3'], true],
    [afterD6, ['d7', 'd6'], true]
  ])
})

test('replays moves to here and a completed demonstration without changing progress', async () => {
  const movesToHere: AppliedMoveFrames = [
    { uci: 'e2e4', resultingFen: afterC3 },
    { uci: 'e7e5', resultingFen: afterD6 }
  ]
  const current = active('demonstration', { movesToHere })
  const next = active('decision', { currentFen: afterD6 })
  const subject = harness(current, {
    advanceOpeningActivity: async () => ({
      session: next,
      activityCompleted: true,
      appliedMoves: movesToHere,
      finalFen: afterD6
    })
  })

  await subject.controller.replayMovesToHere()
  expect(subject.persisted).toEqual([])
  subject.board.setPosition.mockClear()

  await subject.controller.advance()
  subject.board.setPosition.mockClear()
  await subject.controller.replayDemonstration()

  expect(subject.board.setPosition).toHaveBeenCalledTimes(2)
  expect(subject.persisted).toEqual([next])
  expect(subject.controller.view.state.phase).toBe('activity-complete')
})

test('previews a comparison without persisting and can reset', async () => {
  const current = active('comparison', { comparison: comparisonLines })
  const subject = harness(current)

  await subject.controller.previewComparison(0)

  expect(subject.board.setPosition.mock.calls).toEqual([
    [current.current.currentFen, undefined, false],
    [afterC3, ['c2', 'c3'], true],
    [afterD6, ['d7', 'd6'], true]
  ])
  expect(subject.controller.view.comparisonPreviewIndex).toBe(0)
  expect(subject.controller.view.comparisonPreviewPlaying).toBe(false)
  expect(subject.persisted).toEqual([])

  await subject.controller.resetComparisonPreview()

  expect(subject.board.setPosition).toHaveBeenLastCalledWith(
    current.current.currentFen,
    undefined,
    false
  )
  expect(subject.controller.view.comparisonPreviewIndex).toBeNull()
})

test('a second comparison selection cancels a deferred first preview', async () => {
  const firstDelay = deferred<void>()
  const delay = vi.fn()
    .mockImplementationOnce(() => firstDelay.promise)
    .mockResolvedValue(undefined)
  const subject = harness(
    active('comparison', { comparison: comparisonLines }),
    {},
    { ...effects(false), delay }
  )

  const first = subject.controller.previewComparison(0)
  await vi.waitFor(() => expect(delay).toHaveBeenCalled())
  await subject.controller.previewComparison(1)
  firstDelay.resolve()
  await first

  expect(subject.controller.view.comparisonPreviewIndex).toBe(1)
  expect(subject.controller.view.message).toBe('Quiet setup complete.')
  expect(subject.persisted).toEqual([])
})

test('restores a comparison before advance and pause backend actions', async () => {
  const order: string[] = []
  const current = active('comparison', { comparison: comparisonLines })
  const next = active('recap', { activityId: 'next' })
  const subject = harness(current, {
    advanceOpeningActivity: async () => {
      order.push('advance-api')
      return { session: next, activityCompleted: true }
    },
    pauseOpeningSession: async () => { order.push('pause-api') }
  }, effects(false), board((fen) => {
    if (fen === current.current.currentFen) order.push('reset-board')
  }))

  await subject.controller.previewComparison(0)
  order.length = 0
  await subject.controller.advance()
  expect(order.slice(0, 2)).toEqual(['reset-board', 'advance-api'])

  const paused = harness(current, {
    pauseOpeningSession: async () => { order.push('pause-api') }
  }, effects(false), board((fen) => {
    if (fen === current.current.currentFen) order.push('reset-board')
  }))
  await paused.controller.previewComparison(0)
  order.length = 0
  await paused.controller.pause()
  expect(order.slice(0, 2)).toEqual(['reset-board', 'pause-api'])
})

test('session adoption and destroy cancel stale comparison callbacks', async () => {
  const adoptionDelay = deferred<void>()
  const adopted = harness(
    active('comparison', { comparison: comparisonLines }),
    {},
    { ...effects(false), delay: () => adoptionDelay.promise }
  )
  const adoptionWork = adopted.controller.previewComparison(0)
  await Promise.resolve()
  const replacement = active('concept', { activityId: 'replacement' })
  adopted.controller.receiveSession(replacement)
  adoptionDelay.resolve()
  await adoptionWork
  expect(adopted.controller.view.state).toMatchObject({ session: replacement })
  expect(adopted.controller.view.comparisonPreviewIndex).toBeNull()

  const destroyDelay = deferred<void>()
  const destroyed = harness(
    active('comparison', { comparison: comparisonLines }),
    {},
    { ...effects(false), delay: () => destroyDelay.promise }
  )
  const destroyWork = destroyed.controller.previewComparison(0)
  await Promise.resolve()
  destroyed.controller.destroy()
  destroyDelay.resolve()
  await destroyWork
  expect(destroyed.controller.view.comparisonPreviewPlaying).toBe(false)
})

test('reduced motion previews only the final comparison position', async () => {
  const current = active('comparison', { comparison: comparisonLines })
  const subject = harness(current, {}, effects(true))

  await subject.controller.previewComparison(0)

  expect(subject.board.setPosition.mock.calls).toEqual([
    [current.current.currentFen, undefined, false],
    [afterD6, ['d7', 'd6'], false]
  ])
})

test('a comparison board exception restores the activity position without persisting', async () => {
  const current = active('comparison', { comparison: comparisonLines })
  const subject = harness(current, {}, effects(false), board(() => {
    throw new Error('preview adapter failed')
  }))

  await subject.controller.previewComparison(0)

  expect(subject.controller.view.state).toMatchObject({ fen: current.current.currentFen })
  expect(subject.controller.view.notice).toMatch(/preview adapter failed/i)
  expect(subject.controller.view.comparisonPreviewPlaying).toBe(false)
  expect(subject.persisted).toEqual([])
})

test('does not reanimate the optimistic learner move before the course reply', async () => {
  const current = active('decision')
  const next = active('decision', { activityNumber: 2, currentFen: afterD6 })
  const result = expected(next, [
    { uci: 'c2c3', resultingFen: afterC3 },
    { uci: 'd7d6', resultingFen: afterD6 }
  ], afterD6)
  const subject = harness(current, { playOpeningMove: async () => result })

  await subject.controller.play('c2c3')

  expect(subject.board.setPosition.mock.calls).toEqual([
    [afterC3, ['c2', 'c3'], false],
    [afterD6, ['d7', 'd6'], true]
  ])
})

test.each([
  ['alternative', 'Playable alternative'],
  ['off_course', 'Outside this course line']
] as const)('restores %s attempts with neutral, distinct feedback', async (feedback, copy) => {
  const current = active('decision')
  const result: OpeningActivityResult = {
    session: active('decision', { hintLevel: 1 }),
    activityCompleted: false,
    feedback
  }
  const subject = harness(current, { playOpeningMove: async () => result })

  await subject.controller.play('c2c3')

  expect(subject.board.setPosition).toHaveBeenCalledWith(startFen, undefined, true)
  expect(subject.controller.view.state.phase).toBe('ready')
  expect(subject.controller.view.feedback).toBe(feedback)
  expect(subject.controller.view.message).toContain(copy)
})

test('keeps progressive plan, source, and target hint data', async () => {
  const hints: OpeningHintResult[] = [
    { session: active('decision', { hintLevel: 1 }), level: 1, text: 'Prepare d4.', canReveal: false },
    { session: active('decision', { hintLevel: 2 }), level: 2, text: 'Use the c-pawn.', sourceSquare: 'c2', canReveal: false },
    { session: active('decision', { hintLevel: 3 }), level: 3, text: 'Move to c3.', sourceSquare: 'c2', targetSquare: 'c3', canReveal: true }
  ]
  const useOpeningHint = vi.fn()
    .mockResolvedValueOnce(hints[0])
    .mockResolvedValueOnce(hints[1])
    .mockResolvedValueOnce(hints[2])
  const subject = harness(active('decision'), { useOpeningHint })

  for (const hint of hints) {
    await subject.controller.useHint()
    expect(subject.controller.view.state).toMatchObject({ phase: 'ready', hint })
  }
})

test('reveals the course move and records the already-persisted completion', async () => {
  const current = active('decision', { canReveal: true, hintLevel: 3 })
  const done = completed()
  const subject = harness(current, { revealOpeningMove: async () => expected(done) })

  await subject.controller.reveal()

  expect(subject.board.setPosition).toHaveBeenCalledWith(afterC3, ['c2', 'c3'], true)
  expect(subject.controller.view.state).toMatchObject({
    phase: 'activity-complete', result: { session: done }
  })
  expect(subject.persisted).toEqual([done])
})

test('ignores stale move and hint responses after navigation', async () => {
  const moveGate = deferred<OpeningActivityResult>()
  const subject = harness(active('decision'), { playOpeningMove: () => moveGate.promise })
  const replacement = active('decision', { activityId: 'replacement', activityNumber: 4 })

  const work = subject.controller.play('c2c3')
  subject.controller.receiveSession(replacement)
  moveGate.resolve(expected(completed()))
  await work

  expect(subject.controller.view.state).toMatchObject({ session: replacement })
  expect(subject.persisted).toEqual([])

  const hintGate = deferred<OpeningHintResult>()
  const second = harness(active('decision'), { useOpeningHint: () => hintGate.promise })
  const hintWork = second.controller.useHint()
  second.controller.receiveSession(replacement)
  hintGate.resolve({ session: active('decision'), level: 1, text: 'stale', canReveal: false })
  await hintWork
  expect(second.controller.view.state).toMatchObject({ session: replacement })
  expect(second.controller.view.message).not.toContain('stale')
})

test('recovers the final FEN and announces an animation warning', async () => {
  const throwingBoard = board(() => { throw new Error('adapter failed') })
  const subject = harness(
    active('decision'),
    { playOpeningMove: async () => expected(active('decision', { currentFen: afterC3 })) },
    effects(false),
    throwingBoard
  )

  await subject.controller.play('c2c3')

  expect(subject.controller.view.state).toMatchObject({ phase: 'activity-complete', fen: afterC3 })
  expect(subject.controller.view.notice).toMatch(/animation failed.*final position was restored/i)
  expect(subject.controller.view.boardGeneration).toBeGreaterThan(0)
})

test('pauses without discarding persisted state and restarts from a checkpoint', async () => {
  const pauseOpeningSession = vi.fn(async () => {})
  const paused = harness(active('decision'), { pauseOpeningSession })
  await paused.controller.pause()
  expect(pauseOpeningSession).toHaveBeenCalledWith(fakeOpeningSession.sessionId)
  expect(paused.homes).toEqual([false])
  expect(paused.persisted).toEqual([])

  const restartRequired: OpeningSessionView = {
    sessionId: fakeOpeningSession.sessionId,
    mode: 'lesson',
    status: 'restart_required',
    courseId: fakeOpeningSession.courseId,
    generationId: fakeOpeningSession.generationId,
    lessonId: fakeOpeningSession.lessonId,
    courseTitle: fakeOpeningSession.courseTitle,
    path: fakeOpeningSession.path,
    depth: fakeOpeningSession.depth,
    notice: 'Course updated. Restart from a safe checkpoint.'
  }
  const checkpoint = active('concept', { activityId: 'checkpoint' })
  const restartOpeningSession = vi.fn(async () => checkpoint)
  const restarted = harness(restartRequired, { restartOpeningSession })
  await restarted.controller.restart()
  expect(restartOpeningSession).toHaveBeenCalledWith(fakeOpeningSession.sessionId)
  expect(restarted.controller.view.state).toMatchObject({ phase: 'passive', session: checkpoint })
  expect(restarted.changes).toEqual([checkpoint])
})

test('reduced motion jumps directly to the authoritative final FEN', async () => {
  const subject = harness(
    active('demonstration'),
    {
      advanceOpeningActivity: async () => ({
        session: active('decision', { currentFen: afterD6 }),
        activityCompleted: true,
        appliedMoves: [
          { uci: 'c2c3', resultingFen: afterC3 },
          { uci: 'd7d6', resultingFen: afterD6 }
        ],
        finalFen: afterD6
      })
    },
    effects(true)
  )

  await subject.controller.advance()

  expect(subject.board.setPosition).toHaveBeenCalledTimes(1)
  expect(subject.board.setPosition).toHaveBeenCalledWith(afterD6, ['d7', 'd6'], false)
})
