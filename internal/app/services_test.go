package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"chess-trainer/internal/importjob"
	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/storage"
)

func TestServicesDoesNotExposeBackupImplementation(t *testing.T) {
	if _, exposed := reflect.TypeOf(Services{}).FieldByName("Backup"); exposed {
		t.Fatal("Services exposes the backup implementation instead of owning backup operations")
	}
}

func TestServicesQuiesceForRestoreKeepsInstanceLockUntilShutdown(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	services, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}

	if err := services.quiesceForRestore(); err != nil {
		t.Fatal(err)
	}
	if err := services.quiesceForRestore(); err != nil {
		t.Fatalf("second quiesceForRestore() = %v", err)
	}
	if err := services.UserDB.Ping(); err == nil {
		t.Fatal("user database remained open after restore quiesce")
	}
	contender, err := storage.AcquireDataRootLock(paths.Root)
	if contender != nil {
		contender.Close()
	}
	if !errors.Is(err, storage.ErrDataRootLocked) {
		t.Fatalf("lock acquisition after restore quiesce = %v, want locked", err)
	}

	if err := services.Close(); err != nil {
		t.Fatal(err)
	}
	if err := services.Close(); err != nil {
		t.Fatalf("second Close() = %v", err)
	}
	contender, err = storage.AcquireDataRootLock(paths.Root)
	if err != nil {
		t.Fatalf("lock remains held after shutdown: %v", err)
	}
	if err := contender.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestServicesQuiesceWaitsForActiveOperationLease(t *testing.T) {
	services, err := Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()

	release, err := services.AcquireOperation()
	if err != nil {
		t.Fatal(err)
	}

	quiesced := make(chan error, 1)
	go func() { quiesced <- services.quiesceForRestore() }()
	select {
	case err := <-quiesced:
		t.Fatalf("QuiesceForRestore returned while an operation lease was active: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	release()
	select {
	case err := <-quiesced:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("QuiesceForRestore did not continue after the operation lease was released")
	}
}

func TestServicesRestoreRejectsQuiescedNormalRuntime(t *testing.T) {
	services, err := Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()

	if err := services.quiesceForRestore(); err != nil {
		t.Fatal(err)
	}
	if err := services.RestoreBackup(context.Background(), "/missing.zip"); !errors.Is(err, ErrRuntimeUnavailable) {
		t.Fatalf("RestoreBackup() after quiesce = %v, want runtime unavailable", err)
	}
}

func TestServicesCloseWaitsForMaintenanceTransaction(t *testing.T) {
	services, err := Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	services.maintenanceMu.Lock()

	closed := make(chan error, 1)
	go func() { closed <- services.Close() }()
	select {
	case err := <-closed:
		t.Fatalf("Close returned while a maintenance transaction was active: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	services.maintenanceMu.Unlock()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not continue after the maintenance transaction completed")
	}
}

func TestServicesRestoreKeepsInstanceLockAfterSuccessfulReplacement(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	services, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "backup.zip")
	if err := services.CreateBackup(context.Background(), destination, true); err != nil {
		services.Close()
		t.Fatal(err)
	}
	if err := services.RestoreBackup(context.Background(), destination); err != nil {
		services.Close()
		t.Fatal(err)
	}
	contender, err := storage.AcquireDataRootLock(paths.Root)
	if contender != nil {
		contender.Close()
	}
	if !errors.Is(err, storage.ErrDataRootLocked) {
		services.Close()
		t.Fatalf("lock acquisition after replacement = %v, want locked", err)
	}
	if err := services.Close(); err != nil {
		t.Fatal(err)
	}
}

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
