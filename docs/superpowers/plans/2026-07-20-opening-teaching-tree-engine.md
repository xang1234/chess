# Opening Teaching Tree Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the rigid five-step opening flow with schema-v2 teaching trees, flexible decision-point activities, durable course journeys, roadmap checkpoints, and seamless continuation.

**Architecture:** Normalize schema-v1 and schema-v2 packs into one compiled activity model, persist v2 activities and edges beside the existing move graph, and link one active board session to a per-course journey. Extend the existing service, board, prompt, review, import, and Wails layers; do not create a parallel curriculum engine.

**Tech Stack:** Go 1.25 module, modernc SQLite, Wails 2.12.0, Svelte 3.49, TypeScript 4.6, Vitest/Testing Library, and Playwright.

## Global Constraints

- Follow `docs/superpowers/specs/2026-07-20-opening-course-teaching-tree-design.md`.
- Keep the application offline and local-first; add no runtime network call or hosted dependency.
- Do not change tactical puzzle records, sessions, ratings, or review behavior.
- Keep copyrighted MCO-derived `.ctcourse` data outside Git and release assets.
- Keep schema-v1 packs importable; use schema v2 for new authoring.
- Preserve stable course, lesson, activity, prompt, position, and move IDs.
- Reference is canonical; Quick and Standard are connected filtered routes.
- Reject a schema-v2 lesson that requires the same primary answer from the same position twice.
- Once completed, a legacy or v2 teaching node remains completed when content grows.
- Reuse the board, move rules, animation, sound, hints, accepted alternatives, notes, explorer, importer, course generations, fingerprints, and opening-review scheduler.
- Preserve the current uncommitted concise-note patch. Before creating an isolated implementation worktree, commit that verified patch separately or transfer its exact diff into the worktree without modification.
- Use `apply_patch` for source edits and explicit paths for Git operations.

## File Map

- `internal/openings/types.go`, `coursepack.go`, and new `coursepack_normalize.go`: v2 source model and v1 normalization.
- New `compiler_activities.go` and `compiler_teaching_tree.go`: flexible activities, rooted tree, depth routes, and duplicate-decision protection.
- `internal/storage/migrations/courses/002.sql` plus catalogue readers/writers: v2 content persistence.
- `internal/storage/migrations/user/005.sql`, new `user_store_journey.go`, and progress/session stores: durable journey and sticky completion.
- New `service_tree.go`, `service_activity.go`, and `service_decision.go`: recommendation, activities, and checkpoints.
- `service_rebase.go`: stable activity-ID migration.
- `views.go`, `normal_controller.go`, Wails output, and TypeScript contracts: one validated API boundary.
- New focused Svelte tree, path, activity, and checkpoint components; existing hub/controller/screen become orchestration shells.
- `internal/openings/testdata/mini.ctcourse`: retained schema-v1 fixture.
- New `internal/openings/testdata/tree.ctcourse`: original synthetic schema-v2 two-node journey fixture.

---

### Task 1: Schema-v2 Model and Schema-v1 Normalization

**Files:**
- Modify: `internal/openings/types.go`
- Modify: `internal/openings/coursepack.go`
- Create: `internal/openings/coursepack_normalize.go`
- Modify: `internal/openings/coursepack_test.go`
- Create: `internal/openings/coursepack_normalize_test.go`

**Interfaces:**
- Consumes: `CoursePack`, `LessonStep`, `StepKind`, and `Depth`.
- Produces: `ActivityKind`, `LessonActivity`, `LessonEdge`, and `NormalizeCoursePack(CoursePack) (CoursePack, error)`.

- [ ] **Step 1: Write failing version and normalization tests**

```go
func TestDecodeCoursePackAcceptsSchemaVersionsOneAndTwo(t *testing.T) {
	v1, err := DecodeCoursePack(bytes.NewReader(readMiniCoursePack(t)))
	if err != nil || v1.SchemaVersion != 1 {
		t.Fatalf("schema v1 = %+v err=%v", v1, err)
	}
	v2JSON := bytes.Replace(readMiniCoursePack(t), []byte(`"schemaVersion":1`), []byte(`"schemaVersion":2`), 1)
	v2JSON = bytes.Replace(v2JSON, []byte(`"chapters":`), []byte(`"lessonEdges":[],"chapters":`), 1)
	v2JSON = bytes.Replace(v2JSON, []byte(`"steps":`), []byte(`"activities":`), 1)
	v2JSON = bytes.Replace(v2JSON, []byte(`"stepId":`), []byte(`"activityId":`), -1)
	pack, err := DecodeCoursePack(bytes.NewReader(v2JSON))
	if err != nil || pack.SchemaVersion != 2 {
		t.Fatalf("schema v2 = %+v err=%v", pack, err)
	}
}

func TestNormalizeCoursePackKeepsLegacyActivityIDs(t *testing.T) {
	pack, err := NormalizeCoursePack(decodeMiniPack(t))
	if err != nil {
		t.Fatal(err)
	}
	lesson := pack.Lessons[0]
	if len(lesson.Activities) != 5 || lesson.Activities[0].ActivityID != lesson.Steps[0].StepID {
		t.Fatalf("activities = %#v", lesson.Activities)
	}
	if lesson.Activities[0].Kind != ActivityConcept || lesson.Activities[2].Kind != ActivityDecision {
		t.Fatalf("kinds = %#v", lesson.Activities)
	}
}
```

- [ ] **Step 2: Run the tests and verify failure**

Run: `go test ./internal/openings -run 'TestDecodeCoursePackAcceptsSchemaVersionsOneAndTwo|TestNormalizeCoursePack' -count=1`

Expected: FAIL because schema 2 and activity types are unsupported.

- [ ] **Step 3: Add exact source types**

```go
type ActivityKind string

const (
	ActivityConcept ActivityKind = "concept"
	ActivityDemonstration ActivityKind = "demonstration"
	ActivityDecision ActivityKind = "decision"
	ActivityComparison ActivityKind = "comparison"
	ActivityRecap ActivityKind = "recap"
	ActivityReference ActivityKind = "reference"
)

type LessonEdgeKind string

const (
	EdgeContinuation LessonEdgeKind = "continuation"
	EdgeAlternative LessonEdgeKind = "alternative"
	EdgeReference LessonEdgeKind = "reference"
)

type ActivityLine struct { Label string `json:"label"`; MoveIDs []string `json:"moveIds"` }
type BoardAnnotation struct { Kind string `json:"kind"`; From string `json:"from"`; To string `json:"to,omitempty"`; Label string `json:"label,omitempty"` }

type LessonActivity struct {
	ActivityID string `json:"activityId"`
	Kind ActivityKind `json:"kind"`
	Title string `json:"title"`
	Instruction string `json:"instruction"`
	Required bool `json:"required"`
	PositionID string `json:"positionId,omitempty"`
	NoteIDs []string `json:"noteIds"`
	MoveIDs []string `json:"moveIds"`
	PromptID string `json:"promptId,omitempty"`
	Comparison []ActivityLine `json:"comparison,omitempty"`
	Annotations []BoardAnnotation `json:"annotations,omitempty"`
}

type LessonEdge struct {
	EdgeID string `json:"edgeId"`
	FromLessonID string `json:"fromLessonId"`
	ToLessonID string `json:"toLessonId"`
	Ordinal int `json:"ordinal"`
	Kind LessonEdgeKind `json:"kind"`
	Label string `json:"label,omitempty"`
	MinimumDepth Depth `json:"minimumDepth"`
}
```

