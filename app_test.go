package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/importjob"
	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/storage"
	"chess-trainer/internal/training"
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

func TestAppImportBindingsDelegateValidation(t *testing.T) {
	services, err := appservices.Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()
	app := NewApp(services)

	if _, err := app.StartPuzzleImport(importjob.ImportRequest{}); err == nil {
		t.Fatal("StartPuzzleImport() unexpectedly accepted an empty request")
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

func TestAppImportBindingsPreserveTypedAndLegacyFlows(t *testing.T) {
	importer := bindingImporter{called: make(chan bindingImportCall, 2)}
	emitter := bindingEmitter{finished: make(chan importjob.Result, 2)}
	jobs := importjob.NewService(map[importjob.Kind]importjob.Importer{
		importjob.KindLichess: importer,
	}, nil, emitter)
	defer jobs.Close()
	app := NewApp(&appservices.Services{ImportJobs: jobs})

	request := importjob.ImportRequest{
		Kind: importjob.KindLichess, SourceID: "school", Path: "/school.csv.zst",
	}
	if _, err := app.StartPuzzleImport(request); err != nil {
		t.Fatal(err)
	}
	call := receiveBindingCall(t, importer.called)
	if call.sourceID != request.SourceID || call.path != request.Path {
		t.Fatalf("typed call = %+v", call)
	}
	receiveImportResult(t, emitter.finished)

	if _, err := app.StartLichessImport("/lichess.csv.zst"); err != nil {
		t.Fatal(err)
	}
	call = receiveBindingCall(t, importer.called)
	if call.sourceID != "lichess" || call.path != "/lichess.csv.zst" {
		t.Fatalf("legacy call = %+v", call)
	}
	receiveImportResult(t, emitter.finished)
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
	services, err := appservices.Open(paths)
	if services != nil {
		t.Fatal("Open() returned services for corrupt data")
	}
	var integrityErr *storage.IntegrityError
	if !errors.As(err, &integrityErr) {
		t.Fatalf("Open() err=%v", err)
	}
	app := NewRecoveryApp(paths, integrityErr)
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
	app := NewApp(services)
	app.ctx = context.Background()

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
}
