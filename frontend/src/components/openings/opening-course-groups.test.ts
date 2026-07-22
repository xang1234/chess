import { describe, expect, test } from 'vitest'
import type { OpeningCourseSummary } from '../../lib/api'
import { groupOpeningCourses, perspectiveLabel } from './opening-course-groups'
import { fakeOpeningHome } from '../../test-fakes'

function course(overrides: Partial<OpeningCourseSummary>): OpeningCourseSummary {
  return { ...fakeOpeningHome.courses[0], ...overrides }
}

describe('opening course grouping', () => {
  test('labels course perspectives for reader-facing copy', () => {
    expect(perspectiveLabel('white')).toBe('White')
    expect(perspectiveLabel('black')).toBe('Black')
  })

  test('groups courses by learner perspective without reordering inside a group', () => {
    const groups = groupOpeningCourses([
      course({ courseId: 'italian-white', title: 'Italian Game for White', perspective: 'white' }),
      course({ courseId: 'caro-black', title: 'Caro-Kann for Black', perspective: 'black' }),
      course({ courseId: 'ruy-white', title: 'Ruy Lopez for White', perspective: 'white' })
    ])

    expect(groups).toEqual([
      {
        label: 'White repertoires',
        courses: [
          expect.objectContaining({ courseId: 'italian-white' }),
          expect.objectContaining({ courseId: 'ruy-white' })
        ]
      },
      {
        label: 'Black repertoires',
        courses: [
          expect.objectContaining({ courseId: 'caro-black' })
        ]
      }
    ])
  })
})
