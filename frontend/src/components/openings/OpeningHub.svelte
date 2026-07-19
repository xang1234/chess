<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import type { OpeningDepth, OpeningHomeView } from '../../lib/api'

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

  function changeDepth(event: Event): void {
    if (!course) return
    const depth = (event.currentTarget as HTMLSelectElement).value as OpeningDepth
    dispatch('depth', { courseId: course.courseId, depth })
  }

  function reviewLabel(count: number): string {
    return `Review ${count} due ${count === 1 ? 'position' : 'positions'}`
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
      {#if course.hasResumable}
        <button class="primary" type="button" on:click={() => dispatch('resume')}>
          Continue {course.nextLessonTitle || 'lesson'}
        </button>
      {/if}
      {#if course.dueReviews > 0}
        <button class="secondary" type="button" on:click={() => dispatch('review', course.courseId)}>
          {reviewLabel(course.dueReviews)}
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

    <div class="opening-chapters">
      {#each course.chapters as chapter (chapter.chapterId)}
        <section class="opening-chapter" aria-labelledby={`chapter-${chapter.chapterId}`}>
          <h3 id={`chapter-${chapter.chapterId}`}>{chapter.title}</h3>
          {#if chapter.lessons.length === 0}
            <p class="muted">No lessons at this depth.</p>
          {:else}
            <div class="lesson-list">
              {#each chapter.lessons as lesson (lesson.lessonId)}
                <article class="lesson-row">
                  <div>
                    <strong>{lesson.title}</strong>
                    <span class="muted">
                      {lesson.completed
                        ? 'Complete'
                        : `${lesson.completedSteps} of ${lesson.totalSteps} steps`}
                    </span>
                  </div>
                  <button
                    class="secondary"
                    type="button"
                    on:click={() => dispatch('lesson', {
                      courseId: course.courseId,
                      lessonId: lesson.lessonId
                    })}
                  >{lesson.completed ? 'Study' : 'Start'} {lesson.title}</button>
                </article>
              {/each}
            </div>
          {/if}
        </section>
      {/each}
    </div>
  {/if}
</section>
