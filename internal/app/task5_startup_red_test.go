package app

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"chess-trainer/internal/importing"
	"chess-trainer/internal/importjob"
	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/storage"
)

func TestOpenAcquiresLockBeforeGenerationRecovery(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	db := openTask5CurrentPuzzleDB(t, paths.PuzzlesDB)
	seedTask5BuildingGeneration(t, db, "interrupted", "interrupted-generation", "interrupted-core")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	lock, err := storage.AcquireDataRootLock(paths.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()

	services, err := Open(paths)
	if services != nil {
		services.Close()
	}
	if !errors.Is(err, storage.ErrDataRootLocked) {
		t.Fatalf("Open() error = %v, want data-root lock error", err)
	}

	probe, err := storage.Open(paths.PuzzlesDB)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	var status string
	if err := probe.QueryRow(`SELECT status FROM source_generations
		WHERE generation_id = 'interrupted-generation'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "building" {
		t.Fatalf("generation status = %q, want building while another instance owns the lock", status)
	}
}

func TestSecondServicesOpenCannotAbandonFirstInstanceImport(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	first, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()

	generation, err := first.Catalog.BeginImport(context.Background(), puzzles.Source{
		ID: "live-source", Kind: "test", Path: "/live-import", StartedAt: time.Unix(1, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer generation.Abandon(context.Background())

	second, err := Open(paths)
	if second != nil {
		second.Close()
	}
	if !errors.Is(err, storage.ErrDataRootLocked) {
		t.Fatalf("second Open() error = %v, want data-root lock error", err)
	}

	var status string
	if err := first.PuzzleStore.Reader.QueryRow(`SELECT status FROM source_generations
		WHERE source_id = 'live-source'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "building" {
		t.Fatalf("first instance generation status = %q, want building", status)
	}
}

func TestOpenMigratesUserBeforeLegacyPuzzleRemoval(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	legacy := createTask5SQLiteFixture(t, paths.PuzzlesDB, loadTask5LegacyFixture(t, 2))
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	wantPuzzleHash := hashTask5File(t, paths.PuzzlesDB)

	// Make a valid, current user store advertise v2. Reapplying migration 003
	// fails on its already-present columns, which makes the startup order visible.
	user, err := storage.Open(paths.UserDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(user, "user"); err != nil {
		user.Close()
		t.Fatal(err)
	}
	mustTask5Exec(t, user, `DELETE FROM schema_migrations WHERE version = 3`)
	if err := user.Close(); err != nil {
		t.Fatal(err)
	}

	services, err := Open(paths)
	if services != nil {
		services.Close()
	}
	if err == nil {
		t.Fatal("Open() succeeded despite a deliberately failing user migration")
	}
	if got := hashTask5File(t, paths.PuzzlesDB); got != wantPuzzleHash {
		t.Fatal("legacy puzzle database changed before the user migration succeeded")
	}
	state, probeErr := storage.ProbePuzzleStore(paths.PuzzlesDB)
	if probeErr != nil || !state.Exists || !state.Legacy || state.Format != 2 {
		t.Fatalf("legacy puzzle database after failed user migration = %+v, %v", state, probeErr)
	}
}

func TestOpenBackfillsSnapshotsBeforeLegacyPuzzleRemoval(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	legacy := createTask5SQLiteFixture(t, paths.PuzzlesDB, loadTask5LegacyFixture(t, 2))
	mustTask5Exec(t, legacy, `INSERT INTO sources(
		source_id, kind, imported_at, source_path, checksum
	) VALUES ('legacy-source', 'legacy-kind', 1, '/legacy', 'checksum')`)
	mustTask5Exec(t, legacy, `INSERT INTO puzzles(
		fingerprint, source_fen, prelude_uci, displayed_fen, solver, solution_json, solution_plies
	) VALUES ('legacy-core', 'source-fen', 'e2e4', 'displayed-fen', 'white', '[]', 1)`)
	mustTask5Exec(t, legacy, `INSERT INTO puzzle_sources(
		fingerprint, source_id, rating
	) VALUES ('legacy-core', 'legacy-source', 1660)`)
	mustTask5Exec(t, legacy, `INSERT INTO puzzle_themes(fingerprint, source_id, theme) VALUES
		('legacy-core', 'legacy-source', 'pin'),
		('legacy-core', 'legacy-source', 'fork')`)
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	user := openTask5CurrentUserDB(t, paths.UserDB)
	mustTask5Exec(t, user, `INSERT INTO sessions(
		session_id, mode, status, created_at, updated_at
	) VALUES ('queued-session', 'rated', 'active', 1, 1)`)
	mustTask5Exec(t, user, `INSERT INTO session_items(
		session_id, ordinal, fingerprint, source_id, state_json
	) VALUES ('queued-session', 0, 'legacy-core', 'legacy-source', '{}')`)
	mustTask5Exec(t, user, `INSERT INTO attempts(
		attempt_id, fingerprint, source_id, started_at
	) VALUES ('legacy-attempt', 'legacy-core', 'legacy-source', 1)`)
	if err := user.Close(); err != nil {
		t.Fatal(err)
	}

	services, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	if err := services.Close(); err != nil {
		t.Fatal(err)
	}

	user, err = storage.Open(paths.UserDB)
	if err != nil {
		t.Fatal(err)
	}
	defer user.Close()
	var itemKind, itemThemes, sourceFEN, prelude string
	var itemRating int
	if err := user.QueryRow(`SELECT source_kind, rating_snapshot, themes_json,
		source_fen_snapshot, prelude_uci_snapshot
		FROM session_items WHERE session_id = 'queued-session' AND ordinal = 0`).Scan(
		&itemKind, &itemRating, &itemThemes, &sourceFEN, &prelude,
	); err != nil {
		t.Fatal(err)
	}
	if itemKind != "legacy-kind" || itemRating != 1660 || itemThemes != `["fork","pin"]` ||
		sourceFEN != "source-fen" || prelude != "e2e4" {
		t.Fatalf("queued snapshot = kind %q rating %d themes %s source FEN %q prelude %q",
			itemKind, itemRating, itemThemes, sourceFEN, prelude)
	}
	var attemptKind, attemptThemes string
	var attemptRating int
	if err := user.QueryRow(`SELECT source_kind, rating_snapshot, themes_json
		FROM attempts WHERE attempt_id = 'legacy-attempt'`).Scan(
		&attemptKind, &attemptRating, &attemptThemes,
	); err != nil {
		t.Fatal(err)
	}
	if attemptKind != "legacy-kind" || attemptRating != 1660 || attemptThemes != `["fork","pin"]` {
		t.Fatalf("attempt snapshot = kind %q rating %d themes %s",
			attemptKind, attemptRating, attemptThemes)
	}
	state, err := storage.ProbePuzzleStore(paths.PuzzlesDB)
	if err != nil || !state.Exists || state.Legacy || state.Format != storage.CurrentPuzzleSchemaVersion {
		t.Fatalf("recreated puzzle database = %+v, %v", state, err)
	}
}

func TestOpenRecreatesOnlyExactLegacyV1AndV2(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			paths := storage.PathsAt(t.TempDir())
			if err := paths.Ensure(); err != nil {
				t.Fatal(err)
			}
			legacy := createTask5SQLiteFixture(t, paths.PuzzlesDB, loadTask5LegacyFixture(t, version))
			if err := legacy.Close(); err != nil {
				t.Fatal(err)
			}

			services, err := Open(paths)
			if err != nil {
				t.Fatal(err)
			}
			if err := services.Close(); err != nil {
				t.Fatal(err)
			}

			state, err := storage.ProbePuzzleStore(paths.PuzzlesDB)
			if err != nil || !state.Exists || state.Legacy || state.Format != storage.CurrentPuzzleSchemaVersion {
				t.Fatalf("Open() replacement for legacy v%d = %+v, %v", version, state, err)
			}
		})
	}
}

