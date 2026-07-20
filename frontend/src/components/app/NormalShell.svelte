<script lang="ts">
  import { onMount } from 'svelte'
  import type {
    BuildInfo,
    NormalAPI,
    OpeningDepth,
    OpeningHomeView,
    OpeningPathItem,
    OpeningSessionView,
    SessionView
  } from '../../lib/api'
  import { provideNormalAPI } from '../../lib/api-context'
  import { createImportSession } from '../../lib/import-session'
  import { screen } from '../../lib/navigation'
  import HomeHub from '../home/HomeHub.svelte'
  import ImportPanel from '../import/ImportPanel.svelte'
  import AboutLegal from '../legal/AboutLegal.svelte'
  import OpeningHub from '../openings/OpeningHub.svelte'
  import OpeningLessonScreen from '../openings/OpeningLessonScreen.svelte'
  import VariationExplorer from '../openings/VariationExplorer.svelte'
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
  let openingHome: OpeningHomeView = { courses: [] }
  let activeOpeningSession: OpeningSessionView | null = null
  let deferredOpeningSession: OpeningSessionView | null = null
  let explorerCourseId = ''
  let explorerPositionId = ''
  let explorerDepth: OpeningDepth = 'reference'
  let error = ''
  $: activeOpeningPath = openingPath(activeOpeningSession)
  const puzzleImportSession = createImportSession(() => api, 'puzzle')
  const courseImportSession = createImportSession(() => api, 'course')
  let disconnectImport = (): void => {}
  let refreshedCourseImportJobId = ''
  const disconnectCourseImportRefresh = courseImportSession.subscribe((state) => {
    if (
      state.phase !== 'finished' ||
      state.result.status !== 'succeeded' ||
      state.jobId === refreshedCourseImportJobId
    ) return
    refreshedCourseImportJobId = state.jobId
    void refreshOpeningHome()
  })

  onMount(() => {
    void initialise()
    return () => {
      disconnectImport()
      disconnectCourseImportRefresh()
    }
  })

  async function initialise(): Promise<void> {
    try {
      const disconnectCourse = courseImportSession.connect()
      const disconnectPuzzle = puzzleImportSession.connect()
      disconnectImport = () => {
        disconnectPuzzle()
        disconnectCourse()
      }
      const profile = await api.getProfile()
      if (!profile) {
        screen.set('setup')
      } else {
        const [puzzleSession, openingSessionResult, openingHomeResult] = await Promise.all([
          api.resumeSession(),
          safeOpeningRead(() => api.resumeOpeningSession(), null),
          safeOpeningRead(() => api.getOpeningHome(), { courses: [] })
        ])
        activeSession = puzzleSession
        activeOpeningSession = openingSessionResult.value
        openingHome = openingHomeResult.value
        const openingNotice = openingHomeResult.error || openingSessionResult.error
        if (openingNotice && !openingHome.notice) {
          openingHome = { ...openingHome, notice: openingNotice }
        }
        if ($screen !== 'legal') screen.set('home')
      }
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      loading = false
    }
  }

  async function safeOpeningRead<Value>(
    operation: () => Promise<Value>,
    fallback: Value
  ): Promise<{ value: Value; error?: string }> {
    try {
      return { value: await operation() }
    } catch (cause) {
      return { value: fallback, error: errorMessage(cause) }
    }
  }

  async function refreshOpeningHome(): Promise<void> {
    const result = await safeOpeningRead(() => api.getOpeningHome(), openingHome)
    openingHome = result.error
      ? { ...result.value, notice: result.error }
      : result.value
  }

  function errorMessage(cause: unknown): string {
    return cause instanceof Error ? cause.message : String(cause)
  }

  function showOpeningError(cause: unknown): void {
    openingHome = { ...openingHome, notice: errorMessage(cause) }
    screen.set('openings')
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

  function adoptVisibleOpeningSession(event: CustomEvent<OpeningSessionView>): void {
    activeOpeningSession = event.detail
    deferredOpeningSession = null
    syncOpeningResume(event.detail)
  }

  function rememberPersistedOpeningSession(event: CustomEvent<OpeningSessionView>): void {
    deferredOpeningSession = event.detail
    syncOpeningResume(event.detail)
  }

  function syncOpeningResume(session: OpeningSessionView): void {
    openingHome = {
      ...openingHome,
      courses: openingHome.courses.map((course) => course.courseId === session.courseId
        ? {
          ...course,
          hasResumable: session.status !== 'completed',
          ...(session.status !== 'completed' ? { nextLessonId: session.lessonId } : {})
        }
        : course)
    }
  }

  function goHome(): void {
    if (deferredSession) {
      activeSession = deferredSession.status !== 'completed' && deferredSession.current
        ? deferredSession
        : null
      deferredSession = null
    }
    if (deferredOpeningSession) {
      activeOpeningSession = deferredOpeningSession.status === 'completed'
        ? null
        : deferredOpeningSession
      syncOpeningResume(deferredOpeningSession)
      deferredOpeningSession = null
    } else if (activeOpeningSession) {
      syncOpeningResume(activeOpeningSession)
      if (activeOpeningSession.status === 'completed') activeOpeningSession = null
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

  function leaveOpeningLesson(event: CustomEvent<{ completed: boolean }>): void {
    if (event.detail.completed) {
      if (activeOpeningSession) syncOpeningResume(activeOpeningSession)
      activeOpeningSession = null
      deferredOpeningSession = null
      screen.set('home')
      return
    }
    goHome()
  }

  async function changeOpeningDepth(
    event: CustomEvent<{ courseId: string; depth: OpeningDepth }>
  ): Promise<void> {
    try {
      await api.setOpeningDepth(event.detail.courseId, event.detail.depth)
      openingHome = await api.getOpeningHome()
    } catch (cause) {
      showOpeningError(cause)
    }
  }

  async function startOpeningLesson(
    event: CustomEvent<{ courseId: string; lessonId: string }>
  ): Promise<void> {
    try {
      activeOpeningSession = await api.startOpeningLesson(
        event.detail.courseId,
        event.detail.lessonId
      )
      deferredOpeningSession = null
      syncOpeningResume(activeOpeningSession)
      await refreshOpeningHome()
      screen.set('opening-lesson')
    } catch (cause) {
      showOpeningError(cause)
    }
  }

  async function resumeOpening(): Promise<void> {
    try {
      activeOpeningSession = await api.resumeOpeningSession()
      if (!activeOpeningSession) throw new Error('No opening lesson is ready to continue.')
      deferredOpeningSession = null
      syncOpeningResume(activeOpeningSession)
      screen.set('opening-lesson')
    } catch (cause) {
      showOpeningError(cause)
    }
  }

  async function startOpeningReview(event: CustomEvent<string>): Promise<void> {
    try {
      activeOpeningSession = await api.startOpeningReview(event.detail)
      deferredOpeningSession = null
      syncOpeningResume(activeOpeningSession)
      screen.set('opening-lesson')
    } catch (cause) {
      showOpeningError(cause)
    }
  }

  async function showOpeningTree(): Promise<void> {
    await refreshOpeningHome()
    if (activeOpeningSession?.status === 'completed') activeOpeningSession = null
    deferredOpeningSession = null
    screen.set('openings')
  }

  function openingPath(session: OpeningSessionView | null): OpeningPathItem[] {
    if (!session) return []
    const course = openingHome.courses.find((candidate) => candidate.courseId === session.courseId)
    if (!course) return []
    const nodes = new Map(course.tree.nodes.map((node) => [node.lessonId, node]))
    if (!nodes.has(session.lessonId)) return course.currentPath
    const parents = new Map(course.tree.edges.map((edge) => [edge.toLessonId, edge.fromLessonId]))
    const path: OpeningPathItem[] = []
    const seen = new Set<string>()
    let lessonId: string | undefined = session.lessonId
    while (lessonId && !seen.has(lessonId)) {
      seen.add(lessonId)
      const node = nodes.get(lessonId)
      if (!node) break
      path.push({ lessonId, title: node.title })
      lessonId = parents.get(lessonId)
    }
    return path.reverse()
  }

  function exploreOpening(
    event: CustomEvent<{ courseId: string; positionId: string }>
  ): void {
    explorerCourseId = event.detail.courseId
    explorerPositionId = event.detail.positionId
    explorerDepth = openingHome.courses.find(
      (course) => course.courseId === event.detail.courseId
    )?.depth ?? 'reference'
    screen.set('opening-explorer')
  }
</script>

<div class="app-shell" class:board-active={$screen === 'puzzle' || $screen === 'opening-lesson'}>
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
        {openingHome}
        on:training={openTraining}
        on:practice={() => screen.set('practice')}
        on:openings={() => screen.set('openings')}
        on:games={() => screen.set('games')}
        on:parent={() => screen.set('parent')}
      />
    {:else if $screen === 'openings'}
      <OpeningHub
        home={openingHome}
        on:back={goHome}
        on:depth={changeOpeningDepth}
        on:lesson={startOpeningLesson}
        on:resume={resumeOpening}
        on:review={startOpeningReview}
        on:explore={exploreOpening}
      />
    {:else if $screen === 'opening-lesson' && activeOpeningSession}
      <OpeningLessonScreen
        session={activeOpeningSession}
        path={activeOpeningPath}
        on:change={adoptVisibleOpeningSession}
        on:persisted={rememberPersistedOpeningSession}
        on:continue={startOpeningLesson}
        on:tree={showOpeningTree}
        on:home={leaveOpeningLesson}
      />
    {:else if $screen === 'opening-explorer'}
      <VariationExplorer
        courseId={explorerCourseId}
        rootPositionId={explorerPositionId}
        depth={explorerDepth}
        on:back={() => screen.set('openings')}
      />
    {:else if $screen === 'legal'}
      <AboutLegal {buildInfo} on:back={goHome} />
    {:else if $screen === 'import'}
      <ImportPanel
        puzzleSession={puzzleImportSession}
        courseSession={courseImportSession}
      />
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