Add `Activities []LessonActivity `json:"activities,omitempty"`` beside retained legacy `Steps`, and add `LessonEdges []LessonEdge `json:"lessonEdges,omitempty"`` to `CoursePack`.

- [ ] **Step 4: Accept both schema versions and normalize v1**

Change the decoder guard to accept only 1 or 2. Create `NormalizeCoursePack` that maps Explain to Concept, Watch to Demonstration, and Try/Branch/Recall to Decision while preserving step IDs, positions, notes, moves, and prompts. For v1 packs with multiple lessons, synthesize continuation edges in authored chapter/lesson order. Reject `steps` in schema 2 and `activities` in schema 1.

At the start of `Compile`, call:

```go
normalized, err := NormalizeCoursePack(pack)
if err != nil { return CompiledCourse{}, err }
compiler := newCourseCompiler(normalized, rules)
```

- [ ] **Step 5: Run tests and commit**

Run: `go test ./internal/openings -run 'TestDecodeCoursePack|TestNormalizeCoursePack' -count=1`

Expected: PASS.

```bash
git add internal/openings/types.go internal/openings/coursepack.go internal/openings/coursepack_normalize.go internal/openings/coursepack_test.go internal/openings/coursepack_normalize_test.go
git commit -m "feat: add opening course activity schema"
```

---

### Task 2: Activity and Teaching-Tree Compiler

**Files:**
- Modify: `internal/openings/compiler.go`
- Create: `internal/openings/compiler_activities.go`
- Create: `internal/openings/compiler_teaching_tree.go`
- Delete or reduce: `internal/openings/compiler_lessons.go`
- Create: `internal/openings/compiler_activities_test.go`
- Create: `internal/openings/compiler_teaching_tree_test.go`
- Create: `internal/openings/testdata/tree.ctcourse`

**Interfaces:**
- Consumes: normalized activities/edges and existing move/prompt indexes.
- Produces: `RootLessonID`, `LessonChildren`, `LessonParent`, and `RequiredActivityIDs(Lesson) []string`.

- [ ] **Step 1: Create a two-node synthetic v2 fixture**

Copy the original synthetic graph data from `mini.ctcourse`; set schema 2. Define root lesson `giuoco-plan` with required Concept `giuoco-concept`, Decision `giuoco-c3-decision` using prompt `recall-c3`, and Recap `giuoco-recap`. Define child `two-knights-plan` with Decision `two-knights-d3-decision` using `recall-d3`. Connect them with edge `giuoco-to-two-knights`, ordinal 1, kind `continuation`, minimum depth `standard`.

- [ ] **Step 2: Write failing diagnostics tests**

```go
func TestCompileTeachingTreeRejectsStructuralAndActivityErrors(t *testing.T) {
	tests := []struct { name, code string; mutate func(*CoursePack) }{
		{"cycle", "lesson_tree_cycle", func(p *CoursePack) { p.LessonEdges = append(p.LessonEdges, LessonEdge{EdgeID:"cycle", FromLessonID:"two-knights-plan", ToLessonID:"giuoco-plan", Ordinal:1, Kind:EdgeContinuation, MinimumDepth:DepthStandard}) }},
		{"second parent", "lesson_multiple_parents", func(p *CoursePack) { p.LessonEdges = append(p.LessonEdges, LessonEdge{EdgeID:"parent-2", FromLessonID:"giuoco-plan", ToLessonID:"two-knights-plan", Ordinal:2, Kind:EdgeAlternative, MinimumDepth:DepthStandard}) }},
		{"missing prompt", "missing_prompt", func(p *CoursePack) { p.Lessons[0].Activities[1].PromptID = "" }},
		{"duplicate answer", "duplicate_lesson_decision", func(p *CoursePack) { a := p.Lessons[0].Activities[1]; a.ActivityID="duplicate-c3"; p.Lessons[0].Activities=append(p.Lessons[0].Activities,a) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pack := decodeTreePack(t); tc.mutate(&pack)
			_, err := Compile(pack, chessrules.Rules{})
			var validation *ValidationError
			if !errors.As(err, &validation) || !hasDiagnosticCode(validation.Diagnostics, tc.code) { t.Fatalf("error=%v diagnostics=%+v want=%s", err, validation, tc.code) }
		})
	}
}
```

- [ ] **Step 3: Run the test and verify failure**

Run: `go test ./internal/openings -run TestCompileTeachingTree -count=1`

Expected: FAIL because the compiled tree and diagnostics do not exist.

- [ ] **Step 4: Add compiled indexes and activity validation**

Extend `CompiledCourse` with `RootLessonID string`, `LessonChildren map[string][]LessonEdge`, and `LessonParent map[string]LessonEdge`. Initialize them in every constructor/load path.

Implement `RequiredActivityIDs`. Validate activity IDs, text, depth, position, notes, annotations, and allowed fields. Demonstration requires a connected move sequence. Decision requires a prompt at the same position and permits at most one automatic opponent continuation. Comparison requires at least two labeled connected lines. Concept/Recap/Reference forbid prompts and required moves. Reference activities must be optional: they are excluded from the primary activity cursor and exposed as collapsed lesson-level Deeper Analysis sections, so reading them never blocks node completion.

For schema 2 only, key each required decision by `prompt.PositionID + "\x00" + prompt.PrimaryMoveID`; emit `duplicate_lesson_decision` on duplicates. Legacy normalized lessons are exempt.

- [ ] **Step 5: Validate the rooted tree and depth routes**

Index unique edges, reject missing lessons, sibling ordinal collisions, multiple parents, invalid kinds, cycles, disconnected lessons, and depth inversions. Require one Quick root. Sort children by ordinal then edge ID. At each depth, walk visible edges from the root and emit `lesson_depth_route` for a visible unreachable lesson.

