# Seamless Opening Course Flow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Make opening study continue atomically and invisibly across teaching-tree nodes, resume the exact next activity, and show a concise move-labelled course map in the lesson sidebar.

**Architecture:** Keep node-level opening sessions as the persistence unit, but complete the old session and create the recommended next active session in the same SQLite transaction. Return that active session in the original activity result so the existing lesson screen adopts it without navigation or a second request. Add a server-derived local course-map projection to active lesson sessions, then make the full tree an optional navigation surface rather than a completion checkpoint.

**Tech Stack:** Go 1.26.4, SQLite, Wails v2.12.0, TypeScript, Svelte 3, Vitest, Testing Library, Playwright.

## Global Constraints

- Teaching nodes remain authoring and progress units; they are not learner-facing screen boundaries.
- Node completion produces no checkpoint, celebration, node-completion copy, full-tree navigation, or additional Continue action.
- The existing **Next idea** acknowledgement remains after interactive feedback and behaves identically across same-node and cross-node transitions.
- Passive zero-frame activities gain no pause at a node boundary.
- Home offers direct exact resume and a separate secondary course-map route.
- Opening or closing a course map does not alter the active session.
- **Start here** replaces the single resumable journey only after its backend transaction succeeds; completed activity progress remains saved.
- Sidebar entries show concise numbered SAN where available, a plain-language lesson name, and one of Learned, Current, Next, or Available using text plus a symbol.
- Map state never depends on color alone. Child-facing copy does not use node, cursor, projection, or session.
- Review completion, authoring formats, depth filtering, variation exploration, move feedback, and puzzle training remain unchanged.
- Do not add a database migration or a second cross-node frontend request.

---

### Task 1: Add the typed local course-map projection

**Files:**
- Modify: internal/openings/views.go
- Modify: internal/openings/service_tree.go
- Modify: internal/openings/service_activity.go
- Modify: internal/openings/service_tree_test.go
- Modify: internal/openings/service_response_test.go

**Interfaces:**
- Produces: OpeningCourseMapState, OpeningCourseMapEntryView, and OpeningCourseMapView.
- Produces: (*Service).localCourseMap(ctx, course, depth, currentLessonID) (OpeningCourseMapView, error).
- Produces: OpeningSessionView.CourseMap *OpeningCourseMapView, present only for active lesson sessions.
- Consumes: projectTeachingTree, teachingPathLessonIDs, LessonParent, LessonChildren, and routeToPosition.

- [ ] **Step 1: Write the failing service test**

Add this test with the existing tree fixture:

~~~go
func TestActiveOpeningLessonIncludesConciseLocalCourseMap(t *testing.T) {
	fixture := newTreeServiceFixture(t)
	started, err := fixture.service.StartLesson(
		context.Background(), fixture.compiled.Pack.CourseID, "giuoco-plan",
	)
	if err != nil {
		t.Fatal(err)
	}
	if started.CourseMap == nil {
		t.Fatal("active lesson has no local course map")
	}
	want := []OpeningCourseMapEntryView{
		{
			LessonID: "giuoco-plan", MoveLabel: "3...Bc5",
			Title: "Prepare the center", State: CourseMapCurrent, Selectable: true,
		},
		{
			LessonID: "two-knights-plan", ParentLessonID: "giuoco-plan",
			MoveLabel: "3...Nf6", Title: "Choose the quiet setup",
			State: CourseMapNext, Selectable: true,
		},
	}
	if !reflect.DeepEqual(started.CourseMap.Entries, want) {
		t.Fatalf("course map = %#v, want %#v", started.CourseMap.Entries, want)
	}
	if started.CourseMap.CompletedIdeas != 0 || started.CourseMap.TotalIdeas != 4 {
		t.Fatalf("course ideas = %d/%d", started.CourseMap.CompletedIdeas, started.CourseMap.TotalIdeas)
	}
}
~~~

Extend TestOpeningSessionViewJSONUsesStrictDiscriminatingFields: an active lesson encodes courseMap, while an active review and completed lesson omit it.

- [ ] **Step 2: Run the focused tests and verify RED**

~~~bash
go test ./internal/openings -run 'TestActiveOpeningLessonIncludesConciseLocalCourseMap|TestOpeningSessionViewJSONUsesStrictDiscriminatingFields' -count=1
~~~

Expected: compilation fails because the course-map types and CourseMap field do not exist.

- [ ] **Step 3: Add the Go contract**

Add beside the teaching-tree view types:

~~~go
type OpeningCourseMapState string

const (
	CourseMapLearned   OpeningCourseMapState = "learned"
	CourseMapCurrent   OpeningCourseMapState = "current"
	CourseMapNext      OpeningCourseMapState = "next"
	CourseMapAvailable OpeningCourseMapState = "available"
)

