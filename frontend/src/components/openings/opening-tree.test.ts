import type { OpeningTeachingTree } from '../../lib/api'
import { projectOpeningTree } from './opening-tree'

const tree: OpeningTeachingTree = {
  rootLessonId: 'root',
  nodes: [
    node('root'),
    node('later'),
    node('first'),
    node('hidden', false)
  ],
  edges: [
    edge('root-first', 'root', 'first', 1, 'continuation'),
    edge('root-hidden', 'root', 'hidden', 3, 'reference'),
    edge('root-later', 'root', 'later', 2, 'alternative')
  ]
}

function node(lessonId: string, visible = true): OpeningTeachingTree['nodes'][number] {
  return {
    lessonId,
    chapterId: 'chapter',
    title: lessonId,
    objective: `Learn ${lessonId}`,
    minimumDepth: visible ? 'quick' : 'reference',
    progress: 'available',
    completedActivities: 0,
    requiredActivities: 3,
    recommended: lessonId === 'first',
    reviewDue: false,
    visible
  }
}

function edge(
  edgeId: string,
  fromLessonId: string,
  toLessonId: string,
  ordinal: number,
  kind: OpeningTeachingTree['edges'][number]['kind']
): OpeningTeachingTree['edges'][number] {
  return {
    edgeId,
    fromLessonId,
    toLessonId,
    ordinal,
    kind,
    label: kind === 'alternative' ? 'Another plan' : undefined,
    minimumDepth: kind === 'reference' ? 'reference' : 'quick'
  }
}

test('projects every roadmap node in authored child order with incoming edge context', () => {
  const projected = projectOpeningTree(tree)

  expect(projected.node.lessonId).toBe('root')
  expect(projected.children.map((branch) => branch.node.lessonId))
    .toEqual(['first', 'later', 'hidden'])
  expect(projected.children[1].incoming).toMatchObject({
    kind: 'alternative',
    label: 'Another plan'
  })
  expect(projected.children[2].node.visible).toBe(false)
})

test('rejects missing nodes, duplicate parents, and disconnected roadmap nodes', () => {
  expect(() => projectOpeningTree({
    ...tree,
    edges: [...tree.edges, edge('missing', 'root', 'absent', 4, 'continuation')]
  })).toThrow('absent')

  expect(() => projectOpeningTree({
    ...tree,
    edges: [...tree.edges, edge('second-parent', 'first', 'later', 1, 'continuation')]
  })).toThrow('more than one parent')

  expect(() => projectOpeningTree({
    ...tree,
    nodes: [...tree.nodes, node('orphan')]
  })).toThrow('not connected')
})
