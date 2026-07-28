# Opening Position Context and Graph Feedback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep the current opening position's identity, plan, arrival choices, and continuations visible beside the board, and explain legal unexpected moves using authored graph data.

**Architecture:** Extend server-owned activity and result views with structured data derived from the compiled course graph. Keep route identity aligned with the SAN ribbon, decode the new fields strictly, and render them through focused Svelte components while preserving authoritative board and retry behavior.

**Tech Stack:** Go 1.26.4, Wails 2.12.0, Svelte 3, TypeScript 4.6, Vitest/Testing Library, Playwright, Chessground.

## Global Constraints

- Context describes the authoritative lesson/review position, never a comparison preview.
- **You arrived via** follows the SAN ribbon's canonical route; it does not claim historical learner actions at transpositions.
- `activityRoute.MoveIDs`, `Frames`, and `SAN` come from one route and have equal lengths.
- Branch rows contain only moves visible at the selected depth and preserve authored order.
- Plan copy comes only from the first position note of kind `plan`.
- Evaluations use chess notation plus learner-relative plain language.
- Expected and accepted-alternative grading stays unchanged; every other legal miss increments `IncorrectMoves` once and never completes the activity.
- Deeper moves are labelled **Covered later**; unknown moves are described as not covered.
- Branch chips are list items, not buttons.
- Add no database migration, session field, course-pack field, dependency, engine call, generated explanation, or Continue action.
- Preserve strict decoding, request ownership, stale-response protection, reduced motion, board recovery, and immediate retry.
- Regenerate Wails bindings with pinned Wails 2.12.0; never hand-edit generated models.

## File Structure

### New files

- `internal/openings/service_position_context.go` and test — derive identity and both branch rows.
- `internal/openings/service_move_feedback.go` and test — classify visible, deeper, and unknown moves.
- `frontend/src/components/openings/opening-evaluation.ts` and test — learner-relative evaluations.
- `OpeningPositionContext.svelte` and test — compact board-adjacent card.
- `opening-feedback-copy.ts` and test — deterministic feedback copy.
- `OpeningMoveFeedback.svelte` and test — structured live feedback.

### Existing files

- `internal/openings/views.go`, `service_activity.go`, `service_decision.go` and existing tests.
- `frontend/src/lib/contracts/openings.ts`, `frontend/src/lib/api.test.ts`, fakes, preview API, and Playwright backend.
- `frontend/wailsjs/go/models.ts`, regenerated.
- `opening-controller.ts`, `OpeningLessonScreen.svelte`, `frontend/src/styles/app.css` and tests.
- `frontend/tests/openings.spec.ts`.

---

### Task 1: Preserve canonical route identity

**Files:**
- Modify: `internal/openings/service_activity.go:389-450`
- Modify: `internal/openings/service_activity_test.go:1-55`

**Interfaces:**
- Produces `activityRoute{MoveIDs, Frames, SAN, VariationName}` with aligned arrays.
- Consumed by Task 2.

- [ ] **Step 1: Write failing tests**

```go
func TestActivityRouteKeepsMoveIDsFramesAndSANAligned(t *testing.T) {
	fixture := newOpeningServiceFixture(t)
	route, err := fixture.service.routeToPosition(
		fixture.compiled, "after-bc5", DepthReference)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"white-e4", "black-e5", "white-nf3",
		"black-nc6", "white-bc4", "black-bc5",
	}
	if !reflect.DeepEqual(route.MoveIDs, want) {
		t.Fatalf("IDs = %v, want %v", route.MoveIDs, want)
	}
	if len(route.MoveIDs) != len(route.Frames) ||
		len(route.MoveIDs) != len(route.SAN) {
		t.Fatalf("lengths = %d/%d/%d",
			len(route.MoveIDs), len(route.Frames), len(route.SAN))
	}
}

func TestActivityRouteReturnsNonNilEmptyRootSlices(t *testing.T) {
	fixture := newOpeningServiceFixture(t)
	route, err := fixture.service.routeToPosition(
		fixture.compiled, fixture.compiled.Pack.RootPositionID, DepthQuick)
	if err != nil {
		t.Fatal(err)
	}
	if route.MoveIDs == nil || route.Frames == nil || route.SAN == nil ||
		len(route.MoveIDs)+len(route.Frames)+len(route.SAN) != 0 {
		t.Fatalf("root route = %#v", route)
	}
}
```

