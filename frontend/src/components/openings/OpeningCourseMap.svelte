<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import type { OpeningCourseMap, OpeningCourseMapState } from '../../lib/api'

  export let map: OpeningCourseMap

  const dispatch = createEventDispatcher<{ open: void }>()

  function stateLabel(state: OpeningCourseMapState): string {
    switch (state) {
      case 'learned': return '✓ Learned'
      case 'current': return '▶ Current'
      case 'next': return '→ Next'
      case 'available': return '○ Available'
    }
  }

  function entryDepth(lessonId: string): number {
    const entries = new Map(map.entries.map((entry) => [entry.lessonId, entry]))
    const visited = new Set<string>([lessonId])
    let current = entries.get(lessonId)
    let depth = 0
    while (current?.parentLessonId && !visited.has(current.parentLessonId)) {
      const parent = entries.get(current.parentLessonId)
      if (!parent) break
      visited.add(parent.lessonId)
      depth++
      current = parent
    }
    return depth
  }

  function accessibleName(
    moveLabel: string | undefined,
    title: string,
    state: OpeningCourseMapState
  ): string {
    return [moveLabel, title, stateLabel(state)].filter(Boolean).join('. ')
  }
</script>

<nav class="opening-course-map" aria-label="Course map">
  <div class="opening-course-map-heading">
    <div>
      <h3>Course map</h3>
      <p>{map.completedIdeas} of {map.totalIdeas} course ideas learned</p>
    </div>
    <button class="quiet-action" type="button" on:click={() => dispatch('open')}>
      View full map
    </button>
  </div>
  <ol>
    {#each map.entries as entry (entry.lessonId)}
      <li
        class={`course-map-${entry.state}`}
        aria-current={entry.state === 'current' ? 'step' : undefined}
        aria-label={accessibleName(entry.moveLabel, entry.title, entry.state)}
        style={`--course-map-depth: ${entryDepth(entry.lessonId)}; --course-map-indent: ${Math.min(entryDepth(entry.lessonId), 3) * 12}px`}
      >
        {#if entry.moveLabel}<span class="course-map-move">{entry.moveLabel}</span>{/if}
        <strong>{entry.title}</strong>
        <span class="course-map-state">{stateLabel(entry.state)}</span>
      </li>
    {/each}
  </ol>
</nav>
