# Generational Puzzle Catalogue Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans task-by-task. Use superpowers:test-driven-development for every behavior change and superpowers:verification-before-completion before any completion claim.

**Goal:** Replace the staging-and-merge puzzle database with a source-aware generational catalogue that remains readable during imports, activates in constant time, preserves learner history when the disposable catalogue is rebuilt, and imports the downloaded Lichess dataset within one hour.

**Architecture:** Build the version-3 store and repository alongside the legacy catalogue so every intermediate commit compiles. Then perform one atomic production cutover: migrate/backfill `user.sqlite`, recognize and replace only the exact legacy puzzle schema, switch all consumers to source-aware projections, and remove the legacy repository. A single writer gate serializes imports and bounded cleanup; a read-only pool serves active heads from one SQLite snapshot per public read. Session-visible occurrence data is snapshotted in `user.sqlite`, while stable core content remains fingerprint-addressed.

**Tech Stack:** Go 1.25, modernc SQLite in WAL mode, Wails v2.12, Svelte 3, Vitest, Playwright.

**Design source:** `docs/superpowers/specs/2026-07-15-generational-puzzle-catalogue-design.md`

## Global Constraints

- Work only in `/Users/admin/Documents/Work/chess/.worktrees/chess-trainer` on `feature/chess-trainer`.
- Preserve `user.sqlite` and `library.sqlite`. Destructive recreation is authorized only for an exactly recognized legacy `puzzles.sqlite`.
- Preserve fingerprint bytes exactly. The payload is JSON containing trimmed displayed FEN, solver color, and recursively normalized solution nodes; UCI strings are trimmed and lowercased before SHA-256.
- Use RED, GREEN, refactor. Capture the failing test output before production edits.
- Every task checkpoint must pass `go test ./...`; filtered `-run` commands do not excuse package compile failures.
- Keep wall-clock assertions behind `//go:build performance`; ordinary and race suites remain deterministic.
- Put final schema DDL only in migrations. A temporary version-3 bootstrap asset is allowed solely while legacy and generational implementations coexist, then moves atomically into `migrations/puzzles/003.sql`.
- Use bound SQL parameters and prepare repeated import statements once per batch transaction.
- Do not add the tactical PGN importer, game library, engine implementation, learner-action transaction redesign, restore redesign, or root frontend state-machine redesign in this slice.

## Task 0: Checkpoint the Existing Backup, Recovery, and Release Work

**Files:**

- Verify only: `app.go`, `app_test.go`, `main.go`, `internal/backup/**`, `internal/storage/integrity.go`, `internal/storage/space_darwin*`, `internal/chessrules/rules*`, `frontend/**`, `docs/operations/**`

- [ ] **Step 1: Audit the dirty tree**

Run:

```bash
git status --short
git diff --check
git diff --stat
```

Expected: only the already-reviewed backup/recovery/release work is dirty. The design and plan are not part of this implementation checkpoint.

- [ ] **Step 2: Verify the current backend**

Run:

```bash
env -u CHESS_TRAINER_LICHESS_PATH go test ./... -count=1
```

Expected: PASS; the opt-in real dataset test skips.

- [ ] **Step 3: Verify the current frontend**

Run:

```bash
(cd frontend && npm test -- --run --single-thread)
(cd frontend && npm run check)
(cd frontend && npm run build)
```

Expected: all unit/component tests PASS, Svelte check reports no errors, and Vite builds.

- [ ] **Step 4: Commit the existing work separately**

Run:

```bash
git add app.go app_test.go main.go internal frontend docs/operations
git diff --cached --check
git commit -m "feat: add backup recovery and release checks"
```

Expected: the prior work is recoverable before the catalogue rewrite.

## Task 1: Add Version-3 Primitives and Storage Alongside the Legacy Path

This task is deliberately additive. Do not replace the legacy `Catalog`, `StagedImport`, `SQLiteCatalog`, or puzzle migrations yet.

**Files:**

- Create: `internal/domain/chess.go`
- Create: `internal/domain/themes.go`
- Create: `internal/domain/themes_test.go`
- Create: `internal/puzzles/catalog_models.go`
- Create: `internal/puzzles/catalog_v3.go`
- Create: `internal/puzzles/catalog_models_test.go`
- Create: `internal/storage/puzzle_schema_v3.sql`
- Create: `internal/storage/puzzle_store.go`
- Create: `internal/storage/puzzle_store_test.go`
- Modify: `internal/domain/puzzle.go`
- Modify: `internal/puzzles/fingerprint.go`
- Modify: `internal/puzzles/fingerprint_test.go`

- [ ] **Step 1: Write failing normalization and fingerprint-compatibility tests**

Add:

```go
func TestNormalizeThemesTrimsDeduplicatesAndSorts(t *testing.T)
func TestCoreFingerprintMatchesLegacyFingerprint(t *testing.T)
func TestTrainingPuzzleKeepsSourceOccurrencesIndependent(t *testing.T)
```

The fingerprint test constructs one legacy `domain.Puzzle` and the equivalent `PuzzleCore`, then compares both to the existing golden digest. The occurrence test uses the same core with different source FEN, prelude, rating, themes, external ID, URL, attribution, and metadata.

Run:

```bash
go test ./internal/domain ./internal/puzzles -run 'Test(NormalizeThemes|CoreFingerprint|TrainingPuzzle)' -count=1
```

Expected: FAIL because the neutral helper and version-3 models do not exist.

- [ ] **Step 2: Move stable chess primitives without deleting compatibility types**

Move `Color` and `MoveNode` from `domain/puzzle.go` into `domain/chess.go`. Keep the legacy `SourceRef` and `Puzzle` aggregate in `domain/puzzle.go` until Task 5.

Add:

```go
func NormalizeThemes(values []string) []string
```

It trims, removes empty strings, deduplicates exact normalized names, and returns lexical order. Both storage backfill and puzzle imports use this lower-layer helper, avoiding a `storage <-> puzzles` cycle.

- [ ] **Step 3: Add source-aware models with no persistence identity leakage**

Define:

```go
type PuzzleKey struct {
    Fingerprint string
    SourceID    string
}

type PuzzleCore struct {
    Fingerprint   string
    DisplayedFEN  string
    Solver        domain.Color
    Solution      []domain.MoveNode
    SolutionPlies int
}

type PuzzleOccurrence struct {
    SourceID    string
    SourceKind  string
    ExternalID  string
    SourceFEN   string
    PreludeUCI  string
    Rating      *int
    Popularity  *int
    PlayCount   *int
    URL         string
    Attribution string
    Metadata    map[string]any
    Themes      []string
    Ordinal     int64
}

type TrainingPuzzle struct {
    Core       PuzzleCore
    Occurrence PuzzleOccurrence
}

func (p TrainingPuzzle) Key() PuzzleKey

type SourceSummary struct {
    SourceID             string
    Kind                 string
    MinimumRating        *int
    MaximumRating        *int
    MaximumSolutionPlies int
}
```

Do not expose `generation_id` on importer or consumer DTOs. The generation writer stamps that persistence identity.

- [ ] **Step 4: Add version-3 interfaces under coexistence-safe names**

Retain the old `Catalog` interface and add:

```go
type GenerationSource struct {
    ID        string
    Kind      string
    Path      string
    StartedAt time.Time
}

type GenerationImport interface {
    Add(context.Context, TrainingPuzzle) error
    Reject(Rejection)
    Seal(context.Context, string) (ImportReport, error)
    Activate(context.Context) error
    Abandon(context.Context) error
}

type CatalogReader interface {
    Get(context.Context, PuzzleKey) (TrainingPuzzle, error)
    Resolve(context.Context, string, string) (TrainingPuzzle, error)
    RatedCandidates(context.Context, int, int, []string, int) ([]TrainingPuzzle, error)
    FreePracticeCandidates(context.Context, string, *int, *int, []string, *int, int) ([]TrainingPuzzle, error)
    ActiveSourceSummaries(context.Context) ([]SourceSummary, error)
    ActiveThemes(context.Context) ([]string, error)
}

type CatalogWriter interface {
    BeginImport(context.Context, GenerationSource) (GenerationImport, error)
}

type CatalogMaintenance interface {
    RecoverStartup(context.Context) error
    CleanupBatch(context.Context, int) (bool, error)
}

type GenerationCatalog interface {
    CatalogReader
    CatalogWriter
    CatalogMaintenance
}
```

Add `ErrCatalogCorrupt`, `ErrHeadChanged`, and a typed source-kind mismatch error.

- [ ] **Step 5: Keep old and new fingerprints byte-identical**

Add `CoreFingerprint(PuzzleCore)` with the exact existing JSON payload and make the legacy `Fingerprint(domain.Puzzle)` delegate through a core projection. Do not change the golden digests.

- [ ] **Step 6: Write failing dual-handle storage tests**

Add:

```go
func TestOpenGenerationPuzzleStoreCreatesExactSchema(t *testing.T)
func TestOpenGenerationPuzzleStoreRejectsExistingEmptyFile(t *testing.T)
func TestPuzzleStoreEscapesSpecialPathCharacters(t *testing.T)
func TestPuzzleStoreAppliesPragmasToEveryConnection(t *testing.T)
func TestPuzzleStoreReaderRejectsWrites(t *testing.T)
func TestPuzzleStoreReadsDuringRealUncommittedWrite(t *testing.T)
```

Use a path containing spaces, `#`, and `?`. Acquire four reader `sql.Conn` values concurrently and check `foreign_keys=1` and `busy_timeout=5000` on every connection. The concurrency test must execute an insert in an uncommitted writer transaction, not merely open a deferred transaction.

Run:

```bash
go test ./internal/storage -run 'Test(OpenGenerationPuzzleStore|PuzzleStore)' -count=1
```

Expected: FAIL because the store does not exist.

- [ ] **Step 7: Add the temporary version-3 schema asset**

Create `puzzle_schema_v3.sql` with these tables and constraints:

- `sources(source_id PRIMARY KEY, kind NOT NULL)`.
- `source_generations` with `building|sealed|abandoned`, positive timestamps, optional checksum/sealed time, and `UNIQUE(source_id,generation_id)`.
- `source_heads` with a composite foreign key to the matching source generation.
- `puzzle_cores` with solver check, positive solution plies, and stable fields only.
- `puzzle_occurrences` with positive ordinal, source-specific fields, `PRIMARY KEY(generation_id,fingerprint)`, and cascading generation FK.
- `occurrence_themes` with composite occurrence FK and primary key.
- `schema_migrations` containing only version 3 while this standalone bootstrap asset is used.

Create migration-owned indexes:

```sql
CREATE INDEX idx_generations_cleanup
  ON source_generations(status, generation_id);
CREATE INDEX idx_source_heads_generation
  ON source_heads(generation_id);
CREATE INDEX idx_occurrences_rating
  ON puzzle_occurrences(generation_id, rating, fingerprint);
CREATE INDEX idx_occurrences_fingerprint
  ON puzzle_occurrences(fingerprint, generation_id);
CREATE INDEX idx_occurrence_themes_theme
  ON occurrence_themes(generation_id, theme, fingerprint);
```

Do not add the redundant `(generation_id,fingerprint)` occurrence index because the primary key already covers it.

- [ ] **Step 8: Implement URI-safe writer-first store opening**

Add:

```go
type PuzzleStore struct {
    Reader *sql.DB
    Writer *sql.DB
}

func OpenGenerationPuzzleStore(path string) (*PuzzleStore, error)
func (s *PuzzleStore) Close() error
```

Build modernc file URIs with `net/url`, not string concatenation. Apply repeated `_pragma=busy_timeout(5000)` and `_pragma=foreign_keys(ON)` in both DSNs. Open the writer first with one open/idle connection, bootstrap the temporary schema only when the path did not exist before the call, set WAL once, then open a `mode=ro` reader with four open/idle connections. Reject an existing empty or unversioned file and never use `immutable=1`.

Run:

```bash
go test ./internal/domain ./internal/puzzles ./internal/storage -count=1
go test ./... -count=1
```

Expected: PASS, including the untouched legacy application path.

- [ ] **Step 9: Commit the coexistence foundation**

Run:

```bash
git add internal/domain internal/puzzles/catalog_models.go internal/puzzles/catalog_v3.go internal/puzzles/catalog_models_test.go internal/puzzles/fingerprint.go internal/puzzles/fingerprint_test.go internal/storage/puzzle_schema_v3.sql internal/storage/puzzle_store.go internal/storage/puzzle_store_test.go
git diff --cached --check
git commit -m "feat: add generational puzzle foundations"
```

## Task 2: Build the Generational Repository Beside the Legacy Repository

The final files are introduced now, but the implementation type is temporarily named `GenerationalSQLiteCatalog`. The legacy `SQLiteCatalog` remains in use until Task 5.

**Files:**

- Create: `internal/puzzles/catalog_import.go`
- Create: `internal/puzzles/catalog_import_test.go`
- Create: `internal/puzzles/catalog_reader.go`
- Create: `internal/puzzles/catalog_reader_test.go`
- Create: `internal/puzzles/catalog_cleanup.go`
- Create: `internal/puzzles/catalog_cleanup_test.go`
- Modify: `internal/puzzles/catalog_v3.go`

- [ ] **Step 1: Write failing generation-lifecycle tests**

Add:

```go
func TestBuildingGenerationIsInvisible(t *testing.T)
func TestGenerationLocalDuplicateLastRecordWins(t *testing.T)
func TestSharedCoreRetainsIndependentOccurrences(t *testing.T)
func TestCoreContentMismatchReturnsCatalogCorrupt(t *testing.T)
func TestExistingSourceKindMismatchPreservesHead(t *testing.T)
func TestSealRequiresBuildingGenerationAndChecksum(t *testing.T)
func TestActivateRequiresOwnSealedGeneration(t *testing.T)
func TestActivateCASWithExpectedHead(t *testing.T)
func TestActivateCASWithNoExpectedHead(t *testing.T)
func TestAbandonDoesNotDeleteOrChangeHead(t *testing.T)
```

Run:

```bash
go test ./internal/puzzles -run 'Test(BuildingGeneration|GenerationLocal|SharedCore|CoreContent|ExistingSourceKind|Seal|Activate|Abandon)' -count=1
```

