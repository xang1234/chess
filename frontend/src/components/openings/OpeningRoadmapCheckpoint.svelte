<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import type { OpeningRoadmapCheckpoint } from '../../lib/api'
  import OpeningPathContext from './OpeningPathContext.svelte'

  export let checkpoint: OpeningRoadmapCheckpoint
  export let courseId: string

  const dispatch = createEventDispatcher<{
    continue: { courseId: string; lessonId: string }
    tree: void
    home: { completed: boolean }
  }>()

  $: completedTitle = checkpoint.path.at(-1)?.title || 'Lesson'

  function continueNext(): void {
    if (!checkpoint.recommendedLessonId) return
    dispatch('continue', { courseId, lessonId: checkpoint.recommendedLessonId })
  }
</script>

<section class="opening-checkpoint panel" aria-labelledby="opening-checkpoint-title">
  <p class="eyebrow">Course checkpoint</p>
  <h2 id="opening-checkpoint-title">{completedTitle} complete</h2>
  <OpeningPathContext path={checkpoint.path} />
  <p class="checkpoint-progress">
    {checkpoint.completedLessons} of {checkpoint.totalLessons} lessons complete
  </p>
  <p>
    {checkpoint.availableLessonIds.length} {checkpoint.availableLessonIds.length === 1
      ? 'lesson is'
      : 'lessons are'} now available.
  </p>

  <div class="opening-checkpoint-actions">
    {#if checkpoint.recommendedLessonId}
      <button
        class="primary"
        type="button"
        on:click={continueNext}
      >Continue to {checkpoint.recommendedLessonTitle || 'next lesson'}</button>
    {/if}
    <button class="secondary" type="button" on:click={() => dispatch('tree')}>
      View course tree
    </button>
    <button class="quiet-action" type="button" on:click={() => dispatch('home', { completed: true })}>
      Stop for now
    </button>
  </div>
</section>
