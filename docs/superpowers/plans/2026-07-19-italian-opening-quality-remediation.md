# Italian Opening Quality Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate the six privacy, transaction, state-model, controller-runtime, frontend-boundary, and import-ownership findings without changing the Italian opening product scope or public wire behavior.

**Architecture:** Release verification uses an exact path-and-digest fixture allowlist. Opening progress uses one user-database revision transaction and explicit nested session concepts. Shared Go import infrastructure and shared frontend decoder/board infrastructure each have one canonical owner, while domain state machines remain separate.

**Tech Stack:** Go language level 1.25 with release toolchain 1.26.4, SQLite, Wails v2.12.0, TypeScript 4.6, Svelte 3.59.2, Vitest 0.34, Playwright.

## Global Constraints

- Follow red-green-refactor for every behavior change and observe each specified failure before production edits.
- Keep the private Italian course external; only the reviewed synthetic fixture may be tracked.
- Preserve the current product scope: one White-side Italian course with quick, standard, and reference depths.
- Keep the course catalog and user-progress SQLite databases separate.
- Preserve public controller method names, Wails JSON field names, frontend behavior, and private course-pack schema.
- Do not turn puzzle and opening domain state machines into a generic state machine.
- Use composition for shared frontend infrastructure.
- Use `apply_patch` for source edits and preserve unrelated user changes.
- Do not hand-edit generated Wails code. Regenerate only if a public Go wire signature actually changes.
- Keep every owned Go, TypeScript, and Svelte file below 1,000 physical lines.
- Each task must finish with its focused suite green and a dedicated commit.

**Design reference:** `docs/superpowers/specs/2026-07-19-italian-opening-quality-remediation-design.md`

---

### Task 1: Enforce an exact, hashed synthetic-course allowlist

**Files:**

- Modify: `scripts/release-policy.mjs`
- Modify: `scripts/verify-release.mjs`
- Modify: `scripts/verify-release.test.mjs`

**Interfaces:**

```js
export const COURSE_FIXTURE_SHA256 = Object.freeze({
  'internal/openings/testdata/mini.ctcourse':
    'bcaee75d5f5dafcae827baf5a9baaef28d6a931b906314fe52186271b5af2054',
})

export function assertCourseFixtureBoundary(
  paths,
  {
    root = process.cwd(),
    readFixture = (filename) => readFileSync(filename),
  } = {},
) {}
```

- [ ] **Step 1: Replace the permissive release test with failing boundary cases**

In `scripts/verify-release.test.mjs`, derive `repositoryRoot` with `fileURLToPath(new URL('..', import.meta.url))`. Add one passing call for the real `mini.ctcourse` fixture and four rejecting calls:

```js
assert.doesNotThrow(() => assertCourseFixtureBoundary(
  ['internal/openings/testdata/mini.ctcourse', 'README.md'],
  { root: repositoryRoot },
))
assert.throws(
  () => assertCourseFixtureBoundary(
    ['internal/openings/testdata/private-source.ctcourse'],
    { root: repositoryRoot },
  ),
  /unreviewed opening course fixture/,
)
assert.throws(
  () => assertCourseFixtureBoundary(
    ['internal/openings/testdata/private-source.CTCOURSE'],
    { root: repositoryRoot },
  ),
  /unreviewed opening course fixture/,
)
assert.throws(
  () => assertCourseFixtureBoundary(
    ['internal/openings/testdata/..\/testdata\/mini.ctcourse'],
    { root: repositoryRoot },
  ),
  /non-canonical opening course path/,
)
assert.throws(
  () => assertCourseFixtureBoundary(
    ['internal/openings/testdata/mini.ctcourse'],
    { root: repositoryRoot, readFixture: () => Buffer.from('changed') },
  ),
  /opening course fixture digest differs/,
)
```

- [ ] **Step 2: Run the release-policy test and verify RED**

Run:

```bash
node --test scripts/verify-release.test.mjs
```

Expected: the arbitrary testdata and uppercase-extension cases are accepted, and the digest case does not fail.

- [ ] **Step 3: Implement fail-closed path and digest checks**

Import `createHash` from `node:crypto` and `readFileSync` from `node:fs`. For every case-insensitive `.ctcourse` path:

1. Require the original slash-separated Git path to equal `path.posix.normalize(filename)` and reject absolute or parent-relative paths.
2. Look up the exact path in `COURSE_FIXTURE_SHA256`.
3. Read `path.join(root, filename)` through `readFixture`.
4. Calculate lowercase SHA-256 and compare it with the allowlist.

Use these exact error expressions:

```js
throw new Error('non-canonical opening course path: ' + filename)
throw new Error('unreviewed opening course fixture: ' + filename)
throw new Error('opening course fixture digest differs: ' + filename)
```

