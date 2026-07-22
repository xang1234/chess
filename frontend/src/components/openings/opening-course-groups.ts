import type { OpeningCourseSummary, OpeningPerspective } from '../../lib/api'

export type OpeningCourseGroup = {
  label: string
  courses: OpeningCourseSummary[]
}

export function perspectiveLabel(perspective: OpeningPerspective): string {
  return perspective === 'white' ? 'White' : 'Black'
}

export function groupOpeningCourses(courses: OpeningCourseSummary[]): OpeningCourseGroup[] {
  const white = courses.filter((course) => course.perspective === 'white')
  const black = courses.filter((course) => course.perspective === 'black')
  return [
    { label: 'White repertoires', courses: white },
    { label: 'Black repertoires', courses: black }
  ].filter((group) => group.courses.length > 0)
}
