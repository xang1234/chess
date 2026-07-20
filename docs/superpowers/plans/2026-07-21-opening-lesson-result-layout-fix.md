# Opening Lesson Result and Layout Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the passive-activity `appliedMoves` contract failure and keep long opening lessons inside a board-safe scrolling card with permanently visible actions.

**Architecture:** Preserve the frontend's strict discriminated result decoder and correct the result semantics at the Go service source: only learner decisions carry `expected` feedback. Restructure the active lesson panel into an activity-keyed informational scroll region plus a fixed action footer, then prove the behavior in both component tests and a real desktop browser.

**Tech Stack:** Go 1.26.4, Wails 2.12.0, Svelte 3, TypeScript, Vitest/Testing Library, Playwright, CSS dynamic viewport units.

## Global Constraints

- Do not loosen validation for expected learner decisions: `feedback: "expected"` requires non-empty authoritative `appliedMoves` and `finalFen`.
- Concept, comparison, recap, and reference completions omit `feedback`, `appliedMoves`, and `finalFen`.
- Demonstrations omit learner feedback but retain non-empty authoritative frames and a final FEN.
- The desktop card height is `min(760px, calc(100dvh - 120px))`, preceded by the equivalent `vh` fallback.
- The stacked card height is `min(620px, calc(100dvh - 32px))`, preceded by the equivalent `vh` fallback.
- Course path, heading, progress, activity content, and feedback scroll together; the action footer never enters the scroll region.
- Recreate the scroll region when `current.activityId` changes so every activity starts at scroll position zero.
- Do not change variation explorer, course-tree, progress, review, recommendation, or private course content behavior.
- No new runtime dependency.

---

### Task 1: Make opening activity feedback semantically correct

**Files:**
- Modify: `internal/openings/service_activity_test.go:12-69`
- Modify: `internal/openings/service_test.go:113-152`
- Modify: `internal/openings/service_activity.go:47-116`
- Verify: `internal/openings/service_response_test.go:39-48`
- Verify: `frontend/src/lib/api.test.ts:341-390`

**Interfaces:**
- Consumes: `completeLessonActivity(..., attempt *AttemptRecord, ..., frames []domain.AppliedMove, finalFEN string) (OpeningActivityResult, error)`.
- Produces: `OpeningActivityResult.Feedback == ""` when `attempt == nil`; `FeedbackExpected` when `attempt != nil`.
- Preserves: demonstration frames/final FEN and decision frames/final FEN.

- [ ] **Step 1: Write failing service assertions for passive and demonstration semantics**

In `TestOpeningServiceCompletesActivitiesAndReturnsCheckpoint`, strengthen the concept assertion:

```go
if !concept.ActivityCompleted || concept.Session.Current.CompletedIdeas != 1 ||
	concept.Feedback != "" || len(concept.AppliedMoves) != 0 || concept.FinalFEN != "" {
	t.Fatalf("concept=%+v", concept)
}
```

In `TestOpeningServiceSequencesLessonFeedbackHintsAndCompletion`, replace the two advance-result checks with:

```go
if advanced.Feedback != "" || len(advanced.AppliedMoves) != 0 || advanced.FinalFEN != "" {
	t.Fatalf("explain advance = %+v", advanced)
}

// After advancing the demonstration:
if advanced.Feedback != "" || len(advanced.AppliedMoves) != 6 ||
	advanced.FinalFEN != fixture.compiled.Positions["after-bc5"].FEN {
	t.Fatalf("watch advance = %+v", advanced)
}
```

Keep the existing learner-decision assertion:

```go
if decision.Feedback != FeedbackExpected || len(decision.AppliedMoves) != 1 {
	t.Fatalf("decision=%+v", decision)
}
```

- [ ] **Step 2: Run the focused service tests and verify RED**

Run:

```bash
go test ./internal/openings \
  -run 'TestOpeningService(CompletesActivitiesAndReturnsCheckpoint|SequencesLessonFeedbackHintsAndCompletion)' \
  -count=1
```

Expected: FAIL because concept/demonstration results currently contain `FeedbackExpected`.

- [ ] **Step 3: Return expected feedback only for a real learner attempt**

In `completeLessonActivity`, replace the direct return literal with:

```go
result := OpeningActivityResult{
	Session: view, ActivityCompleted: true, StepCompleted: true,
	AppliedMoves: frames, FinalFEN: finalFEN, Checkpoint: checkpoint,
}
if attempt != nil {
	result.Feedback = FeedbackExpected
}
return result, nil
```

Do not change `completeReviewDecision`; review decisions are learner attempts and continue returning `FeedbackExpected`.