Replace `validateLessons` with `validateTeachingTree` and `validateActivities`; remove phase ordering, required phase sets, Explain-first, and Recall-last rules.

- [ ] **Step 6: Run all Go tests and commit**

Run: `go test ./... -count=1`

Expected: PASS for v1 compatibility and v2 anti-repetition.

```bash
git add internal/openings/compiler.go internal/openings/compiler_activities.go internal/openings/compiler_teaching_tree.go internal/openings/compiler_lessons.go internal/openings/compiler_activities_test.go internal/openings/compiler_teaching_tree_test.go internal/openings/testdata/tree.ctcourse
git commit -m "feat: compile opening teaching trees"
```

---

### Task 3: Persist Schema-v2 Content

**Files:**
- Create: `internal/storage/migrations/courses/002.sql`
- Modify: `internal/openings/catalog_insert.go`
- Modify: `internal/openings/catalog_read.go`
- Modify: `internal/openings/catalog_read_rows.go`
- Modify: `internal/openings/catalog_test.go`

**Interfaces:**
- Consumes: compiled v1/v2 packs.
- Produces: generation-scoped round-trip storage for v2 edges/activities while reading legacy steps.

- [ ] **Step 1: Write a failing v2 round-trip test**

```go
func TestSQLiteCatalogRoundTripsTeachingTree(t *testing.T) {
	ctx := context.Background(); catalog := NewSQLiteCatalog(openCourseCatalogTestDB(t)); want := compileTreeCourse(t)
	result, err := catalog.Replace(ctx, want, "/private/tree.ctcourse", "sha-tree"); if err != nil { t.Fatal(err) }
	got, err := catalog.LoadGeneration(ctx, result.GenerationID); if err != nil { t.Fatal(err) }
	if got.RootLessonID != "giuoco-plan" || len(got.LessonChildren["giuoco-plan"]) != 1 { t.Fatalf("tree=%+v", got) }
	if !reflect.DeepEqual(got.Lessons["giuoco-plan"].Activities, want.Lessons["giuoco-plan"].Activities) { t.Fatalf("activities=%#v", got.Lessons["giuoco-plan"].Activities) }
}
```

- [ ] **Step 2: Run and verify failure**

Run: `go test ./internal/openings -run TestSQLiteCatalogRoundTripsTeachingTree -count=1`

Expected: FAIL because v2 tables do not exist.

- [ ] **Step 3: Add course tables**

```sql
CREATE TABLE course_lesson_edges (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  edge_id TEXT NOT NULL, from_lesson_id TEXT NOT NULL, to_lesson_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL CHECK(ordinal > 0),
  kind TEXT NOT NULL CHECK(kind IN ('continuation','alternative','reference')),
  label TEXT NOT NULL,
  minimum_depth TEXT NOT NULL CHECK(minimum_depth IN ('quick','standard','reference')),
  PRIMARY KEY(generation_id, edge_id), UNIQUE(generation_id, from_lesson_id, ordinal),
  FOREIGN KEY(generation_id, from_lesson_id) REFERENCES course_lessons(generation_id, lesson_id),
  FOREIGN KEY(generation_id, to_lesson_id) REFERENCES course_lessons(generation_id, lesson_id)
);
CREATE TABLE course_lesson_activities (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  lesson_id TEXT NOT NULL, ordinal INTEGER NOT NULL CHECK(ordinal > 0), activity_id TEXT NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('concept','demonstration','decision','comparison','recap','reference')),
  required INTEGER NOT NULL CHECK(required IN (0,1)), position_id TEXT, data_json TEXT NOT NULL,
  PRIMARY KEY(generation_id, lesson_id, activity_id), UNIQUE(generation_id, lesson_id, ordinal),
  FOREIGN KEY(generation_id, lesson_id) REFERENCES course_lessons(generation_id, lesson_id),
  FOREIGN KEY(generation_id, position_id) REFERENCES course_positions(generation_id, position_id)
);
CREATE INDEX idx_course_lesson_edges_parent ON course_lesson_edges(generation_id, from_lesson_id, ordinal);
```

- [ ] **Step 4: Insert/load by schema version**

Always insert lesson rows. For schema 1, retain `course_lesson_steps`; for schema 2, insert encoded activities and then edges after all lessons. Load based on stored generation schema version. Normalize v1 rows after reconstruction; load v2 rows directly. Call one shared `indexCompiledTeachingTree` so loaded and freshly compiled courses expose identical indexes.

- [ ] **Step 5: Run tests and commit**

Run: `go test ./internal/storage ./internal/openings -count=1`

Expected: PASS, including cleanup cascade tests.

```bash
git add internal/storage/migrations/courses/002.sql internal/openings/catalog_insert.go internal/openings/catalog_read.go internal/openings/catalog_read_rows.go internal/openings/catalog_test.go
git commit -m "feat: persist opening teaching trees"
```

---

### Task 4: Course Journey and Sticky Activity Progress

**Files:**
- Create: `internal/storage/migrations/user/005.sql`
- Modify: `internal/openings/user_store.go`
- Create: `internal/openings/user_store_journey.go`
- Modify: `internal/openings/user_store_progress.go`
- Modify: `internal/openings/user_store_session.go`
- Modify: `internal/openings/user_store_test.go`

**Interfaces:**
- Produces: `CourseJourney`, `Journey`, `SaveJourney`, `RecordActivityProgress`, and session `ActivityIndex`.

- [ ] **Step 1: Write failing persistence tests**

```go
func TestUserStoreJourneyRoundTrip(t *testing.T) {
	ctx := context.Background(); now := time.Date(2026,time.July,20,12,0,0,0,time.UTC); store := NewUserStore(openOpeningUserTestDB(t))
	want := CourseJourney{CourseID:"italian-white",Depth:DepthStandard,CurrentLessonID:"giuoco-plan",CurrentActivityID:"giuoco-c3-decision",PathLessonIDs:[]string{"foundations","giuoco-plan"},LastRecommendedLessonID:"two-knights-plan",ActiveSessionID:"session-1",CreatedAt:now,UpdatedAt:now}
	if err := store.SaveJourney(ctx,want); err != nil { t.Fatal(err) }
	got, err := store.Journey(ctx,want.CourseID,DepthReference); if err != nil || !reflect.DeepEqual(got,want) { t.Fatalf("got=%#v want=%#v err=%v",got,want,err) }
}

func TestCompletedOpeningNodeStaysCompleteWhenRequirementsGrow(t *testing.T) {
	ctx := context.Background(); store := NewUserStore(openOpeningUserTestDB(t)); now := time.Date(2026,time.July,20,12,0,0,0,time.UTC)
	err := store.RecordActivityProgress(ctx,ActivityProgressUpdate{CourseID:"italian-white",LessonID:"giuoco-plan",CompletedActivityID:"decision-c3",RequiredActivityIDs:[]string{"decision-c3"},Now:now}); if err != nil { t.Fatal(err) }
	progress, err := store.LessonProgress(ctx,"italian-white","giuoco-plan",[]string{"decision-c3","new-required"})
	if err != nil || !progress.Completed { t.Fatalf("progress=%+v err=%v",progress,err) }
}
```