func TestOpenPreservesUnknownNewerModifiedAndCorruptPuzzleFiles(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, string)
	}{
		{
			name: "newer",
			setup: func(t *testing.T, path string) {
				createTask5CrashSQLiteFixture(t, path, `
					CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY);
					INSERT INTO schema_migrations(version) VALUES (4);
					CREATE TABLE future_catalogue_state(value TEXT);
					INSERT INTO future_catalogue_state(value) VALUES ('preserve me');
				`)
			},
		},
		{
			name: "modified legacy",
			setup: func(t *testing.T, path string) {
				createTask5CrashSQLiteFixture(t, path, loadTask5LegacyFixture(t, 2)+`
					CREATE TABLE local_extension(value TEXT);
					INSERT INTO local_extension(value) VALUES ('preserve me');
				`)
			},
		},
		{
			name: "corrupt",
			setup: func(t *testing.T, path string) {
				for candidate, contents := range map[string][]byte{
					path:          []byte("not a sqlite database"),
					path + "-wal": []byte("unrecognized wal data"),
					path + "-shm": []byte("unrecognized shm data"),
				} {
					if err := os.WriteFile(candidate, contents, 0o600); err != nil {
						t.Fatal(err)
					}
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paths := storage.PathsAt(t.TempDir())
			if err := paths.Ensure(); err != nil {
				t.Fatal(err)
			}
			test.setup(t, paths.PuzzlesDB)
			want := snapshotTask5PuzzleFiles(t, paths.PuzzlesDB)

			services, err := Open(paths)
			if services != nil {
				services.Close()
			}
			if err == nil {
				t.Fatal("Open() accepted a puzzle database that is not exact legacy v1/v2 or current")
			}
			assertTask5PuzzleFilesPreserved(t, paths.PuzzlesDB, want)
		})
	}
}