type OpeningCourseMapEntryView struct {
	LessonID       string                `json:"lessonId"`
	ParentLessonID string                `json:"parentLessonId,omitempty"`
	MoveLabel      string                `json:"moveLabel,omitempty"`
	Title          string                `json:"title"`
	State          OpeningCourseMapState `json:"state"`
	Selectable     bool                  `json:"selectable"`
}

type OpeningCourseMapView struct {
	CompletedIdeas int                         `json:"completedIdeas"`
	TotalIdeas     int                         `json:"totalIdeas"`
	Entries []OpeningCourseMapEntryView `json:"entries"`
}
~~~

Add this field to OpeningSessionView:

~~~go
CourseMap *OpeningCourseMapView `json:"courseMap,omitempty"`
~~~

- [ ] **Step 4: Implement deterministic local selection**

The helper performs these exact operations:

1. Project progress at the session depth.
2. Copy the progress map, mark the current lesson complete in the copy, and use the existing recommendation order to calculate the next lesson.
3. Include the current teaching path, visible children of the current lesson, visible siblings of the current lesson, and the calculated next lesson.
4. Remove duplicates and hidden nodes.
5. Emit entries in teachingTreeLessonOrder.
6. Assign one state with precedence: current, next, learned, available.
7. Sum course-wide completed and required ideas across visible lessons. Count
   every required idea in a sticky-completed lesson as learned; otherwise use
   its persisted completed-activity count.

Derive move labels from the last SAN on each lesson's start-position route:

~~~go
func numberedLastSAN(values []string) string {
	if len(values) == 0 {
		return ""
	}
	ply := len(values) - 1
	separator := "."
	if ply%2 == 1 {
		separator = "..."
	}
	return fmt.Sprintf("%d%s%s", ply/2+1, separator, values[ply])
}
~~~

Use s.routeToPosition(course, lesson.StartPositionID, depth). Do not substitute prose edge labels for numbered SAN.

Add a branching projection test that gives the current lesson both a visible
alternative and a hidden deeper sibling. Assert the visible sibling appears as
Available, the hidden sibling is omitted, every child carries its parent lesson
ID, and CompletedIdeas/TotalIdeas cover the entire visible course rather than
only the current lesson.

- [ ] **Step 5: Attach the projection only to active lessons**

In sessionView, after building an active lesson activity:

~~~go
courseMap, err := s.localCourseMap(ctx, course, session.Depth, session.LessonID)
if err != nil {
	return OpeningSessionView{}, err
}
view.CourseMap = &courseMap
~~~

Active reviews and completed views leave CourseMap nil.

- [ ] **Step 6: Run focused and package tests**

~~~bash
gofmt -w internal/openings/views.go internal/openings/service_tree.go internal/openings/service_activity.go internal/openings/service_tree_test.go internal/openings/service_response_test.go
go test ./internal/openings -run 'TestActiveOpeningLessonIncludesConciseLocalCourseMap|TestOpeningSessionViewJSONUsesStrictDiscriminatingFields|TestOpeningHomeReturnsTeachingTreeAndRecommendation' -count=1
go test ./internal/openings -count=1
~~~

Expected: all pass; review and completed JSON contain no courseMap.

- [ ] **Step 7: Commit**

~~~bash
git add internal/openings/views.go internal/openings/service_tree.go internal/openings/service_activity.go internal/openings/service_tree_test.go internal/openings/service_response_test.go
git commit -m "feat: expose opening lesson course map"
~~~

---

### Task 2: Make node-session rollover atomic in SQLite

**Files:**
- Modify: internal/openings/user_store.go
- Modify: internal/openings/user_store_activity.go
- Modify: internal/openings/user_store_test.go
- Modify: internal/openings/service_activity.go only to accept the new store return value

**Interfaces:**
- Changes: LessonActivityCompletion.NextSession *SessionSeed.
- Changes: (*UserStore).CompleteLessonActivity(context.Context, LessonActivityCompletion) (StoredSession, error).
- Contract: without NextSession, return the updated current session; with NextSession, complete the current session, insert the next active session, save progress and journey, and return the next session in one transaction.

- [ ] **Step 1: Write the failing atomic-rollover test**

Create a current lesson session through CreateLessonSession. Submit a LessonActivityCompletion whose current session is completed, journey points to two-knights-plan, and NextSession seeds that lesson. Assert:

~~~go
persisted, err := store.CompleteLessonActivity(ctx, completion)
if err != nil {
	t.Fatal(err)
}
if persisted.ID == current.ID || persisted.Status != OpeningStatusActive ||
	persisted.LessonID != "two-knights-plan" {
	t.Fatalf("persisted next session = %+v", persisted)
}
retired, err := store.LoadSession(ctx, current.ID)
if err != nil || retired.Status != OpeningStatusCompleted {
	t.Fatalf("retired session = %+v, err=%v", retired, err)
}
resumable, err := store.ResumableSession(ctx)
if err != nil || resumable == nil || resumable.ID != persisted.ID {
	t.Fatalf("resumable = %+v, err=%v", resumable, err)
}
~~~