- [ ] **Step 2: Run and verify failure**

Run: `go test ./internal/openings -run 'TestUserStoreJourney|TestCompletedOpeningNode' -count=1`

Expected: FAIL because journey/activity APIs do not exist.

- [ ] **Step 3: Add migration and types**

In `user/005.sql`, rename session `step_index` to `activity_index`; rename progress JSON/count columns from step to activity terminology; create `opening_course_journeys(course_id PRIMARY KEY, depth, current_lesson_id, current_activity_id, path_lesson_ids_json, last_recommended_lesson_id, active_session_id, created_at, updated_at)`. Backfill one row from every resumable opening session, then from preferences when no journey exists.

```go
type CourseJourney struct {
	CourseID string; Depth Depth; CurrentLessonID string; CurrentActivityID string
	PathLessonIDs []string; LastRecommendedLessonID string; ActiveSessionID string; CreatedAt time.Time; UpdatedAt time.Time
}
type ActivityProgressUpdate struct {
	CourseID string; LessonID string; CompletedActivityID string; RequiredActivityIDs []string; Now time.Time
}
```

- [ ] **Step 4: Implement journey CRUD and sticky completion**

Validate depth, nonblank unique path IDs, and timestamps. Encode paths as JSON. Preserve `created_at` on upsert and return an empty initialized journey with default depth on `sql.ErrNoRows`.

`RecordActivityProgress` merges the completed ID into ordered stored IDs, sets `completed_at` when every current required ID is present, and uses `completed_at = COALESCE(opening_lesson_progress.completed_at, excluded.completed_at)`. `LessonProgress` treats any existing `completed_at` as authoritative even when current required IDs differ.

Rename `StoredSession.StepIndex`, `SessionSeed`, restart checkpoints, SQL scans, and messages to `ActivityIndex`.

- [ ] **Step 5: Run tests and commit**

Run: `go test ./internal/storage ./internal/openings -count=1`

Expected: PASS, including migration of completed rows and one active session.

```bash
git add internal/storage/migrations/user/005.sql internal/openings/user_store.go internal/openings/user_store_journey.go internal/openings/user_store_progress.go internal/openings/user_store_session.go internal/openings/user_store_test.go
git commit -m "feat: persist opening course journeys"
```

---

### Task 5: Tree Projection and Recommendation Service

**Files:**
- Create: `internal/openings/service_tree.go`
- Modify: `internal/openings/service.go`
- Modify: `internal/openings/views.go`
- Create: `internal/openings/service_tree_test.go`

**Interfaces:**
- Produces: `OpeningTeachingTreeView`, node progress flags, current path, and deterministic recommendation.

- [ ] **Step 1: Write failing home tests**

```go
func TestOpeningHomeReturnsTeachingTreeAndRecommendation(t *testing.T) {
	fixture := newTreeServiceFixture(t); home, err := fixture.service.Home(context.Background()); if err != nil { t.Fatal(err) }
	course := home.Courses[0]
	if course.Tree.RootLessonID!="giuoco-plan" || len(course.Tree.Nodes)!=2 || len(course.Tree.Edges)!=1 { t.Fatalf("tree=%+v",course.Tree) }
	if course.RecommendedLessonID!="giuoco-plan" || !course.Tree.Nodes[0].Recommended { t.Fatalf("course=%+v",course) }
}
```

Add cases for exact in-progress journey, continuation after completion, Quick visibility, hidden-node preservation, all-visible-complete empty recommendation, and due-review badge mapped from prompt to lesson.

- [ ] **Step 2: Run and verify failure**

Run: `go test ./internal/openings -run 'TestOpeningHomeReturnsTeachingTree|TestRecommendation' -count=1`

Expected: FAIL because tree views do not exist.

- [ ] **Step 3: Define and fill views**

Add `OpeningNodeProgress` values `available`, `in_progress`, `completed`; `OpeningTeachingNodeView` with an independent `Visible` flag; `OpeningTeachingEdgeView`; `OpeningTeachingTreeView`; and `OpeningPathItem`. Return the whole tree at every depth so hidden branches remain useful roadmap context, but compute counts, selection, and recommendation only from nodes visible at the selected depth. Extend course summary with Tree, current lesson/activity/path, recommended lesson ID/title, and resumable flag.

Choose recommendation in this order: visible resumable current lesson; visible incomplete journey lesson; first incomplete continuation child of deepest completed path node; continuation-first preorder; authored alternatives/references; empty when all visible nodes complete. All visible nodes remain startable.

Map due prompt fingerprints to the lesson activities that use them. Change `SetDepth` to persist journey depth; if current lesson becomes hidden, preserve progress/path and choose a visible recommendation.

- [ ] **Step 4: Run tests and commit**

Run: `go test ./internal/openings -run 'TestOpeningHome|TestRecommendation|TestOpeningServiceHome' -count=1`

Expected: PASS.

```bash
git add internal/openings/service_tree.go internal/openings/service.go internal/openings/views.go internal/openings/service_tree_test.go
git commit -m "feat: expose opening teaching tree progress"
```

---

### Task 6: Activity Sessions and Roadmap Checkpoints

**Files:**
- Create: `internal/openings/service_activity.go`
- Create: `internal/openings/service_decision.go`
- Modify: `internal/openings/service_steps.go`
- Modify: `internal/openings/service.go`
- Modify: `internal/openings/views.go`
- Create: `internal/openings/user_store_activity.go`
- Modify: `internal/openings/user_store_session.go`
- Modify: `internal/openings/user_store_progress.go`
- Modify: `internal/openings/user_store_review.go`
- Create: `internal/openings/service_activity_test.go`
- Modify: `internal/openings/service_test.go`
- Modify: `internal/openings/user_store_test.go`

**Interfaces:**
- Consumes: lesson activities, `ActivityIndex`, journey persistence, and existing prompt/review helpers.
- Produces: `OpeningActivityView`, `OpeningActivityResult`, `OpeningRoadmapCheckpoint`, and `Service.AdvanceActivity`.

- [ ] **Step 1: Write failing end-to-end service tests**

