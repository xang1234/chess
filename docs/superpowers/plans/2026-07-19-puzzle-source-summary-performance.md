# Puzzle Source Summary Performance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make unrated Free Practice filters load without scanning millions of puzzle occurrences by persisting each generation's maximum solution length.

**Architecture:** Puzzle schema v4 adds `maximum_solution_plies` to immutable source generations. Exact v3 catalogues are recognized as supported predecessors and atomically backfilled in place; future imports compute the value from the compact winner database and commit it while sealing. `ActiveSourceSummaries` then reads the scalar directly while retaining indexed rating endpoint lookups.

**Tech Stack:** Go 1.25+, `database/sql`, modernc SQLite, Wails v2, Go build tags for performance tests, Svelte/Vite production build.

## Global Constraints

- Implement only the primary source-summary fix; do not change the 200-puzzle candidate pool or hydration path.
- Preserve exact `MaximumSolutionPlies` behavior for existing and future imports.
- Upgrade exact v3 catalogues in place without deleting, recreating, or reimporting them.
- Keep exact logical and physical schema validation; modified or unknown catalogues remain untouched and enter recovery.
- Keep legacy v1/v2 snapshot backfill and recreation behavior unchanged.
- A warmed `ActiveSourceSummaries` call over a 100,000-row active generation must complete in less than 250 milliseconds.
- Do not stage or modify the unrelated Italian-lesson files already dirty in the worktree.

## File Structure

- `internal/storage/migrations/puzzles/004.sql` — append and backfill the generation statistic, then replace the v3 marker.
- `internal/storage/puzzle_schema_signatures.go` — retain exact v3 and v4 logical signatures.
- `internal/storage/puzzle_store.go` — declare v4 current and expose a narrowly scoped existing-store upgrade.
- `internal/storage/legacy_probe.go` — report exact v3 as upgradable and non-legacy.
- `internal/storage/puzzle_upgrade_test.go` — focused v3 recognition, upgrade, backfill, and refusal tests.
- `internal/storage/puzzle_store_test.go` — assert newly created v4 shape and constraints.
- `internal/storage/legacy_puzzle_test.go` — move future-version fixtures from 4 to 5.
- `internal/app/services.go` — route an upgradable catalogue before normal puzzle-store opening.
- `internal/app/task5_startup_red_test.go` — prove normal application startup upgrades v3 without catalogue recreation.
- `internal/puzzles/catalog_import.go` — carry and atomically seal the computed maximum.
- `internal/puzzles/catalog_materialize.go` — read the maximum from the attached winner database.
- `internal/puzzles/catalog_facet_test.go` — cover non-empty and empty generation summary persistence.
- `internal/puzzles/catalog_reader.go` — replace occurrence aggregation with the stored generation scalar.
- `internal/puzzles/catalog_reader_test.go` — keep manual generation fixtures faithful and verify summary semantics/query plan.
- `internal/puzzles/catalog_performance_test.go` — enforce the 250-millisecond large-catalogue gate.

---

### Task 1: Exact Puzzle Schema v3-to-v4 Upgrade

**Files:**
- Create: `internal/storage/migrations/puzzles/004.sql`
- Create: `internal/storage/puzzle_upgrade_test.go`
- Modify: `internal/storage/puzzle_schema_signatures.go`
- Modify: `internal/storage/puzzle_store.go`
- Modify: `internal/storage/legacy_probe.go`
- Modify: `internal/storage/puzzle_store_test.go`
- Modify: `internal/storage/legacy_puzzle_test.go`

**Interfaces:**
- Produces: `PuzzleFileState.Upgradable bool`.
- Produces: `UpgradePuzzleStore(path string) error` accepting only an exact v3 predecessor.
- Produces: schema v4 column `source_generations.maximum_solution_plies INTEGER NOT NULL DEFAULT 0`.
- Consumes: existing `Migrate`, exact logical signatures, physical validation, and `puzzleStoreDSN`.

- [ ] **Step 1: Write failing storage tests for exact predecessor recognition and backfill**

Create `internal/storage/puzzle_upgrade_test.go` with a helper that executes
`currentPuzzleSchemaFixture(t)`, inserts two generations and two cores, then
asserts the new public behavior:

```go
func TestProbeRecognizesExactV3AsUpgradableNonLegacy(t *testing.T) {
	path := populatedPuzzleSchemaV3Fixture(t)

	state, err := ProbePuzzleStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Exists || state.Legacy || !state.Upgradable || state.Format != 3 {
		t.Fatalf("ProbePuzzleStore() = %+v, want exact upgradable v3", state)
	}
}

func TestUpgradePuzzleStoreBackfillsExactGenerationMaximums(t *testing.T) {
	path := populatedPuzzleSchemaV3Fixture(t)

	if err := UpgradePuzzleStore(path); err != nil {
		t.Fatal(err)
	}
	state, err := ProbePuzzleStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if state.Format != 4 || state.Legacy || state.Upgradable {
		t.Fatalf("upgraded state = %+v, want current v4", state)
	}

	db := openPuzzleValidationFixture(t, path)
	for generationID, want := range map[string]int{
		"populated-generation": 5,
		"empty-generation":     0,
	} {
		var got int
		if err := db.QueryRow(`SELECT maximum_solution_plies
			FROM source_generations WHERE generation_id = ?`, generationID).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("generation %q maximum = %d, want %d", generationID, got, want)
		}
	}
	assertV3FixtureContentUnchangedExceptSummary(t, db)
}
```

Add `TestUpgradePuzzleStoreRejectsTamperedV3WithoutChangingIt`, which adds an
unexpected table, hashes the main database before the call, expects an error,
and compares the hash afterward.

- [ ] **Step 2: Run the new tests and verify the red failure**

Run:

```bash
go test ./internal/storage -run 'Test(ProbeRecognizesExactV3AsUpgradableNonLegacy|UpgradePuzzleStore)' -count=1
```

Expected: compile failure because `PuzzleFileState.Upgradable` and
`UpgradePuzzleStore` do not exist.

- [ ] **Step 3: Add migration 004**

Create `internal/storage/migrations/puzzles/004.sql`:

```sql
ALTER TABLE source_generations
ADD COLUMN maximum_solution_plies INTEGER NOT NULL DEFAULT 0
CHECK (maximum_solution_plies >= 0);

UPDATE source_generations
SET maximum_solution_plies = COALESCE((
  SELECT MAX(core.solution_plies)
  FROM puzzle_occurrences AS occurrence
  JOIN puzzle_cores AS core ON core.fingerprint = occurrence.fingerprint
  WHERE occurrence.generation_id = source_generations.generation_id
), 0);

DELETE FROM schema_migrations WHERE version = 3;
```

The migration runner inserts marker 4 in the same transaction after executing
the file, leaving exactly one marker.

- [ ] **Step 4: Represent v3 and v4 as separate exact signatures**

In `internal/storage/puzzle_schema_signatures.go`, rename the existing map to
`puzzleSchemaV3`, derive `currentPuzzleSchema` as an independent map whose
`source_generations` columns append:

```go
puzzleColumnWithDefault("maximum_solution_plies", "INTEGER", 1, 0, "0"),
```

Change recognition to return an upgradable flag:

```go
func recognizedPuzzleSchema(versions []int) (
	puzzleSchemaSignature,
	bool,
	bool,
	bool,
) {
	switch {
	case slices.Equal(versions, []int{1}):
		return legacyPuzzleSchemaV1, true, false, true
	case slices.Equal(versions, []int{1, 2}):
		return legacyPuzzleSchemaV2, true, false, true
	case slices.Equal(versions, []int{3}):
		return puzzleSchemaV3, false, true, true
	case slices.Equal(versions, []int{CurrentPuzzleSchemaVersion}):
		return currentPuzzleSchema, false, false, true
	default:
		return nil, false, false, false
	}
}
```

Do not alias the v3 slice and append into it; both maps must remain immutable
exact signatures.

- [ ] **Step 5: Implement guarded in-place upgrade**

In `internal/storage/legacy_probe.go`, add `Upgradable bool` to
`PuzzleFileState`, receive the fourth recognition result, and set it only after
logical and physical validation succeeds.

In `internal/storage/puzzle_store.go`, set:

```go
const CurrentPuzzleSchemaVersion = 4
```

Add `UpgradePuzzleStore(path string) error`. It must:

1. call `ProbePuzzleStore` and require `Exists && Upgradable && !Legacy && Format == 3`;
2. open the existing file with `puzzleStoreDSN(path, false, false)`;
3. configure a single connection and ping it;
4. re-read migration markers and the logical/physical schema on that same handle, requiring exact v3;
5. call `Migrate(db, "puzzles")`;
6. call `validatePuzzleSchema(db)`;
7. close the database with `errors.Join` so close failures are retained.

Use the same writer cache, auto-checkpoint, timeout, foreign-key, and
synchronous pragmas as `OpenPuzzleStore`; extract a private `openPuzzleWriter`
helper only after the upgrade test is green, then use it from both call sites.

- [ ] **Step 6: Update exact-schema and future-version assertions**

In `internal/storage/puzzle_store_test.go`, expect marker `[4]`, rename the
current-probe test to v4, append `maximum_solution_plies` to the expected
generation columns, and prove a negative value violates its check.

In `internal/storage/legacy_puzzle_test.go`, change fixtures intended to be
newer than the application from version 4 to version 5 and update their
`Found`/`Format` assertions. Keep v3 fixtures at version 3 so the new upgrade
tests exercise the real predecessor.

- [ ] **Step 7: Run storage tests and commit the independently working upgrade**

Run:

```bash
gofmt -w internal/storage/puzzle_schema_signatures.go internal/storage/puzzle_store.go internal/storage/legacy_probe.go internal/storage/puzzle_store_test.go internal/storage/legacy_puzzle_test.go internal/storage/puzzle_upgrade_test.go
go test ./internal/storage -count=1
```

Expected: all storage tests pass, including exact rejection tests.

Commit only Task 1 files:

```bash
git add internal/storage/migrations/puzzles/004.sql internal/storage/puzzle_schema_signatures.go internal/storage/puzzle_store.go internal/storage/legacy_probe.go internal/storage/puzzle_store_test.go internal/storage/legacy_puzzle_test.go internal/storage/puzzle_upgrade_test.go
git commit -m "feat: upgrade puzzle catalogue summary schema"
```

---

### Task 2: Route Exact v3 Through Application Startup

**Files:**
- Modify: `internal/app/services.go`
- Modify: `internal/app/task5_startup_red_test.go`

**Interfaces:**
- Consumes: `PuzzleFileState.Upgradable` and `storage.UpgradePuzzleStore(path)` from Task 1.
- Produces: `preparePuzzleStore` upgrades exact v3 before `OpenPuzzleStore` while preserving v1/v2 recreation.

- [ ] **Step 1: Write a failing full-startup test**

Add `loadTask5PuzzleV3Fixture`, which reads
`internal/storage/migrations/puzzles/003.sql` relative to the test source and
prefixes the single v3 migration marker. Add:

```go
func TestOpenUpgradesExactV3PuzzleCatalogueInPlace(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	v3 := createTask5SQLiteFixture(t, paths.PuzzlesDB, loadTask5PuzzleV3Fixture(t))
	seedTask5ActiveGeneration(t, v3)
	if _, err := v3.Exec(`UPDATE puzzle_cores SET solution_plies = 7
		WHERE fingerprint = 'active-core'`); err != nil {
		t.Fatal(err)
	}
	if err := v3.Close(); err != nil {
		t.Fatal(err)
	}

	services, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()

	var maximum, occurrences int
	if err := services.PuzzleStore.Reader.QueryRow(`SELECT maximum_solution_plies
		FROM source_generations WHERE generation_id = 'active-generation'`).Scan(&maximum); err != nil {
		t.Fatal(err)
	}
	if err := services.PuzzleStore.Reader.QueryRow(`SELECT COUNT(*)
		FROM puzzle_occurrences WHERE generation_id = 'active-generation'`).Scan(&occurrences); err != nil {
		t.Fatal(err)
	}
	if maximum != 7 || occurrences != 1 {
		t.Fatalf("upgraded generation maximum=%d occurrences=%d, want 7 and 1", maximum, occurrences)
	}
}
```

- [ ] **Step 2: Run the startup test and verify the red failure**

Run:

```bash
go test ./internal/app -run TestOpenUpgradesExactV3PuzzleCatalogueInPlace -count=1
```

Expected: startup refuses to open schema v3 after preflight because it has not
yet been routed through the upgrade.

- [ ] **Step 3: Route the upgradable state before legacy handling**

Change `preparePuzzleStore` in `internal/app/services.go` to:

```go
if !state.Exists {
	return nil
}
if state.Upgradable {
	return storage.UpgradePuzzleStore(path)
}
if !state.Legacy {
	return nil
}
```

Keep the existing read-only snapshot backfill and `CloseAndRemove` block below
this branch without other changes.

- [ ] **Step 4: Update the app's future puzzle fixture and run startup coverage**

In `TestOpenPreservesUnknownNewerModifiedAndCorruptPuzzleFiles`, change the
future puzzle marker from 4 to 5. Do not change `internal/app/preflight_test.go`
where version 4 belongs to the independently versioned library store.

Run:

```bash
gofmt -w internal/app/services.go internal/app/task5_startup_red_test.go
go test ./internal/app -count=1
```

Expected: the new startup test and all legacy/recovery tests pass.

- [ ] **Step 5: Commit startup routing**

```bash
git add internal/app/services.go internal/app/task5_startup_red_test.go
git commit -m "feat: upgrade puzzle summaries during startup"
```

---

### Task 3: Persist Maximum Solution Length While Sealing Imports

**Files:**
- Modify: `internal/puzzles/catalog_import.go`
- Modify: `internal/puzzles/catalog_materialize.go`
- Modify: `internal/puzzles/catalog_facet_test.go`

**Interfaces:**
- Produces: private `sqliteGenerationImport.maximumSolutionPlies int`.
- Consumes: attached `generation_winner.winner_rows(solution_plies)`.
- Produces: the seal transition atomically writes the statistic with checksum and timestamp.

- [ ] **Step 1: Write failing non-empty and empty seal tests**

Add to `internal/puzzles/catalog_facet_test.go`:

```go
func TestSealPersistsMaximumSolutionPliesFromGenerationWinners(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("summary-max", "test", "/summary-max")
	short := testTrainingPuzzle(source, "short", 0)
	short.Occurrence.Rating = nil
	short.Core.Solution = linearTestSolution("f7f8", "h8h7")
	short.Core.SolutionPlies = 2
	long := testTrainingPuzzle(source, "long", 0)
	long.Occurrence.Rating = nil
	long.Occurrence.Ordinal = 2
	long.Core.Solution = linearTestSolution(
		"f7f8", "h8h7", "f8f7", "h7h8", "f7f8", "h8h7", "f8f7",
	)
	long.Core.SolutionPlies = 7

	importing := beginGenerationImport(t, catalog, source)
	if err := importing.Add(context.Background(), short); err != nil {
		t.Fatal(err)
	}
	if err := importing.Add(context.Background(), long); err != nil {
		t.Fatal(err)
	}
	if _, err := importing.Seal(context.Background(), "summary-max-checksum"); err != nil {
		t.Fatal(err)
	}
	assertGenerationMaximumSolutionPlies(t, store.Reader, source.Path, 7)
}

func TestSealPersistsZeroMaximumForEmptyGeneration(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("summary-empty", "test", "/summary-empty")
	importing := beginGenerationImport(t, catalog, source)
	if _, err := importing.Seal(context.Background(), "summary-empty-checksum"); err != nil {
		t.Fatal(err)
	}
	assertGenerationMaximumSolutionPlies(t, store.Reader, source.Path, 0)
}

func linearTestSolution(moves ...string) []domain.MoveNode {
	var branch []domain.MoveNode
	for index := len(moves) - 1; index >= 0; index-- {
		branch = []domain.MoveNode{{UCI: moves[index], Children: branch}}
	}
	return branch
}
```

Add `chess-trainer/internal/domain` to the test imports. The helper keeps each
solution tree's normalized move count consistent with `SolutionPlies`.

- [ ] **Step 2: Run the seal tests and verify the red failure**

Run:

```bash
go test ./internal/puzzles -run 'TestSealPersists(MaximumSolutionPliesFromGenerationWinners|ZeroMaximumForEmptyGeneration)' -count=1
```

Expected: the non-empty test reads zero rather than seven.

- [ ] **Step 3: Read the winner statistic before detach**

Add `maximumSolutionPlies int` to `sqliteGenerationImport`. In
`materializeWinner`, immediately after attaching and validating the winner
database, scan:

```go
if err := s.catalog.writeDB.QueryRowContext(
	ctx,
	`SELECT COALESCE(MAX(solution_plies), 0)
	 FROM generation_winner.winner_rows`,
).Scan(&s.maximumSolutionPlies); err != nil {
	return fmt.Errorf("read generation maximum solution plies: %w", err)
}
```

- [ ] **Step 4: Make statistic persistence part of the seal transition**

Change the conditional update in `sealMaterializedGeneration` to:

```go
`UPDATE source_generations
 SET status = 'sealed', checksum = ?, sealed_at = ?, maximum_solution_plies = ?
 WHERE generation_id = ? AND source_id = ? AND status = 'building'`
```

Pass `s.maximumSolutionPlies` before generation and source IDs. Do not write the
statistic from a separate transaction.

- [ ] **Step 5: Run puzzle import and materialization tests, then commit**

Run:

```bash
gofmt -w internal/puzzles/catalog_import.go internal/puzzles/catalog_materialize.go internal/puzzles/catalog_facet_test.go
go test ./internal/puzzles -run 'Test(Seal|Generation|Materialize|ActiveThemes)' -count=1
```

Expected: all selected tests pass.

Commit:

```bash
git add internal/puzzles/catalog_import.go internal/puzzles/catalog_materialize.go internal/puzzles/catalog_facet_test.go
git commit -m "feat: persist generation solution summary"
```

---

### Task 4: Make Active Source Summaries Constant-Time

**Files:**
- Modify: `internal/puzzles/catalog_reader.go`
- Modify: `internal/puzzles/catalog_reader_test.go`
- Modify: `internal/puzzles/catalog_performance_test.go`

**Interfaces:**
- Consumes: sealed `source_generations.maximum_solution_plies` from Tasks 1 and 3.
- Produces: unchanged `ActiveSourceSummaries(context.Context) ([]SourceSummary, error)` behavior.
- Produces: package-private `activeSourceSummariesSQL` for execution and query-plan regression testing.

- [ ] **Step 1: Write a failing query-plan regression test**

Extract the current query text into `activeSourceSummariesSQL` only after the
test has first failed to compile. Add:

```go
func TestActiveSourceSummaryPlanDoesNotVisitOccurrenceOrCoreRows(t *testing.T) {
	_, store := openTestGenerationalCatalog(t)
	rows, err := store.Reader.Query(
		`EXPLAIN QUERY PLAN `+activeSourceSummariesSQL,
		nullPuzzleRatingKey,
		nullPuzzleRatingKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var details []string
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		details = append(details, detail)
	}
	plan := strings.ToLower(strings.Join(details, "\n"))
	for _, forbidden := range []string{"puzzle_occurrences", "puzzle_cores", "temp b-tree for group by"} {
		if strings.Contains(plan, forbidden) {
			t.Fatalf("source summary plan visits %q:\n%s", forbidden, plan)
		}
	}
}
```

- [ ] **Step 2: Run the plan test and verify the red failure**

Run:

```bash
go test ./internal/puzzles -run TestActiveSourceSummaryPlanDoesNotVisitOccurrenceOrCoreRows -count=1
```

Expected: compile failure before the query constant is extracted, then a
behavioral failure showing occurrence/core access once the existing query is
assigned to the constant.

- [ ] **Step 3: Replace aggregation with the generation scalar**

Define `activeSourceSummariesSQL` in `catalog_reader.go` with the same rating
subqueries and this outer shape:

```sql
SELECT head.source_id,
       source.kind,
       (SELECT rated.rating_key
        FROM occurrence_ratings AS rated
        WHERE rated.generation_id = head.generation_id
          AND rated.rating_key <> ?
        ORDER BY rated.rating_key, rated.fingerprint
        LIMIT 1),
       (SELECT rated.rating_key
        FROM occurrence_ratings AS rated
        WHERE rated.generation_id = head.generation_id
          AND rated.rating_key <> ?
        ORDER BY rated.rating_key DESC, rated.fingerprint DESC
        LIMIT 1),
       generation.maximum_solution_plies
FROM source_heads AS head
JOIN source_generations AS generation
  ON generation.source_id = head.source_id
 AND generation.generation_id = head.generation_id
JOIN sources AS source ON source.source_id = head.source_id
WHERE generation.status = 'sealed'
ORDER BY head.source_id
```

Remove the occurrence/core joins and `GROUP BY`. Execute the constant from
`ActiveSourceSummaries`; leave response scanning unchanged.

