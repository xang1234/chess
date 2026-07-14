package app

import (
	"os"
	"testing"

	"chess-trainer/internal/storage"
)

func TestOpenCreatesAndClosesAllStores(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	services, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{paths.PuzzlesDB, paths.UserDB, paths.LibraryDB} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("database %q: %v", path, err)
		}
	}
	if services.Catalog == nil || services.ImportJobs == nil || services.Training == nil {
		t.Fatal("core services were not composed")
	}
	if err := services.Close(); err != nil {
		t.Fatal(err)
	}
	if err := services.Close(); err != nil {
		t.Fatalf("second Close()=%v", err)
	}
}