Add a rollback test with this trigger:

~~~sql
CREATE TRIGGER fail_opening_next_session
BEFORE INSERT ON opening_sessions
WHEN NEW.lesson_id = 'two-knights-plan'
BEGIN SELECT RAISE(ABORT, 'forced next-session failure'); END
~~~

After the error, assert the old session is active, exactly one session exists, no new progress row exists, and the journey still points to the old lesson.

- [ ] **Step 2: Run and verify RED**

~~~bash
go test ./internal/openings -run 'TestCompleteLessonActivityCreatesNextSessionAtomically|TestCompleteLessonActivityRollsBackFailedNextSession' -count=1
~~~

Expected: compilation fails because NextSession and the returned StoredSession do not exist.

- [ ] **Step 3: Extend and validate the command**

Add NextSession *SessionSeed. When non-nil, validate before opening the transaction:

- current status is completed;
- next mode is lesson;
- course, generation, and depth match the current session;
- next lesson matches Journey.CurrentLessonID;
- next seed and journey pass existing validators.

Return an explicit error for every mismatch.

- [ ] **Step 4: Insert the next session in the existing transaction**

After updating the current session to completed and before saving the journey:

1. construct a StoredSession with uuid.NewString and active status;
2. encode and insert it with the same columns as CreateLessonSession;
3. save the journey;
4. commit; and
5. return the next session.

The old-session update must precede the insert so the single-resumable index remains satisfied. Without NextSession, return completion.Session.

- [ ] **Step 5: Adapt the existing service call without changing behavior**

Use:

~~~go
persisted, err := s.store.CompleteLessonActivity(ctx, completion)
if err != nil {
	return OpeningActivityResult{}, err
}
session = persisted
~~~

NextSession remains nil at this task boundary.

- [ ] **Step 6: Run rollback and package tests**

~~~bash
gofmt -w internal/openings/user_store.go internal/openings/user_store_activity.go internal/openings/user_store_test.go internal/openings/service_activity.go
go test ./internal/openings -run 'TestCompleteLessonActivityCreatesNextSessionAtomically|TestCompleteLessonActivityRollsBackFailedNextSession|TestOpeningActivityCompletionRollsBackSessionProgressAndJourney|TestOpeningDecisionCompletionRollsBackAttemptReviewProgressAndJourney' -count=1
go test ./internal/openings -count=1
~~~

- [ ] **Step 7: Commit**

~~~bash
git add internal/openings/user_store.go internal/openings/user_store_activity.go internal/openings/user_store_test.go internal/openings/service_activity.go
git commit -m "feat: roll opening sessions atomically"
~~~

---

### Task 3: Continue the service directly into the next teaching node

**Files:**
- Modify: internal/openings/service.go
- Modify: internal/openings/service_activity.go
- Modify: internal/openings/service_activity_test.go
- Modify: internal/openings/service_rebase_test.go
- Modify: internal/openings/service_test.go
- Modify: internal/openings/views.go

**Interfaces:**
- Consumes: Task 2's atomic NextSession rollover.
- Produces: studySessionSeed(course, generationID, lesson, depth, activityIndex, now) (SessionSeed, error).
- Changes: a completed study activity returns an active next session or final completed course view; never a roadmap checkpoint.
- Removes: OpeningRoadmapCheckpoint, OpeningActivityResult.Checkpoint, and lessonCheckpointAfterCompletion.

- [ ] **Step 1: Replace the checkpoint service test**

Rename TestOpeningServiceCompletesActivitiesAndReturnsCheckpoint to TestOpeningServiceContinuesAcrossTeachingNodes. Keep same-node assertions and replace final assertions with:

~~~go
done, err := fixture.service.AdvanceActivity(ctx, started.SessionID)
if err != nil {
	t.Fatal(err)
}
if done.Session.Status != OpeningStatusActive || done.Session.Current == nil ||
	done.Session.LessonID != "two-knights-plan" ||
	done.Session.Current.ActivityID != "two-knights-d3-decision" ||
	done.Session.SessionID == started.SessionID {
	t.Fatalf("cross-node result = %+v", done)
}
resumed, err := fixture.service.Resume(ctx)
if err != nil || resumed == nil || resumed.SessionID != done.Session.SessionID ||
	resumed.Current == nil || resumed.Current.ActivityID != "two-knights-d3-decision" {
	t.Fatalf("resumed = %+v, err=%v", resumed, err)
}
~~~

