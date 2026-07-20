<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import type { OpeningDepth, OpeningHomeView } from '../../lib/api'
  import OpeningTeachingTree from './OpeningTeachingTree.svelte'

  export let home: OpeningHomeView

  const dispatch = createEventDispatcher<{
    back: void
    depth: { courseId: string; depth: OpeningDepth }
    lesson: { courseId: string; lessonId: string }
    resume: void
    review: string
    explore: { courseId: string; positionId: string }
  }>()

  $: course = home.courses[0]
  $: continuationLessonId = course?.hasResumable
    ? course.currentLessonId
    : course?.recommendedLessonId || course?.nextLessonId
  $: continuationTitle = course?.hasResumable
    ? course.currentPath.at(-1)?.title || course.nextLessonTitle
    : course?.recommendedLessonTitle || course?.nextLessonTitle

  function changeDepth(event: Event): void {
    if (!course) return
    const depth = (event.currentTarget as HTMLSelectElement).value as OpeningDepth
    dispatch('depth', { courseId: course.courseId, depth })
  }

  function reviewLabel(count: number, resumable: boolean): string {
    const due = `${count} due ${count === 1 ? 'position' : 'positions'}`
    return resumable ? `Continue review — ${due}` : `Review ${due}`
  }

  function continueLearning(): void {
    if (!course) return
    if (course.hasResumable) {
      dispatch('resume')
    } else if (continuationLessonId) {
      dispatch('lesson', { courseId: course.courseId, lessonId: continuationLessonId })
    }
  }
</script>

<section class="opening-hub" aria-labelledby="opening-hub-title">
  <div class="opening-hub-header">
    <button class="secondary" type="button" on:click={() => dispatch('back')}>Back</button>
    <div>
      <p class="eyebrow">Learn openings</p>
      <h2 id="opening-hub-title">{course?.title ?? 'Opening courses'}</h2>
      {#if course}
        <p class="muted">A repertoire for {course.perspective === 'white' ? 'White' : 'Black'}</p>
      {/if}
    </div>
  </div>

  {#if home.notice}
    <p class="course-notice" role="status">{home.notice}</p>
  {/if}

  {#if !course}
    <div class="opening-empty panel">
      <h3>No private course imported</h3>
      <p>Import a private .ctcourse file from Parent settings.</p>
    </div>
  {:else}
    <div class="opening-overview">
      <div class="opening-progress" aria-label="Course progress">
        <strong>{course.completedLessons} of {course.totalLessons} lessons complete</strong>
        <span>{course.dueReviews} review {course.dueReviews === 1 ? 'position' : 'positions'} due</span>
      </div>
      <label class="depth-control">
        Course depth
        <select value={course.depth} on:change={changeDepth}>
          <option value="quick">Quick</option>
          <option value="standard">Standard</option>
          <option value="reference">Reference</option>
        </select>
      </label>
    </div>

    <div class="opening-actions">
      {#if course.hasResumable || continuationLessonId}
        <button class="primary" type="button" on:click={continueLearning}>
          Continue learning{continuationTitle ? ` — ${continuationTitle}` : ''}
        </button>
      {/if}
      {#if course.dueReviews > 0}
        <button class="secondary" type="button" on:click={() => dispatch('review', course.courseId)}>
          {reviewLabel(course.dueReviews, course.hasResumableReview)}
        </button>
      {/if}
      <button
        class="secondary"
        type="button"
        on:click={() => dispatch('explore', {
          courseId: course.courseId,
          positionId: course.rootPositionId
        })}
      >Explore variations</button>
    </div>

    <OpeningTeachingTree
      tree={course.tree}
      courseTitle={course.title}
      on:lesson={(event) => dispatch('lesson', {
        courseId: course.courseId,
        lessonId: event.detail
      })}
    />
  {/if}
</section>
