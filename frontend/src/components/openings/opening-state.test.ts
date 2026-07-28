import type {
  ActiveOpeningLessonView,
  CompletedOpeningSessionView,
  OpeningActivityResult,
  OpeningHintResult
} from '../../lib/api'
import { fakeOpeningSession } from '../../test-fakes'
import {
  acknowledgeOpeningActivity,
  beginOpeningAnimation,
  beginOpeningRequest,
  completeOpeningActivity,
  completeOpeningActivityWithoutPause,
  finishOpeningHint,
  initialiseOpening,
  markOpeningFeedback
} from './opening-state'

const nextFen = 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4'
function active(overrides: Partial<ActiveOpeningLessonView> = {}): ActiveOpeningLessonView {
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

test('completes an active zero-frame result without an acknowledgement phase', () => {
  const concept = active()
  const requesting = beginOpeningRequest(initialiseOpening(concept), 'advance', 7)
  const next = active({
    current: {
      ...fakeOpeningSession.current,
      kind: 'recap',
      activityId: 'recap',
      title: 'Keep the plan'
    }
  })

  expect(completeOpeningActivityWithoutPause(requesting, 7, {
    session: next,
    activityCompleted: true
  })).toMatchObject({ phase: 'passive', session: next, fen: next.current.currentFen })
})

test('refuses to fast-forward final course completion', () => {
  const requesting = beginOpeningRequest(initialiseOpening(active()), 'advance', 8)
  expect(() => completeOpeningActivityWithoutPause(requesting, 8, {
    session: completed(),
    activityCompleted: true
  })).toThrow('active next activity')
})

test('acknowledges an animated cross-node result as the next idea', () => {
  const current = active({ current: { ...fakeOpeningSession.current, kind: 'decision' } })
	const pending = active({
		sessionId: 'opening-session-2',
		lessonId: 'giuoco-d4',
		path: [
			...fakeOpeningSession.path,
			{ lessonId: 'giuoco-d4', title: 'Occupy the centre with d4' }
		],
		courseMap: {
			completedIdeas: 3,
			totalIdeas: 4,
			entries: [{
				lessonId: 'giuoco-c3', moveLabel: '3...Bc5', title: 'Giuoco Piano',
				state: 'learned', selectable: true
			}, {
				lessonId: 'giuoco-d4', parentLessonId: 'giuoco-c3', moveLabel: '4.c3',
				title: 'Play d4', state: 'current', selectable: true
			}]
		},
		current: {
			...fakeOpeningSession.current,
			activityId: 'd4-concept', kind: 'concept', title: 'Occupy the centre'
		}
	})
  const requesting = beginOpeningRequest(initialiseOpening(current), 'move', 1)
  const animating = beginOpeningAnimation(requesting, 1)
  const result: Extract<OpeningActivityResult, { activityCompleted: true }> = {
		session: pending, activityCompleted: true
	}
  const activityComplete = completeOpeningActivity(animating, 1, nextFen, result, 'Recalled.')

  expect(activityComplete.phase).toBe('activity-complete')
	expect(acknowledgeOpeningActivity(activityComplete)).toMatchObject({
		phase: 'passive',
		session: {
			sessionId: 'opening-session-2',
			lessonId: 'giuoco-d4',
			current: { activityId: 'd4-concept' }
		}
	})
})
