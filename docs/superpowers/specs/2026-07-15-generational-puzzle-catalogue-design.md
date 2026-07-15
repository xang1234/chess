# Generational Puzzle Catalogue Remediation Design

**Date:** 2026-07-15

## Purpose

Replace the current staging-and-merge puzzle catalogue with a generational catalogue that remains readable during imports, activates a completed source in constant time, survives interrupted imports, and gives source-specific metadata one authoritative owner.

This is the first remediation slice from the thermo-nuclear review. It addresses the full-dataset import failure, abandoned staging data, nondeterministic concurrent imports, conflicting canonical/source metadata, source-unaware training retrieval, catalogue startup cost, and the performance assertion that makes the race suite unreliable.

## Scope

This slice will:

- Recreate `puzzles.sqlite` with a generational schema. The existing puzzle catalogue is explicitly disposable and must be reimported.
- Preserve `user.sqlite`, including settings, sessions, attempts, review state, and rating history.
- Separate fingerprint-stable chess content from source-generation occurrences.
- Make active catalogue reads source-aware.
- Snapshot source-specific training metadata into `user.sqlite` so later reimports cannot rewrite history.
- Permit only one puzzle-catalogue import job at a time.
- Replace the Lichess-specific job entry point with a typed, content-neutral puzzle import request.
- Make activation a small pointer-switch transaction.
- Detect abandoned imports at startup and clean abandoned or superseded generations in bounded batches.
- Remove full-catalogue integrity scanning from normal startup.
- Preserve the existing streaming, bounded-memory, cancellation, progress, and import-report behavior.

This slice will not:

- Add the tactical PGN importer or game library.
- Redesign learner-action transaction boundaries; that is remediation slice 2.
- Redesign backup restoration; that is remediation slice 3.
- Redesign the root frontend state machine or generated Wails adapter; that is remediation slice 4.
- Add an engine implementation or engine binary.

## Approaches Considered

### Generational catalogue in one SQLite database

Each import writes an inactive generation. Active reads join through a small source-head table. A successful import switches the head pointer in one transaction, and cleanup removes unreferenced generations later.

This approach is selected because it retains cross-source deduplication and indexed queries while making incomplete data invisible and activation cheap.

### One SQLite database per source generation

Swapping files would provide strong write isolation, but cross-source deduplication and filtering would require attached databases or an external index. Backup, cleanup, and query composition would become more complicated than the local application warrants.

### Optimize the existing staging merge

The existing merge could be tuned with more indexes and temporary tables, but it would retain a large activation transaction, shared mutable staging, crash-leak cleanup, and split source semantics. The real six-million-row acceptance run already timed out after one hour in this path.

## Domain Model

### PuzzleCore

`PuzzleCore` owns only data covered by the stable fingerprint:

- Fingerprint.
- Displayed FEN.
- Solver color.
- Normalized solution tree.
- Solution ply count used by indexed filters.

The fingerprint remains the SHA-256 digest of the existing JSON payload: trimmed displayed FEN, solver color, and the recursively normalized solution tree.

### PuzzleOccurrence

`PuzzleOccurrence` owns data supplied by one source generation:

- Logical source ID and kind. Generation ID is persistence identity stamped by the generation writer; it is not exposed to import adapters or training consumers.
- Source-local external ID.
- Optional source FEN and presentation prelude.
- Rating, popularity, and play count.
- Source URL, attribution, and raw metadata.
- Source-specific themes.
- Input ordinal for deterministic duplicate handling and reporting.

Source FEN and prelude are occurrence data because two sources can reach the same displayed position and solution through different presentations.

### TrainingPuzzle

`TrainingPuzzle` is the source-aware projection consumed by scheduling and solving. It contains one `PuzzleCore` and exactly one active `PuzzleOccurrence`. It is addressed by `PuzzleKey{Fingerprint, SourceID}` rather than by fingerprint alone.

