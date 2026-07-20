<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import type { OpeningActivityView } from '../../lib/api'

  export let activity: OpeningActivityView
  export let canReplayDemonstration = false

  const dispatch = createEventDispatcher<{
    replayMoves: void
    replayDemonstration: void
  }>()

  $: hasDeeperAnalysis = activity.referenceNoteTexts.length > 0 ||
    activity.referenceSections.length > 0
</script>

<div class="opening-activity-content">
  <p class="opening-activity-instruction">{activity.instruction}</p>

  {#if activity.teachingNoteTexts.length > 0}
    <div class="teaching-notes">
      {#each activity.teachingNoteTexts as note}<p>{note}</p>{/each}
    </div>
  {/if}

  {#if activity.comparison.length > 0}
    <div class="opening-comparison" aria-label="Plan comparison">
      {#each activity.comparison as line (line.label)}
        <article>
          <strong>{line.label}</strong>
          <span>{line.moveIds.join(' → ')}</span>
        </article>
      {/each}
    </div>
  {/if}

  {#if activity.movesToHere.length > 0 || canReplayDemonstration}
    <div class="opening-replay-actions">
      {#if activity.movesToHere.length > 0}
        <button class="quiet-action" type="button" on:click={() => dispatch('replayMoves')}>
          Replay moves to here
        </button>
      {/if}
      {#if canReplayDemonstration}
        <button class="quiet-action" type="button" on:click={() => dispatch('replayDemonstration')}>
          Replay demonstration
        </button>
      {/if}
    </div>
  {/if}

  {#if hasDeeperAnalysis}
    <details class="opening-deeper-analysis">
      <summary>Deeper analysis</summary>
      {#each activity.referenceNoteTexts as note}<p>{note}</p>{/each}
      {#each activity.referenceSections as section (section.activityId)}
        <section>
          <h3>{section.title}</h3>
          <p>{section.instruction}</p>
          {#each section.noteTexts as note}<p>{note}</p>{/each}
        </section>
      {/each}
    </details>
  {/if}
</div>
