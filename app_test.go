package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/importjob"
	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/storage"
	"chess-trainer/internal/training"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type bindingImportCall struct {
	sourceID string
	path     string
}

type bindingImporter struct {
	called chan bindingImportCall
}

type bindingEmitter struct {
	finished chan importjob.Result
}

type bindingInspectionAdapter struct {
	format    puzzles.ImportFormat
	sourceID  string
	inspected chan string
}

type fakeNativeDialogs struct {
	openPath string
	open     func(runtime.OpenDialogOptions)
}

func (d fakeNativeDialogs) OpenFileDialog(
	_ context.Context,
	options runtime.OpenDialogOptions,
) (string, error) {
	if d.open != nil {
		d.open(options)
	}
	return d.openPath, nil
}

func (fakeNativeDialogs) SaveFileDialog(
	context.Context,
	runtime.SaveDialogOptions,
) (string, error) {
	return "", nil
}

func (i bindingImporter) Import(
	_ context.Context,
	sourceID string,
	path string,
	_ puzzles.ProgressSink,
) (puzzles.ImportReport, error) {
	i.called <- bindingImportCall{sourceID: sourceID, path: path}
	return puzzles.ImportReport{}, nil
}

func (e bindingEmitter) Progress(string, puzzles.Progress) {}

func (e bindingEmitter) Finished(result importjob.Result) {
	e.finished <- result
}

func (a bindingInspectionAdapter) Format() puzzles.ImportFormat {
	return a.format
}

func (a bindingInspectionAdapter) Inspect(
	_ context.Context,
	path string,
) (puzzles.ImportInspection, bool, error) {
	if a.inspected != nil {
		a.inspected <- path
	}
	return puzzles.ImportInspection{
		SourceID: a.sourceID, SourceIDOrigin: puzzles.SourceIDEmbedded,
	}, true, nil
}

func (bindingInspectionAdapter) NewDecoder(
	io.Reader,
	puzzles.ImportInspection,
) (puzzles.PuzzleDecoder, error) {
	return nil, errors.New("binding inspection adapter does not decode")
}

func TestAppImportBindingsDelegateValidation(t *testing.T) {
	services, err := appservices.Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()
	app := NewNormalController(services)

	if _, err := app.InspectPuzzleImport(""); err == nil {
		t.Fatal("InspectPuzzleImport() unexpectedly accepted an empty path")
	}
	if _, err := app.StartPuzzleImport(""); err == nil {
		t.Fatal("StartPuzzleImport() unexpectedly accepted an empty path")
	}
	if _, err := app.StartLichessImport(""); err == nil {
		t.Fatal("StartLichessImport() unexpectedly accepted an empty path")
	}
	if err := app.CancelImport("missing"); err == nil {
		t.Fatal("CancelImport() unexpectedly accepted an unknown job")
	}
	if _, err := app.GetImportResult("missing"); err == nil {
		t.Fatal("GetImportResult() unexpectedly accepted an unknown job")
	}
}

func TestChoosePuzzleImportFileUsesMacSafeZstandardFilter(t *testing.T) {
	dialogs := fakeNativeDialogs{
		openPath: "/tmp/lichess.csv.zst",
		open: func(options runtime.OpenDialogOptions) {
			if options.Title != "Choose a puzzle collection" {
				t.Fatalf("Title = %q", options.Title)
			}
			wantPatterns := []string{"*.zst", "*.pgn", "*.json", "*.fns", "*.txt"}
			if len(options.Filters) != len(wantPatterns) {
				t.Fatalf("Filters = %+v", options.Filters)
			}
			for index, want := range wantPatterns {
				if options.Filters[index].Pattern != want {
					t.Fatalf("Filters[%d].Pattern = %q, want %q", index, options.Filters[index].Pattern, want)
				}
			}
		},
	}
	path, err := (&NormalController{actions: &controllerActions{
		ctx: context.Background(), dialogs: dialogs,
	}}).ChoosePuzzleImportFile()
	if err != nil || path != "/tmp/lichess.csv.zst" {
		t.Fatalf("ChoosePuzzleImportFile() = %q, %v", path, err)
	}

	path, err = (&NormalController{
		actions: &controllerActions{ctx: context.Background(), dialogs: fakeNativeDialogs{}},
	}).ChoosePuzzleImportFile()
	if err != nil || path != "" {
		t.Fatalf("cancelled ChoosePuzzleImportFile() = %q, %v", path, err)
	}
}

func TestInspectPuzzleImportDelegatesToCollectionImporter(t *testing.T) {
	inspected := make(chan string, 1)
	collection := &puzzles.CollectionImporter{Adapters: []puzzles.PuzzleAdapter{
		bindingInspectionAdapter{
			format: puzzles.FormatCanonicalJSON, sourceID: "authoritative-source", inspected: inspected,
		},
	}}
	app := NewNormalController(&appservices.Services{Importer: collection})
	path := filepath.Join(t.TempDir(), "collection.json")

	inspection, err := app.InspectPuzzleImport(path)
	if err != nil {
		t.Fatal(err)
	}
	normalizedPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := <-inspected; got != normalizedPath {
		t.Fatalf("inspected path = %q, want %q", got, normalizedPath)
	}
	if inspection.Format != puzzles.FormatCanonicalJSON ||
		inspection.SourceID != "authoritative-source" ||
		inspection.Path != normalizedPath {
		t.Fatalf("inspection = %+v", inspection)
	}
}