```go
func TestOpeningServiceCompletesActivitiesAndReturnsCheckpoint(t *testing.T) {
	ctx := context.Background(); fixture := newTreeServiceFixture(t)
	started, err := fixture.service.StartLesson(ctx,fixture.compiled.Pack.CourseID,"giuoco-plan")
	if err != nil || started.Current == nil || started.Current.Kind != ActivityConcept { t.Fatalf("started=%+v err=%v",started,err) }
	concept, err := fixture.service.AdvanceActivity(ctx,started.SessionID)
	if err != nil || concept.Session.Current.Kind != ActivityDecision { t.Fatalf("concept=%+v err=%v",concept,err) }
	decision, err := fixture.service.PlayMove(ctx,started.SessionID,"c2c3")
	if err != nil || decision.Session.Current.Kind != ActivityRecap { t.Fatalf("decision=%+v err=%v",decision,err) }
	done, err := fixture.service.AdvanceActivity(ctx,started.SessionID)
	if err != nil || done.Checkpoint == nil || done.Checkpoint.RecommendedLessonID != "two-knights-plan" { t.Fatalf("done=%+v err=%v",done,err) }
}
```

Add a restart test that pauses on `giuoco-c3-decision`, recreates the service, and resumes the same activity/FEN without replaying Concept. Add a Reference test proving optional sections appear under Deeper Analysis but are never inserted into the required cursor or completion count. Add transaction-failure tests for both lesson start and activity completion, verifying that session, progress, and journey either all persist or all remain unchanged.

- [ ] **Step 2: Run and verify failure**

Run: `go test ./internal/openings -run 'TestOpeningServiceCompletesActivities|TestOpeningServiceResumesExactActivity' -count=1`

Expected: FAIL because sessions still expose legacy steps.

- [ ] **Step 3: Define the activity/checkpoint contract**

```go
type OpeningActivityView struct {
	ActivityID string `json:"activityId"`; Kind ActivityKind `json:"kind"`; Title string `json:"title"`
	Instruction string `json:"instruction"`; Required bool `json:"required"`; VariationName string `json:"variationName,omitempty"`
	PositionID string `json:"positionId,omitempty"`; CurrentFEN string `json:"currentFen"`; Orientation Perspective `json:"orientation"`
	LegalMoves []string `json:"legalMoves"`; TeachingNoteTexts []string `json:"teachingNoteTexts"`; ReferenceNoteTexts []string `json:"referenceNoteTexts"`
	Comparison []ActivityLine `json:"comparison"`; Annotations []BoardAnnotation `json:"annotations"`; MovesToHere []domain.AppliedMove `json:"movesToHere"`
	ActivityNumber int `json:"activityNumber"`; ActivityTotal int `json:"activityTotal"`; CompletedIdeas int `json:"completedIdeas"`; RequiredIdeas int `json:"requiredIdeas"`
	HintLevel int `json:"hintLevel"`; CanReveal bool `json:"canReveal"`
	ReferenceSections []OpeningReferenceSection `json:"referenceSections"`
}

type OpeningReferenceSection struct {
	ActivityID string `json:"activityId"`; Title string `json:"title"`; Instruction string `json:"instruction"`
	PositionID string `json:"positionId,omitempty"`; NoteTexts []string `json:"noteTexts"`; Annotations []BoardAnnotation `json:"annotations"`
}

type OpeningRoadmapCheckpoint struct {
	CompletedLessonID string `json:"completedLessonId"`; Path []OpeningPathItem `json:"path"`; AvailableLessonIDs []string `json:"availableLessonIds"`
	RecommendedLessonID string `json:"recommendedLessonId,omitempty"`; RecommendedLessonTitle string `json:"recommendedLessonTitle,omitempty"`
	CompletedLessons int `json:"completedLessons"`; TotalLessons int `json:"totalLessons"`
}
```

Rename result to `OpeningActivityResult` with `ActivityCompleted bool` and optional `Checkpoint`. Keep `OpeningSummary` only for review completion.

- [ ] **Step 4: Start at the correct activity and persist atomically**

`StartLesson` loads activity progress, starts the first incomplete required activity, or the first required activity when restudying a completed node. Optional Reference activities never become the primary cursor; the view returns them as `ReferenceSections`. It stores journey lesson/activity/path/session.

Add a lesson-start boundary that creates the session, fills the journey's active-session/current-activity fields, and upserts the journey in one transaction. Keep the existing `CreateSession` path for reviews.

```go
func (s *UserStore) CreateLessonSession(
	ctx context.Context, seed SessionSeed, journey CourseJourney, now time.Time,
) (StoredSession, CourseJourney, error)
```

`AdvanceActivity` accepts Concept, Demonstration, Comparison, Recap, and Reference. Demonstration applies connected `MoveIDs` and returns authoritative frames. Build next session and journey first, then persist session, activity ID, lesson progress, and journey in one transaction. A failed transaction must not advance visible or stored state.

Add the atomic store boundary explicitly:

```go
type LessonActivityCompletion struct {
	Session StoredSession
	Journey CourseJourney
	ActivityID string
	RequiredActivityIDs []string
	Attempt *AttemptRecord
	SemanticFingerprint string
	Outcome ReviewOutcome
	Now time.Time
}

func (s *UserStore) CompleteLessonActivity(ctx context.Context, completion LessonActivityCompletion) error
```

`Attempt` is nil for passive activities. When non-nil, the same transaction also writes attempt metrics, prompt progress, and review scheduling.

- [ ] **Step 5: Generalize decision completion and retain review behavior**

Move lesson prompt behavior into `service_decision.go`. Lesson moves are allowed only for Decision; review uses its synthetic recall activity. Preserve legality, alternatives, retries, hints, reveal, attempt metrics, fingerprints, and scheduling. Lesson decision completion records one activity. Review completion advances the review cursor and returns the existing summary without a roadmap checkpoint.

When the last required lesson activity completes, mark the lesson session completed, clear journey active-session/current-activity fields, retain completed lesson/path, compute recommendation, and return a checkpoint.

- [ ] **Step 6: Add deterministic Moves to Here**

Implement breadth-first traversal from root position using authored outgoing move order and visible depth. Return the first connected path to the activity position as `domain.AppliedMove` frames. Root-position activities return an empty array; unreachable activities are already compiler errors.

- [ ] **Step 7: Run tests and commit**

Run: `go test ./internal/openings -run 'TestOpeningService|TestOpeningReview|TestOpeningCompletePrompt' -count=1 && go test ./... -count=1`

Expected: PASS; lessons return checkpoints and reviews remain separate.

