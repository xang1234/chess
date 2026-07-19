# Italian Opening Quality Remediation Design

**Date:** 2026-07-19
**Status:** Approved
**Branch:** `codex/italian-opening-course`

## Context

The Italian opening course branch is functionally complete and passes its Go, frontend, type, end-to-end, and release-policy suites. A thermo-nuclear quality review nevertheless found six structural problems: a path-only privacy guard, non-atomic user-progress rebasing, an overloaded persisted session state, duplicated interactive-board controller infrastructure, oversized frontend API modules with duplicated decoders, and shared import behavior that remains owned by the puzzle package.

This remediation keeps the current product scope: one private, external, White-side Italian course with quick, standard, and reference depths. It changes internal boundaries and failure behavior without adding opening content or new user-facing features.

## Goals

1. Make it mechanically impossible to track an unreviewed course pack by merely placing it in the testdata directory.
2. Make each user-database course revision atomic across session rebasing and review reconciliation.
3. Give persisted session concepts explicit types and remove the optional whole-session attempt snapshot.
4. Reuse one small interactive-board runtime between puzzle and opening controllers.
5. Split frontend contracts and API implementations along domain boundaries without breaking existing consumers.
6. Make `internal/importing` the canonical owner of shared import contracts and validation helpers.

## Non-goals

- Changing course-pack schema or authoring private Italian content.
- Adding Black-side course UI or multi-course selection.
- Combining the course catalog and user-progress SQLite databases.
- Rewriting the puzzle or opening state machines into a generic state machine.
- Changing generated Wails JSON payloads or public controller method names.

## 1. Synthetic Fixture Boundary

`scripts/release-policy.mjs` will define an exact allowlist whose keys are repository-relative fixture paths and whose values are lowercase SHA-256 digests. Initially the only entry is `internal/openings/testdata/mini.ctcourse` with the digest calculated from the committed bytes.

`assertCourseFixtureBoundary` will:

- match `.ctcourse` suffixes case-insensitively;
- reject every course pack whose normalized repository path is absent from the allowlist;
- read each allowed fixture and reject it when its current digest differs from the reviewed digest;
- reject path traversal and non-normalized aliases rather than normalizing them into an allowed path.

The function will accept injected file-reading and hashing dependencies in tests. Release verification will provide repository-rooted production implementations. Tests will prove that an arbitrary pack under `internal/openings/testdata`, an uppercase extension, an allowed path with changed bytes, and a traversal alias are rejected.

The allowlist is intentionally code-reviewed and explicit. A filename convention or course metadata field is not accepted as proof that content is synthetic.

## 2. Atomic User Course Revisions

The catalog and user databases remain separate. Course import will continue to atomically activate a catalog generation inside the catalog database. User progress will reconcile when a workflow consumes the new generation; no cross-database transaction will be implied.

`UserStore` will expose one command for the user-database mutation boundary:

```go
type CourseRevision struct {
    CourseID          string
    PromptFingerprints map[string]string
    SessionRebase     *SessionRebase
    Now               time.Time
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

The method will open one user-database transaction, optionally update the session with optimistic generation matching, archive or refresh stale review rows, and commit once. Any error rolls back both effects. Existing `RebaseSession` and `ReconcileReviews` write APIs will be removed or reduced to unexported transaction helpers so service code cannot recreate the partial-update sequence.

`Service.Resume`, `Service.Restart`, and review-rebase paths will construct a `CourseRevision` and call the atomic command. Starting a fresh review will reconcile reviews before selecting the due queue through the same command without a session rebase.

`Service.Home` and `courseSummary` will be read-only. Due counts will be computed against the active course's prompt/fingerprint set so stale rows do not appear while waiting for explicit reconciliation. A home request must succeed against a database configured to reject writes.

## 3. Explicit Persisted Session Concepts

The opening session state is new on this unmerged branch, so its JSON representation can be corrected before release without a legacy-data migration. The persisted state will be composed from named concepts:

```go
type PositionState struct {
    PositionID    string
    CurrentFEN    string
    PlayedMoveIDs []string
}

type AttemptState struct {
    AttemptID         string
    PromptID          string
    StartedAt         time.Time
    HintLevel         int
    IncorrectMoves    int
    AlternativesTried int
    HintsUsed         int
    Revealed          bool
}

type ReviewCursor struct {
    PromptIDs []string
    Index     int
}

type SessionSummary struct {
    CompletedPrompts   int
    PositionsRecalled  int
    BranchesRecognized int
    Retried            int
    UsedHint           int
    Revealed           int
}

type RestartCheckpoint struct {
    StepIndex int
}

