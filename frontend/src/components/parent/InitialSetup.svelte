<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import { getAPI, type Profile } from '../../lib/api'

  const dispatch = createEventDispatcher<{ complete: Profile }>()
  let learnerRating = 1200
  let sessionSize: 5 | 10 | 15 = 10
  let saving = false
  let error = ''

  async function save(): Promise<void> {
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

  <label>
    <span>Starting rating</span>
    <input type="number" min="400" max="3000" step="50" bind:value={learnerRating} />
  </label>

  <label>
    <span>Puzzles per session</span>
    <select bind:value={sessionSize}>
      <option value={5}>5 puzzles</option>
      <option value={10}>10 puzzles</option>
      <option value={15}>15 puzzles</option>
    </select>
  </label>

  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <button class="primary" type="button" on:click={save} disabled={saving}>
    {saving ? 'Saving…' : 'Save and continue'}
  </button>
</section>