- [ ] **Step 2: Verify RED**

```bash
go test ./internal/openings -run '^TestActivityRoute' -count=1
```

Expected: compile failure because `MoveIDs` is absent.

- [ ] **Step 3: Implement aligned IDs**

```go
type activityRoute struct {
	MoveIDs       []string
	Frames        []domain.AppliedMove
	SAN           []string
	VariationName string
}
```

Return `MoveIDs: append([]string{}, moveIDs...)` from `activityRouteForMoveIDs`. Return explicit empty `MoveIDs`, `Frames`, and `SAN` at the root. Keep BFS ordering unchanged.

- [ ] **Step 4: Format, verify, commit**

```bash
gofmt -w internal/openings/service_activity.go internal/openings/service_activity_test.go
go test ./internal/openings -run 'TestActivityRoute|TestOpeningServiceCompletesActivities' -count=1
git add internal/openings/service_activity.go internal/openings/service_activity_test.go
git commit -m "feat: align opening context routes"
```

---

### Task 2: Derive position context

**Files:**
- Create: `internal/openings/service_position_context.go`
- Create: `internal/openings/service_position_context_test.go`
- Modify: `internal/openings/views.go:112-142`
- Modify: `internal/openings/service_activity.go:287-347`
- Modify: `internal/openings/service_activity_test.go`

**Interfaces:**
- Produces `OpeningPositionContextView` on every active activity.
- Produces `buildPositionContext(course, positionID, depth, route)`.
- Produces `firstNoteTextOfKinds(course, noteIDs, kinds...)` for Task 3.

- [ ] **Step 1: Write failing derivation tests**

At Reference depth for `after-bc5`, mutate `italian-overview` to kind `plan` and assert:

```go
if context.Label != "Giuoco Piano" ||
	context.PlanText != note.Text ||
	context.Evaluation.Code != EvaluationEqual {
	t.Fatalf("context = %+v", context)
}
if got := contextMoveIDs(context.ArrivedVia); !reflect.DeepEqual(
	got, []string{"black-bc5", "black-nf6"}) {
	t.Fatalf("arrivals = %v", got)
}
if !context.ArrivedVia[0].Selected || context.ArrivedVia[1].Selected {
	t.Fatalf("selection = %+v", context.ArrivedVia)
}
if got := contextMoveIDs(context.NextMoves); !reflect.DeepEqual(
	got, []string{"white-c3", "white-b4"}) {
	t.Fatalf("next = %v", got)
}
```

Add Quick assertions for only `black-bc5`/`white-c3` and a root assertion for a non-nil empty arrival list.

Add a transposition test using the existing `installExplorerTransposition`
fixture. Reload the active compiled course, derive the route and context for
`trans-knights`, then assert the selected arrival is exactly the final route
move used by the SAN ribbon:

```go
selectedID := route.MoveIDs[len(route.MoveIDs)-1]
selectedCount := 0
for _, move := range positionContext.ArrivedVia {
	if move.Selected {
		selectedCount++
		if move.MoveID != selectedID {
			t.Fatalf("selected arrival = %q, route ends with %q", move.MoveID, selectedID)
		}
	}
}
if selectedCount != 1 {
	t.Fatalf("selected arrival count = %d", selectedCount)
}
```

- [ ] **Step 2: Verify RED**

```bash
go test ./internal/openings -run '^TestBuildPositionContext' -count=1
```

- [ ] **Step 3: Define public types**

```go
type OpeningContextMoveView struct {
	MoveID        string
	SAN           string
	Role          TrainingRole
	VariationName string
	Evaluation    Evaluation
	Selected      bool
}

type OpeningPositionContextView struct {
	PositionID string
	Label      string
	Evaluation Evaluation
	PlanText   string
	ArrivedVia []OpeningContextMoveView
	NextMoves  []OpeningContextMoveView
}
```