Expected: FAIL because the generational implementation is absent.

- [ ] **Step 2: Construct the repository with separate handles**

Implement:

```go
type GenerationalSQLiteCatalog struct {
    readDB  *sql.DB
    writeDB *sql.DB
}

func NewGenerationalSQLiteCatalog(readDB, writeDB *sql.DB) *GenerationalSQLiteCatalog
```

`BeginImport` validates the source and creates a UUID. In one writer transaction, it inserts the stable source with `ON CONFLICT DO NOTHING`, verifies its existing kind exactly, captures whether a prior head exists and its generation ID, and inserts the `building` generation. This makes expected-head capture linearizable. A kind mismatch rolls back and changes neither source nor head.

- [ ] **Step 3: Insert batches directly into the inactive generation**

Buffer 1,000 puzzles. In each transaction prepare core insert/select, occurrence insert/update, and theme delete/insert statements once. For every puzzle:

1. Verify occurrence source ID/kind matches the import source.
2. Marshal normalized stable solution and occurrence metadata.
3. Insert the core with `ON CONFLICT DO NOTHING`.
4. On conflict, compare displayed FEN, solver, solution JSON, and solution plies exactly; mismatch wraps `ErrCatalogCorrupt`.
5. Insert the occurrence with `ON CONFLICT DO NOTHING`; `RowsAffected=1` increments accepted.
6. On duplicate, increment duplicates and update all source fields plus ordinal so the last valid row wins.
7. Delete and reinsert that occurrence's normalized themes.

Clear the buffer only after commit. `Reject` retains at most 100 examples.

- [ ] **Step 4: Make seal and activation explicit state transitions**

`Seal(ctx, checksum)` flushes, requires non-empty normalized checksum, and conditionally updates only `building -> sealed`; require `RowsAffected=1` and set `sealed_at`.

`Activate(ctx)` starts a short transaction, verifies its generation is sealed and has a checksum, then uses distinct CAS paths:

- expected prior head: conditional `UPDATE ... WHERE generation_id = expected`;
- expected no head: `INSERT ... ON CONFLICT DO NOTHING`.

Require one affected row or return `ErrHeadChanged`. Activation changes only the head pointer and never cleans rows or rebuilds indexes.

`Abandon` conditionally changes only `building -> abandoned` and discards memory. A sealed, unheaded generation remains immutable and cleanup-eligible.

- [ ] **Step 5: Write failing active-reader snapshot tests**

Add:

```go
func TestGetRequiresFingerprintAndSource(t *testing.T)
func TestResolvePrefersRequestedActiveSource(t *testing.T)
func TestResolveFallsBackLexicographically(t *testing.T)
func TestCandidatesAndMetadataUseOnlyActiveHeads(t *testing.T)
func TestRatedCandidatesReturnOneDeterministicOccurrencePerFingerprint(t *testing.T)
func TestReaderSeesOldHeadAcrossManyImportBatches(t *testing.T)
func TestReaderNeverMixesOccurrenceAcrossActivationAndCleanup(t *testing.T)
```

The last test repeatedly activates and cleans generations while readers assert each returned row is wholly old or wholly new, never a core/occurrence/theme mixture.

Run:

```bash
go test ./internal/puzzles -run 'Test(GetRequires|Resolve|Candidates|Reader)' -count=1
```

Expected: FAIL because active readers do not exist.

- [ ] **Step 6: Implement every public read as one SQLite snapshot**

All active reads join through `source_heads`. Use either one aggregate SELECT that returns ordered theme JSON with each occurrence, or one read-only transaction bound to one reader connection for the entire public method. Never fetch candidate keys and hydrate them outside that snapshot.

`Get` filters fingerprint and source. `Resolve` tries the preferred source then lexical active source. Rated candidates apply filters, choose at most one occurrence per fingerprint using lexical source ID as the deterministic tie-break, and only then apply `LIMIT`, so duplicate active-source occurrences cannot underfill a session. Free-practice candidates are already scoped to one source. Both preserve rating/theme/depth/exclusion semantics. `ActiveSourceSummaries` and `ActiveThemes` ignore building, abandoned, and superseded generations.

- [ ] **Step 7: Write failing recovery and exact-budget cleanup tests**

Add:

```go
func TestRecoverStartupMarksBuildingAbandoned(t *testing.T)
func TestCleanupNeverTouchesActiveGeneration(t *testing.T)
func TestCleanupUsesOnePhysicalRowBudget(t *testing.T)
func TestCleanupResumesAndConverges(t *testing.T)
func TestCleanupPreservesSharedCoreAndRemovesOrphanCore(t *testing.T)
```

Seed more rows than the limit and count physical rows before/after each call. Verify no cascade makes a call delete more than `limit` rows across all tables.

- [ ] **Step 8: Implement bounded idempotent cleanup**

`RecoverStartup` performs one short `building -> abandoned` update; production will call it only while holding the data-root instance lock.

`CleanupBatch(ctx, limit)` uses one decrementing budget across every phase and table. Drain themes first. Delete only occurrences with no remaining themes, preventing FK cascades from exceeding the budget. Then delete empty eligible generations, unused sources, and unreferenced cores, decrementing from `RowsAffected` after each statement. Eligible generations are abandoned or sealed and absent from `source_heads`. Return `more=true` while eligible rows remain.

Run:

```bash
go test ./internal/puzzles -count=1
go test ./... -count=1
```

Expected: PASS with both catalogue implementations present.

- [ ] **Step 9: Commit the parallel repository**

Run:

```bash
git add internal/puzzles/catalog_v3.go internal/puzzles/catalog_import.go internal/puzzles/catalog_import_test.go internal/puzzles/catalog_reader.go internal/puzzles/catalog_reader_test.go internal/puzzles/catalog_cleanup.go internal/puzzles/catalog_cleanup_test.go
git diff --cached --check
git commit -m "feat: implement immutable puzzle generations"
```

## Task 3: Add Durable User Snapshots, Exact Legacy Recognition, and Instance Locking

This task remains compatible with the old runtime. The migration adds nullable columns; current legacy writes may leave them null until the atomic cutover.

**Files:**

- Create: `internal/storage/migrations/user/003.sql`
- Create: `internal/storage/testdata/legacy_puzzles_v1.sql`
- Create: `internal/storage/testdata/legacy_puzzles_v2.sql`
- Create: `internal/storage/legacy_puzzle.go`
- Create: `internal/storage/legacy_puzzle_test.go`
- Create: `internal/storage/data_root_lock.go`
- Create: `internal/storage/data_root_lock_test.go`
- Modify: `internal/storage/puzzle_store.go`
- Modify: `internal/storage/puzzle_store_test.go`
- Modify: `internal/storage/sqlite_test.go`

- [ ] **Step 1: Write failing user-migration tests**

Add a migration test that asserts the following columns exist and are nullable:

```sql
ALTER TABLE session_items ADD COLUMN source_kind TEXT;
ALTER TABLE session_items ADD COLUMN rating_snapshot INTEGER;
ALTER TABLE session_items ADD COLUMN themes_json TEXT;
ALTER TABLE session_items ADD COLUMN source_fen_snapshot TEXT;
ALTER TABLE session_items ADD COLUMN prelude_uci_snapshot TEXT;

ALTER TABLE attempts ADD COLUMN source_kind TEXT;
ALTER TABLE attempts ADD COLUMN rating_snapshot INTEGER;
ALTER TABLE attempts ADD COLUMN themes_json TEXT;

ALTER TABLE review_state ADD COLUMN preferred_source_id TEXT;
```

