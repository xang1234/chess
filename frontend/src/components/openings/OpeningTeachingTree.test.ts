import { fireEvent, render, screen } from '@testing-library/svelte'
import type { OpeningTeachingTree } from '../../lib/api'
import OpeningTeachingTreeComponent from './OpeningTeachingTree.svelte'

const tree: OpeningTeachingTree = {
  rootLessonId: 'foundation',
  nodes: [
    {
      lessonId: 'foundation', chapterId: 'foundations', title: 'Reach the Italian',
      objective: 'Reach the shared starting position.', minimumDepth: 'quick',
      progress: 'completed', completedActivities: 3, requiredActivities: 3,
      recommended: false, reviewDue: true, visible: true
    },
    {
      lessonId: 'quiet', chapterId: 'giuoco', title: 'Choose the quiet setup',
      objective: 'Prepare the centre.', minimumDepth: 'quick', progress: 'in_progress',
      completedActivities: 1, requiredActivities: 3, recommended: true,
      reviewDue: false, visible: true
    },
    {
      lessonId: 'deep', chapterId: 'giuoco', title: 'Study a deep sideline',
      objective: 'Compare rare systems.', minimumDepth: 'reference', progress: 'available',
      completedActivities: 0, requiredActivities: 2, recommended: false,
      reviewDue: false, visible: false
    },
    {
      lessonId: 'next', chapterId: 'giuoco', title: 'Add the central break',
      objective: 'Play d4 at the right moment.', minimumDepth: 'quick', progress: 'available',
      completedActivities: 0, requiredActivities: 2, recommended: true,
      reviewDue: false, visible: true
    },
    {
      lessonId: 'other', chapterId: 'giuoco', title: 'Try another setup',
      objective: 'Compare another plan.', minimumDepth: 'quick', progress: 'available',
      completedActivities: 0, requiredActivities: 2, recommended: false,
      reviewDue: false, visible: true
    }
  ],
  edges: [
    {
      edgeId: 'foundation-quiet', fromLessonId: 'foundation', toLessonId: 'quiet',
      ordinal: 1, kind: 'continuation', label: '…Bc5', minimumDepth: 'quick'
    },
    {
      edgeId: 'quiet-deep', fromLessonId: 'quiet', toLessonId: 'deep',
      ordinal: 1, kind: 'reference', label: 'Deeper line', minimumDepth: 'reference'
    },
    {
      edgeId: 'quiet-next', fromLessonId: 'quiet', toLessonId: 'next',
      ordinal: 2, kind: 'continuation', label: 'Play c3', minimumDepth: 'quick'
    },
    {
      edgeId: 'quiet-other', fromLessonId: 'quiet', toLessonId: 'other',
      ordinal: 3, kind: 'alternative', label: 'Try d3', minimumDepth: 'quick'
    }
  ]
}

test('renders semantic roadmap progress and starts any visible node', async () => {
  const { component } = render(OpeningTeachingTreeComponent, {
    tree,
    courseTitle: 'Italian Game',
    currentLessonId: 'quiet'
  })
  const starts: string[] = []
  const resumes: unknown[] = []
  component.$on('lesson', (event) => starts.push(event.detail))
  component.$on('resume', (event) => resumes.push(event.detail))

  expect(screen.getByRole('tree', { name: 'Italian Game course roadmap' })).toBeInTheDocument()
  expect(screen.getByRole('treeitem', {
    name: /Choose the quiet setup.*In progress.*Recommended.*1 of 3 ideas/i
  })).toBeInTheDocument()
  expect(screen.getByRole('treeitem', {
    name: /Reach the Italian.*Complete.*Review due/i
  })).toBeInTheDocument()
  expect(screen.getByRole('treeitem', {
    name: /Study a deep sideline.*Hidden at this depth/i
  })).toBeInTheDocument()
  expect(screen.getByText('…Bc5')).toBeInTheDocument()

  await fireEvent.click(screen.getByRole('button', { name: 'Return to your lesson' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Start next idea here' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Study again' }))
	const startHere = screen.getAllByRole('button', { name: 'Start here' })
	const available = startHere.find((button) => !button.hasAttribute('disabled'))
	if (!available) throw new Error('available Start here action is missing')
  await fireEvent.click(available)

  expect(resumes).toHaveLength(1)
  expect(starts).toEqual(['next', 'foundation', 'other'])
  expect(startHere.some((button) => button.hasAttribute('disabled'))).toBe(true)
})
