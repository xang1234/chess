package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"chess-trainer/internal/storage"
)

func TestServiceDoesNotOwnRestoreOrchestration(t *testing.T) {
	if _, exposed := reflect.TypeOf((*Service)(nil)).MethodByName("Restore"); exposed {
		t.Fatal("backup.Service exposes full restore orchestration")
	}
}

func openBackupStores(t *testing.T, paths storage.Paths) (*sql.DB, *sql.DB) {
	t.Helper()
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	userDB, err := storage.Open(paths.UserDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(userDB, "user"); err != nil {
		t.Fatal(err)
	}
	libraryDB, err := storage.Open(paths.LibraryDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(libraryDB, "library"); err != nil {
		t.Fatal(err)
	}
	return userDB, libraryDB
}

func TestServiceCreatesAndRestoresValidatedBackup(t *testing.T) {
	paths := storage.PathsAt(filepath.Join(t.TempDir(), "data"))
	userDB, libraryDB := openBackupStores(t, paths)
	if _, err := userDB.Exec(`INSERT INTO profile(
		id, learner_rating, session_size, created_at, updated_at
	) VALUES (1, 1234, 10, 1, 1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := userDB.Exec(`INSERT INTO attempts(
		attempt_id, fingerprint, source_id, started_at, completed_at,
		incorrect_moves, hints_used, solution_revealed, first_try, duration_ms
	) VALUES ('attempt-1', 'puzzle-1', 'lichess', 1, 2, 0, 0, 0, 1, 1000)`); err != nil {
		t.Fatal(err)
	}
	if _, err := libraryDB.Exec(`INSERT INTO library_metadata(key, value) VALUES ('fixture', 'original')`); err != nil {
		t.Fatal(err)
	}
	service := NewService(paths)
	service.now = func() time.Time { return time.Unix(123, 0) }
	destination := filepath.Join(t.TempDir(), "trainer-backup.zip")

	if err := service.Create(context.Background(), destination, true); err != nil {
		t.Fatal(err)
	}
	manifest := readManifest(t, destination)
	if manifest.Version != 1 || manifest.Files["user.sqlite"] == "" || manifest.Files["library.sqlite"] == "" {
		t.Fatalf("manifest=%+v", manifest)
	}
	if _, included := manifest.Files["courses.sqlite"]; included {
		t.Fatalf("replaceable course content entered backup manifest: %+v", manifest)
	}
	for name, digest := range manifest.Files {
		if len(digest) != 64 || strings.ToLower(digest) != digest {
			t.Fatalf("%s digest=%q", name, digest)
		}
	}
	if _, err := userDB.Exec(`UPDATE profile SET learner_rating = 1800`); err != nil {
		t.Fatal(err)
	}
	if _, err := libraryDB.Exec(`UPDATE library_metadata SET value = 'mutated' WHERE key = 'fixture'`); err != nil {
		t.Fatal(err)
	}
	prepared, err := service.PrepareRestore(context.Background(), destination)
	if err != nil {
		t.Fatal(err)
	}
	defer prepared.Close()
	if err := errorsJoin(userDB.Close(), libraryDB.Close()); err != nil {
		t.Fatal(err)
	}
	if err := prepared.Install(); err != nil {
		t.Fatal(err)
	}

	restoredUser, restoredLibrary := openBackupStores(t, paths)
	defer restoredUser.Close()
	defer restoredLibrary.Close()
	var rating float64
	if err := restoredUser.QueryRow(`SELECT learner_rating FROM profile WHERE id = 1`).Scan(&rating); err != nil || rating != 1234 {
		t.Fatalf("rating=%v err=%v", rating, err)
	}
	var attempts int
	if err := restoredUser.QueryRow(`SELECT COUNT(*) FROM attempts`).Scan(&attempts); err != nil || attempts != 1 {
		t.Fatalf("attempts=%d err=%v", attempts, err)
	}
	var libraryValue string
	if err := restoredLibrary.QueryRow(`SELECT value FROM library_metadata WHERE key = 'fixture'`).Scan(&libraryValue); err != nil || libraryValue != "original" {
		t.Fatalf("library value=%q err=%v", libraryValue, err)
	}
	entries, err := os.ReadDir(paths.BackupsDir)
	if err != nil || len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), "pre-restore-") {
		t.Fatalf("pre-restore entries=%v err=%v", entries, err)
	}
}

func TestServiceRejectsCorruptBackupWithoutTouchingLiveData(t *testing.T) {
	paths := storage.PathsAt(filepath.Join(t.TempDir(), "data"))
	userDB, libraryDB := openBackupStores(t, paths)
	defer userDB.Close()
	defer libraryDB.Close()
	if _, err := userDB.Exec(`INSERT INTO profile(
		id, learner_rating, session_size, created_at, updated_at
	) VALUES (1, 1400, 5, 1, 1)`); err != nil {
		t.Fatal(err)
	}
	service := NewService(paths)
	valid := filepath.Join(t.TempDir(), "valid.zip")
	corrupt := filepath.Join(t.TempDir(), "corrupt.zip")
	if err := service.Create(context.Background(), valid, false); err != nil {
		t.Fatal(err)
	}
	corruptZipEntry(t, valid, corrupt, "user.sqlite")

	if prepared, err := service.PrepareRestore(context.Background(), corrupt); err == nil {
		prepared.Close()
		t.Fatal("PrepareRestore() accepted a corrupt database")
	}
	var rating float64
	if err := userDB.QueryRow(`SELECT learner_rating FROM profile WHERE id = 1`).Scan(&rating); err != nil || rating != 1400 {
		t.Fatalf("live rating=%v err=%v", rating, err)
	}
}

func TestServiceRejectsRestoreThatExceedsAvailableSpace(t *testing.T) {
	service := NewService(storage.PathsAt(t.TempDir()))
	files := map[string]*zip.File{
		"user.sqlite": {FileHeader: zip.FileHeader{UncompressedSize64: 1024}},
	}
	service.availableBytes = func(string) (uint64, error) {
		return 0, nil
	}
	err := service.ensureRestoreSpace(files, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "not enough free disk space") {
		t.Fatalf("ensureRestoreSpace() = %v, want insufficient-space error", err)
	}
}

func TestServiceEnforcesExtractionByteLimit(t *testing.T) {
	source := zipFileWithContents(t, "user.sqlite", bytes.Repeat([]byte("x"), 32))
	service := NewService(storage.PathsAt(t.TempDir()))
	destination := filepath.Join(t.TempDir(), "user.sqlite")
	err := service.extractFile(source, destination, 8)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("extractFile() = %v, want size-limit error", err)
	}
}

func TestReplaceDatabasesRollsBackInstalledFileWhenChmodFails(t *testing.T) {
	paths := storage.PathsAt(filepath.Join(t.TempDir(), "data"))
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	original := []byte("original user database")
	if err := os.WriteFile(paths.UserDB, original, 0o600); err != nil {
		t.Fatal(err)
	}
	extracted := t.TempDir()
	if err := os.WriteFile(filepath.Join(extracted, "user.sqlite"), []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}

	service := NewService(paths)
	chmodErr := errors.New("injected chmod failure")
	service.chmod = func(string, os.FileMode) error { return chmodErr }
	err := service.replaceDatabases(
		extracted,
		Manifest{Files: map[string]string{"user.sqlite": "unused"}},
	)
	if !errors.Is(err, chmodErr) {
		t.Fatalf("replaceDatabases() = %v, want injected chmod failure", err)
	}
	restored, readErr := os.ReadFile(paths.UserDB)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(restored, original) {
		t.Fatalf("live database after rollback = %q, want original", restored)
	}
}

func assertDataRootLocked(t *testing.T, root string) {
	t.Helper()
	contender, err := storage.AcquireDataRootLock(root)
	if contender != nil {
		contender.Close()
	}
	if !errors.Is(err, storage.ErrDataRootLocked) {
		t.Fatalf("AcquireDataRootLock() = %v, want locked", err)
	}
}

func TestReplaceDatabasesRetainsPreRestoreWhenRollbackCannotRestoreDatabase(t *testing.T) {
	paths := storage.PathsAt(filepath.Join(t.TempDir(), "data"))
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	lock, err := storage.AcquireDataRootLock(paths.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	assertDataRootLocked(t, paths.Root)
	if err := os.WriteFile(paths.UserDB, []byte("original user database"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.LibraryDB, []byte("original library database"), 0o600); err != nil {
		t.Fatal(err)
	}

	extracted, err := os.MkdirTemp(paths.Root, ".rollback-fault-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(extracted) })
	installedLibrary := filepath.Join(extracted, "library.sqlite")
	if err := os.Mkdir(installedLibrary, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installedLibrary, "blocker"), []byte("keep directory non-empty"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(paths.LibraryDB, 0o700) })

	service := NewService(paths)
	service.now = func() time.Time { return time.Date(2026, 7, 15, 10, 30, 0, 0, time.UTC) }
	preRestore := filepath.Join(paths.BackupsDir, "pre-restore-"+service.now().Format("20060102-150405"))
	err = service.replaceDatabases(extracted, Manifest{Files: map[string]string{
		"library.sqlite": "unused",
		"user.sqlite":    "unused",
	}})
	if err == nil {
		t.Fatal("replaceDatabases() succeeded despite injected install and rollback failures")
	}
	assertDataRootLocked(t, paths.Root)
	if !strings.Contains(err.Error(), preRestore) {
		t.Errorf("replaceDatabases() error %q does not identify retained pre-restore directory %q", err, preRestore)
	}
	preserved, readErr := os.ReadFile(filepath.Join(preRestore, "library.sqlite"))
	if readErr != nil {
		t.Fatalf("preserved library database is unavailable: %v", readErr)
	}
	if string(preserved) != "original library database" {
		t.Fatalf("preserved library database = %q", preserved)
	}
	restoredUser, readErr := os.ReadFile(paths.UserDB)
	if readErr != nil || string(restoredUser) != "original user database" {
		t.Fatalf("restored user database = %q, err=%v", restoredUser, readErr)
	}
}

func TestServiceCreateRejectsManagedDatabaseDestinations(t *testing.T) {
	tests := []struct {
		name        string
		destination func(storage.Paths) string
	}{
		{name: "user", destination: func(paths storage.Paths) string { return paths.UserDB }},
		{name: "library", destination: func(paths storage.Paths) string { return paths.LibraryDB }},
		{name: "puzzles", destination: func(paths storage.Paths) string { return paths.PuzzlesDB }},
		{name: "courses", destination: func(paths storage.Paths) string { return paths.CoursesDB }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paths := managedDatabaseFixture(t)
			destination := test.destination(paths)
			assertManagedDestinationRejected(t, paths, destination, destination)
		})
	}
}

func TestServiceCreateRejectsManagedDatabaseDestinationAliases(t *testing.T) {
	t.Run("dot-dot", func(t *testing.T) {
		paths := managedDatabaseFixture(t)
		destination := paths.BackupsDir + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "user.sqlite"
		assertManagedDestinationRejected(t, paths, destination, paths.UserDB)
	})

	t.Run("symlinked-parent", func(t *testing.T) {
		paths := managedDatabaseFixture(t)
		aliasRoot := filepath.Join(t.TempDir(), "data-alias")
		if err := os.Symlink(paths.Root, aliasRoot); err != nil {
			t.Fatal(err)
		}
		assertManagedDestinationRejected(t, paths, filepath.Join(aliasRoot, "user.sqlite"), paths.UserDB)
	})

	t.Run("symlinked-file", func(t *testing.T) {
		paths := managedDatabaseFixture(t)
		alias := filepath.Join(t.TempDir(), "user-alias.sqlite")
		if err := os.Symlink(paths.UserDB, alias); err != nil {
			t.Fatal(err)
		}
		assertManagedDestinationRejected(t, paths, alias, paths.UserDB)
		info, err := os.Lstat(alias)
		if err != nil {
			t.Fatalf("destination alias was removed: %v", err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatal("destination alias was replaced")
		}
	})
}

func TestServiceCreateRejectsCaseVariantManagedDatabaseDestination(t *testing.T) {
	paths := managedDatabaseFixture(t)
	destination := filepath.Join(paths.Root, "USER.SQLITE")
	managedInfo, err := os.Stat(paths.UserDB)
	if err != nil {
		t.Fatal(err)
	}
	destinationInfo, err := os.Stat(destination)
	if os.IsNotExist(err) {
		t.Skip("temp filesystem is case-sensitive: uppercase alias does not resolve to existing user.sqlite")
	}
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(destinationInfo, managedInfo) {
		t.Fatal("case-variant destination exists but does not identify user.sqlite")
	}
	assertManagedDestinationRejected(t, paths, destination, paths.UserDB)
}

func managedDatabaseFixture(t *testing.T) storage.Paths {
	t.Helper()
	paths := storage.PathsAt(filepath.Join(t.TempDir(), "data"))
	userDB, libraryDB := openBackupStores(t, paths)
	if err := errorsJoin(userDB.Close(), libraryDB.Close()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.PuzzlesDB, []byte("managed puzzles database"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.CoursesDB, []byte("managed courses database"), 0o600); err != nil {
		t.Fatal(err)
	}
	return paths
}

func assertManagedDestinationRejected(t *testing.T, paths storage.Paths, destination, managed string) {
	t.Helper()
	before, err := os.ReadFile(managed)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(paths)
	if err := service.Create(context.Background(), destination, true); err == nil {
		t.Errorf("Create() accepted managed database destination %q", destination)
	}
	after, err := os.ReadFile(managed)
	if err != nil {
		t.Fatalf("managed database became unreadable: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("Create() replaced the managed database")
	}
}

func readManifest(t *testing.T, path string) Manifest {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.Name != "manifest.json" {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer opened.Close()
		var manifest Manifest
		if err := json.NewDecoder(opened).Decode(&manifest); err != nil {
			t.Fatal(err)
		}
		return manifest
	}
	t.Fatal("manifest.json is missing")
	return Manifest{}
}

func zipFileWithContents(t *testing.T, name string, contents []byte) *zip.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.zip")
	output, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(output)
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	return reader.File[0]
}

func corruptZipEntry(t *testing.T, source, destination, entryName string) {
	t.Helper()
	reader, err := zip.OpenReader(source)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	output, err := os.Create(destination)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(output)
	for _, file := range reader.File {
		header := file.FileHeader
		created, err := writer.CreateHeader(&header)
		if err != nil {
			t.Fatal(err)
		}
		opened, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		if file.Name == entryName {
			_, err = created.Write([]byte("not a sqlite database"))
		} else {
			_, err = io.Copy(created, opened)
		}
		opened.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
}

func errorsJoin(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