Run:

```bash
go test ./internal/storage -run TestUserMigration003 -count=1
```

Expected: FAIL because migration 003 is absent.

- [ ] **Step 2: Add the compatibility migration**

Create `migrations/user/003.sql` with exactly those additive columns. Do not add non-null defaults that invent attribution for legacy attempts.

- [ ] **Step 3: Freeze exact legacy schema fixtures before the cutover**

Copy the complete current puzzle migration-1 schema into `legacy_puzzles_v1.sql`. Build `legacy_puzzles_v2.sql` from migration 1 plus migration 2. Each fixture includes the exact `schema_migrations` rows expected for that version. These fixtures become the only retained copy of the old schema after Task 5 removes production migrations 001 and 002.

- [ ] **Step 4: Write failing conservative-probe tests**

Add:

```go
func TestProbeRecognizesExactLegacyV1AndV2(t *testing.T)
func TestProbeRecognizesExactCurrentV3(t *testing.T)
func TestProbeRejectsLegacyVersionWithChangedColumns(t *testing.T)
func TestProbePreservesUnknownNewerDatabaseAndSidecars(t *testing.T)
func TestProbePreservesCorruptOrUnversionedDatabaseAndSidecars(t *testing.T)
func TestRemoveRecognizedLegacyRefusesEveryOtherState(t *testing.T)
```

Hash the database and WAL before unknown/corrupt probe/removal attempts and verify they remain unchanged and no sidecar is deleted. SQLite may rebuild transient `-shm` coordination bytes even for a read-only live attachment, so SHM byte identity is not a data-preservation invariant. Recognition requires the exact migration set plus expected table/column signatures; a version number alone is insufficient.

Run:

```bash
go test ./internal/storage -run 'Test(Probe|RemoveRecognized)' -count=1
```

Expected: FAIL because conservative probing does not exist.

- [ ] **Step 5: Implement non-destructive probing and guarded removal**

Add:

```go
const CurrentPuzzleSchemaVersion = 3

type PuzzleFileState struct {
    Exists bool
    Legacy bool
    Format int
}

type PuzzleSchemaVersionError struct {
    Path      string
    Found     int
    Supported int
}

func ProbePuzzleStore(path string) (PuzzleFileState, error)
func OpenLegacyPuzzleReadOnly(path string) (*sql.DB, error)
func RemoveRecognizedLegacyPuzzleStore(path string) error
```

Probe with URI-safe `mode=ro`, small schema queries, `schema_migrations`, `sqlite_master`, and `PRAGMA table_info`; do not run full `quick_check` and do not copy a potentially multi-gigabyte current catalogue at startup. An existing empty, corrupt, unversioned, modified-legacy, or newer database is preserved and rejected. Guarded removal re-probes, refuses non-legacy state, closes all handles, then removes only the puzzle DB and its exact `-wal`/`-shm` sidecars.

- [ ] **Step 6: Write failing history-proportional backfill tests**

Add:

```go
func TestBackfillLegacySnapshotsMatchesFingerprintAndSource(t *testing.T)
func TestBackfillLegacySnapshotsIncludesQueuedPresentation(t *testing.T)
func TestBackfillLegacySnapshotsLeavesUnknownRowsNull(t *testing.T)
func TestBackfillLegacySnapshotsDoesNotOverwriteValues(t *testing.T)
func TestBackfillLegacySnapshotsIsIdempotentAndBatched(t *testing.T)
```

Instrument the legacy database or count lookup keys so the test proves work scales with distinct null learner-history keys, not with all catalogue rows. Include an unknown key so cursor pagination terminates even though its null values remain.

Run:

```bash
go test ./internal/storage -run TestBackfillLegacySnapshots -count=1
```

Expected: FAIL because backfill is absent.

- [ ] **Step 7: Implement bounded cross-store backfill**

Add:

```go
func BackfillLegacyPuzzleSnapshots(
    ctx context.Context,
    userDB *sql.DB,
    legacyPuzzles *sql.DB,
) error
```

Keyset-page distinct `(fingerprint,source_id)` values whose session or attempt snapshot is null. Fully read and close the user result rows before starting a user transaction on the one-connection pool. Query only those keys from the read-only legacy catalogue, including ordered themes. Update only null columns:

- session items: source kind, rating, normalized themes, source FEN, and prelude;
- attempts: source kind, rating, and normalized themes.

Advance the keyset cursor even for unmatched rows so unknown keys do not loop forever. Do not attach databases and do not mutate the legacy catalogue.

- [ ] **Step 8: Write failing single-instance lock tests**

Add:

```go
func TestAcquireDataRootLockRejectsSecondProcessOwner(t *testing.T)
func TestDataRootLockIsReleasedOnClose(t *testing.T)
```

Use a small dedicated SQLite lock file in the data root with a held `BEGIN IMMEDIATE` transaction, so the kernel/SQLite releases ownership after a crash and no stale lock file can block startup.

- [ ] **Step 9: Implement the data-root lock**

Add:

```go
type DataRootLock struct { /* private DB/connection ownership */ }

func AcquireDataRootLock(root string) (*DataRootLock, error)
func (l *DataRootLock) Close() error
```

Use zero/short busy timeout for a prompt typed “already running” error. The production composition will acquire this before probing or recovering generations and release it last.

Run:

```bash
go test ./internal/storage -count=1
go test ./... -count=1
```

Expected: PASS; the old runtime still operates with nullable new user columns.

- [ ] **Step 10: Commit preservation infrastructure**

Run:

```bash
git add internal/storage
git diff --cached --check
git commit -m "feat: preserve puzzle provenance across catalogue rebuilds"
```

## Task 4: Serialize Typed Import Jobs and Coordinate Cleanup/Shutdown

The coordinator can wrap the legacy Lichess importer now. Catalogue maintenance is optional until Task 5 passes the generational catalogue.

**Files:**

- Modify: `internal/importjob/service.go`
- Modify: `internal/importjob/service_test.go`
- Modify: `internal/app/services.go`
- Modify: `internal/app/services_test.go`
- Modify: `app.go`
- Modify: `app_test.go`

- [ ] **Step 1: Write failing typed-job and lifecycle tests**

Add:

```go
func TestStartRoutesRequestByKind(t *testing.T)
func TestStartRejectsSecondActivePuzzleImport(t *testing.T)
func TestResultKeepsMonotonicProgress(t *testing.T)
func TestCompletedResultRemainsQueryableAfterLaterJob(t *testing.T)
func TestJobReachesExactlyOneTerminalState(t *testing.T)
func TestTerminalJobAllowsNextImport(t *testing.T)
func TestCleanupStartsAfterTerminalAndNeverOverlapsImport(t *testing.T)
func TestNewImportPreemptsFurtherCleanupBatches(t *testing.T)
func TestCloseCancelsAndWaitsForImporterAndCleanup(t *testing.T)
```

