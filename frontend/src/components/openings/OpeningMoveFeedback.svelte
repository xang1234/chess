<script lang="ts">
  import type {
    OpeningMoveFeedback as FeedbackKind,
    OpeningMoveFeedbackDetail,
    OpeningPerspective
  } from '../../lib/api'
  import { openingFeedbackCopy } from './opening-feedback-copy'

  export let feedback: FeedbackKind | null
  export let detail: OpeningMoveFeedbackDetail | null
  export let orientation: OpeningPerspective
  export let fallbackMessage = ''

  $: copy = detail ? openingFeedbackCopy(detail, orientation) : null
</script>

<div
  class="move-feedback"
  class:has-feedback={feedback !== null}
  role="status"
  aria-live="polite"
  aria-atomic="true"
>
  {#if copy}
    <h3>{copy.heading}</h3>
    <p>{copy.explanation}</p>
    <p class="contrast">{copy.contrast}</p>
  {:else if fallbackMessage}
    <p>{fallbackMessage}</p>
  {/if}
</div>

<style>
  .move-feedback { display: grid; min-height: 72px; align-content: center; gap: 5px; }
  h3 { margin: 0; color: #f7f4e9; font-size: 1rem; }
  p { margin: 0; color: #e8e4d8; font-weight: 800; line-height: 1.4; }
  .has-feedback p { color: #c8dfcf; }
  .has-feedback .contrast { color: #f4d27b; }
</style>
