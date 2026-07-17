<script lang="ts">
  import { onMount } from 'svelte'
  import type { BuildInfo, NormalAPI, SessionView } from '../../lib/api'
  import { provideNormalAPI } from '../../lib/api-context'
  import { createImportSession } from '../../lib/import-session'
  import { screen } from '../../lib/navigation'
  import HomeHub from '../home/HomeHub.svelte'
  import ImportPanel from '../import/ImportPanel.svelte'
  import AboutLegal from '../legal/AboutLegal.svelte'
  import InitialSetup from '../parent/InitialSetup.svelte'
  import ParentDashboard from '../parent/ParentDashboard.svelte'
  import FreePractice from '../practice/FreePractice.svelte'
  import PuzzleScreen from '../puzzle/PuzzleScreen.svelte'

  export let api: NormalAPI
  export let buildInfo: BuildInfo
  provideNormalAPI(api)

  let loading = true
  let activeSession: SessionView | null = null
  let deferredSession: SessionView | null = null
  let error = ''
  const importSession = createImportSession(() => api)
  let disconnectImport = (): void => {}

  onMount(() => {
    void initialise()
    return () => disconnectImport()
  })

  async function initialise(): Promise<void> {
    try {
      disconnectImport = importSession.connect()
      const profile = await api.getProfile()
      if (!profile) {
        screen.set('setup')
      } else {
        activeSession = await api.resumeSession()
        screen.set('home')
      }
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      loading = false
    }
  }

  async function openTraining(): Promise<void> {
    error = ''
    try {
      activeSession = activeSession
        ? await api.resumeSession()
        : await api.startGuided()
      screen.set('puzzle')
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    }
  }

  function adoptVisibleSession(event: CustomEvent<SessionView>): void {
    activeSession = event.detail
    deferredSession = null
  }

  function rememberPersistedSession(event: CustomEvent<SessionView>): void {
    deferredSession = event.detail
  }

  function goHome(): void {
    if (deferredSession) {
      activeSession = deferredSession.status !== 'completed' && deferredSession.current
        ? deferredSession
        : null
      deferredSession = null
    }
    screen.set('home')
  }

  function leaveTraining(event: CustomEvent<{ completed: boolean }>): void {
    if (event.detail.completed) {
      activeSession = null
      deferredSession = null
      screen.set('home')
      return
    }
    goHome()
  }

  function startPractice(event: CustomEvent<SessionView>): void {
    activeSession = event.detail
    screen.set('puzzle')
  }
</script>

<div class="app-shell" class:puzzle-active={$screen === 'puzzle'}>
  <header class="app-header">
    <button class="brand" type="button" on:click={goHome} aria-label="Chess Trainer home">
      <span class="brand-mark" aria-hidden="true">♞</span>
      <h1>Chess Trainer</h1>
    </button>
    <button class="header-action" type="button" on:click={() => screen.set('legal')}>
      About &amp; Legal
    </button>
  </header>

  <main class="app-main">
    {#if loading}
      <p class="loading" aria-live="polite">Opening your chess room…</p>
    {:else if error}
      <section class="panel" role="alert">
        <h2>Something needs attention</h2>
        <p>{error}</p>
        <button class="secondary" type="button" on:click={goHome}>Back home</button>
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
    {:else if $screen === 'legal'}
      <AboutLegal {buildInfo} on:back={goHome} />
    {:else if $screen === 'import'}
      <ImportPanel session={importSession} />
    {:else if $screen === 'puzzle' && activeSession}
      <PuzzleScreen
        session={activeSession}
        on:change={adoptVisibleSession}
        on:persisted={rememberPersistedSession}
        on:home={leaveTraining}
      />
    {:else if $screen === 'practice'}
      <FreePractice on:start={startPractice} />
    {:else if $screen === 'parent'}
      <ParentDashboard on:import={() => screen.set('import')} />
    {:else}
      <section class="panel placeholder-panel">
        <h2>{$screen === 'puzzle' ? 'Puzzle board' : 'Game Library'}</h2>
        <p>{$screen === 'games' ? 'Game Library is planned for a future milestone.' : 'This area is the next part of the build.'}</p>
        <div class="button-row">
          <button class="secondary" type="button" on:click={goHome}>Back home</button>
        </div>
      </section>
    {/if}
  </main>
</div>
