<script lang="ts">
  import { onMount } from 'svelte'
  import NormalShell from './components/app/NormalShell.svelte'
  import RecoveryShell from './components/app/RecoveryShell.svelte'
  import {
    loadApplicationAPI,
    type ApplicationAPI
  } from './lib/api'

  export let loadAPI: () => Promise<ApplicationAPI> = loadApplicationAPI

  let application: ApplicationAPI | null = null
  let error = ''

  onMount(async () => {
    try {
      application = await loadAPI()
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    }
  })
</script>

<svelte:head><title>Chess Trainer</title></svelte:head>

{#if application?.mode === 'normal'}
  <NormalShell api={application.api} />
{:else if application?.mode === 'recovery'}
  <RecoveryShell api={application.api} />
{:else}
  <div class="app-shell">
    <header class="app-header">
      <div class="brand">
        <span class="brand-mark" aria-hidden="true">♞</span>
        <h1>Chess Trainer</h1>
      </div>
    </header>
    <main class="app-main">
      {#if error}
        <section class="panel" role="alert">
          <h2>Something needs attention</h2>
          <p>{error}</p>
        </section>
      {:else}
        <p class="loading" aria-live="polite">Opening your chess room…</p>
      {/if}
    </main>
  </div>
{/if}
