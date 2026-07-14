<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte'
  import { getAPI, type ParentSummary, type Profile } from '../../lib/api'

  const dispatch = createEventDispatcher<{ import: void }>()
  let profile: Profile | null = null
  let summary: ParentSummary | null = null
  let learnerRating = 1200
  let sessionSize: 5 | 10 | 15 = 10
  let loading = true
  let saving = false
  let saved = false
  let error = ''

  $: ratings = summary?.ratingTrend.map((point) => point.rating) ?? []
  $: minimumRating = ratings.length > 0 ? Math.min(...ratings) : summary?.learnerRating ?? 0
  $: maximumRating = ratings.length > 0 ? Math.max(...ratings) : summary?.learnerRating ?? 0
  $: trendPoints = chartPoints(ratings, minimumRating, maximumRating)

  onMount(async () => {
    try {
      const [loadedProfile, loadedSummary] = await Promise.all([
        getAPI().getProfile(),
        getAPI().getParentSummary()
      ])
      profile = loadedProfile
      summary = loadedSummary
      if (profile) {
        learnerRating = Math.round(profile.learnerRating)
        sessionSize = profile.sessionSize
      }
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      loading = false
    }
  })

  function chartPoints(values: number[], minimum: number, maximum: number): string {
    if (values.length === 0) return ''
    const spread = maximum - minimum || 1
    return values.map((value, index) => {
      const x = values.length === 1 ? 50 : 5 + (index / (values.length - 1)) * 90
      const y = 55 - ((value - minimum) / spread) * 45
      return `${x},${y}`
    }).join(' ')
  }

  function title(value: string): string {
    return value.charAt(0).toUpperCase() + value.slice(1)
  }

  async function save(): Promise<void> {
    saving = true
    saved = false
    error = ''
    try {
      await getAPI().updateProfile({ learnerRating, sessionSize })
      saved = true
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      saving = false
    }
  }
</script>