func TestPuzzleRecreationLeavesCurrentUserDatabaseBytesAndRowsUnchanged(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	legacy := createTask5SQLiteFixture(t, paths.PuzzlesDB, loadTask5LegacyFixture(t, 2))
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	user := openTask5CurrentUserDB(t, paths.UserDB)
	mustTask5Exec(t, user, `INSERT INTO profile(
		id, learner_rating, session_size, created_at, updated_at
	) VALUES (1, 1530, 10, 1, 2)`)
	mustTask5Exec(t, user, `INSERT INTO rating_history(rating, recorded_at) VALUES (1530, 2)`)
	mustTask5Exec(t, user, `INSERT INTO sessions(
		session_id, mode, status, created_at, updated_at, current_index
	) VALUES ('preserved-session', 'guided', 'completed', 10, 20, 1)`)
	mustTask5Exec(t, user, `INSERT INTO session_items(
		session_id, ordinal, fingerprint, source_id, state_json,
		source_kind, rating_snapshot, themes_json, source_fen_snapshot, prelude_uci_snapshot
	) VALUES (
		'preserved-session', 0, 'preserved-core', 'preserved-source', '{"completed":true}',
		'lichess', 1520, '["fork","pin"]', 'source-fen', 'e2e4'
	)`)
	mustTask5Exec(t, user, `INSERT INTO attempts(
		attempt_id, session_id, fingerprint, source_id, started_at, completed_at,
		incorrect_moves, hints_used, solution_revealed, first_try, duration_ms,
		source_kind, rating_snapshot, themes_json
	) VALUES (
		'preserved-attempt', 'preserved-session', 'preserved-core', 'preserved-source', 10, 20,
		1, 1, 0, 0, 5000, 'lichess', 1520, '["fork","pin"]'
	)`)
	mustTask5Exec(t, user, `INSERT INTO review_state(
		fingerprint, due_at, interval_index, successful_reviews, last_outcome, preferred_source_id
	) VALUES ('preserved-core', 100, 1, 2, 'clean', 'preserved-source')`)
	if err := user.Close(); err != nil {
		t.Fatal(err)
	}

	before := checkpointAndSnapshotTask5User(t, paths.UserDB)
	services, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	if err := services.Close(); err != nil {
		t.Fatal(err)
	}
	after := checkpointAndSnapshotTask5User(t, paths.UserDB)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("current user database changed during puzzle recreation:\nbefore=%+v\nafter=%+v", before, after)
	}
}