Also assert the journey path is Giuoco then Two Knights, the old session is completed, and the returned course map marks Giuoco learned and Two Knights current.

- [ ] **Step 2: Add final-course and rollback tests**

Complete Two Knights after automatic transition. Assert the result is completed with no Current and ResumableSession returns nil.

Add a service-level next-session insert trigger matching Task 2. Complete the last Giuoco activity and assert failure leaves Giuoco active at the recap, creates no Two Knights session, adds no recap progress, and leaves the journey at Giuoco.

Add a course-revision test that rolls into Two Knights, imports a compatible
generation in which that exact next activity is removed, and resumes. Assert
the existing rebase logic either selects the valid retained unfinished activity
or returns the explicit restart-required view; it must not return an empty
active session or the removed activity.

- [ ] **Step 3: Run and verify RED**

~~~bash
go test ./internal/openings -run 'TestOpeningServiceContinuesAcrossTeachingNodes|TestOpeningServiceCompletesOnlyAfterVisibleCourse|TestOpeningServiceRollsBackCrossNodeContinuation' -count=1
~~~

Expected: Giuoco still returns a completed session and the new tests fail.

- [ ] **Step 4: Factor session-seed creation**

Move initial-position and stateForActivity construction from StartLesson into:

~~~go
func (s *Service) studySessionSeed(
	course CompiledCourse,
	generationID string,
	lesson Lesson,
	depth Depth,
	activityIndex int,
	now time.Time,
) (SessionSeed, error)
~~~

Validate the activity index, load lesson.StartPositionID, start with an empty played-move list, and return a lesson-mode seed. Keep explicit StartLesson behavior unchanged by using the helper.

- [ ] **Step 5: Seed the next node before persistence**

In completeLessonActivity's no-more-activities branch:

1. mark the old session completed and clear its attempt;
2. project teaching progress and mark the current lesson complete in memory;
3. compute the next visible unfinished recommendation without treating the old session as resumable;
4. when a recommendation exists, load its first incomplete required activity and build its SessionSeed;
5. move Journey.CurrentLessonID and PathLessonIDs to that lesson;
6. assign completion.NextSession; and
7. use the StoredSession returned by CompleteLessonActivity for sessionView.

Do not call CreateLessonSession after the transaction.

- [ ] **Step 6: Remove the checkpoint contract**

Delete lessonCheckpointAfterCompletion, OpeningRoadmapCheckpoint, and Checkpoint. Build results as:

~~~go
result := OpeningActivityResult{
	Session: view, ActivityCompleted: true, StepCompleted: true,
	AppliedMoves: frames, FinalFEN: finalFEN,
}
~~~

When no visible unfinished recommendation remains, the store returns the completed old session. Review completion remains unchanged.

- [ ] **Step 7: Update rebase and legacy expectations**

Intermediate lesson-boundary tests now expect an active next session. Tests representing completion of the full visible course continue expecting completed status. Rebase tests may accept a changed session ID after rollover but must still require the exact activity and FEN.

- [ ] **Step 8: Run the full Go checks**

~~~bash
gofmt -w internal/openings/service.go internal/openings/service_activity.go internal/openings/service_activity_test.go internal/openings/service_rebase_test.go internal/openings/service_test.go internal/openings/views.go
go test ./internal/openings -count=1
go test ./... -count=1
go vet ./...
rg -n "OpeningRoadmapCheckpoint|lessonCheckpointAfterCompletion|Checkpoint" internal/openings
~~~

Expected: tests and vet pass; the final search returns no study checkpoint implementation.

- [ ] **Step 9: Commit**

~~~bash
git add internal/openings/service.go internal/openings/service_activity.go internal/openings/service_activity_test.go internal/openings/service_rebase_test.go internal/openings/service_test.go internal/openings/views.go
git commit -m "feat: continue opening course across nodes"
~~~

---

### Task 4: Decode continuous sessions and course maps strictly

**Files:**
- Modify: frontend/src/lib/contracts/openings.ts
- Modify: frontend/src/lib/api.test.ts
- Modify: frontend/src/lib/contracts/opening-journey.test.ts
- Modify: frontend/src/lib/opening-context-contract.test.ts
- Modify: frontend/src/test-fakes.ts
- Modify: frontend/src/lib/api/preview.ts

**Interfaces:**
- Consumes: Tasks 1 and 3 JSON contracts.
- Produces: OpeningCourseMapState, OpeningCourseMapEntry, and OpeningCourseMap.
- Produces: active lesson sessions with required courseMap; active reviews and completed sessions omit it.
- Removes: OpeningRoadmapCheckpoint and checkpoint from OpeningActivityResult.

- [ ] **Step 1: Add failing decoder tests**

Extend the active lesson payload with:

~~~ts
courseMap: {
  completedIdeas: 0,
  totalIdeas: 4,
  entries: [
    {
      lessonId: 'giuoco-c3', moveLabel: '3...Bc5',
      title: 'Giuoco Piano', state: 'current', selectable: true
    },
    {
      lessonId: 'giuoco-d4', parentLessonId: 'giuoco-c3',
      moveLabel: '4.c3', title: 'Play d4', state: 'next', selectable: true
    }
  ]
}
~~~

Assert it decodes unchanged. Add cases rejecting unknown state, missing lessonId, missing title, missing selectable, an active lesson without courseMap, and an active review with courseMap. Add a completed activity result returning an active session with a different lesson ID and no checkpoint.

- [ ] **Step 2: Run and verify RED**

~~~bash
npm --prefix frontend test -- --run src/lib/api.test.ts src/lib/contracts/opening-journey.test.ts src/lib/opening-context-contract.test.ts --single-thread
~~~

Expected: courseMap is dropped and checkpoint remains accepted.

- [ ] **Step 3: Add strict map types**

~~~ts
export type OpeningCourseMapState = 'learned' | 'current' | 'next' | 'available'
export type OpeningCourseMapEntry = {
  lessonId: string
  parentLessonId?: string
  moveLabel?: string
  title: string
  state: OpeningCourseMapState
  selectable: boolean
}
export type OpeningCourseMap = {
  completedIdeas: number
  totalIdeas: number
  entries: OpeningCourseMapEntry[]
}
~~~

Split active sessions so ActiveOpeningLessonView has mode lesson and required courseMap, while ActiveOpeningReviewView has mode review and courseMap?: never. Define ActiveOpeningSessionView as their union.

- [ ] **Step 4: Implement strict decoding and remove checkpoint**

Decode every map field. Require non-negative completedIdeas, positive
totalIdeas, and completedIdeas not greater than totalIdeas. Require courseMap
for active lessons and reject it for reviews and completed sessions. Delete
OpeningRoadmapCheckpoint and decodeCheckpoint. If raw.checkpoint is present,
throw:

~~~ts
throw new Error(path + '.checkpoint is no longer supported')
~~~

Completed results remain legal with either an active next lesson or a completed final session.

- [ ] **Step 5: Update fakes and preview**

Add fakeOpeningCourseMap and attach it to all active lesson fakes. Review fakes omit it.

Change preview study data to two nodes. Finishing the first returns a new active second-node session in the same result and updates previewOpeningSession before returning. Only finishing the second returns a completed lesson session. Remove preview checkpoint creation.

- [ ] **Step 6: Run contract tests and checks**

~~~bash
npm --prefix frontend test -- --run src/lib/api.test.ts src/lib/contracts/opening-journey.test.ts src/lib/opening-context-contract.test.ts --single-thread
npm --prefix frontend run check
~~~

- [ ] **Step 7: Commit**

~~~bash
git add frontend/src/lib/contracts/openings.ts frontend/src/lib/api.test.ts frontend/src/lib/contracts/opening-journey.test.ts frontend/src/lib/opening-context-contract.test.ts frontend/src/test-fakes.ts frontend/src/lib/api/preview.ts
git commit -m "feat: decode continuous opening journeys"
~~~

---

### Task 5: Remove checkpoint state from the lesson controller

**Files:**
- Modify: frontend/src/components/openings/opening-state.ts
- Modify: frontend/src/components/openings/opening-state.test.ts
- Modify: frontend/src/components/openings/opening-controller.ts
- Modify: frontend/src/components/openings/opening-controller.test.ts
- Modify: frontend/src/components/openings/OpeningLessonScreen.svelte
- Modify: frontend/src/components/openings/OpeningLessonScreen.test.ts
- Delete: frontend/src/components/openings/OpeningRoadmapCheckpoint.svelte
- Delete: frontend/src/components/openings/OpeningRoadmapCheckpoint.test.ts

**Interfaces:**
- Consumes: checkpoint-free OpeningActivityResult.
- Changes: acknowledgeOpeningActivity always initializes the returned session.
- Removes: OpeningState phase checkpoint and every checkpoint event path.

- [ ] **Step 1: Write failing state and controller tests**

Build an activity-complete result whose returned session has a new session ID, lesson ID, path, current activity, and map. Assert:

~~~ts
const completed = completeOpeningActivity(animating, 7, finalFen, result, 'Course move found.')
const next = acknowledgeOpeningActivity(completed)
expect(next).toMatchObject({
  phase: 'passive',
  session: {
    sessionId: 'opening-session-2',
    lessonId: 'giuoco-d4',
    current: { activityId: 'd4-concept' }
  }
})
~~~

Add a controller test proving persisted receives the new active session before acknowledgement and change receives it after acknowledgeActivity. Add a screen test proving Next idea changes the heading/map without emitting continue, tree, or home.