Use blocking importers, blocking maintenance, and a recording emitter. Verify the busy error exposes the active job ID.

Run:

```bash
go test ./internal/importjob -count=1
```

Expected: FAIL because jobs are hard-coded and unconstrained.

- [ ] **Step 2: Add typed request, busy error, and durable result**

Define:

```go
type Kind string

const KindLichess Kind = "lichess"

type ImportRequest struct {
    Kind     Kind   `json:"kind"`
    SourceID string `json:"sourceId"`
    Path     string `json:"path"`
}

type BusyError struct {
    ActiveJobID string `json:"activeJobId"`
}

type Result struct {
    JobID    string               `json:"jobId"`
    Request  ImportRequest        `json:"request"`
    Status   Status               `json:"status"`
    Progress puzzles.Progress     `json:"progress"`
    Report   puzzles.ImportReport `json:"report"`
    Error    string               `json:"error,omitempty"`
}

type Maintenance interface {
    CleanupBatch(context.Context, int) (bool, error)
}

func NewService(
    importers map[Kind]Importer,
    maintenance Maintenance,
    emitter Emitter,
) *Service
```

Copy the importer map. `Start(context.Context, ImportRequest)` validates kind/source/path and reserves the sole active job atomically.

- [ ] **Step 3: Implement monotonic snapshots and one terminal transition**

Store every job result. Progress callbacks update stored counters using per-field maxima before emitting. Completion preserves request/progress, writes exactly one terminal state, clears the active ID, emits the terminal snapshot, and triggers cleanup. A later job never evicts an earlier result.

- [ ] **Step 4: Gate import and maintenance writes**

Use a writer mutex around the complete importer call and each cleanup batch. Lock order is explicit: `Start` reserves state and registers the goroutine, releases the state mutex, then the goroutine acquires the writer gate; importer completion releases the writer gate before taking state again. The one cleanup worker acquires the writer gate, briefly takes state to recheck that no job is active, releases state, and only then calls maintenance. No path holds state while waiting for the writer gate. If a job reserves the slot while a batch is running, its importer waits for that batch; further cleanup sees the active ID and yields.

- [ ] **Step 5: Add cancel-and-wait shutdown**

Create one long-lived cleanup worker and register it with the wait group before the service constructor returns. Under the same state mutex, `Start` checks `closing`, reserves the job, and calls `WaitGroup.Add(1)` before launching it. `Close` sets `closing` under that mutex before releasing it, so no `Add` can race `Wait`; it then cancels the active job and cleanup context and waits for both worker and jobs. Reject later starts and do not close any SQLite handle until this returns.

- [ ] **Step 6: Adapt application bindings without changing the frontend flow**

Construct the service with a Lichess importer map and nil maintenance for the legacy checkpoint. Expose:

```go
func (a *App) StartPuzzleImport(request importjob.ImportRequest) (string, error) {
    return a.services.ImportJobs.Start(a.ctx, request)
}

func (a *App) StartLichessImport(path string) (string, error) {
    return a.StartPuzzleImport(importjob.ImportRequest{
        Kind: importjob.KindLichess,
        SourceID: "lichess",
        Path: path,
    })
}
```

Keep `StartLichessImport` so the deferred frontend-state redesign stays out of scope. Change `Services.Close` to close/wait for import jobs before database handles.

Run:

```bash
go test ./internal/importjob ./internal/app . -count=1
go test ./... -count=1
```

Expected: PASS on the legacy catalogue with serialized typed jobs.

- [ ] **Step 7: Commit the coordinator**

Run:

```bash
git add internal/importjob internal/app app.go app_test.go
git diff --cached --check
git commit -m "feat: serialize puzzle imports and maintenance"
```

## Task 5: Atomically Cut Production Over to the Generational Catalogue

This is one intentional integration checkpoint. The migration chain, catalogue implementation, importer, consumers, and app composition must switch together. Do not commit or claim GREEN midway through this task; the working tree may temporarily fail to compile while old interfaces are removed.

**Files:**

- Create: `internal/storage/migrations/puzzles/003.sql` by moving/adapting `internal/storage/puzzle_schema_v3.sql`
- Modify: `internal/storage/puzzle_store.go`
- Modify: `internal/storage/puzzle_store_test.go`
- Modify: `internal/storage/integrity.go`
- Modify: `internal/storage/sqlite_test.go`
- Modify: `internal/puzzles/catalog.go`
- Modify: `internal/puzzles/catalog_import.go`
- Modify: `internal/puzzles/catalog_reader.go`
- Modify: `internal/puzzles/catalog_cleanup.go`
- Modify: `internal/puzzles/fingerprint.go`
- Modify: `internal/puzzles/fingerprint_test.go`
- Modify: `internal/puzzles/catalog_models_test.go`
- Modify: `internal/puzzles/lichess_importer.go`
- Modify: `internal/puzzles/lichess_importer_test.go`
- Modify: `internal/puzzles/full_import_test.go`
- Modify: `internal/training/scheduler.go`
- Modify: `internal/training/scheduler_test.go`
- Modify: `internal/training/service.go`
- Modify: `internal/training/service_test.go`
- Modify: `internal/training/user_store.go`
- Modify: `internal/training/review.go`
- Modify: `internal/training/review_test.go`
- Modify: `internal/profile/service.go`
- Modify: `internal/profile/service_test.go`
- Modify: `internal/app/services.go`
- Modify: `internal/app/services_test.go`
- Modify: `app.go`
- Modify: `app_test.go`
- Modify: `main.go`
- Delete: `internal/storage/puzzle_schema_v3.sql`
- Delete: `internal/storage/migrations/puzzles/001.sql`
- Delete: `internal/storage/migrations/puzzles/002.sql`
- Delete: `internal/puzzles/catalog_v3.go` after moving final contracts to `catalog.go`
- Delete: `internal/puzzles/sqlite_catalog.go`
- Delete: `internal/puzzles/sqlite_catalog_test.go`
- Delete: `internal/domain/puzzle.go`

- [ ] **Step 1: Write failing Lichess adapter and failure-semantics tests**

Add:

```go
func TestLichessProducesCoreAndOccurrence(t *testing.T)
func TestLichessPreflightUsesCatalogueVolume(t *testing.T)
func TestLichessPreflightFailureCreatesNoGeneration(t *testing.T)
func TestLichessCancellationReturnsPromptlyAndOnlyAbandons(t *testing.T)
func TestLichessFailureAndCancellationPreservePreviousHead(t *testing.T)
func TestLichessSealsThenActivates(t *testing.T)
```

Use different fake source and catalogue volumes; assert the free-space callback receives the catalogue directory. The cancellation fake must prove no synchronous physical deletion occurs and the call returns within a deterministic channel-based deadline rather than a broad timing sleep.

Run:

```bash
go test ./internal/puzzles -run 'TestLichess' -count=1
```

Expected: FAIL against the legacy staging contract.

- [ ] **Step 2: Write failing source-aware scheduler and solving tests**

Add:

```go
func TestGuidedReviewResolvesPreferredSource(t *testing.T)
func TestGuidedReviewFallsBackLexicographically(t *testing.T)
func TestGuidedReviewRemainsDormantWithoutActiveOccurrence(t *testing.T)
func TestNewAndPracticeCandidatesKeepSelectedOccurrence(t *testing.T)
func TestEverySolvePathLooksUpFingerprintAndSource(t *testing.T)
```

