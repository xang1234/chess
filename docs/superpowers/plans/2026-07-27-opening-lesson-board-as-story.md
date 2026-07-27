# Opening Lesson Board-as-Story Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make opening lessons progress without empty completion screens, preserve a continuous board, show the current SAN journey, and let learners play authored comparison lines.

**Architecture:** Keep the server authoritative for progress and every move frame. Add one guarded controller fast path for successful zero-frame activities, derive SAN and comparison frames from the compiled move graph, and render those additions through focused Svelte components. Comparison playback remains client-only preview state owned by the existing cancellable board runtime.

**Tech Stack:** Go 1.26.4, Wails 2.12.0, Svelte 3, TypeScript 4.6, Vitest/Testing Library, Playwright, Chessground.

## Global Constraints

- Auto-advance only a successful result with no applied moves, no checkpoint, and an active returned session with a current activity.
- Checkpoints, completed sessions, demonstrations, learner moves, and reveals retain an explicit completion pause.
- Emit the persisted-session event before the visible-session change event on the auto-advance path.
- Compare complete FEN strings before deciding whether to increment `boardGeneration`.
- Do not infer cross-activity animation frames from `movesToHere` prefixes.
- `movesToHere` and `movesToHereSan` come from the same server-selected route and have equal lengths.
- Comparison SAN and frames are non-empty, equal-length, authoritative server output.
- Comparison playback never changes persisted session, progress, review state, or `playedMoveIds`.
- The SAN ribbon is display-only; individual plies are not interactive in this release.
- Do not add XP, streaks, ratings, confidence prompts, a lesson transcript, or global keyboard shortcuts.
- Do not change the course-pack schema or require private course re-authoring.
- Preserve strict decoding, native buttons, reduced motion, board recovery, and 46px touch targets.
- Regenerate Wails bindings with pinned Wails 2.12.0; do not hand-edit generated models.
- Add no runtime dependency.

## File Structure

### New files

- `frontend/src/components/openings/OpeningMoveRibbon.svelte` — renders the variation badge and numbered SAN route beneath the board.
- `frontend/src/components/openings/OpeningMoveRibbon.test.ts` — verifies SAN numbering, empty-route copy, last-ply emphasis, and accessibility.

### Existing files with focused changes

- `internal/openings/views.go` and `service_activity.go` — produce route SAN, variation metadata, and comparison frames.
- `frontend/src/lib/contracts/openings.ts` — strictly decodes the expanded views.
- `frontend/src/components/openings/opening-state.ts` — validates the zero-frame transition.
- `frontend/src/components/openings/opening-controller.ts` — owns fast-forwarding, remount decisions, and preview lifecycle.
- `OpeningActivityContent.svelte` — renders comparison controls.
- `OpeningLessonScreen.svelte` — composes the ribbon and meaningful action labels.
- `frontend/src/styles/app.css` — reserves desktop height for board plus ribbon.
- Existing Go, Vitest, preview, fake, and Playwright files provide regression coverage.

---

### Task 1: Fast-forward successful zero-frame activities

**Files:**
- Modify: `frontend/src/components/openings/opening-state.ts:80-150`
- Modify: `frontend/src/components/openings/opening-state.test.ts:1-150`
- Modify: `frontend/src/components/openings/opening-controller.ts:126-445`
- Modify: `frontend/src/components/openings/opening-controller.test.ts:120-180`

**Interfaces:**
- Produces: `completeOpeningActivityWithoutPause(state, requestId, result): OpeningState`.
- Consumes: a requesting state and a completed result with an active returned session.
- Preserves: the existing completion/acknowledgement reducers for results that need inspection.

- [ ] **Step 1: Write failing state-transition tests**

Add to `opening-state.test.ts`:

```ts
test('completes an active zero-frame result without an acknowledgement phase', () => {
  const concept = active()
  const requesting = beginOpeningRequest(initialiseOpening(concept), 'advance', 7)
  const next = active({
    current: {
      ...fakeOpeningSession.current,
      kind: 'recap',
      activityId: 'recap',
      title: 'Keep the plan'
    }
  })
  expect(completeOpeningActivityWithoutPause(requesting, 7, {
    session: next,
    activityCompleted: true
  })).toMatchObject({ phase: 'passive', session: next, fen: next.current.currentFen })
})

test('refuses to fast-forward a checkpoint', () => {
  const requesting = beginOpeningRequest(initialiseOpening(active()), 'advance', 8)
  expect(() => completeOpeningActivityWithoutPause(requesting, 8, {
    session: completed(),
    activityCompleted: true,
    checkpoint
  })).toThrow('active next activity')
})
```

Use the existing active, completed, and checkpoint fixture shapes in that file.

- [ ] **Step 2: Run the state tests and verify RED**

Run:

```bash
npm --prefix frontend test -- --run src/components/openings/opening-state.test.ts
```

Expected: FAIL because the reducer is not exported.

- [ ] **Step 3: Implement the pure transition guard**

Add beside `completeOpeningActivity`:

```ts
export function completeOpeningActivityWithoutPause(
  state: OpeningState,
  requestId: number,
  result: Extract<OpeningActivityResult, { activityCompleted: true }>
): OpeningState {
  if (!acceptsOpeningResponse(state, requestId)) return state
  if (state.operation !== 'advance' || result.appliedMoves !== undefined ||
    result.checkpoint !== undefined || result.session.status !== 'active') {
    throw new Error('zero-frame completion requires an active next activity')
  }
  return initialiseOpening(result.session)
}
```

