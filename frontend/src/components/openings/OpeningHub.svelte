<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import type { OpeningDepth, OpeningHomeView } from '../../lib/api'
  import OpeningTeachingTree from './OpeningTeachingTree.svelte'
  import { groupOpeningCourses, perspectiveLabel } from './opening-course-groups'

  export let home: OpeningHomeView
  export let selectedCourseId = ''

  const dispatch = createEventDispatcher<{
    back: void
    select: string
    depth: { courseId: string; depth: OpeningDepth }
    lesson: { courseId: string; lessonId: string }
    resume: void
    review: string
    explore: { courseId: string; positionId: string }
  }>()

  $: onlyCourse = singleCourse(home)
  $: course = home.courses.find((candidate) => candidate.courseId === selectedCourseId) ?? onlyCourse
  $: courseGroups = groupOpeningCourses(home.courses)
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

  function singleCourse(value: OpeningHomeView) {
    if (value.courses.length !== 1) return undefined
    const [course] = value.courses
    return course
  }

  function changeCourse(event: Event): void {
    dispatch('select', (event.currentTarget as HTMLSelectElement).value)
  }

  function reviewLabel(count: number, resumable: boolean): string {
    if (resumable && count === 0) return 'Continue review'
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
        <p class="muted">A repertoire for {perspectiveLabel(course.perspective)}</p>
      {/if}
    </div>
  </div>

  {#if home.notice}
    <p class="course-notice" role="status">{home.notice}</p>
  {/if}

  {#if home.courses.length > 1}
    <label class="course-control">
      Opening course
      <select value={course?.courseId ?? ''} on:change={changeCourse}>
        <option value="" disabled>Choose a course</option>
        {#each courseGroups as group (group.label)}
          <optgroup label={group.label}>
            {#each group.courses as candidate (candidate.courseId)}
              <option value={candidate.courseId}>{candidate.title}</option>
            {/each}
          </optgroup>
        {/each}
      </select>
    </label>
  {/if}

  {#if home.courses.length === 0}
    <div class="opening-empty panel">
      <h3>No private course imported</h3>
      <p>Import a private .ctcourse file from Parent settings.</p>
    </div>
  {:else if !course}
    <div class="opening-empty panel">
      <h3>Choose an opening course</h3>
      <p>Select the repertoire you want to study.</p>
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
      {#if course.hasResumableReview || course.dueReviews > 0}
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
