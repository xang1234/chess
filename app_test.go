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
	"chess-trainer/internal/importing"
	"chess-trainer/internal/importjob"
	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/storage"
	"chess-trainer/internal/training"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type bindingImportCall struct {
	inspection importing.Inspection
}

type bindingImporter struct {
	called chan bindingImportCall
}

type bindingEmitter struct {
	finished chan importjob.Result
}

type bindingInspectionAdapter struct {
	descriptor puzzles.ImportFormatDescriptor
	sourceID   string
	inspected  chan string
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

func (bindingImporter) Supports(format puzzles.ImportFormat) bool {
	return format == puzzles.FormatCanonicalJSON
}

func (i bindingImporter) Import(
	_ context.Context,
	inspection importing.Inspection,
	_ importing.ProgressSink,
) (puzzles.ImportReport, error) {
	i.called <- bindingImportCall{inspection: inspection}
	return puzzles.ImportReport{}, nil
}

func (e bindingEmitter) Progress(string, importing.Progress) {}

func (e bindingEmitter) Finished(result importjob.Result) {
	e.finished <- result
}

func (a bindingInspectionAdapter) Descriptor() puzzles.ImportFormatDescriptor {
	if a.descriptor != (puzzles.ImportFormatDescriptor{}) {
		return a.descriptor
	}
	return puzzles.ImportFormatDescriptor{
		Format: puzzles.FormatCanonicalJSON, Label: "Canonical JSON",
		CanonicalExtension: ".json", FileFilterDescription: "JSON collection",
	}
}

func (a bindingInspectionAdapter) Inspect(
	_ context.Context,
	path string,
) (importing.Inspection, bool, error) {
	if a.inspected != nil {
		a.inspected <- path
	}
	return importing.Inspection{
		SourceID: a.sourceID, SourceIDOrigin: puzzles.SourceIDEmbedded,
	}, true, nil
}

func (bindingInspectionAdapter) NewDecoder(
	io.Reader,
	importing.Inspection,
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
	if _, err := app.StartPuzzleImport(importing.Inspection{}); err == nil {
		t.Fatal("StartPuzzleImport() unexpectedly accepted an empty path")
	}
	if _, err := app.InspectOpeningCourseImport(""); err == nil {
		t.Fatal("InspectOpeningCourseImport() unexpectedly accepted an empty path")
	}
	if _, err := app.StartOpeningCourseImport(importing.Inspection{}); err == nil {
		t.Fatal("StartOpeningCourseImport() unexpectedly accepted an empty path")
	}
	if err := app.CancelImport("missing"); err == nil {
		t.Fatal("CancelImport() unexpectedly accepted an unknown job")
	}
	if _, err := app.GetImportResult("missing"); err == nil {
		t.Fatal("GetImportResult() unexpectedly accepted an unknown job")
	}
}

func TestChooseOpeningCourseFileUsesDedicatedFilter(t *testing.T) {
	dialogs := fakeNativeDialogs{
		openPath: "/tmp/italian.ctcourse",
		open: func(options runtime.OpenDialogOptions) {
			if options.Title != "Choose an opening course" {
				t.Fatalf("Title = %q", options.Title)
			}
			want := []runtime.FileFilter{{
				DisplayName: "Opening course (*.ctcourse)", Pattern: "*.ctcourse",
			}}
			if len(options.Filters) != 1 || options.Filters[0] != want[0] {
				t.Fatalf("Filters = %+v, want %+v", options.Filters, want)
			}
		},
	}
	path, err := (&NormalController{
		actions: &controllerActions{ctx: context.Background(), dialogs: dialogs},
	}).ChooseOpeningCourseFile()
	if err != nil || path != "/tmp/italian.ctcourse" {
		t.Fatalf("ChooseOpeningCourseFile() = %q, %v", path, err)
	}
}

func TestOpeningCourseImportBindingsInspectAndStartConfirmedCourse(t *testing.T) {
	services, err := appservices.Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()
	app := NewNormalController(services)
	path := filepath.Join(t.TempDir(), "italian.ctcourse")
	contents, err := os.ReadFile(filepath.Join("internal", "openings", "testdata", "mini.ctcourse"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	inspection, err := app.InspectOpeningCourseImport(path)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Format != "coursepack" || inspection.SourceID != "synthetic-italian" {
		t.Fatalf("course inspection = %+v", inspection)
	}
	jobID, err := app.StartOpeningCourseImport(inspection)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		result, err := app.GetImportResult(jobID)
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != importjob.Running {
			if result.Status != importjob.Succeeded || result.Report.Counts["moves"] != 10 {
				t.Fatalf("course result = %+v", result)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("course import did not finish")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestChoosePuzzleImportFileUsesMacSafeZstandardFilter(t *testing.T) {
	descriptors := []puzzles.ImportFormatDescriptor{
		{
			Format: puzzles.FormatLichess, Label: "Lichess",
			CanonicalExtension: ".zst", FileFilterDescription: "Zstandard archive",
		},
		{
			Format: puzzles.FormatCanonicalJSON, Label: "Canonical JSON",
			CanonicalExtension: ".json", FileFilterDescription: "JSON collection",
		},
		{
			Format: puzzles.FormatTacticalPGN, Label: "Tactical PGN",
			CanonicalExtension: ".pgn", FileFilterDescription: "PGN collection",
		},
		{
			Format: puzzles.FormatLucasFNS, Label: "Lucas FNS",
			CanonicalExtension: ".fns", FileFilterDescription: "Lucas collection",
		},
		{
			Format: puzzles.FormatLinearFENUCI, Label: "Linear FEN/UCI",
			CanonicalExtension: ".txt", FileFilterDescription: "FEN/UCI collection",
		},
	}
	adapters := make([]puzzles.PuzzleAdapter, 0, len(descriptors))
	for _, descriptor := range descriptors {
		adapters = append(adapters, bindingInspectionAdapter{
			descriptor: descriptor,
		})
	}
	services := &appservices.Services{Importer: &puzzles.CollectionImporter{Adapters: adapters}}
	dialogs := fakeNativeDialogs{
		openPath: "/tmp/lichess.csv.zst",
		open: func(options runtime.OpenDialogOptions) {
			if options.Title != "Choose a puzzle collection" {
				t.Fatalf("Title = %q", options.Title)
			}
			wantFilters := []runtime.FileFilter{
				{DisplayName: "Zstandard archive (*.zst)", Pattern: "*.zst"},
				{DisplayName: "JSON collection (*.json)", Pattern: "*.json"},
				{DisplayName: "PGN collection (*.pgn)", Pattern: "*.pgn"},
				{DisplayName: "Lucas collection (*.fns)", Pattern: "*.fns"},
				{DisplayName: "FEN/UCI collection (*.txt)", Pattern: "*.txt"},
			}
			if len(options.Filters) != len(wantFilters) {
				t.Fatalf("Filters = %+v", options.Filters)
			}
			for index, want := range wantFilters {
				if options.Filters[index] != want {
					t.Fatalf("Filters[%d] = %+v, want %+v", index, options.Filters[index], want)
				}
			}
		},
	}
	path, err := (&NormalController{
		actions:  &controllerActions{ctx: context.Background(), dialogs: dialogs},
		services: services,
	}).ChoosePuzzleImportFile()
	if err != nil || path != "/tmp/lichess.csv.zst" {
		t.Fatalf("ChoosePuzzleImportFile() = %q, %v", path, err)
	}

	path, err = (&NormalController{
		actions:  &controllerActions{ctx: context.Background(), dialogs: fakeNativeDialogs{}},
		services: services,
	}).ChoosePuzzleImportFile()
	if err != nil || path != "" {
		t.Fatalf("cancelled ChoosePuzzleImportFile() = %q, %v", path, err)
	}
}

func TestInspectPuzzleImportDelegatesToCollectionImporter(t *testing.T) {
	inspected := make(chan string, 1)
	collection := &puzzles.CollectionImporter{Adapters: []puzzles.PuzzleAdapter{
		bindingInspectionAdapter{
			sourceID: "authoritative-source", inspected: inspected,
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

func TestStartPuzzleImportPassesConfirmedInspection(t *testing.T) {
	importer := bindingImporter{called: make(chan bindingImportCall, 2)}
	emitter := bindingEmitter{finished: make(chan importjob.Result, 2)}
	jobs := importjob.NewService(importer, nil, emitter)
	defer jobs.Close()
	app := NewNormalController(&appservices.Services{ImportJobs: jobs})
	path := filepath.Join(t.TempDir(), "selected.json")
	normalizedPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	inspection := importing.Inspection{
		Path: normalizedPath, Filename: filepath.Base(normalizedPath),
		Format: puzzles.FormatCanonicalJSON, SourceID: "authoritative-source",
		SourceIDOrigin: puzzles.SourceIDEmbedded, SourceName: "Club",
	}

	if _, err := app.StartPuzzleImport(inspection); err != nil {
		t.Fatal(err)
	}
	call := receiveBindingCall(t, importer.called)
	if call.inspection != inspection {
		t.Fatalf("generic call = %+v, want confirmed inspection %+v", call, inspection)
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
	if _, err := app.StartPuzzleImport(importing.Inspection{Path: "/collection.json"}); !errors.Is(err, appservices.ErrRuntimeUnavailable) {
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