The strict decoder already forbids an empty `appliedMoves` wire array.

- [ ] **Step 4: Replace the old controller expectation with a failing fast-path test**

Replace the explicit-Continue concept test in `opening-controller.test.ts`:

```ts
test('advances a zero-frame result immediately and orders session events', async () => {
  const explain = active('concept')
  const watch = active('demonstration', {
    activityId: 'watch-c3', activityNumber: 2, title: 'Watch c3'
  })
  const order: string[] = []
  const subject = harness(explain, {
    advanceOpeningActivity: async () => ({ session: watch, activityCompleted: true })
  }, effects(false), board(), order)

  await subject.controller.advance()

  expect(subject.controller.view.state).toMatchObject({ phase: 'passive', session: watch })
  expect(subject.controller.view.message).toBe('')
  expect(subject.controller.view.announcement).toBe('Next idea: Watch c3.')
  expect(subject.persisted).toEqual([watch])
  expect(subject.changes).toEqual([watch])
  expect(order).toEqual(['persisted', 'change'])
  expect(subject.views.map((view) => view.state.phase)).not.toContain('activity-complete')
  expect(subject.views.map((view) => view.state.phase)).not.toContain('animating')
})
```

Extend `harness` with an optional event-order array. Add another test proving a
checkpoint result still reaches `activity-complete` and then `checkpoint`.

- [ ] **Step 5: Run the controller test and verify RED**

Run:

```bash
npm --prefix frontend test -- --run src/components/openings/opening-controller.test.ts
```

Expected: FAIL because the controller still publishes both intermediate phases.

- [ ] **Step 6: Branch before animation in `finishSuccessful`**

Import the new reducer and add this before `beginOpeningAnimation`:

```ts
if (frames.length === 0 && result.checkpoint === undefined &&
  result.session.status === 'active') {
  const next = completeOpeningActivityWithoutPause(this.requireState(), owned.id, result)
  this.runtime.cancelRequest()
  this.publishState(next, {
    message: '', feedback: null, notice: result.session.notice ?? '',
    announcement: `Next idea: ${result.session.current.title}.`,
    boardGeneration: this.view.boardGeneration + 1,
    lastFrames: [], lastMove: undefined
  })
  this.events.persisted(result.session)
  this.events.change(result.session)
  return
}
```

Leave the framed animation/completion path unchanged. Task 2 replaces the
temporary generation increment with exact-FEN logic.

- [ ] **Step 7: Run focused tests and verify GREEN**

```bash
npm --prefix frontend test -- --run \
  src/components/openings/opening-state.test.ts \
  src/components/openings/opening-controller.test.ts
```

Expected: PASS, including retry into the fast path and framed-result pauses.

- [ ] **Step 8: Commit the zero-frame transition**

```bash
git add frontend/src/components/openings/opening-state.ts \
  frontend/src/components/openings/opening-state.test.ts \
  frontend/src/components/openings/opening-controller.ts \
  frontend/src/components/openings/opening-controller.test.ts
git commit -m "feat: skip empty opening completion screens"
```

---

### Task 2: Preserve same-FEN boards and name remaining actions

**Files:**
- Modify: `frontend/src/components/openings/opening-controller.ts:250-345`
- Modify: `frontend/src/components/openings/opening-controller.test.ts:120-380`
- Modify: `frontend/src/components/openings/OpeningLessonScreen.svelte:195-255`
- Modify: `frontend/src/components/openings/OpeningLessonScreen.test.ts:80-190`

**Interfaces:**
- Produces: exact-FEN board-generation selection shared by fast-forward, acknowledgement, and session adoption.
- Produces: `passiveActionLabel(kind): string` in the lesson screen.
- Consumes: Task 1's fast path.

- [ ] **Step 1: Write failing same-FEN and different-FEN tests**

```ts
test('preserves the board instance when the next activity has the same FEN', async () => {
  const explain = active('concept')
  const recap = active('recap', {
    activityId: 'recap', currentFen: explain.current.currentFen
  })
  const subject = harness(explain, {
    advanceOpeningActivity: async () => ({ session: recap, activityCompleted: true })
  })
  const generation = subject.controller.view.boardGeneration
  await subject.controller.advance()
  expect(subject.controller.view.boardGeneration).toBe(generation)
})

test('replaces the board when the next activity has a different FEN', async () => {
  const subject = harness(active('concept'), {
    advanceOpeningActivity: async () => ({
      session: active('recap', { activityId: 'recap', currentFen: afterC3 }),
      activityCompleted: true
    })
  })
  const generation = subject.controller.view.boardGeneration
  await subject.controller.advance()
  expect(subject.controller.view.boardGeneration).toBe(generation + 1)
})
```

Add same/different-FEN coverage for `receiveSession`; retain the board-error
test proving recovery can still force remount.

- [ ] **Step 2: Run controller tests and verify RED**

```bash
npm --prefix frontend test -- --run src/components/openings/opening-controller.test.ts
```

Expected: same-FEN assertions FAIL.

- [ ] **Step 3: Centralize exact-FEN remount selection**

Add module helpers:

```ts
function visibleFen(state: OpeningState): string | undefined {
  return 'fen' in state ? state.fen : undefined
}

function targetSessionFen(session: OpeningSessionView): string | undefined {
  return session.status === 'active' ? session.current.currentFen : undefined
}
```

