<script lang="ts">
  import { onMount } from 'svelte'
  import type { RecoveryAPI, RecoveryState } from '../../lib/api'
  import { provideRecoveryAPI } from '../../lib/api-context'
  import RecoveryPanel from '../parent/RecoveryPanel.svelte'

  export let api: RecoveryAPI
  provideRecoveryAPI(api)

  let loading = true
  let state: RecoveryState | null = null
  let error = ''

  onMount(async () => {
    try {
      state = await api.getRecoveryState()
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      loading = false
    }
  })
</script>

<div class="app-shell">
  <header class="app-header">
    <div class="brand">
      <span class="brand-mark" aria-hidden="true">♞</span>
      <h1>Chess Trainer</h1>
    </div>
  </header>

  <main class="app-main">
    {#if loading}
      <p class="loading" aria-live="polite">Opening recovery tools…</p>
    {:else if error}
      <section class="panel" role="alert">
        <h2>Something needs attention</h2>
        <p>{error}</p>
      </section>
    {:else if state}
      <RecoveryPanel {state} />
    {/if}
  </main>
</div>
