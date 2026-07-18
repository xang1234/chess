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

	"github.com/klauspost/compress/zstd"
)

var _ importjob.Importer = (*puzzles.CollectionImporter)(nil)

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

func TestOpenComposesEveryPuzzleImportFormat(t *testing.T) {
	services, err := Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()

	directory := t.TempDir()
	fixtures := []struct {
		name     string
		filename string
		contents string
		format   puzzles.ImportFormat
		zstd     bool
	}{
		{
			name: "Lichess zstandard", filename: "lichess.csv.zst", format: puzzles.FormatLichess, zstd: true,
			contents: `PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags
mate1,8/5Q1k/6K1/8/8/8/8/8 b - - 0 1,h7h8 f7f8,1200,60,95,200,mate mateIn1,https://lichess.org/example,
`,
		},
		{
			name: "canonical JSON", filename: "club.json", format: puzzles.FormatCanonicalJSON,
			contents: `{
  "schema":"chess-trainer-puzzles/v1",
  "source":{"id":"club-json"},
  "puzzles":[{
    "id":"json-1",
    "displayedFen":"4k3/8/8/8/8/8/4P3/4K3 w - - 0 1",
    "solver":"white",
    "solution":[{"uci":"e2e4","children":[{"uci":"e8f7"}]}]
  }]
}`,
		},
		{
			name: "tactical PGN", filename: "club.pgn", format: puzzles.FormatTacticalPGN,
			contents: `[Event "Direct solver turn"]
[SourceId "club-pgn"]
[PuzzleId "white-1"]
[SetUp "1"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"]
[Black "?"]

1. e4 Kf7 *
`,
		},
		{
			name: "Lucas FNS", filename: "pin.fns", format: puzzles.FormatLucasFNS,
			contents: "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1|Difficulty **|1. e4 Kf7 (1... Kd7) 2. Kf2 *\n",
		},
		{
			name: "linear FEN UCI", filename: "larion.txt", format: puzzles.FormatLinearFENUCI,
			contents: "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1 e2e4 e8f7 1375\n",
		},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			path := filepath.Join(directory, fixture.filename)
			writeServiceImportFixture(t, path, fixture.contents, fixture.zstd)
			inspection, err := services.Importer.Inspect(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			if inspection.Format != fixture.format {
				t.Fatalf("inspection format = %q, want %q", inspection.Format, fixture.format)
			}

			jobID, err := services.ImportJobs.Start(context.Background(), inspection)
			if err != nil {
				t.Fatalf("route %q is not configured: %v", inspection.Format, err)
			}
			result := waitForServiceImportResult(t, services.ImportJobs, jobID)
			if result.Status != importjob.Succeeded {
				t.Fatalf("route %q result = %+v", inspection.Format, result)
			}
			if result.Inspection != inspection {
				t.Fatalf("route %q inspection = %+v, want %+v", inspection.Format, result.Inspection, inspection)
			}
		})
	}
}

func writeServiceImportFixture(t *testing.T, path, contents string, compressed bool) {
	t.Helper()
	if !compressed {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := zstd.NewWriter(file)
	if err != nil {
		file.Close()
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(contents)); err != nil {
		writer.Close()
		file.Close()
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func waitForServiceImportResult(
	t *testing.T,
	jobs *importjob.Service,
	jobID string,
) importjob.Result {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		result, err := jobs.Result(jobID)
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != importjob.Running {
			return result
		}
		if time.Now().After(deadline) {
			t.Fatalf("import job %q did not finish", jobID)
		}
		time.Sleep(time.Millisecond)
	}
}

type closeBlockingImporter struct {
	started chan context.Context
	release <-chan struct{}
}

func (closeBlockingImporter) Supports(format puzzles.ImportFormat) bool {
	return format == puzzles.FormatLichess
}

func (i closeBlockingImporter) Import(
	ctx context.Context,
	_ puzzles.ImportInspection,
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
	jobs := importjob.NewService(importer, nil, nil)
	services := &Services{PuzzleStore: puzzleStore, ImportJobs: jobs}
	_, err = jobs.Start(context.Background(), puzzles.ImportInspection{
		Format: puzzles.FormatLichess, SourceID: "lichess", Path: "/puzzles",
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