Extend `assertRequiredTrackedFiles` options with `courseFixtureRoot` and `readCourseFixture` and delegate both to `assertCourseFixtureBoundary`. In `verifyRelease`, pass `buildRoot` as `courseFixtureRoot` so verification hashes the isolated tagged input, not ambient checkout bytes.

- [ ] **Step 4: Verify GREEN and commit**

```bash
node --test scripts/verify-release.test.mjs
git diff --check
git add scripts/release-policy.mjs scripts/verify-release.mjs scripts/verify-release.test.mjs
git commit -m "fix: enforce synthetic course fixture boundary"
```

Expected: 29 or more release-policy tests pass, including all new privacy cases.

---

### Task 2: Make internal/importing the canonical shared import owner

**Files:**

- Create: `internal/importing/inspection.go`
- Create: `internal/importing/inspection_test.go`
- Modify: `internal/openings/importer.go`
- Modify: `internal/openings/importer_test.go`
- Modify: `internal/puzzles/collection_importer.go`
- Modify: `internal/puzzles/collection_importer_test.go`
- Modify: `internal/puzzles/canonical_json.go`
- Modify: `internal/puzzles/canonical_json_test.go`
- Modify: `internal/puzzles/full_import_test.go`
- Modify: `internal/puzzles/lichess_importer.go`
- Modify: `internal/puzzles/lichess_importer_test.go`
- Modify: `internal/puzzles/lichess_generation_test.go`
- Modify: `internal/puzzles/linear_fen_uci.go`
- Modify: `internal/puzzles/linear_fen_uci_test.go`
- Modify: `internal/puzzles/lucas_fns.go`
- Modify: `internal/puzzles/lucas_fns_test.go`
- Modify: `internal/puzzles/multi_format_import_test.go`
- Modify: `internal/puzzles/tactical_pgn.go`
- Modify: `internal/puzzles/tactical_pgn_test.go`
- Modify: `normal_controller.go`
- Modify: `app_test.go`
- Modify: `internal/app/services_test.go`
- Modify: `internal/app/task5_startup_red_test.go`
- Modify: `internal/importjob/service_cleanup_test.go`
- Modify: `internal/importjob/service_lifecycle_test.go`
- Modify: `internal/importjob/service_test_support_test.go`

**Interfaces:**

```go
func NormalizePath(importPath, subject string) (string, error)
func CompareInspection(current, expected Inspection, subject string) error
```

- [ ] **Step 1: Add failing shared-helper tests**

Create table-driven tests in `internal/importing/inspection_test.go`:

```go
func TestCompareInspectionReportsSubjectAndChangedField(t *testing.T) {
    base := Inspection{
        Path: "/tmp/a.ctcourse", Filename: "a.ctcourse",
        Format: "coursepack", SourceID: "italian-white",
        SourceIDOrigin: SourceIDEmbedded, SourceName: "Italian",
        URL: "https://example.invalid/source", Attribution: "Reference",
    }
    changed := base
    changed.SourceID = "other"
    err := CompareInspection(changed, base, "course import")
    if err == nil || !strings.Contains(err.Error(), "course import source ID changed after inspection") {
        t.Fatalf("CompareInspection() error = %v", err)
    }
}

func TestNormalizePathRequiresSubjectAndResolvesSymlink(t *testing.T) {
    if _, err := NormalizePath(" ", "puzzle import"); err == nil ||
        err.Error() != "puzzle import path is required" {
        t.Fatalf("empty path error = %v", err)
    }
    directory := t.TempDir()
    target := filepath.Join(directory, "target.txt")
    if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
        t.Fatal(err)
    }
    link := filepath.Join(directory, "link.txt")
    if err := os.Symlink(target, link); err != nil {
        t.Fatal(err)
    }
    got, err := NormalizePath(link, "puzzle import")
    if err != nil || got != target {
        t.Fatalf("NormalizePath() = %q, %v; want %q", got, err, target)
    }
}
```

- [ ] **Step 2: Run the importing tests and verify RED**

Run: `go test ./internal/importing -run 'Test(CompareInspection|NormalizePath)' -count=1`

Expected: compile failure because both helpers are undefined.

- [ ] **Step 3: Implement the canonical helpers**

`NormalizePath` trims input, returns `<subject> path is required` for empty input, obtains an absolute clean path, and replaces it with a clean symlink target when `filepath.EvalSymlinks` succeeds.

`CompareInspection` compares exactly these identity fields in order: path, filename, format, source ID, source ID origin, source name, source URL, source attribution. It returns:

```go
fmt.Errorf("%s %s changed after inspection: got %q, want %q",
    subject, field.name, field.current, field.want)
```

- [ ] **Step 4: Migrate puzzle and opening callers**

Replace every `normalizeImportPath` call with `importing.NormalizePath(path, "puzzle import")` and every `normalizeCourseImportPath` call with `importing.NormalizePath(path, "course import")`. Replace both local comparison functions with `importing.CompareInspection` using the same subjects. Delete the four local helper/error functions and their now-unused `filepath` imports.

Update the direct normalization assertion in `lichess_generation_test.go` to call the shared helper. Preserve all existing stale-inspection error wording.

In `normal_controller.go` and every cross-package app/import-job test listed above, import `chess-trainer/internal/importing` and replace:

- `puzzles.ImportInspection` with `importing.Inspection`;
- `puzzles.Progress` with `importing.Progress`;
- `puzzles.ProgressSink` with `importing.ProgressSink`.

Inside `internal/puzzles`, replace `ImportInspection`, `Progress`, and `ProgressSink` with their `importing` package names in the production adapters and tests listed above. Remove all three exported aliases from `collection_importer.go`. This is a mechanical ownership migration only; do not change JSON tags, format constants, adapter behavior, or progress ordering.

- [ ] **Step 5: Verify GREEN and commit**

```bash
gofmt -w internal/importing internal/openings/importer.go internal/openings/importer_test.go internal/puzzles normal_controller.go app_test.go internal/app/services_test.go internal/app/task5_startup_red_test.go internal/importjob/service_cleanup_test.go internal/importjob/service_lifecycle_test.go internal/importjob/service_test_support_test.go
go test ./internal/importing ./internal/openings ./internal/puzzles ./internal/importjob ./internal/app . -count=1
rg -n 'normalize(Course)?ImportPath|compare(Course)?ImportInspection|puzzles\.(ImportInspection|Progress|ProgressSink)|type (ImportInspection|Progress|ProgressSink) =' internal normal_controller.go app_test.go
git add internal/importing internal/openings/importer.go internal/openings/importer_test.go internal/puzzles normal_controller.go app_test.go internal/app/services_test.go internal/app/task5_startup_red_test.go internal/importjob/service_cleanup_test.go internal/importjob/service_lifecycle_test.go internal/importjob/service_test_support_test.go
git commit -m "refactor: centralize shared import inspection"
```

Expected: tests pass and the final `rg` returns no matches.

---

### Task 3: Make course revision writes atomic and home read-only

**Files:**

- Modify: `internal/openings/user_store.go`
- Modify: `internal/openings/user_store_rebase.go`
- Modify: `internal/openings/user_store_test.go`
- Modify: `internal/openings/service.go`
- Modify: `internal/openings/service_rebase.go`
- Modify: `internal/openings/service_test.go`
- Modify: `internal/openings/service_rebase_test.go`

**Interfaces:**

```go
type CourseRevision struct {
    CourseID           string
    PromptFingerprints map[string]string
    SessionRebase      *SessionRebase
    Now                time.Time
}

type SessionRebase struct {
    PreviousGenerationID string
    Session              StoredSession
}

func (s *UserStore) ApplyCourseRevision(
    ctx context.Context,
    revision CourseRevision,
) error
```

- [ ] **Step 1: Add the failing atomic-rollback test**

Replace the independent rebase/reconcile store tests with `TestApplyCourseRevisionRollsBackSessionWhenReviewArchiveFails`. Create a session in generation 1 and an active stale review. Install this SQLite trigger:

```sql
CREATE TRIGGER fail_opening_review_archive
BEFORE UPDATE OF status ON opening_review_state
WHEN NEW.status = 'archived'
BEGIN
  SELECT RAISE(ABORT, 'forced review archive failure');
END
```

Call `ApplyCourseRevision` with generation 2 and no active fingerprints. Require an error containing `forced review archive failure`, then assert the loaded session still has generation 1 and the review remains active.

- [ ] **Step 2: Add the failing read-only home test**

In `service_test.go`, add `TestOpeningServiceHomeIsReadOnlyForStaleReviews`. Insert a due review with an obsolete fingerprint and install the same rejecting trigger. Call `fixture.service.Home(ctx)`. Require success and `DueReviews == 0`. This test must fail because `courseSummary` currently attempts to archive the stale row.

- [ ] **Step 3: Run focused tests and verify RED**

```bash
go test ./internal/openings -run 'Test(ApplyCourseRevisionRollsBack|OpeningServiceHomeIsReadOnly)' -count=1
```

Expected: compile failure for `ApplyCourseRevision`, followed by a trigger failure from `Home` once the store API exists.

- [ ] **Step 4: Implement one user-database transaction**