<section class="parent-dashboard" aria-labelledby="parent-title">
  <div class="dashboard-heading">
    <div>
      <p class="eyebrow">For parents</p>
      <h2 id="parent-title">Progress & settings</h2>
    </div>
    <button class="secondary" type="button" on:click={() => dispatch('import')}>Import puzzles</button>
  </div>

  {#if loading}
    <p class="loading" aria-live="polite">Gathering progress…</p>
  {:else if summary}
    <div class="metric-grid">
      <article><span>Current level</span><strong>{Math.round(summary.learnerRating)}</strong></article>
      <article><span>First-try accuracy</span><strong>{summary.firstAttemptAccuracy}%</strong></article>
      <article><span>Hint use</span><strong>{summary.hintRate}%</strong></article>
      <article><span>Review queue</span><strong>{summary.dueReviews} due</strong></article>
    </div>

    <div class="dashboard-grid">
      <article class="dashboard-card trend-card">
        <h3>Rating trend</h3>
        {#if ratings.length > 0}
          <svg role="img" aria-label="Learner rating trend" viewBox="0 0 100 60" preserveAspectRatio="none">
            <polyline points={trendPoints} fill="none" stroke="currentColor" stroke-width="3" vector-effect="non-scaling-stroke" />
          </svg>
          <div class="trend-labels">
            <span>Minimum {Math.round(minimumRating)}</span>
            <span>Current {Math.round(summary.learnerRating)}</span>
            <span>Maximum {Math.round(maximumRating)}</span>
          </div>
        {:else}
          <p class="muted">The trend will appear after guided sessions.</p>
        {/if}
      </article>

      <article class="dashboard-card settings-card">
        <h3>Guided training settings</h3>
        <form on:submit|preventDefault={save}>
          <label>
            Current learner rating
            <input type="number" min="400" max="3000" bind:value={learnerRating} />
          </label>
          <label>
            Puzzles per guided session
            <select bind:value={sessionSize}>
              <option value={5}>5 puzzles</option>
              <option value={10}>10 puzzles</option>
              <option value={15}>15 puzzles</option>
            </select>
          </label>
          <button class="primary" type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save settings'}</button>
          {#if saved}<span class="saved" role="status">Settings saved</span>{/if}
        </form>
      </article>
    </div>

    <article class="dashboard-card table-card">
      <h3>Theme performance</h3>
      {#if summary.themePerformance.length > 0}
        <div class="table-scroll">
          <table>
            <thead><tr><th scope="col">Theme</th><th scope="col">Attempts</th><th scope="col">First try</th></tr></thead>
            <tbody>
              {#each summary.themePerformance as theme}
                <tr><td>{theme.theme}</td><td>{theme.attempts}</td><td>{theme.accuracy}%</td></tr>
              {/each}
            </tbody>
          </table>
        </div>
      {:else}
        <p class="muted">Theme results will appear after some puzzles.</p>
      {/if}
    </article>

    <article class="dashboard-card table-card">
      <h3>Recent sessions</h3>
      {#if summary.recentSessions.length > 0}
        <div class="table-scroll">
          <table>
            <thead><tr><th scope="col">Type</th><th scope="col">Completed</th><th scope="col">First try</th><th scope="col">Hints</th></tr></thead>
            <tbody>
              {#each summary.recentSessions as recent}
                <tr>
                  <td>{title(recent.mode)}</td>
                  <td>{recent.completed}/{recent.total}</td>
                  <td>{recent.firstTry}</td>
                  <td>{recent.usedHint}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {:else}
        <p class="muted">No completed sessions yet.</p>
      {/if}
    </article>
  {/if}

  {#if error}<p class="error" role="alert">{error}</p>{/if}
</section>

<style>
  .parent-dashboard { width: 100%; }
  .dashboard-heading { display: flex; margin-bottom: 24px; align-items: end; justify-content: space-between; gap: 16px; }
  .dashboard-heading h2 { margin: 6px 0 0; font-size: clamp(1.8rem, 4vw, 2.6rem); }
  .metric-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; }
  .metric-grid article, .dashboard-card {
    padding: 20px;
    border: 1px solid var(--ivory-200);
    border-radius: var(--radius-medium);
    background: var(--ivory-50);
    box-shadow: var(--shadow-soft);
  }
  .metric-grid article { display: grid; gap: 6px; }
  .metric-grid span { color: var(--ink-700); font-size: 0.88rem; font-weight: 800; }
  .metric-grid strong { color: var(--forest-700); font-size: 1.7rem; }
  .dashboard-grid { display: grid; margin-top: 16px; grid-template-columns: 1.2fr 1fr; gap: 16px; }
  .dashboard-card { margin-top: 16px; }
  .dashboard-grid .dashboard-card { margin-top: 0; }
  .dashboard-card h3 { margin: 0 0 16px; }
  .trend-card svg { width: 100%; height: 170px; color: var(--forest-600); background: linear-gradient(#fffdf7, #f4e5bd); border-radius: 12px; }
  .trend-labels { display: flex; margin-top: 10px; justify-content: space-between; color: var(--ink-700); font-size: 0.78rem; font-weight: 800; }
  .settings-card label { margin: 10px 0; }
  .settings-card form .primary { margin-top: 6px; }
  .saved { margin-left: 12px; color: var(--forest-700); font-weight: 800; }
  .table-scroll { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; }
  th, td { padding: 11px 10px; border-bottom: 1px solid var(--ivory-200); text-align: left; }
  th { color: var(--ink-700); font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.05em; }
  @media (max-width: 850px) {
    .metric-grid { grid-template-columns: 1fr 1fr; }
    .dashboard-grid { grid-template-columns: 1fr; }
  }
  @media (max-width: 520px) {
    .dashboard-heading { align-items: start; flex-direction: column; }
    .metric-grid { grid-template-columns: 1fr; }
    .trend-labels { flex-direction: column; gap: 4px; }
  }
</style>