- [ ] **Step 4: Run backend and frontend contract checks and verify GREEN**

Run:

```bash
go test ./internal/openings -count=1
npm --prefix frontend test -- --run src/lib/api.test.ts src/components/openings/opening-controller.test.ts
```

Expected: all focused tests PASS. The existing decoder test must still reject `feedback: "expected"` without `appliedMoves`, while accepting a passive completion with those fields omitted.

- [ ] **Step 5: Commit the semantic contract fix**

```bash
git add internal/openings/service_activity.go \
  internal/openings/service_activity_test.go internal/openings/service_test.go
git commit -m "fix: distinguish passive opening completions"
```

---

### Task 2: Bound lesson details and pin the action footer

**Files:**
- Modify: `frontend/src/components/openings/OpeningLessonScreen.test.ts:85-127`
- Modify: `frontend/tests/test-backend.ts:514-571`
- Modify: `frontend/tests/openings.spec.ts:46-80`
- Modify: `frontend/src/components/openings/OpeningLessonScreen.svelte:146-330`

**Interfaces:**
- Produces: `role="region" aria-label="Opening lesson details" tabindex="0"` for activity-scoped informational content.
- Produces: `role="group" aria-label="Opening lesson actions"` outside that region.
- Preserves: all existing controller events and button conditions.
- Uses: `{#key current.activityId}` to replace the region and reset native `scrollTop` on activity changes.

- [ ] **Step 1: Write the failing component structure/reset test**

Add this test to `OpeningLessonScreen.test.ts`:

```ts
test('keeps actions outside the activity-scoped lesson details region', async () => {
  const first = active('concept', {
    activityId: 'long-concept',
    title: 'A long concept',
    teachingNoteTexts: Array.from({ length: 16 }, (_, index) => `Detail ${index + 1}`)
  })
  const { component } = render(OpeningLessonScreen, {
    session: first,
    effects: effects(),
    boardAdapterFactory: boardHarness().factory
  }, withNormalAPI(fakeAPI()))

  const details = await screen.findByRole('region', { name: 'Opening lesson details' })
  const actions = screen.getByRole('group', { name: 'Opening lesson actions' })
  expect(details).toContainElement(screen.getByRole('heading', { name: 'A long concept' }))
  expect(details).not.toContainElement(actions)

  details.scrollTop = 180
  component.$set({
    session: active('recap', { activityId: 'next-recap', title: 'Next recap' })
  })

  await screen.findByRole('heading', { name: 'Next recap' })
  const nextDetails = screen.getByRole('region', { name: 'Opening lesson details' })
  expect(nextDetails).not.toBe(details)
  expect(nextDetails.scrollTop).toBe(0)
})
```

- [ ] **Step 2: Add long synthetic content and a failing desktop browser assertion**

In the test backend's `giuoco-concept`, keep the existing first teaching note and append synthetic detail:

```ts
teachingNoteTexts: [
  'Connect c3 with the later d4 break; this is one plan, not a move drill.',
  ...Array.from(
    { length: 16 },
    (_, index) => `Extended lesson detail ${index + 1}: compare the central plan before continuing.`
  )
],
```

In `openings.spec.ts`, after the first concept is visible and before opening deeper analysis, add:

```ts
const lessonDetails = page.getByRole('region', { name: 'Opening lesson details' })
const lessonActions = page.getByRole('group', { name: 'Opening lesson actions' })
const lessonBoard = chessgroundBoard(page)

await expect.poll(() => lessonDetails.evaluate(
  (element) => element.scrollHeight > element.clientHeight
)).toBe(true)

const boardBefore = await lessonBoard.boundingBox()
const actionsBefore = await lessonActions.boundingBox()
const viewport = page.viewportSize()
if (!boardBefore || !actionsBefore || !viewport) throw new Error('lesson layout has no bounds')
expect(boardBefore.y).toBeGreaterThanOrEqual(0)
expect(boardBefore.y + boardBefore.height).toBeLessThanOrEqual(viewport.height)

await lessonDetails.evaluate((element) => { element.scrollTop = element.scrollHeight })
await expect.poll(() => lessonDetails.evaluate((element) => element.scrollTop)).toBeGreaterThan(0)
const boardAfter = await lessonBoard.boundingBox()
const actionsAfter = await lessonActions.boundingBox()
if (!boardAfter || !actionsAfter) throw new Error('lesson layout lost its bounds')
expect(Math.abs(boardAfter.y - boardBefore.y)).toBeLessThanOrEqual(1)
expect(Math.abs(actionsAfter.y - actionsBefore.y)).toBeLessThanOrEqual(1)
```