The fake records `Resolve(fingerprint,preferredSourceID)` and exact `Get(PuzzleKey)` calls for resume, view, move, hint, reveal, and completion.

Run:

```bash
go test ./internal/training -run 'Test(GuidedReview|NewAndPractice|EverySolvePath)' -count=1
```

Expected: FAIL because training still uses fingerprint-only `domain.Puzzle`.

- [ ] **Step 3: Write failing session/attempt snapshot tests**

Add:

```go
func TestCreateSessionWritesCompleteOccurrenceSnapshot(t *testing.T)
func TestQueuedAttemptCopiesStoredSnapshot(t *testing.T)
func TestQueuedSessionPresentationSurvivesSameSourceReimport(t *testing.T)
func TestHintAndRatingUseStoredSnapshotAfterReimport(t *testing.T)
func TestNewAttemptsWriteSourceKindRatingAndThemes(t *testing.T)
func TestReviewCompletionStoresActualPreferredSource(t *testing.T)
```

In the reimport tests, keep the same fingerprint/source but change rating, themes, source FEN, and prelude. Assert the queued session continues to display and score against its original snapshot. Stable displayed FEN, solver, and solution remain supplied by the fingerprint-identical core.

Run:

```bash
go test ./internal/training -run 'Test(CreateSessionWrites|Queued|HintAndRating|NewAttempts|ReviewCompletion)' -count=1
```

Expected: FAIL because new writes do not populate migration-3 columns.

- [ ] **Step 4: Write failing parent-report tests**

Add:

```go
func TestThemePerformanceUsesAttemptSnapshotsOnly(t *testing.T)
func TestThemePerformanceOmitsNullLegacyThemes(t *testing.T)
func TestThemePerformanceSurvivesCatalogueRecreation(t *testing.T)
func TestPracticeFiltersUseActiveCataloguePort(t *testing.T)
func TestRatingBoundsUseActiveSourceSummaries(t *testing.T)
```

Remove or replace the puzzle catalogue after completing attempts; the parent summary must remain unchanged.

Run:

```bash
go test ./internal/profile -run 'Test(ThemePerformance|PracticeFilters|RatingBounds)' -count=1
```

Expected: FAIL because profile reporting still joins mutable puzzle tables.

- [ ] **Step 5: Write failing startup, lock, recreation, and shutdown tests**

Add:

```go
func TestOpenAcquiresLockBeforeGenerationRecovery(t *testing.T)
func TestSecondServicesOpenCannotAbandonFirstInstanceImport(t *testing.T)
func TestOpenMigratesUserBeforeLegacyPuzzleRemoval(t *testing.T)
func TestOpenBackfillsSnapshotsBeforeLegacyPuzzleRemoval(t *testing.T)
func TestOpenRecreatesOnlyExactLegacyV1AndV2(t *testing.T)
func TestOpenPreservesUnknownNewerModifiedAndCorruptPuzzleFiles(t *testing.T)
func TestPuzzleRecreationLeavesCurrentUserDatabaseBytesAndRowsUnchanged(t *testing.T)
func TestNormalStartupNeverQuickChecksPuzzleCatalogue(t *testing.T)
func TestOpenRecoversBuildingGenerationAndTriggersBoundedCleanup(t *testing.T)
func TestServicesCloseWaitsBeforeClosingPuzzleHandles(t *testing.T)
```

For file-preservation cases, hash database and WAL data and assert no DB/WAL/SHM path is deleted; allow transient SHM bytes to be rebuilt by SQLite. For the user preservation case, start from a current closed database with no null snapshots, hash it, run `Open`/`Close`, checkpoint, and compare both bytes and rows.

Run:

```bash
go test ./internal/app ./internal/storage -run 'Test(Open|SecondServices|PuzzleRecreation|NormalStartup|ServicesClose)' -count=1
```

Expected: FAIL because startup still scans/opens the legacy puzzle DB first and owns one handle.

- [ ] **Step 6: Finalize the migration chain and puzzle store**

Copy the v3 DDL into `migrations/puzzles/003.sql` but remove the temporary asset's `schema_migrations` DDL/insert; the generic migration runner owns those. Delete production puzzle migrations 001 and 002 after their exact DDL is safely in Task 3 test fixtures. Add the `performance` build tag to the still-legacy full-import test immediately, then adapt its body in Task 6; this keeps the atomic cutover compiling without running a stale test contract.

Rename the opener to:

```go
func OpenPuzzleStore(path string) (*PuzzleStore, error)
```

For a missing file, open the writer, run generic `Migrate(...,"puzzles")`, set WAL, validate required tables, and open the read-only pool. For existing version 3, migrate/no-op and validate. Refuse legacy or unknown files; application composition handles recognized legacy before calling it.

Run:

```bash
test "$(find internal/storage/migrations/puzzles -maxdepth 1 -name '*.sql' | wc -l | tr -d ' ')" = 1
go test ./internal/storage -run 'Test(OpenGenerationPuzzleStore|PuzzleStore|Probe|Backfill|DataRoot)' -count=1
```

Expected: storage tests PASS with only migration 003 embedded.

- [ ] **Step 7: Promote the generational contracts and repository**

Rewrite `catalog.go` to expose final `Source`, `GenerationImport`, `CatalogReader`, `CatalogWriter`, `CatalogMaintenance`, and combined `Catalog`. Remove the coexistence-only names/file. Rename `GenerationalSQLiteCatalog` and its constructor to:

```go
type SQLiteCatalog struct { /* separate read/write handles */ }

func NewSQLiteCatalog(readDB, writeDB *sql.DB) *SQLiteCatalog
```

Delete the staging repository and its tests. Preserve all Task 2 behavior. Verify no repository creates/drops migration-owned indexes at runtime.

- [ ] **Step 8: Adapt Lichess to the generation lifecycle**

Give the importer an explicit catalogue directory and require production composition to set it:

```go
type LichessImporter struct {
    Catalog          CatalogWriter
    Rules            chessrules.Rules
    CatalogDirectory string
    AvailableBytes   func(string) (uint64, error)
}
```

Run disk preflight against `CatalogDirectory` before `BeginImport`. Parse each row into `TrainingPuzzle`; set occurrence source ID/kind and ordinal, normalize themes, and preserve provenance. Retain zstd streaming, one decoder, bounded memory, cancellation checks, checksum, progress, and 100 rejection examples.

Failure before seal calls `Abandon` using a short non-cancelled cleanup context. Cancellation never deletes rows. Success calls `Seal(checksum)` then `Activate`. Activation failure leaves a sealed unheaded generation for cleanup and never changes the old head.

- [ ] **Step 9: Make training source-aware and snapshot-authoritative**

Replace optional capability assertions with one explicit constructor port:

```go
type TrainingCatalogPort interface {
    Get(context.Context, puzzles.PuzzleKey) (puzzles.TrainingPuzzle, error)
    Resolve(context.Context, string, string) (puzzles.TrainingPuzzle, error)
    RatedCandidates(context.Context, int, int, []string, int) ([]puzzles.TrainingPuzzle, error)
    FreePracticeCandidates(context.Context, string, *int, *int, []string, *int, int) ([]puzzles.TrainingPuzzle, error)
}

type ScheduledPuzzle struct {
    Puzzle        puzzles.TrainingPuzzle
    Kind          ScheduledKind
    UpdatesRating bool
}
```