Give fields JSON names `moveId`, `san`, `role`, `variationName,omitempty`, `evaluation`, `selected`, `positionId`, `label`, `planText,omitempty`, `arrivedVia`, and `nextMoves`. Add required `PositionContext` with JSON name `positionContext` to `OpeningActivityView`.

- [ ] **Step 4: Implement the pure helper**

```go
func buildPositionContext(
	course CompiledCourse,
	positionID string,
	depth Depth,
	route activityRoute,
) (OpeningPositionContextView, error) {
	position, exists := course.Positions[positionID]
	if !exists {
		return OpeningPositionContextView{}, fmt.Errorf(
			"opening position %q is unavailable", positionID)
	}
	if len(route.MoveIDs) != len(route.Frames) ||
		len(route.MoveIDs) != len(route.SAN) {
		return OpeningPositionContextView{}, fmt.Errorf(
			"opening position %q has a misaligned route", positionID)
	}
	view := OpeningPositionContextView{
		PositionID: positionID,
		Label: position.Label,
		Evaluation: position.Evaluation,
		PlanText: firstNoteTextOfKinds(course, position.NoteIDs, "plan"),
		ArrivedVia: []OpeningContextMoveView{},
		NextMoves: []OpeningContextMoveView{},
	}
	if len(route.MoveIDs) > 0 {
		selectedID := route.MoveIDs[len(route.MoveIDs)-1]
		selected, ok := course.Moves[selectedID]
		if !ok || selected.ToPositionID != positionID {
			return OpeningPositionContextView{}, fmt.Errorf(
				"position %q has invalid route move %q", positionID, selectedID)
		}
		view.ArrivedVia = contextMoves(
			course.VisibleMoves(selected.FromPositionID, depth), selectedID)
	}
	view.NextMoves = contextMoves(course.VisibleMoves(positionID, depth), "")
	return view, nil
}
```

`contextMoves` maps all public move fields and marks only `selectedID`. `firstNoteTextOfKinds` scans requested kinds in priority order and note IDs in authored order.

- [ ] **Step 5: Attach context to activities**

In `buildActivityView` prefer `session.State.Position.PositionID` and fall back to `activity.PositionID` only when empty. Derive context after the route, wrap errors with activity ID, and assign it in the view. Add StartLesson and StartReview integration assertions that context position equals current position.

- [ ] **Step 6: Format, verify, commit**

```bash
gofmt -w internal/openings/views.go internal/openings/service_activity.go \
  internal/openings/service_activity_test.go \
  internal/openings/service_position_context.go \
  internal/openings/service_position_context_test.go
go test ./internal/openings -count=1
git add internal/openings/views.go internal/openings/service_activity.go \
  internal/openings/service_activity_test.go \
  internal/openings/service_position_context.go \
  internal/openings/service_position_context_test.go
git commit -m "feat: expose opening position context"
```

---

### Task 3: Add graph-aware feedback

**Files:**
- Create: `internal/openings/service_move_feedback.go`
- Create: `internal/openings/service_move_feedback_test.go`
- Modify: `internal/openings/views.go`
- Modify: `internal/openings/service_decision.go:12-56`
- Modify: `internal/openings/service_test.go`
- Modify: `internal/openings/service_response_test.go`

**Interfaces:**
- Produces `OpeningMoveFeedbackDetail` and optional `FeedbackDetail`.
- Preserves `MoveFeedback` as grading discriminator.

- [ ] **Step 1: Write failing service tests**

Cover these exact cases:

- Reference `b2b4`: alternative, visible, SAN `b4`, variation `Evans idea`, expected `c3`.
- A custom visible `d2d3` outgoing move from `after-bc5` that is not an
  accepted alternative: off-course, visible, and one incorrect move. Build the
  fixture by appending `after-giuoco-d3` to `pack.Positions`, a
  `white-d3-giuoco` repertoire move to `pack.Moves`, its coverage ID to
  `ExpectedReferences`, compiling, and replacing the fixture catalogue before
  starting the lesson.
- Quick `b2b4`: off-course, deeper, required depth Reference.
- `d2d3`: off-course, unknown, computed SAN `d3`, evaluation None, empty variation/note.
- Existing counters: visible accepted alternative increments only `AlternativesTried`; deeper/unknown increment `IncorrectMoves` once.