After the two Continue clicks reach “Choose the preparation”, add:

```ts
await expect.poll(() => page.getByRole('region', {
  name: 'Opening lesson details'
}).evaluate((element) => element.scrollTop)).toBe(0)
```

- [ ] **Step 3: Run component and browser tests and verify RED**

Run:

```bash
npm --prefix frontend test -- --run src/components/openings/OpeningLessonScreen.test.ts
npm --prefix frontend run test:e2e -- openings.spec.ts
```

Expected: FAIL because no labelled internal scroll region or labelled fixed action group exists; the long content grows the current card.

- [ ] **Step 4: Introduce the activity-keyed details region and fixed footer**

In the active lesson `<aside>`, wrap every informational element from `OpeningPathContext` through `.opening-feedback` in:

```svelte
{#key current.activityId}
  <div
    class="opening-lesson-scroll"
    role="region"
    aria-label="Opening lesson details"
    tabindex="0"
  >
    <OpeningPathContext path={state.session.path} />

    <div class="opening-lesson-heading">
      <div>
        <p class="eyebrow">
          Opening course{current.variationName ? ` · ${current.variationName}` : ''}
        </p>
        <h2>{current.title}</h2>
      </div>
      <button
        class="sound-toggle"
        type="button"
        aria-label={view.soundMuted ? 'Turn sound on' : 'Mute sounds'}
        aria-pressed={view.soundMuted}
        on:click={() => controller.toggleSound()}
      >
        <span aria-hidden="true">{view.soundMuted ? '🔇' : '🔊'}</span>
      </button>
    </div>

    <div>
      <p class="progress-label">
        Idea {current.activityNumber} of {current.activityTotal}
        · {current.completedIdeas} learned
      </p>
      <div class="progress-track" aria-hidden="true">
        <span style={`width: ${(current.activityNumber / current.activityTotal) * 100}%`}></span>
      </div>
    </div>

    <OpeningActivityContent
      activity={current}
      {canReplayDemonstration}
      on:replayMoves={() => controller.replayMovesToHere()}
      on:replayDemonstration={() => controller.replayDemonstration()}
    />

    <div class="opening-feedback" aria-live="polite" aria-atomic="true">
      {#if view.message}<p class:neutral={view.feedback !== null}>{view.message}</p>{/if}
      {#if view.notice}<p class="notice">{view.notice}</p>{/if}
      {#if view.announcement && view.announcement !== view.message && view.announcement !== view.notice}
        <p class="visually-hidden">{view.announcement}</p>
      {/if}
    </div>
  </div>
{/key}
```

Keep `.opening-actions` immediately after the keyed block and add semantics while preserving its button branches:

```svelte
<div class="opening-actions" role="group" aria-label="Opening lesson actions">
  {#if state.phase === 'activity-complete'}
    <button class="primary" type="button" on:click={() => controller.acknowledgeActivity()}>
      Continue
    </button>
  {:else if state.phase === 'failed' && state.recoverable && state.retryOperation}
    <button class="primary" type="button" on:click={() => controller.retry()}>
      Retry
    </button>
    {#if state.retryOperation !== 'pause'}
      <button class="quiet-action" type="button" on:click={() => controller.pause()}>
        Pause lesson
      </button>
    {/if}
  {:else if state.phase === 'passive'}
    <button class="primary" type="button" on:click={() => controller.advance()}>
      Continue
    </button>
    <button class="quiet-action" type="button" on:click={() => controller.pause()}>
      Pause lesson
    </button>
  {:else}
    <button class="primary" type="button" disabled={!inputEnabled} on:click={() => controller.useHint()}>
      Hint
    </button>
    {#if canReveal}
      <button class="secondary" type="button" disabled={!inputEnabled} on:click={() => controller.reveal()}>
        Show course move
      </button>
    {/if}
    <button class="quiet-action" type="button" disabled={!inputEnabled} on:click={() => controller.pause()}>
      Pause lesson
    </button>
  {/if}
</div>
```

- [ ] **Step 5: Apply the bounded two-zone CSS**

Replace the lesson panel sizing and add the scroll/footer rules:

```css
.opening-lesson-panel {
  display: flex;
  height: min(760px, calc(100vh - 120px));
  height: min(760px, calc(100dvh - 120px));
  min-height: 0;
  padding: 26px;
  overflow: hidden;
  flex-direction: column;
  gap: 16px;
  border: 1px solid #3b4843;
  border-radius: var(--radius-large);
  color: #f7f4e9;
  background: var(--charcoal-800);
  box-shadow: var(--shadow-soft);
}
.opening-lesson-scroll {
  display: flex;
  min-height: 0;
  padding-right: 4px;
  overflow-y: auto;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 20px;
  scrollbar-gutter: stable;
}
.opening-lesson-scroll:focus-visible {
  outline: 2px solid var(--amber-400);
  outline-offset: 2px;
}
.opening-feedback {
  display: grid;
  min-height: 72px;
  align-content: center;
  gap: 7px;
}
.opening-actions {
  display: grid;
  flex: 0 0 auto;
  gap: 10px;
}
```

At `@media (max-width: 860px)`, replace the current `min-height: auto` override with:

```css
.opening-lesson-panel {
  height: min(620px, calc(100vh - 32px));
  height: min(620px, calc(100dvh - 32px));
}
```

- [ ] **Step 6: Run focused frontend checks and verify GREEN**

Run:

```bash
npm --prefix frontend test -- --run src/components/openings/OpeningLessonScreen.test.ts
npm --prefix frontend run check
npm --prefix frontend run test:e2e -- openings.spec.ts
```

Expected: component tests PASS, Svelte/TypeScript report zero diagnostics, and both Chromium and WebKit opening tests PASS with an overflowing details region, stable board/footer positions, and scroll reset.

- [ ] **Step 7: Commit the bounded lesson layout**

```bash
git add frontend/src/components/openings/OpeningLessonScreen.svelte \
  frontend/src/components/openings/OpeningLessonScreen.test.ts \
  frontend/tests/test-backend.ts frontend/tests/openings.spec.ts
git commit -m "fix: keep opening lesson actions visible"
```

---

### Task 3: Verify and rebuild the macOS application

**Files:**
- Verify only: all Go and frontend source
- Generate ignored artifact: `build/bin/Chess Trainer.app`
- Normalize if regenerated: `frontend/wailsjs/go/models.ts`
- Normalize modes: `frontend/wailsjs/runtime/package.json`, `frontend/wailsjs/runtime/runtime.d.ts`, `frontend/wailsjs/runtime/runtime.js`

**Interfaces:**
- Consumes: the completed semantic and layout commits.
- Produces: a clean, self-signed Apple Silicon local app at `build/bin/Chess Trainer.app`.

- [ ] **Step 1: Run the complete automated gate**

Run:

```bash
go test ./... -count=1
go vet ./...
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run verify:licenses
npm --prefix frontend run test:e2e -- openings.spec.ts
npm --prefix frontend run build
git diff --check
```

Expected: all Go packages PASS; all frontend test files PASS; Svelte/TypeScript show zero diagnostics; legal verification, opening E2E in Chromium/WebKit, production bundle, and diff check all exit zero.

- [ ] **Step 2: Audit the exact regression boundaries**

Run:

```bash
rg -n 'Feedback: FeedbackExpected' internal/openings/service_activity.go
rg -n 'opening-lesson-scroll|Opening lesson actions|100dvh - 120px' \
  frontend/src/components/openings/OpeningLessonScreen.svelte
git status --short --branch
```

Expected: no unconditional `FeedbackExpected` assignment remains in `completeLessonActivity`; the layout markers exist; only committed feature history is present and the worktree is clean.

- [ ] **Step 3: Rebuild with the pinned local Wails toolchain**

Run:

```bash
GOWORK=off GOTOOLCHAIN=local \
  go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json \
  frontend/wailsjs/runtime/runtime.d.ts \
  frontend/wailsjs/runtime/runtime.js
```

Expected: Wails reports `Done` for bindings, frontend, native compilation, packaging, and self-signing.

- [ ] **Step 4: Verify and launch the rebuilt application**

Run:

```bash
git diff --check
git status --short --branch
codesign --verify --deep --strict --verbose=2 "build/bin/Chess Trainer.app"
file "build/bin/Chess Trainer.app/Contents/MacOS/Chess Trainer"
open "build/bin/Chess Trainer.app"
```

Expected: the source tree is clean; codesign reports “valid on disk” and “satisfies its Designated Requirement”; `file` reports a Mach-O 64-bit arm64 executable; the rebuilt app launches.

---

## Final Review Checklist

- [ ] Passive results no longer serialize learner move feedback.
- [ ] Demonstrations still animate authoritative frames.
- [ ] Expected decisions remain decoder-strict.
- [ ] Long lesson details overflow internally.
- [ ] Action buttons and board remain fixed while details scroll.
- [ ] New activities start at the top of the details region.
- [ ] All checks and the local macOS rebuild pass.