Reviews use `Resolve` with `PreferredSourceID`; new/practice items retain the selected occurrence. Remove `firstSourceID`.

Add to stored session state:

```go
type attemptSnapshot struct {
    SourceKind string
    Rating     *int
    Themes     []string
    SourceFEN  string
    PreludeUCI string
}
```

`CreateSession` writes it to `session_items`; `loadItems` reads it; `insertAttempt` copies reporting fields. All solve paths fetch the exact active key to obtain/validate the stable core, but use the stored snapshot for source presentation, hints, and rating. A missing exact occurrence follows existing unavailable-item behavior.

Add `PreferredSourceID` to `ReviewState`, load it, preserve it through `NextReview`, and upsert the actual source used at completion. Existing null values map to empty string and deterministic `Resolve` fallback.

- [ ] **Step 10: Remove mutable catalogue dependencies from parent reporting**

Define:

```go
type ProfileCatalogPort interface {
    ActiveSourceSummaries(context.Context) ([]puzzles.SourceSummary, error)
    ActiveThemes(context.Context) ([]string, error)
}

func NewService(
    userDB *sql.DB,
    catalog ProfileCatalogPort,
    store *training.UserStore,
) *Service
```

Build practice filters and rating bounds through the port. Compute historical themes only from `attempts.themes_json`:

```sql
SELECT CAST(theme.value AS TEXT),
       COUNT(*),
       COALESCE(SUM(a.first_try), 0)
FROM attempts a
JOIN json_each(a.themes_json) AS theme
WHERE a.completed_at IS NOT NULL
  AND theme.type = 'text'
GROUP BY CAST(theme.value AS TEXT)
ORDER BY CAST(theme.value AS TEXT);
```

Delete the dynamic puzzle-table lookup and its batching constant.

- [ ] **Step 11: Recompose startup in destructive-safe order**

Replace `CheckExistingIntegrity` with `CheckDurableIntegrity`, which scans only user and library databases. Use this order:

1. Ensure the data root.
2. Acquire `DataRootLock`.
3. Integrity-check durable user/library stores.
4. Open and migrate `user.sqlite`.
5. Probe `puzzles.sqlite`.
6. If and only if exact legacy: open read-only, backfill null snapshots, close, guarded-remove DB/WAL/SHM.
7. Open the v3 `PuzzleStore` and construct `SQLiteCatalog`.
8. Run `RecoverStartup` while the instance lock is held.
9. Open/migrate `library.sqlite`.
10. Construct Lichess importer, typed job service with catalogue maintenance, training, profile, and backup.
11. Trigger the job service's bounded cleanup worker once for startup leftovers.

Store `*storage.PuzzleStore` and `*storage.DataRootLock` on `Services`. On close: stop/wait import and maintenance goroutines, close reader/writer handles, close other databases, then release the data-root lock last. Make all error paths use the same order.

- [ ] **Step 12: Delete the obsolete aggregate and prove the integrated backend**

Delete `domain.Puzzle` and `SourceRef` after no consumer references them. Remove the temporary legacy fingerprint delegate and update its compatibility test to use the frozen golden input/digest directly. Keep `domain.Color`, `MoveNode`, and frontend session view types.

Run:

```bash
rg -n 'domain\.Puzzle|StagedImport|import_staging|dropSecondaryCatalogIndexes|firstSourceID|puzzleDB' internal || true
rg -n 'CREATE INDEX|DROP INDEX' internal/puzzles || true
go test ./internal/puzzles ./internal/training ./internal/profile ./internal/storage ./internal/importjob ./internal/app . -count=1
go test ./... -count=1
```

Expected: searches return no legacy aggregate/staging/runtime-index code and the complete backend passes.

- [ ] **Step 13: Regenerate Wails bindings and verify frontend compatibility**

Run:

```bash
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 generate module
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
git diff --check
(cd frontend && npm test -- --run --single-thread)
(cd frontend && npm run check)
(cd frontend && npm run build)
```

Expected: generated models include typed job result fields; the existing Lichess UI continues through its compatibility method and all frontend checks PASS.

- [ ] **Step 14: Commit the atomic cutover**

Run:

```bash
git add app.go app_test.go main.go internal frontend/wailsjs
git diff --cached --check
git commit -m "feat: activate the generational puzzle catalogue"
```

## Task 6: Prove Full-Dataset and Constant-Time Performance

**Files:**

- Modify: `internal/puzzles/full_import_test.go`
- Create: `internal/puzzles/catalog_performance_test.go`
- Modify as a measured regression requires: catalogue/import implementation files

**Measured refinement:** Profiling replaced superlinear direct final-table writes with two non-overlapping disposable catalogue-local SQLite artifacts. The ordered writer appends normalized rows to an append stage in 1,000-row transactions. Sealing performs one wide external window sort over `(fingerprint, row_id DESC)` into a separate full-width, fingerprint-clustered winner database; that same sort selects the last stream row, counts duplicates, and computes exact stable-core conflicts without a per-winner append-stage lookup. After the winner normally closes, the append stage is deleted before any durable final-table write. The sole final writer then sequentially scans the winner for cores and canonical occurrences, and sequentially reads it into destination-order rating and theme sorts. Each final phase disables automatic checkpointing, commits under WAL `synchronous=NORMAL`, runs a truncating checkpoint, and restores the 16,384-page setting. Because destination rating/theme order randomizes their fingerprint-first parent lookups at full scale, those two phases suspend foreign-key probes only on the sole writer outside transactions and restore enforcement on every path. Before seal, authoritative winner-versus-destination rating and theme audits compare positive expected and negative actual full keys with one grouped external sort and sequential scans, catching missing, extra, and replaced rows without per-row parent searches; a scoped SQLite foreign-key check validates occurrence rows built with enforcement enabled. The final schema uses a fingerprint-first `puzzle_occurrences` primary key with a canonical theme snapshot, one generation/fingerprint maintenance index for bounded cleanup, generation/theme/fingerprint occurrence-theme storage, a compact generation/rating-key/fingerprint `occurrence_ratings` table, no other reverse secondary indexes, and an exact sealed-generation theme facet. `rating_key` uses the signed 64-bit minimum solely to order null-rated occurrences before rated ones in unbounded practice. Candidate queries count indexed live theme membership and choose membership-first at 1,000 rows or fewer and rating-first above that threshold.

A 500,000-row synthetic tracer completed in 1m57.494s: 1m13.455s staging, 3.060s winner compaction, 40.300s across the four checkpointed final phases, and 0.529s for facet plus seal. It retained exact last-record-wins results for 499,964 winners and 36 duplicates, increased Go heap by about 10 MiB and RSS by about 204 MiB, used a 1.300 GB peak database footprint against the full-gate-derived 2.392 GB ceiling, activated in 141 microseconds, and returned common-theme and rare-theme probes in 23.47 ms and 2.48 ms respectively. The generation/fingerprint cleanup index was added after this tracer; occurrence rows already arrive in its key order and the measured footprint retained about 1.092 GB of reserve. This evidence validates the promoted shape but does not replace the downloaded full-dataset acceptance gate below.