func TestNormalStartupNeverQuickChecksPuzzleCatalogue(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	db := openTask5CurrentPuzzleDB(t, paths.PuzzlesDB)
	mustTask5Exec(t, db, `PRAGMA ignore_check_constraints=ON`)
	mustTask5Exec(t, db, `INSERT INTO sources(source_id, kind) VALUES ('sentinel', 'test')`)
	mustTask5Exec(t, db, `INSERT INTO source_generations(
		generation_id, source_id, status, source_path, started_at
	) VALUES ('sentinel-generation', 'sentinel', 'quick-check-sentinel', '/sentinel', 1)`)
	mustTask5Exec(t, db, `PRAGMA ignore_check_constraints=OFF`)
	var quickCheck string
	if err := db.QueryRow(`PRAGMA quick_check`).Scan(&quickCheck); err != nil {
		t.Fatal(err)
	}
	if quickCheck == "ok" {
		t.Fatal("test fixture does not distinguish a quick check from the normal startup probes")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	services, err := Open(paths)
	if err != nil {
		t.Fatalf("normal startup rejected a catalogue only PRAGMA quick_check scans: %v", err)
	}
	if err := services.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenRecoversBuildingGenerationAndTriggersBoundedCleanup(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	db := openTask5CurrentPuzzleDB(t, paths.PuzzlesDB)
	seedTask5ActiveGeneration(t, db)
	seedTask5BuildingGeneration(t, db, "orphan", "orphan-generation", "orphan-core")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	services, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()

	deadline := time.Now().Add(3 * time.Second)
	for {
		var orphanGenerations int
		if err := services.PuzzleStore.Reader.QueryRow(`SELECT COUNT(*) FROM source_generations
			WHERE generation_id = 'orphan-generation'`).Scan(&orphanGenerations); err != nil {
			t.Fatal(err)
		}
		if orphanGenerations == 0 {
			break
		}
		if time.Now().After(deadline) {
			var status string
			_ = services.PuzzleStore.Reader.QueryRow(`SELECT status FROM source_generations
				WHERE generation_id = 'orphan-generation'`).Scan(&status)
			t.Fatalf("startup cleanup did not remove recovered generation; last status=%q", status)
		}
		time.Sleep(10 * time.Millisecond)
	}

	var activeHeads, activeOccurrences, orphanCores int
	if err := services.PuzzleStore.Reader.QueryRow(`SELECT COUNT(*) FROM source_heads
		WHERE source_id = 'active' AND generation_id = 'active-generation'`).Scan(&activeHeads); err != nil {
		t.Fatal(err)
	}
	if err := services.PuzzleStore.Reader.QueryRow(`SELECT COUNT(*) FROM puzzle_occurrences
		WHERE generation_id = 'active-generation' AND fingerprint = 'active-core'`).Scan(&activeOccurrences); err != nil {
		t.Fatal(err)
	}
	if err := services.PuzzleStore.Reader.QueryRow(`SELECT COUNT(*) FROM puzzle_cores
		WHERE fingerprint = 'orphan-core'`).Scan(&orphanCores); err != nil {
		t.Fatal(err)
	}
	if activeHeads != 1 || activeOccurrences != 1 || orphanCores != 0 {
		t.Fatalf("cleanup result = active heads %d occurrences %d orphan cores %d",
			activeHeads, activeOccurrences, orphanCores)
	}
}

type task5CloseBlockingImporter struct {
	started chan context.Context
	release <-chan struct{}
}

func (task5CloseBlockingImporter) Supports(format puzzles.ImportFormat) bool {
	return format == puzzles.FormatLichess
}

func (i task5CloseBlockingImporter) Import(
	ctx context.Context,
	_ importing.Inspection,
	_ importing.ProgressSink,
) (puzzles.ImportReport, error) {
	i.started <- ctx
	<-ctx.Done()
	<-i.release
	return puzzles.ImportReport{}, ctx.Err()
}

func TestServicesCloseWaitsBeforeClosingPuzzleHandles(t *testing.T) {
	services, err := Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	services.ImportJobs.Close()

	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseImport := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseImport)
	importer := task5CloseBlockingImporter{started: make(chan context.Context, 1), release: release}
	services.ImportJobs = importjob.NewService(importer, nil, nil)
	if _, err := services.ImportJobs.Start(context.Background(), importing.Inspection{
		Format: puzzles.FormatLichess, SourceID: "blocking", Path: "/blocking",
	}); err != nil {
		t.Fatal(err)
	}
	jobCtx := <-importer.started

	closed := make(chan error, 1)
	go func() { closed <- services.Close() }()
	select {
	case <-jobCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("Services.Close() did not cancel the active import")
	}
	for name, handle := range map[string]*sql.DB{
		"reader": services.PuzzleStore.Reader,
		"writer": services.PuzzleStore.Writer,
	} {
		if err := handle.Ping(); err != nil {
			t.Fatalf("puzzle %s closed before import exited: %v", name, err)
		}
	}
	select {
	case err := <-closed:
		t.Fatalf("Services.Close() returned before import exited: %v", err)
	default:
	}

	releaseImport()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Services.Close() did not return after import exited")
	}
	for name, handle := range map[string]*sql.DB{
		"reader": services.PuzzleStore.Reader,
		"writer": services.PuzzleStore.Writer,
	} {
		if err := handle.Ping(); err == nil {
			t.Fatalf("puzzle %s remained open after Services.Close()", name)
		}
	}
}