- [ ] **Step 2: Run and verify RED**

~~~bash
npm --prefix frontend test -- --run src/components/openings/opening-state.test.ts src/components/openings/opening-controller.test.ts src/components/openings/OpeningLessonScreen.test.ts --single-thread
~~~

- [ ] **Step 3: Collapse acknowledgement**

~~~ts
export function acknowledgeOpeningActivity(state: OpeningState): OpeningState {
  if (state.phase !== 'activity-complete') {
    throw new Error(state.phase + ' state cannot acknowledge a completed activity')
  }
  return initialiseOpening(state.result.session)
}
~~~

Remove checkpoint from the state union and terminal guards.

- [ ] **Step 4: Simplify successful controller transitions**

The zero-frame fast path requires only active returned status. In acknowledgeActivity, announce Next idea plus its title for active results and Opening course complete for final lesson completion. Continue publishing the durable returned session before acknowledgement so Home and restart use the next activity even if Next idea is not pressed.

- [ ] **Step 5: Remove checkpoint rendering**

Delete the checkpoint branch/import. Completed lesson copy becomes **Opening course complete!**; review copy remains **Opening review complete!**. Delete both OpeningRoadmapCheckpoint files.

- [ ] **Step 6: Run focused tests and checks**

~~~bash
npm --prefix frontend test -- --run src/components/openings/opening-state.test.ts src/components/openings/opening-controller.test.ts src/components/openings/OpeningLessonScreen.test.ts --single-thread
npm --prefix frontend run check
rg -n "OpeningRoadmapCheckpoint|phase === 'checkpoint'|checkpoint:" frontend/src
~~~

Expected: tests/check pass and the search returns no checkpoint UI/state code.

- [ ] **Step 7: Commit**

~~~bash
git add frontend/src/components/openings/opening-state.ts frontend/src/components/openings/opening-state.test.ts frontend/src/components/openings/opening-controller.ts frontend/src/components/openings/opening-controller.test.ts frontend/src/components/openings/OpeningLessonScreen.svelte frontend/src/components/openings/OpeningLessonScreen.test.ts frontend/src/components/openings/OpeningRoadmapCheckpoint.svelte frontend/src/components/openings/OpeningRoadmapCheckpoint.test.ts
git commit -m "feat: keep opening lessons in one flow"
~~~

---

### Task 6: Render the concise move-labelled sidebar map

**Files:**
- Create: frontend/src/components/openings/OpeningCourseMap.svelte
- Create: frontend/src/components/openings/OpeningCourseMap.test.ts
- Modify: frontend/src/components/openings/OpeningLessonScreen.svelte
- Modify: frontend/src/components/openings/OpeningLessonScreen.test.ts
- Modify: frontend/src/styles/app.css

**Interfaces:**
- Consumes: OpeningCourseMap.
- Produces: OpeningCourseMap.svelte with prop map and event open.
- Produces: exact state labels ✓ Learned, ▶ Current, → Next, and ○ Available.

- [ ] **Step 1: Write the failing component test**

Render all four states. Assert Course map navigation, the text **0 of 7 course
ideas learned**, 3...Bc5, Giuoco Piano, all four exact state labels, current
aria-current=step, accessible names containing move/title/state, and one open
event from View full map.

- [ ] **Step 2: Run and verify RED**

~~~bash
npm --prefix frontend test -- --run src/components/openings/OpeningCourseMap.test.ts --single-thread
~~~

Expected: module not found.

- [ ] **Step 3: Implement the component**

Use a noninteractive ordered list plus one action:

