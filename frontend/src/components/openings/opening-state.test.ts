import type {
  ActiveOpeningSessionView,
  CompletedOpeningSessionView,
  OpeningActivityResult,
  OpeningHintResult,
  OpeningRoadmapCheckpoint
} from '../../lib/api'
import { fakeOpeningSession } from '../../test-fakes'
import {
  acknowledgeOpeningActivity,
  beginOpeningAnimation,
  beginOpeningRequest,
  completeOpeningActivity,
  finishOpeningHint,
  initialiseOpening,
  markOpeningFeedback
} from './opening-state'

const nextFen = 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'
const checkpoint: OpeningRoadmapCheckpoint = {
  completedLessonId: 'giuoco-c3',
  path: [
    { lessonId: 'foundations-e4', title: 'Reach the Italian' },
    { lessonId: 'giuoco-c3', title: 'Prepare d4 with c3' }
  ],
  availableLessonIds: ['giuoco-d4'],
  recommendedLessonId: 'giuoco-d4',
  recommendedLessonTitle: 'Occupy the centre with d4',
  completedLessons: 2,
  totalLessons: 5
}

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
    courseTitle: fakeOpeningSession.courseTitle,
    path: fakeOpeningSession.path,
    depth: fakeOpeningSession.depth
  }
}

test('initialises passive teaching steps and ready move prompts', () => {
  expect(initialiseOpening(active()).phase).toBe('passive')
  expect(initialiseOpening(active({ current: { ...fakeOpeningSession.current, kind: 'decision' } })).phase)
    .toBe('ready')
})

test('ignores stale feedback and hint responses', () => {
  const requesting = beginOpeningRequest(
    initialiseOpening(active({ current: { ...fakeOpeningSession.current, kind: 'decision' } })),
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
      current: { ...fakeOpeningSession.current, kind: 'decision', currentFen: 'before-fen' }
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

test('successful animation holds the persisted next activity result until Continue', () => {
  const current = active({ current: { ...fakeOpeningSession.current, kind: 'demonstration' } })
  const pending = active({
    current: {
      ...fakeOpeningSession.current,
      kind: 'decision',
      activityId: 'try-c3',
      activityNumber: 2,
      currentFen: nextFen,
      legalMoves: ['c2c3']
    }
  })
  const requesting = beginOpeningRequest(initialiseOpening(current), 'advance', 1)
  const animating = beginOpeningAnimation(requesting, 1)
  const result: OpeningActivityResult = { session: pending, activityCompleted: true }
  const activityComplete = completeOpeningActivity(
    animating,
    1,
    nextFen,
    result,
    'Course move shown.'
  )

  expect(activityComplete).toMatchObject({
    phase: 'activity-complete',
    session: current,
    fen: nextFen,
    result,
    message: 'Course move shown.'
  })
  const acknowledged = acknowledgeOpeningActivity(activityComplete)
  expect(acknowledged).toMatchObject({ phase: 'ready', session: pending, fen: nextFen })
})

test('acknowledges lesson completion into a roadmap checkpoint', () => {
  const current = active({ current: { ...fakeOpeningSession.current, kind: 'decision' } })
  const requesting = beginOpeningRequest(initialiseOpening(current), 'move', 1)
  const animating = beginOpeningAnimation(requesting, 1)
  const result: OpeningActivityResult = {
    session: completed(),
    activityCompleted: true,
    checkpoint
  }
  const activityComplete = completeOpeningActivity(animating, 1, nextFen, result, 'Recalled.')

  expect(activityComplete.phase).toBe('activity-complete')
  expect(acknowledgeOpeningActivity(activityComplete)).toEqual({
    phase: 'checkpoint',
    session: completed(),
    checkpoint
  })
})