Refactor `user_store_rebase.go` so `ApplyCourseRevision` validates the course ID, timestamp, fingerprints, and optional rebase before opening one transaction. Extract:

```go
func rebaseSessionTx(
    ctx context.Context,
    tx *sql.Tx,
    rebase SessionRebase,
    now time.Time,
) error

func reconcileReviewsTx(
    ctx context.Context,
    tx *sql.Tx,
    courseID string,
    activeFingerprints map[string]string,
) error
```

The exported method calls `rebaseSessionTx` first when present, then `reconcileReviewsTx`, then commits. Preserve `requireOneSessionRow` optimistic conflict behavior. Return stage-wrapped errors `rebase opening session: ...` and `reconcile opening reviews: ...`. Delete exported `RebaseSession` and `ReconcileReviews` so partial orchestration is impossible.

- [ ] **Step 5: Migrate every service revision path**

Add a private service helper:

```go
func (s *Service) applyCourseRevision(
    ctx context.Context,
    course CompiledCourse,
    rebase *SessionRebase,
) error {
    return s.store.ApplyCourseRevision(ctx, CourseRevision{
        CourseID: course.Pack.CourseID,
        PromptFingerprints: coursePromptFingerprints(course),
        SessionRebase: rebase,
        Now: s.now().UTC(),
    })
}
```

Use it for current-generation reconciliation, compatible lesson rebases, review rebases, lesson restart, and review restart. For review restart, query due rows and filter them against the active course before constructing the new session, then atomically rebase and reconcile. Do not reconcile before computing the queue.

Delete `persistReviewRebase`. Remove reconciliation from `courseSummary`; retain its existing fingerprint and depth filtering so home counts remain correct without writes.

- [ ] **Step 6: Verify no split writes remain**

```bash
gofmt -w internal/openings/user_store.go internal/openings/user_store_rebase.go internal/openings/user_store_test.go internal/openings/service.go internal/openings/service_rebase.go internal/openings/service_test.go internal/openings/service_rebase_test.go
go test ./internal/openings -count=1
rg -n 'RebaseSession|ReconcileReviews|persistReviewRebase' internal/openings --glob '*.go'
```

Expected: tests pass; `rg` finds no callable APIs or sequential service writes.

- [ ] **Step 7: Commit**

```bash
git add internal/openings/user_store.go internal/openings/user_store_rebase.go internal/openings/user_store_test.go internal/openings/service.go internal/openings/service_rebase.go internal/openings/service_test.go internal/openings/service_rebase_test.go
git commit -m "fix: apply opening course revisions atomically"
```

---

### Task 4: Split persisted session state and require explicit attempts

**Files:**

- Modify: `internal/openings/user_store.go`
- Modify: `internal/openings/user_store_session.go`
- Modify: `internal/openings/user_store_review.go`
- Modify: `internal/openings/user_store_progress.go`
- Modify: `internal/openings/user_store_test.go`
- Modify: `internal/openings/service.go`
- Modify: `internal/openings/service_steps.go`
- Modify: `internal/openings/service_rebase.go`
- Modify: `internal/openings/service_test.go`
- Modify: `internal/openings/service_rebase_test.go`

**Interfaces:**

```go
type PositionState struct {
    PositionID    string   `json:"positionId"`
    CurrentFEN    string   `json:"currentFen"`
    PlayedMoveIDs []string `json:"playedMoveIds"`
}

type AttemptState struct {
    AttemptID         string    `json:"attemptId"`
    PromptID          string    `json:"promptId"`
    StartedAt         time.Time `json:"startedAt"`
    HintLevel         int       `json:"hintLevel"`
    IncorrectMoves    int       `json:"incorrectMoves"`
    AlternativesTried int       `json:"alternativesTried"`
    HintsUsed         int       `json:"hintsUsed"`
    Revealed          bool      `json:"revealed"`
}

type ReviewCursor struct {
    PromptIDs []string `json:"promptIds"`
    Index     int      `json:"index"`
}

type SessionSummary struct {
    CompletedPrompts   int `json:"completedPrompts"`
    PositionsRecalled  int `json:"positionsRecalled"`
    BranchesRecognized int `json:"branchesRecognized"`
    Retried            int `json:"retried"`
    UsedHint           int `json:"usedHint"`
    Revealed           int `json:"revealed"`
}

type RestartCheckpoint struct {
    StepIndex int `json:"stepIndex"`
}

type SessionState struct {
    Position PositionState      `json:"position"`
    Attempt  *AttemptState      `json:"attempt,omitempty"`
    Review   *ReviewCursor      `json:"review,omitempty"`
    Summary  SessionSummary     `json:"summary"`
    Restart  *RestartCheckpoint `json:"restart,omitempty"`
}

type AttemptRecord struct {
    AttemptID, PromptID               string
    StartedAt                         time.Time
    IncorrectMoves, AlternativesTried int
    HintsUsed                         int
    Revealed                          bool
}

type PromptCompletion struct {
    Session             StoredSession
    Attempt             AttemptRecord
    SemanticFingerprint string
    Outcome             ReviewOutcome
    CompletedStepIDs    []string
}
```

