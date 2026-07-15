package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

//go:embed testdata/legacy_puzzles_v1.sql
var legacyPuzzlesV1Fixture string

//go:embed testdata/legacy_puzzles_v2.sql
var legacyPuzzlesV2Fixture string

func TestProbeRecognizesExactLegacyV1AndV2(t *testing.T) {
	fixtures := []struct {
		version int
		sql     string
	}{
		{version: 1, sql: legacyPuzzlesV1Fixture},
		{version: 2, sql: legacyPuzzlesV2Fixture},
	}
	for _, fixture := range fixtures {
		t.Run(fmt.Sprintf("v%d", fixture.version), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "puzzles #?.sqlite")
			db := createSQLiteFixture(t, path, fixture.sql)
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}

			state, err := ProbePuzzleStore(path)
			if err != nil {
				t.Fatal(err)
			}
			if !state.Exists || !state.Legacy || state.Format != fixture.version {
				t.Fatalf("ProbePuzzleStore() = %+v, want existing legacy v%d", state, fixture.version)
			}

			legacy, err := OpenLegacyPuzzleReadOnly(path)
			if err != nil {
				t.Fatal(err)
			}
			defer legacy.Close()
			if _, err := legacy.Exec(`DELETE FROM sources`); err == nil {
				t.Fatal("OpenLegacyPuzzleReadOnly() returned a writable database")
			}
		})
	}
}

func TestProbeRejectsLegacyVersionWithChangedColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "puzzles.sqlite")
	db := createSQLiteFixture(t, path, legacyPuzzlesV1Fixture)
	if _, err := db.Exec(`ALTER TABLE puzzles ADD COLUMN unexpected TEXT`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	wantHash := hashFile(t, path)

	state, err := ProbePuzzleStore(path)
	if err == nil {
		t.Fatal("ProbePuzzleStore() accepted a changed legacy schema")
	}
	if !state.Exists || state.Legacy || state.Format != 1 {
		t.Fatalf("ProbePuzzleStore() state = %+v, want rejected existing format 1", state)
	}
	if gotHash := hashFile(t, path); gotHash != wantHash {
		t.Fatal("ProbePuzzleStore() changed the rejected legacy database")
	}
}

