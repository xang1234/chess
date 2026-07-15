package backup

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chess-trainer/internal/storage"
)

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
	closed := false
	service := NewService(paths, func() error {
		closed = true
		return errorsJoin(userDB.Close(), libraryDB.Close())
	})
	service.now = func() time.Time { return time.Unix(123, 0) }
	destination := filepath.Join(t.TempDir(), "trainer-backup.zip")

	if err := service.Create(context.Background(), destination, true); err != nil {
		t.Fatal(err)
	}
	manifest := readManifest(t, destination)
	if manifest.Version != 1 || manifest.Files["user.sqlite"] == "" || manifest.Files["library.sqlite"] == "" {
		t.Fatalf("manifest=%+v", manifest)
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
	if err := service.Restore(context.Background(), destination); err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Fatal("restore did not close live databases before replacement")
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
	closed := false
	service := NewService(paths, func() error {
		closed = true
		return errorsJoin(userDB.Close(), libraryDB.Close())
	})
	valid := filepath.Join(t.TempDir(), "valid.zip")
	corrupt := filepath.Join(t.TempDir(), "corrupt.zip")
	if err := service.Create(context.Background(), valid, false); err != nil {
		t.Fatal(err)
	}
	corruptZipEntry(t, valid, corrupt, "user.sqlite")

	if err := service.Restore(context.Background(), corrupt); err == nil {
		t.Fatal("Restore() accepted a corrupt database")
	}
	if closed {
		t.Fatal("corrupt restore closed or replaced live databases")
	}
	var rating float64
	if err := userDB.QueryRow(`SELECT learner_rating FROM profile WHERE id = 1`).Scan(&rating); err != nil || rating != 1400 {
		t.Fatalf("live rating=%v err=%v", rating, err)
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