The first downloaded full-dataset run exposed a per-winner append-stage row-ID probe that delayed final materialization until 46m53s. Folding exact conflict detection into winner aggregation moved that boundary to about 20m, but a second run then showed the joined wide-row materialization path growing the core WAL from 529 MiB to 723 MiB in 86 seconds and projecting beyond the 65-minute gate once occurrence, rating, and theme phases were included. The focused red-green refinement therefore requires the one-sort full-width winner architecture above, exact `EXPLAIN` proof that compaction performs no staged-row search, an append-deleted-before-final lifecycle test, and a large synthetic tracer before the downloaded gate is attempted again.

The first full-width-winner acceptance run reached six million staged rows in 27m22s but timed out at 65 minutes in rating materialization. VDBE evidence showed one fingerprint-first parent `Found` probe per destination-rating-ordered row. Suspending those probes reduced the 500,000-row materialization gate from 39.049s to 15.629s, but the next full run crossed one hour in the generic `PRAGMA foreign_key_check`, which performed the same cache-hostile child-order parent probes. Focused red-green tests then replaced only the two large-child checks with the exact ordered winner audits above. The final 500,000-row gate completed in 23.055s total, and the downloaded dataset passed in 29m04.914s with 6,052,811 accepted rows, 4,545 duplicates, zero rejections, 1.805ms activation, 22 MiB heap growth, and 299 MiB RSS growth.

- [x] **Step 1: Isolate timing assertions from correctness/race tests**

Add `//go:build performance` to `full_import_test.go` and the new performance file. No ordinary test may assert wall-clock ingestion, activation, or query speed.

- [x] **Step 2: Add synthetic activation and candidate-query gates**

Add:

```go
func TestGenerationActivationCompletesWithinFiveSeconds(t *testing.T)
func TestActiveCandidateQueryCompletesWithin250Milliseconds(t *testing.T)
```

Populate enough sealed rows to expose accidental copying/deletion. Start activation timing immediately before `Activate`. Warm the read pool before candidate timing.

Run:

```bash
go test -tags=performance ./internal/puzzles \
  -run 'Test(GenerationActivation|ActiveCandidateQuery)' -count=1
```

Expected: PASS; activation updates only one head row.

- [x] **Step 3: Upgrade the real Lichess acceptance test**

Use the dual-handle store and a test-only catalogue decorator that times the exact `GenerationImport.Activate` call without production instrumentation. Seed a prior head for the same source. On many progress callbacks across ingestion batches, read through the public reader and assert the complete old occurrence remains visible.

Assert:

- more than 1,000,000 accepted occurrences;
- total import completes in less than one hour;
- exact activation completes in less than five seconds;
- peak heap growth is at most 256 MiB;
- maximum RSS growth is at most 384 MiB;
- rejection ratio is below 0.1%;
- no decompressed CSV is created;
- post-activation candidates are source-aware and available.

- [x] **Step 4: Run the actual downloaded dataset gate**

Run:

```bash
: "${CHESS_TRAINER_LICHESS_PATH:?set CHESS_TRAINER_LICHESS_PATH to the downloaded .zst file}"
CHESS_TRAINER_LICHESS_PATH="$CHESS_TRAINER_LICHESS_PATH" \
  go test -tags=performance ./internal/puzzles \
  -run '^TestFullLichessImport$' -count=1 -timeout=65m
```

Expected: the test itself asserts elapsed time below one hour and PASSes before the 65-minute process timeout. If it fails, retain progress/timing evidence, add a focused failing regression test or benchmark, fix the measured bottleneck, and rerun.

- [x] **Step 5: Re-run ordinary correctness after performance changes**

Run:

```bash
go test ./... -count=1
```

Expected: PASS; performance-tagged files are not compiled.

- [x] **Step 6: Commit full-scale coverage**

Run:

```bash
git add internal/puzzles
git diff --cached --check
git commit -m "test: verify puzzle catalogue at full scale"
```

## Task 7: Final Verification, Operations Documentation, and Branch Audit

**Files:**

- Modify: `docs/operations/local-build.md`
- Modify only if verification exposes a defect: implementation/test files from Tasks 1–6

- [ ] **Step 1: Format and run static checks**

Run:

```bash
rg --files internal -g '*.go' -0 | xargs -0 gofmt -w
gofmt -w app.go app_test.go main.go
go vet ./...
git diff --check
```

Expected: formatting is stable, vet reports no findings, and whitespace checks PASS.

- [ ] **Step 2: Run complete backend correctness and race suites**

Run:

```bash
go test ./... -count=1
go test -race ./... -count=1
```

Expected: both PASS; no performance-tagged test runs.

- [ ] **Step 3: Run frontend, browser, and packaged-app gates**

Run:

```bash
(cd frontend && npm test -- --run --single-thread)
(cd frontend && npm run check)
(cd frontend && npm run build)
(cd frontend && npm run test:e2e)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
git diff --check
```

Expected: component tests, Svelte check, Vite build, Chromium/WebKit E2E, and the macOS Wails app build all PASS.

- [ ] **Step 4: Run architecture invariant searches**

Run:

```bash
test "$(find internal/storage/migrations/puzzles -maxdepth 1 -name '*.sql' | wc -l | tr -d ' ')" = 1
rg -n 'PRAGMA quick_check' internal
rg -n 'import_staging|domain\.Puzzle|StagedImport|DROP INDEX' internal || true
rg -n 'context\.Background\(\)' internal/importjob internal/app
```

Expected: one puzzle migration; quick-check appears only in explicit durable/diagnostic paths and never normal puzzle startup; obsolete staging symbols are absent; background contexts are limited to documented bounded abandonment/startup cases, not job lifetime.

- [ ] **Step 5: Update local operations documentation**

Document:

- `puzzles.sqlite` is a disposable versioned cache while `user.sqlite` is durable;
- exact legacy recognition/backfill precedes puzzle recreation;
- unknown/corrupt/newer puzzle stores are preserved;
- imports and cleanup are serialized and shutdown waits for them;
- normal startup does not scan the full puzzle catalogue;
- the explicit performance-tagged commands and real Lichess gate;
- how to rebuild the macOS Wails app.

- [ ] **Step 6: Re-run the thermo-nuclear maintainability review**

Apply `thermo-nuclear-code-quality-review` to the final diff. Treat actionable catalogue-slice findings as new RED tests/fixes, then repeat Steps 1–4. Record genuinely deferred findings only when they match the already-approved later remediation slices.

- [ ] **Step 7: Commit documentation and any review fixes**

Run:

```bash
git add docs/operations app.go app_test.go main.go internal frontend
git diff --cached --check
git commit -m "docs: document generational catalogue operations"
```

If there are no code or documentation changes after review, do not create an empty commit.

- [ ] **Step 8: Audit the finished branch**

Run:

```bash
git status --short
git log --oneline --decorate -15
git diff "$(git merge-base main HEAD)"..HEAD --stat
```

Expected: clean worktree, concern-separated commits, and the approved spec/plan plus verified implementation on `feature/chess-trainer`.