func TestProbeRejectsLegacyVersionWithLookalikeSQLiteTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "puzzles.sqlite")
	db := createSQLiteFixture(t, path, legacyPuzzlesV2Fixture)
	if _, err := db.Exec(`CREATE TABLE sqliteXextension(value TEXT)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	wantHash := hashFile(t, path)

	state, err := ProbePuzzleStore(path)
	if err == nil {
		t.Fatal("ProbePuzzleStore() accepted a legacy schema with a lookalike SQLite table")
	}
	if !state.Exists || state.Legacy || state.Format != 2 {
		t.Fatalf("ProbePuzzleStore() state = %+v, want rejected existing format 2", state)
	}
	if gotHash := hashFile(t, path); gotHash != wantHash {
		t.Fatal("ProbePuzzleStore() changed the rejected legacy database")
	}
}

func TestProbeRejectsLegacyVersionWithGeneratedColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "puzzles.sqlite")
	db := createSQLiteFixture(t, path, legacyPuzzlesV1Fixture)
	if _, err := db.Exec(`ALTER TABLE puzzles ADD COLUMN derived_fingerprint TEXT
		GENERATED ALWAYS AS (fingerprint) VIRTUAL`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	wantHash := hashFile(t, path)

	state, err := ProbePuzzleStore(path)
	if err == nil {
		t.Fatal("ProbePuzzleStore() accepted a legacy schema with a generated column")
	}
	if !state.Exists || state.Legacy || state.Format != 1 {
		t.Fatalf("ProbePuzzleStore() state = %+v, want rejected existing format 1", state)
	}
	if gotHash := hashFile(t, path); gotHash != wantHash {
		t.Fatal("ProbePuzzleStore() changed the rejected legacy database")
	}
}

func TestProbePreservesUnknownNewerDatabaseAndSidecars(t *testing.T) {
	path := filepath.Join(t.TempDir(), "puzzles.sqlite")
	createCrashSQLiteFixture(t, path, puzzleSchemaV3+`
INSERT INTO schema_migrations(version) VALUES (4);
CREATE TABLE future_catalogue_state(value TEXT);
INSERT INTO future_catalogue_state(value) VALUES ('keep me');
`)
	want := snapshotPreservedPuzzleFiles(t, path)

	state, err := ProbePuzzleStore(path)
	var versionErr *PuzzleSchemaVersionError
	if !errors.As(err, &versionErr) {
		t.Fatalf("ProbePuzzleStore() err = %v, want PuzzleSchemaVersionError", err)
	}
	if versionErr.Path != path || versionErr.Found != 4 || versionErr.Supported != CurrentPuzzleSchemaVersion {
		t.Fatalf("PuzzleSchemaVersionError = %+v", versionErr)
	}
	if !state.Exists || state.Legacy || state.Format != 4 {
		t.Fatalf("ProbePuzzleStore() state = %+v, want rejected existing format 4", state)
	}
	assertPuzzleFilesPreserved(t, path, want)

	if err := RemoveRecognizedLegacyPuzzleStore(path); err == nil {
		t.Fatal("RemoveRecognizedLegacyPuzzleStore() removed a newer database")
	}
	assertPuzzleFilesPreserved(t, path, want)
}

func TestProbePreservesCorruptOrUnversionedDatabaseAndSidecars(t *testing.T) {
	t.Run("corrupt", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "puzzles.sqlite")
		if err := os.WriteFile(path, []byte("not a sqlite database"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path+"-wal", []byte("unrecognized wal"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path+"-shm", []byte("unrecognized shm"), 0o600); err != nil {
			t.Fatal(err)
		}
		assertRejectedPuzzleFilesPreserved(t, path)
	})

	t.Run("unversioned", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "puzzles.sqlite")
		createCrashSQLiteFixture(t, path, `
CREATE TABLE unrelated(value TEXT);
INSERT INTO unrelated(value) VALUES ('keep me');
`)
		assertRejectedPuzzleFilesPreserved(t, path)
	})
}

func TestRemoveRecognizedLegacyRefusesEveryOtherState(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "puzzles.sqlite")
		if err := RemoveRecognizedLegacyPuzzleStore(path); err == nil {
			t.Fatal("RemoveRecognizedLegacyPuzzleStore() accepted a missing database")
		}
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("removal created the missing database: %v", err)
		}
	})

	t.Run("empty", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "puzzles.sqlite")
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		wantHash := hashFile(t, path)
		if err := RemoveRecognizedLegacyPuzzleStore(path); err == nil {
			t.Fatal("RemoveRecognizedLegacyPuzzleStore() accepted an empty database")
		}
		if gotHash := hashFile(t, path); gotHash != wantHash {
			t.Fatal("removal changed the empty database")
		}
	})

	t.Run("current", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "puzzles.sqlite")
		openTestGenerationPuzzleStore(t, path)
		wantHash := hashFile(t, path)
		if err := RemoveRecognizedLegacyPuzzleStore(path); err == nil {
			t.Fatal("RemoveRecognizedLegacyPuzzleStore() accepted the current database")
		}
		if gotHash := hashFile(t, path); gotHash != wantHash {
			t.Fatal("removal changed the current database")
		}
	})

	t.Run("modified legacy", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "puzzles.sqlite")
		db := createSQLiteFixture(t, path, legacyPuzzlesV2Fixture)
		if _, err := db.Exec(`CREATE TABLE local_extension(value TEXT)`); err != nil {
			t.Fatal(err)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		wantHash := hashFile(t, path)
		if err := RemoveRecognizedLegacyPuzzleStore(path); err == nil {
			t.Fatal("RemoveRecognizedLegacyPuzzleStore() accepted a modified legacy database")
		}
		if gotHash := hashFile(t, path); gotHash != wantHash {
			t.Fatal("removal changed the modified legacy database")
		}
	})

	t.Run("exact legacy", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "puzzles.sqlite")
		createCrashSQLiteFixture(t, path, legacyPuzzlesV2Fixture)
		if err := RemoveRecognizedLegacyPuzzleStore(path); err != nil {
			t.Fatal(err)
		}
		for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
			if _, err := os.Stat(candidate); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("removed legacy path %s still exists: %v", candidate, err)
			}
		}
	})
}

func TestRemoveRecognizedLegacyPreservesPathReplacement(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "puzzles.sqlite")
	legacy := createSQLiteFixture(t, path, legacyPuzzlesV2Fixture)
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	pending, err := prepareRecognizedLegacyRemoval(path)
	if err != nil {
		t.Fatal(err)
	}

	legacyBackup := filepath.Join(root, "validated-legacy.sqlite")
	if err := os.Rename(path, legacyBackup); err != nil {
		t.Fatal(err)
	}
	legacyHash := hashFile(t, legacyBackup)
	replacement := filepath.Join(root, "replacement.sqlite")
	createCrashSQLiteFixture(t, replacement, puzzleSchemaV3+`
INSERT INTO schema_migrations(version) VALUES (4);
CREATE TABLE future_catalogue_state(value TEXT);
INSERT INTO future_catalogue_state(value) VALUES ('preserve replacement');
`)
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := os.Rename(replacement+suffix, path+suffix); err != nil {
			t.Fatal(err)
		}
	}
	wantReplacement := snapshotPreservedPuzzleFiles(t, path)

	if err := pending.remove(); err == nil {
		t.Fatal("validated legacy removal deleted a newer path replacement")
	}
	assertPuzzleFilesPreserved(t, path, wantReplacement)
	if got := hashFile(t, legacyBackup); got != legacyHash {
		t.Fatal("validated legacy backup changed during rejected removal")
	}
	state, err := ProbePuzzleStore(path)
	var versionErr *PuzzleSchemaVersionError
	if !errors.As(err, &versionErr) || state.Format != 4 {
		t.Fatalf("restored replacement probe = %+v, %v; want newer format 4", state, err)
	}
}

func TestRemoveRecognizedLegacyPreservesExactLegacyPathReplacement(t *testing.T) {
	fixtures := []struct {
		name string
		sql  string
	}{
		{name: "v1", sql: legacyPuzzlesV1Fixture},
		{name: "v2", sql: legacyPuzzlesV2Fixture},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "puzzles.sqlite")
			createMarkedLegacyCrashSQLiteFixture(t, path, fixture.sql, "validated-"+fixture.name)

			pending, err := prepareRecognizedLegacyRemoval(path)
			if err != nil {
				t.Fatal(err)
			}

			validatedBackup := filepath.Join(root, "validated-legacy.sqlite")
			for _, suffix := range []string{"", "-wal", "-shm"} {
				if err := os.Rename(path+suffix, validatedBackup+suffix); err != nil {
					t.Fatal(err)
				}
			}
			wantValidated := snapshotPreservedPuzzleFiles(t, validatedBackup)

			createMarkedLegacyCrashSQLiteFixture(t, path, fixture.sql, "replacement-"+fixture.name)
			wantReplacement := snapshotPreservedPuzzleFiles(t, path)
			if wantReplacement.database == wantValidated.database {
				t.Fatal("test setup produced byte-identical validated and replacement databases")
			}
			if wantReplacement.wal == wantValidated.wal {
				t.Fatal("test setup produced byte-identical validated and replacement WALs")
			}

			if err := pending.remove(); err == nil {
				t.Fatal("validated legacy removal deleted an exact-legacy path replacement")
			}
			assertPuzzleFilesPreserved(t, path, wantReplacement)
			assertPuzzleFilesPreserved(t, validatedBackup, wantValidated)
		})
	}
}

func TestBackfillLegacySnapshotsMatchesFingerprintAndSource(t *testing.T) {
	legacy := openPopulatedLegacyDatabase(t, func(t *testing.T, tx *sql.Tx) {
		insertLegacyPuzzle(t, tx, "shared", "source-a", "kind-a", 1100, "fen-shared", "prelude-shared", "zeta", "alpha")
		insertLegacyPuzzle(t, tx, "shared", "source-b", "kind-b", 2200, "fen-shared", "prelude-shared", " beta ", "beta", "")
		insertLegacyPuzzle(t, tx, "other", "source-a", "kind-a", 3300, "fen-other", "prelude-other", "gamma")
	})
	user := openMigratedUserDatabase(t)
	insertUserAttempt(t, user, "attempt-shared", "shared", "source-b")
	insertUserAttempt(t, user, "attempt-other", "other", "source-a")

	if err := BackfillLegacyPuzzleSnapshots(contextWithStorageTestTimeout(t), user, legacy); err != nil {
		t.Fatal(err)
	}

	assertAttemptSnapshot(t, user, "attempt-shared", "kind-b", 2200, `["beta"]`)
	assertAttemptSnapshot(t, user, "attempt-other", "kind-a", 3300, `["gamma"]`)
}

func TestBackfillLegacySnapshotsIncludesQueuedPresentation(t *testing.T) {
	legacy := openPopulatedLegacyDatabase(t, func(t *testing.T, tx *sql.Tx) {
		insertLegacyPuzzle(t, tx, "queued", "lichess", "lichess-csv", 1450, "source-fen", "e2e4", "pin", "fork")
	})
	user := openMigratedUserDatabase(t)
	insertUserSessionItem(t, user, "session", 0, "queued", "lichess")
	insertUserAttempt(t, user, "attempt", "queued", "lichess")

	if err := BackfillLegacyPuzzleSnapshots(contextWithStorageTestTimeout(t), user, legacy); err != nil {
		t.Fatal(err)
	}

	var sourceKind, themes, sourceFEN, prelude sql.NullString
	var rating sql.NullInt64
	if err := user.QueryRow(`SELECT source_kind, rating_snapshot, themes_json,
		source_fen_snapshot, prelude_uci_snapshot
		FROM session_items WHERE session_id = 'session' AND ordinal = 0`).Scan(
		&sourceKind,
		&rating,
		&themes,
		&sourceFEN,
		&prelude,
	); err != nil {
		t.Fatal(err)
	}
	if sourceKind.String != "lichess-csv" || rating.Int64 != 1450 || themes.String != `["fork","pin"]` ||
		sourceFEN.String != "source-fen" || prelude.String != "e2e4" {
		t.Fatalf(
			"queued snapshots = kind %v rating %v themes %v source FEN %v prelude %v",
			sourceKind,
			rating,
			themes,
			sourceFEN,
			prelude,
		)
	}
	assertAttemptSnapshot(t, user, "attempt", "lichess-csv", 1450, `["fork","pin"]`)
}

func TestBackfillLegacySnapshotsLeavesUnknownRowsNull(t *testing.T) {
	legacy := openPopulatedLegacyDatabase(t, func(t *testing.T, tx *sql.Tx) {
		insertLegacyPuzzle(t, tx, "known", "source", "kind", 1200, "fen", "", "fork")
	})
	user := openMigratedUserDatabase(t)
	insertUserSessionItem(t, user, "session", 0, "unknown", "missing-source")
	insertUserAttempt(t, user, "attempt", "unknown", "missing-source")

	if err := BackfillLegacyPuzzleSnapshots(contextWithStorageTestTimeout(t), user, legacy); err != nil {
		t.Fatal(err)
	}

	var itemValues, attemptValues int
	if err := user.QueryRow(`SELECT
		(source_kind IS NOT NULL) + (rating_snapshot IS NOT NULL) +
		(themes_json IS NOT NULL) + (source_fen_snapshot IS NOT NULL) +
		(prelude_uci_snapshot IS NOT NULL)
		FROM session_items WHERE session_id = 'session'`).Scan(&itemValues); err != nil {
		t.Fatal(err)
	}
	if err := user.QueryRow(`SELECT
		(source_kind IS NOT NULL) + (rating_snapshot IS NOT NULL) + (themes_json IS NOT NULL)
		FROM attempts WHERE attempt_id = 'attempt'`).Scan(&attemptValues); err != nil {
		t.Fatal(err)
	}
	if itemValues != 0 || attemptValues != 0 {
		t.Fatalf("unknown snapshots populated: session values=%d attempt values=%d", itemValues, attemptValues)
	}
}

func TestBackfillLegacySnapshotsDoesNotOverwriteValues(t *testing.T) {
	legacy := openPopulatedLegacyDatabase(t, func(t *testing.T, tx *sql.Tx) {
		insertLegacyPuzzle(t, tx, "known", "source", "legacy-kind", 1500, "legacy-fen", "legacy-prelude", "fork")
	})
	user := openMigratedUserDatabase(t)
	insertUserSessionItem(t, user, "session", 0, "known", "source")
	insertUserAttempt(t, user, "attempt", "known", "source")
	if _, err := user.Exec(`UPDATE session_items SET
		source_kind = 'kept-kind', rating_snapshot = 999, themes_json = '["kept"]',
		prelude_uci_snapshot = 'kept-prelude'
		WHERE session_id = 'session' AND ordinal = 0`); err != nil {
		t.Fatal(err)
	}
	if _, err := user.Exec(`UPDATE attempts SET source_kind = 'kept-kind', rating_snapshot = 999
		WHERE attempt_id = 'attempt'`); err != nil {
		t.Fatal(err)
	}

	if err := BackfillLegacyPuzzleSnapshots(contextWithStorageTestTimeout(t), user, legacy); err != nil {
		t.Fatal(err)
	}

	var sourceKind, themes, sourceFEN, prelude string
	var rating int
	if err := user.QueryRow(`SELECT source_kind, rating_snapshot, themes_json,
		source_fen_snapshot, prelude_uci_snapshot
		FROM session_items WHERE session_id = 'session' AND ordinal = 0`).Scan(
		&sourceKind,
		&rating,
		&themes,
		&sourceFEN,
		&prelude,
	); err != nil {
		t.Fatal(err)
	}
	if sourceKind != "kept-kind" || rating != 999 || themes != `["kept"]` ||
		sourceFEN != "legacy-fen" || prelude != "kept-prelude" {
		t.Fatalf(
			"session snapshots = kind %q rating %d themes %s source FEN %q prelude %q",
			sourceKind,
			rating,
			themes,
			sourceFEN,
			prelude,
		)
	}
	assertAttemptSnapshot(t, user, "attempt", "kept-kind", 999, `["fork"]`)
}

func TestBackfillLegacySnapshotsIsIdempotentAndBatched(t *testing.T) {
	const irrelevantCatalogueRows = 1000
	historyKeys := legacySnapshotBatchSize*2 + 7
	legacy := openPopulatedLegacyDatabase(t, func(t *testing.T, tx *sql.Tx) {
		for index := range historyKeys {
			fingerprint := fmt.Sprintf("history-%04d", index)
			insertLegacyPuzzle(t, tx, fingerprint, "source", "kind", 1000+index, "fen-"+fingerprint, "", "zeta", "alpha")
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO sources(source_id, kind, imported_at, source_path, checksum)
			VALUES ('irrelevant', 'ignored', 1, '/tmp/ignored', 'checksum')`); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`WITH RECURSIVE numbers(value) AS (
			SELECT 0 UNION ALL SELECT value + 1 FROM numbers WHERE value + 1 < %d
		)
		INSERT INTO puzzles(fingerprint, displayed_fen, solver, solution_json, solution_plies)
		SELECT printf('irrelevant-%%04d', value), 'fen', 'white', '[]', 1 FROM numbers`, irrelevantCatalogueRows)); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(`INSERT INTO puzzle_sources(fingerprint, source_id)
			SELECT fingerprint, 'irrelevant' FROM puzzles WHERE fingerprint LIKE 'irrelevant-%'`); err != nil {
			t.Fatal(err)
		}
	})
	user := openMigratedUserDatabase(t)
	if _, err := user.Exec(`INSERT INTO sessions(
		session_id, mode, status, created_at, updated_at
	) VALUES ('session', 'rated', 'active', 1, 1)`); err != nil {
		t.Fatal(err)
	}
	tx, err := user.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for index := range historyKeys {
		fingerprint := fmt.Sprintf("history-%04d", index)
		if _, err := tx.Exec(`INSERT INTO session_items(
			session_id, ordinal, fingerprint, source_id, state_json
		) VALUES ('session', ?, ?, 'source', '{}')`, index, fingerprint); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		if _, err := tx.Exec(`INSERT INTO attempts(attempt_id, fingerprint, source_id, started_at)
			VALUES (?, ?, 'source', 1)`, fmt.Sprintf("attempt-%04d", index), fingerprint); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
	}
	if _, err := tx.Exec(`INSERT INTO attempts(attempt_id, fingerprint, source_id, started_at)
		VALUES ('unknown-attempt', 'zzzz-unknown', 'source', 1)`); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	query, args := legacySnapshotLookupQuery([]legacySnapshotKey{{fingerprint: "history-0000", sourceID: "source"}})
	if len(args) != 2 {
		t.Fatalf("lookup bind arguments = %d, want exactly one requested key", len(args))
	}
	assertLegacyLookupUsesKeyIndex(t, legacy, query, args)

	ctx := contextWithStorageTestTimeout(t)
	if err := BackfillLegacyPuzzleSnapshots(ctx, user, legacy); err != nil {
		t.Fatal(err)
	}
	var attemptsFilled, itemsFilled int
	if err := user.QueryRow(`SELECT COUNT(*) FROM attempts WHERE source_kind IS NOT NULL`).Scan(&attemptsFilled); err != nil {
		t.Fatal(err)
	}
	if err := user.QueryRow(`SELECT COUNT(*) FROM session_items WHERE source_kind IS NOT NULL`).Scan(&itemsFilled); err != nil {
		t.Fatal(err)
	}
	if attemptsFilled != historyKeys || itemsFilled != historyKeys {
		t.Fatalf("filled attempts=%d items=%d, want %d each", attemptsFilled, itemsFilled, historyKeys)
	}

	var changesBefore, changesAfter int64
	if err := user.QueryRow(`SELECT total_changes()`).Scan(&changesBefore); err != nil {
		t.Fatal(err)
	}
	if err := BackfillLegacyPuzzleSnapshots(ctx, user, legacy); err != nil {
		t.Fatal(err)
	}
	if err := user.QueryRow(`SELECT total_changes()`).Scan(&changesAfter); err != nil {
		t.Fatal(err)
	}
	if changesAfter != changesBefore {
		t.Fatalf("second backfill changed %d rows", changesAfter-changesBefore)
	}
	var unknownValues int
	if err := user.QueryRow(`SELECT
		(source_kind IS NOT NULL) + (rating_snapshot IS NOT NULL) + (themes_json IS NOT NULL)
		FROM attempts WHERE attempt_id = 'unknown-attempt'`).Scan(&unknownValues); err != nil {
		t.Fatal(err)
	}
	if unknownValues != 0 {
		t.Fatalf("unknown row received %d snapshot values", unknownValues)
	}
}

