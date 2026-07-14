<script lang="ts">
  import { onMount } from 'svelte'
  import HomeHub from './components/home/HomeHub.svelte'
  import ImportPanel from './components/import/ImportPanel.svelte'
  import InitialSetup from './components/parent/InitialSetup.svelte'
  import { getAPI, type SessionView } from './lib/api'
  import { screen } from './lib/navigation'

  let loading = true
  let activeSession: SessionView | null = null
  let error = ''

  onMount(async () => {
    try {
      const profile = await getAPI().getProfile()
      if (!profile) {
        screen.set('setup')
      } else {
        activeSession = await getAPI().resumeSession()
        screen.set('home')
      }
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      loading = false
    }
  })

  async function openTraining(): Promise<void> {
    error = ''
    try {
      activeSession = activeSession
        ? await getAPI().resumeSession()
        : await getAPI().startGuided()
      screen.set('puzzle')
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    }
  }
</script>

<svelte:head><title>Chess Trainer</title></svelte:head>

<div class="app-shell">
  <header class="app-header">
    <button class="brand" type="button" on:click={() => screen.set('home')} aria-label="Chess Trainer home">
      <span class="brand-mark" aria-hidden="true">♞</span>
      <h1>Chess Trainer</h1>
    </button>
  </header>

  <main class="app-main">
    {#if loading}
      <p class="loading" aria-live="polite">Opening your chess room…</p>
    {:else if error}
      <section class="panel" role="alert">
        <h2>Something needs attention</h2>
        <p>{error}</p>
        <button class="secondary" type="button" on:click={() => screen.set('home')}>Back home</button>
      </section>
    {:else if $screen === 'setup'}
      <InitialSetup on:complete={() => screen.set('home')} />
    {:else if $screen === 'home'}
      <HomeHub
        activeSession={activeSession !== null}
        on:training={openTraining}
        on:practice={() => screen.set('practice')}
        on:games={() => screen.set('games')}
        on:parent={() => screen.set('parent')}
      />
    {:else if $screen === 'import'}
      <ImportPanel />
    {:else}
      <section class="panel placeholder-panel">
        <h2>{$screen === 'puzzle' ? 'Puzzle board' : $screen === 'practice' ? 'Free Practice' : $screen === 'games' ? 'Game Library' : 'Parent area'}</h2>
        <p>This area is the next part of the build.</p>
        <div class="button-row">
          {#if $screen === 'parent'}
            <button class="primary" type="button" on:click={() => screen.set('import')}>Import puzzles</button>
          {/if}
          <button class="secondary" type="button" on:click={() => screen.set('home')}>Back home</button>
        </div>
      </section>
    {/if}
  </main>
</div>