- [ ] **Step 1: Add failing state-invariant tests**

Add table cases around `validateStoredSession`:

- lesson plus non-nil review cursor returns `lesson session cannot carry a review cursor`;
- review plus nil review cursor returns `review session requires a review cursor`;
- restart plus non-nil attempt returns `restart-required session cannot carry an attempt`;
- negative review index or attempt counters return their current field-specific errors.

Add `TestAttemptRecordRequiresActiveAttempt` around the new `attemptRecord` helper and require a nil attempt to return `active opening prompt requires an attempt`. Change `TestOpeningCompletePromptUpdatesAttemptReviewProgressAndSessionAtomically` to pass a concrete `AttemptRecord`. Add a case with the zero value and require `opening attempt ID is required`.

- [ ] **Step 2: Run focused tests and verify RED**

Run: `go test ./internal/openings -run 'Test(SessionState|OpeningCompletePrompt)' -count=1`

Expected: compile failures for the nested types and required `Attempt` field.

- [ ] **Step 3: Introduce nested state and migrate field access mechanically**

Use this exact mapping throughout service and tests:

| Old | New |
|---|---|
| `state.PositionID`, `CurrentFEN`, `PlayedMoveIDs` | `state.Position.PositionID`, `CurrentFEN`, `PlayedMoveIDs` |
| `state.AttemptID` through `state.Revealed` | `state.Attempt.<field>` after a nil check |
| `state.ReviewPromptIDs`, `ReviewIndex` | `state.Review.PromptIDs`, `state.Review.Index` |
| completion counters | `state.Summary.<field>` |
| `RestartStepIndex` | `state.Restart.StepIndex` |

`stateForLessonStep` and `stateForReviewPrompt` construct `PositionState` and create an `AttemptState` only for prompt-bearing steps. Review constructors require a `ReviewCursor`. `resetAttemptState` sets `Attempt = nil` without clearing position, review cursor, or summary.

Update `validateStoredSession` to enforce the specified mode/status invariants before encoding. Because the feature is unmerged, encode only the new nested JSON shape; do not add a legacy decoder.

- [ ] **Step 4: Make attempt persistence explicit**

Add:

```go
func attemptRecord(state *AttemptState) (AttemptRecord, error) {
    if state == nil {
        return AttemptRecord{}, errors.New("active opening prompt requires an attempt")
    }
    return AttemptRecord{
        AttemptID: state.AttemptID, PromptID: state.PromptID,
        StartedAt: state.StartedAt, IncorrectMoves: state.IncorrectMoves,
        AlternativesTried: state.AlternativesTried,
        HintsUsed: state.HintsUsed, Revealed: state.Revealed,
    }, nil
}
```

In `completePrompt`, capture the record before advancing or clearing `Session.State.Attempt` and pass it by value to `PromptCompletion`. In `CompletePrompt` and validation, read only `completion.Attempt`. Delete `AttemptState *SessionState` and `promptCompletionAttempt`.

- [ ] **Step 5: Centralize lesson progress SQL**

Move the existing `opening_lesson_progress` upsert into:

```go
func upsertLessonProgress(
    ctx context.Context,
    tx *sql.Tx,
    courseID, lessonID string,
    completedStepIDs []string,
    now time.Time,
) error
```

Call it only from `CompletePrompt` when `CompletedStepIDs` is non-empty. Delete the exported `CompleteLesson` method. Rewrite `TestOpeningLessonProgressProjectsStableStepIDs` to complete a real final prompt through `CompletePrompt`, then retain its original inserted/shortened step-ID projection assertions.

- [ ] **Step 6: Verify GREEN and commit**

```bash
gofmt -w internal/openings/user_store.go internal/openings/user_store_session.go internal/openings/user_store_review.go internal/openings/user_store_progress.go internal/openings/user_store_test.go internal/openings/service.go internal/openings/service_steps.go internal/openings/service_rebase.go internal/openings/service_test.go internal/openings/service_rebase_test.go
go test ./internal/openings -count=1
rg -n 'AttemptState \*SessionState|promptCompletionAttempt|func \(s \*UserStore\) CompleteLesson|State\.(PositionID|CurrentFEN|ReviewPromptIDs|ReviewIndex|AttemptID|RestartStepIndex)' internal/openings --glob '*.go'
git add internal/openings
git commit -m "refactor: model opening session state explicitly"
```

