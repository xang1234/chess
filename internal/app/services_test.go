package app

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"chess-trainer/internal/importjob"
	"chess-trainer/internal/puzzles"
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

type closeBlockingImporter struct {
	started chan context.Context
	release <-chan struct{}
}

func (i closeBlockingImporter) Import(
	ctx context.Context,
	_ string,
	_ string,
	_ puzzles.ProgressSink,
) (puzzles.ImportReport, error) {
	i.started <- ctx
	<-ctx.Done()
	<-i.release
	return puzzles.ImportReport{}, ctx.Err()
}

func TestServicesCloseWaitsForImportJobsBeforeDatabases(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	puzzleStore, err := storage.OpenPuzzleStore(paths.PuzzlesDB)
	if err != nil {
		t.Fatal(err)
	}

	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseImport := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseImport)
	importer := closeBlockingImporter{started: make(chan context.Context, 1), release: release}
	jobs := importjob.NewService(map[importjob.Kind]importjob.Importer{
		importjob.KindLichess: importer,
	}, nil, nil)
	services := &Services{PuzzleStore: puzzleStore, ImportJobs: jobs}
	_, err = jobs.Start(context.Background(), importjob.ImportRequest{
		Kind: importjob.KindLichess, SourceID: "lichess", Path: "/puzzles",
	})
	if err != nil {
		t.Fatal(err)
	}
	jobCtx := <-importer.started

	closed := make(chan error, 1)
	go func() { closed <- services.Close() }()
	select {
	case <-jobCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("Services.Close did not cancel the active import")
	}
	if err := puzzleStore.Writer.Ping(); err != nil {
		t.Fatalf("database closed before import exited: %v", err)
	}
	select {
	case err := <-closed:
		t.Fatalf("Services.Close returned before import exited: %v", err)
	default:
	}

	releaseImport()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Services.Close did not return after import exited")
	}
	if err := puzzleStore.Writer.Ping(); err == nil {
		t.Fatal("database remained open after Services.Close")
	}
}