Add a private `generationForTargetFen(targetFen)` returning the current
generation for an identical full FEN and current generation plus one for a
different FEN. Use it in Task 1's branch, `acknowledgeActivity`, and
`adoptSession`. Calculate before publishing. Remove the duplicated consecutive
`runtime.cancelRequest()` from `acknowledgeActivity`.

- [ ] **Step 4: Write failing action-label tests**

```ts
test.each([
  ['concept', 'Got it'],
  ['recap', 'Got it'],
  ['comparison', 'Done comparing'],
  ['demonstration', 'Play the line']
] as const)('labels the %s action as %s', async (kind, label) => {
  const rendered = render(OpeningLessonScreen, {
    session: active(kind), effects: effects(),
    boardAdapterFactory: boardHarness().factory
  }, withNormalAPI(fakeAPI()))
  expect(await screen.findByRole('button', { name: label })).toBeInTheDocument()
  rendered.unmount()
})
```

Assert a framed decision/demonstration completion renders `Next idea`.

- [ ] **Step 5: Run screen tests and verify RED**

```bash
npm --prefix frontend test -- --run src/components/openings/OpeningLessonScreen.test.ts
```

Expected: FAIL because labels are still generic.

- [ ] **Step 6: Implement purpose-specific labels**

```ts
function passiveActionLabel(kind: OpeningActivityView['kind']): string {
  switch (kind) {
    case 'comparison': return 'Done comparing'
    case 'demonstration': return 'Play the line'
    default: return 'Got it'
  }
}
```

Render `Next idea` for `activity-complete` and this function for `passive`.
Do not add keydown handlers to native buttons. Update the main screen flow test
to click `Got it` once and immediately find the next activity.

- [ ] **Step 7: Run focused tests and verify GREEN**

```bash
npm --prefix frontend test -- --run \
  src/components/openings/opening-controller.test.ts \
  src/components/openings/OpeningLessonScreen.test.ts
```

Expected: PASS.

- [ ] **Step 8: Commit continuity and action copy**

```bash
git add frontend/src/components/openings/opening-controller.ts \
  frontend/src/components/openings/opening-controller.test.ts \
  frontend/src/components/openings/OpeningLessonScreen.svelte \
  frontend/src/components/openings/OpeningLessonScreen.test.ts
git commit -m "feat: preserve continuous opening boards"
```

---

### Task 3: Derive and decode the SAN journey contract

**Files:**
- Modify: `internal/openings/views.go:112-140`
- Modify: `internal/openings/service_activity.go:295-410`
- Modify: `internal/openings/service_activity_test.go:1-190`
- Regenerate: `frontend/wailsjs/go/models.ts`
- Modify: `frontend/src/lib/contracts/openings.ts:55-110,300-390`
- Modify: `frontend/src/lib/api.test.ts:110-165,300-325`
- Modify: `frontend/src/lib/api/preview.ts:125-175`
- Modify: `frontend/src/test-fakes.ts:95-135`
- Modify: `frontend/tests/test-backend.ts:550-650`

**Interfaces:**
- Produces Go JSON and TypeScript: `movesToHereSan: string[]`.
- Produces internal Go: `activityRoute{Frames, SAN, VariationName}`.
- Preserves: `movesToHere` frames from the identical BFS-selected move-ID route.

- [ ] **Step 1: Write failing service assertions for aligned SAN and passive variation**

Extend `TestOpeningServiceCompletesActivitiesAndReturnsCheckpoint`:

```go
if !reflect.DeepEqual(started.Current.MovesToHereSAN,
	[]string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Bc5"}) {
	t.Fatalf("moves to here SAN=%v", started.Current.MovesToHereSAN)
}
if len(started.Current.MovesToHere) != len(started.Current.MovesToHereSAN) {
	t.Fatalf("route frames=%d SAN=%d",
		len(started.Current.MovesToHere), len(started.Current.MovesToHereSAN))
}
if started.Current.VariationName == "" {
	t.Fatalf("passive variation name is empty: %+v", started.Current)
}
```

Add `TestOpeningServiceRootActivityRoute` using a root-position concept and
expect both route arrays to be empty without error.

- [ ] **Step 2: Run the Go tests and verify RED**

```bash
go test ./internal/openings \
  -run 'TestOpeningService(CompletesActivitiesAndReturnsCheckpoint|RootActivityRoute)' \
  -count=1
```

Expected: compile failure because `MovesToHereSAN` does not exist.

- [ ] **Step 3: Add one internal route result**

Add to `OpeningActivityView`:

```go
MovesToHereSAN []string `json:"movesToHereSan"`
```

Add in `service_activity.go`:

```go
type activityRoute struct {
	Frames        []domain.AppliedMove
	SAN           []string
	VariationName string
}

func (s *Service) activityRouteForMoveIDs(
	course CompiledCourse,
	moveIDs []string,
) (activityRoute, error) {
	frames, _, err := s.applyMoveIDs(course, course.Pack.RootFEN, moveIDs)
	if err != nil {
		return activityRoute{}, err
	}
	san := make([]string, 0, len(moveIDs))
	var variationName string
	for _, moveID := range moveIDs {
		move := course.Moves[moveID]
		san = append(san, move.SAN)
		if move.VariationName != "" {
			variationName = move.VariationName
		}
	}
	return activityRoute{Frames: frames, SAN: san, VariationName: variationName}, nil
}
```

Refactor `movesToPosition` into `routeToPosition` while retaining BFS order.
Return non-nil empty arrays at the root. In `buildActivityView`, assign frames,
SAN, and route variation. Use prompt-primary/activity-move variation only when
the route variation is empty.