Rated and free-practice candidate queries return `TrainingPuzzle` values. Global rated candidates choose at most one active occurrence per fingerprint before applying the result limit, using lexical source ID as the deterministic tie-break; source-scoped practice is already unique per fingerprint. Session items persist fingerprint and source ID and snapshot the selected occurrence's session-visible fields. Move, hint, reveal, resume, and completion paths validate the exact active fingerprint/source key and use its stable core, while stored source presentation, themes, and rating remain authoritative for that queued session. This avoids pinning obsolete generations while preventing a same-source reimport from rewriting an in-progress session.

## SQLite Schema

The recreated catalogue contains these logical tables:

### `sources`

- `source_id TEXT PRIMARY KEY`
- `kind TEXT NOT NULL`

This table is the stable identity of a configured source. Its kind is immutable: reusing a source ID with a different kind is an error. Import path, checksum, and timestamps belong to a generation, not to the logical source.

### `source_generations`

- `generation_id TEXT PRIMARY KEY`
- `source_id TEXT NOT NULL REFERENCES sources(source_id)`
- `status TEXT NOT NULL CHECK(status IN ('building', 'sealed', 'abandoned'))`
- `source_path TEXT NOT NULL`
- `checksum TEXT`
- `started_at INTEGER NOT NULL`
- `sealed_at INTEGER`
- `UNIQUE(source_id, generation_id)`

`building` is writable and invisible. `sealed` is immutable and eligible to become a source head. `abandoned` is invisible and eligible for cleanup. Active status is deliberately not duplicated here.

### `source_heads`

- `source_id TEXT PRIMARY KEY REFERENCES sources(source_id)`
- `generation_id TEXT NOT NULL`
- Composite foreign key `(source_id, generation_id)` to `source_generations`.

This pointer is the sole authority for which generation is active. A sealed generation not referenced by `source_heads` is superseded and eligible for cleanup.

### `puzzle_cores`

- `fingerprint TEXT PRIMARY KEY`
- `displayed_fen TEXT NOT NULL`
- `solver TEXT NOT NULL`
- `solution_json TEXT NOT NULL`
- `solution_plies INTEGER NOT NULL`

Inserting an existing fingerprint must verify that its canonical fields match. A mismatch is treated as catalogue corruption rather than as an ordinary duplicate.

### `puzzle_occurrences`

- `generation_id TEXT NOT NULL REFERENCES source_generations(generation_id) ON DELETE CASCADE`
- `fingerprint TEXT NOT NULL REFERENCES puzzle_cores(fingerprint)`
- Source FEN, prelude, external ID, rating, popularity, play count, URL, attribution, metadata, and ordinal columns.
- Primary key `(generation_id, fingerprint)`.

The last valid record for a repeated fingerprint within one input file wins deterministically. The report counts the first retained occurrence as accepted and subsequent input records for the same generation and fingerprint as duplicates. A canonical puzzle already present through another source is still an accepted occurrence because its source-specific provenance is retained.

### `occurrence_themes`

- `generation_id TEXT NOT NULL`
- `fingerprint TEXT NOT NULL`
- `theme TEXT NOT NULL`
- Primary key `(generation_id, fingerprint, theme)`.
- Composite foreign key to `puzzle_occurrences` with cascading deletion.

Indexes cover active-generation rating queries, active-generation theme queries, generation cleanup, and fingerprint lookup. Schema DDL lives only in migrations; repositories do not recreate migration-owned indexes at runtime.

## Import Lifecycle

The import coordinator accepts a typed request containing `Kind`, `SourceID`, and `Path`. Lichess is the first registered adapter, but the coordinator does not hard-code it.

Only one puzzle-catalogue import may run at a time. A second start request returns a typed busy error containing the active job ID. Job state remains queryable through a result snapshot even if an event listener misses progress or completion. The coordinator also owns a single writer gate and a cancellable maintenance worker: imports hold the gate for their lifetime, cleanup takes it for one bounded batch at a time, and a newly reserved import stops further cleanup batches.

The writer performs these steps:

1. In one transaction, validate/insert the immutable source identity, capture the prior head, and create a `building` generation before reading source rows.
2. Stream and validate the source without creating a decompressed file.
3. Write cores, occurrences, and occurrence themes in bounded transactions.
4. Count rejected records and repeated generation-local fingerprints while retaining at most 100 rejection examples.
5. Finish the checksum and seal the generation.
6. In one short transaction, compare the expected previous head and switch `source_heads` to the sealed generation.
7. Emit the terminal job snapshot and trigger the bounded cleanup worker.

The compare-and-swap condition prevents an obsolete writer from replacing a newer head even if job exclusion is later relaxed. Activation never deletes live rows or rebuilds global indexes. Service shutdown first prevents new jobs, cancels active import and maintenance contexts, waits for their registered goroutines, and only then permits SQLite handles to close.

The puzzle store exposes separate handles for reads and writes against the same WAL-mode database. The writer handle has exactly one connection and serializes import, activation, and cleanup work. The reader handle has a small bounded pool so training queries can continue against the prior source head while the writer commits batches. Every public catalogue read uses one SQL statement or one read transaction so activation and cleanup cannot mix rows from different snapshots. Connection-scoped foreign-key and busy-timeout pragmas are configured for every connection rather than only for the first connection opened. Cleanup runs only when no import is writing.

## Failure and Crash Semantics

- Parse, validation, disk, or database failures leave the existing head unchanged.
- Cancellation marks the generation abandoned and returns promptly; physical deletion is asynchronous.
- Startup marks every leftover `building` generation abandoned because no import goroutine survives process restart.
- Startup acquires a data-root single-instance lock before recovery, so a second live process cannot abandon the first process's import.
- Cleanup deletes occurrence/theme rows in bounded batches, yielding between transactions.
- Cleanup removes a core only when no occurrence in any generation references it.
- Sealed generations not referenced by a source head are safe to clean.
- Cleanup is idempotent. A crash during cleanup resumes from remaining rows on the next run.
- Active reads always join through `source_heads`, so building, abandoned, and superseded generations are invisible.

## User-History Preservation

`user.sqlite` receives an additive migration. Attempts snapshot the fields required for durable reporting:

- Existing fingerprint and source ID.
- Source kind.
- Rating at the time the puzzle was scheduled.
- Normalized source themes as JSON.

The same reporting snapshot columns are stored on every queued `session_item`, together with source FEN and presentation prelude used by the puzzle view. Only the current item has an attempt row, so later attempts must copy the metadata selected when the session was scheduled rather than consulting a possibly reimported catalogue. Attempt rows do not need source FEN or prelude because those fields are not used for historical reporting.

The new columns are nullable for schema compatibility. Before a recognized legacy puzzle catalogue is deleted, startup performs a one-time cross-store backfill for legacy attempts and queued session items whose snapshot is null. It reads source kind, rating, themes, source FEN, and prelude from the matching legacy fingerprint/source rows, storing presentation fields only on session items. A valid legacy catalogue therefore retains its existing historical theme attribution and queued presentation through recreation. If a legacy row has no matching occurrence, its snapshot remains null and reporting omits unknown themes instead of inventing attribution from a future replacement catalogue.

Review state also records a preferred source ID. Existing rows receive a null preferred source and choose a deterministic active occurrence when next scheduled. New or updated review rows retain the occurrence used by the attempt. If that source no longer contains the fingerprint, scheduling may choose another active occurrence for the same core; if none exists, the review remains dormant until the fingerprint returns.

Parent theme metrics read attempt snapshots from `user.sqlite`; they no longer reconstruct history from the mutable current puzzle catalogue.

## Legacy Catalogue Recreation and Startup

The application acquires a data-root single-instance lock, then opens and migrates `user.sqlite` before replacing the puzzle catalogue. If the puzzle store has the exactly recognized legacy schema and migration set, it backfills null attempt and session-item snapshots, closes the legacy handle, and only then recreates `puzzles.sqlite`.

Puzzle-store startup performs a cheap schema/version probe. A known legacy puzzle schema is closed, its database and WAL/SHM sidecars are removed, and a fresh generational catalogue is created. This behavior is authorized only for `puzzles.sqlite`; no user or game-library database is rebuilt by this path.