- [ ] **Step 4: Update manual reader fixtures and semantic assertions**

In `seedActiveReaderGeneration`, calculate the maximum from the supplied
`TrainingPuzzle` values and include it in the sealed generation insert. Preserve
zero for no puzzles.

Extend `TestActiveSourceSummaryRatingBoundsUseOrderedMembership` to assert the
stored maximum as well as the rating bounds. Existing active-head and
multi-format tests must continue proving rated and unrated summaries retain
their current response shape.

- [ ] **Step 5: Add the 100,000-row performance gate**

In `seedSyntheticSealedGeneration`, set `maximum_solution_plies = 1` when
sealing its synthetic rows. Add to the performance-tagged file:

```go
func TestActiveSourceSummariesCompleteWithin250Milliseconds(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	ctx := context.Background()
	source := testSource("summary-performance", "synthetic", "/summary-performance")
	active, _ := seedSyntheticSealedGeneration(
		t, catalog, store, source, "summary-performance", syntheticPerformanceRows,
	)
	if err := active.Activate(ctx); err != nil {
		t.Fatal(err)
	}
	warmPuzzleReadPool(t, store.Reader)
	if _, err := catalog.ActiveSourceSummaries(ctx); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	summaries, err := catalog.ActiveSourceSummaries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	if elapsed >= 250*time.Millisecond {
		t.Fatalf("active source summaries took %s, want less than 250ms", elapsed)
	}
	if len(summaries) != 1 || summaries[0].MaximumSolutionPlies != 1 {
		t.Fatalf("summaries = %+v, want one source with maximum 1", summaries)
	}
}
```

- [ ] **Step 6: Run query, semantic, and performance tests**

Run:

```bash
gofmt -w internal/puzzles/catalog_reader.go internal/puzzles/catalog_reader_test.go internal/puzzles/catalog_performance_test.go
go test ./internal/puzzles -run 'Test(ActiveSource|CandidatesAndMetadata|MultiFormat)' -count=1
go test -tags performance ./internal/puzzles -run 'TestActiveSourceSummariesCompleteWithin250Milliseconds' -count=1
```

Expected: unit tests pass; the query-plan test reports no occurrence/core/temp
grouping work; the performance test passes below 250 milliseconds.

- [ ] **Step 7: Commit the constant-time read and regression gates**

```bash
git add internal/puzzles/catalog_reader.go internal/puzzles/catalog_reader_test.go internal/puzzles/catalog_performance_test.go
git commit -m "perf: make puzzle source summaries constant time"
```

---

### Task 5: Full Verification and Application Rebuild

**Files:**
- Verify only; no expected source changes.
- Preserve the unrelated Italian-lesson worktree changes.

**Interfaces:**
- Consumes: complete v4 upgrade, startup, import, and query slices.
- Produces: rebuilt `build/bin/Chess Trainer.app`.

- [ ] **Step 1: Verify focused backend behavior without cached results**

Run:

```bash
go test ./internal/storage ./internal/puzzles ./internal/app ./internal/profile -count=1
go test -tags performance ./internal/puzzles -count=1
```

Expected: all focused and performance tests pass.

- [ ] **Step 2: Run repository-wide Go verification**

Run:

```bash
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
```

Expected: all packages pass, race detector reports no races, and vet reports no
diagnostics.

- [ ] **Step 3: Verify the unchanged frontend and legal build inputs**

Run:

```bash
npm --prefix frontend test -- --run --threads=false
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
```

Expected: Vitest, Svelte/TypeScript checks, Vite build, and license verification
all pass. Existing Italian-lesson tests run against the preserved dirty files.

- [ ] **Step 4: Rebuild the macOS application**

Run:

```bash
wails build -clean
```

Expected: Wails succeeds and refreshes
`build/bin/Chess Trainer.app` without changing tracked source files.

- [ ] **Step 5: Audit scope and record measured evidence**

Run:

```bash
git status --short
git log -5 --oneline
git diff --check HEAD~3..HEAD
```

Expected: the puzzle-performance commits contain only the files named by Tasks
1–4; the pre-existing Italian files remain dirty and unstaged; no whitespace
errors are reported. Report the measured performance-test duration and note the
one-time v3 upgrade expectation separately from recurring filter latency.