type SessionState struct {
    Position PositionState
    Attempt  *AttemptState
    Review   *ReviewCursor
    Summary  SessionSummary
    Restart  *RestartCheckpoint
}
```

Validation will enforce mode-specific invariants: lesson sessions cannot carry a review cursor; review sessions must carry one; active prompt steps require an attempt; passive steps do not; restart state cannot coexist with an active attempt.

`PromptCompletion` will require a value-type `AttemptRecord` containing only immutable attempt facts needed for persistence. Service code will capture that record before it advances the session, rather than cloning an entire `SessionState` and passing an optional pointer as an escape hatch.

Lesson completion SQL will exist in one transaction helper used by `CompletePrompt`. The standalone production-unused `CompleteLesson` method will be removed; its useful coverage will move to `CompletePrompt` tests.

## 4. Shared Interactive-Board Runtime

A new composition helper under `frontend/src/components/chess/` will own only infrastructure shared by puzzle and opening controllers:

- mounted/destroyed lifecycle;
- `RequestOwner` start, cancel, and current-token checks;
- board attachment and authoritative-position recovery;
- animation-port construction;
- reduced-motion and sound lifecycle;
- pending board-warning collection and deduplication.

The runtime will receive narrow callbacks for reading the authoritative FEN and publishing a recovered FEN or warning. It will not know puzzle states, opening states, operations, result contracts, or user-facing messages.

`PuzzleController` and `OpeningController` will retain their separate state machines, request/result interpretation, and domain events. They will delegate lifecycle and board effects to the runtime. Existing thin puzzle re-export modules for request ownership, animation, and effects will be removed after consumers import the canonical chess modules directly.

The runtime will have focused unit tests for cancellation, stale tokens, recovery, warning deduplication, reduced-motion behavior, and sound teardown. Existing controller tests will continue to prove domain-specific behavior.

## 5. Frontend Contract and API Modules

The public import surface remains stable through barrel files, but implementation moves into focused modules:

```text
frontend/src/lib/contracts/
  decoder.ts          shared runtime decoder primitives
  application.ts      application mode and build contracts
  imports.ts          import inspection, progress, result contracts
  puzzles.ts          puzzle session and move contracts
  openings.ts         opening home, lesson, and explorer contracts

frontend/src/lib/api/
  production.ts       Wails-backed adapters only
  preview.ts          preview fixtures and simulators only
```

`api-contract.ts` will re-export the domain contracts so existing imports do not require a repository-wide migration. `opening-explorer-contract.ts` will become a compatibility re-export or be removed after its consumers use the openings contract module. Decoder primitives will have one canonical implementation.

`api.ts` will define or re-export the API interfaces, choose production versus preview, and contain no preview course/puzzle simulation. Production bindings will not import preview fixtures.

Contract tests will remain organized by domain and will include the explorer decoder cases currently covered by the separate module. Preview tests will import the preview adapter directly when testing simulated behavior.

## 6. Canonical Shared Import Infrastructure

`internal/importing` will own:

```go
func NormalizePath(path, subject string) (string, error)
func CompareInspection(current, expected Inspection, subject string) error
```

`NormalizePath` will preserve the current absolute, clean, and best-effort symlink resolution semantics while using `subject` in actionable errors. `CompareInspection` will compare every shared inspection identity field and use `subject` only for the error prefix.

Puzzle and course importers will call these helpers and delete their local copies. `normal_controller.go` and the Wails event emitter will use `importing.Inspection` and `importing.Progress` directly. Puzzle aliases will be removed where they are no longer required; domain-specific adapter aliases may remain only when they genuinely express puzzle behavior.

The JSON contract and generated bindings must remain byte-for-byte compatible at the field-name level. Shared helper tests will cover puzzle and course error prefixes, path normalization, symlink behavior, and all compared fields.

## Error Handling and Recovery

- Fixture verification fails closed and reports the exact rejected path or digest mismatch.
- `ApplyCourseRevision` wraps transaction stage errors and preserves optimistic generation-conflict behavior.
- A failed user reconciliation leaves the previous session generation and every review row unchanged.
- Read-only home queries ignore stale fingerprints rather than mutating them.
- Board runtime failures recover to the authoritative FEN and publish one deduplicated warning without swallowing the original operation failure.
- Contract decoders keep their current path-rich error messages after moving modules.
- Import helper errors retain the current `puzzle import` or `course import` subject wording.

## Test Strategy

Implementation will use red-green-refactor cycles in this order:

1. Release-policy tests reproduce arbitrary-testdata, case, traversal, and digest failures.
2. User-store tests inject a review-update failure and prove session rebasing rolls back; service tests prove home performs no writes and filters stale review rows.
3. Session-state tests express mode invariants and required attempt records before the persistence refactor.
4. Interactive-board runtime tests establish the shared lifecycle before either controller delegates to it.
5. Contract and API tests lock exports and decoder behavior before files move.
6. Importing package tests establish canonical comparison and normalization behavior before callers migrate.

After each slice, its focused tests and affected package suite run. Final verification includes:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- frontend Vitest suite
- Svelte and Playwright type checks
- frontend build and license verification
- Playwright end-to-end suite
- release-policy suite and full release verification
- `git diff --check`

## Completion Criteria

- Every course fixture is both exactly allowlisted and content-hash verified.
- No service path commits a session rebase separately from review reconciliation.
- Opening home performs no database writes.
- Attempt persistence does not accept `SessionState` or an optional attempt snapshot.
- Lesson completion SQL has one implementation.
- Puzzle and opening controllers share the infrastructure runtime without sharing domain state machines.
- Decoder primitives, production/preview adapters, inspection comparison, and path normalization each have one canonical implementation.
- Existing public UI behavior and JSON field names remain unchanged.
- All focused and full verification commands pass from a clean worktree.