An unknown newer schema is never deleted. It produces a typed incompatibility error.

Normal startup does not run `PRAGMA quick_check` across the entire puzzle catalogue. SQLite open, migration, and required-table probes provide the normal fast path. Full integrity checking belongs to explicit diagnostics, restore validation, or the later unclean-shutdown design.

## Code Boundaries

The current catalogue implementation is decomposed by responsibility:

- `internal/puzzles/catalog_models.go`: puzzle core, occurrence, key, and training projection.
- `internal/puzzles/catalog_reader.go`: active source-aware retrieval and candidate queries.
- `internal/puzzles/catalog_import.go`: generation creation, batched writes, sealing, and activation.
- `internal/puzzles/catalog_cleanup.go`: crash recovery and bounded garbage collection.
- `internal/puzzles/catalog.go`: public interfaces and import report types only.
- `internal/importjob/service.go`: typed request routing, global puzzle-job exclusion, cancellation, and snapshots.
- `internal/storage/puzzle_store.go`: cheap probe, known-legacy recreation, and the bounded read/serialized-write handles.

The generated Wails and frontend import-state redesign is deliberately deferred to remediation slice 4. This slice may add the typed backend binding while preserving the current Lichess-specific public method as a temporary adapter until that frontend slice replaces it.

## Testing Strategy

Implementation follows red-green-refactor. Every behavior change begins with a failing test.

### Catalogue tests

- Building generations are invisible to active reads.
- A reader can retrieve the prior active generation while an import batch transaction is open.
- Activation switches the source head without deleting or rebuilding indexes.
- An aborted generation leaves the prior head readable.
- Startup abandons a simulated crashed generation.
- Bounded cleanup resumes after interruption and preserves referenced cores.
- Same-core occurrences retain independent rating, themes, provenance, source FEN, and prelude.
- Training lookup requires fingerprint plus source ID.
- Reimport removes only the replaced source generation from active queries.
- Generation-local duplicates are counted deterministically.
- A canonical fingerprint/data mismatch fails as corruption.

### Job tests

- A second puzzle import is rejected while one is active.
- Request kind chooses the registered importer without hard-coded source IDs.
- Result snapshots progress monotonically to one terminal state.
- Cancellation marks the generation abandoned and preserves the active head.

### User-history tests

- Attempt snapshots survive source reimport and catalogue recreation.
- Parent theme metrics use snapshots rather than the current catalogue.
- Review scheduling prefers its recorded source and falls back deterministically.

### Startup tests

- A known legacy puzzle catalogue is recreated.
- `user.sqlite` is byte-for-byte untouched by puzzle recreation.
- An unknown newer catalogue is preserved and rejected.
- Normal startup does not invoke a full puzzle integrity scan.

### Performance tests

- The existing 20,000-row wall-clock assertion moves out of the race-enabled correctness suite into an explicit performance test or benchmark.
- Synthetic activation is measured separately from ingestion and must complete within five seconds.
- Candidate retrieval remains below 250 ms under representative local load.
- The opt-in real Lichess test must import more than one million accepted occurrences, retain bounded heap/RSS growth, create no decompressed CSV, activate within five seconds, and complete within its one-hour timeout.

## Acceptance Criteria

- The real downloaded Lichess database imports and activates within the one-hour acceptance-test timeout.
- Training and indexed reads remain available from the previous head throughout ingestion.
- A failed, cancelled, or process-interrupted import never changes active source content.
- Activation performs no source-wide delete, catalogue-wide merge, or global index rebuild.
- Additional puzzle sources can retain independent metadata for a shared canonical fingerprint.
- Attempts and parent theme history remain stable after source reimport or complete puzzle-catalogue recreation.
- Queued session items retain source kind, rating, themes, source FEN, and prelude selected at scheduling time, even when their attempts start after a restart or same-source reimport.
- Normal startup does not scan the complete puzzle catalogue.
- `go test ./...`, the serial frontend test suite, the frontend production build, and the race suite pass; explicit performance tests run separately from the race suite.