func assertLegacyLookupUsesKeyIndex(t *testing.T, db *sql.DB, query string, args []any) {
	t.Helper()
	rows, err := db.Query(`EXPLAIN QUERY PLAN `+query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var plan []string
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		plan = append(plan, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(plan, "\n")
	if !strings.Contains(joined, "SEARCH") || !strings.Contains(joined, "puzzle_sources") ||
		strings.Contains(joined, "SCAN puzzle_sources") {
		t.Fatalf("legacy lookup does not use a requested-key index:\n%s", joined)
	}
}

func openPopulatedLegacyDatabase(t *testing.T, populate func(*testing.T, *sql.Tx)) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "legacy.sqlite")
	writer := createSQLiteFixture(t, path, legacyPuzzlesV2Fixture)
	tx, err := writer.Begin()
	if err != nil {
		writer.Close()
		t.Fatal(err)
	}
	populate(t, tx)
	if err := tx.Commit(); err != nil {
		writer.Close()
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	legacy, err := OpenLegacyPuzzleReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := legacy.Close(); err != nil {
			t.Errorf("close legacy database: %v", err)
		}
	})
	return legacy
}

func insertLegacyPuzzle(
	t *testing.T,
	tx *sql.Tx,
	fingerprint,
	sourceID,
	kind string,
	rating int,
	sourceFEN,
	prelude string,
	themes ...string,
) {
	t.Helper()
	if _, err := tx.Exec(`INSERT OR IGNORE INTO sources(
		source_id, kind, imported_at, source_path, checksum
	) VALUES (?, ?, 1, '/tmp/source', 'checksum')`, sourceID, kind); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO puzzles(
		fingerprint, source_fen, prelude_uci, displayed_fen, solver, solution_json, solution_plies
	) VALUES (?, NULLIF(?, ''), NULLIF(?, ''), 'displayed-fen', 'white', '[]', 1)`,
		fingerprint,
		sourceFEN,
		prelude,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO puzzle_sources(fingerprint, source_id, rating)
		VALUES (?, ?, ?)`, fingerprint, sourceID, rating); err != nil {
		t.Fatal(err)
	}
	for _, theme := range themes {
		if _, err := tx.Exec(`INSERT INTO puzzle_themes(fingerprint, source_id, theme)
			VALUES (?, ?, ?)`, fingerprint, sourceID, theme); err != nil {
			t.Fatal(err)
		}
	}
}

