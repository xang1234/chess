<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import type { OpeningHomeView } from '../../lib/api'

  export let activeSession = false
  export let openingHome: OpeningHomeView = { courses: [] }
  const dispatch = createEventDispatcher<{
    training: void
    practice: void
    openings: void
    games: void
    parent: void
  }>()

  $: openingCopy = openingSummary(openingHome)

  function openingSummary(home: OpeningHomeView): string {
    if (home.courses.length === 0) return 'Import a private course'
    if (home.courses.length > 1) return `Choose from ${home.courses.length} opening courses`
    const [course] = home.courses
    if (course.hasResumable) return `Continue ${course.title}`
    const next = course.recommendedLessonTitle || course.nextLessonTitle
    return next ? `Next: ${next}` : `Explore ${course.title}`
  }
</script>

<section class="home-hub" aria-labelledby="home-title">
  <div class="welcome-copy">
    <p class="eyebrow">Ready when you are</p>
    <h2 id="home-title">What would you like to play?</h2>
  </div>

  <div class="hub-grid">
    <button
      class="hub-card primary-card"
      type="button"
      aria-label={activeSession ? "Continue today's training" : "Start today's training"}
      on:click={() => dispatch('training')}
    >
      <span class="card-icon" aria-hidden="true">♟</span>
      <strong>{activeSession ? "Continue today's training" : "Start today's training"}</strong>
      <span>Ten focused puzzles, including reviews</span>
    </button>

    <button class="hub-card" type="button" aria-label="Free Practice" on:click={() => dispatch('practice')}>
      <span class="card-icon" aria-hidden="true">◎</span>
      <strong>Free Practice</strong>
      <span>Choose themes and difficulty</span>
    </button>

    <button class="hub-card opening-card" type="button" aria-label="Learn Openings" on:click={() => dispatch('openings')}>
      <span class="card-icon" aria-hidden="true">♘</span>
      <strong>Learn Openings</strong>
      <span>{openingCopy}</span>
      {#if openingHome.notice}<span class="card-notice">{openingHome.notice}</span>{/if}
    </button>

    <button class="hub-card" type="button" aria-label="Game Library" on:click={() => dispatch('games')}>
      <span class="card-icon" aria-hidden="true">♜</span>
      <strong>Game Library</strong>
      <span>Planned for a future milestone</span>
    </button>
  </div>

  <button class="settings-button" type="button" aria-label="Parent settings" on:click={() => dispatch('parent')}>⚙</button>
</section>