func TestStartPuzzleImportUsesAuthoritativeInspection(t *testing.T) {
	importer := bindingImporter{called: make(chan bindingImportCall, 2)}
	emitter := bindingEmitter{finished: make(chan importjob.Result, 2)}
	jobs := importjob.NewService(map[importjob.Kind]importjob.Importer{
		importjob.KindCanonicalJSON: importer,
	}, nil, emitter)
	defer jobs.Close()
	collection := &puzzles.CollectionImporter{Adapters: []puzzles.PuzzleAdapter{
		bindingInspectionAdapter{
			format: puzzles.FormatCanonicalJSON, sourceID: "authoritative-source",
		},
	}}
	app := NewNormalController(&appservices.Services{Importer: collection, ImportJobs: jobs})
	path := filepath.Join(t.TempDir(), "selected.json")

	if _, err := app.StartPuzzleImport(path); err != nil {
		t.Fatal(err)
	}
	call := receiveBindingCall(t, importer.called)
	normalizedPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if call.sourceID != "authoritative-source" || call.path != normalizedPath {
		t.Fatalf("generic call = %+v, want authoritative source/path", call)
	}
	receiveImportResult(t, emitter.finished)
}

func TestStartLichessImportDelegatesToGenericInspectionFlow(t *testing.T) {
	importer := bindingImporter{called: make(chan bindingImportCall, 1)}
	emitter := bindingEmitter{finished: make(chan importjob.Result, 1)}
	jobs := importjob.NewService(map[importjob.Kind]importjob.Importer{
		importjob.KindCanonicalJSON: importer,
	}, nil, emitter)
	defer jobs.Close()
	collection := &puzzles.CollectionImporter{Adapters: []puzzles.PuzzleAdapter{
		bindingInspectionAdapter{
			format: puzzles.FormatCanonicalJSON, sourceID: "legacy-authoritative-source",
		},
	}}
	app := NewNormalController(&appservices.Services{Importer: collection, ImportJobs: jobs})
	path := filepath.Join(t.TempDir(), "legacy-name.csv.zst")

	if _, err := app.StartLichessImport(path); err != nil {
		t.Fatal(err)
	}
	call := receiveBindingCall(t, importer.called)
	normalizedPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if call.sourceID != "legacy-authoritative-source" || call.path != normalizedPath {
		t.Fatalf("legacy call = %+v", call)
	}
	receiveImportResult(t, emitter.finished)
}

func TestGenericImportBindingsRespectNormalOperationLifecycle(t *testing.T) {
	services := &appservices.Services{Importer: &puzzles.CollectionImporter{}}
	if err := services.Close(); err != nil {
		t.Fatal(err)
	}
	app := NewNormalController(services)
	if _, err := app.InspectPuzzleImport("/collection.json"); !errors.Is(err, appservices.ErrRuntimeUnavailable) {
		t.Fatalf("InspectPuzzleImport() error = %v, want runtime unavailable", err)
	}
	if _, err := app.StartPuzzleImport("/collection.json"); !errors.Is(err, appservices.ErrRuntimeUnavailable) {
		t.Fatalf("StartPuzzleImport() error = %v, want runtime unavailable", err)
	}
}

func receiveBindingCall(t *testing.T, calls <-chan bindingImportCall) bindingImportCall {
	t.Helper()
	select {
	case call := <-calls:
		return call
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for import binding call")
		return bindingImportCall{}
	}
}

func receiveImportResult(t *testing.T, results <-chan importjob.Result) importjob.Result {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for import result")
		return importjob.Result{}
	}
}

func TestRecoveryModePreservesCorruptData(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.UserDB, []byte("corrupt user database"), 0o600); err != nil {
		t.Fatal(err)
	}
	services, err := appservices.OpenApplication(paths)
	if services == nil {
		t.Fatal("OpenApplication() did not return recovery lifecycle")
	}
	defer services.Close()
	var recoveryErr *appservices.RecoveryRequiredError
	if !errors.As(err, &recoveryErr) {
		t.Fatalf("OpenApplication() err=%v", err)
	}
	app := NewRecoveryController(services, recoveryErr)
	state := app.GetRecoveryState()
	if !state.Required || state.Path != paths.UserDB || state.Detail == "" {
		t.Fatalf("recovery state=%+v", state)
	}
	for _, path := range []string{paths.PuzzlesDB, paths.LibraryDB} {
		if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("recovery startup created %s: %v", path, statErr)
		}
	}
}

func TestAppProfileBindingsRoundTrip(t *testing.T) {
	services, err := appservices.Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()
	app := NewNormalController(services)
	app.actions.ctx = context.Background()

	profile, err := app.GetProfile()
	if err != nil || profile != nil {
		t.Fatalf("GetProfile()=%+v err=%v", profile, err)
	}
	want := training.Profile{LearnerRating: 1200, SessionSize: 10}
	if err := app.UpdateProfile(want); err != nil {
		t.Fatal(err)
	}
	profile, err = app.GetProfile()
	if err != nil || profile == nil || *profile != want {
		t.Fatalf("GetProfile()=%+v err=%v", profile, err)
	}
	filters, err := app.GetPracticeFilters()
	if err != nil {
		t.Fatal(err)
	}
	if filters.LearnerRatingBounds != puzzles.DefaultLearnerRatingBounds() {
		t.Fatalf("GetPracticeFilters() learner bounds = %+v, want fallback %+v",
			filters.LearnerRatingBounds, puzzles.DefaultLearnerRatingBounds())
	}
}