```bash
git add internal/openings/service_activity.go internal/openings/service_decision.go internal/openings/service_steps.go internal/openings/service.go internal/openings/views.go internal/openings/user_store_activity.go internal/openings/user_store_session.go internal/openings/user_store_progress.go internal/openings/user_store_review.go internal/openings/service_activity_test.go internal/openings/service_test.go internal/openings/user_store_test.go
git commit -m "feat: run decision-point opening lessons"
```

---

### Task 7: Safe Activity-ID Rebase

**Files:**
- Modify: `internal/openings/service_rebase.go`
- Modify: `internal/openings/user_store_rebase.go`
- Modify: `internal/openings/service_rebase_test.go`
- Modify: `internal/openings/user_store_test.go`

**Interfaces:**
- Consumes: normalized activity IDs, journey, active generations, and fingerprints.
- Produces: exact activity rebase, explicit nearest checkpoint restart, and atomic session/journey/review revision.

- [ ] **Step 1: Write failing rebase cases**

```go
func TestOpeningRebaseKeepsMatchingActivityAndJourney(t *testing.T) {
	fixture := newTreeRebaseFixture(t)
	fixture.startAtActivity(t,"giuoco-plan","giuoco-c3-decision")
	fixture.importRevisionKeepingActivity(t,"giuoco-c3-decision")
	resumed, err := fixture.service.Resume(context.Background())
	if err != nil || resumed == nil || resumed.Current.ActivityID != "giuoco-c3-decision" { t.Fatalf("resumed=%+v err=%v",resumed,err) }
	journey, err := fixture.store.Journey(context.Background(),fixture.compiled.Pack.CourseID,DepthReference)
	if err != nil || journey.CurrentActivityID != "giuoco-c3-decision" { t.Fatalf("journey=%+v err=%v",journey,err) }
}
```

Add cases for a removed current activity, removed lesson, completed legacy lesson with new required v2 activities, and forced review-archive rollback.

- [ ] **Step 2: Run and verify failure**

Run: `go test ./internal/openings -run 'TestOpeningRebase|TestApplyCourseRevision' -count=1`

Expected: FAIL on activity/journey assertions.

- [ ] **Step 3: Rebase by stable ID and include journey in the transaction**

Resolve old current activity ID from normalized lesson plus `ActivityIndex`. Match new activity by ID before ordinal. Compatibility compares kind, position, prompt fingerprint, and demonstration move IDs. If missing, restart at the last compatible preceding activity; clear attempt and show preserved-progress notice.

Extend `CourseRevision` with `Journey *CourseJourney`. Validate matching course IDs and update session, journey, and review reconciliation in one SQL transaction. Forced review failure rolls back all three. Never clear non-null lesson `completed_at`.

- [ ] **Step 4: Run tests and commit**

Run: `go test ./internal/openings -run 'TestOpeningRebase|TestUserStoreCourseRevision|TestApplyCourseRevision' -count=1`

Expected: PASS.

```bash
git add internal/openings/service_rebase.go internal/openings/user_store_rebase.go internal/openings/service_rebase_test.go internal/openings/user_store_test.go
git commit -m "feat: rebase opening activity journeys"
```

---

### Task 8: Wails and TypeScript Contracts

**Files:**
- Modify: `normal_controller.go`
- Modify: `controllers_test.go`
- Modify: `frontend/src/lib/contracts/openings.ts`
- Modify: `frontend/src/lib/api/types.ts`
- Modify: `frontend/src/lib/api/production.ts`
- Modify: `frontend/src/lib/api/preview.ts`
- Modify: `frontend/src/lib/api.test.ts`
- Modify: `frontend/src/test-fakes.ts`
- Regenerate: `frontend/wailsjs/go/main/NormalController.d.ts`
- Regenerate: `frontend/wailsjs/go/main/NormalController.js`
- Regenerate: `frontend/wailsjs/go/models.ts`

**Interfaces:**
- Produces: strict `OpeningTeachingTree`, `OpeningActivityView`, `OpeningRoadmapCheckpoint`, and `advanceOpeningActivity` frontend API.

- [ ] **Step 1: Write failing decoder tests**

```ts
test('decodes opening tree and checkpoint', () => {
  const home = decodeOpeningHome(rawTreeHome)
  expect(home.courses[0].tree.rootLessonId).toBe('giuoco-plan')
  expect(home.courses[0].tree.nodes[0]).toMatchObject({ progress: 'in_progress', recommended: true })
  const result = decodeOpeningActivityResult(rawCheckpointResult)
  expect(result.checkpoint?.recommendedLessonId).toBe('two-knights-plan')
})
```

Add rejection cases for unknown activity kind, invalid node progress, missing checkpoint path, and activity number above total.

- [ ] **Step 2: Run and verify failure**

Run: `npm --prefix frontend test -- --run src/lib/api.test.ts --single-thread`

Expected: FAIL because v2 decoders do not exist.

- [ ] **Step 3: Mirror Go types exactly**

```ts
export type OpeningActivityKind = 'concept'|'demonstration'|'decision'|'comparison'|'recap'|'reference'
export type OpeningLessonEdgeKind = 'continuation'|'alternative'|'reference'
export type OpeningNodeProgress = 'available'|'in_progress'|'completed'
export type OpeningTeachingNode = {
  lessonId:string; chapterId:string; title:string; objective:string; minimumDepth:OpeningDepth
  progress:OpeningNodeProgress; completedActivities:number; requiredActivities:number
  recommended:boolean; reviewDue:boolean; visible:boolean
}
export type OpeningTeachingEdge = { edgeId:string; fromLessonId:string; toLessonId:string; ordinal:number; kind:OpeningLessonEdgeKind; label?:string; minimumDepth:OpeningDepth }
export type OpeningTeachingTree = { rootLessonId:string; nodes:OpeningTeachingNode[]; edges:OpeningTeachingEdge[] }
```

Mirror every activity/checkpoint/reference-section field from Task 6 and keep strict discriminated session/result decoding.

- [ ] **Step 4: Add activity-named API and regenerate bindings**

Add `NormalController.AdvanceOpeningActivity` calling `Service.AdvanceActivity`. Keep `AdvanceOpeningStep` as a thin one-release alias. Expose `advanceOpeningActivity` in `NormalAPI`; update production/preview implementations.

Run: `go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 generate module`

Expected: generated bindings contain `AdvanceOpeningActivity` and new view models.

- [ ] **Step 5: Run tests and commit**

Run: `go test . ./internal/openings -count=1 && npm --prefix frontend test -- --run src/lib/api.test.ts --single-thread && npm --prefix frontend run check`

Expected: PASS.