- [ ] **Step 4: Format and verify the Go change**

```bash
gofmt -w internal/openings/views.go internal/openings/service_activity.go \
  internal/openings/service_activity_test.go
go test ./internal/openings -count=1
```

Expected: PASS.

- [ ] **Step 5: Regenerate Wails models**

```bash
GOWORK=off GOTOOLCHAIN=local \
  go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 generate module -tags bindings
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json \
  frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
```

Verify generated `OpeningActivityView` contains `movesToHereSan: string[]` and
inspect `git status --short` for unexpected generated changes.

- [ ] **Step 6: Write failing frontend decoder tests**

Add `movesToHereSan: []` to `openingStepPayload`, then add:

```ts
test('opening activity decoder keeps aligned route SAN', () => {
  const current = {
    ...openingStepPayload,
    movesToHere: [
      { uci: 'e2e4', resultingFen: 'after-e4' },
      { uci: 'e7e5', resultingFen: 'after-e5' }
    ],
    movesToHereSan: ['e4', 'e5']
  }
  expect(decodeOpeningSession({ ...activeOpeningPayload, current })).toMatchObject({
    current: { movesToHereSan: ['e4', 'e5'] }
  })
})

test('opening activity decoder rejects route SAN with a different length', () => {
  expect(() => decodeOpeningSession({
    ...activeOpeningPayload,
    current: { ...openingStepPayload, movesToHereSan: ['e4'] }
  })).toThrow('movesToHereSan must align with movesToHere')
})
```

- [ ] **Step 7: Run API tests and verify RED**

```bash
npm --prefix frontend test -- --run src/lib/api.test.ts
```

Expected: FAIL because the decoder lacks this required field.

- [ ] **Step 8: Add strict aligned decoding and migrate fixtures**

Add `movesToHereSan: string[]` to `OpeningActivityView`. Decode before return:

```ts
const movesToHere = decodeFramesAllowEmpty(raw.movesToHere, `${path}.movesToHere`)
const movesToHereSan = array(raw.movesToHereSan, `${path}.movesToHereSan`, string)
if (movesToHere.length !== movesToHereSan.length) {
  throw new Error(`${path}.movesToHereSan must align with movesToHere`)
}
```

Return both. Add `movesToHereSan` to `fakeOpeningSession`, preview activities,
and every explicit Playwright `WireOpeningActivity`.

- [ ] **Step 9: Run contract, type, and service checks**

```bash
npm --prefix frontend test -- --run src/lib/api.test.ts
npm --prefix frontend run check
go test ./internal/openings -count=1
git diff --check
```

Expected: PASS.

- [ ] **Step 10: Commit the SAN journey contract**

```bash
git add internal/openings/views.go internal/openings/service_activity.go \
  internal/openings/service_activity_test.go frontend/wailsjs \
  frontend/src/lib/contracts/openings.ts frontend/src/lib/api.test.ts \
  frontend/src/lib/api/preview.ts frontend/src/test-fakes.ts \
  frontend/tests/test-backend.ts
git commit -m "feat: expose opening SAN journeys"
```

---

### Task 4: Render the persistent SAN journey ribbon

**Files:**
- Create: `frontend/src/components/openings/OpeningMoveRibbon.svelte`
- Create: `frontend/src/components/openings/OpeningMoveRibbon.test.ts`
- Modify: `frontend/src/components/openings/OpeningLessonScreen.svelte:1-210`
- Modify: `frontend/src/components/openings/OpeningLessonScreen.test.ts:80-150`
- Modify: `frontend/src/styles/app.css:88-97,360-370`

**Interfaces:**
- Props: `variationName?: string`, `moves: readonly string[]`.
- Consumes: `current.variationName` and `current.movesToHereSan` from Task 3.
- Preserves: a labelled, display-only SAN list and existing board input.

- [ ] **Step 1: Write failing ribbon tests**

Create `OpeningMoveRibbon.test.ts`:

```ts
import { render, screen } from '@testing-library/svelte'
import OpeningMoveRibbon from './OpeningMoveRibbon.svelte'

test('renders variation, numbered SAN, and current ply', () => {
  render(OpeningMoveRibbon, {
    variationName: 'Italian Game',
    moves: ['e4', 'e5', 'Nf3', 'Nc6', 'Bc4']
  })
  expect(screen.getByText('Italian Game')).toBeVisible()
  expect(screen.getByRole('list', { name: 'Moves to this position' }))
    .toHaveTextContent('1. e4 e5 2. Nf3 Nc6 3. Bc4')
  expect(screen.getByText('Bc4').closest('li')).toHaveAttribute('aria-current', 'step')
})

test('labels an empty route as the starting position', () => {
  render(OpeningMoveRibbon, { moves: [] })
  expect(screen.getByText('Starting position')).toBeVisible()
  expect(screen.queryByRole('button')).not.toBeInTheDocument()
})
```

- [ ] **Step 2: Run ribbon tests and verify RED**

```bash
npm --prefix frontend test -- --run src/components/openings/OpeningMoveRibbon.test.ts
```

Expected: FAIL because the component does not exist.

- [ ] **Step 3: Implement the ribbon component**

Create this structure and add focused component CSS:

```svelte
<script lang="ts">
  import { afterUpdate } from 'svelte'
  export let variationName: string | undefined = undefined
  export let moves: readonly string[] = []
  let plyElements: HTMLLIElement[] = []
  afterUpdate(() => plyElements.at(-1)?.scrollIntoView?.({
    block: 'nearest', inline: 'end'
  }))
  const prefix = (index: number) => index % 2 === 0 ? `${Math.floor(index / 2) + 1}.` : ''
</script>

<section class="opening-move-ribbon" aria-label="Opening line">
  {#if variationName}<span class="variation-badge">{variationName}</span>{/if}
  {#if moves.length === 0}
    <span>Starting position</span>
  {:else}
    <ol aria-label="Moves to this position">
      {#each moves as move, index}
        <li bind:this={plyElements[index]}
          aria-current={index === moves.length - 1 ? 'step' : undefined}>
          {#if prefix(index)}<span class="move-number">{prefix(index)}</span>{/if}
          <span>{move}</span>
        </li>
      {/each}
    </ol>
  {/if}
</section>
```

Use a single horizontal row, `overflow-x: auto`, a 52px maximum height, a
distinct current ply, and existing charcoal/amber/ivory tokens.

- [ ] **Step 4: Compose the ribbon beneath the board**

Import the component and render after the keyed `ChessBoard`:

```svelte
<OpeningMoveRibbon
  variationName={current.variationName}
  moves={current.movesToHereSan}
/>
```

Add screen coverage proving the ribbon is outside `Opening lesson details` and
remains visible while the lesson panel scrolls.

- [ ] **Step 5: Reserve viewport height**

Split the shared puzzle/opening rule in `app.css`. Keep puzzle sizing unchanged
and use:

```css
.board-active .opening-lesson-layout .board-wrap {
  width: min(100%, calc(100vh - 184px), 760px);
  width: min(100%, calc(100dvh - 184px), 760px);
  aspect-ratio: 1;
}
```

At the narrow breakpoint restore `width: min(100%, 760px)`. Make
`.opening-board-stage` a centered grid with a 12px gap.

- [ ] **Step 6: Run ribbon, screen, static, and build checks**

```bash
npm --prefix frontend test -- --run \
  src/components/openings/OpeningMoveRibbon.test.ts \
  src/components/openings/OpeningLessonScreen.test.ts
npm --prefix frontend run check
npm --prefix frontend run build
```

Expected: PASS.

- [ ] **Step 7: Commit the ribbon**

```bash
git add frontend/src/components/openings/OpeningMoveRibbon.svelte \
  frontend/src/components/openings/OpeningMoveRibbon.test.ts \
  frontend/src/components/openings/OpeningLessonScreen.svelte \
  frontend/src/components/openings/OpeningLessonScreen.test.ts \
  frontend/src/styles/app.css
git commit -m "feat: show opening move journey"
```

---

### Task 5: Return authoritative frames for comparison lines

**Files:**
- Modify: `internal/openings/views.go:112-118`
- Modify: `internal/openings/service_activity.go:295-365`
- Modify: `internal/openings/service_activity_test.go:145-185`
- Regenerate: `frontend/wailsjs/go/models.ts`
- Modify: `frontend/src/lib/contracts/openings.ts:55-80,300-380`
- Modify: `frontend/src/lib/api.test.ts:110-165,300-335`
- Modify: `frontend/src/lib/api/preview.ts:125-175`
- Modify: `frontend/src/test-fakes.ts:95-135`
- Modify: `frontend/tests/test-backend.ts:550-650`

**Interfaces:**
- Produces Go: `OpeningActivityLine{Label, Moves, Frames}`.
- Produces TypeScript: `OpeningActivityLine` with non-empty `AppliedMoveFrames`.
- Consumes: the comparison activity position FEN and compiled move IDs.

- [ ] **Step 1: Replace the SAN-only Go test with a failing frame test**

```go
func TestActivityComparisonLinesReturnAlignedSANAndFrames(t *testing.T) {
	course, err := Compile(decodeTreePack(t), chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{rules: chessrules.Rules{}}
	lines, err := service.activityComparisonLines(course, course.Pack.RootFEN, []ActivityLine{{
		Label: "Develop the Italian", MoveIDs: []string{"white-e4", "black-e5", "white-nf3"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || !reflect.DeepEqual(lines[0].Moves, []string{"e4", "e5", "Nf3"}) ||
		len(lines[0].Frames) != 3 || lines[0].Frames[2].UCI != "g1f3" {
		t.Fatalf("comparison lines=%+v", lines)
	}
}
```

Add an incompatible-starting-FEN case and require the error to contain the
comparison label.

- [ ] **Step 2: Run comparison service tests and verify RED**

```bash
go test ./internal/openings -run '^TestActivityComparisonLines' -count=1
```

Expected: compile failure because `Frames` and the error-returning helper do not exist.

- [ ] **Step 3: Build frames from the activity position**

Add:

```go
Frames []domain.AppliedMove `json:"frames"`
```

Change the helper signature to:

```go
func (s *Service) activityComparisonLines(
	course CompiledCourse,
	startingFEN string,
	lines []ActivityLine,
) ([]OpeningActivityLine, error)
```

For every line, collect SAN, call
`s.applyMoveIDs(course, startingFEN, line.MoveIDs)`, wrap an error as
`fmt.Errorf("build comparison %q: %w", line.Label, err)`, and append label,
SAN, and frames. In `buildActivityView`, call with `position.FEN`, handle the
error, then assign the returned comparison.

- [ ] **Step 4: Format and verify Go**

