<script lang="ts">
  import { createEventDispatcher } from 'svelte'

  export let activeSession = false
  const dispatch = createEventDispatcher<{
    training: void
    practice: void
    games: void
    parent: void
  }>()
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

    <button class="hub-card" type="button" aria-label="Game Library" on:click={() => dispatch('games')}>
      <span class="card-icon" aria-hidden="true">♜</span>
      <strong>Game Library</strong>
      <span>Planned for a future milestone</span>
    </button>
  </div>

  <button class="settings-button" type="button" aria-label="Parent settings" on:click={() => dispatch('parent')}>⚙</button>
</section>
