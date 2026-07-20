# Opening Teaching Tree Quality Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the duplicated navigation, graph, and activity representations identified by the structural review while preserving existing Italian-course data and behavior.

**Architecture:** Preferences own depth, journeys own lesson/path continuity, and sessions own the active activity and board state. Backend responses project all navigation context consumed by explicit frontend course selection. Course activity rows use relational core fields plus a payload-only JSON document, and one pure tree-index builder serves validation and catalog loading.

**Tech Stack:** Go 1.24, SQLite migrations, Svelte 4, TypeScript, Vitest, Wails bindings.

## Global Constraints

- Preserve local-first privacy and existing imported course/review progress.
- Keep schema-v1 course packs importable during the compatibility window.
- Do not add another opening in this remediation.
- Use failing tests before each production change.
- Do not hand-edit generated Wails bindings; regenerate them after Go contract changes.

---

### Task 1: Canonical journey persistence

**Files:**
- Create: `internal/storage/migrations/user/006.sql`
- Modify: `internal/storage/sqlite_test.go`
- Modify: `internal/openings/user_store.go`
- Modify: `internal/openings/user_store_journey.go`
- Modify: `internal/openings/user_store_activity.go`
- Modify: `internal/openings/user_store_rebase.go`
- Modify: `internal/openings/service.go`
- Modify: `internal/openings/service_activity.go`
- Modify: `internal/openings/service_rebase.go`
- Test: `internal/openings/user_store_test.go`
- Test: `internal/openings/service_test.go`
- Test: `internal/openings/service_activity_test.go`

**Interfaces:**
- Produces: `CourseJourney{CourseID, CurrentLessonID, PathLessonIDs, CreatedAt, UpdatedAt}` and atomic migration of the effective existing depth into `opening_preferences`.

- [ ] **Step 1: Write failing storage and service tests.** Assert migration six, effective-depth preservation, a journey table without depth/activity/recommendation/session columns, and course summaries deriving current activity from the resumable session.
- [ ] **Step 2: Run `go test ./internal/storage ./internal/openings -run 'TestMigrations|TestUserStoreJourney|TestOpeningService' -count=1`.** Expected: failure on migration count/schema and the new projection assertions.
- [ ] **Step 3: Add migration 006 and reduce `CourseJourney`.** Rebuild `opening_course_journeys(course_id,current_lesson_id,path_lesson_ids_json,created_at,updated_at)` after copying journey depth to preferences with an upsert. Remove mirrored field reads/writes and make services use the selected preference plus resumable session.
- [ ] **Step 4: Run the focused Go tests.** Expected: PASS.

### Task 2: Authoritative activity rows and one tree index

**Files:**
- Modify: `internal/openings/catalog_insert.go`
- Modify: `internal/openings/catalog_read_rows.go`
- Modify: `internal/openings/compiler_teaching_tree.go`
- Test: `internal/openings/catalog_test.go`
- Test: `internal/openings/compiler_test.go`

**Interfaces:**
- Produces: payload-only activity JSON helpers and `buildTeachingTreeIndex(CompiledCourse) teachingTreeIndex` reused by compile and load paths.

- [ ] **Step 1: Write failing round-trip corruption tests.** Change a stored core activity column and assert the loader uses or rejects it rather than silently trusting stale JSON. Add an index-builder test covering deterministic children, parents, and root.
- [ ] **Step 2: Run `go test ./internal/openings -run 'TestSQLiteCatalog.*Activity|TestCompile.*TeachingTree' -count=1`.** Expected: failure because JSON currently owns all activity fields and compilation builds the index twice.
- [ ] **Step 3: Store only activity-specific fields in JSON and scan core columns.** Reconstruct `LessonActivity` from authoritative columns plus the decoded payload. Extract and reuse one pure tree-index builder.
- [ ] **Step 4: Run the focused Go tests.** Expected: PASS.

### Task 3: Backend-owned session navigation contract

**Files:**
- Modify: `internal/openings/views.go`
- Modify: `internal/openings/service_activity.go`
- Test: `internal/openings/service_activity_test.go`
- Modify: `frontend/src/lib/contracts/openings.ts`
- Test: `frontend/src/lib/api.test.ts`
- Regenerate: `frontend/wailsjs/go/models.ts`

**Interfaces:**
- Produces: every `OpeningSessionView` contains `courseTitle` and `path: OpeningPathItem[]` built by the backend.

- [ ] **Step 1: Add failing Go and decoder tests.** Assert an active, completed, review, and restart-required session each carries the course title and canonical path.
- [ ] **Step 2: Run focused Go and frontend contract tests.** Expected: failure because session navigation fields are absent.
- [ ] **Step 3: Populate and decode the fields, then regenerate Wails bindings.** Use `openingPathItems(course, teachingPathLessonIDs(course, session.LessonID))` in `sessionView`.
- [ ] **Step 4: Re-run focused tests.** Expected: PASS.

### Task 4: Explicit frontend course selection

**Files:**
- Modify: `frontend/src/components/app/NormalShell.svelte`
- Modify: `frontend/src/components/home/HomeHub.svelte`
- Modify: `frontend/src/components/openings/OpeningHub.svelte`
- Modify: `frontend/src/components/openings/OpeningLessonScreen.svelte`
- Test: `frontend/src/App.test.ts`
- Test: `frontend/src/components/home/HomeHub.test.ts`
- Test: `frontend/src/components/openings/OpeningHub.test.ts`
- Test: `frontend/src/components/openings/OpeningLessonScreen.test.ts`

**Interfaces:**
- Produces: `selectedOpeningCourseId`, course-aware hub events, and lesson rendering that consumes `session.courseTitle` and `session.path`.

- [ ] **Step 1: Write failing multi-course component tests.** Assert the chosen course—not array position—is shown, dispatch includes course ID, and lesson headers use the response title/path.
- [ ] **Step 2: Run the four focused frontend tests.** Expected: failure on first-course assumptions and hard-coded Italian copy.
- [ ] **Step 3: Implement explicit selection and remove local projection.** Delete `syncOpeningResume` and `openingPath`; refresh authoritative home state after persistence transitions; pass the session directly to the lesson screen.
- [ ] **Step 4: Re-run focused frontend tests.** Expected: PASS.

### Task 5: Repository verification

**Files:**
- Modify only files needed to correct failures caused by Tasks 1-4.

**Interfaces:**
- Produces: a clean, generated, release-ready branch.

- [ ] **Step 1: Run `gofmt -w` on changed Go files and regenerate Wails bindings.**
- [ ] **Step 2: Run `go test ./... -count=1`, the complete frontend unit suite, `npm --prefix frontend run check`, opening Playwright tests, frontend build, license verification, `go vet ./...`, and `git diff --check`.** Expected: every command exits zero.
- [ ] **Step 3: Inspect `git diff --stat`, `git diff`, and `git status --short` for accidental generated/private artifacts.** Expected: only remediation files and regenerated contracts are changed.
