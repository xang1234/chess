package app

import (
	"errors"
	"os"
	"strings"
	"testing"

	"chess-trainer/internal/storage"
)

func TestPreflightRejectsLaterPuzzleBeforeMigratingUser(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	createTask5PendingUserMigration(t, paths.UserDB)
	wantUser := hashTask5File(t, paths.UserDB)
	createTask5CrashSQLiteFixture(t, paths.PuzzlesDB, `
		CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY);
		INSERT INTO schema_migrations(version) VALUES (4);
		CREATE TABLE future_data(value TEXT);
		INSERT INTO future_data(value) VALUES ('preserve me');
	`)

	services, err := OpenApplication(paths)
	if services == nil {
		t.Fatal("OpenApplication() did not retain recovery lifecycle")
	}
	defer services.Close()
	var recoveryErr *RecoveryRequiredError
	if !errors.As(err, &recoveryErr) || recoveryErr.Path != paths.PuzzlesDB {
		t.Fatalf("OpenApplication() error = %v, want puzzle recovery", err)
	}
	if got := hashTask5File(t, paths.UserDB); got != wantUser {
		t.Fatal("puzzle preflight failure changed the earlier user database")
	}
	assertTask5MigrationMissing(t, paths.UserDB, 3)
	assertNoTask5PreflightScratch(t, paths.Root)
}

func TestPreflightRejectsLaterLibraryBeforeMigratingUserOrRemovingLegacyPuzzle(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	createTask5PendingUserMigration(t, paths.UserDB)
	createTask5CrashSQLiteFixture(t, paths.PuzzlesDB, loadTask5LegacyFixture(t, 2))
	wantUser := hashTask5File(t, paths.UserDB)
	wantPuzzle := snapshotTask5PuzzleFiles(t, paths.PuzzlesDB)
	library, err := storage.Open(paths.LibraryDB)
	if err != nil {
		t.Fatal(err)
	}
	mustTask5Exec(t, library, `CREATE TABLE schema_migrations(unexpected INTEGER)`)
	mustTask5Exec(t, library, `CREATE TABLE library_metadata(
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`)
	if err := library.Close(); err != nil {
		t.Fatal(err)
	}

	services, err := OpenApplication(paths)
	if services == nil {
		t.Fatal("OpenApplication() did not retain recovery lifecycle")
	}
	defer services.Close()
	var recoveryErr *RecoveryRequiredError
	if !errors.As(err, &recoveryErr) || recoveryErr.Path != paths.LibraryDB {
		t.Fatalf("OpenApplication() error = %v, want library recovery", err)
	}
	if got := hashTask5File(t, paths.UserDB); got != wantUser {
		t.Fatal("library preflight failure changed the earlier user database")
	}
	assertTask5MigrationMissing(t, paths.UserDB, 3)
	assertTask5PuzzleFilesPreserved(t, paths.PuzzlesDB, wantPuzzle)
	state, probeErr := storage.ProbePuzzleStore(paths.PuzzlesDB)
	if probeErr != nil || !state.Exists || !state.Legacy || state.Format != 2 {
		t.Fatalf("legacy puzzle database after library preflight failure = %+v, %v", state, probeErr)
	}
	assertNoTask5PreflightScratch(t, paths.Root)
}

func TestPreflightAllowsNormalPendingMigration(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	createTask5PendingUserMigration(t, paths.UserDB)

	services, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	if err := services.Close(); err != nil {
		t.Fatal(err)
	}
	db := openTask5CurrentUserDB(t, paths.UserDB)
	defer db.Close()
	var applied int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 3`).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Fatalf("migration 3 applied count = %d, want 1", applied)
	}
	assertNoTask5PreflightScratch(t, paths.Root)
}

func TestPreflightDoesNotCreateMissingStoreBeforeLaterFailure(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	createTask5CrashSQLiteFixture(t, paths.PuzzlesDB, loadTask5LegacyFixture(t, 2))
	wantPuzzle := snapshotTask5PuzzleFiles(t, paths.PuzzlesDB)
	library, err := storage.Open(paths.LibraryDB)
	if err != nil {
		t.Fatal(err)
	}
	mustTask5Exec(t, library, `CREATE TABLE schema_migrations(unexpected INTEGER)`)
	if err := library.Close(); err != nil {
		t.Fatal(err)
	}

	services, err := OpenApplication(paths)
	if services == nil {
		t.Fatal("OpenApplication() did not retain recovery lifecycle")
	}
	defer services.Close()
	var recoveryErr *RecoveryRequiredError
	if !errors.As(err, &recoveryErr) || recoveryErr.Path != paths.LibraryDB {
		t.Fatalf("OpenApplication() error = %v, want library recovery", err)
	}
	if _, err := os.Stat(paths.UserDB); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preflight created missing user database: %v", err)
	}
	assertTask5PuzzleFilesPreserved(t, paths.PuzzlesDB, wantPuzzle)
	assertNoTask5PreflightScratch(t, paths.Root)
}

func createTask5PendingUserMigration(t *testing.T, path string) {
	t.Helper()
	user := openTask5CurrentUserDB(t, path)
	for _, statement := range []string{
		`ALTER TABLE session_items DROP COLUMN prelude_uci_snapshot`,
		`ALTER TABLE session_items DROP COLUMN source_fen_snapshot`,
		`ALTER TABLE attempts DROP COLUMN themes_json`,
		`ALTER TABLE attempts DROP COLUMN rating_snapshot`,
		`ALTER TABLE attempts DROP COLUMN source_kind`,
		`ALTER TABLE session_items DROP COLUMN themes_json`,
		`ALTER TABLE session_items DROP COLUMN rating_snapshot`,
		`ALTER TABLE session_items DROP COLUMN source_kind`,
		`ALTER TABLE review_state DROP COLUMN preferred_source_id`,
		`DELETE FROM schema_migrations WHERE version = 3`,
	} {
		mustTask5Exec(t, user, statement)
	}
	if err := user.Close(); err != nil {
		t.Fatal(err)
	}
}

func assertNoTask5PreflightScratch(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".startup-preflight-") {
			t.Fatalf("startup preflight scratch was not removed: %s", entry.Name())
		}
	}
}

func assertTask5MigrationMissing(t *testing.T, path string, version int) {
	t.Helper()
	db, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var applied int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != 0 {
		t.Fatalf("migration %d was applied before all stores passed preflight", version)
	}
}
