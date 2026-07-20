import { fireEvent, render, screen } from '@testing-library/svelte'
import type { OpeningRoadmapCheckpoint as Checkpoint } from '../../lib/api'
import OpeningRoadmapCheckpoint from './OpeningRoadmapCheckpoint.svelte'

const checkpoint: Checkpoint = {
  completedLessonId: 'giuoco-c3',
  path: [
    { lessonId: 'foundation', title: 'Reach the Italian' },
    { lessonId: 'giuoco-c3', title: 'Prepare d4 with c3' }
  ],
  availableLessonIds: ['giuoco-d4', 'giuoco-quiet-d3'],
  recommendedLessonId: 'giuoco-d4',
  recommendedLessonTitle: 'Occupy the centre with d4',
  completedLessons: 2,
  totalLessons: 5
}

test('continues directly to the recommendation or lets the learner inspect the tree or stop', async () => {
  const { component } = render(OpeningRoadmapCheckpoint, {
    checkpoint,
    courseId: 'italian-white'
  })
  const continuations: unknown[] = []
  const tree = vi.fn()
  const home = vi.fn()
  component.$on('continue', (event) => continuations.push(event.detail))
  component.$on('tree', tree)
  component.$on('home', home)

  expect(screen.getByRole('heading', { name: 'Prepare d4 with c3 complete' })).toBeInTheDocument()
  expect(screen.getByText('2 of 5 lessons complete')).toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button', { name: 'Continue to Occupy the centre with d4' }))
  expect(continuations).toEqual([{
    courseId: 'italian-white',
    lessonId: 'giuoco-d4'
  }])
  await fireEvent.click(screen.getByRole('button', { name: 'View course tree' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Stop for now' }))
  expect(tree).toHaveBeenCalledOnce()
  expect(home).toHaveBeenCalledOnce()
})