func insertUserSessionItem(
	t *testing.T,
	db *sql.DB,
	sessionID string,
	ordinal int,
	fingerprint,
	sourceID string,
) {
	t.Helper()
	if _, err := db.Exec(`INSERT OR IGNORE INTO sessions(
		session_id, mode, status, created_at, updated_at
	) VALUES (?, 'rated', 'active', 1, 1)`, sessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO session_items(
		session_id, ordinal, fingerprint, source_id, state_json
	) VALUES (?, ?, ?, ?, '{}')`, sessionID, ordinal, fingerprint, sourceID); err != nil {
		t.Fatal(err)
	}
}

func insertUserAttempt(t *testing.T, db *sql.DB, attemptID, fingerprint, sourceID string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO attempts(attempt_id, fingerprint, source_id, started_at)
		VALUES (?, ?, ?, 1)`, attemptID, fingerprint, sourceID); err != nil {
		t.Fatal(err)
	}
}

func assertAttemptSnapshot(
	t *testing.T,
	db *sql.DB,
	attemptID,
	wantKind string,
	wantRating int,
	wantThemes string,
) {
	t.Helper()
	var kind, themes sql.NullString
	var rating sql.NullInt64
	if err := db.QueryRow(`SELECT source_kind, rating_snapshot, themes_json
		FROM attempts WHERE attempt_id = ?`, attemptID).Scan(&kind, &rating, &themes); err != nil {
		t.Fatal(err)
	}
	if !kind.Valid || kind.String != wantKind || !rating.Valid || rating.Int64 != int64(wantRating) ||
		!themes.Valid || themes.String != wantThemes {
		t.Fatalf("attempt %s snapshot = kind %v rating %v themes %v", attemptID, kind, rating, themes)
	}
}

