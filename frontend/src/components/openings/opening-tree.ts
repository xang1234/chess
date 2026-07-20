import type { OpeningTeachingEdge, OpeningTeachingNode, OpeningTeachingTree } from '../../lib/api'

export type OpeningTreeBranch = {
  node: OpeningTeachingNode
  incoming?: OpeningTeachingEdge
  children: OpeningTreeBranch[]
}

export function projectOpeningTree(tree: OpeningTeachingTree): OpeningTreeBranch {
  const nodes = new Map<string, OpeningTeachingNode>()
  for (const node of tree.nodes) {
    if (nodes.has(node.lessonId)) throw new Error(`duplicate opening lesson ${node.lessonId}`)
    nodes.set(node.lessonId, node)
  }
  const root = nodes.get(tree.rootLessonId)
  if (!root) throw new Error(`opening roadmap root ${tree.rootLessonId} does not exist`)

  const incoming = new Map<string, OpeningTeachingEdge>()
  const outgoing = new Map<string, OpeningTeachingEdge[]>()
  for (const edge of tree.edges) {
    if (!nodes.has(edge.fromLessonId)) {
      throw new Error(`opening roadmap edge ${edge.edgeId} references missing lesson ${edge.fromLessonId}`)
    }
    if (!nodes.has(edge.toLessonId)) {
      throw new Error(`opening roadmap edge ${edge.edgeId} references missing lesson ${edge.toLessonId}`)
    }
    if (edge.toLessonId === tree.rootLessonId) {
      throw new Error(`opening roadmap root ${tree.rootLessonId} must not have a parent`)
    }
    const previous = incoming.get(edge.toLessonId)
    if (previous) {
      throw new Error(
        `opening lesson ${edge.toLessonId} has more than one parent (${previous.edgeId}, ${edge.edgeId})`
      )
    }
    incoming.set(edge.toLessonId, edge)
    outgoing.set(edge.fromLessonId, [...(outgoing.get(edge.fromLessonId) ?? []), edge])
  }

  const visited = new Set<string>()
  const visiting = new Set<string>()
  const build = (node: OpeningTeachingNode, via?: OpeningTeachingEdge): OpeningTreeBranch => {
    if (visiting.has(node.lessonId)) throw new Error(`opening roadmap contains a cycle at ${node.lessonId}`)
    if (visited.has(node.lessonId)) throw new Error(`opening lesson ${node.lessonId} is repeated in the roadmap`)
    visiting.add(node.lessonId)
    const edges = [...(outgoing.get(node.lessonId) ?? [])]
      .sort((left, right) => left.ordinal - right.ordinal || left.edgeId.localeCompare(right.edgeId))
    const children = edges.map((edge) => build(nodes.get(edge.toLessonId)!, edge))
    visiting.delete(node.lessonId)
    visited.add(node.lessonId)
    return { node, incoming: via, children }
  }

  const projection = build(root)
  if (visited.size !== nodes.size) {
    const disconnected = [...nodes.keys()].filter((lessonId) => !visited.has(lessonId)).sort()
    throw new Error(`opening lessons are not connected to ${tree.rootLessonId}: ${disconnected.join(', ')}`)
  }
  return projection
}