Add a pure helper case whose matched move references `warning`, `plan`, and
`explanation` notes; assert explanation wins regardless of note-ID order. This
locks the approved note-kind priority independently of author ordering.

- [ ] **Step 2: Verify RED**

```bash
go test ./internal/openings -run '^TestPlayMove(Explains|Marks)' -count=1
```

- [ ] **Step 3: Define feedback types**

```go
type OpeningFeedbackCoverage string

const (
	FeedbackCoverageVisible OpeningFeedbackCoverage = "visible"
	FeedbackCoverageDeeper  OpeningFeedbackCoverage = "deeper"
	FeedbackCoverageUnknown OpeningFeedbackCoverage = "unknown"
)

type OpeningMoveFeedbackDetail struct {
	Coverage      OpeningFeedbackCoverage
	PlayedSAN     string
	VariationName string
	Evaluation    Evaluation
	NoteText      string
	ExpectedSAN   string
	PlanText      string
	RequiredDepth Depth
}
```

Give fields the approved camel-case JSON names and `omitempty` on variation, note, plan, and required depth. Add pointer `FeedbackDetail` with JSON name `feedbackDetail,omitempty` to `OpeningActivityResult`.

- [ ] **Step 4: Implement deterministic lookup**

Initialize unknown detail with EvaluationNone, expected SAN, and current-position plan. Compute unknown SAN with `RulesPort.SAN`. Scan current-position outgoing moves in authored order for matching UCI; on a match copy SAN, variation, evaluation, and first note of kinds `explanation`, `plan`, `warning`, `evaluation`. Use `visibleAtDepth` to choose visible/deeper and set required depth only for deeper.

- [ ] **Step 5: Attach detail without grading changes**

Call lookup after the expected branch. Attach it in the accepted-alternative branch, then attach it to the remaining miss branch after incrementing `IncorrectMoves` exactly once. Use existing message wording for unknown, `known course move` for visible, and `covered at <depth> depth` for deeper.

- [ ] **Step 6: Add response regression**

Marshal a completed result without detail and assert `feedbackDetail` is omitted. Retain illegal-move no-write, completion, review, and alternative counter tests.

- [ ] **Step 7: Format, verify, commit**

```bash
gofmt -w internal/openings/views.go internal/openings/service_decision.go \
  internal/openings/service_move_feedback.go \
  internal/openings/service_move_feedback_test.go \
  internal/openings/service_test.go internal/openings/service_response_test.go
go test ./internal/openings -count=1
git add internal/openings/views.go internal/openings/service_decision.go \
  internal/openings/service_move_feedback.go \
  internal/openings/service_move_feedback_test.go \
  internal/openings/service_test.go internal/openings/service_response_test.go
git commit -m "feat: explain authored opening moves"
```

---

### Task 4: Strictly decode the contract

**Files:**
- Modify: `frontend/src/lib/contracts/openings.ts`
- Modify: `frontend/src/lib/api.test.ts`
- Modify: `frontend/src/test-fakes.ts`
- Modify: `frontend/src/lib/api/preview.ts`
- Modify: `frontend/tests/test-backend.ts`
- Regenerate: `frontend/wailsjs/go/models.ts`

**Interfaces:**
- Produces `OpeningContextMove`, `OpeningPositionContext`, `OpeningFeedbackCoverage`, `OpeningMoveFeedbackDetail`.
- Adds required `positionContext` and optional incomplete-result `feedbackDetail`.

- [ ] **Step 1: Write failing decoder tests**

Add selected Bc5 arrival/c3 continuation context to `openingStepPayload`. Reject missing context, bad role, non-boolean selected, bad evaluation, and non-array branches.

Add a valid detail:

```ts
feedbackDetail: {
  coverage: 'deeper',
  playedSan: 'b4',
  variationName: 'Evans idea',
  evaluation: { code: 'unclear' },
  expectedSan: 'c3',
  planText: 'Prepare d4.',
  requiredDepth: 'reference'
}
```