Expected: tests pass and the final `rg` returns no old flat-state or duplicate-completion APIs.

---

### Task 5: Split frontend contracts and production/preview APIs

**Files:**

- Create: `frontend/src/lib/contracts/decoder.ts`
- Create: `frontend/src/lib/contracts/application.ts`
- Create: `frontend/src/lib/contracts/imports.ts`
- Create: `frontend/src/lib/contracts/puzzles.ts`
- Create: `frontend/src/lib/contracts/openings.ts`
- Create: `frontend/src/lib/api/types.ts`
- Create: `frontend/src/lib/api/production.ts`
- Create: `frontend/src/lib/api/preview.ts`
- Modify: `frontend/src/lib/api-contract.ts`
- Modify: `frontend/src/lib/opening-explorer-contract.ts`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/lib/api.test.ts`

**Interfaces:**

- `contracts/decoder.ts` is the sole owner of `record`, `string`, `optionalString`, `number`, integer decoders, `boolean`, `array`, `enumeration`, and `numberRecord`.
- `api/types.ts` owns `PracticeRequest`, `BackupAPI`, `NormalAPI`, `RecoveryAPI`, and `ApplicationAPI`.
- `api/production.ts` exports `loadProductionApplicationAPI()`.
- `api/preview.ts` exports `loadPreviewApplicationAPI()` and `resetPreviewStateForTest()`.
- `api-contract.ts` and `api.ts` remain compatibility barrels.

- [ ] **Step 1: Add failing module-boundary imports**

At the top of `api.test.ts`, import:

```ts
import { decodeOpeningHome } from './contracts/openings'
import { decodeImportInspection } from './contracts/imports'
import { loadPreviewApplicationAPI } from './api/preview'
```

Add one test that decodes the existing opening-home fixture and import-inspection fixture through the new domain modules, then loads preview mode and requires `mode === 'normal'`. Use existing fixture literals rather than reduced duplicates.

- [ ] **Step 2: Run the API test and verify RED**

```bash
npm --prefix frontend test -- --run --single-thread src/lib/api.test.ts
```

Expected: module-resolution failures for `contracts/openings`, `contracts/imports`, and `api/preview`.

- [ ] **Step 3: Extract one decoder foundation**

Create `contracts/decoder.ts` by moving and exporting the decoder primitives from `api-contract.ts`. Make `positiveInteger` and `nonNegativeInteger` use the same finite-integer semantics currently enforced by `api-contract.ts`. Delete the duplicate primitives from `opening-explorer-contract.ts`.

Move types and decoders without behavior changes:

- application/build/profile/recovery/parent/practice to `application.ts`;
- import types and decoders to `imports.ts`;
- puzzle/session/move/hint types and decoders to `puzzles.ts`;
- all opening lesson, home, result, hint, evaluation, source, explorer types and decoders to `openings.ts`.

Use type-only imports between domain modules. `openings.ts` imports `AppliedMoveFrames` from `puzzles.ts`.

- [ ] **Step 4: Preserve compatibility barrels**

Reduce `api-contract.ts` to:

```ts
export * from './contracts/application'
export * from './contracts/imports'
export * from './contracts/puzzles'
export * from './contracts/openings'
```

Reduce `opening-explorer-contract.ts` to re-export the explorer types and decoder from `contracts/openings`. Existing application imports must continue compiling unchanged.

- [ ] **Step 5: Separate API interfaces and adapters**

Move API interfaces into `api/types.ts`. Move Wails imports, decoder adaptation, production bootstrap cache, and production normal/recovery objects into `api/production.ts`. Export only:

```ts
export async function loadProductionApplicationAPI(): Promise<ApplicationAPI>
```

Move all preview constants, mutable preview state, helpers, and adapters into `api/preview.ts`. Add:

```ts
export function resetPreviewStateForTest(): void {
  previewProfile = null
  previewSession = null
  previewIncorrect = new Set<number>()
  previewOpeningDepth = 'reference'
  previewOpeningSession = null
  previewOpeningStepIndex = 0
  previewOpeningHintLevel = 0
}

export async function loadPreviewApplicationAPI(): Promise<ApplicationAPI> {
  return { mode: 'normal', buildInfo: previewBuildInfo, api: previewNormalAPI }
}
```

Keep `api.ts` as the public barrel plus environment selector:

```ts
export * from './api-contract'
export type {
  ApplicationAPI,
  BackupAPI,
  NormalAPI,
  PracticeRequest,
  RecoveryAPI
} from './api/types'