func openTask5CurrentPuzzleDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(db, "puzzles"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db
}

func openTask5CurrentUserDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(db, "user"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db
}

func seedTask5BuildingGeneration(
	t *testing.T,
	db *sql.DB,
	sourceID,
	generationID,
	fingerprint string,
) {
	t.Helper()
	mustTask5Exec(t, db, `INSERT INTO sources(source_id, kind) VALUES (?, 'test')`, sourceID)
	mustTask5Exec(t, db, `INSERT INTO source_generations(
		generation_id, source_id, status, source_path, started_at
	) VALUES (?, ?, 'building', '/interrupted', 1)`, generationID, sourceID)
	mustTask5Exec(t, db, `INSERT INTO puzzle_cores(
		fingerprint, displayed_fen, solver, solution_json, solution_plies
	) VALUES (?, 'displayed-fen', 'white', '[]', 1)`, fingerprint)
	mustTask5Exec(t, db, `INSERT INTO puzzle_occurrences(
		generation_id, fingerprint, metadata_json, ordinal
	) VALUES (?, ?, '{}', 1)`, generationID, fingerprint)
	mustTask5Exec(t, db, `INSERT INTO occurrence_themes(generation_id, fingerprint, theme)
		VALUES (?, ?, 'fork')`, generationID, fingerprint)
}

func seedTask5ActiveGeneration(t *testing.T, db *sql.DB) {
	t.Helper()
	mustTask5Exec(t, db, `INSERT INTO sources(source_id, kind) VALUES ('active', 'test')`)
	mustTask5Exec(t, db, `INSERT INTO source_generations(
		generation_id, source_id, status, source_path, checksum, started_at, sealed_at
	) VALUES ('active-generation', 'active', 'sealed', '/active', 'checksum', 1, 2)`)
	mustTask5Exec(t, db, `INSERT INTO source_heads(source_id, generation_id)
		VALUES ('active', 'active-generation')`)
	mustTask5Exec(t, db, `INSERT INTO puzzle_cores(
		fingerprint, displayed_fen, solver, solution_json, solution_plies
	) VALUES ('active-core', 'displayed-fen', 'white', '[]', 1)`)
	mustTask5Exec(t, db, `INSERT INTO puzzle_occurrences(
		generation_id, fingerprint, metadata_json, ordinal
	) VALUES ('active-generation', 'active-core', '{}', 1)`)
}

func mustTask5Exec(t *testing.T, db *sql.DB, statement string, args ...any) {
	t.Helper()
	if _, err := db.Exec(statement, args...); err != nil {
		t.Fatal(err)
	}
}

func loadTask5LegacyFixture(t *testing.T, version int) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate startup test source")
	}
	path := filepath.Join(
		filepath.Dir(currentFile),
		"..",
		"storage",
		"testdata",
		fmt.Sprintf("legacy_puzzles_v%d.sql", version),
	)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}

func createTask5SQLiteFixture(t *testing.T, path, fixture string) *sql.DB {
	t.Helper()
	db, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(fixture); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db
}

func createTask5CrashSQLiteFixture(t *testing.T, path, fixture string) {
	t.Helper()
	db := createTask5SQLiteFixture(t, path, fixture)
	mustTask5Exec(t, db, `PRAGMA wal_autocheckpoint=0`)
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

type task5PreservedPuzzleFiles struct {
	database [sha256.Size]byte
	wal      [sha256.Size]byte
}

func snapshotTask5PuzzleFiles(t *testing.T, path string) task5PreservedPuzzleFiles {
	t.Helper()
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		if _, err := os.Stat(candidate); err != nil {
			t.Fatalf("required preservation path %s: %v", candidate, err)
		}
	}
	return task5PreservedPuzzleFiles{
		database: hashTask5File(t, path),
		wal:      hashTask5File(t, path+"-wal"),
	}
}