Reject unknown coverage, missing evaluation/expected SAN, deeper without depth, non-deeper with depth, and detail on completed results.

- [ ] **Step 2: Verify RED**

```bash
npm --prefix frontend test -- --run src/lib/api.test.ts
```

- [ ] **Step 3: Define and decode types**

```ts
export type OpeningContextMove = {
  moveId: string
  san: string
  role: OpeningTrainingRole
  variationName?: string
  evaluation: OpeningEvaluation
  selected: boolean
}
export type OpeningPositionContext = {
  positionId: string
  label: string
  evaluation: OpeningEvaluation
  planText?: string
  arrivedVia: OpeningContextMove[]
  nextMoves: OpeningContextMove[]
}
export type OpeningFeedbackCoverage = 'visible' | 'deeper' | 'unknown'
export type OpeningMoveFeedbackDetail = {
  coverage: OpeningFeedbackCoverage
  playedSan: string
  variationName?: string
  evaluation: OpeningEvaluation
  noteText?: string
  expectedSan: string
  planText?: string
  requiredDepth?: OpeningDepth
}
```

Use existing decoder primitives. Require context on every active activity. Decode detail only for alternative/off-course results. Enforce the required-depth invariant.

- [ ] **Step 4: Update active fixtures**

Export `fakeOpeningPositionContext` and attach it to the main fake. Add equivalent context to every preview and Playwright backend activity, with empty arrays rather than undefined. Preview b4 returns visible Evans detail; generic misses return unknown detail.

- [ ] **Step 5: Regenerate bindings**

```bash
GOWORK=off GOTOOLCHAIN=local \
  go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 generate module -tags bindings
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json \
  frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
git status --short
```

Only expected generated/type files may change.

- [ ] **Step 6: Verify and commit**

```bash
npm --prefix frontend test -- --run \
  src/lib/api.test.ts src/lib/contracts/opening-journey.test.ts
npm --prefix frontend run check
git add frontend/src/lib/contracts/openings.ts frontend/src/lib/api.test.ts \
  frontend/src/test-fakes.ts frontend/src/lib/api/preview.ts \
  frontend/tests/test-backend.ts frontend/wailsjs/go/models.ts
git commit -m "feat: decode opening context contracts"
```

---

### Task 5: Format evaluations

**Files:**
- Create: `frontend/src/components/openings/opening-evaluation.ts`
- Create: `frontend/src/components/openings/opening-evaluation.test.ts`

**Interfaces:**
- Produces `formatOpeningEvaluation(evaluation, perspective): { symbol, label, text } | null`.

- [ ] **Step 1: Write the full table test**

Cover every code from both perspectives. Required checks:

```ts
expect(formatOpeningEvaluation({ code: 'white_slight' }, 'white'))
  .toEqual({ symbol: '⩲', label: 'Slightly better for you',
    text: '⩲ Slightly better for you' })
expect(formatOpeningEvaluation({ code: 'white_slight' }, 'black'))
  .toEqual({ symbol: '⩲', label: 'Slightly better for White',
    text: '⩲ Slightly better for White' })
expect(formatOpeningEvaluation({ code: 'none' }, 'white')).toBeNull()
expect(formatOpeningEvaluation(
  { code: 'white_slight', sourceSymbol: '+=' }, 'white'
)).toMatchObject({ symbol: '+=', label: 'Slightly better for you' })
```

Use the exact approved evaluation table for all remaining rows.

- [ ] **Step 2: Verify RED**

```bash
npm --prefix frontend test -- --run \
  src/components/openings/opening-evaluation.test.ts
```

- [ ] **Step 3: Implement exhaustive switch**

Return null for None, use non-empty source symbol as visible symbol, and finish with a `never` guard.

- [ ] **Step 4: Verify and commit**

```bash
npm --prefix frontend test -- --run \
  src/components/openings/opening-evaluation.test.ts
git add frontend/src/components/openings/opening-evaluation.ts \
  frontend/src/components/openings/opening-evaluation.test.ts
git commit -m "feat: format opening evaluations"
```

---

### Task 6: Render the compact context card

