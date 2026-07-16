//go:build !bindings

package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/storage"
)

func TestRecoveryApplicationStartupDoesNotRequireImportJobs(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.UserDB, []byte("corrupt user database"), 0o600); err != nil {
		t.Fatal(err)
	}
	app, lifecycle, err := newApplicationAt(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer lifecycle.Close()
	if !app.GetRecoveryState().Required {
		t.Fatal("fixture did not construct a recovery application")
	}

	app.startup(context.Background())
	if lifecycle.ImportJobs != nil {
		t.Fatal("recovery startup unexpectedly composed import jobs")
	}
}

func TestNewApplicationAtBuildsRecoveryForPreservedNewerPuzzleSchema(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(paths.PuzzlesDB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY);
		INSERT INTO schema_migrations(version) VALUES (4);
		CREATE TABLE future_data(value TEXT);
		INSERT INTO future_data(value) VALUES ('preserve me')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	before := fileSHA256(t, paths.PuzzlesDB)

	app, lifecycle, err := newApplicationAt(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer lifecycle.Close()
	state := app.GetRecoveryState()
	if !state.Required || state.Path != paths.PuzzlesDB || state.Detail == "" {
		t.Fatalf("recovery state = %+v", state)
	}
	if after := fileSHA256(t, paths.PuzzlesDB); after != before {
		t.Fatal("recovery startup mutated the incompatible puzzle database")
	}
	contender, lockErr := storage.AcquireDataRootLock(paths.Root)
	if contender != nil {
		contender.Close()
	}
	if !errors.Is(lockErr, storage.ErrDataRootLocked) {
		t.Fatalf("recovery shell did not retain data-root lock: %v", lockErr)
	}
}

func TestNewApplicationAtBuildsRecoveryForPreservedModifiedPuzzleSchema(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	store, err := storage.OpenPuzzleStore(paths.PuzzlesDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(paths.PuzzlesDB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE local_extension(value TEXT)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	before := fileSHA256(t, paths.PuzzlesDB)

	app, lifecycle, err := newApplicationAt(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer lifecycle.Close()
	state := app.GetRecoveryState()
	if !state.Required || state.Path != paths.PuzzlesDB || state.Detail == "" {
		t.Fatalf("recovery state = %+v", state)
	}
	if after := fileSHA256(t, paths.PuzzlesDB); after != before {
		t.Fatal("recovery startup mutated the modified puzzle database")
	}
}

func TestOpenApplicationTypesUserMigrationFailureForRecovery(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(paths.UserDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(db, "user"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM schema_migrations WHERE version = 3`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	services, err := appservices.OpenApplication(paths)
	if services == nil {
		t.Fatal("OpenApplication() did not return recovery lifecycle")
	}
	defer services.Close()
	var recoveryErr *appservices.RecoveryRequiredError
	if !errors.As(err, &recoveryErr) {
		t.Fatalf("OpenApplication() error = %v, want RecoveryRequiredError", err)
	}
	if recoveryErr.Path != paths.UserDB || recoveryErr.Detail == "" {
		t.Fatalf("RecoveryRequiredError = %+v", recoveryErr)
	}

	probe, err := storage.Open(paths.UserDB)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	var applied int
	if err := probe.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 3`).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != 0 {
		t.Fatal("failed startup migration mutated the preserved user schema")
	}
}

func TestOpenApplicationTypesLibraryMigrationFailureWithoutMutation(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(paths.LibraryDB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE schema_migrations(unexpected INTEGER);
		CREATE TABLE library_metadata(key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO library_metadata(key, value) VALUES ('sentinel', 'preserve me')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	services, err := appservices.OpenApplication(paths)
	if services == nil {
		t.Fatal("OpenApplication() did not return recovery lifecycle")
	}
	defer services.Close()
	var recoveryErr *appservices.RecoveryRequiredError
	if !errors.As(err, &recoveryErr) {
		t.Fatalf("OpenApplication() error = %v, want RecoveryRequiredError", err)
	}
	if recoveryErr.Path != paths.LibraryDB || recoveryErr.Detail == "" {
		t.Fatalf("RecoveryRequiredError = %+v", recoveryErr)
	}

	probe, err := storage.Open(paths.LibraryDB)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	var value string
	if err := probe.QueryRow(`SELECT value FROM library_metadata WHERE key = 'sentinel'`).Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "preserve me" {
		t.Fatalf("failed library startup migration changed sentinel to %q", value)
	}
	var versionColumnCount int
	if err := probe.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('schema_migrations') WHERE name = 'version'`).Scan(&versionColumnCount); err != nil {
		t.Fatal(err)
	}
	if versionColumnCount != 0 {
		t.Fatal("failed library startup migration changed the preserved migration schema")
	}
}

func TestNewApplicationAtKeepsSuccessfulStartupUnchanged(t *testing.T) {
	app, services, err := newApplicationAt(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()
	if app.services != services {
		t.Fatal("normal application did not retain its fully opened services")
	}
	if state := app.GetRecoveryState(); state.Required {
		t.Fatalf("normal startup entered recovery: %+v", state)
	}
}

func fileSHA256(t *testing.T, path string) [32]byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(contents)
}