```bash
git add normal_controller.go controllers_test.go frontend/src/lib/contracts/openings.ts frontend/src/lib/api/types.ts frontend/src/lib/api/production.ts frontend/src/lib/api/preview.ts frontend/src/lib/api.test.ts frontend/src/test-fakes.ts frontend/wailsjs/go/main/NormalController.d.ts frontend/wailsjs/go/main/NormalController.js frontend/wailsjs/go/models.ts
git commit -m "feat: expose opening activity journey contracts"
```

---

### Task 9: Accessible Teaching-Tree Home

**Files:**
- Create: `frontend/src/components/openings/opening-tree.ts`
- Create: `frontend/src/components/openings/opening-tree.test.ts`
- Create: `frontend/src/components/openings/OpeningTeachingTree.svelte`
- Create: `frontend/src/components/openings/OpeningTeachingTree.test.ts`
- Modify: `frontend/src/components/openings/OpeningHub.svelte`
- Modify: `frontend/src/components/openings/OpeningHub.test.ts`
- Modify: `frontend/src/components/home/HomeHub.svelte`
- Modify: `frontend/src/components/home/HomeHub.test.ts`
- Modify: `frontend/src/styles/app.css`

**Interfaces:**
- Produces: pure `projectOpeningTree`, semantic nested tree, visual connectors, and Continue Learning hub action.

- [ ] **Step 1: Write failing projection/component tests**

```ts
export type OpeningTreeBranch = { node: OpeningTeachingNode; incoming?: OpeningTeachingEdge; children: OpeningTreeBranch[] }
export function projectOpeningTree(tree: OpeningTeachingTree): OpeningTreeBranch
```

Test authored child order, missing node, duplicate parent, edge labels, and visible/hidden-by-depth projection. Then:

```ts
test('renders semantic progress and starts any visible node', async () => {
  const { component } = render(OpeningTeachingTree,{tree:fakeOpeningTree}); const starts:string[]=[]
  component.$on('lesson',event=>starts.push(event.detail))
  expect(screen.getByRole('tree',{name:'Italian Game course roadmap'})).toBeInTheDocument()
  expect(screen.getByRole('treeitem',{name:/Prepare the centre.*Recommended.*1 of 3 ideas/i})).toBeInTheDocument()
  await fireEvent.click(screen.getByRole('button',{name:'Study Choose the quiet setup'}))
  expect(starts).toEqual(['two-knights-plan'])
})
```

- [ ] **Step 2: Run and verify failure**

Run: `npm --prefix frontend test -- --run src/components/openings/opening-tree.test.ts src/components/openings/OpeningTeachingTree.test.ts src/components/openings/OpeningHub.test.ts --single-thread`

Expected: FAIL because tree UI does not exist.

- [ ] **Step 3: Implement projection, semantic tree, and hub**

Build node/outgoing maps, validate one root/parent, and return nested branches in authored edge order. Render nested `<ul role="tree">`/`<ul role="group">`; keep every visible-node Study button in native tab order. Draw connectors from the same DOM structure. Include Complete, In progress, Recommended, Review due, Available, and Hidden at this depth labels. Hidden nodes remain visible as roadmap context but cannot start until the learner selects a sufficient depth.

Replace hub chapter rows with the tree. Continue Learning resumes exact work or starts recommendation. Keep Review Due and Explore Variations separate. Update HomeHub copy to `Continue your Italian course` or `Next: <title>`.

- [ ] **Step 4: Add responsive styles and test**

At 760px and below, stack branches, keep current path/immediate children readable, and avoid horizontal scrolling. Pair state colors with text/icons.

Run: `npm --prefix frontend test -- --run src/components/openings/OpeningTeachingTree.test.ts src/components/openings/OpeningHub.test.ts src/components/home/HomeHub.test.ts --single-thread && npm --prefix frontend run check`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/openings/opening-tree.ts frontend/src/components/openings/opening-tree.test.ts frontend/src/components/openings/OpeningTeachingTree.svelte frontend/src/components/openings/OpeningTeachingTree.test.ts frontend/src/components/openings/OpeningHub.svelte frontend/src/components/openings/OpeningHub.test.ts frontend/src/components/home/HomeHub.svelte frontend/src/components/home/HomeHub.test.ts frontend/src/styles/app.css
git commit -m "feat: show opening teaching tree"
```

---

### Task 10: Continuous Lesson and Checkpoint UI

**Files:**
- Create: `frontend/src/components/openings/OpeningPathContext.svelte`
- Create: `frontend/src/components/openings/OpeningPathContext.test.ts`
- Create: `frontend/src/components/openings/OpeningActivityContent.svelte`
- Create: `frontend/src/components/openings/OpeningActivityContent.test.ts`
- Create: `frontend/src/components/openings/OpeningRoadmapCheckpoint.svelte`
- Create: `frontend/src/components/openings/OpeningRoadmapCheckpoint.test.ts`
- Modify: `frontend/src/components/openings/opening-state.ts`
- Modify: `frontend/src/components/openings/opening-state.test.ts`
- Modify: `frontend/src/components/openings/opening-controller.ts`
- Modify: `frontend/src/components/openings/opening-controller.test.ts`
- Modify: `frontend/src/components/openings/OpeningLessonScreen.svelte`
- Modify: `frontend/src/components/openings/OpeningLessonScreen.test.ts`
- Modify: `frontend/src/components/chess/ChessBoard.svelte`
- Modify: `frontend/src/components/chess/ChessBoard.test.ts`
- Modify: `frontend/src/components/chess/chessground-adapter.ts`
- Modify: `frontend/src/components/chess/chessground-adapter.test.ts`
- Modify: `frontend/src/styles/chessground-theme.css`
- Modify: `frontend/src/components/app/NormalShell.svelte`
- Modify: `frontend/src/App.test.ts`

**Interfaces:**
- Produces: activity-aware state machine, path, replay controls, checkpoint Continue, and separate review summary.

- [ ] **Step 1: Write failing state and component tests**

```ts
test('acknowledges lesson completion into checkpoint', () => {
  const state = completeOpeningActivity(animatingDecision,1,afterC3,rawCheckpointResult)
  expect(acknowledgeOpeningActivity(state)).toEqual({phase:'checkpoint',session:completedLessonSession,checkpoint:fakeRoadmapCheckpoint})
})
```

Test path text; passive Continue; decision board/hints; read-only square and arrow annotations; two comparison lines; collapsed Deeper Analysis; Moves to Here replay; demonstration replay; checkpoint primary Continue; Stop for Now; and review summary Back to Course.

- [ ] **Step 2: Run and verify failure**

Run: `npm --prefix frontend test -- --run src/components/openings/opening-state.test.ts src/components/openings/opening-controller.test.ts src/components/openings/OpeningLessonScreen.test.ts src/components/chess/ChessBoard.test.ts src/components/chess/chessground-adapter.test.ts --single-thread`

Expected: FAIL because activity/checkpoint phases do not exist.

- [ ] **Step 3: Generalize state/controller**

Use phases `passive`, `ready`, `requesting`, `animating`, `activity-complete`, `checkpoint`, `summary`, `restart-required`, and `failed`. Concept/Demonstration/Comparison/Recap/Reference are passive; Decision is ready. Preserve authoritative FEN reconciliation and keep the entire pending `OpeningActivityResult` until acknowledgment.

Store last authoritative frames in controller view. `replayMovesToHere()` and `replayDemonstration()` reuse the board port, reduced-motion branch, cancellation token, and recovery warning; replay never calls the backend or changes progress.

- [ ] **Step 4: Add read-only board annotations without coupling chess UI to course contracts**

Define a generic chess-component type in `chessground-adapter.ts` rather than importing the openings contract:

```ts
export type BoardAnnotation =
  | { kind:'square'; from:Key }
  | { kind:'arrow'; from:Key; to:Key }