**Files:**
- Create: `frontend/src/components/openings/OpeningPositionContext.svelte`
- Create: `frontend/src/components/openings/OpeningPositionContext.test.ts`
- Modify: `frontend/src/components/openings/OpeningLessonScreen.svelte`
- Modify: `frontend/src/components/openings/OpeningLessonScreen.test.ts`
- Modify: `frontend/src/styles/app.css:95-99`

**Interfaces:**
- Props `context` and `orientation`.
- Produces labelled region `Position: <name>` with no controls.

- [ ] **Step 1: Write failing component tests**

Assert heading, evaluation, plan, labelled arrival/next lists, Bc5 Played marker, c3 continuation, and zero buttons. For empty content assert Current position, Starting position, End of this course line, and no generic plan.

- [ ] **Step 2: Verify RED**

```bash
npm --prefix frontend test -- --run \
  src/components/openings/OpeningPositionContext.test.ts
```

- [ ] **Step 3: Implement semantic component**

Use a labelled `section`, `h3` name, optional evaluation/plan, `h4` branch labels, and keyed `ul` lists. Each item shows SAN, optional variation, and explicit Played text. Style side-by-side desktop groups, stacked narrow groups, internally scrollable rows, visible selected border, and visually clamped plan with full DOM text.

- [ ] **Step 4: Compose and preserve preview context**

Render immediately after `OpeningMoveRibbon`:

```svelte
<OpeningPositionContext
  context={current.positionContext}
  orientation={current.orientation}
/>
```

Do not key it separately or bind it to preview state. Extend comparison tests to assert unchanged context through preview/reset.

- [ ] **Step 5: Reserve bounded height**

```css
.board-active .opening-lesson-layout .board-wrap {
  width: min(100%, calc(100vh - 320px), 760px);
  width: min(100%, calc(100dvh - 320px), 760px);
  aspect-ratio: 1;
}
```

- [ ] **Step 6: Verify and commit**

```bash
npm --prefix frontend test -- --run \
  src/components/openings/OpeningPositionContext.test.ts \
  src/components/openings/OpeningLessonScreen.test.ts
npm --prefix frontend run check
git add frontend/src/components/openings/OpeningPositionContext.svelte \
  frontend/src/components/openings/OpeningPositionContext.test.ts \
  frontend/src/components/openings/OpeningLessonScreen.svelte \
  frontend/src/components/openings/OpeningLessonScreen.test.ts \
  frontend/src/styles/app.css
git commit -m "feat: show opening position context"
```

---

### Task 7: Render structured feedback

**Files:**
- Create: `frontend/src/components/openings/opening-feedback-copy.ts`
- Create: `frontend/src/components/openings/opening-feedback-copy.test.ts`
- Create: `frontend/src/components/openings/OpeningMoveFeedback.svelte`
- Create: `frontend/src/components/openings/OpeningMoveFeedback.test.ts`
- Modify: `frontend/src/components/openings/opening-controller.ts` and test
- Modify: `frontend/src/components/openings/OpeningLessonScreen.svelte` and test

**Interfaces:**
- Produces `openingFeedbackCopy` returning heading, explanation, contrast, announcement.
- Adds `OpeningControllerView.feedbackDetail`.

- [ ] **Step 1: Write failing copy tests**

Require visible alternative copy with heading `b4 — Evans idea`, evaluation/note sentence, and c3/plan contrast. Require deeper copy to contain `Covered later at Reference depth.`. Require unknown copy `This move is not covered from this position.` with no invented metadata.

- [ ] **Step 2: Implement deterministic copy**

Use Task 5 formatter and fixed Quick/Standard/Reference labels. Join only non-empty sentences; never infer move quality.

- [ ] **Step 3: Test and implement component**

Test structured content, polite atomic live region, fallback message, and zero buttons. Props are feedback, detail, orientation, fallbackMessage. Render heading/explanation/contrast when detail exists, otherwise fallback.

- [ ] **Step 4: Retain controller detail**

Initialize `feedbackDetail: null`. In `finishFeedback` store result detail and use full copy for announcement. Clear stale detail whenever feedback is explicitly reset:

```ts
feedbackDetail: overrides.feedbackDetail === undefined
  ? overrides.feedback === null ? null : current?.feedbackDetail ?? null
  : overrides.feedbackDetail,
```