~~~svelte
<nav class="opening-course-map" aria-label="Course map">
  <div class="opening-course-map-heading">
    <div>
      <h3>Course map</h3>
      <p>{map.completedIdeas} of {map.totalIdeas} course ideas learned</p>
    </div>
    <button class="quiet-action" type="button" on:click={() => dispatch('open')}>
      View full map
    </button>
  </div>
  <ol>
    {#each map.entries as entry (entry.lessonId)}
      <li
        class={'course-map-' + entry.state}
        aria-current={entry.state === 'current' ? 'step' : undefined}
        aria-label={[entry.moveLabel, entry.title, stateLabel(entry.state)].filter(Boolean).join('. ')}
      >
        {#if entry.moveLabel}<span class="course-map-move">{entry.moveLabel}</span>{/if}
        <strong>{entry.title}</strong>
        <span class="course-map-state">{stateLabel(entry.state)}</span>
      </li>
    {/each}
  </ol>
</nav>
~~~

Rows are not clickable; deliberate selection happens in the full map.

Compute each row's indentation by following parentLessonId through the entries
map, stopping at a missing parent or repeated ID. Expose the resulting depth as
a CSS custom property on the row. This keeps nested paths concise without
requiring another frontend tree model.

- [ ] **Step 4: Integrate in the lesson sidebar**

For active lessons, render immediately after OpeningPathContext and forward open
to the existing tree event. Replace the node-local **Idea N of M** progress
label and bar with the map's course-wide completedIdeas/totalIdeas so progress
never resets at a node boundary. Reviews keep their existing review progress.
Do not key the entire lesson layout by session or lesson ID.

- [ ] **Step 5: Add compact responsive styling**

Use existing variables. Limit the list to five visible rows before internal scrolling; use a fixed-width notation column; keep titles concise; show state text at every width; use indentation/connectors without horizontal page overflow; emphasize current/next typographically; retain symbols/text without color.

- [ ] **Step 6: Test cross-node state movement**

Initial sidebar: Giuoco Current, d4 Next, and the course-wide idea count. After
feedback and Next idea: Giuoco Learned, d4 Current, completedIdeas increases,
totalIdeas stays fixed, the new move/title is visible, and no route event is
emitted. Assert the old node-local progress string does not reappear.

- [ ] **Step 7: Run tests and checks**

~~~bash
npm --prefix frontend test -- --run src/components/openings/OpeningCourseMap.test.ts src/components/openings/OpeningLessonScreen.test.ts --single-thread
npm --prefix frontend run check
~~~

- [ ] **Step 8: Commit**

~~~bash
git add frontend/src/components/openings/OpeningCourseMap.svelte frontend/src/components/openings/OpeningCourseMap.test.ts frontend/src/components/openings/OpeningLessonScreen.svelte frontend/src/components/openings/OpeningLessonScreen.test.ts frontend/src/styles/app.css
git commit -m "feat: show opening course map in lesson sidebar"
~~~

---

### Task 7: Make resume primary and full-map navigation intentional

**Files:**
- Modify: frontend/src/components/home/HomeHub.svelte
- Modify: frontend/src/components/home/HomeHub.test.ts
- Modify: frontend/src/components/app/NormalShell.svelte
- Modify: frontend/src/components/openings/OpeningHub.svelte
- Modify: frontend/src/components/openings/OpeningHub.test.ts
- Modify: frontend/src/components/openings/OpeningTeachingTree.svelte
- Modify: frontend/src/components/openings/OpeningTeachingTree.test.ts
- Modify: frontend/src/components/openings/OpeningTreeBranch.svelte
- Modify: frontend/src/App.test.ts
- Modify: frontend/src/styles/app.css

**Interfaces:**
- Produces: HomeHub event continueOpening plus existing openings.
- Changes: OpeningTeachingTree accepts currentLessonId = '' and emits resume when current is chosen.
- Consumes: existing atomic StartLesson for intentional Start here.

- [ ] **Step 1: Write failing Home and tree tests**

For one resumable course, assert buttons named Continue opening lesson and Course map. First emits continueOpening; second emits openings.

Render the tree with currentLessonId. Assert exact actions:
- current: Return to your lesson;
- recommended non-current: Start next idea here;
- completed non-current: Study again;
- other visible: Start here.

Current emits resume; others emit lesson ID.

- [ ] **Step 2: Run and verify RED**

~~~bash
npm --prefix frontend test -- --run src/components/home/HomeHub.test.ts src/components/openings/OpeningTeachingTree.test.ts src/components/openings/OpeningHub.test.ts src/App.test.ts --single-thread
~~~

- [ ] **Step 3: Split Home continuation from map navigation**

With one resumable opening, render primary Continue opening lesson and secondary Course map. With several courses, continue the one resumable course directly because only one opening session can be resumable. With no resumable course, retain Learn Openings. Wire continueOpening to resumeOpening and openings to openOpeningHub.

- [ ] **Step 4: Preserve active state while viewing the map**

showOpeningTree retains activeOpeningSession and deferredOpeningSession for an active lesson, refreshes course progress, and navigates to openings. Return to your lesson calls resumeOpening. Do not pause merely to inspect the map.

- [ ] **Step 5: Add contextual tree actions**

Pass course.hasResumable ? course.currentLessonId : '' through OpeningHub, OpeningTeachingTree, and OpeningTreeBranch. Use:

~~~ts
function actionLabel(): string {
  if (branch.node.lessonId === currentLessonId) return 'Return to your lesson'
  if (branch.node.recommended) return 'Start next idea here'
  if (branch.node.progress === 'completed') return 'Study again'
  return 'Start here'
}
~~~

Current emits resume; others emit lesson. Preserve hidden disabling and accessible state labels.

- [ ] **Step 6: Test return, successful Start here, and failed Start here**

In App.test.ts:
1. open map from an active lesson and return to the same activity;
2. open again, choose another visible node, and adopt startOpeningLesson's returned session;
3. make startOpeningLesson reject, show the hub error, then Return to your lesson and resume the original session.

- [ ] **Step 7: Run tests and checks**

~~~bash
npm --prefix frontend test -- --run src/components/home/HomeHub.test.ts src/components/openings/OpeningTeachingTree.test.ts src/components/openings/OpeningHub.test.ts src/App.test.ts --single-thread
npm --prefix frontend run check
~~~

- [ ] **Step 8: Commit**

~~~bash
git add frontend/src/components/home/HomeHub.svelte frontend/src/components/home/HomeHub.test.ts frontend/src/components/app/NormalShell.svelte frontend/src/components/openings/OpeningHub.svelte frontend/src/components/openings/OpeningHub.test.ts frontend/src/components/openings/OpeningTeachingTree.svelte frontend/src/components/openings/OpeningTeachingTree.test.ts frontend/src/components/openings/OpeningTreeBranch.svelte frontend/src/App.test.ts frontend/src/styles/app.css
git commit -m "feat: separate opening resume from course navigation"
~~~

---

### Task 8: Verify the uninterrupted flow and rebuild

**Files:**
- Modify: frontend/tests/openings.spec.ts
- Modify: frontend/tests/test-backend.ts
- Modify: frontend/tests/test-backend-state.ts when the next session/activity needs persisted inspection
- Modify: docs/operations/board-acceptance.md
- Normalize if regenerated: frontend/wailsjs/go/models.ts
- Generate ignored: build/bin/Chess Trainer.app

**Interfaces:**
- Consumes: Tasks 1–7.
- Produces: browser proof of cross-node continuity, exact resume, map return, alternate selection, and final completion.
- Produces: verified signed arm64 macOS bundle.

- [ ] **Step 1: Rewrite the two-node Playwright journey**

The backend's final Giuoco activity returns an active Two Knights session and stores it as resumable. Replace checkpoint assertions with:

~~~ts
await page.getByRole('button', { name: 'Done comparing' }).click()
await page.getByRole('button', { name: 'Next idea' }).click()
await expect(page.getByRole('heading', { name: 'Choose the quiet setup' })).toBeVisible()
await expect(page.getByRole('heading', { name: /complete/i })).toHaveCount(0)
const map = page.getByRole('navigation', { name: 'Course map' })
await expect(map).toContainText('✓ Learned')
await expect(map).toContainText('▶ Current')
await expect(map).toContainText('3...Nf6')
await expect(map).toContainText('3 of 4 course ideas learned')
~~~

Retain the existing board-host continuity probe for same-node activities. Add a
separate probe to .opening-lesson-layout before the cross-node transition and
assert that layout element remains mounted afterward. The ChessBoard may
legitimately be regenerated when the next node has a different starting FEN;
that internal reset must not be mistaken for screen navigation.

- [ ] **Step 2: Add exact restart/resume coverage**

Immediately after the backend commits the next node, navigate Home. Assert Continue opening lesson. Reload, click it, and assert exact Two Knights activity title, board state, session ID in backend state, and sidebar Current without visiting the hub.

- [ ] **Step 3: Add map-return and alternate-start coverage**

From an active lesson: View full map, Return to your lesson, verify the same activity, reopen, choose Study again on a completed node, and verify prior completed progress remains. Record explicit selection separately from automatic rollover in backend state.

- [ ] **Step 4: Update final completion and acceptance notes**

After the last visible unfinished node, assert exactly one Opening course complete summary and no active opening continuation. Add manual checks to board-acceptance.md for same-screen rollover, move-labelled map states, direct restart resume, map return, and Start here.

- [ ] **Step 5: Run focused end-to-end tests**

~~~bash
npm --prefix frontend run test:e2e -- openings.spec.ts
~~~

- [ ] **Step 6: Run complete verification**

~~~bash
go test -race ./... -count=1
go mod verify
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run test:e2e
git diff --check
~~~

- [ ] **Step 7: Rebuild with pinned Wails**

~~~bash
GOWORK=off GOTOOLCHAIN=local \
  go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json \
  frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
~~~

- [ ] **Step 8: Verify bundle and repository state**

~~~bash
codesign --verify --deep --strict --verbose=2 "build/bin/Chess Trainer.app"
file "build/bin/Chess Trainer.app/Contents/MacOS/Chess Trainer"
git diff --check
git status --short
~~~

Expected: signature passes, binary is Mach-O arm64, and only intentional test/documentation changes remain.

- [ ] **Step 9: Commit**

~~~bash
git add frontend/tests/openings.spec.ts frontend/tests/test-backend.ts frontend/tests/test-backend-state.ts docs/operations/board-acceptance.md frontend/wailsjs/go/models.ts
git commit -m "test: verify seamless opening course flow"
~~~

If generated files are unchanged, omit them from git add.
