import type {
  ActiveOpeningSessionView,
  CompletedOpeningSessionView,
  OpeningHintResult
} from '../../lib/api'
import { fakeOpeningSession } from '../../test-fakes'
import {
  acknowledgeOpeningStep,
  beginOpeningAnimation,
  beginOpeningRequest,
  completeOpeningStep,
  finishOpeningHint,
  initialiseOpening,
  markOpeningFeedback
} from './opening-state'

const nextFen = 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'

function active(overrides: Partial<ActiveOpeningSessionView> = {}): ActiveOpeningSessionView {
  return {
    ...fakeOpeningSession,
    ...overrides,
    current: { ...fakeOpeningSession.current, ...(overrides.current ?? {}) }
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
    depth: fakeOpeningSession.depth,
    summary: {
      totalPrompts: 3,
      positionsRecalled: 1,
      branchesRecognized: 1,
      retried: 1,
      usedHint: 1,
      revealed: 0
    }
  }
}

test('initialises passive teaching steps and ready move prompts', () => {
  expect(initialiseOpening(active()).phase).toBe('passive')
  expect(initialiseOpening(active({ current: { ...fakeOpeningSession.current, kind: 'try' } })).phase)
    .toBe('ready')
})

test('ignores stale feedback and hint responses', () => {
  const requesting = beginOpeningRequest(
    initialiseOpening(active({ current: { ...fakeOpeningSession.current, kind: 'try' } })),
    'move',
    2
  )
  const staleFeedback = markOpeningFeedback(requesting, 1, active(), 'alternative')
  const hint: OpeningHintResult = {
    session: active(), level: 1, text: 'Prepare d4.', canReveal: false
  }
  const staleHint = finishOpeningHint(requesting, 1, hint)

  expect(staleFeedback).toBe(requesting)
  expect(staleHint).toBe(requesting)
})

test.each(['alternative', 'off_course'] as const)(
  '%s feedback restores the returned authoritative position',
  (feedback) => {
    const original = active({
      current: { ...fakeOpeningSession.current, kind: 'branch', currentFen: 'before-fen' }
    })
    const returned = active({
      current: { ...original.current, currentFen: 'authoritative-fen', hintLevel: 1 }
    })
    const requesting = beginOpeningRequest(initialiseOpening(original), 'move', 1)
    const state = markOpeningFeedback(requesting, 1, returned, feedback)

    expect(state).toMatchObject({
      phase: 'ready',
      session: returned,
      fen: 'authoritative-fen',
      hint: null
    })
  }
)

test('successful animation holds the persisted next step until Continue', () => {
  const current = active({ current: { ...fakeOpeningSession.current, kind: 'watch' } })
  const pending = active({
    current: {
      ...fakeOpeningSession.current,
      kind: 'try',
      stepId: 'try-c3',
      stepNumber: 2,
      currentFen: nextFen,
      legalMoves: ['c2c3']
    }
  })
  const requesting = beginOpeningRequest(initialiseOpening(current), 'advance', 1)
  const animating = beginOpeningAnimation(requesting, 1)
  const stepComplete = completeOpeningStep(
    animating,
    1,
    nextFen,
    pending,
    'Course move shown.'
  )

  expect(stepComplete).toMatchObject({
    phase: 'step-complete',
    session: current,
    fen: nextFen,
    pending,
    message: 'Course move shown.'
  })
  const acknowledged = acknowledgeOpeningStep(stepComplete)
  expect(acknowledged).toMatchObject({ phase: 'ready', session: pending, fen: nextFen })
})

test('a completed pending session becomes summary only after Continue', () => {
  const current = active({ current: { ...fakeOpeningSession.current, kind: 'recall' } })
  const requesting = beginOpeningRequest(initialiseOpening(current), 'move', 1)
  const animating = beginOpeningAnimation(requesting, 1)
  const stepComplete = completeOpeningStep(animating, 1, nextFen, completed(), 'Recalled.')

  expect(stepComplete.phase).toBe('step-complete')
  expect(acknowledgeOpeningStep(stepComplete)).toMatchObject({
    phase: 'summary',
    session: { status: 'completed' }
  })
})
