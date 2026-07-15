package main

import (
	"context"
	"errors"
	"os"
	"testing"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/storage"
	"chess-trainer/internal/training"
)

func TestAppImportBindingsDelegateValidation(t *testing.T) {
	services, err := appservices.Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()
	app := NewApp(services)

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
