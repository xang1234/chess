<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte'
  import { getAPI, type Profile, type RatingBounds } from '../../lib/api'

  const dispatch = createEventDispatcher<{ complete: Profile }>()
  let learnerRating = 1200
  let sessionSize: 5 | 10 | 15 = 10
  let saving = false
  let loading = true
  let error = ''
  let learnerRatingBounds: RatingBounds | null = null

  onMount(async () => {
    try {
      const filters = await getAPI().getPracticeFilters()
      learnerRatingBounds = filters.learnerRatingBounds
      learnerRating = Math.min(
        Math.max(learnerRating, learnerRatingBounds.minimum),
        learnerRatingBounds.maximum
      )
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      loading = false
    }
  })

  async function save(): Promise<void> {
    if (!learnerRatingBounds) return
    if (learnerRating < learnerRatingBounds.minimum || learnerRating > learnerRatingBounds.maximum) {
      error = `Learner rating must be between ${learnerRatingBounds.minimum} and ${learnerRatingBounds.maximum}`
      return
    }
    saving = true
    error = ''
    const profile: Profile = { learnerRating, sessionSize }
    try {
      await getAPI().updateProfile(profile)
      dispatch('complete', profile)
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      saving = false
    }
  }
</script>

<section class="panel setup-panel" aria-labelledby="setup-title">
  <p class="eyebrow">Welcome</p>
  <h2 id="setup-title">Set up today’s training</h2>
  <p class="muted">A parent can change these choices later.</p>

  {#if learnerRatingBounds}
    <label>
      <span>Starting rating</span>
      <input
        type="number"
        min={learnerRatingBounds.minimum}
        max={learnerRatingBounds.maximum}
        step="50"
        aria-describedby="starting-rating-help"
        bind:value={learnerRating}
      />
    </label>
    <p id="starting-rating-help" class="muted">
      Available rating range: {learnerRatingBounds.minimum}–{learnerRatingBounds.maximum}
    </p>
  {/if}

  <label>
    <span>Puzzles per session</span>
    <select bind:value={sessionSize}>
      <option value={5}>5 puzzles</option>
      <option value={10}>10 puzzles</option>
      <option value={15}>15 puzzles</option>
    </select>
  </label>

  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <button class="primary" type="button" on:click={save} disabled={saving || loading || !learnerRatingBounds}>
    {saving ? 'Saving…' : loading ? 'Loading…' : 'Save and continue'}
  </button>
</section>
