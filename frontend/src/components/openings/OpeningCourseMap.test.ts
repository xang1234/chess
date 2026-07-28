import { fireEvent, render, screen } from '@testing-library/svelte'
import type { OpeningCourseMap as CourseMap } from '../../lib/api'
import OpeningCourseMap from './OpeningCourseMap.svelte'

const map: CourseMap = {
  completedIdeas: 0,
  totalIdeas: 7,
  entries: [{
    lessonId: 'learned', moveLabel: '2.Nf3', title: 'Reach the Italian',
    state: 'learned', selectable: true
  }, {
    lessonId: 'current', parentLessonId: 'learned', moveLabel: '3...Bc5',
    title: 'Giuoco Piano', state: 'current', selectable: true
  }, {
    lessonId: 'next', parentLessonId: 'current', moveLabel: '4.c3',
    title: 'Prepare d4', state: 'next', selectable: true
  }, {
    lessonId: 'available', parentLessonId: 'current', moveLabel: '4.b4',
    title: 'Evans idea', state: 'available', selectable: true
  }]
}

test('renders a concise accessible move-labelled course map', async () => {
  const { component } = render(OpeningCourseMap, { map })
  const opens: unknown[] = []
  component.$on('open', (event) => opens.push(event.detail))

  const navigation = screen.getByRole('navigation', { name: 'Course map' })
  expect(navigation).toHaveTextContent('0 of 7 course ideas learned')
  expect(navigation).toHaveTextContent('3...Bc5')
  expect(navigation).toHaveTextContent('Giuoco Piano')
  for (const label of ['✓ Learned', '▶ Current', '→ Next', '○ Available']) {
    expect(navigation).toHaveTextContent(label)
  }
  expect(screen.getByLabelText('3...Bc5. Giuoco Piano. ▶ Current'))
    .toHaveAttribute('aria-current', 'step')

  await fireEvent.click(screen.getByRole('button', { name: 'View full map' }))
  expect(opens).toHaveLength(1)
})
