import { describe, expect, it } from 'vitest'
import type { OpeningMoveFeedbackDetail } from '../../lib/api'
import { openingFeedbackCopy } from './opening-feedback-copy'

const visible: OpeningMoveFeedbackDetail = {
  coverage: 'visible',
  playedSan: 'b4',
  variationName: 'Evans idea',
  evaluation: { code: 'unclear' },
  noteText: 'A sound alternative that keeps pressure on the centre.',
  expectedSan: 'c3',
  planText: 'Prepare d4 while keeping the bishop active.'
}

describe('openingFeedbackCopy', () => {
  it('explains a visible authored alternative and contrasts the lesson move', () => {
    const copy = openingFeedbackCopy(visible, 'white')

    expect(copy.heading).toBe('b4 — Evans idea')
    expect(copy.explanation).toContain('This is a course-known idea.')
    expect(copy.explanation).toContain('∞ Unclear position.')
    expect(copy.explanation).toContain('A sound alternative that keeps pressure on the centre.')
    expect(copy.contrast).toBe(
      'This lesson is practising c3: Prepare d4 while keeping the bishop active.'
    )
    expect(copy.announcement).toBe(
      `${copy.heading}. ${copy.explanation} ${copy.contrast}`
    )
  })

  it('names the depth for an authored move covered later', () => {
    const copy = openingFeedbackCopy({
      ...visible,
      coverage: 'deeper',
      variationName: 'Evans Gambit',
      requiredDepth: 'reference'
    }, 'white')

    expect(copy.explanation).toContain('Covered later at Reference depth.')
  })

  it('honestly labels an unknown move without invented metadata', () => {
    const copy = openingFeedbackCopy({
      coverage: 'unknown',
      playedSan: 'd3',
      evaluation: { code: 'none' },
      expectedSan: 'c3'
    }, 'white')

    expect(copy).toMatchObject({
      heading: 'd3',
      explanation: 'This move is not covered from this position.',
      contrast: 'This lesson is practising c3.'
    })
    expect(copy.announcement).not.toMatch(/variation|evaluation|bad|mistake/i)
  })
})