```

Add optional `annotations` to `BoardInteraction` and `ChessBoard`. Map arrows to Chessground `drawable.autoShapes` with a fixed course brush while keeping `drawable.enabled = false`; map squares to an `opening-annotation` custom square class in `chessground-theme.css`. The opening screen translates validated `OpeningBoardAnnotation` values into this generic prop. Keep decision hints, wrong-move feedback, and lesson annotations as distinct layers so one cannot erase another.

Tests must prove arrows remain visible while learner drawing stays disabled, square annotations compose with existing marker classes, annotations update through `configure`, and an empty list removes stale shapes/classes.

- [ ] **Step 5: Add focused components and seamless shell routing**

`OpeningPathContext` renders an ordered breadcrumb. `OpeningActivityContent` renders teaching text/comparisons and the lesson's optional `ReferenceSections` inside collapsed Deeper Analysis. Expanding a reference section does not mutate the main activity cursor or node completion. `OpeningRoadmapCheckpoint` dispatches Continue with lesson ID, Tree, or Home.

Add events:

```ts
continue: { courseId:string; lessonId:string }
tree: void
home: { completed:boolean }
change: OpeningSessionView
persisted: OpeningSessionView
```

NormalShell routes Continue through existing lesson start while remaining on `opening-lesson`; Tree refreshes home and opens `openings`. Completing a lesson replaces only its active session, never course journey progress.

- [ ] **Step 6: Run tests and commit**

Run: `npm --prefix frontend test -- --run src/components/openings src/components/chess/ChessBoard.test.ts src/components/chess/chessground-adapter.test.ts src/App.test.ts --single-thread && npm --prefix frontend run check`

Expected: PASS; no primary journey copy refers to Step 1 of 5 or Back Home.

```bash
git add frontend/src/components/openings/OpeningPathContext.svelte frontend/src/components/openings/OpeningPathContext.test.ts frontend/src/components/openings/OpeningActivityContent.svelte frontend/src/components/openings/OpeningActivityContent.test.ts frontend/src/components/openings/OpeningRoadmapCheckpoint.svelte frontend/src/components/openings/OpeningRoadmapCheckpoint.test.ts frontend/src/components/openings/opening-state.ts frontend/src/components/openings/opening-state.test.ts frontend/src/components/openings/opening-controller.ts frontend/src/components/openings/opening-controller.test.ts frontend/src/components/openings/OpeningLessonScreen.svelte frontend/src/components/openings/OpeningLessonScreen.test.ts frontend/src/components/chess/ChessBoard.svelte frontend/src/components/chess/ChessBoard.test.ts frontend/src/components/chess/chessground-adapter.ts frontend/src/components/chess/chessground-adapter.test.ts frontend/src/styles/chessground-theme.css frontend/src/components/app/NormalShell.svelte frontend/src/App.test.ts
git commit -m "feat: continue opening lessons through checkpoints"
```

---

### Task 11: Authoring Docs, End-to-End Journey, and Verification

**Files:**
- Modify: `docs/operations/opening-course-authoring.md`
- Modify: `cmd/coursepack/main_test.go`
- Modify: `frontend/tests/test-backend.ts`
- Modify: `frontend/tests/openings.spec.ts`
- Modify: `scripts/test-backend-structure.test.mjs` only if the fake backend is split
- Modify: `README.md` only where it describes the fixed staircase

**Interfaces:**
- Produces: documented schema v2, synthetic multi-node acceptance, and green repository verification.

- [ ] **Step 1: Rewrite the failing Playwright flow**

Test importing v2 fixture, semantic roadmap, hidden-depth context, Reference depth, collapsed optional Reference material, Concept once, `c3` once, Recap, checkpoint, direct next lesson, pause/resume exact `d3` decision, review scheduling, depth switch without loss, explorer, and unchanged puzzle training.

```ts
await expect.poll(() => backendState<string[]>(page,'openingMoves')).toEqual(['c2c3','d2d3'])
```

- [ ] **Step 2: Run and verify failure**

Run: `npm --prefix frontend run test:e2e -- openings.spec.ts`

Expected: FAIL until fake backend uses v2 contracts.

- [ ] **Step 3: Update fake backend and authoring guide**

Return the exact two-node tree/activity/checkpoint used by Go fixture and persist a fake journey cursor for resume. Keep review and puzzles separate.

Document root `lessonEdges`, lesson `activities`, six activity kinds, one-parent tree, edge kinds, cumulative depth, required/optional activities, duplicate-decision rejection, stable-ID completion, v1 compatibility, validator commands, and external private-pack rule. Examples use only synthetic content.

Extend validator structural counts with `activities` and `lessonEdges`; test v1 normalized counts and v2 two-lesson/one-edge/four-activity counts.

- [ ] **Step 4: Run full verification**

```bash
gofmt -w internal/openings/*.go normal_controller.go controllers_test.go cmd/coursepack/*.go
go test ./... -count=1
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run test:e2e -- openings.spec.ts
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
git diff --check
git status --short
```

Expected: every command exits 0; Git shows no private `.ctcourse` outside reviewed synthetic fixtures.

- [ ] **Step 5: Commit and review**

```bash
git add docs/operations/opening-course-authoring.md cmd/coursepack/main_test.go frontend/tests/test-backend.ts frontend/tests/openings.spec.ts scripts/test-backend-structure.test.mjs README.md
git commit -m "test: verify opening teaching tree journey"
```

Use `superpowers:requesting-code-review` on the complete engine diff. Address blocking findings with `superpowers:receiving-code-review`, rerun the full verification matrix, and do not begin private Italian v2 authoring until the reusable engine passes review.