type preservedPuzzleFiles struct {
	database [sha256.Size]byte
	wal      [sha256.Size]byte
}

func assertRejectedPuzzleFilesPreserved(t *testing.T, path string) {
	t.Helper()
	want := snapshotPreservedPuzzleFiles(t, path)
	state, err := ProbePuzzleStore(path)
	if err == nil {
		t.Fatal("ProbePuzzleStore() accepted an unknown database")
	}
	if !state.Exists || state.Legacy {
		t.Fatalf("ProbePuzzleStore() state = %+v, want rejected existing database", state)
	}
	assertPuzzleFilesPreserved(t, path, want)

	if err := RemoveRecognizedLegacyPuzzleStore(path); err == nil {
		t.Fatal("RemoveRecognizedLegacyPuzzleStore() accepted an unknown database")
	}
	assertPuzzleFilesPreserved(t, path, want)
}

func snapshotPreservedPuzzleFiles(t *testing.T, path string) preservedPuzzleFiles {
	t.Helper()
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		if _, err := os.Stat(candidate); err != nil {
			t.Fatalf("required preservation file %s: %v", candidate, err)
		}
	}
	return preservedPuzzleFiles{database: hashFile(t, path), wal: hashFile(t, path+"-wal")}
}

func assertPuzzleFilesPreserved(t *testing.T, path string, want preservedPuzzleFiles) {
	t.Helper()
	if got := hashFile(t, path); got != want.database {
		t.Fatal("database bytes changed during a rejected operation")
	}
	if got := hashFile(t, path+"-wal"); got != want.wal {
		t.Fatal("WAL bytes changed during a rejected operation")
	}
	if _, err := os.Stat(path + "-shm"); err != nil {
		t.Fatalf("SHM sidecar was removed during a rejected operation: %v", err)
	}
}