```bash
gofmt -w internal/openings/views.go internal/openings/service_activity.go \
  internal/openings/service_activity_test.go
go test ./internal/openings -count=1
```

Expected: PASS.

- [ ] **Step 5: Regenerate Wails models**

```bash
GOWORK=off GOTOOLCHAIN=local \
  go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 generate module -tags bindings
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json \
  frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
```

Verify generated `OpeningActivityLine` converts `frames`.

- [ ] **Step 6: Write failing strict decoder tests**

```ts
test('opening decoder keeps authoritative comparison frames', () => {
  const comparison = [{
    label: 'Active centre', moves: ['c3', 'd4'],
    frames: [
      { uci: 'c2c3', resultingFen: 'after-c3' },
      { uci: 'd2d4', resultingFen: 'after-d4' }
    ]
  }]
  expect(decodeOpeningSession({
    ...activeOpeningPayload,
    current: { ...openingStepPayload, kind: 'comparison', comparison }
  })).toMatchObject({ current: { comparison } })
})

test('opening decoder rejects mismatched comparison frames', () => {
  expect(() => decodeOpeningSession({
    ...activeOpeningPayload,
    current: {
      ...openingStepPayload, kind: 'comparison', comparison: [{
        label: 'Mismatch', moves: ['c3', 'd4'],
        frames: [{ uci: 'c2c3', resultingFen: 'after-c3' }]
      }]
    }
  })).toThrow('frames must align with moves')
})
```

Add another case with `frames: []` expecting `authoritative move frames`.

- [ ] **Step 7: Implement strict decoding and migrate fixtures**

```ts
export type OpeningActivityLine = {
  label: string
  moves: string[]
  frames: AppliedMoveFrames
}
```

Add `decodeComparisonLine`, call `decodeFrames`, compare lengths, and throw
`${path}.frames must align with moves`. Add valid frames to every non-empty
comparison fixture; empty `comparison: []` values remain unchanged.

- [ ] **Step 8: Run contract, type, and service checks**

```bash
npm --prefix frontend test -- --run src/lib/api.test.ts
npm --prefix frontend run check
go test ./internal/openings -count=1
git diff --check
```

Expected: PASS.

- [ ] **Step 9: Commit the comparison contract**

```bash
git add internal/openings/views.go internal/openings/service_activity.go \
  internal/openings/service_activity_test.go frontend/wailsjs \
  frontend/src/lib/contracts/openings.ts frontend/src/lib/api.test.ts \
  frontend/src/lib/api/preview.ts frontend/src/test-fakes.ts \
  frontend/tests/test-backend.ts
git commit -m "feat: expose opening comparison frames"
```

---

### Task 6: Add cancellable comparison previews to the controller

**Files:**
- Modify: `frontend/src/components/openings/opening-controller.ts:35-500`
- Modify: `frontend/src/components/openings/opening-controller.test.ts:175-380`

**Interfaces:**
- Produces view fields: `comparisonPreviewIndex: number | null`, `comparisonPreviewPlaying: boolean`.
- Produces methods: `previewComparison(index: number): Promise<void>` and `resetComparisonPreview(): Promise<void>`.
- Consumes: Task 5 frames and `InteractiveBoardRuntime` request ownership.

- [ ] **Step 1: Write failing preview tests**

Add a fixture:

```ts
const comparisonLines: ActiveOpeningSessionView['current']['comparison'] = [{
  label: 'Active centre', moves: ['c3', 'd4'],
  frames: [
    { uci: 'c2c3', resultingFen: afterC3 },
    { uci: 'd7d6', resultingFen: afterD6 }
  ]
}]
```

Add the main lifecycle test:

```ts
test('previews a comparison without persisting and can reset', async () => {
  const current = active('comparison', { comparison: comparisonLines })
  const subject = harness(current)
  await subject.controller.previewComparison(0)

  expect(subject.board.setPosition.mock.calls).toEqual([
    [current.current.currentFen, undefined, false],
    [afterC3, ['c2', 'c3'], true],
    [afterD6, ['d7', 'd6'], true]
  ])
  expect(subject.controller.view.comparisonPreviewIndex).toBe(0)
  expect(subject.persisted).toEqual([])

  await subject.controller.resetComparisonPreview()
  expect(subject.board.setPosition).toHaveBeenLastCalledWith(
    current.current.currentFen, undefined, false
  )
  expect(subject.controller.view.comparisonPreviewIndex).toBeNull()
})
```

Add focused cases proving a second selection cancels a deferred first preview;
`advance` restores before its API call; pause/session adoption/destroy cancel;
reduced motion writes only the final comparison FEN; and a board exception
restores `currentFen`, leaves persistence empty, and shows a warning.

- [ ] **Step 2: Run controller tests and verify RED**

```bash
npm --prefix frontend test -- --run src/components/openings/opening-controller.test.ts
```

Expected: compile failure because preview fields and methods do not exist.

- [ ] **Step 3: Add preview state to the controller view**

```ts
comparisonPreviewIndex: number | null
comparisonPreviewPlaying: boolean
```

Initialize both in `mount`, preserve them in `publishState`, and clear them in
session adoption, successful completion, acknowledgement, pause, restart, and
destroy lifecycle paths.

Because `null` is the meaningful cleared value, use an own-property check when
publishing the selected index rather than nullish fallback:

```ts
comparisonPreviewIndex: Object.prototype.hasOwnProperty.call(
  overrides, 'comparisonPreviewIndex'
)
  ? overrides.comparisonPreviewIndex ?? null
  : current?.comparisonPreviewIndex ?? null,
comparisonPreviewPlaying: overrides.comparisonPreviewPlaying ??
  current?.comparisonPreviewPlaying ?? false,
```