func assertTask5PuzzleFilesPreserved(t *testing.T, path string, want task5PreservedPuzzleFiles) {
	t.Helper()
	if got := hashTask5File(t, path); got != want.database {
		t.Fatal("puzzle database bytes changed during rejected startup")
	}
	if got := hashTask5File(t, path+"-wal"); got != want.wal {
		t.Fatal("puzzle WAL bytes changed during rejected startup")
	}
	if _, err := os.Stat(path + "-shm"); err != nil {
		t.Fatalf("puzzle SHM path was deleted during rejected startup: %v", err)
	}
}

func hashTask5File(t *testing.T, path string) [sha256.Size]byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(contents)
}

type task5OptionalDigest struct {
	exists bool
	hash   [sha256.Size]byte
}

type task5UserSnapshot struct {
	database task5OptionalDigest
	wal      task5OptionalDigest
	rows     []string
}

func checkpointAndSnapshotTask5User(t *testing.T, path string) task5UserSnapshot {
	t.Helper()
	db, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var busy, logFrames, checkpointed int
	if err := db.QueryRow(`PRAGMA wal_checkpoint(TRUNCATE)`).Scan(
		&busy, &logFrames, &checkpointed,
	); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if busy != 0 || logFrames != 0 || checkpointed != 0 {
		db.Close()
		t.Fatalf("user checkpoint = busy %d log %d checkpointed %d, want fully truncated",
			busy, logFrames, checkpointed)
	}
	var nullSnapshots int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM session_items
		 WHERE source_kind IS NULL OR rating_snapshot IS NULL OR themes_json IS NULL
		    OR source_fen_snapshot IS NULL OR prelude_uci_snapshot IS NULL)
		+
		(SELECT COUNT(*) FROM attempts
		 WHERE source_kind IS NULL OR rating_snapshot IS NULL OR themes_json IS NULL)`).Scan(
		&nullSnapshots,
	); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if nullSnapshots != 0 {
		db.Close()
		t.Fatalf("current user fixture has %d rows with null snapshots", nullSnapshots)
	}

	queries := []string{
		`SELECT COALESCE(group_concat(value, char(10)), '') FROM (
			SELECT json_array(version) AS value FROM schema_migrations ORDER BY version)`,
		`SELECT COALESCE(group_concat(value, char(10)), '') FROM (
			SELECT json_array(id, learner_rating, session_size, created_at, updated_at) AS value
			FROM profile ORDER BY id)`,
		`SELECT COALESCE(group_concat(value, char(10)), '') FROM (
			SELECT json_array(session_id, mode, status, created_at, updated_at, current_index) AS value
			FROM sessions ORDER BY session_id)`,
		`SELECT COALESCE(group_concat(value, char(10)), '') FROM (
			SELECT json_array(session_id, ordinal, fingerprint, source_id, state_json,
				source_kind, rating_snapshot, themes_json, source_fen_snapshot, prelude_uci_snapshot) AS value
			FROM session_items ORDER BY session_id, ordinal)`,
		`SELECT COALESCE(group_concat(value, char(10)), '') FROM (
			SELECT json_array(attempt_id, session_id, fingerprint, source_id, started_at, completed_at,
				incorrect_moves, hints_used, solution_revealed, first_try, duration_ms,
				source_kind, rating_snapshot, themes_json) AS value
			FROM attempts ORDER BY attempt_id)`,
		`SELECT COALESCE(group_concat(value, char(10)), '') FROM (
			SELECT json_array(fingerprint, due_at, interval_index, successful_reviews,
				last_outcome, preferred_source_id) AS value
			FROM review_state ORDER BY fingerprint)`,
		`SELECT COALESCE(group_concat(value, char(10)), '') FROM (
			SELECT json_array(rating_history_id, rating, recorded_at) AS value
			FROM rating_history ORDER BY rating_history_id)`,
	}
	rows := make([]string, len(queries))
	for index, query := range queries {
		if err := db.QueryRow(query).Scan(&rows[index]); err != nil {
			db.Close()
			t.Fatal(err)
		}
	}
	snapshot := task5UserSnapshot{
		database: optionalTask5Digest(t, path),
		wal:      optionalTask5Digest(t, path+"-wal"),
		rows:     rows,
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func optionalTask5Digest(t *testing.T, path string) task5OptionalDigest {
	t.Helper()
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return task5OptionalDigest{}
	}
	if err != nil {
		t.Fatal(err)
	}
	return task5OptionalDigest{exists: true, hash: sha256.Sum256(contents)}
}