export function loadApplicationAPI(): Promise<ApplicationAPI> {
  return typeof window !== 'undefined' && window.go
    ? loadProductionApplicationAPI()
    : loadPreviewApplicationAPI()
}
```

- [ ] **Step 6: Verify module sizes, behavior, and commit**

```bash
npm --prefix frontend test -- --run --single-thread src/lib/api.test.ts
npm --prefix frontend run check
wc -l frontend/src/lib/api-contract.ts frontend/src/lib/api.ts frontend/src/lib/contracts/*.ts frontend/src/lib/api/*.ts
rg -n '^function (record|string|optionalString|positiveInteger|nonNegativeInteger|array|enumeration)' frontend/src/lib
git add frontend/src/lib/api-contract.ts frontend/src/lib/opening-explorer-contract.ts frontend/src/lib/api.ts frontend/src/lib/api.test.ts frontend/src/lib/contracts frontend/src/lib/api
git commit -m "refactor: split frontend API boundaries"
```

Expected: API tests and checks pass; no new module exceeds 500 lines; each decoder primitive has one definition.

---

### Task 6: Share the interactive-board infrastructure runtime

**Files:**

- Create: `frontend/src/components/chess/interactive-board-runtime.ts`
- Create: `frontend/src/components/chess/interactive-board-runtime.test.ts`
- Modify: `frontend/src/components/openings/opening-controller.ts`
- Modify: `frontend/src/components/openings/opening-controller.test.ts`
- Modify: `frontend/src/components/puzzle/puzzle-controller.ts`
- Modify: `frontend/src/components/puzzle/puzzle-controller.test.ts`
- Modify: `frontend/src/components/puzzle/move-animation.test.ts`
- Delete: `frontend/src/components/puzzle/move-animation.ts`
- Delete: `frontend/src/components/puzzle/puzzle-effects.ts`
- Delete: `frontend/src/components/puzzle/request-owner.ts`

**Interfaces:**

```ts
export type InteractiveBoardPort = {
  setPosition(
    fen: string,
    lastMove?: [Square, Square],
    animate?: boolean
  ): void
}

export type InteractiveBoardHooks = {
  publishPosition(
    fen: string,
    lastMove: [Square, Square] | undefined,
    replaceBoard: boolean
  ): void
}

export class InteractiveBoardRuntime {
  constructor(effects: BoardEffects, hooks: InteractiveBoardHooks)
  mount(): { reducedMotion: boolean; soundMuted: boolean }
  destroy(): void
  attachBoard(board: InteractiveBoardPort | undefined): void
  startRequest(): RequestToken
  cancelRequest(): void
  isCurrent(token: RequestToken): boolean
  unlockFromPointer(): void
  unlockFromKeyboard(key: string): void
  toggleSound(): boolean
  playSound(kind: 'move' | 'capture' | 'correct' | 'incorrect'): void
  reconcilePosition(fen: string, signal: AbortSignal, animate: boolean): Promise<string>
  animationPort(): AnimationPort
  noteBoardError(message: string): void
  consumeWarnings(...messages: string[]): string
}
```

- [ ] **Step 1: Add failing runtime lifecycle tests**

Create a fake board, sound, and effects. Add separate tests proving:

1. `startRequest` aborts its predecessor and stale tokens fail `isCurrent`.
2. `destroy` aborts the current request, destroys sound, and detaches the board.
3. A throwing board causes `reconcilePosition` to call `publishPosition` with `replaceBoard === true` and return the current reconciliation warning.
4. `consumeWarnings('one', 'one', 'two')` returns `one two` once and clears pending warnings.
5. reduced-motion and sound-muted values come from effects/sound at mount.

- [ ] **Step 2: Run the runtime test and verify RED**

```bash
npm --prefix frontend test -- --run --single-thread src/components/chess/interactive-board-runtime.test.ts
```

Expected: module-resolution failure because the runtime does not exist.

- [ ] **Step 3: Implement the composition runtime**

Build the runtime from canonical `board-effects.ts`, `request-owner.ts`, `move-animation.ts`, and `sound.ts`. It owns the request owner, board, sound, mounted flag, sound-unlock flag, recovery flag, and pending warning.

`reconcilePosition` publishes the authoritative FEN before setting the board. On non-abort failure it publishes the same FEN with `replaceBoard = true` and returns `Board reconciliation failed: <message>. The saved position was restored.`

`animationPort` sets both published state and board position and delegates recovery through the same replace-board hook. `consumeWarnings` includes the pending warning, removes empty values, deduplicates while retaining order, clears recovery state, and returns one space-joined string.

- [ ] **Step 4: Delegate both controllers without sharing domain state**

Replace controller-owned request, board, sound, unlock, recovery, warning, `reconcilePosition`, `animationPort`, `recoverBoard`, and `isCurrent` fields/methods with one runtime instance.

For opening, its hook calls `updateStateFen(fen, lastMove, replaceBoard ? view.boardGeneration + 1 : view.boardGeneration)`. For puzzle, its hook updates the puzzle state's FEN/last move and increments `boardGeneration` only when `replaceBoard` is true.

Keep subscribers, current views, state-machine transitions, request/result interpretation, and events in their controllers. Replace direct sound calls with `runtime.playSound` and request calls with the runtime API.

Import `BoardEffects`, `animateAppliedMoves`, and `RequestOwner` directly from `components/chess` in all remaining consumers. Delete the three puzzle pass-through modules and update `move-animation.test.ts` imports.

- [ ] **Step 5: Verify GREEN, duplication removal, and commit**

```bash
npm --prefix frontend test -- --run --single-thread src/components/chess/interactive-board-runtime.test.ts src/components/openings/opening-controller.test.ts src/components/puzzle/puzzle-controller.test.ts src/components/puzzle/move-animation.test.ts
npm --prefix frontend run check
rg -n 'private (readonly requests|board:|sound:|soundUnlockStarted|recoveringBoard|pendingBoardWarning)|private async reconcilePosition|private animationPort|private recoverBoard' frontend/src/components/openings/opening-controller.ts frontend/src/components/puzzle/puzzle-controller.ts
git add frontend/src/components/chess/interactive-board-runtime.ts frontend/src/components/chess/interactive-board-runtime.test.ts frontend/src/components/openings/opening-controller.ts frontend/src/components/openings/opening-controller.test.ts frontend/src/components/puzzle/puzzle-controller.ts frontend/src/components/puzzle/puzzle-controller.test.ts frontend/src/components/puzzle/move-animation.test.ts frontend/src/components/puzzle/move-animation.ts frontend/src/components/puzzle/puzzle-effects.ts frontend/src/components/puzzle/request-owner.ts
git commit -m "refactor: share interactive board runtime"
```

Expected: focused tests and checks pass; `rg` returns no duplicated runtime ownership in either controller.

---

### Task 7: Full verification and strict architectural re-review

**Files:**

- Modify only when verification exposes a regression: files already owned by Tasks 1-6
- Do not create a broad cleanup commit

- [ ] **Step 1: Run backend verification**

```bash
go test ./...
go test -race ./...
go vet ./...
```

Expected: all packages pass with no race or vet diagnostics.

- [ ] **Step 2: Run frontend and release verification**

```bash
npm --prefix frontend test -- --run
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
npm --prefix frontend run test:e2e
node --test scripts/verify-release.test.mjs
node --test scripts/*.test.mjs
bash -n scripts/build-release.sh
```

Expected: at least the current 241 unit tests pass; Svelte and TypeScript report zero diagnostics; build, licenses, end-to-end, release-policy, and release-orchestration tests pass. Existing intentionally skipped end-to-end cases remain skipped. A signed release invocation remains the release workflow's responsibility because it requires a tag, isolated release root, signing identity, notarization, and Gatekeeper.

- [ ] **Step 3: Run structural acceptance scans**

```bash
git diff --check main...HEAD
git status --short
wc -l frontend/src/lib/api-contract.ts frontend/src/lib/api.ts frontend/src/components/openings/opening-controller.ts frontend/src/components/puzzle/puzzle-controller.ts
rg -n 'internal/openings/testdata/.*\.ctcourse' scripts/release-policy.mjs
rg -n 'RebaseSession|ReconcileReviews|AttemptState \*SessionState|promptCompletionAttempt|puzzles\.(ImportInspection|Progress)' internal normal_controller.go
rg -n '^function (record|string|optionalString|positiveInteger|nonNegativeInteger|array|enumeration)' frontend/src/lib
```

Expected:

- the only worktree change before the final handoff is the already committed implementation;
- compatibility barrels and both controllers are materially smaller;
- the fixture scan shows only `mini.ctcourse` and its reviewed digest;
- removed backend APIs and puzzle-owned shared contracts have no matches;
- every decoder primitive has one canonical definition.

- [ ] **Step 4: Re-run the thermo-nuclear review**

Inspect the complete `main...HEAD` diff and explicitly verify each original finding against its completion criterion. If a presumptive blocker remains, first add a failing regression test, then make the smallest correction and rerun its affected suite before declaring completion.

- [ ] **Step 5: Record a verification correction only when needed**

If verification required code changes, stage only those files and commit:

```bash
git commit -m "fix: close opening remediation regressions"
```

If no files changed, do not create an empty commit.