Test detail retention, full announcement, authoritative FEN reconciliation, and clearing on the next request.

- [ ] **Step 5: Compose feedback**

Replace the inline message paragraph with:

```svelte
<OpeningMoveFeedback
  feedback={view.feedback}
  detail={view.feedbackDetail}
  orientation={current.orientation}
  fallbackMessage={view.message}
/>
```

Keep notice and non-duplicate hidden announcements adjacent.

- [ ] **Step 6: Verify and commit**

```bash
npm --prefix frontend test -- --run \
  src/components/openings/opening-feedback-copy.test.ts \
  src/components/openings/OpeningMoveFeedback.test.ts \
  src/components/openings/opening-controller.test.ts \
  src/components/openings/OpeningLessonScreen.test.ts
npm --prefix frontend run check
git add frontend/src/components/openings/opening-feedback-copy.ts \
  frontend/src/components/openings/opening-feedback-copy.test.ts \
  frontend/src/components/openings/OpeningMoveFeedback.svelte \
  frontend/src/components/openings/OpeningMoveFeedback.test.ts \
  frontend/src/components/openings/opening-controller.ts \
  frontend/src/components/openings/opening-controller.test.ts \
  frontend/src/components/openings/OpeningLessonScreen.svelte \
  frontend/src/components/openings/OpeningLessonScreen.test.ts
git commit -m "feat: render graph-aware opening feedback"
```

---

### Task 8: End-to-end verification and app rebuild

**Files:**
- Modify: `frontend/tests/test-backend.ts`
- Modify: `frontend/tests/openings.spec.ts`
- Normalize: `frontend/wailsjs/go/models.ts`
- Generate ignored: `build/bin/Chess Trainer.app`

**Interfaces:**
- Produces desktop/narrow evidence and verified local app bundle.

- [ ] **Step 1: Complete browser backend**

At Reference return Bc5/Nf6 arrivals and c3/b4 continuations; at Quick omit Nf6/b4. Return visible authored detail for b4 and graph-known off-course detail for d3 while preserving the active session.

- [ ] **Step 2: Extend desktop flow**

Assert context name/evaluation/plan, both labelled lists, Played marker, and no buttons. Play b4 and d3, assert structured feedback and unchanged Giuoco board, then immediately play c3. At 1280x720 assert board and context bottoms are inside the viewport.

- [ ] **Step 3: Add narrow acceptance**

At 390x844 assert both lists are visible, body/card have no horizontal overflow, and branch lists handle their own scrolling.

- [ ] **Step 4: Run focused browser test**

```bash
npm --prefix frontend run test:e2e -- openings.spec.ts
```

- [ ] **Step 5: Run full matrix**

```bash
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go mod verify
node frontend/scripts/generate-sounds.mjs --check
node scripts/generate-third-party-notices.mjs --check
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
npm --prefix frontend run test:e2e
node --test scripts/*.test.mjs
bash -n scripts/build-release.sh
git diff --check
```

- [ ] **Step 6: Build and verify macOS app**

```bash
GOWORK=off GOTOOLCHAIN=local \
  go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json \
  frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
git diff --check
codesign --verify --deep --strict --verbose=2 "build/bin/Chess Trainer.app"
file "build/bin/Chess Trainer.app/Contents/MacOS/Chess Trainer"
```

- [ ] **Step 7: Manual acceptance**

Open the rebuilt app with synthetic white and black courses. Verify context during concept, decision, comparison preview/reset, and review; depth filtering; learner-relative evaluations; immediate retry after visible/deeper/unknown misses; keyboard order without focusable chips; full desktop board visibility; and narrow scrolling without page overflow.

- [ ] **Step 8: Commit and confirm state**

```bash
git add frontend/tests/test-backend.ts frontend/tests/openings.spec.ts \
  frontend/wailsjs/go/models.ts
git commit -m "test: verify opening context experience"
git status --short
git log -10 --oneline
```

Expected: clean worktree, eight implementation commits in order, and rebuilt ignored bundle at `build/bin/Chess Trainer.app`.
