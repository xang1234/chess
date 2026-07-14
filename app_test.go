package main

import (
	"context"
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
