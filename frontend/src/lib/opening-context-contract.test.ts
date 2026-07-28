import { decodeOpeningActivityResult, decodeOpeningSession } from './api-contract'

const openingStepPayload = {
  activityId: 'try-c3',
  kind: 'decision',
  title: 'Prepare the centre',
  instruction: 'Choose the course move.',
  required: true,
  variationName: 'Giuoco Piano',
  positionId: 'after-bc5',
  currentFen: 'opening-fen',
  orientation: 'white',
  legalMoves: ['c2c3', 'd2d3'],
  teachingNoteTexts: ['Prepare d4.'],
  referenceNoteTexts: ['A detailed source line.'],
  comparison: [],
  annotations: [{ kind: 'square', from: 'd4', label: 'central break' }],
  movesToHere: [],
  movesToHereSan: [],
  positionContext: {
    positionId: 'after-bc5',
    label: 'Giuoco Piano',
    evaluation: { code: 'equal', sourceSymbol: '=' },
    planText: 'Prepare d4 while keeping the bishop active.',
    arrivedVia: [{
      moveId: 'black-bc5', san: 'Bc5', role: 'opponent',
      variationName: 'Giuoco Piano', evaluation: { code: 'equal' }, selected: true
    }],
    nextMoves: [{
      moveId: 'white-c3', san: 'c3', role: 'repertoire',
      variationName: 'Giuoco Piano', evaluation: { code: 'white_slight' }, selected: false
    }]
  },
  activityNumber: 3,
  activityTotal: 5,
  completedIdeas: 2,
  requiredIdeas: 5,
  hintLevel: 0,
  canReveal: false,
  referenceSections: []
}

const activeOpeningPayload = {
  sessionId: 'opening-session',
  mode: 'lesson',
  status: 'active',
  courseId: 'italian-white',
  generationId: 'generation-1',
  lessonId: 'giuoco-c3',
  courseTitle: 'Italian Game for White',
  path: [{ lessonId: 'giuoco-c3', title: 'Prepare d4 with c3' }],
  depth: 'reference',
  courseMap: {
    completedIdeas: 0,
    totalIdeas: 4,
    entries: [{
      lessonId: 'giuoco-c3', moveLabel: '3...Bc5', title: 'Giuoco Piano',
      state: 'current', selectable: true
    }]
  },
  current: openingStepPayload
}

test('opening activity decoder retains structured position context', () => {
  expect(decodeOpeningSession(activeOpeningPayload)).toMatchObject({
    current: {
      positionContext: {
        positionId: 'after-bc5',
        label: 'Giuoco Piano',
        arrivedVia: [{ moveId: 'black-bc5', selected: true }],
        nextMoves: [{ moveId: 'white-c3', selected: false }]
      }
    }
  })
})

test.each([
  ['positionContext', { ...openingStepPayload, positionContext: undefined }],
  ['role', {
    ...openingStepPayload,
    positionContext: {
      ...openingStepPayload.positionContext,
      nextMoves: [{ ...openingStepPayload.positionContext.nextMoves[0], role: 'idea' }]
    }
  }],
  ['selected', {
    ...openingStepPayload,
    positionContext: {
      ...openingStepPayload.positionContext,
      arrivedVia: [{ ...openingStepPayload.positionContext.arrivedVia[0], selected: 'yes' }]
    }
  }]
])('opening activity decoder rejects malformed %s context', (field, current) => {
  expect(() => decodeOpeningSession({ ...activeOpeningPayload, current })).toThrow(field)
})

test('opening result decoder requires authoritative frames for an expected prompt completion', () => {
  expect(() => decodeOpeningActivityResult({
    session: activeOpeningPayload,
    activityCompleted: true,
    feedback: 'expected'
  })).toThrow('appliedMoves')
  expect(() => decodeOpeningActivityResult({
    session: activeOpeningPayload,
    activityCompleted: true,
    feedback: 'expected',
    appliedMoves: [{ uci: 'c2c3', resultingFen: 'after-c3' }]
  })).toThrow('finalFen')
  expect(decodeOpeningActivityResult({
    session: activeOpeningPayload,
    activityCompleted: true,
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
      activityCompleted: false,
      feedback,
      message: 'Try the course move.'
    }
    expect(decodeOpeningActivityResult(base)).toMatchObject({ feedback, activityCompleted: false })
    expect(() => decodeOpeningActivityResult({
      ...base,
      appliedMoves: [{ uci: 'd2d3', resultingFen: 'wrong-fen' }],
      finalFen: 'wrong-fen'
    })).toThrow('must not mutate')
  }
)

test('opening result decoder retains structured graph feedback', () => {
  const feedbackDetail = {
    coverage: 'deeper', playedSan: 'b4', variationName: 'Evans idea',
    evaluation: { code: 'unclear' }, expectedSan: 'c3', planText: 'Prepare d4.',
    requiredDepth: 'reference'
  }
  expect(decodeOpeningActivityResult({
    session: activeOpeningPayload,
    activityCompleted: false,
    feedback: 'off_course',
    feedbackDetail
  })).toMatchObject({ feedbackDetail })
})

test.each([
  ['coverage', { coverage: 'later', playedSan: 'b4', evaluation: { code: 'unclear' }, expectedSan: 'c3' }],
  ['evaluation', { coverage: 'unknown', playedSan: 'd3', expectedSan: 'c3' }],
  ['expectedSan', { coverage: 'unknown', playedSan: 'd3', evaluation: { code: 'none' } }],
  ['requiredDepth', { coverage: 'deeper', playedSan: 'b4', evaluation: { code: 'unclear' }, expectedSan: 'c3' }]
])('opening result decoder rejects malformed feedback detail %s', (field, feedbackDetail) => {
  expect(() => decodeOpeningActivityResult({
    session: activeOpeningPayload,
    activityCompleted: false,
    feedback: 'off_course',
    feedbackDetail
  })).toThrow(field)
})

test('opening result decoder rejects feedback detail on completed results', () => {
  expect(() => decodeOpeningActivityResult({
    session: activeOpeningPayload,
    activityCompleted: true,
    appliedMoves: [{ uci: 'c2c3', resultingFen: 'after-c3' }],
    finalFen: 'after-c3',
    feedback: 'expected',
    feedbackDetail: {
      coverage: 'visible', playedSan: 'c3',
      evaluation: { code: 'equal' }, expectedSan: 'c3'
    }
  })).toThrow('feedbackDetail')
})