func hashFile(t *testing.T, path string) [sha256.Size]byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(contents)
}

func createSQLiteFixture(t *testing.T, path, fixture string) *sql.DB {
	t.Helper()
	dsn, err := puzzleStoreDSN(path, false, true)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if _, err := db.Exec(fixture); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db
}

func createCrashSQLiteFixture(t *testing.T, path, fixture string) {
	t.Helper()
	dsn, err := puzzleStoreDSN(path, false, true)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA wal_autocheckpoint=0`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if _, err := db.Exec(fixture); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master`).Scan(new(int)); err != nil {
		db.Close()
		t.Fatal(err)
	}

	files := make(map[string][]byte, 3)
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		contents, err := os.ReadFile(candidate)
		if err != nil {
			db.Close()
			t.Fatalf("capture live SQLite file %s: %v", candidate, err)
		}
		files[candidate] = contents
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	for candidate, contents := range files {
		if err := os.WriteFile(candidate, contents, 0o600); err != nil {
			t.Fatalf("restore crash SQLite file %s: %v", candidate, err)
		}
	}
}

func createMarkedLegacyCrashSQLiteFixture(t *testing.T, path, fixture, marker string) {
	t.Helper()
	db := createSQLiteFixture(t, path, fixture)
	if _, err := db.Exec(`INSERT INTO sources(
		source_id, kind, imported_at, source_path, checksum
	) VALUES (?, 'test', 1, '/main', 'main')`, "main-"+marker); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	dsn, err := puzzleStoreDSN(path, false, false)
	if err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA wal_autocheckpoint=0`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sources(
		source_id, kind, imported_at, source_path, checksum
	) VALUES (?, 'test', 1, '/wal', 'wal')`, "wal-"+marker); err != nil {
		db.Close()
		t.Fatal(err)
	}

	files := make(map[string][]byte, 3)
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		contents, err := os.ReadFile(candidate)
		if err != nil {
			db.Close()
			t.Fatalf("capture marked live SQLite file %s: %v", candidate, err)
		}
		files[candidate] = contents
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	for candidate, contents := range files {
		if err := os.WriteFile(candidate, contents, 0o600); err != nil {
			t.Fatalf("restore marked crash SQLite file %s: %v", candidate, err)
		}
	}
}

func openMigratedUserDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "user.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db, "user"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close user database: %v", err)
		}
	})
	return db
}

func contextWithStorageTestTimeout(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