- [ ] **Step 4: Implement authoritative preview playback**

Add:

```ts
async previewComparison(index: number): Promise<void> {
  const state = this.requireState()
  if (state.phase !== 'passive' || state.session.current.kind !== 'comparison') return
  const line = state.session.current.comparison[index]
  if (!line) return

  const token = this.runtime.startRequest()
  this.update({
    comparisonPreviewIndex: index,
    comparisonPreviewPlaying: true,
    message: `Playing ${line.label}…`,
    announcement: `Playing ${line.label}.`
  })
  const resetWarning = await this.runtime.reconcilePosition(
    state.session.current.currentFen, token.controller.signal, false
  )
  if (!this.runtime.isCurrent(token)) return
  if (resetWarning) {
    await this.afterRender()
    if (!this.runtime.isCurrent(token)) return
  }

  const animation = await animateAppliedMoves({
    port: this.runtime.animationPort(),
    startingFen: state.session.current.currentFen,
    appliedMoves: line.frames,
    finalFen: line.frames[line.frames.length - 1].resultingFen,
    reducedMotion: this.view.reducedMotion,
    signal: token.controller.signal,
    onStep: (kind) => this.runtime.playSound(kind)
  })
  if (!this.runtime.isCurrent(token) || animation.status === 'aborted') return
  let warning = animation.warning ?? resetWarning
  if (animation.warning) {
    await this.afterRender()
    if (!this.runtime.isCurrent(token)) return
    const restoreWarning = await this.runtime.reconcilePosition(
      state.session.current.currentFen, token.controller.signal, false
    )
    if (!this.runtime.isCurrent(token)) return
    warning = [animation.warning, restoreWarning].filter(Boolean).join(' ')
  }
  this.runtime.cancelRequest()
  this.update({
    comparisonPreviewIndex: index,
    comparisonPreviewPlaying: false,
    message: `${line.label} complete.`,
    announcement: `${line.label} complete.`,
    notice: warning
  })
  this.applyBoardWarnings()
}
```

Implement `resetComparisonPreview()` with a new token and
`runtime.reconcilePosition(current.currentFen, signal, false)`. On success,
cancel the token, clear preview fields and last-move highlighting, and announce
`Comparison position reset.`.

- [ ] **Step 5: Restore previews before backend actions**

At the start of `advance()` and `pause()`, await a private
`resetComparisonPreviewIfNeeded()` before `beginOwnedRequest`. It is a no-op
unless a preview is selected or playing. Session adoption, restart, and destroy
cancel the active token. Neither preview method emits session events.

- [ ] **Step 6: Run controller and static checks**

```bash
npm --prefix frontend test -- --run src/components/openings/opening-controller.test.ts
npm --prefix frontend run check
```

Expected: PASS; stale preview callbacks cannot alter an adopted session.

- [ ] **Step 7: Commit preview behavior**

```bash
git add frontend/src/components/openings/opening-controller.ts \
  frontend/src/components/openings/opening-controller.test.ts
git commit -m "feat: preview opening comparison lines"
```

---

### Task 7: Turn comparison lines into board controls

**Files:**
- Modify: `frontend/src/components/openings/OpeningActivityContent.svelte:1-130`
- Modify: `frontend/src/components/openings/OpeningActivityContent.test.ts:1-100`
- Modify: `frontend/src/components/openings/OpeningLessonScreen.svelte:45-235`
- Modify: `frontend/src/components/openings/OpeningLessonScreen.test.ts:80-250`

**Interfaces:**
- Content props: `comparisonPreviewIndex: number | null`, `comparisonPreviewPlaying: boolean`.
- Content events: `previewComparison: { index: number }`, `resetComparison: void`.
- The screen connects those events to Task 6 controller methods.

- [ ] **Step 1: Write failing comparison-control tests**

Update the fixture with frames and add:

```ts
test('dispatches playable comparisons and exposes reset', async () => {
  const { component } = render(OpeningActivityContent, {
    activity: comparisonActivity(),
    comparisonPreviewIndex: 0,
    comparisonPreviewPlaying: false
  })
  const previews: number[] = []
  const resets = vi.fn()
  component.$on('previewComparison', (event) => previews.push(event.detail.index))
  component.$on('resetComparison', resets)

  expect(screen.getByRole('button', {
    name: 'Play Active centre: c3, d4'
  })).toHaveAttribute('aria-pressed', 'true')
  await fireEvent.click(screen.getByRole('button', {
    name: 'Play Quiet setup: d3, Re1'
  }))
  expect(previews).toEqual([1])
  await fireEvent.click(screen.getByRole('button', { name: 'Reset position' }))
  expect(resets).toHaveBeenCalledOnce()
})
```

Also assert SAN remains visible and existing replay/deeper-analysis events are
unchanged.

- [ ] **Step 2: Run content tests and verify RED**

```bash
npm --prefix frontend test -- --run src/components/openings/OpeningActivityContent.test.ts
```

Expected: FAIL because comparisons are inert articles.

- [ ] **Step 3: Implement native comparison buttons**

Add props and dispatcher event types, then replace each article with:

```svelte
<button
  class="comparison-line"
  class:selected={comparisonPreviewIndex === index}
  type="button"
  aria-label={`Play ${line.label}: ${line.moves.join(', ')}`}
  aria-pressed={comparisonPreviewIndex === index}
  on:click={() => dispatch('previewComparison', { index })}
>
  <strong>{line.label}</strong>
  <span>{line.moves.join(' → ')}</span>
  <span aria-hidden="true">
    {comparisonPreviewIndex === index && comparisonPreviewPlaying ? 'Playing…' : '▶'}
  </span>
</button>
```

Render `Reset position` only when an index is selected. Retain 46px height,
visible focus, selected contrast, and the existing comparison grid.

- [ ] **Step 4: Connect the lesson screen**

Pass both view fields to `OpeningActivityContent` and add:

```svelte
on:previewComparison={(event) => controller.previewComparison(event.detail.index)}
on:resetComparison={() => controller.resetComparisonPreview()}
```

Add a screen test that clicks a comparison, observes adapter frames, resets,
and confirms no persisted event was dispatched.

- [ ] **Step 5: Run component, controller, and static checks**

```bash
npm --prefix frontend test -- --run \
  src/components/openings/OpeningActivityContent.test.ts \
  src/components/openings/OpeningLessonScreen.test.ts \
  src/components/openings/opening-controller.test.ts
npm --prefix frontend run check
```

Expected: PASS.

- [ ] **Step 6: Commit playable comparison UI**

```bash
git add frontend/src/components/openings/OpeningActivityContent.svelte \
  frontend/src/components/openings/OpeningActivityContent.test.ts \
  frontend/src/components/openings/OpeningLessonScreen.svelte \
  frontend/src/components/openings/OpeningLessonScreen.test.ts
git commit -m "feat: play opening comparisons on board"
```

---

### Task 8: Prove the complete lesson rhythm and repository gates

**Files:**
- Modify: `frontend/tests/test-backend.ts:550-720`
- Modify: `frontend/tests/openings.spec.ts:45-145`
- Normalize if regenerated: `frontend/wailsjs/go/models.ts`

**Interfaces:**
- Consumes: Tasks 1-7.
- Produces browser acceptance for `Got it → move → Next idea → comparison → checkpoint`.

- [ ] **Step 1: Make the synthetic recap an authored comparison**

Replace the third `giuoco-c3` activity with a comparison at `afterC3Fen`. Give
it route SAN for `e4 e5 Nf3 Nc6 Bc4 Bc5 c3` and:

```ts
const afterD6Fen = 'r1bqk1nr/ppp2ppp/2np4/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R w KQkq - 0 5'

comparison: [
  {
    label: 'Solid setup', moves: ['d6'],
    frames: [{ uci: 'd7d6', resultingFen: afterD6Fen }]
  },
  {
    label: 'Quick development', moves: ['Nf6'],
    frames: [{
      uci: 'g8f6',
      resultingFen: 'r1bqk2r/pppp1ppp/2n2n2/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R w KQkq - 1 5'
    }]
  }
]
```

Use title `Compare Black's setups`, instruction `Play both replies and notice
where Black develops.`, and variation `Giuoco Piano`.

- [ ] **Step 2: Rewrite the browser flow around meaningful actions**

In `openings.spec.ts`:

1. Click `Got it` once and expect `Choose the preparation` immediately.
2. Make `c2c3`, click `Next idea`, and expect `Compare Black's setups`.
3. Assert the ribbon contains `1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5 4. c3`.
4. Play `Solid setup: d6`, assert selection, then reset.
5. Play `Quick development: Nf6` and assert selection.
6. Click `Done comparing` once and expect the checkpoint immediately.
7. Retain pause/resume, review, explorer, and puzzle-isolation assertions.

Before clicking `Got it`, mark the current Chessground host and prove the same
DOM node survives the same-FEN transition:

```ts
const host = page.locator('.chessground-host')
await host.evaluate((element) => { element.setAttribute('data-continuity-probe', 'kept') })
await page.getByRole('button', { name: 'Got it' }).click()
await expect(page.locator('.chessground-host')).toHaveAttribute('data-continuity-probe', 'kept')
```

- [ ] **Step 3: Run the opening browser test**

```bash
npm --prefix frontend run test:e2e -- openings.spec.ts
```

Expected: PASS with one click for each zero-frame completion and explicit
pauses only after board-changing results.

- [ ] **Step 4: Run the full automated matrix**

```bash
go test ./... -count=1
npm --prefix frontend test -- --run
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run test:e2e
git diff --check
```

Expected: every command exits 0.

- [ ] **Step 5: Build and verify the macOS app**

```bash
GOWORK=off GOTOOLCHAIN=local \
  go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json \
  frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
git diff --check
codesign --verify --deep --strict --verbose=2 "build/bin/Chess Trainer.app"
```

Expected: the app is built and codesign verifies it on disk.

- [ ] **Step 6: Perform focused manual acceptance**

Open the built app with a synthetic course and verify: one-click concepts; no
same-FEN board flash; accurate in-viewport SAN; both comparison lines play,
switch, and reset; demonstrations and moves retain `Next idea`; checkpoints
remain; keyboard-only and reduced-motion flows remain usable.

- [ ] **Step 7: Commit browser acceptance and generated normalization**

```bash
git add frontend/tests/test-backend.ts frontend/tests/openings.spec.ts \
  frontend/wailsjs/go/models.ts
git commit -m "test: verify streamlined opening lessons"
```

- [ ] **Step 8: Confirm final repository state**

```bash
git status --short
git log -8 --oneline
```

Expected: clean worktree and the eight task commits visible in order.
