<script lang="ts">
  import { onMount } from 'svelte'
  import HomeHub from './components/home/HomeHub.svelte'
  import ImportPanel from './components/import/ImportPanel.svelte'
  import InitialSetup from './components/parent/InitialSetup.svelte'
  import ParentDashboard from './components/parent/ParentDashboard.svelte'
  import FreePractice from './components/practice/FreePractice.svelte'
  import PuzzleScreen from './components/puzzle/PuzzleScreen.svelte'
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

  function leaveTraining(event: CustomEvent<{ completed: boolean }>): void {
    if (event.detail.completed) activeSession = null
    screen.set('home')
  }

  function startPractice(event: CustomEvent<SessionView>): void {
    activeSession = event.detail
    screen.set('puzzle')
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
        activeSession={activeSession !== null && activeSession.status !== 'completed'}
        on:training={openTraining}
        on:practice={() => screen.set('practice')}
        on:games={() => screen.set('games')}
        on:parent={() => screen.set('parent')}
      />
    {:else if $screen === 'import'}
      <ImportPanel />
    {:else if $screen === 'puzzle' && activeSession}
      <PuzzleScreen
        session={activeSession}
        on:change={(event) => { activeSession = event.detail }}
        on:home={leaveTraining}
      />
    {:else if $screen === 'practice'}
      <FreePractice on:start={startPractice} />
    {:else if $screen === 'parent'}
      <ParentDashboard on:import={() => screen.set('import')} />
    {:else}
      <section class="panel placeholder-panel">
        <h2>{$screen === 'puzzle' ? 'Puzzle board' : 'Game Library'}</h2>
        <p>This area is the next part of the build.</p>
        <div class="button-row">
          <button class="secondary" type="button" on:click={() => screen.set('home')}>Back home</button>
        </div>
      </section>
    {/if}
  </main>
</div>
