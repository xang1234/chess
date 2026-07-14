<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte'
  import { getAPI, type PracticeFilters, type PracticeSource, type SessionView } from '../../lib/api'

  const dispatch = createEventDispatcher<{ start: SessionView }>()
  let filters: PracticeFilters | null = null
  let sourceId = ''
  let useRating = false
  let minimumRating = 0
  let maximumRating = 0
  let themes: string[] = []
  let maximumSolutionPlies = 1
  let loading = true
  let starting = false
  let error = ''

  $: selectedSource = filters?.sources.find((source) => source.id === sourceId)

  onMount(async () => {
    try {
      filters = await getAPI().getPracticeFilters()
      if (filters.sources.length > 0) selectSource(filters.sources[0])
      maximumSolutionPlies = Math.max(1, filters.maximumSolutionPlies)
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      loading = false
    }
  })

  function sourceChanged(): void {
    const source = filters?.sources.find((candidate) => candidate.id === sourceId)
    if (source) selectSource(source)
  }

  function selectSource(source: PracticeSource): void {
    sourceId = source.id
    useRating = false
    minimumRating = source.minimumRating
    maximumRating = source.maximumRating
    themes = []
    maximumSolutionPlies = Math.max(1, source.maximumPlies)
  }

  function toggleTheme(theme: string, checked: boolean): void {
    themes = checked
      ? [...themes, theme].sort()
      : themes.filter((value) => value !== theme)
  }

  async function start(): Promise<void> {
    starting = true
    error = ''
    try {
      const session = await getAPI().startFreePractice({
        sourceId,
        ...(useRating ? { minimumRating, maximumRating } : {}),
        themes,
        maximumSolutionPlies
      })
      dispatch('start', session)
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      starting = false
    }
  }
</script>

<section class="practice panel" aria-labelledby="practice-title">
  <p class="eyebrow">Choose your challenge</p>
  <h2 id="practice-title">Free Practice</h2>
  <p class="muted">Pick what you want to work on. Practice never changes your guided-training level.</p>

  {#if loading}
    <p class="loading" aria-live="polite">Opening the puzzle shelf…</p>
  {:else if filters && filters.sources.length > 0}
    <form on:submit|preventDefault={start}>
      <label>
        Puzzle source
        <select bind:value={sourceId} on:change={sourceChanged}>
          {#each filters.sources as source}
            <option value={source.id}>{source.kind === 'lichess' ? 'Lichess puzzles' : source.id}</option>
          {/each}
        </select>
      </label>

      {#if selectedSource?.hasRatingRange}
        <label class="check-row">
          <input type="checkbox" bind:checked={useRating} />
          Limit by rating
        </label>
        {#if useRating}
          <div class="two-fields">
            <label>
              Minimum rating
              <input type="number" min={selectedSource.minimumRating} max={maximumRating} bind:value={minimumRating} />
            </label>
            <label>
              Maximum rating
              <input type="number" min={minimumRating} max={selectedSource.maximumRating} bind:value={maximumRating} />
            </label>
          </div>
        {/if}
      {/if}

      {#if filters.themes.length > 0}
        <fieldset>
          <legend>What would you like to practise?</legend>
          <div class="theme-grid">
            {#each filters.themes as theme}
              <label class="theme-chip">
                <input
                  type="checkbox"
                  aria-label={theme}
                  checked={themes.includes(theme)}
                  on:change={(event) => toggleTheme(theme, event.currentTarget.checked)}
                />
                {theme}
              </label>
            {/each}
          </div>
        </fieldset>
      {/if}

      <label>
        Maximum solution length
        <select bind:value={maximumSolutionPlies}>
          {#each Array(selectedSource?.maximumPlies ?? filters.maximumSolutionPlies) as _, index}
            <option value={index + 1}>{index + 1} {index === 0 ? 'move' : 'moves'}</option>
          {/each}
        </select>
      </label>

      {#if error}<p class="error" role="alert">{error}</p>{/if}
      <button class="primary start-button" type="submit" disabled={starting || !sourceId}>
        {starting ? 'Starting…' : 'Start practice'}
      </button>
    </form>
  {:else}
    <p>No puzzle sources are available yet. Ask a parent to import puzzles first.</p>
  {/if}
</section>

<style>
  .practice { width: min(720px, 100%); }
  .practice form { margin-top: 24px; }
  .two-fields { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
  .check-row { display: flex; align-items: center; }
  .check-row input, .theme-chip input { width: 22px; min-height: 22px; margin: 0; }
  fieldset { margin: 24px 0; padding: 0; border: 0; }
  legend { margin-bottom: 10px; font-weight: 800; }
  .theme-grid { display: flex; flex-wrap: wrap; gap: 8px; }
  .theme-chip {
    display: flex;
    margin: 0;
    padding: 9px 12px;
    flex-direction: row;
    align-items: center;
    border: 1px solid var(--ivory-200);
    border-radius: 999px;
    background: var(--white);
    font-weight: 700;
  }
  .start-button { width: 100%; margin-top: 8px; }
  @media (max-width: 560px) { .two-fields { grid-template-columns: 1fr; gap: 0; } }
</style>
